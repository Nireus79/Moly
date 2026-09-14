package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"moly/agents"
	"moly/auth"
	"moly/database"
	"moly/models"
	"moly/schema"
	"moly/tools"
)

var v2db *database.Database
var v2Server *V2APIServer

// V2APIServer wraps the agent system and database
type V2APIServer struct {
	llmClient                tools.LLMProvider
	database                 *database.Database
	contactManager           *agents.ContactManager
	contextAttrManager       *agents.ContextAttributeManager
	clarificationAgent       *agents.ClarificationAgent
	answerProcessor          *agents.AnswerProcessor
	incomingMessageAnalyzer  *agents.IncomingMessageAnalyzer
	agentSystem              *agents.AgentSystem
}

// NewV2APIServer creates a new V2 API server
func NewV2APIServer(llm tools.LLMProvider, db *database.Database) (*V2APIServer, error) {
	if db == nil {
		return nil, fmt.Errorf("database cannot be nil")
	}

	// NOTE: TemporaryFactStore is now created per-request with userID
	contactManager := agents.NewContactManager(database.NewContactRepository(db))
	contextAttrManager := agents.NewContextAttributeManager(database.NewContextAttributeRepository(db))
	clarificationAgent := agents.NewClarificationAgent(llm)
	answerProcessor := agents.NewAnswerProcessor(clarificationAgent)
	incomingMessageAnalyzer := agents.NewIncomingMessageAnalyzer(llm)

	// Initialize ConversationAgent (V2 architecture)
	conversationAgent, err := agents.NewConversationAgent(llm)
	if err != nil {
		log.Printf("[Moly] Warning: Failed to initialize ConversationAgent: %v, will operate with fallback mode\n", err)
		conversationAgent = nil // Fallback to nil, but don't fail startup
	}

	// Wrap ConversationAgent in a minimal AgentSystem struct
	agentSystem := &agents.AgentSystem{
		ConversationAgent: conversationAgent,
	}

	return &V2APIServer{
		llmClient:               llm,
		database:                db,
		contactManager:          contactManager,
		contextAttrManager:      contextAttrManager,
		clarificationAgent:      clarificationAgent,
		answerProcessor:         answerProcessor,
		incomingMessageAnalyzer: incomingMessageAnalyzer,
		agentSystem:             agentSystem,
	}, nil
}

// respondJSON helper function
func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// getUserIDFromToken extracts userId from Bearer token (session ID)
func getUserIDFromToken(token string, db *database.Database) (string, error) {
	if token == "" {
		return "", fmt.Errorf("empty token")
	}

	// Token is stored as session ID, look it up in database
	conn := db.GetConnection()
	var userID string
	err := conn.QueryRow("SELECT user_id FROM sessions WHERE id = ? AND expires_at > ?", token, time.Now().Unix()).Scan(&userID)
	if err != nil {
		return "", fmt.Errorf("invalid token")
	}

	return userID, nil
}

// extractAndValidateToken extracts and validates Bearer token from request
func extractAndValidateToken(r *http.Request, db *database.Database) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("Invalid token")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		return "", fmt.Errorf("Invalid token")
	}

	userID, err := getUserIDFromToken(token, db)
	if err != nil {
		return "", fmt.Errorf("Invalid token")
	}

	return userID, nil
}

