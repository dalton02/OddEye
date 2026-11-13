package autostart

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func desktopFile() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "autostart", "oddeye.desktop"), nil
}

func enable() error {
	exe, err := executable()
	if err != nil {
		return err
	}
	path, err := desktopFile()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	quoted := `"` + strings.ReplaceAll(exe, `"`, `\"`) + `"`
	content := fmt.Sprintf("[Desktop Entry]\nType=Application\nName=%s\nExec=%s %s\nIcon=oddeye\nX-GNOME-Autostart-enabled=true\nTerminal=false\n", appName, quoted, HiddenFlag)
	return os.WriteFile(path, []byte(content), 0o644)
}

func disable() error {
	path, err := desktopFile()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func enabled() bool {
	path, err := desktopFile()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}
