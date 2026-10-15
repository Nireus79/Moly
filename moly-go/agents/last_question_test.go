package agents

import "testing"

func TestLastQuestionOf(t *testing.T) {
	cases := map[string]string{
		"Got it, so the heating is broken. What exactly is going wrong?":                      "What exactly is going wrong?",
		"Thanks. Here is a draft:\n\n\"Dear Anna, can you help?\"\n\nShall I make it warmer?": "Shall I make it warmer?",
		"No question in this one.":        "",
		"Ευχαριστώ. Ποιο είναι το όνομα;": "Ποιο είναι το όνομα;",
		"Which of these: a? or b?":        "or b?",
	}
	for in, want := range cases {
		if got := lastQuestionOf(in); got != want {
			t.Errorf("lastQuestionOf(%q) = %q, want %q", in, got, want)
		}
	}
}
