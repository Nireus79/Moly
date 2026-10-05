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
	if summary.UserID == "" || summary.ConversationID == "" || summary.MessageID == "" {
		return fmt.Errorf("message summary must have userId, conversationId, and messageId")
	}

	now := time.Now().Unix()
	if summary.CreatedAt == 0 {
		summary.CreatedAt = now
	}
	if summary.UpdatedAt == 0 {
		summary.UpdatedAt = now
	}

	// Marshal JSON fields
	entitiesJSON, _ := json.Marshal(summary.ExtractedEntities)
	entityTypesJSON, _ := json.Marshal(summary.EntityTypes)
	keyPhrasesJSON, _ := json.Marshal(summary.KeyPhrases)

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
func (r *MessageSummaryRepository) DeleteMessageSummary(messageID string) error {
	query := "DELETE FROM message_summaries WHERE message_id = ?"
	_, err := r.db.Exec(query, messageID)
	if err != nil {
		return fmt.Errorf("failed to delete message summary: %w", err)
	}
	return nil
}

// DeleteConversationMessageSummaries deletes all summaries for a conversation
func (r *MessageSummaryRepository) DeleteConversationMessageSummaries(conversationID string) error {
	query := "DELETE FROM message_summaries WHERE conversation_id = ?"
	_, err := r.db.Exec(query, conversationID)
	if err != nil {
		return fmt.Errorf("failed to delete conversation message summaries: %w", err)
	}
	return nil
}
