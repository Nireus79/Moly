package agents

import "testing"

func TestCanSkip(t *testing.T) {
	for _, k := range []ReplyKind{KindNameQuestion, KindGapQuestion, KindTopicShift, KindContact, KindSocratic} {
		if !CanSkip(k, 0.5) || CanSkip(k, 0) {
			t.Errorf("%s: skippable only once something was answered", k)
		}
	}
	for _, k := range []ReplyKind{KindDeny, KindSafetyAlert, KindGreeting, KindConfirm, KindClarity, KindPrinciple, KindPersistent, KindIntentCheck, KindHelp} {
		if CanSkip(k, 1) {
			t.Errorf("%s must never be skippable", k)
		}
	}
}
