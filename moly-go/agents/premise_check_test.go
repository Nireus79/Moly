package agents

import "testing"

// approvedLLM is a provider whose models Moly was tested with.
type approvedLLM struct {
	cannedLLM
	approved bool
}

func (a *approvedLLM) RepliesAreApproved() bool { return a.approved }

// With the models Moly was tested with the reply is not checked; the check is one call fewer per reply.
func TestRepliesFromApprovedModelsAreNotChecked(t *testing.T) {
	llm := &approvedLLM{cannedLLM: cannedLLM{answer: `{"unsupported": "something"}`}, approved: true}
	ca := &conversationAgent{llmClient: llm}
	if got := ca.groundedReply("hello", "Did he leave recently?", "dialogue"); got != "Did he leave recently?" || len(llm.seen) != 0 {
		t.Fatalf("an approved model's reply goes out unchecked, got %q after %d calls", got, len(llm.seen))
	}
}

// Any other model is checked, and a flagged reply is rewritten.
func TestRepliesFromOtherModelsAreChecked(t *testing.T) {
	llm := &approvedLLM{cannedLLM: cannedLLM{answer: `{"unsupported": ""}`}, approved: false}
	ca := &conversationAgent{llmClient: llm}
	ca.groundedReply("hello", "Did he leave recently?", "dialogue")
	if len(llm.seen) != 1 {
		t.Fatalf("the reply is checked once, got %d calls", len(llm.seen))
	}
	plain := &cannedLLM{answer: `{"unsupported": ""}`} // a provider that says nothing about approval counts as not approved
	(&conversationAgent{llmClient: plain}).groundedReply("hello", "Did he leave recently?", "dialogue")
	if len(plain.seen) != 1 {
		t.Fatalf("a provider that does not say is checked, got %d calls", len(plain.seen))
	}
}

// The operator can ask for every reply to be checked.
func TestTheOperatorCanForceTheCheck(t *testing.T) {
	t.Setenv("MOLY_PREMISE_CHECK", "always")
	llm := &approvedLLM{cannedLLM: cannedLLM{answer: `{"unsupported": ""}`}, approved: true}
	(&conversationAgent{llmClient: llm}).groundedReply("hello", "Did he leave recently?", "dialogue")
	if len(llm.seen) != 1 {
		t.Fatalf("forced: one call, got %d", len(llm.seen))
	}
}
