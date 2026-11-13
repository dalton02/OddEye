package main

import (
	"embed"
	"log"
	"os"
	"slices"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"

	"oddeye/internal/autostart"
	"oddeye/internal/config"
	"oddeye/internal/music"
	"oddeye/internal/proc"
	"oddeye/internal/torrent"
	"oddeye/internal/updater"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

// version é gravada no build de release (-ldflags "-X main.version=0.1.0"). "dev" desliga o atualizador.
var version = "dev"

func main() {
	// Reaberto depois de uma atualização: espera o processo antigo sair para não cair no bloqueio de instância única.
	updater.WaitForParent(os.Args[1:], 15*time.Second)
	updater.Cleanup()
	proc.EnsurePath()
	library, err := music.NewStore()
	if err != nil {
		log.Fatal(err)
	}
	store, err := config.NewStore()
	if err != nil {
		log.Fatal(err)
	}
	app := NewApp(library, store, version)
	if app.playlists, app.torrents, err = newMusicServices(); err != nil {
		log.Fatal(err)
	}
	assetMiddleware := library.Middleware
	if app.torrents != nil {
		assetMiddleware = torrent.Chain(library.Middleware, app.torrents.Middleware)
	}
	err = wails.Run(&options.App{
		Title:     "Oddeye",
		Width:     1440,
		Height:    920,
		MinWidth:  960,
		MinHeight: 640,
		// Fechar a janela só a esconde: o app continua na bandeja vigiando as PRs. Sair é pelo menu da bandeja.
		HideWindowOnClose: true,
		StartHidden:       slices.Contains(os.Args[1:], autostart.HiddenFlag),
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId:               "io.github.oddeye",
			OnSecondInstanceLaunch: app.secondInstance,
		},
		AssetServer:      &assetserver.Options{Assets: assets, Middleware: assetMiddleware},
		BackgroundColour: &options.RGBA{R: 3, G: 4, B: 7, A: 255},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind:             []interface{}{app},
		Linux:            &linux.Options{Icon: appIcon, ProgramName: "oddeye"},
		Mac: &mac.Options{
			About: &mac.AboutInfo{Title: "Oddeye", Message: "Tour guiado de pull requests com o Claude Code.", Icon: appIcon},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
