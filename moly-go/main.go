package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

var mdb *Database
var analytics *Analytics
var safetyChecker *SafetyChecker
var proxyCmd *exec.Cmd

// findCORSProxyScript locates the CORS proxy script using the following strategy:
// 1. Check MOLY_PROXY_PATH environment variable (highest priority)
// 2. Check relative to binary location (for packaged installations)
// 3. Check relative to current working directory (for development)
// 4. Check standard installation directories (Linux /opt, Windows Program Files, macOS /Applications)
// Returns error if script not found in any location
func findCORSProxyScript() (string, error) {
	// Strategy 1: Check if MOLY_PROXY_PATH is set (highest priority - explicit user override)
	if proxyPath := os.Getenv("MOLY_PROXY_PATH"); proxyPath != "" {
		if _, err := os.Stat(proxyPath); err == nil {
			return proxyPath, nil
		}
		return "", fmt.Errorf("MOLY_PROXY_PATH set but file not found: %s", proxyPath)
	}

	// Strategy 2: Check relative to binary location (../moly-proxy/bin/moly-proxy.js)
	exePath, err := os.Executable()
	if err == nil {
		projectRoot := filepath.Join(filepath.Dir(exePath), "..", "..")
		candidate := filepath.Join(projectRoot, "moly-proxy", "bin", "moly-proxy.js")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	// Strategy 3: Check relative to current working directory
	candidates := []string{
		"moly-proxy/bin/moly-proxy.js",
		"../moly-proxy/bin/moly-proxy.js",
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	// Strategy 4: Check standard installation directories by OS
	var installCandidates []string
	switch runtime.GOOS {
	case "linux":
		installCandidates = []string{
			"/opt/moly/moly-proxy.js",
			"/opt/moly/bin/moly-proxy.js",
			filepath.Join(os.ExpandEnv("$HOME"), ".local", "share", "moly", "moly-proxy.js"),
		}
	case "darwin":
		installCandidates = []string{
			"/Applications/Moly/moly-proxy.js",
			"/usr/local/opt/moly/moly-proxy.js",
		}
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = os.ExpandEnv("$USERPROFILE\\AppData\\Roaming")
		}
		installCandidates = []string{
			filepath.Join(appData, "Moly", "moly-proxy.js"),
			"C:\\Program Files\\Moly\\moly-proxy.js",
			"C:\\Program Files (x86)\\Moly\\moly-proxy.js",
		}
	}

	for _, candidate := range installCandidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("CORS proxy script not found. Tried multiple locations. Set MOLY_PROXY_PATH environment variable to specify location")
}

func startCORSProxy() error {
	proxyScript, err := findCORSProxyScript()
	if err != nil {
		return fmt.Errorf("CORS proxy: %v", err)
	}

	// Start CORS Proxy with Node.js
	proxyCmd = exec.Command("node", proxyScript)
	proxyCmd.Stdout = os.Stdout
	proxyCmd.Stderr = os.Stderr

	if err := proxyCmd.Start(); err != nil {
		return fmt.Errorf("failed to start CORS proxy: %v", err)
	}

	log.Printf("[Moly] CORS Proxy started (PID %d)", proxyCmd.Process.Pid)

	// Wait a moment for proxy to become ready
	time.Sleep(500 * time.Millisecond)

	// Check if proxy is responding
	for i := 0; i < 5; i++ {
		resp, err := http.Get("http://127.0.0.1:11435/api/tags")
		if err == nil {
			resp.Body.Close()
			log.Printf("[Moly] CORS Proxy responding on http://127.0.0.1:11435")
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}

	return fmt.Errorf("CORS proxy started but not responding on port 11435")
}

func main() {
	log.SetFlags(log.Lshortfile)

	// Load configuration from environment, config file, or defaults
	config := LoadConfig()

	// Initialize legacy config (backward compatibility)
	if err := initConfig(); err != nil {
		log.Fatalf("Failed to initialize config: %v", err)
	}

	// Initialize database
	var err error
	mdb, err = initDatabase()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer mdb.close()

	// Initialize analytics
	analytics = NewAnalytics(mdb)

	// Initialize safety checker
	safetyChecker = NewSafetyChecker()

	// Start CORS Proxy (auto-start for browser communication)
	if err := startCORSProxy(); err != nil {
		log.Printf("[Moly] WARNING: Could not start CORS Proxy: %v", err)
		log.Printf("[Moly] Continuing without CORS proxy - extension will try direct Ollama communication")
	} else {
		log.Printf("[Moly] CORS Proxy ready for browser requests")
	}

	// Setup HTTP routes - only endpoints used by extension
	http.HandleFunc("/api/status", handleStatus)
	http.HandleFunc("/api/providers", handleProviders)
	http.HandleFunc("/api/models/list", handleListModels)
	http.HandleFunc("/api/models/pull", handlePullModel)
	http.HandleFunc("/api/models/remove", handleRemoveModel)
	http.HandleFunc("/api/ollama/start", handleStartOllama)
	http.HandleFunc("/api/ollama/stop", handleStopOllama)
	http.HandleFunc("/api/settings", handleSettings)
	http.HandleFunc("/api/contacts", handleContacts)
	http.HandleFunc("/api/contacts/delete", handleDeleteContact)
	http.HandleFunc("/api/interactions", handleInteractions)
	http.HandleFunc("/api/analyze-mode-shift", handleAnalyzeModeShift)
	http.HandleFunc("/api/check-safety", handleCheckSafety)
	http.HandleFunc("/api/evaluate-constitution", handleEvaluateConstitution)
	http.HandleFunc("/api/generate-questions", handleGenerateQuestions)
	http.HandleFunc("/api/constitution-principles", handleGetPrinciples)
	http.HandleFunc("/api/conversations", handleConversations)
	http.HandleFunc("/api/conversations/context", handleConversationContext)
	http.HandleFunc("/sidebar.html", handleSidebarHTML)
	http.HandleFunc("/", handleRoot)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("[Moly] Received shutdown signal")

		// Stop CORS proxy
		if proxyCmd != nil && proxyCmd.Process != nil {
			log.Println("[Moly] Stopping CORS Proxy...")
			proxyCmd.Process.Kill()
			proxyCmd.Wait()
		}

		// Cleanup database
		if mdb != nil {
			mdb.close()
		}

		log.Println("[Moly] Shutting down cleanly")
		os.Exit(0)
	}()

	// Start server
	addr := config.Host + config.Port
	log.Printf("[Moly] Desktop app initialized")
	log.Printf("[Moly] Sidebar server listening on %s%s", config.Host, config.Port)
	log.Printf("[Moly] Ready: Go backend (%s) + CORS Proxy (%s)", config.Port, config.CORSProxyPort)

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

// Contact handlers

func handleContacts(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// Get all contacts
		contacts, err := mdb.getAllContacts()
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"contacts": contacts,
		})
	} else if r.Method == http.MethodPost {
		// Create or update contact
		var req map[string]string
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid JSON")
			return
		}

		name := req["name"]
		if name == "" {
			respondError(w, http.StatusBadRequest, "Name required")
			return
		}

		contact, err := mdb.createOrUpdateContact(
			name,
			req["relationship"],
			req["platform"],
			req["notes"],
		)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"contact": contact,
		})
	} else {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func handleInteractions(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		// Record interaction
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid JSON")
			return
		}

		contactID := int(req["contact_id"].(float64))
		platform := req["platform"].(string)
		topic := req["topic"].(string)
		sentiment := req["sentiment"].(string)
		summary := req["ai_summary"].(string)
		notes := req["user_notes"].(string)

		err := mdb.recordInteraction(contactID, platform, topic, sentiment, summary, notes)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
		})
	} else if r.Method == http.MethodGet {
		// Get recent interactions for a contact
		contactIDStr := r.URL.Query().Get("contact_id")
		if contactIDStr == "" {
			respondError(w, http.StatusBadRequest, "contact_id required")
			return
		}

		var contactID int
		_, err := fmt.Sscanf(contactIDStr, "%d", &contactID)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid contact_id")
			return
		}

		limit := 10
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			fmt.Sscanf(limitStr, "%d", &limit)
		}

		interactions, err := mdb.getRecentInteractions(contactID, limit)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"interactions": interactions,
		})
	} else {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func handleDeleteContact(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	contactIDVal, ok := req["contact_id"]
	if !ok {
		respondError(w, http.StatusBadRequest, "contact_id required")
		return
	}
	contactID := int(contactIDVal.(float64))

	// Delete all interactions for this contact first
	_, err := mdb.conn.Exec("DELETE FROM interactions WHERE contact_id = ?", contactID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete interactions: %v", err))
		return
	}

	// Delete the contact
	_, err = mdb.conn.Exec("DELETE FROM contacts WHERE id = ?", contactID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete contact: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Contact deleted successfully",
	})
}

