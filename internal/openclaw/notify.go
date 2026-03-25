package openclaw

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// NotifyReload signals openclaw to reload its configuration via SIGUSR1.
//
// Lookup order:
//  1. If pid > 0: send SIGUSR1 directly to that PID.
//  2. If pidFile is set and readable: read PID from file, send SIGUSR1.
//  3. Otherwise: return an error (config is already written to disk;
//     openclaw will pick it up on next restart).
func NotifyReload(pid int, pidFile string) error {
	if pid > 0 {
		return notifyViaSigusr1(pid)
	}
	if pidFile != "" {
		if p, err := readPIDFile(pidFile); err == nil && p > 0 {
			return notifyViaSigusr1(p)
		}
	}
	return fmt.Errorf("cannot notify openclaw: OPENCLAW_PID not set and PID file not found (%s)", pidFile)
}

func readPIDFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("invalid PID in %s: %w", path, err)
	}
	if pid <= 0 {
		return 0, fmt.Errorf("invalid PID %d in %s", pid, path)
	}
	return pid, nil
}

func notifyViaSigusr1(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find process %d: %w", pid, err)
	}
	if err := sendSigusr1(proc); err != nil {
		return fmt.Errorf("SIGUSR1 to PID %d: %w", pid, err)
	}
	return nil
}
