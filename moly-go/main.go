package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

const (
	Port = ":11436"
	Host = "127.0.0.1"
)

func main() {
	log.SetFlags(log.Lshortfile)

	// Initialize config
	if err := initConfig(); err != nil {
		log.Fatalf("Failed to initialize config: %v", err)
	}

	// Setup HTTP routes
	http.HandleFunc("/api/status", handleStatus)
	http.HandleFunc("/api/first-run-check", handleFirstRunCheck)
	http.HandleFunc("/api/models/list", handleListModels)
	http.HandleFunc("/api/models/pull", handlePullModel)
	http.HandleFunc("/api/models/remove", handleRemoveModel)
	http.HandleFunc("/api/ollama/start", handleStartOllama)
	http.HandleFunc("/api/ollama/stop", handleStopOllama)
	http.HandleFunc("/api/settings", handleSettings)
	http.HandleFunc("/api/chat", handleChat)
	http.HandleFunc("/sidebar.html", handleSidebarHTML)
	http.HandleFunc("/", handleRoot)

	// Start server
	addr := Host + Port
	log.Printf("[Moly] Desktop app initialized")
	log.Printf("[Moly] Sidebar server listening on %s%s", Host, Port)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

// API Handlers

func handleStatus(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "running"})
}

func handleFirstRunCheck(w http.ResponseWriter, r *http.Request) {
	installed, running := checkOllama()
	config := loadConfig()
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"ollama_installed":     installed,
		"ollama_running":       running,
		"first_run_complete":   config.FirstRunComplete,
	})
}

func handleListModels(w http.ResponseWriter, r *http.Request) {
	models, err := getOllamaModels()
	if err != nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"models": []interface{}{},
			"error":  err.Error(),
		})
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"models": models,
		"error":  nil,
	})
}

func handlePullModel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	modelName := r.URL.Query().Get("name")
	if modelName == "" {
		var req map[string]string
		json.NewDecoder(r.Body).Decode(&req)
		modelName = req["name"]
	}

	if modelName == "" {
		respondError(w, http.StatusBadRequest, "Model name required")
		return
	}

	err := pullOllamaModel(modelName)
	if err != nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"error":   nil,
	})
}

func handleRemoveModel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	modelName := r.URL.Query().Get("name")
	if modelName == "" {
		var req map[string]string
		json.NewDecoder(r.Body).Decode(&req)
		modelName = req["name"]
	}

	if modelName == "" {
		respondError(w, http.StatusBadRequest, "Model name required")
		return
	}

	err := removeOllamaModel(modelName)
	if err != nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"error":   nil,
	})
}

func handleStartOllama(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	err := startOllama()
	if err != nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Ollama started",
	})
}

func handleStopOllama(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	err := stopOllama()
	if err != nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Ollama stopped",
	})
}

func handleSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		config := loadConfig()
		respondJSON(w, http.StatusOK, config)
	} else if r.Method == http.MethodPost {
		var updates map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid JSON")
			return
		}

		config := loadConfig()
		// Merge updates
		for key, value := range updates {
			switch key {
			case "provider":
				if s, ok := value.(string); ok {
					config.Provider = s
				}
			case "model":
				if s, ok := value.(string); ok {
					config.Model = s
				}
			case "tone":
				if s, ok := value.(string); ok {
					config.Tone = s
				}
			case "mode":
				if s, ok := value.(string); ok {
					config.Mode = s
				}
			case "api_keys":
				if m, ok := value.(map[string]interface{}); ok {
					config.APIKeys = m
				}
			}
		}

		if err := saveConfig(config); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"config":  config,
		})
	} else {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func handleSidebarHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	fmt.Fprint(w, sidebarHTML)
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"message": "Moly Desktop App"})
}

func getConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "moly", "config.json")
}
