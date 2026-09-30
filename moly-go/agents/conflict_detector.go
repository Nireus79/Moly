package agents

import (
	"context"
	"fmt"
	"log"

	"moly/database"
	"moly/models"
)

// ConflictDetectorResult represents a detected conflict
type ConflictDetectorResult struct {
	Entity      models.ExtractedEntity // What was extracted
	ExistingValue interface{}           // What's in database
	Type        string                  // "subject_mismatch", "value_contradiction", "new_entity"
	Severity    string                  // "high", "medium", "low"
	Confidence  float64                 // How confident we are in the conflict
	Resolution  string                  // "ask_clarification", "update", "ignore"
	Description string                  // Human-readable description
}

// ConflictDetector: Detects conflicts between extracted entities and database
type ConflictDetector struct {
	database      *database.Database
	contactRepo   *database.ContactRepository
	contextAttrRepo *database.ContextAttributeRepository
}

// NewConflictDetector creates a new conflict detector
func NewConflictDetector(db *database.Database) *ConflictDetector {
	return &ConflictDetector{
		database:        db,
		contactRepo:     database.NewContactRepository(db),
		contextAttrRepo: database.NewContextAttributeRepository(db),
	}
}

// DetectConflicts checks extracted entities against database for conflicts
func (cd *ConflictDetector) DetectConflicts(
	ctx context.Context,
	userID, conversationID string,
	entities []models.ExtractedEntity,
) ([]ConflictDetectorResult, error) {

	var conflicts []ConflictDetectorResult

	log.Printf("[ConflictDetector] Checking %d extracted entities for conflicts", len(entities))

	for _, entity := range entities {
		// Skip low-confidence entities
		if entity.Confidence < 0.60 {
			continue
		}

		switch entity.Type {
		case "contact":
			// Check for contact subject mismatches
			contactConflict := cd.detectContactConflict(ctx, userID, entity)
			if contactConflict != nil {
				conflicts = append(conflicts, *contactConflict)
			}

		case "preference", "characteristic":
			// Check for preference/characteristic conflicts
			prefConflict := cd.detectPreferenceConflict(ctx, userID, entity)
			if prefConflict != nil {
				conflicts = append(conflicts, *prefConflict)
			}
		}
	}

	if len(conflicts) > 0 {
		log.Printf("[ConflictDetector] ✓ Detected %d conflicts", len(conflicts))
	}

	return conflicts, nil
}

// detectContactConflict checks if a contact's subject has changed
func (cd *ConflictDetector) detectContactConflict(
	ctx context.Context,
	userID string,
	extracted models.ExtractedEntity,
) *ConflictDetectorResult {

	// Look for existing contact in database
	existing, err := cd.contactRepo.GetByName(userID, extracted.Value)
	if err != nil || existing == nil {
		// No existing contact, not a conflict
		return nil
	}

	// Check if subject (she/he/user) has changed
	// This is the multi-person boundary case
	// Note: Contact.Type in database is the relationship type, not subject
	// We check if this extracted entity has a different subject than before
	if extracted.Subject != "" && extracted.Subject != "user" {
		// If subject is explicitly not "user", check for mismatch
		log.Printf("[ConflictDetector] ⚠ Subject marked as non-user for contact %s: %s",
			extracted.Value, extracted.Subject)

		return &ConflictDetectorResult{
			Entity:        extracted,
			ExistingValue: existing,
			Type:          "subject_mismatch",
			Severity:      "medium",
			Confidence:    extracted.Confidence,
			Resolution:    "ask_clarification",
			Description:   fmt.Sprintf("Contact '%s' is described with subject '%s' - needs clarification", extracted.Value, extracted.Subject),
		}
	}

	return nil
}

// detectPreferenceConflict checks if a preference value has changed significantly
func (cd *ConflictDetector) detectPreferenceConflict(
	ctx context.Context,
	userID string,
	extracted models.ExtractedEntity,
) *ConflictDetectorResult {

	// Look for existing preference in context attributes
	conn := cd.database.GetConnection()
	if conn == nil {
		return nil
	}

	var existingValue string
	var existingSubject string
	err := conn.QueryRow(
		`SELECT value, attributed_to FROM context_attributes
		 WHERE user_id = ? AND type = ?
		 ORDER BY created_at DESC LIMIT 1`,
		userID,
		extracted.Type,
	).Scan(&existingValue, &existingSubject)

	if err != nil {
		// No existing preference, not a conflict
		return nil
	}

	// Check for value contradiction
	if existingValue != extracted.Value && existingSubject == extracted.Subject {
		log.Printf("[ConflictDetector] ⚠ Preference contradiction for %s: '%s' → '%s'",
			extracted.Type, existingValue, extracted.Value)

		return &ConflictDetectorResult{
			Entity:         extracted,
			ExistingValue:  existingValue,
			Type:           "value_contradiction",
			Severity:       "medium",
			Confidence:     extracted.Confidence * 0.9, // Lower confidence for contradictions
			Resolution:     "ask_clarification",
			Description:    fmt.Sprintf("You previously said your %s was '%s' but now you say it's '%s'", extracted.Type, existingValue, extracted.Value),
		}
	}

	// Check for subject change (multi-person context)
	if extracted.Subject != "" && existingSubject != "" && existingSubject != extracted.Subject {
		log.Printf("[ConflictDetector] ⚠ Subject change for %s: %s → %s",
			extracted.Type, existingSubject, extracted.Subject)

		return &ConflictDetectorResult{
			Entity:         extracted,
			ExistingValue:  existingValue,
			Type:           "subject_mismatch",
			Severity:       "low",
			Confidence:     extracted.Confidence * 0.8,
			Resolution:     "ask_clarification",
			Description:    fmt.Sprintf("The preference '%s' was previously about '%s' but now about '%s'", extracted.Value, existingSubject, extracted.Subject),
		}
	}

	return nil
}
