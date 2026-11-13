package review

import (
	"regexp"
	"strconv"
	"strings"
)

// DiffLines descreve, em números de linha do arquivo novo (lado RIGHT), o que o patch toca.
type DiffLines struct {
	Added       []int `json:"added"`
	Commentable []int `json:"commentable"` // adicionadas + contexto: as únicas que a API aceita em comentário de linha
}

var hunkHeader = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

func ParseHunks(patch string) DiffLines {
	out := DiffLines{Added: []int{}, Commentable: []int{}}
	right := 0
	inHunk := false
	for _, line := range strings.Split(patch, "\n") {
		if m := hunkHeader.FindStringSubmatch(line); m != nil {
			right, _ = strconv.Atoi(m[1])
			inHunk = true
			continue
		}
		if !inHunk || line == "" {
			continue
		}
		switch {
		case strings.HasPrefix(line, "+"):
			out.Added = append(out.Added, right)
			out.Commentable = append(out.Commentable, right)
			right++
		case strings.HasPrefix(line, "-"), strings.HasPrefix(line, `\`):
		default:
			out.Commentable = append(out.Commentable, right)
			right++
		}
	}
	return out
}
