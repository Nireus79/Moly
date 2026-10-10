package agents

import (
	"context"
	"log"
	"strings"

	"moly/models"
	"moly/tools"
)

// DeepeningJudgment is the model's view of whether a Socratic question would help now, and which of Moly's principles the
// situation touches (ORCHESTRATOR_DESIGN.md, "Socratic deepening": replaces the complexity score and the principle word lists).
type DeepeningJudgment struct {
	Worth      bool
	Principles []string // ids from the constitution, in the model's order
}

// JudgeDeepening asks the model, in one call, whether the situation is one a reflective question would help (a real choice,
// a feeling, a trade-off, a person who is affected) and which principles it touches. Anything unreadable or failed means
// "not worth": Moly then helps, which is the safe side. Only ids that exist in the constitution are returned.
func JudgeDeepening(ctx context.Context, llm tools.LLMProvider, message, goal, contact string, history []models.Message, principles []models.Principle) DeepeningJudgment {
	none := DeepeningJudgment{}
	if llm == nil || strings.TrimSpace(message) == "" || len(principles) == 0 {
		return none
	}
	list := ""
	known := map[string]bool{}
	for _, p := range principles {
		known[p.ID] = true
		list += "- " + p.ID + ": " + p.Description + "\n"
	}
	recent := ""
	start := len(history) - 6
	if start < 0 {
		start = 0
	}
	for _, m := range history[start:] {
		text := []rune(m.Content)
		if len(text) > 300 {
			text = text[:300]
		}
		recent += m.Role + ": " + string(text) + "\n"
	}
	req := &tools.LLMRequest{
		SystemPrompt: "You judge whether a reflective (Socratic) question would help the user now, and which principles the situation touches.",
		UserPrompt: "The user's goal: " + goal + "\nThe person involved: " + contact + "\nRecent conversation:\n" + recent +
			"\nThe user's latest message: \"" + message + "\"\n\nPrinciples:\n" + list + "\n" +
			"worth: true only if the situation has something the user may not have thought through (a real choice, a feeling, a trade-off, " +
			"someone who is affected). false if the request is plain and the user just needs the result.\n" +
			"principles: the ids from the list that this situation really touches (none if none does).\n" +
			"Respond with ONLY JSON: {\"worth\": true|false, \"principles\": [\"id\", ...]}",
		MaxTokens:   120,
		Temperature: 0.1,
	}
	resp, err := llm.Call(ctx, req)
	if err != nil {
		log.Printf("[Deepening] judgement failed: %v (not worth)", err)
		return none
	}
	var out struct {
		Worth      tools.LenientBool `json:"worth"`
		Principles []string          `json:"principles"`
	}
	if err := tools.SafeJSONParse("Deepening", []byte(resp.Content), &out); err != nil {
		log.Printf("[Deepening] judgement unreadable (not worth)")
		return none
	}
	j := DeepeningJudgment{Worth: bool(out.Worth)}
	for _, id := range out.Principles {
		if known[id] {
			j.Principles = append(j.Principles, id)
		}
	}
	log.Printf("[Deepening] worth=%v principles=%v", j.Worth, j.Principles)
	return j
}
