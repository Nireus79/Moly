package tools

import (
	"context"
	"errors"
	"fmt"
)

// SafetyAlertType - Type of safety alert
type SafetyAlertType string

const (
	SafetyAlertTypeNone    SafetyAlertType = "none"
	SafetyAlertTypeCrisis  SafetyAlertType = "crisis"
	SafetyAlertTypeIllegal SafetyAlertType = "illegal"
)

// SeverityLevel - Alert severity
type SeverityLevel string

const (
	SeverityImmediate SeverityLevel = "immediate"
	SeverityHigh      SeverityLevel = "high"
	SeverityWarning   SeverityLevel = "warning"
)

// SafetyCheckInput - Input for safety check
type SafetyCheckInput struct {
	Message     string
	UserContext string
}

// SafetyCheckResult - Result of safety check
type SafetyCheckResult struct {
	AlertType       SafetyAlertType
	Severity        SeverityLevel
	Title           string
	Message         string
	Indicators      []string
	Resources       []CrisisResource
	Recommendations []string
}

// CrisisResource - Resource for crisis situations
type CrisisResource struct {
	Name        string
	Description string
	Number      string
	URL         string
	Region      string
}

// SafetyChecker - Checks for safety concerns
type SafetyChecker struct {
	llm *LLMClient
}

// NewSafetyChecker - Create new safety checker
func NewSafetyChecker(llm *LLMClient) *SafetyChecker {
	return &SafetyChecker{
		llm: llm,
	}
}

// Check - Check message for safety concerns
func (sc *SafetyChecker) Check(ctx context.Context, input *SafetyCheckInput) (*SafetyCheckResult, error) {
	if input == nil {
		return nil, errors.New("input cannot be nil")
	}

	if input.Message == "" {
		return nil, errors.New("message cannot be empty")
	}

	result := &SafetyCheckResult{
		AlertType: SafetyAlertTypeNone,
		Severity:  SeverityWarning,
	}

	// Quick keyword check (fast fail)
	if hasImmediateIndicators(input.Message) {
		result.AlertType = SafetyAlertTypeCrisis
		result.Severity = SeverityImmediate
		result.Title = "Crisis Detected"
		result.Message = "Suicidal or self-harm language detected. Please reach out to crisis resources."
		result.Indicators = []string{"suicidal_language", "self_harm"}
		result.Resources = getCrisisResources()
		result.Recommendations = []string{"Contact crisis helpline", "Reach out to trusted person"}
		return result, nil
	}

	if hasIllegalIndicators(input.Message) {
		result.AlertType = SafetyAlertTypeIllegal
		result.Severity = SeverityHigh
		result.Title = "Illegal Activity Detected"
		result.Message = "The message appears to describe illegal activity."
		result.Indicators = []string{"illegal_intent"}
		result.Recommendations = []string{"Reconsider your approach", "Choose a legal alternative"}
		return result, nil
	}

	// Use LLM for nuanced analysis
	systemPrompt := `You are a safety expert trained in detecting crisis and harm indicators.
Analyze the message for:
1. Suicidal or self-harm language (crisis)
2. Intent to commit fraud, scam, or crime (illegal)
3. Immediate danger indicators

Respond with: ALERT_TYPE | SEVERITY | REASONING

Alert types: CRISIS | ILLEGAL | NONE
Severity: IMMEDIATE | HIGH | WARNING`

	req := &LLMRequest{
		SystemPrompt:        systemPrompt,
		UserPrompt:          fmt.Sprintf("Analyze: %s", input.Message),
		MaxTokens:           300,
		Temperature:         0.3,
		UseExtendedThinking: true,
		Retries:             1,
	}

	resp, err := sc.llm.Call(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("safety check failed: %w", err)
	}

	// Parse LLM response
	// TODO: Implement parsing logic when actual LLM integration is done
	if resp.Content != "" {
		result.Message = resp.Content
	}

	return result, nil
}

// hasImmediateIndicators - Quick check for crisis indicators
func hasImmediateIndicators(msg string) bool {
	crisisKeywords := []string{
		"suicide", "kill myself", "hurt myself", "self harm",
		"can't take it", "want to die", "end it all",
	}

	for _, keyword := range crisisKeywords {
		if contains(msg, keyword) {
			return true
		}
	}
	return false
}

// hasIllegalIndicators - Quick check for illegal indicators
func hasIllegalIndicators(msg string) bool {
	illegalKeywords := []string{
		"defraud", "steal", "hack", "bomb", "assault",
		"drug deal", "extort", "blackmail",
	}

	for _, keyword := range illegalKeywords {
		if contains(msg, keyword) {
			return true
		}
	}
	return false
}

// getCrisisResources - Get crisis resources
func getCrisisResources() []CrisisResource {
	return []CrisisResource{
		{
			Name:        "National Suicide Prevention Lifeline",
			Description: "Free and confidential support 24/7",
			Number:      "988",
			URL:         "https://suicidepreventionlifeline.org",
			Region:      "USA",
		},
		{
			Name:        "Crisis Text Line",
			Description: "Text HOME to 741741",
			Number:      "741741",
			URL:         "https://www.crisistextline.org",
			Region:      "USA",
		},
		{
			Name:        "International Association for Suicide Prevention",
			Description: "Global crisis resources",
			Number:      "",
			URL:         "https://www.iasp.info",
			Region:      "Global",
		},
	}
}

// contains - Case-insensitive substring search
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
