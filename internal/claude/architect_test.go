package claude

import (
	"strings"
	"testing"

	"oddeye/internal/gh"
)

func TestSanitizeBlueprintDropsDanglingReferences(t *testing.T) {
	bp := Blueprint{
		Kind: "poema",
		Nodes: []Node{
			{ID: "api", Label: "API", Kind: "interface", Change: "changed", Files: []string{"a.go", "ghost.go"}},
			{ID: "fila", Kind: "nave", Change: "new", Files: []string{"a.go", "b.go"}},
			{ID: "api", Label: "repetido"},
			{ID: ""},
		},
		Edges: []Edge{
			{From: "api", To: "fila", Style: "async", Change: "new"},
			{From: "api", To: "fila"},
			{From: "api", To: "sumiu"},
			{From: "api", To: "api"},
		},
		Flows: []Flow{
			{Title: "ok", Steps: []Step{{From: "api", To: "fila"}, {From: "sumiu", To: "api"}, {From: "fila", To: "fila"}}},
			{Title: "vazio", Steps: []Step{{From: "x", To: "y"}}},
		},
		Risks: []Risk{{Title: "r", Level: "enorme", Node: "sumiu"}, {Title: ""}},
	}
	got := SanitizeBlueprint(bp, []gh.File{{Path: "a.go", Additions: 10, Deletions: 2}, {Path: "b.go", Additions: 5}})

	if got.Kind != "feature" {
		t.Errorf("Kind = %q", got.Kind)
	}
	if len(got.Nodes) != 2 || got.Nodes[1].Kind != "service" || got.Nodes[1].Label != "fila" {
		t.Fatalf("Nodes = %+v", got.Nodes)
	}
	if w := got.Nodes[0].Weight; len(got.Nodes[0].Files) != 1 || w.Additions != 10 || w.Deletions != 2 {
		t.Errorf("api: files = %v, weight = %+v", got.Nodes[0].Files, w)
	}
	if w := got.Nodes[1].Weight; len(got.Nodes[1].Files) != 1 || w.Additions != 5 {
		t.Errorf("um arquivo não pode pesar em dois componentes: files = %v, weight = %+v", got.Nodes[1].Files, w)
	}
	if len(got.Edges) != 1 || got.Edges[0].Style != "async" {
		t.Errorf("Edges = %+v", got.Edges)
	}
	if len(got.Flows) != 1 || len(got.Flows[0].Steps) != 2 || got.Flows[0].Steps[0].Change != "context" {
		t.Errorf("Flows = %+v", got.Flows)
	}
	if len(got.Risks) != 1 || got.Risks[0].Level != "medium" || got.Risks[0].Node != "" {
		t.Errorf("Risks = %+v", got.Risks)
	}
	if got.Decisions == nil || got.Impact == nil || got.Questions == nil {
		t.Error("as listas vazias deveriam virar [] no JSON, não null")
	}
}

func TestSanitizeBlueprintEnforcesLimitsAndPlainText(t *testing.T) {
	bp := Blueprint{
		Summary:   "Troca o `approveWithdrawal` por uma fila.",
		Impact:    []ImpactAxis{{Axis: "performance", Score: 9}, {Axis: "simplicity", Score: -7}, {Axis: "performance", Score: 0}, {Axis: "beleza", Score: 1}},
		Questions: []string{"a", " ", "b", "c", "d", "e", "f", "g"},
	}
	for i := 0; i < MaxNodes+5; i++ {
		bp.Nodes = append(bp.Nodes, Node{ID: strings.Repeat("n", i+1)})
	}
	got := SanitizeBlueprint(bp, nil)

	if strings.Contains(got.Summary, "`") {
		t.Errorf("Summary ainda tem crase: %q", got.Summary)
	}
	if len(got.Nodes) != MaxNodes || len(got.Questions) != MaxQuestions {
		t.Errorf("limites: %d nós, %d perguntas", len(got.Nodes), len(got.Questions))
	}
	if len(got.Impact) != 2 || got.Impact[0].Axis != "simplicity" || got.Impact[0].Score != -2 || got.Impact[1].Score != 2 {
		t.Errorf("Impact = %+v", got.Impact)
	}
}

func TestBuildArchitectPromptKeepsContractWithCustomInstructions(t *testing.T) {
	files := []gh.File{{Path: "a.go", Patch: "@@ -1 +1 @@\n+y"}}
	p := BuildArchitectPrompt(gh.PRInfo{Title: "T"}, files, nil, []string{"cmd/api/main.go", "README.md"}, "Responda em inglês.")
	if !strings.HasPrefix(p, "Responda em inglês.") || strings.Contains(p, "arquiteto de software explicando") {
		t.Error("as instruções customizadas deveriam substituir as padrão")
	}
	for _, want := range []string{"nada de código no texto", "- cmd/api/ (1)", "- (raiz) (1)", "[   1] +y"} {
		if !strings.Contains(p, want) {
			t.Errorf("o prompt deveria conter %q", want)
		}
	}
}

func TestTreeSummaryFallsBackToShallowerDepth(t *testing.T) {
	var paths []string
	for i := 0; i < maxTreeLines+10; i++ {
		paths = append(paths, "src/mod/"+strings.Repeat("x", i+1)+"/f.go")
	}
	if got := treeSummary(paths); !strings.HasPrefix(got, "- src/mod/ (") || strings.Count(got, "\n") != 1 {
		t.Errorf("treeSummary = %q", got)
	}
}

func TestCacheKeyDiffersByMode(t *testing.T) {
	if CacheKey("x", "") == ArchitectCacheKey("x", "") {
		t.Error("o mesmo texto de instruções não pode dar a mesma chave nos dois modos")
	}
	if ArchitectCacheKey("", "") != ArchitectCacheKey(DefaultArchitectPrompt, "") {
		t.Error("vazio e o padrão explícito deveriam dar a mesma chave")
	}
	if ParseMode("architect") != ModeArchitect || ParseMode("qualquer") != ModeCode || ParseMode("") != ModeCode {
		t.Error("ParseMode deveria cair em code para valores desconhecidos")
	}
}
