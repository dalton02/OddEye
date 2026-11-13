package torrent

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseRoute(t *testing.T) {
	h := strings.Repeat("ab", 20)
	cases := []struct {
		in       string
		hash     string
		path     string
		ok       bool
	}{
		{"/torrent/" + h + "/disc%201/01%20faixa.mp3", h, "disc 1/01 faixa.mp3", true},
		{"/torrent/" + strings.ToUpper(h) + "/a.mp3", h, "a.mp3", true},
		{"/torrent/" + h + "/", "", "", false},
		{"/torrent/curto/a.mp3", "", "", false},
		{"/torrent/" + h, "", "", false},
		{"/torrent/" + h + "/%zz", "", "", false},
	}
	for _, c := range cases {
		hash, path, ok := ParseRoute(c.in)
		if ok != c.ok || hash != c.hash || path != c.path {
			t.Errorf("ParseRoute(%q) = %q, %q, %v", c.in, hash, path, ok)
		}
	}
}

func TestEngineLifecycleAndMiddleware(t *testing.T) {
	e, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	if list := e.List(); len(list) != 0 {
		t.Fatalf("List = %+v", list)
	}
	if _, err := e.Files("xyz"); err == nil {
		t.Fatal("infohash inválido deveria falhar")
	}
	if err := e.Remove(strings.Repeat("0", 40), true); err != nil {
		t.Fatalf("remover torrent desconhecido: %v", err)
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusTeapot) })
	h := e.Middleware(next)
	for path, want := range map[string]int{
		"/index.html": http.StatusTeapot,
		"/torrent/" + strings.Repeat("ab", 20) + "/x.txt": http.StatusNotFound,
		"/torrent/" + strings.Repeat("ab", 20) + "/x.mp3": http.StatusNotFound,
		"/torrent/../../etc/passwd":                        http.StatusNotFound,
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != want {
			t.Errorf("%s: código %d, queria %d", path, rec.Code, want)
		}
	}
}

func TestAddRejectsEmpty(t *testing.T) {
	e, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	if _, err := e.Add(t.Context(), "  ", ""); err == nil {
		t.Fatal("fonte vazia deveria falhar")
	}
}
