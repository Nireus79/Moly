package agents

import (
	"testing"
)

// TestResponseValidatorSummary verifies GetSummary works
func TestResponseValidatorSummary(t *testing.T) {
	t.Logf("Phase 3: Testing summary generation")

	validator := NewResponseValidator(nil)

	// Valid result
	validResult := &ValidationResult{
		IsValid: true,
		Reason:  "Test passed",
	}

	summary := validator.GetSummary(validResult)
	if !stringContainsValidation(summary, "✓") {
		t.Errorf("Valid summary should contain checkmark: %s", summary)
	}
	t.Logf("✓ Valid summary: %s", summary)

	// Invalid result
	invalidResult := &ValidationResult{
		IsValid:     false,
		Reason:      "Test failed",
		Severity:    "high",
		ShouldBlock: true,
	}

	summary = validator.GetSummary(invalidResult)
	if !stringContainsValidation(summary, "❌") {
		t.Errorf("Invalid summary should contain X: %s", summary)
	}
	t.Logf("✓ Invalid summary: %s", summary)
}

// Helper function
func stringContainsValidation(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
