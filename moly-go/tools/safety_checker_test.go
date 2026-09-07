package tools

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestNewSafetyChecker(t *testing.T) {
	os.Setenv("CLAUDE_API_KEY", "test-key")
	defer os.Unsetenv("CLAUDE_API_KEY")

	llm, err := NewLLMClient()
	if err != nil {
		t.Fatalf("Failed to create LLM client: %v", err)
	}

	checker := NewSafetyChecker(llm)
	if checker == nil {
		t.Error("NewSafetyChecker returned nil")
	}
}

func TestSafetyCheckerKeywordDetection(t *testing.T) {
	os.Setenv("CLAUDE_API_KEY", "test-key")
	defer os.Unsetenv("CLAUDE_API_KEY")

	llm, _ := NewLLMClient()
	checker := NewSafetyChecker(llm)

	tests := []struct {
		name      string
		message   string
		wantAlert SafetyAlertType
	}{
		{
			name:      "crisis keyword",
			message:   "I want to kill myself",
			wantAlert: SafetyAlertTypeCrisis,
		},
		{
			name:      "self harm keyword",
			message:   "I'm thinking about self harm",
			wantAlert: SafetyAlertTypeCrisis,
		},
		{
			name:      "no alert",
			message:   "I want to go for a walk",
			wantAlert: SafetyAlertTypeNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			result, err := checker.Check(ctx, &SafetyCheckInput{
				Message: tt.message,
			})

			if err != nil {
				t.Fatalf("Check() error = %v", err)
			}

			if result.AlertType != tt.wantAlert {
				t.Errorf("AlertType = %s, want %s", result.AlertType, tt.wantAlert)
			}
		})
	}
}

func TestSafetyCheckerNilInput(t *testing.T) {
	os.Setenv("CLAUDE_API_KEY", "test-key")
	defer os.Unsetenv("CLAUDE_API_KEY")

	llm, _ := NewLLMClient()
	checker := NewSafetyChecker(llm)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := checker.Check(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil input")
	}
}

func TestSafetyCheckerEmptyMessage(t *testing.T) {
	os.Setenv("CLAUDE_API_KEY", "test-key")
	defer os.Unsetenv("CLAUDE_API_KEY")

	llm, _ := NewLLMClient()
	checker := NewSafetyChecker(llm)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := checker.Check(ctx, &SafetyCheckInput{
		Message: "",
	})
	if err == nil {
		t.Error("Expected error for empty message")
	}
}

func TestCrisisResources(t *testing.T) {
	resources := getCrisisResources()

	if len(resources) == 0 {
		t.Error("Expected crisis resources to be non-empty")
	}

	for _, r := range resources {
		if r.Name == "" {
			t.Error("Crisis resource missing name")
		}
		if r.Description == "" {
			t.Error("Crisis resource missing description")
		}
	}
}

func TestImmediateIndicators(t *testing.T) {
	tests := []struct {
		message string
		expect  bool
	}{
		{"I want to suicide", true},
		{"kill myself", true},
		{"self harm", true},
		{"I love my life", false},
		{"Hello there", false},
	}

	for _, tt := range tests {
		got := hasImmediateIndicators(tt.message)
		if got != tt.expect {
			t.Errorf("hasImmediateIndicators(%q) = %v, want %v", tt.message, got, tt.expect)
		}
	}
}

func TestIllegalIndicators(t *testing.T) {
	tests := []struct {
		message string
		expect  bool
	}{
		{"I want to defraud them", true},
		{"Let's hack into the system", true},
		{"I'm going to steal", true},
		{"I love programming", false},
		{"Hello world", false},
	}

	for _, tt := range tests {
		got := hasIllegalIndicators(tt.message)
		if got != tt.expect {
			t.Errorf("hasIllegalIndicators(%q) = %v, want %v", tt.message, got, tt.expect)
		}
	}
}