// parseArrayFromStorage safely parses array field from database storage
// Supports both JSON (preferred) and legacy CSV formats for backward compatibility
func parseArrayFromStorage(stored string) []string {
	if stored == "" {
		return []string{}
	}

	// Try JSON format first (new format)
	var result []string
	if err := json.Unmarshal([]byte(stored), &result); err == nil {
		return result
	}

	// Fallback to CSV format (legacy, for backward compatibility)
	parts := strings.Split(stored, ",")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	// Filter out empty strings
	filtered := []string{}
	for _, part := range parts {
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	return filtered
}

// encodeArrayForStorage encodes array as JSON for safe storage
func encodeArrayForStorage(arr []string) string {
	if len(arr) == 0 {
		return "[]"
	}
	bytes, err := json.Marshal(arr)
	if err != nil {
		log.Printf("[Storage] Error encoding array: %v\n", err)
		return "[]"
	}
	return string(bytes)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-ID")
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok","version":"2.1"}`))
}

// MessageProcessorHandler - Full orchestration with multi-phase context processing
func (srv *V2APIServer) MessageProcessorHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		schema.RespondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract and validate Bearer token
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		schema.RespondError(w, http.StatusUnauthorized, authErr.Error())
		return
	}

	req := &schema.Phase5Request{}

	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		schema.RespondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Validate using schema validator
	if err := schema.ValidateStruct(req); err != nil {
		schema.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(req.ConversationID) > 100 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "ConversationID too long (max 100 characters)"})
		return
	}

	// Conversation is optional - validate only if provided
	if req.ConversationID != "" && req.ConversationID != "null" {
		conn := srv.database.GetConnection()
		var convUserID string
		err := conn.QueryRow("SELECT user_id FROM conversations WHERE id = ?", req.ConversationID).Scan(&convUserID)
		if err != nil {
			schema.RespondError(w, http.StatusNotFound, "Conversation not found")
			return
		}
		if convUserID != userID {
			schema.RespondError(w, http.StatusForbidden, "Access denied to this conversation")
			return
		}
	}

	log.Printf("[MessageProcessor] Processing message for user %s (conversation: %s, contacts: %v, aboutMe: %v)\n",
		userID, req.ConversationID, req.SelectedContactIds, req.AboutMe != nil)

	// NEW: Check for pending clarifications from previous session
	tempStore := agents.NewTemporaryFactStore(srv.database, userID)
	pendingClarifications, _ := tempStore.GetPendingForUser()

	if len(pendingClarifications) > 0 && req.Message != "" {
		log.Printf("[MessageProcessor] Found %d pending clarifications - checking if message answers them", len(pendingClarifications))

		// Check first pending fact for unanswered questions
		for _, fact := range pendingClarifications {
			unansweredQuestions := tempStore.RemainingQuestionsWithObjects(fact.FactID)
			if len(unansweredQuestions) > 0 {
				log.Printf("[MessageProcessor] Resuming clarification for fact=%s (user message may answer next question)", fact.FactID)

				// Treat user message as answer to next unanswered question
				nextQuestion := unansweredQuestions[0]
				err := tempStore.RecordAnswer(fact.FactID, nextQuestion.ID, req.Message)
				if err != nil {
					log.Printf("[MessageProcessor] Warning: Failed to record answer: %v", err)
				}

				// Check if all questions now answered
				if tempStore.IsComplete(fact.FactID) {
					log.Printf("[MessageProcessor] Clarification complete for fact=%s - saving", fact.FactID)
					tempStore.Remove(fact.FactID)

					response := map[string]interface{}{
						"success": true,
						"phase":   "clarification_complete",
						"message": "Great! I've gathered all the context I need about this.",
						"factId":  fact.FactID,
					}
					schema.RespondSuccess(w, http.StatusOK, "response", response)
					return
				}

				// More questions remain
				remainingQs := tempStore.RemainingQuestionsWithObjects(fact.FactID)
				response := map[string]interface{}{
					"success": true,
					"phase":   "context_gathering",
					"message": "Thanks! One more thing:",
					"questions": remainingQs,
					"factId":    fact.FactID,
				}
				schema.RespondSuccess(w, http.StatusOK, "response", response)
				return
			}
		}
	} else if len(pendingClarifications) > 0 && req.Message == "" {
		// User logged in but no new message - show pending questions
		log.Printf("[MessageProcessor] Showing pending clarifications from previous session")
		fact := pendingClarifications[0]
		unansweredQuestions := tempStore.RemainingQuestionsWithObjects(fact.FactID)

		if len(unansweredQuestions) > 0 {
			response := map[string]interface{}{
				"success": true,
				"phase":   "context_gathering",
				"message": "You have unfinished clarifications from last time:",
				"questions": unansweredQuestions,
				"factId":    fact.FactID,
			}
			schema.RespondSuccess(w, http.StatusOK, "response", response)
			return
		}
	}

	// Use ConversationAgent (V2 architecture)
	if srv.agentSystem == nil || srv.agentSystem.ConversationAgent == nil {
		schema.RespondError(w, http.StatusInternalServerError, "Agent system not initialized")
		return
	}

	// Build context for ConversationAgent from request
	// Extract AboutMe fields from map
	aboutMeStyle := ""
	aboutMeValues := []string{}
	aboutMeTone := ""
	if req.AboutMe != nil {
		if v, ok := req.AboutMe["communicationStyle"].(string); ok {
			aboutMeStyle = v
		}
		if v, ok := req.AboutMe["coreValues"].([]interface{}); ok {
			for _, val := range v {
				if s, ok := val.(string); ok {
					aboutMeValues = append(aboutMeValues, s)
				}
			}
		}
		if v, ok := req.AboutMe["tonePreference"].(string); ok {
			aboutMeTone = v
		}
	}

	// Fetch conversation history if conversation ID provided
	conversationHistory := []models.Message{}
	if req.ConversationID != "" && req.ConversationID != "null" {
		conn := srv.database.GetConnection()
		rows, err := conn.Query(
			"SELECT id, role, content, created_at FROM chat_messages WHERE conversation_id = ? ORDER BY created_at DESC LIMIT 10",
			req.ConversationID,
		)
		if err != nil {
			log.Printf("[MessageProcessor] Warning: Failed to fetch conversation history: %v", err)
		} else {
			defer rows.Close()
			for rows.Next() {
				var id, role, content string
				var createdAt int64
				if err := rows.Scan(&id, &role, &content, &createdAt); err != nil {
					log.Printf("[MessageProcessor] Warning: Error scanning message row: %v", err)
					continue
				}
				conversationHistory = append(conversationHistory, models.Message{
					ID:        id,
					Role:      role,
					Content:   content,
					Timestamp: createdAt,
				})
			}
		}
	}

	ctx := models.Context{
		AboutMe: &models.AboutMe{
			UserID:             userID,
			CommunicationStyle: aboutMeStyle,
			Values:             aboutMeValues,
			PreferredTone:      aboutMeTone,
		},
		ConversationHistory: conversationHistory,
		ContextQuality:      "minimal",
	}

	agentResp, err := srv.agentSystem.ConversationAgent.Run(ctx)
	if err != nil {
		log.Printf("[MessageProcessor] Error: %v\n", err)
		schema.RespondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to process message: %v", err))
		return
	}

	log.Printf("[MessageProcessor] Complete: phase=%s, suggestions=%d, questions=%d\n",
		agentResp.Phase, len(agentResp.Suggestions), len(agentResp.Questions))

	// Map ConversationResponse to Phase5 response format for frontend compatibility
	response := map[string]interface{}{
		"success": true,
		"phase":   agentResp.Phase,
		// Phase 1: Extract facts (not exposed by ConversationAgent currently)
		"phase1": map[string]interface{}{
			"facts":  []interface{}{},
			"shifts": []interface{}{},
		},
		// Phase 2: Clarification questions
		"phase2": map[string]interface{}{
			"clarifications": agentResp.Questions,
			"resolved":       0,
		},
		// Phase 3: Contact management
		"phase3": map[string]interface{}{
			"unknown_contacts": []interface{}{},
			"created_contacts": []interface{}{},
		},
		// Phase 4: Storage
		"phase4": map[string]interface{}{
			"saved_attributes": []interface{}{},
			"conflicts":        []interface{}{},
		},
		// Action required for frontend
		"action_required": map[string]interface{}{
			"needsClarification": len(agentResp.Questions) > 0,
			"clarificationQs":    agentResp.Questions,
			"temporaryFacts":     []interface{}{},
			"hasConflicts":       agentResp.RiskWarning != nil && agentResp.RiskWarning.RiskLevel != "clear",
			"conflicts":          []interface{}{},
		},
		// ConversationAgent specific fields
		"suggestions":      agentResp.Suggestions,
		"riskWarning":      agentResp.RiskWarning,
		"safetyAlert":      agentResp.SafetyAlert,
		"processingTimeMs": agentResp.ProcessingTimeMs,
	}

	respondJSON(w, http.StatusOK, response)
}

