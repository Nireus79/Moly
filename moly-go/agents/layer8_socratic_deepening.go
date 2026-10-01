package agents

import (
	"context"
	"log"
	"time"

	"moly/tools"
)

// Layer8SocraticDeepening asks philosophical questions to explore user perspective
// Socratic method: guide user to discover insights through questions, not statements
type Layer8SocraticDeepening struct {
	questioner *SocraticQuestioner
}

// SocraticQuestioner generates Socratic questions
type SocraticQuestioner struct {
	minMaturityRequired float64 // Only ask if mature enough
}

// NewLayer8SocraticDeepening creates Socratic questioning layer
func NewLayer8SocraticDeepening() *Layer8SocraticDeepening {
	return &Layer8SocraticDeepening{
		questioner: &SocraticQuestioner{
			minMaturityRequired: 0.5, // Requires 50% maturity
		},
	}
}

// Name returns the layer identifier
func (l8 *Layer8SocraticDeepening) Name() string {
	return "Layer8-SocraticDeepening"
}

// Priority returns layer priority
func (l8 *Layer8SocraticDeepening) Priority() int {
	return 50 // Medium-low priority
}

// CanSkip returns true if not enough maturity or too many gaps
func (l8 *Layer8SocraticDeepening) CanSkip(lc *tools.LayerContext) bool {
	// Skip if immature
	if lc.Layer3 != nil && lc.Layer3.MaturityScore < l8.questioner.minMaturityRequired {
		return true
	}

	// Skip if too many unresolved gaps
	if lc.Layer4 != nil && lc.Layer4.CriticalGaps != nil && len(lc.Layer4.CriticalGaps) > 2 {
		return true
	}

	// Skip if ambiguous request not yet resolved
	if lc.Layer6 != nil && lc.Layer6.IsAmbiguous {
		return true
	}

	return false
}

// Process executes Socratic questioning
func (l8 *Layer8SocraticDeepening) Process(ctx context.Context, lc *tools.LayerContext) (*tools.LayerContext, error) {
	startTime := time.Now()

	// Generate Socratic questions based on context
	questions := l8.questioner.GenerateSocraticQuestions(lc)

	// Store results
	depthLevel := calculateDepthLevel(lc)
	depth := "surface"
	if depthLevel == 2 {
		depth = "moderate"
	} else if depthLevel == 3 {
		depth = "deep"
	}

	lc.Layer8 = &tools.Layer8Result{
		SocraticQuestions: questions,
		Depth:             depth,
		QuestionStrategy:  "explore_values",
	}

	if len(questions) > 0 {
		log.Printf("[Layer8] 🤔 Generated %d Socratic questions (depth=%s, duration=%.2fs)",
			len(questions), depth, time.Since(startTime).Seconds())
	} else {
		log.Printf("[Layer8] ✓ No Socratic questions needed (duration=%.2fs)", time.Since(startTime).Seconds())
	}

	return lc, nil
}

// GenerateSocraticQuestions generates philosophical questions
func (sq *SocraticQuestioner) GenerateSocraticQuestions(lc *tools.LayerContext) []string {
	questions := make([]string, 0)

	// Question depth depends on maturity and context
	if lc.Layer3 != nil && lc.Layer3.MaturityScore > 0.7 {
		// Deep questions for mature context
		questions = append(questions, "What would it look like if this situation changed?")
		questions = append(questions, "What's the core value or principle at stake here?")
		questions = append(questions, "How does this relate to what matters most to you?")
	}

	if lc.Layer3 != nil && lc.Layer3.MaturityScore > 0.5 {
		// Medium depth for developing context
		questions = append(questions, "What's one assumption you're making about this?")
		questions = append(questions, "How might someone else see this differently?")
	}

	// Limit to 3 questions
	if len(questions) > 3 {
		return questions[:3]
	}

	return questions
}

// Helper: Calculate conversation depth level
func calculateDepthLevel(lc *tools.LayerContext) int {
	depth := 1

	if lc.Layer3 != nil && lc.Layer3.MaturityScore > 0.3 {
		depth = 2
	}
	if lc.Layer3 != nil && lc.Layer3.MaturityScore > 0.6 {
		depth = 3
	}

	return depth
}
