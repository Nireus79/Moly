package tools

import (
	"context"
	"fmt"
	"log"
	"strings"

	"moly/models"
)

// ResponseGenerator creates natural, contextual LLM-generated responses
type ResponseGenerator struct {
	llmClient LLMProvider
}

// NewResponseGenerator creates a new generator
func NewResponseGenerator(llm LLMProvider) *ResponseGenerator {
	return &ResponseGenerator{llmClient: llm}
}

// callLLM is a helper to make LLM calls with consistent formatting
func (rg *ResponseGenerator) callLLM(systemPrompt, userPrompt string) (string, error) {
	if rg.llmClient == nil {
		return "", fmt.Errorf("no LLM client available")
	}

	req := &LLMRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Temperature:  0.7,
	}

	resp, err := rg.llmClient.Call(context.Background(), req)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(resp.Content), nil
}

// GenerateEmptyMessageResponse generates a response for empty user input
func (rg *ResponseGenerator) GenerateEmptyMessageResponse(ctx models.Context) string {
	if rg.llmClient == nil {
		return "What's on your mind?"
	}

	systemPrompt := "You are Moly, a communication coach. Generate a brief, warm prompt asking the user to share. One sentence only. Be conversational, not robotic."
	userPrompt := buildEmptyMessagePrompt(ctx)

	response, err := rg.callLLM(systemPrompt, userPrompt)
	if err != nil {
		log.Printf("[ResponseGenerator] Warning: Failed to generate empty message response: %v", err)
		return "What's on your mind?"
	}

	return response
}

// GenerateNeedsClarificationResponse generates a response when context is missing
func (rg *ResponseGenerator) GenerateNeedsClarificationResponse(ctx models.Context, missingAboutMe, missingIntention bool) string {
	if rg.llmClient == nil {
		return "I'd like to understand you better."
	}

	systemPrompt := "You are Moly, a communication coach. Generate a warm, natural question asking for missing context. One to two sentences. Be conversational."
	userPrompt := buildNeedsClarificationPrompt(ctx, missingAboutMe, missingIntention)

	response, err := rg.callLLM(systemPrompt, userPrompt)
	if err != nil {
		log.Printf("[ResponseGenerator] Warning: Failed to generate clarification response: %v", err)
		return "Tell me more?"
	}

	return response
}

// GenerateGapClarificationResponse generates targeted clarification questions for specific identified gaps
func (rg *ResponseGenerator) GenerateGapClarificationResponse(ctx models.Context, gaps []string) string {
	if rg.llmClient == nil || len(gaps) == 0 {
		return "I'd like to understand you better."
	}

	systemPrompt := `You are Moly, a communication coach. You've identified some missing context to better understand the user's situation.
Generate ONE natural, warm clarifying question about the most important missing piece.
One to two sentences. Be conversational and specific.`

	userPrompt := buildGapClarificationPrompt(ctx, gaps)

	response, err := rg.callLLM(systemPrompt, userPrompt)
	if err != nil {
		log.Printf("[ResponseGenerator] Warning: Failed to generate gap clarification response: %v", err)
		// Fallback to asking about the first gap
		return gapToDefaultQuestion(gaps[0])
	}

	return response
}

// GenerateIntentClarificationResponse generates a question when user's intent is unclear
func (rg *ResponseGenerator) GenerateIntentClarificationResponse(ctx models.Context, userMessage string) string {
	if rg.llmClient == nil {
		return "I want to make sure I understand what you're looking for. What's most important to you right now?"
	}

	systemPrompt := `You are Moly, a communication coach. The user's message is a bit unclear about what they're actually trying to figure out.
Generate ONE natural, warm clarifying question to understand their actual intent or goal.
One to two sentences. Ask what they're really looking for or what matters most.`

	userPrompt := buildIntentClarificationPrompt(ctx, userMessage)

	response, err := rg.callLLM(systemPrompt, userPrompt)
	if err != nil {
		log.Printf("[ResponseGenerator] Warning: Failed to generate intent clarification: %v", err)
		return "What's most important to you in this situation?"
	}

	return response
}

// GenerateFallbackResponse generates a response when agent processing fails
func (rg *ResponseGenerator) GenerateFallbackResponse(ctx models.Context, userMessage string) string {
	if rg.llmClient == nil {
		return "I'm here to listen."
	}

	systemPrompt := "You are Moly, a communication coach. Generate a brief, empathetic response showing you're listening. One to two sentences. Be warm and engaged."
	userPrompt := buildFallbackPrompt(ctx, userMessage)

	response, err := rg.callLLM(systemPrompt, userPrompt)
	if err != nil {
		log.Printf("[ResponseGenerator] Warning: Failed to generate fallback response: %v", err)
		return "I'm here to listen."
	}

	return response
}

