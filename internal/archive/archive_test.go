package archive

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSearch(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("q")
		_, _ = w.Write([]byte(`{"response":{"numFound":2,"docs":[
			{"identifier":"gd77","title":"Live 1977","creator":["Grateful Dead","Band"],"date":"1977-05-08T00:00:00Z","downloads":42,"licenseurl":"http://cc"},
			{"identifier":"semtitulo","downloads":1},
			{"title":"sem id"}
		]}}`))
	}))
	defer srv.Close()
	c := New()
	c.BaseURL = srv.URL

	res, err := c.Search(context.Background(), "grateful", 1)
	if err != nil {
		t.Fatal(err)
	}
	if gotQuery != "mediatype:audio AND (grateful)" {
		t.Fatalf("query = %q", gotQuery)
	}
	if res.Total != 2 || len(res.Items) != 2 {
		t.Fatalf("res = %+v", res)
	}
	first := res.Items[0]
	if first.Creator != "Grateful Dead, Band" || first.Date != "1977-05-08" || first.License != "http://cc" {
		t.Fatalf("first = %+v", first)
	}
	if res.Items[1].Title != "semtitulo" {
		t.Fatalf("título padrão = %q", res.Items[1].Title)
	}
}

func TestSearchEmptyQuery(t *testing.T) {
	res, err := New().Search(context.Background(), "   ", 1)
	if err != nil || len(res.Items) != 0 {
		t.Fatalf("res = %+v, err = %v", res, err)
	}
}

func TestSearchHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	c := New()
	c.BaseURL = srv.URL
	if _, err := c.Search(context.Background(), "x", 1); err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("err = %v", err)
	}
}

func TestTorrentURL(t *testing.T) {
	c := New()
	if got := c.TorrentURL("gd77"); got != "https://archive.org/download/gd77/gd77_archive.torrent" {
		t.Fatal(got)
	}
}
