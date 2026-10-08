package agents

import (
	"fmt"
	"log"
	"moly/database"
	"moly/models"
	"regexp"
	"strings"
)

// ProgressiveNamingDetector finds and applies name updates from user messages
type ProgressiveNamingDetector struct {
	db *database.Database
}

// NewProgressiveNamingDetector creates a new detector
func NewProgressiveNamingDetector(db *database.Database) *ProgressiveNamingDetector {
	return &ProgressiveNamingDetector{
		db: db,
	}
}

// NamingUpdate represents a detected name update
type NamingUpdate struct {
	ContactID   int64
	ContactName string
	NewName     string
	Confidence  float64
	Pattern     string // Which pattern was matched
}

// DetectNamingPatterns finds naming patterns in the message
// Returns a list of contacts that should be renamed
func (pnd *ProgressiveNamingDetector) DetectNamingPatterns(
	message string,
	activeContacts []*models.Contact,
) []*NamingUpdate {
	if message == "" || len(activeContacts) == 0 {
		return nil
	}

	updates := make([]*NamingUpdate, 0)

	// Pattern 1: "name is X" or "name's X"
	// Examples: "her name is Emily", "his name is Marcus", "their name is Alex"
	nameIsPattern := regexp.MustCompile(`(?:(?:her|his|their|my|your|its)\s+)?name\s+(?:is|'s|as)\s+([A-Z][a-z]+(?:\s+[A-Z][a-z]+)?)`)
	matches := nameIsPattern.FindAllStringSubmatchIndex(message, -1)
	for _, match := range matches {
		nameStart := match[2]
		nameEnd := match[3]
		if nameStart < len(message) && nameEnd <= len(message) {
			newName := strings.TrimSpace(message[nameStart:nameEnd])

			// Find which contact this applies to (based on preceding pronoun)
			precedingText := message[:nameStart]
			contact := findContactByPronoun(precedingText, activeContacts)
			if contact != nil {
				updates = append(updates, &NamingUpdate{
					ContactID:   contact.ID,
					ContactName: contact.Name,
					NewName:     newName,
					Confidence:  0.95, // High confidence for explicit naming
					Pattern:     "name_is",
				})
				log.Printf("[ProgressiveNaming] Pattern 'name is X': %s → %s (contact ID: %d)", contact.Name, newName, contact.ID)
			}
		}
	}

	// Pattern 2: "girlfriend/boyfriend/colleague is X"
	// Examples: "My girlfriend is Emily", "The colleague is Marcus"
	relationshipPattern := regexp.MustCompile(`(?:my\s+|the\s+)?(?:girlfriend|boyfriend|colleague|friend|coworker|sister|brother|boss|mentor)\s+(?:is|named)\s+([A-Z][a-z]+(?:\s+[A-Z][a-z]+)?)`)
	matches = relationshipPattern.FindAllStringSubmatchIndex(message, -1)
	for _, match := range matches {
		nameStart := match[2]
		nameEnd := match[3]
		if nameStart < len(message) && nameEnd <= len(message) {
			newName := strings.TrimSpace(message[nameStart:nameEnd])

			// Find contact by relationship type
			relationshipStart := match[0]
			relationshipText := message[relationshipStart:nameStart]
			contact := findContactByRelationshipKeyword(relationshipText, activeContacts)
			if contact != nil {
				updates = append(updates, &NamingUpdate{
					ContactID:   contact.ID,
					ContactName: contact.Name,
					NewName:     newName,
					Confidence:  0.90,
					Pattern:     "relationship_is",
				})
				log.Printf("[ProgressiveNaming] Pattern 'relationship is X': %s → %s (contact ID: %d)", contact.Name, newName, contact.ID)
			}
		}
	}

	// Pattern 3: "She's/He's/They're named X"
	// Examples: "She's named Emily", "He's named Marcus"
	namedPattern := regexp.MustCompile(`(?:she's|he's|they're|i'm|you're)\s+named\s+([A-Z][a-z]+(?:\s+[A-Z][a-z]+)?)`)
	matches = namedPattern.FindAllStringSubmatchIndex(message, -1)
	for _, match := range matches {
		nameStart := match[2]
		nameEnd := match[3]
		if nameStart < len(message) && nameEnd <= len(message) {
			newName := strings.TrimSpace(message[nameStart:nameEnd])

			// Find contact by pronoun
			precedingText := message[:nameStart]
			contact := findContactByPronoun(precedingText, activeContacts)
			if contact != nil {
				updates = append(updates, &NamingUpdate{
					ContactID:   contact.ID,
					ContactName: contact.Name,
					NewName:     newName,
					Confidence:  0.85,
					Pattern:     "pronoun_named",
				})
				log.Printf("[ProgressiveNaming] Pattern 'pronoun named X': %s → %s (contact ID: %d)", contact.Name, newName, contact.ID)
			}
		}
	}

	log.Printf("[ProgressiveNaming] Detected %d potential name updates", len(updates))
	return updates
}

