package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	Version          string                 `json:"version"`
	Provider         string                 `json:"provider"`
	Model            string                 `json:"model"`
	OllamaInstalled  bool                   `json:"ollama_installed"`
	OllamaRunning    bool                   `json:"ollama_running"`
	InstalledModels  []interface{}          `json:"installed_models"`
	APIKeys          map[string]interface{} `json:"api_keys"`
	FirstRunComplete bool                   `json:"first_run_complete"`
	Tone             string                 `json:"tone"`
	Mode             string                 `json:"mode"`
	CreatedAt        string                 `json:"created_at"`
	UpdatedAt        string                 `json:"updated_at"`
}

func getDefaultConfig() Config {
	return Config{
		Version:          "1.0",
		Provider:         "local",
		Model:            "mistral",
		OllamaInstalled:  false,
		OllamaRunning:    false,
		InstalledModels:  []interface{}{},
		APIKeys:          make(map[string]interface{}),
		FirstRunComplete: false,
		Tone:             "friendly",
		Mode:             "direct",
		CreatedAt:        time.Now().Format(time.RFC3339),
		UpdatedAt:        time.Now().Format(time.RFC3339),
	}
}

func initConfig() error {
	configPath := getConfigPath()
	configDir := filepath.Dir(configPath)

	// Create config directory
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	// Create default config if doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config := getDefaultConfig()
		if err := saveConfig(config); err != nil {
			return err
		}
	}

	return nil
}

func loadConfig() Config {
	configPath := getConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		return getDefaultConfig()
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return getDefaultConfig()
	}

	return config
}

func saveConfig(config Config) error {
	config.UpdatedAt = time.Now().Format(time.RFC3339)
	configPath := getConfigPath()

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}
