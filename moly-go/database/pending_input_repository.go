package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"moly/models"
)

// PendingInputRepository - Unified repository for clarifications, conflicts, and approvals
type PendingInputRepository struct {
	db *Database
}

// PendingInput - Unified pending input item
type PendingInput struct {
	ID             int64
	UserID         string
	ConversationID string
	Type           string // "clarification" | "conflict" | "approval"
	Subtype        string
	Question       string
	Context        json.RawMessage
	CreatedAt      int64
	ResolvedAt     *int64
	Resolution     *string
	Applied        bool
	Metadata       json.RawMessage
}

// NewPendingInputRepository - Create new pending input repository
func NewPendingInputRepository(db *Database) *PendingInputRepository {
	return &PendingInputRepository{db: db}
}

// Create - Insert new pending input item
func (r *PendingInputRepository) Create(userID, conversationID, inputType, subtype, question string, context interface{}) (int64, error) {
	log.Printf("[PendingInputRepository] Creating pending input: user=%s type=%s subtype=%s", userID, inputType, subtype)

	contextJSON, err := json.Marshal(context)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal context: %w", err)
	}

	now := time.Now().Unix()
	query := `
		INSERT INTO pending_input (user_id, conversation_id, type, subtype, question, context, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(query, userID, conversationID, inputType, subtype, question, string(contextJSON), now)
	if err != nil {
		log.Printf("[PendingInputRepository] ERROR creating pending input: %v", err)
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}

	log.Printf("[PendingInputRepository] Pending input created: id=%d", id)
	return id, nil
}

// GetUnresolved - Get all unresolved pending inputs for user
func (r *PendingInputRepository) GetUnresolved(userID string) ([]PendingInput, error) {
	query := `
		SELECT id, user_id, conversation_id, type, subtype, question, context, created_at, resolved_at, resolution, applied, metadata
		FROM pending_input
		WHERE user_id = ? AND resolved_at IS NULL
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query unresolved pending inputs: %w", err)
	}
	defer rows.Close()

	var results []PendingInput
	for rows.Next() {
		var pi PendingInput
		var metadata sql.NullString

		err := rows.Scan(&pi.ID, &pi.UserID, &pi.ConversationID, &pi.Type, &pi.Subtype, &pi.Question, &pi.Context, &pi.CreatedAt, &pi.ResolvedAt, &pi.Resolution, &pi.Applied, &metadata)
		if err != nil {
			log.Printf("[PendingInputRepository] ERROR scanning row: %v", err)
			continue
		}

		if metadata.Valid {
			pi.Metadata = json.RawMessage(metadata.String)
		}

		results = append(results, pi)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating results: %w", err)
	}

	log.Printf("[PendingInputRepository] Found %d unresolved items for user %s", len(results), userID)
	return results, nil
}

// GetByID - Get specific pending input by ID
func (r *PendingInputRepository) GetByID(id int64) (*PendingInput, error) {
	query := `
		SELECT id, user_id, conversation_id, type, subtype, question, context, created_at, resolved_at, resolution, applied, metadata
		FROM pending_input
		WHERE id = ?
	`

	var pi PendingInput
	var metadata sql.NullString

	err := r.db.QueryRow(query, id).Scan(&pi.ID, &pi.UserID, &pi.ConversationID, &pi.Type, &pi.Subtype, &pi.Question, &pi.Context, &pi.CreatedAt, &pi.ResolvedAt, &pi.Resolution, &pi.Applied, &metadata)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query pending input: %w", err)
	}

	if metadata.Valid {
		pi.Metadata = json.RawMessage(metadata.String)
	}

	return &pi, nil
}

