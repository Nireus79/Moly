package agents

import (
	"context"
	"testing"

	"moly/tools"
)

type fixedAnswerLLM string

func (f fixedAnswerLLM) Call(ctx context.Context, req *tools.LLMRequest) (*tools.LLMResponse, error) {
	return &tools.LLMResponse{Content: string(f)}, nil
}

func TestAKnownAnswerNamingThePhraseItselfIsANewPerson(t *testing.T) {
	known := []string{"sister (name not given yet)"}
	got := JudgePersonReference(context.Background(), fixedAnswerLLM(`{"refersTo": "known", "name": "Lena"}`), "Lena", "Her name is Lena.", "", known)
	if got.Kind != refNew {
		t.Fatalf("a name the list does not have is a new person, got %+v", got)
	}
	got = JudgePersonReference(context.Background(), fixedAnswerLLM(`{"refersTo": "known", "name": "Someone Else"}`), "Lena", "Her name is Lena.", "", known)
	if got.Kind != refUnsure {
		t.Fatalf("a different unlisted name stays unsure, got %+v", got)
	}
}
