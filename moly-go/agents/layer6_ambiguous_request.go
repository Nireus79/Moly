package agents

import (
	"context"
	"log"
	"time"

	"moly/tools"
)

// Layer6AmbiguousRequestHandler detects and handles ambiguous requests
// PHASE 6: Boundary with Layer 4: Layer 6 asks WHAT (clarify goal/intent),
// Layer 4 asks about gaps GIVEN a clear goal
// Example:
//   M: "Help with the girl" (ambiguous)
//   L6: "Help with dating, conflict, something else?" (clarify WHAT)
//   L4: "Who is the girl? How do you know her?" (fill gaps in known goal)
// Ambiguous = unclear intention, unclear subject, multiple valid interpretations
type Layer6AmbiguousRequestHandler struct {
	clarifier *AmbiguousDetector
}

// AmbiguousDetector analyzes whether a request is ambiguous
type AmbiguousDetector struct {
	minMaturityToSkip float64 // Skip Layer 6 if maturity >= this
}

// NewLayer6AmbiguousRequestHandler creates a new ambiguity detection layer
func NewLayer6AmbiguousRequestHandler() *Layer6AmbiguousRequestHandler {
	return &Layer6AmbiguousRequestHandler{
		clarifier: &AmbiguousDetector{
			minMaturityToSkip: 0.7, // Skip if mature context
		},
	}
}

// Name returns the layer identifier
func (l6 *Layer6AmbiguousRequestHandler) Name() string {
	return "Layer6-AmbiguousRequest"
}

// Priority returns layer priority
func (l6 *Layer6AmbiguousRequestHandler) Priority() int {
	return 65 // Medium priority - ambiguity requires clarification but doesn't block
}

// CanSkip returns true if context is mature (no need to ask for clarification)
func (l6 *Layer6AmbiguousRequestHandler) CanSkip(lc *tools.LayerContext) bool {
	// Skip if we have mature context
	if lc.Layer3 != nil && lc.Layer3.MaturityScore >= l6.clarifier.minMaturityToSkip {
		return true
	}

	// Skip if no gaps (clear context)
	if lc.Layer4 != nil && lc.Layer4.GapCount == 0 {
		return true
	}

	return false
}

// Process executes ambiguity detection
func (l6 *Layer6AmbiguousRequestHandler) Process(ctx context.Context, lc *tools.LayerContext) (*tools.LayerContext, error) {
	startTime := time.Now()

	// Detect if request is ambiguous
	isAmbiguous := l6.clarifier.IsAmbiguous(lc)

	clarificationQuestions := make([]string, 0)
	if isAmbiguous {
		// Generate clarification questions
		clarificationQuestions = l6.clarifier.GenerateClarificationQuestions(lc)
	}

	// Store results
	lc.Layer6 = &tools.Layer6Result{
		IsAmbiguous:            isAmbiguous,
		AmbiguousElements:      detectAmbiguousElements(lc),
		ClarificationQuestions: clarificationQuestions,
		ShouldProceedToResponse: !isAmbiguous,
	}

	if isAmbiguous {
		log.Printf("[Layer6] ⚠️ Ambiguous request detected (questions=%d, duration=%.2fs)",
			len(clarificationQuestions), time.Since(startTime).Seconds())
	} else {
		log.Printf("[Layer6] ✓ Request is clear, proceeding (duration=%.2fs)", time.Since(startTime).Seconds())
	}

	return lc, nil
}

// IsAmbiguous determines if request is ambiguous
// NOTE: Gaps are NOT the same as ambiguous!
// - Gap: Missing context (Layer 4 asks clarification)
// - Ambiguous: Unclear intention / multiple interpretations (Layer 6 checks)
func (ad *AmbiguousDetector) IsAmbiguous(lc *tools.LayerContext) bool {
	// DO NOT check gaps here - that's Layer 4's responsibility
	// Gaps mean "I need more context" not "Your request is unclear"
	// Example: "Help me write a message" is CLEAR and UNAMBIGUOUS
	// even if we have gaps about profile, communication style, etc.

	// Ambiguous if REQUEST itself has multiple possible interpretations
	if lc.Analysis != nil && lc.Analysis.ExtractedEntities != nil {
		ambiguousCount := 0
		for _, entity := range lc.Analysis.ExtractedEntities {
			if entity.IsAmbiguous {
				ambiguousCount++
			}
		}
		if ambiguousCount > 0 {
			return true
		}
	}

	// Ambiguous if extraction confidence is too low to understand intent
	if lc.GetExtractionConfidence() < 0.6 {
		return true
	}

	return false
}

// GenerateClarificationQuestions generates 4-part ambiguity clarification per spec
// PHASE 2: Ask intent → context → affected parties → desired outcome
func (ad *AmbiguousDetector) GenerateClarificationQuestions(lc *tools.LayerContext) []string {
	questions := make([]string, 0)

	// PHASE 2: 4-part structured clarification per MOLY_11_LAYER_SYSTEM.md spec

	// Part 1: Intent - What exactly is user asking for?
	questions = append(questions,
		"Help me understand what you're asking for. What exactly do you need help with?")

	// Part 2: Context - Give more information about the situation
	questions = append(questions,
		"Can you give me more context about the situation?")

	// Part 3: Affected parties - Who else is involved?
	questions = append(questions,
		"Who else is involved in this situation?")

	// Part 4: Desired outcome - What outcome are you hoping for?
	questions = append(questions,
		"What outcome are you hoping for?")

	log.Printf("[Layer6] ℹ Generated 4-part clarification questions (PHASE 2: intent, context, parties, outcome)")

	return questions
}

// Helper: Detect which elements are ambiguous
func detectAmbiguousElements(lc *tools.LayerContext) []string {
	elements := make([]string, 0)

	if lc.Analysis != nil && lc.Analysis.ExtractedEntities != nil {
		for _, entity := range lc.Analysis.ExtractedEntities {
			if entity.IsAmbiguous {
				elements = append(elements, entity.Value)
			}
		}
	}

	return elements
}
