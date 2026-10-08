package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// ContextConflict represents a conflict when new extracted context differs from saved
type ContextConflict struct {
	ID                int64                  `json:"id"`
	UserID            string                 `json:"userId"`
	ConversationID    string                 `json:"conversationId"`
	ConflictType      string                 `json:"type"`     // "aboutme_communication_style", "contact_name", etc.
	Severity          string                 `json:"severity"` // "low", "medium", "high"
	SavedValue        interface{}            `json:"savedValue"`
	ExtractedValue    interface{}            `json:"extractedValue"`
	Description       string                 `json:"description"`
	Status            string                 `json:"status"`     // "unresolved", "resolved"
	Resolution        string                 `json:"resolution"` // "keep_saved", "use_extracted", "merge"
	ResolutionDetails map[string]interface{} `json:"resolutionDetails"`
	CreatedAt         int64                  `json:"createdAt"`
	ResolvedAt        int64                  `json:"resolvedAt"`
}

// ContextConflictRepository handles conflict storage
type ContextConflictRepository struct {
	db *Database
}

// NewContextConflictRepository creates a new repository
func NewContextConflictRepository(db *Database) *ContextConflictRepository {
	return &ContextConflictRepository{db: db}
}

// DetectConflict compares new and saved values, returns conflict if different
func (r *ContextConflictRepository) DetectConflict(
	userID, conversationID string,
	conflictType string,
	savedValue, extractedValue interface{},
) *ContextConflict {
	// Quick comparison - if same, no conflict
	savedStr, _ := json.Marshal(savedValue)
	extractedStr, _ := json.Marshal(extractedValue)

	if string(savedStr) == string(extractedStr) {
		return nil // No conflict
	}

	// Different values = conflict
	severity := "low"
	description := fmt.Sprintf("Context changed: %s previously was %v, now seems to be %v",
		conflictType, savedValue, extractedValue)

	// Higher severity for core attributes
	if conflictType == "aboutme_communication_style" {
		severity = "medium"
		description = fmt.Sprintf(
			"Communication style conflict: You said you're '%v', but this message suggests '%v'",
			savedValue, extractedValue)
	} else if conflictType == "contact_name" {
		severity = "high"
		description = fmt.Sprintf(
			"Contact conflict: Previously '%v', but this message mentions '%v'",
			savedValue, extractedValue)
	}

	now := time.Now().Unix()
	return &ContextConflict{
		UserID:         userID,
		ConversationID: conversationID,
		ConflictType:   conflictType,
		Severity:       severity,
		SavedValue:     savedValue,
		ExtractedValue: extractedValue,
		Description:    description,
		Status:         "unresolved",
		CreatedAt:      now,
	}
}

