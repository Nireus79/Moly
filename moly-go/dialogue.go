package main

import (
	"strings"

	"moly/database"
	"moly/models"
)

// fillDialogue adds to the packet what the builder cannot know: the locked goal and the questions still open. Both come from
// what is stored; no model is called. (What Moly learned earlier lives in the summary.)
func fillDialogue(a *models.AnalysisContext, open []string) {
	if a == nil {
		return
	}
	if a.Dialogue == nil {
		a.Dialogue = &models.DialoguePacket{}
	}
	p := a.Dialogue
	p.Goal = a.PrimaryGoal
	if p.Goal == "" {
		p.Goal = a.PreviousGoal
	}
	if len(open) > 3 {
		open = open[:3]
	}
	p.Open = open
}

// memoryFactLines writes what the memory stored of a conversation as lines for the dialogue: the facts, and the traits of each person. At most the
// latest 40, each cut short.
func memoryFactLines(snap models.Memory) []string {
	who := map[string]string{models.MemoryUser: "The user"}
	for _, p := range snap.People {
		name := p.Name
		if name == "" {
			name = "the " + p.Label
		}
		who[p.ID] = name
	}
	cut := func(s string) string {
		if r := []rune(strings.TrimSpace(s)); len(r) > 200 {
			return string(r[:200]) + " ..."
		}
		return strings.TrimSpace(s)
	}
	var lines []string
	for _, p := range snap.People {
		for _, tr := range p.Traits {
			lines = append(lines, who[p.ID]+": "+cut(tr))
		}
	}
	for _, f := range snap.Facts {
		owner := who[f.Person]
		if owner == "" {
			owner = "The user"
		}
		lines = append(lines, owner+": "+cut(f.Fact))
	}
	if len(lines) > 40 {
		lines = lines[len(lines)-40:]
	}
	return lines
}

// addMemoryFacts puts the memory's facts into the dialogue (memory mode only).
func (srv *APIServer) addMemoryFacts(a *models.AnalysisContext, userID, conversationID string) {
	if a == nil || a.Dialogue == nil || conversationID == "" || srv.database == nil {
		return
	}
	snap, err := database.NewMemoryStore(srv.database.GetConnection()).Snapshot(userID, conversationID)
	if err != nil {
		return
	}
	a.Dialogue.Facts = memoryFactLines(snap)
	// the questions Moly asked that are still open are the memory's, not the old clarification store's
	var open []string
	for _, q := range snap.Questions {
		if q.State == models.QuestionOpen && q.Kind != "confirm_goal" {
			open = append(open, q.Text)
		}
	}
	if len(open) > 3 {
		open = open[len(open)-3:]
	}
	a.Dialogue.Open = open
}
