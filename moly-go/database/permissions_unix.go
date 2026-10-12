//go:build !windows

package database

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// HardenDataDir makes the data directory and the database files readable by the owner only (0700 / 0600) and sets the process umask so
// that files created later (the write-ahead log, the shared-memory file, backups) are private from the start. The database is
// encrypted; this keeps its file names, sizes and old copies from other accounts of the machine as well.
func HardenDataDir(dir string) {
	syscall.Umask(0o077)
	_ = os.Chmod(dir, 0o700)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() && (strings.Contains(e.Name(), ".db") || strings.HasSuffix(e.Name(), ".log")) {
			_ = os.Chmod(filepath.Join(dir, e.Name()), 0o600)
		}
	}
}
