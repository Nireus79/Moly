package tools

import (
	"strings"
	"testing"

	"moly/models"
)


// A question Moly writes is spoken by Moly to the user (found live: "Hey there! I'm so grateful for all the help Anna has
// been providing", the model writing as the user), and it carries the gap it is meant to ask.
func TestGapClarificationPromptSetsTheSpeakerAndCarriesTheGap(t *testing.T) {
	got := buildGapClarificationPrompt(models.Context{ConversationHistory: []models.Message{{Role: "user", Content: "Warm and informal."}}}, []string{"What kind of help has Anna given you?"})
	for _, want := range []string{"You are Moly, speaking TO the user", "Never write as the user", "What kind of help has Anna given you?", "Ask exactly ONE question"} {
		if !strings.Contains(got, want) {
			t.Errorf("the prompt must contain %q:\n%s", want, got)
		}
	}
}
