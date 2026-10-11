package main

import "moly/models"

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
