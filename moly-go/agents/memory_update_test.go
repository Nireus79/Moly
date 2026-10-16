package agents

import (
	"testing"

	"moly/models"
)

func TestGuardNextRefusesAQuestionThatSaysNothingOrRepeatsATopic(t *testing.T) {
	mem := models.Memory{
		People: []models.MemoryPerson{{ID: "p1", Name: "Kyle", NameStatus: "named"}, {ID: "p2", Label: "supplier", NameStatus: "declined"}, {ID: "p3", Label: "father", NameStatus: "unnamed"}},
		Questions: []models.MemoryQuestion{
			{ID: "q1", Kind: "gap", Topic: "p1 Involvement", State: models.QuestionAnswered},
			{ID: "q2", Kind: "gap", Topic: "p1 delay", State: models.QuestionSkipped},
			{ID: "q3", Kind: "gap", Topic: "p1 hours", State: models.QuestionSkipped},
			{ID: "q4", Kind: "gap", Topic: "p1 hours", State: models.QuestionSkipped},
		},
	}
	cases := []struct {
		name string
		next MemoryNext
		want string
	}{
		{"no stated effect", MemoryNext{Action: "ask", Kind: "gap", Topic: "p1 pay"}, "answer"},
		{"topic already answered, other spacing and case", MemoryNext{Action: "ask", Kind: "gap", Topic: "p1  involvement", Changes: "x"}, "answer"},
		{"topic skipped once may return", MemoryNext{Action: "ask", Kind: "gap", Topic: "p1 delay", Changes: "x"}, "ask"},
		{"topic skipped twice does not", MemoryNext{Action: "ask", Kind: "gap", Topic: "p1 hours", Changes: "x"}, "answer"},
		{"a name for a named person", MemoryNext{Action: "ask", Kind: "name", About: "p1", Topic: "p1 name", Changes: "x"}, "answer"},
		{"a name for a person who refused", MemoryNext{Action: "ask", Kind: "name", About: "p2", Topic: "p2 name", Changes: "x"}, "answer"},
		{"a name for a person without one", MemoryNext{Action: "ask", Kind: "name", About: "p3", Topic: "p3 name", Changes: "x"}, "ask"},
		{"a confirmation that confirms nothing", MemoryNext{Action: "confirm"}, "answer"},
		{"an answer stays", MemoryNext{Action: "answer"}, "answer"},
		{"result now stays", MemoryNext{Action: "result_now"}, "result_now"},
	}
	for _, c := range cases {
		if got, _ := GuardNext(mem, c.next); got.Action != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got.Action, c.want)
		}
	}
}
