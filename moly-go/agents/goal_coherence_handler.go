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
// The relation itself comes from the model (JudgeGoalRelation, in Layer 1); this only turns it into the structure Layer 4 and the
// response strategy read.
func AnalyzeGoalCoherence(lc *tools.LayerContext) *models.GoalCoherence {
	if lc == nil || lc.Analysis == nil {
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

	// No primary goal locked yet (first real goal), or the current goal is the primary goal
	if primaryGoal == "" {
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

	// How the current goal relates to the locked one is judged once, by the model, in Layer 1 (JudgeGoalRelation); a
	// different wording is not a different goal, so no text comparison decides it here. Without a judgement (identical
	// text, or the model could not answer) the goal is the same: an unjudged goal never counts as a shift.
	switch GoalRelation(lc.GoalRelation) {
	case GoalDifferent:
		coherence.IsSameGoal = false
		coherence.GoalProgression = "different"
		coherence.ShouldAddressNew = true
		log.Printf("[GoalCoherence] Message %d: New goal - primary=%q, current=%q",
			lc.Analysis.MessageCount, primaryGoal, currentGoal)
	case GoalRefinement, GoalSubstep:
		coherence.IsSameGoal = false
		coherence.GoalProgression = "related_subgoal"
		coherence.ShouldAddressNew = true
		coherence.Relationship = fmt.Sprintf("%q is part of achieving %q", currentGoal, primaryGoal)
		log.Printf("[GoalCoherence] Message %d: Related goal - %s", lc.Analysis.MessageCount, coherence.Relationship)
	default:
		coherence.IsSameGoal = true
		coherence.GoalProgression = "same"
		log.Printf("[GoalCoherence] Message %d: Same goal - primary=%q, current=%q",
			lc.Analysis.MessageCount, primaryGoal, currentGoal)
	}
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
