package playlist

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func folder(name string) Track { return Track{Source: SourceFolder, Name: name} }

func titles(p Playlist) string {
	var out []string
	for _, t := range p.Tracks {
		out = append(out, t.Title)
	}
	return strings.Join(out, ",")
}

func TestCRUDAndPersistence(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create("  "); err == nil {
		t.Fatal("nome vazio deveria falhar")
	}
	p, err := s.Create(" Foco ")
	if err != nil || p.Name != "Foco" {
		t.Fatalf("Create = %+v, %v", p, err)
	}
	p, err = s.AddTracks(p.ID, []Track{folder("a.mp3"), folder("b.ogg"), folder("a.mp3")})
	if err != nil || titles(p) != "a,b" {
		t.Fatalf("AddTracks = %q, %v", titles(p), err)
	}
	torrentTrack := Track{Source: SourceTorrent, InfoHash: strings.Repeat("a", 40), Path: "disc/03 c.flac", Title: "Faixa C"}
	if p, err = s.AddTracks(p.ID, []Track{torrentTrack}); err != nil || titles(p) != "a,b,Faixa C" {
		t.Fatalf("AddTracks torrent = %q, %v", titles(p), err)
	}
	if _, err := s.AddTracks(p.ID, []Track{{Source: SourceTorrent, InfoHash: "curto", Path: "x"}}); err == nil {
		t.Fatal("faixa inválida deveria falhar")
	}
	if err := s.Rename(p.ID, "Trabalho"); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	lists := reopened.List()
	if len(lists) != 1 || lists[0].Name != "Trabalho" || titles(lists[0]) != "a,b,Faixa C" {
		t.Fatalf("persistido = %+v", lists)
	}

	if err := reopened.Delete(p.ID); err != nil || len(reopened.List()) != 0 {
		t.Fatalf("Delete: %v", err)
	}
	if err := reopened.Delete(p.ID); err == nil {
		t.Fatal("apagar de novo deveria falhar")
	}
}

func TestMoveAndRemove(t *testing.T) {
	s, _ := Open(t.TempDir())
	p, _ := s.Create("x")
	p, _ = s.AddTracks(p.ID, []Track{folder("1.mp3"), folder("2.mp3"), folder("3.mp3"), folder("4.mp3")})

	p, err := s.Move(p.ID, 0, 3)
	if err != nil || titles(p) != "2,3,4,1" {
		t.Fatalf("Move 0→3 = %q, %v", titles(p), err)
	}
	p, _ = s.Move(p.ID, 3, 1)
	if titles(p) != "2,1,3,4" {
		t.Fatalf("Move 3→1 = %q", titles(p))
	}
	if _, err := s.Move(p.ID, 0, 9); err == nil {
		t.Fatal("posição fora deveria falhar")
	}
	p, err = s.RemoveTrack(p.ID, p.Tracks[1].ID)
	if err != nil || titles(p) != "2,3,4" {
		t.Fatalf("RemoveTrack = %q, %v", titles(p), err)
	}
	/* a lista devolvida não pode compartilhar memória com a interna */
	listed := s.List()
	listed[0].Tracks[0].Title = "mexido"
	if titles(s.List()[0]) != "2,3,4" {
		t.Fatal("List expôs o estado interno")
	}
}

func TestCorruptFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, fileName), []byte("{nada"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Open(dir)
	if err != nil || len(s.List()) != 0 {
		t.Fatalf("Open corrompido: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, fileName+".bad")); err != nil {
		t.Fatal("o arquivo corrompido deveria ter sido guardado como .bad")
	}
}

func TestDefaultTitle(t *testing.T) {
	if got := DefaultTitle(Track{Source: SourceTorrent, Path: "cd1/01 - Intro.flac"}); got != "01 - Intro" {
		t.Fatal(got)
	}
	if got := DefaultTitle(folder("musica.mp3")); got != "musica" {
		t.Fatal(got)
	}
}
