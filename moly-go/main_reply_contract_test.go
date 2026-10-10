package main

import (
	"strings"
	"testing"
)

// PHASE 1 reply contract: the reply that is sent is the reply that is saved, and an empty
// reply is an error. These cases cover the three shapes a response map can take.
func TestResolveReplyTextUsesResponseText(t *testing.T) {
	text, err := resolveReplyText(map[string]interface{}{"response": "Hello there"}, "ignored")
	if err != nil || text != "Hello there" {
		t.Fatalf("got %q, %v", text, err)
	}
}

func TestResolveReplyTextFallsBackToAgentText(t *testing.T) {
	text, err := resolveReplyText(map[string]interface{}{"phase": "responding"}, "Agent says hi")
	if err != nil || text != "Agent says hi" {
		t.Fatalf("got %q, %v", text, err)
	}
}

func TestResolveReplyTextComposesClarification(t *testing.T) {
	response := map[string]interface{}{
		"status": "clarification_needed",
		"clarification": map[string]interface{}{
			"question": "Who do you mean?",
			"options":  []string{"A) Your friend", "B) Someone else?"},
		},
	}
	text, err := resolveReplyText(response, "Agent text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(text, "Agent text") {
		t.Fatalf("clarification reply malformed: %q", text)
	}
}

func TestResolveReplyTextEmptyIsError(t *testing.T) {
	if _, err := resolveReplyText(map[string]interface{}{"phase": "responding"}, "   "); err == nil {
		t.Fatal("empty reply must be an error, not a silent success")
	}
}

func TestUnwrapQuotedReply(t *testing.T) {
	cases := map[string]string{
		`"Hello there."`:   "Hello there.",
		"“Hello there.”":   "Hello there.",
		`Say "hi" to her.`: `Say "hi" to her.`,
		`"One" and "two"`:  `"One" and "two"`,
		"Plain text":       "Plain text",
		`"`:                `"`,
	}
	for in, want := range cases {
		if got := unwrapQuotedReply(in); got != want {
			t.Errorf("unwrapQuotedReply(%q) = %q, want %q", in, got, want)
		}
	}
}
