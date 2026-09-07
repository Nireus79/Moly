package agents

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"moly/models"
)

// contextManager - Manages knowledge base (About Me, Contacts, Reflections)
type contextManager struct {
	userID string
	db     interface{} // Generic interface to avoid circular imports
}

// NewContextManager - Create new context manager (no database)
func NewContextManager(userID string) (models.ContextManagerAgent, error) {
	if userID == "" {
		return nil, errors.New("userID cannot be empty")
	}

	return &contextManager{
		userID: userID,
		db:     nil,
	}, nil
}

// NewContextManagerWithDB - Create new context manager with database access
func NewContextManagerWithDB(userID string, db interface{}) (models.ContextManagerAgent, error) {
	if userID == "" {
		return nil, errors.New("userID cannot be empty")
	}

	return &contextManager{
		userID: userID,
		db:     db,
	}, nil
}

// GetAboutMe - Retrieve user's About Me profile
func (cm *contextManager) GetAboutMe(userID string) (*models.AboutMe, error) {
	if userID == "" {
		return nil, errors.New("userID cannot be empty")
	}

	if cm.db == nil {
		return &models.AboutMe{UserID: userID, CreatedAt: time.Now().Unix(), UpdatedAt: time.Now().Unix()}, nil
	}

	db := cm.db.(*sql.DB)
	var mode, tone, length, response, platform string
	var updatedAt time.Time

	err := db.QueryRow(`
		SELECT communication_mode, preferred_tone, average_message_length,
		       response_time_preference, primary_platform, updated_at
		FROM behavior_patterns LIMIT 1
	`).Scan(&mode, &tone, &length, &response, &platform, &updatedAt)

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return &models.AboutMe{
		UserID:              userID,
		CommunicationStyle: mode,
		Values:             []string{tone},
		PreferredTone:      tone,
		Notes:              length + " messages, " + response + " responses",
		CreatedAt:          updatedAt.Unix(),
		UpdatedAt:          updatedAt.Unix(),
	}, nil
}

// SetAboutMe - Save/update About Me profile
func (cm *contextManager) SetAboutMe(userID string, aboutMe *models.AboutMe) error {
	if userID == "" {
		return errors.New("userID cannot be empty")
	}

	if aboutMe == nil {
		return errors.New("aboutMe cannot be nil")
	}

	if cm.db == nil {
		return nil
	}

	db := cm.db.(*sql.DB)
	_, err := db.Exec(`
		UPDATE behavior_patterns SET
			communication_mode = ?,
			preferred_tone = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE 1=1
	`, aboutMe.CommunicationStyle, aboutMe.PreferredTone)

	return err
}

// GetContact - Retrieve contact information
func (cm *contextManager) GetContact(userID, contactID string) (*models.Contact, error) {
	if userID == "" || contactID == "" {
		return nil, errors.New("userID and contactID cannot be empty")
	}

	if cm.db == nil {
		return &models.Contact{ID: contactID, UserID: userID, CreatedAt: time.Now().Unix(), UpdatedAt: time.Now().Unix()}, nil
	}

	db := cm.db.(*sql.DB)
	var id int
	var name, relationship, platform, notes, style string
	var createdAt, updatedAt time.Time

	err := db.QueryRow(`
		SELECT id, name, relationship, platform, notes, communication_style, created_at, updated_at
		FROM contacts WHERE id = ?
	`, contactID).Scan(&id, &name, &relationship, &platform, &notes, &style, &createdAt, &updatedAt)

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if err == sql.ErrNoRows {
		return &models.Contact{ID: contactID, UserID: userID, CreatedAt: time.Now().Unix(), UpdatedAt: time.Now().Unix()}, nil
	}

	return &models.Contact{
		ID:                       contactID,
		UserID:                   userID,
		Name:                     name,
		Relationship:             relationship,
		CommunicationPreferences: style,
		Notes:                    notes,
		CreatedAt:                createdAt.Unix(),
		UpdatedAt:                updatedAt.Unix(),
	}, nil
}

// GetContacts - Retrieve all user's contacts
func (cm *contextManager) GetContacts(userID string) ([]models.Contact, error) {
	if userID == "" {
		return nil, errors.New("userID cannot be empty")
	}

	if cm.db == nil {
		return []models.Contact{}, nil
	}

	db := cm.db.(*sql.DB)
	rows, err := db.Query(`
		SELECT id, name, relationship, platform, notes, communication_style, created_at, updated_at
		FROM contacts ORDER BY created_at DESC
	`)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	defer rows.Close()

	var contacts []models.Contact
	for rows.Next() {
		var id int
		var name, relationship, platform, notes, style string
		var createdAt, updatedAt time.Time

		if err := rows.Scan(&id, &name, &relationship, &platform, &notes, &style, &createdAt, &updatedAt); err != nil {
			continue
		}

		contacts = append(contacts, models.Contact{
			ID:                       string(rune(id)),
			UserID:                   userID,
			Name:                     name,
			Relationship:             relationship,
			CommunicationPreferences: style,
			Notes:                    notes,
			CreatedAt:                createdAt.Unix(),
			UpdatedAt:                updatedAt.Unix(),
		})
	}

	return contacts, nil
}

