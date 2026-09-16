package tools

import (
	"context"
	"testing"

	"moly/models"
)

// TestHarmAnalyzerPrincipleViolation verifies principle violations are detected and added to response
func TestHarmAnalyzerPrincipleViolation(t *testing.T) {
	// Create mock LLM with response override for harm analysis
	mockLLM := NewMockLLMClient()
	mockLLM.SetResponseOverride("analyze", `{
		"intent": "advice about violence",
		"harm_type": "physical violence",
		"severity": "critical",
		"affected_parties": ["third party"],
		"reasoning": "Response suggests violent action",
		"intervention": "BLOCK",
		"suggested_alternative": "Suggest legal options instead",
		"modified_response": "",
		"explanation": "I cannot provide advice about violence",
		"violated_principles": ["Safety", "Non-harm"]
	}`)

	// Create analyzer with constitution
	analyzer := NewHarmAnalyzer(mockLLM)
	constitution := &models.Constitution{
		SupremePrinciples: []models.Principle{
			{
				Name:           "Safety",
				Severity:       "critical",
				Description:    "Protect physical safety",
				CheckKeywords:  []string{"hit", "punch", "violence", "hurt"},
			},
			{
				Name:           "Non-harm",
				Severity:       "critical",
				Description:    "Avoid causing harm",
				CheckKeywords:  []string{"harm", "injury", "damage"},
			},
		},
	}
	analyzer.SetConstitution(constitution)

	// Test with response containing violation keywords
	response := "You should hit them when they disagree with you to cause harm. It will show them who is in charge."
	userVuln := &UserVulnerability{
		TraumaHistory:  true,
		Confidence:     "low",
	}
	contactTraits := &ContactTraits{
		Relationship: "romantic",
		Sensitivity:  "high",
	}

	analysis, err := analyzer.AnalyzeResponse(context.Background(), response, userVuln, contactTraits)

	// Verify no error
	if err != nil {
		t.Fatalf("AnalyzeResponse returned error: %v", err)
	}

	// Verify violation was detected
	if analysis == nil {
		t.Fatal("AnalyzeResponse returned nil analysis")
	}

	if len(analysis.ViolatedPrinciples) == 0 {
		t.Error("Expected violated principles to be detected, got none")
	}

	// Verify all expected principles are present
	expectedPrinciples := map[string]bool{
		"Safety":   false,
		"Non-harm": false,
	}
	for _, p := range analysis.ViolatedPrinciples {
		expectedPrinciples[p] = true
	}

	for principle, found := range expectedPrinciples {
		if !found {
			t.Errorf("Expected principle %q not found in violation list", principle)
		}
	}

	// Verify intervention is appropriate for critical violations
	if analysis.Intervention != "BLOCK" {
		t.Errorf("Expected intervention BLOCK for critical violation, got %s", analysis.Intervention)
	}

	// Verify severity
	if analysis.Severity != "critical" {
		t.Errorf("Expected severity critical, got %s", analysis.Severity)
	}
}

// TestHarmAnalyzerNoViolation verifies responses without violations are handled correctly
func TestHarmAnalyzerNoViolation(t *testing.T) {
	mockLLM := NewMockLLMClient()
	mockLLM.SetResponseOverride("conversation", `{
		"intent": "general conversation",
		"harm_type": "none",
		"severity": "none",
		"affected_parties": [],
		"reasoning": "This response is safe",
		"intervention": "PROCEED",
		"suggested_alternative": "",
		"modified_response": "",
		"explanation": "",
		"violated_principles": []
	}`)

	analyzer := NewHarmAnalyzer(mockLLM)
	constitution := &models.Constitution{
		SupremePrinciples: []models.Principle{
			{
				Name:           "Safety",
				Severity:       "critical",
				Description:    "Protect physical safety",
				CheckKeywords:  []string{"hit", "punch", "violence"},
			},
		},
	}
	analyzer.SetConstitution(constitution)

	response := "You should try calling them to have an honest conversation."

	analysis, err := analyzer.AnalyzeResponse(context.Background(), response, nil, nil)

	if err != nil {
		t.Fatalf("AnalyzeResponse returned error: %v", err)
	}

	if analysis == nil {
		t.Fatal("AnalyzeResponse returned nil")
	}

	if len(analysis.ViolatedPrinciples) > 0 {
		t.Errorf("Expected no violations for safe response, got %v", analysis.ViolatedPrinciples)
	}

	if analysis.Intervention != "PROCEED" {
		t.Errorf("Expected PROCEED for safe response, got %s", analysis.Intervention)
	}

	if analysis.Severity != "none" {
		t.Errorf("Expected severity none, got %s", analysis.Severity)
	}
}

