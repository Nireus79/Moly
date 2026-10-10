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

// JudgeSwitchAnswer asks the model whether the user agreed to change the goal to newGoal. Only a clear yes changes the
// locked goal: anything unreadable or failed is "unclear", and the lock stays.
func JudgeSwitchAnswer(ctx context.Context, llm tools.LLMProvider, question, answer, newGoal string) SwitchAnswer {
	if llm == nil || strings.TrimSpace(answer) == "" {
		return SwitchUnclear
	}
	req := &tools.LLMRequest{
		SystemPrompt: "You judge whether the user agreed to change what a conversation is working on.",
		UserPrompt: "Moly asked: \"" + question + "\"\nThe user answered: \"" + answer + "\"\nThe proposed new goal: \"" + newGoal + "\"\n\n" +
			"Did the user clearly agree to change to the new goal (\"yes\"), clearly refuse and keep the old one (\"no\"), or is it \"unclear\"?\n" +
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
