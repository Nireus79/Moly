package agents

import (
	"context"
	"log"
	"regexp"
	"strings"
	"unicode"

	"moly/tools"
)

// A refusal is a product rule (CLAUDE.md): short, names who to ask, no help lines, no questions. The evaluator builds it in English
// (tools/constitutional_evaluator.go); a user who writes another language must get the same refusal in that language, not English
// (found with the tester's report, 2026-10-29). refusalInUsersLanguage has the fast model say the same sentence in the language of the
// user's words, checks the text, tries once more, and only if both attempts fail sends the English sentence: failing safe matters more
// than the language here.
func (ca *conversationAgent) refusalInUsersLanguage(userMessage, base string) string {
	if ca.llmClient == nil || strings.TrimSpace(base) == "" || strings.TrimSpace(userMessage) == "" {
		return base
	}
	for attempt := 1; attempt <= 2; attempt++ {
		resp, err := ca.llmClient.Call(context.Background(), &tools.LLMRequest{
			Role:         tools.RoleFast,
			SystemPrompt: "You translate one short message of Moly, a communication thinking partner. You change nothing but the language and, where the language has a formal and an informal \"you\", the form of address: the informal singular unless the user's own words are formal.",
			UserPrompt: "The user wrote: \"" + userMessage + "\"\n\nMoly's message, in English:\n\"" + base + "\"\n\n" +
				"Write exactly this message in the language the user writes in (the language of their own words, never that of a name, a number or a quoted word). " +
				"Keep its meaning and its length. Add nothing: no question, no advice, no list, no phone number, no website, no extra sentence. " +
				"Where the language has a formal and an informal \"you\", use the form the user uses for themselves; if their words do not show it, use the informal singular. " +
				"If the user writes English, repeat the message as it is. Output only the message.",
			MaxTokens:   120,
			Temperature: 0.1,
		})
		if err != nil {
			log.Printf("[ConversationAgent] Refusal in the user's language failed (attempt %d): %v", attempt, err)
			continue
		}
		out := strings.Trim(strings.TrimSpace(resp.Content), "\"“”")
		if ok, why := refusalTextOK(out, base, userMessage); ok {
			return out
		} else {
			log.Printf("[ConversationAgent] Refusal in the user's language rejected (attempt %d): %s", attempt, why)
		}
	}
	return base
}

var refusalForbidden = regexp.MustCompile(`(?i)https?://|www\.|\d{3,}`)

// refusalTextOK checks a refusal before it is sent: one short paragraph, no question, no help line (a number or an address), and in the
// script of the user's words when those are not Latin.
func refusalTextOK(out, base, userMessage string) (bool, string) {
	switch {
	case strings.TrimSpace(out) == "":
		return false, "empty"
	case countQuestionMarks(out) > 0:
		return false, "asks a question"
	case strings.Contains(out, "\n"):
		return false, "more than one paragraph"
	case len([]rune(out)) > 2*len([]rune(base))+60:
		return false, "much longer than the refusal"
	case refusalForbidden.MatchString(out):
		return false, "holds a number or an address"
	}
	if script := dominantScript(userMessage); script != nil {
		letters, inScript := 0, 0
		for _, r := range out {
			if unicode.IsLetter(r) {
				letters++
				if unicode.Is(script, r) {
					inScript++
				}
			}
		}
		if letters == 0 || inScript*2 < letters {
			return false, "not in the script of the user's words"
		}
	}
	return true, ""
}

// dominantScript is the non-Latin script most of the user's letters are in, or nil when the letters are mostly Latin.
func dominantScript(text string) *unicode.RangeTable {
	scripts := []*unicode.RangeTable{unicode.Greek, unicode.Cyrillic, unicode.Arabic, unicode.Hebrew, unicode.Han, unicode.Hiragana, unicode.Katakana,
		unicode.Hangul, unicode.Devanagari, unicode.Thai}
	letters := 0
	count := make([]int, len(scripts))
	for _, r := range text {
		if !unicode.IsLetter(r) {
			continue
		}
		letters++
		for i, s := range scripts {
			if unicode.Is(s, r) {
				count[i]++
				break
			}
		}
	}
	best, bestN := -1, 0
	for i, n := range count {
		if n > bestN {
			best, bestN = i, n
		}
	}
	if best >= 0 && bestN*2 >= letters {
		return scripts[best]
	}
	return nil
}
