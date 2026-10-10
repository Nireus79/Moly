package agents

import (
	"context"
	"fmt"
	"time"

	"moly/database"
)

// Layer5ConflictHandler handles Layer 5: Conflict Detection & Clarification
// When user says something that conflicts with what they said before,
// ask a clarification question instead of giving potentially misaligned advice
type Layer5ConflictHandler struct {
	conflictDetector     *ConflictDetector
	clarificationEngine  *ClarificationEngine
	clarificationRepo    *database.ClarificationQuestionRepository
	clarificationHistory *ClarificationHistory
	db                   *database.Database
}

// NewLayer5ConflictHandler creates a new Layer 5 handler
func NewLayer5ConflictHandler(db *database.Database) *Layer5ConflictHandler {
	return &Layer5ConflictHandler{
		conflictDetector:     NewConflictDetector(db),
		clarificationEngine:  NewClarificationEngine(),
		clarificationRepo:    database.NewClarificationQuestionRepository(db),
		clarificationHistory: NewClarificationHistory(db),
		db:                   db,
	}
}

// generateConflictQuestion creates a specific clarification question for a conflict
func (lch *Layer5ConflictHandler) generateConflictQuestion(
	ctx context.Context,
	conflict ConflictDetectorResult,
	userID string,
	conversationID string,
) *database.ClarificationQuestion {

	question := &database.ClarificationQuestion{
		ID:                fmt.Sprintf("clarif_%d", time.Now().UnixNano()),
		UserID:            userID,
		ConversationID:    conversationID,
		ClarificationType: "context", // Layer 5 type
		Priority:          2,         // Important but not critical
		Status:            "active",
		CreatedAt:         time.Now().Unix(),
	}

	// Generate question text based on conflict type
	switch conflict.Type {
	case "subject_mismatch":
		question.QuestionText = fmt.Sprintf(
			"You mentioned '%s' earlier with a different context. Can you clarify what's changed? %s",
			conflict.Entity.Value,
			conflict.Description,
		)

	case "value_contradiction":
		question.QuestionText = fmt.Sprintf(
			"I noticed you described yourself as '%v' before, but now you're saying '%v'. Can you help me understand what changed? %s",
			conflict.ExistingValue,
			conflict.Entity.Value,
			conflict.Description,
		)

	case "characteristic_conflict":
		question.QuestionText = fmt.Sprintf(
			"Previously you mentioned you were %v, but now you're saying %v. Could you clarify which better describes you right now?",
			conflict.ExistingValue,
			conflict.Entity.Value,
		)

	default:
		question.QuestionText = fmt.Sprintf(
			"I want to make sure I understand correctly: %s. Can you clarify?",
			conflict.Description,
		)
	}

	// Link to the conflict (for traceability)
	question.ContextNotes = fmt.Sprintf(
		"Conflict type: %s | Severity: %s | Confidence: %.2f",
		conflict.Type,
		conflict.Severity,
		conflict.Confidence,
	)

	// Add linked facts from the conflict
	question.LinkedFacts = []string{
		fmt.Sprintf("extracted:%s", conflict.Entity.Value),
		fmt.Sprintf("existing:%v", conflict.ExistingValue),
	}

	return question
}
