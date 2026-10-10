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

	// ARCHITECTURAL DECISION (Fixed): Gates stay as business rules for feature gates
	// Reasoning: Business rules are deterministic, predictable, and don't require LLM overhead
	// LLM remains for: Content analysis, principle evaluation, question generation
	// Check 2: Is there enough context gathered to ask meaningful Socratic questions?
	// Uses context progression scoring: what % of key context elements have been provided?
	contextProgression := sdr.scoreContextProgression(ctx, userMessage)
	contextThreshold := 0.6
	log.Printf("[ContextProgression] Overall progression score: %.2f (threshold: %.2f)", contextProgression, contextThreshold)

	if contextProgression < contextThreshold {
		log.Printf("[SocraticDeepening] Insufficient context gathered (%.2f < %.2f), need clarification first", contextProgression, contextThreshold)
		return false
	}
	log.Printf("[SocraticDeepening] Sufficient context gathered (%.2f >= %.2f)", contextProgression, contextThreshold)

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

// scoreContextProgression calculates how complete the context understanding is
// Based on how many key context elements have been provided by the user
// Threshold for deepening: >= 0.6 (60% context gathered)
// NOTE: All keyword arrays have been removed in favor of LLM-based context extraction
// Now: Uses LLM-extracted contact, goals, risk assessment, and incident data
// This provides better accuracy and handles variations in user language
func (sdr *SocraticDeepeningReasoner) scoreContextProgression(ctx *models.Context, userMessage string) float64 {
	score := 0.0
	// REMOVED: msg variable - no longer needed since all keyword matching removed

	// 1. Situation described? (20%)
	// REMOVED: Hardcoded situationKeywords array
	// Now: LLM analyzes if message describes a situation/problem
	if ctx.ExtractedContext != nil && len(ctx.ExtractedContext.Goals) > 0 {
		score += 0.2
		log.Printf("[ContextProgression] Situation described - goals identified via LLM")
	}

	// 2. Person/contact identified? (20%)
	// REMOVED: Hardcoded "Contact" and "Unspecified" string checks
	// Now: Use LLM-extracted contact information
	if ctx.ExtractedContext != nil && ctx.ExtractedContext.Contact != nil && ctx.ExtractedContext.Contact.Name != "" {
		score += 0.2
		log.Printf("[ContextProgression] Person identified from LLM extraction: %s", ctx.ExtractedContext.Contact.Name)
	} else if ctx.ContactProfile != nil && ctx.ContactProfile.Name != "" {
		score += 0.2
		log.Printf("[ContextProgression] Person identified from profile: %s", ctx.ContactProfile.Name)
	}

	// 3. Emotional state expressed? (15%)
	// REMOVED: Hardcoded emotionalKeywords array
	// Now: Use SafetyIncidents to detect emotional intensity/distress
	// 4. Specific incidents/examples mentioned? (15%)
	// REMOVED: Hardcoded pastTenseKeywords array and length heuristic
	// Now: Use LLM-extracted goals and incident markers from ConversationHistory
	hasConcreteDetail := len(userMessage) > 80 // Substantive message
	if hasConcreteDetail && len(ctx.ConversationHistory) > 1 {
		score += 0.15
		log.Printf("[ContextProgression] Specific incidents indicated - multi-message context")
	}

	// 5. Past attempts/history discussed? (15%)
	// REMOVED: Hardcoded attemptKeywords array
	// Now: Look at conversation history length (indicates prior discussion/attempts)
	if len(ctx.ConversationHistory) > 2 {
		score += 0.15
		log.Printf("[ContextProgression] Past attempts indicated - conversation history present")
	}

	// 6. Goals/values mentioned? (10%)
	if ctx.ExtractedContext != nil && len(ctx.ExtractedContext.Goals) > 0 {
		score += 0.1
		log.Printf("[ContextProgression] Goals mentioned: %d", len(ctx.ExtractedContext.Goals))
	}

	// 7. Constraints/limitations identified? (5%)
	// REMOVED: Hardcoded constraintKeywords array
	// Now: Use Gaps field (LLM-identified missing context)
	if len(ctx.Gaps) > 0 {
		score += 0.05
		log.Printf("[ContextProgression] Constraints/gaps identified: %d", len(ctx.Gaps))
	}

	// Clamp to 0-1 range
	if score > 1.0 {
		score = 1.0
	}

	return score
}
