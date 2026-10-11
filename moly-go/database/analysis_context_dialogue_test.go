package database

import (
	"strings"
	"testing"

	"moly/models"
)

func history() []models.Message {
	return []models.Message{
		{Role: "user", Content: "My manager Dana gives me too many tasks."},
		{Role: "assistant", Content: "Would you like me to draft a message to Dana?"},
		{Role: "user", Content: "Yes"}, // the current message, last
	}
}

// The prompt carries the memory and Moly's last message. The user's earlier message is added only while the summary lags.
func TestDialogueCarriesMolysLastMessageAndTheUserBeforeItOnlyWhenTheSummaryLags(t *testing.T) {
	ctx := &models.AnalysisContext{ConversationSummary: &models.ConversationSummary{Arc: "The user is overloaded by Dana.", MessageCount: 2}}
	p := dialogueFrom(ctx, history(), "Yes")
	if p.Summary != "The user is overloaded by Dana." || p.MolyLast != "Would you like me to draft a message to Dana?" {
		t.Fatalf("summary and Moly's last message, got %+v", p)
	}
	if p.UserBefore != "" {
		t.Fatalf("the summary covers the 2 earlier messages: the user's message is in it, got %q", p.UserBefore)
	}
	lagging := &models.AnalysisContext{ConversationSummary: &models.ConversationSummary{Arc: "old", MessageCount: 0}}
	if p := dialogueFrom(lagging, history(), "Yes"); p.UserBefore != "My manager Dana gives me too many tasks." {
		t.Fatalf("the summary does not cover it yet: the user's message is added, got %q", p.UserBefore)
	}
	if p := dialogueFrom(&models.AnalysisContext{}, history(), "Yes"); !strings.Contains(p.UserBefore, "Dana") {
		t.Fatal("with no summary at all the user's message is the only memory")
	}
	if p := dialogueFrom(&models.AnalysisContext{}, nil, "Hello"); p.Text() != "" {
		t.Fatalf("a first message has no dialogue, got %q", p.Text())
	}
}
