//go:build !windows

package openclaw

import (
	"os"
	"syscall"
)

func sendSigusr1(proc *os.Process) error {
	return proc.Signal(syscall.SIGUSR1)
}
