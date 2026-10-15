package agents

import (
	"testing"

	"moly/models"
)

func TestTrackerDetectsIntentShiftAcrossRestart(t *testing.T) {
	first := NewContextChangeTracker()
	first.DetectIntentionChange(&models.AnalysisContext{CachedIntentAnalysis: &models.IntentAnalysis{Intent: "ask"}})
	state := first.ExportState()

	restored := NewContextChangeTrackerFromState(state)
	changed, prev, cur := restored.DetectIntentionChange(&models.AnalysisContext{CachedIntentAnalysis: &models.IntentAnalysis{Intent: "vent"}})
	if !changed || prev != "ask" || cur != "vent" {
		t.Fatalf("intent shift not detected after restore: changed=%v prev=%q cur=%q", changed, prev, cur)
	}
}

func TestTrackerWithoutStateSeesNoShift(t *testing.T) {
	fresh := NewContextChangeTrackerFromState(nil)
	changed, _, _ := fresh.DetectIntentionChange(&models.AnalysisContext{CachedIntentAnalysis: &models.IntentAnalysis{Intent: "vent"}})
	if changed {
		t.Fatal("a new tracker has no previous intent, so it cannot report a shift")
	}
}

// A message that states no goal does not remove the goal (found live, 2026-10-15: "GOAL SHIFT: Added 0, removed 1" on an answer).
func TestTrackerAMessageWithoutAGoalRemovesNothing(t *testing.T) {
	cct := NewContextChangeTracker()
	cct.DetectGoalChange(&models.AnalysisContext{GoalProgression: []string{"write a first message"}})
	changed, added, removed := cct.DetectGoalChange(&models.AnalysisContext{})
	if changed || len(added) != 0 || len(removed) != 0 {
		t.Fatalf("no goal in the message is no change, got changed=%v added=%v removed=%v", changed, added, removed)
	}
	changed, _, removed = cct.DetectGoalChange(&models.AnalysisContext{GoalProgression: []string{"write a first message"}})
	if changed || len(removed) != 0 {
		t.Fatalf("the same goal again is no change, got changed=%v removed=%v", changed, removed)
	}
}
