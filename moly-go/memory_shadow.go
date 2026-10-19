package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"moly/agents"
	"moly/database"
	"moly/models"
	"moly/privacy"
	"moly/tools"
)

// The memory update in the turn (docs/MEMORY_REDESIGN.md). One model call per message reads the memory of the conversation, the last messages and
// the new message; its operations are checked and applied by the memory store, the people are written to the contacts for the readers that have
// not moved to the memory yet, and its next action gives the name question, the confirmation and the gap or clarity question.

type shadowResult struct {
	update   *agents.MemoryUpdate
	applied  int
	rejected []string
	results  []database.OpResult
	took     time.Duration
	err      error
}

type memoryShadow struct {
	run                               func() shadowResult
	once                              sync.Once
	res                               shadowResult
	ch                                chan shadowResult
	store                             *database.MemoryStore
	userID, conversationID, messageID string
	sources                           []string
	repo                              *database.ContactRepository // the conversation's people, to write the contacts again after a question
	llm                               tools.LLMProvider
	recent                            []agents.MemoryRecent
	message                           string
	committed                         bool
}

// shadowQuestionKinds maps the reply kinds of the old path that ask something to the kinds of the memory's questions.
var shadowQuestionKinds = map[string]string{
	"name_question": "name", "gap_question": "gap", "clarity_question": "clarity", "principle_question": "principle",
	"confirm_question": "confirm", "persistent_question": "principle", "intent_check_question": "clarity", "socratic_question": "gap",
}

// startMemoryShadow starts the update for this message. history is the stored conversation with the new message possibly in it.
func (srv *APIServer) startMemoryShadow(userID, conversationID, messageID, message string, history []models.Message, repo *database.ContactRepository) *memoryShadow {
	if srv.llmClient == nil || conversationID == "" || conversationID == "null" {
		return nil
	}
	var recent []agents.MemoryRecent
	for _, m := range history {
		if m.ID == messageID {
			continue
		}
		role := "user"
		if m.Role != "user" {
			role = "moly"
		}
		recent = append(recent, agents.MemoryRecent{Role: role, Text: m.Content})
	}
	if len(recent) > 3 {
		recent = recent[len(recent)-3:]
	}
	sources := []string{message}
	for _, r := range recent {
		sources = append(sources, r.Text)
	}
	store := database.NewMemoryStore(srv.database.GetConnection())
	s := &memoryShadow{store: store, userID: userID, conversationID: conversationID, messageID: messageID, sources: sources, ch: make(chan shadowResult, 1), repo: repo,
		llm:              srv.llmClient, recent: recent, message: message}
	go func() { s.ch <- s.compute(message) }()
	return s
}

// compute asks the model for the update of one message.
func (s *memoryShadow) compute(message string) shadowResult {
	started := time.Now()
	var res shadowResult
	defer func() { res.took = time.Since(started) }()
	snap, err := s.store.Snapshot(s.userID, s.conversationID)
	if err != nil {
		res.err = err
		return res
	}
	ctx, cancel := context.WithTimeout(context.Background(), llmCallTimeout)
	defer cancel()
	up, err := agents.RunMemoryUpdate(ctx, s.llm, snap, s.recent, message)
	if err != nil {
		res.err = err
		return res
	}
	res.update = up
	for _, op := range up.Ops { // what the update learned is user content: masked in the log like everything else
		privacy.Register(op.Name, op.Label, op.Role, op.Trait, op.Goal, op.Fact, op.Evidence)
	}
	return res
}

// useMessage is called once the extraction has cut the secrets out of the message. The update ran beside the extraction, on the message as typed;
// when the cleaned message differs, that answer is dropped (nothing of it was applied) and the update is asked again on the cleaned message, so a
// secret never reaches the memory.
func (s *memoryShadow) useMessage(cleaned string) {
	if s == nil || cleaned == s.message {
		return
	}
	s.result() // wait for the first run, then drop it
	log.Printf("[Privacy] the memory update is asked again on the message without its secrets")
	s.message = cleaned
	s.sources = append([]string{cleaned}, s.sources[1:]...)
	s.once = sync.Once{}
	s.ch = make(chan shadowResult, 1)
	s.ch <- s.compute(cleaned)
}

