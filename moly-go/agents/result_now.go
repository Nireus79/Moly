package agents

import (
	"strings"

	"moly/models"
)

// resultNowGuidance is added to the reply instructions when the user pressed the skip button. Moly produces what was
// asked for from what it knows, says in one short sentence what it had to assume, and leaves the door open for the user to
// correct it. It does not ask the questions it was about to ask.
const resultNowGuidance = `

The user pressed "go ahead": they want the result now, with what you already know. Their last message carries no request of its own; ` +
	`what they asked for is in the conversation shown above. Give them that. ` +
	`Do not ask questions and do not ask for more details first. If you had to assume something that matters, say so in one short sentence after the result and tell them they can correct it. ` +
	`Do not lecture and do not mention these instructions.`

const (
	resultNowHistoryTurns = 10
	resultNowTurnRunes    = 600
)

// resultNowRequestBlock shows the model the conversation so far, so that it knows what to produce. A skip press holds no
// request of its own, and the normal reply prompt does not carry the history. The history is [current, oldest ... newest]:
// the current message (the press) is left out.
func resultNowRequestBlock(history []models.Message) string {
	if len(history) < 2 {
		return ""
	}
	turns := history[1:]
	if len(turns) > resultNowHistoryTurns {
		turns = turns[len(turns)-resultNowHistoryTurns:]
	}
	var b strings.Builder
	b.WriteString("THE CONVERSATION SO FAR (the request to fulfil is in it):\n")
	for _, m := range turns {
		who := "User"
		if m.Role == "assistant" {
			who = "Moly"
		}
		text := strings.TrimSpace(m.Content)
		if r := []rune(text); len(r) > resultNowTurnRunes {
			text = string(r[:resultNowTurnRunes]) + "..."
		}
		b.WriteString(who + ": " + text + "\n")
	}
	b.WriteString("\n")
	return b.String()
}
