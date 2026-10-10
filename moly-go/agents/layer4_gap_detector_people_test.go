package agents

import (
	"strings"
	"testing"

	"moly/models"
)

func TestPeopleStatusBlock(t *testing.T) {
	ctx := &models.AnalysisContext{RelevantContacts: []models.Contact{{Name: "Christine", NameStatus: "named"}}}
	block, unnamed := peopleStatusBlock(ctx)
	if unnamed {
		t.Fatal("a named person must not count as unnamed")
	}
	if !strings.Contains(block, "Christine") || !strings.Contains(block, "Do NOT ask for a full or real name") {
		t.Fatalf("block must say the name is answered, got %q", block)
	}
	if !strings.Contains(nameRule(false), "Do NOT ask for any person's name") {
		t.Fatal("with every name known, the prompt must forbid asking for a name")
	}

	ctx.RelevantContacts = []models.Contact{{Name: "girl", NameStatus: "asked"}}
	_, unnamed = peopleStatusBlock(ctx)
	if !unnamed || !strings.Contains(nameRule(true), "may be asked for a name") {
		t.Fatal("an unnamed person may be asked for a name")
	}
}
