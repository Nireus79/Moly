package agents

import (
	"context"
	"log"
	"sort"
	"strconv"
	"strings"

	"moly/database"
	"moly/models"
	"moly/tools"
)

// Pending questions (ORCHESTRATOR_DESIGN.md, "Pending questions"): a question the user skipped is saved as pending. It is
// not forgotten and not asked again at once: on each later message the model says whether the new message answered it;
// if not, it stays a candidate gap, ranked together with the new gaps by how much each blocks the goal.

// JudgePendingAnswered asks the model which of the open items (a question asked and not answered yet, or a skipped one) the
// user's message answers. The result has one entry per item. The items are put to the model as what they are: questions Moly
// asked, and the user's reply; the model answers per question number in an object, so a wrong count of entries cannot make the
// answer unusable. (Two earlier forms failed on the live model: a list of booleans came back with one entry too many, and a
// "list only those you are sure of" prompt answered "none" to a plain answer, in 6 of 12 replays. This form was right in 20 of
// 20.) Anything unreadable or failed answers none: an item is only closed on a judgement.
func JudgePendingAnswered(ctx context.Context, llm tools.LLMProvider, message string, pending []string) []bool {
	none := make([]bool, len(pending))
	if llm == nil || len(pending) == 0 || strings.TrimSpace(message) == "" {
		return none
	}
	list := ""
	for i, q := range pending {
		list += strconv.Itoa(i+1) + ". " + q + "\n"
	}
	req := &tools.LLMRequest{
		SystemPrompt: "You judge whether a reply answers questions.",
		UserPrompt: "Moly asked the user these questions:\n" + list + "\nThe user replied: \"" + message + "\"\n\n" +
			"For each question, did the user's reply answer it, even briefly or only in part of the reply?\n" +
			"Respond with ONLY JSON that has one entry per question number, for example {\"1\": true, \"2\": false}",
		MaxTokens:   80,
		Temperature: 0.1,
	}
	resp, err := llm.Call(ctx, req)
	if err != nil {
		log.Printf("[PendingQuestions] judgement failed: %v (none closed)", err)
		return none
	}
	var out map[string]tools.LenientBool
	if err := tools.SafeJSONParse("PendingQuestions", []byte(resp.Content), &out); err != nil {
		log.Printf("[PendingQuestions] judgement not readable (none closed): %v", err)
		return none
	}
	answered := make([]bool, len(pending))
	for i := range pending {
		answered[i] = bool(out[strconv.Itoa(i+1)])
	}
	return answered
}

// JudgePendingStillWorthAsking asks the model, for each pending question, whether it is still worth asking given everything said
// since: a question is not worth asking when the user has already told the answer, when later messages made it moot, or when the
// conversation has moved to something it no longer serves. The result has one entry per question, true = still worth asking.
// Anything unreadable or failed keeps every question: a question the user may need is not dropped on a guess. The answer is a
// keyed object ({"1": true}) like the other per-question judges. Without a dialogue to read there is nothing to judge.
func JudgePendingStillWorthAsking(ctx context.Context, llm tools.LLMProvider, dialogue *models.DialoguePacket, message string, pending []string) []bool {
	keep := make([]bool, len(pending))
	for i := range keep {
		keep[i] = true
	}
	block := dialogue.Text()
	if llm == nil || len(pending) == 0 || block == "" {
		return keep
	}
	list := ""
	for i, q := range pending {
		list += strconv.Itoa(i+1) + ". " + q + "\n"
	}
	resp, err := llm.Call(ctx, &tools.LLMRequest{
		SystemPrompt: "You judge whether an unanswered question is still worth asking.",
		UserPrompt: block + "The user's newest message: \"" + message + "\"\n\nMoly asked the user these questions earlier and the user did not answer them:\n" + list + "\n" +
			"For each question: is it still worth asking now? Answer true if the answer would still help Moly with the goal. " +
			"Answer false if the user has already told the answer (in any message, in any words), if what the user said since makes the question moot, " +
			"or if the conversation has moved on to something the question no longer serves. " +
			"Respond with ONLY JSON that has one entry per question number, for example {\"1\": true, \"2\": false}",
		MaxTokens:   80,
		Temperature: 0.1,
	})
	if err != nil {
		log.Printf("[PendingQuestions] relevance judgement failed: %v (all kept)", err)
		return keep
	}
	var out map[string]tools.LenientBool
	if err := tools.SafeJSONParse("PendingRelevance", []byte(resp.Content), &out); err != nil || len(out) == 0 {
		log.Printf("[PendingQuestions] relevance judgement not readable (all kept)")
		return keep
	}
	for i := range pending {
		if v, ok := out[strconv.Itoa(i+1)]; ok {
			keep[i] = bool(v)
		}
	}
	return keep
}

