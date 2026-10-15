package tools

import (
	"strings"
	"testing"

	"moly/models"
)


// A question Moly writes is spoken by Moly to the user (found live: "Hey there! I'm so grateful for all the help Anna has
// been providing", the model writing as the user), and it carries the gap it is meant to ask.
func TestGapClarificationPromptSetsTheSpeakerAndCarriesTheGap(t *testing.T) {
	got := buildGapClarificationPrompt(models.Context{ConversationHistory: []models.Message{{Role: "user", Content: "Warm and informal."}}}, []string{"What kind of help has Anna given you?"}, "", "")
	for _, want := range []string{"You are Moly, speaking TO the user", "Never write as the user", "What kind of help has Anna given you?", "Ask exactly ONE question"} {
		if !strings.Contains(got, want) {
			t.Errorf("the prompt must contain %q:\n%s", want, got)
		}
	}
}

// Found live (2026-10-14): the question writer was told the FIRST message of the history was what the user "just said", saw no
// people and no relations, and wrote "your father" for someone else's father. It now gets the dialogue and the real latest message.
func TestGapClarificationPromptCarriesTheDialogueAndTheLatestMessage(t *testing.T) {
	dialogue := "The people involved:\n- Alex (Kyle's father; the user's former boss)\n"
	got := buildGapClarificationPrompt(models.Context{ConversationHistory: []models.Message{{Role: "user", Content: "the very first message"}}},
		[]string{"Did he leave recently?"}, dialogue, "the message just received")
	for _, want := range []string{"Alex (Kyle's father; the user's former boss)", "the message just received", "only if the conversation says so"} {
		if !strings.Contains(got, want) {
			t.Errorf("the prompt must contain %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "the very first message") {
		t.Error("the first message of the history is not what the user just said")
	}
}
