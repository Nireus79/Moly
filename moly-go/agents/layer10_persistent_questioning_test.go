package agents

import (
	"context"
	"moly/tools"
	"testing"
)

func TestLayer10New(t *testing.T) {
	l10 := NewLayer10PersistentQuestioning(nil, nil)
	if l10 == nil {
		t.Fatal("Failed to create Layer10")
	}
}

func TestLayer10Name(t *testing.T) {
	l10 := NewLayer10PersistentQuestioning(nil, nil)
	if l10.Name() != "Layer10-PersistentQuestioning" {
		t.Error("Name mismatch")
	}
}

func TestLayer10Priority(t *testing.T) {
	l10 := NewLayer10PersistentQuestioning(nil, nil)
	if l10.Priority() != 45 {
		t.Errorf("Expected priority 45, got %d", l10.Priority())
	}
}

func TestLayer10Process(t *testing.T) {
	l10 := NewLayer10PersistentQuestioning(nil, nil)

	lc := &tools.LayerContext{
		Layer7: &tools.Layer7Result{
			ViolationDetected: true,
		},
	}

	result, err := l10.Process(context.Background(), lc)
	if err != nil {
		t.Errorf("Process failed: %v", err)
	}

	if result.Layer10 == nil {
		t.Error("Layer10 result should not be nil")
	}

	if len(result.Layer10.PersistentQuestions) == 0 {
		t.Error("Should generate questions")
	}
}
