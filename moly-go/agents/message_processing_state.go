package agents

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"moly/database"
	"moly/models"
	"moly/tools"
)

// MessageProcessingState tracks pipeline stage completion for a specific message within a conversation
type MessageProcessingState struct {
	UserID            string
	ConversationID    string
	MessageID         string                 // Unique identifier for this message
	CompletedStages   map[string]bool        // {stage_name: true/false}
	StageResults      map[string]interface{} // Generic storage for stage results
	CreatedAt         int64
	UpdatedAt         int64
	Version           int64 // For optimistic locking
}

// Stage names (7 granular stages)
const (
	StageContextExtraction   = "context_extraction"
	StageRiskAssessment      = "risk_assessment"
	StageSafetyCheck         = "safety_check"
	StageInsightExtraction   = "insight_extraction"
	StageQuestionGeneration  = "question_generation"
	StageResponseGeneration  = "response_generation"
	StageEthicalGate         = "ethical_gate"
)

// MessageProcessingStateManager manages per-message pipeline state
type MessageProcessingStateManager struct {
	db *database.Database
}

// NewMessageProcessingStateManager creates a new manager
func NewMessageProcessingStateManager(db *database.Database) *MessageProcessingStateManager {
	return &MessageProcessingStateManager{db: db}
}

// GetOrCreateState loads existing state or creates new one for this message
func (mpsm *MessageProcessingStateManager) GetOrCreateState(
	userID string,
	conversationID string,
	messageID string,
) (*MessageProcessingState, error) {
	conn := mpsm.db.GetConnection()

	// Try to load existing state
	var loadedCompletedStagesJSON, loadedResultJSON string
	var createdAt, version int64

	err := conn.QueryRow(`
		SELECT completed_stages, stage_results, created_at, version
		FROM message_processing_state
		WHERE user_id = ? AND conversation_id = ? AND message_id = ?
	`, userID, conversationID, messageID).Scan(&loadedCompletedStagesJSON, &loadedResultJSON, &createdAt, &version)

	if err == nil {
		// State exists, parse it
		completedStages := mpsm.parseCompletedStages(loadedCompletedStagesJSON)
		stageResults := mpsm.parseStageResults(loadedResultJSON)

		log.Printf("[MessageProcessingState] Loaded existing state for message %s: completed_stages=%v",
			messageID, completedStages)

		return &MessageProcessingState{
			UserID:          userID,
			ConversationID:  conversationID,
			MessageID:       messageID,
			CompletedStages: completedStages,
			StageResults:    stageResults,
			CreatedAt:       createdAt,
			UpdatedAt:       time.Now().Unix(),
			Version:         version,
		}, nil
	} else if err != sql.ErrNoRows {
		log.Printf("[MessageProcessingState] Error loading state: %v", err)
		return nil, err
	}

	// Create new state - all stages initially incomplete
	now := time.Now().Unix()
	newState := &MessageProcessingState{
		UserID:          userID,
		ConversationID:  conversationID,
		MessageID:       messageID,
		CompletedStages: initializeCompletedStages(),
		StageResults:    make(map[string]interface{}),
		CreatedAt:       now,
		UpdatedAt:       now,
		Version:         1,
	}

	// Save to database
	completedStagesJSON := mpsm.serializeCompletedStages(newState.CompletedStages)
	stageResultsJSON := mpsm.serializeStageResults(newState.StageResults)

	_, err2 := conn.Exec(`
		INSERT INTO message_processing_state
		(user_id, conversation_id, message_id, completed_stages, stage_results, created_at, updated_at, version)
		VALUES (?, ?, ?, ?, ?, ?, ?, 1)
		ON CONFLICT(user_id, conversation_id, message_id) DO NOTHING
	`, userID, conversationID, messageID, completedStagesJSON, stageResultsJSON, now, now)

	if err2 != nil {
		log.Printf("[MessageProcessingState] Error creating state: %v", err2)
		return nil, err2
	}

	log.Printf("[MessageProcessingState] Created new state for message %s", messageID)
	return newState, nil
}

// MarkStageComplete marks a stage as completed and stores its result
func (mpsm *MessageProcessingStateManager) MarkStageComplete(
	state *MessageProcessingState,
	stageName string,
	result interface{},
) error {
	if state.CompletedStages == nil {
		state.CompletedStages = initializeCompletedStages()
	}

	state.CompletedStages[stageName] = true
	state.StageResults[stageName] = result
	state.UpdatedAt = time.Now().Unix()

	// Update database
	conn := mpsm.db.GetConnection()
	completedJSON := mpsm.serializeCompletedStages(state.CompletedStages)
	resultJSON := mpsm.serializeStageResults(state.StageResults)

	result_, err := conn.Exec(`
		UPDATE message_processing_state
		SET completed_stages = ?, stage_results = ?, updated_at = ?, version = version + 1
		WHERE user_id = ? AND conversation_id = ? AND message_id = ? AND version = ?
	`, completedJSON, resultJSON, state.UpdatedAt, state.UserID, state.ConversationID, state.MessageID, state.Version)

	if err != nil {
		log.Printf("[MessageProcessingState] Error marking stage complete: %v", err)
		return err
	}

	rowsAffected, err := result_.RowsAffected()
	if err != nil || rowsAffected == 0 {
		log.Printf("[MessageProcessingState] Warning: Stage update failed or version mismatch")
		return fmt.Errorf("version mismatch or no rows affected")
	}

	state.Version++
	log.Printf("[MessageProcessingState] Marked stage '%s' as complete for message %s", stageName, state.MessageID)
	return nil
}

