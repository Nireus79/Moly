package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
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
	msg := fmt.Sprintf("WriteBatch called: message=%v response=%v insights=%d pending=%v",
		req.Message != nil, req.Response != nil, len(req.Insights), req.PendingInput != nil)
	log.Printf("[BatchWriter] %s", msg)

	// Write to file immediately to debug
	if f, err := os.OpenFile("/tmp/moly_pending_input.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		fmt.Fprintf(f, "[BATCH_WRITE_START] %s\n", msg)
		f.Close()
	}

	if req.PendingInput != nil {
		log.Printf("[BatchWriter] DEBUG: PendingInput details: UserID=%s Type=%s Subtype=%s CreatedAt=%d",
			req.PendingInput.UserID, req.PendingInput.Type, req.PendingInput.Subtype, req.PendingInput.CreatedAt)
	}

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
		log.Printf("[BatchWriter] Checking pending input: %v", req.PendingInput != nil)
		debugLog(fmt.Sprintf("About to check pending input: %v", req.PendingInput != nil))

		if req.PendingInput != nil {
			debugLog(fmt.Sprintf("PENDING INPUT FOUND! Type=%s", req.PendingInput.Type))
			log.Printf("[BatchWriter] PENDING INPUT FOUND: type=%s", req.PendingInput.Type)
			if err := bw.savePendingInput(tx, req.PendingInput); err != nil {
				debugLog(fmt.Sprintf("ERROR saving: %v", err))
				return fmt.Errorf("failed to save pending input: %w", err)
			}
		} else {
			debugLog("No pending input to save")
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
		log.Printf("[BatchWriter] ❌ BATCH WRITE FAILED: %v (type: %T)", err, err)
		log.Printf("[BatchWriter] Error details: %+v", err)
		return err
	}

	log.Printf("[BatchWriter] ✅ Batch write completed successfully")
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
	log.Printf("[BatchWriter] savePendingInput called: user=%s type=%s conv=%s", pi.UserID, pi.Type, pi.ConversationID)

	// Debug: write to file
	debugLog(fmt.Sprintf("savePendingInput: user=%s type=%s conv=%s created_at=%d",
		pi.UserID, pi.Type, pi.ConversationID, pi.CreatedAt))

	query := `
		INSERT INTO pending_input (user_id, conversation_id, type, subtype, question, context, created_at, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := tx.Exec(query, pi.UserID, pi.ConversationID, pi.Type, pi.Subtype, pi.Question, string(pi.Context), pi.CreatedAt, string(pi.Metadata))
	if err != nil {
		msg := fmt.Sprintf("ERROR savePendingInput: %v", err)
		log.Printf("[BatchWriter] %s", msg)
		debugLog(msg)
		return fmt.Errorf("insert pending input failed: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	msg := fmt.Sprintf("✅ Saved pending_input: rows=%d", rowsAffected)
	log.Printf("[BatchWriter] %s", msg)
	debugLog(msg)
	return nil
}

// debugLog - Write to debug log file
func debugLog(msg string) {
	if f, err := os.OpenFile("/tmp/moly_pending_input.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		defer f.Close()
		fmt.Fprintf(f, "[%s] %s\n", time.Now().Format("15:04:05"), msg)
	}
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
