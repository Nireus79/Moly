package agents

import (
	"log"

	"moly/models"
	"moly/tools"
)

// ResponseGenerationStrategy determines how to generate a response based on context
type ResponseGenerationStrategy struct {
	Goal                   string                // User's goal for this message
	Topic                  string                // Conversation focus
	ExtractionConfidence   float64               // How confident are we in extraction?
	Maturity               float64               // Context maturity score
	GapCount               int                   // How many gaps detected?
	StrategyType           string                // "acknowledge_and_guide", "ask_goal_aligned_gaps", "clarify_extraction"
	ShouldValidateResponse bool                  // Always validate before sending
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
		ShouldValidateResponse: true,          // Always validate
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
