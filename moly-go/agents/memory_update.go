package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"moly/models"
	"moly/tools"
)

// The update call of the memory redesign (docs/MEMORY_REDESIGN.md): one model call per message that reads the whole memory of the
// conversation, the last messages word for word and the new message, and answers with the operations that change the memory and the next
// action. It does not write the reply. The operations are checked and applied by database.MemoryStore; this file only asks and reads.

// MemoryNext is what Moly does after the update (defined in models, so the reply context can carry it).
type MemoryNext = models.MemoryNext

// MemoryUpdate is the read answer of the update call.
type MemoryUpdate struct {
	Ops  []models.MemoryOp `json:"ops"`
	Next MemoryNext        `json:"next"`
}

// MemoryRecent is one of the last messages.
type MemoryRecent struct {
	Role string `json:"role"` // "user" or "moly"
	Text string `json:"text"`
}

const memoryUpdateSystem = "You keep the memory of a conversation between a user and Moly, a communication thinking partner. " +
	"You read the memory, the last messages and the new message, and you answer with the changes to the memory and what Moly does next. You never write a reply."

// memoryUpdateRules is the whole contract of the call. It names no person and no situation; every rule says a kind of thing.
const memoryUpdateRules = `Answer with ONLY one JSON object: {"ops": [...], "next": {"action": "...", "kind": "...", "about": "...", "topic": "...", "missing": "...", "changes": "..."}}.

Every operation carries "evidence": words copied exactly from the new message or from the last messages that show the change. An operation without such words is wrong. People are written only by their id from the memory ("p1"), and the user is "user".

Operations (use only these, only the fields shown). A person you add in this answer is called new1, new2 ... in the order you add them; the operations that follow in the same answer point to that person by that word, never by a guess at an id:
- {"op":"add_person","label":"the user's word for the person, without my, his, her, the or a","name":"the name, only if a message gives one","evidence":"..."}
- {"op":"name_person","person":"p?","name":"...","evidence":"..."}  a message gives the name of a listed person who has none
- {"op":"merge_person","person":"p?","into":"p?","evidence":"..."}  two entries are one person (for example a listed person who has no name is said to have the name of another listed person)
- {"op":"set_role","person":"p?","role":"the role word","of":"user or p?","former":false,"evidence":"..."}
- {"op":"correct_role","person":"p?","from":"old role","to":"new role","evidence":"..."}
- {"op":"add_trait","person":"p? or user","trait":"a short statement","evidence":"..."}
- {"op":"add_fact","person":"p? or user","fact":"...","evidence":"..."}
- {"op":"close_question","question":"q?","state":"answered|declined","evidence":"..."}

Who a phrase points to:
- A name that a listed person has is that person. Never add a person whose name is listed.
- A pronoun points to the listed person it fits. With nobody to point to, add nobody and ask who is meant.
- A phrase for a role ("the boss", "his father", "her daughter", a trade) points to a listed person only when the memory states that role for that person. A role stated between two listed people also names the other end: if one listed person is the parent of another, "the parent of B" is the first and "A's child" is the second. A role no listed person has is a NEW person: add it with its label and no name.
- A listed person's role never changes because the user mentions a different role. Do not attach a new role word to a listed person unless the message says that this person IS that role.
- A person described only through another person (a relative of the boss) says nothing of how they relate to the user; do not give a role to the user from it.

Names: when the memory has an open question asking for a name, the next message either gives the name (name_person, and the question closes), declines or says the name does not matter (close_question declined), or says something else (the question stays open). The name of a listed person that appears in a sentence about something else is not an answer.

Roles: use the user's words. "ex" or "former" is not part of the role: write the role and set "former": true. A role to the user has "of":"user". A role to another listed person has that person's id. When the user corrects a role ("no, she is actually ..."), use correct_role with the old role from the memory.

Traits and facts: a trait belongs to the person the sentence is about. Text the user pastes or quotes that another person wrote is about its author: the "I" in it is that author, so add the author as a person if the memory lacks them and attach the traits to them, never to the user. Never guess anyone's gender, age or feelings; write only what the messages say, as plain statements (do not add "according to the user").

Goal: the goal of the conversation (memory.goal) is decided elsewhere. Never set it and never propose a switch; use it to know what the user is working on.

Next action ("next"):
- "answer": the request is clear enough, a greeting, a thank you, or the user answered what was open and nothing essential is missing.
- "ask": something essential is missing. A question is worth asking only when (1) the answer would change what Moly writes or stores, (2) nothing in the memory, the last messages or the user's words already holds it, and (3) a reasonable reader would not find it obvious. Give "topic": the matter in a few words with the person id ("p2 involvement"), "missing": what exactly is missing, in a short phrase, and "changes": what the answer would change. Never ask a topic that appears in "questions" as open, answered or declined; a skipped topic may be asked once more only when it is still essential. Never ask for a name the memory already holds. A new person the user plans to speak or write to, or asks what to say to, needs a name first (kind "name"). A message that tells a situation or a worry but asks for nothing needs one question about what the user wants (kind "gap"). The kind is "name" for a person who needs a name, "gap" for a missing fact, "clarity" for a message that cannot be read.
- "confirm": a fact is doubtful and was not stored; say what in "kind".
- "result_now": the user asks to go ahead, presses skip, or refuses to answer more questions.
When the user repeats something the memory already holds ("I told you ...") or says a question was obvious or already answered, the next action is "answer" and the matter is not asked again. "about" is only a person id from the memory, or empty.
If the message gives instructions about how Moly should work that pull in opposite directions (do this and do not do this), ask which to follow: next "ask", kind "gap", topic "instructions", missing "which of the two instructions to follow".
Prefer "answer" when in doubt after a question was already asked.`

