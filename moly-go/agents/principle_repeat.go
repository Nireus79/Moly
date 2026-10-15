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

// lastQuestionOf is the question a Moly message ended with: the sentence that holds its last question mark. "" when the message asks nothing.
func lastQuestionOf(text string) string {
	t := strings.TrimSpace(text)
	marks := questionMarkIndexes(t)
	if len(marks) == 0 {
		return ""
	}
	end := marks[len(marks)-1]
	head := t[:end]
	start := strings.LastIndexAny(head, ".!\n") + 1
	if len(marks) > 1 {
		if prev := marks[len(marks)-2]; prev+1 > start {
			start = prev + 1
		}
	}
	return strings.TrimSpace(t[start : end+1])
}

// questionAskedBefore says whether Moly already asked this question in the conversation, in these words or in others. Only the questions
// of Moly's earlier messages are compared, not the sentences around them (found in dynamic runs: a gap question came back two turns later
// in other words, and the repeat check on the gap descriptions did not see it).
func (ca *conversationAgent) questionAskedBefore(question string, history []models.Message, analysisCtx *models.AnalysisContext) bool {
	q := strings.TrimSpace(question)
	var asked []string
	for _, m := range history {
		if m.Role != "assistant" {
			continue
		}
		if prev := lastQuestionOf(m.Content); prev != "" {
			if strings.EqualFold(prev, lastQuestionOf(q)) {
				return false // the very same question is an open one asked again on purpose (the user did not answer it); only other words are a repeat
			}
			asked = append(asked, prev)
		}
	}
	if len(asked) == 0 || q == "" {
		return false
	}
	var dialogue *models.DialoguePacket
	if analysisCtx != nil {
		dialogue = analysisCtx.Dialogue
	}
	return len(DropRepeatedGaps(context.Background(), ca.llmClient, []tools.Gap{{Type: "asked", Description: lastQuestionOf(q)}}, dialogue, asked)) == 0
}