func (s *memoryShadow) apply(ops []models.MemoryOp) ([]database.OpResult, error) {
	return s.store.Apply(database.ApplyOptions{UserID: s.userID, ConversationID: s.conversationID, MessageID: s.messageID, Sources: s.sources}, ops)
}

func countResults(results []database.OpResult) (applied int, rejected []string) {
	for _, r := range results {
		if r.Applied {
			applied++
		} else {
			rejected = append(rejected, fmt.Sprintf("%s: %s", r.Op.Op, r.Reason))
		}
	}
	return
}

// result waits for the update (it ran beside the rest of the turn) and returns it; the same value every time.
func (s *memoryShadow) result() shadowResult {
	s.once.Do(func() {
		select {
		case s.res = <-s.ch:
		case <-time.After(llmCallTimeout):
			s.res = shadowResult{err: fmt.Errorf("the memory update did not finish")}
		}
	})
	return s.res
}

// MemoryPeople is what the memory decided about the people of this message, in the form the turn reads it.
type MemoryPeople struct {
	Contact          *models.Contact // the person this message is mostly about
	Doubt            string          // a fact to confirm before anything else
	PendingNameLabel string          // a person who must be asked for a name in this reply
	NameAnswered     bool
	Updated          bool // false when the update failed: the memory is as it was
	Next             agents.MemoryNext // the next action after the question discipline
}

// applyMemoryPeople applies the update to the memory, projects the people into the contacts and says what the turn needs to know.
// refused is true for a greeting, a blocked message or a question about Moly: such a message saves no one.
func (srv *APIServer) applyMemoryPeople(s *memoryShadow, repo *database.ContactRepository, refused bool) MemoryPeople {
	out := MemoryPeople{}
	if s == nil {
		return out
	}
	r := s.result()
	if r.err != nil || r.update == nil {
		log.Printf("[Memory] the update failed (%v): the memory stays as it was and nobody is saved from this message", r.err)
		return out
	}
	if refused {
		log.Printf("[Memory] a message that saves no one: the update is not applied")
		return out
	}
	// The goal and its switch belong to the goal logic (GoalRelation, the switch confirmation); the memory mirrors the locked goal after the turn.
	ops := make([]models.MemoryOp, 0, len(r.update.Ops))
	for _, op := range r.update.Ops {
		if op.Op == models.OpSetGoal || op.Op == models.OpProposeGoalSwitch {
			continue
		}
		ops = append(ops, op)
	}
	results, err := s.apply(ops)
	if err != nil {
		log.Printf("[Memory] the operations could not be applied: %v", err)
		return out
	}
	applied, rejected := countResults(results)
	log.Printf("[Memory] %v: %d operations (%d applied, %d rejected), next=%s/%s", r.took.Round(time.Millisecond), len(results), applied, len(rejected), r.update.Next.Action, r.update.Next.Kind)
	for _, res := range results {
		if !res.Applied {
			raw, _ := json.Marshal(res.Op)
			log.Printf("[Memory] rejected %s: %s", raw, res.Reason)
		}
	}
	out.Updated = true
	if err := s.store.ProjectToContacts(repo, s.userID, s.conversationID); err != nil {
		log.Printf("[Memory] the people could not be written to the contacts: %v", err)
	}
	snap, _ := s.store.Snapshot(s.userID, s.conversationID)
	byID := map[string]models.MemoryPerson{}
	for _, p := range snap.People {
		byID[p.ID] = p
	}
	// the person this message is mostly about: the first person an applied operation touched
	for _, res := range results {
		if !res.Applied {
			continue
		}
		switch res.Op.Op {
		case models.OpNamePerson:
			out.NameAnswered = true
		case models.OpCloseQuestion:
			if res.Op.State == models.QuestionAnswered || res.Op.State == models.QuestionDeclined {
				for _, q := range snap.Questions {
					if q.ID == res.Op.Question && q.Kind == "name" {
						out.NameAnswered = true
					}
				}
			}
		}
		if out.Contact == nil && res.Person != "" {
			if cid := s.store.ContactIDOf(s.userID, res.Person); cid != 0 {
				if c, err := repo.GetByID(s.userID, cid); err == nil && c != nil {
					out.Contact = c
				}
			}
		}
	}
	next, why := agents.GuardNext(snap, r.update.Next)
	if why != "" {
		log.Printf("[Memory] the proposed question is not asked: %s", why)
	}
	out.Next = next
	switch next.Action {
	case "ask":
		if next.Kind == "name" {
			who := next.About
			if _, ok := byID[who]; !ok { // no usable person named: the latest person still without a name
				who = ""
				for _, p := range snap.People {
					if p.NameStatus == "unnamed" || p.NameStatus == "asked" {
						who = p.ID
					}
				}
			}
			if p, ok := byID[who]; ok && p.NameStatus != "named" && p.NameStatus != "declined" {
				out.PendingNameLabel = p.Label
				if out.PendingNameLabel == "" {
					out.PendingNameLabel = "person"
				}
			}
		}
	case "confirm":
		out.Doubt = next.Kind
	}
	return out
}

