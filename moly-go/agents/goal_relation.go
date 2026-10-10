package agents

import (
	"context"
	"log"
	"strings"

	"moly/tools"
)

// GoalRelation is how a goal stated in a message relates to the conversation's locked goal.
type GoalRelation string

const (
	GoalSame       GoalRelation = "same"       // the same aim in other words
	GoalRefinement GoalRelation = "refinement" // the same aim, made more specific
	GoalSubstep    GoalRelation = "substep"    // a step toward the locked aim
	GoalDifferent  GoalRelation = "different"  // another aim: the user changed what they are trying to do
)

// JudgeGoalRelation asks the model how the stated goal relates to the locked one. Only "different" is a shift.
// A different wording is not a different goal, so no text comparison decides this. When the model cannot answer
// the relation is "same": an unjudged goal never causes a shift question.
func JudgeGoalRelation(ctx context.Context, llm tools.LLMProvider, locked, stated string) GoalRelation {
	if llm == nil || strings.TrimSpace(locked) == "" || strings.TrimSpace(stated) == "" {
		return GoalSame
	}
	req := &tools.LLMRequest{
		SystemPrompt: "You judge how a newly stated goal relates to the goal a conversation is about.",
		UserPrompt: "Locked goal: \"" + locked + "\"\nNewly stated goal: \"" + stated + "\"\n\n" +
			"Is the newly stated goal: \"same\" (the same aim in other words), \"refinement\" (the same aim, more specific), " +
			"\"substep\" (a step toward the locked aim), or \"different\" (another aim altogether)?\n" +
			"Different wording alone is not a different goal.\n" +
			"Respond with ONLY JSON: {\"relation\": \"same|refinement|substep|different\"}",
		MaxTokens:   60,
		Temperature: 0.1,
	}
	resp, err := llm.Call(ctx, req)
	if err != nil {
		log.Printf("[GoalRelation] judgement failed: %v (treated as the same goal)", err)
		return GoalSame
	}
	var out struct {
		Relation string `json:"relation"`
	}
	if err := tools.SafeJSONParse("GoalRelation", []byte(resp.Content), &out); err != nil {
		return GoalSame
	}
	switch r := GoalRelation(strings.ToLower(strings.TrimSpace(out.Relation))); r {
	case GoalSame, GoalRefinement, GoalSubstep, GoalDifferent:
		return r
	}
	return GoalSame
}