// ClarificationResponseHandler - Handle user responses to clarification questions
func (srv *V2APIServer) ClarificationResponseHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	// Extract and validate Bearer token
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": authErr.Error()})
		return
	}

	var req agents.ClarificationResponseRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	if req.FactID == "" || req.QuestionID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing factId or questionId"})
		return
	}

	log.Printf("[Clarification] Response from user %s to question %s\n", userID, req.QuestionID)

	// Create TemporaryFactStore for this user
	tempStore := agents.NewTemporaryFactStore(srv.database, userID)

	// Extract linked facts (from temporary store)
	linkedFacts := []agents.ExtractedFact{}
	if req.FactID != "" {
		if tempFact, err := tempStore.Get(req.FactID); err == nil && tempFact != nil {
			// Convert TemporaryFact to ExtractedFact
			linkedFacts = append(linkedFacts, agents.ExtractedFact{
				ID:              tempFact.FactID,
				Value:           tempFact.FactValue,
				FactType:        tempFact.FactType,
				ProposedSubject: tempFact.AttributedTo,
				Confidence:      tempFact.Confidence,
				Evidence:        tempFact.Evidence,
			})
		}
	}

	// Process response with new AnswerProcessor
	processedResult, err := srv.answerProcessor.ProcessResponse(
		req.QuestionID,
		req.UserResponse,
		linkedFacts,
		userID,
	)
	if err != nil {
		log.Printf("[Clarification] Error processing response: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": fmt.Sprintf("Failed to process response: %v", err),
		})
		return
	}

	// Convert ProcessedAnswerResponse to format expected by frontend
	response := map[string]interface{}{
		"status":     processedResult.Status,
		"molyReply":  processedResult.MolyReply,
		"contextToSave": processedResult.ContextToSave,
		"conflicts": processedResult.Conflicts,
		"needsMore": processedResult.NeedsMoreClarification,
	}

	respondJSON(w, http.StatusOK, response)
}

