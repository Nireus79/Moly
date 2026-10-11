package agents

import (
	"context"
	"testing"
)

// Found live: the model answered {"answered": [false, true]} for ONE question and the whole judgement was thrown away.
// The model now names the numbers of the items it answers; a wrong count cannot matter.
func TestPendingJudgeReadsNumbers(t *testing.T) {
	judge := func(content string, n int) []bool {
		items := make([]string, n)
		for i := range items {
			items[i] = "item"
		}
		return JudgePendingAnswered(context.Background(), fixedLLM{content: content}, "a message", items)
	}
	if got := judge(`{"answered": [1]}`, 1); !got[0] {
		t.Fatalf("item 1 answered, got %v", got)
	}
	if got := judge(`{"answered": [2]}`, 3); got[0] || !got[1] || got[2] {
		t.Fatalf("only item 2 answered, got %v", got)
	}
	if got := judge(`{"answered": [1, 7, 0, -1]}`, 1); !got[0] || len(got) != 1 {
		t.Fatalf("numbers out of range are ignored, got %v", got)
	}
	if got := judge("```json\n{\"answered\": [1]}\n```", 1); !got[0] {
		t.Fatalf("a fenced answer is read, got %v", got)
	}
	for _, c := range []string{`{"answered": []}`, "no idea", `{"answered": ["x"]}`} {
		if got := judge(c, 2); got[0] || got[1] {
			t.Fatalf("%q closes nothing, got %v", c, got)
		}
	}
}
