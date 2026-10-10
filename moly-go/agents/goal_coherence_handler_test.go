package agents

import (
	"testing"

	"moly/models"
	"moly/tools"
)

func coherenceCtx(primary, current, relation string) *tools.LayerContext {
	return &tools.LayerContext{
		PrimaryGoal:  primary,
		UserGoal:     current,
		GoalRelation: relation,
		Analysis:     &models.AnalysisContext{MessageCount: 3},
	}
}

// The relation is the model's judgement, never the wording: a reworded goal is not a different goal, and an unjudged
// difference is not a shift.
func TestGoalCoherenceFollowsTheJudgedRelation(t *testing.T) {
	cases := []struct {
		name, relation, want string
		same                 bool
	}{
		{"judged different", "different", "different", false},
		{"judged refinement", "refinement", "related_subgoal", false},
		{"judged substep", "substep", "related_subgoal", false},
		{"judged same", "same", "same", true},
		{"not judged although the wording differs", "", "same", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := AnalyzeGoalCoherence(coherenceCtx("Express gratitude for a favor", "Maintaining friendship and seeking assistance", c.relation))
			if got.GoalProgression != c.want || got.IsSameGoal != c.same {
				t.Fatalf("progression %q same=%v, want %q same=%v", got.GoalProgression, got.IsSameGoal, c.want, c.same)
			}
		})
	}
	if got := AnalyzeGoalCoherence(coherenceCtx("", "anything", "")); got.GoalProgression != "same" {
		t.Fatal("no locked goal yet: nothing to compare")
	}
	if got := AnalyzeGoalCoherence(coherenceCtx("a goal", "", "")); got.GoalProgression != "same" {
		t.Fatal("no goal in this message: the locked goal continues")
	}
}
