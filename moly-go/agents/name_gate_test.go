package agents

import (
	"context"
	"testing"

	"moly/models"
	"moly/tools"
)

// PHASE 5: a person without a given name keeps the label and is not NameKnown.
func TestExtractedContactWithoutNameKeepsLabel(t *testing.T) {
	ce := &ContextExtractor{}
	got, err := ce.parseLLMExtraction(`{"intention":"write a first message","contact":{"name":"","label":"girl on fetlife","relationship":"other","confidence":0.8}}`)
	if err != nil || got == nil || got.Contact == nil {
		t.Fatalf("parse failed: %v %+v", err, got)
	}
	if got.Contact.NameKnown {
		t.Fatal("a contact without a given name must not be NameKnown")
	}
	if got.Contact.Label != "girl on fetlife" || got.Contact.Name != "girl on fetlife" {
		t.Fatalf("label not kept for storage: name=%q label=%q", got.Contact.Name, got.Contact.Label)
	}
}

// PHASE 5: a given name is NameKnown and is the stored name.
func TestExtractedContactWithGivenName(t *testing.T) {
	ce := &ContextExtractor{}
	got, err := ce.parseLLMExtraction(`{"intention":"write a first message","contact":{"name":"Christine","label":"girl","relationship":"other","confidence":0.8}}`)
	if err != nil || got == nil || got.Contact == nil {
		t.Fatalf("parse failed: %v", err)
	}
	if !got.Contact.NameKnown || got.Contact.Name != "Christine" {
		t.Fatalf("given name lost: known=%v name=%q", got.Contact.NameKnown, got.Contact.Name)
	}
}

// PHASE 5: the question text is fixed and uses the user's own label.
func TestNameQuestionUsesLabel(t *testing.T) {
	want := "Can you give me a name for the girl you mentioned?"
	if got := NameQuestion("girl"); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

// PHASE 3: a greeting is not extracted again by layer 1 and never locks a goal.
func TestLayer1SkipsGreeting(t *testing.T) {
	lc := tools.NewLayerContext(&models.AnalysisContext{CurrentMessage: "Hi Moly", IsGreeting: true}, "u1", "m1", "c1")
	out, err := NewLayer1ContextExtractionAdapter(nil, tools.NewExtractionCache()).Process(context.Background(), lc)
	if err != nil {
		t.Fatalf("layer 1: %v", err)
	}
	if out.PrimaryGoal != "" {
		t.Fatalf("greeting locked a goal: %q", out.PrimaryGoal)
	}
}