// severityOfPriority and priorityOfSeverity convert between a stored question priority and a gap severity.
func severityOfPriority(p int) string {
	switch p {
	case 1:
		return "high"
	case 3:
		return "low"
	}
	return "medium"
}

func priorityOfSeverity(s string) int {
	switch severityRank(s) {
	case 0:
		return 1
	case 2:
		return 3
	}
	return 2
}

// severityRank orders severities: lower is more important. Unknown counts as medium.
func severityRank(s string) int {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "critical", "high":
		return 0
	case "low":
		return 2
	}
	return 1
}

// rankGapsBySeverity puts the gaps that block the goal most first (the model rates each gap's severity against the goal).
// Equal severity keeps the order given, then the higher confidence goes first. New gaps are listed before pending ones, so
// at equal severity a new gap is asked before one the user already skipped.
func rankGapsBySeverity(gaps []tools.Gap) []tools.Gap {
	out := append([]tools.Gap(nil), gaps...)
	sort.SliceStable(out, func(i, j int) bool {
		ri, rj := severityRank(out[i].Severity), severityRank(out[j].Severity)
		if ri != rj {
			return ri < rj
		}
		return out[i].Confidence > out[j].Confidence
	})
	return out
}

// mergePendingSkipped brings the questions the user skipped, or left unanswered, back as candidate gaps. Which of them the
// latest message answers was already judged when the message arrived (main.go), so they are only merged here. The press
// of the skip button itself merges nothing: the question is not asked in the reply that skipped it.
func (l4 *Layer4GapDetector) mergePendingSkipped(ctx context.Context, lc *tools.LayerContext, gaps []tools.Gap) []tools.Gap {
	if l4.clarificationHistory == nil || lc.Analysis == nil || lc.Analysis.SkipPressed {
		return gaps
	}
	pending, err := l4.clarificationHistory.PendingSkipped(lc.UserID, lc.ConversationID)
	if err != nil {
		log.Printf("[Layer4] Warning: could not load pending questions: %v", err)
		return gaps
	}
	if len(pending) == 0 {
		return gaps
	}
	// A pending question comes back only while it is still worth asking: one the user's later words answered or made moot, or
	// that the conversation has left behind, is cancelled (kept in the history, no longer open, no longer counted in maturity).
	texts := make([]string, len(pending))
	for i, q := range pending {
		texts[i] = q.Text
	}
	worth := JudgePendingStillWorthAsking(ctx, l4.llmClient, lc.Analysis.Dialogue, lc.Analysis.CurrentMessage, texts)
	var stillOpen []database.PendingQuestion
	for i, q := range pending {
		if worth[i] {
			stillOpen = append(stillOpen, q)
			continue
		}
		if err := l4.clarificationHistory.SetQuestionStatus(lc.UserID, q.ID, "cancelled", 0); err != nil {
			log.Printf("[Layer4] Warning: could not cancel a question that is no longer relevant: %v", err)
			stillOpen = append(stillOpen, q)
			continue
		}
		log.Printf("[Layer4] A pending question is no longer relevant: cancelled")
	}
	pending = stillOpen
	if len(pending) == 0 {
		return gaps
	}
	merged := 0
	have := map[string]bool{}
	for _, g := range gaps {
		have[g.Description] = true
	}
	for _, q := range pending {
		if have[q.Text] {
			continue // the model produced the same question again: one candidate, not two
		}
		gaps = append(gaps, tools.Gap{
			Type: "pending", Description: q.Text, Severity: severityOfPriority(q.Priority), Confidence: 0.8,
			Assumption: "the user skipped this question earlier", GoalTarget: "current_goal", SourceFix: "pending",
		})
		merged++
	}
	if merged > 0 {
		log.Printf("[Layer4] %d pending question(s) are candidate gaps again", merged)
	}
	return gaps
}
