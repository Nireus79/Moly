package agents

import (
	"context"
	"strings"

	"moly/models"
	"moly/tools"
)

// principleAskedBefore says whether a principle question was already asked in this conversation, in these words or in others (judged by
// the model, as for the gaps). Found live, 2026-10-15: the same fixed question came back after the user had answered it at length, and
// again after "Go ahead with what you have"; a question asked once is not asked again.
func (ca *conversationAgent) principleAskedBefore(question string, history []models.Message, analysisCtx *models.AnalysisContext) bool {
	var asked []string
	for _, m := range history {
		if m.Role == "assistant" && strings.TrimSpace(m.Content) != "" {
			if strings.Contains(strings.ToLower(m.Content), strings.ToLower(strings.TrimSpace(question))) {
				return true // the very words
			}
			asked = append(asked, m.Content)
		}
	}
	if len(asked) == 0 {
		return false
	}
	var dialogue *models.DialoguePacket
	if analysisCtx != nil {
		dialogue = analysisCtx.Dialogue
	}
	return len(DropRepeatedGaps(context.Background(), ca.llmClient, []tools.Gap{{Type: "principle", Description: question}}, dialogue, asked)) == 0
}
