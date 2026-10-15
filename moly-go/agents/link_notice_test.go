package agents

import (
	"context"
	"errors"
	"testing"

	"moly/tools"
)

type linkStubLLM struct {
	answer string
	err    error
	calls  int
}

func (s *linkStubLLM) Call(ctx context.Context, req *tools.LLMRequest) (*tools.LLMResponse, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	return &tools.LLMResponse{Content: s.answer}, nil
}

func TestMessageHasLink(t *testing.T) {
	yes := []string{
		"see https://example.org/a/b for it",
		"HTTP://EXAMPLE.ORG",
		"look at www.example.org please",
		"it is on example.org/users/42 somewhere",
		"ftp://files.example.org/x",
	}
	no := []string{
		"I met her at 5 p.m. and we talked.",
		"Her name is Dana, e.g. a friend.",
		"I saved the file notes.txt yesterday.",
		"3.5 percent is fine",
		"",
	}
	for _, m := range yes {
		if !MessageHasLink(m) {
			t.Errorf("a link was not found in %q", m)
		}
	}
	for _, m := range no {
		if MessageHasLink(m) {
			t.Errorf("a link was found in %q", m)
		}
	}
}

func TestLinkNotice(t *testing.T) {
	const withLink = "She posted this: https://example.org/p/1"
	t.Run("no link, no call", func(t *testing.T) {
		s := &linkStubLLM{answer: `{"needed": true, "notice": "x"}`}
		if got := LinkNotice(context.Background(), s, "no link here", "reply"); got != "" || s.calls != 0 {
			t.Fatalf("got %q after %d calls", got, s.calls)
		}
	})
	t.Run("the model writes the notice", func(t *testing.T) {
		s := &linkStubLLM{answer: `{"needed": true, "notice": "Moly does not read web pages."}`}
		if got := LinkNotice(context.Background(), s, withLink, "reply"); got != "Moly does not read web pages." {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("the reply already says it", func(t *testing.T) {
		s := &linkStubLLM{answer: `{"needed": false, "notice": ""}`}
		if got := LinkNotice(context.Background(), s, withLink, "reply"); got != "" {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("a model failure still informs the user", func(t *testing.T) {
		s := &linkStubLLM{err: errors.New("down")}
		if got := LinkNotice(context.Background(), s, withLink, "reply"); got != fallbackLinkNotice {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("an unreadable answer still informs the user", func(t *testing.T) {
		s := &linkStubLLM{answer: "not json at all"}
		if got := LinkNotice(context.Background(), s, withLink, "reply"); got != fallbackLinkNotice {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("a notice that repeats a link is replaced", func(t *testing.T) {
		s := &linkStubLLM{answer: `{"needed": true, "notice": "I will not open https://example.org/p/1"}`}
		if got := LinkNotice(context.Background(), s, withLink, "reply"); got != fallbackLinkNotice {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("no model at all", func(t *testing.T) {
		if got := LinkNotice(context.Background(), nil, withLink, "reply"); got != fallbackLinkNotice {
			t.Fatalf("got %q", got)
		}
	})
}

func TestWithLinkNotice(t *testing.T) {
	if got := WithLinkNotice("", "reply"); got != "reply" {
		t.Fatalf("got %q", got)
	}
	if got := WithLinkNotice("note", "reply"); got != "note\n\nreply" {
		t.Fatalf("got %q", got)
	}
}
