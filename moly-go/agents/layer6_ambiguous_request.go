package agents

import (
	"context"
	"fmt"
	"log"
	"time"

	"moly/tools"
)

// Layer6AmbiguousRequestHandler detects and handles ambiguous requests
// REFACTOR: Uses LLM for dynamic, goal-aligned clarification questions
// Instead of hardcoded "What exactly do you need help with?",
// generates contextual questions based on user's message and extracted intent
type Layer6AmbiguousRequestHandler struct {
	clarifier *AmbiguousDetector
	llmClient tools.LLMProvider
}

// AmbiguousDetector analyzes whether a request is ambiguous
type AmbiguousDetector struct {
	minMaturityToSkip float64
}

// NewLayer6AmbiguousRequestHandler creates ambiguity detection layer with LLM support
func NewLayer6AmbiguousRequestHandler(llmClient tools.LLMProvider) *Layer6AmbiguousRequestHandler {
	return &Layer6AmbiguousRequestHandler{
		clarifier: &AmbiguousDetector{
			minMaturityToSkip: 0.7,
		},
		llmClient: llmClient,
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

	// FIX #11: Phase 3B - Check message summary cache for ambiguity detection
	// Skip ambiguity analysis for cached messages with high confidence
	if lc.HasMessageSummary(lc.MessageID) {
		summary := lc.GetMessageSummary(lc.MessageID)
		if summaryMap, ok := summary.(map[string]interface{}); ok {
			if confidence, ok := summaryMap["confidence"].(float64); ok && confidence >= 0.85 {
				log.Printf("[Layer6] FIX #11: ✓ Using cached summary for %s (confidence=%.2f, skipping ambiguity check)",
					lc.MessageID, confidence)

				// Return cached result - no ambiguity for cached high-confidence messages
				lc.Layer6 = &tools.Layer6Result{
					IsAmbiguous:            false,
					AmbiguousElements:      []string{},
					ClarificationQuestions: []string{},
					ShouldProceedToResponse: true,
				}
				log.Printf("[Layer6] ✓ Ambiguity check complete (cached, duration=%.2fs)",
					time.Since(startTime).Seconds())
				return lc, nil
			}
		}
	}

	// Detect if request is ambiguous
	isAmbiguous := l6.clarifier.IsAmbiguous(lc)

	clarificationQuestions := make([]string, 0)
	if isAmbiguous {
		clarificationQuestions = l6.GenerateClarificationQuestions(lc)
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

// GenerateClarificationQuestions generates 4-part clarification via LLM
// REFACTOR: Dynamic, contextual questions based on actual user message and goal
func (l6 *Layer6AmbiguousRequestHandler) GenerateClarificationQuestions(lc *tools.LayerContext) []string {
	if l6.llmClient == nil {
		log.Printf("[Layer6] ⚠ No LLM client - using fallback")
		return l6.fallbackQuestions()
	}

	questions := make([]string, 0)

	message := ""
	if lc.Analysis != nil {
		message = lc.Analysis.CurrentMessage
	}

	parts := []struct {
		name        string
		description string
	}{
		{"intent", "What they're trying to accomplish"},
		{"context", "Details about the situation"},
		{"parties", "Who else is involved"},
		{"outcome", "What success looks like"},
	}

	for _, part := range parts {
		q := l6.generateLLMQuestion(message, part.name, part.description, lc)
		if q != "" {
			questions = append(questions, q)
		}
	}

	log.Printf("[Layer6] ✓ Generated 4-part questions via LLM (goal-aligned, contextual)")
	return questions
}

// generateLLMQuestion generates ONE contextual question via LLM
func (l6 *Layer6AmbiguousRequestHandler) generateLLMQuestion(
	message, partName, partDescription string,
	lc *tools.LayerContext,
) string {
	contextStr := ""

	if lc.Layer1 != nil && lc.Layer1.ExtractedContext != nil {
		if lc.Layer1.ExtractedContext.Intention != "" {
			contextStr += fmt.Sprintf("Goal: %s\n", lc.Layer1.ExtractedContext.Intention)
		}
	}

	if lc.PreviousGoal != "" {
		contextStr += fmt.Sprintf("Previous goal: %s\n", lc.PreviousGoal)
	}

	if len(lc.AccumulatedExtractedEntities) > 0 {
		contextStr += fmt.Sprintf("(Accumulated %d entities from previous messages)\n", len(lc.AccumulatedExtractedEntities))
	}

	prompt := fmt.Sprintf(`You are Moly, a communication coach. User said:

"%s"

%s
Task: Generate ONE clarifying question about %s (%s).

Requirements:
- Conversational (1-2 sentences)
- Show you understand their situation
- Ask about ONE thing only
- No brackets, no multiple choice
- No preamble

Generate ONLY the question.`, message, contextStr, partName, partDescription)

	resp, err := l6.llmClient.Call(context.Background(), &tools.LLMRequest{
		UserPrompt:  prompt,
		MaxTokens:   100,
		Temperature: 0.7,
	})

	if err != nil {
		log.Printf("[Layer6] LLM error for %s: %v - fallback", partName, err)
		return l6.fallbackQuestionForPart(partName)
	}

	return resp.Content
}

// fallbackQuestions returns static questions if LLM unavailable
func (l6 *Layer6AmbiguousRequestHandler) fallbackQuestions() []string {
	return []string{
		"Help me understand what you're asking for. What exactly do you need help with?",
		"Can you give me more context about the situation?",
		"Who else is involved in this situation?",
		"What outcome are you hoping for?",
	}
}

// fallbackQuestionForPart returns fallback for specific part
func (l6 *Layer6AmbiguousRequestHandler) fallbackQuestionForPart(partName string) string {
	fallbacks := map[string]string{
		"intent":  "What exactly are you trying to accomplish?",
		"context": "Can you tell me more about the situation?",
		"parties": "Who else is involved?",
		"outcome": "What would success look like?",
	}
	if q, ok := fallbacks[partName]; ok {
		return q
	}
	return "Can you tell me more?"
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
