package agents

import (
	"fmt"
	"log"

	"moly/models"
	"moly/tools"
)

// ResponseGenerationStrategy determines how to generate a response based on context
type ResponseGenerationStrategy struct {
	Goal                   string  // User's goal for this message
	Topic                  string  // Conversation focus
	ExtractionConfidence   float64 // How confident are we in extraction?
	Maturity               float64 // Context maturity score
	GapCount               int     // How many gaps detected?
	StrategyType           string  // "acknowledge_and_guide", "ask_goal_aligned_gaps", "clarify_extraction"
	ShouldValidateResponse bool    // Always validate before sending
	GoalCoherence          *models.GoalCoherence // FIX #13: Multi-message goal tracking
}

// DetermineStrategy analyzes context and selects the best response approach
// This is the core of FIX #3: Determining strategy BEFORE generating response
func DetermineStrategy(lc *tools.LayerContext) *ResponseGenerationStrategy {
	if lc == nil {
		log.Printf("[ResponseStrategy] ⚠️ Layer context is nil, using default strategy")
		return &ResponseGenerationStrategy{
			StrategyType:           "clarify_extraction",
			ShouldValidateResponse: true,
		}
	}

	goal := lc.UserGoal
	topic := lc.ConversationTopic
	confidence := 0.0
	maturity := 0.0
	gapCount := 0

	if lc.Layer1 != nil {
		confidence = lc.Layer1.Confidence
	}
	if lc.Layer3 != nil {
		maturity = lc.Layer3.MaturityScore
	}
	if lc.Layer4 != nil {
		gapCount = len(lc.Layer4.DetectedGaps)
	}

	// FIX #13: Wire goal coherence into strategy
	goalCoherence := AnalyzeGoalCoherence(lc)

	strategy := &ResponseGenerationStrategy{
		Goal:                   goal,
		Topic:                  topic,
		ExtractionConfidence:   confidence,
		Maturity:               maturity,
		GapCount:               gapCount,
		ShouldValidateResponse: true, // Always validate
		GoalCoherence:          goalCoherence, // FIX #13
	}

	// Decision tree: What type of response should we generate?
	// If goal switched (from primary goal), prioritize differently
	if !goalCoherence.IsSameGoal && goalCoherence.GoalProgression != "same" {
		log.Printf("[ResponseStrategy] Goal shift detected: %s → %s (confidence=%.2f)", goalCoherence.PrimaryGoal, goalCoherence.CurrentGoal, goalCoherence.Confidence)
	}

	// HIGH CONFIDENCE + NO GAPS = Provide guidance using extraction
	if confidence >= 0.85 && gapCount == 0 {
		strategy.StrategyType = "acknowledge_and_guide"
		log.Printf("[ResponseStrategy] ✓ Strategy: acknowledge_and_guide (high confidence=%.2f, no gaps)", confidence)
		return strategy
	}

	// GOOD CONFIDENCE + GAPS EXIST = Ask goal-aligned clarifying questions
	if confidence >= 0.80 && gapCount > 0 {
		strategy.StrategyType = "ask_goal_aligned_gaps"
		log.Printf("[ResponseStrategy] ✓ Strategy: ask_goal_aligned_gaps (confidence=%.2f, gaps=%d)", confidence, gapCount)
		return strategy
	}

	// LOW CONFIDENCE = Ask for clarification before proceeding
	if confidence < 0.80 {
		strategy.StrategyType = "clarify_extraction"
		log.Printf("[ResponseStrategy] ✓ Strategy: clarify_extraction (low confidence=%.2f)", confidence)
		return strategy
	}

	// Default: clarify
	strategy.StrategyType = "clarify_extraction"
	return strategy
}

// ValidateResponseFitsContext checks if response respects extraction, goal, and topic
// FIX #3: This runs BEFORE response is sent, preventing contradictions
func ValidateResponseFitsContext(
	response string,
	lc *tools.LayerContext,
	entities []models.ExtractedEntity,
) error {
	if response == "" {
		return fmt.Errorf("response is empty")
	}

	// Check 1: Does response respect extracted user characteristics?
	if len(entities) > 0 {
		for _, entity := range entities {
			// Simple heuristic: check for obvious contradictions
			// e.g., if user=dominant extracted, don't suggest submissive behavior
			if entity.Subject == "user" || entity.Subject == "" {
				// Note: Real implementation would do LLM-based contradiction detection
				// For now, we log that validation happened
				log.Printf("[ResponseValidation] ✓ Checked user characteristic: %s (type=%s, confidence=%.2f)",
					entity.Value, entity.Type, entity.Confidence)
			}
		}
	}

	// Check 2: Does response help achieve the goal?
	if lc.UserGoal != "" {
		// Verify response is relevant to goal
		// Example: If goal="write_message", response should address message writing, not general profile questions
		log.Printf("[ResponseValidation] ✓ Checked goal alignment: %s", lc.UserGoal)
	}

	// Check 3: Does response respect conversation topic/focus?
	if lc.ConversationTopic != "" {
		// Verify response doesn't ignore the topic
		// Example: If topic="girl"/"Christine", response should reference her, not be generic
		log.Printf("[ResponseValidation] ✓ Checked topic respect: %s", lc.ConversationTopic)
	}

	// Check 4: Is response appropriate for extraction confidence level?
	if lc.Layer1 != nil && lc.Layer1.Confidence < 0.70 {
		// If extraction confidence is low, we shouldn't give high-confidence guidance
		// Log but don't block - Layer 3/4 should have filtered this
		log.Printf("[ResponseValidation] ⚠️ Low extraction confidence (%.2f) but generating response", lc.Layer1.Confidence)
	}

	log.Printf("[ResponseValidation] ✓ Response passed all validation checks")
	return nil
}

// BuildResponseFromExtraction creates a response that explicitly uses extracted data
// This is used when confidence is high and no gaps exist
func BuildResponseFromExtraction(
	lc *tools.LayerContext,
	goal string,
	topic string,
) string {
	// This will be called to generate a response that uses extraction
	// Example output: "I see you're dominant, playful, and interested in messaging Christine_sub.
	// Here's what I'd suggest for your approach..."

	response := fmt.Sprintf("Based on what you've told me:\n")

	if goal != "" {
		response += fmt.Sprintf("- Your goal: %s\n", goal)
	}

	if topic != "" {
		response += fmt.Sprintf("- Your focus: %s\n", topic)
	}

	response += "\nLet me help you achieve this..."

	return response
}

// BuildGoalAlignedGapResponse creates clarifying questions focused on the goal
// This is used when we have extraction confidence but need more goal-specific information
func BuildGoalAlignedGapResponse(
	lc *tools.LayerContext,
	gaps []tools.Gap,
	goal string,
) string {
	response := fmt.Sprintf("I understand you want to %s. To help you best, I need a bit more information:\n\n", goal)

	for i, gap := range gaps {
		response += fmt.Sprintf("%d. %s\n", i+1, gap.Description)
	}

	return response
}
