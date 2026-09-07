package agents

import (
	"context"
	"errors"
	"time"

	"moly/models"
	"moly/tools"
)

// conversationAgent - Implements the 5-phase conversation flow
type conversationAgent struct {
	llmClient            *tools.LLMClient
	suggestionGenerator  *tools.SuggestionGenerator
	questionGenerator    *tools.QuestionGenerator
	safetyChecker        *tools.SafetyChecker
	constitutionEvaluator *tools.ConstitutionEvaluator
	contextExtractor     *tools.ContextExtractor
}

// NewConversationAgent - Create new conversation agent
func NewConversationAgent(llm *tools.LLMClient) (models.ConversationAgent, error) {
	if llm == nil {
		return nil, errors.New("LLM client cannot be nil")
	}

	return &conversationAgent{
		llmClient:            llm,
		suggestionGenerator:  tools.NewSuggestionGenerator(llm),
		questionGenerator:    tools.NewQuestionGenerator(llm),
		safetyChecker:        tools.NewSafetyChecker(llm),
		constitutionEvaluator: tools.NewConstitutionEvaluator(llm),
		contextExtractor:     tools.NewContextExtractor(llm),
	}, nil
}

// Run - Execute the conversation flow and generate response
func (ca *conversationAgent) Run(ctx models.Context) (*models.ConversationResponse, error) {
	if ctx.AboutMe == nil {
		return nil, errors.New("context must include AboutMe")
	}

	startTime := time.Now()
	response := &models.ConversationResponse{
		Phase: "suggestions_ready",
	}

	// Phase 1: ANALYZE - Determine interaction type
	// Phase 2: CONTEXT - Gather relevant context
	// Phase 3: SAFETY - Check for safety concerns
	// Phase 4: RISK - Detect risk patterns
	// Phase 5: INTENTION - Understand user's goal

	// Phase 1: Quick safety check (fail-fast)
	// This is done synchronously for critical issues
	// TODO: Implement actual safety checking when context has userMessage

	// Phase 2-5: Main processing
	// TODO: Implement full orchestration

	// Graceful fallback: if something fails, still return a response
	if response.Suggestions == nil {
		response.Suggestions = []models.Suggestion{}
	}

	response.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())

	return response, nil
}

// runAnalyzePhase - Determine the type of interaction
func (ca *conversationAgent) runAnalyzePhase(userMessage string) (string, error) {
	if userMessage == "" {
		return "open", nil
	}

	// Classify the message type
	// Types: open (needs suggestions), question (user asking), clarify, check_understanding
	return "open", nil
}

// runContextPhase - Gather and organize relevant context
func (ca *conversationAgent) runContextPhase(ctx context.Context, userID string) (*models.Context, error) {
	// This would normally load from database
	// For now, return a minimal context
	return &models.Context{
		ContextQuality: "minimal",
		Gaps:           []string{"contact_profile", "conversation_history", "behavioral_profile"},
	}, nil
}

// runSafetyPhase - Check for crisis/illegal content
func (ca *conversationAgent) runSafetyPhase(ctx context.Context, message string) (*models.SafetyAlert, error) {
	if message == "" {
		return nil, nil
	}

	input := &tools.SafetyCheckInput{
		Message: message,
	}

	result, err := ca.safetyChecker.Check(ctx, input)
	if err != nil {
		// Graceful fallback: if safety check fails, log and continue
		return nil, nil
	}

	if result.AlertType != tools.SafetyAlertTypeNone {
		alert := &models.SafetyAlert{
			AlertType:      string(result.AlertType),
			Severity:       string(result.Severity),
			Title:          result.Title,
			Message:        result.Message,
			Indicators:     result.Indicators,
			Recommendations: result.Recommendations,
		}

		// Add resources if crisis
		if result.AlertType == tools.SafetyAlertTypeCrisis {
			for _, r := range result.Resources {
				alert.Resources = append(alert.Resources, models.CrisisResource{
					Name:        r.Name,
					Description: r.Description,
					Number:      r.Number,
					URL:         r.URL,
					Region:      r.Region,
				})
			}
		}

		return alert, nil
	}

	return nil, nil
}

// runRiskPhase - Detect concerning user patterns
func (ca *conversationAgent) runRiskPhase(ctx context.Context, message string) (*models.RiskWarning, error) {
	if message == "" {
		return nil, nil
	}

	// Use LLM to analyze for risk patterns
	// TODO: Implement risk pattern detection
	return nil, nil
}

// runIntentionPhase - Understand user's actual goal
func (ca *conversationAgent) runIntentionPhase(ctx context.Context, message string) (string, error) {
	if message == "" {
		return "", nil
	}

	// Extract user's intention from message
	// TODO: Implement intention extraction
	return "", nil
}

// runGeneratePhase - Create suggestions based on context
func (ca *conversationAgent) runGeneratePhase(ctx context.Context, input *tools.SuggestionGeneratorInput) ([]models.Suggestion, error) {
	output, err := ca.suggestionGenerator.Generate(ctx, input)
	if err != nil {
		// Graceful fallback: return empty suggestions
		return []models.Suggestion{}, nil
	}

	if output == nil {
		return []models.Suggestion{}, nil
	}

	suggestions := make([]models.Suggestion, len(output.Suggestions))
	for i, s := range output.Suggestions {
		suggestions[i] = models.Suggestion{
			Index:      s.Index,
			Text:       s.Text,
			Tone:       s.Tone,
			Reasoning:  s.Reasoning,
			Confidence: s.Confidence,
		}
	}

	return suggestions, nil
}

// runReflectPhase - Extract insights from conversation
func (ca *conversationAgent) runReflectPhase(ctx context.Context, message string) (*models.Reflection, error) {
	if message == "" {
		return nil, nil
	}

	input := &tools.ContextExtractorInput{
		Message: message,
	}

	output, err := ca.contextExtractor.Extract(ctx, input)
	if err != nil {
		// Graceful fallback: return nil, no error (optional phase)
		return nil, nil
	}

	if output == nil {
		return nil, nil
	}

	reflection := &models.Reflection{
		Characteristics:       output.NewCharacteristics,
		Interests:            output.NewInterests,
		CommunicationPreferences: output.UpdatedCommunicationPrefs,
		Intentions:           output.Intentions,
		UserQuotes:           output.UserQuotes,
		Status:               "pending_approval",
	}

	return reflection, nil
}
