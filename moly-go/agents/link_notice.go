package agents

import (
	"context"
	"log"
	"regexp"
	"strings"

	"moly/tools"
)

// Moly never opens, visits or reads a web page. A user who pastes a link must be told so, in their own language, and must not be led to
// expect that Moly will look at it. Found in manual chats: replies offered "to look at her profile" or "a summary of the article" for a pasted link.

// readsNoPages is added to the instructions of the models that write replies and questions, so that none of them offers to open a link.
const readsNoPages = " Moly never opens, visits or reads web pages: never offer to look at, read, summarise, check or use what is behind a link; when the user's message depends on a page, say that Moly does not read web pages (a rule, not an inability: \"does not\", never \"cannot\") and ask them to paste the text."

// namesAskedElsewhere is added to the instructions of the models that write other questions: Moly asks for a person's name in its own
// step, once. Found in a dynamic run: the clarity and gap questions asked a user who had refused for the brother's name two more times.
const namesAskedElsewhere = " Never ask for a person's name or nickname: Moly asks for it in a separate step."

// linkPattern finds an address with a scheme, one starting with www., or a domain followed by a path. A bare word with a dot is not a link.
var linkPattern = regexp.MustCompile(`(?i)(?:\b[a-z][a-z0-9+.-]*://|\bwww\.)\S+|\b[a-z0-9-]+(?:\.[a-z0-9-]+)*\.[a-z]{2,}/\S*`)

// MessageHasLink reports whether the message contains a web address.
func MessageHasLink(message string) bool { return linkPattern.MatchString(message) }

// fallbackLinkNotice is used only when the model cannot write the notice.
const fallbackLinkNotice = "Moly does not open or read web pages. If something on that page matters, paste the text here."

// LinkNotice returns the sentence to put in front of the reply when the user's message contains a link, or "" when the message has none
// or the reply already says that Moly does not read the page and promises nothing of the kind. The model writes it in the user's language.
func LinkNotice(ctx context.Context, llm tools.LLMProvider, message, reply string) string {
	if !MessageHasLink(message) {
		return ""
	}
	if llm == nil {
		return fallbackLinkNotice
	}
	resp, err := llm.Call(ctx, &tools.LLMRequest{
		SystemPrompt: "You check what a coach's reply tells a user who pasted a web link.",
		UserPrompt: "The user's message contains a web link. Moly never opens, visits or reads web pages.\n\nUser's message:\n\"" + message + "\"\n\nMoly's reply:\n\"" + reply + "\"\n\n" +
			"Does the reply SAY outright that Moly does not open or read web pages, without suggesting that it will? Asking the user to paste the text is not enough on its own: the reply itself must state it. " +
			"Offering to look at the page, summarise it, check it or use what it says counts as suggesting that it will. " +
			"If the reply already makes this clear and promises nothing of the kind, needed is false. " +
			"Otherwise needed is true and notice is one or two short sentences, in the language of the words the user wrote around the link (not the language of the address), that say Moly does not open or read web pages (it is Moly's rule, so say \"does not\", never \"cannot\") " +
			"and that, if something on the page matters, the user can paste the text here. Plain words: no apology, no greeting, and do not repeat the link.\n" +
			"Respond with ONLY JSON: {\"needed\": true or false, \"notice\": \"...\"}",
		MaxTokens:   300,
		Temperature: 0.2,
	})
	if err != nil {
		log.Printf("[LinkNotice] could not judge the reply: %v (standard notice used)", err)
		return fallbackLinkNotice
	}
	var out struct {
		Needed tools.LenientBool `json:"needed"`
		Notice string            `json:"notice"`
	}
	if err := tools.SafeJSONParse("LinkNotice", []byte(resp.Content), &out); err != nil {
		log.Printf("[LinkNotice] answer not readable: %v (standard notice used)", err)
		return fallbackLinkNotice
	}
	if !bool(out.Needed) {
		return ""
	}
	notice := strings.TrimSpace(out.Notice)
	if notice == "" || linkPattern.MatchString(notice) {
		return fallbackLinkNotice
	}
	return notice
}

// WithLinkNotice puts the notice in front of the reply.
func WithLinkNotice(notice, reply string) string {
	if notice == "" {
		return reply
	}
	return notice + "\n\n" + reply
}
