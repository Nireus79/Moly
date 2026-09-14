package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// ConflictRecord represents a stored conflict in the database
type ConflictRecord struct {
	ID                  int64  `json:"id"`
	UserID              string `json:"userId"`
	ConversationID      string `json:"conversationId"`
	ConflictType        string `json:"conflictType"` // "attribute_contradiction", "name_change", "relationship_change"
	Severity            string `json:"severity"`    // "low", "medium", "high"
	SavedValue          string `json:"savedValue"`  // JSON
	ExtractedValue      string `json:"extractedValue"` // JSON
	Description         string `json:"description"`
	Status              string `json:"status"` // "unresolved", "resolved"
	Resolution          string `json:"resolution"` // "keep_saved", "use_extracted", "merge"
	ResolutionDetails   string `json:"resolutionDetails"` // JSON with additional details
	CreatedAt           int64  `json:"createdAt"`
	ResolvedAt          *int64 `json:"resolvedAt"`
}

// ConflictRepository handles conflict persistence
type ConflictRepository struct {
	db *Database
}

// NewConflictRepository creates a new repository
func NewConflictRepository(db *Database) *ConflictRepository {
	return &ConflictRepository{db: db}
}

// Save stores a conflict record
func (r *ConflictRepository) Save(conflict *ConflictRecord) error {
	log.Printf("[V2] ConflictRepository: saving conflict type=%s severity=%s", conflict.ConflictType, conflict.Severity)

	if conflict.UserID == "" || conflict.ConflictType == "" {
		return fmt.Errorf("userId and conflictType required")
	}

	if conflict.CreatedAt == 0 {
		conflict.CreatedAt = time.Now().Unix()
	}

	query := `
		INSERT INTO context_conflicts
		(user_id, conversation_id, conflict_type, severity, saved_value, extracted_value, description, status, resolution, resolution_details, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(
		query,
		conflict.UserID,
		conflict.ConversationID,
		conflict.ConflictType,
		conflict.Severity,
		conflict.SavedValue,
		conflict.ExtractedValue,
		conflict.Description,
		conflict.Status,
		conflict.Resolution,
		conflict.ResolutionDetails,
		conflict.CreatedAt,
	)

	if err != nil {
		log.Printf("[V2] ConflictRepository ERROR: %v", err)
		return err
	}

	id, _ := result.LastInsertId()
	conflict.ID = id

	log.Printf("[V2] ConflictRepository: saved conflict id=%d", conflict.ID)
	return nil
}

// GetByID retrieves a conflict by ID
func (r *ConflictRepository) GetByID(id int64) (*ConflictRecord, error) {
	query := `
		SELECT id, user_id, conversation_id, conflict_type, severity, saved_value, extracted_value, description, status, resolution, resolution_details, created_at, resolved_at
		FROM context_conflicts
		WHERE id = ?
	`

	row := r.db.QueryRow(query, id)
	conflict := &ConflictRecord{}

	err := row.Scan(
		&conflict.ID,
		&conflict.UserID,
		&conflict.ConversationID,
		&conflict.ConflictType,
		&conflict.Severity,
		&conflict.SavedValue,
		&conflict.ExtractedValue,
		&conflict.Description,
		&conflict.Status,
		&conflict.Resolution,
		&conflict.ResolutionDetails,
		&conflict.CreatedAt,
		&conflict.ResolvedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return conflict, nil
}

// GetUnresolved retrieves all unresolved conflicts for a user
func (r *ConflictRepository) GetUnresolved(userID string) ([]*ConflictRecord, error) {
	query := `
		SELECT id, user_id, conversation_id, conflict_type, severity, saved_value, extracted_value, description, status, resolution, resolution_details, created_at, resolved_at
		FROM context_conflicts
		WHERE user_id = ? AND status = 'unresolved'
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conflicts []*ConflictRecord
	for rows.Next() {
		conflict := &ConflictRecord{}
		err := rows.Scan(
			&conflict.ID,
			&conflict.UserID,
			&conflict.ConversationID,
			&conflict.ConflictType,
			&conflict.Severity,
			&conflict.SavedValue,
			&conflict.ExtractedValue,
			&conflict.Description,
			&conflict.Status,
			&conflict.Resolution,
			&conflict.ResolutionDetails,
			&conflict.CreatedAt,
			&conflict.ResolvedAt,
		)

		if err != nil {
			return nil, err
		}

		conflicts = append(conflicts, conflict)
	}

	return conflicts, rows.Err()
}

