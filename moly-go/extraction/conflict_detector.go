package extraction

import (
	"log"

	"moly/models"
)

// Conflict - Detected conflict between stored and extracted context
type Conflict struct {
	Type           string
	Field          string
	StoredValue    string
	ExtractedValue string
	Context        string
	Severity       string
}

// ConflictDetector - Detects conflicts between stored and extracted context
type ConflictDetector struct {
}

// NewConflictDetector - Create new detector
func NewConflictDetector() *ConflictDetector {
	return &ConflictDetector{}
}

// Detect - Find conflicts in extracted context vs stored aboutMe
func (cd *ConflictDetector) Detect(extraction *ExtractedContext, userAboutMe *models.AboutMe) []Conflict {
	log.Printf("[ConflictDetector] Checking for conflicts: stored_style=%s extracted_style=%s",
		userAboutMe.CommunicationStyle, extraction.Style)

	var conflicts []Conflict

	// Check 1: Communication style conflict
	if extraction.Style != "" && extraction.Style != "neutral" &&
		userAboutMe.CommunicationStyle != "" &&
		extraction.Style != userAboutMe.CommunicationStyle {

		conflicts = append(conflicts, Conflict{
			Type:           "style_conflict",
			Field:          "communication_style",
			StoredValue:    userAboutMe.CommunicationStyle,
			ExtractedValue: extraction.Style,
			Context:        "User's communication style differs from stored preference",
			Severity:       "medium",
		})

		log.Printf("[ConflictDetector] ✓ Style conflict: %s vs %s", userAboutMe.CommunicationStyle, extraction.Style)
	}

	// Check 2: Emotional tone change (tracked for understanding, not conflict)
	// This is informational, not a conflict requiring resolution

	// Check 3: Topic/goal changes (informational)
	// These are tracked as insights, not conflicts

	log.Printf("[ConflictDetector] Found %d conflicts", len(conflicts))
	return conflicts
}

// HasSignificantConflict - Check if conflicts need user input
func (cd *ConflictDetector) HasSignificantConflict(conflicts []Conflict) bool {
	for _, c := range conflicts {
		if c.Type == "style_conflict" {
			return true
		}
	}
	return false
}

// ConflictToQuestion - Convert conflict to clarifying question
func (cd *ConflictDetector) ConflictToQuestion(conflict Conflict) string {
	switch conflict.Type {
	case "style_conflict":
		return "Earlier you mentioned preferring a " + conflict.StoredValue + " style, but now you seem to want a " + conflict.ExtractedValue + " approach. Are these for different situations, or has your preference changed?"

	default:
		return "I notice something different about your preferences. Could you help me understand?"
	}
}
