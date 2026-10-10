package agents

import (
	"log"

	"moly/database"
)

// GapMaturity is the one maturity value (user decision, 2026-10-10): how much of what Moly needed to ask has been
// answered. It counts logical gaps: answered questions over answered plus still-open ones (asked and waiting, or
// skipped by the user: a skipped question is still a gap). With nothing asked yet
// nothing is known, so it is 0. Moly asks every open gap before it gives what the user asked for.
func GapMaturity(answered, open int) float64 {
	if answered <= 0 {
		return 0
	}
	return float64(answered) / float64(answered+open)
}

// ConversationGapMaturity counts a conversation's stored questions and returns its maturity. A store that cannot be
// read gives 0: Moly then knows nothing and keeps asking, which is the safe side.
func ConversationGapMaturity(repo *database.ClarificationQuestionRepository, conversationID string) float64 {
	if repo == nil || conversationID == "" {
		return 0
	}
	questions, err := repo.GetConversationQuestions(conversationID)
	if err != nil {
		return 0
	}
	answered, open := 0, 0
	for _, q := range questions {
		switch q.Status {
		case "answered":
			answered++
		case "active", "skipped": // a skipped question is still an unresolved gap
			open++
		}
	}
	m := GapMaturity(answered, open)
	log.Printf("[GapMaturity] answered=%d open=%d -> %.2f", answered, open, m)
	return m
}
