package database

import (
	"database/sql"
	"encoding/json"
	"log"
	"time"

	"moly/models"
)

// AboutMeRepository - Manages AboutMe records
type AboutMeRepository struct {
	db *Database
}

// NewAboutMeRepository - Create new repository
func NewAboutMeRepository(db *Database) *AboutMeRepository {
	return &AboutMeRepository{db: db}
}

// Save - Save or update AboutMe
func (r *AboutMeRepository) Save(userID string, aboutMe *models.AboutMe) error {
	log.Printf("[Repository] Saving AboutMe for user %s (style=%s values=%d)", userID, aboutMe.CommunicationStyle, len(aboutMe.Values))

	valuesJSON, _ := json.Marshal(aboutMe.Values)
	now := time.Now().Unix()

	query := `
		INSERT INTO about_me (user_id, communication_style, "values", preferred_tone, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			communication_style = excluded.communication_style,
			"values" = excluded."values",
			preferred_tone = excluded.preferred_tone,
			notes = excluded.notes,
			updated_at = excluded.updated_at,
			version = version + 1
	`

	_, err := r.db.Exec(query, userID, aboutMe.CommunicationStyle, string(valuesJSON), aboutMe.PreferredTone, aboutMe.Notes, now, now)
	if err != nil {
		log.Printf("[Repository] ERROR saving AboutMe: %v", err)
	} else {
		log.Printf("[Repository] AboutMe saved successfully")
	}
	return err
}

// Get - Get AboutMe for user
func (r *AboutMeRepository) Get(userID string) (*models.AboutMe, error) {
	query := `SELECT communication_style, "values", preferred_tone, notes, created_at, updated_at FROM about_me WHERE user_id = ?`

	aboutMe := &models.AboutMe{UserID: userID}
	var valuesJSON sql.NullString

	err := r.db.QueryRow(query, userID).Scan(&aboutMe.CommunicationStyle, &valuesJSON, &aboutMe.PreferredTone, &aboutMe.Notes, &aboutMe.CreatedAt, &aboutMe.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found is not an error
		}
		return nil, err
	}

	if valuesJSON.Valid {
		_ = json.Unmarshal([]byte(valuesJSON.String), &aboutMe.Values)
	}

	return aboutMe, nil
}

// ContactRepository - Manages Contact records
type ContactRepository struct {
	db *Database
}

// NewContactRepository - Create new repository
func NewContactRepository(db *Database) *ContactRepository {
	return &ContactRepository{db: db}
}

// Save - Save or update Contact
func (r *ContactRepository) Save(userID string, contact *models.Contact) error {
	log.Printf("[Repository] Saving contact for user %s (name=%s relationship=%s)", userID, contact.Name, contact.Relationship)

	charJSON, _ := json.Marshal(contact.Characteristics)
	now := time.Now().Unix()

	query := `
		INSERT INTO contacts (user_id, name, relationship, characteristics, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, name) DO UPDATE SET
			relationship = excluded.relationship,
			characteristics = excluded.characteristics,
			updated_at = excluded.updated_at
	`

	_, err := r.db.Exec(query, userID, contact.Name, contact.Relationship, string(charJSON), now, now)
	if err != nil {
		log.Printf("[Repository] ERROR saving contact: %v", err)
	} else {
		log.Printf("[Repository] Contact saved successfully")
	}
	return err
}

