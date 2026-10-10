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
	llmClient         LLMProvider
	responseValidator interface{} // PHASE 3: Response validation (set via SetResponseValidator)
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


// GenerateGapClarificationResponse generates targeted clarification questions for specific identified gaps
func (rg *ResponseGenerator) GenerateGapClarificationResponse(ctx models.Context, gaps []string) string {
	// Adaptive greeting: only greet on first message or topic change
	shouldGreet := shouldAddGreeting(ctx)
	greeting := ""
	if shouldGreet {
		greeting = "Hi! I'd love to help you with that.\n\n"
	}

	// FIX #29: Validate gaps array before use
	if rg.llmClient == nil || len(gaps) == 0 {
		return greeting + "I'd like to understand you better."
	}

	// Sanitize gaps before passing to LLM (prevent echoing full message)
	validGaps := make([]string, 0)
	for _, gap := range gaps {
		if gap != "" && len(gap) < 200 { // Skip empty or excessively long gaps
			validGaps = append(validGaps, gap)
		}
	}

	if len(validGaps) == 0 {
		log.Printf("[ResponseGenerator] FIX #29: All gaps invalid or empty, using fallback")
		return greeting + "I'd like to understand you better."
	}

	systemPrompt := `You are Moly, a communication coach. You've identified some missing context to better understand the user's situation.
Generate ONE natural, warm clarifying question about the most important missing piece.
One to two sentences. Be conversational and specific.`

	userPrompt := buildGapClarificationPrompt(ctx, validGaps)

	response, err := rg.callLLM(systemPrompt, userPrompt)
	if err != nil {
		log.Printf("[ResponseGenerator] Warning: Failed to generate gap clarification response: %v", err)
		// FIX #29: Safely access first gap with length check
		if len(validGaps) > 0 {
			return greeting + gapToDefaultQuestion(validGaps[0])
		}
		return greeting + "I'd like to understand you better."
	}

	return greeting + response
}

// shouldAddGreeting determines if a greeting is appropriate
func shouldAddGreeting(ctx models.Context) bool {
	// Greet on first message in conversation
	if ctx.IsFirstMessageInConversation {
		return true
	}

	// Greet if very few messages in conversation history (early in conversation)
	if len(ctx.ConversationHistory) <= 2 {
		return true
	}

	// Don't greet in middle of established conversation
	return false
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






// gapToQuestion maps identified context gaps to natural clarifying questions
func gapToQuestion(gap string) string {
	gapQuestions := map[string]string{
		"communicationStyle":    "How do you usually communicate when something's important? Are you more direct and to-the-point, or do you prefer taking time to explain?",
		"coreValues":            "What matters most to you in a situation like this? What's really important here for you?",
		"contact":               "Tell me more about this person — who are they to you, and what's your relationship like?",
		"relevantReflections":   "Have you been in a situation like this before? What happened then, and how did it turn out?",
		"pastIntention":         "What are you really trying to figure out here? What would a good resolution look like for you?",
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
	lastUserMessage := ""
	if len(ctx.ConversationHistory) > 0 {
		lastUserMessage = ctx.ConversationHistory[0].Content
	}

	return fmt.Sprintf(`The user just said: "%s"

I've understood some parts of their situation, but I'm missing important context about: %s

Their current context:
%s

You are Moly, speaking TO the user. Never write as the user and never claim their thanks, feelings or situation as your own: address them as "you".
Generate a natural, warm clarifying question about what's missing. Ask about %s specifically.
Make it conversational and reference what they've already told you. If they gave several things at once, you may open with one short sentence saying what you understood, then ask only about this one thing; the rest can wait.
Ask exactly ONE question.

Generate ONLY the reply, nothing else.`,
		lastUserMessage,
		gapDescription,
		relatedContext,
		gapDescription,
	)
}

// gapToDescription provides human-readable descriptions of gaps
func gapToDescription(gap string) string {
	descriptions := map[string]string{
		// Legacy gap types
		"communicationStyle":    "their communication style and preferences",
		"coreValues":            "what really matters to them",
		"contact":               "who they're talking about and their relationship",
		"relevantReflections":   "whether they've experienced something similar",
		"pastIntention":         "what they're ultimately trying to figure out",
		"recentSafetyIncidents": "their safety and wellbeing",

		// NEW: Extracted data gap types (message-specific, higher priority)
		"extracted_preference_needs_context":     "how their stated preference applies to this specific situation",
		"extracted_characteristic_needs_context": "how their characteristic or experience informs their approach here",
		"extracted_negation_needs_clarification": "what they would prefer instead of what they've ruled out",
	}

	if desc, ok := descriptions[gap]; ok {
		return desc
	}
	// Not a legacy gap code: the gap is already a description written by the gap detector. Use it as written.
	if strings.TrimSpace(gap) != "" {
		return gap
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
