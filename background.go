package main

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"

	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"oddeye/internal/autostart"
	"oddeye/internal/claude"
	"oddeye/internal/config"
	"oddeye/internal/gh"
	"oddeye/internal/review"
	"oddeye/internal/tray"
	"oddeye/internal/watcher"
)

// Eventos que o backend manda ao frontend.
const (
	eventOpenPR       = "pr:open"        // payload: "owner/repo#n"
	eventWatcher      = "watcher:update" // payload: watcher.Snapshot
	eventShowSettings = "ui:settings"    //
	eventDeviceCode   = "gh:device-code" // payload: gh.DeviceCode
)

// maxIndividualNotifications: acima disso, uma varredura vira um aviso só ("5 PRs novas").
const maxIndividualNotifications = 3

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.startNotifications()
	a.tray = tray.Start(tray.Callbacks{
		OnOpen:     a.showWindow,
		OnCheckNow: func() { a.watcher.CheckNow() },
		OnSettings: func() { a.showWindow(); runtime.EventsEmit(a.ctx, eventShowSettings) },
		OnQuit:     func() { runtime.Quit(a.ctx) },
		OnPR:       func(p gh.OpenPR) { a.openPR(p.Ref) },
		OnUpdate:   a.showUpdate,
	})
	a.watcher = watcher.New(a.store, a.gh, a.notifyNew, func(s watcher.Snapshot) {
		a.tray.Update(s)
		runtime.EventsEmit(a.ctx, eventWatcher, s)
	})
	a.tray.Update(a.watcher.Snapshot())

	// Reescreve a entrada de autostart: o caminho do executável muda quando o app é atualizado ou movido.
	if cfg, err := a.store.Load(); err == nil && cfg.LaunchAtLogin {
		_ = autostart.Set(true)
	}
	go a.watcher.Run(ctx)
	go a.runUpdateChecks(ctx)
}

func (a *App) shutdown(ctx context.Context) {
	if a.tray != nil {
		a.tray.Stop()
	}
	runtime.CleanupNotifications(ctx)
}

func (a *App) startNotifications() {
	if err := runtime.InitializeNotifications(a.ctx); err != nil {
		log.Printf("notificações indisponíveis: %v", err)
		return
	}
	// No macOS isso abre o pedido de permissão na primeira vez; nos outros SOs devolve true direto.
	if ok, _ := runtime.CheckNotificationAuthorization(a.ctx); !ok {
		_, _ = runtime.RequestNotificationAuthorization(a.ctx)
	}
	runtime.OnNotificationResponse(a.ctx, func(r runtime.NotificationResult) {
		if r.Error != nil {
			return
		}
		if _, ok := r.Response.UserInfo["update"]; ok {
			a.showUpdate()
			return
		}
		if raw, ok := r.Response.UserInfo["ref"].(string); ok {
			if ref, err := gh.ParsePRURL(raw); err == nil {
				a.openPR(ref)
				return
			}
		}
		a.showWindow()
	})
}

func (a *App) secondInstance(options.SecondInstanceData) { a.showWindow() }

func (a *App) showWindow() {
	runtime.WindowShow(a.ctx)
	runtime.WindowUnminimise(a.ctx)
}

func (a *App) openPR(ref gh.PRRef) {
	a.showWindow()
	runtime.EventsEmit(a.ctx, eventOpenPR, ref.String())
}

func (a *App) notifyNew(prs []gh.OpenPR) {
	if len(prs) > maxIndividualNotifications {
		_ = runtime.SendNotification(a.ctx, runtime.NotificationOptions{
			ID:    "oddeye-batch",
			Title: fmt.Sprintf("%d PRs novas para revisar", len(prs)),
			Body:  "Abra o Oddeye para ver a lista.",
		})
	} else {
		for _, p := range prs {
			_ = runtime.SendNotification(a.ctx, runtime.NotificationOptions{
				ID:       p.Key(),
				Title:    "Nova PR em " + p.Ref.Owner + "/" + p.Ref.Repo,
				Subtitle: fmt.Sprintf("#%d · %s", p.Ref.Number, p.Author),
				Body:     p.Title,
				Data:     map[string]interface{}{"ref": p.Ref.String()},
			})
		}
	}
	cfg, _ := a.store.Load()
	for _, p := range prs {
		if rc, ok := cfg.Repo(p.Ref.Owner, p.Ref.Repo); ok && rc.PreAnalyze {
			go a.preAnalyze(p.Ref)
		}
	}
}

