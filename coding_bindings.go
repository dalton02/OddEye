package main

import (
	"os"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"oddeye/internal/config"
	"oddeye/internal/term"
	"oddeye/internal/workspace"
)

// Métodos da aba Coding: um Claude Code por aba num pseudo-terminal (internal/term) e o editor
// com árvore, código e alterações do git de cada pasta (internal/workspace).

func (a *App) CodingPrefs() (config.CodingPrefs, error) { return a.store.LoadCoding() }

func (a *App) SaveCodingPrefs(p config.CodingPrefs) error { return a.store.SaveCoding(p) }

// PickFolder abre o seletor de pasta para uma aba nova; devolve "" quando o usuário cancela.
func (a *App) PickFolder() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: "Pasta do projeto"})
}

// TermOpen liga o terminal da aba: inicia o claude (ou o shell, com kind "shell") na pasta ou, se ele
// já roda, devolve o histórico.
func (a *App) TermOpen(id, dir, kind string, cols, rows int) (term.Snapshot, error) {
	return a.term.Open(id, dir, kind, cols, rows)
}

// HomeDir é a pasta pessoal, onde abre o shell quando não há aba para dar a pasta.
func (a *App) HomeDir() (string, error) { return os.UserHomeDir() }

func (a *App) TermWrite(id, data string) error { return a.term.Write(id, data) }

func (a *App) TermResize(id string, cols, rows int) error { return a.term.Resize(id, cols, rows) }

func (a *App) TermClose(id string) { a.term.Close(id) }

func (a *App) WsListDir(root, rel string) ([]workspace.Entry, error) {
	return workspace.ListDir(a.ctx, root, rel)
}

func (a *App) WsReadFile(root, rel string) (workspace.FileContent, error) {
	return workspace.ReadFile(root, rel)
}

func (a *App) WsWriteFile(root, rel, content string) error {
	return workspace.WriteFile(root, rel, content)
}

func (a *App) WsChanges(root string) (workspace.Changes, error) {
	return workspace.GetChanges(a.ctx, root)
}

func (a *App) WsFileDiff(root, rel string) (workspace.FileDiff, error) {
	return workspace.GetFileDiff(a.ctx, root, rel)
}
