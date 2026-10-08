package agents

import (
	"log"
	"moly/database"
	"moly/models"
	"strings"
)

// ContactDetector finds contacts mentioned in a message
type ContactDetector struct {
	db *database.Database
}

// NewContactDetector creates a new contact detector
func NewContactDetector(db *database.Database) *ContactDetector {
	return &ContactDetector{db: db}
}

// DetectInMessage finds all contacts mentioned in a message
// Handles: explicit mentions ("girl", "colleague"), pronouns (she, he), named references (Emily)
func (cd *ContactDetector) DetectInMessage(message string, analysisCtx *models.AnalysisContext) []*models.Contact {
	log.Printf("[ContactDetector] Detecting contacts in message: %q", message)

	var detected []*models.Contact

	// Step 1: Detect explicit contact introductions
	// Patterns: "I want to talk about...", "I'm dating...", "my colleague...", etc.
	explicitContacts := cd.detectExplicitMentions(message)
	detected = append(detected, explicitContacts...)

	// Step 2: Detect named references
	// Look for proper nouns (capitalized names)
	namedContacts := cd.detectNamedReferences(message)
	detected = append(detected, namedContacts...)

	// Step 3: Detect pronouns for previously mentioned contacts
	// Link pronouns to existing contacts if available
	if analysisCtx != nil && len(analysisCtx.RelevantContacts) > 0 {
		// Convert []models.Contact to []*models.Contact
		var relevantPtrs []*models.Contact
		for i := range analysisCtx.RelevantContacts {
			relevantPtrs = append(relevantPtrs, &analysisCtx.RelevantContacts[i])
		}
		pronounContacts := cd.linkPronounsToExistingContacts(message, relevantPtrs)
		detected = append(detected, pronounContacts...)
	}

	log.Printf("[ContactDetector] Found %d contacts", len(detected))
	for i, c := range detected {
		log.Printf("[ContactDetector]   %d: %s (type=%s, pronouns=%v)", i+1, c.Name, c.Relationship, c.Pronouns)
	}

	return detected
}

// detectExplicitMentions finds explicit contact introductions
// Patterns: "girl", "woman", "colleague", "boyfriend", "girlfriend", "boss", etc.
func (cd *ContactDetector) detectExplicitMentions(message string) []*models.Contact {
	var contacts []*models.Contact

	lower := strings.ToLower(message)

	// Contact type patterns
	contactPatterns := map[string]string{
		"girlfriend": "romantic",
		"boyfriend":  "romantic",
		"girl":       "romantic",
		"woman":      "romantic",
		"partner":    "romantic",
		"dating":     "romantic",
		"colleague":  "professional",
		"boss":       "professional",
		"coworker":   "professional",
		"manager":    "professional",
		"friend":     "friend",
		"family":     "family",
		"mother":     "family",
		"father":     "family",
		"sister":     "family",
		"brother":    "family",
	}

	for pattern, contactType := range contactPatterns {
		if strings.Contains(lower, pattern) {
			// Found a mention of this type
			// Extract pronouns if available
			pronouns := cd.extractPronounsFromContext(message, pattern)

			// Create contact with placeholder name
			contact := &models.Contact{
				Name:         models.GeneratePlaceholderName(contactType, 1),
				Relationship: contactType,
				Pronouns:     pronouns,
				Status:       "unnamed",
				Confidence:   0.8, // High confidence for explicit mention
				CreatedVia:   "conversation",
			}

			contacts = append(contacts, contact)
			log.Printf("[ContactDetector] Detected explicit %s mention (pronouns: %v)", contactType, pronouns)
			break // Only detect first mention per message
		}
	}

	return contacts
}

// detectNamedReferences finds explicitly named contacts
// Patterns: "Emily", "Marcus", "my girlfriend Sarah", etc.
func (cd *ContactDetector) detectNamedReferences(message string) []*models.Contact {
	var contacts []*models.Contact

	// Simple heuristic: look for capitalized words that appear to be names
	words := strings.Fields(message)
	for i, word := range words {
		// Check if word is capitalized (likely a name)
		if len(word) > 1 && word[0] >= 'A' && word[0] <= 'Z' {
			// Skip common non-names
			if cd.isCommonWord(word) {
				continue
			}

			// Found potential name
			name := strings.TrimRight(word, ",'\".,!?;:")
			if len(name) < 2 {
				continue
			}

			// Extract context (girlfriend, colleague, etc.)
			contactType := cd.extractContactTypeFromContext(message, i)

			// Extract pronouns
			pronouns := cd.extractPronounsFromContext(message, name)

			contact := &models.Contact{
				Name:         name,
				Relationship: contactType,
				Pronouns:     pronouns,
				Status:       "active",
				Confidence:   0.9, // High confidence for named contact
				CreatedVia:   "conversation",
			}

			contacts = append(contacts, contact)
			log.Printf("[ContactDetector] Detected named contact: %s (type=%s, pronouns=%v)", name, contactType, pronouns)
			break // Only detect first named contact per message
		}
	}

	return contacts
}

