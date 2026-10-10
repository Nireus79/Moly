package agents

import (
	"context"
	"log"
	"strings"

	"moly/tools"
)

// PersonRef is what the words taken as a person really refer to.
type PersonRef struct {
	Kind string // "known" (a person already saved), "new" (a new person) or "none" (not a person)
	Name string // the known person's name, when Kind is "known"
}

const (
	refKnown = "known"
	refNew   = "new"
	refNone  = "none"
)

// JudgePersonReference asks the model who the words the extractor took as a person refer to, given the people the user
// already has. A bare pronoun ("she") usually points to a known person; words about something else ("a bit too formal")
// are not a person at all. It runs only when a new unnamed person is about to be saved. Anything unreadable or failed
// counts as a new person, so a failure never hides someone.
func JudgePersonReference(ctx context.Context, llm tools.LLMProvider, label, message string, known []string) PersonRef {
	newPerson := PersonRef{Kind: refNew}
	if llm == nil || strings.TrimSpace(label) == "" {
		return newPerson
	}
	people := "none"
	if len(known) > 0 {
		people = strings.Join(known, ", ")
	}
	req := &tools.LLMRequest{
		SystemPrompt: "You decide who a phrase refers to.",
		UserPrompt: "The user wrote: \"" + message + "\"\nPhrase taken from it: \"" + label + "\"\nPeople the user already has: " + people + "\n\n" +
			"Does the phrase refer to one of the people the user already has (for example a pronoun that points to them), " +
			"to a new third person (a name, a role, a relationship, or a description of who someone is), " +
			"or is it not a person at all (words about a text, a tone, a feeling or a situation)?\n" +
			"Respond with ONLY JSON: {\"refersTo\": \"known|new|none\", \"name\": \"the known person's name, or empty\"}",
		MaxTokens:   40,
		Temperature: 0.1,
	}
	resp, err := llm.Call(ctx, req)
	if err != nil {
		log.Printf("[Identity] person reference judgement failed: %v (treated as a new person)", err)
		return newPerson
	}
	var out struct {
		RefersTo string `json:"refersTo"`
		Name     string `json:"name"`
	}
	if err := tools.SafeJSONParse("PersonReference", []byte(resp.Content), &out); err != nil {
		return newPerson
	}
	switch strings.ToLower(strings.TrimSpace(out.RefersTo)) {
	case refNone:
		return PersonRef{Kind: refNone}
	case refKnown:
		for _, k := range known {
			if strings.EqualFold(strings.TrimSpace(out.Name), k) {
				return PersonRef{Kind: refKnown, Name: k}
			}
		}
	}
	return newPerson
}
