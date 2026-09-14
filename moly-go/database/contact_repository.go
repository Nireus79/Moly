package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// Contact represents a person in the user's communication network
type Contact struct {
	ID               int64    `json:"id"`
	UserID           string   `json:"userId"`
	Name             string   `json:"name"`
	Relationship     string   `json:"relationship"` // "manager", "colleague", "friend", "family", etc.
	Age              string   `json:"age,omitempty"`
	Traits           []string `json:"traits"` // Discovered characteristics
	FirstMentionedAt int64    `json:"firstMentionedAt,omitempty"`
	CreatedVia       string   `json:"createdVia"` // "conversation", "manual", "import"
	Status           string   `json:"status"`     // "active", "archived"
	Version          int64    `json:"version"`    // For optimistic locking
	CreatedAt        int64    `json:"createdAt"`
	UpdatedAt        int64    `json:"updatedAt"`
}

// ContactRepository handles contact database operations
type ContactRepository struct {
	db *Database
}

// NewContactRepository creates a new contact repository
func NewContactRepository(db *Database) *ContactRepository {
	return &ContactRepository{db: db}
}

// Save creates or updates a contact
func (r *ContactRepository) Save(contact *Contact) error {
	log.Printf("[V2] ContactRepository: saving contact %s for user %s", contact.Name, contact.UserID)

	if contact.UserID == "" || contact.Name == "" {
		return fmt.Errorf("userId and name required")
	}

	// Set timestamps
	if contact.CreatedAt == 0 {
		contact.CreatedAt = time.Now().Unix()
	}
	contact.UpdatedAt = time.Now().Unix()

	// Set version if new
	if contact.Version == 0 {
		contact.Version = 1
	}

	traitsJSON, _ := json.Marshal(contact.Traits)

	query := `
		INSERT OR REPLACE INTO contacts
		(user_id, name, relationship, age, characteristics, first_mentioned_at, created_via, status, version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(
		query,
		contact.UserID,
		contact.Name,
		contact.Relationship,
		contact.Age,
		string(traitsJSON),
		contact.FirstMentionedAt,
		contact.CreatedVia,
		contact.Status,
		contact.Version,
		contact.CreatedAt,
		contact.UpdatedAt,
	)

	if err != nil {
		log.Printf("[V2] ContactRepository ERROR: %v", err)
		return err
	}

	// Get the ID if this was an insert
	if contact.ID == 0 {
		id, _ := result.LastInsertId()
		contact.ID = id
	}

	log.Printf("[V2] ContactRepository: saved contact %s (id=%d)", contact.Name, contact.ID)
	return nil
}

// GetByID retrieves a contact by ID
func (r *ContactRepository) GetByID(contactID int64) (*Contact, error) {
	query := `
		SELECT id, user_id, name, relationship, age, characteristics, first_mentioned_at, created_via, status, version, created_at, updated_at
		FROM contacts
		WHERE id = ? AND status = 'active'
	`

	contact := &Contact{}
	var traitsJSON sql.NullString

	err := r.db.QueryRow(query, contactID).Scan(
		&contact.ID,
		&contact.UserID,
		&contact.Name,
		&contact.Relationship,
		&contact.Age,
		&traitsJSON,
		&contact.FirstMentionedAt,
		&contact.CreatedVia,
		&contact.Status,
		&contact.Version,
		&contact.CreatedAt,
		&contact.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if traitsJSON.Valid {
		json.Unmarshal([]byte(traitsJSON.String), &contact.Traits)
	}

	return contact, nil
}

// GetByName retrieves a contact by user and name
func (r *ContactRepository) GetByName(userID, name string) (*Contact, error) {
	query := `
		SELECT id, user_id, name, relationship, age, characteristics, first_mentioned_at, created_via, status, version, created_at, updated_at
		FROM contacts
		WHERE user_id = ? AND name = ? AND status = 'active'
	`

	contact := &Contact{}
	var traitsJSON sql.NullString

	err := r.db.QueryRow(query, userID, name).Scan(
		&contact.ID,
		&contact.UserID,
		&contact.Name,
		&contact.Relationship,
		&contact.Age,
		&traitsJSON,
		&contact.FirstMentionedAt,
		&contact.CreatedVia,
		&contact.Status,
		&contact.Version,
		&contact.CreatedAt,
		&contact.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if traitsJSON.Valid {
		json.Unmarshal([]byte(traitsJSON.String), &contact.Traits)
	}

	return contact, nil
}

// GetByUserID retrieves all active contacts for a user
func (r *ContactRepository) GetByUserID(userID string) ([]*Contact, error) {
	query := `
		SELECT id, user_id, name, relationship, age, characteristics, first_mentioned_at, created_via, status, version, created_at, updated_at
		FROM contacts
		WHERE user_id = ? AND status = 'active'
		ORDER BY updated_at DESC
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []*Contact
	for rows.Next() {
		contact := &Contact{}
		var traitsJSON sql.NullString

		err := rows.Scan(
			&contact.ID,
			&contact.UserID,
			&contact.Name,
			&contact.Relationship,
			&contact.Age,
			&traitsJSON,
			&contact.FirstMentionedAt,
			&contact.CreatedVia,
			&contact.Status,
			&contact.Version,
			&contact.CreatedAt,
			&contact.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		if traitsJSON.Valid {
			json.Unmarshal([]byte(traitsJSON.String), &contact.Traits)
		}

		contacts = append(contacts, contact)
	}

	return contacts, rows.Err()
}

// GetByRelationship retrieves contacts by relationship type
func (r *ContactRepository) GetByRelationship(userID, relationship string) ([]*Contact, error) {
	query := `
		SELECT id, user_id, name, relationship, age, characteristics, first_mentioned_at, created_via, status, version, created_at, updated_at
		FROM contacts
		WHERE user_id = ? AND relationship = ? AND status = 'active'
		ORDER BY name ASC
	`

	rows, err := r.db.Query(query, userID, relationship)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []*Contact
	for rows.Next() {
		contact := &Contact{}
		var traitsJSON sql.NullString

		err := rows.Scan(
			&contact.ID,
			&contact.UserID,
			&contact.Name,
			&contact.Relationship,
			&contact.Age,
			&traitsJSON,
			&contact.FirstMentionedAt,
			&contact.CreatedVia,
			&contact.Status,
			&contact.Version,
			&contact.CreatedAt,
			&contact.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		if traitsJSON.Valid {
			json.Unmarshal([]byte(traitsJSON.String), &contact.Traits)
		}

		contacts = append(contacts, contact)
	}

	return contacts, rows.Err()
}

// Update updates a contact with optimistic locking
// Returns error if version doesn't match
func (r *ContactRepository) Update(contact *Contact) error {
	if contact.ID == 0 {
		return fmt.Errorf("contact ID required for update")
	}

	oldVersion := contact.Version
	contact.Version++ // Increment version
	contact.UpdatedAt = time.Now().Unix()

	traitsJSON, _ := json.Marshal(contact.Traits)

	query := `
		UPDATE contacts
		SET name = ?, relationship = ?, age = ?, characteristics = ?,
		    first_mentioned_at = ?, created_via = ?, status = ?, version = ?, updated_at = ?
		WHERE id = ? AND version = ?
	`

	result, err := r.db.Exec(
		query,
		contact.Name,
		contact.Relationship,
		contact.Age,
		string(traitsJSON),
		contact.FirstMentionedAt,
		contact.CreatedVia,
		contact.Status,
		contact.Version,
		contact.UpdatedAt,
		contact.ID,
		oldVersion,
	)

	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		// Version mismatch - contact was updated elsewhere
		log.Printf("[V2] ContactRepository: version mismatch for contact %d", contact.ID)
		return fmt.Errorf("contact version mismatch - contact was updated elsewhere")
	}

	return nil
}

// Delete archives a contact (soft delete)
func (r *ContactRepository) Delete(contactID int64) error {
	query := `
		UPDATE contacts
		SET status = 'archived', updated_at = ?
		WHERE id = ?
	`

	_, err := r.db.Exec(query, time.Now().Unix(), contactID)
	return err
}

// AddTrait adds a trait to a contact
func (r *ContactRepository) AddTrait(contactID int64, trait string) error {
	// Get current traits
	var traitsJSON sql.NullString
	err := r.db.QueryRow(
		`SELECT characteristics FROM contacts WHERE id = ?`,
		contactID,
	).Scan(&traitsJSON)

	if err != nil {
		return err
	}

	var traits []string
	if traitsJSON.Valid {
		json.Unmarshal([]byte(traitsJSON.String), &traits)
	}

	// Check if trait already exists
	for _, t := range traits {
		if t == trait {
			return nil // Already have this trait
		}
	}

	// Add new trait
	traits = append(traits, trait)
	traitsBytes, _ := json.Marshal(traits)
	traitsJSON.String = string(traitsBytes)

	// Update contact
	_, err = r.db.Exec(
		`UPDATE contacts SET characteristics = ?, updated_at = ? WHERE id = ?`,
		traitsJSON.String,
		time.Now().Unix(),
		contactID,
	)

	return err
}

// GetAll retrieves all active contacts for a user (backward compatibility)
func (r *ContactRepository) GetAll(userID string) ([]*Contact, error) {
	query := `
		SELECT id, user_id, name, relationship, age, characteristics, first_mentioned_at, created_via, status, version, created_at, updated_at
		FROM contacts
		WHERE user_id = ? AND status = 'active'
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []*Contact
	for rows.Next() {
		contact := &Contact{}
		var charJSON sql.NullString

		err := rows.Scan(
			&contact.ID,
			&contact.UserID,
			&contact.Name,
			&contact.Relationship,
			&contact.Age,
			&charJSON,
			&contact.FirstMentionedAt,
			&contact.CreatedVia,
			&contact.Status,
			&contact.Version,
			&contact.CreatedAt,
			&contact.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		// Parse traits from JSON
		if charJSON.Valid {
			_ = json.Unmarshal([]byte(charJSON.String), &contact.Traits)
		}

		contacts = append(contacts, contact)
	}

	return contacts, rows.Err()
}
