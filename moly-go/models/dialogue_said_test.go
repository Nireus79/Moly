package models

import (
	"strings"
	"testing"
)

// The packet keeps the user's latest messages, each cut short, for the judges that must know what was already told.
func TestUserSaidKeepsTheLatestUserMessagesCutShort(t *testing.T) {
	var older []Message
	for i := 0; i < 9; i++ {
		older = append(older, Message{Role: "user", Content: strings.Repeat("x", 1000) + string(rune('a'+i))}, Message{Role: "assistant", Content: "a question?"})
	}
	p := &DialoguePacket{}
	p.SetUserSaid(older)
	if len(p.UserSaid) != userSaidMessages {
		t.Fatalf("kept %d messages, want %d", len(p.UserSaid), userSaidMessages)
	}
	if len([]rune(p.UserSaid[0])) > userSaidRunes+10 {
		t.Fatalf("a message is cut to %d characters, got %d", userSaidRunes, len([]rune(p.UserSaid[0])))
	}
	if strings.Contains(p.SaidText(), "question?") {
		t.Fatal("Moly's own messages are not part of what the user said")
	}
	if (&DialoguePacket{}).SaidText() != "" || (*DialoguePacket)(nil).SaidText() != "" {
		t.Fatal("nothing said gives an empty block")
	}
}