// GetByConversation retrieves conflicts from a specific conversation
func (r *ConflictRepository) GetByConversation(conversationID string) ([]*ConflictRecord, error) {
	query := `
		SELECT id, user_id, conversation_id, conflict_type, severity, saved_value, extracted_value, description, status, resolution, resolution_details, created_at, resolved_at
		FROM context_conflicts
		WHERE conversation_id = ?
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conflicts []*ConflictRecord
	for rows.Next() {
		conflict := &ConflictRecord{}
		err := rows.Scan(
			&conflict.ID,
			&conflict.UserID,
			&conflict.ConversationID,
			&conflict.ConflictType,
			&conflict.Severity,
			&conflict.SavedValue,
			&conflict.ExtractedValue,
			&conflict.Description,
			&conflict.Status,
			&conflict.Resolution,
			&conflict.ResolutionDetails,
			&conflict.CreatedAt,
			&conflict.ResolvedAt,
		)

		if err != nil {
			return nil, err
		}

		conflicts = append(conflicts, conflict)
	}

	return conflicts, rows.Err()
}

// MarkResolved marks a conflict as resolved
func (r *ConflictRepository) MarkResolved(conflictID int64, resolution string) error {
	log.Printf("[V2] ConflictRepository: marking conflict %d as resolved with resolution=%s", conflictID, resolution)

	now := time.Now().Unix()
	query := `
		UPDATE context_conflicts
		SET status = 'resolved', resolution = ?, resolved_at = ?
		WHERE id = ?
	`

	_, err := r.db.Exec(query, resolution, now, conflictID)
	if err != nil {
		log.Printf("[V2] ConflictRepository ERROR: %v", err)
		return err
	}

	return nil
}

// GetHighSeverityConflicts retrieves high-severity unresolved conflicts
func (r *ConflictRepository) GetHighSeverityConflicts(userID string) ([]*ConflictRecord, error) {
	query := `
		SELECT id, user_id, conversation_id, conflict_type, severity, saved_value, extracted_value, description, status, resolution, resolution_details, created_at, resolved_at
		FROM context_conflicts
		WHERE user_id = ? AND status = 'unresolved' AND severity IN ('high', 'critical')
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conflicts []*ConflictRecord
	for rows.Next() {
		conflict := &ConflictRecord{}
		err := rows.Scan(
			&conflict.ID,
			&conflict.UserID,
			&conflict.ConversationID,
			&conflict.ConflictType,
			&conflict.Severity,
			&conflict.SavedValue,
			&conflict.ExtractedValue,
			&conflict.Description,
			&conflict.Status,
			&conflict.Resolution,
			&conflict.ResolutionDetails,
			&conflict.CreatedAt,
			&conflict.ResolvedAt,
		)

		if err != nil {
			return nil, err
		}

		conflicts = append(conflicts, conflict)
	}

	return conflicts, rows.Err()
}

// StoreResolutionDetails stores additional JSON details about resolution
func (r *ConflictRepository) StoreResolutionDetails(conflictID int64, details map[string]interface{}) error {
	log.Printf("[V2] ConflictRepository: storing resolution details for conflict %d", conflictID)

	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return err
	}

	query := `
		UPDATE context_conflicts
		SET resolution_details = ?
		WHERE id = ?
	`

	_, err = r.db.Exec(query, string(detailsJSON), conflictID)
	return err
}
