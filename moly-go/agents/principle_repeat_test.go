package agents

import (
	"testing"

	"moly/models"
)

func history(assistant ...string) []models.Message {
	var h []models.Message
	for _, a := range assistant {
		h = append(h, models.Message{Role: "user", Content: "..."}, models.Message{Role: "assistant", Content: a})
	}
	return h
}

// The same words are a repeat without asking a model; other words are judged by the model; a question never asked is not a repeat.
func TestAPrincipleQuestionAskedBeforeIsNotAskedAgain(t *testing.T) {
	q := "How would she feel about hearing from you?"
	llm := &cannedLLM{answer: `{"1": true}`}
	ca := &conversationAgent{llmClient: llm}
	if !ca.principleAskedBefore(q, history("Thanks. "+q), nil) || len(llm.seen) != 0 {
		t.Fatalf("the very words are a repeat, with no model call (%d calls)", len(llm.seen))
	}
	if !ca.principleAskedBefore(q, history("Would she be glad to hear from you?"), nil) || len(llm.seen) != 1 {
		t.Fatal("the model says it is the same question in other words")
	}
	different := &conversationAgent{llmClient: &cannedLLM{answer: `{"1": false}`}}
	if different.principleAskedBefore(q, history("What drew you to her profile?"), nil) {
		t.Fatal("a different question is not a repeat")
	}
	if (&conversationAgent{llmClient: llm}).principleAskedBefore(q, nil, nil) {
		t.Fatal("nothing was asked before")
	}
}