// TestHarmAnalyzerEmptyResponse verifies empty responses are handled gracefully
func TestHarmAnalyzerEmptyResponse(t *testing.T) {
	analyzer := NewHarmAnalyzer(&MockLLMClient{})

	analysis, err := analyzer.AnalyzeResponse(context.Background(), "", nil, nil)

	if err != nil {
		t.Fatalf("Empty response should not error, got: %v", err)
	}

	if analysis == nil {
		t.Fatal("AnalyzeResponse returned nil for empty input")
	}

	if analysis.Severity != "none" {
		t.Errorf("Expected severity none for empty response, got %s", analysis.Severity)
	}

	if analysis.Intervention != "PROCEED" {
		t.Errorf("Expected PROCEED for empty response, got %s", analysis.Intervention)
	}
}

// TestHarmAnalyzerLLMFailure verifies graceful degradation when LLM fails
func TestHarmAnalyzerLLMFailure(t *testing.T) {
	mockLLM := NewMockLLMClientWithMode("error")

	analyzer := NewHarmAnalyzer(mockLLM)

	analysis, err := analyzer.AnalyzeResponse(context.Background(), "Some response", nil, nil)

	// Should not return error - graceful degradation
	if err != nil {
		t.Fatalf("Should degrade gracefully on LLM failure, got error: %v", err)
	}

	if analysis == nil {
		t.Fatal("AnalyzeResponse returned nil on LLM failure")
	}

	// Should default to safe behavior (no intervention)
	if analysis.Intervention != "PROCEED" {
		t.Errorf("Expected PROCEED on LLM failure, got %s", analysis.Intervention)
	}

	if analysis.Severity != "none" {
		t.Errorf("Expected severity none on LLM failure, got %s", analysis.Severity)
	}
}

// TestCheckPrinciples verifies keyword matching for principles
func TestCheckPrinciples(t *testing.T) {
	analyzer := NewHarmAnalyzer(&MockLLMClient{})
	constitution := &models.Constitution{
		SupremePrinciples: []models.Principle{
			{
				Name:           "Safety",
				Severity:       "critical",
				Description:    "Protect physical safety",
				CheckKeywords:  []string{"hit", "punch", "violence", "hurt"},
			},
			{
				Name:           "Consent",
				Severity:       "high",
				Description:    "Respect consent",
				CheckKeywords:  []string{"without consent", "force", "pressure"},
			},
		},
	}
	analyzer.SetConstitution(constitution)

	tests := []struct {
		name          string
		response      string
		expectedCount int
		expectedNames []string
	}{
		{
			name:          "Single principle violation",
			response:      "You should hit them",
			expectedCount: 1,
			expectedNames: []string{"Safety"},
		},
		{
			name:          "Multiple principle violations",
			response:      "You should force them without consent to hit them",
			expectedCount: 2,
			expectedNames: []string{"Consent", "Safety"},
		},
		{
			name:          "No violations",
			response:      "You should have a conversation with them",
			expectedCount: 0,
			expectedNames: []string{},
		},
		{
			name:          "Case-insensitive matching",
			response:      "You should HURT them",
			expectedCount: 1,
			expectedNames: []string{"Safety"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations := analyzer.CheckPrinciples(tt.response)

			if len(violations) != tt.expectedCount {
				t.Errorf("Expected %d violations, got %d", tt.expectedCount, len(violations))
			}

			violatedNames := make(map[string]bool)
			for _, v := range violations {
				violatedNames[v.Name] = true
			}

			for _, expected := range tt.expectedNames {
				if !violatedNames[expected] {
					t.Errorf("Expected principle %q not found in violations", expected)
				}
			}
		})
	}
}

// TestConstitutionInjection verifies constitution can be set after creation
func TestConstitutionInjection(t *testing.T) {
	analyzer := NewHarmAnalyzer(NewMockLLMClient())

	if analyzer.constitution != nil {
		t.Error("Expected analyzer to start without constitution")
	}

	constitution := &models.Constitution{
		SupremePrinciples: []models.Principle{
			{
				Name:           "Test",
				Severity:       "high",
				Description:    "Test principle",
				CheckKeywords:  []string{"test"},
			},
		},
	}

	analyzer.SetConstitution(constitution)

	if analyzer.constitution == nil {
		t.Error("Constitution was not injected")
	}

	if len(analyzer.constitution.SupremePrinciples) != 1 {
		t.Errorf("Expected 1 principle, got %d", len(analyzer.constitution.SupremePrinciples))
	}
}
