package agents

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"moly/models"
	"moly/safety"
	"moly/tools"
)

// SafetyRiskAnalyzer combines safety check and risk assessment into one LLM call
// This reduces latency and tokens while maintaining quality
type SafetyRiskAnalyzer struct {
	llmClient tools.LLMProvider
}

// NewSafetyRiskAnalyzer creates a new combined analyzer
func NewSafetyRiskAnalyzer(llm tools.LLMProvider) *SafetyRiskAnalyzer {
	return &SafetyRiskAnalyzer{
		llmClient: llm,
	}
}

// BatchAnalyze performs both safety check and risk assessment in a single LLM call
// Returns (safetyAlert, riskAssessment, error)
func (sra *SafetyRiskAnalyzer) BatchAnalyze(message string) (*safety.SafetyAlert, *models.RiskAssessment, error) {
	if message == "" {
		log.Printf("[SafetyRiskAnalyzer] Empty message, returning nil for both")
		return nil, nil, nil
	}

	if sra.llmClient == nil {
		log.Printf("[SafetyRiskAnalyzer] No LLM available, cannot analyze")
		return nil, nil, nil
	}

	message = strings.TrimSpace(message)
	log.Printf("[SafetyRiskAnalyzer] Batch analyzing message: %d chars", len(message))

	// Combined prompt that asks for both checks in one call
	systemPrompt := `You are a safety expert. Analyze this message for BOTH safety concerns AND contextual risks.

SAFETY CHECK: Determine if this is a crisis (imminent self-harm/suicide) or illegal request
- "crisis": Person explicitly says they want to hurt/kill themselves or others
- "illegal": Asking how to do something illegal (drugs, weapons, fraud, abuse)
- "safe": Everything else (sadness, breakup, depression talk, seeking help, concerns)

RISK ASSESSMENT: Evaluate the broader contextual risk level
- "clear": Safe, person seeking help or expressing normal concerns
- "elevated": Concerning patterns (repeated self-harm talk, persistent despair, escalation)
- "crisis": Imminent danger (active suicidal ideation, active harm planning)

Respond with ONLY valid JSON (no markdown, no extra text):
{
  "safety_alert": {
    "type": "crisis" | "illegal" | "safe",
    "severity": "immediate" | "high" | "warning",
    "title": "Brief title",
    "message": "What the concern is"
  },
  "risk_assessment": {
    "risk_level": "clear" | "elevated" | "crisis",
    "severity": 0-100,
    "assessment": "brief description",
    "recommendation": "proceed" | "caution" | "alert"
  }
}`

	req := &tools.LLMRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   message,
		MaxTokens:    400, // Slightly larger for both outputs
		Temperature:  0.2,
		Retries:      2,
	}

	resp, err := sra.llmClient.Call(context.Background(), req)
	if err != nil {
		log.Printf("[SafetyRiskAnalyzer] LLM call failed: %v, returning nil for both", err)
		return nil, nil, err
	}

	// Parse the combined response
	safetyAlert, riskAssessment := parseBetalysiResult(resp.Content)

	if safetyAlert != nil {
		log.Printf("[SafetyRiskAnalyzer] Safety: %s (%s)", safetyAlert.AlertType, safetyAlert.Severity)
	}
	if riskAssessment != nil {
		log.Printf("[SafetyRiskAnalyzer] Risk: %s (severity=%d)", riskAssessment.RiskLevel, riskAssessment.Severity)
	}

	return safetyAlert, riskAssessment, nil
}

// parseAnalysisResult parses the combined JSON response into both alert and assessment
func parseAnalysisResult(jsonStr string) (*safety.SafetyAlert, *models.RiskAssessment) {
	var result struct {
		SafetyAlert struct {
			Type     string `json:"type"`
			Severity string `json:"severity"`
			Title    string `json:"title"`
			Message  string `json:"message"`
		} `json:"safety_alert"`
		RiskAssessment struct {
			RiskLevel      string `json:"risk_level"`
			Severity       int    `json:"severity"`
			Assessment     string `json:"assessment"`
			Recommendation string `json:"recommendation"`
		} `json:"risk_assessment"`
	}

	err := json.Unmarshal([]byte(jsonStr), &result)
	if err != nil {
		log.Printf("[SafetyRiskAnalyzer] Failed to parse response: %v", err)
		return nil, nil
	}

	// Build SafetyAlert
	var safetyAlert *safety.SafetyAlert
	if result.SafetyAlert.Type != "" && result.SafetyAlert.Type != "safe" {
		safetyAlert = &safety.SafetyAlert{
			AlertType: safety.AlertType(result.SafetyAlert.Type),
			Severity:  safety.AlertSeverity(result.SafetyAlert.Severity),
			Title:     result.SafetyAlert.Title,
			Message:   result.SafetyAlert.Message,
		}
	}

	// Build RiskAssessment
	var riskAssessment *models.RiskAssessment
	if result.RiskAssessment.RiskLevel != "" && result.RiskAssessment.RiskLevel != "clear" {
		riskAssessment = &models.RiskAssessment{
			RiskLevel:      result.RiskAssessment.RiskLevel,
			Severity:       result.RiskAssessment.Severity,
			Pattern:        result.RiskAssessment.Assessment,
			Recommendation: result.RiskAssessment.Recommendation,
			Message:        result.RiskAssessment.Assessment,
		}
	}

	return safetyAlert, riskAssessment
}

// parseBetalysiResult is a wrapper to match the typo in the original code (kept for compatibility)
func parseBetalysiResult(jsonStr string) (*safety.SafetyAlert, *models.RiskAssessment) {
	return parseAnalysisResult(jsonStr)
}
