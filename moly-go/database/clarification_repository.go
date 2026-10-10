package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// ClarificationQuestion represents a question asking user to clarify context
type ClarificationQuestion struct {
	ID                string   `json:"id"`
	UserID            string   `json:"userId"`
	ConversationID    string   `json:"conversationId"`
	ClarificationType string   `json:"clarificationType"` // "subject_clarification", "contact_confirmation", etc.
	QuestionText      string   `json:"questionText"`
	ContextNotes      string   `json:"contextNotes"`
	Options           []string `json:"options"`  // Multiple choice options (if any)
	Priority          int      `json:"priority"` // 1=critical, 2=important, 3=nice-to-have
	Status            string   `json:"status"`   // "active", "answered", "skipped", "cancelled"
	LinkedFacts       []string `json:"linkedFacts"`
	CreatedAt         int64    `json:"createdAt"`
	AnsweredAt        int64    `json:"answeredAt,omitempty"`
}

// ClarificationResponse represents user's answer to a clarification question
type ClarificationResponse struct {
	ID             string `json:"id"`
	QuestionID     string `json:"questionId"`
	UserID         string `json:"userId"`
	ResponseText   string `json:"responseText"`
	SelectedOption string `json:"selectedOption"`
	RespondedAt    int64  `json:"respondedAt"`
	CreatedAt      int64  `json:"createdAt"`
}

// ClarificationQuestionRepository handles clarification questions
type ClarificationQuestionRepository struct {
	db *Database
}

// ClarificationResponseRepository handles clarification responses
type ClarificationResponseRepository struct {
	db *Database
}

// NewClarificationQuestionRepository creates a new repository
func NewClarificationQuestionRepository(db *Database) *ClarificationQuestionRepository {
	return &ClarificationQuestionRepository{db: db}
}

// NewClarificationResponseRepository creates a new repository
func NewClarificationResponseRepository(db *Database) *ClarificationResponseRepository {
	return &ClarificationResponseRepository{db: db}
}

// SaveQuestion persists a clarification question
func (r *ClarificationQuestionRepository) SaveQuestion(question *ClarificationQuestion) error {
	log.Printf("[Database] ClarificationQuestionRepository: saving question %s", question.ID)

	// FIX #31: Comprehensive validation for clarification question
	if question.ID == "" || len(question.ID) > 255 {
		return fmt.Errorf("question.id required and must be <= 255 chars")
	}
	if question.UserID == "" || len(question.UserID) > 255 {
		return fmt.Errorf("userId required and must be <= 255 chars")
	}
	if question.QuestionText == "" || len(question.QuestionText) > 2000 {
		return fmt.Errorf("questionText required and must be <= 2000 chars")
	}

	// Optional fields validation
	if question.ClarificationType != "" {
		validTypes := map[string]bool{"gap": true, "goal": true, "contact": true, "context": true, "safety": true}
		if !validTypes[question.ClarificationType] {
			return fmt.Errorf("invalid clarificationType: %s", question.ClarificationType)
		}
	}

	if question.Status != "" {
		validStatuses := map[string]bool{"active": true, "answered": true, "skipped": true, "cancelled": true}
		if !validStatuses[question.Status] {
			return fmt.Errorf("invalid status: %s", question.Status)
		}
	}

	// Validate JSON fields can be marshaled
	optionsJSON, err := json.Marshal(question.Options)
	if err != nil {
		return fmt.Errorf("failed to marshal options: %w", err)
	}
	factsJSON, err := json.Marshal(question.LinkedFacts)
	if err != nil {
		return fmt.Errorf("failed to marshal linkedFacts: %w", err)
	}

	query := `
		INSERT INTO clarification_questions
		(id, user_id, conversation_id, clarification_type, question_text, context_notes, options, priority, status, linked_facts, created_at, answered_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = r.db.Exec(
		query,
		question.ID,
		question.UserID,
		question.ConversationID,
		question.ClarificationType,
		question.QuestionText,
		question.ContextNotes,
		string(optionsJSON),
		question.Priority,
		question.Status,
		string(factsJSON),
		question.CreatedAt,
		question.AnsweredAt,
	)

	if err != nil {
		log.Printf("[Database] ClarificationQuestionRepository ERROR: %v", err)
		return err
	}

	log.Printf("[Database] ClarificationQuestionRepository: saved question %s", question.ID)
	return nil
}

// GetQuestion retrieves a clarification question by ID
// FIX #6: Added userID for data isolation
func (r *ClarificationQuestionRepository) GetQuestion(userID, questionID string) (*ClarificationQuestion, error) {
	query := `
		SELECT id, user_id, conversation_id, clarification_type, question_text, context_notes,
		       options, priority, status, linked_facts, created_at, answered_at
		FROM clarification_questions
		WHERE id = ? AND user_id = ?
	`

	question := &ClarificationQuestion{}
	var optionsJSON sql.NullString
	var factsJSON sql.NullString

	err := r.db.QueryRow(query, questionID, userID).Scan(
		&question.ID,
		&question.UserID,
		&question.ConversationID,
		&question.ClarificationType,
		&question.QuestionText,
		&question.ContextNotes,
		&optionsJSON,
		&question.Priority,
		&question.Status,
		&factsJSON,
		&question.CreatedAt,
		&question.AnsweredAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if optionsJSON.Valid {
		if err := json.Unmarshal([]byte(optionsJSON.String), &question.Options); err != nil {
			log.Printf("[ClarificationQuestion] ERROR parsing options for question %s: %v", question.ID, err)
		}
	}
	if factsJSON.Valid {
		if err := json.Unmarshal([]byte(factsJSON.String), &question.LinkedFacts); err != nil {
			log.Printf("[ClarificationQuestion] ERROR parsing facts for question %s: %v", question.ID, err)
		}
	}

	return question, nil
}

// GetPendingQuestions retrieves unanswered questions for a user
func (r *ClarificationQuestionRepository) GetPendingQuestions(userID string) ([]*ClarificationQuestion, error) {
	query := `
		SELECT id, user_id, conversation_id, clarification_type, question_text, context_notes,
		       options, priority, status, linked_facts, created_at, answered_at
		FROM clarification_questions
		WHERE user_id = ? AND status = 'pending'
		ORDER BY priority DESC, created_at DESC
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var questions []*ClarificationQuestion
	for rows.Next() {
		question := &ClarificationQuestion{}
		var optionsJSON sql.NullString
		var factsJSON sql.NullString

		err := rows.Scan(
			&question.ID,
			&question.UserID,
			&question.ConversationID,
			&question.ClarificationType,
			&question.QuestionText,
			&question.ContextNotes,
			&optionsJSON,
			&question.Priority,
			&question.Status,
			&factsJSON,
			&question.CreatedAt,
			&question.AnsweredAt,
		)

		if err != nil {
			return nil, err
		}

		if optionsJSON.Valid {
			if err := json.Unmarshal([]byte(optionsJSON.String), &question.Options); err != nil {
				log.Printf("[ClarificationQuestion] ERROR parsing options for question %s: %v", question.ID, err)
			}
		}
		if factsJSON.Valid {
			if err := json.Unmarshal([]byte(factsJSON.String), &question.LinkedFacts); err != nil {
				log.Printf("[ClarificationQuestion] ERROR parsing facts for question %s: %v", question.ID, err)
			}
		}

		questions = append(questions, question)
	}

	return questions, rows.Err()
}

