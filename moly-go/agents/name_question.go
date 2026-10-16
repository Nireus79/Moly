package agents

import (
	"context"
	"log"
	"strings"

	"moly/tools"
)

// nameQuestion asks for the name of a person the user mentioned without one. A message in English gets the fixed English question
// (NameQuestion); another language gets a question written by the model in that language (found in dynamic runs: a Greek user was asked
// in English, and, once the model wrote every name question, English users were asked in German). The model names the language first, so
// that an English message cannot drift into another language; NameQuestion is also the fallback.
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
		SystemPrompt: "You help Moly, a communication thinking partner, ask a user for a person's name.",
		UserPrompt: last + "The user said: \"" + userMessage + "\"\n\nThey mentioned a person without giving a name; the user's own word for the person is: " + l + ".\n\n" +
			"First name the language the USER writes in (look at their own words, not at names, usernames or the person's word). " +
			"Then write one short question in that language that asks for a name for that person (any name will do, a nickname too). " +
			"Respond with ONLY JSON: {\"language\": \"English\", \"question\": \"...\"}",
		MaxTokens:   120,
		Temperature: 0.1,
	}
	resp, err := ca.llmClient.Call(context.Background(), req)
	if err != nil {
		log.Printf("[ConversationAgent] Name question generation failed: %v", err)
		return fallback
	}
	var out struct {
		Language string `json:"language"`
		Question string `json:"question"`
	}
	if err := tools.SafeJSONParse("NameQuestion", []byte(resp.Content), &out); err != nil {
		log.Printf("[ConversationAgent] Name question unreadable: the standard question is used")
		return fallback
	}
	if strings.EqualFold(strings.TrimSpace(out.Language), "english") {
		return fallback
	}
	q := strings.Trim(strings.TrimSpace(out.Question), "\"“”")
	if !meetsQuestionContract(q) {
		return fallback
	}
	return q
}
