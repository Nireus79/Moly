package agents

import (
	"regexp"
	"strings"
)

// A message Moly writes is ready to send: a placeholder in square brackets ("Hello [Name],") is left out. The prompt says so, and the model still
// writes one now and then (found in a dynamic run), so the form is also checked on the text: a short bracketed word that starts with a letter
// and is not a link is removed, with the space before it, and the punctuation left behind is tidied.
var (
	placeholderPattern = regexp.MustCompile(`[ \t]*\[[A-Za-z][^\]\n]{0,29}\]`)
	spaceBeforePunct   = regexp.MustCompile(`[ \t]+([,.;:!?])`)
)

// StripPlaceholders removes bracketed placeholders from a reply.
func StripPlaceholders(text string) string {
	if !strings.Contains(text, "[") {
		return text
	}
	var out strings.Builder
	rest := text
	for {
		loc := placeholderPattern.FindStringIndex(rest)
		if loc == nil {
			out.WriteString(rest)
			break
		}
		end := loc[1]
		if end < len(rest) && rest[end] == '(' { // [text](address): a link, not a placeholder
			out.WriteString(rest[:end])
			rest = rest[end:]
			continue
		}
		out.WriteString(rest[:loc[0]])
		rest = rest[end:]
	}
	return strings.TrimRight(spaceBeforePunct.ReplaceAllString(out.String(), "$1"), " \t\n")
}
