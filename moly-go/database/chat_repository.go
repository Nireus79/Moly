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
		msg.Content = StoredReplyText(msg.Content)

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
