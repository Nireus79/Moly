package agents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"moly/models"
	"moly/tools"
)

// riskMonitor - Detects concerning patterns with educational approach
type riskMonitor struct {
	userID    string
	llmClient tools.LLMProvider
}

// NewRiskMonitor - Create new risk monitor (no LLM)
func NewRiskMonitor(userID string) (models.RiskMonitoringAgent, error) {
	if userID == "" {
		return nil, errors.New("userID cannot be empty")
	}

	return &riskMonitor{
		userID:    userID,
		llmClient: nil,
	}, nil
}

// NewRiskMonitorWithLLM - Create new risk monitor with LLM capability
func NewRiskMonitorWithLLM(userID string, llm tools.LLMProvider) (models.RiskMonitoringAgent, error) {
	log.Printf("[RiskMonitor] Initializing for user %s with LLM", userID)

	if userID == "" {
		log.Printf("[RiskMonitor] ERROR: userID cannot be empty")
		return nil, errors.New("userID cannot be empty")
	}

	return &riskMonitor{
		userID:    userID,
		llmClient: llm,
	}, nil
}

// AssessRisk - LLM-driven risk assessment (no keyword fallback)
func (rm *riskMonitor) AssessRisk(userID string, message string) (*models.RiskAssessment, error) {
	log.Printf("[RiskMonitor] Assessing risk for user %s (message length=%d, LLM available=%v)",
		userID, len(message), rm.llmClient != nil)

	if userID == "" || message == "" {
		log.Printf("[RiskMonitor] ERROR: userID and message cannot be empty")
		return nil, errors.New("userID and message cannot be empty")
	}

	// LLM-only assessment
	if rm.llmClient == nil {
		log.Printf("[RiskMonitor] ERROR: No LLM available, cannot assess risk")
		return nil, errors.New("LLM required for risk assessment")
	}

	log.Printf("[RiskMonitor] Using LLM-based risk assessment")
	return rm.llmRiskAssessment(message)
}

// basicRiskAssessment - Basic heuristic-based risk assessment
func (rm *riskMonitor) basicRiskAssessment(message string) *models.RiskAssessment {
	lower := strings.ToLower(message)

	assessment := &models.RiskAssessment{
		RiskLevel:            "clear",
		Severity:             0,
		EducationalQuestions: []string{},
		Principles:           []models.CommunicationPrinciple{},
		Alternatives:         []string{},
		Recommendation:       "proceed",
		Message:              "Safe to send",
	}

	// Check for crisis indicators
	crisisKeywords := []string{"suicide", "harm", "die", "kill", "death", "overdose", "end it"}
	for _, keyword := range crisisKeywords {
		if strings.Contains(lower, keyword) {
			assessment.RiskLevel = "crisis"
			assessment.Severity = 100
			assessment.Recommendation = "alert"
			assessment.Message = "Crisis indicator detected. Please reach out to support services."
			return assessment
		}
	}

	// Check for harsh language
	harshKeywords := []string{"hate", "stupid", "idiot", "loser", "worthless", "pathetic"}
	for _, keyword := range harshKeywords {
		if strings.Contains(lower, keyword) {
			assessment.RiskLevel = "elevated"
			assessment.Severity = 60
			assessment.Recommendation = "caution"
			assessment.Message = "Message contains harsh language. Consider a gentler approach."
			assessment.EducationalQuestions = []string{
				"How might this language make the other person feel?",
				"Is there a kinder way to express this concern?",
			}
			return assessment
		}
	}

	// Check for workplace relationship concerns
	workplaceKeywords := []string{"boss", "manager", "coworker", "colleague", "supervisor", "work", "job", "fired", "fired", "ignored", "dismissive", "cold", "distant"}
	workplaceCount := 0
	for _, keyword := range workplaceKeywords {
		if strings.Contains(lower, keyword) {
			workplaceCount++
		}
	}
	if workplaceCount >= 2 {
		assessment.RiskLevel = "elevated"
		assessment.Severity = 40
		assessment.Recommendation = "explore"
		assessment.Message = "This involves a workplace relationship concern worth exploring thoughtfully."
		assessment.EducationalQuestions = []string{
			"What specific behaviors concern you most?",
			"How have you communicated your concerns so far?",
			"What would a better relationship look like?",
		}
		return assessment
	}

	// Check for interpersonal conflict with emotional distress
	conflictKeywords := []string{"upset", "worried", "anxious", "stressed", "frustrated", "concerned", "struggling", "difficult"}
	relationshipKeywords := []string{"friend", "family", "partner", "relationship", "person", "people"}
	conflictCount := 0
	relationshipCount := 0
	for _, keyword := range conflictKeywords {
		if strings.Contains(lower, keyword) {
			conflictCount++
		}
	}
	for _, keyword := range relationshipKeywords {
		if strings.Contains(lower, keyword) {
			relationshipCount++
		}
	}
	if conflictCount >= 1 && relationshipCount >= 1 {
		assessment.RiskLevel = "elevated"
		assessment.Severity = 35
		assessment.Recommendation = "support"
		assessment.Message = "Relationship concerns with emotional weight deserve careful attention."
		assessment.EducationalQuestions = []string{
			"What's most important to you in this relationship?",
			"What would help you feel more secure or valued?",
		}
		return assessment
	}

	return assessment
}