// IsStageComplete checks if a stage has been completed
func (mpsm *MessageProcessingStateManager) IsStageComplete(state *MessageProcessingState, stageName string) bool {
	if state.CompletedStages == nil {
		return false
	}
	return state.CompletedStages[stageName]
}

// GetRemainingStages returns stages that haven't been completed yet
func (mpsm *MessageProcessingStateManager) GetRemainingStages(state *MessageProcessingState) []string {
	var remaining []string
	allStages := []string{
		StageContextExtraction,
		StageRiskAssessment,
		StageSafetyCheck,
		StageInsightExtraction,
		StageQuestionGeneration,
		StageResponseGeneration,
		StageEthicalGate,
	}

	for _, stage := range allStages {
		if !mpsm.IsStageComplete(state, stage) {
			remaining = append(remaining, stage)
		}
	}
	return remaining
}

// GetStageResult retrieves the result from a completed stage
func (mpsm *MessageProcessingStateManager) GetStageResult(state *MessageProcessingState, stageName string) interface{} {
	if state.StageResults == nil {
		return nil
	}
	return state.StageResults[stageName]
}

// DeleteState removes the processing state for a message (cleanup after completion)
func (mpsm *MessageProcessingStateManager) DeleteState(userID, conversationID, messageID string) error {
	conn := mpsm.db.GetConnection()

	_, err := conn.Exec(`
		DELETE FROM message_processing_state
		WHERE user_id = ? AND conversation_id = ? AND message_id = ?
	`, userID, conversationID, messageID)

	if err != nil {
		log.Printf("[MessageProcessingState] Error deleting state: %v", err)
		return err
	}

	log.Printf("[MessageProcessingState] Deleted processing state for message %s", messageID)
	return nil
}

// Helper functions

func initializeCompletedStages() map[string]bool {
	return map[string]bool{
		StageContextExtraction:   false,
		StageRiskAssessment:      false,
		StageSafetyCheck:         false,
		StageInsightExtraction:   false,
		StageQuestionGeneration:  false,
		StageResponseGeneration:  false,
		StageEthicalGate:         false,
	}
}

func (mpsm *MessageProcessingStateManager) parseCompletedStages(jsonStr string) map[string]bool {
	stages := initializeCompletedStages()
	if jsonStr == "" {
		return stages
	}

	var parsed map[string]bool
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		log.Printf("[MessageProcessingState] Warning: Failed to parse completed_stages: %v", err)
		return stages
	}

	return parsed
}

func (mpsm *MessageProcessingStateManager) serializeCompletedStages(stages map[string]bool) string {
	data, err := json.Marshal(stages)
	if err != nil {
		log.Printf("[MessageProcessingState] Warning: Failed to serialize completed_stages: %v", err)
		return "{}"
	}
	return string(data)
}

func (mpsm *MessageProcessingStateManager) parseStageResults(jsonStr string) map[string]interface{} {
	results := make(map[string]interface{})
	if jsonStr == "" {
		return results
	}

	if err := json.Unmarshal([]byte(jsonStr), &results); err != nil {
		log.Printf("[MessageProcessingState] Warning: Failed to parse stage_results: %v", err)
		return results
	}

	return results
}

func (mpsm *MessageProcessingStateManager) serializeStageResults(results map[string]interface{}) string {
	data, err := json.Marshal(results)
	if err != nil {
		log.Printf("[MessageProcessingState] Warning: Failed to serialize stage_results: %v", err)
		return "{}"
	}
	return string(data)
}

// Helper to create results object from individual stage results
func CreateContextExtractionResult(extracted *models.ExtractedContext) map[string]interface{} {
	return map[string]interface{}{
		"contact":        extracted.Contact,
		"style":          extracted.Style,
		"intention":      extracted.Intention,
		"extractedAt":    time.Now().Unix(),
	}
}

func CreateRiskAssessmentResult(analysis *tools.HarmAnalysis) map[string]interface{} {
	return map[string]interface{}{
		"severity":       analysis.Severity,
		"intervention":   analysis.Intervention,
		"principles":     analysis.ViolatedPrinciples,
		"assessedAt":     time.Now().Unix(),
	}
}

func CreateResponseGenerationResult(response string, reflection *models.Reflection) map[string]interface{} {
	return map[string]interface{}{
		"response":       response,
		"reflection":     reflection,
		"generatedAt":    time.Now().Unix(),
	}
}
