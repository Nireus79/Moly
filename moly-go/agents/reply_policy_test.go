package agents

import "testing"

// One row per precedence case. The rows are read top to bottom: the first true fact decides.
func TestDecideReplyTable(t *testing.T) {
	cases := []struct {
		name  string
		facts ReplyFacts
		want  ReplyKind
	}{
		{"nothing set: help", ReplyFacts{}, KindHelp},
		{"deny wins over everything", ReplyFacts{Deny: true, Greeting: true, NameUnanswered: true, OpenGap: true, DeepenAllowed: true}, KindDeny},
		{"greeting wins over name, gap and deepening", ReplyFacts{Greeting: true, NameUnanswered: true, OpenGap: true, DeepenAllowed: true}, KindGreeting},
		{"unanswered name wins over gap and deepening", ReplyFacts{NameUnanswered: true, OpenGap: true, DeepenAllowed: true}, KindNameQuestion},
		{"clarity wins over a gap", ReplyFacts{Clarity: true, OpenGap: true, PrincipleOpen: true}, KindClarity},
		{"name wins over clarity", ReplyFacts{NameUnanswered: true, Clarity: true}, KindNameQuestion},
		{"gap wins over a principle concern", ReplyFacts{OpenGap: true, PrincipleOpen: true, Persistent: true, TopicShift: true}, KindGapQuestion},
		{"principle wins over persistence", ReplyFacts{PrincipleOpen: true, Persistent: true, TopicShift: true}, KindPrinciple},
		{"persistence wins over a topic shift", ReplyFacts{Persistent: true, TopicShift: true, DeepenAllowed: true}, KindPersistent},
		{"topic shift wins over deepening", ReplyFacts{TopicShift: true, DeepenAllowed: true}, KindTopicShift},
		{"safety alert wins over greeting", ReplyFacts{SafetyAlert: true, Greeting: true}, KindSafetyAlert},
		{"deny wins over safety alert", ReplyFacts{Deny: true, SafetyAlert: true}, KindDeny},
		{"topic shift wins over contact question", ReplyFacts{TopicShift: true, ContactUnclear: true}, KindTopicShift},
		{"contact question wins over intent check", ReplyFacts{ContactUnclear: true, IntentUnclear: true}, KindContact},
		{"intent check wins over deepening", ReplyFacts{IntentUnclear: true, DeepenAllowed: true}, KindIntentCheck},
		{"open gap wins over deepening", ReplyFacts{OpenGap: true, DeepenAllowed: true}, KindGapQuestion},
		{"deepening when nothing is open", ReplyFacts{DeepenAllowed: true}, KindSocratic},
		{"skip button skips the name question", ReplyFacts{Bypass: true, NameUnanswered: true}, KindHelp},
		{"skip button skips a gap, a shift, a contact question and deepening", ReplyFacts{Bypass: true, OpenGap: true, TopicShift: true, ContactUnclear: true, DeepenAllowed: true}, KindHelp},
		{"skip button does not skip a principle concern", ReplyFacts{Bypass: true, OpenGap: true, PrincipleOpen: true}, KindPrinciple},
		{"skip button does not skip a raised concern", ReplyFacts{Bypass: true, Persistent: true}, KindPersistent},
		{"skip button does not skip a doubtful fact", ReplyFacts{Bypass: true, ConfirmFact: true, OpenGap: true}, KindConfirm},
		{"skip button: the press has no content to find unclear", ReplyFacts{Bypass: true, Clarity: true, IntentUnclear: true}, KindHelp},
		{"an unclear message without a press is still asked", ReplyFacts{Clarity: true}, KindClarity},
		{"an unclear intent without a press is still asked", ReplyFacts{IntentUnclear: true}, KindIntentCheck},
		{"skip button does not skip safety", ReplyFacts{Bypass: true, SafetyAlert: true}, KindSafetyAlert},
		{"name answered and nothing else open: help, not a question", ReplyFacts{NameAnswered: true}, KindHelp},
		{"name answered does not repeat the name question", ReplyFacts{NameAnswered: true, NameUnanswered: false}, KindHelp},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := DecideReply(c.facts)
			if got.Kind != c.want {
				t.Fatalf("kind = %s, want %s (reason: %s)", got.Kind, c.want, got.Reason)
			}
			if got.Reason == "" {
				t.Fatal("every plan must say why")
			}
		})
	}
}

// The decision is deterministic: the same facts always give the same plan.
func TestDecideReplyIsPure(t *testing.T) {
	f := ReplyFacts{NameUnanswered: true, OpenGap: true}
	first := DecideReply(f)
	for i := 0; i < 100; i++ {
		if DecideReply(f) != first {
			t.Fatal("DecideReply changed its answer for the same facts")
		}
	}
}

// Probes run in order and stop at the first fact that decides; later probes never run.
func TestDecideByProbesStopsAtFirstDecision(t *testing.T) {
	ran := []string{}
	probe := func(name string, set func(*ReplyFacts)) factProbe {
		return factProbe{name, func(f *ReplyFacts) { ran = append(ran, name); set(f) }}
	}
	plan, by := decideByProbes(ReplyFacts{}, []factProbe{
		probe("clarity", func(f *ReplyFacts) {}),
		probe("gap", func(f *ReplyFacts) { f.OpenGap = true }),
		probe("principle", func(f *ReplyFacts) { f.PrincipleOpen = true }),
	})
	if plan.Kind != KindGapQuestion || by != "gap" {
		t.Fatalf("got %s by %q", plan.Kind, by)
	}
	if len(ran) != 2 {
		t.Fatalf("the principle probe must not run once the gap decided; ran %v", ran)
	}
}

func TestDecideByProbesBaseFactsSkipAllProbes(t *testing.T) {
	ran := 0
	plan, by := decideByProbes(ReplyFacts{Greeting: true}, []factProbe{{"x", func(f *ReplyFacts) { ran++ }}})
	if plan.Kind != KindGreeting || by != "" || ran != 0 {
		t.Fatalf("a greeting decides by itself: %s by %q, probes run %d", plan.Kind, by, ran)
	}
}

func TestIsQuestionKindAndAskedQuestion(t *testing.T) {
	for _, k := range []ReplyKind{KindNameQuestion, KindClarity, KindGapQuestion, KindPrinciple, KindPersistent, KindTopicShift, KindContact, KindIntentCheck, KindSocratic} {
		if !IsQuestionKind(k) {
			t.Errorf("%s asks a question", k)
		}
	}
	for _, k := range []ReplyKind{KindDeny, KindSafetyAlert, KindGreeting, KindHelp} {
		if IsQuestionKind(k) {
			t.Errorf("%s does not ask a question", k)
		}
	}
	if !AskedQuestion(map[string]interface{}{"replyExit": "gap_question"}) || !AskedQuestion(map[string]interface{}{"replyPolicy": KindNameQuestion}) {
		t.Error("a recorded question kind must count")
	}
	if AskedQuestion(map[string]interface{}{"replyExit": "help"}) || AskedQuestion(nil) {
		t.Error("help or no record is not a question")
	}
}