// GetConversationQuestions retrieves all questions for a conversation
func (r *ClarificationQuestionRepository) GetConversationQuestions(conversationID string) ([]*ClarificationQuestion, error) {
	query := `
		SELECT id, user_id, conversation_id, clarification_type, question_text, context_notes,
		       options, priority, status, linked_facts, created_at, answered_at
		FROM clarification_questions
		WHERE conversation_id = ?
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var questions []*ClarificationQuestion
	for rows.Next() {
		question := &ClarificationQuestion{}
		var optionsJSON sql.NullString
		var factsJSON sql.NullString

		err := rows.Scan(
			&question.ID,
			&question.UserID,
			&question.ConversationID,
			&question.ClarificationType,
			&question.QuestionText,
			&question.ContextNotes,
			&optionsJSON,
			&question.Priority,
			&question.Status,
			&factsJSON,
			&question.CreatedAt,
			&question.AnsweredAt,
		)

		if err != nil {
			return nil, err
		}

		if optionsJSON.Valid {
			if err := json.Unmarshal([]byte(optionsJSON.String), &question.Options); err != nil {
				log.Printf("[ClarificationQuestion] ERROR parsing options for question %s: %v", question.ID, err)
			}
		}
		if factsJSON.Valid {
			if err := json.Unmarshal([]byte(factsJSON.String), &question.LinkedFacts); err != nil {
				log.Printf("[ClarificationQuestion] ERROR parsing facts for question %s: %v", question.ID, err)
			}
		}

		questions = append(questions, question)
	}

	return questions, rows.Err()
}

// MarkAnswered marks a question as answered
func (r *ClarificationQuestionRepository) MarkAnswered(questionID string) error {
	query := `
		UPDATE clarification_questions
		SET status = 'answered', answered_at = ?
		WHERE id = ?
	`

	_, err := r.db.Exec(query, time.Now().Unix(), questionID)
	return err
}

// SaveResponse persists a clarification response
func (r *ClarificationResponseRepository) SaveResponse(response *ClarificationResponse) error {
	log.Printf("[Database] ClarificationResponseRepository: saving response to question %s", response.QuestionID)

	// FIX #32: Validate clarification response before save
	if response.ID == "" || len(response.ID) > 255 {
		return fmt.Errorf("id required and must be <= 255 chars")
	}
	if response.QuestionID == "" || len(response.QuestionID) > 255 {
		return fmt.Errorf("questionId required and must be <= 255 chars")
	}
	if response.UserID == "" || len(response.UserID) > 255 {
		return fmt.Errorf("userId required and must be <= 255 chars")
	}
	if response.RespondedAt == 0 {
		response.RespondedAt = time.Now().Unix()
	}

	query := `
		INSERT INTO clarification_responses
		(id, question_id, user_id, response_text, selected_option, responded_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.Exec(
		query,
		response.ID,
		response.QuestionID,
		response.UserID,
		response.ResponseText,
		response.SelectedOption,
		response.RespondedAt,
		response.CreatedAt,
	)

	if err != nil {
		log.Printf("[Database] ClarificationResponseRepository ERROR: %v", err)
		return err
	}

	log.Printf("[Database] ClarificationResponseRepository: saved response %s", response.ID)
	return nil
}
