//go:build windows

package openclaw

import (
	"fmt"
	"os"
)

// Windows does not support SIGUSR1. This service is intended for Linux/macOS deployment.
func sendSigusr1(_ *os.Process) error {
	return fmt.Errorf("SIGUSR1 is not supported on Windows")
}
