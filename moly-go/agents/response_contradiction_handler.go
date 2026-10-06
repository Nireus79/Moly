package agents

import (
	"context"
	"fmt"
	"log"
	"time"

	"moly/database"
)

// ResponseContradictionHandler generates clarification questions when responses contradict user characteristics
// PHASE 3: Used when ResponseValidator detects misaligned advice
type ResponseContradictionHandler struct {
	clarificationEngine *ClarificationEngine
	clarificationRepo   *database.ClarificationQuestionRepository
	db                  *database.Database
}

// NewResponseContradictionHandler creates a new handler
func NewResponseContradictionHandler(db *database.Database) *ResponseContradictionHandler {
	return &ResponseContradictionHandler{
		clarificationEngine: NewClarificationEngine(),
		clarificationRepo:   database.NewClarificationQuestionRepository(db),
		db:                  db,
	}
}

// GenerateContradictionQuestion creates a clarification question for a response contradiction
// Returns a question asking user to clarify which characteristic is correct
func (rch *ResponseContradictionHandler) GenerateContradictionQuestion(
	ctx context.Context,
	contradiction Contradiction,
	userID string,
	conversationID string,
	responseText string,
) *database.ClarificationQuestion {

	log.Printf("[ResponseContradictionHandler] Generating question for: %s ↔ %s",
		contradiction.ResponseCharacteristic, contradiction.UserCharacteristic)

	question := &database.ClarificationQuestion{
		ID:                fmt.Sprintf("resp_clarif_%d", time.Now().UnixNano()),
		UserID:            userID,
		ConversationID:    conversationID,
		ClarificationType: "context", // Phase 3 type
		Priority:          3,                         // Highest priority - must clarify before proceeding
		Status:            "pending",
		CreatedAt:         time.Now().Unix(),
	}

	// Generate question based on contradiction type
	question.QuestionText = rch.generateQuestionText(contradiction)

	// Link to the contradiction context
	question.ContextNotes = fmt.Sprintf(
		"Response suggested: '%s' | User characteristic: '%s' | Confidence: %.2f",
		contradiction.ResponseCharacteristic,
		contradiction.UserCharacteristic,
		contradiction.Confidence,
	)

	// Add linked facts
	question.LinkedFacts = []string{
		fmt.Sprintf("response:%s", contradiction.ResponseCharacteristic),
		fmt.Sprintf("user:%s", contradiction.UserCharacteristic),
		fmt.Sprintf("response_text:%.100s", responseText),
	}

	return question
}

// generateQuestionText creates a natural, supportive question for the contradiction
func (rch *ResponseContradictionHandler) generateQuestionText(contradiction Contradiction) string {
	// Friendly phrasing that doesn't sound accusatory
	return fmt.Sprintf(
		"I want to make sure I'm understanding you correctly. I suggested exploring '%s', "+
			"but you've described yourself as '%s'. Could you help me understand which better describes you right now? "+
			"People can have both qualities in different contexts, so no pressure—just help me get it straight.",
		contradiction.ResponseCharacteristic,
		contradiction.UserCharacteristic,
	)
}

// GenerateMultipleContradictionQuestion creates a question for when multiple contradictions exist
func (rch *ResponseContradictionHandler) GenerateMultipleContradictionQuestion(
	ctx context.Context,
	contradictions []Contradiction,
	userID string,
	conversationID string,
) *database.ClarificationQuestion {

	log.Printf("[ResponseContradictionHandler] Generating question for %d contradictions",
		len(contradictions))

	question := &database.ClarificationQuestion{
		ID:                fmt.Sprintf("resp_clarif_multi_%d", time.Now().UnixNano()),
		UserID:            userID,
		ConversationID:    conversationID,
		ClarificationType: "context",
		Priority:          3, // Highest priority
		Status:            "pending",
		CreatedAt:         time.Now().Unix(),
	}

	// Build list of contradictions
	var contradictionList string
	for i, c := range contradictions {
		contradictionList += fmt.Sprintf(
			"%d. I suggested '%s' but you've said '%s'\n",
			i+1, c.ResponseCharacteristic, c.UserCharacteristic,
		)
	}

	question.QuestionText = fmt.Sprintf(
		"I want to make sure I'm not giving you misaligned advice. I noticed a few things I said might "+
			"contradict what you've shared about yourself:\n\n%s\n"+
			"Could you help me understand which of these better describes you? "+
			"It's totally normal to have different sides to our personalities.",
		contradictionList,
	)

	// Link all contradictions
	question.ContextNotes = fmt.Sprintf(
		"Multiple contradictions detected: %d | Requires clarification before proceeding",
		len(contradictions),
	)

	for _, c := range contradictions {
		question.LinkedFacts = append(question.LinkedFacts,
			fmt.Sprintf("contradiction:%s↔%s", c.ResponseCharacteristic, c.UserCharacteristic),
		)
	}

	return question
}

