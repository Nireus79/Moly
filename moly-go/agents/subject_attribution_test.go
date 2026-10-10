package agents

import (
	"strings"
	"testing"

	"moly/models"
)

func TestParseSubjectAttributionReadsContactSubject(t *testing.T) {
	got := parseSubjectAttribution(`{"subject": "Christine_sub", "confidence": 0.9, "is_clear": true, "reasoning": "about her"}`)
	if got.Subject != "Christine_sub" || !got.IsClear || got.Confidence != 0.9 {
		t.Fatalf("contact subject lost: %+v", got)
	}
}

func TestParseSubjectAttributionAcceptsFencedJSON(t *testing.T) {
	raw := "Here is the answer:\n```json\n{\"subject\": \"user\", \"confidence\": 0.8, \"is_clear\": true}\n```"
	if got := parseSubjectAttribution(raw); got.Subject != "user" {
		t.Fatalf("fenced JSON not parsed: %+v", got)
	}
}

// An unreadable answer must not be recorded as the user's trait.
func TestUnreadableSubjectIsUnknownNotUser(t *testing.T) {
	for _, raw := range []string{"", "I cannot tell", `{"subject": "user"`, `{"confidence": 0.9}`} {
		if got := parseSubjectAttribution(raw); got.Subject != "" {
			t.Fatalf("%q should give unknown subject, got %q", raw, got.Subject)
		}
	}
}

func TestParseSubjectAttributionClampsConfidence(t *testing.T) {
	if got := parseSubjectAttribution(`{"subject": "user", "confidence": 7}`); got.Confidence != 1 {
		t.Fatalf("confidence not clamped: %v", got.Confidence)
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
