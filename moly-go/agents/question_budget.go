package agents

import (
	"log"

	"moly/models"
)

// QuestionBudget is how many questions that only improve the result Moly asks in one conversation before it gives the result (user
// decision, 2026-10-15: after eight turns of questions a user who had asked for a first message got none; "too many questions").
// It counts the questions Moly actually asked, from the kind recorded on each of its replies: a clarity, gap, principle, persistent,
// intent or Socratic question. Not counted, and never held back by the budget: the question for a name, the confirmation of a doubtful
// fact and anything to do with risk (the safety refusal and alert).
const QuestionBudget = 3

// optionalQuestionKinds are the reply kinds that only make the result better.
var optionalQuestionKinds = map[ReplyKind]bool{
	KindClarity: true, KindGapQuestion: true, KindPrinciple: true, KindPersistent: true, KindIntentCheck: true, KindSocratic: true,
}

// questionsAsked counts the optional questions among Moly's earlier replies of this conversation.
func (ca *conversationAgent) questionsAsked(ctx *models.Context) int {
	n := 0
	for _, m := range ctx.ConversationHistory {
		if m.Role != "assistant" || m.Metadata == nil {
			continue
		}
		kind, _ := m.Metadata["replyExit"].(string)
		if optionalQuestionKinds[ReplyKind(kind)] {
			n++
		}
	}
	return n
}

// budgetSpent says whether the questions that only improve the result have used up their budget.
func (ca *conversationAgent) budgetSpent(ctx *models.Context) bool {
	n := ca.questionsAsked(ctx)
	if n >= QuestionBudget {
		log.Printf("[ConversationAgent] %d questions asked in this conversation: questions that only improve the result wait, the result is given", n)
		return true
	}
	return false
}
