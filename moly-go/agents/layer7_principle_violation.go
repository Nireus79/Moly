package agents

import (
	"context"
	"log"
	"time"

	"moly/tools"
)

// Layer7PrincipleViolationClarification handles messages that possibly violate principles
// Asks clarifying questions before rejecting to understand user intent
type Layer7PrincipleViolationClarification struct {
	clarifier *ViolationClarifier
}

// ViolationClarifier analyzes principle violations
type ViolationClarifier struct {
	minConfidenceThreshold float64
}

// NewLayer7PrincipleViolationClarification creates principle violation layer
func NewLayer7PrincipleViolationClarification() *Layer7PrincipleViolationClarification {
	return &Layer7PrincipleViolationClarification{
		clarifier: &ViolationClarifier{
			minConfidenceThreshold: 0.7,
		},
	}
}

// Name returns the layer identifier
func (l7 *Layer7PrincipleViolationClarification) Name() string {
	return "Layer7-PrincipleViolation"
}

// Priority returns layer priority
func (l7 *Layer7PrincipleViolationClarification) Priority() int {
	return 75 // High priority - principle violations need attention
}

// CanSkip returns false if Layer 2 flagged principles, true otherwise
func (l7 *Layer7PrincipleViolationClarification) CanSkip(lc *tools.LayerContext) bool {
	// Skip if Layer 2 didn't flag any issues
	if lc.Layer2 == nil || !lc.Layer2.IsObviousHarm {
		return true
	}

	return false
}

// Process executes principle violation clarification
func (l7 *Layer7PrincipleViolationClarification) Process(ctx context.Context, lc *tools.LayerContext) (*tools.LayerContext, error) {
	startTime := time.Now()
	log.Printf("[Layer7] ▶ Checking for principle violations")

	// Generate clarifying questions before rejecting
	log.Printf("[Layer7] Generating clarification questions to understand intent")
	questions := l7.clarifier.GenerateClarificationQuestions(lc)
	log.Printf("[Layer7] Generated %d clarification questions", len(questions))

	// Store results
	lc.Layer7 = &tools.Layer7Result{
		ViolationDetected:      true,
		ClarificationQuestions: questions,
		ShouldAskBeforeReject:  len(questions) > 0,
	}

	duration := time.Since(startTime).Seconds()
	log.Printf("[Layer7] ✓ Layer7 complete (violation_detected=true, ask_questions=%v, questions=%d, duration=%.2fs)",
		len(questions) > 0, len(questions), duration)

	return lc, nil
}

// GenerateClarificationQuestions creates 3-part principle violation clarification per spec
// PHASE 3: Ask (1) intent, (2) affected person's perspective, (3) consequences
func (vc *ViolationClarifier) GenerateClarificationQuestions(lc *tools.LayerContext) []string {
	questions := make([]string, 0)

	// Extract goal from Layer 1 if available
	var userGoal string
	if lc.Layer1 != nil && lc.Layer1.ExtractedContext != nil {
		userGoal = lc.Layer1.ExtractedContext.Intention
	}

	// PHASE 3: 3-part structured clarification per MOLY_11_LAYER_SYSTEM.md spec

	// Part 1: Intent - What are they trying to accomplish?
	if len(userGoal) > 0 {
		questions = append(questions,
			"Help me understand your intent for "+userGoal+". What are you trying to accomplish?")
	} else {
		questions = append(questions,
			"Help me understand your intent here. What are you trying to accomplish?")
	}

	// Part 2: Affected person's perspective - How would they feel?
	questions = append(questions,
		"How do you think the other person would feel about this?")

	// Part 3: Consequences - What might happen as a result?
	questions = append(questions,
		"What do you think might happen as a result of this approach?")

	log.Printf("[Layer7] ℹ Generated 3-part clarification questions (PHASE 3: intent, affected view, consequences)")

	return questions
}
