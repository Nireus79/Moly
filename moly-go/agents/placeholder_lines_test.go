package agents

import "testing"

func TestPlaceholderLinesAreRemoved(t *testing.T) {
	in := "Dear Dana,\n\nCould we talk?\n\nBest regards,\n[Your Name]"
	want := "Dear Dana,\n\nCould we talk?\n\nBest regards,"
	if got := withoutPlaceholderLines(in); got != want {
		t.Fatalf("got %q", got)
	}
	keep := "I wrote [the report] for Dana.\nSee you [soon]."
	if got := withoutPlaceholderLines(keep); got != keep {
		t.Fatalf("brackets inside a sentence are kept, got %q", got)
	}
	if withoutPlaceholderLines("[Your Name]") != "" {
		t.Fatal("a reply that is only a placeholder becomes empty")
	}
}
