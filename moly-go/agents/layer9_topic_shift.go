package agents

import (
	"context"
	"log"
	"time"

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

// Process executes topic shift detection
func (l9 *Layer9TopicShiftDetection) Process(ctx context.Context, lc *tools.LayerContext) (*tools.LayerContext, error) {
	startTime := time.Now()

	// Detect if topic/contact has shifted
	shifts := l9.detector.DetectShifts(lc)

	// Store results
	lc.Layer9 = &tools.Layer9Result{
		DetectedShifts:      shifts,
		ShiftCount:          len(shifts),
		RequiresContextSwitch: len(shifts) > 0,
	}

	if len(shifts) > 0 {
		log.Printf("[Layer9] 🔄 Topic shift detected (shifts=%d, duration=%.2fs)",
			len(shifts), time.Since(startTime).Seconds())
	} else {
		log.Printf("[Layer9] ✓ No topic shift (duration=%.2fs)", time.Since(startTime).Seconds())
	}

	return lc, nil
}

// DetectShifts identifies topic or contact changes
func (td *TopicShiftDetector) DetectShifts(lc *tools.LayerContext) []tools.TopicShift {
	shifts := make([]tools.TopicShift, 0)

	// Simple heuristic: no data available for detection yet
	// This is a placeholder for future implementation with conversation history

	return shifts
}
