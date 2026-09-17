package tools

import (
	"context"
	"fmt"
	"log"
)

// ClarificationAskerInput - What context we're missing
type ClarificationAskerInput struct {
	Message             string   // The user's message
	MissingContextTypes []string // "aboutMe", "contact", "intention"
	ConversationHistory []string // Previous messages for context
}

// ClarificationAskerOutput - Natural clarification question
type ClarificationAskerOutput struct {
	Question string // The natural follow-up question
	Focus    string // What we're asking about
}

// ClarificationAsker - Asks for missing context naturally
type ClarificationAsker struct {
	llm LLMProvider
}

// NewClarificationAsker - Create new clarification asker
func NewClarificationAsker(llm LLMProvider) *ClarificationAsker {
	return &ClarificationAsker{llm: llm}
}

// Ask - Ask naturally for missing context
func (ca *ClarificationAsker) Ask(ctx context.Context, input *ClarificationAskerInput) (*ClarificationAskerOutput, error) {
	if input == nil || input.Message == "" {
		return &ClarificationAskerOutput{
			Question: "Tell me more about that?",
			Focus:    "general",
		}, nil
	}

	if ca.llm == nil {
		// Fallback questions
		return ca.fallbackQuestion(input.MissingContextTypes), nil
	}

	systemPrompt := ca.buildSystemPrompt()
	userPrompt := ca.buildUserPrompt(input)

	req := &LLMRequest{
		SystemPrompt:        systemPrompt,
		UserPrompt:          userPrompt,
		Temperature:         0.7,
		MaxTokens:           150,
		UseExtendedThinking: false,
	}

	resp, err := ca.llm.Call(ctx, req)
	if err != nil {
		log.Printf("[ClarificationAsker] Error: %v, using fallback", err)
		return ca.fallbackQuestion(input.MissingContextTypes), nil
	}

	return &ClarificationAskerOutput{
		Question: resp.Content,
		Focus:    input.MissingContextTypes[0], // First missing context type
	}, nil
}

func (ca *ClarificationAsker) buildSystemPrompt() string {
	return `You help Moly ask natural follow-up questions to understand someone better.

Your role: Ask ONE natural, warm question that helps understand their situation.
- Never interrogate or list multiple questions
- Sound like a friend, not an interviewer
- Ask what's most important to understand right now
- Keep it open-ended and genuine
- One sentence maximum

Respond with just the question (no explanation).`
}

func (ca *ClarificationAsker) buildUserPrompt(input *ClarificationAskerInput) string {
	missing := ""
	if len(input.MissingContextTypes) > 0 {
		missing = fmt.Sprintf("We don't know yet: %v\n", input.MissingContextTypes)
	}

	history := ""
	if len(input.ConversationHistory) > 0 {
		history = "Previous: " + input.ConversationHistory[0]
		if len(input.ConversationHistory) > 1 {
			history += " → " + input.ConversationHistory[1]
		}
		history += "\n"
	}

	return fmt.Sprintf(`User just said: "%s"

%s%sWhat would help me understand their situation better? Ask ONE natural question.`, input.Message, missing, history)
}

func (ca *ClarificationAsker) fallbackQuestion(missingTypes []string) *ClarificationAskerOutput {
	if len(missingTypes) == 0 {
		return &ClarificationAskerOutput{
			Question: "Tell me more?",
			Focus:    "general",
		}
	}

	questions := map[string]string{
		"aboutMe":   "What's your communication style like usually?",
		"contact":   "Who are you talking about?",
		"intention": "What are you hoping to figure out?",
	}

	for _, t := range missingTypes {
		if q, ok := questions[t]; ok {
			return &ClarificationAskerOutput{
				Question: q,
				Focus:    t,
			}
		}
	}

	return &ClarificationAskerOutput{
		Question: "Can you tell me more about that?",
		Focus:    missingTypes[0],
	}
}
