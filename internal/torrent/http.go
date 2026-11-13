package torrent

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"oddeye/internal/music"
)

const RoutePrefix = "/torrent/"

// Middleware serve /torrent/<infohash>/<caminho> antes dos assets. O caminho nunca vira caminho
// de disco: ele só é comparado com a lista de arquivos do torrent, então não há como escapar da
// pasta. Só serve áudio já baixado por completo.
func (e *Engine) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, RoutePrefix) {
			next.ServeHTTP(w, r)
			return
		}
		hash, path, ok := ParseRoute(r.URL.EscapedPath())
		if !ok || !music.IsAudio(path) {
			http.NotFound(w, r)
			return
		}
		f, name, err := e.Open(hash, path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		defer f.Close()
		http.ServeContent(w, r, name, time.Time{}, f)
	})
}

// ParseRoute separa infohash e caminho de /torrent/<hash>/<caminho escapado>.
func ParseRoute(escaped string) (hash, path string, ok bool) {
	rest := strings.TrimPrefix(escaped, RoutePrefix)
	hash, encPath, found := strings.Cut(rest, "/")
	if !found || len(hash) != 40 || encPath == "" {
		return "", "", false
	}
	path, err := url.PathUnescape(encPath)
	if err != nil || path == "" {
		return "", "", false
	}
	return strings.ToLower(hash), path, true
}

// Chain encaixa vários middlewares do AssetServer, o primeiro mais por fora.
func Chain(mws ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		for i := len(mws) - 1; i >= 0; i-- {
			next = mws[i](next)
		}
		return next
	}
}
