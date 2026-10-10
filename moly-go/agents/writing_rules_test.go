package agents

import (
	"strings"
	"testing"
)

// A note written for the user must not invent facts or leave placeholders (found live: "[Your Name]" and an invented
// "unexpected time off"). The rule is in the base reply prompt, so it holds for every reply.
func TestBasePromptHasTheWritingRules(t *testing.T) {
	ca := &conversationAgent{}
	got := ca.buildAdaptiveSystemPrompt("", "", "", false, false, ResponseType(""), false)
	for _, want := range []string{"WRITING FOR THE USER", "ready to send", "Do not invent facts", "[Your Name]", "leave it out"} {
		if !strings.Contains(got, want) {
			t.Errorf("the base prompt must contain %q", want)
		}
	}
}
