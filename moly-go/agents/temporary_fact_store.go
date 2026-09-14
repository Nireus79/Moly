package agents

import (
	"moly/schema"
	"database/sql"
	"fmt"
	"log"
	"time"

	"moly/database"
)

// TemporaryFact holds an extracted fact waiting for clarification answers
type TemporaryFact struct {
	FactID              string                   `json:"factId"`
	FactType            string                   `json:"type"`
	FactValue           string                   `json:"value"`
	Evidence            string                   `json:"evidence"`
	Confidence          float64                  `json:"confidence"`
	AttributedTo        string                   `json:"attributedTo"` // "user", "contact_name", etc
	ConversationID      string                   `json:"conversationId"`
	LinkedQuestionIDs   []string                 `json:"linkedQuestionIds"`
	LinkedQuestions     []*schema.ClarificationQuestion `json:"linkedQuestions"` // Full question objects
	ClarificationAnswers map[string]string       `json:"clarificationAnswers"` // questionId -> answer
	Status              string                   `json:"status"` // "pending", "partially_answered", "complete"
	CreatedAt           int64                    `json:"createdAt"`
	UpdatedAt           int64                    `json:"updatedAt"`
	ExpiresAt           int64                    `json:"expiresAt"`
}

// TemporaryFactStore manages facts awaiting clarification in database
type TemporaryFactStore struct {
	db     *database.Database
	userID string
}

// NewTemporaryFactStore creates database-backed fact store
func NewTemporaryFactStore(db *database.Database, userID string) *TemporaryFactStore {
	return &TemporaryFactStore{
		db:     db,
		userID: userID,
	}
}

// Store persists a fact awaiting clarification
func (s *TemporaryFactStore) Store(fact *TemporaryFact) error {
	if fact.FactID == "" {
		return fmt.Errorf("fact ID required")
	}

	conn := s.db.GetConnection()
	now := time.Now().Unix()
	expiresAt := now + (7 * 24 * 3600) // 7 days

	fact.CreatedAt = now
	fact.UpdatedAt = now
	fact.ExpiresAt = expiresAt

	// Insert or update pending clarification
	_, err := conn.Exec(`
		INSERT INTO pending_clarifications
		(id, user_id, conversation_id, fact_id, fact_type, fact_value, attributed_to,
		 evidence, confidence, status, created_at, updated_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(fact_id) DO UPDATE SET
		  status = excluded.status,
		  updated_at = excluded.updated_at
	`,
		fmt.Sprintf("pending_%d_%s", now, fact.FactID),
		s.userID,
		fact.ConversationID,
		fact.FactID,
		fact.FactType,
		fact.FactValue,
		fact.AttributedTo,
		fact.Evidence,
		fact.Confidence,
		"pending",
		now,
		now,
		expiresAt,
	)

	if err != nil {
		log.Printf("[TemporaryFactStore] ERROR storing fact %s: %v", fact.FactID, err)
		return err
	}

	// Store linked questions
	for i, q := range fact.LinkedQuestions {
		_, err := conn.Exec(`
			INSERT INTO clarification_questions
			(id, pending_clarification_id, sequence, question_text, question_type, context, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`,
			q.ID,
			fmt.Sprintf("pending_%d_%s", now, fact.FactID),
			i+1,
			q.Question,
			q.Type,
			q.Context,
			now,
		)
		if err != nil {
			log.Printf("[TemporaryFactStore] WARNING: Failed to store question %s: %v", q.ID, err)
		}
	}

	log.Printf("[TemporaryFactStore] ✓ Persisted fact_id=%s with %d questions", fact.FactID, len(fact.LinkedQuestions))
	return nil
}

