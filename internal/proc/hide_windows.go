//go:build windows

// Package proc guarda o que é específico de SO ao lançar processos filhos.
package proc

import (
	"os/exec"
	"syscall"
)

// HideWindow evita que cada subprocesso (gh, claude) pisque uma janela de console no Windows.
func HideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
}