// GenerateInitialGreeting generates a warm initial greeting
func (rg *ResponseGenerator) GenerateInitialGreeting(userName string) string {
	if rg.llmClient == nil {
		return "Hi, I'm Moly. How can I help you think through things?"
	}

	systemPrompt := "You are Moly, a communication coach. Generate a warm, brief initial greeting. One sentence only. Be conversational and inviting."
	userPrompt := fmt.Sprintf("Generate a greeting for a user named: %s", userName)

	response, err := rg.callLLM(systemPrompt, userPrompt)
	if err != nil {
		log.Printf("[ResponseGenerator] Warning: Failed to generate greeting: %v", err)
		return "Hi, I'm Moly. What's on your mind?"
	}

	return response
}

// GenerateClarificationAcknowledgment generates a response to clarification answers
func (rg *ResponseGenerator) GenerateClarificationAcknowledgment(ctx models.Context, clarificationType string) string {
	if rg.llmClient == nil {
		return "Got it, thanks for clarifying."
	}

	systemPrompt := "You are Moly. Generate a brief, warm acknowledgment. One sentence. Show you heard them."
	userPrompt := buildClarificationAckPrompt(ctx, clarificationType)

	response, err := rg.callLLM(systemPrompt, userPrompt)
	if err != nil {
		log.Printf("[ResponseGenerator] Warning: Failed to generate clarification ack: %v", err)
		return "Thanks for letting me know."
	}

	return response
}

// GenerateConflictQuestion generates a natural conflict resolution question
func (rg *ResponseGenerator) GenerateConflictQuestion(ctx models.Context, conflict *models.ConflictInfo) string {
	if rg.llmClient == nil {
		return fmt.Sprintf("I noticed you mentioned '%v' before, but now '%v'. Can you help me understand?",
			conflict.SavedValue, conflict.ExtractedValue)
	}

	systemPrompt := "You are Moly. Generate a natural, curious question to help clarify a potential conflict. One to two sentences. Be supportive, not accusatory. Sound like a listener."
	userPrompt := buildConflictQuestionPrompt(ctx, conflict)

	response, err := rg.callLLM(systemPrompt, userPrompt)
	if err != nil {
		log.Printf("[ResponseGenerator] Warning: Failed to generate conflict question: %v", err)
		return "I want to make sure I understand you correctly. Can you help clarify?"
	}

	return response
}

// Helper functions to build prompts

func buildEmptyMessagePrompt(ctx models.Context) string {
	return `The user sent an empty message.
Generate a brief, warm prompt asking them to share what's on their mind.
Be conversational and inviting.

Context:
- Communication style: ` + ctx.AboutMe.CommunicationStyle + `
- Message history: ` + fmt.Sprintf("%d messages", len(ctx.ConversationHistory)) + `

Generate ONLY the prompt, nothing else.`
}

func buildNeedsClarificationPrompt(ctx models.Context, missingAboutMe, missingIntention bool) string {
	missingContext := []string{}
	if missingAboutMe {
		missingContext = append(missingContext, "about their communication preferences")
	}
	if missingIntention {
		missingContext = append(missingContext, "about what they're trying to figure out")
	}

	contactName := "something"
	if ctx.ContactProfile != nil {
		contactName = conditionalValue(ctx.ContactProfile.Name, ctx.ContactProfile.Name, "something")
	}

	return fmt.Sprintf(`You need more context from the user about: %s.
Generate a natural, conversational question asking for this missing context.
Reference their existing context to show you're listening.

Existing context:
- Communication style: %s
- About: %s
- Talking about: %s

Generate ONLY the question, nothing else.`,
		strings.Join(missingContext, " and "),
		ctx.AboutMe.CommunicationStyle,
		fmt.Sprintf("%d fields complete", len(ctx.Gaps)),
		contactName,
	)
}

func buildFallbackPrompt(ctx models.Context, userMessage string) string {
	return fmt.Sprintf(`The user just said: "%s"
Generate an empathetic response showing you're listening and engaged.
Be brief and conversational.

Context:
- Their style: %s
- Talking about: %s

Generate ONLY the response, nothing else.`,
		userMessage,
		ctx.AboutMe.CommunicationStyle,
		conditionalValue(ctx.ContactProfile.Name, ctx.ContactProfile.Name, "something"),
	)
}

func buildClarificationAckPrompt(ctx models.Context, clarificationType string) string {
	return fmt.Sprintf(`The user just provided clarification about: %s.
Generate a brief acknowledgment showing you understood them.
Be warm and conversational.

User's communication style: %s

Generate ONLY the acknowledgment, nothing else.`,
		clarificationType,
		ctx.AboutMe.CommunicationStyle,
	)
}

