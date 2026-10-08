package agents

import (
	"log"
	"moly/models"
)

// ConfidenceCalculator computes confidence scores for contact disambiguation
type ConfidenceCalculator struct{}

// NewConfidenceCalculator creates a new calculator
func NewConfidenceCalculator() *ConfidenceCalculator {
	return &ConfidenceCalculator{}
}

// CalculateResolutionConfidence computes confidence for a pronoun resolution
// Returns a score 0.0-1.0 where:
//   >= 0.80: High confidence (assume, don't ask)
//   0.50-0.80: Medium confidence (can assume with note)
//   < 0.50: Low confidence (must ask for clarification)
func (cc *ConfidenceCalculator) CalculateResolutionConfidence(
	pronoun string,
	possibleContacts []*models.Contact,
	recentlyMentioned *models.Contact,
) float64 {
	score := 0.5 // Start with neutral

	// Rule 1: Single possible contact with matching pronoun
	if len(possibleContacts) == 1 {
		if cc.pronounMatches(pronoun, possibleContacts[0]) {
			score = 0.95 // Very high confidence
			log.Printf("[ConfidenceCalculator] Single contact match: %.2f", score)
			return score
		}
	}

	// Rule 2: Multiple possible contacts
	if len(possibleContacts) > 1 {
		matching := cc.filterByPronoun(pronoun, possibleContacts)

		if len(matching) == 1 {
			// Only one contact has this pronoun
			score = 0.85
			log.Printf("[ConfidenceCalculator] Single pronoun match (multi-contact): %.2f", score)
		} else if len(matching) > 1 {
			// Multiple contacts have this pronoun - ambiguous
			if recentlyMentioned != nil && cc.pronounMatches(pronoun, recentlyMentioned) {
				// But last mentioned contact matches
				score = 0.70 // Assume continuity
				log.Printf("[ConfidenceCalculator] Multiple matches, using recency: %.2f", score)
			} else {
				// Truly ambiguous
				score = 0.40 // Must ask
				log.Printf("[ConfidenceCalculator] Ambiguous pronoun (multiple matches): %.2f", score)
			}
		}
	}

	// Rule 3: No contacts yet (first mention)
	if len(possibleContacts) == 0 {
		score = 0.65 // Medium confidence for new contact
		log.Printf("[ConfidenceCalculator] New contact: %.2f", score)
	}

	return score
}

// CalculateContactAmbiguity checks if there's ambiguity in which contact is being referenced
// Returns ambiguity score (higher = more ambiguous)
func (cc *ConfidenceCalculator) CalculateContactAmbiguity(
	message string,
	detectedContacts []*models.Contact,
	activeContacts []*models.Contact,
) float64 {
	ambiguity := 0.0

	// More unnamed contacts = more ambiguous
	unnamedCount := 0
	for _, c := range append(detectedContacts, activeContacts...) {
		if c.Status == "unnamed" {
			unnamedCount++
		}
	}

	if unnamedCount > 1 {
		ambiguity += 0.3 // Multiple unnamed contacts increases ambiguity
	}

	// Multiple feminine pronouns with multiple female contacts = ambiguous
	shePronounCount := cc.countPronounOccurrences(message, "she")
	femaleCount := cc.countFemaleContacts(append(detectedContacts, activeContacts...))

	if shePronounCount > 1 && femaleCount > 1 {
		ambiguity += 0.4 // Multiple "she" references with multiple females
	}

	// Mixed pronouns = potentially ambiguous
	hasShePronoun := cc.hasPronoun(message, "she")
	hasHePronoun := cc.hasPronoun(message, "he")
	if hasShePronoun && hasHePronoun {
		ambiguity += 0.2 // Mixed pronouns
	}

	return ambiguity
}

// filterByPronoun returns contacts that match the given pronoun
func (cc *ConfidenceCalculator) filterByPronoun(pronoun string, contacts []*models.Contact) []*models.Contact {
	var matched []*models.Contact
	for _, c := range contacts {
		if cc.pronounMatches(pronoun, c) {
			matched = append(matched, c)
		}
	}
	return matched
}

// pronounMatches checks if a pronoun matches a contact
func (cc *ConfidenceCalculator) pronounMatches(pronoun string, contact *models.Contact) bool {
	// Check recorded pronouns
	for _, p := range contact.Pronouns {
		if p == pronoun {
			return true
		}
	}

	// Infer from contact type if not recorded
	switch pronoun {
	case "she", "her":
		return contact.Relationship == "romantic" || contact.Relationship == "professional"
	case "he", "him":
		return contact.Relationship != "romantic" // Fallback
	case "they", "them":
		return true // They/them work with anyone
	}

	return false
}

// countPronounOccurrences counts how many times a pronoun appears
func (cc *ConfidenceCalculator) countPronounOccurrences(message string, pronoun string) int {
	count := 0
	pos := 0
	for {
		idx := indexOfWord(message, pronoun, pos)
		if idx == -1 {
			break
		}
		count++
		pos = idx + 1
	}
	return count
}

// countFemaleContacts counts female contacts
func (cc *ConfidenceCalculator) countFemaleContacts(contacts []*models.Contact) int {
	count := 0
	for _, c := range contacts {
		// Check pronouns or name patterns
		for _, p := range c.Pronouns {
			if p == "she" || p == "her" {
				count++
				break
			}
		}
	}
	return count
}

// hasPronoun checks if message contains a pronoun
func (cc *ConfidenceCalculator) hasPronoun(message string, pronoun string) bool {
	return indexOfWord(message, pronoun, 0) != -1
}

// indexOfWord finds the index of a word in a message
func indexOfWord(message string, word string, startPos int) int {
	lowerMsg := toLower(message)
	lowerWord := toLower(word)

	for i := startPos; i < len(lowerMsg); i++ {
		// Look for word at position i
		if i+len(lowerWord) > len(lowerMsg) {
			break
		}

		if lowerMsg[i:i+len(lowerWord)] == lowerWord {
			// Check boundaries (word surrounded by spaces or punctuation)
			before := i == 0 || isWordBoundary(rune(lowerMsg[i-1]))
			after := i+len(lowerWord) == len(lowerMsg) || isWordBoundary(rune(lowerMsg[i+len(lowerWord)]))

			if before && after {
				return i
			}
		}
	}

	return -1
}

// isWordBoundary checks if a character is a word boundary
func isWordBoundary(ch rune) bool {
	return ch == ' ' || ch == '.' || ch == ',' || ch == '!' || ch == '?' || ch == ';' || ch == ':' || ch == '\n' || ch == '\t'
}

// toLower converts string to lowercase (simple version)
func toLower(s string) string {
	result := ""
	for _, ch := range s {
		if ch >= 'A' && ch <= 'Z' {
			result += string(ch + 32)
		} else {
			result += string(ch)
		}
	}
	return result
}
