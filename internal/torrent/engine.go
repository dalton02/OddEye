// Package torrent baixa músicas por BitTorrent. Só os arquivos que o usuário escolhe são baixados,
// cada torrent fica em <dir>/<infohash>/ e o seeding para quando o download termina, para o app de
// bandeja não ficar consumindo upload em segundo plano.
package torrent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	at "github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/storage"

	"oddeye/internal/music"
)

const (
	metadataTimeout = 90 * time.Second
	maxTorrentFile  = 20 << 20
	stateFile       = "torrents.json"
)

// Info resume um torrent para a aba Downloads. Size e Done somam só os arquivos escolhidos.
type Info struct {
	InfoHash string `json:"infoHash"`
	Name     string `json:"name"`
	Title    string `json:"title"`
	Size     int64  `json:"size"`
	Done     int64  `json:"done"`
	Wanted   int    `json:"wanted"`
	Complete int    `json:"complete"`
	Rate     int64  `json:"rate"` // bytes/s
	Peers    int    `json:"peers"`
}

// File é um arquivo de áudio do torrent. Path é o caminho dentro do torrent, sem o nome da raiz.
type File struct {
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	Done     int64  `json:"done"`
	Complete bool   `json:"complete"`
	Wanted   bool   `json:"wanted"`
}

type saved struct {
	InfoHash string   `json:"infoHash"`
	Title    string   `json:"title"`
	Wanted   []string `json:"wanted"`
}

type Engine struct {
	cl      *at.Client
	storage storage.ClientImplCloser
	dir     string
	http    *http.Client

	mu      sync.Mutex
	state   map[string]*saved // por infohash
	lastRx  map[string]int64
	rate    map[string]int64
	active  bool
	closing chan struct{}
}

// New abre o motor com dados em dir e retoma os torrents salvos.
func New(dir string) (*Engine, error) {
	if err := os.MkdirAll(filepath.Join(dir, "meta"), 0o755); err != nil {
		return nil, err
	}
	st := storage.NewFileOpts(storage.NewFileClientOpts{
		ClientBaseDir: dir,
		TorrentDirMaker: func(base string, _ *metainfo.Info, ih metainfo.Hash) string {
			return filepath.Join(base, ih.HexString())
		},
	})
	cfg := at.NewDefaultClientConfig()
	cfg.DefaultStorage = st
	cfg.DataDir = dir
	cfg.ListenPort = 0 // porta livre qualquer: duas instâncias ou outro cliente não brigam pela 42069
	cfg.Seed = false
	cfg.NoDefaultPortForwarding = true
	cl, err := at.NewClient(cfg)
	if err != nil {
		st.Close()
		return nil, err
	}
	e := &Engine{
		cl: cl, storage: st, dir: dir,
		http:  &http.Client{Timeout: 30 * time.Second},
		state: map[string]*saved{}, lastRx: map[string]int64{}, rate: map[string]int64{},
		closing: make(chan struct{}),
	}
	e.restore()
	return e, nil
}

func (e *Engine) Close() {
	close(e.closing)
	e.cl.Close()
	e.storage.Close()
}

func (e *Engine) restore() {
	data, err := os.ReadFile(filepath.Join(e.dir, stateFile))
	if err != nil {
		return
	}
	var list []saved
	if json.Unmarshal(data, &list) != nil {
		log.Printf("torrents.json corrompido: começando sem downloads")
		return
	}
	for _, s := range list {
		t, err := e.cl.AddTorrentFromFile(e.metaPath(s.InfoHash))
		if err != nil {
			log.Printf("não consegui retomar o torrent %s: %v", s.InfoHash, err)
			continue
		}
		s := s
		e.state[s.InfoHash] = &s
		<-t.GotInfo() // veio do .torrent salvo: a info já está presente
		for _, f := range audioFiles(t) {
			for _, w := range s.Wanted {
				if f.DisplayPath() == w {
					f.Download()
				}
			}
		}
	}
}

func (e *Engine) metaPath(hash string) string {
	return filepath.Join(e.dir, "meta", hash+".torrent")
}

// persist grava o estado num temporário e renomeia; chamar com e.mu travado.
func (e *Engine) persist() {
	list := make([]saved, 0, len(e.state))
	for _, s := range e.state {
		list = append(list, *s)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].InfoHash < list[j].InfoHash })
	data, _ := json.MarshalIndent(list, "", "  ")
	tmp := filepath.Join(e.dir, stateFile+".tmp")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		log.Printf("não consegui salvar os torrents: %v", err)
		return
	}
	_ = os.Rename(tmp, filepath.Join(e.dir, stateFile))
}

