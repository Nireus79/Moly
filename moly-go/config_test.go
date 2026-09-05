package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

// Tests for LoadConfig() - new configuration system

func TestLoadConfigDefaults(t *testing.T) {
	// Save current env vars to restore later
	oldPort := os.Getenv("MOLY_PORT")
	oldHost := os.Getenv("MOLY_HOST")
	oldLogLevel := os.Getenv("MOLY_LOG_LEVEL")
	oldProxyPort := os.Getenv("MOLY_CORS_PROXY_PORT")

	defer func() {
		os.Setenv("MOLY_PORT", oldPort)
		os.Setenv("MOLY_HOST", oldHost)
		os.Setenv("MOLY_LOG_LEVEL", oldLogLevel)
		os.Setenv("MOLY_CORS_PROXY_PORT", oldProxyPort)
	}()

	// Unset all env vars to test defaults
	os.Unsetenv("MOLY_PORT")
	os.Unsetenv("MOLY_HOST")
	os.Unsetenv("MOLY_LOG_LEVEL")
	os.Unsetenv("MOLY_CORS_PROXY_PORT")

	config := LoadConfig()

	if config.Port != DefaultServerConfig.Port {
		t.Errorf("Expected default Port %s, got %s", DefaultServerConfig.Port, config.Port)
	}
	if config.Host != DefaultServerConfig.Host {
		t.Errorf("Expected default Host %s, got %s", DefaultServerConfig.Host, config.Host)
	}
	if config.LogLevel != DefaultServerConfig.LogLevel {
		t.Errorf("Expected default LogLevel %s, got %s", DefaultServerConfig.LogLevel, config.LogLevel)
	}
	if config.CORSProxyPort != DefaultServerConfig.CORSProxyPort {
		t.Errorf("Expected default CORSProxyPort %s, got %s", DefaultServerConfig.CORSProxyPort, config.CORSProxyPort)
	}
}

func TestLoadConfigEnvVarOverride(t *testing.T) {
	// Save current env vars
	oldPort := os.Getenv("MOLY_PORT")
	oldHost := os.Getenv("MOLY_HOST")
	oldLogLevel := os.Getenv("MOLY_LOG_LEVEL")
	oldProxyPort := os.Getenv("MOLY_CORS_PROXY_PORT")

	defer func() {
		os.Setenv("MOLY_PORT", oldPort)
		os.Setenv("MOLY_HOST", oldHost)
		os.Setenv("MOLY_LOG_LEVEL", oldLogLevel)
		os.Setenv("MOLY_CORS_PROXY_PORT", oldProxyPort)
	}()

	// Set env vars
	testPort := ":9999"
	testHost := "0.0.0.0"
	testLogLevel := "debug"
	testProxyPort := ":8888"

	os.Setenv("MOLY_PORT", testPort)
	os.Setenv("MOLY_HOST", testHost)
	os.Setenv("MOLY_LOG_LEVEL", testLogLevel)
	os.Setenv("MOLY_CORS_PROXY_PORT", testProxyPort)

	config := LoadConfig()

	if config.Port != testPort {
		t.Errorf("Expected Port %s, got %s (env var not applied)", testPort, config.Port)
	}
	if config.Host != testHost {
		t.Errorf("Expected Host %s, got %s (env var not applied)", testHost, config.Host)
	}
	if config.LogLevel != testLogLevel {
		t.Errorf("Expected LogLevel %s, got %s (env var not applied)", testLogLevel, config.LogLevel)
	}
	if config.CORSProxyPort != testProxyPort {
		t.Errorf("Expected CORSProxyPort %s, got %s (env var not applied)", testProxyPort, config.CORSProxyPort)
	}
}

func TestLoadConfigPartialEnvVarOverride(t *testing.T) {
	// Save current env vars
	oldPort := os.Getenv("MOLY_PORT")
	oldHost := os.Getenv("MOLY_HOST")

	defer func() {
		os.Setenv("MOLY_PORT", oldPort)
		os.Setenv("MOLY_HOST", oldHost)
	}()

	// Set only some env vars
	testPort := ":8765"
	os.Setenv("MOLY_PORT", testPort)
	os.Unsetenv("MOLY_HOST")

	config := LoadConfig()

	if config.Port != testPort {
		t.Errorf("Expected Port %s, got %s", testPort, config.Port)
	}
	if config.Host != DefaultServerConfig.Host {
		t.Errorf("Expected default Host %s for unset env var, got %s", DefaultServerConfig.Host, config.Host)
	}
}

