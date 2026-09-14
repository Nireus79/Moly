package database

import (
	"fmt"
	"log"
	"time"
)

// ContextAttribute represents an extracted fact with WHO it's about
type ContextAttribute struct {
	ID             int64   `json:"id"`
	UserID         string  `json:"userId"`
	ConversationID string  `json:"conversationId"`
	FactType       string  `json:"factType"`      // "style", "trait", "value", "preference", "goal"
	FactValue      string  `json:"factValue"`     // "casual", "detail-oriented"
	AttributedTo   string  `json:"attributedTo"`  // "user", "contact_manager_sarah", "group_team"
	Context        string  `json:"context"`       // "work", "social", "family", "general"
	Confidence     float64 `json:"confidence"`    // 0-1
	Source         string  `json:"source"`        // "explicit", "inferred", "stated_directly"
	Evidence       string  `json:"evidence"`      // Quote from original message
	Version        int64   `json:"version"`
	CreatedAt      int64   `json:"createdAt"`
}

// ContextAttributeRepository handles context attribute storage
type ContextAttributeRepository struct {
	db *Database
}

// NewContextAttributeRepository creates a new repository
func NewContextAttributeRepository(db *Database) *ContextAttributeRepository {
	return &ContextAttributeRepository{db: db}
}

// Save stores a context attribute
func (r *ContextAttributeRepository) Save(attr *ContextAttribute) error {
	log.Printf("[V2] ContextAttributeRepository: SAVE START - %s=%s for %s (conf=%.2f)", attr.FactType, attr.FactValue, attr.AttributedTo, attr.Confidence)

	if attr.UserID == "" || attr.FactValue == "" || attr.AttributedTo == "" {
		log.Printf("[V2] ContextAttributeRepository: VALIDATION FAILED - missing required fields")
		return fmt.Errorf("userId, factValue, and attributedTo required")
	}

	if attr.CreatedAt == 0 {
		attr.CreatedAt = time.Now().Unix()
	}
	if attr.Version == 0 {
		attr.Version = 1
	}

	query := `
		INSERT INTO context_attributes
		(user_id, conversation_id, fact_type, fact_value, attributed_to, context, confidence, source, evidence, version, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(
		query,
		attr.UserID,
		attr.ConversationID,
		attr.FactType,
		attr.FactValue,
		attr.AttributedTo,
		attr.Context,
		attr.Confidence,
		attr.Source,
		attr.Evidence,
		attr.Version,
		attr.CreatedAt,
	)

	if err != nil {
		log.Printf("[V2] ContextAttributeRepository: SAVE FAILED - %v (type=%s value=%s subject=%s)", err, attr.FactType, attr.FactValue, attr.AttributedTo)
		return err
	}

	if attr.ID == 0 {
		id, _ := result.LastInsertId()
		attr.ID = id
	}

	log.Printf("[V2] ContextAttributeRepository: ✓ SAVED id=%d user=%s type=%s value=%s subject=%s context=%s", attr.ID, attr.UserID, attr.FactType, attr.FactValue, attr.AttributedTo, attr.Context)
	return nil
}

// GetForSubject retrieves all attributes for a specific subject
func (r *ContextAttributeRepository) GetForSubject(userID string, subject string) ([]*ContextAttribute, error) {
	log.Printf("[V2] ContextAttributeRepository: QUERY user=%s subject=%s", userID, subject)

	query := `
		SELECT id, user_id, conversation_id, fact_type, fact_value, attributed_to, context, confidence, source, evidence, version, created_at
		FROM context_attributes
		WHERE user_id = ? AND attributed_to = ?
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, userID, subject)
	if err != nil {
		log.Printf("[V2] ContextAttributeRepository: QUERY FAILED - %v", err)
		return nil, err
	}
	defer rows.Close()

	var attributes []*ContextAttribute
	for rows.Next() {
		attr := &ContextAttribute{}
		err := rows.Scan(
			&attr.ID,
			&attr.UserID,
			&attr.ConversationID,
			&attr.FactType,
			&attr.FactValue,
			&attr.AttributedTo,
			&attr.Context,
			&attr.Confidence,
			&attr.Source,
			&attr.Evidence,
			&attr.Version,
			&attr.CreatedAt,
		)

		if err != nil {
			log.Printf("[V2] ContextAttributeRepository: SCAN FAILED - %v", err)
			return nil, err
		}

		attributes = append(attributes, attr)
	}

	log.Printf("[V2] ContextAttributeRepository: ✓ FOUND %d attributes for %s", len(attributes), subject)
	return attributes, rows.Err()
}

// GetForConversation retrieves all attributes from a conversation
func (r *ContextAttributeRepository) GetForConversation(conversationID string) ([]*ContextAttribute, error) {
	query := `
		SELECT id, user_id, conversation_id, fact_type, fact_value, attributed_to, context, confidence, source, evidence, version, created_at
		FROM context_attributes
		WHERE conversation_id = ?
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attributes []*ContextAttribute
	for rows.Next() {
		attr := &ContextAttribute{}
		err := rows.Scan(
			&attr.ID,
			&attr.UserID,
			&attr.ConversationID,
			&attr.FactType,
			&attr.FactValue,
			&attr.AttributedTo,
			&attr.Context,
			&attr.Confidence,
			&attr.Source,
			&attr.Evidence,
			&attr.Version,
			&attr.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		attributes = append(attributes, attr)
	}

	return attributes, rows.Err()
}

// GetByType retrieves attributes of a specific type for a subject
func (r *ContextAttributeRepository) GetByType(userID string, subject string, factType string) ([]*ContextAttribute, error) {
	query := `
		SELECT id, user_id, conversation_id, fact_type, fact_value, attributed_to, context, confidence, source, evidence, version, created_at
		FROM context_attributes
		WHERE user_id = ? AND attributed_to = ? AND fact_type = ?
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, userID, subject, factType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attributes []*ContextAttribute
	for rows.Next() {
		attr := &ContextAttribute{}
		err := rows.Scan(
			&attr.ID,
			&attr.UserID,
			&attr.ConversationID,
			&attr.FactType,
			&attr.FactValue,
			&attr.AttributedTo,
			&attr.Context,
			&attr.Confidence,
			&attr.Source,
			&attr.Evidence,
			&attr.Version,
			&attr.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		attributes = append(attributes, attr)
	}

	return attributes, rows.Err()
}

// GetUserAttributes retrieves all attributes for a user
func (r *ContextAttributeRepository) GetUserAttributes(userID string) ([]*ContextAttribute, error) {
	query := `
		SELECT id, user_id, conversation_id, fact_type, fact_value, attributed_to, context, confidence, source, evidence, version, created_at
		FROM context_attributes
		WHERE user_id = ?
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attributes []*ContextAttribute
	for rows.Next() {
		attr := &ContextAttribute{}
		err := rows.Scan(
			&attr.ID,
			&attr.UserID,
			&attr.ConversationID,
			&attr.FactType,
			&attr.FactValue,
			&attr.AttributedTo,
			&attr.Context,
			&attr.Confidence,
			&attr.Source,
			&attr.Evidence,
			&attr.Version,
			&attr.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		attributes = append(attributes, attr)
	}

	return attributes, rows.Err()
}

// Delete removes an attribute (soft delete via archive pattern)
func (r *ContextAttributeRepository) Delete(attributeID int64) error {
	// For now, we'll do a hard delete since attributes are immutable snapshots
	query := `DELETE FROM context_attributes WHERE id = ?`
	_, err := r.db.Exec(query, attributeID)
	return err
}
