package audio

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func pluginDir(t *testing.T, files ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestCheckReportsMissingAutoSink(t *testing.T) {
	dir := pluginDir(t, corePlugin, "libgstalsa.so")
	got := check([]string{dir}, map[string]string{"ID": "ubuntu", "ID_LIKE": "debian"})
	if got.Available || len(got.Missing) != 1 || !strings.Contains(got.Missing[0], "autoaudiosink") {
		t.Fatalf("check = %+v", got)
	}
	if !strings.HasPrefix(got.Hint, "sudo apt install") {
		t.Errorf("Hint = %q", got.Hint)
	}
}

func TestCheckReportsMissingOutput(t *testing.T) {
	dir := pluginDir(t, corePlugin, autoPlugin)
	got := check([]string{dir}, nil)
	if got.Available || len(got.Missing) != 1 || got.Hint != "" {
		t.Fatalf("check = %+v", got)
	}
}

func TestCheckFindsPluginsSpreadAcrossDirs(t *testing.T) {
	dirs := []string{pluginDir(t, corePlugin), pluginDir(t, autoPlugin), pluginDir(t, "libgstpipewire.so"), "/nao/existe"}
	if got := check(dirs, nil); !got.Available || got.Missing == nil {
		t.Fatalf("check = %+v", got)
	}
}

// Sem achar instalação do GStreamer não dá para afirmar que falta plugin: o som fica ligado.
func TestCheckAssumesAvailableWhenGStreamerNotLocated(t *testing.T) {
	for _, dirs := range [][]string{nil, {t.TempDir()}, {pluginDir(t, "libgstalsa.so")}} {
		if got := check(dirs, nil); !got.Available {
			t.Errorf("check(%v) = %+v", dirs, got)
		}
	}
}

func TestCheckIgnoresDirectoryNamedLikePlugin(t *testing.T) {
	dir := pluginDir(t, corePlugin, "libgstalsa.so")
	if err := os.Mkdir(filepath.Join(dir, autoPlugin), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := check([]string{dir}, nil); got.Available {
		t.Error("uma pasta com nome de plugin não é plugin")
	}
}

func TestInstallHintFollowsParentDistro(t *testing.T) {
	cases := map[string]map[string]string{
		"apt":     {"ID": "linuxmint", "ID_LIKE": "ubuntu debian"},
		"dnf":     {"ID": "nobara", "ID_LIKE": "fedora"},
		"zypper":  {"ID": "opensuse-tumbleweed", "ID_LIKE": "opensuse suse"},
		"pacman":  {"ID": "cachyos", "ID_LIKE": "arch"},
		"apk add": {"ID": "alpine"},
	}
	for want, release := range cases {
		if got := installHint(release); !strings.Contains(got, want) {
			t.Errorf("installHint(%v) = %q, want %q", release, got, want)
		}
	}
	if got := installHint(map[string]string{"ID": "nixos"}); got != "" {
		t.Errorf("distro desconhecida deveria ficar sem comando, veio %q", got)
	}
}

func TestOSReleaseStripsQuotes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "os-release")
	if err := os.WriteFile(path, []byte("NAME=\"Fedora Linux\"\nID=fedora\n# comentário\nID_LIKE='rhel centos'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := osRelease(path)
	if got["ID"] != "fedora" || got["ID_LIKE"] != "rhel centos" || got["NAME"] != "Fedora Linux" {
		t.Errorf("osRelease = %v", got)
	}
	if len(osRelease(filepath.Join(t.TempDir(), "nao-existe"))) != 0 {
		t.Error("arquivo ausente deveria dar mapa vazio")
	}
}

func TestLoadedLibDirsReadsProcMaps(t *testing.T) {
	path := filepath.Join(t.TempDir(), "maps")
	maps := "7f00-7f01 r--p 00000000 08:01 1 /usr/lib64/libgstreamer-1.0.so.0.2400.5\n" +
		"7f02-7f03 r--p 00000000 08:01 2 /usr/lib64/libglib-2.0.so.0\n7f04-7f05 rw-p 00000000 00:00 0\n"
	if err := os.WriteFile(path, []byte(maps), 0o644); err != nil {
		t.Fatal(err)
	}
	got := loadedLibDirs(path)
	if len(got) != 1 || got[0] != "/usr/lib64/gstreamer-1.0" {
		t.Errorf("loadedLibDirs = %v", got)
	}
}
