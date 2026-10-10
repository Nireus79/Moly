package database

import (
	"fmt"
	"strings"
)

// ClarificationHistoryRepository tracks which clarifications have been asked/answered
// FIX #5 (Phase 5): Prevents redundant clarification questions across messages
type ClarificationHistoryRepository struct {
	db *Database
}

// NewClarificationHistoryRepository creates a new history repository
func NewClarificationHistoryRepository(db *Database) *ClarificationHistoryRepository {
	return &ClarificationHistoryRepository{db: db}
}

// AnsweredQuestionTexts returns the texts of answered clarifications in a conversation, newest first.
func (chr *ClarificationHistoryRepository) AnsweredQuestionTexts(userID, conversationID string, limit int) ([]string, error) {
	if chr.db == nil {
		return nil, nil
	}
	rows, err := chr.db.Query(`
		SELECT question_text FROM clarification_questions
		WHERE user_id = ? AND conversation_id = ? AND status = 'answered'
		ORDER BY answered_at DESC
		LIMIT ?
	`, userID, conversationID, limit)
	if err != nil {
		return nil, fmt.Errorf("error loading answered clarifications: %w", err)
	}
	defer rows.Close()

	var texts []string
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err != nil {
			return nil, err
		}
		texts = append(texts, text)
	}
	return texts, rows.Err()
}

// HasQuestionTextBeenAnswered reports whether a question with exactly this text was answered in the conversation.
// A skipped question is pending, not answered: it can be asked again.
func (chr *ClarificationHistoryRepository) HasQuestionTextBeenAnswered(userID, conversationID, questionText string) (bool, error) {
	if chr.db == nil {
		return false, nil
	}
	var count int
	err := chr.db.QueryRow(`
		SELECT COUNT(*) FROM clarification_questions
		WHERE user_id = ? AND conversation_id = ? AND question_text = ? AND status = 'answered'
	`, userID, conversationID, questionText).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("error checking answered clarification: %w", err)
	}
	return count > 0, nil
}

// PendingQuestion is a question the user skipped. It stays open: it is asked again when it is still the most important
// thing to know, and it counts as an open gap in maturity.
type PendingQuestion struct {
	ID       string
	Text     string
	Priority int    // 1 high, 2 medium, 3 low (the severity of the gap when it was asked)
	Status   string // "active" (asked, waiting) or "skipped" (pending)
}

// OpenQuestions returns the questions of a conversation that are not resolved: asked and waiting (active) or skipped
// (pending), oldest first.
func (chr *ClarificationHistoryRepository) OpenQuestions(userID, conversationID string) ([]PendingQuestion, error) {
	return chr.questionsWithStatus(userID, conversationID, "active", "skipped")
}

// PendingSkipped returns the questions the user skipped (or left unanswered) in a conversation, oldest first.
func (chr *ClarificationHistoryRepository) PendingSkipped(userID, conversationID string) ([]PendingQuestion, error) {
	return chr.questionsWithStatus(userID, conversationID, "skipped")
}

func (chr *ClarificationHistoryRepository) questionsWithStatus(userID, conversationID string, statuses ...string) ([]PendingQuestion, error) {
	if chr.db == nil {
		return nil, nil
	}
	marks := strings.TrimSuffix(strings.Repeat("?,", len(statuses)), ",")
	args := []interface{}{userID, conversationID}
	for _, st := range statuses {
		args = append(args, st)
	}
	rows, err := chr.db.Query(`
		SELECT id, question_text, priority, status FROM clarification_questions
		WHERE user_id = ? AND conversation_id = ? AND status IN (`+marks+`)
		ORDER BY created_at ASC
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("error loading pending clarifications: %w", err)
	}
	defer rows.Close()
	var out []PendingQuestion
	for rows.Next() {
		var q PendingQuestion
		if err := rows.Scan(&q.ID, &q.Text, &q.Priority, &q.Status); err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, rows.Err()
}

// SetQuestionStatus moves one question to a new status. Answering a pending question stamps the answer time.
func (chr *ClarificationHistoryRepository) SetQuestionStatus(userID, id, status string, at int64) error {
	if chr.db == nil {
		return nil
	}
	if status == "answered" {
		_, err := chr.db.Exec(`UPDATE clarification_questions SET status = 'answered', answered_at = ? WHERE id = ? AND user_id = ?`, at, id, userID)
		return err
	}
	_, err := chr.db.Exec(`UPDATE clarification_questions SET status = ? WHERE id = ? AND user_id = ?`, status, id, userID)
	return err
}

// ReactivateSkippedByText puts a skipped question with exactly this text back to active, because Moly is asking it
// again. It reports whether there was one: if so, no new row is needed.
func (chr *ClarificationHistoryRepository) ReactivateSkippedByText(userID, conversationID, text string) (bool, error) {
	if chr.db == nil {
		return false, nil
	}
	res, err := chr.db.Exec(`
		UPDATE clarification_questions SET status = 'active', answered_at = 0
		WHERE user_id = ? AND conversation_id = ? AND question_text = ? AND status = 'skipped'
	`, userID, conversationID, text)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// CancelPending drops the pending questions of a conversation: they belonged to a goal the user left.
func (chr *ClarificationHistoryRepository) CancelPending(userID, conversationID string) (int64, error) {
	if chr.db == nil {
		return 0, nil
	}
	res, err := chr.db.Exec(`UPDATE clarification_questions SET status = 'cancelled' WHERE user_id = ? AND conversation_id = ? AND status = 'skipped'`, userID, conversationID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
