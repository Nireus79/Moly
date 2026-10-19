package agents

import (
	"strings"
	"testing"
)

func TestSavedStyleLabelsAreUnderstood(t *testing.T) {
	for label, want := range map[string]string{
		"Casual": "relaxed", "Formal & Respectful": "professional", "Formal": "professional", "Direct": "brief", "Authentic": "honest",
	} {
		if got := styleInstruction(label); !strings.Contains(got, want) {
			t.Errorf("%q gave %q, want it to mention %q", label, got, want)
		}
	}
	if !strings.Contains(styleInstruction(""), "warm yet respectful") {
		t.Error("no style should give the neutral instruction")
	}
}

func TestHumorousToneGivesHumorWithLimits(t *testing.T) {
	got := toneInstruction("Humorous")
	if got == "" || !strings.Contains(got, "wit") || !strings.Contains(got, "drop the jokes") {
		t.Fatalf("humorous tone must ask for humor and name when to drop it: %q", got)
	}
	if toneInstruction("Friendly") != "" {
		t.Error("other tones need no extra instruction")
	}
}
