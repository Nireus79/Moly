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
// user's message answers. The result has one entry per item. The model names the NUMBERS of the items it answers, so a
// wrong count of entries cannot make the answer unusable (found live: it returned two booleans for one question). An item is
// stored as the gap description or as the question, so each is "something Moly still needs to know". Anything unreadable or
// failed answers none: an item is only closed on a clear answer.
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
		SystemPrompt: "You judge whether the user's message gives information Moly still needs.",
		UserPrompt: "The user's message: \"" + message + "\"\n\nWhat Moly still needs to know (numbered; each is a question or a topic):\n" + list + "\n" +
			"Which numbered items does the message clearly answer (it gives the information the item asks for)? List only those. " +
			"Respond with ONLY JSON: {\"answered\": [numbers]} , for example {\"answered\": [1]} or {\"answered\": []}",
		MaxTokens:   60,
		Temperature: 0.1,
	}
	resp, err := llm.Call(ctx, req)
	if err != nil {
		log.Printf("[PendingQuestions] judgement failed: %v (none closed)", err)
		return none
	}
	var out struct {
		Answered []int `json:"answered"`
	}
	if err := tools.SafeJSONParse("PendingQuestions", []byte(resp.Content), &out); err != nil {
		log.Printf("[PendingQuestions] judgement not readable (none closed): %v", err)
		return none
	}
	answered := make([]bool, len(pending))
	for _, n := range out.Answered {
		if n >= 1 && n <= len(pending) {
			answered[n-1] = true
		}
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
