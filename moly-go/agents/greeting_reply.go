package agents

import (
	"context"
	"log"
	"strings"

	"moly/tools"
)

// fallbackGreeting is used when the model cannot write the greeting. It is the whole greeting reply (user decision,
// 2026-10-10): Moly says who it is and asks how it can help, nothing more.
const fallbackGreeting = "I am Moly. How can I help you?"

// maxGreetingWords bounds the model's greeting; a longer text falls back to the fixed one.
const maxGreetingWords = 25

// greetingReply writes the reply to a greeting in the user's language: who Moly is, and one question about how it can
// help. The model's text must meet the one-question contract; otherwise the fixed greeting is used.
func (ca *conversationAgent) greetingReply(userMessage string) string {
	if ca.llmClient == nil {
		return fallbackGreeting
	}
	req := &tools.LLMRequest{
		SystemPrompt: "You write the first reply of Moly, a communication thinking partner.",
		UserPrompt: "The user greeted Moly: \"" + userMessage + "\"\n\n" +
			"Reply in the user's language with one short sentence saying you are Moly, then one short question asking how you can help. " +
			"No advice, no list, no other questions. Output only the reply.",
		MaxTokens:   60,
		Temperature: 0.3,
	}
	resp, err := ca.llmClient.Call(context.Background(), req)
	if err != nil {
		log.Printf("[ConversationAgent] Greeting generation failed: %v", err)
		return fallbackGreeting
	}
	out := strings.Trim(strings.TrimSpace(resp.Content), "\"“”")
	// The prompt asks for two short sentences; a longer text (a live run added a "friendly reminder") is not a greeting.
	if !meetsQuestionContract(out) || len(strings.Fields(out)) > maxGreetingWords {
		return fallbackGreeting
	}
	return out
}
