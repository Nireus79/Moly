package agents

import (
	"context"
	"moly/models"
	"testing"
)

// TestResponseValidatorAlignedResponse verifies aligned response passes validation
func TestResponseValidatorAlignedResponse(t *testing.T) {
	t.Logf("Phase 3: Testing aligned response validation")

	validator := NewResponseValidator(nil)

	// User says they're dominant
	extracted := []models.ExtractedEntity{
		{
			Value:      "dominant",
			Type:       "characteristic",
			Subject:    "user",
			Confidence: 0.95,
		},
	}

	// Response aligns with characteristic
	response := "Since you value taking charge, you might enjoy leadership roles where you can make decisions."

	result := validator.ValidateResponse(context.Background(), extracted, response)

	if !result.IsValid {
		t.Fatalf("Expected validation to pass, but got: %s", result.Reason)
	}

	if result.ShouldBlock {
		t.Fatal("Should not block aligned response")
	}

	if len(result.Contradictions) > 0 {
		t.Fatalf("Expected no contradictions, got %d", len(result.Contradictions))
	}

	t.Logf("✓ Aligned response passed validation")
	t.Logf("✓ Reason: %s", result.Reason)
}

// TestResponseValidatorContradictingResponse verifies contradicting response is blocked
func TestResponseValidatorContradictingResponse(t *testing.T) {
	t.Logf("Phase 3: Testing contradicting response detection")

	validator := NewResponseValidator(nil)

	// User says they're dominant
	extracted := []models.ExtractedEntity{
		{
			Value:      "dominant",
			Type:       "characteristic",
			Subject:    "user",
			Confidence: 0.95,
		},
	}

	// Response contradicts user (suggests submissive)
	response := "You might benefit from exploring your more submissive nature and letting others take the lead."

	result := validator.ValidateResponse(context.Background(), extracted, response)

	if result.IsValid {
		t.Fatal("Expected validation to fail for contradicting response")
	}

	if !result.ShouldBlock {
		t.Fatal("Should block contradicting response")
	}

	if result.Severity != "high" {
		t.Errorf("Expected severity 'high', got '%s'", result.Severity)
	}

	if len(result.Contradictions) == 0 {
		t.Fatal("Expected to detect contradictions")
	}

	t.Logf("✓ Contradicting response blocked")
	t.Logf("✓ Severity: %s", result.Severity)
	t.Logf("✓ Contradictions found: %d", len(result.Contradictions))
	t.Logf("✓ Reason: %s", result.Reason)
}

// TestResponseValidatorMultipleCharacteristics verifies validation with multiple characteristics
func TestResponseValidatorMultipleCharacteristics(t *testing.T) {
	t.Logf("Phase 3: Testing multiple characteristics validation")

	validator := NewResponseValidator(nil)

	// User says they're dominant AND adventurous
	extracted := []models.ExtractedEntity{
		{
			Value:      "dominant",
			Type:       "characteristic",
			Subject:    "user",
			Confidence: 0.95,
		},
		{
			Value:      "adventurous",
			Type:       "characteristic",
			Subject:    "user",
			Confidence: 0.90,
		},
	}

	// Response contradicts one characteristic (adventurous → cautious)
	response := "You should take a more cautious approach to new experiences."

	result := validator.ValidateResponse(context.Background(), extracted, response)

	if result.IsValid {
		t.Fatal("Expected validation to fail for contradicting response")
	}

	if len(result.Contradictions) == 0 {
		t.Fatal("Expected to detect contradiction with 'cautious'")
	}

	t.Logf("✓ Detected %d contradiction(s)", len(result.Contradictions))
	t.Logf("✓ Contradiction: %s", result.Contradictions[0].Evidence)
}