func handleAnalyzeModeShift(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	currentModeVal, ok := req["current_mode"].(string)
	if !ok {
		respondError(w, http.StatusBadRequest, "current_mode required")
		return
	}

	proposedModeVal, ok := req["proposed_mode"].(string)
	if !ok {
		respondError(w, http.StatusBadRequest, "proposed_mode required")
		return
	}

	contextVal, ok := req["context"].(string)
	if !ok {
		contextVal = ""
	}

	engine := NewModeTransitionEngine(mdb)
	analysis, err := engine.AnalyzeModeShift(0, currentModeVal, proposedModeVal, contextVal)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, analysis)
}

func handleCheckSafety(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	messageVal, ok := req["message"].(string)
	if !ok {
		respondError(w, http.StatusBadRequest, "message required")
		return
	}

	alert := safetyChecker.CheckMessage(messageVal)

	if alert == nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"alert_type":      "none",
			"severity":        "",
			"title":           "Safe",
			"message":         "No safety concerns detected",
			"indicators":      []string{},
			"resources":       []interface{}{},
			"recommendations": []string{},
		})
		return
	}

	respondJSON(w, http.StatusOK, alert)
}

func handleEvaluateConstitution(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	messageVal, ok := req["message"].(string)
	if !ok {
		respondError(w, http.StatusBadRequest, "message required")
		return
	}

	evaluator := NewConstitutionEvaluator()
	analysis := evaluator.EvaluateAction(messageVal)

	respondJSON(w, http.StatusOK, analysis)
}

