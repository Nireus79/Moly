package agents

import (
	"context"
	"testing"

	"moly/models"
	"moly/tools"
)

// runLayer1WithIntention runs Layer 1 on a message whose extraction already produced the given goal.
func runLayer1WithIntention(t *testing.T, message, goal, primaryGoal string, totalMessages int) *tools.LayerContext {
	t.Helper()
	analysis := &models.AnalysisContext{
		UserID:              "u1",
		ConversationID:      "c1",
		CurrentMessage:      message,
		ExtractedConfidence: 0.9,
		TotalMessages:       totalMessages,
	}
	if goal != "" {
		analysis.ExtractedEntities = []models.ExtractedEntity{{Type: "goal", Value: goal}}
	}
	lc := tools.NewLayerContext(analysis, "u1", "m1", "c1", nil)
	lc.PrimaryGoal = primaryGoal
	adapter := NewLayer1ContextExtractionAdapter(nil, tools.NewExtractionCache())
	out, err := adapter.Process(context.Background(), lc)
	if err != nil {
		t.Fatalf("layer 1: %v", err)
	}
	return out
}

func TestGreetingDoesNotLockPrimaryGoal(t *testing.T) {
	lc := runLayer1WithIntention(t, "Hello Moly", "", "", 1)
	if lc.PrimaryGoal != "" {
		t.Fatalf("greeting must not become the primary goal, got %q", lc.PrimaryGoal)
	}
}

func TestFirstRealGoalLocksPrimaryGoal(t *testing.T) {
	lc := runLayer1WithIntention(t, "I want to start a conversation with her", "Start a conversation with her", "", 3)
	if lc.PrimaryGoal != "Start a conversation with her" {
		t.Fatalf("first real goal should lock, got %q", lc.PrimaryGoal)
	}
}

func TestLaterGoalDoesNotOverwriteLockedGoal(t *testing.T) {
	lc := runLayer1WithIntention(t, "Now I want to plan dinner", "Plan dinner", "Start a conversation with her", 5)
	if lc.PrimaryGoal != "Start a conversation with her" {
		t.Fatalf("locked goal was overwritten, got %q", lc.PrimaryGoal)
	}
}
