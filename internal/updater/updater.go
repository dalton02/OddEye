// Package updater procura releases novas do Oddeye no GitHub, baixa o arquivo certo para o jeito como o
// app foi instalado, confere o SHA-256 e troca o app instalado pelo novo.
package updater

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Repo é o repositório do GitHub de onde saem as releases.
const Repo = "dalton02/OddEye"

const userAgent = "Oddeye-updater"

// ErrManual indica que o app não consegue se trocar sozinho (sem permissão na pasta, tipo de instalação
// desconhecido etc.); o usuário precisa baixar pela página do release.
var ErrManual = errors.New("não foi possível atualizar automaticamente")

// Kind é o jeito como o app foi instalado, que decide qual arquivo baixar e como trocar.
type Kind string

const (
	KindAppImage Kind = "appimage"
	KindDeb      Kind = "deb"
	KindRpm      Kind = "rpm"
	KindBinary   Kind = "binary" // binário solto no Linux
	KindWindows  Kind = "windows"
	KindMacApp   Kind = "macapp"
	KindUnknown  Kind = "unknown"
)

// NeedsPassword diz se a instalação vai pedir a senha do sistema (pacotes do Linux, via pkexec).
func (k Kind) NeedsPassword() bool { return k == KindDeb || k == KindRpm }

type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
}

// Release é a versão mais nova publicada no GitHub.
type Release struct {
	Version string  `json:"version"` // sem o "v": 0.1.1
	Notes   string  `json:"notes"`
	PageURL string  `json:"pageUrl"`
	Assets  []Asset `json:"-"`
}

type ghRelease struct {
	TagName    string  `json:"tag_name"`
	Body       string  `json:"body"`
	HTMLURL    string  `json:"html_url"`
	Draft      bool    `json:"draft"`
	Prerelease bool    `json:"prerelease"`
	Assets     []Asset `json:"assets"`
}

// Client fala com a API do GitHub. BaseURL existe para os testes apontarem para um servidor local.
type Client struct {
	HTTP    *http.Client
	BaseURL string
}

func NewClient() *Client {
	return &Client{HTTP: &http.Client{Timeout: 30 * time.Minute}, BaseURL: "https://api.github.com"}
}

// Latest devolve a release mais nova se ela for maior que current; nil quando já está atualizado.
func (c *Client) Latest(ctx context.Context, current string) (*Release, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/repos/"+Repo+"/releases/latest", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // ainda não existe release
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub respondeu %s", resp.Status)
	}
	var r ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}
	if r.Draft || r.Prerelease || !Newer(r.TagName, current) {
		return nil, nil
	}
	return &Release{Version: strings.TrimPrefix(r.TagName, "v"), Notes: strings.TrimSpace(r.Body), PageURL: r.HTMLURL, Assets: r.Assets}, nil
}

// Newer diz se a versão a é maior que b ("v0.1.10" > "0.1.9"). Sufixos como "-beta" são ignorados.
func Newer(a, b string) bool {
	pa, oka := parse(a)
	pb, okb := parse(b)
	if !oka || !okb {
		return false
	}
	for i := range pa {
		if pa[i] != pb[i] {
			return pa[i] > pb[i]
		}
	}
	return false
}

