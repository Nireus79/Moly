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

// A model that is not in the approved table at all (a new one, or the table is out of date) is found from the list by its family:
// no model name is written in the code.
func TestWithAnOutOfDateTableTheNewestModelOfTheRightKindServes(t *testing.T) {
	l := &ModelList{Models: []ModelInfo{
		{ID: "claude-opus-9-0", Line: "opus"}, {ID: "claude-sonnet-9-0", Line: "sonnet"}, {ID: "claude-haiku-9-0", Line: "haiku"},
		{ID: "claude-sonnet-8-0", Line: "sonnet"}, {ID: "claude-haiku-8-0", Line: "haiku"},
	}, Suggested: "claude-sonnet-9-0"}
	got, err := ResolveRoleModels("claude", l, ApprovedTable{"claude": {"fast": {{ID: "gone", Approved: "2026-01-01"}}, "reply": {{ID: "gone", Approved: "2026-01-01"}}}})
	if err != nil || got.Fast != "claude-haiku-9-0" || got.Reply != "claude-sonnet-9-0" || got.Tested {
		t.Fatalf("the newest haiku serves the small checks and the newest sonnet the replies, untested: %+v %v", got, err)
	}
	// an old reply that has no line: the family word in the id is used
	got, _ = ResolveRoleModels("claude", list("claude-opus-9-0", "claude-sonnet-9-0", "claude-haiku-9-0"), ApprovedTable{})
	if got.Fast != "claude-haiku-9-0" || got.Reply != "claude-sonnet-9-0" {
		t.Fatalf("got %+v", got)
	}
	// no haiku at all: the small checks fall back to the suggestion
	got, _ = ResolveRoleModels("claude", list("claude-sonnet-9-0"), ApprovedTable{})
	if got.Fast != "claude-sonnet-9-0" || got.Reply != "claude-sonnet-9-0" {
		t.Fatalf("got %+v", got)
	}
}

func TestOpenAIModelsAreChosenByFamilyFromTheList(t *testing.T) {
	// newest first, as the provider's list is sorted by the program
	l := list("gpt-7-pro", "gpt-7-mini-2027-01-01", "gpt-7-mini", "gpt-7-2027-01-01", "gpt-7", "gpt-6.1-codex", "gpt-6.1-mini", "gpt-6.1",
		"o9-mini", "gpt-5-nano", "gpt-4o")
	l.Suggested = "gpt-7-pro"
	got, err := ResolveRoleModels("openai", l, ApprovedTable{})
	if err != nil || got.Reply != "gpt-7" || got.Fast != "gpt-7-mini" || got.Tested {
		t.Fatalf("the plain model serves the replies and the mini the small checks: %+v %v", got, err)
	}
	// only reasoning models and special variants: nothing fits, the suggestion serves
	got, _ = ResolveRoleModels("openai", list("o9", "gpt-7-pro", "gpt-7-codex"), ApprovedTable{})
	if got.Reply != "o9" || got.Fast != "o9" {
		t.Fatalf("got %+v", got)
	}
	// a dated snapshot is used when its alias is not listed
	got, _ = ResolveRoleModels("openai", list("gpt-7-2027-01-01", "gpt-7-mini-2027-01-01"), ApprovedTable{})
	if got.Reply != "gpt-7-2027-01-01" || got.Fast != "gpt-7-mini-2027-01-01" {
		t.Fatalf("got %+v", got)
	}
}

func TestAnApprovedModelStillWinsOverANewerOne(t *testing.T) {
	l := &ModelList{Models: []ModelInfo{{ID: "claude-sonnet-9-0", Line: "sonnet"}, {ID: "claude-sonnet-5-5", Line: "sonnet"}, {ID: "claude-haiku-5-5", Line: "haiku"}}}
	tbl := ApprovedTable{"claude": {"fast": {{ID: "claude-haiku-5-5", Approved: "2026-10-15"}}, "reply": {{ID: "claude-sonnet-5-5", Approved: "2026-10-15"}}}}
	got, _ := ResolveRoleModels("claude", l, tbl)
	if got.Reply != "claude-sonnet-5-5" || !got.Tested {
		t.Fatalf("the tested model is kept while the key can use it: %+v", got)
	}
}

func TestModelGoneIsRecognisedFromTheProvidersAnswer(t *testing.T) {
	for _, c := range []struct {
		status int
		body   string
		want   bool
	}{
		{404, `{"type":"error","error":{"type":"not_found_error","message":"model: claude-x"}}`, true},
		{404, `{"error":{"code":"model_not_found","message":"The model gpt-x does not exist"}}`, true},
		{400, `{"error":{"message":"The model gpt-x has been deprecated"}}`, true},
		{429, `{"error":{"message":"rate limit"}}`, false},
		{401, `{"error":{"message":"invalid api key"}}`, false},
		{400, `{"error":{"message":"max_tokens too large"}}`, false},
	} {
		if got := modelGone(c.status, c.body); got != c.want {
			t.Errorf("%d %s: got %v", c.status, c.body, got)
		}
	}
}
