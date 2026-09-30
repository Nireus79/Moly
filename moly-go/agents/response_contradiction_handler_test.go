package agents

import (
	"context"
	"testing"
)

// TestResponseContradictionHandlerQuestionGeneration verifies question generation
func TestResponseContradictionHandlerQuestionGeneration(t *testing.T) {
	t.Logf("Phase 3: Testing contradiction question generation")

	handler := NewResponseContradictionHandler(nil)

	contradiction := Contradiction{
		ResponseCharacteristic: "submissive",
		UserCharacteristic:     "dominant",
		IsAntonym:              true,
		Evidence:               "Response suggests submissive approach",
		Confidence:             0.95,
	}

	question := handler.GenerateContradictionQuestion(
		context.Background(),
		contradiction,
		"test_user",
		"test_conv",
		"You might benefit from exploring your more submissive side",
	)

	if question == nil {
		t.Fatal("Question should not be nil")
	}

	if question.QuestionText == "" {
		t.Fatal("Question text should not be empty")
	}

	if question.ClarificationType != "response_contradiction" {
		t.Errorf("Wrong type: expected 'response_contradiction', got '%s'", question.ClarificationType)
	}

	if question.Priority != 3 {
		t.Errorf("Wrong priority: expected 3 (highest), got %d", question.Priority)
	}

	t.Logf("✓ Question generated with type: %s", question.ClarificationType)
	t.Logf("✓ Priority set to highest (3)")
	t.Logf("✓ Question text: %.80s...", question.QuestionText)
}

// TestResponseContradictionHandlerMultipleContradictions verifies handling multiple contradictions
func TestResponseContradictionHandlerMultipleContradictions(t *testing.T) {
	t.Logf("Phase 3: Testing multiple contradictions handling")

	handler := NewResponseContradictionHandler(nil)

	contradictions := []Contradiction{
		{
			ResponseCharacteristic: "submissive",
			UserCharacteristic:     "dominant",
			IsAntonym:              true,
			Confidence:             0.95,
		},
		{
			ResponseCharacteristic: "cautious",
			UserCharacteristic:     "adventurous",
			IsAntonym:              true,
			Confidence:             0.90,
		},
	}

	question := handler.GenerateMultipleContradictionQuestion(
		context.Background(),
		contradictions,
		"test_user",
		"test_conv",
	)

	if question == nil {
		t.Fatal("Question should not be nil")
	}

	if question.ClarificationType != "response_contradiction_multiple" {
		t.Errorf("Wrong type: %s", question.ClarificationType)
	}

	if len(question.LinkedFacts) < 2 {
		t.Errorf("Should link multiple contradictions, got %d facts", len(question.LinkedFacts))
	}

	t.Logf("✓ Multiple contradiction question generated")
	t.Logf("✓ Linked %d contradictions", len(question.LinkedFacts))
}

// TestResponseContradictionHandlerExplainContradiction verifies explanation generation
func TestResponseContradictionHandlerExplainContradiction(t *testing.T) {
	t.Logf("Phase 3: Testing contradiction explanation")

	handler := NewResponseContradictionHandler(nil)

	contradiction := Contradiction{
		ResponseCharacteristic: "submissive",
		UserCharacteristic:     "dominant",
		IsAntonym:              true,
		Evidence:               "Response suggests exploring submissive tendencies",
		Confidence:             0.95,
	}

	explanation := handler.ExplainContradiction(contradiction)

	if explanation == "" {
		t.Fatal("Explanation should not be empty")
	}

	if !stringContainsContradictionHandler(explanation, "dominant") {
		t.Error("Should mention user characteristic")
	}

	if !stringContainsContradictionHandler(explanation, "submissive") {
		t.Error("Should mention response characteristic")
	}

	if !stringContainsContradictionHandler(explanation, "95") {
		t.Error("Should include confidence percentage")
	}

	t.Logf("✓ Explanation includes all details")
	t.Logf("✓ Explanation: %.100s...", explanation)
}

// TestResponseContradictionHandlerGetSummary verifies summary generation
func TestResponseContradictionHandlerGetSummary(t *testing.T) {
	t.Logf("Phase 3: Testing summary generation")

	handler := NewResponseContradictionHandler(nil)

	// Empty contradictions
	summary := handler.GetSummary([]Contradiction{})
	if !stringContainsContradictionHandler(summary, "No contradictions") {
		t.Error("Summary should indicate no contradictions")
	}
	t.Logf("✓ Empty summary: %s", summary)

	// With contradictions
	contradictions := []Contradiction{
		{
			ResponseCharacteristic: "submissive",
			UserCharacteristic:     "dominant",
			Confidence:             0.95,
		},
		{
			ResponseCharacteristic: "cautious",
			UserCharacteristic:     "adventurous",
			Confidence:             0.90,
		},
	}

	summary = handler.GetSummary(contradictions)
	if !stringContainsContradictionHandler(summary, "2") {
		t.Error("Summary should show count of contradictions")
	}

	if !stringContainsContradictionHandler(summary, "dominant") {
		t.Error("Summary should list characteristics")
	}

	t.Logf("✓ Multi-contradiction summary includes all details")
}

