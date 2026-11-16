// Package term roda programas interativos (o Claude Code ou o shell do usuário) num pseudo-terminal, um por aba da tela
// Coding. A saída vai ao frontend por eventos; as sessões vivem aqui e sobrevivem ao frontend
// desmontar o terminal (trocar de aba da home, abrir uma PR), que ao voltar pede o histórico.
package term

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/aymanbagabas/go-pty"
)

// scrollbackMax é quanto da saída recente cada sessão guarda para redesenhar o terminal ao remontar.
const scrollbackMax = 512 * 1024

// Emit manda um evento ao frontend (runtime.EventsEmit no app).
type Emit func(name string, data ...any)

// Chunk é um pedaço da saída do terminal, em base64 (bytes crus: o xterm decodifica o UTF-8).
// Seq cresce a cada pedaço, para o frontend descartar o que já veio no histórico de Open.
type Chunk struct {
	Seq  uint64 `json:"seq"`
	Data string `json:"data"`
}

// Snapshot é o estado de uma sessão ao abrir o terminal: a saída guardada até Seq.
type Snapshot struct {
	Seq        uint64 `json:"seq"`
	Scrollback string `json:"scrollback"` // base64
	Started    bool   `json:"started"`    // true quando o processo acabou de ser iniciado
}

type session struct {
	mu     sync.Mutex
	pty    pty.Pty
	cmd    *pty.Cmd
	alive  bool
	seq    uint64
	buf    []byte
	closed bool // fechada de propósito: o fim do processo não vira evento de saída
}

type Manager struct {
	// Program e Args são o que roda em cada sessão (por padrão, `claude`). Trocados nos testes.
	Program string
	Args    []string

	emit     Emit
	mu       sync.Mutex
	sessions map[string]*session
}

func NewManager(emit Emit) *Manager {
	return &Manager{Program: "claude", emit: emit, sessions: map[string]*session{}}
}

// Open liga o terminal da aba id à pasta dir. Com a sessão viva, só ajusta o tamanho e devolve o
// histórico; sem sessão (ou com o processo encerrado), inicia o programa de novo.
// kind é "shell" para o shell do usuário (como o Konsole abre) ou qualquer outro valor para o Program.
func (m *Manager) Open(id, dir, kind string, cols, rows int) (Snapshot, error) {
	m.mu.Lock()
	s := m.sessions[id]
	m.mu.Unlock()
	if s != nil {
		s.mu.Lock()
		if s.alive {
			_ = s.pty.Resize(cols, rows)
			snap := Snapshot{Seq: s.seq, Scrollback: base64.StdEncoding.EncodeToString(s.buf)}
			s.mu.Unlock()
			return snap, nil
		}
		s.mu.Unlock()
	}
	return m.start(id, dir, kind, cols, rows, s)
}

// ShellProgram é o shell do usuário: $SHELL, ou bash/sh quando ele não existe.
func ShellProgram() string {
	for _, c := range []string{os.Getenv("SHELL"), "/bin/bash", "/bin/sh"} {
		if c == "" {
			continue
		}
		if p, err := exec.LookPath(c); err == nil {
			return p
		}
	}
	return "/bin/sh"
}

func (m *Manager) start(id, dir, kind string, cols, rows int, prev *session) (Snapshot, error) {
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return Snapshot{}, fmt.Errorf("a pasta %s não existe mais", dir)
	}
	program, args := m.Program, m.Args
	if kind == "shell" {
		program, args = ShellProgram(), nil
	}
	bin, err := exec.LookPath(program)
	if err != nil {
		return Snapshot{}, fmt.Errorf("%s não encontrado no PATH", program)
	}
	p, err := pty.New()
	if err != nil {
		return Snapshot{}, fmt.Errorf("não consegui abrir um terminal: %w", err)
	}
	if cols > 0 && rows > 0 {
		_ = p.Resize(cols, rows)
	}
	cmd := p.Command(bin, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor")
	if err := cmd.Start(); err != nil {
		_ = p.Close()
		return Snapshot{}, fmt.Errorf("não consegui iniciar o %s: %w", program, err)
	}

	s := &session{pty: p, cmd: cmd, alive: true}
	if prev != nil {
		// reinício: o histórico anterior continua na tela, a sequência segue de onde parou
		prev.mu.Lock()
		s.seq, s.buf = prev.seq, prev.buf
		prev.mu.Unlock()
	}
	m.mu.Lock()
	m.sessions[id] = s
	m.mu.Unlock()

	snap := Snapshot{Seq: s.seq, Scrollback: base64.StdEncoding.EncodeToString(s.buf), Started: true}
	go m.pump(id, s)
	go m.wait(id, s)
	return snap, nil
}

func (m *Manager) pump(id string, s *session) {
	buf := make([]byte, 32*1024)
	for {
		n, err := s.pty.Read(buf)
		if n > 0 {
			s.mu.Lock()
			s.seq++
			s.buf = append(s.buf, buf[:n]...)
			if over := len(s.buf) - scrollbackMax; over > 0 {
				s.buf = append([]byte(nil), s.buf[over:]...)
			}
			c := Chunk{Seq: s.seq, Data: base64.StdEncoding.EncodeToString(buf[:n])}
			s.mu.Unlock()
			m.emit("term:data:"+id, c)
		}
		if err != nil {
			return
		}
	}
}

// wait espera o processo sair e fecha o PTY. No Unix o lado escravo continua aberto no app, então a
// leitura não termina sozinha: fechar o PTY é o que solta o pump.
func (m *Manager) wait(id string, s *session) {
	_ = s.cmd.Wait()
	time.Sleep(100 * time.Millisecond) // deixa o pump ler o que o processo escreveu por último
	s.mu.Lock()
	s.alive = false
	closed := s.closed
	s.mu.Unlock()
	_ = s.pty.Close()
	if !closed {
		m.emit("term:exit:" + id)
	}
}

func (m *Manager) get(id string) (*session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.sessions[id]
	if s == nil {
		return nil, errors.New("sessão de terminal não encontrada")
	}
	return s, nil
}

func (m *Manager) Write(id string, data string) error {
	s, err := m.get(id)
	if err != nil {
		return err
	}
	s.mu.Lock()
	alive := s.alive
	s.mu.Unlock()
	if !alive {
		return nil
	}
	_, err = s.pty.Write([]byte(data))
	return err
}

func (m *Manager) Resize(id string, cols, rows int) error {
	if cols <= 0 || rows <= 0 {
		return nil
	}
	s, err := m.get(id)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.alive {
		return nil
	}
	return s.pty.Resize(cols, rows)
}

// Close encerra o processo da aba e esquece a sessão.
func (m *Manager) Close(id string) {
	m.mu.Lock()
	s := m.sessions[id]
	delete(m.sessions, id)
	m.mu.Unlock()
	if s != nil {
		s.kill()
	}
}

func (m *Manager) CloseAll() {
	m.mu.Lock()
	all := m.sessions
	m.sessions = map[string]*session{}
	m.mu.Unlock()
	for _, s := range all {
		s.kill()
	}
}

func (s *session) kill() {
	s.mu.Lock()
	s.closed = true
	alive := s.alive
	s.mu.Unlock()
	if alive && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
}
