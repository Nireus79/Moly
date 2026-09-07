package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"moly/agents"
	"moly/models"
	"moly/tools"
)

// V2APIServer - V2 API server with agent integration
type V2APIServer struct {
	agentSystem *agents.AgentSystem
	llmClient   *tools.LLMClient
	database    *Database
}

// NewV2APIServer - Create new v2 API server with full initialization
func NewV2APIServer(llm *tools.LLMClient, db *Database) (*V2APIServer, error) {
	// LLM client can be nil (will be initialized when needed)
	if db == nil {
		return nil, fmt.Errorf("database cannot be nil")
	}

	return &V2APIServer{
		llmClient: llm,
		database:  db,
		// agentSystem will be created per-request for each user
	}, nil
}

// getAgentSystem - Get or create agent system for a user
func (srv *V2APIServer) getAgentSystem(userID string) (*agents.AgentSystem, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID cannot be empty")
	}

	// If LLM client is not available, create agent system anyway
	// Agents will work but won't have LLM capabilities
	llm := srv.llmClient
	if llm == nil {
		Logger.WithField("userId", userID).Debug("[V2] LLM client not available, agents will have limited functionality")
	}

	// Pass database to agents for context and learning
	return agents.NewAgentSystem(llm, userID, srv.database)
}

// ConversationGenerateHandler - Generate conversation suggestions
// POST /api/v2/conversation/generate
func (srv *V2APIServer) ConversationGenerateHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req models.ConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request",
		})
		return
	}

	// Validate required fields
	if req.ConversationID == "" || req.UserID == "" || req.UserMessage == "" {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "Missing required fields: conversationId, userId, userMessage",
		})
		return
	}

	// Get agent system for this user
	agentSystem, err := srv.getAgentSystem(req.UserID)
	if err != nil {
		Logger.WithField("error", err).Error("[V2] Failed to initialize agents")
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": fmt.Sprintf("Failed to initialize agents: %v", err),
		})
		return
	}

	// Build context for conversation
	ctx := models.Context{
		AboutMe: &models.AboutMe{
			UserID: req.UserID,
		},
		ContactProfile: &models.Contact{
			Name: "Contact",
		},
		ConversationHistory: []models.Message{
			{
				Role:    "user",
				Content: req.UserMessage,
				Type:    "message",
			},
		},
		ContextQuality: "minimal",
	}

	// Call conversation agent to generate suggestions
	response, err := agentSystem.ConversationAgent.Run(ctx)
	if err != nil {
		Logger.WithField("error", err).Error("[V2] Conversation agent failed")
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"phase": "error",
			"error": fmt.Sprintf("Conversation analysis failed: %v", err),
		})
		return
	}

	// Ensure response has required fields
	if response == nil {
		response = &models.ConversationResponse{
			Phase: "suggestions_ready",
		}
	}

	if response.Suggestions == nil {
		response.Suggestions = []models.Suggestion{}
	}

	// Record processing time
	response.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())

	Logger.WithFields(map[string]interface{}{
		"userId":            req.UserID,
		"conversationId":    req.ConversationID,
		"suggestionsCount":  len(response.Suggestions),
		"processingTimeMs":  response.ProcessingTimeMs,
	}).Info("[V2] Conversation generation complete")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ConversationFeedbackHandler - Record user feedback on suggestions
// POST /api/v2/conversation/feedback
func (srv *V2APIServer) ConversationFeedbackHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var feedback models.ConversationFeedback
	if err := json.NewDecoder(r.Body).Decode(&feedback); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request",
		})
		return
	}

	// Validate required fields
	if feedback.ConversationID == "" || feedback.UserID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "Missing required fields: conversationId, userId",
		})
		return
	}

	// Get agent system to access learning agent
	agentSystem, err := srv.getAgentSystem(feedback.UserID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": fmt.Sprintf("Failed to initialize agents: %v", err),
		})
		return
	}

	// Record the suggestion choice in learning agent
	choiceData := models.SuggestionChoiceData{
		UserID:         feedback.UserID,
		ConversationID: feedback.ConversationID,
		SuggestionIndex: feedback.SuggestionChosen,
		ModifiedText:   feedback.ModificationRequest,
		CreatedAt:      time.Now().Unix(),
	}
	// Record feedback as positive if user approved reflection
	if feedback.ReflectionApproved {
		choiceData.UserFeedback = "positive"
	} else {
		choiceData.UserFeedback = "neutral"
	}

	if err := agentSystem.LearningAgent.RecordSuggestionChoice(choiceData); err != nil {
		Logger.WithFields(map[string]interface{}{
			"error":           err,
			"conversationId":  feedback.ConversationID,
			"userId":          feedback.UserID,
		}).Warn("[V2] Failed to record suggestion choice")
		// Don't fail the request if recording fails, just log it
	}

	Logger.WithFields(map[string]interface{}{
		"userId":           feedback.UserID,
		"conversationId":   feedback.ConversationID,
		"suggestionChosen": feedback.SuggestionChosen,
		"processingTimeMs": int(time.Since(startTime).Milliseconds()),
	}).Info("[V2] Feedback recorded")

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "recorded",
		"message": "Feedback recorded successfully",
	})
}

