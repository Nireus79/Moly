package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"moly/models"
)

// ContactRepository handles contact database operations.
//
// Who a repository can see: a person with a name belongs to the user and is seen by every conversation. A person without a name
// yet (unnamed, asked, nameless: a label such as "boss") is unresolved, and an unresolved person belongs to the conversation it
// came from. NewContactRepository sees only the resolved ones; InConversation adds the unresolved people of one conversation.
// Nothing unresolved is ever returned outside its own conversation, so it can neither be shown nor be matched by another one.
type ContactRepository struct {
	db             *Database
	conversationID string
}

// NewContactRepository creates a contact repository that sees the resolved (named) people of a user.
func NewContactRepository(db *Database) *ContactRepository {
	return &ContactRepository{db: db}
}

// InConversation returns a repository that also sees the unresolved people of the conversation, and creates them there.
func (r *ContactRepository) InConversation(conversationID string) *ContactRepository {
	return &ContactRepository{db: r.db, conversationID: conversationID}
}

// visible is the one condition every read of contacts applies. Its argument is the repository's conversation ("" matches no row).
const visible = "(name_status = 'named' OR (origin_conversation_id != '' AND origin_conversation_id = ?))"

// The name states are defined in models (Contact.NameStatus); only "named" is resolved.
const (
	NameStatusNamed    = models.NameStatusNamed
	NameStatusUnnamed  = models.NameStatusUnnamed
	NameStatusAsked    = models.NameStatusAsked
	NameStatusNameless = models.NameStatusNameless
)

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

	// A named person belongs to the user; one without a name belongs to a conversation, and cannot be saved without one.
	nameStatus := nameStatusOrDefault(contact.NameStatus)
	origin := ""
	if nameStatus != NameStatusNamed {
		origin = contact.OriginConversationID
		if origin == "" {
			origin = r.conversationID
		}
		if origin == "" {
			return fmt.Errorf("a person without a name belongs to a conversation: none given")
		}
		contact.OriginConversationID = origin
	} else {
		contact.OriginConversationID = ""
	}

	query := `
		INSERT OR REPLACE INTO contacts
		(user_id, name, pronouns, relationship, age, characteristics, first_mentioned_at, created_via, status, version, created_at, updated_at, confidence, last_mentioned_at, contact_role, involved_intentions, past_successes, dependencies, name_status, origin_conversation_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
		nameStatus,
		origin,
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
		"SELECT "+contactColumns+" FROM contacts WHERE id = ? AND user_id = ? AND status = 'active' AND "+visible,
		contactID, userID, r.conversationID,
	))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return contact, err
}

// GetByName returns the person with this name or label, among the people this repository sees (see ContactRepository).
func (r *ContactRepository) GetByName(userID, name string) (*models.Contact, error) {
	contact, err := scanContact(r.db.QueryRow(
		"SELECT "+contactColumns+" FROM contacts WHERE user_id = ? AND name = ? AND status = 'active' AND "+visible+" ORDER BY (name_status = 'named') DESC LIMIT 1",
		userID, name, r.conversationID,
	))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return contact, err
}

// GetNamedByName returns the person who has this name (a resolved one), whatever the conversation.
func (r *ContactRepository) GetNamedByName(userID, name string) (*models.Contact, error) {
	contact, err := scanContact(r.db.QueryRow(
		"SELECT "+contactColumns+" FROM contacts WHERE user_id = ? AND name = ? AND status = 'active' AND name_status = 'named'",
		userID, name,
	))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return contact, err
}

// GetUnresolvedByLabel returns the person of this conversation who is known only by this label, if there is one.
func (r *ContactRepository) GetUnresolvedByLabel(userID, label string) (*models.Contact, error) {
	if r.conversationID == "" {
		return nil, nil
	}
	contact, err := scanContact(r.db.QueryRow(
		"SELECT "+contactColumns+" FROM contacts WHERE user_id = ? AND name = ? AND status = 'active' AND name_status != 'named' AND origin_conversation_id = ?",
		userID, label, r.conversationID,
	))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return contact, err
}

// GetByUserID returns the people this repository sees (see ContactRepository), the most recently updated first.
func (r *ContactRepository) GetByUserID(userID string) ([]*models.Contact, error) {
	rows, err := r.db.Query(
		"SELECT "+contactColumns+" FROM contacts WHERE user_id = ? AND status = 'active' AND "+visible+" ORDER BY updated_at DESC",
		userID, r.conversationID,
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
		return NameStatusNamed
	}
	return status
}

// contactColumns is the one column list used for reading a full contact. Keep it in step with scanContact.
const contactColumns = "id, user_id, name, pronouns, relationship, age, characteristics, first_mentioned_at, created_via, status, version, created_at, updated_at, contact_role, involved_intentions, past_successes, dependencies, name_status, confidence, origin_conversation_id"

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
	var originSQL sql.NullString

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
		&originSQL,
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
	if originSQL.Valid {
		contact.OriginConversationID = originSQL.String
	}
	return contact, nil
}

// SetNameStatus sets the name status of one unresolved contact: "unnamed", "asked" or "nameless". A person becomes named only
// through NameContact, which compares the name with the people the user already has.
func (r *ContactRepository) SetNameStatus(userID string, contactID int64, status string) error {
	return r.SetNameStatusWith(r.db, userID, contactID, status)
}

// SetNameStatusWith is SetNameStatus through the given executor (for example a transaction).
func (r *ContactRepository) SetNameStatusWith(ex Executor, userID string, contactID int64, status string) error {
	if status != NameStatusUnnamed && status != NameStatusAsked && status != NameStatusNameless {
		return fmt.Errorf("invalid name status %q", status)
	}
	_, err := ex.Exec(
		`UPDATE contacts SET name_status = ?, updated_at = ? WHERE id = ? AND user_id = ? AND name_status != 'named'`,
		status, time.Now().Unix(), contactID, userID,
	)
	return err
}

// NameContact gives a person who has no name yet the name the user gave. The name is compared with the people the user already
// has first: when one of them has it, the person is that one and the two are merged (the stored values of the known person are
// kept, blanks are filled, traits are joined) and the known person is returned. Otherwise the same row takes the name, so its
// history stays, and it becomes a person of the user instead of a person of one conversation.
func (r *ContactRepository) NameContact(userID string, contactID int64, newName string) (*models.Contact, error) {
	if newName == "" {
		return nil, fmt.Errorf("name required")
	}
	waiting, err := r.GetByID(userID, contactID)
	if err != nil {
		return nil, err
	}
	if waiting == nil || waiting.NameStatus == NameStatusNamed {
		return nil, fmt.Errorf("no person is waiting for a name with id %d", contactID)
	}
	// The words the user first used for this person ("my manager") are what the person is to them: keep them as the role.
	label := waiting.Name

	known, err := r.GetNamedByName(userID, newName)
	if err != nil {
		return nil, err
	}
	if known != nil {
		if known.ContactRole == "" {
			known.ContactRole = label
		}
		if known.Relationship == "" || known.Relationship == "other" {
			known.Relationship = waiting.Relationship
		}
		known.Characteristics = joinStrings(known.Characteristics, waiting.Characteristics)
		known.InvolvedInIntentions = joinStrings(known.InvolvedInIntentions, waiting.InvolvedInIntentions)
		known.PastSuccesses = joinStrings(known.PastSuccesses, waiting.PastSuccesses)
		known.Dependencies = joinStrings(known.Dependencies, waiting.Dependencies)
		if err := r.Save(known); err != nil {
			return nil, err
		}
		if _, err := r.db.Exec(`DELETE FROM contacts WHERE id = ? AND user_id = ? AND name_status != 'named'`, contactID, userID); err != nil {
			return nil, err
		}
		return known, nil
	}
	role := waiting.ContactRole
	if role == "" {
		role = label
	}
	if _, err := r.db.Exec(
		`UPDATE contacts SET name = ?, name_status = 'named', origin_conversation_id = '', contact_role = ?, updated_at = ? WHERE id = ? AND user_id = ?`,
		newName, role, time.Now().Unix(), contactID, userID,
	); err != nil {
		return nil, err
	}
	waiting.Name, waiting.NameStatus, waiting.OriginConversationID, waiting.ContactRole = newName, NameStatusNamed, "", role
	return waiting, nil
}

func joinStrings(have, more []string) []string {
	seen := make(map[string]bool, len(have))
	for _, h := range have {
		seen[h] = true
	}
	out := append([]string(nil), have...)
	for _, m := range more {
		if m != "" && !seen[m] {
			seen[m] = true
			out = append(out, m)
		}
	}
	return out
}

// ApplyNameAnswer applies the user's message to a name question asked in the previous message of this conversation.
// It returns true when a person of this conversation was waiting for a name (the message was the answer).
// A given name goes through NameContact (compared with the people the user already has). No given name keeps the label: the
// person stays a person of this conversation, known by that label, and is not asked again.
func (r *ContactRepository) ApplyNameAnswer(userID, givenName string) (bool, error) {
	contacts, err := r.GetByUserID(userID)
	if err != nil {
		return false, err
	}
	answered := false
	for _, c := range contacts {
		if c.NameStatus != NameStatusAsked {
			continue
		}
		answered = true
		if givenName != "" && givenName != c.Name {
			if _, err := r.NameContact(userID, c.ID, givenName); err != nil {
				return answered, err
			}
			continue
		}
		if err := r.SetNameStatus(userID, c.ID, NameStatusNameless); err != nil {
			return answered, err
		}
	}
	return answered, nil
}
