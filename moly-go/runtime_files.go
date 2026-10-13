package main

import (
	"fmt"
	"os"
	"path/filepath"

	"moly/config"
)

// ensureConfigDir makes "config/constitution.yaml" and the other configuration files readable from the working directory.
// In a source checkout they already are, and nothing changes. An installed binary has no config folder next to it, so it
// writes its embedded copy to <data dir>/runtime/config (rewritten every start, so it always matches the binary) and
// moves there.
func ensureConfigDir() error {
	if _, err := os.Stat(filepath.Join("config", "constitution.yaml")); err == nil {
		return nil
	}
	root := filepath.Join(molyDataDir(), "runtime")
	dir := filepath.Join(root, "config")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	entries, err := config.Files.ReadDir(".")
	if err != nil {
		return err
	}
	for _, e := range entries {
		data, err := config.Files.ReadFile(e.Name())
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, e.Name()), data, 0o600); err != nil {
			return err
		}
	}
	if err := os.Chdir(root); err != nil {
		return fmt.Errorf("cannot enter %s: %w", root, err)
	}
	return nil
}
