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
	ID                 string   `json:"id"`
	UserID             string   `json:"userId"`
	ConversationID     string   `json:"conversationId"`
	ClarificationType  string   `json:"clarificationType"` // "subject_clarification", "contact_confirmation", etc.
	QuestionText       string   `json:"questionText"`
	ContextNotes       string   `json:"contextNotes"`
	Options            []string `json:"options"` // Multiple choice options (if any)
	Priority           int      `json:"priority"` // 1=critical, 2=important, 3=nice-to-have
	Status             string   `json:"status"`   // "pending", "answered", "skipped"
	LinkedFacts        []string `json:"linkedFacts"`
	CreatedAt          int64    `json:"createdAt"`
	AnsweredAt         int64    `json:"answeredAt,omitempty"`
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
	log.Printf("[V2] ClarificationQuestionRepository: saving question %s", question.ID)

	if question.UserID == "" || question.QuestionText == "" {
		return fmt.Errorf("userId and questionText required")
	}

	optionsJSON, _ := json.Marshal(question.Options)
	factsJSON, _ := json.Marshal(question.LinkedFacts)

	query := `
		INSERT INTO clarification_questions
		(id, user_id, conversation_id, clarification_type, question_text, context_notes, options, priority, status, linked_facts, created_at, answered_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.Exec(
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
		log.Printf("[V2] ClarificationQuestionRepository ERROR: %v", err)
		return err
	}

	log.Printf("[V2] ClarificationQuestionRepository: saved question %s", question.ID)
	return nil
}

// GetQuestion retrieves a clarification question by ID
func (r *ClarificationQuestionRepository) GetQuestion(questionID string) (*ClarificationQuestion, error) {
	query := `
		SELECT id, user_id, conversation_id, clarification_type, question_text, context_notes,
		       options, priority, status, linked_facts, created_at, answered_at
		FROM clarification_questions
		WHERE id = ?
	`

	question := &ClarificationQuestion{}
	var optionsJSON sql.NullString
	var factsJSON sql.NullString

	err := r.db.QueryRow(query, questionID).Scan(
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
		json.Unmarshal([]byte(optionsJSON.String), &question.Options)
	}
	if factsJSON.Valid {
		json.Unmarshal([]byte(factsJSON.String), &question.LinkedFacts)
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
			json.Unmarshal([]byte(optionsJSON.String), &question.Options)
		}
		if factsJSON.Valid {
			json.Unmarshal([]byte(factsJSON.String), &question.LinkedFacts)
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
			json.Unmarshal([]byte(optionsJSON.String), &question.Options)
		}
		if factsJSON.Valid {
			json.Unmarshal([]byte(factsJSON.String), &question.LinkedFacts)
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
	log.Printf("[V2] ClarificationResponseRepository: saving response to question %s", response.QuestionID)

	if response.QuestionID == "" || response.UserID == "" {
		return fmt.Errorf("questionId and userId required")
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
		log.Printf("[V2] ClarificationResponseRepository ERROR: %v", err)
		return err
	}

	log.Printf("[V2] ClarificationResponseRepository: saved response %s", response.ID)
	return nil
}

// GetResponse retrieves a clarification response by ID
func (r *ClarificationResponseRepository) GetResponse(responseID string) (*ClarificationResponse, error) {
	query := `
		SELECT id, question_id, user_id, response_text, selected_option, responded_at, created_at
		FROM clarification_responses
		WHERE id = ?
	`

	response := &ClarificationResponse{}
	err := r.db.QueryRow(query, responseID).Scan(
		&response.ID,
		&response.QuestionID,
		&response.UserID,
		&response.ResponseText,
		&response.SelectedOption,
		&response.RespondedAt,
		&response.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return response, nil
}

// GetQuestionResponse retrieves the response to a specific question (if any)
func (r *ClarificationResponseRepository) GetQuestionResponse(questionID string) (*ClarificationResponse, error) {
	query := `
		SELECT id, question_id, user_id, response_text, selected_option, responded_at, created_at
		FROM clarification_responses
		WHERE question_id = ?
		LIMIT 1
	`

	response := &ClarificationResponse{}
	err := r.db.QueryRow(query, questionID).Scan(
		&response.ID,
		&response.QuestionID,
		&response.UserID,
		&response.ResponseText,
		&response.SelectedOption,
		&response.RespondedAt,
		&response.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return response, nil
}

// GetUserResponses retrieves all responses from a user
func (r *ClarificationResponseRepository) GetUserResponses(userID string) ([]*ClarificationResponse, error) {
	query := `
		SELECT id, question_id, user_id, response_text, selected_option, responded_at, created_at
		FROM clarification_responses
		WHERE user_id = ?
		ORDER BY responded_at DESC
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var responses []*ClarificationResponse
	for rows.Next() {
		response := &ClarificationResponse{}
		err := rows.Scan(
			&response.ID,
			&response.QuestionID,
			&response.UserID,
			&response.ResponseText,
			&response.SelectedOption,
			&response.RespondedAt,
			&response.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		responses = append(responses, response)
	}

	return responses, rows.Err()
}
