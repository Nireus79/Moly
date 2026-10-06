package agents

import (
	"log"
	"strings"

	"moly/models"
)

// ContextChangeTracker monitors changes in intention, goals, and meta-instructions across messages
// FIX #43-45: Track and detect all context changes that should trigger clarifications
type ContextChangeTracker struct {
	previousIntent      string
	previousGoals       []string
	metaInstructionLog  map[string]int // Track instruction mentions over time
}

// NewContextChangeTracker creates a new tracker instance
func NewContextChangeTracker() *ContextChangeTracker {
	return &ContextChangeTracker{
		metaInstructionLog: make(map[string]int),
	}
}

// DetectIntentionChange checks if user's intent changed (FIX #43)
// Intent types: ask, share, help_seek, greet, vent, react, confirm
func (cct *ContextChangeTracker) DetectIntentionChange(ctx *models.AnalysisContext) (bool, string, string) {
	if ctx == nil || ctx.CachedIntentAnalysis == nil {
		return false, "", ""
	}

	currentIntent := ctx.CachedIntentAnalysis.Intent
	if currentIntent == "" {
		return false, "", ""
	}

	// First message - just store
	if cct.previousIntent == "" {
		cct.previousIntent = currentIntent
		return false, "", ""
	}

	// Detect change
	if currentIntent != cct.previousIntent {
		log.Printf("[ContextChangeTracker] 🔄 FIX #43: INTENTION SHIFT: '%s' → '%s'",
			cct.previousIntent, currentIntent)

		prev := cct.previousIntent
		cct.previousIntent = currentIntent
		return true, prev, currentIntent
	}

	return false, "", ""
}

// DetectGoalChange checks if user's goals changed (FIX #44)
func (cct *ContextChangeTracker) DetectGoalChange(ctx *models.AnalysisContext) (bool, []string, []string) {
	if ctx == nil {
		return false, []string{}, []string{}
	}

	// Extract current goals from GoalProgression (last entry is current)
	var currentGoals []string
	if len(ctx.GoalProgression) > 0 {
		currentGoals = []string{ctx.GoalProgression[len(ctx.GoalProgression)-1]}
	}

	// First message - just store
	if len(cct.previousGoals) == 0 {
		cct.previousGoals = currentGoals
		return false, []string{}, []string{}
	}

	// Detect added/removed goals
	prevMap := make(map[string]bool)
	currMap := make(map[string]bool)

	for _, g := range cct.previousGoals {
		prevMap[strings.ToLower(g)] = true
	}
	for _, g := range currentGoals {
		currMap[strings.ToLower(g)] = true
	}

	var added []string
	var removed []string

	// Find additions
	for g := range currMap {
		if !prevMap[g] {
			added = append(added, g)
		}
	}

	// Find removals
	for g := range prevMap {
		if !currMap[g] {
			removed = append(removed, g)
		}
	}

	if len(added) > 0 || len(removed) > 0 {
		log.Printf("[ContextChangeTracker] 🔄 FIX #44: GOAL SHIFT: Added %d, removed %d",
			len(added), len(removed))

		cct.previousGoals = currentGoals
		return true, added, removed
	}

	cct.previousGoals = currentGoals
	return false, []string{}, []string{}
}

// TrackMetaInstruction records mentions of meta-instructions (FIX #45)
// Meta-instructions: scope, focus, restrictions, tone requirements
func (cct *ContextChangeTracker) TrackMetaInstruction(messageText string) {
	if messageText == "" {
		return
	}

	lower := strings.ToLower(messageText)

	// Look for meta-instruction keywords
	instructions := map[string]bool{
		"keep it focused":        strings.Contains(lower, "keep it focused"),
		"don't focus":            strings.Contains(lower, "don't focus") || strings.Contains(lower, "dont focus"),
		"be respectful":          strings.Contains(lower, "respectful"),
		"be direct":              strings.Contains(lower, "be direct") || strings.Contains(lower, "direct"),
		"be casual":              strings.Contains(lower, "casual"),
		"keep it short":          strings.Contains(lower, "keep it short"),
		"just listen":            strings.Contains(lower, "just listen"),
		"give advice":            strings.Contains(lower, "give advice"),
		"don't give advice":      strings.Contains(lower, "don't give advice") || strings.Contains(lower, "dont give"),
		"be careful":             strings.Contains(lower, "be careful"),
		"don't worry":            strings.Contains(lower, "don't worry") || strings.Contains(lower, "dont worry"),
	}

	for instruction, present := range instructions {
		if present {
			cct.metaInstructionLog[instruction]++
			log.Printf("[ContextChangeTracker] 📝 FIX #45: Meta-instruction tracked: %s (count=%d)",
				instruction, cct.metaInstructionLog[instruction])
		}
	}
}

// GetMetaInstructionHistory returns the history of meta-instructions (FIX #45)
func (cct *ContextChangeTracker) GetMetaInstructionHistory() map[string]int {
	return cct.metaInstructionLog
}

// HasContradictoryInstructions detects conflicting meta-instructions (FIX #45)
func (cct *ContextChangeTracker) HasContradictoryInstructions() bool {
	// Check for contradictions
	contradictions := [][]string{
		{"keep it focused", "don't focus"},
		{"give advice", "don't give advice"},
		{"be direct", "be casual"},
		{"just listen", "give advice"},
	}

	for _, pair := range contradictions {
		if cct.metaInstructionLog[pair[0]] > 0 && cct.metaInstructionLog[pair[1]] > 0 {
			log.Printf("[ContextChangeTracker] ⚠️ FIX #45: Contradictory instructions detected: '%s' AND '%s'",
				pair[0], pair[1])
			return true
		}
	}

	return false
}
