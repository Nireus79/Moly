package database

import (
	"fmt"
	"log"
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

// ClarificationRecord represents a historical clarification
type ClarificationRecord struct {
	ID                string
	ConversationID    string
	ClarificationType string
	QuestionText      string
	Status            string // "asked", "answered", "skipped"
	AskedAt           int64
	AnsweredAt        int64
}

// HasBeenAsked checks if a clarification with this text has been asked in this conversation
// FIX #6: Added userID for data isolation
func (chr *ClarificationHistoryRepository) HasBeenAsked(
	userID string,
	conversationID string,
	clarificationType string,
	questionText string,
) (bool, error) {
	if chr.db == nil {
		return false, nil
	}

	query := `
		SELECT COUNT(*) as count
		FROM clarification_questions
		WHERE user_id = ?
		  AND conversation_id = ?
		  AND clarification_type = ?
		  AND question_text = ?
		  AND status IN ('active', 'answered', 'skipped')
	`

	row := chr.db.QueryRow(query, userID, conversationID, clarificationType, questionText)
	var count int
	if err := row.Scan(&count); err != nil {
		log.Printf("[ClarificationHistory] Error checking if asked: %v", err)
		return false, err
	}

	return count > 0, nil
}

// HasBeenAnswered checks if a clarification has been answered
// FIX #6: Added userID for data isolation
func (chr *ClarificationHistoryRepository) HasBeenAnswered(
	userID string,
	conversationID string,
	clarificationType string,
	questionText string,
) (bool, error) {
	if chr.db == nil {
		return false, nil
	}

	query := `
		SELECT COUNT(*) as count
		FROM clarification_questions
		WHERE user_id = ?
		  AND conversation_id = ?
		  AND clarification_type = ?
		  AND question_text = ?
		  AND status = 'answered'
		  AND answered_at > 0
	`

	row := chr.db.QueryRow(query, userID, conversationID, clarificationType, questionText)
	var count int
	if err := row.Scan(&count); err != nil {
		log.Printf("[ClarificationHistory] Error checking if answered: %v", err)
		return false, err
	}

	return count > 0, nil
}

// GetAskedCount returns how many times a clarification type has been asked
// FIX #6: Added userID for data isolation
func (chr *ClarificationHistoryRepository) GetAskedCount(
	userID string,
	conversationID string,
	clarificationType string,
) (int, error) {
	if chr.db == nil {
		return 0, nil
	}

	query := `
		SELECT COUNT(*) as count
		FROM clarification_questions
		WHERE user_id = ?
		  AND conversation_id = ?
		  AND clarification_type = ?
		  AND status IN ('active', 'answered', 'skipped')
	`

	row := chr.db.QueryRow(query, userID, conversationID, clarificationType)
	var count int
	if err := row.Scan(&count); err != nil {
		log.Printf("[ClarificationHistory] Error getting asked count: %v", err)
		return 0, err
	}

	return count, nil
}

// GetRecentClarifications retrieves recent clarifications for the conversation
// FIX #6: Added userID for data isolation
func (chr *ClarificationHistoryRepository) GetRecentClarifications(
	userID string,
	conversationID string,
	limit int,
) ([]ClarificationRecord, error) {
	if chr.db == nil {
		return []ClarificationRecord{}, nil
	}

	if limit <= 0 {
		limit = 10
	}

	query := `
		SELECT id, conversation_id, clarification_type, question_text, status, created_at, answered_at
		FROM clarification_questions
		WHERE user_id = ? AND conversation_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := chr.db.Query(query, userID, conversationID, limit)
	if err != nil {
		log.Printf("[ClarificationHistory] Error getting recent clarifications: %v", err)
		return []ClarificationRecord{}, err
	}
	defer rows.Close()

	records := make([]ClarificationRecord, 0)
	for rows.Next() {
		var record ClarificationRecord
		if err := rows.Scan(
			&record.ID,
			&record.ConversationID,
			&record.ClarificationType,
			&record.QuestionText,
			&record.Status,
			&record.AskedAt,
			&record.AnsweredAt,
		); err != nil {
			log.Printf("[ClarificationHistory] Error scanning clarification: %v", err)
			continue
		}
		records = append(records, record)
	}

	return records, nil
}

// MarkAsAsked marks a clarification as having been asked (status="active")
// This is called when a clarification question is generated and ready to ask
// FIX #6: Added userID for data isolation
func (chr *ClarificationHistoryRepository) MarkAsAsked(userID, questionID string) error {
	if chr.db == nil {
		return nil
	}

	query := `
		UPDATE clarification_questions
		SET status = 'active'
		WHERE id = ? AND user_id = ?
	`

	result, err := chr.db.Exec(query, questionID, userID)
	if err != nil {
		return fmt.Errorf("error marking clarification as asked: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("clarification not found or access denied")
	}

	log.Printf("[ClarificationHistory] Marked clarification %s as asked", questionID)

	return nil
}

// MarkAsAnswered marks a clarification as answered with timestamp
// This is called when user provides a response to the clarification
// FIX #6: Added userID for data isolation
func (chr *ClarificationHistoryRepository) MarkAsAnswered(userID, questionID string, answeredAt int64) error {
	if chr.db == nil {
		return nil
	}

	query := `
		UPDATE clarification_questions
		SET status = 'answered', answered_at = ?
		WHERE id = ? AND user_id = ?
	`

	result, err := chr.db.Exec(query, answeredAt, questionID, userID)
	if err != nil {
		return fmt.Errorf("error marking clarification as answered: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("clarification not found or access denied")
	}

	log.Printf("[ClarificationHistory] Marked clarification %s as answered", questionID)

	return nil
}

// MarkAsSkipped marks a clarification as skipped (not needed/answered differently)
// FIX #6: Added userID for data isolation
func (chr *ClarificationHistoryRepository) MarkAsSkipped(userID, questionID string) error {
	if chr.db == nil {
		return nil
	}

	query := `
		UPDATE clarification_questions
		SET status = 'skipped'
		WHERE id = ? AND user_id = ?
	`

	result, err := chr.db.Exec(query, questionID, userID)
	if err != nil {
		return fmt.Errorf("error marking clarification as skipped: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("clarification not found or access denied")
	}

	log.Printf("[ClarificationHistory] Marked clarification %s as skipped", questionID)

	return nil
}

// ShouldAskClarification determines if a clarification should be asked
// Returns false if already asked or answered, true if fresh/new question
// FIX #6: Added userID for data isolation
func (chr *ClarificationHistoryRepository) ShouldAskClarification(
	userID string,
	conversationID string,
	clarificationType string,
	questionText string,
) (bool, error) {
	// Check if already answered - if yes, don't ask again
	answered, err := chr.HasBeenAnswered(userID, conversationID, clarificationType, questionText)
	if err != nil {
		return true, err // On error, proceed with asking (fail open)
	}
	if answered {
		log.Printf("[ClarificationHistory] Skipping clarification (already answered): %s", clarificationType)
		return false, nil
	}

	// Check if already asked recently
	asked, err := chr.HasBeenAsked(userID, conversationID, clarificationType, questionText)
	if err != nil {
		return true, err // On error, proceed with asking (fail open)
	}

	if asked {
		// Already asked - but not yet answered, might ask again
		// depending on phase/maturity (this could be enhanced)
		log.Printf("[ClarificationHistory] Clarification already asked: %s", clarificationType)
		// For Phase 5, we'll be conservative: don't ask twice
		return false, nil
	}

	return true, nil
}
