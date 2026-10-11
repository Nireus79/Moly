package agents

import (
	"context"
	"testing"
)

// The model answers per question number; a wrong count of entries cannot discard the judgement (found live: two booleans for
// one question), and a missing number means "not answered".
func TestPendingJudgeReadsPerQuestionNumber(t *testing.T) {
	judge := func(content string, n int) []bool {
		items := make([]string, n)
		for i := range items {
			items[i] = "item"
		}
		return JudgePendingAnswered(context.Background(), fixedLLM{content: content}, "a message", items)
	}
	if got := judge(`{"1": true}`, 1); !got[0] {
		t.Fatalf("item 1 answered, got %v", got)
	}
	if got := judge(`{"1": false, "2": true, "3": false}`, 3); got[0] || !got[1] || got[2] {
		t.Fatalf("only item 2 answered, got %v", got)
	}
	if got := judge(`{"1": true, "2": true, "9": true}`, 1); !got[0] || len(got) != 1 {
		t.Fatalf("an entry for a question that does not exist is ignored, got %v", got)
	}
	if got := judge(`{"2": true}`, 2); got[0] || !got[1] {
		t.Fatalf("a missing number is not answered, got %v", got)
	}
	if got := judge(`{"1": "true"}`, 1); !got[0] {
		t.Fatalf("a string boolean is read, got %v", got)
	}
	if got := judge("```json\n{\"1\": true}\n```", 1); !got[0] {
		t.Fatalf("a fenced answer is read, got %v", got)
	}
	for _, c := range []string{`{}`, "no idea", `{"1": false}`, `[true]`} {
		if got := judge(c, 2); got[0] || got[1] {
			t.Fatalf("%q closes nothing, got %v", c, got)
		}
	}
}