// Get retrieves a pending fact by ID
func (s *TemporaryFactStore) Get(factID string) (*TemporaryFact, error) {
	conn := s.db.GetConnection()

	var id, factType, factValue, attributedTo, evidence, conversationID, status string
	var confidence float64
	var createdAt, updatedAt, expiresAt int64

	err := conn.QueryRow(`
		SELECT id, fact_type, fact_value, attributed_to, evidence, confidence,
		       conversation_id, status, created_at, updated_at, expires_at
		FROM pending_clarifications
		WHERE fact_id = ? AND user_id = ? AND status != 'abandoned'
	`, factID, s.userID).Scan(
		&id, &factType, &factValue, &attributedTo, &evidence, &confidence,
		&conversationID, &status, &createdAt, &updatedAt, &expiresAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("fact not found: %s", factID)
	}
	if err != nil {
		log.Printf("[TemporaryFactStore] ERROR getting fact: %v", err)
		return nil, err
	}

	fact := &TemporaryFact{
		FactID:               factID,
		FactType:             factType,
		FactValue:            factValue,
		AttributedTo:         attributedTo,
		Evidence:             evidence,
		Confidence:           confidence,
		ConversationID:       conversationID,
		Status:               status,
		CreatedAt:            createdAt,
		UpdatedAt:            updatedAt,
		ExpiresAt:            expiresAt,
		LinkedQuestionIDs:    []string{},
		LinkedQuestions:      []*schema.ClarificationQuestion{},
		ClarificationAnswers: make(map[string]string),
	}

	// Load linked questions
	s.loadQuestions(fact)

	return fact, nil
}

// GetPendingForUser retrieves all pending clarifications for user
func (s *TemporaryFactStore) GetPendingForUser() ([]*TemporaryFact, error) {
	conn := s.db.GetConnection()

	rows, err := conn.Query(`
		SELECT id, fact_id, fact_type, fact_value, attributed_to, evidence, confidence,
		       conversation_id, status, created_at, updated_at, expires_at
		FROM pending_clarifications
		WHERE user_id = ? AND status IN ('pending', 'partially_answered')
		AND expires_at > ?
		ORDER BY updated_at DESC
		LIMIT 5
	`, s.userID, time.Now().Unix())

	if err != nil {
		log.Printf("[TemporaryFactStore] ERROR querying pending: %v", err)
		return nil, err
	}
	defer rows.Close()

	var facts []*TemporaryFact
	for rows.Next() {
		var id, factID, factType, factValue, attributedTo, evidence, conversationID, status string
		var confidence float64
		var createdAt, updatedAt, expiresAt int64

		if err := rows.Scan(&id, &factID, &factType, &factValue, &attributedTo, &evidence,
			&confidence, &conversationID, &status, &createdAt, &updatedAt, &expiresAt); err != nil {
			log.Printf("[TemporaryFactStore] Warning: skipping row - %v", err)
			continue
		}

		fact := &TemporaryFact{
			FactID:               factID,
			FactType:             factType,
			FactValue:            factValue,
			AttributedTo:         attributedTo,
			Evidence:             evidence,
			Confidence:           confidence,
			ConversationID:       conversationID,
			Status:               status,
			CreatedAt:            createdAt,
			UpdatedAt:            updatedAt,
			ExpiresAt:            expiresAt,
			LinkedQuestionIDs:    []string{},
			LinkedQuestions:      []*schema.ClarificationQuestion{},
			ClarificationAnswers: make(map[string]string),
		}

		s.loadQuestions(fact)
		facts = append(facts, fact)
	}

	log.Printf("[TemporaryFactStore] Retrieved %d pending clarifications for user", len(facts))
	return facts, nil
}

