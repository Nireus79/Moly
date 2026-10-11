package models

import "strings"

// The meaning of a short message depends on what was said before it: "Yes" answers Moly's last question. Every model call that
// interprets the user's message gets the recent exchange with it. These helpers write that exchange as text.

const (
	exchangeUserRunes = 300
	// Moly's own messages are kept longer: its last reply may be the draft the user is now commenting on.
	exchangeMolyRunes = 800
)

// ExchangeBeforeChronological writes the last maxMessages messages before the current one. messages are oldest first and may
// end with the current message (it is left out).
func ExchangeBeforeChronological(messages []Message, current string, maxMessages int) string {
	if n := len(messages); n > 0 && messages[n-1].Role == "user" && strings.TrimSpace(messages[n-1].Content) == strings.TrimSpace(current) {
		messages = messages[:n-1]
	}
	return exchangeText(messages, maxMessages)
}

// MessagesBefore returns the messages before the current one, oldest first, whichever layout the list has (the current message
// last, or first). Only an entry that is the current user message is left out.
func MessagesBefore(messages []Message, current string) []Message {
	same := func(m Message) bool { return m.Role == "user" && strings.TrimSpace(m.Content) == strings.TrimSpace(current) }
	switch n := len(messages); {
	case n > 0 && same(messages[n-1]):
		return messages[:n-1]
	case n > 1 && same(messages[0]):
		return messages[1:]
	}
	return messages
}

// LastMolyMessage returns what Moly said last (limited to the length kept for a draft) and the user message right before it
// (limited to the length kept for a user message). Either is "" when there is none.
func LastMolyMessage(older []Message) (moly, userBefore string) {
	for i := len(older) - 1; i >= 0; i-- {
		if older[i].Role != "assistant" {
			continue
		}
		moly = cut(older[i].Content, exchangeMolyRunes)
		for j := i - 1; j >= 0; j-- {
			if older[j].Role == "user" {
				userBefore = cut(older[j].Content, exchangeUserRunes)
				break
			}
		}
		return moly, userBefore
	}
	return "", ""
}

func cut(text string, limit int) string {
	r := []rune(strings.TrimSpace(text))
	if len(r) > limit {
		r = r[:limit]
	}
	return string(r)
}

// ExchangeBeforeCurrentFirst is the same for the layout the agent uses: messages[0] is the current message, then the
// older ones follow oldest first.
func ExchangeBeforeCurrentFirst(messages []Message, maxMessages int) string {
	if len(messages) > 0 && messages[0].Role == "user" {
		messages = messages[1:]
	}
	return exchangeText(messages, maxMessages)
}

func exchangeText(messages []Message, maxMessages int) string {
	if maxMessages > 0 && len(messages) > maxMessages {
		messages = messages[len(messages)-maxMessages:]
	}
	var b strings.Builder
	for _, m := range messages {
		text := []rune(strings.TrimSpace(m.Content))
		if len(text) == 0 {
			continue
		}
		role, limit := "User", exchangeUserRunes
		if m.Role == "assistant" {
			role, limit = "Moly", exchangeMolyRunes
		}
		if len(text) > limit {
			text = text[:limit]
		}
		b.WriteString(role + ": " + string(text) + "\n")
	}
	return b.String()
}

// WithExchange writes a prompt block for an exchange, or nothing when there is none.
func WithExchange(exchange string) string {
	if strings.TrimSpace(exchange) == "" {
		return ""
	}
	return "RECENT EXCHANGE (only to understand what the current message means; a short reply such as \"yes\", \"no\" or a name answers Moly's last message):\n" + exchange + "\n"
}