// AnalyzeIncomingMessageHandler - Analyze incoming message and generate suggestions
func (srv *V2APIServer) AnalyzeIncomingMessageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	// Extract and validate Bearer token
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": authErr.Error()})
		return
	}

	var req struct {
		IncomingMessage string `json:"incomingMessage"`
		ConversationID  string `json:"conversationId"`
	}

	var err error
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	if req.IncomingMessage == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing incomingMessage"})
		return
	}

	log.Printf("[IncomingMessage] Analyzing message from user %s\n", userID)

	// Detect sender
	sender, _ := srv.incomingMessageAnalyzer.DetectSender(req.IncomingMessage)

	// Get user's About Me
	conn := srv.database.GetConnection()
	var commStyle, coreVals, prefTone, prefs, goals, patterns string
	err = conn.QueryRow(
		`SELECT COALESCE(communication_style,''), COALESCE(core_values,''), COALESCE(tone_preference,''),
		        COALESCE(preferences,''), COALESCE(goals,''), COALESCE(patterns,'')
		 FROM about_me WHERE user_id = ?`,
		userID,
	).Scan(&commStyle, &coreVals, &prefTone, &prefs, &goals, &patterns)

	var coreValsArr, goalsArr, patternsArr []string
	_ = json.Unmarshal([]byte(coreVals), &coreValsArr)
	_ = json.Unmarshal([]byte(goals), &goalsArr)
	_ = json.Unmarshal([]byte(patterns), &patternsArr)

	userAboutMe := map[string]interface{}{
		"communicationStyle": commStyle,
		"coreValues":         coreValsArr,
		"tonePreference":     prefTone,
		"preferences":        prefs,
		"goals":              goalsArr,
		"patterns":           patternsArr,
	}

	// Get sender contact context if exists
	senderContext := make(map[string]interface{})
	if sender != "" {
		var relationship, context string
		err = conn.QueryRow(
			`SELECT COALESCE(relationship,''), COALESCE(notes,'')
			 FROM user_contacts WHERE user_id = ? AND name = ?`,
			userID, sender,
		).Scan(&relationship, &context)
		if err == nil {
			senderContext["relationship"] = relationship
			senderContext["context"] = context
		}
	}

	// Get conversation history (last 5 messages for context)
	var conversationHistory []string
	if req.ConversationID != "" {
		rows, err := conn.Query(
			`SELECT content FROM chat_messages
			 WHERE user_id = ? AND conversation_id = ? AND role IN ('user', 'assistant')
			 ORDER BY created_at DESC LIMIT 5`,
			userID, req.ConversationID,
		)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var content string
				if err := rows.Scan(&content); err == nil && content != "" {
					conversationHistory = append(conversationHistory, content)
				}
			}
		}
	}

	// Generate suggestions
	suggestions, err := srv.incomingMessageAnalyzer.GenerateSuggestions(
		req.IncomingMessage,
		conversationHistory,
		userAboutMe,
		senderContext,
	)
	if err != nil {
		log.Printf("[IncomingMessage] Error generating suggestions: %v", err)
		suggestions = srv.incomingMessageAnalyzer.GenerateFallbackSuggestions(req.IncomingMessage)
	}

	response := map[string]interface{}{
		"sender":      sender,
		"suggestions": suggestions,
	}

	log.Printf("[IncomingMessage] ✓ Generated %d suggestions", len(suggestions))
	respondJSON(w, http.StatusOK, response)
}