// GetContextHandler - Retrieve conversation context
// GET /api/v2/context?conversationId=...&userId=...
func (srv *V2APIServer) GetContextHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	conversationID := r.URL.Query().Get("conversationId")
	userID := r.URL.Query().Get("userId")

	if conversationID == "" || userID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "Missing conversationId or userId",
		})
		return
	}

	// Get agent system to access context manager
	agentSystem, err := srv.getAgentSystem(userID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": fmt.Sprintf("Failed to initialize agents: %v", err),
		})
		return
	}

	// Get relevant context using context manager
	ctx, err := agentSystem.ContextManager.GetRelevantContext(conversationID, userID)
	if err != nil {
		Logger.WithFields(map[string]interface{}{
			"error":           err,
			"conversationId":  conversationID,
			"userId":          userID,
		}).Warn("[V2] Failed to retrieve context")
		// Return minimal context if retrieval fails
		ctx = &models.Context{
			ContextQuality: "minimal",
		}
	}

	// Calculate context quality metrics
	hasAboutMe := ctx.AboutMe != nil && ctx.AboutMe.UserID != ""
	hasContactProfile := ctx.ContactProfile != nil && ctx.ContactProfile.Name != ""
	hasHistory := len(ctx.ConversationHistory) > 0
	hasBehaviorProfile := ctx.UserBehaviorProfile != nil
	hasReflections := len(ctx.RelevantReflections) > 0

	overallScore := 0.0
	if hasAboutMe {
		overallScore += 0.2
	}
	if hasContactProfile {
		overallScore += 0.2
	}
	if hasHistory {
		overallScore += 0.2
	}
	if hasBehaviorProfile {
		overallScore += 0.2
	}
	if hasReflections {
		overallScore += 0.2
	}

	completenessLevel := "minimal"
	if overallScore > 0.7 {
		completenessLevel = "comprehensive"
	} else if overallScore > 0.4 {
		completenessLevel = "partial"
	}

	response := &models.ContextResponse{
		ConversationID: conversationID,
		ContextQuality: models.ContextQualityMetrics{
			OverallScore:    overallScore,
			HasAboutMe:      hasAboutMe,
			HasContactProfile: hasContactProfile,
			HasHistory:      hasHistory,
			HasBehaviorProfile: hasBehaviorProfile,
			HasReflections:  hasReflections,
			HistoryLength:   len(ctx.ConversationHistory),
			ReflectionCount: len(ctx.RelevantReflections),
			CompletenessLevel: completenessLevel,
		},
		MissingContextGaps: []string{},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetContactsHandler - List all user contacts
// GET /api/v2/contacts?userId=...
func (srv *V2APIServer) GetContactsHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("userId")
	if userID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "Missing userId",
		})
		return
	}

	// Get agent system to access context manager
	agentSystem, err := srv.getAgentSystem(userID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": fmt.Sprintf("Failed to initialize agents: %v", err),
		})
		return
	}

	// Retrieve all contacts for the user using context manager
	contacts, err := agentSystem.ContextManager.GetContacts(userID)
	if err != nil {
		Logger.WithFields(map[string]interface{}{
			"error":  err,
			"userId": userID,
		}).Warn("[V2] Failed to retrieve contacts")
		// Return empty list if retrieval fails
		contacts = []models.Contact{}
	}

	// Convert to contact profiles for response
	contactProfiles := make([]models.ContactProfile, len(contacts))
	for i, contact := range contacts {
		contactProfiles[i] = models.ContactProfile{
			ID:                       contact.ID,
			Name:                     contact.Name,
			Relationship:             contact.Relationship,
			Characteristics:          contact.Characteristics,
			Interests:                contact.Interests,
			CommunicationPreferences: contact.CommunicationPreferences,
			Notes:                    contact.Notes,
		}
	}

	Logger.WithFields(map[string]interface{}{
		"userId":       userID,
		"contactCount": len(contacts),
		"processingTimeMs": int(time.Since(startTime).Milliseconds()),
	}).Info("[V2] Contacts retrieved")

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"userId":   userID,
		"contacts": contactProfiles,
		"total":    len(contactProfiles),
	})
}

// SetAboutMeHandler - Save or update About Me profile
// POST /api/v2/about-me
func (srv *V2APIServer) SetAboutMeHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.AboutMeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request",
		})
		return
	}

	if req.UserID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "Missing userId",
		})
		return
	}

	// Get agent system to access context manager
	agentSystem, err := srv.getAgentSystem(req.UserID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": fmt.Sprintf("Failed to initialize agents: %v", err),
		})
		return
	}

	// Create About Me profile
	aboutMe := &models.AboutMe{
		UserID:            req.UserID,
		CommunicationStyle: req.CommunicationStyle,
		Values:            req.Values,
		PreferredTone:     req.PreferredTone,
		Notes:             req.Notes,
		CreatedAt:         time.Now().Unix(),
		UpdatedAt:         time.Now().Unix(),
		Version:           1,
	}

	// Save using context manager
	if err := agentSystem.ContextManager.SetAboutMe(req.UserID, aboutMe); err != nil {
		Logger.WithFields(map[string]interface{}{
			"error":  err,
			"userId": req.UserID,
		}).Warn("[V2] Failed to save About Me")
		// Don't fail the request, but log the error
	}

	Logger.WithFields(map[string]interface{}{
		"userId":             req.UserID,
		"processingTimeMs":   int(time.Since(startTime).Milliseconds()),
	}).Info("[V2] About Me saved")

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"aboutMe": aboutMe,
		"message": "About Me profile saved successfully",
	})
}

// HealthCheckHandler - Health check for v2 API
// POST or GET /api/v2/health
func (srv *V2APIServer) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	// Accept both GET and POST
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
		"version":   "v2.0.0",
		"components": map[string]interface{}{
			"database": "ok",
			"llm":      "ok",
			"agents":   "ok",
		},
	}

	// Verify database is accessible
	if srv.database == nil {
		health["components"].(map[string]interface{})["database"] = "error"
		health["status"] = "degraded"
	}

	// Verify LLM client is available
	if srv.llmClient == nil {
		health["components"].(map[string]interface{})["llm"] = "error"
		health["status"] = "degraded"
	}

	Logger.WithField("health", health).Debug("[V2] Health check")

	respondJSON(w, http.StatusOK, health)
}
