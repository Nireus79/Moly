package services

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"moly/agents"
	"moly/database"
)

// ProfileUpdater applies extraction results to permanent user profile tables
// Handles confidence thresholds, merging, and reinforcement tracking
type ProfileUpdater struct {
	db                *database.Database
	confidenceThreshold float64 // Minimum confidence to update (default 0.6)
}

// NewProfileUpdater creates a new profile updater
func NewProfileUpdater(db *database.Database) *ProfileUpdater {
	return &ProfileUpdater{
		db:                  db,
		confidenceThreshold: 0.6,
	}
}

// SetConfidenceThreshold sets minimum confidence for updates
func (pu *ProfileUpdater) SetConfidenceThreshold(threshold float64) {
	if threshold < 0 {
		threshold = 0
	}
	if threshold > 1 {
		threshold = 1
	}
	pu.confidenceThreshold = threshold
}

// UpdateProfile applies all extracted insights to permanent notes
// Returns count of items updated
func (pu *ProfileUpdater) UpdateProfile(
	userID string,
	result *agents.ExtractionResult,
) (UpdateStats, error) {
	stats := UpdateStats{}

	if result == nil {
		return stats, fmt.Errorf("extraction result is nil")
	}

	if userID == "" {
		return stats, fmt.Errorf("userID cannot be empty")
	}

	log.Printf("[ProfileUpdater] Updating profile for user %s with %d AboutMe, %d patterns, %d contacts, %d goals",
		userID,
		len(result.AboutMeUpdates),
		len(result.PatternDetections),
		len(result.ContactMentions),
		len(result.GoalProgressUpdates))

	// Update each type of insight
	stats.AboutMeUpdated += pu.updateAboutMe(userID, result.AboutMeUpdates)
	stats.PatternsAdded += pu.updatePatterns(userID, result.PatternDetections)
	stats.ContactsAdded += pu.updateContacts(userID, result.ContactMentions)
	stats.GoalsUpdated += pu.updateGoals(userID, result.GoalProgressUpdates)
	stats.TotalUpdates = stats.AboutMeUpdated + stats.PatternsAdded + stats.ContactsAdded + stats.GoalsUpdated

	log.Printf("[ProfileUpdater] Profile update complete: %d AboutMe, %d patterns, %d contacts, %d goals",
		stats.AboutMeUpdated, stats.PatternsAdded, stats.ContactsAdded, stats.GoalsUpdated)

	return stats, nil
}

// UpdateStats tracks what was updated
type UpdateStats struct {
	AboutMeUpdated int
	PatternsAdded  int
	ContactsAdded  int
	GoalsUpdated   int
	TotalUpdates   int
}

// updateAboutMe applies AboutMe insights to about_me_profile table
func (pu *ProfileUpdater) updateAboutMe(userID string, updates []agents.AboutMeUpdate) int {
	count := 0
	for _, update := range updates {
		// Skip if below threshold
		if update.Confidence < pu.confidenceThreshold {
			log.Printf("[ProfileUpdater] Skipping AboutMe (low confidence): %s = %s (%.2f)",
				update.Key, update.Value, update.Confidence)
			continue
		}

		// Get existing profile
		existing, err := pu.getAboutMeProfile(userID)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("[ProfileUpdater] Error getting AboutMe profile: %v", err)
			continue
		}

		// Apply update (merge logic based on key)
		if err := pu.mergeAboutMeUpdate(userID, existing, update); err != nil {
			log.Printf("[ProfileUpdater] Error updating AboutMe: %v", err)
			continue
		}

		count++
	}
	return count
}

// getAboutMeProfile retrieves existing profile for a user
func (pu *ProfileUpdater) getAboutMeProfile(userID string) (map[string]interface{}, error) {
	query := `
		SELECT communication_style, tone_preference, core_values, preferences,
		       communication_notes, extracted_from_count, confidence
		FROM about_me_profile
		WHERE user_id = ?
	`

	row := pu.db.QueryRow(query, userID)
	var style, tone, valuesJSON, prefsJSON, notes sql.NullString
	var count int
	var conf float64

	err := row.Scan(&style, &tone, &valuesJSON, &prefsJSON, &notes, &count, &conf)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"communication_style":     style.String,
		"tone_preference":         tone.String,
		"core_values":             valuesJSON.String,
		"preferences":             prefsJSON.String,
		"communication_notes":     notes.String,
		"extracted_from_count":    count,
		"confidence":              conf,
	}, nil
}

// mergeAboutMeUpdate merges a single AboutMe insight
func (pu *ProfileUpdater) mergeAboutMeUpdate(
	userID string,
	existing map[string]interface{},
	update agents.AboutMeUpdate,
) error {
	now := time.Now().Unix()

	if existing == nil {
		// New profile - insert
		return pu.insertAboutMeProfile(userID, update, now)
	}

	// Update strategy based on key type
	switch update.Key {
	case "communication_style":
		return pu.updateAboutMeField(userID, "communication_style", update.Value, update.Confidence, now)
	case "tone_preference":
		return pu.updateAboutMeField(userID, "tone_preference", update.Value, update.Confidence, now)
	case "value":
		return pu.mergeValue(userID, update.Value, update.Confidence, now)
	case "preference":
		return pu.mergePreference(userID, update.Value, update.Confidence, now)
	default:
		// Generic field update
		return pu.updateAboutMeNotes(userID, update, now)
	}
}

