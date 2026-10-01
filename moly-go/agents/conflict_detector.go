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
	database        *database.Database
	contactRepo     *database.ContactRepository
	contextAttrRepo *database.ContextAttributeRepository
	antonymMap      map[string]string // Maps characteristics to their antonyms
}

// AntonymMap defines opposing characteristics
// Used to detect when user says something contradicting their previous statement
var defaultAntonymMap = map[string]string{
	"dominant":      "submissive",
	"submissive":    "dominant",
	"assertive":     "passive",
	"passive":       "assertive",
	"independent":   "dependent",
	"dependent":     "independent",
	"outgoing":      "introverted",
	"introverted":   "outgoing",
	"ambitious":     "content",
	"content":       "ambitious",
	"adventurous":   "cautious",
	"cautious":      "adventurous",
	"romantic":      "pragmatic",
	"pragmatic":     "romantic",
	"spontaneous":   "planned",
	"planned":       "spontaneous",
	"emotional":     "logical",
	"logical":       "emotional",
	"flexible":      "rigid",
	"rigid":         "flexible",
	"generous":      "frugal",
	"frugal":        "generous",
}

// NewConflictDetector creates a new conflict detector
func NewConflictDetector(db *database.Database) *ConflictDetector {
	return &ConflictDetector{
		database:        db,
		contactRepo:     database.NewContactRepository(db),
		contextAttrRepo: database.NewContextAttributeRepository(db),
		antonymMap:      defaultAntonymMap,
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

	// PHASE 0: Check for INTERNAL contradictions (NEW FIX #3)
	internalConflicts := cd.detectInternalContradictions(entities)
	if len(internalConflicts) > 0 {
		log.Printf("[ConflictDetector] ✓ Detected %d INTERNAL contradictions within this extraction", len(internalConflicts))
		conflicts = append(conflicts, internalConflicts...)
	}

	// PHASE 1: Check against database
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

		case "preference":
			// Check for preference conflicts
			prefConflict := cd.detectPreferenceConflict(ctx, userID, entity)
			if prefConflict != nil {
				conflicts = append(conflicts, *prefConflict)
			}

		case "characteristic":
			// PHASE 2: Check for characteristic conflicts using antonym mapping
			charConflict := cd.detectCharacteristicConflict(ctx, userID, entity)
			if charConflict != nil {
				conflicts = append(conflicts, *charConflict)
			} else {
				// Also check regular preference conflicts as fallback
				prefConflict := cd.detectPreferenceConflict(ctx, userID, entity)
				if prefConflict != nil {
					conflicts = append(conflicts, *prefConflict)
				}
			}
		}
	}

	if len(conflicts) > 0 {
		log.Printf("[ConflictDetector] ✓ Detected %d total conflicts (including internal)", len(conflicts))
	}

	return conflicts, nil
}

// detectInternalContradictions finds contradictions WITHIN the extracted entities (NEW FIX #3)
// For example: same subject saying they're both "dominant" AND "submissive"
func (cd *ConflictDetector) detectInternalContradictions(entities []models.ExtractedEntity) []ConflictDetectorResult {
	var conflicts []ConflictDetectorResult

	// Compare each entity against every other entity
	for i := 0; i < len(entities); i++ {
		for j := i + 1; j < len(entities); j++ {
			ent1 := entities[i]
			ent2 := entities[j]

			// Only check characteristics for antonym conflicts
			if ent1.Type != "characteristic" || ent2.Type != "characteristic" {
				continue
			}

			// Only flag if same subject
			if ent1.Subject != ent2.Subject {
				continue
			}

			// Check if they're antonyms (opposites)
			antonym1, hasAntonym1 := cd.antonymMap[ent1.Value]
			if !hasAntonym1 {
				continue
			}

			if antonym1 == ent2.Value && ent1.Confidence >= 0.6 && ent2.Confidence >= 0.6 {
				conflict := ConflictDetectorResult{
					Entity:        ent1,
					ExistingValue: ent2.Value,
					Type:          "internal_contradiction",
					Severity:      "critical", // High severity - within same extraction
					Confidence:    (ent1.Confidence + ent2.Confidence) / 2,
					Resolution:    "ask_clarification",
					Description: fmt.Sprintf(
						"Internal contradiction: %s says they're both '%s' AND '%s' - which one is true?",
						ent1.Subject, ent1.Value, ent2.Value,
					),
				}
				conflicts = append(conflicts, conflict)
				log.Printf("[ConflictDetector] ⚠ INTERNAL CONTRADICTION: %s has %s but also %s",
					ent1.Subject, ent1.Value, ent2.Value)
			}
		}
	}

	return conflicts
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

// detectCharacteristicConflict checks if a characteristic contradicts previous statements
// using antonym mapping (e.g., "dominant" vs "submissive")
// PHASE 2: Detects when user describes themselves with opposite characteristics
func (cd *ConflictDetector) detectCharacteristicConflict(
	ctx context.Context,
	userID string,
	extracted models.ExtractedEntity,
) *ConflictDetectorResult {

	// Only check characteristics
	if extracted.Type != "characteristic" {
		return nil
	}

	// Check if this characteristic has a known antonym
	antonym, hasAntonym := cd.antonymMap[extracted.Value]
	if !hasAntonym {
		// No antonym mapping, cannot detect conflict this way
		return nil
	}

	// Look for existing characteristic in database
	conn := cd.database.GetConnection()
	if conn == nil {
		return nil
	}

	var existingValue string
	var existingSubject string
	err := conn.QueryRow(
		`SELECT value, attributed_to FROM context_attributes
		 WHERE user_id = ? AND type = 'characteristic'
		 AND value IN (?, ?)
		 ORDER BY created_at DESC LIMIT 1`,
		userID,
		extracted.Value,
		antonym,
	).Scan(&existingValue, &existingSubject)

	if err != nil {
		// No existing characteristic, not a conflict
		return nil
	}

	// Check if existing value is the antonym
	if existingValue == antonym && existingSubject == extracted.Subject {
		log.Printf("[ConflictDetector] ⚠ Characteristic contradiction: '%s' vs '%s' (antonym)",
			extracted.Value, existingValue)

		return &ConflictDetectorResult{
			Entity:        extracted,
			ExistingValue: existingValue,
			Type:          "characteristic_conflict",
			Severity:      "high", // High severity - fundamental contradiction
			Confidence:    extracted.Confidence * 0.95,
			Resolution:    "ask_clarification",
			Description:   fmt.Sprintf(
				"You described yourself as '%s' before, but now you're saying '%s' - these are opposite characteristics",
				existingValue,
				extracted.Value,
			),
		}
	}

	return nil
}

// IsCharacteristicAntonym checks if two characteristics are antonyms
func (cd *ConflictDetector) IsCharacteristicAntonym(char1, char2 string) bool {
	return cd.antonymMap[char1] == char2 || cd.antonymMap[char2] == char1
}

// GetAntonym returns the antonym of a characteristic, or empty string if none exists
func (cd *ConflictDetector) GetAntonym(characteristic string) string {
	if antonym, exists := cd.antonymMap[characteristic]; exists {
		return antonym
	}
	return ""
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
