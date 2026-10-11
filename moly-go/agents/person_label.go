package agents

import (
	"context"
	"log"
	"strings"

	"moly/models"
	"moly/tools"
)

// PersonRef is what the words taken as a person really refer to.
type PersonRef struct {
	Kind string // "known" (a person already saved), "new" (a new person), "none" (not a person) or "unsure" (not judged)
	Name string // the known person's name, when Kind is "known"
}

const (
	refKnown = "known"
	refNew   = "new"
	refNone  = "none"
	// refUnsure: the judgement failed or could not be read. Moly does not know who is meant, so it asks (doubt is not saved).
	refUnsure = "unsure"
)

// The prompt names what each answer means. The first wording ("a new third person ... or not a person at all") was answered
// "none" for "girlfriend" and "new" for "family" and for a pronoun with nobody to point to; 14 of 24 right on Claude haiku,
// 22 of 24 with this one (live D11: "a name for the family you mentioned").
// JudgePersonReference asks the model who the words the extractor took as a person refer to, given the people the user
// already has. A bare pronoun ("she") usually points to a known person; words about something else ("a bit too formal")
// are not a person at all. It runs only when a new unnamed person is about to be saved. Anything unreadable or failed
// is "unsure": the person is not saved, and the reply asks who is meant (found live: a failed judgement saved the pronoun
// "her" as a second person).
func JudgePersonReference(ctx context.Context, llm tools.LLMProvider, label, message, exchange string, known []string) PersonRef {
	newPerson := PersonRef{Kind: refNew}
	unsure := PersonRef{Kind: refUnsure}
	if llm == nil || strings.TrimSpace(label) == "" {
		return unsure
	}
	people := "none"
	if len(known) > 0 {
		people = strings.Join(known, ", ")
	}
	req := &tools.LLMRequest{
		SystemPrompt: "You decide who a phrase refers to.",
		UserPrompt: models.WithExchange(exchange) + "The user wrote: \"" + message + "\"\nPhrase taken from it: \"" + label + "\"\nPeople the user already has: " + people + "\n\n" +
			"Decide what the phrase is:\n" +
			"- \"known\": it points to one of the people the user already has (a pronoun such as she or he, or their name or role). Give that person's name.\n" +
			"- \"new\": it names ONE person the user does not have yet: a name, or a role or relationship such as girlfriend, boss, sister, neighbour.\n" +
			"- \"none\": it is not one person: a group (family, team, everyone, people), a pronoun with nobody to point to, or words about a text, a tone, a feeling or a situation.\n" +
			"Respond with ONLY JSON: {\"refersTo\": \"known|new|none\", \"name\": \"the known person's name, or empty\"}",
		MaxTokens:   40,
		Temperature: 0.1,
	}
	resp, err := llm.Call(ctx, req)
	if err != nil {
		log.Printf("[Identity] person reference judgement failed: %v (not saved; the reply asks)", err)
		return unsure
	}
	var out struct {
		RefersTo string `json:"refersTo"`
		Name     string `json:"name"`
	}
	if err := tools.SafeJSONParse("PersonReference", []byte(resp.Content), &out); err != nil {
		log.Printf("[Identity] person reference judgement unreadable (not saved; the reply asks)")
		return unsure
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