func parse(v string) ([3]int, bool) {
	var out [3]int
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

// PickAsset escolhe o arquivo do release para o tipo de instalação, pelos nomes que o workflow gera.
func PickAsset(kind Kind, version string, assets []Asset) (Asset, bool) {
	var want string
	switch kind {
	case KindAppImage:
		want = "Oddeye-" + version + "-x86_64.AppImage"
	case KindDeb:
		want = "oddeye_" + version + "_amd64.deb"
	case KindRpm:
		want = "oddeye-" + version + ".x86_64.rpm"
	case KindBinary:
		want = "oddeye-" + version + "-linux-amd64"
	case KindWindows:
		want = "oddeye-" + version + "-windows-portable.exe"
	case KindMacApp:
		want = "Oddeye-" + version + "-macos-universal.zip"
	default:
		return Asset{}, false
	}
	for _, a := range assets {
		if a.Name == want {
			return a, true
		}
	}
	return Asset{}, false
}

// Download baixa o arquivo do release para uma pasta temporária e confere o SHA-256 com o SHA256SUMS.txt
// do mesmo release. progress recebe bytes baixados e total (total pode ser 0 se desconhecido).
func (c *Client) Download(ctx context.Context, rel *Release, asset Asset, progress func(done, total int64)) (string, error) {
	sums, ok := findAsset(rel.Assets, "SHA256SUMS.txt")
	if !ok {
		return "", errors.New("o release não tem SHA256SUMS.txt")
	}
	sumData, err := c.fetch(ctx, sums.URL)
	if err != nil {
		return "", fmt.Errorf("baixando SHA256SUMS.txt: %w", err)
	}
	want, ok := ParseSums(string(sumData))[asset.Name]
	if !ok {
		return "", fmt.Errorf("SHA256SUMS.txt não lista %s", asset.Name)
	}

	dir, err := os.MkdirTemp("", "oddeye-update-")
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, asset.Name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.URL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download respondeu %s", resp.Status)
	}
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	total := resp.ContentLength
	if total <= 0 {
		total = asset.Size
	}
	pr := &progressReader{r: resp.Body, total: total, fn: progress}
	_, err = io.Copy(io.MultiWriter(f, h), pr)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return "", err
	}
	if got := hex.EncodeToString(h.Sum(nil)); !strings.EqualFold(got, want) {
		_ = os.RemoveAll(dir)
		return "", errors.New("o arquivo baixado não confere com o SHA256SUMS.txt; atualização cancelada")
	}
	return path, nil
}

func (c *Client) fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("respondeu %s", resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 1<<20))
}

func findAsset(assets []Asset, name string) (Asset, bool) {
	for _, a := range assets {
		if a.Name == name {
			return a, true
		}
	}
	return Asset{}, false
}

// ParseSums lê a saída do sha256sum ("<hash>  <arquivo>" por linha).
func ParseSums(text string) map[string]string {
	out := map[string]string{}
	sc := bufio.NewScanner(strings.NewReader(text))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) != 2 || len(fields[0]) != 64 {
			continue
		}
		out[strings.TrimPrefix(fields[1], "*")] = fields[0]
	}
	return out
}

type progressReader struct {
	r     io.Reader
	done  int64
	total int64
	last  time.Time
	fn    func(done, total int64)
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	p.done += int64(n)
	if p.fn != nil && (time.Since(p.last) > 100*time.Millisecond || err == io.EOF) {
		p.last = time.Now()
		p.fn(p.done, p.total)
	}
	return n, err
}

// replaceFile troca dst por src de forma atômica: copia para dst.new na mesma pasta e renomeia por cima.
func replaceFile(src, dst string, mode os.FileMode) error {
	tmp := dst + ".new"
	if err := copyFile(src, tmp, mode); err != nil {
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// manual transforma erro de permissão em ErrManual, para a UI oferecer o download pela página.
func manual(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, os.ErrPermission) {
		return fmt.Errorf("%w: %v", ErrManual, err)
	}
	return err
}

// Relaunch abre o executável (já atualizado) com --wait-pid apontando para este processo. O processo novo
// espera este sair antes de abrir a janela, senão o bloqueio de instância única o mandaria embora.
func Relaunch(exe string, extraArgs ...string) error {
	args := append([]string{WaitPIDFlag + "=" + strconv.Itoa(os.Getpid())}, extraArgs...)
	cmd := exec.Command(exe, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, nil, nil
	detach(cmd)
	return cmd.Start()
}

// WaitPIDFlag é o argumento que o app recebe ao ser reaberto depois de uma atualização.
const WaitPIDFlag = "--wait-pid"

// WaitForParent lê --wait-pid dos argumentos e espera o processo antigo terminar (até timeout).
func WaitForParent(args []string, timeout time.Duration) {
	for _, a := range args {
		if !strings.HasPrefix(a, WaitPIDFlag+"=") {
			continue
		}
		pid, err := strconv.Atoi(strings.TrimPrefix(a, WaitPIDFlag+"="))
		if err != nil || pid <= 0 {
			return
		}
		deadline := time.Now().Add(timeout)
		for time.Now().Before(deadline) && processAlive(pid) {
			time.Sleep(150 * time.Millisecond)
		}
		return
	}
}
