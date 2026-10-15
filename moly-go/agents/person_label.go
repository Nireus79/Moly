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
// JudgePersonReference asks the model who the words the extractor took as a person (a label, or a name the extractor believes
// is one) refer to, given the people this conversation has already brought up. A bare pronoun ("she") usually points to a person already brought up; words about something else ("a bit too formal")
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
		UserPrompt: models.WithExchange(exchange) + "The user wrote: \"" + message + "\"\nPhrase taken from it: \"" + label + "\"\nPeople already brought up in this conversation: " + people + "\n\n" +
			"Decide what the phrase is:\n" +
			"- \"known\": it points to one of the people already brought up in this conversation (a pronoun, their name, the role the list gives them, or the description the list gives them: a phrase for a role points to the listed person whose role in the list is that role, and a phrase that describes a person in the way the list describes a listed person points to that person). Give that person's name only, without the role. A person who is not in the list is never \"known\".\n" +
			"- \"new\": it names ONE person who is not in the list: a name, or a word for any role or relationship the person has to the user. A role is a new person unless the list gives exactly that role to one of the listed people; a different role, or a name with no role, never matches a listed person, EXCEPT when the phrase is a role word (never a name) and the user's own sentence says that a pronoun already pointing to a listed person IS that role (the user is correcting how the person is related to them): then the phrase is that listed person, \"known\". A name is \"new\" unless it is a listed person's name, even when the sentence also gives that person's role. A pronoun (she, he, her, him, they) is never \"new\": it either points to someone in the list (\"known\") or to nobody (\"none\").\n" +
			"- \"none\": it is not one person: a group (family, team, everyone, people), an organization or institution (a bank, a company, the council, an office), a pronoun with nobody to point to, or words about a text, a tone, a feeling or a situation.\n" +
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
			// a listed person may carry their role: "Dana (the user's boss)"; the name is what comes before it
			bare := k
			if i := strings.Index(k, " ("); i > 0 {
				bare = k[:i]
			}
			given := strings.TrimSpace(out.Name)
			if strings.EqualFold(given, k) || strings.EqualFold(given, bare) {
				return PersonRef{Kind: refKnown, Name: bare}
			}
		}
		if len(known) == 0 {
			// "known" while nobody has been brought up cannot be right (a weak model says it for "my boss"): the words name a new
			// person, and its name is asked. Making the user confirm a person who was never mentioned is worse.
			log.Printf("[Identity] person reference judgement said known with nobody in the list: taken as a new person")
			return newPerson
		}
		// "known" but not any listed person: the model did not say who. Moly does not guess; the reply asks.
		log.Printf("[Identity] person reference judgement named nobody in the list (not saved; the reply asks)")
		return unsure
	}
	return newPerson
}