// linkPronounsToExistingContacts links pronouns to contacts from previous context
func (cd *ContactDetector) linkPronounsToExistingContacts(message string, existingContacts []*models.Contact) []*models.Contact {
	var contacts []*models.Contact

	pronouns := cd.extractAllPronouns(message)
	if len(pronouns) == 0 {
		return contacts
	}

	// For each pronoun, try to link to existing contact
	for _, pronoun := range pronouns {
		for _, existing := range existingContacts {
			if cd.pronounMatchesContact(pronoun, existing) {
				// Pronoun refers to this existing contact
				// Update the contact's characteristics from this message
				log.Printf("[ContactDetector] Linked pronoun %q to contact %s", pronoun, existing.Name)
				contacts = append(contacts, existing)
				break
			}
		}
	}

	return contacts
}

// extractPronounsFromContext finds pronouns associated with a contact mention
func (cd *ContactDetector) extractPronounsFromContext(message string, contactMention string) []string {
	var pronouns []string

	lower := strings.ToLower(message)

	// Look for pronouns near the mention
	patterns := map[string][]string{
		"she": {"she", "her", "hers"},
		"he":  {"he", "him", "his"},
		"they": {"they", "them", "their", "theirs"},
	}

	for _, forms := range patterns {
		for _, form := range forms {
			if strings.Contains(lower, form) {
				pronouns = append(pronouns, form)
			}
		}
	}

	// Deduplicate
	seen := make(map[string]bool)
	unique := []string{}
	for _, p := range pronouns {
		if !seen[p] {
			seen[p] = true
			unique = append(unique, p)
		}
	}

	return unique
}

// extractAllPronouns finds all pronouns in the message
func (cd *ContactDetector) extractAllPronouns(message string) []string {
	var pronouns []string
	lower := strings.ToLower(message)

	pronounMap := []string{"she", "her", "hers", "he", "him", "his", "they", "them", "their", "theirs"}
	for _, p := range pronounMap {
		if strings.Contains(lower, p) {
			pronouns = append(pronouns, p)
		}
	}

	return pronouns
}

// extractContactTypeFromContext determines contact type from surrounding text
func (cd *ContactDetector) extractContactTypeFromContext(message string, _ int) string {
	lower := strings.ToLower(message)

	if strings.Contains(lower, "girlfriend") || strings.Contains(lower, "boyfriend") ||
		strings.Contains(lower, "dating") || strings.Contains(lower, "partner") {
		return "romantic"
	} else if strings.Contains(lower, "colleague") || strings.Contains(lower, "coworker") ||
		strings.Contains(lower, "boss") || strings.Contains(lower, "work") {
		return "professional"
	} else if strings.Contains(lower, "friend") {
		return "friend"
	} else if strings.Contains(lower, "family") || strings.Contains(lower, "mother") ||
		strings.Contains(lower, "father") || strings.Contains(lower, "sister") ||
		strings.Contains(lower, "brother") {
		return "family"
	}

	return "other"
}

// pronounMatchesContact checks if a pronoun matches a contact
func (cd *ContactDetector) pronounMatchesContact(pronoun string, contact *models.Contact) bool {
	lower := strings.ToLower(pronoun)

	// Check if contact has this pronoun recorded
	for _, p := range contact.Pronouns {
		if strings.ToLower(p) == lower {
			return true
		}
	}

	// Infer pronoun from name if not recorded
	// TODO: Implement gender detection from name

	return false
}

// isCommonWord checks if a capitalized word is a common word (not a name)
func (cd *ContactDetector) isCommonWord(word string) bool {
	commonWords := map[string]bool{
		"I":          true,
		"A":          true,
		"The":        true,
		"And":        true,
		"But":        true,
		"Or":         true,
		"In":         true,
		"On":         true,
		"At":         true,
		"To":         true,
		"From":       true,
		"With":       true,
		"My":         true,
		"Your":       true,
		"His":        true,
		"Her":        true,
		"Their":      true,
		"That":       true,
		"This":       true,
		"What":       true,
		"When":       true,
		"Where":      true,
		"Why":        true,
		"How":        true,
		"Who":        true,
		"Monday":     true,
		"Tuesday":    true,
		"Wednesday":  true,
		"Thursday":   true,
		"Friday":     true,
		"Saturday":   true,
		"Sunday":     true,
		"January":    true,
		"February":   true,
		"March":      true,
		"April":      true,
		"May":        true,
		"June":       true,
		"July":       true,
		"August":     true,
		"September":  true,
		"October":    true,
		"November":   true,
		"December":   true,
	}

	return commonWords[word]
}
