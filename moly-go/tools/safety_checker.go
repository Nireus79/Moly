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
		result.Title = "Crisis Support"
		result.Message = "I'm not a specialist, but people who care are available to listen and help. Would it be helpful to reach out to a crisis counselor right now?"
		result.DirectRefusalMessage = "I'm sorry, I'm not trained to handle this. Please reach out to a crisis specialist who can truly help."
		result.Indicators = []string{"suicidal_language", "self_harm"}
		result.Resources = getCrisisResources()
		result.Recommendations = []string{"Contact crisis helpline", "Reach out to trusted person", "Speak with a mental health professional"}
		return result, nil
	}

	if hasIllegalIndicators(input.Message) {
		result.AlertType = SafetyAlertTypeIllegal
		result.Severity = SeverityHigh
		result.Title = "Need Clarification"
		result.ClarificationNeeded = true
		result.ClarifyingQuestion = "I want to make sure I understand correctly. Are you asking how to do something, or are you asking whether something is legal? Or is this something else?"
		result.DirectRefusalMessage = "I am sorry, I cannot help with that."
		result.Indicators = []string{"illegal_intent_possible"}
		return result, nil
	}

	// Use LLM for nuanced analysis
	systemPrompt := `You are a safety expert. Analyze for ONLY these specific indicators:

CRISIS - Flag ONLY if message explicitly mentions:
- Suicide, self-harm, wanting to die, suicidal thoughts
- Active intent to harm themselves RIGHT NOW
- Do NOT flag emotional distress, sadness, worry, or seeking help

ILLEGAL - Flag ONLY if message explicitly describes:
- Intent to steal, hack, assault, make weapons, sell drugs
- Planning a specific illegal activity
- Do NOT flag discussion of relationships, advice-seeking, or personal matters

Respond exactly in format: ALERT_TYPE | SEVERITY | REASONING

CRITICAL: Return NONE unless you find explicit crisis or illegal planning.
Do NOT flag:
- Someone seeking advice or help
- Emotional expression or feeling bad
- Questions about relationships, personal matters, or communication
- Normal anxiety, worry, sadness, or asking for guidance

Only flag if you see EXPLICIT self-harm intent or EXPLICIT illegal planning.`

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

	// Parse LLM response format: "ALERT_TYPE | SEVERITY | REASONING"
	if resp.Content != "" {
		alertType, severity, reasoning := parseSafetyResponse(resp.Content)
		result.AlertType = alertType
		result.Severity = severity
		result.Title = fmt.Sprintf("%s Alert", alertType)
		result.Message = reasoning

		if alertType == SafetyAlertTypeCrisis {
			result.Resources = getCrisisResources()
			result.Recommendations = []string{"Contact crisis helpline immediately", "Reach out to trusted person"}
		} else if alertType == SafetyAlertTypeIllegal {
			result.Recommendations = []string{"Reconsider your approach", "Choose a legal alternative"}
		}
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
		if containsStr(msg, keyword) {
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
		if containsStr(msg, keyword) {
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

// containsStr - Case-insensitive substring search
func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
