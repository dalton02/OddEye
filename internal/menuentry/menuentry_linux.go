// Package menuentry põe o app no menu de aplicativos quando ele roda de um AppImage ou de um binário
// solto, que não passam por instalador. O .deb, o .rpm, o instalador do Windows e o .app do macOS já
// cuidam disso sozinhos.
package menuentry

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Ensure grava (ou atualiza) a entrada do menu e o ícone na pasta de dados do usuário. É chamada a cada
// abertura porque o caminho do executável muda quando o AppImage é movido.
func Ensure(icon []byte) error {
	exe, err := executable()
	if err != nil {
		return err
	}
	// Instalado por pacote: o /usr/share/applications/oddeye.desktop já existe.
	if strings.HasPrefix(exe, "/usr/") || strings.HasPrefix(exe, "/opt/") {
		return nil
	}
	data, err := dataDir()
	if err != nil {
		return err
	}
	// Direto em icons/, fora de um tema: vale para qualquer tamanho e também resolve o "Icon=oddeye"
	// da entrada de autostart.
	iconPath := filepath.Join(data, "icons", "oddeye.png")
	if err := writeIfChanged(iconPath, icon); err != nil {
		return err
	}
	return writeIfChanged(filepath.Join(data, "applications", "oddeye.desktop"), []byte(entry(exe, iconPath)))
}

func entry(exe, icon string) string {
	quoted := `"` + strings.ReplaceAll(exe, `"`, `\"`) + `"`
	// TryExec: se o arquivo for apagado, o menu esconde a entrada em vez de deixar um atalho quebrado.
	return fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=Oddeye
GenericName=Revisor de pull requests
Comment=Avisa de PRs novas e faz o tour guiado com o Claude Code
TryExec=%s
Exec=%s
Icon=%s
Terminal=false
Categories=Development;
StartupWMClass=oddeye
`, exe, quoted, icon)
}

func executable() (string, error) {
	// AppImage roda de um ponto de montagem temporário: o caminho estável é o do arquivo .AppImage.
	if appimage := os.Getenv("APPIMAGE"); appimage != "" {
		return appimage, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	return exe, nil
}

func dataDir() (string, error) {
	if dir := os.Getenv("XDG_DATA_HOME"); filepath.IsAbs(dir) {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share"), nil
}

func writeIfChanged(path string, content []byte) error {
	if old, err := os.ReadFile(path); err == nil && bytes.Equal(old, content) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}
