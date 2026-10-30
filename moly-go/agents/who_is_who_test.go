package agents

import (
	"testing"

	"moly/models"
)

func TestWhoIsWhoAsksOnceWhenSeveralSpeakAndUserIsUnknown(t *testing.T) {
	mem := models.Memory{}
	got, why := GuardNext(mem, MemoryNext{Action: "answer", Voices: 3, UserIs: "unknown"})
	if got.Action != "ask" || got.Topic != whoIsWhoTopic || why == "" {
		t.Fatalf("expected the who-is-who question, got %+v (%s)", got, why)
	}
	for _, state := range []string{models.QuestionAnswered, models.QuestionDeclined, "open"} {
		mem.Questions = []models.MemoryQuestion{{Topic: whoIsWhoTopic, State: state}}
		if got, _ := GuardNext(mem, MemoryNext{Action: "answer", Voices: 3, UserIs: "unknown"}); got.Action != "answer" {
			t.Fatalf("state %s: asked again: %+v", state, got)
		}
	}
}

func TestWhoIsWhoNotForOneVoiceOrKnownUser(t *testing.T) {
	for _, n := range []MemoryNext{
		{Action: "answer", Voices: 1, UserIs: "unknown"},
		{Action: "answer", Voices: 0},
		{Action: "answer", Voices: 2, UserIs: "p1"},
		{Action: "answer", Voices: 2, UserIs: "none"},
	} {
		if got, _ := GuardNext(models.Memory{}, n); got.Action != "answer" {
			t.Fatalf("%+v: forced a question: %+v", n, got)
		}
	}
}
