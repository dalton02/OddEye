package claude

import (
	"strings"
	"testing"

	"oddeye/internal/gh"
)

func TestNumberPatchUsesNewFileLineNumbers(t *testing.T) {
	got := numberPatch("@@ -5,3 +7,3 @@ fn\n ctx\n-old\n+new\n end")
	want := "@@ -5,3 +7,3 @@ fn\n[   7]  ctx\n       -old\n[   8] +new\n[   9]  end\n"
	if got != want {
		t.Errorf("numberPatch =\n%s\nwant\n%s", got, want)
	}
}

func TestSanitizeDropsUnknownAndDuplicatePaths(t *testing.T) {
	a := Analysis{Files: []FileAnalysis{{Path: "a.go"}, {Path: "ghost.go"}, {Path: "a.go"}, {Path: "b.go"}}}
	got := Sanitize(a, []gh.File{{Path: "a.go"}, {Path: "b.go"}})
	if len(got.Files) != 2 || got.Files[0].Path != "a.go" || got.Files[1].Path != "b.go" {
		t.Errorf("Sanitize = %+v", got.Files)
	}
}

func TestBuildPromptOmitsDiffBeyondBudget(t *testing.T) {
	big := "@@ -1 +1 @@\n+" + strings.Repeat("x", maxPromptSize)
	p := BuildPrompt(gh.PRInfo{Title: "T"}, []gh.File{{Path: "huge.go", Patch: big}, {Path: "small.go", Patch: "@@ -1 +1 @@\n+y"}}, nil, "")
	if !strings.Contains(p, "(diff omitido por tamanho)") || !strings.Contains(p, "[   1] +y") {
		t.Error("o arquivo grande deveria ser omitido e o pequeno mantido")
	}
}

func TestBuildPromptKeepsFormatContractWithCustomInstructions(t *testing.T) {
	p := BuildPrompt(gh.PRInfo{Title: "T"}, []gh.File{{Path: "a.go", Patch: "@@ -1 +1 @@\n+y"}}, nil, "Seja breve e responda em inglês.")
	if !strings.HasPrefix(p, "Seja breve") || strings.Contains(p, "revisor de código sênior") {
		t.Error("as instruções customizadas deveriam substituir as padrão")
	}
	if !strings.Contains(p, "# Formato da resposta") || !strings.Contains(p, "no máximo 15 arquivos") {
		t.Error("o contrato de formato deveria entrar mesmo com instruções customizadas")
	}
}

func TestCacheKeyChangesWithInstructionsAndModel(t *testing.T) {
	base := CacheKey("", "")
	if base != CacheKey(DefaultReviewerPrompt, "") {
		t.Error("vazio e o padrão explícito deveriam dar a mesma chave")
	}
	if base == CacheKey("outro", "") || base == CacheKey("", "sonnet") {
		t.Error("trocar instruções ou modelo deveria mudar a chave")
	}
}