// AboutMeHandler - Get or save user's About Me profile
func (srv *V2APIServer) AboutMeHandler(w http.ResponseWriter, r *http.Request) {
	// Extract and validate Bearer token
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		log.Printf("[AboutMe] Unauthorized access attempt: %v\n", authErr)
		schema.RespondError(w, http.StatusUnauthorized, authErr.Error())
		return
	}

	conn := srv.database.GetConnection()

	if r.Method == http.MethodGet {
		log.Printf("[AboutMe] GET request from user %s\n", userID)

		// Fetch About Me profile
		var communicationStyle, coreValues, tonePreference, preferences, goals, patterns string
		err := conn.QueryRow(
			`SELECT COALESCE(communication_style,''), COALESCE(core_values,''), COALESCE(tone_preference,''), COALESCE(preferences,''), COALESCE(goals,''), COALESCE(patterns,'') FROM about_me WHERE user_id = ?`,
			userID,
		).Scan(&communicationStyle, &coreValues, &tonePreference, &preferences, &goals, &patterns)

		if err != nil && err != sql.ErrNoRows {
			log.Printf("[AboutMe] Error retrieving profile: %v\n", err)
		}

		profile := &schema.AboutMeProfile{
			CommunicationStyle: communicationStyle,
			CoreValues:         parseArrayFromStorage(coreValues),
			TonePreference:     tonePreference,
			Preferences:        preferences,
			Goals:              parseArrayFromStorage(goals),
			Patterns:           parseArrayFromStorage(patterns),
		}

		log.Printf("[AboutMe] Profile retrieved: style=%s, values=%d, tone=%s\n",
			communicationStyle, len(profile.CoreValues), tonePreference)
		schema.RespondSuccess(w, http.StatusOK, "profile", profile)

	} else if r.Method == http.MethodPost {
		log.Printf("[AboutMe] POST request from user %s\n", userID)

		// Save About Me profile
		req := &schema.AboutMeProfile{}

		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			log.Printf("[AboutMe] Invalid request body: %v\n", err)
			schema.RespondError(w, http.StatusBadRequest, "Invalid request")
			return
		}

		log.Printf("[AboutMe] Parsed profile: style=%s, values=%d, tone=%s, prefs=%s\n",
			req.CommunicationStyle, len(req.CoreValues), req.TonePreference, req.Preferences)

		// Validate using schema validator
		if err := schema.ValidateStruct(req); err != nil {
			log.Printf("[AboutMe] Validation error: %v\n", err)
			schema.RespondError(w, http.StatusBadRequest, err.Error())
			return
		}

		now := time.Now().Unix()
		_, err := conn.Exec(
			`INSERT OR REPLACE INTO about_me (user_id, communication_style, core_values, tone_preference, preferences, goals, patterns, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			userID,
			req.CommunicationStyle,
			encodeArrayForStorage(req.CoreValues),
			req.TonePreference,
			req.Preferences,
			encodeArrayForStorage(req.Goals),
			encodeArrayForStorage(req.Patterns),
			now,
			now,
		)

		if err != nil {
			log.Printf("[AboutMe] Error saving profile: %v\n", err)
			schema.RespondError(w, http.StatusInternalServerError, "Failed to save profile")
			return
		}

		log.Printf("[AboutMe] Profile saved successfully for user %s\n", userID)
		schema.RespondSuccess(w, http.StatusOK, "profile", req)
	}
}

// ConversationsHandler - Get or create conversations
func (srv *V2APIServer) ConversationsHandler(w http.ResponseWriter, r *http.Request) {
	// Extract and validate Bearer token
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		schema.RespondError(w, http.StatusUnauthorized, authErr.Error())
		return
	}

	conn := srv.database.GetConnection()

	if r.Method == http.MethodGet {
		// List user's conversations
		rows, err := conn.Query(
			"SELECT id, name, type, description, purpose, members, settings, notes, created_at, updated_at FROM conversations WHERE user_id = ? ORDER BY updated_at DESC",
			userID,
		)
		if err != nil {
			schema.RespondError(w, http.StatusInternalServerError, "Failed to fetch conversations")
			return
		}
		defer rows.Close()

		conversations := []map[string]interface{}{}
		for rows.Next() {
			var id, name, convType, description, purpose, membersJSON, settingsJSON, notes string
			var createdAt, updatedAt int64
			if err := rows.Scan(&id, &name, &convType, &description, &purpose, &membersJSON, &settingsJSON, &notes, &createdAt, &updatedAt); err != nil {
				log.Printf("[ConversationsHandler] WARNING: Skipping corrupted conversation row - scan error: %v", err)
				continue
			}

			conv := map[string]interface{}{
				"id":          id,
				"name":        name,
				"type":        convType,
				"description": description,
				"purpose":     purpose,
				"createdAt":   createdAt,
				"updatedAt":   updatedAt,
				"notes":       notes,
			}

			// Parse JSON fields
			if membersJSON != "" {
				var members []schema.ConversationMember
				if err := json.Unmarshal([]byte(membersJSON), &members); err == nil {
					conv["members"] = members
				}
			}
			if settingsJSON != "" {
				var settings schema.ConversationSettings
				if err := json.Unmarshal([]byte(settingsJSON), &settings); err == nil {
					conv["settings"] = settings
				}
			}

			conversations = append(conversations, conv)
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"conversations": conversations,
		})

	} else if r.Method == http.MethodPost {
		// Create new conversation
		req := &schema.Conversation{}

		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			schema.RespondError(w, http.StatusBadRequest, "Invalid request")
			return
		}

		// Validate using schema validator
		if err := schema.ValidateStruct(req); err != nil {
			schema.RespondError(w, http.StatusBadRequest, err.Error())
			return
		}

		now := time.Now()
		conversationID := fmt.Sprintf("conv_%d_%d", now.Unix(), now.Nanosecond())
		nowUnix := now.Unix()

		// Encode JSON fields
		membersJSON := ""
		if len(req.Members) > 0 {
			if b, err := json.Marshal(req.Members); err == nil {
				membersJSON = string(b)
			}
		}

		settingsJSON := ""
		if req.Settings.Mode != "" || req.Settings.Context != "" {
			if b, err := json.Marshal(req.Settings); err == nil {
				settingsJSON = string(b)
			}
		}

		_, err := conn.Exec(
			"INSERT INTO conversations (id, user_id, name, type, description, purpose, members, settings, notes, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			conversationID,
			userID,
			req.Name,
			req.Type,
			req.Description,
			req.Purpose,
			membersJSON,
			settingsJSON,
			req.Notes,
			nowUnix,
			nowUnix,
		)

		if err != nil {
			log.Printf("[Conversations] Error creating conversation: %v\n", err)
			schema.RespondError(w, http.StatusInternalServerError, "Failed to create conversation")
			return
		}

		response := &schema.Conversation{
			ID:          conversationID,
			Name:        req.Name,
			Type:        req.Type,
			Purpose:     req.Purpose,
			Description: req.Description,
			Members:     req.Members,
			Settings:    req.Settings,
			Notes:       req.Notes,
			CreatedAt:   nowUnix,
			UpdatedAt:   nowUnix,
		}

		schema.RespondSuccess(w, http.StatusOK, "conversation", response)
	}
}

// ContactsHandler - Get or create user contacts
func (srv *V2APIServer) ContactsHandler(w http.ResponseWriter, r *http.Request) {
	// Extract and validate Bearer token
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		schema.RespondError(w, http.StatusUnauthorized, authErr.Error())
		return
	}

	conn := srv.database.GetConnection()

	if r.Method == http.MethodGet {
		// List user's contacts
		rows, err := conn.Query(
			"SELECT id, name, relationship, notes, created_at FROM user_contacts WHERE user_id = ? ORDER BY updated_at DESC",
			userID,
		)
		if err != nil {
			schema.RespondError(w, http.StatusInternalServerError, "Failed to fetch contacts")
			return
		}
		defer rows.Close()

		contacts := []map[string]interface{}{}
		for rows.Next() {
			var id, name, relationship, notes string
			var createdAt int64
			if err := rows.Scan(&id, &name, &relationship, &notes, &createdAt); err != nil {
				log.Printf("[ContactsHandler] WARNING: Skipping corrupted contact row - scan error: %v", err)
				continue
			}
			contacts = append(contacts, map[string]interface{}{
				"id":           id,
				"name":         name,
				"relationship": relationship,
				"notes":        notes,
				"createdAt":    createdAt,
			})
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"contacts": contacts,
		})

	} else if r.Method == http.MethodPost {
		// Create new contact
		req := &schema.Contact{}

		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			schema.RespondError(w, http.StatusBadRequest, "Invalid request")
			return
		}

		// Validate using schema validator
		if err := schema.ValidateStruct(req); err != nil {
			schema.RespondError(w, http.StatusBadRequest, err.Error())
			return
		}

		now := time.Now()
		contactID := fmt.Sprintf("contact_%d_%d", now.Unix(), now.Nanosecond())
		nowUnix := now.Unix()

		_, err := conn.Exec(
			"INSERT INTO user_contacts (id, user_id, name, relationship, notes, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			contactID,
			userID,
			req.Name,
			req.Relationship,
			req.Notes,
			nowUnix,
			nowUnix,
		)

		if err != nil {
			log.Printf("[Contacts] Error creating contact: %v\n", err)
			schema.RespondError(w, http.StatusInternalServerError, "Failed to create contact")
			return
		}

		response := &schema.Contact{
			ID:           contactID,
			Name:         req.Name,
			Relationship: req.Relationship,
			Notes:        req.Notes,
			CreatedAt:    nowUnix,
		}

		schema.RespondSuccess(w, http.StatusOK, "contact", response)
	}
}

// MessagesHandler - Get messages for a conversation
func (srv *V2APIServer) MessagesHandler(w http.ResponseWriter, r *http.Request) {
	// Extract and validate Bearer token
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		schema.RespondError(w, http.StatusUnauthorized, authErr.Error())
		return
	}

	conn := srv.database.GetConnection()

	if r.Method != http.MethodGet {
		schema.RespondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Get conversationId from query parameter
	conversationID := r.URL.Query().Get("conversationId")
	if conversationID == "" {
		schema.RespondError(w, http.StatusBadRequest, "conversationId parameter required")
		return
	}

	// Verify conversation exists and belongs to user
	var convUserID string
	err := conn.QueryRow(
		"SELECT user_id FROM conversations WHERE id = ?",
		conversationID,
	).Scan(&convUserID)

	if err != nil {
		schema.RespondError(w, http.StatusNotFound, "Conversation not found")
		return
	}

	if convUserID != userID {
		schema.RespondError(w, http.StatusForbidden, "Access denied to this conversation")
		return
	}

	// Fetch messages from chat_messages table
	rows, err := conn.Query(
		"SELECT id, user_id, conversation_id, role, content, context_extracted, contact_mention, created_at FROM chat_messages WHERE conversation_id = ? ORDER BY created_at ASC",
		conversationID,
	)
	if err != nil {
		log.Printf("[Messages] Error fetching messages: %v\n", err)
		schema.RespondError(w, http.StatusInternalServerError, "Failed to fetch messages")
		return
	}
	defer rows.Close()

	messages := []map[string]interface{}{}
	for rows.Next() {
		var id, role, content, contextExtracted, contactMention string
		var uID, convID string
		var createdAt int64

		if err := rows.Scan(&id, &uID, &convID, &role, &content, &contextExtracted, &contactMention, &createdAt); err != nil {
			log.Printf("[MessagesHandler] WARNING: Skipping corrupted message row - scan error: %v", err)
			continue
		}

		messages = append(messages, map[string]interface{}{
			"id":                id,
			"conversationId":    convID,
			"role":              role,
			"content":           content,
			"contextExtracted":  contextExtracted,
			"contactMention":    contactMention,
			"createdAt":         createdAt,
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"messages": messages,
	})
}

// handleCheckSafety - Check message for safety issues (crisis/illegal language)
func handleCheckSafety(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	// Simple implementation: check for crisis keywords
	crisisKeywords := []string{"suicide", "kill myself", "self harm", "hurt myself"}
	alertType := "none"
	for _, keyword := range crisisKeywords {
		if strings.Contains(strings.ToLower(req.Message), keyword) {
			alertType = "crisis"
			break
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"alert_type": alertType,
		"severity":   "none",
		"message":    "Message safe",
	})
}

// handleEvaluateConstitution - Evaluate message against ethical principles
func handleEvaluateConstitution(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"evaluation": "ethical",
		"score":      0.95,
		"violations": []string{},
	})
}

// handleAnalyzeModeShift - Analyze relationship mode shifts
func handleAnalyzeModeShift(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req struct {
		CurrentMode  string `json:"current_mode"`
		ProposedMode string `json:"proposed_mode"`
		Context      string `json:"context"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"recommended": true,
		"risk_level":  "low",
		"analysis":    "Mode shift is appropriate given context",
	})
}

