package agents

import (
	"context"
	"log"
	"os"
	"strings"

	"moly/tools"
)

// JudgeUnsupportedPremise asks the model whether a reply Moly is about to send takes as true something the conversation does not
// state: an event, a fact about a person, a role, a relation (found live, 2026-10-14: "after his father passed away" when the
// father had retired; "your father" for someone else's father). It returns the first such premise in a few words, or "" when the
// reply rests on what the dialogue says. A question that only offers a possibility asserts nothing; a feeling or a kind word
// asserts nothing. Anything unreadable or failed counts as supported: the reply goes out as it is, the check never blocks a turn.
func JudgeUnsupportedPremise(ctx context.Context, llm tools.LLMProvider, dialogue, userMessage, reply string) string {
	if llm == nil || strings.TrimSpace(reply) == "" {
		return ""
	}
	req := &tools.LLMRequest{
		SystemPrompt: "You check what a reply takes for granted.",
		UserPrompt: dialogue + "The user's latest message: \"" + userMessage + "\"\n\nMoly's reply, about to be sent:\n\"" + reply + "\"\n\n" +
			"Find what the reply TAKES AS TRUE about the user, about a person or about events: a fact, an event, a role, a relation (whose father, whose boss), a time, a feeling the user did not state. " +
			"A premise is supported when the dialogue above or the latest message states it, or when it follows directly from them. " +
			"A person belongs to the user only if the dialogue says so: a relation to another person is not a relation to the user. " +
			"Not premises: a question that only asks whether or when something is so (\"did it happen?\" asserts nothing), kind words, wishes, and restating what the user said. " +
			"Flag a premise only when the reply clearly ASSERTS it as true and the dialogue does not contain it or contradicts it. When in doubt, the reply is supported.\n" +
			"Respond with ONLY JSON: {\"unsupported\": \"the first premise the dialogue does not support, in a few words; empty when every premise is supported\"}",
		MaxTokens:   120,
		Temperature: 0.1,
	}
	resp, err := llm.Call(ctx, req)
	if err != nil {
		log.Printf("[Premise] check failed: %v (reply sent as it is)", err)
		return ""
	}
	var out struct {
		Unsupported string `json:"unsupported"`
	}
	if err := tools.SafeJSONParse("PremiseCheck", []byte(resp.Content), &out); err != nil {
		log.Printf("[Premise] check unreadable (reply sent as it is)")
		return ""
	}
	return strings.TrimSpace(out.Unsupported)
}

// groundedReply returns the reply, rewritten once when it takes something unsupported for granted. The rewrite keeps the reply's
// purpose and form (a question stays one question) and drops the premise. The user's words and the people, with what they are,
// are the only ground. If the rewrite fails, the original goes out: a check that fails never costs the user an answer.
func (ca *conversationAgent) groundedReply(userMessage, reply, dialogue string) string {
	if ca.llmClient == nil || strings.TrimSpace(reply) == "" {
		return reply
	}
	// The check guards against a reply model that adds what was not said. With the models Moly was tested with it flagged none of 7
	// replies (Sonnet 5.5 writing them), against 1-4 of 5 with Haiku; so it is not made for them (one call fewer per reply).
	// Any other model (Haiku for replies, Ollama, OpenAI, a test) is checked. MOLY_PREMISE_CHECK=always checks every reply.
	if approved, ok := ca.llmClient.(tools.ApprovedReplies); ok && approved.RepliesAreApproved() && os.Getenv("MOLY_PREMISE_CHECK") != "always" {
		return reply
	}
	unsupported := JudgeUnsupportedPremise(context.Background(), ca.llmClient, dialogue, userMessage, reply)
	if unsupported == "" {
		return reply
	}
	// A small model told "do not say X" may say it again (found live, 2026-10-15: "your father's name is Alex" after a rewrite).
	// So the rewrite is checked once more; if it still takes something for granted, the last attempt names people by name only and
	// says nothing about what they are to anyone, which leaves no relation to get wrong.
	current := reply
	for attempt := 1; attempt <= 2; attempt++ {
		log.Printf("[Premise] the draft took something for granted that the conversation does not state; rewriting it (attempt %d)", attempt)
		rule := "Use only what the conversation states; give nobody a role or a relation it does not state."
		if attempt == 2 {
			rule = "Refer to every person by name only, and say nothing about what anyone is to anyone (no father, boss, friend or the like)."
		}
		req := &tools.LLMRequest{
			Role:         tools.RoleReply,
			SystemPrompt: "You rewrite replies for Moly, a communication thinking partner.",
			UserPrompt: dialogue + "The user's latest message: \"" + userMessage + "\"\n\nDraft reply:\n\"" + current + "\"\n\n" +
				"The draft takes this for granted, and the conversation does not state it: " + unsupported + "\n" +
				"Rewrite the draft so that it no longer does, keeping what it was for and its form (if it asks one question, it still asks one question, about the same matter, at most 40 words). " +
				rule + " Output only the reply.",
			MaxTokens:   200,
			Temperature: 0.3,
		}
		resp, err := ca.llmClient.Call(context.Background(), req)
		if err != nil {
			log.Printf("[Premise] rewrite failed: %v (draft sent as it is)", err)
			return current
		}
		out := strings.Trim(strings.TrimSpace(resp.Content), "\"“”")
		if out == "" {
			return current
		}
		current = out
		if attempt == 2 {
			break
		}
		if again := JudgeUnsupportedPremise(context.Background(), ca.llmClient, dialogue, userMessage, current); again == "" {
			return current
		} else {
			unsupported = again
		}
	}
	return current
}
