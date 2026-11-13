package proc

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// EnsurePath completa o PATH do processo para achar o gh e o claude. Um app aberto pelo Finder, pelo
// menu de aplicativos ou pelo autostart não herda o PATH do shell do usuário: no macOS ele é só
// /usr/bin:/bin, e instalações via Homebrew, npm, nvm ou o instalador do Claude Code ficam de fora.
func EnsurePath() {
	parts := filepath.SplitList(os.Getenv("PATH"))
	if runtime.GOOS != "windows" {
		parts = append(parts, loginShellPath()...)
	}
	parts = append(parts, knownDirs()...)
	seen := map[string]bool{}
	var out []string
	for _, p := range parts {
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	_ = os.Setenv("PATH", strings.Join(out, string(os.PathListSeparator)))
}

// loginShellPath pergunta o PATH ao shell interativo de login, que carrega .zshrc/.bashrc (nvm, asdf etc.).
func loginShellPath() []string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	const marker = "__ODDEYE_PATH__"
	out, err := exec.CommandContext(ctx, shell, "-ilc", "echo "+marker+"$PATH").Output()
	if err != nil {
		return nil
	}
	s := string(out)
	i := strings.LastIndex(s, marker)
	if i < 0 {
		return nil
	}
	return filepath.SplitList(strings.TrimSpace(s[i+len(marker):]))
}

func knownDirs() []string {
	home, _ := os.UserHomeDir()
	if runtime.GOOS == "windows" {
		appData, localAppData := os.Getenv("APPDATA"), os.Getenv("LOCALAPPDATA")
		return []string{
			filepath.Join(os.Getenv("ProgramFiles"), "GitHub CLI"),
			filepath.Join(appData, "npm"),
			filepath.Join(localAppData, "Programs", "claude"),
			filepath.Join(home, ".local", "bin"),
			filepath.Join(home, ".claude", "local"),
		}
	}
	return []string{
		"/opt/homebrew/bin",
		"/usr/local/bin",
		"/home/linuxbrew/.linuxbrew/bin",
		filepath.Join(home, ".local", "bin"),
		filepath.Join(home, ".claude", "local"),
		filepath.Join(home, ".npm-global", "bin"),
		filepath.Join(home, ".bun", "bin"),
		filepath.Join(home, ".volta", "bin"),
	}
}
