package watcher

import (
	"context"
	"strings"
	"sync"
	"time"

	"oddeye/internal/config"
	"oddeye/internal/gh"
)

type Source interface {
	OpenPRs(ctx context.Context, repos []string) ([]gh.OpenPR, error)
	Viewer(ctx context.Context) (string, error)
}

// Snapshot é o que a bandeja e a tela inicial mostram.
type Snapshot struct {
	Pending   []gh.OpenPR `json:"pending"`
	LastPoll  time.Time   `json:"lastPoll" ts_type:"string"`
	LastError string      `json:"lastError"`
	Checking  bool        `json:"checking"`
	Repos     int         `json:"repos"`
}

type Watcher struct {
	store    *config.Store
	src      Source
	onNew    func([]gh.OpenPR)
	onUpdate func(Snapshot)
	trigger  chan struct{}

	mu       sync.Mutex
	viewer   string
	checking bool
}

// New cria o watcher. onNew recebe as PRs novas de cada varredura; onUpdate, o estado a cada mudança.
func New(store *config.Store, src Source, onNew func([]gh.OpenPR), onUpdate func(Snapshot)) *Watcher {
	return &Watcher{store: store, src: src, onNew: onNew, onUpdate: onUpdate, trigger: make(chan struct{}, 1)}
}

// Run varre na hora e depois a cada intervalo configurado, até ctx acabar. O intervalo é relido a
// cada ciclo, então salvar a configuração e chamar CheckNow basta para aplicá-lo.
func (w *Watcher) Run(ctx context.Context) {
	for {
		w.poll(ctx)
		cfg, _ := w.store.Load()
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(cfg.PollIntervalMinutes) * time.Minute):
		case <-w.trigger:
		}
	}
}

// CheckNow antecipa a próxima varredura; chamadas repetidas enquanto uma está pendente se fundem.
func (w *Watcher) CheckNow() {
	select {
	case w.trigger <- struct{}{}:
	default:
	}
}

func (w *Watcher) Snapshot() Snapshot {
	st, _ := w.store.LoadState()
	cfg, _ := w.store.Load()
	w.mu.Lock()
	defer w.mu.Unlock()
	pending := st.Pending
	if pending == nil {
		pending = []gh.OpenPR{}
	}
	return Snapshot{Pending: pending, LastPoll: st.LastPoll, LastError: st.LastError, Checking: w.checking, Repos: len(cfg.Repos)}
}

// MarkOpened tira a PR da lista de pendentes quando o usuário a abre no app.
func (w *Watcher) MarkOpened(ref gh.PRRef) {
	key := strings.ToLower(ref.String())
	w.mu.Lock()
	st, err := w.store.LoadState()
	if err != nil {
		w.mu.Unlock()
		return
	}
	kept := st.Pending[:0:0]
	for _, p := range st.Pending {
		if p.Key() != key {
			kept = append(kept, p)
		}
	}
	changed := len(kept) != len(st.Pending)
	st.Pending = kept
	if changed {
		_ = w.store.SaveState(st)
	}
	w.mu.Unlock()
	if changed {
		w.onUpdate(w.Snapshot())
	}
}

func (w *Watcher) setChecking(v bool) {
	w.mu.Lock()
	w.checking = v
	w.mu.Unlock()
	w.onUpdate(w.Snapshot())
}

func (w *Watcher) poll(ctx context.Context) {
	cfg, err := w.store.Load()
	if err != nil || len(cfg.Repos) == 0 {
		return
	}
	w.setChecking(true)
	defer w.setChecking(false)

	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	fresh, err := w.scan(ctx, cfg)
	if ctx.Err() != nil && err != nil {
		return
	}
	if len(fresh) > 0 {
		w.onNew(fresh)
	}
}

func (w *Watcher) scan(ctx context.Context, cfg config.Config) ([]gh.OpenPR, error) {
	viewer, err := w.viewerLogin(ctx)
	var prs []gh.OpenPR
	if err == nil {
		names := make([]string, 0, len(cfg.Repos))
		for _, r := range cfg.Repos {
			names = append(names, r.FullName())
		}
		prs, err = w.src.OpenPRs(ctx, names)
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	st, loadErr := w.store.LoadState()
	if loadErr != nil {
		return nil, loadErr
	}
	st.LastPoll = time.Now()
	if err != nil {
		// Falha de rede ou do gh: mantém o estado como estava (nada some da caixa de entrada) e só registra.
		st.LastError = err.Error()
		_ = w.store.SaveState(st)
		return nil, err
	}
	st.LastError = ""
	fresh := Evaluate(&st, cfg, viewer, prs, st.LastPoll)
	return fresh, w.store.SaveState(st)
}

func (w *Watcher) viewerLogin(ctx context.Context) (string, error) {
	w.mu.Lock()
	v := w.viewer
	w.mu.Unlock()
	if v != "" {
		return v, nil
	}
	v, err := w.src.Viewer(ctx)
	if err != nil {
		return "", err
	}
	w.mu.Lock()
	w.viewer = v
	w.mu.Unlock()
	return v, nil
}

// ResetViewer esquece o login em cache (após trocar de conta no gh).
func (w *Watcher) ResetViewer() {
	w.mu.Lock()
	w.viewer = ""
	w.mu.Unlock()
}
