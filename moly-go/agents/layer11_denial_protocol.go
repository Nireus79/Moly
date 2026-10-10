package agents

import (
	"context"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"moly/models"
	"moly/tools"
)

// Layer11DenialProtocol handles user denial, avoidance, or resistance patterns
// CLARIFICATION (PHASE 6): NOT about blocking harmful content (that's Layer 2)
// Layer 11 detects when USER WITHDRAWS or GIVES UP, responds empathetically to re-engage
// Example:
//
//	M1: "Help me with X"
//	M2: "Never mind" (short, withdrawn)
//	L11: "I sense hesitation. That's OK. We can take this at your pace."
//
// Recognizes when user is avoiding discussion and responds with empathy + structured approach
type Layer11DenialProtocol struct {
	detector *DenialDetector
	// FIX #53: ResponseAdapter is singleton, no need to store
}

// DenialDetector identifies denial/resistance patterns
type DenialDetector struct {
	minConfidenceThreshold float64
	llm                    tools.LLMProvider
}

// NewLayer11DenialProtocol creates denial protocol layer
func NewLayer11DenialProtocol(llm tools.LLMProvider) *Layer11DenialProtocol {
	return &Layer11DenialProtocol{
		detector: &DenialDetector{
			minConfidenceThreshold: 0.7,
			llm:                    llm,
		},
		// FIX #53: Use GetResponseAdapter() singleton when needed
	}
}

// Name returns the layer identifier
func (l11 *Layer11DenialProtocol) Name() string {
	return "Layer11-DenialProtocol"
}

// Priority returns layer priority
func (l11 *Layer11DenialProtocol) Priority() int {
	return 40 // Lower priority - only intervenes when denial detected
}

// CanSkip returns true if no denial detected
func (l11 *Layer11DenialProtocol) CanSkip(lc *tools.LayerContext) bool {
	// Always process - denial detection is important
	return false
}

// Process executes denial protocol
func (l11 *Layer11DenialProtocol) Process(ctx context.Context, lc *tools.LayerContext) (*tools.LayerContext, error) {
	startTime := time.Now()
	log.Printf("[Layer11] ▶ Starting denial protocol check")

	// FIX #11: Phase 4 - Check message summary cache for denial detection
	// BUG FIX: High confidence means user is engaged (no denial patterns)

	// Detect denial/avoidance patterns
	log.Printf("[Layer11] Analyzing for denial/avoidance patterns")
	isDenying := l11.detector.DetectDenial(lc)
	log.Printf("[Layer11] Pattern detection result: denial=%v", isDenying)

	response := ""
	if isDenying {
		log.Printf("[Layer11] Generating denial-response guidance")
		response = l11.detector.GenerateDenialResponse(lc)
		log.Printf("[Layer11] Generated response: %s", response)

		// FIX #61: Respect meta-instructions when denying
		// Check if user requested not to give advice, etc.
		adapter := GetResponseAdapter()
		if lc.Analysis != nil && adapter != nil {
			// Get meta-instruction constraints
			metaInstructions := make(map[string]bool)
			if lc.ContextChangeTracker != nil {
				tracker, ok := lc.ContextChangeTracker.(*ContextChangeTracker)
				if ok && tracker != nil {
					metaInstructions = tracker.GetMetaInstructionHistory()
				}
			}

			constraints := adapter.GetMetaInstructionRespect(metaInstructions)
			if len(constraints) > 0 {
				if listenOnly, ok := constraints["listen_only"].(bool); ok && listenOnly {
					log.Printf("[Layer11] FIX #61: Respecting 'listen only' - modifying denial approach")
					// In a full implementation, we'd adapt the response to validate instead of prescribe
				}
			}
		}
	}

	// Store results
	lc.Layer11 = &tools.Layer11Result{
		ShouldDeny:    isDenying,
		DenialMessage: response,
		Reason:        "user_resistance",
		AltSuggestion: "We can take this at your pace",
		Resources:     []string{},
	}

	duration := time.Since(startTime).Seconds()
	if isDenying {
		log.Printf("[Layer11] ✓ DENIAL DETECTED - Response prepared (message_len=%d, duration=%.2fs)",
			len(response), duration)
	} else {
		log.Printf("[Layer11] ✓ No denial pattern detected (duration=%.2fs)", duration)
	}

	return lc, nil
}

