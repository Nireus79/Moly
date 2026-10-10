package database

import (
	"database/sql"
	"fmt"
	"time"
)

// ConversationContextRepository stores per-conversation working state (previous extraction,
// change tracking) so a chat can resume after a restart.
type ConversationContextRepository struct {
	db *Database
}

func NewConversationContextRepository(db *Database) *ConversationContextRepository {
	return &ConversationContextRepository{db: db}
}

func (r *ConversationContextRepository) Save(userID, conversationID string, state []byte) error {
	return r.SaveWith(r.db.GetConnection(), userID, conversationID, state)
}

// SaveWith is Save through the given executor (for example a transaction).
func (r *ConversationContextRepository) SaveWith(ex Executor, userID, conversationID string, state []byte) error {
	_, err := ex.Exec(`
		INSERT INTO conversation_context (conversation_id, user_id, state, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(conversation_id) DO UPDATE SET state = excluded.state, updated_at = excluded.updated_at
		WHERE conversation_context.user_id = excluded.user_id`,
		conversationID, userID, string(state), time.Now().Unix())
	if err != nil {
		return fmt.Errorf("failed to save conversation context: %w", err)
	}
	return nil
}

// Load returns nil (and no error) when the conversation has no saved state for this user.
func (r *ConversationContextRepository) Load(userID, conversationID string) ([]byte, error) {
	var state string
	err := r.db.GetConnection().QueryRow(
		`SELECT state FROM conversation_context WHERE conversation_id = ? AND user_id = ?`,
		conversationID, userID).Scan(&state)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load conversation context: %w", err)
	}
	return []byte(state), nil
}
