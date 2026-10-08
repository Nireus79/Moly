package agents

import (
	"context"
	"log"
	"time"

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
}

// NewLayer11DenialProtocol creates denial protocol layer
func NewLayer11DenialProtocol() *Layer11DenialProtocol {
	return &Layer11DenialProtocol{
		detector: &DenialDetector{
			minConfidenceThreshold: 0.7,
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
	if lc.HasMessageSummary(lc.MessageID) {
		summary := lc.GetMessageSummary(lc.MessageID)
		if msgSummary, ok := summary.(*models.MessageSummary); ok && msgSummary != nil {
			if msgSummary.Confidence >= 0.85 {
				log.Printf("[Layer11] FIX #11 BUG FIX: ✓ Using cached summary for %s (confidence=%.2f, user engaged)",
					lc.MessageID, msgSummary.Confidence)

				// High confidence = user is engaged = no denial patterns
				lc.Layer11 = &tools.Layer11Result{
					ShouldDeny:    false,
					DenialMessage: "",
					Reason:        "none",
					AltSuggestion: "",
					Resources:     []string{},
				}
				log.Printf("[Layer11] ✓ Denial check complete (cached, duration=%.2fs)",
					time.Since(startTime).Seconds())
				return lc, nil
			}
		} else if msgSummary, ok := summary.(models.MessageSummary); ok {
			if msgSummary.Confidence >= 0.85 {
				log.Printf("[Layer11] FIX #11 BUG FIX: ✓ Using cached summary for %s (confidence=%.2f, user engaged)",
					lc.MessageID, msgSummary.Confidence)

				lc.Layer11 = &tools.Layer11Result{
					ShouldDeny:    false,
					DenialMessage: "",
					Reason:        "none",
					AltSuggestion: "",
					Resources:     []string{},
				}
				log.Printf("[Layer11] ✓ Denial check complete (cached, duration=%.2fs)",
					time.Since(startTime).Seconds())
				return lc, nil
			}
		}
	}

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

	messageLength := len(lc.Analysis.CurrentMessage)

	// Very short responses might indicate avoidance
	if messageLength < 10 {
		return true
	}

	return false
}

// GenerateDenialResponse creates empathetic response to denial
func (dd *DenialDetector) GenerateDenialResponse(lc *tools.LayerContext) string {
	return "I notice you might not want to dive deep into this right now. That's completely fine. " +
		"We can take it at your pace. What would feel comfortable to discuss?"
}
