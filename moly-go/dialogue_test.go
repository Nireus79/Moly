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

func TestMemoryFactsReachTheDialogueWithTheirOwner(t *testing.T) {
	snap := models.Memory{
		People: []models.MemoryPerson{
			{ID: "p1", Name: "Rosa", Traits: []string{"will handle the client call"}},
			{ID: "p2", Label: "supplier"},
		},
		Facts: []models.MemoryFact{
			{Person: "p1", Fact: "says the deadline moves to Wednesday"},
			{Person: "user", Fact: "wants the message formal"},
			{Person: "p2", Fact: "was late"},
		},
	}
	lines := memoryFactLines(snap)
	want := []string{"Rosa: will handle the client call", "Rosa: says the deadline moves to Wednesday", "The user: wants the message formal", "the supplier: was late"}
	if len(lines) != len(want) {
		t.Fatalf("got %v", lines)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Fatalf("line %d: got %q, want %q", i, lines[i], want[i])
		}
	}
	p := &models.DialoguePacket{Facts: lines}
	if !strings.Contains(p.Text(), "Facts the user gave in this conversation") || !strings.Contains(p.Text(), "Wednesday") {
		t.Fatalf("the packet carries the facts:\n%s", p.Text())
	}
}
