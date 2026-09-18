package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"moly/database"
	"moly/models"
)

// BatchWriteRequest - Collection of items to write in single transaction
type BatchWriteRequest struct {
	UserID           string
	ConversationID   string
	Message          *models.ChatMessage
	Response         *models.ConversationResponse
	Insights         []models.Reflection
	PendingInput     *database.PendingInput
	UpdatedAboutMe   *models.AboutMe
}

// BatchWriter - Writes multiple items in a single transaction
type BatchWriter struct {
	db *database.Database
}

// NewBatchWriter - Create new batch writer
func NewBatchWriter(db *database.Database) *BatchWriter {
	return &BatchWriter{db: db}
}

// WriteBatch - Write all items in request to database in single transaction
// Returns error if any write fails; all writes are rolled back
func (bw *BatchWriter) WriteBatch(req BatchWriteRequest) error {
	log.Printf("[BatchWriter] Starting batch write: message=%v response=%v insights=%d pending=%v",
		req.Message != nil, req.Response != nil, len(req.Insights), req.PendingInput != nil)

	// Use database Transaction helper for automatic rollback
	err := bw.db.Transaction(func(tx *sql.Tx) error {
		// 1. Save user message
		if req.Message != nil {
			if err := bw.saveMessage(tx, req.Message); err != nil {
				return fmt.Errorf("failed to save message: %w", err)
			}
		}

		// 2. Save assistant response
		if req.Response != nil {
			if err := bw.saveResponse(tx, req.UserID, req.ConversationID, req.Response); err != nil {
				return fmt.Errorf("failed to save response: %w", err)
			}
		}

		// 3. Save insights
		for _, insight := range req.Insights {
			if err := bw.saveInsight(tx, &insight); err != nil {
				return fmt.Errorf("failed to save insight: %w", err)
			}
		}

		// 4. Save pending input if present
		if req.PendingInput != nil {
			if err := bw.savePendingInput(tx, req.PendingInput); err != nil {
				return fmt.Errorf("failed to save pending input: %w", err)
			}
		}

		// 5. Update AboutMe if changed
		if req.UpdatedAboutMe != nil {
			if err := bw.updateAboutMe(tx, req.UserID, req.UpdatedAboutMe); err != nil {
				return fmt.Errorf("failed to update about me: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		log.Printf("[BatchWriter] Batch write failed: %v", err)
		return err
	}

	log.Printf("[BatchWriter] Batch write completed successfully")
	return nil
}

// saveMessage - Save chat message in transaction
func (bw *BatchWriter) saveMessage(tx *sql.Tx, msg *models.ChatMessage) error {
	contextJSON, _ := json.Marshal(msg.ContextExtracted)
	contactJSON, _ := json.Marshal(msg.ContactMention)

	query := `
		INSERT INTO chat_messages (id, user_id, conversation_id, role, content, context_extracted, contact_mention, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := tx.Exec(query, msg.ID, msg.UserID, msg.ConversationID, msg.Role, msg.Content, string(contextJSON), string(contactJSON), msg.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert message failed: %w", err)
	}

	log.Printf("[BatchWriter] Message saved: %s", msg.ID)
	return nil
}

// saveResponse - Save agent response in transaction
func (bw *BatchWriter) saveResponse(tx *sql.Tx, userID, conversationID string, resp *models.ConversationResponse) error {
	// Response is saved as assistant message in chat_messages
	query := `
		INSERT INTO chat_messages (id, user_id, conversation_id, role, content, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	responseID := fmt.Sprintf("%s-response-%d", conversationID, time.Now().UnixNano())

	_, err := tx.Exec(query, responseID, userID, conversationID, "assistant", resp.Response, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("insert response failed: %w", err)
	}

	log.Printf("[BatchWriter] Response saved: %s", responseID)
	return nil
}

// saveInsight - Save reflection/insight in transaction
func (bw *BatchWriter) saveInsight(tx *sql.Tx, insight *models.Reflection) error {
	characteristicsJSON, _ := json.Marshal(insight.Characteristics)
	interestsJSON, _ := json.Marshal(insight.Interests)
	intentionsJSON, _ := json.Marshal(insight.Intentions)

	query := `
		INSERT INTO reflections (conversation_id, contact_id, characteristics, interests, intentions, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := tx.Exec(query, insight.ConversationID, insight.ContactID, string(characteristicsJSON), string(interestsJSON), string(intentionsJSON), insight.Status, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("insert insight failed: %w", err)
	}

	log.Printf("[BatchWriter] Insight saved for conversation %s", insight.ConversationID)
	return nil
}

// savePendingInput - Save pending input in transaction
func (bw *BatchWriter) savePendingInput(tx *sql.Tx, pi *database.PendingInput) error {
	query := `
		INSERT INTO pending_input (user_id, conversation_id, type, subtype, question, context, created_at, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := tx.Exec(query, pi.UserID, pi.ConversationID, pi.Type, pi.Subtype, pi.Question, string(pi.Context), time.Now().Unix(), string(pi.Metadata))
	if err != nil {
		return fmt.Errorf("insert pending input failed: %w", err)
	}

	log.Printf("[BatchWriter] Pending input saved: type=%s subtype=%s", pi.Type, pi.Subtype)
	return nil
}

// updateAboutMe - Update about me in transaction
func (bw *BatchWriter) updateAboutMe(tx *sql.Tx, userID string, aboutMe *models.AboutMe) error {
	valuesJSON, _ := json.Marshal(aboutMe.Values)
	goalsJSON, _ := json.Marshal(aboutMe.Goals)
	now := time.Now().Unix()

	query := `
		INSERT INTO about_me (user_id, communication_style, core_values, tone_preference, goals, created_at, updated_at, version)
		VALUES (?, ?, ?, ?, ?, ?, ?, 1)
		ON CONFLICT(user_id) DO UPDATE SET
			communication_style = excluded.communication_style,
			core_values = excluded.core_values,
			tone_preference = excluded.tone_preference,
			goals = excluded.goals,
			updated_at = excluded.updated_at,
			version = version + 1
	`

	_, err := tx.Exec(query, userID, aboutMe.CommunicationStyle, string(valuesJSON), aboutMe.PreferredTone, string(goalsJSON), now, now)
	if err != nil {
		return fmt.Errorf("update about me failed: %w", err)
	}

	log.Printf("[BatchWriter] AboutMe updated for user %s", userID)
	return nil
}
