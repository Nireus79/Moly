package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGetConfigDir(t *testing.T) {
	// Test Linux path
	if runtime.GOOS == "linux" {
		path := getConfigDir("test.db")
		if !strings.Contains(path, ".config") || !strings.Contains(path, "moly") {
			t.Errorf("Linux path incorrect: %s", path)
		}
		if !strings.HasSuffix(path, "test.db") {
			t.Errorf("Filename not appended: %s", path)
		}
	}

	// Test macOS path structure (only on macOS)
	if runtime.GOOS == "darwin" {
		path := getConfigDir("test.db")
		if !strings.Contains(path, "Library") || !strings.Contains(path, "Application Support") {
			t.Errorf("macOS path incorrect: %s", path)
		}
		if !strings.Contains(path, "Moly") {
			t.Errorf("macOS path missing Moly: %s", path)
		}
	}

	// Test Windows path structure (only on Windows)
	if runtime.GOOS == "windows" {
		path := getConfigDir("test.db")
		if !strings.Contains(path, "Moly") {
			t.Errorf("Windows path missing Moly: %s", path)
		}
		if !strings.HasSuffix(path, "test.db") {
			t.Errorf("Windows filename not appended: %s", path)
		}
	}
}

func TestGetConfigDirCreatesDirectory(t *testing.T) {
	// Test that directory gets created
	testFile := "test-" + randomString() + ".db"
	path := getConfigDir(testFile)

	// Extract directory from path
	configDir := filepath.Dir(path)

	// Verify directory exists
	if _, err := os.Stat(configDir); err != nil {
		t.Errorf("Config directory not created: %s", configDir)
	}

	// Cleanup
	os.RemoveAll(filepath.Dir(configDir))
}

func TestGetConfigDirWithXDGHome(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("XDG_CONFIG_HOME only relevant on Linux")
	}

	// Save current env
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", oldXDG)

	// Set custom XDG_CONFIG_HOME
	customPath := "/tmp/custom-xdg-test"
	os.Setenv("XDG_CONFIG_HOME", customPath)

	path := getConfigDir("test.db")

	if !strings.HasPrefix(path, customPath) {
		t.Errorf("XDG_CONFIG_HOME not respected. Expected prefix %s, got %s", customPath, path)
	}

	// Cleanup
	os.RemoveAll(customPath)
}

// Helper function to generate random string for unique test files
func randomString() string {
	return os.Getenv("USER") + "-" + os.Getenv("RANDOM")
}
