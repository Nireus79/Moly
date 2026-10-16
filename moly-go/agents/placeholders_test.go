package agents

import "testing"

func TestStripPlaceholders(t *testing.T) {
	cases := map[string]string{
		"Hello [Name],\n\nSee you.":               "Hello,\n\nSee you.",
		"Dear [boss], I want to talk.":            "Dear, I want to talk.",
		"Thanks,\n[Your name]":                    "Thanks,",
		"No brackets here.":                       "No brackets here.",
		"A [link](https://example.org) stays.":    "A [link](https://example.org) stays.",
		"Draft [1] stays? No: [1] is not a word.": "Draft [1] stays? No: [1] is not a word.",
	}
	for in, want := range cases {
		if got := StripPlaceholders(in); got != want {
			t.Errorf("StripPlaceholders(%q) = %q, want %q", in, got, want)
		}
	}
}