// BuildMemoryUpdatePrompt writes the user prompt of the update call.
func BuildMemoryUpdatePrompt(memory models.Memory, recent []MemoryRecent, message string) string {
	mem, _ := json.Marshal(memory)
	rec, _ := json.Marshal(recent)
	msg, _ := json.Marshal(message)
	return memoryUpdateRules + "\n\nMemory of this conversation:\n" + string(mem) + "\n\nThe last messages (oldest first):\n" + string(rec) +
		"\n\nThe new message from the user:\n" + string(msg) + "\n\nOutput only the JSON object."
}

// RunMemoryUpdate asks the model for the update. A failed or unreadable answer is an error: the caller keeps the memory and applies nothing.
func RunMemoryUpdate(ctx context.Context, llm tools.LLMProvider, memory models.Memory, recent []MemoryRecent, message string) (*MemoryUpdate, error) {
	if llm == nil {
		return nil, fmt.Errorf("no model available")
	}
	resp, err := llm.Call(ctx, &tools.LLMRequest{
		Role:         tools.RoleReply,
		SystemPrompt: memoryUpdateSystem,
		UserPrompt:   BuildMemoryUpdatePrompt(memory, recent, message),
		MaxTokens:    1500,
		Temperature:  0.1,
		Effort:       "low",
	})
	if err != nil {
		return nil, fmt.Errorf("memory update failed: %w", err)
	}
	return parseMemoryUpdate(resp.Content)
}

// parseMemoryUpdate reads the answer. Unknown operations stay in the list (the store rejects them with a reason).
func parseMemoryUpdate(raw string) (*MemoryUpdate, error) {
	var out MemoryUpdate
	if err := tools.SafeJSONParse("MemoryUpdate", []byte(raw), &out); err != nil {
		log.Printf("[MemoryUpdate] answer unreadable")
		return nil, fmt.Errorf("memory update unreadable: %w", err)
	}
	out.Next.Action = strings.ToLower(strings.TrimSpace(out.Next.Action))
	if about := strings.TrimSpace(out.Next.About); len(about) > 1 && (about[0] == 'p' || about[0] == 'P') && strings.Trim(about[1:], "0123456789") == "" {
		out.Next.About = about
	} else {
		out.Next.About = "" // a sentence or anything else is not a person id
	}
	switch out.Next.Action {
	case "answer", "ask", "confirm", "result_now":
	default:
		return nil, fmt.Errorf("memory update gave no usable next action")
	}
	return &out, nil
}

// GuardNext applies the question discipline in code. It returns the next action to use and, when it changed the proposal, why. The proposal
// stays an "ask" only when it states what the answer would change, its topic is not one the memory already asked (open, answered or declined; a
// skipped one may come back once), and a name is not asked for a person who has one or refused to give one.
func GuardNext(memory models.Memory, next MemoryNext) (MemoryNext, string) {
	answer := MemoryNext{Action: "answer"}
	switch next.Action {
	case "confirm":
		if strings.TrimSpace(next.Kind) == "" {
			return answer, "a confirmation must say what it confirms"
		}
	case "ask":
		if strings.TrimSpace(next.Changes) == "" {
			return answer, "a question must say what the answer would change"
		}
		if topic := normalizeTopic(next.Topic); topic != "" {
			skipped := 0
			for _, q := range memory.Questions {
				if normalizeTopic(q.Topic) != topic {
					continue
				}
				if q.State == models.QuestionSkipped {
					skipped++
					continue
				}
				return answer, "the topic was already asked (" + q.State + ")"
			}
			if skipped >= 2 {
				return answer, "the topic was skipped twice"
			}
			if skipped == 1 {
				next.Returning = true // asked once and skipped: it comes back on purpose, once
			}
		}
		if next.Kind == "name" {
			for _, p := range memory.People {
				if p.ID == next.About && (p.NameStatus == "named" || p.NameStatus == "declined") {
					return answer, "the person already has a name or refused to give one"
				}
			}
		}
	}
	return next, ""
}

func normalizeTopic(t string) string { return strings.ToLower(strings.Join(strings.Fields(t), " ")) }
