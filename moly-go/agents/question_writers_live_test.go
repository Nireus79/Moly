package agents

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"
	"unicode"

	"moly/models"
	"moly/tools"
)

// TestLiveQuestionWritersAddressTheUser measures two things on the writers that ask a question, against a real model (MOLY_LIVE_LLM=1;
// MOLY_LIVE_REPLAYS=n, default 5): how often the first attempt breaks the one-question contract (counted from the log lines the writers
// write, which hold no text), and how often a question opens by calling the user with the name of a person they mentioned
// ("Name, ..."), which is wrong because the people in a conversation are third persons. The messages are made-up and mention a named person.
func TestLiveQuestionWritersAddressTheUser(t *testing.T) {
	if os.Getenv("MOLY_LIVE_LLM") != "1" {
		t.Skip("set MOLY_LIVE_LLM=1 to run against a real model")
	}
	c, err := tools.NewLLMClient()
	if err != nil {
		t.Fatal(err)
	}
	replays := 5
	if v := os.Getenv("MOLY_LIVE_REPLAYS"); v != "" {
		replays = 0
		for _, r := range v {
			replays = replays*10 + int(r-'0')
		}
	}
	ca := &conversationAgent{llmClient: c, constitution: &models.Constitution{}}
	// Each case is a later turn of a conversation: the dialogue packet lists the person, as the real turn does.
	cases := []struct{ name, msg, person, goal, rel, last string }{
		{"greek", "Η αδερφή μου η Μαρία δεν μου μιλάει από τότε που μαλώσαμε και θέλω να της γράψω.", "Μαρία", "να γράψω στην αδερφή μου", "αδερφή του χρήστη", "Τι θα ήθελες να καταλάβει από το μήνυμα;"},
		{"english", "My brother Daniel stopped talking to me after we argued and I want to write to him.", "Daniel", "write to my brother", "the user's brother", "What would you like him to understand from the message?"},
		{"spanish", "Mi hermana Lucía no me habla desde que discutimos y quiero escribirle.", "Lucía", "escribirle a mi hermana", "hermana del usuario", "¿Qué te gustaría que entendiera con el mensaje?"},
		{"russian", "Моя сестра Ольга не разговаривает со мной после ссоры, и я хочу ей написать.", "Ольга", "написать моей сестре", "сестра пользователя", "Что бы вы хотели, чтобы она поняла из сообщения?"},
	}
	sites := []struct {
		name string
		run  func(msg, dialogue string) string
	}{
		{"aim", func(m, d string) string {
			return ca.writeQuestionFromAim(m, d, "ask what success would look like for the user's goal (\""+m+"\")")
		}},
		{"aim-assumption", func(m, d string) string {
			return ca.writeQuestionFromAim(m, d, "ask what assumption the user is making about how the other person sees the matter")
		}},
		{"principle", func(m, d string) string { return ca.principleQuestion("stakeholder_consideration", m, m, d) }},
	}
	vocative := func(out, person string) bool {
		o := strings.TrimLeftFunc(out, func(r rune) bool { return !unicode.IsLetter(r) })
		if !strings.HasPrefix(o, person) {
			return false
		}
		rest := strings.TrimPrefix(o, person)
		return strings.HasPrefix(rest, ",") || strings.HasPrefix(rest, ":") || strings.HasPrefix(rest, "!")
	}
	totalBreaks, totalCalls, totalVoc, totalEmpty := 0, 0, 0, 0
	for _, s := range sites {
		for _, cs := range cases {
			var buf bytes.Buffer
			log.SetOutput(&buf)
			voc, empty := 0, 0
			for r := 0; r < replays; r++ {
				packet := &models.DialoguePacket{Goal: cs.goal, People: []string{cs.person + " (" + cs.rel + ")"}, MolyLast: cs.last}
				out := s.run(cs.msg, packet.Text())
				if out == "" {
					empty++
				} else if vocative(out, cs.person) {
					voc++
					t.Logf("  vocative: %s", strings.ReplaceAll(out, "\n", " "))
				}
			}
			log.SetOutput(os.Stderr)
			breaks := strings.Count(buf.String(), "broke the contract") + strings.Count(buf.String(), "not usable")
			totalBreaks += breaks
			totalCalls += replays
			totalVoc += voc
			totalEmpty += empty
			t.Logf("%-14s %-8s contract breaks %d of %d, name as address %d, no question %d", s.name, cs.name, breaks, replays, voc, empty)
		}
	}
	t.Logf("TOTAL: contract breaks %d / %d runs, name as address %d, no question at all %d", totalBreaks, totalCalls, totalVoc, totalEmpty)
}
