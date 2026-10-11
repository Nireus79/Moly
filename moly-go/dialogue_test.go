package main

import (
	"strings"
	"testing"

	"moly/models"
)

func TestFillDialogueAddsTheGoalAndTheOpenQuestions(t *testing.T) {
	a := &models.AnalysisContext{PrimaryGoal: "Ask for a fairer split of tasks", Dialogue: &models.DialoguePacket{MolyLast: "A draft."}}
	fillDialogue(a, []string{"q1", "q2", "q3", "q4"})
	p := a.Dialogue
	if p.Goal != "Ask for a fairer split of tasks" || p.MolyLast != "A draft." {
		t.Fatalf("goal set, Moly's last message kept, got %+v", p)
	}
	if len(p.Open) != 3 {
		t.Fatalf("at most 3 open questions, got %v", p.Open)
	}
	if !strings.Contains(p.Text(), "Ask for a fairer split of tasks") {
		t.Fatal("the text carries the goal")
	}
	a2 := &models.AnalysisContext{PreviousGoal: "old goal"}
	fillDialogue(a2, nil)
	if a2.Dialogue.Goal != "old goal" {
		t.Fatalf("without a locked goal the previous goal is used, got %q", a2.Dialogue.Goal)
	}
	fillDialogue(nil, nil) // no panic
}
