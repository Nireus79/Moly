package agents

import (
	"context"
	"errors"
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

// AssessRisk - Assess risk for a message
func (rm *riskMonitor) AssessRisk(userID string, message string) (*models.RiskAssessment, error) {
	log.Printf("[RiskMonitor] Assessing risk for user %s (message length=%d, LLM available=%v)",
		userID, len(message), rm.llmClient != nil)

	if userID == "" || message == "" {
		log.Printf("[RiskMonitor] ERROR: userID and message cannot be empty")
		return nil, errors.New("userID and message cannot be empty")
	}

	// If no LLM, use basic heuristics
	if rm.llmClient == nil {
		log.Printf("[RiskMonitor] Using heuristic-based risk assessment")
		assessment := rm.basicRiskAssessment(message)
		log.Printf("[RiskMonitor] Risk assessment complete: level=%s severity=%d", assessment.RiskLevel, assessment.Severity)
		return assessment, nil
	}

	// Use LLM for detailed analysis
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

	return assessment
}

// llmRiskAssessment - Use LLM for detailed risk analysis
func (rm *riskMonitor) llmRiskAssessment(message string) (*models.RiskAssessment, error) {
	req := &tools.LLMRequest{
		SystemPrompt: `You are a communication safety coach. Analyze the provided message for:
1. Crisis indicators (self-harm, suicide ideation)
2. Harsh or abusive language
3. Manipulation or coercion
4. Constitutional violations

Respond with JSON: {"riskLevel":"clear|elevated|crisis","severity":0-100,"message":"...","questions":["..."],"principles":["..."],"alternatives":["..."]}`,
		UserPrompt: "Analyze this message for risks: " + message,
		MaxTokens: 500,
		Temperature: 0.3,
		Retries: 1,
	}

	resp, err := rm.llmClient.Call(context.Background(), req)
	if err != nil {
		// Fall back to basic assessment on LLM error
		return rm.basicRiskAssessment(message), nil
	}

	assessment := &models.RiskAssessment{
		RiskLevel:             "clear",
		Severity:              0,
		EducationalQuestions:  []string{},
		Principles:            []models.CommunicationPrinciple{},
		Alternatives:          []string{},
		Recommendation:        "proceed",
		Message:               resp.Content,
	}

	// Parse LLM response (would need JSON parsing in production)
	// For now, check for keywords in response
	lower := strings.ToLower(resp.Content)
	if strings.Contains(lower, "crisis") {
		assessment.RiskLevel = "crisis"
		assessment.Severity = 100
		assessment.Recommendation = "alert"
	} else if strings.Contains(lower, "elevated") {
		assessment.RiskLevel = "elevated"
		assessment.Severity = 60
		assessment.Recommendation = "caution"
	}

	return assessment, nil
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
