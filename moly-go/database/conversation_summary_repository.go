package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"moly/models"
)

// ConversationSummaryRepository handles conversation summary storage and retrieval
type ConversationSummaryRepository struct {
	db *sql.DB
}

// NewConversationSummaryRepository creates a new repository
func NewConversationSummaryRepository(db *sql.DB) *ConversationSummaryRepository {
	return &ConversationSummaryRepository{db: db}
}

// CreateSummary creates a new conversation summary
func (r *ConversationSummaryRepository) CreateSummary(summary *models.ConversationSummary) error {
	// FIX #32: Enhanced validation for conversation summary
	if summary == nil {
		return fmt.Errorf("summary cannot be nil")
	}
	if summary.UserID == "" || len(summary.UserID) > 255 {
		return fmt.Errorf("userId required and must be <= 255 chars")
	}
	if summary.ConversationID == "" || len(summary.ConversationID) > 255 {
		return fmt.Errorf("conversationId required and must be <= 255 chars")
	}
	if summary.Confidence < 0 || summary.Confidence > 1 {
		return fmt.Errorf("confidence must be in range [0,1], got %.2f", summary.Confidence)
	}
	if summary.MessageCount < 0 {
		return fmt.Errorf("messageCount must be >= 0")
	}

	now := time.Now().Unix()
	if summary.CreatedAt == 0 {
		summary.CreatedAt = now
	}
	if summary.UpdatedAt == 0 {
		summary.UpdatedAt = now
	}
	if summary.LastUpdated == 0 {
		summary.LastUpdated = now
	}

	// Marshal JSON arrays with validation
	keyTopicsJSON, marshalErr := json.Marshal(summary.KeyTopics)
	if marshalErr != nil {
		return fmt.Errorf("failed to marshal keyTopics: %w", marshalErr)
	}
	userPatternsJSON, marshalErr := json.Marshal(summary.UserPatterns)
	if marshalErr != nil {
		return fmt.Errorf("failed to marshal userPatterns: %w", marshalErr)
	}
	confirmedChoicesJSON, marshalErr := json.Marshal(summary.ConfirmedChoices)
	if marshalErr != nil {
		return fmt.Errorf("failed to marshal confirmedChoices: %w", marshalErr)
	}
	openQuestionsJSON, marshalErr := json.Marshal(summary.OpenQuestions)
	if marshalErr != nil {
		return fmt.Errorf("failed to marshal openQuestions: %w", marshalErr)
	}

	query := `
		INSERT INTO conversation_summaries (
			user_id, conversation_id, arc, key_topics, user_patterns,
			confirmed_choices, open_questions, message_count, messages_since_update,
			summary_version, confidence, last_updated, created_at, updated_at,
			accumulated_entity_count, accumulated_contact_count, accumulated_values,
			accumulated_characteristics, conflicts_resolved, clarity_progression
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(
		query,
		summary.UserID,
		summary.ConversationID,
		summary.Arc,
		string(keyTopicsJSON),
		string(userPatternsJSON),
		string(confirmedChoicesJSON),
		string(openQuestionsJSON),
		summary.MessageCount,
		summary.MessagesSinceUpdate,
		summary.SummaryVersion,
		summary.Confidence,
		summary.LastUpdated,
		summary.CreatedAt,
		summary.UpdatedAt,
		summary.AccumulatedEntityCount,
		summary.AccumulatedContactCount,
		summary.AccumulatedValues,
		summary.AccumulatedCharacteristics,
		summary.ConflictsResolved,
		summary.ClarityProgression,
	)

	if err != nil {
		log.Printf("[ConversationSummaryRepository] Failed to create summary: %v", err)
		return fmt.Errorf("failed to create summary: %w", err)
	}

	id, err := result.LastInsertId()
	if err == nil {
		summary.ID = id
	}

	log.Printf("[ConversationSummaryRepository] Created summary %d for conversation %s", id, summary.ConversationID)
	return nil
}

// GetSummary retrieves a summary by user and conversation
func (r *ConversationSummaryRepository) GetSummary(userID, conversationID string) (*models.ConversationSummary, error) {
	if userID == "" || conversationID == "" {
		return nil, fmt.Errorf("userID and conversationID are required")
	}

	query := `
		SELECT id, user_id, conversation_id, arc, key_topics, user_patterns,
		       confirmed_choices, open_questions, message_count, messages_since_update,
		       summary_version, confidence, last_updated, created_at, updated_at,
		       accumulated_entity_count, accumulated_contact_count, accumulated_values,
		       accumulated_characteristics, conflicts_resolved, clarity_progression
		FROM conversation_summaries
		WHERE user_id = ? AND conversation_id = ?
	`

	var summary models.ConversationSummary
	var keyTopicsJSON, userPatternsJSON, confirmedChoicesJSON, openQuestionsJSON sql.NullString
	var accumulatedValuesJSON, accumulatedCharacteristicsJSON, clarityProgressionJSON sql.NullString

	err := r.db.QueryRow(query, userID, conversationID).Scan(
		&summary.ID,
		&summary.UserID,
		&summary.ConversationID,
		&summary.Arc,
		&keyTopicsJSON,
		&userPatternsJSON,
		&confirmedChoicesJSON,
		&openQuestionsJSON,
		&summary.MessageCount,
		&summary.MessagesSinceUpdate,
		&summary.SummaryVersion,
		&summary.Confidence,
		&summary.LastUpdated,
		&summary.CreatedAt,
		&summary.UpdatedAt,
		&summary.AccumulatedEntityCount,
		&summary.AccumulatedContactCount,
		&accumulatedValuesJSON,
		&accumulatedCharacteristicsJSON,
		&summary.ConflictsResolved,
		&clarityProgressionJSON,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Return nil if no summary exists (not an error)
	}
	if err != nil {
		log.Printf("[ConversationSummaryRepository] Failed to get summary: %v", err)
		return nil, fmt.Errorf("failed to get summary: %w", err)
	}

	// Unmarshal JSON arrays
	if keyTopicsJSON.Valid {
		if err := json.Unmarshal([]byte(keyTopicsJSON.String), &summary.KeyTopics); err != nil {
			log.Printf("[ConversationSummaryRepository] Failed to unmarshal keyTopics: %v", err)
		}
	}
	if userPatternsJSON.Valid {
		if err := json.Unmarshal([]byte(userPatternsJSON.String), &summary.UserPatterns); err != nil {
			log.Printf("[ConversationSummaryRepository] Failed to unmarshal userPatterns: %v", err)
		}
	}
	if confirmedChoicesJSON.Valid {
		if err := json.Unmarshal([]byte(confirmedChoicesJSON.String), &summary.ConfirmedChoices); err != nil {
			log.Printf("[ConversationSummaryRepository] Failed to unmarshal confirmedChoices: %v", err)
		}
	}
	if openQuestionsJSON.Valid {
		if err := json.Unmarshal([]byte(openQuestionsJSON.String), &summary.OpenQuestions); err != nil {
			log.Printf("[ConversationSummaryRepository] Failed to unmarshal openQuestions: %v", err)
		}
	}
	// FIX #6: Unmarshal accumulated data
	if accumulatedValuesJSON.Valid {
		summary.AccumulatedValues = accumulatedValuesJSON.String
	}
	if accumulatedCharacteristicsJSON.Valid {
		summary.AccumulatedCharacteristics = accumulatedCharacteristicsJSON.String
	}
	if clarityProgressionJSON.Valid {
		summary.ClarityProgression = clarityProgressionJSON.String
	}

	return &summary, nil
}

// UpdateSummary updates an existing conversation summary
func (r *ConversationSummaryRepository) UpdateSummary(summary *models.ConversationSummary) error {
	if summary.ID == 0 || summary.UserID == "" || summary.ConversationID == "" {
		return fmt.Errorf("summary must have id, userId, and conversationId")
	}

	summary.UpdatedAt = time.Now().Unix()

	// Marshal JSON arrays
	keyTopicsJSON, _ := json.Marshal(summary.KeyTopics)
	userPatternsJSON, _ := json.Marshal(summary.UserPatterns)
	confirmedChoicesJSON, _ := json.Marshal(summary.ConfirmedChoices)
	openQuestionsJSON, _ := json.Marshal(summary.OpenQuestions)

	// OPTIMISTIC LOCKING: Include version in WHERE clause to detect concurrent updates
	query := `
		UPDATE conversation_summaries
		SET arc = ?, key_topics = ?, user_patterns = ?, confirmed_choices = ?,
		    open_questions = ?, message_count = ?, messages_since_update = ?,
		    summary_version = ?, confidence = ?, last_updated = ?, updated_at = ?,
		    accumulated_entity_count = ?, accumulated_contact_count = ?,
		    accumulated_values = ?, accumulated_characteristics = ?,
		    conflicts_resolved = ?, clarity_progression = ?
		WHERE id = ? AND user_id = ? AND conversation_id = ? AND summary_version = ?
	`

	result, err := r.db.Exec(
		query,
		summary.Arc,
		string(keyTopicsJSON),
		string(userPatternsJSON),
		string(confirmedChoicesJSON),
		string(openQuestionsJSON),
		summary.MessageCount,
		summary.MessagesSinceUpdate,
		summary.SummaryVersion,
		summary.Confidence,
		summary.LastUpdated,
		summary.UpdatedAt,
		summary.AccumulatedEntityCount,
		summary.AccumulatedContactCount,
		summary.AccumulatedValues,
		summary.AccumulatedCharacteristics,
		summary.ConflictsResolved,
		summary.ClarityProgression,
		summary.ID,
		summary.UserID,
		summary.ConversationID,
		summary.SummaryVersion-1, // Expect previous version
	)

	if err != nil {
		log.Printf("[ConversationSummaryRepository] Failed to update summary: %v", err)
		return fmt.Errorf("failed to update summary: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		log.Printf("[ConversationSummaryRepository] Summary update failed: version conflict (concurrent update detected)")
		return fmt.Errorf("summary version conflict (concurrent update detected) - summary may have been updated elsewhere")
	}

	log.Printf("[ConversationSummaryRepository] Updated summary %d", summary.ID)
	return nil
}

// UpdateMessagesSinceUpdate increments the messages_since_update counter
func (r *ConversationSummaryRepository) UpdateMessagesSinceUpdate(userID, conversationID string, increment int) error {
	if userID == "" || conversationID == "" {
		return fmt.Errorf("userID and conversationID are required")
	}

	query := `
		UPDATE conversation_summaries
		SET messages_since_update = messages_since_update + ?,
		    updated_at = ?
		WHERE user_id = ? AND conversation_id = ?
	`

	_, err := r.db.Exec(query, increment, time.Now().Unix(), userID, conversationID)
	if err != nil {
		log.Printf("[ConversationSummaryRepository] Failed to update messages_since_update: %v", err)
		return fmt.Errorf("failed to update counter: %w", err)
	}

	return nil
}

// ResetMessagesSinceUpdate resets the counter after summary update
func (r *ConversationSummaryRepository) ResetMessagesSinceUpdate(userID, conversationID string) error {
	if userID == "" || conversationID == "" {
		return fmt.Errorf("userID and conversationID are required")
	}

	query := `
		UPDATE conversation_summaries
		SET messages_since_update = 0,
		    updated_at = ?
		WHERE user_id = ? AND conversation_id = ?
	`

	_, err := r.db.Exec(query, time.Now().Unix(), userID, conversationID)
	if err != nil {
		log.Printf("[ConversationSummaryRepository] Failed to reset counter: %v", err)
		return fmt.Errorf("failed to reset counter: %w", err)
	}

	return nil
}

// AddConfirmedChoice adds a choice to the confirmed_choices array (from Layer 3)
func (r *ConversationSummaryRepository) AddConfirmedChoice(userID, conversationID, choice string) error {
	if userID == "" || conversationID == "" || choice == "" {
		return fmt.Errorf("userID, conversationID, and choice are required")
	}

	// Get current summary
	summary, err := r.GetSummary(userID, conversationID)
	if err != nil {
		return err
	}

	if summary == nil {
		return fmt.Errorf("summary not found for conversation")
	}

	// Check if choice already exists
	for _, existing := range summary.ConfirmedChoices {
		if existing == choice {
			return nil // Already exists, no need to add
		}
	}

	// Add new choice
	summary.ConfirmedChoices = append(summary.ConfirmedChoices, choice)
	summary.LastUpdated = time.Now().Unix()

	// Update database
	return r.UpdateSummary(summary)
}

// GetSummariesNeedingUpdate returns summaries that should be updated (messagesSinceUpdate >= threshold)
func (r *ConversationSummaryRepository) GetSummariesNeedingUpdate(userID string, threshold int) ([]*models.ConversationSummary, error) {
	if userID == "" || threshold <= 0 {
		return nil, fmt.Errorf("userID is required and threshold must be > 0")
	}

	query := `
		SELECT id, user_id, conversation_id, arc, key_topics, user_patterns,
		       confirmed_choices, open_questions, message_count, messages_since_update,
		       summary_version, confidence, last_updated, created_at, updated_at
		FROM conversation_summaries
		WHERE user_id = ? AND messages_since_update >= ?
		ORDER BY messages_since_update DESC
	`

	rows, err := r.db.Query(query, userID, threshold)
	if err != nil {
		log.Printf("[ConversationSummaryRepository] Failed to query summaries: %v", err)
		return nil, fmt.Errorf("failed to query summaries: %w", err)
	}
	defer rows.Close()

	var summaries []*models.ConversationSummary

	for rows.Next() {
		var summary models.ConversationSummary
		var keyTopicsJSON, userPatternsJSON, confirmedChoicesJSON, openQuestionsJSON sql.NullString

		err := rows.Scan(
			&summary.ID,
			&summary.UserID,
			&summary.ConversationID,
			&summary.Arc,
			&keyTopicsJSON,
			&userPatternsJSON,
			&confirmedChoicesJSON,
			&openQuestionsJSON,
			&summary.MessageCount,
			&summary.MessagesSinceUpdate,
			&summary.SummaryVersion,
			&summary.Confidence,
			&summary.LastUpdated,
			&summary.CreatedAt,
			&summary.UpdatedAt,
		)

		if err != nil {
			log.Printf("[ConversationSummaryRepository] Failed to scan row: %v", err)
			continue
		}

		// Unmarshal JSON arrays
		if keyTopicsJSON.Valid {
			json.Unmarshal([]byte(keyTopicsJSON.String), &summary.KeyTopics)
		}
		if userPatternsJSON.Valid {
			json.Unmarshal([]byte(userPatternsJSON.String), &summary.UserPatterns)
		}
		if confirmedChoicesJSON.Valid {
			json.Unmarshal([]byte(confirmedChoicesJSON.String), &summary.ConfirmedChoices)
		}
		if openQuestionsJSON.Valid {
			json.Unmarshal([]byte(openQuestionsJSON.String), &summary.OpenQuestions)
		}

		summaries = append(summaries, &summary)
	}

	return summaries, rows.Err()
}
