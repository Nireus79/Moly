package tools

import (
	"database/sql"
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
		"SELECT COALESCE(relationship, '') FROM contacts WHERE user_id = ? AND name = ? AND name_status = 'named'",
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
		"SELECT COALESCE(characteristics, '[]') FROM contacts WHERE user_id = ? AND name = ? AND name_status = 'named'",
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
	if err := SafeJSONParse("DataLoaderHelpers", []byte(charJSON), &chars); err != nil {
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
