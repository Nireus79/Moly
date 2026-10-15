package agents

import "testing"

func TestNormalizeRelationshipIsOneKind(t *testing.T) {
	cases := map[string]string{
		"romantic": "romantic", " Family ": "family", "romantic + other": "romantic", "other | friend": "friend",
		"professional/romantic": "professional", "my sister": "other", "": "", "other": "other",
	}
	for in, want := range cases {
		if got := normalizeRelationship(in); got != want {
			t.Errorf("normalizeRelationship(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNameQuestionUsesPersonForAClauseOrNothing(t *testing.T) {
	cases := map[string]string{
		"manager":                      "Can you give me a name for the manager you mentioned?",
		"someone I met in a chat room": "Can you give me a name for the person you mentioned?",
		"":                             "Can you give me a name for the person you mentioned?",
		"old friend":                   "Can you give me a name for the old friend you mentioned?",
	}
	for in, want := range cases {
		if got := NameQuestion(in); got != want {
			t.Errorf("NameQuestion(%q) = %q, want %q", in, got, want)
		}
	}
}
