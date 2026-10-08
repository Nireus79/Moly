package agents

import (
	"strings"
	"testing"
)

func instructionsIn(message string) map[string]bool {
	cct := NewContextChangeTracker()
	cct.TrackMetaInstruction(message)
	return cct.currentMessageInstructions
}

func TestWordsInDescriptionAreNotInstructions(t *testing.T) {
	msg := "I'm a quiet, casual girl who is direct about what she wants. I love honesty."
	if got := instructionsIn(msg); len(got) != 0 {
		t.Fatalf("description words were read as instructions: %v", got)
	}
}

func TestPastedProfileTextIsNotAnInstruction(t *testing.T) {
	msg := "Her profile says: Roles submissive, Looking for something casual and direct, be respectful of boundaries."
	if got := instructionsIn(msg); len(got) != 0 {
		t.Fatalf("pasted text was read as instructions: %v", got)
	}
}

func TestDirectInstructionsAreRecognised(t *testing.T) {
	got := instructionsIn("Please be direct. Keep it short.")
	if !got["be direct"] || !got["keep it short"] {
		t.Fatalf("explicit instructions missed: %v", got)
	}
}

func TestOppositeInstructionsAreContradictory(t *testing.T) {
	cct := NewContextChangeTracker()
	cct.TrackMetaInstruction("Just listen. Give advice please.")
	if !cct.HasContradictoryInstructions() {
		t.Fatal("'just listen' and 'give advice' should conflict")
	}
}

func TestStyleWordsAreNotContradictory(t *testing.T) {
	cct := NewContextChangeTracker()
	cct.TrackMetaInstruction("Please be direct and be casual.")
	if cct.HasContradictoryInstructions() {
		t.Fatal("'be direct' and 'be casual' are not opposites")
	}
}

func TestConflictGapHasNoDebugOutput(t *testing.T) {
	cct := NewContextChangeTracker()
	cct.TrackMetaInstruction("Just listen. Give advice please.")
	gap := metaConflictGap(cct)
	if gap == nil {
		t.Fatal("expected a conflict gap")
	}
	if strings.Contains(gap.Description, "map[") {
		t.Fatalf("debug map leaked into question: %s", gap.Description)
	}
}
