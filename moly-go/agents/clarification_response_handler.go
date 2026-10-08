package agents

import (
	"fmt"
	"log"
	"moly/config"
	"moly/database"
	"moly/models"
	"moly/schema"
	"strings"
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
	QuestionAnswered bool                            `json:"questionAnswered"`
	FactID           string                          `json:"factId"`
	Status           string                          `json:"status"`             // "active", "answered", "skipped", "cancelled"
	RemainingQs      []*schema.ClarificationQuestion `json:"remainingQuestions"` // Full question objects
	CreatedContact   *models.Contact                 `json:"createdContact,omitempty"`
	SavedAttribute   *database.ContextAttribute      `json:"savedAttribute,omitempty"`
	Error            string                          `json:"error,omitempty"`
}

// ProcessResponse handles a user's answer to a clarification question
func (h *ClarificationResponseHandler) ProcessResponse(req ClarificationResponseRequest) (*ClarificationResponseResult, error) {
	result := &ClarificationResponseResult{
		FactID: req.FactID,
	}

	log.Printf("[Moly] ClarificationResponseHandler: Processing response to question=%s for fact=%s",
		req.QuestionID, req.FactID)

	// Get the temporary fact
	tempFact, err := h.temporaryFactStore.Get(req.FactID)
	if err != nil {
		log.Printf("[Moly] ClarificationResponseHandler: ERROR fact not found: %v", err)
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
		log.Printf("[Moly] ClarificationResponseHandler: ERROR recording answer: %v", err)
		result.Error = fmt.Sprintf("Could not record answer: %v", err)
		return result, err
	}

	result.QuestionAnswered = true
	log.Printf("[Moly] ClarificationResponseHandler: Answer recorded for question=%s", req.QuestionID)

	// Check if all questions answered
	if h.temporaryFactStore.IsComplete(req.FactID) {
		log.Printf("[Moly] ClarificationResponseHandler: All questions answered for fact=%s - proceeding to save", req.FactID)
		result.Status = "complete"

		// Process based on fact type (user fact vs contact fact)
		if tempFact.AttributedTo == "user" {
			// About the user - create or update about_me profile
			err = h.handleUserFact(tempFact)
			if err != nil {
				log.Printf("[Moly] ClarificationResponseHandler: ERROR handling user fact: %v", err)
				result.Error = fmt.Sprintf("Error saving user profile: %v", err)
				return result, err
			}
		} else if strings.HasPrefix(tempFact.AttributedTo, "contact_") {
			// About a contact - create the contact then save the fact
			err = h.handleContactFact(tempFact)
			if err != nil {
				log.Printf("[Moly] ClarificationResponseHandler: ERROR handling contact fact: %v", err)
				result.Error = fmt.Sprintf("Error saving contact: %v", err)
				return result, err
			}
		}

		// Mark fact as saved
		h.temporaryFactStore.Remove(req.FactID)
		result.Status = "saved"
		log.Printf("[Moly] ClarificationResponseHandler: Fact=%s SAVED after clarification", req.FactID)
	} else {
		// Still waiting for more answers
		remaining := h.temporaryFactStore.RemainingQuestionsWithObjects(req.FactID)
		result.RemainingQs = remaining
		result.Status = "active"
		log.Printf("[Moly] ClarificationResponseHandler: Fact=%s still pending (%d remaining questions)", req.FactID, len(remaining))
	}

	return result, nil
}

// handleUserFact processes answers for facts about the user
func (h *ClarificationResponseHandler) handleUserFact(tempFact *TemporaryFact) error {
	log.Printf("[Moly] ClarificationResponseHandler: Handling user fact - saving to about_me")

	// Get the answers
	answers := h.temporaryFactStore.GetAnswers(tempFact.FactID)
	log.Printf("[Moly] ClarificationResponseHandler: User answered %d clarification questions", len(answers))

	// Extract context from answers using configured mapping
	context := config.ContextGeneral
	if len(answers) > 0 {
		// Use keyword mapping from configuration instead of hardcoded strings
		for _, answer := range answers {
			mappedContext := config.MapKeywordToContext(answer)
			if mappedContext != config.ContextGeneral {
				context = mappedContext
				break
			}
		}
	}

	// Create attribute with confirmed context
	attr := &database.ContextAttribute{
		UserID:       h.userID,
		FactType:     tempFact.FactType,
		FactValue:    tempFact.FactValue,
		AttributedTo: "user",
		Context:      context, // Now with confirmed context
		Evidence:     tempFact.Evidence,
		Confidence:   tempFact.Confidence,
		Source:       "explicit_with_context",
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
		log.Printf("[Moly] ClarificationResponseHandler: ERROR saving user attribute: %v", err)
		return err
	}

	log.Printf("[Moly] ClarificationResponseHandler: ✓ SAVED user attribute id=%d type=%s value=%s context=%s",
		savedAttr.ID, savedAttr.FactType, savedAttr.FactValue, savedAttr.Context)

	return nil
}

// handleContactFact processes answers for facts about a contact
func (h *ClarificationResponseHandler) handleContactFact(tempFact *TemporaryFact) error {
	log.Printf("[Moly] ClarificationResponseHandler: Handling contact fact - creating contact and saving attribute")

	// Extract contact name from subject (e.g., "contact_bob" -> "Bob")
	contactName := extractContactName(tempFact.AttributedTo)

	// Get the answers
	answers := h.temporaryFactStore.GetAnswers(tempFact.FactID)
	log.Printf("[Moly] ClarificationResponseHandler: Contact clarification answered %d questions", len(answers))

	// Extract relationship and context from answers using configured mappings
	relationship := config.RelationshipOther
	context := config.ContextGeneral
	for _, answer := range answers {
		// Use configured mapping functions instead of hardcoded keywords
		if mappedRel := config.MapKeywordToRelationship(answer); mappedRel != config.RelationshipOther {
			relationship = mappedRel
		}
		if mappedCtx := config.MapKeywordToContext(answer); mappedCtx != config.ContextGeneral {
			context = mappedCtx
		}
	}

	// Create the contact
	contact, err := h.contactManager.CreateContact(h.userID, contactName, relationship, []string{})
	if err != nil {
		log.Printf("[Moly] ClarificationResponseHandler: ERROR creating contact: %v", err)
		return err
	}

	log.Printf("[Moly] ClarificationResponseHandler: ✓ CREATED contact id=%d name=%s relationship=%s",
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
		log.Printf("[Moly] ClarificationResponseHandler: ERROR saving contact attribute: %v", err)
		return err
	}

	log.Printf("[Moly] ClarificationResponseHandler: ✓ SAVED contact attribute id=%d type=%s value=%s for %s",
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