// loadQuestions loads all questions and answers for a fact
func (s *TemporaryFactStore) loadQuestions(fact *TemporaryFact) {
	conn := s.db.GetConnection()

	// Get pending clarification ID
	var pendingID string
	err := conn.QueryRow(
		"SELECT id FROM pending_clarifications WHERE fact_id = ?",
		fact.FactID,
	).Scan(&pendingID)

	if err != nil {
		log.Printf("[TemporaryFactStore] WARNING: Could not find pending ID: %v", err)
		return
	}

	// Load questions
	rows, err := conn.Query(`
		SELECT id, sequence, question_text, question_type, context
		FROM clarification_questions
		WHERE pending_clarification_id = ?
		ORDER BY sequence ASC
	`, pendingID)

	if err != nil {
		log.Printf("[TemporaryFactStore] WARNING: Failed to load questions: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, questionText, questionType, context string
		var sequence int

		if err := rows.Scan(&id, &sequence, &questionText, &questionType, &context); err != nil {
			continue
		}

		fact.LinkedQuestionIDs = append(fact.LinkedQuestionIDs, id)
		fact.LinkedQuestions = append(fact.LinkedQuestions, &schema.ClarificationQuestion{
			ID:       id,
			Question: questionText,
			Type:     questionType,
			Context:  context,
		})
	}

	// Load answers
	for _, qID := range fact.LinkedQuestionIDs {
		var answer string
		err := conn.QueryRow(
			"SELECT user_answer FROM clarification_answers WHERE clarification_question_id = ?",
			qID,
		).Scan(&answer)

		if err == nil {
			fact.ClarificationAnswers[qID] = answer
		}
	}

	// Update status based on answers
	if len(fact.ClarificationAnswers) == len(fact.LinkedQuestionIDs) {
		fact.Status = "complete"
	} else if len(fact.ClarificationAnswers) > 0 {
		fact.Status = "partially_answered"
	}
}

// RecordAnswer saves user's answer to a question
func (s *TemporaryFactStore) RecordAnswer(factID, questionID, answer string) error {
	conn := s.db.GetConnection()
	now := time.Now().Unix()

	// Insert answer
	_, err := conn.Exec(`
		INSERT INTO clarification_answers (id, clarification_question_id, user_answer, answered_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(clarification_question_id) DO UPDATE SET
		  user_answer = excluded.user_answer,
		  answered_at = excluded.answered_at
	`,
		fmt.Sprintf("answer_%d_%s", now, questionID),
		questionID,
		answer,
		now,
	)

	if err != nil {
		log.Printf("[TemporaryFactStore] ERROR recording answer: %v", err)
		return err
	}

	// Update pending clarification updated_at
	conn.Exec(
		"UPDATE pending_clarifications SET updated_at = ? WHERE fact_id = ?",
		now, factID,
	)

	log.Printf("[TemporaryFactStore] ✓ Recorded answer for fact=%s question=%s", factID, questionID)
	return nil
}

// IsComplete checks if all questions answered for a fact
func (s *TemporaryFactStore) IsComplete(factID string) bool {
	fact, err := s.Get(factID)
	if err != nil {
		return false
	}
	return len(fact.ClarificationAnswers) >= len(fact.LinkedQuestionIDs) && len(fact.LinkedQuestionIDs) > 0
}

// RemainingQuestionsWithObjects returns unanswered questions as full objects
func (s *TemporaryFactStore) RemainingQuestionsWithObjects(factID string) []*schema.ClarificationQuestion {
	fact, err := s.Get(factID)
	if err != nil {
		return []*schema.ClarificationQuestion{}
	}

	var remaining []*schema.ClarificationQuestion
	for i, q := range fact.LinkedQuestions {
		if _, answered := fact.ClarificationAnswers[q.ID]; !answered {
			remaining = append(remaining, q)
		}
		if i >= len(fact.LinkedQuestions)-1 {
			break
		}
	}

	return remaining
}

// Remove marks clarification as complete
func (s *TemporaryFactStore) Remove(factID string) error {
	conn := s.db.GetConnection()
	now := time.Now().Unix()

	_, err := conn.Exec(`
		UPDATE pending_clarifications
		SET status = 'complete', updated_at = ?
		WHERE fact_id = ? AND user_id = ?
	`, now, factID, s.userID)

	if err == nil {
		log.Printf("[TemporaryFactStore] ✓ Marked fact_id=%s as complete", factID)
	}
	return err
}

// GetAnswers retrieves all answers for a fact
func (s *TemporaryFactStore) GetAnswers(factID string) []string {
	fact, err := s.Get(factID)
	if err != nil {
		return []string{}
	}

	var answers []string
	for _, qID := range fact.LinkedQuestionIDs {
		if ans, ok := fact.ClarificationAnswers[qID]; ok {
			answers = append(answers, ans)
		}
	}
	return answers
}