// Add aceita um magnet, uma URL http(s) de .torrent ou o caminho de um .torrent local. Espera os
// metadados chegarem; nenhum arquivo começa a baixar até alguém chamar Download.
func (e *Engine) Add(ctx context.Context, src, title string) (Info, error) {
	src = strings.TrimSpace(src)
	var (
		t   *at.Torrent
		err error
	)
	switch {
	case strings.HasPrefix(src, "magnet:"):
		t, err = e.cl.AddMagnet(src)
	case strings.HasPrefix(src, "http://"), strings.HasPrefix(src, "https://"):
		var mi *metainfo.MetaInfo
		if mi, err = e.fetchTorrent(ctx, src); err == nil {
			t, err = e.cl.AddTorrent(mi)
		}
	case src == "":
		return Info{}, errors.New("informe um magnet ou um arquivo .torrent")
	default:
		t, err = e.cl.AddTorrentFromFile(src)
	}
	if err != nil {
		return Info{}, fmt.Errorf("torrent inválido: %w", err)
	}

	hash := t.InfoHash().HexString()
	ctx, cancel := context.WithTimeout(ctx, metadataTimeout)
	defer cancel()
	select {
	case <-t.GotInfo():
	case <-ctx.Done():
		e.mu.Lock()
		_, known := e.state[hash]
		e.mu.Unlock()
		if !known {
			t.Drop()
		}
		return Info{}, errors.New("os metadados do torrent não chegaram a tempo (sem peers?)")
	}
	if len(audioFiles(t)) == 0 {
		e.mu.Lock()
		_, known := e.state[hash]
		e.mu.Unlock()
		if !known {
			t.Drop()
		}
		return Info{}, errors.New("esse torrent não tem arquivos de áudio")
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.state[hash]; !ok {
		if err := writeMeta(e.metaPath(hash), t.Metainfo()); err != nil {
			return Info{}, err
		}
		if title == "" {
			title = t.Name()
		}
		e.state[hash] = &saved{InfoHash: hash, Title: title, Wanted: []string{}}
		e.persist()
	}
	return e.infoLocked(t), nil
}

func (e *Engine) fetchTorrent(ctx context.Context, u string) (*metainfo.MetaInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "oddeye")
	resp, err := e.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("não consegui baixar o .torrent: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("o .torrent respondeu HTTP %d", resp.StatusCode)
	}
	return metainfo.Load(io.LimitReader(resp.Body, maxTorrentFile))
}

func writeMeta(path string, mi metainfo.MetaInfo) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := mi.Write(f); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func (e *Engine) torrent(hash string) (*at.Torrent, error) {
	var ih metainfo.Hash
	if err := ih.FromHexString(hash); err != nil {
		return nil, errors.New("infohash inválido")
	}
	t, ok := e.cl.Torrent(ih)
	if !ok || t.Info() == nil {
		return nil, errors.New("torrent não encontrado")
	}
	return t, nil
}

func audioFiles(t *at.Torrent) []*at.File {
	var out []*at.File
	for _, f := range t.Files() {
		if music.IsAudio(f.DisplayPath()) {
			out = append(out, f)
		}
	}
	return out
}

func findFile(t *at.Torrent, path string) *at.File {
	for _, f := range audioFiles(t) {
		if f.DisplayPath() == path {
			return f
		}
	}
	return nil
}

