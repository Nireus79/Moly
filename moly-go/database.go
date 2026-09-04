package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
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
	LastInteraction      time.Time `json:"last_interaction"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type Interaction struct {
	ID                int       `json:"id"`
	ContactID         int       `json:"contact_id"`
	Date              time.Time `json:"date"`
	Platform          string    `json:"platform"`
	Topic             string    `json:"topic"`
	Sentiment         string    `json:"sentiment"`
	AISummary         string    `json:"ai_summary"`
	UserNotes         string    `json:"user_notes"`
	Important         bool      `json:"important"`
	ContextMetadata   string    `json:"context_metadata"`
	CreatedAt         time.Time `json:"created_at"`
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

func initDatabase() (*Database, error) {
	dbPath := filepath.Join(os.Getenv("HOME"), ".config", "moly", "moly.db")

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

	CREATE TABLE IF NOT EXISTS interactions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		contact_id INTEGER NOT NULL,
		date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		platform TEXT,
		topic TEXT,
		sentiment TEXT,
		ai_summary TEXT,
		user_notes TEXT,
		important BOOLEAN DEFAULT 0,
		context_metadata TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
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

	CREATE INDEX IF NOT EXISTS idx_interactions_contact ON interactions(contact_id);
	CREATE INDEX IF NOT EXISTS idx_interactions_date ON interactions(date);
	CREATE INDEX IF NOT EXISTS idx_contacts_platform ON contacts(platform);
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
	err := db.conn.QueryRow(`
		SELECT id, name, relationship, platform, notes, communication_style,
			   interaction_count, last_interaction, created_at, updated_at
		FROM contacts WHERE id = ?
	`, id).Scan(&contact.ID, &contact.Name, &contact.Relationship,
		&contact.Platform, &contact.Notes, &contact.CommunicationStyle,
		&contact.InteractionCount, &contact.LastInteraction, &contact.CreatedAt, &contact.UpdatedAt)

	if err != nil {
		return nil, err
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
		err := rows.Scan(&contact.ID, &contact.Name, &contact.Relationship,
			&contact.Platform, &contact.Notes, &contact.CommunicationStyle,
			&contact.InteractionCount, &contact.LastInteraction, &contact.CreatedAt, &contact.UpdatedAt)

		if err != nil {
			return nil, err
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
		err := rows.Scan(&interaction.ID, &interaction.ContactID, &interaction.Date,
			&interaction.Platform, &interaction.Topic, &interaction.Sentiment,
			&interaction.AISummary, &interaction.UserNotes, &interaction.Important,
			&interaction.ContextMetadata, &interaction.CreatedAt)

		if err != nil {
			return nil, err
		}
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
	err := db.conn.QueryRow(`
		SELECT id, communication_mode, preferred_tone, average_message_length,
			   response_time_preference, primary_platform, updated_at
		FROM behavior_patterns LIMIT 1
	`).Scan(&pattern.ID, &pattern.CommunicationMode, &pattern.PreferredTone,
		&pattern.AverageMessageLength, &pattern.ResponseTimePreference,
		&pattern.PrimaryPlatform, &pattern.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return &pattern, nil
}

func (db *Database) close() error {
	return db.conn.Close()
}
