package agents

import (
	"context"
	"log"
	"strings"

	"moly/tools"
)

// nameQuestion asks for the name of a person the user mentioned without one. The model writes it in the user's language (found in a
// dynamic run: a Greek user was asked in English, with the Greek word pasted into the English sentence); NameQuestion is the fallback.
func (ca *conversationAgent) nameQuestion(userMessage, molyLast, label string) string {
	fallback := NameQuestion(label)
	if ca.llmClient == nil || strings.TrimSpace(userMessage) == "" {
		return fallback
	}
	l := strings.TrimSpace(label)
	if l == "" || len(strings.Fields(l)) > 3 {
		l = "the person they mentioned"
	}
	last := ""
	if strings.TrimSpace(molyLast) != "" {
		last = "Moly's earlier message: \"" + molyLast + "\"\n"
	}
	req := &tools.LLMRequest{
		Role:         tools.RoleReply,
		SystemPrompt: "You write one short question for Moly, a communication thinking partner.",
		UserPrompt: last + "The user said: \"" + userMessage + "\"\n\nThey mentioned a person without giving a name; the user's own word for the person is: " + l + ".\n\n" +
			"Write one short question that asks them for a name for that person (any name will do, a nickname too). Write it in the language the user is writing in " +
			"(the language of their message and of Moly's earlier message above), whatever language the person's word comes from. " +
			"Say nothing else and give no advice. Output only the question.",
		MaxTokens:   60,
		Temperature: 0.2,
	}
	resp, err := ca.llmClient.Call(context.Background(), req)
	if err != nil {
		log.Printf("[ConversationAgent] Name question generation failed: %v", err)
		return fallback
	}
	out := strings.Trim(strings.TrimSpace(resp.Content), "\"“”")
	if !meetsQuestionContract(out) {
		return fallback
	}
	return out
}
