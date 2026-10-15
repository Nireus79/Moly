package tools

import "testing"

func list(ids ...string) *ModelList {
	l := &ModelList{}
	for _, id := range ids {
		l.Models = append(l.Models, ModelInfo{ID: id, Label: id})
	}
	if len(ids) > 0 {
		l.Suggested = ids[0]
	}
	return l
}

func table() ApprovedTable {
	return ApprovedTable{"claude": {
		"fast":  {{ID: "haiku-new", Approved: "2026-10-15"}, {ID: "haiku-old", Approved: "2026-01-01"}},
		"reply": {{ID: "sonnet-new", Approved: "2026-10-15"}, {ID: "sonnet-old", Approved: "2026-01-01"}},
	}}
}

// The newest approved model the key can use serves each role.
func TestTheNewestApprovedModelOfTheKeyServesEachRole(t *testing.T) {
	got, err := ResolveRoleModels("claude", list("opus-new", "sonnet-new", "haiku-new", "haiku-old"), table())
	if err != nil || got.Fast != "haiku-new" || got.Reply != "sonnet-new" || !got.Tested {
		t.Fatalf("got %+v %v", got, err)
	}
}

// A retired model, or one the key has no access to, is replaced by the next approved one.
func TestARetiredApprovedModelIsReplacedByTheNextApprovedOne(t *testing.T) {
	got, _ := ResolveRoleModels("claude", list("sonnet-old", "haiku-old"), table())
	if got.Fast != "haiku-old" || got.Reply != "sonnet-old" || !got.Tested {
		t.Fatalf("got %+v", got)
	}
}

// A model newer than the table is never used by itself.
func TestANewerUnapprovedModelIsNotUsed(t *testing.T) {
	got, _ := ResolveRoleModels("claude", list("sonnet-next", "haiku-next", "sonnet-new", "haiku-new"), table())
	if got.Fast != "haiku-new" || got.Reply != "sonnet-new" {
		t.Fatalf("got %+v", got)
	}
}

// With nothing approved available the provider's suggestion serves both roles, and the result says it is untested.
func TestWithNothingApprovedAvailableTheSuggestionServesAndIsMarkedUntested(t *testing.T) {
	got, err := ResolveRoleModels("claude", list("other-1", "other-2"), table())
	if err != nil || got.Fast != "other-1" || got.Reply != "other-1" || got.Tested {
		t.Fatalf("got %+v %v", got, err)
	}
	if _, err := ResolveRoleModels("claude", list(), table()); err == nil {
		t.Fatal("an empty list is an error")
	}
	// a provider the table does not know (OpenAI at first) works the same way
	if got, _ := ResolveRoleModels("openai", list("gpt-x"), table()); got.Fast != "gpt-x" || got.Reply != "gpt-x" || got.Tested {
		t.Fatalf("got %+v", got)
	}
}

// One role approved and the other not: the approved one is used, the result is untested.
func TestOneRoleApprovedAndTheOtherNotIsUntested(t *testing.T) {
	got, _ := ResolveRoleModels("claude", list("haiku-new", "plain"), table())
	if got.Fast != "haiku-new" || got.Reply != "haiku-new" && got.Reply != "plain" || got.Tested {
		t.Fatalf("got %+v", got)
	}
}

// The table shipped with the program parses and names real entries.
func TestTheShippedTableParses(t *testing.T) {
	tab := ApprovedModels()
	if len(tab["claude"]["fast"]) == 0 || len(tab["claude"]["reply"]) == 0 {
		t.Fatalf("the Claude table must name a fast and a reply model: %+v", tab)
	}
	if _, bad := tab["_comment"]; bad {
		t.Fatal("the comment is not a provider")
	}
}
