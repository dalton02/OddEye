package menuentry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureWritesMenuEntryForAppImage(t *testing.T) {
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	t.Setenv("APPIMAGE", "/home/eu/Apps/Oddeye.AppImage")
	if err := Ensure([]byte("png")); err != nil {
		t.Fatal(err)
	}
	icon := filepath.Join(data, "icons", "oddeye.png")
	if got, _ := os.ReadFile(icon); string(got) != "png" {
		t.Errorf("ícone não gravado: %q", got)
	}
	desktop := filepath.Join(data, "applications", "oddeye.desktop")
	got, _ := os.ReadFile(desktop)
	for _, want := range []string{
		`Exec="/home/eu/Apps/Oddeye.AppImage"` + "\n",
		"TryExec=/home/eu/Apps/Oddeye.AppImage\n",
		"Icon=" + icon + "\n",
	} {
		if !strings.Contains(string(got), want) {
			t.Errorf("entrada sem %q:\n%s", want, got)
		}
	}

	// AppImage movido: a próxima abertura corrige o caminho.
	t.Setenv("APPIMAGE", "/opt2/Oddeye.AppImage")
	if err := Ensure([]byte("png")); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(desktop); !strings.Contains(string(got), `Exec="/opt2/Oddeye.AppImage"`) {
		t.Errorf("caminho não atualizado:\n%s", got)
	}
}

func TestEnsureSkipsPackagedInstall(t *testing.T) {
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	t.Setenv("APPIMAGE", "/usr/bin/oddeye")
	if err := Ensure([]byte("png")); err != nil {
		t.Fatal(err)
	}
	if entries, _ := os.ReadDir(data); len(entries) != 0 {
		t.Errorf("instalação por pacote não deveria gravar nada, gravou %v", entries)
	}
}
