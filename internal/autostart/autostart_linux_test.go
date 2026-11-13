package autostart

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLinuxAutostartWritesHiddenDesktopEntry(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPIMAGE", "/opt/apps/Oddeye.AppImage")
	if err := Set(true); err != nil || !Enabled() {
		t.Fatalf("Set(true) = %v, Enabled = %v", err, Enabled())
	}
	data, _ := os.ReadFile(filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "autostart", "oddeye.desktop"))
	if !strings.Contains(string(data), `Exec="/opt/apps/Oddeye.AppImage" --hidden`) {
		t.Errorf("entrada sem o caminho estável do AppImage ou sem --hidden:\n%s", data)
	}
	if err := Set(false); err != nil || Enabled() {
		t.Errorf("Set(false) = %v, Enabled = %v", err, Enabled())
	}
	if err := Set(false); err != nil {
		t.Errorf("desligar duas vezes não deveria falhar: %v", err)
	}
}
