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

var mdb *Database
var analytics *Analytics

func main() {
	log.SetFlags(log.Lshortfile)

	// Initialize config
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

	// Setup HTTP routes
	http.HandleFunc("/api/status", handleStatus)
	http.HandleFunc("/api/first-run-check", handleFirstRunCheck)
	http.HandleFunc("/api/providers", handleProviders)
	http.HandleFunc("/api/api-key", handleAPIKey)
	http.HandleFunc("/api/models/list", handleListModels)
	http.HandleFunc("/api/models/pull", handlePullModel)
	http.HandleFunc("/api/models/remove", handleRemoveModel)
	http.HandleFunc("/api/ollama/start", handleStartOllama)
	http.HandleFunc("/api/ollama/stop", handleStopOllama)
	http.HandleFunc("/api/settings", handleSettings)
	http.HandleFunc("/api/chat", handleChat)
	http.HandleFunc("/api/contacts", handleContacts)
	http.HandleFunc("/api/contacts/delete", handleDeleteContact)
	http.HandleFunc("/api/interactions", handleInteractions)
	http.HandleFunc("/api/draft-message", handleDraftMessage)
	http.HandleFunc("/api/log-conversation", handleLogConversation)
	http.HandleFunc("/api/analyze-context", handleAnalyzeContext)
	http.HandleFunc("/api/extract-insights", handleExtractInsights)
	http.HandleFunc("/api/analytics/contacts", handleAnalyticsContacts)
	http.HandleFunc("/api/analytics/topics", handleAnalyticsTopics)
	http.HandleFunc("/api/analytics/tone", handleAnalyticsTone)
	http.HandleFunc("/api/analytics/summary", handleAnalyticsSummary)
	http.HandleFunc("/api/analytics/patterns", handleAnalyticsPatterns)
	http.HandleFunc("/api/analyze-mode-shift", handleAnalyzeModeShift)
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

func handleAnalyzeContext(w http.ResponseWriter, r *http.Request) {
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

	platformVal, ok := req["platform"].(string)
	if !ok {
		respondError(w, http.StatusBadRequest, "platform required")
		return
	}

	messageStartVal, ok := req["message_start"].(string)
	if !ok {
		respondError(w, http.StatusBadRequest, "message_start required")
		return
	}

	config := loadConfig()
	qa := NewQuestionAgent(mdb, &config, config.Provider)

	analysis, err := qa.AnalyzeContext(contactID, platformVal, messageStartVal)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":  true,
		"analysis": analysis,
	})
}

func handleExtractInsights(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	contactID := int(req["contact_id"].(float64))
	userMessage := req["user_message"].(string)
	molyResponse := req["moly_response"].(string)

	config := loadConfig()
	qa := NewQuestionAgent(mdb, &config, config.Provider)

	insights, err := qa.ExtractInsights(contactID, userMessage, molyResponse)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":  true,
		"insights": insights,
	})
}

func handleAnalyticsContacts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	stats, err := analytics.GetContactStats()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"contacts": stats,
	})
}

func handleAnalyticsTopics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	stats, err := analytics.GetTopicStats()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"topics": stats,
	})
}

func handleAnalyticsTone(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	daysStr := r.URL.Query().Get("days")
	days := 7
	if daysStr != "" {
		fmt.Sscanf(daysStr, "%d", &days)
	}

	stats, err := analytics.GetToneStats(days)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"tones": stats,
	})
}

func handleAnalyticsSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	summary, err := analytics.GetSummary()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"summary": summary,
	})
}

func handleAnalyticsPatterns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	patterns, err := analytics.GetCommunicationPatterns()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"patterns": patterns,
	})
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

