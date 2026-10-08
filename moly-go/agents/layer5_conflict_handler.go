package agents

import (
	"context"
	"fmt"
	"log"
	"time"

	"moly/database"
	"moly/models"
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

// ProcessConflicts takes detected conflicts and generates clarification questions
// Returns list of questions that should be asked to resolve conflicts
func (lch *Layer5ConflictHandler) ProcessConflicts(
	ctx context.Context,
	conflicts []ConflictDetectorResult,
	userID string,
	conversationID string,
	artifact *models.ExtractionArtifact,
) ([]*database.ClarificationQuestion, error) {

	log.Printf("[Layer5] Processing %d detected conflicts", len(conflicts))

	// Verify artifact is locked (Phase 1 enforcement)
	if artifact != nil && !artifact.IsLocked {
		log.Printf("[Layer5] ❌ WARNING: Processing conflicts with unlocked artifact")
		log.Printf("[Layer5]    This violates Phase 1 locking requirement")
	}

	var questions []*database.ClarificationQuestion
	skippedCount := 0

	for i, conflict := range conflicts {
		log.Printf("[Layer5] Processing conflict %d/%d: type=%s, severity=%s",
			i+1, len(conflicts), conflict.Type, conflict.Severity)

		// Check if this conflict was already asked about (prevent re-asking)
		wasAsked, err := lch.clarificationHistory.WasRecentlyAsked(
			userID,
			conversationID,
			conflict.Entity.Value,
			conflict.Type,
		)
		if err != nil {
			log.Printf("[Layer5] Warning: Failed to check history: %v", err)
		} else if wasAsked {
			log.Printf("[Layer5] ⊘ Conflict already asked about, skipping: %s", conflict.Description)
			skippedCount++
			continue
		}

		// Generate clarification question for this conflict
		question := lch.generateConflictQuestion(ctx, conflict, userID, conversationID)
		if question == nil {
			log.Printf("[Layer5] Warning: Failed to generate question for conflict: %s", conflict.Description)
			continue
		}

		// Save question to database
		err = lch.clarificationRepo.SaveQuestion(question)
		if err != nil {
			log.Printf("[Layer5] Warning: Failed to save question: %v", err)
			continue
		}

		log.Printf("[Layer5] ✓ Generated question for conflict: %s", conflict.Description)
		questions = append(questions, question)

		// Record that we asked about this conflict
		err = lch.clarificationHistory.RecordAsked(
			question.ID,
			userID,
			conversationID,
			conflict.Entity.Value,
			conflict.Type,
		)
		if err != nil {
			log.Printf("[Layer5] Warning: Failed to record question: %v", err)
		}
	}

	log.Printf("[Layer5] ✓ Processed conflicts: generated=%d, skipped=%d",
		len(questions), skippedCount)

	return questions, nil
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

// GetConflictSummary returns a human-readable summary of conflicts
func (lch *Layer5ConflictHandler) GetConflictSummary(conflicts []ConflictDetectorResult) string {
	if len(conflicts) == 0 {
		return "No conflicts detected"
	}

	summary := fmt.Sprintf("Found %d conflict(s):\n", len(conflicts))
	for i, conflict := range conflicts {
		summary += fmt.Sprintf(
			"  %d. %s (severity: %s, confidence: %.2f%%)\n",
			i+1,
			conflict.Description,
			conflict.Severity,
			conflict.Confidence*100,
		)
	}
	return summary
}
