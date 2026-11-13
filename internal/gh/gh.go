// Package gh conversa com o GitHub através do CLI `gh`, reaproveitando a autenticação
// que o usuário já tem configurada (os comentários saem na conta dele).
package gh

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"oddeye/internal/proc"
)

type PRRef struct {
	Owner  string `json:"owner"`
	Repo   string `json:"repo"`
	Number int    `json:"number"`
}

func (r PRRef) String() string {
	return fmt.Sprintf("%s/%s#%d", r.Owner, r.Repo, r.Number)
}

func (r PRRef) apiBase() string {
	return fmt.Sprintf("repos/%s/%s/pulls/%d", r.Owner, r.Repo, r.Number)
}

var prPath = regexp.MustCompile(`^/([^/]+)/([^/]+)/pull/(\d+)(?:/.*)?$`)

// ParsePRURL aceita links como https://github.com/o/r/pull/123, com ou sem sufixo
// (/files, /commits), query string ou fragmento. Também aceita a forma curta o/r#123.
func ParsePRURL(raw string) (PRRef, error) {
	raw = strings.TrimSpace(raw)
	if m := regexp.MustCompile(`^([\w.-]+)/([\w.-]+)#(\d+)$`).FindStringSubmatch(raw); m != nil {
		n, _ := strconv.Atoi(m[3])
		return PRRef{Owner: m[1], Repo: m[2], Number: n}, nil
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || !strings.HasSuffix(u.Host, "github.com") {
		return PRRef{}, errors.New("link inválido: use https://github.com/<dono>/<repo>/pull/<número>")
	}
	m := prPath.FindStringSubmatch(strings.TrimSuffix(u.Path, "/"))
	if m == nil {
		return PRRef{}, errors.New("o link não aponta para uma pull request")
	}
	n, _ := strconv.Atoi(m[3])
	return PRRef{Owner: m[1], Repo: m[2], Number: n}, nil
}

type Runner struct{}

func (Runner) run(ctx context.Context, stdin []byte, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "gh", args...)
	proc.HideWindow(cmd)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("gh %s: %s", args[0], msg)
	}
	return stdout.Bytes(), nil
}

// AuthStatus devolve nil quando o gh está instalado e autenticado.
func (r Runner) AuthStatus(ctx context.Context) error {
	if _, err := exec.LookPath("gh"); err != nil {
		return errors.New("GitHub CLI (gh) não encontrado no PATH: instale em https://cli.github.com")
	}
	if _, err := r.run(ctx, nil, "auth", "status"); err != nil {
		return errors.New("gh não está autenticado: rode `gh auth login` no terminal")
	}
	return nil
}

type PRInfo struct {
	Title       string `json:"title"`
	Body        string `json:"body"`
	URL         string `json:"url"`
	State       string `json:"state"`
	HeadRefOid  string `json:"headRefOid"`
	BaseRefOid  string `json:"baseRefOid"`
	HeadRefName string `json:"headRefName"`
	BaseRefName string `json:"baseRefName"`
	Additions   int    `json:"additions"`
	Deletions   int    `json:"deletions"`
	Author      struct {
		Login string `json:"login"`
	} `json:"author"`
}

func (r Runner) PRInfo(ctx context.Context, ref PRRef) (PRInfo, error) {
	out, err := r.run(ctx, nil, "pr", "view", strconv.Itoa(ref.Number),
		"--repo", ref.Owner+"/"+ref.Repo,
		"--json", "title,body,url,state,headRefOid,baseRefOid,headRefName,baseRefName,additions,deletions,author")
	if err != nil {
		return PRInfo{}, err
	}
	var info PRInfo
	if err := json.Unmarshal(out, &info); err != nil {
		return PRInfo{}, fmt.Errorf("resposta inesperada do gh pr view: %w", err)
	}
	return info, nil
}

