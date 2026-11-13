package tray

import (
	"strings"
	"testing"

	"oddeye/internal/gh"
	"oddeye/internal/watcher"
)

func TestStatusLine(t *testing.T) {
	many := make([]gh.OpenPR, MaxPRs+2)
	cases := []struct {
		s    watcher.Snapshot
		want string
	}{
		{watcher.Snapshot{}, "Nenhum repositório configurado"},
		{watcher.Snapshot{Repos: 1, Checking: true}, "Verificando PRs..."},
		{watcher.Snapshot{Repos: 1, LastError: "offline"}, "Falha ao verificar: offline"},
		{watcher.Snapshot{Repos: 1, Pending: many[:1]}, "1 PR nova"},
		{watcher.Snapshot{Repos: 1, Pending: many}, "10 PRs novas (8 no menu)"},
	}
	for _, c := range cases {
		if got := StatusLine(c.s); got != c.want {
			t.Errorf("StatusLine = %q, want %q", got, c.want)
		}
	}
}

func TestPRLabelTruncates(t *testing.T) {
	p := gh.OpenPR{Ref: gh.PRRef{Owner: "acme", Repo: "api", Number: 7}, Title: strings.Repeat("título ", 20)}
	got := PRLabel(p)
	if !strings.HasPrefix(got, "api#7 · título") || !strings.HasSuffix(got, "…") || len([]rune(got)) != 64 {
		t.Errorf("PRLabel = %q", got)
	}
}
