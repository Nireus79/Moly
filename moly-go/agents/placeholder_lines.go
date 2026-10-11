package agents

import (
	"regexp"
	"strings"
)

// placeholderLine is a line that holds only a bracketed placeholder, such as the signature "[Your Name]" a model leaves in a draft.
// The draft is meant to be ready to send, and Moly does not know the user's name for the signature.
var placeholderLine = regexp.MustCompile(`^\s*\[[^\]\n]{1,40}\]\s*$`)

// withoutPlaceholderLines removes the lines that are only a bracketed placeholder.
func withoutPlaceholderLines(text string) string {
	lines := strings.Split(text, "\n")
	kept := lines[:0]
	for _, l := range lines {
		if !placeholderLine.MatchString(l) {
			kept = append(kept, l)
		}
	}
	return strings.TrimSpace(strings.Join(kept, "\n"))
}
