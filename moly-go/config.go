package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

type ServerConfig struct {
	Port           string `json:"port"`
	Host           string `json:"host"`
	DatabasePath   string `json:"database_path"`
	LogLevel       string `json:"log_level"`
	CORSProxyPort  string `json:"cors_proxy_port"`
	OllamaEndpoint string `json:"ollama_endpoint"`
}

var DefaultServerConfig = ServerConfig{
	Port:           ":11436",
	Host:           "127.0.0.1",
	LogLevel:       "info",
	CORSProxyPort:  ":11435",
	OllamaEndpoint: "http://127.0.0.1:11434",
}

func LoadConfig() ServerConfig {
	config := DefaultServerConfig

	// 1. Config file (middle priority - overrides defaults)
	configFile := getConfigFilePath()
	if data, err := os.ReadFile(configFile); err == nil {
		var fileConfig ServerConfig
		if err := json.Unmarshal(data, &fileConfig); err == nil {
			// Apply file config to defaults
			if fileConfig.Port != "" {
				config.Port = fileConfig.Port
			}
			if fileConfig.Host != "" {
				config.Host = fileConfig.Host
			}
			if fileConfig.LogLevel != "" {
				config.LogLevel = fileConfig.LogLevel
			}
			if fileConfig.CORSProxyPort != "" {
				config.CORSProxyPort = fileConfig.CORSProxyPort
			}
			if fileConfig.OllamaEndpoint != "" {
				config.OllamaEndpoint = fileConfig.OllamaEndpoint
			}
		}
	}

	// 2. Environment variables (highest priority - override file and defaults)
	if port := os.Getenv("MOLY_PORT"); port != "" {
		config.Port = port
	}
	if host := os.Getenv("MOLY_HOST"); host != "" {
		config.Host = host
	}
	if level := os.Getenv("MOLY_LOG_LEVEL"); level != "" {
		config.LogLevel = level
	}
	if proxyPort := os.Getenv("MOLY_CORS_PROXY_PORT"); proxyPort != "" {
		config.CORSProxyPort = proxyPort
	}

	// 3. Database path (always use platform-specific directory)
	config.DatabasePath = getConfigDir("moly.db")

	return config
}

func getConfigFilePath() string {
	var configDir string

	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = os.ExpandEnv("$USERPROFILE\\AppData\\Roaming")
		}
		configDir = filepath.Join(appData, "Moly")

	case "darwin":
		configDir = filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "Moly")

	default:
		if xdgHome := os.Getenv("XDG_CONFIG_HOME"); xdgHome != "" {
			configDir = filepath.Join(xdgHome, "moly")
		} else {
			configDir = filepath.Join(os.Getenv("HOME"), ".config", "moly")
		}
	}

	return filepath.Join(configDir, "moly.config.json")
}

// Config struct for application settings (Provider, Model, etc.) - kept for backward compatibility
type Config struct {
	Version            string                 `json:"version"`
	Provider           string                 `json:"provider"`
	Model              string                 `json:"model"`
	Tone               string                 `json:"tone"`
	Mode               string                 `json:"mode"`
	FirstRunComplete   bool                   `json:"first_run_complete"`
	OllamaInstalled    bool                   `json:"ollama_installed"`
	OllamaRunning      bool                   `json:"ollama_running"`
	APIKeys            map[string]interface{} `json:"api_keys"`
	InstalledModels    []string               `json:"installed_models"`
	CreatedAt          string                 `json:"created_at"`
	UpdatedAt          string                 `json:"updated_at"`
}

// getDefaultConfig returns default application configuration
func getDefaultConfig() Config {
	return Config{
		Version:          "1.0",
		Provider:         "local",
		Model:            "mistral",
		Tone:             "friendly",
		Mode:             "direct",
		FirstRunComplete: false,
		OllamaInstalled:  false,
		OllamaRunning:    false,
		APIKeys:          make(map[string]interface{}),
		InstalledModels:  []string{},
	}
}

// loadConfig loads application configuration from file
func loadConfig() Config {
	configPath := getConfigPath()
	config := getDefaultConfig()

	data, err := os.ReadFile(configPath)
	if err != nil {
		return config
	}

	json.Unmarshal(data, &config)
	return config
}

// saveConfig saves application configuration to file
func saveConfig(config Config) error {
	configPath := getConfigPath()
	configDir := filepath.Dir(configPath)

	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}

// initConfig initializes config file if it doesn't exist
func initConfig() error {
	configPath := getConfigPath()

	if _, err := os.Stat(configPath); err == nil {
		return nil
	}

	config := getDefaultConfig()
	return saveConfig(config)
}
