package updater

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Detect descobre como o app foi instalado no Linux.
func Detect() Install {
	if appimage := os.Getenv("APPIMAGE"); appimage != "" {
		return Install{Kind: KindAppImage, Path: appimage, Exe: appimage}
	}
	exe, err := os.Executable()
	if err != nil {
		return Install{Kind: KindUnknown}
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	if strings.HasPrefix(exe, "/usr/") || strings.HasPrefix(exe, "/opt/") {
		if exec.Command("dpkg-query", "-S", exe).Run() == nil {
			return Install{Kind: KindDeb, Path: exe, Exe: exe}
		}
		if exec.Command("rpm", "-qf", exe).Run() == nil {
			return Install{Kind: KindRpm, Path: exe, Exe: exe}
		}
		return Install{Kind: KindUnknown, Path: exe, Exe: exe}
	}
	return Install{Kind: KindBinary, Path: exe, Exe: exe}
}

// Apply troca o app instalado pelo arquivo baixado.
func Apply(in Install, file string) error {
	switch in.Kind {
	case KindAppImage, KindBinary:
		return manual(replaceFile(file, in.Path, 0o755))
	case KindDeb:
		return pkexec(debCommand(file))
	case KindRpm:
		return pkexec(rpmCommand(file))
	default:
		return ErrManual
	}
}

func debCommand(file string) []string {
	if _, err := exec.LookPath("apt-get"); err == nil {
		return []string{"apt-get", "install", "-y", "--allow-downgrades", file}
	}
	return []string{"dpkg", "-i", file}
}

func rpmCommand(file string) []string {
	if _, err := exec.LookPath("dnf"); err == nil {
		return []string{"dnf", "install", "-y", file}
	}
	if _, err := exec.LookPath("zypper"); err == nil {
		return []string{"zypper", "--non-interactive", "install", "--allow-unsigned-rpm", file}
	}
	return []string{"rpm", "-U", file}
}

// pkexec roda o gerenciador de pacotes como root; o sistema mostra a janela de senha.
func pkexec(args []string) error {
	if _, err := exec.LookPath("pkexec"); err != nil {
		return fmt.Errorf("%w: pkexec não encontrado", ErrManual)
	}
	out, err := exec.Command("pkexec", args...).CombinedOutput()
	var exit *exec.ExitError
	if errors.As(err, &exit) && (exit.ExitCode() == 126 || exit.ExitCode() == 127) {
		return errors.New("a senha não foi confirmada; a atualização foi cancelada")
	}
	if err != nil {
		return fmt.Errorf("a instalação do pacote falhou: %v: %s", err, lastLine(string(out)))
	}
	return nil
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return lines[len(lines)-1]
}

// Cleanup não tem o que fazer no Linux (a troca é por rename atômico).
func Cleanup() {}
