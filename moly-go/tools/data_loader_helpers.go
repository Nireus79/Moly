package tools

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
)

// DataLoaderHelper loads current values from database for conflict comparison
type DataLoaderHelper struct {
	db *sql.DB
}

// NewDataLoaderHelper creates a new helper
func NewDataLoaderHelper(db *sql.DB) *DataLoaderHelper {
	return &DataLoaderHelper{db: db}
}

// LoadedValue represents a value loaded from database with its context
type LoadedValue struct {
	Value   interface{} // The actual value (string, []string, etc)
	Context string      // Context it was saved in (work, home, social, general)
	HasData bool        // Whether data was actually found
}

// LoadCurrentStyle loads user's saved communication style
// Returns the saved style value and its context, or empty if not found
func (h *DataLoaderHelper) LoadCurrentStyle(userID string) (*LoadedValue, error) {
	if h.db == nil {
		return &LoadedValue{HasData: false}, fmt.Errorf("database connection is nil")
	}

	var style string
	err := h.db.QueryRow(
		"SELECT COALESCE(communication_style, '') FROM about_me WHERE user_id = ?",
		userID,
	).Scan(&style)

	if err != nil && err != sql.ErrNoRows {
		log.Printf("[DataLoader] Error loading style: %v", err)
		return nil, err
	}

	if err == sql.ErrNoRows || style == "" {
		log.Printf("[DataLoader] No current style found for user %s", userID)
		return &LoadedValue{Value: "", Context: "general", HasData: false}, nil
	}

	log.Printf("[DataLoader] Loaded current style: '%s' for user %s", style, userID)
	return &LoadedValue{
		Value:   style,
		Context: "general", // about_me doesn't track context yet, so assume general
		HasData: true,
	}, nil
}

// LoadCurrentContactRelationship loads user's saved relationship for a contact
func (h *DataLoaderHelper) LoadCurrentContactRelationship(userID, contactName string) (*LoadedValue, error) {
	if h.db == nil {
		return &LoadedValue{HasData: false}, fmt.Errorf("database connection is nil")
	}

	var relationship string
	err := h.db.QueryRow(
		"SELECT COALESCE(relationship, '') FROM user_contacts WHERE user_id = ? AND name = ?",
		userID, contactName,
	).Scan(&relationship)

	if err != nil && err != sql.ErrNoRows {
		log.Printf("[DataLoader] Error loading contact relationship: %v", err)
		return nil, err
	}

	if err == sql.ErrNoRows || relationship == "" {
		log.Printf("[DataLoader] No current relationship found for contact %s (user %s)", contactName, userID)
		return &LoadedValue{Value: "", Context: "general", HasData: false}, nil
	}

	log.Printf("[DataLoader] Loaded current relationship: '%s' for contact %s", relationship, contactName)
	return &LoadedValue{
		Value:   relationship,
		Context: "general",
		HasData: true,
	}, nil
}

// LoadCurrentContactCharacteristics loads user's saved characteristics for a contact
func (h *DataLoaderHelper) LoadCurrentContactCharacteristics(userID, contactName string) (*LoadedValue, error) {
	if h.db == nil {
		return &LoadedValue{HasData: false}, fmt.Errorf("database connection is nil")
	}

	var charJSON string
	err := h.db.QueryRow(
		"SELECT COALESCE(characteristics, '[]') FROM user_contacts WHERE user_id = ? AND name = ?",
		userID, contactName,
	).Scan(&charJSON)

	if err != nil && err != sql.ErrNoRows {
		log.Printf("[DataLoader] Error loading contact characteristics: %v", err)
		return nil, err
	}

	if err == sql.ErrNoRows || charJSON == "[]" {
		log.Printf("[DataLoader] No current characteristics found for contact %s", contactName)
		return &LoadedValue{Value: []string{}, Context: "general", HasData: false}, nil
	}

	// Parse JSON array
	var chars []string
	if err := json.Unmarshal([]byte(charJSON), &chars); err != nil {
		log.Printf("[DataLoader] Error parsing characteristics JSON: %v", err)
		return &LoadedValue{Value: []string{}, Context: "general", HasData: false}, nil
	}

	log.Printf("[DataLoader] Loaded current characteristics: %v for contact %s", chars, contactName)
	return &LoadedValue{
		Value:   chars,
		Context: "general",
		HasData: true,
	}, nil
}

// LoadCurrentIntention loads user's most recent saved intention
func (h *DataLoaderHelper) LoadCurrentIntention(userID string) (*LoadedValue, error) {
	if h.db == nil {
		return &LoadedValue{HasData: false}, fmt.Errorf("database connection is nil")
	}

	var intention string
	var context string
	err := h.db.QueryRow(
		`SELECT COALESCE(fact_value, ''), COALESCE(context, 'general')
		 FROM context_attributes
		 WHERE user_id = ? AND fact_type = 'intention'
		 ORDER BY created_at DESC
		 LIMIT 1`,
		userID,
	).Scan(&intention, &context)

	if err != nil && err != sql.ErrNoRows {
		log.Printf("[DataLoader] Error loading intention: %v", err)
		return nil, err
	}

	if err == sql.ErrNoRows || intention == "" {
		log.Printf("[DataLoader] No current intention found for user %s", userID)
		return &LoadedValue{Value: "", Context: "general", HasData: false}, nil
	}

	log.Printf("[DataLoader] Loaded current intention: '%s' (context: %s)", intention, context)
	return &LoadedValue{
		Value:   intention,
		Context: context,
		HasData: true,
	}, nil
}

// LoadAllIntentionsForUser loads all saved intentions (for context-aware comparison)
// Returns map of context -> intention value
func (h *DataLoaderHelper) LoadAllIntentionsForUser(userID string) (map[string]string, error) {
	if h.db == nil {
		return map[string]string{}, fmt.Errorf("database connection is nil")
	}

	intentionsByContext := make(map[string]string)

	rows, err := h.db.Query(
		`SELECT COALESCE(context, 'general'), fact_value
		 FROM context_attributes
		 WHERE user_id = ? AND fact_type = 'intention'
		 ORDER BY created_at DESC`,
		userID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return intentionsByContext, nil
		}
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var context, value string
		if err := rows.Scan(&context, &value); err != nil {
			log.Printf("[DataLoader] Error scanning intention row: %v", err)
			continue
		}

		// Store by context (most recent per context)
		if _, exists := intentionsByContext[context]; !exists {
			intentionsByContext[context] = value
		}
	}

	if len(intentionsByContext) > 0 {
		log.Printf("[DataLoader] Loaded %d intentions across contexts for user %s", len(intentionsByContext), userID)
	}

	return intentionsByContext, rows.Err()
}
