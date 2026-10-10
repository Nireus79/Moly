package agents

import "moly/models"

// IntentSource is what the intent stage needs: the language-model intent detector.
type IntentSource interface {
	DetectIntentWithLLM(message string, history []models.Message) IntentAnalysis
}

// IntentDecision is the intent of one message. It is decided once, right after extraction (ORCHESTRATOR_DESIGN.md step 3).
type IntentDecision struct {
	Analysis IntentAnalysis
	Greeting bool // the message is a greeting: it has no goal and names no one
	// AnswerOnly: the message only answers the question Moly just asked. An answer states no goal, so it
	// continues the locked goal; a goal the extractor invents from it is dropped.
	AnswerOnly bool
}

// DecideIntent asks the model for the message intent. With no detector or no message the intent is unknown, never a greeting.
// history is the conversation so far, oldest first, without the current message. A message is an answer only when Moly's
// last message was in the history: the model's claim alone is not enough.
func DecideIntent(src IntentSource, message string, history []models.Message) IntentDecision {
	var a IntentAnalysis
	if src != nil && message != "" {
		a = src.DetectIntentWithLLM(message, history)
	}
	answerOnly := a.AnswersQuestion && a.Intent != IntentGreet && len(history) > 0 && history[len(history)-1].Role == "assistant"
	return IntentDecision{Analysis: a, Greeting: a.Intent == IntentGreet, AnswerOnly: answerOnly}
}

// HoldDoubtfulGoal takes a goal Moly is not sure of out of the extraction so that it is not locked, and returns it
// so the user can be asked to confirm it. It returns "" when there is no goal or the goal is not in doubt.
func HoldDoubtfulGoal(ec *models.ExtractedContext) string {
	if ec == nil || ec.Intention == "" || !FactInDoubt(ec.IntentionConfidence) {
		return ""
	}
	held := ec.Intention
	ec.Intention = ""
	return held
}

// StripAnswerGoal removes a goal from a message that only answers a question. The person it names is kept.
func (d IntentDecision) StripAnswerGoal(ec *models.ExtractedContext) {
	if !d.AnswerOnly || ec == nil {
		return
	}
	ec.Intention = ""
}

// StripGreeting removes what a greeting cannot contain: a goal and a person. Other messages are left alone.
func (d IntentDecision) StripGreeting(ec *models.ExtractedContext) {
	if !d.Greeting || ec == nil {
		return
	}
	ec.Intention = ""
	ec.Contact = nil
}

// WithoutGoalEntities returns the entities without goals. A greeting must not leave a goal behind to be locked.
func WithoutGoalEntities(entities []models.ExtractedEntity) []models.ExtractedEntity {
	kept := make([]models.ExtractedEntity, 0, len(entities))
	for _, e := range entities {
		if e.Type == "goal" || e.Type == "goal_component" {
			continue
		}
		kept = append(kept, e)
	}
	return kept
}