func handleGenerateQuestions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	contactNameVal, ok := req["contact_name"].(string)
	if !ok {
		respondError(w, http.StatusBadRequest, "contact_name required")
		return
	}

	contextVal, ok := req["context"].(string)
	if !ok {
		contextVal = ""
	}

	config := loadConfig()

	// Get model from request or use default
	modelVal, ok := req["model"].(string)
	if !ok || modelVal == "" {
		modelVal = config.Model
		if modelVal == "" {
			// Default to mistral if nothing configured
			modelVal = "mistral:latest"
		}
	}

	prompt := fmt.Sprintf(`Based on the following context about a conversation with %s, generate 3-5 thoughtful questions to help the user craft a better message.

Context: %s

Generate questions that help the user:
1. Clarify their intention
2. Consider the other person's perspective
3. Reflect on the relationship dynamics
4. Plan for different responses

Format as a JSON response with:
- questions: array of question strings
- context: brief summary of context understood
- reasoning: why these questions matter`, contactNameVal, contextVal)

	var response string
	var err error
	switch config.Provider {
	case "local":
		response, err = chatWithOllama(prompt, modelVal, "direct")
	case "claude":
		response, err = chatWithClaude(prompt, config.Model, "direct")
	case "openai":
		response, err = chatWithOpenAI(prompt, config.Model, "direct")
	default:
		err = fmt.Errorf("provider not configured")
	}

	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("LLM error: %v", err))
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		result = map[string]interface{}{
			"questions": []string{response},
			"context":   contextVal,
			"reasoning": "Generated from LLM response",
		}
	}

	respondJSON(w, http.StatusOK, result)
}

func handleGetPrinciples(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	evaluator := NewConstitutionEvaluator()
	principles := evaluator.GetPrinciples()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":           true,
		"supreme_principle": evaluator.GetSupremePrinciple(),
		"principles":        principles,
	})
}

// handleConversations handles POST /api/conversations (create)
func handleConversations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Name       string `json:"name"`
		Type       string `json:"type"`
		Purpose    string `json:"purpose"`
		Notes      string `json:"notes"`
		ContactIDs []int  `json:"contact_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// Create conversation in database
	conv, err := mdb.createConversation(req.Name, req.Type, req.Purpose, req.Notes, req.ContactIDs)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create conversation: "+err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"success":     true,
		"conversation": conv,
	})
}

// handleConversationContext handles GET /api/conversations/context?id=<id>
func handleConversationContext(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		ConversationID  int  `json:"conversation_id"`
		IncludeHistory  bool `json:"include_history"`
	}

	if r.Method == http.MethodPost {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request: "+err.Error())
			return
		}
	} else {
		// GET: parse from query
		conversationID := r.URL.Query().Get("id")
		if conversationID == "" {
			respondError(w, http.StatusBadRequest, "Missing conversation_id parameter")
			return
		}
		fmt.Sscanf(conversationID, "%d", &req.ConversationID)
	}

	if req.ConversationID == 0 {
		respondError(w, http.StatusBadRequest, "Invalid conversation_id")
		return
	}

	// Fetch conversation
	conv, err := mdb.getConversation(req.ConversationID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Conversation not found")
		return
	}

	// Fetch members
	members, err := mdb.getConversationMembers(req.ConversationID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch members: "+err.Error())
		return
	}

	// Build context response
	contextResp := map[string]interface{}{
		"success": true,
		"conversation": map[string]interface{}{
			"id":       conv.ID,
			"name":     conv.Name,
			"type":     conv.Type,
			"purpose":  conv.Purpose,
			"notes":    conv.Notes,
		},
		"members": members,
	}

	// Include recent interactions if requested
	if req.IncludeHistory {
		interactions, err := mdb.getConversationInteractions(req.ConversationID)
		if err == nil && len(interactions) > 0 {
			contextResp["recent_interactions"] = interactions
		}
	}

	respondJSON(w, http.StatusOK, contextResp)
}
