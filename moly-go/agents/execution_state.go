package agents

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"moly/database"
)

// ExecutionPhase represents the current phase of conversation
type ExecutionPhase string

const (
	PhaseInitial         ExecutionPhase = "initial"           // Just started
	PhaseGatheringContext ExecutionPhase = "gathering_context" // Collecting about_me, contact, intention
	PhaseProcessing       ExecutionPhase = "processing"        // Analyzing message
	PhaseComplete         ExecutionPhase = "complete"          // Finished with this flow
)

// ConversationExecutionState tracks where we are in the conversation workflow
type ConversationExecutionState struct {
	UserID             string
	ConversationID     string
	Phase              ExecutionPhase
	CoveredCategories  map[string]bool // Track what we've asked about
	CurrentMessageSeq  int             // Which message in conversation
	StartedAt          int64
	UpdatedAt          int64
	Version            int64           // For optimistic locking
}

// ExecutionStateManager manages conversation state in database
type ExecutionStateManager struct {
	db *database.Database
}

// NewExecutionStateManager creates a new state manager
func NewExecutionStateManager(db *database.Database) *ExecutionStateManager {
	return &ExecutionStateManager{db: db}
}

// GetOrCreateState loads existing state or creates new one
func (esm *ExecutionStateManager) GetOrCreateState(
	userID string,
	conversationID string,
) (*ConversationExecutionState, error) {
	conn := esm.db.GetConnection()

	// Try to load existing state
	var phase, coveredJSON string
	var seq, startedAt, version int64

	err := conn.QueryRow(`
		SELECT phase, covered_categories, current_message_seq, started_at, version
		FROM conversation_execution_state
		WHERE user_id = ? AND conversation_id = ?
	`, userID, conversationID).Scan(&phase, &coveredJSON, &seq, &startedAt, &version)

	if err == nil {
		// State exists, parse it
		covered := make(map[string]bool)
		if coveredJSON != "" {
			// Parse JSON covered categories if needed
			// For now, simple implementation
		}

		log.Printf("[ExecutionState] Loaded existing state for conversation %s: phase=%s", conversationID, phase)

		return &ConversationExecutionState{
			UserID:            userID,
			ConversationID:    conversationID,
			Phase:             ExecutionPhase(phase),
			CoveredCategories: covered,
			CurrentMessageSeq: int(seq),
			StartedAt:         startedAt,
			UpdatedAt:         time.Now().Unix(),
			Version:           version,
		}, nil
	} else if err != sql.ErrNoRows {
		log.Printf("[ExecutionState] Error loading state: %v", err)
		return nil, err
	}

	// Create new state
	now := time.Now().Unix()
	newState := &ConversationExecutionState{
		UserID:            userID,
		ConversationID:    conversationID,
		Phase:             PhaseInitial,
		CoveredCategories: make(map[string]bool),
		CurrentMessageSeq: 0,
		StartedAt:         now,
		UpdatedAt:         now,
		Version:           1,
	}

	// Save to database
	_, err = conn.Exec(`
		INSERT INTO conversation_execution_state
		(user_id, conversation_id, phase, covered_categories, current_message_seq, started_at, version)
		VALUES (?, ?, ?, '{}', ?, ?, 1)
		ON CONFLICT(user_id, conversation_id) DO NOTHING
	`, userID, conversationID, PhaseInitial, 0, now)

	if err != nil {
		log.Printf("[ExecutionState] Error creating state: %v", err)
		return nil, err
	}

	log.Printf("[ExecutionState] Created new state for conversation %s", conversationID)
	return newState, nil
}

// UpdatePhase updates the execution phase
func (esm *ExecutionStateManager) UpdatePhase(
	state *ConversationExecutionState,
	newPhase ExecutionPhase,
) error {
	conn := esm.db.GetConnection()
	now := time.Now().Unix()

	result, err := conn.Exec(`
		UPDATE conversation_execution_state
		SET phase = ?, updated_at = ?, version = version + 1
		WHERE user_id = ? AND conversation_id = ? AND version = ?
	`, newPhase, now, state.UserID, state.ConversationID, state.Version)

	if err != nil {
		log.Printf("[ExecutionState] Error updating phase: %v", err)
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		log.Printf("[ExecutionState] Warning: Phase update failed or version mismatch")
		return fmt.Errorf("version mismatch or no rows affected")
	}

	state.Phase = newPhase
	state.UpdatedAt = now
	state.Version++

	log.Printf("[ExecutionState] Updated phase to %s", newPhase)
	return nil
}

// MarkCategoryAsCovered marks a question category as answered
func (esm *ExecutionStateManager) MarkCategoryAsCovered(
	state *ConversationExecutionState,
	category string,
) error {
	if state.CoveredCategories == nil {
		state.CoveredCategories = make(map[string]bool)
	}

	state.CoveredCategories[category] = true

	// Update database
	conn := esm.db.GetConnection()
	now := time.Now().Unix()

	// For now, simple string storage; in production would use JSON
	coveredList := ""
	for cat := range state.CoveredCategories {
		if coveredList != "" {
			coveredList += ","
		}
		coveredList += cat
	}

	_, err := conn.Exec(`
		UPDATE conversation_execution_state
		SET covered_categories = ?, updated_at = ?
		WHERE user_id = ? AND conversation_id = ?
	`, coveredList, now, state.UserID, state.ConversationID)

	if err != nil {
		log.Printf("[ExecutionState] Error marking category as covered: %v", err)
		return err
	}

	log.Printf("[ExecutionState] Marked category '%s' as covered", category)
	return nil
}

// IsCategoryCovered checks if a question category has been answered
func (esm *ExecutionStateManager) IsCategoryCovered(
	state *ConversationExecutionState,
	category string,
) bool {
	return state.CoveredCategories[category]
}

// GetUncoveredCategories returns categories that haven't been asked yet
func (esm *ExecutionStateManager) GetUncoveredCategories(
	state *ConversationExecutionState,
	allCategories []string,
) []string {
	var uncovered []string
	for _, cat := range allCategories {
		if !state.CoveredCategories[cat] {
			uncovered = append(uncovered, cat)
		}
	}
	return uncovered
}