// insertAboutMeProfile creates new AboutMe profile
func (pu *ProfileUpdater) insertAboutMeProfile(
	userID string,
	update agents.AboutMeUpdate,
	now int64,
) error {
	query := `
		INSERT INTO about_me_profile
		(user_id, communication_style, tone_preference, core_values, preferences,
		 communication_notes, extracted_from_count, confidence, last_updated, version)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 1)
	`

	var style, tone sql.NullString
	if update.Key == "communication_style" {
		style = sql.NullString{String: update.Value, Valid: true}
	}
	if update.Key == "tone_preference" {
		tone = sql.NullString{String: update.Value, Valid: true}
	}

	_, err := pu.db.Exec(query,
		userID, style, tone,
		nil, nil, // values, preferences
		fmt.Sprintf("Initialized: %s = %s", update.Key, update.Value),
		1, update.Confidence, now)

	return err
}

// updateAboutMeField updates a single field in AboutMe
func (pu *ProfileUpdater) updateAboutMeField(
	userID, field, value string,
	confidence float64,
	now int64,
) error {
	query := fmt.Sprintf(`
		UPDATE about_me_profile
		SET %s = ?, confidence = ?, extracted_from_count = extracted_from_count + 1, last_updated = ?
		WHERE user_id = ?
	`, field)

	_, err := pu.db.Exec(query, value, confidence, now, userID)
	return err
}

// updateAboutMeNotes appends to communication notes
func (pu *ProfileUpdater) updateAboutMeNotes(
	userID string,
	update agents.AboutMeUpdate,
	now int64,
) error {
	note := fmt.Sprintf("[%s] %s: %s (confidence: %.2f)",
		time.Unix(now, 0).Format("2006-01-02"),
		update.Key, update.Value, update.Confidence)

	query := `
		UPDATE about_me_profile
		SET communication_notes = COALESCE(communication_notes || E'\n', '') || ?,
		    extracted_from_count = extracted_from_count + 1,
		    last_updated = ?
		WHERE user_id = ?
	`

	_, err := pu.db.Exec(query, note, now, userID)
	return err
}

// mergeValue merges a core value (append to list)
func (pu *ProfileUpdater) mergeValue(userID, value string, confidence float64, now int64) error {
	// TODO: Implement JSON array merging for values
	// For now, just log it
	log.Printf("[ProfileUpdater] TODO: Merge value %s for user %s", value, userID)
	return nil
}

// mergePreference merges a preference
func (pu *ProfileUpdater) mergePreference(userID, preference string, confidence float64, now int64) error {
	// TODO: Implement JSON object merging for preferences
	// For now, just log it
	log.Printf("[ProfileUpdater] TODO: Merge preference %s for user %s", preference, userID)
	return nil
}

// updatePatterns adds or updates communication patterns
func (pu *ProfileUpdater) updatePatterns(userID string, detections []agents.PatternDetection) int {
	count := 0
	for _, detection := range detections {
		// Skip if below threshold
		if detection.Confidence < pu.confidenceThreshold {
			log.Printf("[ProfileUpdater] Skipping pattern (low confidence): %s (%.2f)",
				detection.Pattern, detection.Confidence)
			continue
		}

		// Check if pattern already exists
		existingID, err := pu.findPattern(userID, detection.Pattern)
		if err == nil && existingID > 0 {
			// Update existing pattern (increment observation count)
			pu.updateExistingPattern(existingID, detection)
		} else {
			// Create new pattern
			pu.createNewPattern(userID, detection)
		}
		count++
	}
	return count
}

// findPattern checks if pattern already exists
func (pu *ProfileUpdater) findPattern(userID, pattern string) (int, error) {
	query := `
		SELECT id FROM communication_patterns
		WHERE user_id = ? AND pattern = ?
	`
	var id int
	err := pu.db.QueryRow(query, userID, pattern).Scan(&id)
	return id, err
}

// updateExistingPattern increments observation count and updates confidence
func (pu *ProfileUpdater) updateExistingPattern(patternID int, detection agents.PatternDetection) error {
	query := `
		UPDATE communication_patterns
		SET observation_count = observation_count + 1,
		    confidence = (confidence + ?) / 2,
		    last_observed = ?,
		    context_notes = COALESCE(context_notes || E'\n', '') || ?
		WHERE id = ?
	`

	now := time.Now().Unix()
	_, err := pu.db.Exec(query, detection.Confidence, now, detection.Evidence, patternID)
	return err
}

