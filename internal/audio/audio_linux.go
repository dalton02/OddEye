package audio

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Arquivos de plugin do GStreamer que o app procura. Os nomes são os mesmos em todas as distros.
const (
	corePlugin = "libgstcoreelements.so" // vem com o próprio GStreamer: marca uma pasta de plugins de verdade
	autoPlugin = "libgstautodetect.so"   // autoaudiosink, que o WebKit usa para toda saída de som
)

// sinkPlugins são as saídas de som que o autoaudiosink sabe escolher. Basta uma.
var sinkPlugins = []string{"libgstpulseaudio.so", "libgstpipewire.so", "libgstalsa.so", "libgstjack.so", "libgstoss4.so", "libgstossaudio.so"}

// Check procura os plugins de áudio do GStreamer nas pastas de plugin do sistema.
func Check() Status {
	return check(pluginDirs(), osRelease("/etc/os-release"))
}

// check só afirma que falta algo quando achou uma instalação do GStreamer e o plugin não está nela.
// Se não achar instalação nenhuma (layout que não conhecemos, GStreamer compilado num arquivo só),
// responde "disponível": desligar o som por engano é pior que um aviso no log.
func check(dirs []string, release map[string]string) Status {
	ok := Status{Available: true, Missing: []string{}}
	located := false
	for _, d := range dirs {
		if exists(filepath.Join(d, corePlugin)) {
			located = true
			break
		}
	}
	if !located {
		return ok
	}
	var missing []string
	if !anyHas(dirs, autoPlugin) {
		missing = append(missing, "o seletor de saída de áudio do GStreamer (autoaudiosink)")
	}
	if !anyHas(dirs, sinkPlugins...) {
		missing = append(missing, "uma saída de áudio do GStreamer (PulseAudio, PipeWire ou ALSA)")
	}
	if len(missing) == 0 {
		return ok
	}
	return Status{Missing: missing, Hint: installHint(release)}
}

func anyHas(dirs []string, names ...string) bool {
	for _, d := range dirs {
		for _, n := range names {
			if exists(filepath.Join(d, n)) {
				return true
			}
		}
	}
	return false
}

func exists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// pluginDirs junta todas as pastas onde o GStreamer deste sistema pode carregar plugins: as das
// variáveis de ambiente dele, a do usuário, a pasta ao lado da libgstreamer que este processo
// carregou e os caminhos padrão das distros.
func pluginDirs() []string {
	var dirs []string
	for _, env := range []string{"GST_PLUGIN_SYSTEM_PATH_1_0", "GST_PLUGIN_SYSTEM_PATH", "GST_PLUGIN_PATH_1_0", "GST_PLUGIN_PATH"} {
		dirs = append(dirs, filepath.SplitList(os.Getenv(env))...)
	}
	data := os.Getenv("XDG_DATA_HOME")
	if data == "" {
		if home, err := os.UserHomeDir(); err == nil {
			data = filepath.Join(home, ".local", "share")
		}
	}
	if data != "" {
		dirs = append(dirs, filepath.Join(data, "gstreamer-1.0", "plugins"))
	}
	dirs = append(dirs, loadedLibDirs("/proc/self/maps")...)
	for _, pattern := range []string{
		"/usr/lib/gstreamer-1.0",   // Arch, Alpine, Void
		"/usr/lib64/gstreamer-1.0", // Fedora, openSUSE, RHEL
		"/usr/lib/*/gstreamer-1.0", // Debian e Ubuntu (multiarch)
		"/usr/local/lib/gstreamer-1.0", "/usr/local/lib64/gstreamer-1.0", "/usr/local/lib/*/gstreamer-1.0",
		"/run/current-system/sw/lib/gstreamer-1.0", // NixOS
	} {
		if matches, err := filepath.Glob(pattern); err == nil {
			dirs = append(dirs, matches...)
		}
	}
	seen := map[string]bool{}
	out := []string{}
	for _, d := range dirs {
		if d == "" || seen[d] {
			continue
		}
		seen[d] = true
		out = append(out, d)
	}
	return out
}

// loadedLibDirs devolve a pasta de plugins ao lado da libgstreamer mapeada neste processo (o
// WebKitGTK já a carrega), que é a que vale mesmo em layouts fora do padrão.
func loadedLibDirs(maps string) []string {
	f, err := os.Open(maps)
	if err != nil {
		return nil
	}
	defer f.Close()
	var dirs []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 6 {
			continue
		}
		path := fields[len(fields)-1]
		if strings.HasPrefix(filepath.Base(path), "libgstreamer-1.0.so") {
			dirs = append(dirs, filepath.Join(filepath.Dir(path), "gstreamer-1.0"))
		}
	}
	return dirs
}

// osRelease lê os campos de /etc/os-release (ID, ID_LIKE), sem aspas.
func osRelease(path string) map[string]string {
	out := map[string]string{}
	data, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(data), "\n") {
		key, value, found := strings.Cut(strings.TrimSpace(line), "=")
		if found {
			out[key] = strings.Trim(value, `"'`)
		}
	}
	return out
}

// installHint devolve o comando que instala os plugins na distro, pelo ID e pelas distros-mãe (ID_LIKE).
func installHint(release map[string]string) string {
	ids := strings.Fields(strings.ToLower(release["ID"] + " " + release["ID_LIKE"]))
	for _, id := range ids {
		switch id {
		case "debian", "ubuntu":
			return "sudo apt install gstreamer1.0-plugins-good gstreamer1.0-pulseaudio"
		case "fedora", "rhel", "centos":
			return "sudo dnf install gstreamer1-plugins-good"
		case "suse", "opensuse", "opensuse-tumbleweed", "opensuse-leap", "sles":
			return "sudo zypper install gstreamer-plugins-good"
		case "arch":
			return "sudo pacman -S gst-plugins-good"
		case "alpine":
			return "sudo apk add gst-plugins-good"
		case "void":
			return "sudo xbps-install gst-plugins-good1"
		case "gentoo":
			return "sudo emerge media-libs/gst-plugins-good media-plugins/gst-plugins-pulse"
		}
	}
	return ""
}
