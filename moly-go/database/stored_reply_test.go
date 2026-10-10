package database

import "testing"

func TestStoredReplyText(t *testing.T) {
	cases := map[string]string{
		"plain text": "plain text",
		`{"phase":"responding","response":"Hello there"}`: "Hello there",
		`{"response":""}`:        "",
		`{"note":"not a reply"}`: `{"note":"not a reply"}`,
		"{broken":                "{broken",
		"":                       "",
	}
	for in, want := range cases {
		if got := StoredReplyText(in); got != want {
			t.Errorf("StoredReplyText(%q) = %q, want %q", in, got, want)
		}
	}
}
