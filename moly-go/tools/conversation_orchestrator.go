package tools

import (
	"context"
	"fmt"

	"moly/models"
)

// ConversationOrchestrator - Orchestrates complete conversation flows
type ConversationOrchestrator struct {
	responseParser      *ResponseParser
	intentionDetector   *IntentionDetector
	behaviorAnalyzer    *BehaviorAnalyzer
	userProfileBuilder  *UserProfileBuilder
	messageFormatter    *MessageFormatter
	validators          *Validators
	llmClient           LLMProvider
}

// NewConversationOrchestrator - Create new orchestrator
func NewConversationOrchestrator(llm LLMProvider) *ConversationOrchestrator {
	return &ConversationOrchestrator{
		responseParser:     NewResponseParser(llm),
		intentionDetector:  NewIntentionDetector(llm),
		behaviorAnalyzer:   NewBehaviorAnalyzer(),
		userProfileBuilder: NewUserProfileBuilder(llm),
		messageFormatter:   NewMessageFormatter(),
		validators:         NewValidators(),
		llmClient:          llm,
	}
}

// ProcessUserInput - Process complete user input
func (co *ConversationOrchestrator) ProcessUserInput(
	ctx context.Context,
	userID string,
	userMessage string,
) (*ConversationProcessResult, error) {

	// Validate input
	if valid, errMsg := co.validators.ValidateUserID(userID); !valid {
		return nil, fmt.Errorf("invalid user ID: %s", errMsg)
	}

	if valid, errMsg := co.validators.ValidateMessage(userMessage); !valid {
		return nil, fmt.Errorf("invalid message: %s", errMsg)
	}

	result := &ConversationProcessResult{
		UserID:      userID,
		RawMessage:  userMessage,
		ProcessedAt: getCurrentTimestamp(),
	}

	// Step 1: Detect intention
	intentionInput := &IntentionDetectorInput{
		Message:              userMessage,
		UserCommunicationStyle: "unknown",
	}

	intentionOutput, err := co.intentionDetector.Detect(ctx, intentionInput)
	if err == nil && intentionOutput != nil {
		result.DetectedIntention = string(intentionOutput.PrimaryIntention)
		result.IntentionConfidence = intentionOutput.Confidence
		result.EmotionalTone = intentionOutput.Tone
	}

	// Step 2: Check for profanity/safety
	if co.validators.ContainsProfanity(userMessage) {
		result.Flags = append(result.Flags, "contains_profanity")
	}

	if !co.validators.IsSafeText(userMessage) {
		result.Flags = append(result.Flags, "potentially_unsafe")
	}

	// Step 3: Extract metadata
	result.Mentions = co.validators.ExtractMentions(userMessage)
	result.Hashtags = co.validators.ExtractHashtags(userMessage)
	result.URLs = co.validators.ExtractURLs(userMessage)

	// Step 4: Normalize for storage
	result.NormalizedMessage = co.validators.NormalizeText(userMessage)

	// Step 5: Parse response if needed (for user responses to questions)
	parseInput := &ResponseParserInput{
		UserMessage: userMessage,
		Context:     "user_response",
		UserID:      userID,
	}

	parseOutput, err := co.responseParser.Parse(ctx, parseInput)
	if err == nil && parseOutput != nil && parseOutput.ParsedSuccessfully {
		result.ExtractedAboutMe = parseOutput.ExtractedAboutMe
		result.ExtractedContact = parseOutput.ExtractedContact
		result.ExtractionConfidence = parseOutput.Confidence
	}

	return result, nil
}

// GenerateContextualResponse - Generate response based on context
func (co *ConversationOrchestrator) GenerateContextualResponse(
	userInput *ConversationProcessResult,
	userProfile *models.UserBehavioralProfile,
	aboutMe *models.AboutMe,
	contacts []models.Contact,
) *ConversationContextualResponse {

	response := &ConversationContextualResponse{
		UserID:      userInput.UserID,
		ProcessedAt: getCurrentTimestamp(),
	}

	// Determine context completeness
	if userProfile != nil && aboutMe != nil && len(contacts) > 0 {
		response.ContextLevel = "comprehensive"
		response.Recommendations = []string{
			"Ready for personalized suggestions",
			"Using full context for response generation",
		}
	} else if aboutMe != nil || len(contacts) > 0 {
		response.ContextLevel = "partial"
		response.Recommendations = []string{
			"Gathering additional context",
			"Providing based on available information",
		}
	} else {
		response.ContextLevel = "minimal"
		response.Recommendations = []string{
			"Starting context gathering",
			"Ask questions to build profile",
		}
	}

	// Add detected intention
	if userInput.DetectedIntention != "" {
		response.DetectedIntention = userInput.DetectedIntention
	}

	// Add emotional context
	if userInput.EmotionalTone != "" {
		response.EmotionalContext = userInput.EmotionalTone
	}

	return response
}

