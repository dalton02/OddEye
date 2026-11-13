package music

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestMiddlewareServesOnlyAudioInsideFolder(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "music")
	_ = os.MkdirAll(dir, 0o755)
	_ = os.WriteFile(filepath.Join(dir, "01 Faixa.mp3"), []byte("audio"), 0o644)
	_ = os.WriteFile(filepath.Join(root, "segredo.mp3"), []byte("x"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "notas.txt"), []byte("x"), 0o644)
	s := &Store{dir: dir}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(299) })
	h := s.Middleware(next)

	cases := map[string]int{
		"/music/01%20Faixa.mp3":   200,
		"/music/..%2Fsegredo.mp3": 404,
		"/music/../segredo.mp3":   404,
		"/music/notas.txt":        404,
		"/index.html":             299,
	}
	for path, want := range cases {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", "http://wails.localhost"+path, nil))
		if rec.Code != want {
			t.Errorf("%s = %d; want %d", path, rec.Code, want)
		}
	}
	if got := s.List().Tracks; len(got) != 1 || got[0] != "01 Faixa.mp3" {
		t.Errorf("List = %v", got)
	}
}
