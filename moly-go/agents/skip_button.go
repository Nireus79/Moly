package agents

// The skip button is the user's way out of Moly's questions (ORCHESTRATOR_DESIGN.md, "The skip button"). Pressing it
// asks Moly to go ahead with what it has. The server decides when the button is shown and checks the same rules again
// when it is pressed, so a stale or forged press cannot skip anything that must be asked.

// WayOutOpen reports whether the user may skip the questions: at least one question must have been answered.
func WayOutOpen(gapMaturity float64) bool { return gapMaturity > 0 }

// SkippableKind reports whether a question of this kind only makes the result better, so the user may skip it. The other
// questions (a doubtful fact, an unclear message or intent, an engaged principle, a raised concern) are about risk or
// about being able to answer at all: they are never skippable.
func SkippableKind(kind ReplyKind) bool {
	switch kind {
	case KindNameQuestion, KindGapQuestion, KindTopicShift, KindContact, KindSocratic:
		return true
	}
	return false
}

// CanSkip reports whether the reply of this kind carries the skip button.
func CanSkip(kind ReplyKind, gapMaturity float64) bool {
	return SkippableKind(kind) && WayOutOpen(gapMaturity)
}