// ExplainContradiction creates a human-readable explanation of why this is a contradiction
func (rch *ResponseContradictionHandler) ExplainContradiction(contradiction Contradiction) string {
	explanation := fmt.Sprintf(
		"Contradiction Detected:\n"+
			"  You described yourself as: %s\n"+
			"  My response suggested: %s\n"+
			"  These are opposite characteristics\n"+
			"  Confidence in contradiction: %.0f%%",
		contradiction.UserCharacteristic,
		contradiction.ResponseCharacteristic,
		contradiction.Confidence*100,
	)

	if contradiction.Evidence != "" {
		explanation += fmt.Sprintf("\n  Evidence: %s", contradiction.Evidence)
	}

	return explanation
}

// SaveContradictionQuestion saves a contradiction question to the database
func (rch *ResponseContradictionHandler) SaveContradictionQuestion(
	question *database.ClarificationQuestion,
) error {

	if rch.clarificationRepo == nil {
		log.Printf("[ResponseContradictionHandler] Warning: No clarification repo, cannot save question")
		return fmt.Errorf("no clarification repository available")
	}

	err := rch.clarificationRepo.SaveQuestion(question)
	if err != nil {
		log.Printf("[ResponseContradictionHandler] Error saving question: %v", err)
		return err
	}

	log.Printf("[ResponseContradictionHandler] ✓ Contradiction question saved: %s", question.ID)
	return nil
}

// GetSummary returns a human-readable summary of the contradiction and recommended action
func (rch *ResponseContradictionHandler) GetSummary(contradictions []Contradiction) string {
	if len(contradictions) == 0 {
		return "No contradictions to report"
	}

	summary := fmt.Sprintf("Response Contradictions Found: %d\n", len(contradictions))
	for i, c := range contradictions {
		summary += fmt.Sprintf(
			"  %d. '%s' contradicts '%s' (confidence: %.0f%%)\n",
			i+1,
			c.ResponseCharacteristic,
			c.UserCharacteristic,
			c.Confidence*100,
		)
	}
	summary += "\nAction: Clarification question generated and will be sent instead of response.\n"
	summary += "Next: User response will update characteristic record."

	return summary
}

// BlockedResponseLog creates a structured log entry for when a response is blocked
type BlockedResponseLog struct {
	Timestamp          int64
	UserID             string
	ConversationID     string
	ResponseBlocked    string
	Contradictions     []Contradiction
	ClarificationID    string
	Reason             string
}

// LogBlockedResponse creates a log entry for tracking blocked responses
func (rch *ResponseContradictionHandler) LogBlockedResponse(
	userID string,
	conversationID string,
	blockedResponse string,
	contradictions []Contradiction,
	clarificationID string,
) BlockedResponseLog {

	log.Printf("[ResponseContradictionHandler] BLOCKED: %d contradictions detected for user %s",
		len(contradictions), userID)

	return BlockedResponseLog{
		Timestamp:       time.Now().Unix(),
		UserID:          userID,
		ConversationID:  conversationID,
		ResponseBlocked: blockedResponse,
		Contradictions:  contradictions,
		ClarificationID: clarificationID,
		Reason:          "Response contradicts extracted user characteristics",
	}
}
