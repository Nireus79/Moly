package agents

import (
	"context"
	"log"
	"strings"

	"moly/models"
	"moly/tools"
)

// SwitchAnswer is what the user said to "do you want to change what we are working on?".
type SwitchAnswer string

const (
	SwitchYes     SwitchAnswer = "yes"
	SwitchNo      SwitchAnswer = "no"
	SwitchUnclear SwitchAnswer = "unclear"
)

// The prompt names what counts as each answer: the first wording ("clearly agree ... or unclear") made Claude haiku answer
// "unclear" to a plain "Yes, switch." in 3 of 8 replays (15 of 24 right over eight kinds of answer; the version below 24 of 24).
// JudgeSwitchAnswer asks the model whether the user agreed to change the goal to newGoal. Only a clear yes changes the
// locked goal: anything unreadable or failed is "unclear", and the lock stays.
func JudgeSwitchAnswer(ctx context.Context, llm tools.LLMProvider, question, answer, newGoal string) SwitchAnswer {
	if llm == nil || strings.TrimSpace(answer) == "" {
		return SwitchUnclear
	}
	req := &tools.LLMRequest{
		SystemPrompt: "You judge whether the user agreed to change what a conversation is working on.",
		UserPrompt: "Moly asked the user to confirm a change of goal.\nMoly asked: \"" + question + "\"\nThe user answered: \"" + answer + "\"\nThe proposed new goal: \"" + newGoal + "\"\n\n" +
			"Read the answer as a reply to that question.\n" +
			"- \"yes\": the answer agrees to the change (\"yes\", \"yes, switch\", \"ok, let's do that\", \"right\").\n" +
			"- \"no\": the answer declines or keeps the old goal (\"no\", \"stay with the note\").\n" +
			"- \"unclear\": only when the answer does not respond to the question, is conditional, or asks something else.\n" +
			"Respond with ONLY JSON: {\"answer\": \"yes|no|unclear\"}",
		MaxTokens:   40,
		Temperature: 0.1,
	}
	resp, err := llm.Call(ctx, req)
	if err != nil {
		log.Printf("[GoalSwitch] judgement failed: %v (treated as unclear)", err)
		return SwitchUnclear
	}
	var out struct {
		Answer string `json:"answer"`
	}
	if err := tools.SafeJSONParse("GoalSwitch", []byte(resp.Content), &out); err != nil {
		return SwitchUnclear
	}
	switch a := SwitchAnswer(strings.ToLower(strings.TrimSpace(out.Answer))); a {
	case SwitchYes, SwitchNo:
		return a
	}
	return SwitchUnclear
}

// LastAssistantText returns what Moly said last. The history is [current, oldest ... newest].
func LastAssistantText(history []models.Message) string {
	for i := len(history) - 1; i >= 1; i-- {
		if history[i].Role == "assistant" {
			return history[i].Content
		}
	}
	return ""
}
