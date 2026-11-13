// Package music entrega ao webview as músicas de uma pasta escolhida pelo usuário. O app não
// embute nenhuma faixa: quem quer trilha própria coloca os arquivos na pasta.
package music

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"oddeye/internal/appdir"
)

const RoutePrefix = "/music/"

var audioExts = map[string]bool{".mp3": true, ".ogg": true, ".oga": true, ".opus": true, ".wav": true, ".flac": true, ".m4a": true, ".aac": true, ".webm": true}

type Library struct {
	Dir    string   `json:"dir"`
	Tracks []string `json:"tracks"`
}

type settings struct {
	MusicDir string `json:"musicDir"`
}

// Store guarda a pasta escolhida em settings.json e serve os arquivos dela em /music/<nome>.
type Store struct {
	mu       sync.Mutex
	settings string
	dir      string
}

// NewStore usa a pasta salva nas preferências ou, na primeira vez, <config>/oddeye/music (criada vazia).
func NewStore() (*Store, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	s := &Store{settings: filepath.Join(base, appdir.Name, "settings.json"), dir: filepath.Join(base, appdir.Name, "music")}
	if data, err := os.ReadFile(s.settings); err == nil {
		var st settings
		if json.Unmarshal(data, &st) == nil && st.MusicDir != "" {
			s.dir = st.MusicDir
		}
	}
	_ = os.MkdirAll(s.dir, 0o755)
	return s, nil
}

func (s *Store) Dir() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dir
}

// List devolve as faixas da pasta em ordem alfabética (prefixe com números para ordenar a playlist).
func (s *Store) List() Library {
	dir := s.Dir()
	lib := Library{Dir: dir, Tracks: []string{}}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return lib
	}
	for _, e := range entries {
		if !e.IsDir() && audioExts[strings.ToLower(filepath.Ext(e.Name()))] {
			lib.Tracks = append(lib.Tracks, e.Name())
		}
	}
	sort.Strings(lib.Tracks)
	return lib
}

func (s *Store) SetDir(dir string) error {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return errors.New("pasta inválida")
	}
	s.mu.Lock()
	s.dir = dir
	s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(s.settings), 0o755); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(settings{MusicDir: dir}, "", "  ")
	return os.WriteFile(s.settings, data, 0o644)
}

// Middleware intercepta /music/<nome> antes dos assets (também no modo dev, em que o Vite
// responderia index.html para qualquer caminho desconhecido). Só serve arquivo de áudio que
// está diretamente na pasta: o nome não pode carregar separador nem sair dela.
func (s *Store) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, RoutePrefix) {
			next.ServeHTTP(w, r)
			return
		}
		name, err := url.PathUnescape(strings.TrimPrefix(r.URL.EscapedPath(), RoutePrefix))
		if err != nil || name == "" || name != filepath.Base(name) || strings.ContainsAny(name, `/\`) || !audioExts[strings.ToLower(filepath.Ext(name))] {
			http.NotFound(w, r)
			return
		}
		/* ServeFile atende Range, que o <audio> usa para buscar trechos */
		http.ServeFile(w, r, filepath.Join(s.Dir(), name))
	})
}
