package models

import (
	"strings"
	"testing"
)

func TestExchangeLeavesOutTheCurrentMessage(t *testing.T) {
	chrono := []Message{
		{Role: "user", Content: "She is my manager."},
		{Role: "assistant", Content: "Would you like to focus on a fairer distribution?"},
		{Role: "user", Content: "Yes"},
	}
	got := ExchangeBeforeChronological(chrono, "Yes", 4)
	if got != "User: She is my manager.\nMoly: Would you like to focus on a fairer distribution?\n" {
		t.Fatalf("chronological: %q", got)
	}
	currentFirst := []Message{{Role: "user", Content: "Yes"}, chrono[0], chrono[1]}
	if got2 := ExchangeBeforeCurrentFirst(currentFirst, 4); got2 != got {
		t.Fatalf("current-first gives the same exchange, got %q", got2)
	}
	if only := ExchangeBeforeChronological(chrono[2:], "Yes", 4); only != "" {
		t.Fatalf("a first message has no exchange, got %q", only)
	}
	if limited := ExchangeBeforeChronological(chrono, "Yes", 1); !strings.HasPrefix(limited, "Moly:") || strings.Contains(limited, "User:") {
		t.Fatalf("only the last message is kept, got %q", limited)
	}
	if WithExchange("") != "" || !strings.Contains(WithExchange(got), "Would you like to focus") {
		t.Fatal("the prompt block holds the exchange, or nothing")
	}
}

func TestDialoguePacketTextHoldsWhatIsKnownAndIsEmptyWithoutIt(t *testing.T) {
	var none *DialoguePacket
	if none.Text() != "" || (&DialoguePacket{}).Text() != "" {
		t.Fatal("nothing known: nothing written")
	}
	p := &DialoguePacket{
		Summary: "The user prepares a message to her manager Dana; her colleague Mark covers for her and must not be named.",
		Goal:    "Ask for a fairer split of tasks", People: []string{"Dana (the user's manager)", "Mark (a colleague)"},
		Open: []string{"How formal should it be?"}, MolyLast: "Here is a draft.",
	}
	got := p.Text()
	for _, want := range []string{"conversation between the user and Moly", "Mark covers for her", "Ask for a fairer split", "Dana (the user's manager)", "How formal should it be?", "Moly's last message: Here is a draft."} {
		if !strings.Contains(got, want) {
			t.Fatalf("the packet text lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "before that") {
		t.Fatal("the user's earlier message is only written when the summary lags")
	}
	if strings.Index(got, "\nWhat Moly remembers") > strings.Index(got, "\nMoly's last message") {
		t.Fatal("the memory comes first and the flow last")
	}
}

func TestLastMolyMessageAndTheUserBeforeIt(t *testing.T) {
	older := []Message{
		{Role: "user", Content: "first"}, {Role: "assistant", Content: "reply one"},
		{Role: "user", Content: "second"}, {Role: "assistant", Content: "reply two"},
	}
	moly, before := LastMolyMessage(older)
	if moly != "reply two" || before != "second" {
		t.Fatalf("got %q / %q", moly, before)
	}
	if m, b := LastMolyMessage([]Message{{Role: "user", Content: "only"}}); m != "" || b != "" {
		t.Fatalf("no Moly message yet, got %q / %q", m, b)
	}
	current := append(append([]Message{}, older...), Message{Role: "user", Content: "Yes"})
	if got := MessagesBefore(current, "Yes"); len(got) != 4 {
		t.Fatalf("the current message is left out (oldest first), got %d", len(got))
	}
	first := append([]Message{{Role: "user", Content: "Yes"}}, older...)
	if got := MessagesBefore(first, "Yes"); len(got) != 4 || got[0].Content != "first" {
		t.Fatalf("the current message is left out (current first), got %d", len(got))
	}
}

func TestExchangeKeepsMolysDraftLongerThanTheUsersMessage(t *testing.T) {
	long := strings.Repeat("a", 600)
	got := ExchangeBeforeChronological([]Message{{Role: "assistant", Content: long}, {Role: "user", Content: long}}, "x", 4)
	if !strings.Contains(got, "Moly: "+long) || strings.Contains(got, "User: "+long) {
		t.Fatalf("Moly's 600 characters are kept, the user's are cut at 300, got %d chars", len(got))
	}
}
