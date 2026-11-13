package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewer(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"v0.1.1", "0.1.0", true},
		{"0.1.10", "0.1.9", true},
		{"1.0.0", "0.9.9", true},
		{"0.1.0", "0.1.0", false},
		{"0.1.0", "0.2.0", false},
		{"v0.2.0-beta", "0.1.0", true},
		{"lixo", "0.1.0", false},
		{"0.1.1", "dev", false},
	}
	for _, c := range cases {
		if got := Newer(c.a, c.b); got != c.want {
			t.Errorf("Newer(%q, %q) = %v, esperado %v", c.a, c.b, got, c.want)
		}
	}
}

func TestPickAsset(t *testing.T) {
	assets := []Asset{
		{Name: "Oddeye-0.2.0-x86_64.AppImage"}, {Name: "oddeye_0.2.0_amd64.deb"}, {Name: "oddeye-0.2.0.x86_64.rpm"},
		{Name: "oddeye-0.2.0-linux-amd64"}, {Name: "oddeye-0.2.0-windows-portable.exe"}, {Name: "Oddeye-0.2.0-windows-setup.exe"},
		{Name: "Oddeye-0.2.0-macos-universal.zip"}, {Name: "Oddeye-0.2.0-macos-universal.dmg"}, {Name: "SHA256SUMS.txt"},
	}
	want := map[Kind]string{
		KindAppImage: "Oddeye-0.2.0-x86_64.AppImage",
		KindDeb:      "oddeye_0.2.0_amd64.deb",
		KindRpm:      "oddeye-0.2.0.x86_64.rpm",
		KindBinary:   "oddeye-0.2.0-linux-amd64",
		KindWindows:  "oddeye-0.2.0-windows-portable.exe",
		KindMacApp:   "Oddeye-0.2.0-macos-universal.zip",
	}
	for kind, name := range want {
		a, ok := PickAsset(kind, "0.2.0", assets)
		if !ok || a.Name != name {
			t.Errorf("PickAsset(%s) = %q, %v; esperado %q", kind, a.Name, ok, name)
		}
	}
	if _, ok := PickAsset(KindUnknown, "0.2.0", assets); ok {
		t.Errorf("instalação desconhecida não deveria escolher arquivo")
	}
}

func TestParseSums(t *testing.T) {
	a, b := strings.Repeat("a", 64), strings.Repeat("b", 64)
	sums := ParseSums(a + "  Oddeye-0.2.0-x86_64.AppImage\n" + b + " *oddeye_0.2.0_amd64.deb\n\nlinha quebrada\n")
	if sums["Oddeye-0.2.0-x86_64.AppImage"] != a || sums["oddeye_0.2.0_amd64.deb"] != b || len(sums) != 2 {
		t.Errorf("ParseSums = %v", sums)
	}
}

// fakeGitHub serve /releases/latest, o arquivo e o SHA256SUMS.txt (com o hash certo ou errado).
func fakeGitHub(t *testing.T, tag, payload string, goodSum bool) *httptest.Server {
	t.Helper()
	sum := sha256.Sum256([]byte(payload))
	hash := hex.EncodeToString(sum[:])
	if !goodSum {
		hash = strings.Repeat("0", 64)
	}
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/" + Repo + "/releases/latest":
			fmt.Fprintf(w, `{"tag_name":%q,"body":"Notas","html_url":"https://example/rel","assets":[
				{"name":"oddeye-0.2.0-linux-amd64","browser_download_url":"%s/bin","size":%d},
				{"name":"SHA256SUMS.txt","browser_download_url":"%s/sums"}]}`, tag, srv.URL, len(payload), srv.URL)
		case "/bin":
			_, _ = w.Write([]byte(payload))
		case "/sums":
			fmt.Fprintf(w, "%s  oddeye-0.2.0-linux-amd64\n", hash)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestLatestDownloadAndApply(t *testing.T) {
	srv := fakeGitHub(t, "v0.2.0", "binário novo", true)
	c := &Client{HTTP: srv.Client(), BaseURL: srv.URL}

	if rel, err := c.Latest(context.Background(), "0.2.0"); err != nil || rel != nil {
		t.Fatalf("mesma versão: rel = %v, err = %v; esperado nil", rel, err)
	}
	rel, err := c.Latest(context.Background(), "0.1.0")
	if err != nil || rel == nil || rel.Version != "0.2.0" || rel.Notes != "Notas" {
		t.Fatalf("Latest = %+v, %v", rel, err)
	}
	asset, ok := PickAsset(KindBinary, rel.Version, rel.Assets)
	if !ok {
		t.Fatal("asset do binário não encontrado")
	}
	var last int64
	file, err := c.Download(context.Background(), rel, asset, func(done, _ int64) { last = done })
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if last != int64(len("binário novo")) {
		t.Errorf("progresso final = %d", last)
	}

	installed := filepath.Join(t.TempDir(), "oddeye")
	if err := os.WriteFile(installed, []byte("binário velho"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := replaceFile(file, installed, 0o755); err != nil {
		t.Fatalf("replaceFile: %v", err)
	}
	if data, _ := os.ReadFile(installed); string(data) != "binário novo" {
		t.Errorf("conteúdo depois da troca = %q", data)
	}
	if _, err := os.Stat(installed + ".new"); !os.IsNotExist(err) {
		t.Errorf("sobrou o arquivo temporário .new")
	}
}

func TestDownloadRejectsBadChecksum(t *testing.T) {
	srv := fakeGitHub(t, "v0.2.0", "binário adulterado", false)
	c := &Client{HTTP: srv.Client(), BaseURL: srv.URL}
	rel, err := c.Latest(context.Background(), "0.1.0")
	if err != nil || rel == nil {
		t.Fatalf("Latest = %v, %v", rel, err)
	}
	asset, _ := PickAsset(KindBinary, rel.Version, rel.Assets)
	if _, err := c.Download(context.Background(), rel, asset, nil); err == nil {
		t.Fatal("checksum errado deveria cancelar o download")
	}
}

func TestLatestWithoutReleases(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	c := &Client{HTTP: srv.Client(), BaseURL: srv.URL}
	if rel, err := c.Latest(context.Background(), "0.1.0"); err != nil || rel != nil {
		t.Errorf("sem releases: rel = %v, err = %v; esperado nil, nil", rel, err)
	}
}
