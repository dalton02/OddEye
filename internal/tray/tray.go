// Package tray mostra o ícone do app na bandeja do sistema, com as PRs novas no menu e um ponto
// vermelho no ícone enquanto houver PR pendente.
package tray

import (
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"oddeye/internal/gh"
	"oddeye/internal/watcher"
)

// MaxPRs é quantas PRs pendentes cabem no menu; o resto fica na caixa de entrada do app.
const MaxPRs = 8

const (
	idStatus   = 0
	idFirstPR  = 1
	idOpen     = 20
	idCheck    = 21
	idSettings = 22
	idQuit     = 23
	separator  = -1
)

type Callbacks struct {
	OnOpen     func()
	OnCheckNow func()
	OnSettings func()
	OnQuit     func()
	OnPR       func(gh.OpenPR)
}

// menuItem é uma entrada do menu: o backend de cada SO só sabe criar, renomear e esconder itens.
type menuItem struct {
	id      int
	title   string
	visible bool
}

type backend interface {
	start(items []menuItem, onClick func(id int))
	setItem(id int, title string, visible bool)
	setIcon(dot bool)
	setTooltip(text string)
	stop()
}

type Tray struct {
	b  backend
	cb Callbacks

	mu      sync.Mutex
	pending []gh.OpenPR
}

func layout() []menuItem {
	items := []menuItem{{id: idStatus, title: "Nenhuma PR nova", visible: true}, {id: separator}}
	for i := 0; i < MaxPRs; i++ {
		items = append(items, menuItem{id: idFirstPR + i})
	}
	return append(items,
		menuItem{id: separator},
		menuItem{id: idOpen, title: "Abrir Oddeye", visible: true},
		menuItem{id: idCheck, title: "Verificar agora", visible: true},
		menuItem{id: idSettings, title: "Configurações", visible: true},
		menuItem{id: separator},
		menuItem{id: idQuit, title: "Sair", visible: true},
	)
}

// Start cria o ícone na bandeja. As callbacks rodam fora da thread de UI.
func Start(cb Callbacks) *Tray {
	t := &Tray{b: newBackend(), cb: cb}
	t.b.start(layout(), t.clicked)
	t.b.setTooltip("Oddeye")
	return t
}

func (t *Tray) Stop() { t.b.stop() }

func (t *Tray) clicked(id int) {
	switch {
	case id == idOpen:
		t.cb.OnOpen()
	case id == idCheck:
		t.cb.OnCheckNow()
	case id == idSettings:
		t.cb.OnSettings()
	case id == idQuit:
		t.cb.OnQuit()
	case id >= idFirstPR && id < idFirstPR+MaxPRs:
		t.mu.Lock()
		i := id - idFirstPR
		var p *gh.OpenPR
		if i < len(t.pending) {
			p = &t.pending[i]
		}
		t.mu.Unlock()
		if p != nil {
			t.cb.OnPR(*p)
		}
	}
}

// Update reflete o estado do watcher no menu e no ícone.
func (t *Tray) Update(s watcher.Snapshot) {
	t.mu.Lock()
	t.pending = append([]gh.OpenPR(nil), s.Pending...)
	t.mu.Unlock()

	status := StatusLine(s)
	t.b.setItem(idStatus, status, true)
	t.b.setTooltip("Oddeye · " + status)
	for i := 0; i < MaxPRs; i++ {
		if i < len(s.Pending) {
			t.b.setItem(idFirstPR+i, PRLabel(s.Pending[i]), true)
		} else {
			t.b.setItem(idFirstPR+i, "", false)
		}
	}
	t.b.setIcon(len(s.Pending) > 0)
}

// StatusLine é a primeira linha do menu: o que o usuário precisa saber de relance.
func StatusLine(s watcher.Snapshot) string {
	switch {
	case s.Repos == 0:
		return "Nenhum repositório configurado"
	case s.Checking:
		return "Verificando PRs..."
	case s.LastError != "":
		return "Falha ao verificar: " + truncate(s.LastError, 50)
	case len(s.Pending) == 1:
		return "1 PR nova"
	case len(s.Pending) > 1:
		extra := ""
		if len(s.Pending) > MaxPRs {
			extra = fmt.Sprintf(" (%d no menu)", MaxPRs)
		}
		return fmt.Sprintf("%d PRs novas%s", len(s.Pending), extra)
	default:
		return "Nenhuma PR nova"
	}
}

func PRLabel(p gh.OpenPR) string {
	return truncate(fmt.Sprintf("%s#%d · %s", p.Ref.Repo, p.Ref.Number, strings.TrimSpace(p.Title)), 64)
}

func truncate(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n-1]) + "…"
}
