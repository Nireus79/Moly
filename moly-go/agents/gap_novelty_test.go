package agents

import (
	"context"
	"errors"
	"testing"

	"moly/models"
	"moly/tools"
)

type novelLLM struct {
	out string
	err error
}

func (n novelLLM) Call(ctx context.Context, req *tools.LLMRequest) (*tools.LLMResponse, error) {
	return &tools.LLMResponse{Content: n.out}, n.err
}

func TestDropRepeatedGapsKeepsEverythingWhenJudgementFails(t *testing.T) {
	gaps := []tools.Gap{{Type: "a", Description: "Q1?"}, {Type: "b", Description: "Q2?"}}
	d := &models.DialoguePacket{MolyLast: "Did you talk to him?"}
	for _, l := range []novelLLM{{out: "no idea"}, {err: errors.New("down")}} {
		if got := DropRepeatedGaps(context.Background(), l, gaps, d, nil); len(got) != 2 {
			t.Fatalf("an unreadable or failed judgement drops nothing, got %d", len(got))
		}
	}
}

func TestDropRepeatedGapsDropsOnlyTheRepeat(t *testing.T) {
	gaps := []tools.Gap{{Type: "a", Description: "Q1?"}, {Type: "b", Description: "Q2?"}}
	d := &models.DialoguePacket{MolyLast: "Did you talk to him?"}
	got := DropRepeatedGaps(context.Background(), novelLLM{out: "```json\n{\"1\": true, \"2\": false}\n```"}, gaps, d, nil)
	if len(got) != 1 || got[0].Type != "b" {
		t.Fatalf("only the repeat is dropped, got %+v", got)
	}
}

func TestDropRepeatedGapsAsksNothingOnTheFirstMessage(t *testing.T) {
	gaps := []tools.Gap{{Type: "a", Description: "Q1?"}}
	if got := DropRepeatedGaps(context.Background(), novelLLM{out: `{"1": true}`}, gaps, &models.DialoguePacket{}, nil); len(got) != 1 {
		t.Fatalf("nothing was asked before, so nothing repeats, got %d", len(got))
	}
}