// ValidateAndEnrichContext - Validate and enrich user context
func (co *ConversationOrchestrator) ValidateAndEnrichContext(
	ctx context.Context,
	userContext *models.Context,
) (*EnrichedContext, error) {

	enriched := &EnrichedContext{
		OriginalContext: userContext,
		Validations:     make(map[string]bool),
	}

	if userContext == nil {
		enriched.IsValid = false
		enriched.Validations["nil_context"] = false
		return enriched, nil
	}

	// Validate AboutMe
	if userContext.AboutMe != nil && userContext.AboutMe.CommunicationStyle != "" {
		enriched.Validations["about_me"] = true
	} else {
		enriched.Validations["about_me"] = false
	}

	// Validate Contact
	if userContext.ContactProfile != nil && userContext.ContactProfile.Name != "" {
		enriched.Validations["contact"] = true
	} else {
		enriched.Validations["contact"] = false
	}

	// Validate History
	if len(userContext.ConversationHistory) > 0 {
		enriched.Validations["history"] = true
	} else {
		enriched.Validations["history"] = false
	}

	// Calculate overall validity
	validCount := 0
	for _, valid := range enriched.Validations {
		if valid {
			validCount++
		}
	}

	enriched.IsValid = validCount >= 2
	enriched.ValidityScore = float64(validCount) / float64(len(enriched.Validations))

	return enriched, nil
}

// GenerateFollowUpQuestions - Generate smart follow-up questions
func (co *ConversationOrchestrator) GenerateFollowUpQuestions(
	ctx context.Context,
	previousMessage string,
	detectedIntention string,
) ([]string, error) {

	if previousMessage == "" {
		return []string{}, fmt.Errorf("previous message required")
	}

	// Use LLM to generate contextual follow-ups
	if co.llmClient != nil {
		req := &LLMRequest{
			SystemPrompt: `You are a conversation coach. Generate 2-3 natural follow-up questions based on the user's message and intention.
The questions should help you understand:
1. Their emotional state
2. The relationship context
3. Their specific goals

Be conversational, not robotic.`,
			UserPrompt: fmt.Sprintf(`Message: "%s"
Intention: %s

Generate 2-3 follow-up questions (one per line).`, previousMessage, detectedIntention),
			MaxTokens:   300,
			Temperature: 0.7,
			Retries:     1,
		}

		resp, err := co.llmClient.Call(ctx, req)
		if err == nil && resp.Content != "" {
			return parseQuestions(resp.Content), nil
		}
	}

	// Fallback to heuristic questions
	return co.generateDefaultFollowUpQuestions(detectedIntention), nil
}

// generateDefaultFollowUpQuestions - Generate default follow-up questions
func (co *ConversationOrchestrator) generateDefaultFollowUpQuestions(intention string) []string {
	switch intention {
	case "celebrate":
		return []string{
			"How long have you been working toward this?",
			"Who else should celebrate with you?",
		}
	case "apologize":
		return []string{
			"How long have you been thinking about this?",
			"What's the most important thing they should know?",
		}
	case "seek_help":
		return []string{
			"What have you already tried?",
			"What kind of advice or help would be most useful?",
		}
	default:
		return []string{
			"Tell me more about what you're feeling.",
			"What matters most in this situation?",
		}
	}
}

// StructureConversationResponse - Convert all processing into final response
func (co *ConversationOrchestrator) StructureConversationResponse(
	processed *ConversationProcessResult,
	contextual *ConversationContextualResponse,
	suggestions []models.Suggestion,
	questions []string,
) map[string]interface{} {

	return map[string]interface{}{
		"status": "success",
		"meta": map[string]interface{}{
			"processed_at":  processed.ProcessedAt,
			"intention":     processed.DetectedIntention,
			"confidence":    processed.IntentionConfidence,
			"emotional_tone": processed.EmotionalTone,
		},
		"context": map[string]interface{}{
			"level":            contextual.ContextLevel,
			"quality_score":    contextual.QualityScore,
			"recommendations":  contextual.Recommendations,
		},
		"response": map[string]interface{}{
			"suggestions": suggestions,
			"questions":   questions,
			"phase":       determinePhase(questions, suggestions),
		},
		"metadata": map[string]interface{}{
			"mentions":  processed.Mentions,
			"hashtags":  processed.Hashtags,
			"urls":      processed.URLs,
			"flags":     processed.Flags,
		},
	}
}

// determinePhase - Determine conversation phase
func determinePhase(questions []string, suggestions []models.Suggestion) string {
	if len(questions) > 0 {
		return "context_gathering"
	}
	if len(suggestions) > 0 {
		return "suggestions_ready"
	}
	return "analyzing"
}

// getCurrentTimestamp - Get current timestamp
func getCurrentTimestamp() int64 {
	// Import time package in actual implementation
	return 0 // Placeholder
}

// ConversationProcessResult - Result of processing user input
type ConversationProcessResult struct {
	UserID                  string
	RawMessage              string
	NormalizedMessage       string
	DetectedIntention       string
	IntentionConfidence     float64
	EmotionalTone           string
	ExtractedAboutMe        *models.AboutMe
	ExtractedContact        *models.Contact
	ExtractionConfidence    float64
	Mentions                []string
	Hashtags                []string
	URLs                    []string
	Flags                   []string
	ProcessedAt             int64
}

// ConversationContextualResponse - Contextual response details
type ConversationContextualResponse struct {
	UserID             string
	ContextLevel       string // "minimal", "partial", "comprehensive"
	QualityScore       float64
	Recommendations    []string
	DetectedIntention  string
	EmotionalContext   string
	ProcessedAt        int64
}

// EnrichedContext - Enriched context with validation
type EnrichedContext struct {
	OriginalContext *models.Context
	IsValid         bool
	ValidityScore   float64
	Validations     map[string]bool
}
