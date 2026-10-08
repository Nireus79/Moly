package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"moly/models"
)

// MessageSummaryRepository handles message summary storage and retrieval
type MessageSummaryRepository struct {
	db *sql.DB
}

// NewMessageSummaryRepository creates a new repository
func NewMessageSummaryRepository(db *sql.DB) *MessageSummaryRepository {
	return &MessageSummaryRepository{db: db}
}

// SaveMessageSummary saves a message summary to the database
// Phase 1 (FIX #10): Save immediately after extraction for future re-analysis optimization
func (r *MessageSummaryRepository) SaveMessageSummary(summary *models.MessageSummary) error {
	// FIX #32: Validate message summary before save
	if summary.UserID == "" || len(summary.UserID) > 255 {
		return fmt.Errorf("userId required and must be <= 255 chars")
	}
	if summary.ConversationID == "" || len(summary.ConversationID) > 255 {
		return fmt.Errorf("conversationId required and must be <= 255 chars")
	}
	if summary.MessageID == "" || len(summary.MessageID) > 255 {
		return fmt.Errorf("messageId required and must be <= 255 chars")
	}
	if summary.Confidence < 0 || summary.Confidence > 1 {
		return fmt.Errorf("confidence must be in range [0,1], got %.2f", summary.Confidence)
	}

	now := time.Now().Unix()
	if summary.CreatedAt == 0 {
		summary.CreatedAt = now
	}
	if summary.UpdatedAt == 0 {
		summary.UpdatedAt = now
	}

	// Marshal JSON fields with validation
	var marshalErr error
	entitiesJSON, marshalErr := json.Marshal(summary.ExtractedEntities)
	if marshalErr != nil {
		return fmt.Errorf("failed to marshal entities: %w", marshalErr)
	}
	entityTypesJSON, marshalErr := json.Marshal(summary.EntityTypes)
	if marshalErr != nil {
		return fmt.Errorf("failed to marshal entityTypes: %w", marshalErr)
	}
	keyPhrasesJSON, marshalErr := json.Marshal(summary.KeyPhrases)
	if marshalErr != nil {
		return fmt.Errorf("failed to marshal keyPhrases: %w", marshalErr)
	}

	query := `
		INSERT INTO message_summaries (
			message_id, user_id, conversation_id, message_index, role,
			message_length, extracted_entities, entity_types, intention,
			key_phrases, tone, communication_style, confidence, extraction_source,
			topic_shift, has_clarification, processed_at, created_at, updated_at
		) VALUES (
			?, ?, ?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?, ?, ?,
			?, ?, ?, ?, ?
		)
		ON CONFLICT(message_id) DO UPDATE SET
			extracted_entities = ?,
			entity_types = ?,
			intention = ?,
			key_phrases = ?,
			tone = ?,
			communication_style = ?,
			confidence = ?,
			extraction_source = ?,
			topic_shift = ?,
			has_clarification = ?,
			processed_at = ?,
			updated_at = ?
	`

	_, err := r.db.Exec(query,
		// INSERT values
		summary.MessageID,
		summary.UserID,
		summary.ConversationID,
		summary.MessageIndex,
		summary.Role,
		summary.MessageLength,
		string(entitiesJSON),
		string(entityTypesJSON),
		summary.Intention,
		string(keyPhrasesJSON),
		summary.Tone,
		summary.CommunicationStyle,
		summary.Confidence,
		summary.ExtractionSource,
		summary.TopicShift,
		summary.HasClarification,
		summary.ProcessedAt,
		summary.CreatedAt,
		summary.UpdatedAt,
		// UPDATE values
		string(entitiesJSON),
		string(entityTypesJSON),
		summary.Intention,
		string(keyPhrasesJSON),
		summary.Tone,
		summary.CommunicationStyle,
		summary.Confidence,
		summary.ExtractionSource,
		summary.TopicShift,
		summary.HasClarification,
		summary.ProcessedAt,
		now,
	)

	if err != nil {
		return fmt.Errorf("failed to save message summary: %w", err)
	}

	log.Printf("[MessageSummaryRepository] ✓ Saved summary for message %s (entities=%d, confidence=%.2f)",
		summary.MessageID, len(summary.ExtractedEntities), summary.Confidence)

	return nil
}

// GetMessageSummary retrieves a single message summary
func (r *MessageSummaryRepository) GetMessageSummary(messageID string) (*models.MessageSummary, error) {
	query := `
		SELECT id, message_id, user_id, conversation_id, message_index, role,
		       message_length, extracted_entities, entity_types, intention,
		       key_phrases, tone, communication_style, confidence, extraction_source,
		       topic_shift, has_clarification, processed_at, created_at, updated_at
		FROM message_summaries
		WHERE message_id = ?
	`

	summary := &models.MessageSummary{}
	var entitiesJSON, entityTypesJSON, keyPhrasesJSON string

	err := r.db.QueryRow(query, messageID).Scan(
		&summary.ID,
		&summary.MessageID,
		&summary.UserID,
		&summary.ConversationID,
		&summary.MessageIndex,
		&summary.Role,
		&summary.MessageLength,
		&entitiesJSON,
		&entityTypesJSON,
		&summary.Intention,
		&keyPhrasesJSON,
		&summary.Tone,
		&summary.CommunicationStyle,
		&summary.Confidence,
		&summary.ExtractionSource,
		&summary.TopicShift,
		&summary.HasClarification,
		&summary.ProcessedAt,
		&summary.CreatedAt,
		&summary.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Not found is not an error
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get message summary: %w", err)
	}

	// Unmarshal JSON fields
	_ = json.Unmarshal([]byte(entitiesJSON), &summary.ExtractedEntities)
	_ = json.Unmarshal([]byte(entityTypesJSON), &summary.EntityTypes)
	_ = json.Unmarshal([]byte(keyPhrasesJSON), &summary.KeyPhrases)

	return summary, nil
}

