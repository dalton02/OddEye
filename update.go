package main

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"oddeye/internal/updater"
)

// Eventos do atualizador.
const (
	eventUpdateAvailable = "update:available" // payload: UpdateInfo
	eventUpdateProgress  = "update:progress"  // payload: UpdateProgress
)

const (
	firstUpdateCheck = 15 * time.Second
	updateInterval   = 6 * time.Hour
)

// UpdateInfo é o que o modal mostra sobre a versão nova.
type UpdateInfo struct {
	Current       string `json:"current"`
	Version       string `json:"version"`
	Notes         string `json:"notes"`
	PageURL       string `json:"pageUrl"`
	NeedsPassword bool   `json:"needsPassword"` // .deb/.rpm: o sistema vai pedir a senha
	Automatic     bool   `json:"automatic"`     // false: este tipo de instalação só atualiza pela página
}

// UpdateProgress acompanha o "Atualizar agora". Stage: downloading, installing, restarting, error.
type UpdateProgress struct {
	Stage  string `json:"stage"`
	Done   int64  `json:"done"`
	Total  int64  `json:"total"`
	Error  string `json:"error,omitempty"`
	Manual bool   `json:"manual,omitempty"` // erro que só se resolve baixando pela página
}

type updateState struct {
	mu         sync.Mutex
	client     *updater.Client
	available  *updater.Release
	notified   string
	installing bool
}

// currentVersion permite simular uma versão antiga em desenvolvimento (ODDEYE_FAKE_VERSION=0.0.1).
func (a *App) currentVersion() string {
	if v := os.Getenv("ODDEYE_FAKE_VERSION"); v != "" {
		return v
	}
	return a.version
}

func (a *App) Version() string { return a.currentVersion() }

func (a *App) updateClient() *updater.Client {
	a.update.mu.Lock()
	defer a.update.mu.Unlock()
	if a.update.client == nil {
		a.update.client = updater.NewClient()
	}
	return a.update.client
}

func (a *App) runUpdateChecks(ctx context.Context) {
	if a.currentVersion() == "dev" {
		return
	}
	timer := time.NewTimer(firstUpdateCheck)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		if _, err := a.checkUpdate(ctx, false); err != nil {
			log.Printf("verificação de atualização falhou: %v", err)
		}
		timer.Reset(updateInterval)
	}
}

// checkUpdate consulta o GitHub. Sem force, uma versão que o usuário pulou não gera aviso.
func (a *App) checkUpdate(ctx context.Context, force bool) (*UpdateInfo, error) {
	rel, err := a.updateClient().Latest(ctx, a.currentVersion())
	if err != nil {
		return nil, err
	}
	a.update.mu.Lock()
	a.update.available = rel
	a.update.mu.Unlock()
	if rel == nil {
		a.tray.SetUpdate("")
		return nil, nil
	}
	if cfg, _ := a.store.Load(); !force && cfg.SkippedVersion == rel.Version {
		a.tray.SetUpdate("")
		return nil, nil
	}
	info := a.updateInfo(rel)
	a.tray.SetUpdate(rel.Version)
	runtime.EventsEmit(a.ctx, eventUpdateAvailable, info)
	a.notifyUpdate(rel.Version)
	return &info, nil
}

func (a *App) updateInfo(rel *updater.Release) UpdateInfo {
	in := updater.Detect()
	_, hasAsset := updater.PickAsset(in.Kind, rel.Version, rel.Assets)
	return UpdateInfo{
		Current:       a.currentVersion(),
		Version:       rel.Version,
		Notes:         rel.Notes,
		PageURL:       rel.PageURL,
		NeedsPassword: in.Kind.NeedsPassword(),
		Automatic:     hasAsset,
	}
}

// notifyUpdate avisa uma vez por versão, para quem está com a janela fechada (app só na bandeja).
func (a *App) notifyUpdate(version string) {
	a.update.mu.Lock()
	already := a.update.notified == version
	a.update.notified = version
	a.update.mu.Unlock()
	if already {
		return
	}
	_ = runtime.SendNotification(a.ctx, runtime.NotificationOptions{
		ID:    "oddeye-update",
		Title: "Nova versão do Oddeye: v" + version,
		Body:  "Clique para atualizar.",
		Data:  map[string]interface{}{"update": version},
	})
}