// TestResponseContradictionHandlerLogBlockedResponse verifies logging
func TestResponseContradictionHandlerLogBlockedResponse(t *testing.T) {
	t.Logf("Phase 3: Testing blocked response logging")

	handler := NewResponseContradictionHandler(nil)

	contradictions := []Contradiction{
		{
			ResponseCharacteristic: "submissive",
			UserCharacteristic:     "dominant",
			Confidence:             0.95,
		},
	}

	log := handler.LogBlockedResponse(
		"test_user",
		"test_conv",
		"You should explore your submissive side",
		contradictions,
		"q_123",
	)

	if log.UserID != "test_user" {
		t.Error("Log should contain user ID")
	}

	if log.ClarificationID != "q_123" {
		t.Error("Log should contain clarification ID")
	}

	if len(log.Contradictions) != 1 {
		t.Error("Log should contain contradictions")
	}

	t.Logf("✓ Blocked response log created")
	t.Logf("✓ Log timestamp: %d", log.Timestamp)
}

// TestResponseContradictionHandlerQuestionFields verifies all question fields are set
func TestResponseContradictionHandlerQuestionFields(t *testing.T) {
	t.Logf("Phase 3: Testing question field completeness")

	handler := NewResponseContradictionHandler(nil)

	contradiction := Contradiction{
		ResponseCharacteristic: "submissive",
		UserCharacteristic:     "dominant",
		IsAntonym:              true,
		Evidence:               "Test evidence",
		Confidence:             0.95,
	}

	question := handler.GenerateContradictionQuestion(
		context.Background(),
		contradiction,
		"user123",
		"conv456",
		"Test response text",
	)

	// Check all required fields
	checks := map[string]bool{
		"ID":                question.ID != "",
		"UserID":            question.UserID == "user123",
		"ConversationID":    question.ConversationID == "conv456",
		"ClarificationType": question.ClarificationType == "response_contradiction",
		"Priority":          question.Priority == 3,
		"Status":            question.Status == "pending",
		"QuestionText":      question.QuestionText != "",
		"ContextNotes":      question.ContextNotes != "",
		"LinkedFacts":       len(question.LinkedFacts) > 0,
		"CreatedAt":         question.CreatedAt > 0,
	}

	allOK := true
	for field, ok := range checks {
		if !ok {
			t.Errorf("Field check failed: %s", field)
			allOK = false
		} else {
			t.Logf("✓ %s correct", field)
		}
	}

	if !allOK {
		t.Fatal("Some fields were not set correctly")
	}
}

// TestResponseContradictionHandlerNaturalLanguage verifies questions are conversational
func TestResponseContradictionHandlerNaturalLanguage(t *testing.T) {
	t.Logf("Phase 3: Testing natural language quality")

	handler := NewResponseContradictionHandler(nil)

	contradiction := Contradiction{
		ResponseCharacteristic: "submissive",
		UserCharacteristic:     "dominant",
		Confidence:             0.95,
	}

	question := handler.GenerateContradictionQuestion(
		context.Background(),
		contradiction,
		"test_user",
		"test_conv",
		"Test response",
	)

	// Check for conversational markers
	qualityChecks := map[string]bool{
		"First person":   stringContainsContradictionHandler(question.QuestionText, "I"),
		"Supportive tone": stringContainsContradictionHandler(question.QuestionText, "help") ||
		                  stringContainsContradictionHandler(question.QuestionText, "clarify"),
		"Not accusatory": !stringContainsContradictionHandler(question.QuestionText, "wrong") &&
		                 !stringContainsContradictionHandler(question.QuestionText, "incorrect"),
		"Acknowledges nuance": stringContainsContradictionHandler(question.QuestionText, "both") ||
		                       stringContainsContradictionHandler(question.QuestionText, "contexts") ||
		                       stringContainsContradictionHandler(question.QuestionText, "different"),
	}

	for aspect, ok := range qualityChecks {
		if ok {
			t.Logf("✓ %s present", aspect)
		} else {
			t.Logf("⚠ %s missing", aspect)
		}
	}

	t.Logf("✓ Question text: %.100s...", question.QuestionText)
}

// Helper function
func stringContainsContradictionHandler(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
