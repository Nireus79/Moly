package agents

import (
	"context"
	"fmt"
	"log"
	"time"

	"moly/tools"
)

// Layer7PrincipleViolationClarification handles messages that possibly violate principles
// REFACTOR: Uses LLM for dynamic clarifying questions before rejecting
type Layer7PrincipleViolationClarification struct {
	clarifier *ViolationClarifier
	llmClient tools.LLMProvider
}

// ViolationClarifier analyzes principle violations
type ViolationClarifier struct {
	minConfidenceThreshold float64
}

// NewLayer7PrincipleViolationClarification creates principle violation layer with LLM support
func NewLayer7PrincipleViolationClarification(llmClient tools.LLMProvider) *Layer7PrincipleViolationClarification {
	return &Layer7PrincipleViolationClarification{
		clarifier: &ViolationClarifier{
			minConfidenceThreshold: 0.7,
		},
		llmClient: llmClient,
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

	// Generate clarifying questions before rejecting (now via LLM)
	log.Printf("[Layer7] Generating clarification questions to understand intent")
	questions := l7.GenerateClarificationQuestions(lc)
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

// GenerateClarificationQuestions creates 3-part principle violation clarification via LLM
// REFACTOR: Dynamic questions contextualized to the specific principle concern and goal
func (l7 *Layer7PrincipleViolationClarification) GenerateClarificationQuestions(lc *tools.LayerContext) []string {
	if l7.llmClient == nil {
		log.Printf("[Layer7] ⚠ No LLM client - using fallback")
		return l7.fallbackQuestions()
	}

	questions := make([]string, 0)

	message := ""
	if lc.Analysis != nil {
		message = lc.Analysis.CurrentMessage
	}

	// Get detected principle violation if available
	violationPrinciple := "respect and consent"
	if lc.Layer2 != nil && lc.Layer2.MatchedPrinciples != nil && len(lc.Layer2.MatchedPrinciples) > 0 {
		violationPrinciple = lc.Layer2.MatchedPrinciples[0]
	}

	parts := []struct {
		name        string
		description string
	}{
		{"intent", "What you're trying to accomplish"},
		{"perspective", "How the other person would feel"},
		{"consequences", "What might happen as a result"},
	}

	for _, part := range parts {
		q := l7.generateLLMQuestion(message, part.name, part.description, violationPrinciple, lc)
		if q != "" {
			questions = append(questions, q)
		}
	}

	log.Printf("[Layer7] ✓ Generated 3-part questions via LLM (principle-specific)")
	return questions
}

// generateLLMQuestion generates ONE contextual question via LLM for principle violation
func (l7 *Layer7PrincipleViolationClarification) generateLLMQuestion(
	message, partName, partDescription, violationPrinciple string,
	lc *tools.LayerContext,
) string {
	contextStr := ""

	if lc.Layer1 != nil && lc.Layer1.ExtractedContext != nil {
		if lc.Layer1.ExtractedContext.Intention != "" {
			contextStr += fmt.Sprintf("Goal: %s\n", lc.Layer1.ExtractedContext.Intention)
		}
	}

	contextStr += fmt.Sprintf("Concern: The approach may violate the principle of %s\n", violationPrinciple)

	prompt := fmt.Sprintf(`You are Moly, a communication coach. A user said:

"%s"

%s
Task: Generate ONE clarifying question about %s (%s) BEFORE making a judgment.

Requirements:
- Conversational (1-2 sentences)
- Show empathy and openness
- Ask about ONE thing only
- No brackets, no lectures
- No preamble

Generate ONLY the question.`, message, contextStr, partName, partDescription)

	resp, err := l7.llmClient.Call(context.Background(), &tools.LLMRequest{
		UserPrompt:  prompt,
		MaxTokens:   100,
		Temperature: 0.7,
	})

	if err != nil {
		log.Printf("[Layer7] LLM error for %s: %v - fallback", partName, err)
		return l7.fallbackQuestionForPart(partName)
	}

	return resp.Content
}

// fallbackQuestions returns static questions if LLM unavailable
func (l7 *Layer7PrincipleViolationClarification) fallbackQuestions() []string {
	return []string{
		"Help me understand your intent here. What are you trying to accomplish?",
		"How do you think the other person would feel about this?",
		"What do you think might happen as a result of this approach?",
	}
}

// fallbackQuestionForPart returns fallback for specific part
func (l7 *Layer7PrincipleViolationClarification) fallbackQuestionForPart(partName string) string {
	fallbacks := map[string]string{
		"intent":        "What are you trying to accomplish?",
		"perspective":   "How do you think they would feel about this?",
		"consequences":  "What might happen as a result?",
	}
	if q, ok := fallbacks[partName]; ok {
		return q
	}
	return "Can you help me understand this better?"
}
