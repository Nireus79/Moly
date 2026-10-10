package agents

import "testing"

func TestReplacementForBlockedReplyIsOneQuestion(t *testing.T) {
	ca := &conversationAgent{}
	got := ca.replacementForBlockedReply("anything")
	if !meetsQuestionContract(got) {
		t.Fatalf("replacement is not one short question: %q", got)
	}
}
