package database

import (
	"database/sql"
	"fmt"
)

// DeleteConversationData removes a conversation and everything that belongs to it.
// Child rows are deleted explicitly because some tables reference conversations without a cascade.
func DeleteConversationData(conn *sql.DB, userID, conversationID string) error {
	tx, err := conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to start delete: %w", err)
	}
	defer tx.Rollback()

	steps := []string{
		`DELETE FROM chat_messages WHERE conversation_id = ? AND user_id = ?`,
		`DELETE FROM conversation_maturity WHERE conversation_id = ? AND user_id = ?`,
		`DELETE FROM group_references WHERE conversation_id = ? AND user_id = ?`,
		`DELETE FROM response_validations WHERE conversation_id = ? AND user_id = ?`,
		`DELETE FROM messages WHERE conversation_id = ?`,
		`DELETE FROM conversations WHERE id = ? AND user_id = ?`,
	}
	for _, q := range steps {
		args := []interface{}{conversationID, userID}
		if q == `DELETE FROM messages WHERE conversation_id = ?` {
			args = args[:1]
		}
		if _, err := tx.Exec(q, args...); err != nil {
			return fmt.Errorf("failed to delete conversation data: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit delete: %w", err)
	}
	return nil
}