// ApplyNamingUpdates saves the name changes to database
func (pnd *ProgressiveNamingDetector) ApplyNamingUpdates(
	userID string,
	updates []*NamingUpdate,
) error {
	if len(updates) == 0 {
		return nil
	}

	contactRepo := database.NewContactRepository(pnd.db)

	for _, update := range updates {
		log.Printf("[ProgressiveNaming] Applying update: %s → %s (ID: %d)", update.ContactName, update.NewName, update.ContactID)

		// Load the current contact
		contact, err := contactRepo.GetByID(update.ContactID)
		if err != nil {
			log.Printf("[ProgressiveNaming] Warning: Failed to load contact %d: %v", update.ContactID, err)
			continue
		}

		if contact == nil {
			log.Printf("[ProgressiveNaming] Warning: Contact %d not found", update.ContactID)
			continue
		}

		// Update the name
		contact.Name = update.NewName
		contact.Status = "named" // Mark as named (was "active" or "unnamed")
		contact.Confidence = update.Confidence
		contact.UpdatedAt = int64(0) // Will be set by Update()

		// Save to database
		err = contactRepo.Update(contact)
		if err != nil {
			log.Printf("[ProgressiveNaming] Error: Failed to update contact: %v", err)
			continue
		}

		log.Printf("[ProgressiveNaming] ✓ Updated contact: %s → %s", update.ContactName, update.NewName)
	}

	return nil
}

// findContactByPronoun looks backward in text to find pronouns and matches to contacts
func findContactByPronoun(precedingText string, activeContacts []*models.Contact) *models.Contact {
	if len(activeContacts) == 0 {
		return nil
	}

	lowerText := strings.ToLower(precedingText)

	// Check for pronouns in reverse order (last pronoun is most relevant)
	femalePronouns := []string{"she", "her"}
	malePronouns := []string{"he", "him"}
	neutralPronouns := []string{"they", "their", "them"}

	// Check for female pronouns
	for _, pron := range femalePronouns {
		if strings.Contains(lowerText, pron) {
			// For now, just return the first contact
			// In a more sophisticated implementation, we'd match based on contact pronouns
			return activeContacts[0]
		}
	}

	// Check for male pronouns
	for _, pron := range malePronouns {
		if strings.Contains(lowerText, pron) {
			// For now, just return the first contact
			return activeContacts[0]
		}
	}

	// Check for neutral pronouns
	for _, pron := range neutralPronouns {
		if strings.Contains(lowerText, pron) {
			// For now, just return the first contact
			return activeContacts[0]
		}
	}

	// Fallback: return first active contact
	if len(activeContacts) > 0 {
		return activeContacts[0]
	}

	return nil
}

// findContactByRelationshipKeyword matches contact by relationship type keywords
func findContactByRelationshipKeyword(text string, activeContacts []*models.Contact) *models.Contact {
	lowerText := strings.ToLower(text)

	relationshipKeywords := map[string][]string{
		"romantic": {"girlfriend", "boyfriend", "partner", "spouse", "husband", "wife"},
		"professional": {"colleague", "coworker", "boss", "manager", "mentor", "employee"},
		"family": {"sister", "brother", "mother", "father", "parent", "sibling"},
		"friend": {"friend", "mate", "buddy", "pal"},
	}

	// Find which relationship type matches
	for relType, keywords := range relationshipKeywords {
		for _, keyword := range keywords {
			if strings.Contains(lowerText, keyword) {
				// Find first contact with this relationship
				for _, c := range activeContacts {
					if c.Relationship == relType {
						return c
					}
				}
			}
		}
	}

	// Fallback: return first contact
	if len(activeContacts) > 0 {
		return activeContacts[0]
	}

	return nil
}

// GetNameUpdateSummary returns a human-readable summary of updates
func (pnd *ProgressiveNamingDetector) GetNameUpdateSummary(updates []*NamingUpdate) string {
	if len(updates) == 0 {
		return ""
	}

	summaries := make([]string, 0, len(updates))
	for _, u := range updates {
		summaries = append(summaries, fmt.Sprintf("%s → %s", u.ContactName, u.NewName))
	}

	return strings.Join(summaries, ", ")
}
