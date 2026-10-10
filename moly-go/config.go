package main

import (
	"encoding/json"
	"log"
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
	if data, err := os.ReadFile(configFile); err != nil {
		log.Printf("[ServerConfig] Note: No config file at %s, using defaults (this is normal on first run)", configFile)
	} else {
		var fileConfig ServerConfig
		if err := json.Unmarshal(data, &fileConfig); err != nil {
			log.Printf("[ServerConfig] ERROR: Config file at %s is malformed: %v, using defaults", configFile, err)
		} else {
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






// getConfigDir returns the platform-specific config directory for a given filename
func getConfigDir(filename string) string {
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

	return filepath.Join(configDir, filename)
}
