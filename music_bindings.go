package main

import (
	"errors"
	"log"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"oddeye/internal/appdir"
	"oddeye/internal/archive"
	"oddeye/internal/playlist"
	"oddeye/internal/torrent"
)

const eventTorrentProgress = "torrent:progress"

var errNoTorrent = errors.New("o motor de torrent não iniciou; veja o log do app")

// newMusicServices abre playlists e o motor de torrent em <config>/oddeye. Uma falha no torrent
// (porta, disco) não impede o app de abrir: só desliga os downloads.
func newMusicServices() (*playlist.Store, *torrent.Engine, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return nil, nil, err
	}
	dir := filepath.Join(base, appdir.Name)
	lists, err := playlist.Open(dir)
	if err != nil {
		return nil, nil, err
	}
	engine, err := torrent.New(filepath.Join(dir, "torrents"))
	if err != nil {
		log.Printf("motor de torrent indisponível: %v", err)
		return lists, nil, nil
	}
	return lists, engine, nil
}

func (a *App) startMusic() {
	if a.torrents != nil {
		go a.torrents.Watch(func(list []torrent.Info) { runtime.EventsEmit(a.ctx, eventTorrentProgress, list) })
	}
}

func (a *App) stopMusic() {
	if a.torrents != nil {
		a.torrents.Close()
	}
}

func (a *App) SearchArchive(query string, page int) (archive.Result, error) {
	return a.archive.Search(a.ctx, query, page)
}

// AddArchiveItem adiciona o torrent oficial de um item do archive.org (sem baixar nenhuma faixa ainda).
func (a *App) AddArchiveItem(id, title string) (torrent.Info, error) {
	if a.torrents == nil {
		return torrent.Info{}, errNoTorrent
	}
	return a.torrents.Add(a.ctx, a.archive.TorrentURL(id), title)
}

// AddTorrent aceita um magnet ou uma URL de .torrent.
func (a *App) AddTorrent(src string) (torrent.Info, error) {
	if a.torrents == nil {
		return torrent.Info{}, errNoTorrent
	}
	return a.torrents.Add(a.ctx, src, "")
}

// ChooseTorrentFile abre o seletor de .torrent; devolve nil quando o usuário cancela.
func (a *App) ChooseTorrentFile() (*torrent.Info, error) {
	if a.torrents == nil {
		return nil, errNoTorrent
	}
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Abrir arquivo .torrent",
		Filters: []runtime.FileFilter{{DisplayName: "Torrent (*.torrent)", Pattern: "*.torrent"}},
	})
	if err != nil || path == "" {
		return nil, err
	}
	info, err := a.torrents.Add(a.ctx, path, "")
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func (a *App) ListTorrents() []torrent.Info {
	if a.torrents == nil {
		return []torrent.Info{}
	}
	return a.torrents.List()
}

func (a *App) TorrentFiles(hash string) ([]torrent.File, error) {
	if a.torrents == nil {
		return nil, errNoTorrent
	}
	return a.torrents.Files(hash)
}

func (a *App) DownloadTrack(hash, path string, urgent bool) (torrent.File, error) {
	if a.torrents == nil {
		return torrent.File{}, errNoTorrent
	}
	return a.torrents.Download(hash, path, urgent)
}

func (a *App) RemoveTorrent(hash string, deleteFiles bool) error {
	if a.torrents == nil {
		return errNoTorrent
	}
	return a.torrents.Remove(hash, deleteFiles)
}

func (a *App) ListPlaylists() []playlist.Playlist {
	return a.playlists.List()
}

func (a *App) CreatePlaylist(name string) (playlist.Playlist, error) {
	return a.playlists.Create(name)
}

func (a *App) RenamePlaylist(id, name string) error {
	return a.playlists.Rename(id, name)
}

func (a *App) DeletePlaylist(id string) error {
	return a.playlists.Delete(id)
}

func (a *App) AddToPlaylist(id string, tracks []playlist.Track) (playlist.Playlist, error) {
	return a.playlists.AddTracks(id, tracks)
}

func (a *App) RemoveFromPlaylist(id, trackID string) (playlist.Playlist, error) {
	return a.playlists.RemoveTrack(id, trackID)
}

func (a *App) MovePlaylistTrack(id string, from, to int) (playlist.Playlist, error) {
	return a.playlists.Move(id, from, to)
}
