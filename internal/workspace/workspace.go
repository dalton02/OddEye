// Package workspace dá ao editor da tela Coding o que ele precisa de uma pasta local: listar a árvore,
// ler e gravar arquivos e as alterações do git (status, +/− por arquivo e o conteúdo em HEAD).
// Todo caminho recebido é relativo à raiz da aba e nunca pode sair dela.
package workspace

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"oddeye/internal/proc"
)

// MaxFileSize é o maior arquivo que o editor abre.
const MaxFileSize = 2 << 20

// emptyTree é o hash da árvore vazia do git: base do diff num repositório ainda sem commit.
const emptyTree = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

var errOutside = errors.New("caminho fora da pasta do projeto")

type Entry struct {
	Name    string `json:"name"`
	Path    string `json:"path"` // relativo à raiz, com "/"
	Dir     bool   `json:"dir"`
	Ignored bool   `json:"ignored"` // ignorado pelo .gitignore: aparece esmaecido
}

type FileContent struct {
	Content string `json:"content"`
	Binary  bool   `json:"binary"`
	TooBig  bool   `json:"tooBig"`
}

type Change struct {
	Path    string `json:"path"`
	Status  string `json:"status"` // M, A, D, R ou U (novo, fora do git)
	Added   int    `json:"added"`
	Removed int    `json:"removed"`
}

type Changes struct {
	IsRepo bool     `json:"isRepo"`
	Branch string   `json:"branch"`
	Files  []Change `json:"files"`
}

type FileDiff struct {
	Original string `json:"original"` // conteúdo em HEAD (vazio para arquivo novo)
	Modified string `json:"modified"` // conteúdo no disco (vazio para arquivo apagado)
	Binary   bool   `json:"binary"`
}

// resolve junta root e rel garantindo que o resultado continua dentro de root.
func resolve(root, rel string) (string, error) {
	root = filepath.Clean(root)
	if !filepath.IsAbs(root) {
		return "", errors.New("a pasta do projeto precisa ser um caminho absoluto")
	}
	rel = filepath.FromSlash(rel)
	if filepath.IsAbs(rel) || filepath.VolumeName(rel) != "" {
		return "", errOutside
	}
	full := filepath.Join(root, rel)
	r, err := filepath.Rel(root, full)
	if err != nil || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
		return "", errOutside
	}
	return full, nil
}

