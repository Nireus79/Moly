package agents

import (
	"os"
	"regexp"
	"strings"
	"testing"
	"unicode"

	"moly/models"
	"moly/tools"
)

// TestLiveLanguageOfFixedTextSites runs the writers that used to hold fixed English sentences against a real model, with a message in
// each language, and checks the language of what they write (MOLY_LIVE_LLM=1; MOLY_LIVE_REPLAYS=n, default 2). The messages are
// ordinary: no harmful content is needed to see the language of a refusal, because the refusal text is the evaluator's own sentence.
func TestLiveLanguageOfFixedTextSites(t *testing.T) {
	if os.Getenv("MOLY_LIVE_LLM") != "1" {
		t.Skip("set MOLY_LIVE_LLM=1 to run against a real model")
	}
	c, err := tools.NewLLMClient()
	if err != nil {
		t.Fatal(err)
	}
	replays := 2
	if v := os.Getenv("MOLY_LIVE_REPLAYS"); v != "" {
		var n int
		for _, r := range v {
			n = n*10 + int(r-'0')
		}
		replays = n
	}
	ca := &conversationAgent{llmClient: c, constitution: &models.Constitution{}}
	type lang struct {
		name, msg string
		script    *unicode.RangeTable // nil: Latin script, checked by English stop words
	}
	langs := []lang{
		{"english", "I want to write to my manager and ask for a day off next week.", nil},
		{"spanish", "Quiero escribirle a mi jefa para pedirle un día libre la semana que viene.", nil},
		{"greek", "Θέλω να γράψω στον διευθυντή μου και να ζητήσω μια μέρα άδεια την επόμενη εβδομάδα.", unicode.Greek},
		{"russian", "Я хочу написать начальнику и попросить выходной на следующей неделе.", unicode.Cyrillic},
		{"arabic", "أريد أن أكتب إلى مديري وأطلب يوم إجازة الأسبوع القادم.", unicode.Arabic},
		{"chinese", "我想给我的经理写信，请求下周休一天假。", unicode.Han},
	}
	english := regexp.MustCompile(`(?i)\b(the|you|your|what|how|would|could|about|this|that|with)\b`)
	spanishMark := regexp.MustCompile(`(?i)[¿¡áéíóúñ]|\b(qué|cómo|para|con|por|favor|puedo|una?|el|la|los|las|de|en|no|tu|usted)\b`)
	const base = "I can't help with that. Please consider asking advice from a law expert."
	inLanguage := func(l lang, out string) bool {
		if l.name == "english" {
			return true
		}
		if l.script != nil {
			letters, in := 0, 0
			for _, r := range out {
				if unicode.IsLetter(r) {
					letters++
					if unicode.Is(l.script, r) {
						in++
					}
				}
			}
			return letters > 0 && in*2 >= letters
		}
		return spanishMark.MatchString(out) && len(english.FindAllString(out, -1)) < 2
	}
	dialogue := "" // no earlier dialogue: a first message
	sites := []struct {
		name string
		run  func(l lang) string
	}{
		{"socratic", func(l lang) string {
			return ca.writeQuestionFromAim(l.msg, dialogue, "ask what success would look like for the user's goal (\""+l.msg+"\")")
		}},
		{"principle", func(l lang) string {
			return ca.principleQuestion("stakeholder_consideration", l.msg, l.msg, dialogue)
		}},
		{"replacement", func(l lang) string { return ca.replacementForBlockedReply(l.msg) }},
		{"refusal", func(l lang) string { return ca.refusalInUsersLanguage(l.msg, base) }},
	}
	total, good := 0, 0
	for _, s := range sites {
		for _, l := range langs {
			ok := 0
			var sample string
			for r := 0; r < replays; r++ {
				out := s.run(l)
				total++
				pass := strings.TrimSpace(out) != "" && inLanguage(l, out)
				if s.name == "refusal" {
					pass = pass && countQuestionMarks(out) == 0
				} else {
					pass = pass && countQuestionMarks(out) == 1
				}
				if pass {
					ok++
					good++
				} else {
					sample = out
				}
				if r == 0 && sample == "" {
					t.Logf("%-11s %-8s sample: %s", s.name, l.name, strings.ReplaceAll(out, "\n", " "))
				}
			}
			t.Logf("%-11s %-8s %d/%d%s", s.name, l.name, ok, replays, map[bool]string{false: "", true: "   MISS: " + strings.ReplaceAll(sample, "\n", " ")}[ok < replays])
		}
	}
	t.Logf("language of the writers: %d/%d", good, total)
}
