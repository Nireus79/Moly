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
		conditionalValue(ctx.ContactProfile.Name, ctx.ContactProfile.Name, "something"),
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
