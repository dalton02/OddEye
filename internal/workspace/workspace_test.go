package workspace

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func run(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func write(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func repo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run(t, dir, "init", "-q", "-b", "main")
	write(t, dir, "a.go", "package a\n\nfunc A() {}\n")
	write(t, dir, "gone.txt", "1\n2\n")
	write(t, dir, "sub/b.ts", "export const b = 1;\n")
	write(t, dir, ".gitignore", "dist/\n")
	run(t, dir, "add", ".")
	run(t, dir, "commit", "-q", "-m", "init")
	return dir
}

func TestChanges(t *testing.T) {
	dir := repo(t)
	ctx := context.Background()
	write(t, dir, "a.go", "package a\n\nfunc A() { println() }\nfunc B() {}\n")
	write(t, dir, "novo/c.md", "um\ndois\ntres")
	os.Remove(filepath.Join(dir, "gone.txt"))
	write(t, dir, "staged.txt", "x\n")
	run(t, dir, "add", "staged.txt")

	ch, err := GetChanges(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if !ch.IsRepo || ch.Branch != "main" {
		t.Fatalf("repo/branch: %+v", ch)
	}
	got := map[string]Change{}
	for _, f := range ch.Files {
		got[f.Path] = f
	}
	want := map[string]Change{
		"a.go":       {Path: "a.go", Status: "M", Added: 2, Removed: 1},
		"gone.txt":   {Path: "gone.txt", Status: "D", Added: 0, Removed: 2},
		"novo/c.md":  {Path: "novo/c.md", Status: "U", Added: 3},
		"staged.txt": {Path: "staged.txt", Status: "A", Added: 1},
	}
	if len(got) != len(want) {
		t.Fatalf("arquivos: %+v", ch.Files)
	}
	for p, w := range want {
		if got[p] != w {
			t.Errorf("%s: got %+v want %+v", p, got[p], w)
		}
	}
}

func TestChangesOutsideRepo(t *testing.T) {
	ch, err := GetChanges(context.Background(), t.TempDir())
	if err != nil || ch.IsRepo || len(ch.Files) != 0 {
		t.Fatalf("fora de repo: %+v %v", ch, err)
	}
}

func TestListDir(t *testing.T) {
	dir := repo(t)
	write(t, dir, "dist/out.js", "x")
	es, err := ListDir(context.Background(), dir, "")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range es {
		names = append(names, e.Name)
		if e.Name == "dist" && !e.Ignored {
			t.Error("dist deveria estar ignorado")
		}
		if e.Name == "a.go" && e.Ignored {
			t.Error("a.go não é ignorado")
		}
	}
	// pastas primeiro, sem .git
	if names[0] != "dist" || names[1] != "sub" || names[2] != ".gitignore" {
		t.Fatalf("ordem: %v", names)
	}
	sub, err := ListDir(context.Background(), dir, "sub")
	if err != nil || len(sub) != 1 || sub[0].Path != "sub/b.ts" {
		t.Fatalf("sub: %+v %v", sub, err)
	}
}

func TestReadWriteAndDiff(t *testing.T) {
	dir := repo(t)
	ctx := context.Background()
	if err := WriteFile(dir, "a.go", "package a\n"); err != nil {
		t.Fatal(err)
	}
	fc, err := ReadFile(dir, "a.go")
	if err != nil || fc.Content != "package a\n" {
		t.Fatalf("read: %+v %v", fc, err)
	}
	d, err := GetFileDiff(ctx, dir, "a.go")
	if err != nil || d.Original != "package a\n\nfunc A() {}\n" || d.Modified != "package a\n" {
		t.Fatalf("diff: %+v %v", d, err)
	}
	write(t, dir, "new.txt", "oi\n")
	d, _ = GetFileDiff(ctx, dir, "new.txt")
	if d.Original != "" || d.Modified != "oi\n" {
		t.Fatalf("diff novo: %+v", d)
	}
	write(t, dir, "bin.dat", "a\x00b")
	if fc, _ := ReadFile(dir, "bin.dat"); !fc.Binary {
		t.Fatal("esperava binário")
	}
}

func TestRejectsEscape(t *testing.T) {
	dir := repo(t)
	for _, rel := range []string{"../x", "sub/../../x", "/etc/passwd"} {
		if _, err := ReadFile(dir, rel); err != errOutside {
			t.Errorf("%s: esperava errOutside, veio %v", rel, err)
		}
		if err := WriteFile(dir, rel, "x"); err != errOutside {
			t.Errorf("%s: write deveria recusar", rel)
		}
	}
	if _, err := ListDir(context.Background(), dir, ".."); err != errOutside {
		t.Error("ListDir deveria recusar ..")
	}
}
