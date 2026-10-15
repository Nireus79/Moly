package models

import "strings"

// DialoguePacket is what a model call is told about the conversation it is part of, besides the message it analyses. It is
// built once per turn from what is stored (no model call) and stays small whatever the length of the chat:
//   - the summary: Moly's memory of the conversation in plain prose (people, how they relate, what the user wants, what is
//     decided, what is open), rewritten after every turn;
//   - the goal the conversation is working on, and the people with what they are to the user (from the database);
//   - Moly's last message, so the new message connects to the flow: it is what a short reply answers or a comment is about;
//   - the user's message before that, only while the summary has not caught up with it.
//
// The purpose of Moly is to understand the user: a call that sees only the message operates blind (found live: a bare "Yes").
type DialoguePacket struct {
	Summary    string   // the stored summary (prose)
	Goal       string   // the goal the conversation is working on (locked)
	People     []string // the people involved, each with what they are to the user
	Open       []string // questions Moly asked that are still open
	MolyLast   string   // Moly's last message before the current one
	UserBefore string   // the user's message before Moly's last, only when the summary does not cover it yet
	// UserSaid holds the user's own earlier messages of this conversation (the latest few, each cut short). It is read only by the
	// judges that must know whether the user already told something (repeat of a question, gap generation), so that they do not depend
	// on a summary that lags or was cut short (found live, 2026-10-15: the user answered at length how much experience they have,
	// and Moly asked it again the next turn).
	UserSaid []string
}

const (
	userSaidMessages = 6
	userSaidRunes    = 800
)

// SetUserSaid keeps the latest of the user's own messages from the older messages of the conversation.
func (p *DialoguePacket) SetUserSaid(older []Message) {
	var said []string
	for _, m := range older {
		if m.Role != "user" || strings.TrimSpace(m.Content) == "" {
			continue
		}
		t := strings.TrimSpace(m.Content)
		if r := []rune(t); len(r) > userSaidRunes {
			t = string(r[:userSaidRunes]) + " ..."
		}
		said = append(said, t)
	}
	if len(said) > userSaidMessages {
		said = said[len(said)-userSaidMessages:]
	}
	p.UserSaid = said
}

// SaidText writes what the user said earlier as a prompt block, or "" when there is nothing.
func (p *DialoguePacket) SaidText() string {
	if p == nil || len(p.UserSaid) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("What the user said earlier in this conversation, in their own words (oldest first):\n")
	for _, t := range p.UserSaid {
		b.WriteString("- \"" + t + "\"\n")
	}
	return b.String() + "\n"
}

// Text writes the packet as a prompt block, or "" when there is nothing to say. The order is the order of importance for the
// reader: the memory first, the flow last, right before the new message.
func (p *DialoguePacket) Text() string {
	if p == nil {
		return ""
	}
	var b strings.Builder
	line := func(label, value string) {
		if strings.TrimSpace(value) != "" {
			b.WriteString(label + ": " + strings.TrimSpace(value) + "\n")
		}
	}
	list := func(label string, items []string) {
		if len(items) > 0 {
			b.WriteString(label + ":\n")
			for _, it := range items {
				b.WriteString("- " + it + "\n")
			}
		}
	}
	line("What Moly remembers of this conversation", p.Summary)
	line("The goal the user is working on", p.Goal)
	list("People involved", p.People)
	list("Questions Moly asked that are still open", p.Open)
	line("The user said before that (not in the memory yet)", p.UserBefore)
	line("Moly's last message", p.MolyLast)
	if b.Len() == 0 {
		return ""
	}
	return "THE DIALOGUE (a conversation between the user and Moly, a communication coach; the message below is the user's newest turn, and a short reply answers Moly's last message):\n" + b.String() + "\n"
}
