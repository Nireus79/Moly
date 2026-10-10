package agents

import (
	"testing"

	"moly/models"
)

func TestFactConfidence(t *testing.T) {
	cases := []struct {
		name     string
		model    float64
		basis    string
		grounded bool
		want     float64
		doubt    bool
	}{
		{"stated and grounded keeps the model's number", 0.9, "stated", true, 0.9, false},
		{"implied is capped but not in doubt", 0.95, "implied", true, 0.7, false},
		{"no basis is read as implied", 0.95, "", true, 0.7, false},
		{"a guess is in doubt", 0.95, "guessed", true, 0.4, true},
		{"evidence not in the message is in doubt", 0.95, "stated", false, 0.4, true},
		{"no number from the model is not read as high", 0, "stated", true, 0.5, true},
		{"a low number is kept", 0.3, "stated", true, 0.3, true},
		{"above one is clamped", 1.7, "stated", true, 1, false},
	}
	for _, c := range cases {
		got := FactConfidence(c.model, c.basis, c.grounded)
		if got != c.want || FactInDoubt(got) != c.doubt {
			t.Errorf("%s: confidence %v doubt %v, want %v and %v", c.name, got, FactInDoubt(got), c.want, c.doubt)
		}
	}
}

func TestQuoteInMessage(t *testing.T) {
	if !QuoteInMessage("I want to write to  Anna  tonight", "to anna") {
		t.Fatal("case and spacing are ignored")
	}
	if QuoteInMessage("hello", "") || QuoteInMessage("hello", "goodbye") {
		t.Fatal("an empty or missing quote is not grounded")
	}
}

func TestApplyFactConfidenceGoalNeedsAGroundedQuote(t *testing.T) {
	msg := "I want to write a first message to Christine."
	grounded := &models.ExtractedContext{Intention: "write a first message", IntentionBasis: "stated", IntentionEvidence: "write a first message"}
	applyFactConfidence(grounded, msg)
	if FactInDoubt(grounded.IntentionConfidence) {
		t.Fatalf("a stated goal with its words in the message is sure, got %.2f", grounded.IntentionConfidence)
	}
	for name, evidence := range map[string]string{"no quote": "", "a quote that is not in the message": "decide about my job"} {
		g := &models.ExtractedContext{Intention: "something", IntentionBasis: "stated", IntentionEvidence: evidence}
		applyFactConfidence(g, msg)
		if !FactInDoubt(g.IntentionConfidence) {
			t.Errorf("%s: a goal without a grounded quote is in doubt, got %.2f", name, g.IntentionConfidence)
		}
	}
}