// GetByConversation - Get all pending inputs for conversation (all types, all users in that conversation)
func (r *PendingInputRepository) GetByConversation(conversationID string) ([]PendingInput, error) {
	query := `
		SELECT id, user_id, conversation_id, type, subtype, question, context, created_at, resolved_at, resolution, applied, metadata
		FROM pending_input
		WHERE conversation_id = ? AND resolved_at IS NULL
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(query, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending inputs: %w", err)
	}
	defer rows.Close()

	var results []PendingInput
	for rows.Next() {
		var pi PendingInput
		var metadata sql.NullString

		err := rows.Scan(&pi.ID, &pi.UserID, &pi.ConversationID, &pi.Type, &pi.Subtype, &pi.Question, &pi.Context, &pi.CreatedAt, &pi.ResolvedAt, &pi.Resolution, &pi.Applied, &metadata)
		if err != nil {
			log.Printf("[PendingInputRepository] ERROR scanning row: %v", err)
			continue
		}

		if metadata.Valid {
			pi.Metadata = json.RawMessage(metadata.String)
		}

		results = append(results, pi)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating results: %w", err)
	}

	return results, nil
}

// GetByType - Get pending inputs of specific type for user
func (r *PendingInputRepository) GetByType(userID, inputType string) ([]PendingInput, error) {
	query := `
		SELECT id, user_id, conversation_id, type, subtype, question, context, created_at, resolved_at, resolution, applied, metadata
		FROM pending_input
		WHERE user_id = ? AND type = ? AND resolved_at IS NULL
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(query, userID, inputType)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending inputs by type: %w", err)
	}
	defer rows.Close()

	var results []PendingInput
	for rows.Next() {
		var pi PendingInput
		var metadata sql.NullString

		err := rows.Scan(&pi.ID, &pi.UserID, &pi.ConversationID, &pi.Type, &pi.Subtype, &pi.Question, &pi.Context, &pi.CreatedAt, &pi.ResolvedAt, &pi.Resolution, &pi.Applied, &metadata)
		if err != nil {
			log.Printf("[PendingInputRepository] ERROR scanning row: %v", err)
			continue
		}

		if metadata.Valid {
			pi.Metadata = json.RawMessage(metadata.String)
		}

		results = append(results, pi)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating results: %w", err)
	}

	return results, nil
}

// Resolve - Mark as resolved with user's answer
func (r *PendingInputRepository) Resolve(id int64, resolution string) error {
	log.Printf("[PendingInputRepository] Resolving pending input %d with resolution: %s", id, resolution)

	now := time.Now().Unix()
	query := `
		UPDATE pending_input
		SET resolved_at = ?, resolution = ?
		WHERE id = ?
	`

	_, err := r.db.Exec(query, now, resolution, id)
	if err != nil {
		log.Printf("[PendingInputRepository] ERROR resolving: %v", err)
		return err
	}

	log.Printf("[PendingInputRepository] Pending input resolved")
	return nil
}

// MarkApplied - Mark as applied to user model
func (r *PendingInputRepository) MarkApplied(id int64) error {
	log.Printf("[PendingInputRepository] Marking pending input %d as applied", id)

	query := `
		UPDATE pending_input
		SET applied = 1
		WHERE id = ?
	`

	_, err := r.db.Exec(query, id)
	if err != nil {
		log.Printf("[PendingInputRepository] ERROR marking applied: %v", err)
		return err
	}

	log.Printf("[PendingInputRepository] Pending input marked as applied")
	return nil
}

// Delete - Delete pending input (for cleanup)
func (r *PendingInputRepository) Delete(id int64) error {
	query := `DELETE FROM pending_input WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

// ConvertToModelsConflictInfo - Convert PendingInput to models.ConflictInfo for response building
func (pi *PendingInput) ToConflictInfo() (*models.ConflictInfo, error) {
	var ctx map[string]interface{}
	if err := json.Unmarshal(pi.Context, &ctx); err != nil {
		return nil, fmt.Errorf("failed to unmarshal context: %w", err)
	}

	conflict := &models.ConflictInfo{
		ConflictType:   pi.Subtype,
		SavedValue:     fmt.Sprintf("%v", ctx["old_value"]),
		ExtractedValue: fmt.Sprintf("%v", ctx["new_value"]),
		Context:        pi.Question,
	}

	return conflict, nil
}
