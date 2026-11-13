package review

import (
	"reflect"
	"testing"

	"oddeye/internal/gh"
)

func TestSkipReason(t *testing.T) {
	cases := map[string]string{
		"src/use-cases/pay/index.ts":          "",
		"src/use-cases/pay/index.spec.ts":     "teste",
		"internal/gh/gh_test.go":              "teste",
		"test/integration/billing/x.ts":       "teste",
		"documentation/area/regra.md":         "documentação",
		"README.md":                           "documentação",
		"package-lock.json":                   "lockfile/meta",
		"go.sum":                              "lockfile/meta",
		"src/infra/config/prisma/generated/x": "gerado",
	}
	for p, want := range cases {
		if got := SkipReason(gh.File{Path: p, Status: "modified", Patch: "@@ -1 +1 @@\n+x"}); got != want {
			t.Errorf("SkipReason(%q) = %q; want %q", p, got, want)
		}
	}
	if SkipReason(gh.File{Path: "img.bin", Status: "modified"}) == "" {
		t.Error("arquivo sem patch deveria ser descartado")
	}
}

func TestParseHunks(t *testing.T) {
	patch := "@@ -10,4 +10,5 @@ func x() {\n ctx\n-old\n+new1\n+new2\n ctx2\n\\ No newline at end of file\n@@ -40,2 +41,2 @@\n-a\n+b\n c"
	got := ParseHunks(patch)
	want := DiffLines{Added: []int{11, 12, 41}, Commentable: []int{10, 11, 12, 13, 41, 42}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseHunks = %+v; want %+v", got, want)
	}
}

func TestBuildReviewMovesOutsideCommentsToBody(t *testing.T) {
	req := BuildReview("abc", "COMMENT", "Geral", []Draft{
		{Path: "a.go", StartLine: 3, Line: 5, Body: "range", InDiff: true},
		{Path: "a.go", StartLine: 7, Line: 7, Body: "single", InDiff: true},
		{Path: "b.go", StartLine: 90, Line: 90, Body: "fora", InDiff: false},
	})
	wantComments := []gh.ReviewComment{
		{Path: "a.go", StartLine: 3, Line: 5, Side: "RIGHT", StartSide: "RIGHT", Body: "range"},
		{Path: "a.go", Line: 7, Side: "RIGHT", Body: "single"},
	}
	if !reflect.DeepEqual(req.Comments, wantComments) {
		t.Errorf("comments = %+v", req.Comments)
	}
	wantBody := "Geral\n\n### Comentários fora do diff\n\n**`b.go:L90`**\nfora"
	if req.Body != wantBody {
		t.Errorf("body = %q", req.Body)
	}
}
