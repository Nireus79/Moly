package agents

import (
	"context"
	"fmt"
	"log"
	"strings"

	"moly/tools"
)

// confirmQuestion asks the user to confirm a fact Moly understood with doubt and did not save. The model writes it
// under the one-question contract; the fixed question is the fallback.
func (ca *conversationAgent) confirmQuestion(userMessage, fact string) string {
	fallback := fmt.Sprintf("Just to be sure, did you mean %s?", fact)
	if ca.llmClient == nil {
		return fallback
	}
	req := &tools.LLMRequest{
		SystemPrompt: "You write one confirming question for Moly, a communication thinking partner.",
		UserPrompt: "The user said: \"" + userMessage + "\"\n\nMoly understood this, but is not sure of it: " + fact + "\n\n" +
			"Write one short, warm question, in the user's language, that asks the user to confirm or correct exactly that. " +
			"One question only. Do not give advice. Output only the question.",
		MaxTokens:   80,
		Temperature: 0.3,
	}
	resp, err := ca.llmClient.Call(context.Background(), req)
	if err != nil {
		log.Printf("[ConversationAgent] Confirm question generation failed: %v", err)
		return fallback
	}
	out := strings.Trim(strings.TrimSpace(resp.Content), "\"“”")
	if !meetsQuestionContract(out) {
		return fallback
	}
	return out
}
