package tools

import (
	"context"
	"fmt"

	"moly/models"
)

// SessionManager - Manages complete user sessions across multiple conversations
type SessionManager struct {
	orchestrator *ConversationOrchestrator
	llmClient    LLMProvider
}

// NewSessionManager - Create new session manager
func NewSessionManager(llm LLMProvider) *SessionManager {
	return &SessionManager{
		orchestrator: NewConversationOrchestrator(llm),
		llmClient:    llm,
	}
}

// SessionState - Current state of a session
type SessionState struct {
	UserID           string
	ContextLevel     string // "minimal", "partial", "comprehensive"
	HasAboutMe       bool
	HasContact       bool
	HasHistory       bool
	LastIntention    string
	Phase            string // "gathering", "suggesting", "learning"
	ContextualItems  map[string]interface{}
}

// ProcessConversationTurn - Process one full conversation turn
func (sm *SessionManager) ProcessConversationTurn(
	ctx context.Context,
	userID string,
	userMessage string,
	userContext *models.Context,
) (*ConversationTurnResult, error) {

	// Process user input
	processed, err := sm.orchestrator.ProcessUserInput(ctx, userID, userMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to process input: %w", err)
	}

	// Generate contextual response
	contacts := []models.Contact{}
	if userContext.ContactProfile != nil {
		contacts = append(contacts, *userContext.ContactProfile)
	}

	contextual := sm.orchestrator.GenerateContextualResponse(
		processed,
		userContext.UserBehaviorProfile,
		userContext.AboutMe,
		contacts,
	)

	// Determine phase
	phase := sm.determinePhase(contextual, userContext)

	// Generate response components
	var suggestions []models.Suggestion
	var questions []string

	if phase == "gathering" {
		// Generate follow-up questions to gather context
		questions, _ = sm.orchestrator.GenerateFollowUpQuestions(
			ctx,
			userMessage,
			processed.DetectedIntention,
		)
	} else if phase == "suggesting" {
		// Generate personalized suggestions
		// This would call the SuggestionGenerator with full context
		suggestions = sm.generateSuggestions(ctx, processed, userContext)
	}

	// Structure response
	responseStructure := sm.orchestrator.StructureConversationResponse(
		processed,
		contextual,
		suggestions,
		questions,
	)

	return &ConversationTurnResult{
		UserID:            userID,
		ProcessedInput:    processed,
		ContextualResponse: contextual,
		Phase:             phase,
		Suggestions:       suggestions,
		Questions:         questions,
		ResponseStructure: responseStructure,
	}, nil
}

// determinePhase - Determine what phase the conversation is in
func (sm *SessionManager) determinePhase(contextual *ConversationContextualResponse, userContext *models.Context) string {
	// If context is minimal, gather more information
	if contextual.ContextLevel == "minimal" {
		return "gathering"
	}

	// If we have some context, can suggest
	if contextual.ContextLevel == "partial" || contextual.ContextLevel == "comprehensive" {
		return "suggesting"
	}

	return "analyzing"
}

// generateSuggestions - Generate personalized suggestions
func (sm *SessionManager) generateSuggestions(
	ctx context.Context,
	processed *ConversationProcessResult,
	userContext *models.Context,
) []models.Suggestion {

	suggestions := []models.Suggestion{}

	// Without full implementation, return placeholder
	// In real implementation, this would call SuggestionGenerator
	if sm.llmClient != nil {
		// Use LLM to generate suggestions
		// This requires full integration with SuggestionGenerator
	}

	return suggestions
}

// ValidateSession - Check if session is valid and ready
func (sm *SessionManager) ValidateSession(
	userID string,
	userContext *models.Context,
) (bool, []string) {

	issues := []string{}

	if userID == "" {
		issues = append(issues, "user_id_required")
	}

	if userContext == nil {
		issues = append(issues, "context_required")
	} else {
		if userContext.AboutMe == nil || userContext.AboutMe.CommunicationStyle == "" {
			issues = append(issues, "about_me_incomplete")
		}
		if userContext.ContactProfile == nil {
			issues = append(issues, "no_contact")
		}
	}

	return len(issues) == 0, issues
}

// GetSessionState - Get current session state
func (sm *SessionManager) GetSessionState(
	userID string,
	userContext *models.Context,
) *SessionState {

	state := &SessionState{
		UserID:          userID,
		ContextualItems: make(map[string]interface{}),
	}

	if userContext == nil {
		state.ContextLevel = "minimal"
		state.Phase = "gathering"
		return state
	}

	// Determine context level
	hasAbout := userContext.AboutMe != nil && userContext.AboutMe.CommunicationStyle != ""
	hasContact := userContext.ContactProfile != nil && userContext.ContactProfile.Name != ""
	hasHistory := len(userContext.ConversationHistory) > 0

	state.HasAboutMe = hasAbout
	state.HasContact = hasContact
	state.HasHistory = hasHistory

	if hasAbout && hasContact && hasHistory {
		state.ContextLevel = "comprehensive"
		state.Phase = "suggesting"
	} else if hasAbout || hasContact {
		state.ContextLevel = "partial"
		state.Phase = "gathering"
	} else {
		state.ContextLevel = "minimal"
		state.Phase = "gathering"
	}

	// Store contextual items
	if hasAbout {
		state.ContextualItems["communication_style"] = userContext.AboutMe.CommunicationStyle
		if len(userContext.AboutMe.Values) > 0 {
			state.ContextualItems["values"] = userContext.AboutMe.Values
		}
	}

	if hasContact {
		state.ContextualItems["primary_contact"] = userContext.ContactProfile.Name
	}

	if hasHistory {
		state.ContextualItems["message_count"] = len(userContext.ConversationHistory)
	}

	return state
}

// ConversationTurnResult - Result of processing one conversation turn
type ConversationTurnResult struct {
	UserID             string
	ProcessedInput     *ConversationProcessResult
	ContextualResponse *ConversationContextualResponse
	Phase              string
	Suggestions        []models.Suggestion
	Questions          []string
	ResponseStructure  map[string]interface{}
}
