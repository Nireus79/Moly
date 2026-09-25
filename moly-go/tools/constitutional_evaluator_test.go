package tools

import (
	"context"
	"errors"
	"testing"

	"moly/config"
	"moly/models"
)

// loadTestConstitution tries to load constitution from multiple paths for test compatibility
func loadTestConstitution(t *testing.T) *models.Constitution {
	paths := []string{
		"config/constitution.yaml",
		"../config/constitution.yaml",
		"../../config/constitution.yaml",
	}

	var constitution *models.Constitution
	var err error

	for _, p := range paths {
		constitution, err = config.LoadConstitution(p)
		if err == nil {
			return constitution
		}
	}

	t.Fatalf("Failed to load constitution from any path: %v", err)
	return nil
}

// MockLLMProvider returns a fixed response for testing
type MockLLMProvider struct {
	response string
	callCount int
	shouldFail bool
}

func (m *MockLLMProvider) Call(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	m.callCount++
	if m.shouldFail {
		return nil, errors.New("mock LLM failure")
	}
	return &LLMResponse{Content: m.response}, nil
}

// TestZeroSignalAllows verifies that benign text is allowed via principle-based evaluation
func TestZeroSignalAllows(t *testing.T) {
	constitution := loadTestConstitution(t)

	// Mock LLM returns no violations for benign message
	mockLLM := &MockLLMProvider{
		response: `{"violations": []}`,
	}

	evaluator := NewConstitutionalEvaluator(mockLLM, constitution)

	verdict, err := evaluator.Evaluate(context.Background(), "Hello Moly")
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	if !verdict.Allowed {
		t.Errorf("Expected allowed=true, got false for benign message")
	}

	if verdict.OverallSeverity != "clear" {
		t.Errorf("Expected severity='clear', got %q", verdict.OverallSeverity)
	}

	if len(verdict.MatchedPrinciples) != 0 {
		t.Errorf("Expected 0 matched principles, got %d", len(verdict.MatchedPrinciples))
	}

	// LLM should be called for all messages (principle-based evaluation)
	if mockLLM.callCount != 1 {
		t.Errorf("Expected 1 LLM call, got %d", mockLLM.callCount)
	}
}

// TestHarmPrincipleBlocks verifies that harm_prevention violations block the message
func TestHarmPrincipleBlocks(t *testing.T) {
	constitution := loadTestConstitution(t)

	// Mock LLM detects harm_prevention violation
	mockLLM := &MockLLMProvider{
		response: `{"violations": [{"principle_id": "harm_prevention", "evidence": "I want to hurt myself", "reasoning": "Explicit self-harm intent"}]}`,
	}

	evaluator := NewConstitutionalEvaluator(mockLLM, constitution)

	verdict, err := evaluator.Evaluate(context.Background(), "I want to hurt myself")
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	if verdict.Allowed {
		t.Errorf("Expected allowed=false for harm_prevention violation, got true")
	}

	if verdict.OverallSeverity != "critical" {
		t.Errorf("Expected severity='critical' for harm_prevention, got %q", verdict.OverallSeverity)
	}

	if len(verdict.MatchedPrinciples) != 1 {
		t.Errorf("Expected 1 matched principle, got %d", len(verdict.MatchedPrinciples))
	}

	if verdict.MatchedPrinciples[0].PrincipleID != "harm_prevention" {
		t.Errorf("Expected principle_id='harm_prevention', got %q", verdict.MatchedPrinciples[0].PrincipleID)
	}

	// Verify severity is forced to YAML value, not LLM's guess
	if verdict.MatchedPrinciples[0].Severity != "critical" {
		t.Errorf("Expected severity='critical' (from YAML), got %q", verdict.MatchedPrinciples[0].Severity)
	}
}

// TestHallucinatedPrincipleDropped verifies that non-existent principles are rejected
func TestHallucinatedPrincipleDropped(t *testing.T) {
	constitution := loadTestConstitution(t)

	// Mock LLM hallucinates a non-existent principle
	mockLLM := &MockLLMProvider{
		response: `{"violations": [{"principle_id": "made_up_principle", "evidence": "some text", "reasoning": "fake"}]}`,
	}

	evaluator := NewConstitutionalEvaluator(mockLLM, constitution)

	verdict, err := evaluator.Evaluate(context.Background(), "some text")
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	// Hallucinated principle should be dropped
	if len(verdict.MatchedPrinciples) != 0 {
		t.Errorf("Expected hallucinated principle to be dropped, got %d matches", len(verdict.MatchedPrinciples))
	}

	if !verdict.Allowed {
		t.Errorf("Expected allowed=true when hallucination is dropped, got false")
	}
}

// TestUnevidencedViolationDropped verifies that unsupported evidence is rejected
func TestUnevidencedViolationDropped(t *testing.T) {
	constitution := loadTestConstitution(t)

	// Mock LLM claims evidence that doesn't exist in the message
	mockLLM := &MockLLMProvider{
		response: `{"violations": [{"principle_id": "harm_prevention", "evidence": "text not in message", "reasoning": "hallucinated evidence"}]}`,
	}

	evaluator := NewConstitutionalEvaluator(mockLLM, constitution)

	verdict, err := evaluator.Evaluate(context.Background(), "Hello Moly")
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	// Unevidenced violation should be dropped
	if len(verdict.MatchedPrinciples) != 0 {
		t.Errorf("Expected unevidenced violation to be dropped, got %d matches", len(verdict.MatchedPrinciples))
	}

	if !verdict.Allowed {
		t.Errorf("Expected allowed=true when evidence is missing, got false")
	}
}

