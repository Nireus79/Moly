package agents

import "strings"

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
