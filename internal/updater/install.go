package updater

// Install descreve onde o app está instalado.
type Install struct {
	Kind Kind
	Path string // o que é trocado: o AppImage, o binário, o .exe ou o bundle .app
	Exe  string // o executável a reabrir depois da troca
}