// llmRiskAssessment - LLM-driven risk analysis (no static keywords or fallback)
func (rm *riskMonitor) llmRiskAssessment(message string) (*models.RiskAssessment, error) {
	req := &tools.LLMRequest{
		SystemPrompt: `You are a safety expert. Analyze this message for actual risks based on its content and intent.

Assess:
1. Is there EXPLICIT self-harm or suicide intent? (crisis)
2. Is there EXPLICIT harmful/abusive content? (elevated)
3. Is this someone seeking help or expressing concerns? (clear)

Be precise: emotional expression, seeking advice, and discussing difficult topics are NOT risks.

Respond with ONLY this JSON format:
{
  "risk_level": "clear" | "elevated" | "crisis",
  "severity": 0-100,
  "assessment": "brief description",
  "recommendation": "proceed" | "caution" | "alert"
}`,
		UserPrompt:  message,
		MaxTokens:   300,
		Temperature: 0.2,
		Retries:     2,
	}

	resp, err := rm.llmClient.Call(context.Background(), req)
	if err != nil {
		log.Printf("[RiskMonitor] LLM call failed: %v", err)
		return nil, fmt.Errorf("LLM risk assessment failed: %w", err)
	}

	assessment := parseRiskAssessmentResponse(resp.Content)
	log.Printf("[RiskMonitor] Risk assessment: level=%s severity=%d recommendation=%s",
		assessment.RiskLevel, assessment.Severity, assessment.Recommendation)

	return assessment, nil
}

// parseRiskAssessmentResponse - Parse JSON response from LLM
func parseRiskAssessmentResponse(responseText string) *models.RiskAssessment {
	assessment := &models.RiskAssessment{
		RiskLevel:            "clear",
		Severity:             0,
		EducationalQuestions: []string{},
		Principles:           []models.CommunicationPrinciple{},
		Alternatives:         []string{},
		Recommendation:       "proceed",
		Message:              responseText,
	}

	type LLMRiskResponse struct {
		RiskLevel      string `json:"risk_level"`
		Severity       int    `json:"severity"`
		Assessment     string `json:"assessment"`
		Recommendation string `json:"recommendation"`
	}

	var llmResp LLMRiskResponse
	if err := json.Unmarshal([]byte(responseText), &llmResp); err != nil {
		log.Printf("[RiskMonitor] Failed to parse risk response: %v", err)
		return assessment
	}

	assessment.RiskLevel = llmResp.RiskLevel
	assessment.Severity = llmResp.Severity
	assessment.Message = llmResp.Assessment
	assessment.Recommendation = llmResp.Recommendation

	return assessment
}

// DetectPatterns - Detect concerning patterns in user's behavior over time
func (rm *riskMonitor) DetectPatterns(userID string) (*models.UserRiskProfile, error) {
	if userID == "" {
		return nil, errors.New("userID cannot be empty")
	}

	profile := &models.UserRiskProfile{
		UserID:          userID,
		RiskPatterns:    []models.RiskPattern{},
		HighestRisk:     "none",
		InterventionLog: []models.Intervention{},
		UpdatedAt:       time.Now().Unix(),
	}

	// In Phase 1.1, patterns come from recent assessments
	// In Phase 1.2+, would query database for interaction history
	return profile, nil
}

// GenerateEducationalResponse - Generate educational response to risk pattern
func (rm *riskMonitor) GenerateEducationalResponse(risk models.RiskAssessment) ([]string, error) {
	if risk.Message == "" {
		return nil, errors.New("risk assessment message cannot be empty")
	}

	responses := []string{
		"I notice something in what you're planning. Can we explore this together?",
	}

	// Add questions from risk assessment
	if len(risk.EducationalQuestions) > 0 {
		responses = append(responses, risk.EducationalQuestions...)
	}

	return responses, nil
}

// TrackPattern - Record a detected pattern
func (rm *riskMonitor) TrackPattern(userID string, pattern *models.RiskPattern) error {
	if userID == "" {
		return errors.New("userID cannot be empty")
	}

	if pattern == nil {
		return errors.New("pattern cannot be nil")
	}

	// Pattern tracking would save to database in Phase 1.2+
	// For now, just validate
	return nil
}

// GetUserRiskProfile - Retrieve user's risk profile
func (rm *riskMonitor) GetUserRiskProfile(userID string) (*models.UserRiskProfile, error) {
	if userID == "" {
		return nil, errors.New("userID cannot be empty")
	}

	profile := &models.UserRiskProfile{
		UserID:          userID,
		RiskPatterns:    []models.RiskPattern{},
		HighestRisk:     "none",
		InterventionLog: []models.Intervention{},
		UpdatedAt:       time.Now().Unix(),
	}

	// Would load from database in Phase 1.2+
	return profile, nil
}
