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

	// FIX #32: Validate pending input before save
	if userID == "" || len(userID) > 255 {
		return 0, fmt.Errorf("userId required and must be <= 255 chars")
	}
	if conversationID == "" || len(conversationID) > 255 {
		return 0, fmt.Errorf("conversationId required and must be <= 255 chars")
	}
	validTypes := map[string]bool{"clarification": true, "conflict": true, "approval": true}
	if inputType == "" || !validTypes[inputType] {
		return 0, fmt.Errorf("inputType must be one of: clarification, conflict, approval")
	}
	if question == "" || len(question) > 2000 {
		return 0, fmt.Errorf("question required and must be <= 2000 chars")
	}

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
// FIX #6: Added userID for data isolation
func (r *PendingInputRepository) GetByID(userID string, id int64) (*PendingInput, error) {
	query := `
		SELECT id, user_id, conversation_id, type, subtype, question, context, created_at, resolved_at, resolution, applied, metadata
		FROM pending_input
		WHERE id = ? AND user_id = ?
	`

	var pi PendingInput
	var metadata sql.NullString

	err := r.db.QueryRow(query, id, userID).Scan(&pi.ID, &pi.UserID, &pi.ConversationID, &pi.Type, &pi.Subtype, &pi.Question, &pi.Context, &pi.CreatedAt, &pi.ResolvedAt, &pi.Resolution, &pi.Applied, &metadata)
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
// FIX #6: Added userID for data isolation
func (r *PendingInputRepository) Resolve(userID string, id int64, resolution string) error {
	log.Printf("[PendingInputRepository] Resolving pending input %d with resolution: %s", id, resolution)

	now := time.Now().Unix()
	query := `
		UPDATE pending_input
		SET resolved_at = ?, resolution = ?
		WHERE id = ? AND user_id = ?
	`

	result, err := r.db.Exec(query, now, resolution, id, userID)
	if err != nil {
		log.Printf("[PendingInputRepository] ERROR resolving: %v", err)
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("input not found or access denied")
	}

	log.Printf("[PendingInputRepository] Pending input resolved")
	return nil
}

// MarkApplied - Mark as applied to user model
// FIX #6: Added userID for data isolation
func (r *PendingInputRepository) MarkApplied(userID string, id int64) error {
	log.Printf("[PendingInputRepository] Marking pending input %d as applied", id)

	query := `
		UPDATE pending_input
		SET applied = 1
		WHERE id = ? AND user_id = ?
	`

	result, err := r.db.Exec(query, id, userID)
	if err != nil {
		log.Printf("[PendingInputRepository] ERROR marking applied: %v", err)
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("input not found or access denied")
	}

	log.Printf("[PendingInputRepository] Pending input marked as applied")
	return nil
}

// Delete - Delete pending input (for cleanup)
// FIX #6: Added userID for data isolation
func (r *PendingInputRepository) Delete(userID string, id int64) error {
	query := `DELETE FROM pending_input WHERE id = ? AND user_id = ?`
	result, err := r.db.Exec(query, id, userID)
	if err != nil {
		log.Printf("[PendingInputRepository] Error deleting: %v", err)
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("input not found or access denied")
	}
	return nil
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
