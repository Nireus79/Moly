package agents

import (
	"fmt"
	"log"

	"moly/models"
	"moly/tools"
)

// ChangeToClariifcation converts detected context changes into clarification gaps
// FIX #46-49: Bridge between detection and response generation
type ChangeToClarification struct{}

// FIX #60: Constants for clarification confidence levels
const (
	confidenceIntentionChange = 0.85 // High confidence - intent clearly changed
	confidenceGoalRemoval     = 0.9  // Very high - goal removal is significant
	confidenceGoalAddition    = 0.8  // High - goal addition detected
	confidenceMetaConflict    = 0.9  // Very high - contradictions are clear
	severityMedium            = "medium"
	severityHigh              = "high"
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

	// FIX #43: If intention changed, create clarification gap
	intentionChanged, prevIntent, currIntent := tracker.DetectIntentionChange(ctx)
	if intentionChanged {
		gap := tools.Gap{
			Type:        "intention_changed",
			Description: fmt.Sprintf("Your intent shifted from %s to %s. Should I %s?", prevIntent, currIntent, ctc.getActionForIntent(currIntent)),
			Severity:    severityMedium,
			Confidence:  confidenceIntentionChange,
			SourceFix:   "FIX #43",
		}
		// FIX #62: Validate gap before appending
		if ctc.ValidateGap(gap) {
			gaps = append(gaps, gap)
			log.Printf("[ChangeToClarification] FIX #62: Created & validated gap for intention change: %s → %s",
				prevIntent, currIntent)
		}
	}

	// FIX #44: If goals changed significantly, create clarification gap
	goalsChanged, added, removed := tracker.DetectGoalChange(ctx)
	if goalsChanged {
		if len(removed) > 0 {
			// Goal removal is significant
			gap := tools.Gap{
				Type:        "goal_changed",
				Description: fmt.Sprintf("I notice you no longer mention %v. Are you changing direction?", removed),
				Severity:    severityHigh,
				Confidence:  confidenceGoalRemoval,
				SourceFix:   "FIX #44",
			}
			// FIX #62: Validate gap before appending
			if ctc.ValidateGap(gap) {
				gaps = append(gaps, gap)
				log.Printf("[ChangeToClarification] FIX #62: Created & validated gap for goal removal: %v", removed)
			}
		} else if len(added) > 0 {
			// Goal addition less critical but worth noting
			gap := tools.Gap{
				Type:        "goal_changed",
				Description: fmt.Sprintf("You added a new goal: %v. How does this relate to your previous goal?", added),
				Severity:    severityMedium,
				Confidence:  confidenceGoalAddition,
				SourceFix:   "FIX #44",
			}
			// FIX #62: Validate gap before appending
			if ctc.ValidateGap(gap) {
				gaps = append(gaps, gap)
				log.Printf("[ChangeToClarification] FIX #62: Created & validated gap for goal addition: %v", added)
			}
		}
	}

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

// Helper function to determine action based on intent
func (ctc *ChangeToClarification) getActionForIntent(intent string) string {
	actions := map[string]string{
		"ask":       "provide practical advice",
		"vent":      "listen and validate your feelings",
		"share":     "listen and acknowledge",
		"help_seek": "help you work through this",
		"greet":     "greet you back",
		"react":     "respond to what you said",
		"confirm":   "confirm what you meant",
	}

	if action, exists := actions[intent]; exists {
		return action
	}
	return "adjust my response"
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
