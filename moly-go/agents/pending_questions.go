package agents

import (
	"context"
	"log"
	"sort"
	"strconv"
	"strings"

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