func git(ctx context.Context, root string, stdin []byte, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", root, "-c", "core.quotepath=off"}, args...)...)
	proc.HideWindow(cmd)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return out, fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

// ListDir lista uma pasta da árvore: pastas primeiro, em ordem alfabética, sem a .git.
func ListDir(ctx context.Context, root, rel string) ([]Entry, error) {
	dir, err := resolve(root, rel)
	if err != nil {
		return nil, err
	}
	items, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, len(items))
	for _, it := range items {
		if it.Name() == ".git" {
			continue
		}
		isDir := it.IsDir()
		if it.Type()&fs.ModeSymlink != 0 {
			if st, err := os.Stat(filepath.Join(dir, it.Name())); err == nil {
				isDir = st.IsDir()
			}
		}
		p := it.Name()
		if rel != "" && rel != "." {
			p = filepath.ToSlash(filepath.Join(filepath.FromSlash(rel), it.Name()))
		}
		entries = append(entries, Entry{Name: it.Name(), Path: p, Dir: isDir})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Dir != entries[j].Dir {
			return entries[i].Dir
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
	markIgnored(ctx, root, entries)
	return entries, nil
}

// markIgnored pergunta ao git, de uma vez, quais entradas o .gitignore cobre. Fora de um repositório
// nada fica marcado.
func markIgnored(ctx context.Context, root string, entries []Entry) {
	if len(entries) == 0 {
		return
	}
	var in bytes.Buffer
	for _, e := range entries {
		in.WriteString(e.Path)
		in.WriteByte(0)
	}
	out, _ := git(ctx, root, in.Bytes(), "check-ignore", "-z", "--stdin")
	ignored := map[string]bool{}
	for _, p := range strings.Split(string(out), "\x00") {
		if p != "" {
			ignored[p] = true
		}
	}
	for i := range entries {
		entries[i].Ignored = ignored[entries[i].Path]
	}
}

func isBinary(b []byte) bool {
	if len(b) > 8000 {
		b = b[:8000]
	}
	return bytes.IndexByte(b, 0) >= 0
}

func ReadFile(root, rel string) (FileContent, error) {
	full, err := resolve(root, rel)
	if err != nil {
		return FileContent{}, err
	}
	st, err := os.Stat(full)
	if err != nil {
		return FileContent{}, err
	}
	if st.IsDir() {
		return FileContent{}, errors.New("é uma pasta")
	}
	if st.Size() > MaxFileSize {
		return FileContent{TooBig: true}, nil
	}
	b, err := os.ReadFile(full)
	if err != nil {
		return FileContent{}, err
	}
	if isBinary(b) {
		return FileContent{Binary: true}, nil
	}
	return FileContent{Content: string(b)}, nil
}

// WriteFile grava o arquivo mantendo as permissões que ele já tinha.
func WriteFile(root, rel, content string) error {
	full, err := resolve(root, rel)
	if err != nil {
		return err
	}
	mode := fs.FileMode(0o644)
	if st, err := os.Stat(full); err == nil {
		if st.IsDir() {
			return errors.New("é uma pasta")
		}
		mode = st.Mode().Perm()
	}
	return os.WriteFile(full, []byte(content), mode)
}

func hasHead(ctx context.Context, root string) bool {
	_, err := git(ctx, root, nil, "rev-parse", "--verify", "--quiet", "HEAD")
	return err == nil
}

// GetChanges lista o que mudou em relação ao HEAD: staged, não staged e arquivos novos.
func GetChanges(ctx context.Context, root string) (Changes, error) {
	if _, err := resolve(root, ""); err != nil {
		return Changes{}, err
	}
	if out, err := git(ctx, root, nil, "rev-parse", "--is-inside-work-tree"); err != nil || strings.TrimSpace(string(out)) != "true" {
		return Changes{Files: []Change{}}, nil
	}
	res := Changes{IsRepo: true, Files: []Change{}}
	if out, err := git(ctx, root, nil, "symbolic-ref", "--short", "-q", "HEAD"); err == nil {
		res.Branch = strings.TrimSpace(string(out))
	} else if out, err := git(ctx, root, nil, "rev-parse", "--short", "HEAD"); err == nil {
		res.Branch = strings.TrimSpace(string(out)) // HEAD solto: mostra o commit
	}

	out, err := git(ctx, root, nil, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return res, err
	}
	res.Files = parseStatus(out)

	base := "HEAD"
	if !hasHead(ctx, root) {
		base = emptyTree
	}
	if nums, err := git(ctx, root, nil, "diff", base, "--numstat", "-z", "-M"); err == nil {
		counts := parseNumstat(nums)
		for i := range res.Files {
			if c, ok := counts[res.Files[i].Path]; ok {
				res.Files[i].Added, res.Files[i].Removed = c[0], c[1]
			}
		}
	}
	for i, f := range res.Files {
		if f.Status == "U" {
			res.Files[i].Added = countLines(root, f.Path)
		}
	}
	return res, nil
}

// parseStatus lê `git status --porcelain=v1 -z`. Renomeações trazem o caminho antigo logo depois.
func parseStatus(out []byte) []Change {
	files := []Change{}
	parts := strings.Split(string(out), "\x00")
	for i := 0; i < len(parts); i++ {
		e := parts[i]
		if len(e) < 4 {
			continue
		}
		x, y, path := e[0], e[1], e[3:]
		var st string
		switch {
		case x == '?' && y == '?':
			st = "U"
		case x == '!':
			continue
		case x == 'R' || x == 'C':
			st = "R"
			i++ // pula o caminho de origem
		case x == 'D' || y == 'D':
			st = "D"
		case x == 'A':
			st = "A"
		default:
			st = "M"
		}
		files = append(files, Change{Path: path, Status: st})
	}
	sort.SliceStable(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files
}

// parseNumstat lê `git diff --numstat -z`: "add\tdel\tpath\0" ou, numa renomeação,
// "add\tdel\t\0origem\0destino\0". Binários vêm com "-" e contam zero.
func parseNumstat(out []byte) map[string][2]int {
	res := map[string][2]int{}
	parts := strings.Split(string(out), "\x00")
	for i := 0; i < len(parts); i++ {
		f := strings.SplitN(parts[i], "\t", 3)
		if len(f) != 3 {
			continue
		}
		a, _ := strconv.Atoi(f[0])
		d, _ := strconv.Atoi(f[1])
		path := f[2]
		if path == "" && i+2 < len(parts) {
			path = parts[i+2]
			i += 2
		}
		res[path] = [2]int{a, d}
	}
	return res
}

func countLines(root, rel string) int {
	full, err := resolve(root, rel)
	if err != nil {
		return 0
	}
	st, err := os.Stat(full)
	if err != nil || st.Size() > MaxFileSize {
		return 0
	}
	b, err := os.ReadFile(full)
	if err != nil || len(b) == 0 || isBinary(b) {
		return 0
	}
	n := bytes.Count(b, []byte{'\n'})
	if b[len(b)-1] != '\n' {
		n++
	}
	return n
}

// GetFileDiff devolve o arquivo em HEAD e no disco, para o editor de diff.
func GetFileDiff(ctx context.Context, root, rel string) (FileDiff, error) {
	full, err := resolve(root, rel)
	if err != nil {
		return FileDiff{}, err
	}
	var d FileDiff
	if orig, err := git(ctx, root, nil, "show", "HEAD:"+filepath.ToSlash(filepath.Clean(filepath.FromSlash(rel)))); err == nil {
		if isBinary(orig) || len(orig) > MaxFileSize {
			return FileDiff{Binary: true}, nil
		}
		d.Original = string(orig)
	}
	if st, err := os.Stat(full); err == nil && !st.IsDir() {
		if st.Size() > MaxFileSize {
			return FileDiff{Binary: true}, nil
		}
		b, err := os.ReadFile(full)
		if err != nil {
			return FileDiff{}, err
		}
		if isBinary(b) {
			return FileDiff{Binary: true}, nil
		}
		d.Modified = string(b)
	}
	return d, nil
}
