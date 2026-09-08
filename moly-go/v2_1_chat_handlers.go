package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"moly/agents"
	"moly/auth"
	"moly/database"
	"moly/models"
	"moly/tools"
)

// ChatServer handles chat interactions for V2.1
type ChatServer struct {
	db              *database.Database
	llmClient       *tools.LLMClient
	sessionRepo     *auth.SessionRepository
	chatMessageRepo *database.ChatMessageRepository
}

// NewChatServer creates a new chat server
func NewChatServer(db *database.Database, llm *tools.LLMClient) *ChatServer {
	return &ChatServer{
		db:              db,
		llmClient:       llm,
		sessionRepo:     auth.NewSessionRepository(db.GetConnection()),
		chatMessageRepo: database.NewChatMessageRepository(db.GetConnection()),
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
	Logger.WithField("userId", userID[:8]+"...").
		WithField("conversationId", req.ConversationID[:8]+"...").
		Info("[Chat] Processing message")

	chatResponse, err := cs.processChat(userID, req.ConversationID, req.Message)
	if err != nil {
		Logger.WithError(err).Error("[Chat] Failed to process message")
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to process message",
		})
		return
	}

	// Respond with chat response
	respondJSON(w, http.StatusOK, chatResponse)
}

// processChat processes a single chat message
func (cs *ChatServer) processChat(userID, conversationID, userMessage string) (*models.ChatResponse, error) {
	// Get conversation history
	history, err := cs.chatMessageRepo.GetConversationHistory(userID, conversationID, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get history: %w", err)
	}

	// Run chat processing (with 30-second timeout)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Process chat using agents
	conversationAgent, err := agents.NewConversationAgent(cs.llmClient)
	if err != nil {
		// Fallback if agent creation fails
		response := &models.ChatResponse{
			Response:  "I'm here to help. Tell me more about what you're thinking.",
			Timestamp: time.Now().Unix(),
		}
		return response, nil
	}

	// Use the agent to process the chat message
	// We'll cast to the concrete type to access RunChat method
	var response *models.ChatResponse
	if chatAgent, ok := conversationAgent.(*agents.conversationAgent); ok && chatAgent != nil {
		var err error
		response, err = chatAgent.RunChat(ctx, userMessage, history)
		if err != nil {
			Logger.WithError(err).Warn("[Chat] Agent processing failed, using fallback")
			response = &models.ChatResponse{
				Response:  "I'm here to help. Tell me more about what you're thinking.",
				Timestamp: time.Now().Unix(),
			}
		}
	} else {
		// Fallback if type casting fails
		response = &models.ChatResponse{
			Response:  "I'm here to help. Tell me more about what you're thinking.",
			Timestamp: time.Now().Unix(),
		}
	}

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
		ID:              assistantMessageID,
		UserID:          userID,
		ConversationID:  conversationID,
		Role:            "assistant",
		Content:         response.Response,
		ContextExtracted: response.ContextLearned,
		ContactMention:  response.ContactMention,
		CreatedAt:       time.Now().Unix(),
	}

	if err := cs.chatMessageRepo.SaveMessage(assistantMsg); err != nil {
		Logger.WithError(err).Error("[Chat] Failed to save assistant message")
	}

	// TODO: Persist learned context to AboutMe if available
	// (Will be implemented in Phase 2 with full context learning system)

	Logger.WithField("userId", userID[:8]+"...").
		WithField("messageId", assistantMessageID[:8]+"...").
		Info("[Chat] Message processed and saved")

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
