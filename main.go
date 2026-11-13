package main

import (
	"embed"
	"log"
	"os"
	"slices"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"

	"oddeye/internal/autostart"
	"oddeye/internal/config"
	"oddeye/internal/music"
	"oddeye/internal/proc"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

func main() {
	proc.EnsurePath()
	library, err := music.NewStore()
	if err != nil {
		log.Fatal(err)
	}
	store, err := config.NewStore()
	if err != nil {
		log.Fatal(err)
	}
	app := NewApp(library, store)
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
		AssetServer:      &assetserver.Options{Assets: assets, Middleware: library.Middleware},
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
