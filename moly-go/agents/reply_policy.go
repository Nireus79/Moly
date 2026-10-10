package agents

// ReplyKind is the one kind of reply a message gets (ORCHESTRATOR_DESIGN.md section 4, stage 8).
type ReplyKind string

const (
	KindDeny         ReplyKind = "deny"
	KindSafetyAlert  ReplyKind = "safety_alert"
	KindGreeting     ReplyKind = "greeting"
	KindNameQuestion ReplyKind = "name_question"
	KindConfirm      ReplyKind = "confirm_question"
	KindClarity      ReplyKind = "clarity_question"
	KindGapQuestion  ReplyKind = "gap_question"
	KindPrinciple    ReplyKind = "principle_question"
	KindPersistent   ReplyKind = "persistent_question"
	KindTopicShift   ReplyKind = "topic_shift_question"
	KindContact      ReplyKind = "contact_question"
	KindIntentCheck  ReplyKind = "intent_check_question"
	KindSocratic     ReplyKind = "socratic_question"
	KindHelp         ReplyKind = "help"
)

// ReplyFacts are the facts the decision is made from. Nothing else is read.
type ReplyFacts struct {
	Deny           bool // safety verdict: deny
	SafetyAlert    bool // the safety check produced an alert for the user
	Greeting       bool // the message intent is a greeting
	NameUnanswered bool // a person is waiting to be asked for a name
	NameAnswered   bool // this message answered a name question (does not change the kind by itself)
	ConfirmFact    bool // a fact was understood with doubt and not saved: confirm it first
	Clarity        bool // the message is too unclear to answer: a clarifying question comes first
	OpenGap        bool // a goal-blocking gap is open
	PrincipleOpen  bool // a principle is engaged and needs a question
	Persistent     bool // the user insists after a concern was raised
	TopicShift     bool // the user changed subject
	ContactUnclear bool // a message to a person is missing what it needs, or its intent is unclear
	IntentUnclear  bool // what the user wants is unclear
	DeepenAllowed  bool // the deepening gates allowed a Socratic question
	// Bypass: the user pressed the skip button. It skips the questions that only make the result better (a name, a gap,
	// a topic shift, an unclear contact, a Socratic question). The press carries no content of its own, so the checks that
	// judge the clarity and the intent of the message do not apply to it either: the request is in the conversation, and
	// the button is only offered after questions Moly could ask because it understood the request. It never skips safety,
	// a doubtful fact, an engaged principle or a raised concern: those are risk, and the user's choice does not remove risk.
	Bypass bool
}

// ReplyPlan is the decision: the kind, and the first rule that gave it.
type ReplyPlan struct {
	Kind   ReplyKind
	Reason string
}

// DecideReply chooses the reply kind. It is a pure function: the same facts always give the same plan.
// The rules are checked in this order; the first one that holds decides.
func DecideReply(f ReplyFacts) ReplyPlan {
	switch {
	case f.Deny:
		return ReplyPlan{KindDeny, "safety verdict is deny"}
	case f.SafetyAlert:
		return ReplyPlan{KindSafetyAlert, "the safety check raised an alert: show it"}
	case f.Greeting:
		return ReplyPlan{KindGreeting, "the message is a greeting: no goal, no person, no questions"}
	case f.NameUnanswered && !f.Bypass:
		return ReplyPlan{KindNameQuestion, "a person has no name yet: ask for it before anything else"}
	case f.ConfirmFact:
		return ReplyPlan{KindConfirm, "a fact is in doubt and was not saved: confirm it before anything else"}
	case f.Clarity && !f.Bypass:
		return ReplyPlan{KindClarity, "the message is unclear: ask what is unclear before anything else"}
	case f.OpenGap && !f.Bypass:
		return ReplyPlan{KindGapQuestion, "a goal-blocking gap is open: ask it before reflective questions"}
	case f.PrincipleOpen:
		return ReplyPlan{KindPrinciple, "a principle is engaged and nothing blocks the goal: ask about it"}
	case f.Persistent:
		return ReplyPlan{KindPersistent, "the user insists after a concern: ask a deeper question"}
	case f.TopicShift && !f.Bypass:
		return ReplyPlan{KindTopicShift, "the subject changed: ask whether it is connected"}
	case f.ContactUnclear && !f.Bypass:
		return ReplyPlan{KindContact, "a message to a person lacks what it needs: ask before writing it"}
	case f.IntentUnclear && !f.Bypass:
		return ReplyPlan{KindIntentCheck, "what the user wants is unclear: ask what they are trying to figure out"}
	case f.DeepenAllowed && !f.Bypass:
		return ReplyPlan{KindSocratic, "the deepening gates passed"}
	default:
		return ReplyPlan{KindHelp, "nothing blocks help"}
	}
}

// factProbe finds out one fact. A probe may call the model, so probes run lazily, in precedence order.
type factProbe struct {
	name string
	set  func(*ReplyFacts)
}

// decideByProbes asks DecideReply after each probe and stops at the first plan that is not help.
// Probes after the deciding one never run, so a decided reply costs no extra model calls.
// It returns the plan and the name of the probe that decided it ("" when the base facts decided).
func decideByProbes(base ReplyFacts, probes []factProbe) (ReplyPlan, string) {
	f := base
	plan := DecideReply(f)
	if plan.Kind != KindHelp {
		return plan, ""
	}
	for _, p := range probes {
		p.set(&f)
		plan = DecideReply(f)
		if plan.Kind != KindHelp {
			return plan, p.name
		}
	}
	return plan, ""
}

// IsQuestionKind reports whether a reply of this kind asks the user something. A reply asks at most one thing.
func IsQuestionKind(kind ReplyKind) bool {
	switch kind {
	case KindNameQuestion, KindConfirm, KindClarity, KindGapQuestion, KindPrinciple, KindPersistent,
		KindTopicShift, KindContact, KindIntentCheck, KindSocratic:
		return true
	}
	return false
}

// AskedQuestion reports whether the reply metadata records that the agent's reply already asks a question.
func AskedQuestion(metadata map[string]interface{}) bool {
	for _, key := range []string{"replyExit", "replyPolicy"} {
		if v, ok := metadata[key]; ok {
			if s, ok := v.(string); ok && IsQuestionKind(ReplyKind(s)) {
				return true
			}
			if k, ok := v.(ReplyKind); ok && IsQuestionKind(k) {
				return true
			}
		}
	}
	return false
}
