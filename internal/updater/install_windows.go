package updater

import (
	"os"
	"path/filepath"
)

// Detect: no Windows o app é sempre um .exe numa pasta do usuário (instalador por usuário ou portátil).
func Detect() Install {
	exe, err := os.Executable()
	if err != nil {
		return Install{Kind: KindUnknown}
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	return Install{Kind: KindWindows, Path: exe, Exe: exe}
}

// Apply renomeia o .exe em uso (o Windows deixa renomear, não sobrescrever) e põe o novo no lugar.
func Apply(in Install, file string) error {
	if in.Kind != KindWindows {
		return ErrManual
	}
	old := in.Path + ".old"
	_ = os.Remove(old)
	if err := os.Rename(in.Path, old); err != nil {
		return manual(err)
	}
	if err := copyFile(file, in.Path, 0o755); err != nil {
		_ = os.Remove(in.Path)
		_ = os.Rename(old, in.Path)
		return manual(err)
	}
	return nil
}

// Cleanup apaga o .exe antigo que ficou da última atualização.
func Cleanup() {
	if exe, err := os.Executable(); err == nil {
		_ = os.Remove(exe + ".old")
	}
}
