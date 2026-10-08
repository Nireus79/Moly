package agents

import (
	"log"
	"regexp"

	"moly/models"
)

// ContextChangeTracker monitors changes in intention, goals, and meta-instructions across messages
// FIX #43-45: Track and detect all context changes that should trigger clarifications
// FIX #54: Meta-instruction log limited to current message (no unbounded growth)
type ContextChangeTracker struct {
	previousIntent              string
	previousGoals               []string
	currentMessageInstructions  map[string]bool // FIX #54: Track CURRENT message instructions (not cumulative)
	previousMessageInstructions map[string]bool // Previous message for detecting changes
}

// NewContextChangeTracker creates a new tracker instance
// FIX #54: Initialize current message instructions (no cumulative log)
func NewContextChangeTracker() *ContextChangeTracker {
	return &ContextChangeTracker{
		currentMessageInstructions:  make(map[string]bool),
		previousMessageInstructions: make(map[string]bool),
	}
}

// NewContextChangeTrackerFromState restores a tracker from the state saved after the previous message.
func NewContextChangeTrackerFromState(state *models.ContextTrackerState) *ContextChangeTracker {
	cct := NewContextChangeTracker()
	if state == nil {
		return cct
	}
	cct.previousIntent = state.PreviousIntent
	cct.previousGoals = append([]string(nil), state.PreviousGoals...)
	for k, v := range state.CurrentInstruction {
		cct.currentMessageInstructions[k] = v
	}
	return cct
}

// ExportState returns the tracker memory to save for the next message.
func (cct *ContextChangeTracker) ExportState() *models.ContextTrackerState {
	state := &models.ContextTrackerState{
		PreviousIntent:     cct.previousIntent,
		PreviousGoals:      append([]string(nil), cct.previousGoals...),
		CurrentInstruction: map[string]bool{},
	}
	for k, v := range cct.currentMessageInstructions {
		state.CurrentInstruction[k] = v
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

// instructionPatterns recognise a meta-instruction only when it is phrased as a directive:
// at the start of a sentence, or after "please" or "Moly,". Words that merely appear in
// the user's description of themselves or in pasted text do not count.
var instructionPatterns = map[string]*regexp.Regexp{
	"keep it focused":   directive(`keep it focused`),
	"don't focus":       directive(`(don't|do not) focus`),
	"be respectful":     directive(`be respectful`),
	"be direct":         directive(`be direct`),
	"be casual":         directive(`be casual`),
	"keep it short":     directive(`keep it short`),
	"just listen":       directive(`just listen`),
	"give advice":       directive(`give (me )?advice`),
	"don't give advice": directive(`(don't|do not) give (me )?advice`),
	"be careful":        directive(`be careful`),
	"don't worry":       directive(`(don't|do not) worry`),
}

func directive(phrase string) *regexp.Regexp {
	return regexp.MustCompile(`(?i)(^|[.!?\n]\s*|\bplease\s+|\bmoly[,:]?\s+)` + phrase + `\b`)
}

// TrackMetaInstruction records the meta-instructions given in the current message.
func (cct *ContextChangeTracker) TrackMetaInstruction(messageText string) {
	if messageText == "" {
		return
	}

	if cct.currentMessageInstructions != nil {
		cct.previousMessageInstructions = cct.currentMessageInstructions
	}

	cct.currentMessageInstructions = make(map[string]bool, len(instructionPatterns))
	for instruction, pattern := range instructionPatterns {
		if pattern.MatchString(messageText) {
			cct.currentMessageInstructions[instruction] = true
			log.Printf("[ContextChangeTracker] Meta-instruction in current message: %s", instruction)
		}
	}
}

// GetMetaInstructionHistory returns current message instructions (FIX #54: no history logging)
func (cct *ContextChangeTracker) GetMetaInstructionHistory() map[string]bool {
	if cct.currentMessageInstructions == nil {
		return make(map[string]bool)
	}
	return cct.currentMessageInstructions
}

// HasContradictoryInstructions detects conflicting meta-instructions (FIX #45)
// FIX #54: Checks current message only (prevents unbounded log)
func (cct *ContextChangeTracker) HasContradictoryInstructions() bool {
	if cct.currentMessageInstructions == nil {
		return false
	}

	// Check for contradictions in current message
	contradictions := [][]string{
		{"keep it focused", "don't focus"},
		{"give advice", "don't give advice"},
		{"just listen", "give advice"},
	}

	for _, pair := range contradictions {
		if cct.currentMessageInstructions[pair[0]] && cct.currentMessageInstructions[pair[1]] {
			log.Printf("[ContextChangeTracker] ⚠️ FIX #45: Contradictory instructions detected: '%s' AND '%s'",
				pair[0], pair[1])
			return true
		}
	}

	return false
}

// GetAllChanges detects ALL changes in one call (FIX #64: Idempotent)
// This prevents multiple detector calls from corrupting state
// FIX #64: Returns map with all change information to prevent re-calling detectors
func (cct *ContextChangeTracker) GetAllChanges(ctx *models.AnalysisContext) map[string]interface{} {
	result := make(map[string]interface{})

	if ctx == nil {
		return result
	}

	// Get intention changes (FIX #64: Single call per tracker)
	intentionChanged, prevIntent, currIntent := cct.DetectIntentionChange(ctx)
	result["intention_changed"] = intentionChanged
	result["prev_intent"] = prevIntent
	result["curr_intent"] = currIntent

	// Get goal changes (FIX #64: Consistent state)
	goalChanged, added, removed := cct.DetectGoalChange(ctx)
	result["goal_changed"] = goalChanged
	result["goals_added"] = added
	result["goals_removed"] = removed

	// Get meta-instruction info (FIX #64: No repeated tracking)
	contradictory := cct.HasContradictoryInstructions()
	result["has_contradictory_instructions"] = contradictory
	result["meta_instructions"] = cct.GetMetaInstructionHistory()

	log.Printf("[ContextChangeTracker] FIX #64: GetAllChanges completed (single tracker call, no state corruption)")
	return result
}