// createNewPattern inserts a new pattern
func (pu *ProfileUpdater) createNewPattern(userID string, detection agents.PatternDetection) error {
	query := `
		INSERT INTO communication_patterns
		(user_id, pattern, pattern_category, confidence, first_observed, last_observed,
		 observation_count, is_active, is_growth_area, context_notes)
		VALUES (?, ?, ?, ?, ?, ?, 1, true, ?, ?)
	`

	now := time.Now().Unix()
	_, err := pu.db.Exec(query,
		userID, detection.Pattern, detection.Category, detection.Confidence,
		now, now, detection.IsGrowthArea, detection.Evidence)

	return err
}

// updateContacts adds or updates contacts and their communication patterns
func (pu *ProfileUpdater) updateContacts(userID string, mentions []agents.ContactMention) int {
	count := 0
	for _, mention := range mentions {
		// Skip if below threshold
		if mention.Confidence < pu.confidenceThreshold {
			log.Printf("[ProfileUpdater] Skipping contact (low confidence): %s (%.2f)",
				mention.Name, mention.Confidence)
			continue
		}

		// Find or create contact
		contactID, err := pu.findOrCreateContact(userID, mention)
		if err != nil {
			log.Printf("[ProfileUpdater] Error finding/creating contact: %v", err)
			continue
		}

		// Update contact communication patterns
		if err := pu.updateContactPatterns(contactID, mention); err != nil {
			log.Printf("[ProfileUpdater] Error updating contact patterns: %v", err)
			continue
		}

		count++
	}
	return count
}

// findOrCreateContact finds existing contact or creates new one
func (pu *ProfileUpdater) findOrCreateContact(
	userID string,
	mention agents.ContactMention,
) (int, error) {
	// Check if exists
	query := `SELECT id FROM contacts WHERE user_id = ? AND name = ?`
	var id int
	err := pu.db.QueryRow(query, userID, mention.Name).Scan(&id)

	if err == nil {
		// Update existing contact
		updateQuery := `
			UPDATE contacts
			SET last_mentioned = ?, times_mentioned = times_mentioned + 1
			WHERE id = ?
		`
		_, err := pu.db.Exec(updateQuery, time.Now().Unix(), id)
		return id, err
	}

	if err != sql.ErrNoRows {
		return 0, err
	}

	// Create new contact
	insertQuery := `
		INSERT INTO contacts
		(user_id, name, relationship_type, context, first_mentioned,
		 times_mentioned, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 1, ?, ?)
	`

	now := time.Now().Unix()
	result, err := pu.db.Exec(insertQuery,
		userID, mention.Name, mention.RelationshipType, mention.Context,
		now, now, now)

	if err != nil {
		return 0, err
	}

	// Get last insert ID
	lastID, err := result.LastInsertId()
	return int(lastID), err
}

// updateContactPatterns updates communication patterns for a contact
func (pu *ProfileUpdater) updateContactPatterns(contactID int, mention agents.ContactMention) error {
	query := `
		UPDATE contact_communication_patterns
		SET frequency = ?, tone_observed = ?, last_updated = ?, confidence = ?
		WHERE contact_id = ?
	`

	now := time.Now().Unix()
	_, err := pu.db.Exec(query,
		mention.Frequency, mention.ToneObserved, now, mention.Confidence, contactID)

	// If no rows updated, insert new pattern record
	if err == nil {
		return err
	}

	insertQuery := `
		INSERT INTO contact_communication_patterns
		(contact_id, frequency, tone_observed, last_updated, confidence)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err = pu.db.Exec(insertQuery,
		contactID, mention.Frequency, mention.ToneObserved, now, mention.Confidence)

	return err
}

// updateGoals tracks progress on active goals
func (pu *ProfileUpdater) updateGoals(userID string, updates []agents.GoalProgressUpdate) int {
	count := 0
	for _, update := range updates {
		// Skip if below threshold
		if update.Confidence < pu.confidenceThreshold {
			log.Printf("[ProfileUpdater] Skipping goal (low confidence): %s (%.2f)",
				update.GoalDescription, update.Confidence)
			continue
		}

		// Try to link to existing active goal
		// For now, just log it - linking requires semantic matching
		log.Printf("[ProfileUpdater] TODO: Match goal '%s' to existing goals", update.GoalDescription)
		count++
	}
	return count
}

// AddImplicitLearning adds a learning that system inferred (not explicitly stated)
func (pu *ProfileUpdater) AddImplicitLearning(
	userID string,
	learningKey string,
	learningValue string,
	learningType string,
	confidence float64,
	source string,
) error {
	query := `
		INSERT INTO implicit_learning
		(user_id, learning_type, learning_key, learning_value,
		 confidence, source, first_extracted, reinforcement_count)
		VALUES (?, ?, ?, ?, ?, ?, ?, 1)
	`

	now := time.Now().Unix()
	_, err := pu.db.Exec(query,
		userID, learningType, learningKey, learningValue, confidence, source, now)

	return err
}
