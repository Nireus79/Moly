package main

import (
	"encoding/json"
	"net/http"
)

type ProviderInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Available   bool   `json:"available"`
	Active      bool   `json:"active"`
}

type ProvidersResponse struct {
	Providers []ProviderInfo `json:"providers"`
	Active    string         `json:"active"`
}

func handleProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	config := loadConfig()

	// Check availability of each provider
	localAvailable := checkLocalAvailable()
	claudeAvailable := checkClaudeAvailable(config)
	openaiAvailable := checkOpenAIAvailable(config)

	providers := []ProviderInfo{
		{
			ID:          "local",
			Name:        "Local Models (Ollama)",
			Description: "Run models locally, no API key needed",
			Available:   localAvailable,
			Active:      config.Provider == "local",
		},
		{
			ID:          "claude",
			Name:        "Claude (Anthropic)",
			Description: "Fast and intelligent responses",
			Available:   claudeAvailable,
			Active:      config.Provider == "claude",
		},
		{
			ID:          "openai",
			Name:        "ChatGPT (OpenAI)",
			Description: "Powerful language model",
			Available:   openaiAvailable,
			Active:      config.Provider == "openai",
		},
	}

	respondJSON(w, http.StatusOK, ProvidersResponse{
		Providers: providers,
		Active:    config.Provider,
	})
}

func handleAPIKey(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var req map[string]string
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid JSON")
			return
		}

		provider := req["provider"]
		apiKey := req["api_key"]

		if provider == "" || apiKey == "" {
			respondError(w, http.StatusBadRequest, "Provider and api_key required")
			return
		}

		config := loadConfig()

		// Encrypt and store the API key
		encrypted, err := encryptString(apiKey)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to encrypt key")
			return
		}

		if config.APIKeys == nil {
			config.APIKeys = make(map[string]interface{})
		}
		config.APIKeys[provider] = encrypted

		if err := saveConfig(config); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "API key saved",
		})
	} else {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func checkLocalAvailable() bool {
	installed, running := checkOllama()
	return installed && running
}

func checkClaudeAvailable(config Config) bool {
	if apiKey, ok := config.APIKeys["claude"].(string); ok {
		return apiKey != ""
	}
	return false
}

func checkOpenAIAvailable(config Config) bool {
	if apiKey, ok := config.APIKeys["openai"].(string); ok {
		return apiKey != ""
	}
	return false
}

func getPreferredProvider() string {
	config := loadConfig()

	// Check if current provider is available
	switch config.Provider {
	case "local":
		if checkLocalAvailable() {
			return "local"
		}
	case "claude":
		if checkClaudeAvailable(config) {
			return "claude"
		}
	case "openai":
		if checkOpenAIAvailable(config) {
			return "openai"
		}
	}

	// Try preferred order: local -> claude -> openai
	if checkLocalAvailable() {
		return "local"
	}
	if checkClaudeAvailable(config) {
		return "claude"
	}
	if checkOpenAIAvailable(config) {
		return "openai"
	}

	// Fallback to current provider even if unavailable
	return config.Provider
}
