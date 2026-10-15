package agents

import (
	"log"

	"moly/models"
)

// QuestionBudget is how many questions Moly asks in one conversation before it stops asking questions that only make the result
// better and gives the result (user decision, 2026-10-15: after eight turns of questions a user who had asked for a first message
// got none; "too many questions"). It counts every question stored for the conversation, answered, open or skipped. Risk is not
// counted against it: safety, a doubtful fact, an engaged principle and a raised concern are asked whatever has been asked before.
const QuestionBudget = 3

// questionsAsked counts the questions stored for this conversation. A store that cannot be read counts none.
func (ca *conversationAgent) questionsAsked(ctx *models.Context) int {
	if ca.db == nil || ctx.ConversationID == "" {
		return 0
	}
	repo := ca.db.GetClarificationQuestionRepository()
	if repo == nil {
		return 0
	}
	questions, err := repo.GetConversationQuestions(ctx.ConversationID)
	if err != nil {
		return 0
	}
	return len(questions)
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