// GetByName - Get contact by name
func (r *ContactRepository) GetByName(userID string, name string) (*models.Contact, error) {
	query := `SELECT name, relationship, characteristics FROM contacts WHERE user_id = ? AND name = ?`

	contact := &models.Contact{UserID: userID}
	var charJSON sql.NullString

	err := r.db.QueryRow(query, userID, name).Scan(&contact.Name, &contact.Relationship, &charJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if charJSON.Valid {
		_ = json.Unmarshal([]byte(charJSON.String), &contact.Characteristics)
	}

	return contact, nil
}

// GetAll - Get all contacts for user
func (r *ContactRepository) GetAll(userID string) ([]models.Contact, error) {
	query := `SELECT name, relationship, characteristics FROM contacts WHERE user_id = ? ORDER BY created_at DESC`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []models.Contact
	for rows.Next() {
		contact := models.Contact{UserID: userID}
		var charJSON sql.NullString

		if err := rows.Scan(&contact.Name, &contact.Relationship, &charJSON); err != nil {
			return nil, err
		}

		if charJSON.Valid {
			_ = json.Unmarshal([]byte(charJSON.String), &contact.Characteristics)
		}

		contacts = append(contacts, contact)
	}

	return contacts, rows.Err()
}

// InteractionRepository - Manages Interaction records
type InteractionRepository struct {
	db *Database
}

// NewInteractionRepository - Create new repository
func NewInteractionRepository(db *Database) *InteractionRepository {
	return &InteractionRepository{db: db}
}

// Save - Save interaction
func (r *InteractionRepository) Save(userID string, conversationID string, content string, interactionType string, metadata map[string]interface{}) error {
	log.Printf("[Repository] Saving interaction for user %s (conv=%s type=%s content_len=%d)", userID, conversationID, interactionType, len(content))

	metadataJSON, _ := json.Marshal(metadata)
	now := time.Now().Unix()

	query := `
		INSERT INTO interactions (user_id, conversation_id, content, type, timestamp, metadata)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.Exec(query, userID, conversationID, content, interactionType, now, string(metadataJSON))
	if err != nil {
		log.Printf("[Repository] ERROR saving interaction: %v", err)
	} else {
		log.Printf("[Repository] Interaction saved successfully")
	}
	return err
}

// GetConversation - Get all interactions in a conversation
func (r *InteractionRepository) GetConversation(conversationID string, limit int) ([]models.Message, error) {
	if limit == 0 {
		limit = 50
	}

	query := `
		SELECT content, type, timestamp
		FROM interactions
		WHERE conversation_id = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := r.db.Query(query, conversationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		msg := models.Message{}
		var msgType sql.NullString

		if err := rows.Scan(&msg.Content, &msgType, &msg.Timestamp); err != nil {
			return nil, err
		}

		if msgType.Valid {
			msg.Type = msgType.String
		}

		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

// BehaviorPatternRepository - Manages behavior pattern records
type BehaviorPatternRepository struct {
	db *Database
}

// NewBehaviorPatternRepository - Create new repository
func NewBehaviorPatternRepository(db *Database) *BehaviorPatternRepository {
	return &BehaviorPatternRepository{db: db}
}

// Save - Save behavior pattern
func (r *BehaviorPatternRepository) Save(userID string, profile *models.UserBehavioralProfile) error {
	tonesJSON, _ := json.Marshal(profile.CommunicationGoals)

	query := `
		INSERT INTO behavior_patterns (user_id, total_interactions, modification_rate, preferred_tones, communication_style, growth_trend, confidence, last_analyzed)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			total_interactions = excluded.total_interactions,
			modification_rate = excluded.modification_rate,
			preferred_tones = excluded.preferred_tones,
			communication_style = excluded.communication_style,
			growth_trend = excluded.growth_trend,
			confidence = excluded.confidence,
			last_analyzed = excluded.last_analyzed
	`

	_, err := r.db.Exec(query, userID, 0, 0.0, string(tonesJSON), "", "stable", profile.Confidence, time.Now().Unix())
	return err
}

// Get - Get behavior pattern for user
func (r *BehaviorPatternRepository) Get(userID string) (*models.UserBehavioralProfile, error) {
	query := `SELECT total_interactions, modification_rate, preferred_tones, communication_style, growth_trend, confidence FROM behavior_patterns WHERE user_id = ?`

	profile := &models.UserBehavioralProfile{
		CommunicationProfile: make(map[string]interface{}),
		CommunicationGoals:   make(map[string]int),
		SuggestionChoices:    make(map[string]interface{}),
		SuccessMetrics:       make(map[string]interface{}),
		GrowthTrajectory:     make(map[string]interface{}),
	}

	var (
		totalInteractions sql.NullInt64
		modificationRate  sql.NullFloat64
		tonesJSON         sql.NullString
		commStyle         sql.NullString
		growthTrend       sql.NullString
	)

	err := r.db.QueryRow(query, userID).Scan(&totalInteractions, &modificationRate, &tonesJSON, &commStyle, &growthTrend, &profile.Confidence)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Parse JSON fields
	if tonesJSON.Valid {
		_ = json.Unmarshal([]byte(tonesJSON.String), &profile.CommunicationGoals)
	}
	if commStyle.Valid {
		profile.CommunicationProfile["style"] = commStyle.String
	}
	if growthTrend.Valid {
		profile.GrowthTrajectory["trend"] = growthTrend.String
	}

	return profile, nil
}

// ReflectionRepository - Manages Reflection records
type ReflectionRepository struct {
	db *Database
}

// NewReflectionRepository - Create new repository
func NewReflectionRepository(db *Database) *ReflectionRepository {
	return &ReflectionRepository{db: db}
}

// Save - Save reflection
func (r *ReflectionRepository) Save(userID string, reflection *models.Reflection) error {
	charJSON, _ := json.Marshal(reflection.Characteristics)
	interJSON, _ := json.Marshal(reflection.Intentions)
	now := time.Now().Unix()

	query := `
		INSERT INTO reflections (user_id, characteristics, interests, intentions, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.Exec(query, userID, string(charJSON), "", string(interJSON), "pending_approval", now)
	return err
}

// GetPendingApprovals - Get reflections awaiting approval
func (r *ReflectionRepository) GetPendingApprovals(userID string) ([]models.Reflection, error) {
	query := `SELECT characteristics, interests, intentions FROM reflections WHERE user_id = ? AND status = 'pending_approval' ORDER BY created_at DESC`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reflections []models.Reflection
	for rows.Next() {
		r := models.Reflection{}
		var charJSON, interestsJSON, interJSON sql.NullString

		if err := rows.Scan(&charJSON, &interestsJSON, &interJSON); err != nil {
			return nil, err
		}

		if charJSON.Valid {
			_ = json.Unmarshal([]byte(charJSON.String), &r.Characteristics)
		}
		if interJSON.Valid {
			_ = json.Unmarshal([]byte(interJSON.String), &r.Intentions)
		}

		reflections = append(reflections, r)
	}

	return reflections, rows.Err()
}

// Approve - Approve a reflection
func (r *ReflectionRepository) Approve(reflectionID int) error {
	query := `UPDATE reflections SET status = 'approved', approved_at = ? WHERE id = ?`
	_, err := r.db.Exec(query, time.Now().Unix(), reflectionID)
	return err
}

// SuggestionChoiceRepository - Manages suggestion choice tracking
type SuggestionChoiceRepository struct {
	db *Database
}

// NewSuggestionChoiceRepository - Create new repository
func NewSuggestionChoiceRepository(db *Database) *SuggestionChoiceRepository {
	return &SuggestionChoiceRepository{db: db}
}

// Record - Record a suggestion choice
func (r *SuggestionChoiceRepository) Record(userID string, suggestionID string, suggestedText string, userModification string) error {
	query := `
		INSERT INTO suggestion_choices (user_id, suggestion_id, suggested_text, user_modification, chosen_at)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err := r.db.Exec(query, userID, suggestionID, suggestedText, userModification, time.Now().Unix())
	return err
}

// GetUserChoices - Get all choices for a user
func (r *SuggestionChoiceRepository) GetUserChoices(userID string, limit int) ([]map[string]interface{}, error) {
	if limit == 0 {
		limit = 100
	}

	query := `SELECT suggestion_id, suggested_text, user_modification, chosen_at FROM suggestion_choices WHERE user_id = ? ORDER BY chosen_at DESC LIMIT ?`

	rows, err := r.db.Query(query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var choices []map[string]interface{}
	for rows.Next() {
		var suggestionID, suggestedText, userMod sql.NullString
		var chosenAt int64

		if err := rows.Scan(&suggestionID, &suggestedText, &userMod, &chosenAt); err != nil {
			return nil, err
		}

		choice := map[string]interface{}{
			"suggestion_id":     suggestionID.String,
			"suggested_text":    suggestedText.String,
			"user_modification": userMod.String,
			"chosen_at":         chosenAt,
		}
		choices = append(choices, choice)
	}

	return choices, rows.Err()
}

// SafetyIncidentRepository - Manages safety incident tracking
type SafetyIncidentRepository struct {
	db *Database
}

// NewSafetyIncidentRepository - Create new repository
func NewSafetyIncidentRepository(db *Database) *SafetyIncidentRepository {
	return &SafetyIncidentRepository{db: db}
}

// Record - Record a safety incident
func (r *SafetyIncidentRepository) Record(userID string, severity string, content string, detectedBy string) error {
	query := `
		INSERT INTO safety_incidents (user_id, severity, detected_at, content, detected_by)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err := r.db.Exec(query, userID, severity, time.Now().Unix(), content, detectedBy)
	return err
}

// GetRecentIncidents - Get recent incidents for a user
func (r *SafetyIncidentRepository) GetRecentIncidents(userID string, limit int) ([]map[string]interface{}, error) {
	if limit == 0 {
		limit = 20
	}

	query := `SELECT severity, content, detected_at, detected_by FROM safety_incidents WHERE user_id = ? ORDER BY detected_at DESC LIMIT ?`

	rows, err := r.db.Query(query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var incidents []map[string]interface{}
	for rows.Next() {
		var severity, content, detectedBy sql.NullString
		var detectedAt int64

		if err := rows.Scan(&severity, &content, &detectedAt, &detectedBy); err != nil {
			return nil, err
		}

		incident := map[string]interface{}{
			"severity":    severity.String,
			"content":     content.String,
			"detected_at": detectedAt,
			"detected_by": detectedBy.String,
		}
		incidents = append(incidents, incident)
	}

	return incidents, rows.Err()
}
