package agents

import (
	"context"
	"log"
	"strings"

	"moly/tools"
)

// A reply that asks the user something asks one short question about one thing (ORCHESTRATOR_DESIGN.md step 6).
// Every question reply passes through singleQuestion, whatever wrote it: a template, a layer or the model.
// The contract is checked on the text, not on the user's words: exactly one question mark, and a short reply.
// A reply may open with one short sentence saying what Moly understood, then ask the one question. When the user gives
// several things at once, Moly does not take them all: it confirms what it understood and asks about one thing.
const (
	maxQuestionWords = 45
	maxQuestionRunes = 400
)

// meetsQuestionContract reports whether the text already asks one short question.
func meetsQuestionContract(text string) bool {
	t := strings.TrimSpace(text)
	return t != "" && strings.Count(t, "?") == 1 && len(strings.Fields(t)) <= maxQuestionWords
}

// firstQuestion is the fixed fallback: the first question of the text, cut at its question mark and with the
// sentences before it dropped. It returns "" when the text holds no question.
func firstQuestion(text string) string {
	t := strings.TrimSpace(text)
	end := strings.Index(t, "?")
	if end < 0 {
		return ""
	}
	t = t[:end+1]
	start := strings.LastIndexAny(t[:end], ".!\n") + 1
	return strings.TrimSpace(t[start:])
}

// singleQuestion returns the text when it meets the contract. Otherwise the model rewrites it once as one short
// question; if that fails the contract, the first question of the original is used; if there is none, the text stays.
func (ca *conversationAgent) singleQuestion(userMessage, text string) string {
	if meetsQuestionContract(text) {
		return text
	}
	if ca.llmClient != nil {
		req := &tools.LLMRequest{
			SystemPrompt: "You rewrite replies so that each asks the user exactly one question.",
			UserPrompt: "The user said: \"" + userMessage + "\"\n\nDraft reply:\n" + text + "\n\n" +
				"Rewrite the draft as a short, warm reply of at most 40 words that asks exactly ONE question about ONE thing: the single most useful thing to ask next. " +
				"If the user gave several things at once, you may start with one short sentence saying what you understood, then ask only about one of them; the rest can wait. " +
				"Drop every other question and any list. Keep the subject and the names from the draft. Do not answer for the user. " +
				"Output only the reply.",
			MaxTokens:   120,
			Temperature: 0.3,
		}
		if resp, err := ca.llmClient.Call(context.Background(), req); err != nil {
			log.Printf("[ConversationAgent] One-question rewrite failed: %v", err)
		} else {
			out := strings.Trim(strings.TrimSpace(resp.Content), "\"“”")
			if meetsQuestionContract(out) && len([]rune(out)) <= maxQuestionRunes {
				return out
			}
			log.Printf("[ConversationAgent] One-question rewrite broke the contract; using the first question")
		}
	}
	if q := firstQuestion(text); q != "" {
		return q
	}
	return text
}
