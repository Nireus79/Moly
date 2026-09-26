package agents

import (
	"moly/schema"
	"fmt"
	"log"
	"strings"

	"moly/database"
	"moly/models"
)

// ClarificationResponseHandler processes user's answers to clarification questions
type ClarificationResponseHandler struct {
	temporaryFactStore *TemporaryFactStore
	contactManager     *ContactManager
	contextAttrManager *ContextAttributeManager
	userID             string
	database           *database.Database
}

// NewClarificationResponseHandler creates a new response handler
func NewClarificationResponseHandler(
	factStore *TemporaryFactStore,
	contactMgr *ContactManager,
	contextAttrMgr *ContextAttributeManager,
	userID string,
	db *database.Database,
) *ClarificationResponseHandler {
	return &ClarificationResponseHandler{
		temporaryFactStore: factStore,
		contactManager:     contactMgr,
		contextAttrManager: contextAttrMgr,
		userID:             userID,
		database:           db,
	}
}

// ClarificationResponseRequest represents a user's answer to a question
type ClarificationResponseRequest struct {
	QuestionID     string `json:"questionId"`
	FactID         string `json:"factId"`
	SelectedOption string `json:"selectedOption"`
	UserResponse   string `json:"userResponse"`
}

// ClarificationResponseResult shows what happened when processing the answer
type ClarificationResponseResult struct {
	QuestionAnswered bool                       `json:"questionAnswered"`
	FactID           string                     `json:"factId"`
	Status           string                     `json:"status"` // "pending", "complete", "saved"
	RemainingQs      []*schema.ClarificationQuestion   `json:"remainingQuestions"` // Full question objects
	CreatedContact   *models.Contact            `json:"createdContact,omitempty"`
	SavedAttribute   *database.ContextAttribute `json:"savedAttribute,omitempty"`
	Error            string                     `json:"error,omitempty"`
}

// ProcessResponse handles a user's answer to a clarification question
func (h *ClarificationResponseHandler) ProcessResponse(req ClarificationResponseRequest) (*ClarificationResponseResult, error) {
	result := &ClarificationResponseResult{
		FactID: req.FactID,
	}

	log.Printf("[V2] ClarificationResponseHandler: Processing response to question=%s for fact=%s",
		req.QuestionID, req.FactID)

	// Get the temporary fact
	tempFact, err := h.temporaryFactStore.Get(req.FactID)
	if err != nil {
		log.Printf("[V2] ClarificationResponseHandler: ERROR fact not found: %v", err)
		result.Error = fmt.Sprintf("Fact not found: %s", req.FactID)
		return result, err
	}

	// Record the answer
	answer := req.SelectedOption
	if answer == "" {
		answer = req.UserResponse
	}

	err = h.temporaryFactStore.RecordAnswer(req.FactID, req.QuestionID, answer)
	if err != nil {
		log.Printf("[V2] ClarificationResponseHandler: ERROR recording answer: %v", err)
		result.Error = fmt.Sprintf("Could not record answer: %v", err)
		return result, err
	}

	result.QuestionAnswered = true
	log.Printf("[V2] ClarificationResponseHandler: Answer recorded for question=%s", req.QuestionID)

	// Check if all questions answered
	if h.temporaryFactStore.IsComplete(req.FactID) {
		log.Printf("[V2] ClarificationResponseHandler: All questions answered for fact=%s - proceeding to save", req.FactID)
		result.Status = "complete"

		// Process based on fact type (user fact vs contact fact)
		if tempFact.AttributedTo == "user" {
			// About the user - create or update about_me profile
			err = h.handleUserFact(tempFact)
			if err != nil {
				log.Printf("[V2] ClarificationResponseHandler: ERROR handling user fact: %v", err)
				result.Error = fmt.Sprintf("Error saving user profile: %v", err)
				return result, err
			}
		} else if strings.HasPrefix(tempFact.AttributedTo, "contact_") {
			// About a contact - create the contact then save the fact
			err = h.handleContactFact(tempFact)
			if err != nil {
				log.Printf("[V2] ClarificationResponseHandler: ERROR handling contact fact: %v", err)
				result.Error = fmt.Sprintf("Error saving contact: %v", err)
				return result, err
			}
		}

		// Mark fact as saved
		h.temporaryFactStore.Remove(req.FactID)
		result.Status = "saved"
		log.Printf("[V2] ClarificationResponseHandler: Fact=%s SAVED after clarification", req.FactID)
	} else {
		// Still waiting for more answers
		remaining := h.temporaryFactStore.RemainingQuestionsWithObjects(req.FactID)
		result.RemainingQs = remaining
		result.Status = "pending"
		log.Printf("[V2] ClarificationResponseHandler: Fact=%s still pending (%d remaining questions)", req.FactID, len(remaining))
	}

	return result, nil
}

