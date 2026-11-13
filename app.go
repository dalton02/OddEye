package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"oddeye/internal/appdir"
	"oddeye/internal/audio"
	"oddeye/internal/claude"
	"oddeye/internal/config"
	"oddeye/internal/gh"
	"oddeye/internal/music"
	"oddeye/internal/review"
	"oddeye/internal/tray"
	"oddeye/internal/watcher"
)

type App struct {
	ctx     context.Context
	gh      gh.Runner
	music   *music.Store
	store   *config.Store
	watcher *watcher.Watcher
	tray    *tray.Tray
	version string
	update  updateState

	mu         sync.Mutex
	cancel     context.CancelFunc
	authCancel context.CancelFunc
	session    *session
	inflight   map[string]chan struct{} // análises em andamento, por arquivo de cache

	preAnalyzeMu sync.Mutex
}

type session struct {
	ref      gh.PRRef
	info     gh.PRInfo
	files    map[string]gh.File
	repo     map[string]bool // todos os arquivos do repositório no head: abríveis como contexto
	analysis map[string]claude.FileAnalysis
	drafts   *review.DraftStore
	content  sync.Map // path -> string
}

func NewApp(library *music.Store, store *config.Store, version string) *App {
	return &App{music: library, store: store, version: version, inflight: map[string]chan struct{}{}}
}

func (a *App) ListMusic() music.Library {
	return a.music.List()
}

// ChooseMusicFolder abre o seletor de pasta; devolve nil quando o usuário cancela.
func (a *App) ChooseMusicFolder() (*music.Library, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Pasta com as músicas do Oddeye",
		DefaultDirectory: a.music.Dir(),
	})
	if err != nil || dir == "" {
		return nil, err
	}
	if err := a.music.SetDir(dir); err != nil {
		return nil, err
	}
	lib := a.music.List()
	return &lib, nil
}

// AudioStatus diz se o webview consegue tocar som. O frontend consulta antes de criar qualquer
// áudio: no Linux sem os plugins do GStreamer, tentar tocar só gera erro no log.
func (a *App) AudioStatus() audio.Status {
	return audio.Check()
}

func (a *App) OpenMusicFolder() {
	p := filepath.ToSlash(a.music.Dir())
	if !strings.HasPrefix(p, "/") {
		p = "/" + p // C:/Users/... no Windows vira file:///C:/Users/...
	}
	runtime.BrowserOpenURL(a.ctx, (&url.URL{Scheme: "file", Path: p}).String())
}

type DepsStatus struct {
	GhError       string   `json:"ghError"`
	GhInstalled   bool     `json:"ghInstalled"`
	MissingScopes []string `json:"missingScopes"`
	ClaudeError   string   `json:"claudeError"`
}

func (a *App) CheckDeps() DepsStatus {
	s := DepsStatus{MissingScopes: []string{}}
	_, lookErr := exec.LookPath("gh")
	s.GhInstalled = lookErr == nil
	if err := a.gh.AuthStatus(a.ctx); err != nil {
		s.GhError = err.Error()
	} else if missing := a.gh.MissingScopes(a.ctx); len(missing) > 0 {
		s.MissingScopes = missing
	}
	if err := claude.Available(); err != nil {
		s.ClaudeError = err.Error()
	}
	return s
}

type FileEntry struct {
	Path        string             `json:"path"`
	Status      string             `json:"status"`
	Additions   int                `json:"additions"`
	Deletions   int                `json:"deletions"`
	Severity    string             `json:"severity"`
	Why         string             `json:"why"`
	Explanation string             `json:"explanation"`
	Attention   []claude.Attention `json:"attention"`
	References  []claude.Reference `json:"references"`
}

