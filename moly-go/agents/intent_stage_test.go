package agents

import (
	"testing"

	"moly/models"
)

// The intent judge took "Her name is Dana, and also help me write to my brother" for a plain answer in 4 of 5 live replays, and an
// answer loses its goal. The verdict stands only when the extraction does not state an aim of its own in the message.
func TestConfirmAnswerOnlyNeedsTheExtractionToAgree(t *testing.T) {
	msg := "Her name is Dana, and also help me write to my brother"
	stated := &models.ExtractedContext{Intention: "write to my brother", IntentionBasis: "stated", IntentionEvidence: "help me write to my brother"}
	d := IntentDecision{AnswerOnly: true}
	d.ConfirmAnswerOnly(stated, msg)
	if d.AnswerOnly {
		t.Fatal("a stated, quoted aim in the message means it is not only an answer")
	}
	for name, ec := range map[string]*models.ExtractedContext{
		"no goal":           {},
		"goal not quoted":   {Intention: "write to my manager", IntentionBasis: "stated", IntentionEvidence: "write to my manager"},
		"goal only implied": {Intention: "write to my brother", IntentionBasis: "implied", IntentionEvidence: "help me write to my brother"},
		"nil":               nil,
	} {
		d := IntentDecision{AnswerOnly: true}
		d.ConfirmAnswerOnly(ec, msg)
		if !d.AnswerOnly {
			t.Errorf("%s: the answer-only verdict stands", name)
		}
	}
}