// finish runs when the reply is resolved. In shadow mode it logs the update beside the reply kind the old path chose; in both modes the question
// the reply asked becomes an open question of the memory, so that the next message can answer it.
func (s *memoryShadow) finish(oldKind, replyText string) {
	if s == nil {
		return
	}
	r := s.result()
	if kind, ok := shadowQuestionKinds[oldKind]; ok && replyText != "" {
		about := ""
		if kind == "name" {
			if snap, err := s.store.Snapshot(s.userID, s.conversationID); err == nil {
				for _, p := range snap.People {
					if p.NameStatus == "unnamed" || p.NameStatus == "asked" {
						about = p.ID
					}
				}
			}
		}
		topic := ""
		if r.update != nil && r.update.Next.Action == "ask" && r.update.Next.Kind == kind {
			topic = r.update.Next.Topic
		}
		if _, err := s.store.OpenQuestion(s.userID, s.conversationID, s.messageID, kind, about, replyText, topic); err != nil {
			log.Printf("[Memory] could not record the question: %v", err)
		} else if kind == "name" && s.repo != nil {
			// the person asked about is now "asked": the contacts show it
			if err := s.store.ProjectToContacts(s.repo, s.userID, s.conversationID); err != nil {
				log.Printf("[Memory] the people could not be written to the contacts: %v", err)
			}
		}
	}
}

// installMemoryHooks lets the code that is shared with the old path read the memory's questions when memory mode is on.
func installMemoryHooks(db *database.Database) {
	agents.MaturityHook = func(conversationID string) (float64, bool) {
		if db == nil {
			return 0, false
		}
		answered, open := database.NewMemoryStore(db.GetConnection()).GapCountsByConversation(conversationID)
		return agents.GapMaturity(answered, open), true
	}
}

// markCommitted says the turn was saved: the memory stays as it is.
func (s *memoryShadow) markCommitted() {
	if s != nil {
		s.committed = true
	}
}

// rollbackUnlessCommitted undoes what the memory update did for this message when the turn failed before it was saved (a failed commit, an error
// before the reply): nothing from a turn that was not delivered is kept, and the retry meets the memory as it was.
func (s *memoryShadow) rollbackUnlessCommitted() {
	if s == nil || s.committed {
		return
	}
	r := s.result()
	if r.update == nil {
		return // the update failed or did nothing: nothing to undo
	}
	if err := s.store.UndoMessage(s.userID, s.conversationID, s.messageID); err != nil {
		log.Printf("[Memory] the turn failed and its memory changes could not be undone: %v", err)
		return
	}
	if s.repo != nil {
		if err := s.store.ProjectToContacts(s.repo, s.userID, s.conversationID); err != nil {
			log.Printf("[Memory] the people could not be written to the contacts: %v", err)
		}
	}
	log.Printf("[Memory] the turn was not saved: its changes to the memory are undone")
}
