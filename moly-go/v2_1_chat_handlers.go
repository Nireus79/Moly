package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"moly/agents"
	"moly/auth"
	"moly/database"
	"moly/models"
	"moly/services"
	"moly/tools"
)

// ChatServer handles chat interactions for V2.1
type ChatServer struct {
	db                *database.Database
	llmClient         tools.LLMProvider
	sessionRepo       *auth.SessionRepository
	chatMessageRepo   *database.ChatMessageRepository
	conversationAgent models.ConversationAgent
}

// NewChatServer creates a new chat server
func NewChatServer(db *database.Database, llm tools.LLMProvider) *ChatServer {
	// Initialize ConversationAgent
	agent, err := agents.NewConversationAgent(llm)
	if err != nil {
		Logger.WithError(err).Warn("[Chat] Failed to initialize ConversationAgent")
		agent = nil
	}

	return &ChatServer{
		db:                db,
		llmClient:         llm,
		sessionRepo:       auth.NewSessionRepository(db.GetConnection()),
		chatMessageRepo:   database.NewChatMessageRepository(db.GetConnection()),
		conversationAgent: agent,
	}
}

// ChatHandler handles POST /api/v2.1/chat
// Expects Authorization: Bearer {sessionId} header
func (cs *ChatServer) ChatHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)

	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
			"error": "Method not allowed",
		})
		return
	}

	// Extract and validate session
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		respondJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"error": "Missing Authorization header",
		})
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		respondJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"error": "Invalid Authorization header format",
		})
		return
	}

	sessionID := parts[1]

	// Validate session
	session, err := cs.sessionRepo.ValidateSession(sessionID)
	if err != nil {
		Logger.WithError(err).Debug("[Chat] Session validation failed")
		respondJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"error": "Invalid or expired session",
		})
		return
	}

	userID := session.UserID

	// Parse request
	var req models.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request format",
		})
		return
	}

	// Validate required fields
	if req.Message == "" || req.ConversationID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "message and conversationId are required",
		})
		return
	}

	// Process chat message
	userIDStr := userID
	if len(userID) > 8 {
		userIDStr = userID[:8] + "..."
	}
	convStr := req.ConversationID
	if len(convStr) > 8 {
		convStr = convStr[:8] + "..."
	}
	Logger.WithField("userId", userIDStr).
		WithField("conversationId", convStr).
		WithField("messageLength", len(req.Message)).
		Info("[Chat] INCOMING: Message received from user")

	chatResponse, err := cs.processChat(userID, req.ConversationID, req.Message)
	if err != nil {
		Logger.WithError(err).
			WithField("userId", userIDStr).
			WithField("conversationId", convStr).
			Error("[Chat] ERROR: Failed to process message")
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to process message",
		})
		return
	}

	// Respond with chat response
	Logger.WithField("userId", userIDStr).
		WithField("conversationId", convStr).
		WithField("messageId", chatResponse.MessageID).
		WithField("responseLength", len(chatResponse.Response)).
		Info("[Chat] OUTGOING: Response sent to user")
	respondJSON(w, http.StatusOK, chatResponse)
}

