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
	confidenceIntentionChange    = 0.85  // High confidence - intent clearly changed
	confidenceGoalRemoval        = 0.9   // Very high - goal removal is significant
	confidenceGoalAddition       = 0.8   // High - goal addition detected
	confidenceMetaConflict       = 0.9   // Very high - contradictions are clear
	severityMedium               = "medium"
	severityHigh                 = "high"
)

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
			Confidence:  confidenceIntentionChange,  // FIX #60: Use constant
			SourceFix:   "FIX #43",
		}
		gaps = append(gaps, gap)
		log.Printf("[ChangeToClarification] FIX #60: Created gap for intention change: %s → %s",
			prevIntent, currIntent)
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
				Confidence:  confidenceGoalRemoval,  // FIX #60: Use constant
				SourceFix:   "FIX #44",
			}
			gaps = append(gaps, gap)
			log.Printf("[ChangeToClarification] FIX #60: Created gap for goal removal: %v", removed)
		} else if len(added) > 0 {
			// Goal addition less critical but worth noting
			gap := tools.Gap{
				Type:        "goal_changed",
				Description: fmt.Sprintf("You added a new goal: %v. How does this relate to your previous goal?", added),
				Severity:    severityMedium,
				Confidence:  confidenceGoalAddition,  // FIX #60: Use constant
				SourceFix:   "FIX #44",
			}
			gaps = append(gaps, gap)
			log.Printf("[ChangeToClarification] FIX #60: Created gap for goal addition: %v", added)
		}
	}

	// FIX #45: If meta-instructions are contradictory, create clarification gap
	if tracker.HasContradictoryInstructions() {
		history := tracker.GetMetaInstructionHistory()
		gap := tools.Gap{
			Type:        "meta_instruction_conflict",
			Description: fmt.Sprintf("You've given me conflicting instructions (%v). Which should I follow?", history),
			Severity:    severityHigh,
			Confidence:  confidenceMetaConflict,  // FIX #60: Use constant
			SourceFix:   "FIX #45",
		}
		gaps = append(gaps, gap)
		log.Printf("[ChangeToClarification] FIX #60: Created gap for meta-instruction conflict")
	}

	return gaps
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
