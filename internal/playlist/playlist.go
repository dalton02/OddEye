// Package playlist guarda as playlists do usuário em playlists.json. Uma faixa aponta para um
// arquivo da pasta local ou para um arquivo dentro de um torrent.
package playlist

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	SourceFolder  = "folder"
	SourceTorrent = "torrent"
	fileName      = "playlists.json"
)

type Track struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Source   string `json:"source"`
	Name     string `json:"name,omitempty"`     // pasta local
	InfoHash string `json:"infoHash,omitempty"` // torrent
	Path     string `json:"path,omitempty"`     // torrent
}

type Playlist struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Tracks []Track `json:"tracks"`
}

type Store struct {
	mu    sync.Mutex
	path  string
	lists []Playlist
}

// Open lê playlists.json de dir. Arquivo ausente ou corrompido começa vazio: perder as playlists
// é ruim, mas impedir o app de abrir é pior (o corrompido fica salvo ao lado como .bad).
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &Store{path: filepath.Join(dir, fileName), lists: []Playlist{}}
	data, err := os.ReadFile(s.path)
	if err == nil {
		var lists []Playlist
		if json.Unmarshal(data, &lists) == nil {
			s.lists = lists
		} else {
			log.Printf("playlists.json corrompido: começando vazio")
			_ = os.Rename(s.path, s.path+".bad")
		}
	}
	for i := range s.lists {
		if s.lists[i].Tracks == nil {
			s.lists[i].Tracks = []Track{}
		}
	}
	return s, nil
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Store) saveLocked() error {
	data, err := json.MarshalIndent(s.lists, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *Store) List() []Playlist {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Playlist, len(s.lists))
	for i, p := range s.lists {
		p.Tracks = append([]Track{}, p.Tracks...)
		out[i] = p
	}
	return out
}

func (s *Store) findLocked(id string) (int, error) {
	for i, p := range s.lists {
		if p.ID == id {
			return i, nil
		}
	}
	return -1, errors.New("playlist não encontrada")
}

func cleanName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("dê um nome para a playlist")
	}
	if len([]rune(name)) > 80 {
		name = string([]rune(name)[:80])
	}
	return name, nil
}

func (s *Store) Create(name string) (Playlist, error) {
	name, err := cleanName(name)
	if err != nil {
		return Playlist{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p := Playlist{ID: newID(), Name: name, Tracks: []Track{}}
	s.lists = append(s.lists, p)
	return p, s.saveLocked()
}

func (s *Store) Rename(id, name string) error {
	name, err := cleanName(name)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	i, err := s.findLocked(id)
	if err != nil {
		return err
	}
	s.lists[i].Name = name
	return s.saveLocked()
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	i, err := s.findLocked(id)
	if err != nil {
		return err
	}
	s.lists = append(s.lists[:i], s.lists[i+1:]...)
	return s.saveLocked()
}

func validTrack(t Track) error {
	switch t.Source {
	case SourceFolder:
		if t.Name == "" {
			return errors.New("faixa da pasta sem nome de arquivo")
		}
	case SourceTorrent:
		if len(t.InfoHash) != 40 || t.Path == "" {
			return errors.New("faixa de torrent incompleta")
		}
	default:
		return errors.New("origem de faixa desconhecida")
	}
	return nil
}

// AddTracks acrescenta faixas no fim, ignorando as que já estão na playlist.
func (s *Store) AddTracks(id string, tracks []Track) (Playlist, error) {
	for _, t := range tracks {
		if err := validTrack(t); err != nil {
			return Playlist{}, err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	i, err := s.findLocked(id)
	if err != nil {
		return Playlist{}, err
	}
	seen := map[string]bool{}
	for _, t := range s.lists[i].Tracks {
		seen[Key(t)] = true
	}
	for _, t := range tracks {
		if seen[Key(t)] {
			continue
		}
		seen[Key(t)] = true
		t.ID = newID()
		if t.Title == "" {
			t.Title = DefaultTitle(t)
		}
		s.lists[i].Tracks = append(s.lists[i].Tracks, t)
	}
	return s.lists[i], s.saveLocked()
}

func (s *Store) RemoveTrack(id, trackID string) (Playlist, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i, err := s.findLocked(id)
	if err != nil {
		return Playlist{}, err
	}
	tracks := s.lists[i].Tracks
	for j, t := range tracks {
		if t.ID == trackID {
			s.lists[i].Tracks = append(tracks[:j:j], tracks[j+1:]...)
			return s.lists[i], s.saveLocked()
		}
	}
	return Playlist{}, errors.New("faixa não encontrada")
}

// Move leva a faixa da posição from para a posição to.
func (s *Store) Move(id string, from, to int) (Playlist, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i, err := s.findLocked(id)
	if err != nil {
		return Playlist{}, err
	}
	tracks := s.lists[i].Tracks
	if from < 0 || from >= len(tracks) || to < 0 || to >= len(tracks) {
		return Playlist{}, errors.New("posição inválida")
	}
	t := tracks[from]
	tracks = append(tracks[:from], tracks[from+1:]...)
	tracks = append(tracks[:to], append([]Track{t}, tracks[to:]...)...)
	s.lists[i].Tracks = tracks
	return s.lists[i], s.saveLocked()
}

// Key identifica o arquivo para onde a faixa aponta (duas faixas com a mesma Key são a mesma música).
func Key(t Track) string {
	if t.Source == SourceTorrent {
		return "t:" + strings.ToLower(t.InfoHash) + ":" + t.Path
	}
	return "f:" + t.Name
}

// DefaultTitle é o nome do arquivo sem pasta nem extensão.
func DefaultTitle(t Track) string {
	name := t.Name
	if t.Source == SourceTorrent {
		name = t.Path
	}
	name = filepath.Base(filepath.ToSlash(name))
	return strings.TrimSuffix(name, filepath.Ext(name))
}
