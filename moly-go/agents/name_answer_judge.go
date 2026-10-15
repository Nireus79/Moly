package agents

import (
	"context"
	"log"
	"strings"

	"moly/models"
	"moly/tools"
)

// NameOutcome says what a message did to the name questions that are open.
type NameOutcome struct {
	Kind string // nameGiven, nameDeclined or nameNotAnswered
	Name string // nameGiven: the name
	For  string // nameGiven: which waiting person it is for (one of the labels given), "" when only one waits
}

const (
	nameGiven       = "given"
	nameDeclined    = "declined"
	nameNotAnswered = "other"
)

// JudgeNameAnswer asks the model whether a message answers the name question(s) Moly has open: it gives a name, it declines to
// give one, or it is about something else. Only the first two close a question (found live, 2026-10-15: every message after a
// name question closed it, so a reply about something else made the person "nameless" for good). Anything unreadable or failed is
// "not answered": the question stays open, and nothing is lost.
func JudgeNameAnswer(ctx context.Context, llm tools.LLMProvider, exchange, message string, waiting []string) NameOutcome {
	other := NameOutcome{Kind: nameNotAnswered}
	if llm == nil || len(waiting) == 0 || strings.TrimSpace(message) == "" {
		return other
	}
	req := &tools.LLMRequest{
		SystemPrompt: "You decide whether a message answers a question about a person's name.",
		UserPrompt: models.WithExchange(exchange) + "Moly asked the user for the name of: " + strings.Join(waiting, "; ") + "\n" +
			"The user's message: \"" + message + "\"\n\n" +
			"Decide:\n" +
			"- \"given\": the message gives the name of one of those people: a real name, not a pronoun, a role or a description. Give the name, and say which of the listed people it is for (copy the listed words).\n" +
			"- \"declined\": the user SAYS they will not or cannot give the name (they refuse, or say they do not know it). A message that merely contains no name is not a decline.\n" +
			"- \"other\": anything else: the message is about something else, asks a question back, or describes the person without naming them.\n" +
			"Respond with ONLY JSON: {\"outcome\": \"given|declined|other\", \"name\": \"the name, or empty\", \"for\": \"the listed words of the person it is for, or empty\"}",
		MaxTokens:   80,
		Temperature: 0.1,
	}
	resp, err := llm.Call(ctx, req)
	if err != nil {
		log.Printf("[Identity] name answer judgement failed: %v (the question stays open)", err)
		return other
	}
	var out struct {
		Outcome string `json:"outcome"`
		Name    string `json:"name"`
		For     string `json:"for"`
	}
	if err := tools.SafeJSONParse("NameAnswer", []byte(resp.Content), &out); err != nil {
		log.Printf("[Identity] name answer judgement unreadable (the question stays open)")
		return other
	}
	switch strings.ToLower(strings.TrimSpace(out.Outcome)) {
	case nameDeclined:
		return NameOutcome{Kind: nameDeclined}
	case nameGiven:
		name := strings.TrimSpace(out.Name)
		if name == "" {
			return other
		}
		target := ""
		for _, w := range waiting {
			if strings.EqualFold(strings.TrimSpace(out.For), w) {
				target = w
			}
		}
		if len(waiting) > 1 && target == "" {
			return other // several wait and the model did not say which: Moly does not guess
		}
		return NameOutcome{Kind: nameGiven, Name: name, For: target}
	}
	return other
}
