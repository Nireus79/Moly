package agents

import (
	"context"
	"log"
	"time"

	"moly/models"
	"moly/tools"
)

// Layer9TopicShiftDetection detects when user changes topics or who they're discussing
// Helps maintain conversation continuity and prevents context loss
type Layer9TopicShiftDetection struct {
	detector *TopicShiftDetector
}

// TopicShiftDetector analyzes topic/contact shifts
type TopicShiftDetector struct {
	minShiftConfidence float64
}

// NewLayer9TopicShiftDetection creates topic shift detection layer
func NewLayer9TopicShiftDetection() *Layer9TopicShiftDetection {
	return &Layer9TopicShiftDetection{
		detector: &TopicShiftDetector{
			minShiftConfidence: 0.6,
		},
	}
}

// Name returns the layer identifier
func (l9 *Layer9TopicShiftDetection) Name() string {
	return "Layer9-TopicShift"
}

// Priority returns layer priority
func (l9 *Layer9TopicShiftDetection) Priority() int {
	return 55 // Medium-low priority
}

// CanSkip returns true if not enough context to detect shifts
func (l9 *Layer9TopicShiftDetection) CanSkip(lc *tools.LayerContext) bool {
	// Skip if gaps exist (don't detect shifts while clarifying)
	if lc.Layer4 != nil && lc.Layer4.GapCount > 0 {
		return true
	}

	return false
}

// Process executes topic shift detection and context reset
func (l9 *Layer9TopicShiftDetection) Process(ctx context.Context, lc *tools.LayerContext) (*tools.LayerContext, error) {
	startTime := time.Now()

	// Detect if topic/contact has shifted
	shifts := l9.detector.DetectShifts(lc)

	// PHASE 6: Reset context when topic shift detected
	contextReset := false
	if len(shifts) > 0 {
		log.Printf("[Layer9] 🔄 Topic shift detected (shifts=%d) - clearing accumulated context",
			len(shifts))

		// Clear accumulated context from previous topic
		lc.AccumulatedExtractedEntities = []models.ExtractedEntity{}
		lc.PreviousGoal = ""
		lc.PreviousValues = []string{}

		contextReset = true

		log.Printf("[Layer9] ✓ Context reset for new topic")
	}

	// Store results
	lc.Layer9 = &tools.Layer9Result{
		DetectedShifts:        shifts,
		ShiftCount:            len(shifts),
		RequiresContextSwitch: len(shifts) > 0,
		ShouldResetContext:    contextReset,
	}

	if len(shifts) > 0 {
		log.Printf("[Layer9] ✓ Topic shift handled (shifts=%d, context_reset=%v, duration=%.2fs)",
			len(shifts), contextReset, time.Since(startTime).Seconds())
	} else {
		log.Printf("[Layer9] ✓ No topic shift (duration=%.2fs)", time.Since(startTime).Seconds())
	}

	return lc, nil
}

// DetectShifts identifies topic or contact changes
func (td *TopicShiftDetector) DetectShifts(lc *tools.LayerContext) []tools.TopicShift {
	shifts := make([]tools.TopicShift, 0)

	if lc.Analysis == nil {
		return shifts
	}

	// Get previous contacts from relevant contacts list
	prevContactMap := make(map[string]bool)
	for _, contact := range lc.Analysis.RelevantContacts {
		prevContactMap[contact.Name] = true
	}

	// Detect contact shifts (user switched to talking about someone else)
	currentContactMap := make(map[string]bool)
	for _, contact := range lc.Analysis.Contacts {
		currentContactMap[contact.Name] = true
	}

	// Find contacts that were discussed before but not now (shift detected)
	for prevContact := range prevContactMap {
		if !currentContactMap[prevContact] && len(lc.Analysis.Contacts) > 0 {
			shifts = append(shifts, tools.TopicShift{
				Type:       "contact_change",
				Severity:   "medium",
				Confidence: 0.7,
			})
			break // Only report one shift per message
		}
	}

	// Detect characteristic/goal shifts (may indicate topic change)
	if lc.Layer1 != nil && lc.Layer1.ExtractedContext != nil {
		// Simple heuristic: if new characteristics are extracted,
		// it may indicate a topic or context shift
		if len(lc.Analysis.ExtractedCharacteristics) > 0 &&
			len(lc.Analysis.UserProfile.CommunicationStyle) > 0 {
			log.Printf("[Layer9] Detected potential characteristic change (may indicate topic shift)")
		}
	}

	return shifts
}