// GetConversationMessageSummaries retrieves all message summaries for a conversation
func (r *MessageSummaryRepository) GetConversationMessageSummaries(conversationID string) ([]models.MessageSummary, error) {
	query := `
		SELECT id, message_id, user_id, conversation_id, message_index, role,
		       message_length, extracted_entities, entity_types, intention,
		       key_phrases, tone, communication_style, confidence, extraction_source,
		       topic_shift, has_clarification, processed_at, created_at, updated_at
		FROM message_summaries
		WHERE conversation_id = ?
		ORDER BY message_index ASC
	`

	rows, err := r.db.Query(query, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to query message summaries: %w", err)
	}
	defer rows.Close()

	var summaries []models.MessageSummary

	for rows.Next() {
		summary := models.MessageSummary{}
		var entitiesJSON, entityTypesJSON, keyPhrasesJSON string

		err := rows.Scan(
			&summary.ID,
			&summary.MessageID,
			&summary.UserID,
			&summary.ConversationID,
			&summary.MessageIndex,
			&summary.Role,
			&summary.MessageLength,
			&entitiesJSON,
			&entityTypesJSON,
			&summary.Intention,
			&keyPhrasesJSON,
			&summary.Tone,
			&summary.CommunicationStyle,
			&summary.Confidence,
			&summary.ExtractionSource,
			&summary.TopicShift,
			&summary.HasClarification,
			&summary.ProcessedAt,
			&summary.CreatedAt,
			&summary.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan message summary: %w", err)
		}

		// Unmarshal JSON fields
		_ = json.Unmarshal([]byte(entitiesJSON), &summary.ExtractedEntities)
		_ = json.Unmarshal([]byte(entityTypesJSON), &summary.EntityTypes)
		_ = json.Unmarshal([]byte(keyPhrasesJSON), &summary.KeyPhrases)

		summaries = append(summaries, summary)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating message summaries: %w", err)
	}

	return summaries, nil
}

// GetRecentMessageSummaries retrieves the last N message summaries for a conversation
func (r *MessageSummaryRepository) GetRecentMessageSummaries(conversationID string, limit int) ([]models.MessageSummary, error) {
	query := `
		SELECT id, message_id, user_id, conversation_id, message_index, role,
		       message_length, extracted_entities, entity_types, intention,
		       key_phrases, tone, communication_style, confidence, extraction_source,
		       topic_shift, has_clarification, processed_at, created_at, updated_at
		FROM message_summaries
		WHERE conversation_id = ?
		ORDER BY message_index DESC
		LIMIT ?
	`

	rows, err := r.db.Query(query, conversationID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent message summaries: %w", err)
	}
	defer rows.Close()

	var summaries []models.MessageSummary

	for rows.Next() {
		summary := models.MessageSummary{}
		var entitiesJSON, entityTypesJSON, keyPhrasesJSON string

		err := rows.Scan(
			&summary.ID,
			&summary.MessageID,
			&summary.UserID,
			&summary.ConversationID,
			&summary.MessageIndex,
			&summary.Role,
			&summary.MessageLength,
			&entitiesJSON,
			&entityTypesJSON,
			&summary.Intention,
			&keyPhrasesJSON,
			&summary.Tone,
			&summary.CommunicationStyle,
			&summary.Confidence,
			&summary.ExtractionSource,
			&summary.TopicShift,
			&summary.HasClarification,
			&summary.ProcessedAt,
			&summary.CreatedAt,
			&summary.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan message summary: %w", err)
		}

		// Unmarshal JSON fields
		_ = json.Unmarshal([]byte(entitiesJSON), &summary.ExtractedEntities)
		_ = json.Unmarshal([]byte(entityTypesJSON), &summary.EntityTypes)
		_ = json.Unmarshal([]byte(keyPhrasesJSON), &summary.KeyPhrases)

		summaries = append(summaries, summary)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating message summaries: %w", err)
	}

	// Reverse to get chronological order
	for i, j := 0, len(summaries)-1; i < j; i, j = i+1, j-1 {
		summaries[i], summaries[j] = summaries[j], summaries[i]
	}

	return summaries, nil
}

// DeleteMessageSummary deletes a message summary
// FIX #6: Added userID for data isolation
func (r *MessageSummaryRepository) DeleteMessageSummary(userID, messageID string) error {
	query := "DELETE FROM message_summaries WHERE message_id = ? AND user_id = ?"
	result, err := r.db.Exec(query, messageID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete message summary: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("message summary not found or access denied")
	}
	return nil
}

// DeleteConversationMessageSummaries deletes all summaries for a conversation
// FIX #6: Added userID for data isolation
func (r *MessageSummaryRepository) DeleteConversationMessageSummaries(userID, conversationID string) error {
	query := "DELETE FROM message_summaries WHERE conversation_id = ? AND user_id = ?"
	_, err := r.db.Exec(query, conversationID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete conversation message summaries: %w", err)
	}
	return nil
}