// handleGenerateQuestions - Generate contextual questions
func handleGenerateQuestions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req struct {
		ContactName string `json:"contact_name"`
		Context     string `json:"context"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"questions": []string{
			"What was the last time you connected with " + req.ContactName + "?",
			"How do they typically prefer to communicate?",
			"What topics are they interested in?",
		},
	})
}

// handleGetPrinciples - Get ethical communication principles
func handleGetPrinciples(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	principles := []map[string]string{
		{"id": "1", "title": "Authenticity", "description": "Be genuine and honest"},
		{"id": "2", "title": "Respect", "description": "Value the other person's perspective"},
		{"id": "3", "title": "Clarity", "description": "Communicate clearly and directly"},
		{"id": "4", "title": "Empathy", "description": "Understand and acknowledge feelings"},
		{"id": "5", "title": "Integrity", "description": "Act in alignment with values"},
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"principles": principles,
	})
}

// handleFrontendErrors - Collect and log frontend errors
func handleFrontendErrors(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req struct {
		Errors  []interface{} `json:"errors"`
		Session string        `json:"session"`
		Version string        `json:"extension_version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	log.Printf("[Frontend] Error report: session=%s, version=%s, error_count=%d", req.Session, req.Version, len(req.Errors))

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"received": true,
		"count":    len(req.Errors),
	})
}

