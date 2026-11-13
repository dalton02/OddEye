package updater

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Detect acha o bundle .app que contém o executável (…/Oddeye.app/Contents/MacOS/oddeye).
func Detect() Install {
	exe, err := os.Executable()
	if err != nil {
		return Install{Kind: KindUnknown}
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	bundle := filepath.Dir(filepath.Dir(filepath.Dir(exe)))
	if !strings.HasSuffix(bundle, ".app") {
		return Install{Kind: KindUnknown, Path: exe, Exe: exe}
	}
	return Install{Kind: KindMacApp, Path: bundle, Exe: exe}
}

// Apply extrai o .app do zip (ditto preserva permissões e links do bundle) e troca o bundle inteiro.
func Apply(in Install, file string) error {
	if in.Kind != KindMacApp {
		return ErrManual
	}
	parent := filepath.Dir(in.Path)
	staging := filepath.Join(parent, ".oddeye-update")
	_ = os.RemoveAll(staging)
	if err := os.Mkdir(staging, 0o755); err != nil {
		return manual(err)
	}
	defer os.RemoveAll(staging)
	if out, err := exec.Command("ditto", "-x", "-k", file, staging).CombinedOutput(); err != nil {
		return fmt.Errorf("extraindo a atualização: %v: %s", err, out)
	}
	fresh := filepath.Join(staging, filepath.Base(in.Path))
	if _, err := os.Stat(fresh); err != nil {
		return fmt.Errorf("o zip não contém %s", filepath.Base(in.Path))
	}
	old := filepath.Join(staging, "old.app")
	if err := os.Rename(in.Path, old); err != nil {
		return manual(err)
	}
	if err := os.Rename(fresh, in.Path); err != nil {
		_ = os.Rename(old, in.Path)
		return manual(err)
	}
	return nil
}

// Cleanup não tem o que fazer no macOS (o bundle antigo some com a pasta de staging).
func Cleanup() {}
