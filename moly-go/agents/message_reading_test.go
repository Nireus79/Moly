package agents

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"moly/tools"
)

// cannedLLM answers every call with one text (or an error) and keeps what it was asked.
type cannedLLM struct {
	answer string
	err    error
	seen   []*tools.LLMRequest
}

func (c *cannedLLM) Call(_ context.Context, req *tools.LLMRequest) (*tools.LLMResponse, error) {
	c.seen = append(c.seen, req)
	if c.err != nil {
		return nil, c.err
	}
	return &tools.LLMResponse{Content: c.answer}, nil
}

// One call answers both judges: the intent with its three flags, and the meta-instruction with "contradictory".
func TestOneReadingGivesTheIntentAndTheMetaInstruction(t *testing.T) {
	llm := &cannedLLM{answer: "```json\n{\"intent\": {\"intent\": \"asking\", \"confidence\": 0.9, \"answersQuestion\": true, \"adjustsDraft\": false, \"aboutMoly\": false}, " +
		"\"meta\": {\"isMetaInstruction\": true, \"type\": \"instruction\", \"confidence\": 0.8, \"contradictory\": true, \"targetBehavior\": \"short\"}}\n```"}
	r := ReadMessage(context.Background(), llm, "Be formal and casual.", "Moly: Which tone?\n")
	if !r.OK || r.Intent.Intent != IntentAsk || !r.Intent.AnswersQuestion || r.Intent.AdjustsDraft || r.Intent.AboutMoly {
		t.Fatalf("intent: %+v", r)
	}
	if r.Meta == nil || r.Meta.Type != "instruction" || !r.Meta.Contradictory {
		t.Fatalf("meta: %+v", r.Meta)
	}
	if len(llm.seen) != 1 {
		t.Fatalf("one model call, got %d", len(llm.seen))
	}
}

// A message that is no instruction gives no meta-instruction, whatever the intent.
func TestAMessageThatIsNoInstructionGivesNoMetaInstruction(t *testing.T) {
	llm := &cannedLLM{answer: `{"intent": {"intent": "sharing", "confidence": 0.9}, "meta": {"isMetaInstruction": false, "type": "none"}}`}
	r := ReadMessage(context.Background(), llm, "I had a hard week.", "")
	if !r.OK || r.Meta != nil || r.Intent.Intent != IntentShare {
		t.Fatalf("%+v", r)
	}
}

// A failed call or an unreadable answer is not OK, so that the caller asks the two judges separately.
func TestAFailedOrUnreadableReadingFallsBackToTheSeparateJudges(t *testing.T) {
	for name, llm := range map[string]*cannedLLM{
		"error":     {err: fmt.Errorf("the provider did not answer")},
		"not json":  {answer: "I think it is a question."},
		"no intent": {answer: `{"meta": {"isMetaInstruction": false}}`},
	} {
		if r := ReadMessage(context.Background(), llm, "Hello", ""); r.OK {
			t.Errorf("%s: must not be OK, got %+v", name, r)
		}
	}
	if r := ReadMessage(context.Background(), nil, "Hello", ""); r.OK {
		t.Error("no model: not OK")
	}
	if r := ReadMessage(context.Background(), &cannedLLM{answer: "{}"}, "  ", ""); r.OK {
		t.Error("an empty message is not read")
	}
}

// The merged prompt is made of the two judges' own definitions, so that they cannot drift apart.
func TestTheMergedPromptHoldsBothJudgesDefinitions(t *testing.T) {
	p := readingSystemPrompt()
	for _, want := range []string{"Intent definitions:", `"greeting"`, `"aboutMoly" is true ONLY`, "Meta-instructions include:", `"contradictory" is true ONLY`, `"isMetaInstruction"`} {
		if !strings.Contains(p, want) {
			t.Errorf("the merged prompt lacks %q", want)
		}
	}
	if !strings.Contains(p, "in two ways") {
		t.Error("the scripted tests recognise the merged prompt by 'in two ways'")
	}
}
