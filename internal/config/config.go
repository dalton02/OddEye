// Package config guarda as preferências do usuário (config.json) e o estado do watcher (state.json)
// no diretório de configuração do sistema. Preferência e estado ficam separados para que apagar o
// estado (PRs já vistas) nunca leve junto a lista de repositórios.
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"oddeye/internal/appdir"
	"oddeye/internal/gh"
)

const (
	DefaultPollMinutes = 5
	MinPollMinutes     = 2
)

type RepoConfig struct {
	Owner               string `json:"owner"`
	Name                string `json:"name"`
	OnlyReviewRequested bool   `json:"onlyReviewRequested"`
	PreAnalyze          bool   `json:"preAnalyze"`
	PromptOverride      string `json:"promptOverride"`
}

func (r RepoConfig) FullName() string { return r.Owner + "/" + r.Name }

type Config struct {
	PollIntervalMinutes int          `json:"pollIntervalMinutes"`
	Repos               []RepoConfig `json:"repos"`
	ReviewerPrompt      string       `json:"reviewerPrompt"` // vazio = prompt padrão embutido
	Model               string       `json:"model"`          // vazio = padrão do Claude Code
	LaunchAtLogin       bool         `json:"launchAtLogin"`
	// AutostartNoticeSeen marca que o usuário já viu o aviso de que o app abre junto com o sistema.
	AutostartNoticeSeen bool `json:"autostartNoticeSeen"`
	// SkippedVersion é a versão que o usuário pediu para pular; o aviso volta só na próxima.
	SkippedVersion string `json:"skippedVersion"`
	NotifyOwnPRs   bool   `json:"notifyOwnPRs"`
	IncludeDrafts  bool   `json:"includeDrafts"`
}

// Normalize corrige valores fora da faixa e remove repositórios repetidos ou vazios.
func (c Config) Normalize() Config {
	if c.PollIntervalMinutes == 0 {
		c.PollIntervalMinutes = DefaultPollMinutes
	}
	if c.PollIntervalMinutes < MinPollMinutes {
		c.PollIntervalMinutes = MinPollMinutes
	}
	seen := map[string]bool{}
	repos := []RepoConfig{}
	for _, r := range c.Repos {
		r.Owner, r.Name = strings.TrimSpace(r.Owner), strings.TrimSpace(r.Name)
		key := strings.ToLower(r.FullName())
		if r.Owner == "" || r.Name == "" || seen[key] {
			continue
		}
		seen[key] = true
		repos = append(repos, r)
	}
	c.Repos = repos
	c.Model = strings.TrimSpace(c.Model)
	return c
}

// PromptFor devolve as instruções do revisor para o repositório: a do repo, a global ou "" (padrão).
func (c Config) PromptFor(owner, name string) string {
	for _, r := range c.Repos {
		if strings.EqualFold(r.Owner, owner) && strings.EqualFold(r.Name, name) && strings.TrimSpace(r.PromptOverride) != "" {
			return r.PromptOverride
		}
	}
	return c.ReviewerPrompt
}

func (c Config) Repo(owner, name string) (RepoConfig, bool) {
	for _, r := range c.Repos {
		if strings.EqualFold(r.Owner, owner) && strings.EqualFold(r.Name, name) {
			return r, true
		}
	}
	return RepoConfig{}, false
}

// SeenPR é o que o watcher já mostrou ao usuário: a PR deixa de ser "nova" assim que entra aqui.
type SeenPR struct {
	HeadSha string    `json:"headSha"`
	SeenAt  time.Time `json:"seenAt"`
}

type State struct {
	Seen       map[string]SeenPR `json:"seen"`       // chave: owner/repo#n (minúsculo)
	KnownRepos map[string]bool   `json:"knownRepos"` // repos que já passaram pela primeira varredura
	Pending    []gh.OpenPR       `json:"pending"`    // PRs notificadas e ainda não abertas no app
	LastPoll   time.Time         `json:"lastPoll"`
	LastError  string            `json:"lastError"`
}

type Store struct {
	mu  sync.Mutex
	dir string
}

func NewStore() (*Store, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return NewStoreAt(filepath.Join(base, appdir.Name)), nil
}

func NewStoreAt(dir string) *Store { return &Store{dir: dir} }

func (s *Store) Dir() string { return s.dir }

// Load lê a configuração. Na primeira execução (sem config.json) a abertura no login vem ligada,
// para o app ficar na bandeja depois de reiniciar o computador sem o usuário precisar procurar a opção.
func (s *Store) Load() (Config, error) {
	c := Config{LaunchAtLogin: true}
	err := s.read("config.json", &c)
	return c.Normalize(), err
}

func (s *Store) Save(c Config) (Config, error) {
	c = c.Normalize()
	return c, s.write("config.json", c)
}

func (s *Store) LoadState() (State, error) {
	var st State
	err := s.read("state.json", &st)
	if st.Seen == nil {
		st.Seen = map[string]SeenPR{}
	}
	if st.KnownRepos == nil {
		st.KnownRepos = map[string]bool{}
	}
	return st, err
}

func (s *Store) SaveState(st State) error { return s.write("state.json", st) }

func (s *Store) read(name string, v any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(filepath.Join(s.dir, name))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// write grava num arquivo temporário e renomeia, para que uma queda no meio não deixe JSON pela metade.
func (s *Store) write(name string, v any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(s.dir, name+".tmp")
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(s.dir, name))
}
