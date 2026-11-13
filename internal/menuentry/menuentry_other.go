//go:build !linux

// Package menuentry põe o app no menu de aplicativos quando ele roda de um AppImage ou de um binário
// solto no Linux. No Windows e no macOS quem faz isso é o instalador e o bundle .app.
package menuentry

func Ensure([]byte) error { return nil }
