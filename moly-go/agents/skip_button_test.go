package agents

import "testing"

func TestCanSkip(t *testing.T) {
	for _, k := range []ReplyKind{KindNameQuestion, KindGapQuestion} {
		if !CanSkip(k, 0.5) || CanSkip(k, 0) {
			t.Errorf("%s: skippable only once something was answered", k)
		}
	}
	if !CanSkip(KindSocratic, 0) || !CanSkip(KindSocratic, 1) {
		t.Error("a reflective question is skippable at any time")
	}
	for _, k := range []ReplyKind{KindDeny, KindSafetyAlert, KindGreeting, KindConfirm, KindClarity, KindPrinciple, KindPersistent, KindIntentCheck, KindHelp} {
		if CanSkip(k, 1) {
			t.Errorf("%s must never be skippable", k)
		}
	}
}