func buildConflictQuestionPrompt(ctx models.Context, conflict *models.ConflictInfo) string {
	return fmt.Sprintf(`The user previously said: "%v" but now said: "%v".
Generate a natural, curious question to help them clarify or reconcile this.
Sound like a supportive listener, not an interrogator.

Conflict type: %s
User's communication style: %s

Generate ONLY the question, nothing else.`,
		conflict.SavedValue,
		conflict.ExtractedValue,
		conflict.ConflictType,
		ctx.AboutMe.CommunicationStyle,
	)
}

func conditionalValue(value, trueVal, falseVal string) string {
	if value != "" && value != "Contact" {
		return trueVal
	}
	return falseVal
}

// gapToQuestion maps identified context gaps to natural clarifying questions
func gapToQuestion(gap string) string {
	gapQuestions := map[string]string{
		"communicationStyle": "How do you usually communicate when something's important? Are you more direct and to-the-point, or do you prefer taking time to explain?",
		"coreValues": "What matters most to you in a situation like this? What's really important here for you?",
		"contact": "Tell me more about this person — who are they to you, and what's your relationship like?",
		"relevantReflections": "Have you been in a situation like this before? What happened then, and how did it turn out?",
		"pastIntention": "What are you really trying to figure out here? What would a good resolution look like for you?",
		"recentSafetyIncidents": "I want to make sure you're okay. Can you tell me more about what you're dealing with?",
	}

	if q, ok := gapQuestions[gap]; ok {
		return q
	}
	return "Tell me more about this situation."
}

// gapToDefaultQuestion returns a safe fallback question for a gap
func gapToDefaultQuestion(gap string) string {
	return gapToQuestion(gap)
}

// buildGapClarificationPrompt builds a prompt that targets specific identified gaps
func buildGapClarificationPrompt(ctx models.Context, gaps []string) string {
	if len(gaps) == 0 {
		return "Generate a question asking the user to tell you more about their situation."
	}

	// Prioritize which gap to ask about (some are more important than others)
	primaryGap := gaps[0]
	for _, gap := range gaps {
		// Prioritize certain gaps
		if gap == "pastIntention" || gap == "contact" {
			primaryGap = gap
			break
		}
	}

	gapDescription := gapToDescription(primaryGap)
	relatedContext := buildContextSummary(ctx)

	return fmt.Sprintf(`The user just said: "%s"

I've understood some parts of their situation, but I'm missing important context about: %s

Their current context:
%s

Generate a natural, warm clarifying question about what's missing. Ask about %s specifically.
Make it conversational and reference what they've already told you.

Generate ONLY the question, nothing else.`,
		ctx.ConversationHistory[0].Content,
		gapDescription,
		relatedContext,
		gapDescription,
	)
}

// gapToDescription provides human-readable descriptions of gaps
func gapToDescription(gap string) string {
	descriptions := map[string]string{
		"communicationStyle": "their communication style and preferences",
		"coreValues": "what really matters to them",
		"contact": "who they're talking about and their relationship",
		"relevantReflections": "whether they've experienced something similar",
		"pastIntention": "what they're ultimately trying to figure out",
		"recentSafetyIncidents": "their safety and wellbeing",
	}

	if desc, ok := descriptions[gap]; ok {
		return desc
	}
	return "more details"
}

// buildIntentClarificationPrompt builds a prompt for clarifying unclear intent
func buildIntentClarificationPrompt(ctx models.Context, userMessage string) string {
	return fmt.Sprintf(`The user just said: "%s"

I'm not quite sure what they're really trying to figure out or what matters most to them.

Generate a natural question that asks them to clarify their actual goal or intent.
Make it warm and conversational, not interrogative.

Generate ONLY the question, nothing else.`,
		userMessage,
	)
}

// buildContextSummary creates a brief summary of what we already know
func buildContextSummary(ctx models.Context) string {
	var summary []string

	if ctx.AboutMe != nil {
		if ctx.AboutMe.CommunicationStyle != "" {
			summary = append(summary, fmt.Sprintf("- Communication style: %s", ctx.AboutMe.CommunicationStyle))
		}
		if len(ctx.AboutMe.Values) > 0 {
			summary = append(summary, fmt.Sprintf("- Values: %s", strings.Join(ctx.AboutMe.Values, ", ")))
		}
	}

	if ctx.ContactProfile != nil && ctx.ContactProfile.Name != "" {
		summary = append(summary, fmt.Sprintf("- Talking about: %s (%s)", ctx.ContactProfile.Name, ctx.ContactProfile.Relationship))
	}

	if ctx.PastIntention != "" {
		summary = append(summary, fmt.Sprintf("- Goal: %s", ctx.PastIntention))
	}

	if len(summary) == 0 {
		return "- (just starting to understand their situation)"
	}

	return strings.Join(summary, "\n")
}
