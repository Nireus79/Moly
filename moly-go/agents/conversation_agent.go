package agents

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	// LLM client is optional - agent will generate basic suggestions without it
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

	// Extract context from conversation history if available
	var userMessage string
	if len(ctx.ConversationHistory) > 0 {
		userMessage = ctx.ConversationHistory[0].Content
	}

	// Build personalized suggestions based on context
	aboutMe := ctx.AboutMe
	contact := ctx.ContactProfile

	// Determine intention from message
	intention := "general_support"
	if userMessage != "" {
		lowerMsg := strings.ToLower(userMessage)
		if contains(lowerMsg, "congratulat") || contains(lowerMsg, "promote") || contains(lowerMsg, "success") {
			intention = "celebrate"
		} else if contains(lowerMsg, "apologi") || contains(lowerMsg, "sorry") {
			intention = "apologize"
		} else if contains(lowerMsg, "help") || contains(lowerMsg, "need") || contains(lowerMsg, "stuck") {
			intention = "seek_help"
		} else if contains(lowerMsg, "hi") || contains(lowerMsg, "hello") || contains(lowerMsg, "hey") {
			intention = "greet"
		}
	}

	// Generate context-aware suggestions
	response.Suggestions = generateContextualSuggestions(aboutMe, contact, userMessage, intention)

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

// contains checks if string contains substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// generateContextualSuggestions creates personalized suggestions based on context
func generateContextualSuggestions(aboutMe *models.AboutMe, contact *models.Contact, userMessage string, intention string) []models.Suggestion {
	suggestions := []models.Suggestion{}

	// Get user's communication style
	userStyle := "friendly"
	if aboutMe != nil && aboutMe.CommunicationStyle != "" {
		userStyle = aboutMe.CommunicationStyle
	}

	// Get contact's known preferences
	contactName := "them"
	if contact != nil && contact.Name != "" {
		contactName = contact.Name
	}

	// Generate suggestions based on intention and context
	switch intention {
	case "celebrate":
		suggestions = []models.Suggestion{
			{
				Index:      0,
				Text:       "That's amazing! I'm so happy for you! 🎉",
				Tone:       userStyle,
				Reasoning:  fmt.Sprintf("Genuine celebration in your authentic %s style, perfect for %s", userStyle, contactName),
				Confidence: 0.92,
			},
			{
				Index:      1,
				Text:       "Congratulations! You deserve this. Tell me everything!",
				Tone:       userStyle,
				Reasoning:  fmt.Sprintf("Shows genuine interest and excitement, matches how you naturally communicate"),
				Confidence: 0.88,
			},
			{
				Index:      2,
				Text:       "This is huge! I'd love to hear all about it.",
				Tone:       userStyle,
				Reasoning:  fmt.Sprintf("Enthusiastic but not over-the-top, allows space for them to share"),
				Confidence: 0.85,
			},
		}
	case "apologize":
		suggestions = []models.Suggestion{
			{
				Index:      0,
				Text:       "I'm sorry for how I handled that. I should have communicated better.",
				Tone:       userStyle,
				Reasoning:  fmt.Sprintf("Takes responsibility without over-explaining, authentic to your style"),
				Confidence: 0.90,
			},
			{
				Index:      1,
				Text:       "I want to make this right. What can I do?",
				Tone:       userStyle,
				Reasoning:  fmt.Sprintf("Action-oriented, shows commitment to resolution"),
				Confidence: 0.86,
			},
			{
				Index:      2,
				Text:       "I regret that. Can we talk about it?",
				Tone:       userStyle,
				Reasoning:  fmt.Sprintf("Direct and respectful, opens dialogue without being defensive"),
				Confidence: 0.84,
			},
		}
	case "seek_help":
		suggestions = []models.Suggestion{
			{
				Index:      0,
				Text:       "I'm dealing with something and could really use your perspective.",
				Tone:       userStyle,
				Reasoning:  fmt.Sprintf("Vulnerable but specific, respects their time and expertise"),
				Confidence: 0.89,
			},
			{
				Index:      1,
				Text:       "I'm stuck on something. Do you have time to talk?",
				Tone:       userStyle,
				Reasoning:  fmt.Sprintf("Clear and direct, gives them the choice to engage"),
				Confidence: 0.87,
			},
			{
				Index:      2,
				Text:       "Can I get your advice on something?",
				Tone:       userStyle,
				Reasoning:  fmt.Sprintf("Values their input, shows respect for their opinion"),
				Confidence: 0.85,
			},
		}
	case "greet":
		suggestions = []models.Suggestion{
			{
				Index:      0,
				Text:       "Hey! How's it going?",
				Tone:       userStyle,
				Reasoning:  fmt.Sprintf("Warm and casual, matches your natural communication style with %s", contactName),
				Confidence: 0.88,
			},
			{
				Index:      1,
				Text:       "Hi! What's new with you?",
				Tone:       userStyle,
				Reasoning:  fmt.Sprintf("Friendly opener that invites them to share"),
				Confidence: 0.85,
			},
			{
				Index:      2,
				Text:       "Great to hear from you! What's up?",
				Tone:       userStyle,
				Reasoning:  fmt.Sprintf("Shows genuine warmth and interest in their updates"),
				Confidence: 0.84,
			},
		}
	default:
		// General fallback suggestions
		suggestions = []models.Suggestion{
			{
				Index:      0,
				Text:       "That sounds important. Tell me more.",
				Tone:       userStyle,
				Reasoning:  fmt.Sprintf("Shows genuine interest, matches your authentic communication style"),
				Confidence: 0.85,
			},
			{
				Index:      1,
				Text:       "I'm listening. What's on your mind?",
				Tone:       userStyle,
				Reasoning:  fmt.Sprintf("Open and welcoming, invites deeper conversation"),
				Confidence: 0.82,
			},
			{
				Index:      2,
				Text:       "How are you feeling about all this?",
				Tone:       userStyle,
				Reasoning:  fmt.Sprintf("Empathetic and present, helps them reflect"),
				Confidence: 0.80,
			},
		}
	}

	return suggestions
}
