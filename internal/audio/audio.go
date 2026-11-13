// Package audio diz se o webview consegue tocar som neste sistema. No Linux o WebKitGTK toca áudio
// pelo GStreamer, e os plugins que ele precisa vêm em pacotes separados que nem toda instalação tem
// (AppImage, instalação mínima, distro sem os "recomendados"). Sem eles o WebKit não toca nada e ainda
// enche o log de erros a cada tentativa, então o app confere antes e nem tenta.
package audio

// Status é o resultado da conferência. Missing e Hint só vêm preenchidos quando Available é false.
type Status struct {
	Available bool     `json:"available"`
	Missing   []string `json:"missing"` // o que falta, em texto para o usuário
	Hint      string   `json:"hint"`    // comando de instalação para a distro, ou "" se não soubermos
}
