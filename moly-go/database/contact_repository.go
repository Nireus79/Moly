package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"moly/models"
)

// ContactRepository handles contact database operations
type ContactRepository struct {
	db *Database
}

// NewContactRepository creates a new contact repository
func NewContactRepository(db *Database) *ContactRepository {
	return &ContactRepository{db: db}
}

// Save creates or updates a contact
func (r *ContactRepository) Save(contact *models.Contact) error {
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

	traitsJSON, _ := json.Marshal(contact.Characteristics)
	pronounsJSON, _ := json.Marshal(contact.Pronouns)
	intentionsJSON, _ := json.Marshal(contact.InvolvedInIntentions)
	successesJSON, _ := json.Marshal(contact.PastSuccesses)
	dependenciesJSON, _ := json.Marshal(contact.Dependencies)

	query := `
		INSERT OR REPLACE INTO contacts
		(user_id, name, pronouns, relationship, age, characteristics, first_mentioned_at, created_via, status, version, created_at, updated_at, confidence, last_mentioned_at, contact_role, involved_intentions, past_successes, dependencies)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	// Default confidence if not set
	confidence := 0.5
	if contact.Confidence > 0 {
		confidence = contact.Confidence
	}

	lastMentioned := time.Now().Unix()
	if contact.LastMentionedAt > 0 {
		lastMentioned = contact.LastMentionedAt
	}

	result, err := r.db.Exec(
		query,
		contact.UserID,
		contact.Name,
		string(pronounsJSON),
		contact.Relationship,
		contact.Age,
		string(traitsJSON),
		contact.FirstMentionedAt,
		contact.CreatedVia,
		contact.Status,
		contact.Version,
		contact.CreatedAt,
		contact.UpdatedAt,
		confidence,
		lastMentioned,
		contact.ContactRole,
		string(intentionsJSON),
		string(successesJSON),
		string(dependenciesJSON),
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
func (r *ContactRepository) GetByID(contactID int64) (*models.Contact, error) {
	query := `
		SELECT id, user_id, name, pronouns, relationship, age, characteristics, first_mentioned_at, created_via, status, version, created_at, updated_at, contact_role, involved_intentions, past_successes, dependencies, contact_role, involved_intentions, past_successes, dependencies
		FROM contacts
		WHERE id = ? AND status = 'active'
	`

	contact := &models.Contact{}
	var traitsJSON sql.NullString
	var pronounsJSON sql.NullString
	var ageSQL sql.NullString
	var firstMentionedSQL sql.NullInt64
	var createdViaSQL sql.NullString
	var contactRoleSQL sql.NullString
	var intentionsJSON sql.NullString
	var successesJSON sql.NullString
	var dependenciesJSON sql.NullString

	err := r.db.QueryRow(query, contactID).Scan(
		&contact.ID,
		&contact.UserID,
		&contact.Name,
		&pronounsJSON,
		&contact.Relationship,
		&ageSQL,
		&traitsJSON,
		&firstMentionedSQL,
		&createdViaSQL,
		&contact.Status,
		&contact.Version,
		&contact.CreatedAt,
		&contact.UpdatedAt,
		&contactRoleSQL,
		&intentionsJSON,
		&successesJSON,
		&dependenciesJSON,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Handle pronouns
	if pronounsJSON.Valid && pronounsJSON.String != "" {
		json.Unmarshal([]byte(pronounsJSON.String), &contact.Pronouns)
	}

	if ageSQL.Valid {
		contact.Age = ageSQL.String
	}

	if traitsJSON.Valid {
		json.Unmarshal([]byte(traitsJSON.String), &contact.Characteristics)
	}

	if firstMentionedSQL.Valid {
		contact.FirstMentionedAt = firstMentionedSQL.Int64
	}

	if createdViaSQL.Valid {
		contact.CreatedVia = createdViaSQL.String
	}

	// Handle WHAT context
	if contactRoleSQL.Valid {
		contact.ContactRole = contactRoleSQL.String
	}

	if intentionsJSON.Valid && intentionsJSON.String != "" {
		json.Unmarshal([]byte(intentionsJSON.String), &contact.InvolvedInIntentions)
	}

	if successesJSON.Valid && successesJSON.String != "" {
		json.Unmarshal([]byte(successesJSON.String), &contact.PastSuccesses)
	}

	if dependenciesJSON.Valid && dependenciesJSON.String != "" {
		json.Unmarshal([]byte(dependenciesJSON.String), &contact.Dependencies)
	}

	return contact, nil
}

// GetByName retrieves a contact by user and name
func (r *ContactRepository) GetByName(userID, name string) (*models.Contact, error) {
	query := `
		SELECT id, user_id, name, pronouns, relationship, age, characteristics, first_mentioned_at, created_via, status, version, created_at, updated_at, contact_role, involved_intentions, past_successes, dependencies
		FROM contacts
		WHERE user_id = ? AND name = ? AND status = 'active'
	`

	contact := &models.Contact{}
	var traitsJSON sql.NullString
	var pronounsJSON sql.NullString
	var ageSQL sql.NullString
	var firstMentionedSQL sql.NullInt64
	var createdViaSQL sql.NullString

	err := r.db.QueryRow(query, userID, name).Scan(
		&contact.ID,
		&contact.UserID,
		&contact.Name,
		&pronounsJSON,
		&contact.Relationship,
		&ageSQL,
		&traitsJSON,
		&firstMentionedSQL,
		&createdViaSQL,
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

	// Handle pronouns
	if pronounsJSON.Valid && pronounsJSON.String != "" {
		json.Unmarshal([]byte(pronounsJSON.String), &contact.Pronouns)
	}

	if ageSQL.Valid {
		contact.Age = ageSQL.String
	}

	if traitsJSON.Valid {
		json.Unmarshal([]byte(traitsJSON.String), &contact.Characteristics)
	}

	if firstMentionedSQL.Valid {
		contact.FirstMentionedAt = firstMentionedSQL.Int64
	}

	if createdViaSQL.Valid {
		contact.CreatedVia = createdViaSQL.String
	}

	return contact, nil
}

// GetByUserID retrieves all active contacts for a user
func (r *ContactRepository) GetByUserID(userID string) ([]*models.Contact, error) {
	query := `
		SELECT id, user_id, name, pronouns, relationship, age, characteristics, first_mentioned_at, created_via, status, version, created_at, updated_at, contact_role, involved_intentions, past_successes, dependencies
		FROM contacts
		WHERE user_id = ? AND status = 'active'
		ORDER BY updated_at DESC
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []*models.Contact
	for rows.Next() {
		contact := &models.Contact{}
		var traitsJSON sql.NullString
		var pronounsJSON sql.NullString
		var ageSQL sql.NullString
		var firstMentionedSQL sql.NullInt64
		var createdViaSQL sql.NullString

		err := rows.Scan(
			&contact.ID,
			&contact.UserID,
			&contact.Name,
			&pronounsJSON,
			&contact.Relationship,
			&ageSQL,
			&traitsJSON,
			&firstMentionedSQL,
			&createdViaSQL,
			&contact.Status,
			&contact.Version,
			&contact.CreatedAt,
			&contact.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		// Handle pronouns
		if pronounsJSON.Valid && pronounsJSON.String != "" {
			json.Unmarshal([]byte(pronounsJSON.String), &contact.Pronouns)
		}

		if ageSQL.Valid {
			contact.Age = ageSQL.String
		}

		if traitsJSON.Valid {
			json.Unmarshal([]byte(traitsJSON.String), &contact.Characteristics)
		}

		if firstMentionedSQL.Valid {
			contact.FirstMentionedAt = firstMentionedSQL.Int64
		}

		if createdViaSQL.Valid {
			contact.CreatedVia = createdViaSQL.String
		}

		contacts = append(contacts, contact)
	}

	return contacts, rows.Err()
}

// GetByRelationship retrieves contacts by relationship type
func (r *ContactRepository) GetByRelationship(userID, relationship string) ([]*models.Contact, error) {
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

	var contacts []*models.Contact
	for rows.Next() {
		contact := &models.Contact{}
		var traitsJSON sql.NullString
		var ageSQL sql.NullString
		var firstMentionedSQL sql.NullInt64
		var createdViaSQL sql.NullString

		err := rows.Scan(
			&contact.ID,
			&contact.UserID,
			&contact.Name,
			&contact.Relationship,
			&ageSQL,
			&traitsJSON,
			&firstMentionedSQL,
			&createdViaSQL,
			&contact.Status,
			&contact.Version,
			&contact.CreatedAt,
			&contact.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		if ageSQL.Valid {
			contact.Age = ageSQL.String
		}

		if traitsJSON.Valid {
			json.Unmarshal([]byte(traitsJSON.String), &contact.Characteristics)
		}

		if firstMentionedSQL.Valid {
			contact.FirstMentionedAt = firstMentionedSQL.Int64
		}

		if createdViaSQL.Valid {
			contact.CreatedVia = createdViaSQL.String
		}

		contacts = append(contacts, contact)
	}

	return contacts, rows.Err()
}

// Update updates a contact with optimistic locking
// Returns error if version doesn't match
func (r *ContactRepository) Update(contact *models.Contact) error {
	if contact.ID == 0 {
		return fmt.Errorf("contact ID required for update")
	}

	oldVersion := contact.Version
	contact.Version++ // Increment version
	contact.UpdatedAt = time.Now().Unix()

	traitsJSON, _ := json.Marshal(contact.Characteristics)

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
func (r *ContactRepository) GetAll(userID string) ([]*models.Contact, error) {
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

	var contacts []*models.Contact
	for rows.Next() {
		contact := &models.Contact{}
		var charJSON sql.NullString
		var ageSQL sql.NullString
		var firstMentionedSQL sql.NullInt64
		var createdViaSQL sql.NullString

		err := rows.Scan(
			&contact.ID,
			&contact.UserID,
			&contact.Name,
			&contact.Relationship,
			&ageSQL,
			&charJSON,
			&firstMentionedSQL,
			&createdViaSQL,
			&contact.Status,
			&contact.Version,
			&contact.CreatedAt,
			&contact.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		if ageSQL.Valid {
			contact.Age = ageSQL.String
		}

		// Parse characteristics from JSON
		if charJSON.Valid {
			if err := json.Unmarshal([]byte(charJSON.String), &contact.Characteristics); err != nil {
				log.Printf("[ContactRepository] WARNING: Failed to unmarshal characteristics JSON for contact %d: %v", contact.ID, err)
			}
		}

		if firstMentionedSQL.Valid {
			contact.FirstMentionedAt = firstMentionedSQL.Int64
		}

		if createdViaSQL.Valid {
			contact.CreatedVia = createdViaSQL.String
		}

		contacts = append(contacts, contact)
	}

	return contacts, rows.Err()
}

// SaveExtractedContact saves a contact extracted from message context (Gap 1: Contact persistence)
// Creates new contact or updates mention tracking for existing contact
func (r *ContactRepository) SaveExtractedContact(userID, conversationID string, extractedContact interface{}, confidence float64) error {
	log.Printf("[ContactRepository] Saving extracted contact with confidence %.2f", confidence)

	// Extract fields from ExtractedContact (dynamic type)
	contactMap, ok := extractedContact.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid contact format")
	}

	name, _ := contactMap["name"].(string)
	relationship, _ := contactMap["relationship"].(string)

	if name == "" {
		return fmt.Errorf("contact name required")
	}

	// Check if contact already exists
	existing, _ := r.GetByName(userID, name)

	now := time.Now().Unix()
	if existing != nil {
		// Update mention tracking
		existing.LastMentionedAt = now
		existing.ExtractionCount++
		if confidence > existing.Confidence {
			existing.Confidence = confidence // Update if higher confidence
		}
		return r.Save(existing)
	}

	// Create new contact
	contact := &models.Contact{
		UserID:           userID,
		Name:             name,
		Relationship:     relationship,
		CreatedVia:       "conversation",
		Status:           "active",
		Confidence:       confidence,
		FirstMentionedAt: now,
		LastMentionedAt:  now,
		ExtractionCount:  1,
	}

	return r.Save(contact)
}

// RecordContactMention updates last_mentioned_at for tracking (Gap 1 part 2)
func (r *ContactRepository) RecordContactMention(contactID int64) error {
	query := `UPDATE contacts SET last_mentioned_at = ?, extraction_count = extraction_count + 1 WHERE id = ?`
	_, err := r.db.Exec(query, time.Now().Unix(), contactID)
	return err
}

// UpdateFromClarification applies a correction from clarification to a contact
// Used when user clarifies what was previously extracted incorrectly
func (r *ContactRepository) UpdateFromClarification(contactID int64, correction string) error {
	log.Printf("[V2] ContactRepository: updating contact %d from clarification", contactID)

	// Get current contact to merge with correction
	contact, err := r.GetByID(contactID)
	if err != nil {
		log.Printf("[V2] Error getting contact for update: %v", err)
		return err
	}

	// Mark that this was corrected via clarification
	if contact.Notes == "" {
		contact.Notes = fmt.Sprintf("Corrected via clarification: %s", correction)
	} else {
		contact.Notes = fmt.Sprintf("%s\nCorrected via clarification: %s", contact.Notes, correction)
	}

	contact.UpdatedAt = time.Now().Unix()

	query := `
		UPDATE contacts
		SET notes = ?, updated_at = ?, extraction_count = extraction_count + 1
		WHERE id = ?
	`

	_, err = r.db.Exec(query, contact.Notes, contact.UpdatedAt, contactID)
	if err != nil {
		log.Printf("[V2] ContactRepository: error updating contact from clarification: %v", err)
		return err
	}

	log.Printf("[V2] ContactRepository: contact %d updated from clarification", contactID)
	return nil
}

// MarkExtractionSuperseded marks an old extraction as corrected by a new clarification
// This tracks correction history in the contact notes
func (r *ContactRepository) MarkExtractionSuperseded(contactID int64, oldValue string, newValue string) error {
	log.Printf("[V2] ContactRepository: marking extraction superseded for contact %d", contactID)

	contact, err := r.GetByID(contactID)
	if err != nil {
		return err
	}

	// Record the correction in notes
	correctionNote := fmt.Sprintf("Superseded: '%s' → '%s' (clarified)", oldValue, newValue)
	if contact.Notes == "" {
		contact.Notes = correctionNote
	} else {
		contact.Notes = fmt.Sprintf("%s\n%s", contact.Notes, correctionNote)
	}

	query := `UPDATE contacts SET notes = ?, updated_at = ? WHERE id = ?`
	_, err = r.db.Exec(query, contact.Notes, time.Now().Unix(), contactID)

	if err != nil {
		log.Printf("[V2] ContactRepository: error marking extraction superseded: %v", err)
		return err
	}

	log.Printf("[V2] ContactRepository: extraction marked superseded for contact %d", contactID)
	return nil
}

// UnmarshalWHATContext extracts and unmarshals WHAT context fields from SQL nulls
func unmarshalWHATContext(contact *models.Contact, contactRoleSQL, intentionsJSON, successesJSON, dependenciesJSON sql.NullString) {
	if contactRoleSQL.Valid {
		contact.ContactRole = contactRoleSQL.String
	}

	if intentionsJSON.Valid && intentionsJSON.String != "" {
		json.Unmarshal([]byte(intentionsJSON.String), &contact.InvolvedInIntentions)
	}

	if successesJSON.Valid && successesJSON.String != "" {
		json.Unmarshal([]byte(successesJSON.String), &contact.PastSuccesses)
	}

	if dependenciesJSON.Valid && dependenciesJSON.String != "" {
		json.Unmarshal([]byte(dependenciesJSON.String), &contact.Dependencies)
	}
}
