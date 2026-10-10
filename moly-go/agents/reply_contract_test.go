package agents

import (
	"strings"
	"testing"

	"moly/models"
)

// A low-confidence contact forces the clarification branch. The reply the user sees must still
// carry the agent's text and the clarification question, never an empty reply.
func TestClarificationReplyKeepsAgentTextAndQuestion(t *testing.T) {
	crf := &ContactResponseFormatter{}
	contacts := []*models.Contact{{Name: "", Relationship: "friend", Confidence: 0.0, Status: "active"}}

	resp := crf.FormatResponse(nil, contacts, "Agent text about the goal", nil)
	clar, ok := resp["clarification"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected a clarification map, got %v", resp)
	}

	text := ComposeReplyWithClarification("Agent text about the goal", clar)
	if !strings.HasPrefix(text, "Agent text about the goal") {
		t.Fatalf("agent text was dropped: %q", text)
	}
	question, _ := clar["question"].(string)
	if question == "" || !strings.Contains(text, question) {
		t.Fatalf("clarification question missing from reply: %q", text)
	}
	if !strings.Contains(text, "A) ") {
		t.Fatalf("clarification options missing from reply: %q", text)
	}
}

func TestComposeReplyWithoutAgentText(t *testing.T) {
	clar := map[string]interface{}{
		"question": "Who do you mean?",
		"options":  []string{"A) Your friend", "B) Someone else?"},
	}
	text := ComposeReplyWithClarification("", clar)
	if text != "Who do you mean?\nA) Your friend\nB) Someone else?" {
		t.Fatalf("unexpected reply: %q", text)
	}
}
