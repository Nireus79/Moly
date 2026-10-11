package agents

import "testing"

func TestExtractionParsesJSONInAFenceOrASentence(t *testing.T) {
	ce := &ContextExtractor{}
	for _, in := range []string{
		"```json\n{\"intention\": \"write to Dana\", \"intentionBasis\": \"stated\", \"contact\": null}\n```",
		"Here is the extraction: {\"intention\": \"write to Dana\", \"intentionBasis\": \"stated\", \"contact\": null} Hope it helps.",
		"{\"intention\": \"write to Dana\", \"intentionBasis\": \"stated\", \"contact\": null}",
	} {
		got, err := ce.parseLLMExtraction(in)
		if err != nil || got.Intention != "write to Dana" {
			t.Fatalf("%q: err=%v got=%+v", in, err, got)
		}
	}
	if _, err := ce.parseLLMExtraction("no json at all"); err == nil {
		t.Fatal("text without JSON is still an error")
	}
}

func TestExtractionKeepsEveryPersonTheMessageMentions(t *testing.T) {
	ce := &ContextExtractor{}
	got, err := ce.parseLLMExtraction(`{"intention": "ask Dana to share the work with Mark", "intentionBasis": "stated",
		"contact": {"name": "Dana", "label": "manager", "role": "manager", "confidence": 0.9, "basis": "stated"},
		"otherContacts": [{"name": "Mark", "label": "colleague", "confidence": 0.9, "basis": "stated"},
		                  {"name": "dana", "confidence": 0.9}]}`)
	if err != nil {
		t.Fatal(err)
	}
	if got.Contact == nil || got.Contact.Name != "Dana" || len(got.OtherContacts) != 1 || got.OtherContacts[0].Name != "Mark" {
		t.Fatalf("both people must be kept, the repeat dropped: %+v %+v", got.Contact, got.OtherContacts)
	}
	if n := len(got.People()); n != 2 {
		t.Fatalf("People() = %d, want 2", n)
	}
}
