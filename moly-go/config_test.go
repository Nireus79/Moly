package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGetDefaultConfig(t *testing.T) {
	config := getDefaultConfig()

	tests := []struct {
		name     string
		expected interface{}
		actual   interface{}
	}{
		{"Version", "1.0", config.Version},
		{"Provider", "local", config.Provider},
		{"Model", "mistral", config.Model},
		{"FirstRunComplete", false, config.FirstRunComplete},
		{"Mode", "direct", config.Mode},
		{"OllamaInstalled", false, config.OllamaInstalled},
		{"OllamaRunning", false, config.OllamaRunning},
	}

	for _, test := range tests {
		if test.expected != test.actual {
			t.Errorf("%s: expected %v, got %v", test.name, test.expected, test.actual)
		}
	}
}

func TestConfigSaveAndLoad(t *testing.T) {
	// Use temp directory for testing
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Create config directory
	configDir := filepath.Join(tempDir, ".config", "moly")
	os.MkdirAll(configDir, 0755)

	config := Config{
		Version:          "1.0",
		Provider:         "claude",
		Model:            "claude-3-sonnet",
		FirstRunComplete: true,
		Mode:             "socratic",
		APIKeys: map[string]interface{}{
			"claude": "test-key-123",
		},
	}

	// Save config
	if err := saveConfig(config); err != nil {
		t.Fatalf("saveConfig failed: %v", err)
	}

	// Load config
	loaded := loadConfig()

	if loaded.Provider != "claude" {
		t.Errorf("Provider: expected claude, got %s", loaded.Provider)
	}
	if loaded.Model != "claude-3-sonnet" {
		t.Errorf("Model: expected claude-3-sonnet, got %s", loaded.Model)
	}
	if !loaded.FirstRunComplete {
		t.Errorf("FirstRunComplete: expected true, got false")
	}
	if loaded.Mode != "socratic" {
		t.Errorf("Mode: expected socratic, got %s", loaded.Mode)
	}
}

func TestConfigInitialization(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Config file should not exist yet
	configPath := getConfigPath()
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatal("Config file should not exist before initialization")
	}

	// Initialize config
	if err := initConfig(); err != nil {
		t.Fatalf("initConfig failed: %v", err)
	}

	// Config file should exist now
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file should exist after initialization")
	}

	// Should be loadable
	config := loadConfig()
	if config.Version != "1.0" {
		t.Errorf("Initial config version: expected 1.0, got %s", config.Version)
	}
}

func TestConfigUpdateTimestamp(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	os.MkdirAll(filepath.Join(tempDir, ".config", "moly"), 0755)

	config := getDefaultConfig()

	// Save config first time
	saveConfig(config)
	loaded1 := loadConfig()

	// Wait to ensure timestamp changes
	time.Sleep(1 * time.Millisecond)

	// Modify and save again
	config.Model = "llama2"
	saveConfig(config)
	loaded2 := loadConfig()

	// Timestamps should be set
	if loaded1.UpdatedAt == "" {
		t.Error("UpdatedAt should not be empty on first save")
	}
	if loaded2.UpdatedAt == "" {
		t.Error("UpdatedAt should not be empty on second save")
	}
	if loaded2.Model != "llama2" {
		t.Errorf("Model should be updated to llama2, got %s", loaded2.Model)
	}
}

func TestConfigDefaultValues(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	os.MkdirAll(filepath.Join(tempDir, ".config", "moly"), 0755)

	// Initialize fresh config
	initConfig()
	config := loadConfig()

	// Verify all defaults
	if config.Provider != "local" {
		t.Errorf("Default provider should be 'local', got %s", config.Provider)
	}
	if config.Model != "mistral" {
		t.Errorf("Default model should be 'mistral', got %s", config.Model)
	}
	if config.InstalledModels == nil {
		t.Error("InstalledModels should not be nil")
	}
	if config.APIKeys == nil {
		t.Error("APIKeys should not be nil")
	}
}
