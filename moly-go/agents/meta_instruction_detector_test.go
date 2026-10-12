package agents

import (
	"context"
	"strings"
	"testing"
)

// The meta-instruction detector is the model alone: no grammar pattern or phrase list is tried before it.
func TestMetaInstructionIsWhatTheModelSays(t *testing.T) {
	llm := &recordingLLM{out: `{"isMetaInstruction": true, "type": "focus", "confidence": 0.9, "targetTopic": "my sister's wedding"}`}
	got := NewMetaInstructionDetector(llm).DetectWithExchange(context.Background(), "Let's focus on my sister's wedding.", "Moly: What matters most?")
	if got == nil || got.Type != "focus" || got.TargetTopic != "my sister's wedding" || got.Source != "llm" {
		t.Fatalf("got %+v", got)
	}
	if !strings.Contains(llm.prompt, "What matters most?") {
		t.Errorf("the model sees the recent exchange, got prompt:\n%s", llm.prompt)
	}
}

func TestMetaInstructionNeedsTheModelToAgree(t *testing.T) {
	no := &recordingLLM{out: `{"isMetaInstruction": false, "type": "none", "confidence": 0.9}`}
	// "Lace is my focus" and "you are Moly" used to be caught by a grammar pattern and a phrase check before the model was asked.
	for _, msg := range []string{"Lace is my focus", "You are Moly", "Her name is Dana."} {
		if got := NewMetaInstructionDetector(no).Detect(context.Background(), msg); got != nil {
			t.Errorf("%q: the model said no, got %+v", msg, got)
		}
	}
	unreadable := &recordingLLM{out: "I think so?"}
	if got := NewMetaInstructionDetector(unreadable).Detect(context.Background(), "Let's focus on X"); got != nil {
		t.Errorf("an unreadable answer is no meta-instruction, got %+v", got)
	}
	if got := NewMetaInstructionDetector(nil).Detect(context.Background(), "Let's focus on X"); got != nil {
		t.Errorf("without a model there is no meta-instruction, got %+v", got)
	}
	fresh := &recordingLLM{out: `{"isMetaInstruction": true, "type": "focus"}`}
	if got := NewMetaInstructionDetector(fresh).Detect(context.Background(), "  "); got != nil || fresh.prompt != "" {
		t.Errorf("an empty message is never sent to the model, got %+v prompt %q", got, fresh.prompt)
	}
}
