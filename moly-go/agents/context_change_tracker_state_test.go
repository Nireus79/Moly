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
