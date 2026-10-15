package agents

import (
	"strings"
	"testing"

	"moly/models"
)

// The subject of an entity comes from the extraction form, with no model call (plan step 0b, 2026-10-15): the user's goals, values,
// style and traits are about "user"; a trait of a person carries its person in its value and no separate subject; a person and a
// concern have none, as before.
func TestEntitiesCarryTheirSubjectFromTheForm(t *testing.T) {
	ec := &models.ExtractedContext{
		Intention:              "ask for a raise",
		Goals:                  []string{"keep the job"},
		UserValues:             []string{"honesty"},
		Style:                  &models.ExtractedStyle{Style: "direct", Confidence: 0.8},
		UserCharacteristics:    []string{"worried"},
		ContactCharacteristics: map[string][]string{"Dana": {"busy"}},
		IntentionPrinciples:    []string{"fairness"},
		Contact:                &models.ExtractedContact{Name: "Dana", Confidence: 0.9},
	}
	bySubject := map[string]string{}
	for _, e := range entitiesFromContext(ec) {
		bySubject[e.Type] = e.Subject
	}
	for _, typ := range []string{"goal", "goal_component", "value", "style", "characteristic"} {
		if bySubject[typ] != "user" {
			t.Errorf("%s should be about the user, got %q", typ, bySubject[typ])
		}
	}
	for _, typ := range []string{"contact", "contact_characteristic", "concern"} {
		if bySubject[typ] != "" {
			t.Errorf("%s keeps no separate subject, got %q", typ, bySubject[typ])
		}
	}
}

// No entity is marked ambiguous by the extraction: an unreadable model answer used to do that (found live, 2026-10-14), and one
// ambiguous entity made Layer 6 call the whole request ambiguous.
func TestNoEntityIsAmbiguousFromTheForm(t *testing.T) {
	ec := &models.ExtractedContext{UserCharacteristics: []string{"worried", "long-tenured"}, Intention: "keep the job"}
	for _, e := range entitiesFromContext(ec) {
		if e.IsAmbiguous {
			t.Errorf("%s %q is ambiguous", e.Type, e.Value)
		}
	}
}

func TestAccumulatedContextKeepsContactTraitsOutOfUserProfile(t *testing.T) {
	entities := []models.ExtractedEntity{
		{Type: "characteristic", Value: "dominant", Subject: "user"},
		{Type: "characteristic", Value: "submissive", Subject: "Christine_sub"},
		{Type: "characteristic", Value: "playful"}, // unknown subject: not attributed to anyone
	}
	out := formatAccumulatedEntities(entities)

	if !strings.Contains(out, "They've described themselves as: dominant\n") {
		t.Fatalf("user trait missing: %q", out)
	}
	if strings.Contains(out, "They've described themselves as: dominant, submissive") ||
		strings.Contains(out, "They've described themselves as: submissive") {
		t.Fatalf("contact trait presented as the user's: %q", out)
	}
	if !strings.Contains(out, "Traits of Christine_sub (NOT the user): submissive") {
		t.Fatalf("contact trait not labelled: %q", out)
	}
	if strings.Contains(out, "playful") {
		t.Fatalf("unknown-subject trait should be left out: %q", out)
	}
}

// What the form says about the user reaches the reply as the user's, and what it says about a person as that person's: the reply
// model is told "They've described themselves as ..." only for the user's own traits.
func TestUserTraitsFromTheFormReachTheReplyAsTheirs(t *testing.T) {
	ec := &models.ExtractedContext{
		UserCharacteristics:    []string{"worried"},
		ContactCharacteristics: map[string][]string{"Dana": {"busy"}},
	}
	out := formatAccumulatedEntities(entitiesFromContext(ec))
	if !strings.Contains(out, "They've described themselves as: worried") {
		t.Fatalf("the user's trait must be told as theirs, got %q", out)
	}
	if strings.Contains(out, "described themselves as: busy") || strings.Contains(out, "worried, busy") {
		t.Fatalf("a person's trait must not be told as the user's, got %q", out)
	}
}
