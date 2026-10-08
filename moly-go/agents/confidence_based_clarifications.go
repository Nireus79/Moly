package agents

import (
	"fmt"
	"log"
	"time"

	"moly/database"
	"moly/models"
)

// ConfidenceBasedClarifications generates clarification questions based on extraction confidence
// Phase 2 of FIX #3: Only ask clarifications for low-confidence extractions
type ConfidenceBasedClarifications struct {
	clarificationEngine *ClarificationEngine
}

// NewConfidenceBasedClarifications creates a new clarification generator
func NewConfidenceBasedClarifications() *ConfidenceBasedClarifications {
	return &ConfidenceBasedClarifications{
		clarificationEngine: NewClarificationEngine(),
	}
}

// GenerateClarificationsForExtraction creates questions for low-confidence fields
// User said: message
// Semantic extraction: context
// For each field with confidence < 0.80, generate a clarification question
func (cbc *ConfidenceBasedClarifications) GenerateClarificationsForExtraction(
	userMessage string,
	extracted *models.ExtractedContext,
) []*database.ClarificationQuestion {
	return cbc.GenerateClarificationsForExtractionWithProfile(userMessage, extracted, nil)
}

// GenerateClarificationsForExtractionWithProfile is the full version with optional AboutMe
// Pass userProfile to avoid asking about characteristics already in AboutMe
func (cbc *ConfidenceBasedClarifications) GenerateClarificationsForExtractionWithProfile(
	userMessage string,
	extracted *models.ExtractedContext,
	userProfile *models.AboutMe,
) []*database.ClarificationQuestion {

	var clarifications []*database.ClarificationQuestion
	now := time.Now().Unix()

	log.Printf("[ConfidenceBasedClarifications] Analyzing extraction confidence...")

	// GOAL: Check if intention confidence is low
	if extracted.Intention == "" || extracted.IntentionConfidence < 0.80 {
		log.Printf("[ConfidenceBasedClarifications] Goal confidence low (%.2f) - ask clarification", extracted.IntentionConfidence)
		q := &database.ClarificationQuestion{
			ID:                "q_goal_confidence",
			ClarificationType: "confidence_low_goal",
			Priority:          1,
			Status:            "active",
			QuestionText:      "What are you trying to accomplish with this message?",
			ContextNotes:      fmt.Sprintf("You said: \"%s\"\n\nTo help you better, I want to understand your goal.", userMessage),
			CreatedAt:         now,
		}
		clarifications = append(clarifications, q)
	}

	// CONTACT: Check if contact confidence is low
	if extracted.Contact == nil || extracted.Contact.Confidence < 0.80 {
		log.Printf("[ConfidenceBasedClarifications] Contact confidence low - ask clarification")
		q := &database.ClarificationQuestion{
			ID:                "q_contact_confidence",
			ClarificationType: "confidence_low_contact",
			Priority:          1,
			Status:            "active",
			QuestionText:      "Who are you writing to or talking about?",
			ContextNotes:      fmt.Sprintf("You said: \"%s\"\n\nI want to make sure I understand who this is about.", userMessage),
			CreatedAt:         now,
		}
		clarifications = append(clarifications, q)
	}

	// STYLE: Check if style confidence is low
	if extracted.Style == nil || extracted.Style.Confidence < 0.80 {
		log.Printf("[ConfidenceBasedClarifications] Style confidence low - ask clarification")
		q := &database.ClarificationQuestion{
			ID:                "q_style_confidence",
			ClarificationType: "confidence_low_style",
			Priority:          2,
			Status:            "active",
			QuestionText:      "How do you want to come across in this? (e.g., direct, gentle, playful, formal)",
			ContextNotes:      fmt.Sprintf("You said: \"%s\"\n\nWhat tone would work best?", userMessage),
			CreatedAt:         now,
		}
		clarifications = append(clarifications, q)
	}

	// VALUES: Check if values are missing or low confidence
	if len(extracted.UserValues) == 0 {
		log.Printf("[ConfidenceBasedClarifications] No values detected - ask clarification")
		q := &database.ClarificationQuestion{
			ID:                "q_values_confidence",
			ClarificationType: "confidence_low_values",
			Priority:          2,
			Status:            "active",
			QuestionText:      "What's important to you in this situation? What do you value?",
			ContextNotes:      fmt.Sprintf("You said: \"%s\"\n\nUnderstanding your values helps me guide you better.", userMessage),
			CreatedAt:         now,
		}
		clarifications = append(clarifications, q)
	}

	// CHARACTERISTICS: Check if characteristics are missing
	// Prefer accumulated AboutMe characteristics, don't ask if already have them
	if userProfile != nil && len(userProfile.Characteristics) > 0 {
		log.Printf("[ConfidenceBasedClarifications] Already have user characteristics in AboutMe: %d traits", len(userProfile.Characteristics))
	} else if len(extracted.UserCharacteristics) == 0 {
		log.Printf("[ConfidenceBasedClarifications] No characteristics detected - optional")
		// Don't ask automatically - user can clarify if they want
	}

	// CONCERNS: Check if concerns are detected or should ask about them
	if len(extracted.IntentionPrinciples) == 0 {
		log.Printf("[ConfidenceBasedClarifications] No concerns/principles detected - optional")
		// Don't ask automatically - user will share concerns naturally
	}

	if len(clarifications) > 0 {
		log.Printf("[ConfidenceBasedClarifications] Generated %d clarification questions", len(clarifications))
	} else {
		log.Printf("[ConfidenceBasedClarifications] All extractions confident - no clarifications needed")
	}

	return clarifications
}