type SessionView struct {
	Ref        gh.PRRef         `json:"ref"`
	Title      string           `json:"title"`
	URL        string           `json:"url"`
	Author     string           `json:"author"`
	HeadBranch string           `json:"headBranch"`
	BaseBranch string           `json:"baseBranch"`
	Additions  int              `json:"additions"`
	Deletions  int              `json:"deletions"`
	TotalFiles int              `json:"totalFiles"`
	Summary    string           `json:"summary"`
	Files      []FileEntry      `json:"files"`
	Others     []FileEntry      `json:"others"` // candidatos que o Claude não escolheu
	Skipped    []review.Skipped `json:"skipped"`
	Drafts     []review.Draft   `json:"drafts"`
	FromCache  bool             `json:"fromCache"`
	RepoFiles  []string         `json:"repoFiles"` // para o frontend transformar citações de arquivo em links
	Mode       claude.Mode      `json:"mode"`
	// Blueprint só vem no modo Arquiteto; nele Files fica vazio e todos os candidatos vão em Others.
	Blueprint *claude.Blueprint `json:"blueprint"`
}

// LoadPR busca a PR, filtra testes e docs e pede a análise ao Claude no modo pedido ("code" ou
// "architect"). A análise fica em cache por commit de head e por modo: reabrir a mesma PR sem commit
// novo não gasta outra análise.
func (a *App) LoadPR(link string, modeName string, forceRefresh bool) (*SessionView, error) {
	mode := claude.ParseMode(modeName)
	ref, err := gh.ParsePRURL(link)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.mu.Lock()
	if a.cancel != nil {
		a.cancel()
	}
	a.cancel = cancel
	a.mu.Unlock()
	defer cancel()

	emit := func(p claude.Progress) { runtime.EventsEmit(a.ctx, "analysis:progress", p) }
	emit(claude.Progress{Stage: "fetching", Message: "Buscando a PR " + ref.String() + " no GitHub..."})

	info, err := a.gh.PRInfo(ctx, ref)
	if err != nil {
		return nil, err
	}
	var (
		repoFiles []string
		treeErr   error
		treeDone  = make(chan struct{})
	)
	go func() {
		defer close(treeDone)
		repoFiles, _, treeErr = a.gh.Tree(ctx, ref, info.HeadRefOid)
	}()
	files, err := a.gh.Files(ctx, ref)
	if err != nil {
		return nil, err
	}
	candidates, skipped := review.Split(files)
	emit(claude.Progress{Stage: "fetching", Message: fmt.Sprintf("%d arquivos na PR, %d sobraram depois de tirar testes e docs.", len(files), len(candidates))})

	var (
		analysis  claude.Analysis
		blueprint *claude.Blueprint
		cached    bool
	)
	if mode == claude.ModeArchitect {
		tree := func() []string {
			<-treeDone
			return repoFiles
		}
		bp, hit, err := a.blueprint(ctx, ref, info, candidates, skipped, tree, forceRefresh, emit)
		if err != nil {
			return nil, err
		}
		blueprint, cached = &bp, hit
		analysis.Summary = bp.Summary
	} else {
		analysis, cached, err = a.analysis(ctx, ref, info, candidates, skipped, forceRefresh, emit)
		if err != nil {
			return nil, err
		}
	}

	drafts, err := review.NewDraftStore(ref)
	if err != nil {
		return nil, err
	}
	<-treeDone
	if treeErr != nil {
		// Sem a árvore só os arquivos da PR viram link; não vale derrubar a sessão por isso.
		repoFiles = nil
	}
	s := &session{ref: ref, info: info, files: map[string]gh.File{}, repo: map[string]bool{}, analysis: map[string]claude.FileAnalysis{}, drafts: drafts}
	for _, p := range repoFiles {
		s.repo[p] = true
	}
	for _, f := range files {
		s.files[f.Path] = f
	}
	view := &SessionView{
		Ref: ref, Title: info.Title, URL: info.URL, Author: info.Author.Login,
		HeadBranch: info.HeadRefName, BaseBranch: info.BaseRefName,
		Additions: info.Additions, Deletions: info.Deletions, TotalFiles: len(files),
		Summary: analysis.Summary, Files: []FileEntry{}, Others: []FileEntry{}, Skipped: skipped,
		Drafts: drafts.List(), FromCache: cached, RepoFiles: repoFiles,
		Mode: mode, Blueprint: blueprint,
	}
	if view.RepoFiles == nil {
		view.RepoFiles = []string{}
	}
	if view.Skipped == nil {
		view.Skipped = []review.Skipped{}
	}
	for _, fa := range analysis.Files {
		s.analysis[fa.Path] = fa
		f := s.files[fa.Path]
		view.Files = append(view.Files, FileEntry{
			Path: f.Path, Status: f.Status, Additions: f.Additions, Deletions: f.Deletions,
			Severity: fa.Severity, Why: fa.Why, Explanation: fa.Explanation, Attention: fa.Attention,
			References: fa.References,
		})
	}
	for _, f := range candidates {
		if _, ok := s.analysis[f.Path]; !ok {
			view.Others = append(view.Others, FileEntry{Path: f.Path, Status: f.Status, Additions: f.Additions, Deletions: f.Deletions})
		}
	}
	a.mu.Lock()
	a.session = s
	a.mu.Unlock()
	if a.watcher != nil {
		a.watcher.MarkOpened(ref)
	}
	go a.prefetchTour(s, analysis.Files)
	return view, nil
}

