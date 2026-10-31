package agents

import (
	"log"

	"moly/models"
)

// SocraticDeepeningReasoner decides whether and how to deepen a conversation with Socratic questions
// It implements Layer 1 of the Socratic integration plan
type SocraticDeepeningReasoner struct {
	questionSelector *SocraticQuestionSelector
}

// NewSocraticDeepeningReasoner creates a new deepening reasoner
func NewSocraticDeepeningReasoner(selector *SocraticQuestionSelector) *SocraticDeepeningReasoner {
	return &SocraticDeepeningReasoner{
		questionSelector: selector,
	}
}

// ShouldDeepen determines if we should pursue Socratic deepening in this conversation
// Applies Layer 1 logic from the plan: context sufficient? complex? not repeated? emotionally ready?
func (sdr *SocraticDeepeningReasoner) ShouldDeepen(
	ctx *models.Context,
	userMessage string,
	previousQuestions []models.SocraticQuestion,
) bool {
	log.Printf("[SocraticDeepening] Evaluating if should deepen conversation")

	// Check 1: Do we have minimum context?
	// If contact or intention missing, we're still in clarification mode, not deepening
	hasMinimumContext := sdr.hasMinimumContext(ctx)
	if !hasMinimumContext {
		log.Printf("[SocraticDeepening] Insufficient context (contact or intention missing), skipping deepening")
		return false
	}

	// Whether the situation is worth a reflective question is the model's call (JudgeDeepening, last gate in deepeningAllowed). A
	// context score used to stand here: it counted fields of the extraction, stayed at 0.5 in real chats and blocked every question.

	// Check 4: Have we already explored this deeply?
	// Don't overwhelm with too many questions
	questionsAsked := len(previousQuestions)
	if questionsAsked >= 4 {
		log.Printf("[SocraticDeepening] Already asked %d questions, skipping to avoid overwhelm", questionsAsked)
		return false
	}

	// Check 5: Is user emotionally ready for questioning?
	// If they're in crisis (very_negative), validate first before asking
	emotionalReadiness := sdr.assessEmotionalReadiness(ctx)
	if emotionalReadiness < 0.3 {
		log.Printf("[SocraticDeepening] Low emotional readiness (%.2f), user needs support first", emotionalReadiness)
		return false
	}
	log.Printf("[SocraticDeepening] Adequate emotional readiness (%.2f)", emotionalReadiness)

	log.Printf("[SocraticDeepening] ✓ All checks passed, should deepen")
	return true
}

// hasMinimumContext checks if we have the essential pieces of context
// Requires: Some understanding of who we're talking about (contact/topic) and what they want (intention)
func (sdr *SocraticDeepeningReasoner) hasMinimumContext(ctx *models.Context) bool {
	// Check 1: Do we have SOME contact/topic information?
	hasContact := false
	if ctx.ContactProfile != nil && ctx.ContactProfile.Name != "" && ctx.ContactProfile.Name != "Contact" {
		hasContact = true
		log.Printf("[SocraticDeepening] Has contact: %s (%s)", ctx.ContactProfile.Name, ctx.ContactProfile.Relationship)
	} else if ctx.ExtractedContext != nil && ctx.ExtractedContext.Contact != nil {
		hasContact = true
		log.Printf("[SocraticDeepening] Has extracted contact: %s", ctx.ExtractedContext.Contact.Name)
	}

	// Check 2: Do we have clear intention?
	hasIntention := false
	if ctx.ExtractedContext != nil && ctx.ExtractedContext.Intention != "" {
		hasIntention = true
		log.Printf("[SocraticDeepening] Has intention: %s", ctx.ExtractedContext.Intention)
	} else if ctx.PastIntention != "" {
		hasIntention = true
		log.Printf("[SocraticDeepening] Has past intention: %s", ctx.PastIntention)
	}

	// Both must be true for minimum context
	return hasContact && hasIntention
}

// assessEmotionalReadiness checks if user is in a state where questions are appropriate
// Returns score 0-1 where 0=needs support, 1=ready for exploration
func (sdr *SocraticDeepeningReasoner) assessEmotionalReadiness(ctx *models.Context) float64 {
	readiness := 0.7 // Default: assume moderate readiness

	// Check for crisis-level incidents (they need support, not questions)
	if len(ctx.RecentSafetyIncidents) > 0 {
		for _, incident := range ctx.RecentSafetyIncidents {
			if incident.Severity == "critical" {
				readiness = 0.1 // Very low readiness, needs immediate support
				log.Printf("[SocraticDeepening] Critical safety incident detected, low readiness")
				return readiness
			} else if incident.Severity == "high" {
				readiness = 0.3 // Low readiness, needs support first
				log.Printf("[SocraticDeepening] High-severity incident detected, reduced readiness")
			}
		}
	}

	// If no incidents and complex situation, user is ready
	return readiness
}

// SelectQuestion determines which Socratic question to ask next
// Uses the SocraticQuestionSelector to intelligently choose based on context
func (sdr *SocraticDeepeningReasoner) SelectQuestion(
	ctx *models.Context,
	userMessage string,
	previousQuestions []models.SocraticQuestion,
) (*models.SocraticQuestion, string) {
	if sdr.questionSelector == nil {
		log.Printf("[SocraticDeepening] No question selector available")
		return nil, ""
	}

	log.Printf("[SocraticDeepening] Selecting next question (already asked: %d)", len(previousQuestions))

	// Use the selector's built-in logic
	// It analyzes ambiguities, maps to principles, finds uncovered categories, and selects approach
	question := sdr.questionSelector.SelectNextQuestion(ctx, userMessage, previousQuestions)

	if question == nil {
		log.Printf("[SocraticDeepening] No suitable question found")
		return nil, ""
	}

	log.Printf("[SocraticDeepening] Selected question: %s (approach: %s, depth: %d)",
		question.ID, question.SocraticApproach, question.DepthLevel)

	return question, question.SocraticApproach
}
