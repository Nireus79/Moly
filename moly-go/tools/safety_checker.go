package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	AlertType            SafetyAlertType
	Severity             SeverityLevel
	Title                string
	Message              string
	Indicators           []string
	Resources            []CrisisResource
	Recommendations      []string
	ClarificationNeeded  bool   // Try clarifying questions first
	ClarifyingQuestion   string // What to ask for clarification
	DirectRefusalMessage string // If user persists: "I am sorry, I cannot do that"
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
	llm LLMProvider
}

// NewSafetyChecker - Create new safety checker
func NewSafetyChecker(llm LLMProvider) *SafetyChecker {
	return &SafetyChecker{
		llm: llm,
	}
}

// Check - LLM-driven safety analysis (no static keywords)
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

	// LLM-only analysis: no static keyword detection
	systemPrompt := `You are a safety expert analyzing a message for crisis or illegal intent.

CRISIS INDICATORS - Flag ONLY if message shows:
- EXPLICIT mention of suicide, self-harm, or wanting to die
- ACTIVE immediate intent to harm themselves or others
- Do NOT flag emotional distress, sadness, worry, or asking for help

ILLEGAL INDICATORS - Flag ONLY if message shows:
- EXPLICIT intent to commit an illegal act (steal, hack, assault, etc.)
- Planning a specific illegal activity
- Do NOT flag ethical discussions, questions about legality, or advice-seeking

Respond in this format:
ALERT_TYPE | SEVERITY | INDICATOR | REASON

Where ALERT_TYPE is: NONE | CRISIS | ILLEGAL
Where SEVERITY is: IMMEDIATE | HIGH | WARNING (or NONE if ALERT_TYPE is NONE)

Critical rules:
- Return NONE if message is just seeking help, expressing emotions, or asking questions
- Return NONE if message describes concerning thoughts but NOT active intent
- Only flag CRISIS if you see explicit self-harm intent
- Only flag ILLEGAL if you see explicit planning of illegal activity`

	req := &LLMRequest{
		SystemPrompt:        systemPrompt,
		UserPrompt:          fmt.Sprintf("Analyze this message:\n\n%s", input.Message),
		MaxTokens:           300,
		Temperature:         0.2, // Very strict interpretation
		UseExtendedThinking: true,
		Retries:             1,
	}

	resp, err := sc.llm.Call(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("safety check failed: %w", err)
	}

	// Parse LLM response
	if resp.Content != "" {
		alertType, severity, indicator, reasoning := parseSafetyResponseV2(resp.Content)
		result.AlertType = alertType
		result.Severity = severity
		result.Title = fmt.Sprintf("%s Alert", alertType)
		result.Message = reasoning
		result.Indicators = []string{indicator}

		if alertType == SafetyAlertTypeCrisis {
			result.Resources = getCrisisResources()
			result.Recommendations = []string{"Contact crisis helpline immediately", "Reach out to trusted person"}
		} else if alertType == SafetyAlertTypeIllegal {
			result.ClarificationNeeded = true
			result.ClarifyingQuestion = "I want to make sure I understand. Are you asking about this topic, or describing something you're planning? Help me understand your actual intent."
			result.Recommendations = []string{"Reconsider your approach", "Choose a legal alternative"}
		}
	}

	return result, nil
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

// parseSafetyResponse - Parse LLM response in format "ALERT_TYPE | SEVERITY | REASONING"
func parseSafetyResponse(content string) (SafetyAlertType, SeverityLevel, string) {
	if content == "" {
		return SafetyAlertTypeNone, SeverityWarning, ""
	}

	parts := strings.Split(content, "|")
	if len(parts) < 3 {
		return SafetyAlertTypeNone, SeverityWarning, content
	}

	alertTypeStr := strings.TrimSpace(parts[0])
	severityStr := strings.TrimSpace(parts[1])
	reasoning := strings.TrimSpace(parts[2])

	alertType := SafetyAlertTypeNone
	if strings.ToUpper(alertTypeStr) == "CRISIS" {
		alertType = SafetyAlertTypeCrisis
	} else if strings.ToUpper(alertTypeStr) == "ILLEGAL" {
		alertType = SafetyAlertTypeIllegal
	}

	severity := SeverityWarning
	if strings.ToUpper(severityStr) == "IMMEDIATE" {
		severity = SeverityImmediate
	} else if strings.ToUpper(severityStr) == "HIGH" {
		severity = SeverityHigh
	}

	return alertType, severity, reasoning
}

// parseSafetyResponseV2 - Parse LLM response format: "ALERT_TYPE | SEVERITY | INDICATOR | REASON"
func parseSafetyResponseV2(response string) (SafetyAlertType, SeverityLevel, string, string) {
	parts := strings.Split(response, "|")
	if len(parts) < 4 {
		// Fallback: assume NONE if we can't parse
		return SafetyAlertTypeNone, SeverityWarning, "", strings.TrimSpace(response)
	}

	alertTypeStr := strings.TrimSpace(parts[0])
	severityStr := strings.TrimSpace(parts[1])
	indicator := strings.TrimSpace(parts[2])
	reason := strings.TrimSpace(parts[3])

	alertType := SafetyAlertTypeNone
	if strings.ToUpper(alertTypeStr) == "CRISIS" {
		alertType = SafetyAlertTypeCrisis
	} else if strings.ToUpper(alertTypeStr) == "ILLEGAL" {
		alertType = SafetyAlertTypeIllegal
	}

	severity := SeverityWarning
	if strings.ToUpper(severityStr) == "IMMEDIATE" {
		severity = SeverityImmediate
	} else if strings.ToUpper(severityStr) == "HIGH" {
		severity = SeverityHigh
	}

	return alertType, severity, indicator, reason
}

// containsStr - Case-insensitive substring search
func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
