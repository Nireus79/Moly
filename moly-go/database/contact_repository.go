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
	log.Printf("[Database] ContactRepository: saving contact %s for user %s", contact.Name, contact.UserID)

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
		(user_id, name, pronouns, relationship, age, characteristics, first_mentioned_at, created_via, status, version, created_at, updated_at, confidence, last_mentioned_at, contact_role, involved_intentions, past_successes, dependencies, name_status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
		nameStatusOrDefault(contact.NameStatus),
	)

	if err != nil {
		log.Printf("[Database] ContactRepository ERROR: %v", err)
		return err
	}

	// Get the ID if this was an insert
	if contact.ID == 0 {
		id, _ := result.LastInsertId()
		contact.ID = id
	}

	log.Printf("[Database] ContactRepository: saved contact %s (id=%d)", contact.Name, contact.ID)
	return nil
}

// GetByID retrieves a contact by ID with user_id validation (data isolation)
func (r *ContactRepository) GetByID(userID string, contactID int64) (*models.Contact, error) {
	contact, err := scanContact(r.db.QueryRow(
		"SELECT "+contactColumns+" FROM contacts WHERE id = ? AND user_id = ? AND status = 'active'",
		contactID, userID,
	))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return contact, err
}

func (r *ContactRepository) GetByName(userID, name string) (*models.Contact, error) {
	contact, err := scanContact(r.db.QueryRow(
		"SELECT "+contactColumns+" FROM contacts WHERE user_id = ? AND name = ? AND status = 'active'",
		userID, name,
	))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return contact, err
}

func (r *ContactRepository) GetByUserID(userID string) ([]*models.Contact, error) {
	rows, err := r.db.Query(
		"SELECT "+contactColumns+" FROM contacts WHERE user_id = ? AND status = 'active' ORDER BY updated_at DESC",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []*models.Contact
	for rows.Next() {
		contact, err := scanContact(rows)
		if err != nil {
			return nil, err
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
		log.Printf("[Database] ContactRepository: version mismatch for contact %d", contact.ID)
		return fmt.Errorf("contact version mismatch - contact was updated elsewhere")
	}

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

// nameStatusOrDefault returns the stored name status; a contact without one has a real name.
func nameStatusOrDefault(status string) string {
	if status == "" {
		return "named"
	}
	return status
}

// contactColumns is the one column list used for reading a full contact. Keep it in step with scanContact.
const contactColumns = "id, user_id, name, pronouns, relationship, age, characteristics, first_mentioned_at, created_via, status, version, created_at, updated_at, contact_role, involved_intentions, past_successes, dependencies, name_status, confidence"

// scanContact reads one row selected with contactColumns. It is the only place that decodes a full contact row.
func scanContact(row interface {
	Scan(dest ...interface{}) error
}) (*models.Contact, error) {
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
	var nameStatusSQL sql.NullString
	var confidenceSQL sql.NullFloat64

	if err := row.Scan(
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
		&nameStatusSQL,
		&confidenceSQL,
	); err != nil {
		return nil, err
	}

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
	unmarshalWHATContext(contact, contactRoleSQL, intentionsJSON, successesJSON, dependenciesJSON)

	contact.NameStatus = "named"
	if nameStatusSQL.Valid && nameStatusSQL.String != "" {
		contact.NameStatus = nameStatusSQL.String
	}
	if confidenceSQL.Valid {
		contact.Confidence = confidenceSQL.Float64
	}
	return contact, nil
}

// SetNameStatus sets the name status of one contact: "named", "unnamed" or "asked".
func (r *ContactRepository) SetNameStatus(userID string, contactID int64, status string) error {
	return r.SetNameStatusWith(r.db, userID, contactID, status)
}

// SetNameStatusWith is SetNameStatus through the given executor (for example a transaction).
func (r *ContactRepository) SetNameStatusWith(ex Executor, userID string, contactID int64, status string) error {
	if status != "named" && status != "unnamed" && status != "asked" {
		return fmt.Errorf("invalid name status %q", status)
	}
	_, err := ex.Exec(
		`UPDATE contacts SET name_status = ?, updated_at = ? WHERE id = ? AND user_id = ?`,
		status, time.Now().Unix(), contactID, userID,
	)
	return err
}

// RenameContact gives an unnamed contact its real name. It updates the same row, so the contact's history stays.
func (r *ContactRepository) RenameContact(userID string, contactID int64, newName string) error {
	if newName == "" {
		return fmt.Errorf("name required")
	}
	_, err := r.db.Exec(
		`UPDATE contacts SET name = ?, name_status = 'named', updated_at = ? WHERE id = ? AND user_id = ?`,
		newName, time.Now().Unix(), contactID, userID,
	)
	return err
}

// ApplyNameAnswer applies the user's message to a name question asked in the previous message.
// It returns true when a contact was waiting for a name (the message was the answer).
// A given name renames that contact. No given name keeps the label; the contact is not asked again.
func (r *ContactRepository) ApplyNameAnswer(userID, givenName string) (bool, error) {
	contacts, err := r.GetByUserID(userID)
	if err != nil {
		return false, err
	}
	answered := false
	for _, c := range contacts {
		if c.NameStatus != "asked" {
			continue
		}
		answered = true
		if givenName != "" && givenName != c.Name {
			// The words the user first used for this person ("my manager") are what the person is to them: keep them as the role.
			if c.ContactRole == "" {
				if _, err := r.db.Exec(`UPDATE contacts SET contact_role = ? WHERE id = ? AND user_id = ?`, c.Name, c.ID, userID); err != nil {
					return answered, err
				}
			}
			if err := r.RenameContact(userID, c.ID, givenName); err != nil {
				return answered, err
			}
			continue
		}
		if err := r.SetNameStatus(userID, c.ID, "named"); err != nil {
			return answered, err
		}
	}
	return answered, nil
}

// GetSoleNamedByRelationship returns the one named contact with this relationship, or nil when there are none or several.
// It is used when a message refers to a person without a name, and it is the only named person of that kind.
func (r *ContactRepository) GetSoleNamedByRelationship(userID, relationship string) (*models.Contact, error) {
	if relationship == "" {
		return nil, nil
	}
	contacts, err := r.GetByUserID(userID)
	if err != nil {
		return nil, err
	}
	var match *models.Contact
	for _, c := range contacts {
		if c.NameStatus != "named" || c.Relationship != relationship {
			continue
		}
		if match != nil {
			return nil, nil
		}
		match = c
	}
	return match, nil
}