// analysis devolve a triagem do Claude para o commit atual da PR, do cache quando existe. A chave do
// cache inclui o prompt e o modelo da configuração.
func (a *App) analysis(ctx context.Context, ref gh.PRRef, info gh.PRInfo, candidates []gh.File, skipped []review.Skipped, forceRefresh bool, emit func(claude.Progress)) (claude.Analysis, bool, error) {
	if len(candidates) == 0 {
		return claude.Analysis{Summary: "Essa PR só mexe em testes, documentação ou arquivos gerados. Nada crítico para revisar por aqui!"}, false, nil
	}
	cfg, _ := a.store.Load()
	instructions := cfg.PromptFor(ref.Owner, ref.Repo)
	model := claude.ResolveModel(cfg.Model)
	key := cachePath(ref, info.HeadRefOid, claude.CacheKey(instructions, model), claude.ModeCode)

	return cachedRun(a, ctx, key, forceRefresh, emit, func() (claude.Analysis, error) {
		prompt := claude.BuildPrompt(info, candidates, skipped, instructions)
		result, err := claude.Analyze(ctx, prompt, model, emit)
		if err != nil {
			return claude.Analysis{}, err
		}
		return claude.Sanitize(result, candidates), nil
	})
}

// blueprint devolve a leitura de arquitetura da PR para o commit atual, do cache quando existe. tree
// só é chamada quando a análise vai mesmo rodar, porque esperar a árvore do repositório custa uma ida
// ao GitHub que o cache dispensa.
func (a *App) blueprint(ctx context.Context, ref gh.PRRef, info gh.PRInfo, candidates []gh.File, skipped []review.Skipped, tree func() []string, forceRefresh bool, emit func(claude.Progress)) (claude.Blueprint, bool, error) {
	if len(candidates) == 0 {
		empty := claude.Blueprint{Summary: "Essa PR só mexe em testes, documentação ou arquivos gerados. Não há mudança de arquitetura para desenhar aqui."}
		return claude.SanitizeBlueprint(empty, nil), false, nil
	}
	cfg, _ := a.store.Load()
	model := claude.ResolveModel(cfg.Model)
	key := cachePath(ref, info.HeadRefOid, claude.ArchitectCacheKey(cfg.ArchitectPrompt, model), claude.ModeArchitect)

	return cachedRun(a, ctx, key, forceRefresh, emit, func() (claude.Blueprint, error) {
		prompt := claude.BuildArchitectPrompt(info, candidates, skipped, tree(), cfg.ArchitectPrompt)
		result, err := claude.AnalyzeArchitecture(ctx, prompt, model, emit)
		if err != nil {
			return claude.Blueprint{}, err
		}
		return claude.SanitizeBlueprint(result, candidates), nil
	})
}