// preAnalyze roda a triagem em segundo plano para que abrir a PR pela notificação seja instantâneo.
// Uma de cada vez: cada análise é uma sessão do Claude Code na assinatura do usuário.
func (a *App) preAnalyze(ref gh.PRRef) {
	a.preAnalyzeMu.Lock()
	defer a.preAnalyzeMu.Unlock()
	defer func() {
		if r := recover(); r != nil {
			log.Printf("pré-análise de %s falhou: %v\n%s", ref, r, debug.Stack())
		}
	}()
	info, err := a.gh.PRInfo(a.ctx, ref)
	if err != nil {
		return
	}
	files, err := a.gh.Files(a.ctx, ref)
	if err != nil {
		return
	}
	candidates, skipped := review.Split(files)
	if _, _, err := a.analysis(a.ctx, ref, info, candidates, skipped, false, func(claude.Progress) {}); err != nil {
		log.Printf("pré-análise de %s falhou: %v", ref, err)
	}
}

type SettingsView struct {
	Config        config.Config `json:"config"`
	DefaultPrompt string        `json:"defaultPrompt"`
	ConfigDir     string        `json:"configDir"`
}

func (a *App) GetSettings() (SettingsView, error) {
	cfg, err := a.store.Load()
	cfg.LaunchAtLogin = autostart.Enabled()
	return SettingsView{Config: cfg, DefaultPrompt: claude.DefaultReviewerPrompt, ConfigDir: a.store.Dir()}, err
}

// SaveSettings grava a configuração, aplica o autostart e dispara uma varredura com os repositórios novos.
func (a *App) SaveSettings(cfg config.Config) (config.Config, error) {
	if cfg.ReviewerPrompt == claude.DefaultReviewerPrompt {
		cfg.ReviewerPrompt = "" // guarda vazio para acompanhar melhorias futuras do prompt padrão
	}
	saved, err := a.store.Save(cfg)
	if err != nil {
		return saved, err
	}
	if err := autostart.Set(saved.LaunchAtLogin); err != nil {
		return saved, fmt.Errorf("configuração salva, mas não consegui ajustar o início automático: %w", err)
	}
	a.watcher.CheckNow()
	return saved, nil
}

func (a *App) ListRepos() ([]gh.Repo, error) { return a.gh.ListRepos(a.ctx) }

func (a *App) WatcherState() watcher.Snapshot { return a.watcher.Snapshot() }

func (a *App) CheckNow() { a.watcher.CheckNow() }

// AuthorizeGitHub faz login no gh ou amplia os escopos, mostrando o código do device flow no app.
func (a *App) AuthorizeGitHub() error {
	ctx, cancel := context.WithCancel(a.ctx)
	a.mu.Lock()
	if a.authCancel != nil {
		a.authCancel()
	}
	a.authCancel = cancel
	a.mu.Unlock()
	defer cancel()
	loggedIn := a.gh.AuthStatus(ctx) == nil
	err := a.gh.Authorize(ctx, loggedIn, func(dc gh.DeviceCode) {
		runtime.EventsEmit(a.ctx, eventDeviceCode, dc)
		runtime.BrowserOpenURL(a.ctx, dc.URL)
	})
	if err == nil {
		a.watcher.ResetViewer()
		a.watcher.CheckNow()
	}
	return err
}

func (a *App) CancelAuthorize() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.authCancel != nil {
		a.authCancel()
	}
}
