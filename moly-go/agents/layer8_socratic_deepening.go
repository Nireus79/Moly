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

// CanSkip returns true if prerequisites not met
// PHASE 5: Check all 4 prerequisites per spec before Socratic questions
func (l8 *Layer8SocraticDeepening) CanSkip(lc *tools.LayerContext) bool {
	if lc.IsGreeting {
		return true // PHASE 3: a greeting has no goal and no gaps
	}
	// Prerequisite 1: Maturity ≥ 0.5
	if lc.Layer3 != nil && lc.Layer3.MaturityScore < l8.questioner.minMaturityRequired {
		log.Printf("[Layer8] Skipping: Maturity too low (%.2f < %.2f)",
			lc.Layer3.MaturityScore, l8.questioner.minMaturityRequired)
		return true
	}

	// Prerequisite 3: No unresolved principle concerns. Layer 7 used to mark this; it was removed, its trigger was exactly
	// "Layer 2 matched a principle", so the skip keeps that condition.
	if lc.Layer2 != nil && len(lc.Layer2.MatchedPrinciples) > 0 {
		log.Printf("[Layer8] Skipping: Principle violation still flagged")
		return true
	}

	// Prerequisite 4: No critical gaps (ANY gap means prerequisites not met)
	// FIXED: Changed from > 2 to > 0 - even 1 critical gap means clarification needed first
	if lc.Layer4 != nil && lc.Layer4.CriticalGaps != nil && len(lc.Layer4.CriticalGaps) > 0 {
		log.Printf("[Layer8] Skipping: Critical gaps found (%d) - need clarification first",
			len(lc.Layer4.CriticalGaps))
		return true
	}

	// All prerequisites met
	log.Printf("[Layer8] ✓ All 4 prerequisites passed - ready for Socratic deepening")
	return false
}

// Process executes Socratic questioning
func (l8 *Layer8SocraticDeepening) Process(ctx context.Context, lc *tools.LayerContext) (*tools.LayerContext, error) {
	startTime := time.Now()

	// FIX #11: Phase 4 - Check message summary cache for Socratic questions
	// BUG FIX: High confidence means mature context (no need for deepening questions)

	// Generate Socratic questions based on context
	questions := l8.questioner.GenerateSocraticAims(lc)

	// Store results
	depthLevel := calculateDepthLevel(lc)
	depth := "surface"
	if depthLevel == 2 {
		depth = "moderate"
	} else if depthLevel == 3 {
		depth = "deep"
	}

	lc.Layer8 = &tools.Layer8Result{
		SocraticAims:      questions,
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

// GenerateSocraticAims says what the Socratic question should do. It returns aims, not questions: a fixed English sentence with the user's
// goal pasted into it was shown to users of other languages (tester report, 2026-10-29). The reply writes the question from the aim,
// in the language of the user's own words (reply_exits.go, KindSocratic).
func (sq *SocraticQuestioner) GenerateSocraticAims(lc *tools.LayerContext) []string {
	aims := make([]string, 0)

	var userGoal string
	var userValues []string
	if lc.Layer1 != nil && lc.Layer1.ExtractedContext != nil {
		userGoal = lc.Layer1.ExtractedContext.Intention
		userValues = lc.Layer1.ExtractedContext.UserValues
	}
	about := "this situation"
	if len(userGoal) > 0 {
		about = "the user's goal (\"" + userGoal + "\")"
	}

	if lc.Layer3 != nil && lc.Layer3.MaturityScore > 0.7 {
		aims = append(aims, "ask what success would look like for "+about)
		aims = append(aims, "ask what core value or principle is at stake in "+about)
		if len(userValues) > 0 {
			aims = append(aims, "ask how the user's value of \""+userValues[0]+"\" guides their approach to "+about)
		}
	}
	if lc.Layer3 != nil && lc.Layer3.MaturityScore > 0.5 {
		aims = append(aims, "ask what one assumption the user is making about "+about)
		aims = append(aims, "ask how the other person might see "+about+" differently")
	}

	if len(aims) > 3 {
		return aims[:3]
	}
	return aims
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
