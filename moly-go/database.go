package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	conn *sql.DB
	path string
}

type Contact struct {
	ID                   int       `json:"id"`
	Name                 string    `json:"name"`
	Relationship         string    `json:"relationship"`
	Platform             string    `json:"platform"`
	Notes                string    `json:"notes"`
	CommunicationStyle   string    `json:"communication_style"`
	InteractionCount     int       `json:"interaction_count"`
	LastInteraction      *time.Time `json:"last_interaction"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type Conversation struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"` // 'single', 'group', 'generic'
	Purpose   string    `json:"purpose"` // 'relationship', 'cover_letter', 'advice', etc
	Notes     string    `json:"notes"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ConversationMember struct {
	ID             int       `json:"id"`
	ConversationID int       `json:"conversation_id"`
	ContactID      int       `json:"contact_id"`
	AddedAt        time.Time `json:"added_at"`
}

type Interaction struct {
	ID             int        `json:"id"`
	ConversationID int        `json:"conversation_id"`
	ContactID      int        `json:"contact_id"` // deprecated, use conversation_id
	Date           time.Time  `json:"date"`
	Platform       string     `json:"platform"`
	Topic          string     `json:"topic"`
	Sentiment      string     `json:"sentiment"`
	AISummary      string     `json:"ai_summary"`
	UserNotes      string     `json:"user_notes"`
	Important      bool       `json:"important"`
	ContextMetadata string    `json:"context_metadata"`
	CreatedAt      time.Time  `json:"created_at"`
}

type BehaviorPattern struct {
	ID                      int    `json:"id"`
	CommunicationMode       string `json:"communication_mode"`
	PreferredTone           string `json:"preferred_tone"`
	AverageMessageLength    string `json:"average_message_length"`
	ResponseTimePreference  string `json:"response_time_preference"`
	PrimaryPlatform         string `json:"primary_platform"`
	UpdatedAt               time.Time `json:"updated_at"`
}

func getConfigDir(filename string) string {
	var configDir string

	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = os.ExpandEnv("$USERPROFILE\\AppData\\Roaming")
		}
		configDir = filepath.Join(appData, "Moly")

	case "darwin":
		configDir = filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "Moly")

	default:
		if xdgHome := os.Getenv("XDG_CONFIG_HOME"); xdgHome != "" {
			configDir = filepath.Join(xdgHome, "moly")
		} else {
			configDir = filepath.Join(os.Getenv("HOME"), ".config", "moly")
		}
	}

	os.MkdirAll(configDir, 0700)
	return filepath.Join(configDir, filename)
}

func initDatabase() (*Database, error) {
	dbPath := getConfigDir("moly.db")

	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	db := &Database{conn: conn, path: dbPath}

	// Create tables if they don't exist
	if err := db.createTables(); err != nil {
		return nil, err
	}

	return db, nil
}

func (db *Database) createTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS contacts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		relationship TEXT,
		platform TEXT,
		notes TEXT,
		communication_style TEXT,
		interaction_count INTEGER DEFAULT 0,
		last_interaction TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(name, platform)
	);

	CREATE TABLE IF NOT EXISTS conversations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		type TEXT DEFAULT 'generic',
		purpose TEXT,
		notes TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS conversation_members (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		conversation_id INTEGER NOT NULL,
		contact_id INTEGER NOT NULL,
		added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(conversation_id) REFERENCES conversations(id),
		FOREIGN KEY(contact_id) REFERENCES contacts(id),
		UNIQUE(conversation_id, contact_id)
	);

	CREATE TABLE IF NOT EXISTS interactions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		conversation_id INTEGER,
		contact_id INTEGER,
		date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		platform TEXT,
		topic TEXT,
		sentiment TEXT,
		ai_summary TEXT,
		user_notes TEXT,
		important BOOLEAN DEFAULT 0,
		context_metadata TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(conversation_id) REFERENCES conversations(id),
		FOREIGN KEY(contact_id) REFERENCES contacts(id)
	);

	CREATE TABLE IF NOT EXISTS behavior_patterns (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		communication_mode TEXT DEFAULT 'direct',
		preferred_tone TEXT DEFAULT 'friendly',
		average_message_length TEXT DEFAULT 'medium',
		response_time_preference TEXT DEFAULT 'thoughtful',
		primary_platform TEXT,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_interactions_conversation ON interactions(conversation_id);
	CREATE INDEX IF NOT EXISTS idx_interactions_contact ON interactions(contact_id);
	CREATE INDEX IF NOT EXISTS idx_interactions_date ON interactions(date);
	CREATE INDEX IF NOT EXISTS idx_contacts_platform ON contacts(platform);
	CREATE INDEX IF NOT EXISTS idx_conversation_members ON conversation_members(conversation_id);
	`

	if _, err := db.conn.Exec(schema); err != nil {
		return fmt.Errorf("failed to create tables: %v", err)
	}

	// Initialize default behavior pattern if not exists
	var count int
	db.conn.QueryRow("SELECT COUNT(*) FROM behavior_patterns").Scan(&count)
	if count == 0 {
		_, err := db.conn.Exec(`
			INSERT INTO behavior_patterns (communication_mode, preferred_tone, average_message_length, response_time_preference)
			VALUES ('direct', 'friendly', 'medium', 'thoughtful')
		`)
		if err != nil {
			return fmt.Errorf("failed to initialize behavior pattern: %v", err)
		}
	}

	return nil
}

