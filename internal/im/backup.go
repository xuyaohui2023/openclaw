package im

// Backup rotation logic that mirrors openclaw's src/config/backup-rotation.ts.
//
// Scheme (configBackupCount = 5):
//   openclaw.json          ← live config
//   openclaw.json.bak      ← most-recent backup (created each write)
//   openclaw.json.bak.1    ← previous backup
//   openclaw.json.bak.2
//   openclaw.json.bak.3
//   openclaw.json.bak.4    ← oldest backup (deleted when ring is full)
//
// Before every write the service:
//  1. Rotates the numbered ring (.bak.4 deleted, .bak.3→.bak.4, …, .bak→.bak.1)
//  2. Copies the current live file to .bak
//  3. Hardens every .bak* to mode 0600
//  4. Removes any orphan .bak.* files whose suffix falls outside the ring

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const configBackupCount = 5

// maintainConfigBackups performs the full backup cycle for configPath.
// Errors are best-effort and never block the caller from writing.
func maintainConfigBackups(configPath string) {
	backupBase := configPath + ".bak"
	rotateConfigBackups(configPath, backupBase)
	copyConfigToBackup(configPath, backupBase)
	hardenBackupPermissions(backupBase)
	cleanOrphanBackups(configPath, backupBase)
}

// rotateConfigBackups shifts the numbered ring one slot upward,
// making room for a new primary .bak.
func rotateConfigBackups(configPath, backupBase string) {
	maxIndex := configBackupCount - 1

	// Drop the oldest slot — best-effort
	_ = os.Remove(fmt.Sprintf("%s.%d", backupBase, maxIndex))

	// Shift: .bak.(N-2) → .bak.(N-1), …, .bak.1 → .bak.2
	for idx := maxIndex - 1; idx >= 1; idx-- {
		from := fmt.Sprintf("%s.%d", backupBase, idx)
		to := fmt.Sprintf("%s.%d", backupBase, idx+1)
		_ = os.Rename(from, to) // best-effort
	}

	// .bak → .bak.1
	_ = os.Rename(backupBase, backupBase+".1") // best-effort
}

// copyConfigToBackup copies the live config file to <configPath>.bak.
func copyConfigToBackup(configPath, backupBase string) {
	src, err := os.Open(configPath)
	if err != nil {
		return // file may not exist yet — skip
	}
	defer src.Close()

	dst, err := os.OpenFile(backupBase, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return
	}
	defer dst.Close()

	_, _ = io.Copy(dst, src)
}

// hardenBackupPermissions sets mode 0600 on all .bak files.
// Mirrors hardenBackupPermissions() in backup-rotation.ts.
func hardenBackupPermissions(backupBase string) {
	_ = os.Chmod(backupBase, 0o600)
	for i := 1; i < configBackupCount; i++ {
		_ = os.Chmod(fmt.Sprintf("%s.%d", backupBase, i), 0o600)
	}
}

// cleanOrphanBackups removes .bak.* files whose suffix is not in the
// managed ring (1 … configBackupCount-1).
// Mirrors cleanOrphanBackups() in backup-rotation.ts.
func cleanOrphanBackups(configPath, backupBase string) {
	dir := filepath.Dir(configPath)
	base := filepath.Base(configPath)
	prefix := base + ".bak."

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	// Build valid suffix set: "1", "2", …, "4"
	valid := make(map[string]struct{}, configBackupCount-1)
	for i := 1; i < configBackupCount; i++ {
		valid[fmt.Sprintf("%d", i)] = struct{}{}
	}

	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		suffix := strings.TrimPrefix(name, prefix)
		if _, ok := valid[suffix]; ok {
			continue
		}
		// Orphan — remove best-effort
		_ = os.Remove(filepath.Join(dir, name))
	}
}