// Files lista os arquivos de áudio do torrent em ordem de caminho.
func (e *Engine) Files(hash string) ([]File, error) {
	t, err := e.torrent(hash)
	if err != nil {
		return nil, err
	}
	e.mu.Lock()
	wanted := map[string]bool{}
	if s := e.state[hash]; s != nil {
		for _, w := range s.Wanted {
			wanted[w] = true
		}
	}
	e.mu.Unlock()
	out := []File{}
	for _, f := range audioFiles(t) {
		done := f.BytesCompleted()
		out = append(out, File{Path: f.DisplayPath(), Size: f.Length(), Done: done, Complete: done == f.Length(), Wanted: wanted[f.DisplayPath()]})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// Download marca o arquivo para baixar. urgent é para a faixa que o usuário quer ouvir agora.
func (e *Engine) Download(hash, path string, urgent bool) (File, error) {
	t, err := e.torrent(hash)
	if err != nil {
		return File{}, err
	}
	f := findFile(t, path)
	if f == nil {
		return File{}, errors.New("arquivo não encontrado no torrent")
	}
	if urgent {
		f.SetPriority(at.PiecePriorityHigh)
	} else if f.Priority() == at.PiecePriorityNone {
		f.Download()
	}
	e.mu.Lock()
	s := e.state[hash]
	if s == nil {
		s = &saved{InfoHash: hash, Title: t.Name(), Wanted: []string{}}
		e.state[hash] = s
	}
	known := false
	for _, w := range s.Wanted {
		known = known || w == path
	}
	if !known {
		s.Wanted = append(s.Wanted, path)
		e.persist()
	}
	e.mu.Unlock()
	done := f.BytesCompleted()
	return File{Path: path, Size: f.Length(), Done: done, Complete: done == f.Length(), Wanted: true}, nil
}

// Remove tira o torrent do app; com deleteFiles, apaga também o que já foi baixado.
func (e *Engine) Remove(hash string, deleteFiles bool) error {
	var ih metainfo.Hash
	if err := ih.FromHexString(hash); err != nil {
		return errors.New("infohash inválido")
	}
	if t, ok := e.cl.Torrent(ih); ok {
		t.Drop()
	}
	e.mu.Lock()
	delete(e.state, hash)
	delete(e.lastRx, hash)
	delete(e.rate, hash)
	e.persist()
	e.mu.Unlock()
	_ = os.Remove(e.metaPath(hash))
	if deleteFiles {
		/* o diretório é sempre <dir>/<infohash hex>, validado acima: não há como sair da pasta */
		return os.RemoveAll(filepath.Join(e.dir, ih.HexString()))
	}
	return nil
}

// List devolve os torrents conhecidos, em ordem de título.
func (e *Engine) List() []Info {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := []Info{}
	for hash := range e.state {
		t, err := e.torrent(hash)
		if err != nil {
			continue
		}
		out = append(out, e.infoLocked(t))
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Title) < strings.ToLower(out[j].Title) })
	return out
}

func (e *Engine) infoLocked(t *at.Torrent) Info {
	hash := t.InfoHash().HexString()
	info := Info{InfoHash: hash, Name: t.Name(), Title: t.Name(), Rate: e.rate[hash]}
	wanted := map[string]bool{}
	if s := e.state[hash]; s != nil {
		info.Title = s.Title
		for _, w := range s.Wanted {
			wanted[w] = true
		}
	}
	for _, f := range audioFiles(t) {
		if !wanted[f.DisplayPath()] {
			continue
		}
		done := f.BytesCompleted()
		info.Wanted++
		info.Size += f.Length()
		info.Done += done
		if done == f.Length() {
			info.Complete++
		}
	}
	info.Peers = t.Stats().ActivePeers
	return info
}

// Watch chama emit a cada segundo enquanto houver download em andamento, e uma última vez quando
// tudo termina, para a interface mostrar 100%.
func (e *Engine) Watch(emit func([]Info)) {
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		select {
		case <-e.closing:
			return
		case <-tick.C:
		}
		e.mu.Lock()
		for hash := range e.state {
			t, err := e.torrent(hash)
			if err != nil {
				continue
			}
			st := t.Stats()
			rx := st.BytesReadUsefulData.Int64()
			if prev, ok := e.lastRx[hash]; ok && rx >= prev {
				e.rate[hash] = rx - prev
			}
			e.lastRx[hash] = rx
		}
		wasActive := e.active
		e.mu.Unlock()

		list := e.List()
		active := false
		for _, i := range list {
			active = active || i.Done < i.Size
		}
		e.mu.Lock()
		e.active = active
		e.mu.Unlock()
		if active || wasActive {
			emit(list)
		}
	}
}

// Open abre um arquivo de áudio já completo para leitura.
func (e *Engine) Open(hash, path string) (io.ReadSeekCloser, string, error) {
	t, err := e.torrent(hash)
	if err != nil {
		return nil, "", err
	}
	f := findFile(t, path)
	if f == nil {
		return nil, "", errors.New("arquivo não encontrado")
	}
	if f.BytesCompleted() != f.Length() {
		return nil, "", errors.New("arquivo ainda baixando")
	}
	r := f.NewReader()
	return r, filepath.Base(path), nil
}