func (db *Database) createOrUpdateContact(name, relationship, platform, notes string) (*Contact, error) {
	now := time.Now()

	result, err := db.conn.Exec(`
		INSERT INTO contacts (name, relationship, platform, notes, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(name, platform) DO UPDATE SET
			relationship = excluded.relationship,
			notes = excluded.notes,
			updated_at = excluded.updated_at
	`, name, relationship, platform, notes, now)

	if err != nil {
		return nil, fmt.Errorf("failed to create/update contact: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		// If conflict, fetch existing
		var contact Contact
		err := db.conn.QueryRow(`
			SELECT id, name, relationship, platform, notes, communication_style,
				   interaction_count, last_interaction, created_at, updated_at
			FROM contacts WHERE name = ? AND platform = ?
		`, name, platform).Scan(&contact.ID, &contact.Name, &contact.Relationship,
			&contact.Platform, &contact.Notes, &contact.CommunicationStyle,
			&contact.InteractionCount, &contact.LastInteraction, &contact.CreatedAt, &contact.UpdatedAt)

		if err != nil {
			return nil, err
		}
		return &contact, nil
	}

	contact := &Contact{
		ID:            int(id),
		Name:          name,
		Relationship:  relationship,
		Platform:      platform,
		Notes:         notes,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	return contact, nil
}

func (db *Database) getContact(id int) (*Contact, error) {
	var contact Contact
	var lastInteraction sql.NullTime
	var relationship, platform, notes, communicationStyle sql.NullString

	err := db.conn.QueryRow(`
		SELECT id, name, relationship, platform, notes, communication_style,
			   interaction_count, last_interaction, created_at, updated_at
		FROM contacts WHERE id = ?
	`, id).Scan(&contact.ID, &contact.Name, &relationship,
		&platform, &notes, &communicationStyle,
		&contact.InteractionCount, &lastInteraction, &contact.CreatedAt, &contact.UpdatedAt)

	if err != nil {
		return nil, err
	}

	contact.Relationship = relationship.String
	contact.Platform = platform.String
	contact.Notes = notes.String
	contact.CommunicationStyle = communicationStyle.String
	if lastInteraction.Valid {
		contact.LastInteraction = &lastInteraction.Time
	}

	return &contact, nil
}

func (db *Database) getAllContacts() ([]Contact, error) {
	rows, err := db.conn.Query(`
		SELECT id, name, relationship, platform, notes, communication_style,
			   interaction_count, last_interaction, created_at, updated_at
		FROM contacts ORDER BY last_interaction DESC NULLS LAST
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []Contact
	for rows.Next() {
		var contact Contact
		var lastInteraction sql.NullTime
		var relationship, platform, notes, communicationStyle sql.NullString

		err := rows.Scan(&contact.ID, &contact.Name, &relationship,
			&platform, &notes, &communicationStyle,
			&contact.InteractionCount, &lastInteraction, &contact.CreatedAt, &contact.UpdatedAt)

		if err != nil {
			return nil, err
		}

		contact.Relationship = relationship.String
		contact.Platform = platform.String
		contact.Notes = notes.String
		contact.CommunicationStyle = communicationStyle.String
		if lastInteraction.Valid {
			contact.LastInteraction = &lastInteraction.Time
		}

		contacts = append(contacts, contact)
	}

	return contacts, nil
}

func (db *Database) recordInteraction(contactID int, platform, topic, sentiment, summary, userNotes string) error {
	_, err := db.conn.Exec(`
		INSERT INTO interactions (contact_id, platform, topic, sentiment, ai_summary, user_notes)
		VALUES (?, ?, ?, ?, ?, ?)
	`, contactID, platform, topic, sentiment, summary, userNotes)

	if err != nil {
		return fmt.Errorf("failed to record interaction: %v", err)
	}

	// Update last_interaction and increment counter
	_, err = db.conn.Exec(`
		UPDATE contacts SET last_interaction = CURRENT_TIMESTAMP, interaction_count = interaction_count + 1
		WHERE id = ?
	`, contactID)

	return err
}

func (db *Database) getRecentInteractions(contactID int, limit int) ([]Interaction, error) {
	rows, err := db.conn.Query(`
		SELECT id, contact_id, date, platform, topic, sentiment, ai_summary, user_notes, important, context_metadata, created_at
		FROM interactions WHERE contact_id = ? ORDER BY date DESC LIMIT ?
	`, contactID, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var interactions []Interaction
	for rows.Next() {
		var interaction Interaction
		var platform, topic, sentiment, aiSummary, userNotes, contextMetadata sql.NullString

		err := rows.Scan(&interaction.ID, &interaction.ContactID, &interaction.Date,
			&platform, &topic, &sentiment,
			&aiSummary, &userNotes, &interaction.Important,
			&contextMetadata, &interaction.CreatedAt)

		if err != nil {
			return nil, err
		}

		interaction.Platform = platform.String
		interaction.Topic = topic.String
		interaction.Sentiment = sentiment.String
		interaction.AISummary = aiSummary.String
		interaction.UserNotes = userNotes.String
		interaction.ContextMetadata = contextMetadata.String

		interactions = append(interactions, interaction)
	}

	return interactions, nil
}

func (db *Database) updateBehaviorPattern(mode, tone, messageLength, responseTime, platform string) error {
	_, err := db.conn.Exec(`
		UPDATE behavior_patterns SET
			communication_mode = COALESCE(NULLIF(?, ''), communication_mode),
			preferred_tone = COALESCE(NULLIF(?, ''), preferred_tone),
			average_message_length = COALESCE(NULLIF(?, ''), average_message_length),
			response_time_preference = COALESCE(NULLIF(?, ''), response_time_preference),
			primary_platform = COALESCE(NULLIF(?, ''), primary_platform),
			updated_at = CURRENT_TIMESTAMP
	`, mode, tone, messageLength, responseTime, platform)

	return err
}

func (db *Database) getBehaviorPattern() (*BehaviorPattern, error) {
	var pattern BehaviorPattern
	var primaryPlatform sql.NullString
	var communicationMode, preferredTone, messageLength, responseTime sql.NullString

	err := db.conn.QueryRow(`
		SELECT id, communication_mode, preferred_tone, average_message_length,
			   response_time_preference, primary_platform, updated_at
		FROM behavior_patterns LIMIT 1
	`).Scan(&pattern.ID, &communicationMode, &preferredTone,
		&messageLength, &responseTime,
		&primaryPlatform, &pattern.UpdatedAt)

	if err != nil {
		return nil, err
	}

	pattern.CommunicationMode = communicationMode.String
	pattern.PreferredTone = preferredTone.String
	pattern.AverageMessageLength = messageLength.String
	pattern.ResponseTimePreference = responseTime.String
	pattern.PrimaryPlatform = primaryPlatform.String

	return &pattern, nil
}

// Conversation methods
func (db *Database) createConversation(name, conversationType, purpose, notes string, contactIDs []int) (*Conversation, error) {
	now := time.Now()

	result, err := db.conn.Exec(`
		INSERT INTO conversations (name, type, purpose, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, name, conversationType, purpose, notes, now, now)

	if err != nil {
		return nil, fmt.Errorf("failed to create conversation: %v", err)
	}

	conversationID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	// Add members if provided
	for _, contactID := range contactIDs {
		_, err := db.conn.Exec(`
			INSERT INTO conversation_members (conversation_id, contact_id, added_at)
			VALUES (?, ?, ?)
		`, conversationID, contactID, now)

		if err != nil {
			return nil, fmt.Errorf("failed to add member to conversation: %v", err)
		}
	}

	return &Conversation{
		ID:        int(conversationID),
		Name:      name,
		Type:      conversationType,
		Purpose:   purpose,
		Notes:     notes,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (db *Database) getConversation(id int) (*Conversation, error) {
	var conv Conversation
	var name, purpose, notes sql.NullString

	err := db.conn.QueryRow(`
		SELECT id, name, type, purpose, notes, created_at, updated_at
		FROM conversations WHERE id = ?
	`, id).Scan(&conv.ID, &name, &conv.Type, &purpose, &notes, &conv.CreatedAt, &conv.UpdatedAt)

	if err != nil {
		return nil, err
	}

	conv.Name = name.String
	conv.Purpose = purpose.String
	conv.Notes = notes.String

	return &conv, nil
}

func (db *Database) getConversationMembers(conversationID int) ([]Contact, error) {
	rows, err := db.conn.Query(`
		SELECT c.id, c.name, c.relationship, c.platform, c.notes, c.communication_style,
		       c.interaction_count, c.last_interaction, c.created_at, c.updated_at
		FROM contacts c
		JOIN conversation_members cm ON c.id = cm.contact_id
		WHERE cm.conversation_id = ?
		ORDER BY cm.added_at
	`, conversationID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []Contact
	for rows.Next() {
		var contact Contact
		var lastInteraction sql.NullTime
		var relationship, platform, notes, communicationStyle sql.NullString

		err := rows.Scan(&contact.ID, &contact.Name, &relationship,
			&platform, &notes, &communicationStyle,
			&contact.InteractionCount, &lastInteraction, &contact.CreatedAt, &contact.UpdatedAt)

		if err != nil {
			return nil, err
		}

		contact.Relationship = relationship.String
		contact.Platform = platform.String
		contact.Notes = notes.String
		contact.CommunicationStyle = communicationStyle.String
		if lastInteraction.Valid {
			contact.LastInteraction = &lastInteraction.Time
		}

		contacts = append(contacts, contact)
	}

	return contacts, nil
}

func (db *Database) getConversationInteractions(conversationID int) ([]Interaction, error) {
	rows, err := db.conn.Query(`
		SELECT id, conversation_id, contact_id, date, platform, topic, sentiment, ai_summary, user_notes, important, context_metadata, created_at
		FROM interactions WHERE conversation_id = ? ORDER BY date DESC
	`, conversationID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var interactions []Interaction
	for rows.Next() {
		var interaction Interaction
		var contactID sql.NullInt64
		var platform, topic, sentiment, aiSummary, userNotes, contextMetadata sql.NullString

		err := rows.Scan(&interaction.ID, &interaction.ConversationID, &contactID, &interaction.Date,
			&platform, &topic, &sentiment,
			&aiSummary, &userNotes, &interaction.Important,
			&contextMetadata, &interaction.CreatedAt)

		if err != nil {
			return nil, err
		}

		if contactID.Valid {
			interaction.ContactID = int(contactID.Int64)
		}
		interaction.Platform = platform.String
		interaction.Topic = topic.String
		interaction.Sentiment = sentiment.String
		interaction.AISummary = aiSummary.String
		interaction.UserNotes = userNotes.String
		interaction.ContextMetadata = contextMetadata.String

		interactions = append(interactions, interaction)
	}

	return interactions, nil
}

func (db *Database) close() error {
	return db.conn.Close()
}
