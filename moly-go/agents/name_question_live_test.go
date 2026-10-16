package agents

import (
	"os"
	"regexp"
	"testing"

	"moly/tools"
)

// A live check (off by default): English messages get the standard English question; other languages get a question in their own script or
// language. Found in dynamic runs: an English user was asked in German.
func TestLiveNameQuestionLanguage(t *testing.T) {
	if os.Getenv("MOLY_LIVE_LLM") == "" {
		t.Skip()
	}
	c, err := tools.NewLLMClient()
	if err != nil {
		t.Skip(err)
	}
	ca := &conversationAgent{llmClient: c}
	cases := []struct {
		msg, label string
		english    bool
		script     *regexp.Regexp
	}{
		{"I want to write a first message to a woman I noticed on a hobby forum. I think we have things in common.", "woman", true, nil},
		{"I am so tired of my sister ignoring me and I do not know what to say to her.", "sister", true, nil},
		{"Write to my brother. He borrowed money last spring and has not paid it back.", "brother", true, nil},
		{"Θέλω να γράψω στη διευθύντριά μου για μια καθυστέρηση στο έργο.", "διευθύντριά", false, regexp.MustCompile(`\p{Greek}`)},
		{"Quiero escribirle a mi jefa sobre un retraso en el proyecto.", "jefa", false, regexp.MustCompile(`(?i)nombre|llam`)},
	}
	for _, k := range cases {
		ok := 0
		var last string
		for i := 0; i < 3; i++ {
			last = ca.nameQuestion(k.msg, "", k.label)
			isEnglishFixed := last == NameQuestion(k.label)
			if k.english && isEnglishFixed || !k.english && !isEnglishFixed && k.script.MatchString(last) {
				ok++
			}
		}
		t.Logf("%d/3 for %q -> %q", ok, k.msg[:30], last)
		if ok < 2 {
			t.Errorf("wrong language for %q: %q", k.msg, last)
		}
	}
}
