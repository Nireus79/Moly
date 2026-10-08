package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"moly/models"
)

// ChatMessageRepository handles chat message storage
type ChatMessageRepository struct {
	db *sql.DB
}

// NewChatMessageRepository creates a new chat message repository
func NewChatMessageRepository(db *sql.DB) *ChatMessageRepository {
	return &ChatMessageRepository{db: db}
}

// SaveMessage saves a chat message to the database
func (r *ChatMessageRepository) SaveMessage(msg *models.ChatMessage) error {
	// FIX #30: Comprehensive pre-save validation
	if err := validateChatMessageBeforeSave(msg); err != nil {
		return err
	}

	query := `
		INSERT INTO chat_messages (id, user_id, conversation_id, role, content, context_extracted, contact_mention, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	var contextJSON, contactJSON []byte
	if msg.ContextExtracted != nil {
		b, err := json.Marshal(msg.ContextExtracted)
		if err != nil {
			return fmt.Errorf("failed to marshal context: %w", err)
		}
		contextJSON = b
	}

	if msg.ContactMention != nil {
		b, err := json.Marshal(msg.ContactMention)
		if err != nil {
			return fmt.Errorf("failed to marshal contact mention: %w", err)
		}
		contactJSON = b
	}

	_, err := r.db.Exec(
		query,
		msg.ID,
		msg.UserID,
		msg.ConversationID,
		msg.Role,
		msg.Content,
		contextJSON,
		contactJSON,
		msg.CreatedAt,
	)

	return err
}

// GetMessage retrieves a message by ID
// FIX #6: Added userID parameter for data isolation
func (r *ChatMessageRepository) GetMessage(userID, messageID string) (*models.ChatMessage, error) {
	query := `
		SELECT id, user_id, conversation_id, role, content, context_extracted, contact_mention, created_at
		FROM chat_messages
		WHERE id = ? AND user_id = ?
	`

	var msg models.ChatMessage
	var contextJSON, contactJSON sql.NullString

	err := r.db.QueryRow(query, messageID, userID).Scan(
		&msg.ID,
		&msg.UserID,
		&msg.ConversationID,
		&msg.Role,
		&msg.Content,
		&contextJSON,
		&contactJSON,
		&msg.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("message not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	// Unmarshal JSON fields
	if contextJSON.Valid {
		err := json.Unmarshal([]byte(contextJSON.String), &msg.ContextExtracted)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal context: %w", err)
		}
	}

	if contactJSON.Valid {
		var contact models.ContactMentionDetected
		err := json.Unmarshal([]byte(contactJSON.String), &contact)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal contact mention: %w", err)
		}
		msg.ContactMention = &contact
	}

	return &msg, nil
}

// GetConversationHistory retrieves messages in a conversation
func (r *ChatMessageRepository) GetConversationHistory(userID, conversationID string, limit int) ([]*models.ChatMessage, error) {
	if limit <= 0 {
		limit = 50 // Default limit
	}

	query := `
		SELECT id, user_id, conversation_id, role, content, context_extracted, contact_mention, created_at
		FROM chat_messages
		WHERE user_id = ? AND conversation_id = ?
		ORDER BY created_at ASC
		LIMIT ?
	`

	rows, err := r.db.Query(query, userID, conversationID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []*models.ChatMessage
	for rows.Next() {
		var msg models.ChatMessage
		var contextJSON, contactJSON sql.NullString

		err := rows.Scan(
			&msg.ID,
			&msg.UserID,
			&msg.ConversationID,
			&msg.Role,
			&msg.Content,
			&contextJSON,
			&contactJSON,
			&msg.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		// Unmarshal JSON fields
		if contextJSON.Valid {
			err := json.Unmarshal([]byte(contextJSON.String), &msg.ContextExtracted)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal context: %w", err)
			}
		}

		if contactJSON.Valid {
			var contact models.ContactMentionDetected
			err := json.Unmarshal([]byte(contactJSON.String), &contact)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal contact mention: %w", err)
			}
			msg.ContactMention = &contact
		}

		messages = append(messages, &msg)
	}

	return messages, rows.Err()
}

// FIX #6: Added userID parameter for data isolation
// DeleteMessage removes a message (verifies ownership)
func (r *ChatMessageRepository) DeleteMessage(userID, messageID string) error {
	query := `DELETE FROM chat_messages WHERE id = ? AND user_id = ?`
	result, err := r.db.Exec(query, messageID, userID)
	if err != nil {
		log.Printf("[ChatRepository] Error deleting message: %v", err)
		return err
	}
	// FIX #5: Verify deletion succeeded
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("message not found or access denied")
	}
	return nil
}

// FIX #6: Added userID parameter for data isolation
// DeleteConversation removes all messages in a conversation (verifies ownership)
func (r *ChatMessageRepository) DeleteConversation(userID, conversationID string) error {
	query := `DELETE FROM chat_messages WHERE conversation_id = ? AND user_id = ?`
	result, err := r.db.Exec(query, conversationID, userID)
	if err != nil {
		log.Printf("[ChatRepository] Error deleting conversation: %v", err)
		return err
	}
	// FIX #5: Verify deletion succeeded
	if _, err := result.RowsAffected(); err != nil {
		return err
	}
	return nil
}

// GetMessageCount returns number of messages in a conversation
func (r *ChatMessageRepository) GetMessageCount(conversationID string) (int, error) {
	query := `SELECT COUNT(*) FROM chat_messages WHERE conversation_id = ?`
	var count int
	err := r.db.QueryRow(query, conversationID).Scan(&count)
	return count, err
}

// GetLastMessage gets the most recent message in a conversation
func (r *ChatMessageRepository) GetLastMessage(conversationID string) (*models.ChatMessage, error) {
	query := `
		SELECT id, user_id, conversation_id, role, content, context_extracted, contact_mention, created_at
		FROM chat_messages
		WHERE conversation_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`

	var msg models.ChatMessage
	var contextJSON, contactJSON sql.NullString

	err := r.db.QueryRow(query, conversationID).Scan(
		&msg.ID,
		&msg.UserID,
		&msg.ConversationID,
		&msg.Role,
		&msg.Content,
		&contextJSON,
		&contactJSON,
		&msg.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No messages yet
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get last message: %w", err)
	}

	// Unmarshal JSON fields
	if contextJSON.Valid {
		err := json.Unmarshal([]byte(contextJSON.String), &msg.ContextExtracted)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal context: %w", err)
		}
	}

	if contactJSON.Valid {
		var contact models.ContactMentionDetected
		err := json.Unmarshal([]byte(contactJSON.String), &contact)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal contact mention: %w", err)
		}
		msg.ContactMention = &contact
	}

	return &msg, nil
}

// DeleteOldMessages deletes messages older than the specified number of days
// This implements retention policy: keep last 30 days of messages per conversation
func (r *ChatMessageRepository) DeleteOldMessages(conversationID string, daysToKeep int) (int64, error) {
	if daysToKeep <= 0 {
		daysToKeep = 30 // Default: keep last 30 days
	}

	// Calculate timestamp for cutoff date
	cutoffTime := time.Now().AddDate(0, 0, -daysToKeep).Unix()

	query := `
		DELETE FROM chat_messages
		WHERE conversation_id = ? AND created_at < ?
	`

	result, err := r.db.Exec(query, conversationID, cutoffTime)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old messages: %w", err)
	}

	return result.RowsAffected()
}

// CleanupConversationHistory cleans up a conversation:
// - Deletes messages older than retention period
// - Keeps at least the last N messages even if they're old
func (r *ChatMessageRepository) CleanupConversationHistory(userID, conversationID string, daysToKeep, minMessagesToKeep int) error {
	if daysToKeep <= 0 {
		daysToKeep = 30
	}
	if minMessagesToKeep <= 0 {
		minMessagesToKeep = 10
	}

	// First, delete messages older than retention period
	cutoffTime := time.Now().AddDate(0, 0, -daysToKeep).Unix()

	query := `
		DELETE FROM chat_messages
		WHERE user_id = ?
		  AND conversation_id = ?
		  AND created_at < ?
		  AND id NOT IN (
		    SELECT id FROM chat_messages
		    WHERE user_id = ? AND conversation_id = ?
		    ORDER BY created_at DESC
		    LIMIT ?
		  )
	`

	_, err := r.db.Exec(query, userID, conversationID, cutoffTime, userID, conversationID, minMessagesToKeep)
	return err
}

// FIX #30: validateChatMessageBeforeSave ensures message is valid before database write
func validateChatMessageBeforeSave(msg *models.ChatMessage) error {
	// Required fields
	if msg.ID == "" {
		return fmt.Errorf("message must have id")
	}
	if msg.UserID == "" {
		return fmt.Errorf("message must have userId")
	}
	if msg.ConversationID == "" {
		return fmt.Errorf("message must have conversationId")
	}

	// Role must be valid enum
	validRoles := map[string]bool{
		"user":      true,
		"assistant": true,
		"system":    true,
	}
	if msg.Role == "" || !validRoles[msg.Role] {
		return fmt.Errorf("message must have valid role (got: %q)", msg.Role)
	}

	// Content length validation
	if len(msg.Content) > 100000 { // 100KB max
		return fmt.Errorf("message content too long (%d bytes, max 100000)", len(msg.Content))
	}

	// IDs should be reasonable length
	if len(msg.ID) > 255 {
		return fmt.Errorf("message id too long (%d chars, max 255)", len(msg.ID))
	}
	if len(msg.UserID) > 255 {
		return fmt.Errorf("userId too long (%d chars, max 255)", len(msg.UserID))
	}
	if len(msg.ConversationID) > 255 {
		return fmt.Errorf("conversationId too long (%d chars, max 255)", len(msg.ConversationID))
	}

	// CreatedAt should be reasonable (not in far future)
	// CreatedAt is Unix timestamp (int64)
	oneYearFromNow := time.Now().AddDate(1, 0, 0).Unix()
	if msg.CreatedAt > oneYearFromNow {
		return fmt.Errorf("message createdAt is in the future")
	}

	return nil
}