// cachedRun devolve o resultado guardado em key ou roda compute e guarda. Se a mesma análise já está
// rodando (pré-análise do watcher), espera por ela em vez de gastar outra chamada ao Claude. O bool
// diz se o resultado veio do cache.
func cachedRun[T any](a *App, ctx context.Context, key string, forceRefresh bool, emit func(claude.Progress), compute func() (T, error)) (T, bool, error) {
	var zero T
	for {
		if !forceRefresh {
			if cached, ok := loadCached[T](key); ok {
				return cached, true, nil
			}
		}
		a.mu.Lock()
		wait, running := a.inflight[key]
		if !running {
			done := make(chan struct{})
			a.inflight[key] = done
			a.mu.Unlock()
			defer func() {
				a.mu.Lock()
				delete(a.inflight, key)
				a.mu.Unlock()
				close(done)
			}()
			break
		}
		a.mu.Unlock()
		emit(claude.Progress{Stage: "thinking", Message: "Essa PR já está sendo analisada em segundo plano, aguardando..."})
		select {
		case <-wait:
			forceRefresh = false
		case <-ctx.Done():
			return zero, false, errors.New("análise cancelada")
		}
	}

	result, err := compute()
	if err != nil {
		return zero, false, err
	}
	saveCached(key, result)
	return result, false, nil
}

const prefetchConcurrency = 4

// prefetchTour baixa em paralelo o conteúdo de todos os arquivos do tour, para que a navegação
// não espere uma ida ao GitHub a cada seta.
func (a *App) prefetchTour(s *session, files []claude.FileAnalysis) {
	sem := make(chan struct{}, prefetchConcurrency)
	var wg sync.WaitGroup
	for _, fa := range files {
		f, ok := s.files[fa.Path]
		if !ok {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			_, _ = a.content(s, f)
		}()
	}
	wg.Wait()
}

func (a *App) CancelAnalysis() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cancel != nil {
		a.cancel()
	}
}

type FileView struct {
	Path        string             `json:"path"`
	Status      string             `json:"status"`
	Content     string             `json:"content"`
	Added       []int              `json:"added"`
	Commentable []int              `json:"commentable"`
	Severity    string             `json:"severity"`
	Why         string             `json:"why"`
	Explanation string             `json:"explanation"`
	Attention   []claude.Attention `json:"attention"`
	ReadOnly    bool               `json:"readOnly"` // arquivo removido: mostra a versão da base, sem comentário de linha
	InPR        bool               `json:"inPR"`     // false: arquivo de contexto, sem diff; comentário vai no corpo do review
}

// fileFor devolve o arquivo da PR ou, para um arquivo de contexto do repositório, um registro sem diff.
func (s *session) fileFor(path string) (gh.File, bool, error) {
	if f, ok := s.files[path]; ok {
		return f, true, nil
	}
	if s.repo[path] {
		return gh.File{Path: path, Status: "unchanged"}, false, nil
	}
	return gh.File{}, false, fmt.Errorf("arquivo %s não existe no repositório", path)
}

func (a *App) current() (*session, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.session == nil {
		return nil, errors.New("nenhuma PR carregada")
	}
	return a.session, nil
}

func (a *App) OpenFile(path string) (*FileView, error) {
	s, err := a.current()
	if err != nil {
		return nil, err
	}
	f, inPR, err := s.fileFor(path)
	if err != nil {
		return nil, err
	}
	fa := s.analysis[path]
	lines := review.ParseHunks(f.Patch)
	view := &FileView{
		Path: path, Status: f.Status, Added: lines.Added, Commentable: lines.Commentable,
		Severity: fa.Severity, Why: fa.Why, Explanation: fa.Explanation, Attention: fa.Attention,
		ReadOnly: f.Status == "removed", InPR: inPR,
	}
	if view.Attention == nil {
		view.Attention = []claude.Attention{}
	}
	content, err := a.content(s, f)
	if err != nil {
		return nil, err
	}
	view.Content = content
	return view, nil
}