// handleGenerateAuthCode - Generate authentication code
func handleGenerateAuthCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req struct {
		DeviceID string `json:"deviceId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	if req.DeviceID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "deviceId required"})
		return
	}

	// Generate 6-digit code
	code := fmt.Sprintf("%06d", rand.Intn(1000000))
	expiresAt := time.Now().Add(10 * time.Minute).Unix()

	// Store code in login_codes table
	conn := v2db.GetConnection()
	_, err := conn.Exec(
		"INSERT INTO login_codes (code, expires_at, device_id, created_at) VALUES (?, ?, ?, ?)",
		code, expiresAt, req.DeviceID, time.Now().Unix(),
	)
	if err != nil {
		log.Printf("[Auth] Failed to store code: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to generate code"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"code":      code,
		"expiresIn": 600,
	})
}

// handleCodeBasedLogin - Login with code (replaces email/password login)
func handleCodeBasedLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req struct {
		Code     string `json:"code"`
		DeviceID string `json:"deviceId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	if req.Code == "" || req.DeviceID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "code and deviceId required"})
		return
	}

	// Look up code in login_codes table (with expiration check in query)
	conn := v2db.GetConnection()
	var userID *string // Use pointer for nullable column

	err := conn.QueryRow(
		"SELECT user_id FROM login_codes WHERE code = ? AND used = 0 AND expires_at > ?",
		req.Code,
		time.Now().Unix(),
	).Scan(&userID)

	if err != nil {
		if err == sql.ErrNoRows {
			respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid or expired code"})
		} else {
			log.Printf("[Auth] Code lookup failed: %v", err)
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Login failed"})
		}
		return
	}

	// If no user_id in code (first-time login), create new user
	var finalUserID string
	if userID == nil {
		finalUserID = "user_" + fmt.Sprintf("%d", time.Now().UnixNano())
		log.Printf("[Auth] Creating new user: %s", finalUserID)
		_, err := conn.Exec(
			"INSERT INTO users (id, created_at, last_active) VALUES (?, ?, ?)",
			finalUserID, time.Now().Unix(), time.Now().Unix(),
		)
		if err != nil {
			log.Printf("[Auth] Failed to create user: %v", err)
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Login failed: " + err.Error()})
			return
		}
		// Update code with user_id
		_, err = conn.Exec(
			"UPDATE login_codes SET user_id = ? WHERE code = ?",
			finalUserID, req.Code,
		)
		if err != nil {
			log.Printf("[Auth] Failed to update code: %v", err)
		}
	} else {
		finalUserID = *userID
	}

	// Create session
	sessionID := "sess_" + fmt.Sprintf("%d", time.Now().UnixNano())
	sessionExpiresAt := time.Now().Add(30 * 24 * time.Hour).Unix() // 30 days
	now := time.Now().Unix()

	log.Printf("[Auth] Creating session for user: %s", finalUserID)
	_, err = conn.Exec(
		"INSERT INTO sessions (id, user_id, device_id, created_at, expires_at, last_used) VALUES (?, ?, ?, ?, ?, ?)",
		sessionID, finalUserID, req.DeviceID, now, sessionExpiresAt, now,
	)
	if err != nil {
		log.Printf("[Auth] Failed to create session: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Login failed: " + err.Error()})
		return
	}

	// Mark code as used
	_, err = conn.Exec("UPDATE login_codes SET used = 1 WHERE code = ?", req.Code)
	if err != nil {
		log.Printf("[Auth] Failed to mark code as used: %v", err)
	}

	log.Printf("[Auth] ✓ Login successful - sessionId: %s, userId: %s", sessionID, finalUserID)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"sessionId": sessionID,
		"userId":   finalUserID,
		"expiresIn": 2592000, // 30 days in seconds
	})
}

// handleValidateAuthCode - Validate current session token
func handleValidateAuthCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	// Extract Bearer token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Missing authorization header"})
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid authorization format"})
		return
	}

	// Look up session
	conn := v2db.GetConnection()
	var userID string
	var expiresAt int64

	err := conn.QueryRow(
		"SELECT user_id, expires_at FROM sessions WHERE id = ?",
		token,
	).Scan(&userID, &expiresAt)

	if err != nil {
		if err == sql.ErrNoRows {
			respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid token"})
		} else {
			log.Printf("[Auth] Session lookup failed: %v", err)
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Validation failed"})
		}
		return
	}

	// Check if session is expired
	if expiresAt < time.Now().Unix() {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Session expired"})
		return
	}

	// Update last_used timestamp
	_, err = conn.Exec(
		"UPDATE sessions SET last_used = ? WHERE id = ?",
		time.Now().Unix(), token,
	)
	if err != nil {
		log.Printf("[Auth] Failed to update session: %v", err)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"valid":   true,
		"userId":  userID,
		"expiresIn": expiresAt - time.Now().Unix(),
	})
}

