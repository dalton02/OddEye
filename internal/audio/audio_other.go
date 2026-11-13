//go:build !linux

package audio

// Check fora do Linux: WebView2 e WKWebView tocam áudio com o que o próprio sistema traz.
func Check() Status {
	return Status{Available: true, Missing: []string{}}
}
