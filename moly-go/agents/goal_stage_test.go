package agents

import (
	"context"
	"errors"
	"testing"

	"moly/models"
)

func TestResolveGoalTable(t *testing.T) {
	cases := []struct {
		name           string
		locked, stated string
		relation       GoalRelation
		want           GoalState
	}{
		{"nothing yet, nothing stated", "", "", GoalSame, GoalState{}},
		{"first goal is locked", "", "write a first message", GoalSame, GoalState{Locked: "write a first message", Working: "write a first message", JustLocked: true}},
		{"no goal stated continues the lock", "write a first message", "", GoalSame, GoalState{Locked: "write a first message", Working: "write a first message"}},
		{"same goal again is not a change", "write a first message", "Write a first message", GoalSame, GoalState{Locked: "write a first message", Working: "Write a first message"}},
		{"a reworded aim is not a change", "write a first message", "initiate a conversation with her", GoalRefinement, GoalState{Locked: "write a first message", Working: "initiate a conversation with her"}},
		{"a step toward the aim is not a change", "write a first message", "choose a topic for it", GoalSubstep, GoalState{Locked: "write a first message", Working: "choose a topic for it"}},
		{"different goal is a change and does not replace the lock", "write a first message", "decide whether to disclose", GoalDifferent, GoalState{Locked: "write a first message", Working: "decide whether to disclose", Changed: true}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ResolveGoal(c.locked, c.stated, c.relation); got != c.want {
				t.Fatalf("got %+v, want %+v", got, c.want)
			}
		})
	}
}

type fixedIntent struct{ a IntentAnalysis }

func (f fixedIntent) DetectIntentWithLLM(string, []models.Message) IntentAnalysis { return f.a }

func TestDecideIntentGreetingStripsGoalAndPerson(t *testing.T) {
	d := DecideIntent(fixedIntent{IntentAnalysis{Intent: IntentGreet, Confidence: 0.9}}, "Hi Moly", nil)
	if !d.Greeting {
		t.Fatal("a greeting intent must be flagged")
	}
	ec := &models.ExtractedContext{Intention: "talk", Contact: &models.ExtractedContact{Name: "x"}}
	d.StripGreeting(ec)
	if ec.Intention != "" || ec.Contact != nil {
		t.Fatalf("a greeting keeps no goal and no person: %+v", ec)
	}
}

func TestDecideIntentOtherIntentsAreUntouched(t *testing.T) {
	d := DecideIntent(fixedIntent{IntentAnalysis{Intent: IntentShare, Confidence: 0.9}}, "I want to write to a girl", nil)
	ec := &models.ExtractedContext{Intention: "write", Contact: &models.ExtractedContact{Name: "girl"}}
	d.StripGreeting(ec)
	if d.Greeting || ec.Intention != "write" || ec.Contact == nil {
		t.Fatalf("only greetings are stripped: %+v greeting=%v", ec, d.Greeting)
	}
}

func TestDecideIntentWithoutDetectorOrMessageIsNotAGreeting(t *testing.T) {
	if DecideIntent(nil, "Hi", nil).Greeting {
		t.Fatal("no detector: unknown, not a greeting")
	}
	if DecideIntent(fixedIntent{IntentAnalysis{Intent: IntentGreet}}, "", nil).Greeting {
		t.Fatal("no message: not a greeting")
	}
}

func TestWithoutGoalEntities(t *testing.T) {
	in := []models.ExtractedEntity{{Type: "goal"}, {Type: "contact"}, {Type: "goal_component"}, {Type: "value"}}
	out := WithoutGoalEntities(in)
	if len(out) != 2 || out[0].Type != "contact" || out[1].Type != "value" {
		t.Fatalf("got %+v", out)
	}
}

func TestDecideIntentAnswerOnly(t *testing.T) {
	ask := []models.Message{{Role: "user", Content: "x"}, {Role: "assistant", Content: "Can you give me a name?"}}
	claim := fixedIntent{IntentAnalysis{Intent: IntentShare, Confidence: 0.9, AnswersQuestion: true}}

	if !DecideIntent(claim, "Her name is Christine.", ask).AnswerOnly {
		t.Fatal("a claimed answer after a Moly message must be answer-only")
	}
	if DecideIntent(claim, "Her name is Christine.", nil).AnswerOnly {
		t.Fatal("with no earlier Moly message nothing can be answered")
	}
	if DecideIntent(claim, "Her name is Christine.", ask[:1]).AnswerOnly {
		t.Fatal("when the last message was the user's, nothing was asked")
	}
	if DecideIntent(fixedIntent{IntentAnalysis{Intent: IntentShare}}, "x", ask).AnswerOnly {
		t.Fatal("no claim, no answer-only")
	}
	if DecideIntent(fixedIntent{IntentAnalysis{Intent: IntentGreet, AnswersQuestion: true}}, "Hi", ask).AnswerOnly {
		t.Fatal("a greeting is not an answer")
	}

	ec := &models.ExtractedContext{Intention: "identify who Christine is", Contact: &models.ExtractedContact{Name: "Christine"}}
	DecideIntent(claim, "Her name is Christine.", ask).StripAnswerGoal(ec)
	if ec.Intention != "" || ec.Contact == nil {
		t.Fatalf("the goal goes, the person stays: %+v", ec)
	}
}

func TestJudgeGoalRelation(t *testing.T) {
	cases := map[string]GoalRelation{
		`{"relation": "different"}`:  GoalDifferent,
		`{"relation": "Refinement"}`: GoalRefinement,
		`{"relation": "nonsense"}`:   GoalSame,
		`not json`:                   GoalSame,
	}
	for out, want := range cases {
		if got := JudgeGoalRelation(context.Background(), &rewriteLLM{out: out}, "a", "b"); got != want {
			t.Errorf("%s: got %s, want %s", out, got, want)
		}
	}
	if JudgeGoalRelation(context.Background(), (&rewriteLLM{err: errors.New("down")}), "a", "b") != GoalSame {
		t.Fatal("a failed judgement must not report a shift")
	}
	if JudgeGoalRelation(context.Background(), nil, "a", "b") != GoalSame || JudgeGoalRelation(context.Background(), &rewriteLLM{out: `{"relation":"different"}`}, "", "b") != GoalSame {
		t.Fatal("nothing to compare, no shift")
	}
}
