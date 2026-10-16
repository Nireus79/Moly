package database

import (
	"database/sql"
	"fmt"
	"strings"
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
		// a person the user named belongs to the user: when another conversation mentions them they stay (their home moves there); the rest of
		// the conversation's memory (its unnamed people, roles, traits, questions, facts, goal, audit trail) goes with it through the cascade
		`UPDATE memory_people SET conversation_id = (SELECT m.conversation_id FROM memory_mentions m WHERE m.person_id = memory_people.id AND m.conversation_id != ? LIMIT 1)
		 WHERE user_id = ? AND conversation_id = ? AND name != '' AND status = 'active'
		 AND EXISTS (SELECT 1 FROM memory_mentions m WHERE m.person_id = memory_people.id AND m.conversation_id != ?)`,
		`DELETE FROM chat_messages WHERE conversation_id = ? AND user_id = ?`,
		`DELETE FROM conversation_maturity WHERE conversation_id = ? AND user_id = ?`,
		`DELETE FROM group_references WHERE conversation_id = ? AND user_id = ?`,
		`DELETE FROM response_validations WHERE conversation_id = ? AND user_id = ?`,
		`DELETE FROM messages WHERE conversation_id = ?`,
		// what is unresolved belongs to the conversation it came from and goes with it: the people still waiting for a name,
		// and the conflicts it saved (a named person belongs to the user and stays)
		`DELETE FROM contacts WHERE origin_conversation_id = ? AND user_id = ? AND name_status != 'named'`,
		`DELETE FROM context_conflicts WHERE conversation_id = ? AND user_id = ?`,
		// insights of the conversation the user did not confirm (the confirmed ones are the user's own)
		`DELETE FROM reflections WHERE conversation_id = ? AND user_id = ? AND status != 'approved'`,
		`DELETE FROM conversations WHERE id = ? AND user_id = ?`,
	}
	for _, q := range steps {
		args := []interface{}{conversationID, userID}
		if q == `DELETE FROM messages WHERE conversation_id = ?` {
			args = args[:1]
		}
		if strings.HasPrefix(q, "UPDATE memory_people") {
			args = []interface{}{conversationID, userID, conversationID, conversationID}
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