// processChat processes a single chat message
func (cs *ChatServer) processChat(userID, conversationID, userMessage string) (*models.ChatResponse, error) {
	start := time.Now()
	Logger.WithField("userId", userID[:8]+"...").
		WithField("conversationId", conversationID[:8]+"...").
		Debug("[Chat] PROCESS_START: Retrieving conversation history")

	// Get conversation history for Phase 2 integration
	history, err := cs.chatMessageRepo.GetConversationHistory(userID, conversationID, 10)
	if err != nil {
		Logger.WithError(err).
			WithField("userId", userID[:8]+"...").
			Warn("[Chat] WARN: Failed to retrieve history")
		// Don't fail - continue with empty history
	} else {
		Logger.WithField("userId", userID[:8]+"...").
			WithField("historySize", len(history)).
			Debug("[Chat] HISTORY_LOADED: Retrieved conversation history")
	}

	// Run chat processing (with 30-second timeout)
	ctxTimeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get conversation context for the agent
	conversationContext := models.Context{
		ContextQuality: "minimal",
	}

	// Try to run ConversationAgent to generate response
	var agentError error
	var responseText string
	agentStart := time.Now()

	if cs.conversationAgent != nil {
		agentResponse, err := cs.conversationAgent.Run(conversationContext)
		agentElapsed := time.Since(agentStart).Milliseconds()
		if err == nil && agentResponse != nil {
			// Extract response from agent suggestions if available
			if len(agentResponse.Suggestions) > 0 {
				responseText = agentResponse.Suggestions[0].Text
			} else if agentResponse.Error != "" {
				responseText = "I encountered an issue. " + agentResponse.Error
			}
			Logger.WithField("userId", userID[:8]+"...").
				WithField("agentTimeMs", agentElapsed).
				WithField("suggestions", len(agentResponse.Suggestions)).
				Debug("[Chat] AGENT_SUCCESS: Response generated")
		} else {
			// Fallback if agent fails
			agentError = err
			Logger.WithError(agentError).
				WithField("userId", userID[:8]+"...").
				WithField("agentTimeMs", agentElapsed).
				Warn("[Chat] AGENT_FAILED: Agent error")
		}
	} else {
		Logger.WithField("userId", userID[:8]+"...").
			Warn("[Chat] AGENT_UNAVAILABLE: ConversationAgent not initialized")
	}

	// Use fallback if agent didn't generate response
	if responseText == "" {
		if agentError != nil {
			Logger.WithError(agentError).
				WithField("userId", userID[:8]+"...").
				Warn("[Chat] FALLBACK: Using hardcoded response due to agent error")
		} else if cs.conversationAgent == nil {
			Logger.WithField("userId", userID[:8]+"...").
				Warn("[Chat] FALLBACK: Using hardcoded response (no agent)")
		}

		responseText = "I'm here to help. Tell me more about what you're thinking."
	}

	response := &models.ChatResponse{
		Response:  responseText,
		Timestamp: time.Now().Unix(),
	}
	_ = ctxTimeout // Potential use for async operations in future

	// Generate message IDs
	userMessageID := generateMessageID()
	assistantMessageID := generateMessageID()
	response.MessageID = assistantMessageID

	// Save user message
	userMsg := &models.ChatMessage{
		ID:             userMessageID,
		UserID:         userID,
		ConversationID: conversationID,
		Role:           "user",
		Content:        userMessage,
		CreatedAt:      time.Now().Unix(),
	}

	if err := cs.chatMessageRepo.SaveMessage(userMsg); err != nil {
		Logger.WithError(err).Error("[Chat] Failed to save user message")
	}

	// Save assistant message
	assistantMsg := &models.ChatMessage{
		ID:               assistantMessageID,
		UserID:           userID,
		ConversationID:   conversationID,
		Role:             "assistant",
		Content:          response.Response,
		ContextExtracted: response.ContextLearned,
		ContactMention:   response.ContactMention,
		CreatedAt:        time.Now().Unix(),
	}

	if err := cs.chatMessageRepo.SaveMessage(assistantMsg); err != nil {
		Logger.WithError(err).Error("[Chat] Failed to save assistant message")
	}

	// Persist learned context to profile if agent extracted insights
	if len(response.ContextLearned) > 0 && cs.conversationAgent != nil {
		Logger.WithField("userId", userID[:8]+"...").
			WithField("contextItems", len(response.ContextLearned)).
			Debug("[Chat] PERSISTENCE: Persisting learned context to profile")

		// Get profile updater to persist context
		profileUpdater := services.NewProfileUpdater(cs.db)

		// Use AddImplicitLearning to persist insights the agent extracted
		// This is a simplified approach - the full context persistence
		// happens when the extraction job processes the conversation
		if err := profileUpdater.AddImplicitLearning(
			userID,
			"chat_interaction",
			response.Response,
			"conversation_insight",
			0.7, // Moderate confidence for chat-derived insights
			"chat_response",
		); err != nil {
			Logger.WithError(err).
				WithField("userId", userID[:8]+"...").
				Error("[Chat] ERROR: Failed to persist context to profile")
		} else {
			Logger.WithField("userId", userID[:8]+"...").
				WithField("contextItems", len(response.ContextLearned)).
				Info("[Chat] SUCCESS: Context persisted to profile")
		}
	} else if len(response.ContextLearned) == 0 {
		Logger.WithField("userId", userID[:8]+"...").
			Debug("[Chat] NO_CONTEXT: No learned context to persist")
	}

	elapsed := time.Since(start).Milliseconds()
	Logger.WithField("userId", userID[:8]+"...").
		WithField("messageId", assistantMessageID[:8]+"...").
		WithField("totalTimeMs", elapsed).
		Info("[Chat] COMPLETE: Message processed and saved")

	return response, nil
}

// GetConversationHistoryHandler retrieves chat history
// GET /api/v2.1/conversations/{conversationId}/messages
func (cs *ChatServer) GetConversationHistoryHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)

	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != http.MethodGet {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
			"error": "Method not allowed",
		})
		return
	}

	// Extract and validate session
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		respondJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"error": "Missing Authorization header",
		})
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		respondJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"error": "Invalid Authorization header format",
		})
		return
	}

	sessionID := parts[1]
	session, err := cs.sessionRepo.ValidateSession(sessionID)
	if err != nil {
		respondJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"error": "Invalid or expired session",
		})
		return
	}

	// Extract conversation ID from query param
	conversationID := r.URL.Query().Get("conversationId")
	if conversationID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "conversationId query parameter is required",
		})
		return
	}

	// Get conversation history
	messages, err := cs.chatMessageRepo.GetConversationHistory(session.UserID, conversationID, 100)
	if err != nil {
		Logger.WithError(err).Error("[Chat] Failed to get history")
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve conversation history",
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"messages": messages,
	})
}

// DeleteConversationHandler deletes a conversation
// DELETE /api/v2.1/conversations/{conversationId}
func (cs *ChatServer) DeleteConversationHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)

	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != http.MethodDelete {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
			"error": "Method not allowed",
		})
		return
	}

	// Extract and validate session
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		respondJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"error": "Missing Authorization header",
		})
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		respondJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"error": "Invalid Authorization header format",
		})
		return
	}

	sessionID := parts[1]
	_, err := cs.sessionRepo.ValidateSession(sessionID)
	if err != nil {
		respondJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"error": "Invalid or expired session",
		})
		return
	}

	// Extract conversation ID
	conversationID := r.URL.Query().Get("conversationId")
	if conversationID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "conversationId query parameter is required",
		})
		return
	}

	// Delete conversation
	if err := cs.chatMessageRepo.DeleteConversation(conversationID); err != nil {
		Logger.WithError(err).Error("[Chat] Failed to delete conversation")
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to delete conversation",
		})
		return
	}

	Logger.WithField("conversationId", conversationID[:8]+"...").Info("[Chat] Conversation deleted")

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

// generateMessageID generates a unique message ID
func generateMessageID() string {
	return "msg_" + uuid.New().String()[:16]
}

// deduplicate removes duplicates from a string slice
func deduplicate(s []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}