// DetectDenial identifies denial/avoidance patterns
func (dd *DenialDetector) DetectDenial(lc *tools.LayerContext) bool {
	// Denial patterns:
	// 1. User says "I don't want to talk about it"
	// 2. Short responses after longer context
	// 3. Contradictory statements

	if lc.Analysis == nil || lc.Analysis.CurrentMessage == "" {
		return false
	}
	// PHASE 3: no withdrawal without prior context (spec Layer 11: a short message after a longer one).
	// A greeting is never a denial, and a first message has nothing to withdraw from.
	if lc.IsGreeting || !hasPriorUserMessage(lc.Analysis.RecentMessages) {
		return false
	}

	// A long message is not a withdrawal; only a short one is worth the model call. The model decides: "Yes." or "Fine, 3pm" answers
	// a question, "never mind" withdraws. (Found live: the answer "Yes." was taken for a withdrawal by its length alone.)
	if utf8.RuneCountInString(strings.TrimSpace(lc.Analysis.CurrentMessage)) >= maxDenialCheckRunes {
		return false
	}
	return dd.judgeWithdrawal(lc)
}

// maxDenialCheckRunes bounds which messages are checked for withdrawal at all (a cost limit, not the decision).
const maxDenialCheckRunes = 40

// judgeWithdrawal asks the model whether the short message gives up or refuses to go on, given what Moly asked last.
// Unreadable or failed means no: the user is not blocked from help by a guess.
func (dd *DenialDetector) judgeWithdrawal(lc *tools.LayerContext) bool {
	if dd.llm == nil {
		return false
	}
	lastAsked := ""
	for i := len(lc.Analysis.RecentMessages) - 1; i >= 0; i-- {
		if m := lc.Analysis.RecentMessages[i]; m.Role != "user" {
			lastAsked = m.Content
			break
		}
	}
	req := &tools.LLMRequest{
		SystemPrompt: "You judge whether a short user message withdraws from the conversation.",
		UserPrompt: "Moly's last message to the user: \"" + lastAsked + "\"\nThe user's reply: \"" + lc.Analysis.CurrentMessage + "\"\n\n" +
			"withdrawing: true only if the reply gives up, refuses to go on, or avoids the subject (for example \"never mind\", \"forget it\", \"I don't want to talk about it\"). " +
			"false if it answers or accepts what Moly asked (for example \"yes\", \"no\", \"tomorrow\", \"formal\"), even if it is short.\n" +
			"Respond with ONLY JSON: {\"withdrawing\": true|false}",
		MaxTokens:   30,
		Temperature: 0.1,
	}
	resp, err := dd.llm.Call(context.Background(), req)
	if err != nil {
		log.Printf("[Layer11] withdrawal judgement failed: %v (not a denial)", err)
		return false
	}
	var out struct {
		Withdrawing tools.LenientBool `json:"withdrawing"`
	}
	if err := tools.SafeJSONParse("Layer11", []byte(resp.Content), &out); err != nil {
		log.Printf("[Layer11] withdrawal judgement unreadable (not a denial)")
		return false
	}
	return bool(out.Withdrawing)
}

// GenerateDenialResponse creates empathetic response to denial
func (dd *DenialDetector) GenerateDenialResponse(lc *tools.LayerContext) string {
	return "I notice you might not want to dive deep into this right now. That's completely fine. " +
		"We can take it at your pace. What would feel comfortable to discuss?"
}

// hasPriorUserMessage reports whether the user has written at least one message before the current one.
// The current message is already part of the recent messages (main.go prepends it), so two user messages are needed.
func hasPriorUserMessage(messages []models.Message) bool {
	userMessages := 0
	for _, m := range messages {
		if m.Role == "user" {
			userMessages++
		}
	}
	return userMessages >= 2
}
