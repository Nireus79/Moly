package agents

import (
	"context"
	"testing"

	"moly/models"
	"moly/tools"
)

// PHASE 2 contract: a cached summary of the current message must not decide a layer.
// Layer 11 would have reported "user engaged" from a 0.95 cached summary; it must evaluate the message instead.
func TestLayer11IgnoresCachedSummaryOfCurrentMessage(t *testing.T) {
	l11 := NewLayer11DenialProtocol()
	msgID := "msg_current"
	lc := &tools.LayerContext{
		MessageID: msgID,
		Analysis: &models.AnalysisContext{
			CurrentMessage: "No",
			RecentMessages: []models.Message{
				{Role: "user", Content: "I want to write a first message to a girl I saw on fetlife."},
				{Role: "assistant", Content: "What would you like to say?"},
				{Role: "user", Content: "No"},
			},
		},
		MessageSummaryCache: map[string]interface{}{
			msgID: &models.MessageSummary{MessageID: msgID, Confidence: 0.95},
		},
	}
	result, err := l11.Process(context.Background(), lc)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}
	if !result.Layer11.ShouldDeny {
		t.Fatal("Layer 11 took the cached summary instead of evaluating the message")
	}
}