// showUpdate abre a janela com o modal (clique no menu da bandeja ou na notificação).
func (a *App) showUpdate() {
	a.showWindow()
	a.update.mu.Lock()
	rel := a.update.available
	a.update.mu.Unlock()
	if rel != nil {
		runtime.EventsEmit(a.ctx, eventUpdateAvailable, a.updateInfo(rel))
	}
}

// CheckUpdate é o "Procurar atualizações" das configurações: mostra a versão mesmo se ela foi pulada.
func (a *App) CheckUpdate() (*UpdateInfo, error) {
	if a.currentVersion() == "dev" {
		return nil, errors.New("build de desenvolvimento não procura atualizações")
	}
	return a.checkUpdate(a.ctx, true)
}

// PendingUpdate devolve a atualização já encontrada (o frontend chama ao montar, para não perder o aviso).
func (a *App) PendingUpdate() *UpdateInfo {
	a.update.mu.Lock()
	rel := a.update.available
	a.update.mu.Unlock()
	if rel == nil {
		return nil
	}
	if cfg, _ := a.store.Load(); cfg.SkippedVersion == rel.Version {
		return nil
	}
	info := a.updateInfo(rel)
	return &info
}

func (a *App) SkipUpdate(version string) error {
	cfg, err := a.store.Load()
	if err != nil {
		return err
	}
	cfg.SkippedVersion = version
	if _, err := a.store.Save(cfg); err != nil {
		return err
	}
	a.tray.SetUpdate("")
	return nil
}

func (a *App) OpenReleasePage() {
	a.update.mu.Lock()
	rel := a.update.available
	a.update.mu.Unlock()
	url := "https://github.com/" + updater.Repo + "/releases/latest"
	if rel != nil && rel.PageURL != "" {
		url = rel.PageURL
	}
	runtime.BrowserOpenURL(a.ctx, url)
}

// InstallUpdate baixa, confere, troca o app instalado e reinicia. O progresso vai pelo evento update:progress.
func (a *App) InstallUpdate() error {
	a.update.mu.Lock()
	rel := a.update.available
	if a.update.installing {
		a.update.mu.Unlock()
		return errors.New("a atualização já está em andamento")
	}
	a.update.installing = true
	a.update.mu.Unlock()
	defer func() {
		a.update.mu.Lock()
		a.update.installing = false
		a.update.mu.Unlock()
	}()

	fail := func(err error) error {
		runtime.EventsEmit(a.ctx, eventUpdateProgress, UpdateProgress{
			Stage: "error", Error: err.Error(), Manual: errors.Is(err, updater.ErrManual),
		})
		return err
	}
	if rel == nil {
		return fail(errors.New("nenhuma atualização disponível"))
	}
	in := updater.Detect()
	asset, ok := updater.PickAsset(in.Kind, rel.Version, rel.Assets)
	if !ok {
		return fail(updater.ErrManual)
	}

	file, err := a.updateClient().Download(a.ctx, rel, asset, func(done, total int64) {
		runtime.EventsEmit(a.ctx, eventUpdateProgress, UpdateProgress{Stage: "downloading", Done: done, Total: total})
	})
	if err != nil {
		return fail(err)
	}
	defer os.RemoveAll(filepath.Dir(file))

	// Build de desenvolvimento: testa só o download e o checksum, sem trocar o binário do `wails dev`.
	if a.version == "dev" {
		return fail(errors.New("download e checksum ok; build de desenvolvimento não se substitui"))
	}

	runtime.EventsEmit(a.ctx, eventUpdateProgress, UpdateProgress{Stage: "installing"})
	if err := updater.Apply(in, file); err != nil {
		return fail(err)
	}
	runtime.EventsEmit(a.ctx, eventUpdateProgress, UpdateProgress{Stage: "restarting"})
	if err := updater.Relaunch(in.Exe); err != nil {
		return fail(errors.New("a atualização foi instalada, mas não consegui reabrir o app: abra o Oddeye de novo"))
	}
	time.Sleep(300 * time.Millisecond) // deixa o modal mostrar "Reiniciando…"
	runtime.Quit(a.ctx)
	return nil
}