func TestLoadConfigEmptyEnvVarIgnored(t *testing.T) {
	// Save current env vars
	oldPort := os.Getenv("MOLY_PORT")

	defer func() {
		os.Setenv("MOLY_PORT", oldPort)
	}()

	// Set env var to empty string (should be ignored and use default)
	os.Setenv("MOLY_PORT", "")

	config := LoadConfig()

	if config.Port != DefaultServerConfig.Port {
		t.Errorf("Expected empty env var to be ignored and use default %s, got %s", DefaultServerConfig.Port, config.Port)
	}
}

func TestLoadConfigFilePath(t *testing.T) {
	path := getConfigFilePath()

	// Verify path contains expected components based on OS
	if runtime.GOOS == "linux" {
		if !strings.Contains(path, ".config") || !strings.Contains(path, "moly") {
			t.Errorf("Linux config path incorrect: %s", path)
		}
	} else if runtime.GOOS == "darwin" {
		if !strings.Contains(path, "Library") || !strings.Contains(path, "Application Support") {
			t.Errorf("macOS config path incorrect: %s", path)
		}
	} else if runtime.GOOS == "windows" {
		if !strings.Contains(path, "Moly") {
			t.Errorf("Windows config path incorrect: %s", path)
		}
	}

	// Verify file ends with moly.config.json
	if !strings.HasSuffix(path, "moly.config.json") {
		t.Errorf("Config path doesn't end with moly.config.json: %s", path)
	}
}

func TestLoadConfigPrecedenceEnvOverFile(t *testing.T) {
	// Save env vars
	oldPort := os.Getenv("MOLY_PORT")

	defer func() {
		os.Setenv("MOLY_PORT", oldPort)
	}()

	// Environment variables should take precedence
	os.Setenv("MOLY_PORT", ":9999")

	config := LoadConfig()

	// Env var should override file (if file exists)
	if config.Port != ":9999" {
		t.Errorf("Expected env var to override, got %s", config.Port)
	}
}

func TestLoadConfigDatabasePathUsesPlatformDir(t *testing.T) {
	config := LoadConfig()

	// DatabasePath should be set and use the platform-specific directory
	if config.DatabasePath == "" {
		t.Errorf("DatabasePath not set in config")
	}

	// Verify it's in the correct OS-specific location
	if runtime.GOOS == "linux" {
		if !strings.Contains(config.DatabasePath, ".config") {
			t.Errorf("Linux database path not in .config: %s", config.DatabasePath)
		}
	} else if runtime.GOOS == "darwin" {
		if !strings.Contains(config.DatabasePath, "Library") {
			t.Errorf("macOS database path not in Library: %s", config.DatabasePath)
		}
	} else if runtime.GOOS == "windows" {
		if !strings.Contains(config.DatabasePath, "Moly") {
			t.Errorf("Windows database path not in Moly: %s", config.DatabasePath)
		}
	}

	// Should end with moly.db
	if !strings.HasSuffix(config.DatabasePath, "moly.db") {
		t.Errorf("Database path doesn't end with moly.db: %s", config.DatabasePath)
	}
}

func TestDefaultServerConfigValues(t *testing.T) {
	// Verify DefaultServerConfig struct has expected values
	if DefaultServerConfig.Port != ":11436" {
		t.Errorf("DefaultServerConfig.Port should be :11436, got %s", DefaultServerConfig.Port)
	}
	if DefaultServerConfig.Host != "127.0.0.1" {
		t.Errorf("DefaultServerConfig.Host should be 127.0.0.1, got %s", DefaultServerConfig.Host)
	}
	if DefaultServerConfig.LogLevel != "info" {
		t.Errorf("DefaultServerConfig.LogLevel should be info, got %s", DefaultServerConfig.LogLevel)
	}
	if DefaultServerConfig.CORSProxyPort != ":11435" {
		t.Errorf("DefaultServerConfig.CORSProxyPort should be :11435, got %s", DefaultServerConfig.CORSProxyPort)
	}
}
