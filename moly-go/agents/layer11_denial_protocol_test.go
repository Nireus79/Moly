package agents

import (
	"context"
	"moly/models"
	"moly/tools"
	"testing"
)

func TestLayer11New(t *testing.T) {
	l11 := NewLayer11DenialProtocol()
	if l11 == nil {
		t.Fatal("Failed to create Layer11")
	}
}

func TestLayer11Name(t *testing.T) {
	l11 := NewLayer11DenialProtocol()
	if l11.Name() != "Layer11-DenialProtocol" {
		t.Error("Name mismatch")
	}
}

func TestLayer11Priority(t *testing.T) {
	l11 := NewLayer11DenialProtocol()
	if l11.Priority() != 40 {
		t.Errorf("Expected priority 40, got %d", l11.Priority())
	}
}

func TestLayer11NeverSkips(t *testing.T) {
	l11 := NewLayer11DenialProtocol()
	lc := &tools.LayerContext{}

	if l11.CanSkip(lc) {
		t.Error("Layer11 should never skip")
	}
}

func TestLayer11ProcessNoDenial(t *testing.T) {
	l11 := NewLayer11DenialProtocol()
	lc := &tools.LayerContext{
		Analysis: &models.AnalysisContext{
			CurrentMessage: "I think this is a complex situation with many factors to consider.",
		},
	}

	result, err := l11.Process(context.Background(), lc)
	if err != nil {
		t.Errorf("Process failed: %v", err)
	}

	if result.Layer11 == nil {
		t.Error("Layer11 result should not be nil")
	}

	if result.Layer11.ShouldDeny {
		t.Error("Should not detect denial in detailed response")
	}
}

// A short reply is a denial only after the user has written a longer message before it.
func TestLayer11ProcessDetectsDenial(t *testing.T) {
	l11 := NewLayer11DenialProtocol()
	lc := &tools.LayerContext{
		Analysis: &models.AnalysisContext{
			CurrentMessage: "No",
			RecentMessages: []models.Message{
				{Role: "user", Content: "I want to write a first message to a girl I saw on fetlife."},
				{Role: "assistant", Content: "What would you like to say?"},
				{Role: "user", Content: "No"},
			},
		},
	}

	result, err := l11.Process(context.Background(), lc)
	if err != nil {
		t.Errorf("Process failed: %v", err)
	}

	if !result.Layer11.ShouldDeny {
		t.Error("Should detect denial in very short response")
	}

	if result.Layer11.DenialMessage == "" {
		t.Error("Should generate denial response")
	}
}

func TestLayer11GenerateDenialResponse(t *testing.T) {
	dd := &DenialDetector{}
	response := dd.GenerateDenialResponse(nil)

	if len(response) == 0 {
		t.Error("Should generate non-empty response")
	}

	if response == "" {
		t.Error("Response should contain empathetic message")
	}
}

// PHASE 3: a first message, even a short one, is never a withdrawal.
func TestLayer11FirstShortMessageIsNotDenial(t *testing.T) {
	l11 := NewLayer11DenialProtocol()
	lc := &tools.LayerContext{
		Analysis: &models.AnalysisContext{
			CurrentMessage: "Hi Moly",
			RecentMessages: []models.Message{{Role: "user", Content: "Hi Moly"}},
		},
	}
	result, err := l11.Process(context.Background(), lc)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}
	if result.Layer11.ShouldDeny {
		t.Error("a first message must not be treated as a denial")
	}
}

// PHASE 3: a greeting is never a denial, even with prior messages.
func TestLayer11GreetingIsNotDenial(t *testing.T) {
	l11 := NewLayer11DenialProtocol()
	lc := &tools.LayerContext{
		IsGreeting: true,
		Analysis: &models.AnalysisContext{
			CurrentMessage: "Hi",
			RecentMessages: []models.Message{
				{Role: "user", Content: "I want to write a message."},
				{Role: "user", Content: "Hi"},
			},
		},
	}
	result, err := l11.Process(context.Background(), lc)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}
	if result.Layer11.ShouldDeny {
		t.Error("a greeting must not be treated as a denial")
	}
}
