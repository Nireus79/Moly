package agents

import (
	"context"
	"log"
	"strings"

	"moly/models"
	"moly/tools"
)

// JudgeTraitChange asks the model whether the traits a message gives about a person say that the person CHANGED, against what
// Moly knew before this message. Rewordings, more detail and a different side of the same person are not a change; only a
// description that cannot hold together with the earlier one is (the person was one way and is now the opposite). Two lists are
// never compared by their words: the same trait is worded differently from one message to the next (found live 2026-10-14: the
// traits of one message were compared with themselves, reworded, and Moly asked the user to confirm a "change").
//
// Anything unreadable or failed counts as "no change": the traits are added to what is known and nothing is replaced, so the
// worst case is a missing question, never a lost fact.
func JudgeTraitChange(ctx context.Context, llm tools.LLMProvider, exchange string, before, incoming []string) bool {
	if llm == nil || len(before) == 0 || len(incoming) == 0 {
		return false
	}
	req := &tools.LLMRequest{
		SystemPrompt: "You decide whether a person has changed.",
		UserPrompt: models.WithExchange(exchange) + "What is already known about one person: " + strings.Join(before, "; ") + "\n" +
			"What the user's latest message says about the same person: " + strings.Join(incoming, "; ") + "\n\n" +
			"Decide:\n" +
			"- \"changed\": the new description cannot hold together with the known one; the person was one way and is now described as the opposite, or as clearly different.\n" +
			"- \"same\": anything else: the same traits in other words, more detail, a new side of the person, or traits that fit together.\n" +
			"Respond with ONLY JSON: {\"verdict\": \"changed|same\"}",
		MaxTokens:   30,
		Temperature: 0.1,
	}
	resp, err := llm.Call(ctx, req)
	if err != nil {
		log.Printf("[Identity] trait change judgement failed: %v (taken as no change)", err)
		return false
	}
	var out struct {
		Verdict string `json:"verdict"`
	}
	if err := tools.SafeJSONParse("TraitChange", []byte(resp.Content), &out); err != nil {
		log.Printf("[Identity] trait change judgement unreadable (taken as no change)")
		return false
	}
	return strings.EqualFold(strings.TrimSpace(out.Verdict), "changed")
}
