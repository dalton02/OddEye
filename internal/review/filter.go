// Package review guarda as regras do revisor: quais arquivos merecem ir para o Claude,
// em quais linhas o GitHub aceita comentário e os rascunhos do review.
package review

import (
	"path"
	"strings"

	"oddeye/internal/gh"
)

var skippedDirs = []string{
	"test", "tests", "__tests__", "__snapshots__", "spec", "specs", "e2e", "fixtures", "testdata", "mocks",
	"docs", "doc", "documentation",
	"node_modules", "vendor", "dist", "build", "generated", ".github",
}

var skippedNames = map[string]bool{
	"package-lock.json": true, "yarn.lock": true, "pnpm-lock.yaml": true, "bun.lockb": true, "bun.lock": true,
	"go.sum": true, "Cargo.lock": true, "poetry.lock": true, "Gemfile.lock": true, "composer.lock": true,
	"CHANGELOG": true, "LICENSE": true, ".gitignore": true, ".prettierrc": true, ".editorconfig": true,
}

var skippedExts = map[string]bool{
	".md": true, ".mdx": true, ".txt": true, ".rst": true, ".adoc": true, ".lock": true, ".snap": true,
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".svg": true, ".ico": true, ".webp": true,
	".pdf": true, ".woff": true, ".woff2": true, ".ttf": true, ".map": true,
}

var testMarkers = []string{".spec.", ".test.", ".int-spec.", ".e2e-spec.", "_test.", "_spec.", ".stories."}

// SkipReason devolve por que um arquivo não vale a atenção do revisor (teste, doc, lockfile,
// gerado, binário), ou "" quando ele deve seguir para a triagem do Claude.
func SkipReason(f gh.File) string {
	p := strings.ToLower(f.Path)
	base := path.Base(p)
	for _, m := range testMarkers {
		if strings.Contains(base, m) {
			return "teste"
		}
	}
	for _, seg := range strings.Split(path.Dir(p), "/") {
		for _, d := range skippedDirs {
			if seg == d {
				if strings.HasPrefix(d, "doc") {
					return "documentação"
				}
				if d == "node_modules" || d == "vendor" || d == "dist" || d == "build" || d == "generated" {
					return "gerado"
				}
				if d == ".github" {
					return "configuração de CI"
				}
				return "teste"
			}
		}
	}
	if skippedNames[path.Base(f.Path)] {
		return "lockfile/meta"
	}
	ext := path.Ext(p)
	if ext == ".md" || ext == ".mdx" || ext == ".txt" || ext == ".rst" || ext == ".adoc" {
		return "documentação"
	}
	if skippedExts[ext] {
		return "binário/gerado"
	}
	if strings.HasSuffix(base, ".min.js") || strings.HasSuffix(base, ".min.css") || strings.Contains(base, ".generated.") || strings.HasSuffix(base, ".pb.go") {
		return "gerado"
	}
	if f.Patch == "" && f.Status != "removed" && f.Status != "renamed" {
		return "sem diff (binário ou grande demais)"
	}
	if f.Status == "renamed" && f.Additions == 0 && f.Deletions == 0 {
		return "só renomeado"
	}
	return ""
}

type Skipped struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// Split separa os arquivos candidatos dos descartados pelo filtro determinístico.
func Split(files []gh.File) (candidates []gh.File, skipped []Skipped) {
	for _, f := range files {
		if r := SkipReason(f); r != "" {
			skipped = append(skipped, Skipped{Path: f.Path, Reason: r})
			continue
		}
		candidates = append(candidates, f)
	}
	return candidates, skipped
}
