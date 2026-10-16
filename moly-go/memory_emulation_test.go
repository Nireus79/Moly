package main

import (
	"encoding/json"
	"strings"
	"unicode"

	"moly/models"
)

// emulateMemoryUpdate answers the memory update call for the scripted scenarios that were written for the old pipeline. It reads the same
// scripts those scenarios already have (the extraction of the message, the gap script, the clarity marker) and turns them into what the update
// call would answer, so a scenario keeps testing the product rule it was written for through the memory path. A scenario can still script the
// update call itself (scriptedLLM.memoryUpdate).
func emulateMemoryUpdate(prompt string, l scriptedLLM) string {
	var mem models.Memory
	if i := strings.Index(prompt, "Memory of this conversation:\n"); i >= 0 {
		rest := prompt[i+len("Memory of this conversation:\n"):]
		if j := strings.Index(rest, "\n\nThe last messages"); j >= 0 {
			json.Unmarshal([]byte(rest[:j]), &mem)
		}
	}
	msg := ""
	if i := strings.LastIndex(prompt, "The new message from the user:\n"); i >= 0 {
		rest := prompt[i+len("The new message from the user:\n"):]
		if j := strings.Index(rest, "\n\nOutput only"); j >= 0 {
			json.Unmarshal([]byte(rest[:j]), &msg)
		}
	}
	var ext map[string]interface{}
	json.Unmarshal([]byte(extractionFor("Message: \""+msg+"\"\n")), &ext)

	type op map[string]interface{}
	var ops []op
	nextNew := 0
	byName := map[string]string{}
	byLabel := map[string]string{}
	var waiting []models.MemoryPerson
	for _, p := range mem.People {
		if p.Name != "" {
			byName[strings.ToLower(p.Name)] = p.ID
		}
		if p.Label != "" {
			byLabel[strings.ToLower(p.Label)] = p.ID
		}
		if p.NameStatus == "unnamed" || p.NameStatus == "asked" {
			waiting = append(waiting, p)
		}
	}
	var lastUnnamed, lastUnnamedLabel string
	if len(waiting) > 0 {
		lastUnnamed = waiting[len(waiting)-1].ID
		lastUnnamedLabel = waiting[len(waiting)-1].Label
	}
	var openName string
	for _, q := range mem.Questions {
		if q.Kind == "name" && q.State == "open" {
			openName = q.ID
		}
	}
	answeredName := false
	switch {
	case strings.Contains(msg, "NAME-IS-DANA") && len(waiting) > 0:
		ops = append(ops, op{"op": "name_person", "person": waiting[0].ID, "name": "Dana", "evidence": "NAME-IS-DANA"})
		answeredName = true
	case strings.Contains(msg, "DECLINE-NAME") && openName != "":
		ops = append(ops, op{"op": "close_question", "question": openName, "state": "declined", "evidence": "DECLINE-NAME"})
		answeredName = true
	}
	if marker := map[bool]string{true: "ANSWERS-PENDING", false: "Quite informal"}[strings.Contains(msg, "ANSWERS-PENDING")]; strings.Contains(msg, marker) {
		for _, q := range mem.Questions {
			if q.State == "open" && (q.Kind == "gap" || q.Kind == "clarity" || q.Kind == "principle") {
				ops = append(ops, op{"op": "close_question", "question": q.ID, "state": "answered", "evidence": marker})
			}
		}
	}
	pronoun := map[string]bool{"she": true, "her": true, "he": true, "him": true, "they": true, "them": true}
	addContact := func(c map[string]interface{}) {
		if c == nil {
			return
		}
		name, _ := c["name"].(string)
		label, _ := c["label"].(string)
		role, _ := c["role"].(string)
		evidence, _ := c["evidence"].(string)
		if !strings.Contains(strings.ToLower(msg), strings.ToLower(evidence)) || evidence == "" {
			evidence = label
			if name != "" {
				evidence = name
			}
		}
		ref := ""
		switch {
		case name != "" && !strings.EqualFold(name, label) && label != "" && name == strings.TrimSpace(name) && c["nameKnown"] == nil && unicode.IsUpper([]rune(name)[0]):
			fallthrough
		case name != "":
			if id, ok := byName[strings.ToLower(name)]; ok {
				ref = id
			} else if id, ok := byLabel[strings.ToLower(label)]; ok && label != "" && !strings.EqualFold(name, label) {
				ref = id
				ops = append(ops, op{"op": "name_person", "person": id, "name": name, "evidence": evidence})
				answeredName = true
				if lastUnnamed == id {
					lastUnnamed = ""
				}
			} else {
				nextNew++
				ops = append(ops, op{"op": "add_person", "name": name, "evidence": evidence})
				ref = "new" + string(rune('0'+nextNew))
			}
		case label != "" && !pronoun[strings.ToLower(label)]:
			if id, ok := byLabel[strings.ToLower(label)]; ok {
				ref = id
			} else if len(mem.People) == 1 && strings.Contains(strings.ToLower(label), "formal") {
				return // words about the text are no person
			} else {
				nextNew++
				ops = append(ops, op{"op": "add_person", "label": label, "evidence": evidence})
				ref = "new" + string(rune('0'+nextNew))
				lastUnnamed = ref
				lastUnnamedLabel = label
			}
		}
		if ref != "" && role != "" && strings.Contains(strings.ToLower(msg), strings.ToLower(role)) {
			ops = append(ops, op{"op": "set_role", "person": ref, "role": role, "of": "user", "evidence": role})
		}
		if ref != "" {
			if traits, ok := c["traits"].([]interface{}); ok {
				for _, tr := range traits {
					if s, ok := tr.(string); ok && s != "" {
						ops = append(ops, op{"op": "add_trait", "person": ref, "trait": s, "evidence": evidence})
					}
				}
			}
		}
	}
	greeting := strings.Contains(msg, "Hi Moly")
	if !greeting {
		if c, ok := ext["contact"].(map[string]interface{}); ok {
			addContact(c)
		}
		if others, ok := ext["otherContacts"].([]interface{}); ok {
			for _, o := range others {
				if c, ok := o.(map[string]interface{}); ok {
					addContact(c)
				}
			}
		}
	}
	goal := mem.Goal
	intention, _ := ext["intention"].(string)
	if intention != "" && goal == "" {
		ev, _ := ext["intentionEvidence"].(string)
		if ev == "" || !strings.Contains(msg, ev) {
			ev = msg
		}
		ops = append(ops, op{"op": "set_goal", "goal": intention, "evidence": ev})
		goal = intention
	}

	next := map[string]interface{}{"action": "answer"}
	switch {
	case lastUnnamed != "" && goal != "" && !answeredName && !nameAskedAbout(mem, lastUnnamed) && !strings.Contains(msg, "ABOUT-SOMETHING-ELSE"):
		next = map[string]interface{}{"action": "ask", "kind": "name", "about": lastUnnamed, "topic": "name of " + lastUnnamedLabel, "changes": "who to address"}
	case strings.Contains(msg, "CONTRADICTING-ORDERS"):
		next = map[string]interface{}{"action": "ask", "kind": "gap", "topic": "instructions", "missing": "You've given me instructions that pull in opposite directions. Which should I follow?", "changes": "which instruction to follow"}
	case l.unclear != "" && strings.Contains(msg, l.unclear):
		next = map[string]interface{}{"action": "ask", "kind": "clarity", "topic": "unclear " + l.unclear, "missing": "CLARITY-QUESTION?", "changes": "what to write"}
	case l.gaps != nil && goal != "" && !greeting:
		if raw, err := l.gaps(); err == nil {
			var gaps []map[string]interface{}
			if json.Unmarshal([]byte(raw), &gaps) == nil && len(gaps) > 0 {
				d, _ := gaps[0]["description"].(string)
				as, _ := gaps[0]["assumption"].(string)
				if as == "" {
					as = "the draft"
				}
				next = map[string]interface{}{"action": "ask", "kind": "gap", "topic": d, "missing": d, "changes": as}
			}
		}
	}
	out, _ := json.Marshal(map[string]interface{}{"ops": ops, "next": next})
	return string(out)
}

// nameAskedAbout reports whether the memory already holds a name question about the person that was asked and not closed.
func nameAskedAbout(mem models.Memory, person string) bool {
	for _, q := range mem.Questions {
		if q.Kind == "name" && q.About == person && (q.State == "open" || q.State == "declined") {
			return true
		}
	}
	return false
}
