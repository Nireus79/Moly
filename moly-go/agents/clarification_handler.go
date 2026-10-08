package agents

import (
	"log"
	"moly/database"
	"moly/models"
	"strings"
)

// ClarificationHandler processes clarification responses
type ClarificationHandler struct {
	db *database.Database
}

// NewClarificationHandler creates a new handler
func NewClarificationHandler(db *database.Database) *ClarificationHandler {
	return &ClarificationHandler{
		db: db,
	}
}

// IsClarificationResponse checks if a message is answering a clarification
func (ch *ClarificationHandler) IsClarificationResponse(message string, clarificationID string) bool {
	if clarificationID == "" {
		return false
	}

	// Check if message starts with A), B), C) etc.
	lower := strings.ToLower(strings.TrimSpace(message))
	if len(lower) > 0 && lower[0] >= 'a' && lower[0] <= 'd' {
		if len(lower) > 1 && (lower[1] == ')' || lower[1] == '.') {
			return true
		}
	}

	return true // For now, assume any response to pending clarification is an answer
}

// ProcessResponse handles a clarification response
// Returns: updated contacts, error
func (ch *ClarificationHandler) ProcessResponse(
	userID string,
	conversationID string,
	clarificationID string,
	answerMessage string,
	activeContacts []*models.Contact,
) ([]*models.Contact, error) {
	log.Printf("[ClarificationHandler] Processing response to clarification %s", clarificationID)

	// Extract answer (A, B, C, etc.)
	answer := extractAnswerOption(answerMessage)
	log.Printf("[ClarificationHandler] Extracted answer: %s", answer)

	// In Phase 3, we process the answer and update contact resolutions
	// The answer maps to one of the active contacts

	updatedContacts := make([]*models.Contact, 0, len(activeContacts))

	// Map answer to contact index (A=0, B=1, C=2, etc.)
	answerIndex := int(answer[0] - 'A')

	if answerIndex >= 0 && answerIndex < len(activeContacts) {
		// The user selected this contact
		selectedContact := activeContacts[answerIndex]
		log.Printf("[ClarificationHandler] User selected: %s", selectedContact.Name)

		// Mark selected contact as active
		selectedContact.Status = "active"
		updatedContacts = append(updatedContacts, selectedContact)

		// Mark other contacts as inactive/secondary
		for i, contact := range activeContacts {
			if i != answerIndex {
				contact.Status = "secondary"
				updatedContacts = append(updatedContacts, contact)
			}
		}
	} else {
		// Invalid answer, keep all contacts
		updatedContacts = activeContacts
	}

	log.Printf("[ClarificationHandler] Processed response: updated %d contacts", len(updatedContacts))
	return updatedContacts, nil
}

// ExtractAnswerOption extracts A/B/C from message
func extractAnswerOption(message string) string {
	lower := strings.ToLower(strings.TrimSpace(message))

	if len(lower) > 0 && lower[0] >= 'a' && lower[0] <= 'd' {
		return string(lower[0])
	}

	// Try to find in words
	words := strings.Fields(lower)
	for _, word := range words {
		word = strings.TrimRight(word, ").:-")
		if len(word) == 1 && word[0] >= 'a' && word[0] <= 'd' {
			return word
		}
	}

	// Default to A if can't parse
	return "a"
}

// ResolveContactFromAnswer resolves which contact the user meant
// Returns: the resolved contact and whether it was successful
func (ch *ClarificationHandler) ResolveContactFromAnswer(
	answer string,
	contacts []*models.Contact,
) (*models.Contact, bool) {
	// Convert answer to index
	if len(answer) == 0 {
		return nil, false
	}

	answerIndex := int(answer[0] - 'A')
	if answerIndex < 0 || answerIndex >= len(contacts) {
		return nil, false
	}

	return contacts[answerIndex], true
}

// UpdateContactResolutions updates contact metadata after clarification
func (ch *ClarificationHandler) UpdateContactResolutions(
	selectedContact *models.Contact,
	allContacts []*models.Contact,
) error {
	log.Printf("[ClarificationHandler] Updating contact resolutions for %s", selectedContact.Name)

	// Mark as explicitly confirmed
	selectedContact.Confidence = 0.99
	selectedContact.Status = "confirmed"

	// Could update other contacts as "not this one" but leaving for now

	return nil
}
