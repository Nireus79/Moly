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

	// TODO: Architectural decision - should gates be LLM-based or stay as business rules?
	// Gap #2 in audit: Gates check prerequisites (maturity, first message, gaps, phase, safety, risk, intent, preferences)
	// Current: Business rule thresholds (0.6 for context, 0.3 for complexity, etc.)
	// Options: (A) Keep as business rules - acceptable for feature gates (B) Make LLM-based - ask LLM "Is user ready to deepen?"
	// Check 2: Is there enough context gathered to ask meaningful Socratic questions?
	// Uses context progression scoring: what % of key context elements have been provided?
	contextProgression := sdr.scoreContextProgression(ctx, userMessage)
	log.Printf("[ContextProgression] Overall progression score: %.2f", contextProgression)

	if contextProgression < 0.6 {
		log.Printf("[SocraticDeepening] Insufficient context gathered (%.2f < 0.6), need clarification first", contextProgression)
		return false
	}
	log.Printf("[SocraticDeepening] Sufficient context gathered (%.2f >= 0.6)", contextProgression)

	// Check 3: Is the situation complex enough to warrant deepening?
	// Factors: emotional intensity, risk level, ambiguity, uncertainty
	complexity := sdr.assessComplexity(ctx, userMessage)
	if complexity < 0.3 {
		log.Printf("[SocraticDeepening] Low complexity (%.2f), skipping deepening", complexity)
		return false
	}
	log.Printf("[SocraticDeepening] Adequate complexity (%.2f) detected", complexity)

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

// assessComplexity evaluates how complex/nuanced the conversation is
// Returns score 0-1 based on emotional intensity, risk, ambiguity, uncertainty
func (sdr *SocraticDeepeningReasoner) assessComplexity(ctx *models.Context, userMessage string) float64 {
	score := 0.0

	// TODO: Remove hardcoded severity/risk thresholds (60, 30, "elevated", "high", "immediate")
	// Gap #2 in audit: Gates use hardcoded business rule thresholds instead of LLM evaluation
	// Current: severity >= 60 (0.3 score), >= 30 (0.15 score); risk level string matching
	// Solution: Either keep as business rules OR use LLM to evaluate readiness (architectural decision)
	// Factor 1: Emotional intensity (30% weight)
	// High risk/severity indicates emotional intensity warranting deeper exploration
	if ctx.LastRiskAssessment != nil {
		if severity, ok := ctx.LastRiskAssessment["severity"].(float64); ok {
			if severity >= 60 {
				score += 0.3 // Maximum emotional intensity factor
				log.Printf("[SocraticDeepening] High emotional intensity detected (severity=%.0f)", severity)
			} else if severity >= 30 {
				score += 0.15 // Medium intensity
				log.Printf("[SocraticDeepening] Moderate emotional intensity detected (severity=%.0f)", severity)
			}
		}
	}

	// Factor 2: Risk level (30% weight)
	if ctx.LastRiskAssessment != nil {
		riskLevel, ok := ctx.LastRiskAssessment["level"]
		if ok {
			riskStr, isString := riskLevel.(string)
			if isString {
				switch riskStr {
				case "elevated":
					score += 0.3
					log.Printf("[SocraticDeepening] Elevated risk detected")
				case "high", "immediate":
					score += 0.4
					log.Printf("[SocraticDeepening] High/immediate risk detected")
				}
			}
		}
	}

	// Factor 3: Number of unknowns/ambiguities (20% weight)
	unknowns := len(ctx.Gaps)
	if unknowns > 0 {
		ambiguityScore := float64(unknowns) * 0.1
		if ambiguityScore > 0.2 {
			ambiguityScore = 0.2
		}
		score += ambiguityScore
		log.Printf("[SocraticDeepening] %d context gaps detected", unknowns)
	}

	// TODO: Remove hardcoded "complete" string check for context quality
	// Gap #2 in audit: Gate checks use hardcoded business rules instead of LLM evaluation
	// Current: Checks if ctx.ContextQuality == "complete" string literal
	// Solution: Either keep as business rule OR use LLM to evaluate context quality sufficiency (architectural decision)
	// Factor 4: Context quality (20% weight)
	// Only add bonus if context is COMPLETE - don't deepen with minimal/partial context
	if ctx.ContextQuality == "complete" {
		score += 0.2
		log.Printf("[SocraticDeepening] Context quality is complete, room for deepening")
	} else {
		log.Printf("[SocraticDeepening] Context quality is %s, insufficient for deepening", ctx.ContextQuality)
	}

	// Clamp to 0-1 range
	if score > 1.0 {
		score = 1.0
	}

	return score
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
// TODO: Remove all hardcoded keyword arrays in this method
// Gap #2 in audit: scoreContextProgression uses 6 separate keyword arrays for context scoring
// Arrays: situationKeywords, emotionalKeywords, pastTenseKeywords, attemptKeywords, constraintKeywords
// Solution: Use LLM to evaluate context completeness via principle-based analysis instead of keyword scanning
// Ask LLM: "What % of context is complete? Rate: situation, person, emotion, examples, history, goals, constraints"
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
	if ctx.LastRiskAssessment != nil {
		if severity, ok := ctx.LastRiskAssessment["severity"].(float64); ok && severity > 0 {
			score += 0.15
			log.Printf("[ContextProgression] Emotional intensity detected via risk assessment")
		}
	}

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

// SelectFollowUp determines if there's a natural follow-up question
// Used for chaining questions across multiple user messages
func (sdr *SocraticDeepeningReasoner) SelectFollowUp(
	lastQuestion *models.SocraticQuestion,
	userResponse string,
) (*models.SocraticQuestion, string) {
	if sdr.questionSelector == nil {
		return nil, ""
	}

	if lastQuestion == nil {
		log.Printf("[SocraticDeepening] No previous question, can't select follow-up")
		return nil, ""
	}

	if len(lastQuestion.FollowUpQuestions) == 0 {
		log.Printf("[SocraticDeepening] No follow-ups defined for question %s", lastQuestion.ID)
		return nil, ""
	}

	followUp := sdr.questionSelector.SelectFollowUp(lastQuestion, userResponse)
	if followUp == nil {
		log.Printf("[SocraticDeepening] Follow-up selection failed")
		return nil, ""
	}

	log.Printf("[SocraticDeepening] Selected follow-up: %s (approach: %s)",
		followUp.ID, followUp.SocraticApproach)

	return followUp, followUp.SocraticApproach
}
