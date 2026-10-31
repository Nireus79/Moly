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
	prompt := last + "The user said: \"" + userMessage + "\"\n\nThey mentioned a person without giving a name; the user's own word for the person is: " + l + ".\n\n" +
		"First name the language the USER writes in (look at their own words, not at names, usernames or the person's word). " +
		"Then write one short question in that language that asks for a name for that person (any name will do, a nickname too)." + tools.InUsersLanguage + " " +
		"Respond with ONLY JSON: {\"language\": \"English\", \"question\": \"...\"}"
	// The reply role first, then the fast role; the fixed English question only when neither gave a usable one (found live, 2026-10-30:
	// a Greek user was asked in English once in three runs, with nothing in the log to say why).
	for _, role := range []string{tools.RoleReply, tools.RoleFast} {
		resp, err := ca.llmClient.Call(context.Background(), &tools.LLMRequest{
			Role:         role,
			SystemPrompt: "You help Moly, a communication thinking partner, ask a user for a person's name.",
			UserPrompt:   prompt,
			MaxTokens:    120,
			Temperature:  0.1,
		})
		if err != nil {
			log.Printf("[ConversationAgent] Name question (%s role) failed: %v", role, err)
			continue
		}
		var out struct {
			Language string `json:"language"`
			Question string `json:"question"`
		}
		if err := tools.SafeJSONParse("NameQuestion", []byte(resp.Content), &out); err != nil {
			log.Printf("[ConversationAgent] Name question (%s role) unreadable", role)
			continue
		}
		if strings.EqualFold(strings.TrimSpace(out.Language), "english") {
			return fallback
		}
		q := strings.Trim(strings.TrimSpace(out.Question), "\"“”")
		if !meetsQuestionContract(q) {
			log.Printf("[ConversationAgent] Name question (%s role) broke the question contract", role)
			continue
		}
		return q
	}
	log.Printf("[ConversationAgent] No model wrote the name question: the standard English question is used")
	return fallback
}
