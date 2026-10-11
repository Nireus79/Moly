package agents

import "testing"

func TestBareLabel(t *testing.T) {
	for in, want := range map[string]string{"my boss": "boss", "My manager": "manager", "the girl": "girl", "sister": "sister", "his sister": "sister", " a colleague ": "colleague", "": ""} {
		if got := bareLabel(in); got != want {
			t.Errorf("bareLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExtractionKeepsTheRoleAndABareLabel(t *testing.T) {
	ce := &ContextExtractor{}
	got, err := ce.parseLLMExtraction(`{"intention": "write to Dana", "intentionBasis": "stated", "contact": {"name": "Dana", "label": "my manager Dana", "role": "my manager", "relationship": "professional", "confidence": 0.9, "evidence": "my manager Dana", "basis": "stated"}}`)
	if err != nil || got.Contact == nil || got.Contact.Role != "manager" || got.Contact.Name != "Dana" {
		t.Fatalf("err=%v contact=%+v", err, got.Contact)
	}
	got, _ = ce.parseLLMExtraction(`{"intention": "write", "contact": {"name": "", "label": "my boss", "relationship": "professional", "confidence": 0.9, "evidence": "my boss", "basis": "stated"}}`)
	if got.Contact == nil || got.Contact.Name != "boss" || got.Contact.NameKnown {
		t.Fatalf("an unnamed person is stored under the bare label, got %+v", got.Contact)
	}
}