func (a *App) content(s *session, f gh.File) (string, error) {
	if c, ok := s.content.Load(f.Path); ok {
		return c.(string), nil
	}
	path, sha := f.Path, s.info.HeadRefOid
	if f.Status == "removed" {
		sha = s.info.BaseRefOid
	}
	c, err := a.gh.FileContent(a.ctx, s.ref, path, sha)
	if err != nil {
		return "", err
	}
	s.content.Store(f.Path, c)
	return c, nil
}

type DraftInput struct {
	ID        string `json:"id"`
	Path      string `json:"path"`
	StartLine int    `json:"startLine"`
	Line      int    `json:"line"`
	Body      string `json:"body"`
}

func (a *App) SaveDraft(in DraftInput) ([]review.Draft, error) {
	s, err := a.current()
	if err != nil {
		return nil, err
	}
	f, _, err := s.fileFor(in.Path)
	if err != nil {
		return nil, err
	}
	if in.StartLine == 0 || in.StartLine > in.Line {
		in.StartLine = in.Line
	}
	commentable := map[int]bool{}
	for _, l := range review.ParseHunks(f.Patch).Commentable {
		commentable[l] = true
	}
	inDiff := f.Status != "removed" && f.Status != "unchanged"
	for l := in.StartLine; l <= in.Line && inDiff; l++ {
		inDiff = commentable[l]
	}
	if _, err := s.drafts.Upsert(review.Draft{ID: in.ID, Path: in.Path, StartLine: in.StartLine, Line: in.Line, Body: in.Body, InDiff: inDiff}); err != nil {
		return nil, err
	}
	return s.drafts.List(), nil
}

func (a *App) RemoveDraft(id string) ([]review.Draft, error) {
	s, err := a.current()
	if err != nil {
		return nil, err
	}
	if err := s.drafts.Remove(id); err != nil {
		return nil, err
	}
	return s.drafts.List(), nil
}

// SubmitReview publica todos os rascunhos num único review e limpa os rascunhos locais.
func (a *App) SubmitReview(event, body string) (string, error) {
	s, err := a.current()
	if err != nil {
		return "", err
	}
	switch event {
	case "COMMENT", "APPROVE", "REQUEST_CHANGES":
	default:
		return "", errors.New("tipo de review inválido")
	}
	drafts := s.drafts.List()
	if len(drafts) == 0 && body == "" && event != "APPROVE" {
		return "", errors.New("escreva algum comentário antes de enviar")
	}
	req := review.BuildReview(s.info.HeadRefOid, event, body, drafts)
	res, err := a.gh.SubmitReview(a.ctx, s.ref, req)
	if err != nil {
		return "", err
	}
	_ = s.drafts.Clear()
	if res.HTMLURL == "" {
		res.HTMLURL = s.info.URL
	}
	return res.HTMLURL, nil
}

func (a *App) OpenURL(url string) {
	runtime.BrowserOpenURL(a.ctx, url)
}

// cachePath identifica a análise pelo commit, pelo modo e pelo que muda o resultado (promptKey:
// instruções e modelo). O sufixo de versão de cada modo invalida o cache quando o formato da análise
// ou o contrato do prompt daquele modo mudam.
func cachePath(ref gh.PRRef, sha, promptKey string, mode claude.Mode) string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	suffix := "v3"
	if mode == claude.ModeArchitect {
		suffix = "architect.v1"
	}
	return filepath.Join(dir, appdir.Name, "analysis", fmt.Sprintf("%s_%s_%d_%s_%s.%s.json", ref.Owner, ref.Repo, ref.Number, sha, promptKey, suffix))
}

func loadCached[T any](p string) (T, bool) {
	var v T
	if p == "" {
		return v, false
	}
	data, err := os.ReadFile(p)
	if err != nil || json.Unmarshal(data, &v) != nil {
		var zero T
		return zero, false
	}
	return v, true
}

func saveCached[T any](p string, v T) {
	if p == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	data, _ := json.Marshal(v)
	_ = os.WriteFile(p, data, 0o644)
}