// Save stores a conflict in the database
func (r *ContextConflictRepository) Save(conflict *ContextConflict) error {
	// FIX #32: Validate context conflict before save
	if conflict.UserID == "" || len(conflict.UserID) > 255 {
		return fmt.Errorf("userId required and must be <= 255 chars")
	}
	if conflict.ConflictType == "" {
		return fmt.Errorf("conflictType required")
	}
	validSeverities := map[string]bool{"low": true, "medium": true, "high": true, "critical": true}
	if conflict.Severity == "" || !validSeverities[conflict.Severity] {
		return fmt.Errorf("severity must be one of: low, medium, high, critical")
	}
	validStatuses := map[string]bool{"unresolved": true, "resolved": true, "ignored": true}
	if conflict.Status != "" && !validStatuses[conflict.Status] {
		return fmt.Errorf("status must be one of: unresolved, resolved, ignored")
	}

	var marshalErr error
	savedJSON, marshalErr := json.Marshal(conflict.SavedValue)
	if marshalErr != nil {
		return fmt.Errorf("failed to marshal savedValue: %w", marshalErr)
	}
	extractedJSON, marshalErr := json.Marshal(conflict.ExtractedValue)
	if marshalErr != nil {
		return fmt.Errorf("failed to marshal extractedValue: %w", marshalErr)
	}
	detailsJSON, marshalErr := json.Marshal(conflict.ResolutionDetails)
	if marshalErr != nil {
		return fmt.Errorf("failed to marshal resolutionDetails: %w", marshalErr)
	}

	query := `
		INSERT INTO context_conflicts (user_id, conversation_id, conflict_type, severity, saved_value, extracted_value, description, status, resolution, resolution_details, created_at, resolved_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(
		query,
		conflict.UserID,
		conflict.ConversationID,
		conflict.ConflictType,
		conflict.Severity,
		string(savedJSON),
		string(extractedJSON),
		conflict.Description,
		conflict.Status,
		conflict.Resolution,
		string(detailsJSON),
		conflict.CreatedAt,
		conflict.ResolvedAt,
	)

	if err == nil && conflict.ID == 0 {
		id, _ := result.LastInsertId()
		conflict.ID = id
	}

	return err
}

// GetUnresolved retrieves unresolved conflicts for a user
func (r *ContextConflictRepository) GetUnresolved(userID string) ([]*ContextConflict, error) {
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

	var conflicts []*ContextConflict
	for rows.Next() {
		var c ContextConflict
		var savedJSON, extractedJSON, detailsJSON sql.NullString

		err := rows.Scan(
			&c.ID,
			&c.UserID,
			&c.ConversationID,
			&c.ConflictType,
			&c.Severity,
			&savedJSON,
			&extractedJSON,
			&c.Description,
			&c.Status,
			&c.Resolution,
			&detailsJSON,
			&c.CreatedAt,
			&c.ResolvedAt,
		)

		if err != nil {
			return nil, err
		}

		if savedJSON.Valid {
			json.Unmarshal([]byte(savedJSON.String), &c.SavedValue)
		}
		if extractedJSON.Valid {
			json.Unmarshal([]byte(extractedJSON.String), &c.ExtractedValue)
		}
		if detailsJSON.Valid {
			json.Unmarshal([]byte(detailsJSON.String), &c.ResolutionDetails)
		}

		conflicts = append(conflicts, &c)
	}

	return conflicts, rows.Err()
}

// GetResolved retrieves resolved conflicts for a user (for learning system)
func (r *ContextConflictRepository) GetResolved(userID string) ([]*ContextConflict, error) {
	query := `
		SELECT id, user_id, conversation_id, conflict_type, severity, saved_value, extracted_value, description, status, resolution, resolution_details, created_at, resolved_at
		FROM context_conflicts
		WHERE user_id = ? AND status = 'resolved'
		ORDER BY resolved_at DESC
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conflicts []*ContextConflict
	for rows.Next() {
		var c ContextConflict
		var savedJSON, extractedJSON, detailsJSON sql.NullString

		err := rows.Scan(
			&c.ID,
			&c.UserID,
			&c.ConversationID,
			&c.ConflictType,
			&c.Severity,
			&savedJSON,
			&extractedJSON,
			&c.Description,
			&c.Status,
			&c.Resolution,
			&detailsJSON,
			&c.CreatedAt,
			&c.ResolvedAt,
		)

		if err != nil {
			return nil, err
		}

		if savedJSON.Valid {
			json.Unmarshal([]byte(savedJSON.String), &c.SavedValue)
		}
		if extractedJSON.Valid {
			json.Unmarshal([]byte(extractedJSON.String), &c.ExtractedValue)
		}
		if detailsJSON.Valid {
			json.Unmarshal([]byte(detailsJSON.String), &c.ResolutionDetails)
		}

		conflicts = append(conflicts, &c)
	}

	return conflicts, rows.Err()
}

// Resolve marks a conflict as resolved with user's choice
func (r *ContextConflictRepository) Resolve(conflictID int64, resolution string, resolutionDetails map[string]interface{}) error {
	detailsJSON, _ := json.Marshal(resolutionDetails)
	now := time.Now().Unix()

	query := `
		UPDATE context_conflicts
		SET status = 'resolved', resolution = ?, resolution_details = ?, resolved_at = ?
		WHERE id = ?
	`

	_, err := r.db.Exec(query, resolution, string(detailsJSON), now, conflictID)
	return err
}

// GetForConversation retrieves all conflicts for a conversation
func (r *ContextConflictRepository) GetForConversation(conversationID string) ([]*ContextConflict, error) {
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

	var conflicts []*ContextConflict
	for rows.Next() {
		var c ContextConflict
		var savedJSON, extractedJSON, detailsJSON sql.NullString

		err := rows.Scan(
			&c.ID,
			&c.UserID,
			&c.ConversationID,
			&c.ConflictType,
			&c.Severity,
			&savedJSON,
			&extractedJSON,
			&c.Description,
			&c.Status,
			&c.Resolution,
			&detailsJSON,
			&c.CreatedAt,
			&c.ResolvedAt,
		)

		if err != nil {
			return nil, err
		}

		if savedJSON.Valid {
			json.Unmarshal([]byte(savedJSON.String), &c.SavedValue)
		}
		if extractedJSON.Valid {
			json.Unmarshal([]byte(extractedJSON.String), &c.ExtractedValue)
		}
		if detailsJSON.Valid {
			json.Unmarshal([]byte(detailsJSON.String), &c.ResolutionDetails)
		}

		conflicts = append(conflicts, &c)
	}

	return conflicts, rows.Err()
}
