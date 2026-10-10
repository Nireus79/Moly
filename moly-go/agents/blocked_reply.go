package agents

import (
	"context"
	"log"
	"strings"

	"moly/tools"
)

const fallbackBlockedReplyQuestion = "What would you like to focus on first?"

// replacementForBlockedReply is what the user sees when the output gate flags Moly's own draft. The user's message
// already passed the safety stage, so the gate is a second opinion on the draft, not a verdict on the user: the
// draft is dropped and Moly asks one question instead of refusing (a refusal belongs to the safety stage only).
func (ca *conversationAgent) replacementForBlockedReply(userMessage string) string {
	if ca.llmClient == nil {
		return fallbackBlockedReplyQuestion
	}
	req := &tools.LLMRequest{
		SystemPrompt: "You write one question for Moly, a communication thinking partner.",
		UserPrompt: "The user said: \"" + userMessage + "\"\n\n" +
			"Write one short, warm question, in the user's language, that helps the user say what they want to work on first. " +
			"If they mentioned several things, you may start with one short sentence of what you understood. " +
			"One question only. Do not give advice and do not refuse. Output only the reply.",
		MaxTokens:   80,
		Temperature: 0.3,
	}
	resp, err := ca.llmClient.Call(context.Background(), req)
	if err != nil {
		log.Printf("[ConversationAgent] Replacement question failed: %v", err)
		return fallbackBlockedReplyQuestion
	}
	out := strings.Trim(strings.TrimSpace(resp.Content), "\"“”")
	if !meetsQuestionContract(out) {
		return fallbackBlockedReplyQuestion
	}
	return out
}
