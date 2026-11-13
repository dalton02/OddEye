// Package autostart liga ou desliga a abertura do app no login do usuário. O app sobe com --hidden,
// direto na bandeja, sem abrir a janela.
package autostart

import "os"

const (
	appName    = "Oddeye"
	appID      = "io.github.oddeye"
	HiddenFlag = "--hidden"
)

func executable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	// AppImage roda de um ponto de montagem temporário: o caminho estável é o do arquivo .AppImage.
	if appimage := os.Getenv("APPIMAGE"); appimage != "" {
		return appimage, nil
	}
	return exe, nil
}

// Set liga (enabled=true) ou desliga a abertura no login.
func Set(enabled bool) error {
	if enabled {
		return enable()
	}
	return disable()
}

func Enabled() bool { return enabled() }
