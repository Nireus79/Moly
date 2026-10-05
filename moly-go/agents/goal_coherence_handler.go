package agents

import (
	"fmt"
	"log"
	"strings"

	"moly/models"
	"moly/tools"
)

// AnalyzeGoalCoherence determines how current message's goal relates to primary goal
// Used for multi-message handling to prevent goal confusion
func AnalyzeGoalCoherence(lc *tools.LayerContext) *models.GoalCoherence {
	if lc == nil {
		return &models.GoalCoherence{
			GoalProgression: "unknown",
			Confidence:      0.0,
		}
	}

	primaryGoal := lc.PrimaryGoal
	currentGoal := lc.UserGoal

	coherence := &models.GoalCoherence{
		PrimaryGoal: primaryGoal,
		CurrentGoal: currentGoal,
		Confidence:  0.85, // High confidence in our analysis
	}

	// Message 1: Primary goal IS current goal
	if lc.IsMessageOne {
		coherence.IsSameGoal = true
		coherence.GoalProgression = "same"
		coherence.ShouldAddressNew = false
		log.Printf("[GoalCoherence] Message 1: Primary goal = %q (setting as foundation)", primaryGoal)
		return coherence
	}

	// Message 2+: Compare to primary goal
	if primaryGoal == "" {
		// No primary goal set (shouldn't happen but handle gracefully)
		coherence.IsSameGoal = true
		coherence.GoalProgression = "unknown"
		return coherence
	}

	if currentGoal == "" {
		// No current goal detected
		coherence.IsSameGoal = true
		coherence.GoalProgression = "same" // Assume continuing same goal
		log.Printf("[GoalCoherence] Message %d: No new goal detected, continuing primary goal", lc.Analysis.MessageCount)
		return coherence
	}

	// Check if same goal
	if normalizeGoal(currentGoal) == normalizeGoal(primaryGoal) {
		coherence.IsSameGoal = true
		coherence.GoalProgression = "same"
		log.Printf("[GoalCoherence] Message %d: Same goal - primary=%q, current=%q",
			lc.Analysis.MessageCount, primaryGoal, currentGoal)
		return coherence
	}

	// Check if subgoal (current goal is part of achieving primary goal)
	if isSubgoal(currentGoal, primaryGoal) {
		coherence.IsSameGoal = false
		coherence.GoalProgression = "related_subgoal"
		coherence.ShouldAddressNew = true
		coherence.Relationship = fmt.Sprintf("%q is part of achieving %q", currentGoal, primaryGoal)
		log.Printf("[GoalCoherence] Message %d: Related subgoal - %s", lc.Analysis.MessageCount, coherence.Relationship)
		return coherence
	}

	// Different goal
	coherence.IsSameGoal = false
	coherence.GoalProgression = "different"
	coherence.ShouldAddressNew = true
	log.Printf("[GoalCoherence] Message %d: New goal - primary=%q, current=%q",
		lc.Analysis.MessageCount, primaryGoal, currentGoal)

	return coherence
}

// normalizeGoal converts goal text to normalized form for comparison
// Removes variations like "write_message", "write message", "message writing"
func normalizeGoal(goal string) string {
	if goal == "" {
		return ""
	}
	// Simple normalization: lowercase, replace spaces/underscores
	goal = strings.ToLower(goal)
	goal = strings.ReplaceAll(goal, "_", " ")
	goal = strings.TrimSpace(goal)
	return goal
}

// isSubgoal checks if goal1 is a subgoal of goal2
// Example: "decide_disclosure" is a subgoal of "write_message" to Christine
func isSubgoal(goal1, goal2 string) bool {
	goal1 = normalizeGoal(goal1)
	goal2 = normalizeGoal(goal2)

	// Subgoal mapping: common relationships
	subgoalMap := map[string][]string{
		"write message": {
			"decide disclosure",
			"choose tone",
			"pick opening angle",
			"decide forward level",
		},
		"build relationship": {
			"write message",
			"have conversation",
			"decide meeting",
		},
		"decide meeting": {
			"plan location",
			"choose time",
			"prepare topic",
		},
	}

	for parent, subgoals := range subgoalMap {
		if goal2 == parent {
			for _, subgoal := range subgoals {
				if goal1 == subgoal {
					return true
				}
			}
		}
	}

	return false
}

// ShouldSkipPreviousGaps checks if previous message's gaps have been answered
// Used to prevent asking same question twice
func ShouldSkipPreviousGaps(lc *tools.LayerContext) bool {
	if lc == nil || lc.Layer4 == nil {
		return false
	}

	// Check if current extraction indicates previous gaps were addressed
	// This is a heuristic: if we have high-confidence new extraction and it differs from before,
	// assume user answered previous gaps
	if lc.Layer1 != nil && lc.Layer1.Confidence >= 0.80 {
		// User provided new information with high confidence
		// Likely they were answering previous gaps
		log.Printf("[GoalCoherence] Skipping previous gaps: high confidence extraction indicates clarification answered")
		return true
	}

	return false
}
