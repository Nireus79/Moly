package agents

import (
	"context"
	"fmt"
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
		// FIX #7: Log skip reason for debugging
		log.Printf("[Layer9] ⏭ SKIP: Gaps detected (%d gaps) - skip topic shift detection during clarification", lc.Layer4.GapCount)
		return true
	}

	return false
}

// Process executes topic shift detection and context reset
func (l9 *Layer9TopicShiftDetection) Process(ctx context.Context, lc *tools.LayerContext) (*tools.LayerContext, error) {
	startTime := time.Now()

	// FIX #11: Phase 4 - Check message summary cache for topic shift detection
	// BUG FIX: High confidence means consistent topic (no shift)
	if lc.HasMessageSummary(lc.MessageID) {
		summary := lc.GetMessageSummary(lc.MessageID)
		if msgSummary, ok := summary.(*models.MessageSummary); ok && msgSummary != nil {
			if msgSummary.Confidence >= 0.85 {
				log.Printf("[Layer9] FIX #11 BUG FIX: ✓ Using cached summary for %s (confidence=%.2f, consistent topic)",
					lc.MessageID, msgSummary.Confidence)

				// High confidence = consistent topic = no shifts
				lc.Layer9 = &tools.Layer9Result{
					DetectedShifts:        []tools.TopicShift{},
					ShiftCount:            0,
					RequiresContextSwitch: false,
					TopicShifted:          false,
					ContactShifted:        false,
				}
				log.Printf("[Layer9] ✓ Topic shift detection complete (cached, duration=%.2fs)",
					time.Since(startTime).Seconds())
				return lc, nil
			}
		} else if msgSummary, ok := summary.(models.MessageSummary); ok {
			if msgSummary.Confidence >= 0.85 {
				log.Printf("[Layer9] FIX #11 BUG FIX: ✓ Using cached summary for %s (confidence=%.2f, consistent topic)",
					lc.MessageID, msgSummary.Confidence)

				lc.Layer9 = &tools.Layer9Result{
					DetectedShifts:        []tools.TopicShift{},
					ShiftCount:            0,
					RequiresContextSwitch: false,
					TopicShifted:          false,
					ContactShifted:        false,
				}
				log.Printf("[Layer9] ✓ Topic shift detection complete (cached, duration=%.2fs)",
					time.Since(startTime).Seconds())
				return lc, nil
			}
		}
	}

	// Detect if topic/contact has shifted
	shifts := l9.detector.DetectShifts(lc)

	// FIX #57: When topic shift detected, create actual clarification gaps
	contextReset := false
	clarificationGaps := []tools.Gap{}

	if len(shifts) > 0 {
		log.Printf("[Layer9] 🚨 FIX #57: Topic shift detected (shifts=%d) - creating clarification gaps",
			len(shifts))

		// FIX #57: Create actual Gap objects for Layer 4
		if lc.Analysis != nil && len(lc.Analysis.RelevantContacts) > 0 {
			newContact := lc.Analysis.RelevantContacts[0].Name
			previousContact := getPreviousPrimaryContact(lc.Analysis.RelevantContacts)

			clarificationText := fmt.Sprintf(
				"I notice we've switched from discussing %s to discussing %s. Are you sure you want to change topics?",
				previousContact,
				newContact,
			)

			// Create Gap object (FIX #57)
			// FIX #72 Phase 2: Tag with GoalTarget
			gap := tools.Gap{
				Type:        "topic_shift",
				Description: clarificationText,
				Severity:    "medium",
				Confidence:  0.75,
				SourceFix:   "FIX #57",
				GoalTarget:  "both", // Topic shift can affect both primary and current goals
			}
			clarificationGaps = append(clarificationGaps, gap)
			log.Printf("[Layer9] FIX #57: Created topic shift gap: %s", clarificationText)
		}

		// After clarification, reset context
		lc.AccumulatedExtractedEntities = []models.ExtractedEntity{}
		lc.PreviousGoal = ""
		lc.PreviousValues = []string{}

		contextReset = true
		log.Printf("[Layer9] ✓ Context reset for new topic")
	}

	// Store clarification gaps in layer result (FIX #57)
	if len(clarificationGaps) > 0 {
		// Store for potential Layer 4 integration
		lc.Layer9ClarificationGaps = clarificationGaps
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
// FIX #58: Add comprehensive nil checks to prevent 159 nil dereferences
func (td *TopicShiftDetector) DetectShifts(lc *tools.LayerContext) []tools.TopicShift {
	shifts := make([]tools.TopicShift, 0)

	// FIX #58: Check LayerContext and Analysis before accessing
	if lc == nil || lc.Analysis == nil {
		return shifts
	}

	// Get previous contacts from relevant contacts list
	prevContactMap := make(map[string]bool)
	if lc.Analysis.RelevantContacts != nil { // FIX #58: Nil check
		for _, contact := range lc.Analysis.RelevantContacts {
			if contact.Name != "" { // FIX #58: Check for non-empty name
				prevContactMap[contact.Name] = true
			}
		}
	}

	// Detect contact shifts (user switched to talking about someone else)
	currentContactMap := make(map[string]bool)
	if lc.Analysis.Contacts != nil { // FIX #58: Nil check
		for _, contact := range lc.Analysis.Contacts {
			if contact.Name != "" { // FIX #58: Check for non-empty name
				currentContactMap[contact.Name] = true
			}
		}
	}

	// Find contacts that were discussed before but not now (shift detected)
	if len(prevContactMap) > 0 && len(lc.Analysis.Contacts) > 0 { // FIX #58: Check array length
		for prevContact := range prevContactMap {
			if !currentContactMap[prevContact] {
				shifts = append(shifts, tools.TopicShift{
					Type:       "contact_change",
					Severity:   "medium",
					Confidence: 0.7,
				})
				break // Only report one shift per message
			}
		}
	}

	// Detect characteristic/goal shifts (may indicate topic change)
	// FIX #58: Add proper nil checks before dereferencing
	if lc.Layer1 != nil && lc.Layer1.ExtractedContext != nil {
		if lc.Analysis.ExtractedCharacteristics != nil && len(lc.Analysis.ExtractedCharacteristics) > 0 &&
			lc.Analysis.UserProfile != nil && len(lc.Analysis.UserProfile.CommunicationStyle) > 0 {
			log.Printf("[Layer9] FIX #58: Detected potential characteristic change (may indicate topic shift)")
		}
	}

	return shifts
}

// getPreviousPrimaryContact extracts primary contact name from relevant contacts (FIX #42)
func getPreviousPrimaryContact(contacts []models.Contact) string {
	for _, c := range contacts {
		if c.Name != "" {
			return c.Name
		}
	}
	return "the previous topic"
}