// handleUserFact processes answers for facts about the user
func (h *ClarificationResponseHandler) handleUserFact(tempFact *TemporaryFact) error {
	log.Printf("[V2] ClarificationResponseHandler: Handling user fact - saving to about_me")

	// Get the answers
	answers := h.temporaryFactStore.GetAnswers(tempFact.FactID)
	log.Printf("[V2] ClarificationResponseHandler: User answered %d clarification questions", len(answers))

	// Extract context from answers (this is simplified - in production would be more complex)
	context := "general"
	if len(answers) > 0 {
		// TODO: REMOVE hardcoded context mapping
		// Problem: Using strings.Contains for "work"/"personal" in user answers
		// Solution: Ask LLM to categorize user answer context instead of keyword matching
		// Or: Pre-define fixed answer options to avoid need for parsing
		// First answer typically about context
		for _, answer := range answers {
			if strings.Contains(answer, "work") {
				context = "work"
				break
			} else if strings.Contains(answer, "personal") {
				context = "personal"
				break
			}
		}
	}

	// Create attribute with confirmed context
	attr := &database.ContextAttribute{
		UserID:         h.userID,
		FactType:       tempFact.FactType,
		FactValue:      tempFact.FactValue,
		AttributedTo:   "user",
		Context:        context, // Now with confirmed context
		Evidence:       tempFact.Evidence,
		Confidence:     tempFact.Confidence,
		Source:         "explicit_with_context",
	}

	// Save to database
	savedAttr, err := h.contextAttrManager.SaveAttribute(
		h.userID,
		"", // conversationId unknown at this point
		attr.FactType,
		attr.FactValue,
		attr.AttributedTo,
		attr.Context,
		attr.Evidence,
		attr.Confidence,
	)

	if err != nil {
		log.Printf("[V2] ClarificationResponseHandler: ERROR saving user attribute: %v", err)
		return err
	}

	log.Printf("[V2] ClarificationResponseHandler: ✓ SAVED user attribute id=%d type=%s value=%s context=%s",
		savedAttr.ID, savedAttr.FactType, savedAttr.FactValue, savedAttr.Context)

	return nil
}

// handleContactFact processes answers for facts about a contact
func (h *ClarificationResponseHandler) handleContactFact(tempFact *TemporaryFact) error {
	log.Printf("[V2] ClarificationResponseHandler: Handling contact fact - creating contact and saving attribute")

	// Extract contact name from subject (e.g., "contact_bob" -> "Bob")
	contactName := extractContactName(tempFact.AttributedTo)

	// Get the answers
	answers := h.temporaryFactStore.GetAnswers(tempFact.FactID)
	log.Printf("[V2] ClarificationResponseHandler: Contact clarification answered %d questions", len(answers))

	// TODO: REMOVE hardcoded relationship mapping (Boss/Manager/Colleague/Friend/Family)
	// Problem: Using strings.Contains to parse user answer for contact relationship type
	// Keywords: "Boss", "Manager", "Colleague", "Coworker", "Friend", "Family"
	// Can be easily bypassed by rewording (e.g., "my direct supervisor" instead of "boss")
	// Solution: Either:
	//   A) Ask LLM to categorize relationship from user answer
	//   B) Use fixed multiple-choice answers ("1: Boss", "2: Friend", etc.) instead of free text
	// TODO: REMOVE hardcoded context mapping (work/personal)
	// Extract relationship from answers
	relationship := ""
	context := ""
	for _, answer := range answers {
		if strings.Contains(answer, "Boss") || strings.Contains(answer, "Manager") {
			relationship = "boss"
		} else if strings.Contains(answer, "Colleague") || strings.Contains(answer, "Coworker") {
			relationship = "colleague"
		} else if strings.Contains(answer, "Friend") {
			relationship = "friend"
		} else if strings.Contains(answer, "Family") {
			relationship = "family"
		}

		if strings.Contains(answer, "work") {
			context = "work"
		} else if strings.Contains(answer, "personal") {
			context = "personal"
		}
	}

	if context == "" {
		context = "general"
	}

	// Create the contact
	contact, err := h.contactManager.CreateContact(h.userID, contactName, relationship, []string{})
	if err != nil {
		log.Printf("[V2] ClarificationResponseHandler: ERROR creating contact: %v", err)
		return err
	}

	log.Printf("[V2] ClarificationResponseHandler: ✓ CREATED contact id=%d name=%s relationship=%s",
		contact.ID, contact.Name, contact.Relationship)

	// Now save the original attribute with confirmed context
	attr := &database.ContextAttribute{
		UserID:       h.userID,
		FactType:     tempFact.FactType,
		FactValue:    tempFact.FactValue,
		AttributedTo: tempFact.AttributedTo, // contact_bob format
		Context:      context,               // Now with confirmed context
		Evidence:     tempFact.Evidence,
		Confidence:   tempFact.Confidence,
		Source:       "explicit_with_context",
	}

	savedAttr, err := h.contextAttrManager.SaveAttribute(
		h.userID,
		"", // conversationId unknown
		attr.FactType,
		attr.FactValue,
		attr.AttributedTo,
		attr.Context,
		attr.Evidence,
		attr.Confidence,
	)

	if err != nil {
		log.Printf("[V2] ClarificationResponseHandler: ERROR saving contact attribute: %v", err)
		return err
	}

	log.Printf("[V2] ClarificationResponseHandler: ✓ SAVED contact attribute id=%d type=%s value=%s for %s",
		savedAttr.ID, savedAttr.FactType, savedAttr.FactValue, contactName)

	return nil
}

// extractContactName extracts the contact name from a contact ID string
// Examples: "contact_bob" -> "bob", "contact_sarah_smith" -> "sarah_smith"
func extractContactName(attributedTo string) string {
	if strings.HasPrefix(attributedTo, "contact_") {
		name := strings.TrimPrefix(attributedTo, "contact_")
		// Replace underscores with spaces and capitalize
		name = strings.ReplaceAll(name, "_", " ")
		return strings.Title(name)
	}
	return attributedTo
}
