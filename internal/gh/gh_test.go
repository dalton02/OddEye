package gh

import (
	"strings"
	"testing"
)

func TestParsePRURL(t *testing.T) {
	ok := map[string]PRRef{
		"https://github.com/acme/api/pull/42":                   {"acme", "api", 42},
		"https://github.com/acme/api/pull/42/files":             {"acme", "api", 42},
		"https://github.com/acme/api/pull/42/files?w=1#diff-ab": {"acme", "api", 42},
		"github.com/acme/my.repo/pull/7/":                       {"acme", "my.repo", 7},
		"  acme/api#99 ":                                        {"acme", "api", 99},
	}
	for in, want := range ok {
		got, err := ParsePRURL(in)
		if err != nil || got != want {
			t.Errorf("ParsePRURL(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	for _, in := range []string{"https://gitlab.com/a/b/pull/1", "https://github.com/acme/api/issues/3", "lixo"} {
		if _, err := ParsePRURL(in); err == nil {
			t.Errorf("ParsePRURL(%q) deveria falhar", in)
		}
	}
}

func TestParseMissingScopes(t *testing.T) {
	full := "  - Token scopes: 'gist', 'read:org', 'repo', 'workflow'"
	if got := parseMissingScopes(full); len(got) != 0 {
		t.Errorf("com todos os escopos = %v", got)
	}
	if got := parseMissingScopes("  - Token scopes: 'gist', 'repo'"); len(got) != 1 || got[0] != "read:org" {
		t.Errorf("sem read:org = %v", got)
	}
	if got := parseMissingScopes("✓ Logged in to github.com (GH_TOKEN)"); got != nil {
		t.Errorf("sem linha de escopos deveria ser nil, veio %v", got)
	}
}

func TestDeviceCodeParsing(t *testing.T) {
	out := "! One-time code (AB12-CD34) copied to clipboard\nOpen this URL to continue in your web browser: https://github.com/login/device\n"
	if got := deviceCode.FindString(out); got != "AB12-CD34" {
		t.Errorf("code = %q", got)
	}
	if got := deviceURL.FindString(out); got != "https://github.com/login/device" {
		t.Errorf("url = %q", got)
	}
}

func TestSearchQueriesBatchesRepos(t *testing.T) {
	repos := make([]string, 45)
	for i := range repos {
		repos[i] = "o/r"
	}
	qs := searchQueries(repos)
	if len(qs) != 3 || strings.Count(qs[0], "repo:") != 20 || strings.Count(qs[2], "repo:") != 5 {
		t.Errorf("searchQueries = %d buscas", len(qs))
	}
}

func TestParseSearch(t *testing.T) {
	out := []byte(`{"data":{"search":{"pageInfo":{"hasNextPage":true,"endCursor":"c1"},"nodes":[
	  {"number":7,"title":"T","url":"u","isDraft":false,"createdAt":"2026-11-01T10:00:00Z","headRefOid":"abc",
	   "author":{"login":"ana"},"repository":{"name":"api","owner":{"login":"acme"}},
	   "reviewRequests":{"nodes":[{"requestedReviewer":{"login":"bob"}},{"requestedReviewer":{}}]}},
	  {}]}}}`)
	prs, cursor, err := parseSearch(out)
	if err != nil || cursor != "c1" || len(prs) != 1 {
		t.Fatalf("parseSearch = %+v, %q, %v", prs, cursor, err)
	}
	p := prs[0]
	if p.Key() != "acme/api#7" || p.Author != "ana" || len(p.ReviewRequested) != 1 || p.ReviewRequested[0] != "bob" {
		t.Errorf("PR = %+v", p)
	}
}
