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

// conversationAgent - Implements the 5-phase conversation flow
type conversationAgent struct {
	llmClient             tools.LLMProvider
	suggestionGenerator   *tools.SuggestionGenerator
	questionGenerator     *tools.QuestionGenerator
	safetyChecker         *tools.SafetyChecker
	constitutionEvaluator *tools.ConstitutionEvaluator
	contextExtractor      *tools.ContextExtractor
}

// NewConversationAgent - Create new conversation agent
func NewConversationAgent(llm tools.LLMProvider) (models.ConversationAgent, error) {
	// LLM client is optional - agent will generate basic suggestions without it
	return &conversationAgent{
		llmClient:             llm,
		suggestionGenerator:   tools.NewSuggestionGenerator(llm),
		questionGenerator:     tools.NewQuestionGenerator(llm),
		safetyChecker:         tools.NewSafetyChecker(llm),
		constitutionEvaluator: tools.NewConstitutionEvaluator(llm),
		contextExtractor:      tools.NewContextExtractor(llm),
	}, nil
}

// Run - Execute the conversation flow and generate response
func (ca *conversationAgent) Run(ctx models.Context) (*models.ConversationResponse, error) {
	log.Printf("[ConversationAgent] Starting conversation flow with context level: %s", ctx.ContextQuality)

	if ctx.AboutMe == nil {
		log.Printf("[ConversationAgent] ERROR: context must include AboutMe")
		return nil, errors.New("context must include AboutMe")
	}

	startTime := time.Now()
	response := &models.ConversationResponse{}

	// Extract context from conversation history if available
	var userMessage string
	if len(ctx.ConversationHistory) > 0 {
		userMessage = ctx.ConversationHistory[0].Content
	}
	log.Printf("[ConversationAgent] User message: %.80s...", userMessage)

	// Phase 1: ANALYZE - Check what context we have
	aboutMe := ctx.AboutMe
	contact := ctx.ContactProfile
	hasAboutMe := aboutMe != nil && (aboutMe.CommunicationStyle != "" || len(aboutMe.Values) > 0)
	hasContact := contact != nil && contact.Name != "" && contact.Name != "Contact"
	hasIntention := false

	log.Printf("[ConversationAgent] Context analysis: hasAboutMe=%v hasContact=%v", hasAboutMe, hasContact)

	// Detect intention from message
	intention := "general_support"
	if userMessage != "" {
		lowerMsg := strings.ToLower(userMessage)
		if contains(lowerMsg, "congratulat") || contains(lowerMsg, "promote") || contains(lowerMsg, "success") {
			intention = "celebrate"
			hasIntention = true
		} else if contains(lowerMsg, "apologi") || contains(lowerMsg, "sorry") {
			intention = "apologize"
			hasIntention = true
		} else if contains(lowerMsg, "help") || contains(lowerMsg, "need") || contains(lowerMsg, "stuck") {
			intention = "seek_help"
			hasIntention = true
		} else if contains(lowerMsg, "hi") || contains(lowerMsg, "hello") || contains(lowerMsg, "hey") {
			intention = "greet"
			hasIntention = true
		}
	}
	log.Printf("[ConversationAgent] Detected intention: %s (hasIntention=%v)", intention, hasIntention)

	// Phase 2: DECIDE - Gathering context vs. suggesting
	// Require: AboutMe, Contact, and Intention for good suggestions
	missingContext := !hasAboutMe || !hasContact || !hasIntention

	if missingContext {
		// Phase 3a: EXECUTE - Ask Socratic questions to gather context
		log.Printf("[ConversationAgent] Missing context - entering context gathering phase (missingContext=%v)", missingContext)
		response.Phase = "context_gathering"
		response.Questions = generateContextGatheringQuestions(hasAboutMe, hasContact, hasIntention, userMessage)
		log.Printf("[ConversationAgent] Generated %d context gathering questions", len(response.Questions))
		response.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())
		return response, nil
	}

	// Phase 2b: SAFETY CHECK - Detect risks before suggesting
	log.Printf("[ConversationAgent] Running safety check on user message")
	safetyAlert, err := ca.runSafetyPhase(context.Background(), userMessage)
	if err != nil {
		log.Printf("[ConversationAgent] Safety check error (non-fatal): %v", err)
	}

	if safetyAlert != nil {
		// Safety issue detected - return alert instead of suggestions
		log.Printf("[ConversationAgent] Safety alert: %s (severity: %s)", safetyAlert.AlertType, safetyAlert.Severity)
		response.Phase = "safety_alert"
		response.SafetyAlert = safetyAlert
		response.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())
		return response, nil
	}

	// Phase 3b: EXECUTE - Generate personalized suggestions (we have complete context and passed safety)
	log.Printf("[ConversationAgent] All context available and safety checks passed - generating suggestions")
	response.Phase = "suggestions_ready"

	// Try to use LLM for generation if available, fall back to hardcoded if not
	if ca.llmClient != nil {
		log.Printf("[ConversationAgent] Using LLM for suggestion generation")
		llmSuggestions := ca.generateLLMSuggestions(ctx, userMessage, intention)
		if len(llmSuggestions) > 0 {
			log.Printf("[ConversationAgent] LLM generated %d suggestions", len(llmSuggestions))
			response.Suggestions = llmSuggestions
		} else {
			// Fallback to context-aware suggestions
			log.Printf("[ConversationAgent] LLM returned no suggestions, using contextual fallback")
			response.Suggestions = generateContextualSuggestions(aboutMe, contact, userMessage, intention)
		}
	} else {
		// No LLM available, use context-aware suggestions
		response.Suggestions = generateContextualSuggestions(aboutMe, contact, userMessage, intention)
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
			AlertType:       string(result.AlertType),
			Severity:        string(result.Severity),
			Title:           result.Title,
			Message:         result.Message,
			Indicators:      result.Indicators,
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

// runRiskPhase - Detect concerning user patterns using LLM
func (ca *conversationAgent) runRiskPhase(ctx context.Context, message string) (*models.RiskWarning, error) {
	if message == "" {
		return nil, nil
	}

	if ca.llmClient == nil {
		return nil, nil // No LLM available, skip risk checking
	}

	// Use LLM to analyze for risk patterns
	prompt := `Analyze this message for concerning communication patterns.
Look for: threats, self-harm, emotional abuse language, manipulation, or escalation.
Be conservative - only flag if clearly concerning.

Message: "%s"

Respond with JSON only (no explanation):
{
  "has_risk": boolean,
  "risk_level": "low|medium|high",
  "pattern": "string or null",
  "reasoning": "brief explanation"
}

If no risk, respond: {"has_risk": false, "risk_level": "low", "pattern": null, "reasoning": "safe"}`

	req := &tools.LLMRequest{
		SystemPrompt: "You are a communication safety analyzer. Be concise and conservative.",
		UserPrompt:   fmt.Sprintf(prompt, message),
		Temperature:  0.2, // Low temperature for consistency
		MaxTokens:    200,
	}

	resp, err := ca.llmClient.Call(ctx, req)
	if err != nil {
		log.Printf("[ConversationAgent] ERROR: Risk analysis LLM call failed: %v", err)
		return nil, nil // Graceful fallback on error
	}

	// Parse response
	var riskData struct {
		HasRisk   bool   `json:"has_risk"`
		RiskLevel string `json:"risk_level"`
		Pattern   string `json:"pattern"`
		Reasoning string `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(resp.Content), &riskData); err != nil {
		log.Printf("[ConversationAgent] ERROR: Failed to parse risk response: %v (content: %s)", err, resp.Content)
		return nil, nil
	}

	// Log analysis result
	if !riskData.HasRisk {
		log.Printf("[ConversationAgent] SAFETY_CHECK: No risk detected (pattern: safe)")
		return nil, nil
	}

	// Risk detected - log and escalate
	log.Printf("[ConversationAgent] SAFETY_ALERT: Risk detected (level=%s, pattern=%s)", riskData.RiskLevel, riskData.Pattern)

	// Map risk level to severity score
	severityMap := map[string]int{
		"low":    3,
		"medium": 6,
		"high":   9,
	}
	severity := severityMap[riskData.RiskLevel]
	if severity == 0 {
		severity = 5 // Default to medium
	}

	// Return risk warning
	warning := &models.RiskWarning{
		RiskLevel:      riskData.RiskLevel,
		Pattern:        riskData.Pattern,
		Severity:       severity,
		Message:        riskData.Reasoning,
		Recommendation: "educate_first",
	}
	log.Printf("[ConversationAgent] RISK_WARNING: Severity=%d, Pattern='%s', Reason='%s'",
		severity, riskData.Pattern, riskData.Reasoning)

	log.Printf("[ConversationAgent] Risk detected: %s (severity: %d)", riskData.Pattern, severity)
	return warning, nil
}

// runIntentionPhase - Understand user's actual communication goal
func (ca *conversationAgent) runIntentionPhase(ctx context.Context, message string) (string, error) {
	if message == "" {
		return "", nil
	}

	if ca.llmClient == nil {
		return "", nil // No LLM available, skip intention detection
	}

	// Extract intention using LLM
	prompt := `Analyze this message and identify the user's primary communication intention.

Possible intentions:
- celebrate: sharing good news or excitement
- apologize: expressing regret or making amends
- seek_help: asking for advice or support
- clarify: wanting to understand something better
- inform: sharing information
- request: asking for something to be done
- express_feeling: sharing emotions or concerns
- set_boundary: establishing limits
- resolve_conflict: trying to fix a disagreement
- show_appreciation: expressing gratitude or praise

Message: "%s"

Respond with ONLY the intention word (lowercase), nothing else. Must be one of the listed intentions above.`

	req := &tools.LLMRequest{
		SystemPrompt: "You are a communication analyzer. Respond with ONLY the intention word.",
		UserPrompt:   fmt.Sprintf(prompt, message),
		Temperature:  0.2,
		MaxTokens:    20,
	}

	resp, err := ca.llmClient.Call(ctx, req)
	if err != nil {
		log.Printf("[ConversationAgent] ERROR: Intention extraction LLM call failed: %v", err)
		return "", nil
	}

	intention := strings.TrimSpace(strings.ToLower(resp.Content))

	// Validate intention is one of the valid options
	validIntentions := map[string]bool{
		"celebrate":         true,
		"apologize":         true,
		"seek_help":         true,
		"clarify":           true,
		"inform":            true,
		"request":           true,
		"express_feeling":   true,
		"set_boundary":      true,
		"resolve_conflict":  true,
		"show_appreciation": true,
	}

	if !validIntentions[intention] {
		log.Printf("[ConversationAgent] WARN: Invalid intention '%s' from LLM, defaulting to 'inform'", intention)
		intention = "inform"
	}

	log.Printf("[ConversationAgent] INTENTION_DETECTED: %s (valid=%v)", intention, validIntentions[intention])
	return intention, nil
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
		Characteristics:          output.NewCharacteristics,
		Interests:                output.NewInterests,
		CommunicationPreferences: output.UpdatedCommunicationPrefs,
		Intentions:               output.Intentions,
		UserQuotes:               output.UserQuotes,
		Status:                   "pending_approval",
	}

	return reflection, nil
}

// generateContextGatheringQuestions - Generate Socratic questions to gather missing context
// Returns ONE focused question at a time for conversational flow
func generateContextGatheringQuestions(hasAboutMe, hasContact, hasIntention bool, userMessage string) []string {
	// Gather context in progressive order: AboutMe → Contact → Intention

	if !hasAboutMe {
		// Start by understanding the user
		return []string{
			"I'd love to help you craft a message. Tell me about yourself - what's your communication style like? Are you more formal, casual, playful, or a mix?",
		}
	}

	if !hasContact {
		// Then understand who they're talking to
		return []string{
			"Now, who are you wanting to message? Tell me their name and what your relationship is like.",
		}
	}

	if !hasIntention {
		// Finally understand what they want to achieve
		return []string{
			"What's your intention with this message? Are you celebrating something, apologizing, asking for help, or starting a conversation?",
		}
	}

	// Fallback - shouldn't reach here if logic is correct
	return []string{"Tell me more about what you're trying to communicate."}
}

// contains checks if string contains substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// generateLLMSuggestions - Generate suggestions using LLM
func (ca *conversationAgent) generateLLMSuggestions(ctx models.Context, userMessage string, intention string) []models.Suggestion {
	if ca.llmClient == nil || ca.suggestionGenerator == nil {
		return []models.Suggestion{}
	}

	aboutMe := ctx.AboutMe
	contact := ctx.ContactProfile

	// Build input for suggestion generator
	input := &tools.SuggestionGeneratorInput{
		UserMessage:            userMessage,
		UserCommunicationStyle: "friendly",
		UserValues:             []string{},
		ContactCharacteristics: []string{},
		ContactInterests:       []string{},
		ContactRelationship:    "friend",
		UserIntention:          intention,
		Mode:                   "direct",
		Tone:                   "friendly",
	}

	if aboutMe != nil {
		input.UserCommunicationStyle = aboutMe.CommunicationStyle
		input.UserValues = aboutMe.Values
		input.Tone = aboutMe.PreferredTone
	}

	if contact != nil {
		input.ContactRelationship = contact.Relationship
		input.ContactCharacteristics = contact.Characteristics
		input.ContactInterests = contact.Interests
	}

	// Call suggestion generator
	genCtx := context.Background()
	output, err := ca.suggestionGenerator.Generate(genCtx, input)
	if err != nil {
		return []models.Suggestion{}
	}

	if output == nil || len(output.Suggestions) == 0 {
		return []models.Suggestion{}
	}

	// Convert to models.Suggestion
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

	return suggestions
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