// TestSeverityForcedFromYAML verifies that severity always comes from constitution
func TestSeverityForcedFromYAML(t *testing.T) {
	constitution := loadTestConstitution(t)

	// Mock LLM claims this is "critical" but it's actually "medium" in YAML
	mockLLM := &MockLLMProvider{
		response: `{"violations": [{"principle_id": "growth_and_learning", "evidence": "quickly", "reasoning": "LLM says critical"}]}`,
	}

	evaluator := NewConstitutionalEvaluator(mockLLM, constitution)

	verdict, err := evaluator.Evaluate(context.Background(), "I want to learn quickly")
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	if len(verdict.MatchedPrinciples) == 0 {
		t.Fatalf("Expected 1 matched principle, got 0")
	}

	// Severity should be "medium" from YAML, NOT what LLM claimed
	if verdict.MatchedPrinciples[0].Severity != "medium" {
		t.Errorf("Expected severity='medium' (from YAML), got %q (should not be LLM's claim)",
			verdict.MatchedPrinciples[0].Severity)
	}
}

// TestEmptyMessageAllowed verifies that empty messages don't error
func TestEmptyMessageAllowed(t *testing.T) {
	constitution := loadTestConstitution(t)

	mockLLM := &MockLLMProvider{}
	evaluator := NewConstitutionalEvaluator(mockLLM, constitution)

	verdict, err := evaluator.Evaluate(context.Background(), "")
	if err != nil {
		t.Fatalf("Evaluate failed on empty message: %v", err)
	}

	if !verdict.Allowed {
		t.Errorf("Expected allowed=true for empty message")
	}

	// LLM should NOT have been called for empty message
	if mockLLM.callCount != 0 {
		t.Errorf("Expected 0 LLM calls for empty message, got %d", mockLLM.callCount)
	}
}

// TestMultiplePrincipleMatches verifies that multiple violations are collected in Tier 2
// This test uses keywords that trigger signals without Tier 1a hard-blocks
func TestMultiplePrincipleMatches(t *testing.T) {
	constitution := loadTestConstitution(t)

	// Mock LLM detects multiple violations (no Tier 1a hard-blocks in this message)
	mockLLM := &MockLLMProvider{
		response: `{"violations": [
			{"principle_id": "harm_prevention", "evidence": "harm", "reasoning": "potential harm"},
			{"principle_id": "user_autonomy", "evidence": "should", "reasoning": "autonomy violation"}
		]}`,
	}

	evaluator := NewConstitutionalEvaluator(mockLLM, constitution)

	// Message has principle signals (harm, should) but no Tier 1a hard-block phrases
	verdict, err := evaluator.Evaluate(context.Background(), "What harm should I do to myself?")
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	if len(verdict.MatchedPrinciples) != 2 {
		t.Errorf("Expected 2 matched principles, got %d", len(verdict.MatchedPrinciples))
	}

	// Max severity should be critical (from harm_prevention)
	if verdict.OverallSeverity != "critical" {
		t.Errorf("Expected severity='critical' (max of harm_prevention and user_autonomy), got %q",
			verdict.OverallSeverity)
	}

	if verdict.Allowed {
		t.Errorf("Expected allowed=false when critical principle is violated")
	}
}

// TestToSafetyAlert verifies verdict -> SafetyAlert conversion
func TestToSafetyAlert(t *testing.T) {
	verdict := &ConstitutionalVerdict{
		Allowed:         false,
		OverallSeverity: "critical",
		MatchedPrinciples: []PrincipleMatch{
			{
				PrincipleID: "harm_prevention",
				Name:        "Harm Prevention",
				Severity:    "critical",
				Evidence:    "I want to hurt myself",
			},
		},
		Reasoning: "Harm principle violated",
		Confidence: 0.95,
	}

	alert := verdict.ToSafetyAlert()
	if alert == nil {
		t.Fatalf("Expected non-nil SafetyAlert")
	}

	if alert.AlertType != "crisis" {
		t.Errorf("Expected alert_type='crisis', got %q", alert.AlertType)
	}

	if alert.Severity != "immediate" {
		t.Errorf("Expected severity='immediate', got %q", alert.Severity)
	}
}

// TestToRiskAssessment verifies verdict -> RiskAssessment conversion
func TestToRiskAssessment(t *testing.T) {
	verdict := &ConstitutionalVerdict{
		Allowed:         false,
		OverallSeverity: "high",
		MatchedPrinciples: []PrincipleMatch{
			{
				PrincipleID: "consent_and_respect",
				Severity:    "high",
			},
		},
		Reasoning:  "Consent principle violated",
		Confidence: 0.90,
	}

	ra := verdict.ToRiskAssessment()
	if ra == nil {
		t.Fatalf("Expected non-nil RiskAssessment")
	}

	if ra.RiskLevel != "elevated" {
		t.Errorf("Expected risk_level='elevated', got %q", ra.RiskLevel)
	}

	if ra.Recommendation != "caution" {
		t.Errorf("Expected recommendation='caution', got %q", ra.Recommendation)
	}
}
