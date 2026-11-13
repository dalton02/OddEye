package watcher

import (
	"context"
	"errors"
	"testing"
	"time"

	"oddeye/internal/config"
	"oddeye/internal/gh"
)

func pr(repo string, n int, author string) gh.OpenPR {
	return gh.OpenPR{Ref: gh.PRRef{Owner: "acme", Repo: repo, Number: n}, Author: author, HeadSha: "sha"}
}

func newState() config.State {
	return config.State{Seen: map[string]config.SeenPR{}, KnownRepos: map[string]bool{}}
}

var now = time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)

func TestFirstScanMarksSeenWithoutNotifying(t *testing.T) {
	st := newState()
	cfg := config.Config{Repos: []config.RepoConfig{{Owner: "acme", Name: "api"}}}
	if got := Evaluate(&st, cfg, "me", []gh.OpenPR{pr("api", 1, "ana"), pr("api", 2, "bob")}, now); len(got) != 0 {
		t.Fatalf("primeira varredura notificou %d PRs", len(got))
	}
	got := Evaluate(&st, cfg, "me", []gh.OpenPR{pr("api", 1, "ana"), pr("api", 2, "bob"), pr("api", 3, "ana")}, now)
	if len(got) != 1 || got[0].Ref.Number != 3 || len(st.Pending) != 1 {
		t.Errorf("segunda varredura = %+v, pending %d", got, len(st.Pending))
	}
	if again := Evaluate(&st, cfg, "me", []gh.OpenPR{pr("api", 3, "ana")}, now); len(again) != 0 {
		t.Error("a mesma PR não deveria notificar duas vezes")
	}
}

func TestFiltersDraftsOwnAndReviewRequested(t *testing.T) {
	st := newState()
	st.KnownRepos["acme/api"] = true
	cfg := config.Config{Repos: []config.RepoConfig{{Owner: "acme", Name: "api", OnlyReviewRequested: true}}}
	draft := pr("api", 1, "ana")
	draft.IsDraft = true
	draft.ReviewRequested = []string{"me"}
	asked := pr("api", 2, "ana")
	asked.ReviewRequested = []string{"Me"}
	notAsked := pr("api", 3, "ana")
	own := pr("api", 4, "me")
	got := Evaluate(&st, cfg, "me", []gh.OpenPR{draft, asked, notAsked, own}, now)
	if len(got) != 1 || got[0].Ref.Number != 2 {
		t.Fatalf("Evaluate = %+v", got)
	}
	notAsked.ReviewRequested = []string{"me"}
	draft.IsDraft = false
	got = Evaluate(&st, cfg, "me", []gh.OpenPR{draft, asked, notAsked, own}, now)
	if len(got) != 2 {
		t.Errorf("PRs que passaram a atender o filtro deveriam notificar, veio %+v", got)
	}
}

func TestPendingDropsClosedPRsAndRemovedRepos(t *testing.T) {
	st := newState()
	st.KnownRepos["acme/api"] = true
	st.KnownRepos["acme/old"] = true
	cfg := config.Config{Repos: []config.RepoConfig{{Owner: "acme", Name: "api"}}}
	Evaluate(&st, cfg, "me", []gh.OpenPR{pr("api", 1, "ana"), pr("api", 2, "ana")}, now)
	Evaluate(&st, cfg, "me", []gh.OpenPR{pr("api", 2, "ana")}, now)
	if len(st.Pending) != 1 || st.Pending[0].Ref.Number != 2 {
		t.Errorf("pending = %+v", st.Pending)
	}
	if st.KnownRepos["acme/old"] {
		t.Error("repo removido da config deveria sair de KnownRepos para ter primeira varredura silenciosa se voltar")
	}
}

func TestSeenRetention(t *testing.T) {
	st := newState()
	st.Seen["acme/api#1"] = config.SeenPR{SeenAt: now.Add(-15 * 24 * time.Hour)}
	st.Seen["acme/api#2"] = config.SeenPR{SeenAt: now.Add(-24 * time.Hour)}
	Evaluate(&st, config.Config{Repos: []config.RepoConfig{{Owner: "acme", Name: "api"}}}, "me", nil, now)
	if _, ok := st.Seen["acme/api#1"]; ok {
		t.Error("PR fechada há mais de 14 dias deveria ser esquecida")
	}
	if _, ok := st.Seen["acme/api#2"]; !ok {
		t.Error("PR vista recentemente deveria continuar marcada")
	}
}

type fakeSource struct {
	prs []gh.OpenPR
	err error
}

func (f *fakeSource) OpenPRs(context.Context, []string) ([]gh.OpenPR, error) { return f.prs, f.err }
func (f *fakeSource) Viewer(context.Context) (string, error)                 { return "me", nil }

func TestPollErrorKeepsPendingAndRecordsError(t *testing.T) {
	store := config.NewStoreAt(t.TempDir())
	_, _ = store.Save(config.Config{Repos: []config.RepoConfig{{Owner: "acme", Name: "api"}}})
	src := &fakeSource{prs: []gh.OpenPR{pr("api", 1, "ana")}}
	var notified []gh.OpenPR
	w := New(store, src, func(p []gh.OpenPR) { notified = append(notified, p...) }, func(Snapshot) {})
	w.poll(context.Background())
	src.prs = append(src.prs, pr("api", 2, "ana"))
	w.poll(context.Background())
	src.err = errors.New("offline")
	w.poll(context.Background())
	snap := w.Snapshot()
	if len(notified) != 1 || len(snap.Pending) != 1 || snap.LastError != "offline" {
		t.Errorf("notified=%d pending=%d err=%q", len(notified), len(snap.Pending), snap.LastError)
	}
	w.MarkOpened(gh.PRRef{Owner: "ACME", Repo: "api", Number: 2})
	if len(w.Snapshot().Pending) != 0 {
		t.Error("MarkOpened deveria tirar a PR dos pendentes")
	}
}