// TestResponseValidatorAntonymDetection verifies antonym pairs are detected
func TestResponseValidatorAntonymDetection(t *testing.T) {
	t.Logf("Phase 3: Testing antonym pair detection")

	validator := NewResponseValidator(nil)

	testCases := []struct {
		userChar     string
		responseChar string
		shouldDetect bool
	}{
		{"dominant", "submissive", true},
		{"assertive", "passive", true},
		{"independent", "dependent", true},
		{"outgoing", "introverted", true},
		{"ambitious", "content", true},
		{"adventurous", "cautious", true},
		{"romantic", "pragmatic", true},
		{"emotional", "logical", true},
		{"flexible", "rigid", true},
		{"generous", "frugal", true},
	}

	for _, tc := range testCases {
		extracted := []models.ExtractedEntity{
			{
				Value:      tc.userChar,
				Type:       "characteristic",
				Subject:    "user",
				Confidence: 0.95,
			},
		}

		response := "Consider exploring your more " + tc.responseChar + " approach."

		result := validator.ValidateResponse(context.Background(), extracted, response)

		if tc.shouldDetect && result.IsValid {
			t.Errorf("Should detect %s ↔ %s as antonyms, but validation passed",
				tc.userChar, tc.responseChar)
		}

		if tc.shouldDetect && !result.ShouldBlock {
			t.Errorf("Should block response for %s ↔ %s contradiction",
				tc.userChar, tc.responseChar)
		}

		t.Logf("✓ %s ↔ %s: antonym detection verified", tc.userChar, tc.responseChar)
	}
}

// TestResponseValidatorEmptyResponse handles empty response
func TestResponseValidatorEmptyResponse(t *testing.T) {
	t.Logf("Phase 3: Testing empty response handling")

	validator := NewResponseValidator(nil)

	extracted := []models.ExtractedEntity{
		{
			Value:      "dominant",
			Type:       "characteristic",
			Subject:    "user",
			Confidence: 0.95,
		},
	}

	result := validator.ValidateResponse(context.Background(), extracted, "")

	if !result.IsValid {
		t.Fatal("Empty response should pass validation")
	}

	t.Logf("✓ Empty response passes validation")
}

// TestResponseValidatorNoCharacteristics handles response with no characteristics
func TestResponseValidatorNoCharacteristics(t *testing.T) {
	t.Logf("Phase 3: Testing response with no characteristics")

	validator := NewResponseValidator(nil)

	extracted := []models.ExtractedEntity{
		{
			Value:      "dominant",
			Type:       "characteristic",
			Subject:    "user",
			Confidence: 0.95,
		},
	}

	response := "That's an interesting perspective. Tell me more about your goals."

	result := validator.ValidateResponse(context.Background(), extracted, response)

	if !result.IsValid {
		t.Fatal("Response with no characteristics should pass validation")
	}

	t.Logf("✓ Response with no characteristics passes validation")
}

// TestResponseValidatorShouldBlockMethod verifies ShouldBlockResponse works correctly
func TestResponseValidatorShouldBlockMethod(t *testing.T) {
	t.Logf("Phase 3: Testing ShouldBlockResponse method")

	validator := NewResponseValidator(nil)

	// Test case 1: High severity, should block
	result1 := &ValidationResult{
		IsValid:     false,
		Severity:    "high",
		ShouldBlock: true,
	}

	if !validator.ShouldBlockResponse(result1) {
		t.Fatal("Should block high-severity contradictions")
	}
	t.Logf("✓ Blocks high-severity results")

	// Test case 2: Low severity, should not block
	result2 := &ValidationResult{
		IsValid:     false,
		Severity:    "low",
		ShouldBlock: false,
	}

	if validator.ShouldBlockResponse(result2) {
		t.Fatal("Should not block low-severity results")
	}
	t.Logf("✓ Passes low-severity results")

	// Test case 3: Nil result
	if validator.ShouldBlockResponse(nil) {
		t.Fatal("Should not block nil result")
	}
	t.Logf("✓ Handles nil result safely")
}

// TestResponseValidatorClarificationQuestion verifies question generation
func TestResponseValidatorClarificationQuestion(t *testing.T) {
	t.Logf("Phase 3: Testing clarification question generation")

	validator := NewResponseValidator(nil)

	extracted := []models.ExtractedEntity{
		{
			Value:      "dominant",
			Type:       "characteristic",
			Subject:    "user",
			Confidence: 0.95,
		},
	}

	response := "You might explore your more submissive tendencies."

	result := validator.ValidateResponse(context.Background(), extracted, response)

	if result.RecommendedQuestion == "" {
		t.Fatal("Expected clarification question for contradiction")
	}

	if !stringContainsValidation(result.RecommendedQuestion, "dominant") ||
		!stringContainsValidation(result.RecommendedQuestion, "submissive") {
		t.Errorf("Question should mention both characteristics: %s", result.RecommendedQuestion)
	}

	t.Logf("✓ Question generated: %.80s...", result.RecommendedQuestion)
}

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