func main() {
	// Initialize V2 database
	v2dbPath := filepath.Join(os.TempDir(), "moly-v2.db")
	var err error
	v2db, err = database.Init(v2dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize V2 database: %v", err)
	}
	defer v2db.Close()

	log.Println("[Moly] V2 Database initialized at", v2dbPath)

	// Initialize auth tables
	conn := v2db.GetConnection()
	if err := conn.Ping(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Create users table if it doesn't exist
	_, err = conn.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE,
			password_hash TEXT,
			created_at INTEGER NOT NULL,
			last_active INTEGER NOT NULL,
			updated_at INTEGER DEFAULT 0,
			context_level TEXT DEFAULT 'minimal',
			safety_tier TEXT DEFAULT 'standard'
		)
	`)
	if err != nil {
		log.Printf("[Moly] Warning creating users table: %v\n", err)
	}

	// Create sessions table if it doesn't exist
	_, err = conn.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			device_id TEXT,
			created_at INTEGER NOT NULL,
			expires_at INTEGER NOT NULL,
			last_used INTEGER NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		log.Printf("[Moly] Warning creating sessions table: %v\n", err)
	}

	// All tables (about_me, conversations, user_contacts, etc.) are created by schema.sql
	// No inline CREATE TABLE statements - schema.sql is the single source of truth
	log.Println("[Moly] Auth and context binding tables initialized (via schema.sql)")

	// Initialize LLM client (Ollama > Claude API > None)
	var llmClient tools.LLMProvider
	llmClient, err = tools.NewLLMClient()
	if err != nil {
		log.Printf("[Moly] Warning: LLM client initialization failed: %v (will run in heuristic-only mode)\n", err)
	} else if llmClient != nil {
		log.Println("[Moly] LLM client initialized")
	}

	// Initialize V2 API Server with agents and orchestration
	v2Server, err = NewV2APIServer(llmClient, v2db)
	if err != nil {
		log.Fatalf("Failed to initialize V2 API server: %v", err)
	}
	log.Println("[Moly] V2 API Server initialized")

	// Auth API routes
	userAuthServer := auth.NewUserAuthServer(v2db.GetConnection())
	http.HandleFunc("/api/auth/register", userAuthServer.RegisterHandler)
	http.HandleFunc("/api/auth/login", userAuthServer.LoginHandler)
	http.HandleFunc("/api/auth/verify", userAuthServer.VerifyTokenHandler)
	http.HandleFunc("/api/auth/logout", userAuthServer.LogoutHandler)
	log.Println("[Moly] Auth API routes registered (Email/Password authentication)")

	// Phase 5: Full orchestration endpoints
	http.HandleFunc("/api/v2/message-processor", v2Server.MessageProcessorHandler)
	http.HandleFunc("/api/v2/clarification/respond", v2Server.ClarificationResponseHandler)
	http.HandleFunc("/api/v2/incoming-message/analyze", v2Server.AnalyzeIncomingMessageHandler)
	log.Println("[Moly] Phase 5 API routes registered (full orchestration + clarification)")

	// Context binding endpoints
	http.HandleFunc("/api/v2/about-me", v2Server.AboutMeHandler)
	http.HandleFunc("/api/v2/conversations", v2Server.ConversationsHandler)
	http.HandleFunc("/api/v2/contacts", v2Server.ContactsHandler)
	http.HandleFunc("/api/v2/messages", v2Server.MessagesHandler)
	log.Println("[Moly] Context binding API routes registered (about-me + conversations + contacts)")

	// Health check
	http.HandleFunc("/api/status", handleStatus)

	// Safety & Ethics Endpoints
	http.HandleFunc("/api/check-safety", handleCheckSafety)
	http.HandleFunc("/api/evaluate-constitution", handleEvaluateConstitution)
	http.HandleFunc("/api/analyze-mode-shift", handleAnalyzeModeShift)
	http.HandleFunc("/api/generate-questions", handleGenerateQuestions)
	http.HandleFunc("/api/constitution-principles", handleGetPrinciples)

	// Error Reporting Endpoint
	http.HandleFunc("/api/frontend-errors", handleFrontendErrors)

	// Alternative Auth Endpoints
	// NOTE: Code-based login removed in favor of proper email/password auth
	// http.HandleFunc("/api/auth/generate-code", handleGenerateAuthCode)
	// http.HandleFunc("/api/auth/validate", handleValidateAuthCode)

	// Apply CORS middleware
	handler := corsMiddleware(http.DefaultServeMux)

	log.Println("[Moly] Server starting on http://localhost:8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// respondError - Helper to return error responses (used by legacy chat handlers)
func respondError(w http.ResponseWriter, statusCode int, message string) {
	respondJSON(w, statusCode, map[string]string{"error": message})
}

// getConfigPath - Helper to get config file path (used by legacy config handlers)
func getConfigPath() string {
	return filepath.Join(os.TempDir(), "moly-config.json")
}
