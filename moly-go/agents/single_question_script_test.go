package agents

import "testing"

func TestTheQuestionContractReadsQuestionMarksOfOtherScripts(t *testing.T) {
	yes := []string{"What is her name?", "Ποιο είναι το όνομα της διευθύντριας;", "Ποιο είναι το όνομα;", "ما اسمها؟", "彼女の名前は？"}
	for _, q := range yes {
		if !meetsQuestionContract(q) {
			t.Errorf("one question not recognised: %q", q)
		}
	}
	no := []string{"Ποιο είναι το όνομα; Και πόσων ετών;", "Two? Questions?", "No question here.", "Ένα απλό σχόλιο."}
	for _, q := range no {
		if meetsQuestionContract(q) {
			t.Errorf("not one question: %q", q)
		}
	}
	if got := firstQuestion("Ευχαριστώ. Ποιο είναι το όνομα; Και η ηλικία;"); got != "Ποιο είναι το όνομα;" {
		t.Errorf("first Greek question = %q", got)
	}
	if got := firstQuestion("Thanks. What is her name? And her age?"); got != "What is her name?" {
		t.Errorf("first question = %q", got)
	}
	// a semicolon in English text is not a question mark
	if countQuestionMarks("I came; I saw; I left.") != 0 {
		t.Errorf("a semicolon in English text counted as a question mark")
	}
}
