package tray

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa
#include <stdlib.h>
void trayStart(const void *normal, int normalLen, const void *dot, int dotLen, const int *ids, const char **titles, const int *visible, int count);
void traySetItem(int id, const char *title, int visible);
void traySetIcon(int dot);
void traySetTooltip(const char *text);
void trayStop(void);
*/
import "C"

import (
	"sync"
	"unsafe"
)

// darwinBackend cria o NSStatusItem direto em Objective-C (tray_darwin.m). O fyne.io/systray não serve
// aqui: no macOS ele assume o delegate do NSApplication, que já pertence ao Wails.
type darwinBackend struct{}

var (
	clickMu      sync.Mutex
	clickHandler func(int)
)

//export goTrayClicked
func goTrayClicked(id C.int) {
	clickMu.Lock()
	h := clickHandler
	clickMu.Unlock()
	if h != nil {
		go h(int(id))
	}
}

func newBackend() backend { return darwinBackend{} }

func (darwinBackend) start(items []menuItem, onClick func(id int)) {
	clickMu.Lock()
	clickHandler = onClick
	clickMu.Unlock()

	n := len(items)
	ids := make([]C.int, n)
	vis := make([]C.int, n)
	titles := C.malloc(C.size_t(n) * C.size_t(unsafe.Sizeof(uintptr(0))))
	defer C.free(titles)
	tarr := unsafe.Slice((**C.char)(titles), n)
	for i, it := range items {
		ids[i] = C.int(it.id)
		if it.visible {
			vis[i] = 1
		}
		tarr[i] = C.CString(it.title)
	}
	defer func() {
		for _, p := range tarr {
			C.free(unsafe.Pointer(p))
		}
	}()
	C.trayStart(
		unsafe.Pointer(&iconNormal[0]), C.int(len(iconNormal)),
		unsafe.Pointer(&iconDot[0]), C.int(len(iconDot)),
		&ids[0], (**C.char)(titles), &vis[0], C.int(n),
	)
}

func (darwinBackend) setItem(id int, title string, visible bool) {
	cs := C.CString(title)
	defer C.free(unsafe.Pointer(cs))
	v := C.int(0)
	if visible {
		v = 1
	}
	C.traySetItem(C.int(id), cs, v)
}

func (darwinBackend) setIcon(dot bool) {
	d := C.int(0)
	if dot {
		d = 1
	}
	C.traySetIcon(d)
}

func (darwinBackend) setTooltip(text string) {
	cs := C.CString(text)
	defer C.free(unsafe.Pointer(cs))
	C.traySetTooltip(cs)
}

func (darwinBackend) stop() { C.trayStop() }