func handleDraftMessage(w http.ResponseWriter, r *http.Request) {
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

	intentionVal, ok := req["intention"].(string)
	if !ok {
		respondError(w, http.StatusBadRequest, "intention required")
		return
	}

	draftVal, ok := req["draft"].(string)
	if !ok {
		respondError(w, http.StatusBadRequest, "draft required")
		return
	}

	// Get contact info
	contact, err := mdb.getContact(contactID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Contact not found")
		return
	}

	// Get recent interactions for context
	interactions, err := mdb.getRecentInteractions(contactID, 5)
	if err != nil {
		interactions = []Interaction{}
	}

	// Build context about the contact
	contextStr := fmt.Sprintf(`Contact Profile:
- Name: %s
- Relationship: %s
- Platform: %s
- Notes: %s
- Interaction Count: %d
- Last Interaction: %v

User's Intention: %s

Current Draft: "%s"

Recent Interaction Topics:`,
		contact.Name, contact.Relationship, contact.Platform, contact.Notes,
		contact.InteractionCount, contact.LastInteraction,
		intentionVal, draftVal)

	for i, inter := range interactions {
		if i > 3 {
			break
		}
		contextStr += fmt.Sprintf("\n- %s (Tone: %s)", inter.Topic, inter.Sentiment)
	}

	// Ask LLM to help craft the message
	prompt := fmt.Sprintf(`You are helping someone craft a message to send to %s.

%s

Your job is to:
1. Understand the user's intention
2. Consider the contact's personality and communication style
3. Review the user's current draft
4. Suggest improvements to make it better

Please provide:
- Analysis of the current draft (what works, what could be improved)
- 2-3 alternative versions of the message
- Key points to consider based on the relationship history

Be conversational and helpful. Consider tone, clarity, and the relationship context.`,
		contact.Name, contextStr)

	config := loadConfig()

	var response string
	switch config.Provider {
	case "local":
		response, err = chatWithOllama(prompt, config.Model, "direct")
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

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"contact": contact.Name,
		"suggestion": response,
	})
}

func handleLogConversation(w http.ResponseWriter, r *http.Request) {
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

	userMessageVal, ok := req["user_message"].(string)
	if !ok {
		respondError(w, http.StatusBadRequest, "user_message required")
		return
	}

	contactReplyVal, ok := req["contact_reply"].(string)
	if !ok {
		respondError(w, http.StatusBadRequest, "contact_reply required")
		return
	}

	platformVal, ok := req["platform"].(string)
	if !ok {
		platformVal = "unknown"
	}

	// Get contact
	contact, err := mdb.getContact(contactID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Contact not found")
		return
	}

	// Extract insights from exchange
	config := loadConfig()
	qa := NewQuestionAgent(mdb, &config, config.Provider)

	insights, err := qa.ExtractInsights(contactID, userMessageVal, contactReplyVal)
	if err != nil {
		insights = &InsightExtraction{
			ToneDetected: "unknown",
			Topics: []string{},
			CommunicationStyle: "unknown",
		}
	}

	// Record user's message
	err = mdb.recordInteraction(contactID, platformVal, "initial message", "user-sent",
		fmt.Sprintf("User: %s", userMessageVal), userMessageVal)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to record message: %v", err))
		return
	}

	// Record contact's reply
	err = mdb.recordInteraction(contactID, platformVal, "reply", insights.ToneDetected,
		fmt.Sprintf("%s: %s", contact.Name, contactReplyVal), contactReplyVal)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to record reply: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"contact": contact.Name,
		"message": "Conversation logged successfully",
		"insights": insights,
		"interactions_recorded": 2,
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

	contactIDVal, ok := req["contact_id"]
	if !ok {
		respondError(w, http.StatusBadRequest, "contact_id required")
		return
	}
	contactID := int(contactIDVal.(float64))

	currentModeVal, ok := req["current_mode"].(string)
	if !ok {
		respondError(w, http.StatusBadRequest, "current_mode required")
		return
	}

	desiredModeVal, ok := req["desired_mode"].(string)
	if !ok {
		respondError(w, http.StatusBadRequest, "desired_mode required")
		return
	}

	contextVal, ok := req["context"].(string)
	if !ok {
		contextVal = ""
	}

	engine := NewModeTransitionEngine(mdb)
	analysis, err := engine.AnalyzeModeShift(contactID, currentModeVal, desiredModeVal, contextVal)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"analysis": analysis,
	})
}
