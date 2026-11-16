//go:build !windows

package term

import (
	"encoding/base64"
	"strings"
	"sync"
	"testing"
	"time"
)

type recorder struct {
	mu     sync.Mutex
	out    strings.Builder
	exited chan struct{}
}

func newRecorder() *recorder { return &recorder{exited: make(chan struct{}, 1)} }

func (r *recorder) emit(name string, data ...any) {
	switch {
	case strings.HasPrefix(name, "term:data:"):
		b, _ := base64.StdEncoding.DecodeString(data[0].(Chunk).Data)
		r.mu.Lock()
		r.out.Write(b)
		r.mu.Unlock()
	case strings.HasPrefix(name, "term:exit:"):
		r.exited <- struct{}{}
	}
}

func (r *recorder) text() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.out.String()
}

func TestRunsInDirAndReportsExit(t *testing.T) {
	r := newRecorder()
	m := NewManager(r.emit)
	m.Program, m.Args = "sh", []string{"-c", "pwd; echo pronto"}
	dir := t.TempDir()

	snap, err := m.Open("a", dir, "", 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	if !snap.Started {
		t.Fatal("esperava processo novo")
	}
	select {
	case <-r.exited:
	case <-time.After(5 * time.Second):
		t.Fatal("o processo não terminou")
	}
	out := r.text()
	if !strings.Contains(out, "pronto") {
		t.Fatalf("saída inesperada: %q", out)
	}
	// reabrir uma sessão encerrada reinicia o programa mantendo o histórico
	snap, err = m.Open("a", dir, "", 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	hist, _ := base64.StdEncoding.DecodeString(snap.Scrollback)
	if !snap.Started || !strings.Contains(string(hist), "pronto") {
		t.Fatalf("reinício sem histórico: %+v %q", snap, hist)
	}
	m.CloseAll()
}

func TestWriteAndReopenAlive(t *testing.T) {
	r := newRecorder()
	m := NewManager(r.emit)
	m.Program, m.Args = "sh", []string{"-c", "read l; echo got:$l; sleep 5"}

	if _, err := m.Open("b", t.TempDir(), "", 80, 24); err != nil {
		t.Fatal(err)
	}
	if err := m.Write("b", "oi\r"); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for !strings.Contains(r.text(), "got:oi") {
		if time.Now().After(deadline) {
			t.Fatalf("eco não chegou: %q", r.text())
		}
		time.Sleep(20 * time.Millisecond)
	}
	snap, err := m.Open("b", "/nao-importa", "", 100, 30)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Started || snap.Seq == 0 {
		t.Fatalf("sessão viva deveria só devolver o histórico: %+v", snap)
	}
	m.Close("b")
	select {
	case <-r.exited:
		t.Fatal("fechar de propósito não deve emitir saída")
	case <-time.After(300 * time.Millisecond):
	}
}

func TestMissingDir(t *testing.T) {
	m := NewManager(func(string, ...any) {})
	m.Program = "sh"
	if _, err := m.Open("c", "/nao/existe/mesmo", "", 80, 24); err == nil {
		t.Fatal("esperava erro de pasta inexistente")
	}
}

func TestShellKindRunsUserShell(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	r := newRecorder()
	m := NewManager(r.emit)
	m.Program = "programa-que-nao-existe" // o shell não usa o Program
	dir := t.TempDir()

	if _, err := m.Open("d", dir, "shell", 80, 24); err != nil {
		t.Fatal(err)
	}
	if err := m.Write("d", "echo em:$PWD; exit\r"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-r.exited:
	case <-time.After(5 * time.Second):
		t.Fatalf("o shell não saiu: %q", r.text())
	}
	if !strings.Contains(r.text(), "em:"+dir) {
		t.Fatalf("o shell não rodou na pasta: %q", r.text())
	}
}
