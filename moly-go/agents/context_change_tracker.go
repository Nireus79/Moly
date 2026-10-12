package agents

import (
	"log"

	"moly/models"
)

// ContextChangeTracker monitors changes in intention, goals, and meta-instructions across messages
// FIX #43-45: Track and detect all context changes that should trigger clarifications
// FIX #54: Meta-instruction log limited to current message (no unbounded growth)
type ContextChangeTracker struct {
	previousIntent string
	previousGoals  []string
}

// NewContextChangeTracker creates a new tracker instance
// FIX #54: Initialize current message instructions (no cumulative log)
func NewContextChangeTracker() *ContextChangeTracker {
	return &ContextChangeTracker{}
}

// NewContextChangeTrackerFromState restores a tracker from the state saved after the previous message.
func NewContextChangeTrackerFromState(state *models.ContextTrackerState) *ContextChangeTracker {
	cct := NewContextChangeTracker()
	if state == nil {
		return cct
	}
	cct.previousIntent = state.PreviousIntent
	cct.previousGoals = append([]string(nil), state.PreviousGoals...)
	return cct
}

// ExportState returns the tracker memory to save for the next message.
func (cct *ContextChangeTracker) ExportState() *models.ContextTrackerState {
	state := &models.ContextTrackerState{
		PreviousIntent: cct.previousIntent,
		PreviousGoals:  append([]string(nil), cct.previousGoals...),
	}
	return state
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

	// Detect added/removed goals (FIX #55: compare exact casing, not lowercased)
	prevMap := make(map[string]bool)
	currMap := make(map[string]bool)

	for _, g := range cct.previousGoals {
		prevMap[g] = true // FIX #55: Exact comparison, preserve casing
	}
	for _, g := range currentGoals {
		currMap[g] = true // FIX #55: Exact comparison, preserve casing
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