// PRBodyHTML devolve a descrição da PR já renderizada pelo GitHub. As imagens e vídeos anexados vêm
// com URLs assinadas que expiram em poucos minutos, por isso o resultado não deve ir para cache.
func (r Runner) PRBodyHTML(ctx context.Context, ref PRRef) (string, error) {
	out, err := r.run(ctx, nil, "api", "-H", "Accept: application/vnd.github.html+json", ref.apiBase(), "--jq", ".body_html // \"\"")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

type File struct {
	Path         string `json:"filename"`
	Status       string `json:"status"` // added, removed, modified, renamed, copied, changed, unchanged
	PreviousPath string `json:"previous_filename,omitempty"`
	Additions    int    `json:"additions"`
	Deletions    int    `json:"deletions"`
	Patch        string `json:"patch,omitempty"` // vazio em binários e em diffs grandes demais para a API
}

func (r Runner) Files(ctx context.Context, ref PRRef) ([]File, error) {
	// --slurp junta as páginas num array de arrays.
	out, err := r.run(ctx, nil, "api", "--paginate", "--slurp", ref.apiBase()+"/files?per_page=100")
	if err != nil {
		return nil, err
	}
	var pages [][]File
	if err := json.Unmarshal(out, &pages); err != nil {
		return nil, fmt.Errorf("resposta inesperada da lista de arquivos: %w", err)
	}
	var files []File
	for _, p := range pages {
		files = append(files, p...)
	}
	return files, nil
}

// FileContent devolve o conteúdo bruto do arquivo na revisão informada.
func (r Runner) FileContent(ctx context.Context, ref PRRef, path, sha string) (string, error) {
	escaped := make([]string, 0)
	for _, seg := range strings.Split(path, "/") {
		escaped = append(escaped, url.PathEscape(seg))
	}
	endpoint := fmt.Sprintf("repos/%s/%s/contents/%s?ref=%s", ref.Owner, ref.Repo, strings.Join(escaped, "/"), sha)
	out, err := r.run(ctx, nil, "api", "-H", "Accept: application/vnd.github.raw", endpoint)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// Tree lista os caminhos de todos os arquivos do repositório na revisão. truncated indica que o
// GitHub cortou a listagem (repositórios muito grandes): a lista então é parcial.
func (r Runner) Tree(ctx context.Context, ref PRRef, sha string) (paths []string, truncated bool, err error) {
	out, err := r.run(ctx, nil, "api", fmt.Sprintf("repos/%s/%s/git/trees/%s?recursive=1", ref.Owner, ref.Repo, sha))
	if err != nil {
		return nil, false, err
	}
	var res struct {
		Truncated bool `json:"truncated"`
		Tree      []struct {
			Path string `json:"path"`
			Type string `json:"type"`
		} `json:"tree"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		return nil, false, fmt.Errorf("resposta inesperada da árvore do repositório: %w", err)
	}
	for _, e := range res.Tree {
		if e.Type == "blob" {
			paths = append(paths, e.Path)
		}
	}
	return paths, res.Truncated, nil
}

type ReviewComment struct {
	Path      string `json:"path"`
	Line      int    `json:"line"`
	StartLine int    `json:"start_line,omitempty"`
	Side      string `json:"side"`
	StartSide string `json:"start_side,omitempty"`
	Body      string `json:"body"`
}

type ReviewRequest struct {
	CommitID string          `json:"commit_id"`
	Event    string          `json:"event"` // COMMENT, APPROVE, REQUEST_CHANGES
	Body     string          `json:"body,omitempty"`
	Comments []ReviewComment `json:"comments"`
}

type ReviewResult struct {
	ID      int64  `json:"id"`
	HTMLURL string `json:"html_url"`
}

func (r Runner) SubmitReview(ctx context.Context, ref PRRef, req ReviewRequest) (ReviewResult, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return ReviewResult{}, err
	}
	out, err := r.run(ctx, payload, "api", "-X", "POST", ref.apiBase()+"/reviews", "--input", "-")
	if err != nil {
		return ReviewResult{}, err
	}
	var res ReviewResult
	_ = json.Unmarshal(out, &res)
	return res, nil
}
