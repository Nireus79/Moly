package agents

import (
	"context"
	"moly/tools"
	"testing"
)

func TestLayer9New(t *testing.T) {
	l9 := NewLayer9TopicShiftDetection()
	if l9 == nil {
		t.Fatal("Failed to create Layer9")
	}
}

func TestLayer9Name(t *testing.T) {
	l9 := NewLayer9TopicShiftDetection()
	if l9.Name() != "Layer9-TopicShift" {
		t.Error("Name mismatch")
	}
}

func TestLayer9Priority(t *testing.T) {
	l9 := NewLayer9TopicShiftDetection()
	if l9.Priority() != 55 {
		t.Errorf("Expected priority 55, got %d", l9.Priority())
	}
}

func TestLayer9SkipWithGaps(t *testing.T) {
	l9 := NewLayer9TopicShiftDetection()
	lc := &tools.LayerContext{
		Layer4: &tools.Layer4Result{
			GapCount: 2,
		},
	}

	if !l9.CanSkip(lc) {
		t.Error("Should skip with gaps")
	}
}

func TestLayer9ProcessNoShift(t *testing.T) {
	l9 := NewLayer9TopicShiftDetection()
	lc := &tools.LayerContext{
		Layer4: &tools.Layer4Result{
			GapCount: 0,
		},
	}

	result, err := l9.Process(context.Background(), lc)
	if err != nil {
		t.Errorf("Process failed: %v", err)
	}

	if result.Layer9 == nil {
		t.Error("Layer9 result should not be nil")
	}
}