// ProcessClarificationAnswer integrates user's response back into extracted context
// When user answers a clarification question, update the extraction with high confidence
func (cbc *ConfidenceBasedClarifications) ProcessClarificationAnswer(
	clarificationID string,
	userAnswer string,
	extracted *models.ExtractedContext,
) *models.ExtractedContext {

	log.Printf("[ConfidenceBasedClarifications] Processing clarification answer for %s: %q", clarificationID, userAnswer)

	// Map clarification type to extraction field and update with high confidence
	switch clarificationID {
	case "q_goal_confidence":
		// User clarified their goal - use their words directly, high confidence
		extracted.Intention = userAnswer
		extracted.IntentionConfidence = 0.95 // User's own words = high confidence
		log.Printf("[ConfidenceBasedClarifications] Updated goal to: %q (confidence: 0.95)", userAnswer)

	case "q_contact_confidence":
		// User clarified who/what they're talking about
		// Could extract name and relationship from answer, but for now just note high confidence
		if extracted.Contact == nil {
			extracted.Contact = &models.ExtractedContact{}
		}
		extracted.Contact.Evidence = userAnswer
		extracted.Contact.Confidence = 0.95
		log.Printf("[ConfidenceBasedClarifications] Updated contact context (confidence: 0.95)")

	case "q_style_confidence":
		// User clarified style/tone
		if extracted.Style == nil {
			extracted.Style = &models.ExtractedStyle{}
		}
		extracted.Style.Style = userAnswer
		extracted.Style.Confidence = 0.95
		log.Printf("[ConfidenceBasedClarifications] Updated style to: %q (confidence: 0.95)", userAnswer)

	case "q_values_confidence":
		// User clarified what they value
		// Add to values list
		extracted.UserValues = append(extracted.UserValues, userAnswer)
		log.Printf("[ConfidenceBasedClarifications] Added value: %q", userAnswer)

	default:
		log.Printf("[ConfidenceBasedClarifications] Unknown clarification type: %s", clarificationID)
	}

	return extracted
}

// ShouldAskClarifications determines if we should ask any clarification questions
func (cbc *ConfidenceBasedClarifications) ShouldAskClarifications(extracted *models.ExtractedContext) bool {
	// Ask if any critical field has low confidence
	if extracted.Intention == "" || extracted.IntentionConfidence < 0.80 {
		return true
	}
	if extracted.Contact == nil || extracted.Contact.Confidence < 0.80 {
		return true
	}
	if extracted.Style == nil || extracted.Style.Confidence < 0.80 {
		return true
	}
	return false
}
