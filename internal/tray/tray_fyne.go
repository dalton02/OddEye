//go:build !darwin

package tray

import (
	"runtime"
	"sync"

	"fyne.io/systray"
)

// fyneBackend usa o fyne.io/systray: StatusNotifierItem via D-Bus no Linux e Shell_NotifyIcon no
// Windows. O systray.Run roda numa goroutine presa à própria thread, porque no Windows a janela oculta
// da bandeja e o loop de mensagens precisam estar na mesma thread, e ela não pode ser a do Wails.
type fyneBackend struct {
	ready chan struct{}
	once  sync.Once

	mu    sync.Mutex
	items map[int]*systray.MenuItem
}

func newBackend() backend {
	return &fyneBackend{ready: make(chan struct{}), items: map[int]*systray.MenuItem{}}
}

func (f *fyneBackend) start(items []menuItem, onClick func(id int)) {
	go func() {
		runtime.LockOSThread()
		systray.Run(func() {
			systray.SetIcon(iconNormal)
			systray.SetTitle("")
			systray.SetOnTapped(func() { onClick(idOpen) })
			f.mu.Lock()
			for _, it := range items {
				if it.id == separator {
					systray.AddSeparator()
					continue
				}
				mi := systray.AddMenuItem(it.title, "")
				if it.id == idStatus {
					mi.Disable()
				}
				if !it.visible {
					mi.Hide()
				}
				f.items[it.id] = mi
				go func(id int, mi *systray.MenuItem) {
					for range mi.ClickedCh {
						onClick(id)
					}
				}(it.id, mi)
			}
			f.mu.Unlock()
			close(f.ready)
		}, nil)
	}()
}

func (f *fyneBackend) item(id int) *systray.MenuItem {
	<-f.ready
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.items[id]
}

func (f *fyneBackend) setItem(id int, title string, visible bool) {
	mi := f.item(id)
	if mi == nil {
		return
	}
	if visible {
		mi.SetTitle(title)
		mi.Show()
	} else {
		mi.Hide()
	}
}

func (f *fyneBackend) setIcon(dot bool) {
	<-f.ready
	if dot {
		systray.SetIcon(iconDot)
	} else {
		systray.SetIcon(iconNormal)
	}
}

func (f *fyneBackend) setTooltip(text string) {
	<-f.ready
	systray.SetTooltip(text)
}

func (f *fyneBackend) stop() { f.once.Do(systray.Quit) }
