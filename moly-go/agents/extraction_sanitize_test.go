package agents

import (
	"strings"
	"testing"

	"moly/models"
)

func TestSanitizeCleansTraitLists(t *testing.T) {
	got := cleanList([]string{" Shy ", "shy", "USER|likes hiking|0.8", "see https://example.org/x", "a very long phrase that is clearly a sentence and not a trait", "", "Dana", "\"quiet\"."}, "Dana")
	want := []string{"Shy", "likes hiking", "quiet"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSanitizeContactKeepsAPersonByLabelWhenTheNameIsASentence(t *testing.T) {
	c := sanitizeContact(&models.ExtractedContact{Name: "she is the woman that I met last week at the club?", Label: "woman", NameKnown: true, Confidence: 1.7, Traits: []string{"kind", "Kind"}})
	if c == nil || c.Name != "woman" || c.NameKnown || c.Confidence != 1 || len(c.Traits) != 1 {
		t.Fatalf("got %+v", c)
	}
	if sanitizeContact(&models.ExtractedContact{Name: "see https://example.org/u/1"}) != nil {
		t.Fatal("an address alone names no one")
	}
	if c := sanitizeContact(&models.ExtractedContact{Name: "Dana", Role: "the person who has been my manager for six years now"}); c == nil || c.Role != "" || c.Name != "Dana" {
		t.Fatalf("a sentence is no role, got %+v", c)
	}
}

func TestSanitizeExtractionChecksEveryField(t *testing.T) {
	ec := &models.ExtractedContext{
		Contact:                &models.ExtractedContact{Name: "Dana", Traits: []string{"warm"}},
		OtherContacts:          []*models.ExtractedContact{{Name: "https://example.org/x"}},
		UserCharacteristics:    []string{"tall", "TALL"},
		ContactCharacteristics: map[string][]string{"Dana": {"warm", "Dana"}, "": {"x"}},
		Relations:              []models.ExtractedRelation{{Person: "Dana", Relation: "sister", Of: "user"}, {Person: "", Relation: "x"}, {Person: "Dana", Relation: "one two three four five six"}},
	}
	sanitizeExtraction(ec)
	if len(ec.OtherContacts) != 0 || len(ec.UserCharacteristics) != 1 || len(ec.ContactCharacteristics) != 1 || len(ec.ContactCharacteristics["Dana"]) != 1 || len(ec.Relations) != 1 {
		t.Fatalf("got %+v", ec)
	}
}
