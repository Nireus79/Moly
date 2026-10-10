package agents

import (
	"log"

	"moly/models"
	"moly/tools"
)

// ChangeToClariifcation converts detected context changes into clarification gaps
// FIX #46-49: Bridge between detection and response generation
type ChangeToClarification struct{}

// FIX #60: Constants for clarification confidence levels
const (
	confidenceMetaConflict = 0.9 // Very high - contradictions are clear
	severityHigh           = "high"
)

// ValidateGap checks if a gap is valid before using (FIX #62)
func (ctc *ChangeToClarification) ValidateGap(gap tools.Gap) bool {
	// FIX #62: Validate gap before appending
	if gap.Type == "" {
		log.Printf("[ChangeToClarification] FIX #62: Rejected gap - empty type")
		return false
	}
	if gap.Description == "" {
		log.Printf("[ChangeToClarification] FIX #62: Rejected gap - empty description")
		return false
	}
	if gap.Confidence < 0 || gap.Confidence > 1 {
		log.Printf("[ChangeToClarification] FIX #62: Rejected gap - invalid confidence %.2f", gap.Confidence)
		return false
	}
	if gap.Severity != "low" && gap.Severity != "medium" && gap.Severity != "high" && gap.Severity != "critical" {
		log.Printf("[ChangeToClarification] FIX #62: Rejected gap - invalid severity %s", gap.Severity)
		return false
	}
	return true
}

// GenerateGapsFromChanges creates Gap objects for Layer 4 based on detected changes (FIX #46)
func (ctc *ChangeToClarification) GenerateGapsFromChanges(
	ctx *models.AnalysisContext,
	tracker *ContextChangeTracker,
) []tools.Gap {
	gaps := []tools.Gap{}

	if ctx == nil || tracker == nil {
		return gaps
	}

	// A change of goal or intention is not turned into a gap here. It was a text difference between goal lists
	// ("I notice you no longer mention [X]. Are you changing direction?"), and it fired when the user only refined the goal
	// (found live, 2026-10-10). The model judges how a new goal relates to the locked one (Layer 1, GoalRelation), and a real
	// switch is confirmed with the user (goal_switch.go).

	// FIX #45: If meta-instructions are contradictory, create clarification gap
	if gap := metaConflictGap(tracker); gap != nil {
		// FIX #62: Validate gap before appending
		if ctc.ValidateGap(*gap) {
			gaps = append(gaps, *gap)
			log.Printf("[ChangeToClarification] FIX #62: Created & validated gap for meta-instruction conflict")
		}
	}

	// FIX #63: Deduplicate gaps to prevent asking same question twice
	// (Multiple layers might generate same gap type)
	deduplicatedGaps := ctc.DeduplicateGaps(gaps)

	return deduplicatedGaps
}

// DeduplicateGaps removes duplicate gap types to prevent asking same question twice (FIX #63)
func (ctc *ChangeToClarification) DeduplicateGaps(gaps []tools.Gap) []tools.Gap {
	if len(gaps) == 0 {
		return gaps
	}

	seenTypes := make(map[string]bool)
	uniqueGaps := []tools.Gap{}

	for _, gap := range gaps {
		if !seenTypes[gap.Type] {
			uniqueGaps = append(uniqueGaps, gap)
			seenTypes[gap.Type] = true
			log.Printf("[ChangeToClarification] FIX #63: Added gap type '%s' (deduplicated)", gap.Type)
		} else {
			log.Printf("[ChangeToClarification] FIX #63: Skipped duplicate gap type '%s'", gap.Type)
		}
	}

	if len(gaps) > len(uniqueGaps) {
		log.Printf("[ChangeToClarification] FIX #63: Deduplicated %d → %d gaps", len(gaps), len(uniqueGaps))
	}

	return uniqueGaps
}

// metaConflictGap returns a clarification gap when the user's instructions contradict each other.
func metaConflictGap(tracker *ContextChangeTracker) *tools.Gap {
	if !tracker.HasContradictoryInstructions() {
		return nil
	}
	return &tools.Gap{
		Type:        "meta_instruction_conflict",
		Description: "You've given me instructions that pull in opposite directions. Which should I follow?",
		Severity:    severityHigh,
		Confidence:  confidenceMetaConflict,
		SourceFix:   "FIX #45",
	}
}