// CreateContact - Create new contact
func (cm *contextManager) CreateContact(userID string, contact *models.Contact) (*models.Contact, error) {
	if userID == "" {
		return nil, errors.New("userID cannot be empty")
	}

	if contact == nil {
		return nil, errors.New("contact cannot be nil")
	}

	contact.UserID = userID
	contact.CreatedAt = time.Now().Unix()
	contact.UpdatedAt = time.Now().Unix()

	if cm.db == nil {
		return contact, nil
	}

	db := cm.db.(*sql.DB)
	result, err := db.Exec(`
		INSERT INTO contacts (name, relationship, platform, notes, communication_style)
		VALUES (?, ?, ?, ?, ?)
	`, contact.Name, contact.Relationship, "", contact.Notes, contact.CommunicationPreferences)

	if err == nil {
		id, _ := result.LastInsertId()
		contact.ID = string(rune(id))
	}

	return contact, err
}

// UpdateContact - Update contact information
func (cm *contextManager) UpdateContact(userID, contactID string, updates models.Contact) error {
	if userID == "" || contactID == "" {
		return errors.New("userID and contactID cannot be empty")
	}

	if cm.db == nil {
		return nil
	}

	db := cm.db.(*sql.DB)
	_, err := db.Exec(`
		UPDATE contacts SET
			name = ?, relationship = ?, notes = ?, communication_style = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, updates.Name, updates.Relationship, updates.Notes, updates.CommunicationPreferences, contactID)

	return err
}

// GetRelevantContext - Retrieve context relevant for a conversation
func (cm *contextManager) GetRelevantContext(conversationID, userID string) (*models.Context, error) {
	if conversationID == "" || userID == "" {
		return nil, errors.New("conversationID and userID cannot be empty")
	}

	// Load AboutMe
	aboutMe, _ := cm.GetAboutMe(userID)

	// Load conversation history if database available
	var history []models.Message
	if cm.db != nil {
		db := cm.db.(*sql.DB)
		rows, err := db.Query(`
			SELECT role, content, created_at FROM interactions
			WHERE conversation_id = ? ORDER BY created_at DESC LIMIT 10
		`, conversationID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var role, content string
				var ts time.Time
				if err := rows.Scan(&role, &content, &ts); err == nil {
					history = append(history, models.Message{
						Role:    role,
						Content: content,
						Type:    "message",
					})
				}
			}
		}
	}

	// Load contacts
	contacts, _ := cm.GetContacts(userID)
	var contactProfile *models.Contact
	if len(contacts) > 0 {
		contactProfile = &contacts[0]
	}

	context := &models.Context{
		AboutMe:              aboutMe,
		ContactProfile:       contactProfile,
		ConversationHistory: history,
		ContextQuality:      "partial",
		Gaps:                []string{},
	}

	if aboutMe != nil && len(history) > 0 && contactProfile != nil {
		context.ContextQuality = "comprehensive"
	}

	return context, nil
}

// SaveReflection - Save extracted reflection (pending approval)
func (cm *contextManager) SaveReflection(conversationID string, reflection *models.Reflection) error {
	if conversationID == "" {
		return errors.New("conversationID cannot be empty")
	}

	if reflection == nil {
		return errors.New("reflection cannot be nil")
	}

	reflection.ConversationID = conversationID
	reflection.Status = "pending_approval"

	if cm.db == nil {
		return nil
	}

	db := cm.db.(*sql.DB)
	data, _ := json.Marshal(reflection)
	_, err := db.Exec(`
		INSERT INTO interactions (conversation_id, topic, ai_summary, context_metadata)
		VALUES (?, ?, ?, ?)
	`, conversationID, "reflection", string(data), string(data))

	return err
}

// ApproveReflection - User approves a reflection
func (cm *contextManager) ApproveReflection(conversationID string, reflection *models.Reflection) error {
	if conversationID == "" {
		return errors.New("conversationID cannot be empty")
	}

	if reflection == nil {
		return errors.New("reflection cannot be nil")
	}

	reflection.Status = "approved"
	reflection.ApprovedAt = time.Now().Unix()

	if cm.db == nil {
		return nil
	}

	db := cm.db.(*sql.DB)
	data, _ := json.Marshal(reflection)
	_, err := db.Exec(`
		UPDATE interactions SET
			context_metadata = ?, ai_summary = 'approved_reflection'
		WHERE conversation_id = ? AND topic = 'reflection'
		ORDER BY created_at DESC LIMIT 1
	`, string(data), conversationID)

	return err
}

// AppendMessage - Add message to conversation history
func (cm *contextManager) AppendMessage(conversationID string, message *models.Message) error {
	if conversationID == "" {
		return errors.New("conversationID cannot be empty")
	}

	if message == nil {
		return errors.New("message cannot be nil")
	}

	message.ConversationID = conversationID
	message.Timestamp = time.Now().Unix()

	if cm.db == nil {
		return nil
	}

	db := cm.db.(*sql.DB)
	_, err := db.Exec(`
		INSERT INTO interactions (conversation_id, role, content, topic)
		VALUES (?, ?, ?, 'message')
	`, conversationID, message.Role, message.Content)

	return err
}
