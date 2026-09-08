package database

import (
	"database/sql"
	"encoding/json"
	"fmt"

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
	if msg.ID == "" || msg.UserID == "" || msg.ConversationID == "" {
		return fmt.Errorf("message must have id, userId, and conversationId")
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
func (r *ChatMessageRepository) GetMessage(messageID string) (*models.ChatMessage, error) {
	query := `
		SELECT id, user_id, conversation_id, role, content, context_extracted, contact_mention, created_at
		FROM chat_messages
		WHERE id = ?
	`

	var msg models.ChatMessage
	var contextJSON, contactJSON sql.NullString

	err := r.db.QueryRow(query, messageID).Scan(
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

// DeleteMessage removes a message
func (r *ChatMessageRepository) DeleteMessage(messageID string) error {
	query := `DELETE FROM chat_messages WHERE id = ?`
	_, err := r.db.Exec(query, messageID)
	return err
}

// DeleteConversation removes all messages in a conversation
func (r *ChatMessageRepository) DeleteConversation(conversationID string) error {
	query := `DELETE FROM chat_messages WHERE conversation_id = ?`
	_, err := r.db.Exec(query, conversationID)
	return err
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
