package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"moly/models"
)

// MemoryStore keeps the structured memory of a conversation (docs/MEMORY_REDESIGN.md). Every change arrives as an operation; each one is
// checked here and either applied or rejected with a reason, and every one is recorded (memory_ops), so a wrong memory can be traced to the
// message and the operation that made it. The rules are code, not judgements:
//   - an operation carries evidence, the exact words of the message or the last messages; without them it is rejected;
//   - a name never makes a second person (a named person is unique for the user; an unnamed one is unique by label in its conversation);
//   - a role is in the user's words (the evidence holds the role word), and "former"/"ex" make the role former, never part of the word;
//   - a goal that exists is replaced only when the switch was confirmed.
type MemoryStore struct{ db *sql.DB }

// NewMemoryStore opens the store on the database connection.
func NewMemoryStore(db *sql.DB) *MemoryStore { return &MemoryStore{db: db} }

// ApplyOptions say whose memory a batch of operations changes and what it may be checked against.
type ApplyOptions struct {
	UserID, ConversationID, MessageID string
	Sources                           []string // the new message and the last messages: the evidence must occur in one of them
	GoalSwitchConfirmed               bool     // the user has said yes to the switch: set_goal may replace the goal
}

// OpResult is what happened to one operation.
type OpResult struct {
	Op      models.MemoryOp
	Applied bool
	Reason  string // why it was rejected, or a note on an applied one
	Person  string // the id of the person it created or changed, when there is one
}

const (
	opApplied  = "applied"
	opRejected = "rejected"
	opUndone   = "undone"
)

// applier is one batch inside one transaction.
type applier struct {
	tx      *sql.Tx
	o       ApplyOptions
	now     int64
	added   int               // how many add_person operations this batch has had
	aliases map[string]string // "new1", "new2" ...: the people the batch itself added, in order, for the operations that follow
}

// resolve turns a reference to a person added earlier in the same batch ("new1") into its id.
func (a *applier) resolve(ref string) string {
	if id, ok := a.aliases[strings.ToLower(strings.TrimSpace(ref))]; ok {
		return id
	}
	return ref
}

// Apply checks and applies the operations in order, in one transaction. A rejected operation changes nothing and does not stop the others.
func (s *MemoryStore) Apply(o ApplyOptions, ops []models.MemoryOp) ([]OpResult, error) {
	if o.UserID == "" || o.ConversationID == "" {
		return nil, fmt.Errorf("userId and conversationId required")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	a := &applier{tx: tx, o: o, now: time.Now().Unix(), aliases: map[string]string{}}
	results := make([]OpResult, 0, len(ops))
	for _, op := range ops {
		op = normalizeOp(op)
		op.Person, op.Into, op.Of = a.resolve(op.Person), a.resolve(op.Into), a.resolve(op.Of)
		raw, _ := json.Marshal(op)
		res, err := tx.Exec(`INSERT INTO memory_ops (user_id, conversation_id, message_id, op, status, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
			o.UserID, o.ConversationID, o.MessageID, string(raw), opRejected, a.now)
		if err != nil {
			return nil, err
		}
		opID, _ := res.LastInsertId()
		r := a.apply(opID, op)
		r.Op = op
		if op.Op == models.OpAddPerson {
			a.added++
			if r.Person != "" {
				a.aliases[fmt.Sprintf("new%d", a.added)] = r.Person
			}
		}
		status := opRejected
		if r.Applied {
			status = opApplied
		}
		if _, err := tx.Exec(`UPDATE memory_ops SET status = ?, reason = ? WHERE id = ?`, status, r.Reason, opID); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, tx.Commit()
}

// normalizeOp trims the text of an operation and moves "former"/"ex" out of a role word (found live: "former ex boss").
func normalizeOp(op models.MemoryOp) models.MemoryOp {
	for _, f := range []*string{&op.Op, &op.Person, &op.Into, &op.Label, &op.Name, &op.Role, &op.Of, &op.From, &op.To, &op.Trait, &op.Goal, &op.Question, &op.State, &op.Fact, &op.Evidence} {
		*f = strings.TrimSpace(*f)
	}
	op.Op = strings.ToLower(op.Op)
	if op.Op == models.OpSetRole {
		op.Role, op.Former = splitFormer(op.Role, op.Former)
	}
	if op.Op == models.OpCorrectRole {
		op.To, op.Former = splitFormer(op.To, op.Former)
	}
	return op
}

func splitFormer(role string, former bool) (string, bool) {
	for {
		lower := strings.ToLower(role)
		switch {
		case strings.HasPrefix(lower, "former "):
			role, former = strings.TrimSpace(role[len("former "):]), true
		case strings.HasPrefix(lower, "ex-"):
			role, former = strings.TrimSpace(role[len("ex-"):]), true
		case strings.HasPrefix(lower, "ex "):
			role, former = strings.TrimSpace(role[len("ex "):]), true
		default:
			return role, former
		}
	}
}

func (a *applier) apply(opID int64, op models.MemoryOp) OpResult {
	reject := func(format string, args ...interface{}) OpResult {
		return OpResult{Reason: fmt.Sprintf(format, args...)}
	}
	needsEvidence := op.Op != models.OpCloseQuestion || op.State != models.QuestionSkipped
	if needsEvidence {
		if op.Evidence == "" {
			return reject("no evidence")
		}
		if !a.inSources(op.Evidence) {
			return reject("the evidence is not in the message or the last messages")
		}
	}
	switch op.Op {
	case models.OpAddPerson:
		return a.addPerson(opID, op)
	case models.OpNamePerson:
		return a.namePerson(opID, op)
	case models.OpMergePerson:
		return a.mergePerson(opID, op)
	case models.OpSetRole:
		return a.setRole(opID, op)
	case models.OpCorrectRole:
		return a.correctRole(opID, op)
	case models.OpAddTrait:
		return a.addTrait(opID, op)
	case models.OpSetGoal:
		return a.setGoal(opID, op)
	case models.OpProposeGoalSwitch:
		return a.proposeSwitch(opID, op)
	case models.OpCloseQuestion:
		return a.closeQuestion(opID, op)
	case models.OpAddFact:
		return a.addFact(opID, op)
	}
	return reject("unknown operation %q", op.Op)
}

// inSources reports whether the evidence occurs in one of the sources (case and spacing ignored).
func (a *applier) inSources(evidence string) bool {
	e := squeeze(evidence)
	if e == "" {
		return false
	}
	for _, s := range a.o.Sources {
		if strings.Contains(squeeze(s), e) {
			return true
		}
	}
	return false
}

// wordInSources reports whether the word occurs in one of the sources.
func (a *applier) wordInSources(word string) bool {
	if strings.TrimSpace(word) == "" {
		return false
	}
	for _, src := range a.o.Sources {
		if containsWord(src, word) {
			return true
		}
	}
	return false
}

func squeeze(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), " ")) }

func containsWord(text, word string) bool {
	return word != "" && strings.Contains(squeeze(text), squeeze(word))
}

// person ids are "p<number>".
func personKey(id int64) string { return "p" + strconv.FormatInt(id, 10) }

func parseKey(prefix, id string) (int64, bool) {
	if len(id) < 2 || !strings.EqualFold(id[:1], prefix) {
		return 0, false
	}
	n, err := strconv.ParseInt(id[1:], 10, 64)
	return n, err == nil && n > 0
}

// person loads an active person who belongs to this conversation (it began here or was mentioned here).
func (a *applier) person(id string) (pid int64, name, label, status string, ok bool) {
	n, good := parseKey("p", id)
	if !good {
		return 0, "", "", "", false
	}
	err := a.tx.QueryRow(`SELECT p.id, p.name, p.label, p.name_status FROM memory_people p
		WHERE p.id = ? AND p.user_id = ? AND p.status = 'active'
		AND (p.conversation_id = ? OR EXISTS (SELECT 1 FROM memory_mentions m WHERE m.person_id = p.id AND m.conversation_id = ?))`,
		n, a.o.UserID, a.o.ConversationID, a.o.ConversationID).Scan(&pid, &name, &label, &status)
	return pid, name, label, status, err == nil
}

func (a *applier) addPerson(opID int64, op models.MemoryOp) OpResult {
	if op.Name == "" && op.Label == "" {
		return OpResult{Reason: "a person needs a name or a label"}
	}
	// The name or the label must be the user's own word: in the evidence, or at least in the message and the last messages (the update may quote
	// a whole sentence as evidence and put the name in the neighbouring one).
	if !containsWord(op.Evidence, op.Name) && !containsWord(op.Evidence, op.Label) && !a.wordInSources(op.Name) && !a.wordInSources(op.Label) {
		return OpResult{Reason: "neither the name nor the label is in the user's words"}
	}
	if op.Name != "" {
		var id int64
		var origin string
		err := a.tx.QueryRow(`SELECT id, conversation_id FROM memory_people WHERE user_id = ? AND name = ? COLLATE NOCASE AND status = 'active'`, a.o.UserID, op.Name).Scan(&id, &origin)
		if err == nil {
			if _, _, _, _, here := a.person(personKey(id)); here {
				return OpResult{Person: personKey(id), Reason: fmt.Sprintf("%s is already in this conversation: use it", personKey(id))}
			}
			// a person the user named in another conversation: this conversation now mentions them
			if _, err := a.tx.Exec(`INSERT OR IGNORE INTO memory_mentions (conversation_id, person_id, user_id, created_at) VALUES (?, ?, ?, ?)`, a.o.ConversationID, id, a.o.UserID, a.now); err != nil {
				return OpResult{Reason: err.Error()}
			}
			return OpResult{Applied: true, Person: personKey(id), Reason: "a person the user named before"}
		}
	} else {
		var id int64
		err := a.tx.QueryRow(`SELECT id FROM memory_people WHERE user_id = ? AND conversation_id = ? AND name = '' AND label = ? COLLATE NOCASE AND status = 'active'`,
			a.o.UserID, a.o.ConversationID, op.Label).Scan(&id)
		if err == nil {
			return OpResult{Person: personKey(id), Reason: fmt.Sprintf("%s already has this label in this conversation: use it", personKey(id))}
		}
	}
	status := "unnamed"
	if op.Name != "" {
		status = "named"
	}
	res, err := a.tx.Exec(`INSERT INTO memory_people (user_id, conversation_id, name, label, name_status, source_op_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		a.o.UserID, a.o.ConversationID, op.Name, op.Label, status, opID, a.now, a.now)
	if err != nil {
		return OpResult{Reason: err.Error()}
	}
	id, _ := res.LastInsertId()
	return OpResult{Applied: true, Person: personKey(id)}
}

func (a *applier) namePerson(opID int64, op models.MemoryOp) OpResult {
	pid, name, _, status, ok := a.person(op.Person)
	if !ok {
		return OpResult{Reason: "no such person in this conversation"}
	}
	if op.Name == "" || !containsWord(op.Evidence, op.Name) {
		return OpResult{Reason: "the evidence does not hold the name"}
	}
	if status == "named" {
		if strings.EqualFold(name, op.Name) {
			return OpResult{Applied: true, Person: op.Person, Reason: "already stored"}
		}
		return OpResult{Reason: "the person already has a name"}
	}
	var other int64
	if err := a.tx.QueryRow(`SELECT id FROM memory_people WHERE user_id = ? AND name = ? COLLATE NOCASE AND status = 'active' AND id != ?`, a.o.UserID, op.Name, pid).Scan(&other); err == nil {
		if _, _, _, _, here := a.person(personKey(other)); here {
			return OpResult{Reason: fmt.Sprintf("the name belongs to %s: merge_person instead", personKey(other))}
		}
		// the name is a person the user named in another conversation: the name comes first, the comparison after (docs/ORCHESTRATOR_DESIGN.md,
		// 2026-10-14) - the person who waited for a name is that person, and this conversation now mentions them
		if err := a.mergeInto(pid, other); err != nil {
			return OpResult{Reason: err.Error()}
		}
		a.tx.Exec(`INSERT OR IGNORE INTO memory_mentions (conversation_id, person_id, user_id, created_at) VALUES (?, ?, ?, ?)`, a.o.ConversationID, other, a.o.UserID, a.now)
		a.tx.Exec(`UPDATE memory_questions SET state = 'answered', closed_op_id = ? WHERE about_person_id = ? AND kind = 'name' AND state = 'open'`, opID, other)
		return OpResult{Applied: true, Person: personKey(other), Reason: "a person the user named before"}
	}
	before, _ := json.Marshal(map[string]string{"name": name, "status": status})
	if _, err := a.tx.Exec(`UPDATE memory_people SET name = ?, name_status = 'named', updated_at = ? WHERE id = ?`, op.Name, a.now, pid); err != nil {
		return OpResult{Reason: err.Error()}
	}
	a.tx.Exec(`UPDATE memory_ops SET before = ? WHERE id = ?`, string(before), opID)
	// the name question about this person is answered
	a.tx.Exec(`UPDATE memory_questions SET state = 'answered', closed_op_id = ? WHERE about_person_id = ? AND kind = 'name' AND state = 'open'`, opID, pid)
	return OpResult{Applied: true, Person: op.Person}
}

func (a *applier) mergePerson(opID int64, op models.MemoryOp) OpResult {
	from, _, _, _, ok1 := a.person(op.Person)
	into, _, _, _, ok2 := a.person(op.Into)
	if !ok1 || !ok2 || from == into {
		return OpResult{Reason: "merge needs two different people of this conversation"}
	}
	if err := a.mergeInto(from, into); err != nil {
		return OpResult{Reason: err.Error()}
	}
	return OpResult{Applied: true, Person: op.Into}
}

// mergeInto makes the person "from" part of the person "into": what was known of the first is known of the second, and the first is gone.
func (a *applier) mergeInto(from, into int64) error {
	var fromLabel, intoLabel string
	a.tx.QueryRow(`SELECT label FROM memory_people WHERE id = ?`, from).Scan(&fromLabel)
	a.tx.QueryRow(`SELECT label FROM memory_people WHERE id = ?`, into).Scan(&intoLabel)
	for _, q := range []string{
		`UPDATE OR IGNORE memory_roles SET person_id = ? WHERE person_id = ?`,
		`UPDATE OR IGNORE memory_roles SET of_person_id = ? WHERE of_person_id = ?`,
		`UPDATE memory_traits SET person_id = ? WHERE person_id = ?`,
		`UPDATE memory_facts SET person_id = ? WHERE person_id = ?`,
		`UPDATE memory_questions SET about_person_id = ? WHERE about_person_id = ?`,
		`UPDATE OR IGNORE memory_mentions SET person_id = ? WHERE person_id = ?`,
	} {
		if _, err := a.tx.Exec(q, into, from); err != nil {
			return err
		}
	}
	if intoLabel == "" && fromLabel != "" {
		a.tx.Exec(`UPDATE memory_people SET label = ? WHERE id = ?`, fromLabel, into)
	}
	_, err := a.tx.Exec(`UPDATE memory_people SET status = 'merged', updated_at = ? WHERE id = ?`, a.now, from)
	return err
}

// roleOwner reads the "of" of a role: the user, or another person of this conversation.
func (a *applier) roleOwner(of string, self int64) (sql.NullInt64, bool) {
	if of == "" || strings.EqualFold(of, models.MemoryUser) {
		return sql.NullInt64{}, true
	}
	pid, _, _, _, ok := a.person(of)
	if !ok || pid == self {
		return sql.NullInt64{}, false
	}
	return sql.NullInt64{Int64: pid, Valid: true}, true
}

func (a *applier) setRole(opID int64, op models.MemoryOp) OpResult {
	pid, _, _, _, ok := a.person(op.Person)
	if !ok {
		return OpResult{Reason: "no such person in this conversation"}
	}
	if op.Role == "" {
		return OpResult{Reason: "no role"}
	}
	of, ok := a.roleOwner(op.Of, pid)
	if !ok {
		return OpResult{Reason: "the role is of someone who is not in this conversation"}
	}
	// What a person is to the user is in the user's words. What a person is to another person may be read from the other end of the relation
	// ("A is the son of B" makes B the parent of A), so the role word need not be there; the evidence still must be.
	if !of.Valid && !containsWord(op.Evidence, op.Role) {
		return OpResult{Reason: "the evidence does not hold the role word"}
	}
	var id int64
	var former bool
	err := a.tx.QueryRow(`SELECT id, former FROM memory_roles WHERE person_id = ? AND role = ? COLLATE NOCASE AND IFNULL(of_person_id, 0) = ? AND ended_op_id IS NULL`, pid, op.Role, of.Int64).Scan(&id, &former)
	if err == nil {
		if former == op.Former {
			return OpResult{Applied: true, Person: op.Person, Reason: "already stored"}
		}
		a.tx.Exec(`UPDATE memory_roles SET ended_op_id = ? WHERE id = ?`, opID, id)
	}
	if _, err := a.tx.Exec(`INSERT INTO memory_roles (user_id, person_id, role, of_person_id, former, source_op_id, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		a.o.UserID, pid, op.Role, of, op.Former, opID, a.now); err != nil {
		return OpResult{Reason: err.Error()}
	}
	return OpResult{Applied: true, Person: op.Person}
}

func (a *applier) correctRole(opID int64, op models.MemoryOp) OpResult {
	pid, _, _, _, ok := a.person(op.Person)
	if !ok {
		return OpResult{Reason: "no such person in this conversation"}
	}
	if op.From == "" || op.To == "" || !containsWord(op.Evidence, op.To) {
		return OpResult{Reason: "a correction needs the old role, and the evidence must hold the new one"}
	}
	var id int64
	var of sql.NullInt64
	var former bool
	if err := a.tx.QueryRow(`SELECT id, of_person_id, former FROM memory_roles WHERE person_id = ? AND role = ? COLLATE NOCASE AND ended_op_id IS NULL`, pid, op.From).Scan(&id, &of, &former); err != nil {
		return OpResult{Reason: "the person has no such role to correct"}
	}
	if _, err := a.tx.Exec(`UPDATE memory_roles SET ended_op_id = ? WHERE id = ?`, opID, id); err != nil {
		return OpResult{Reason: err.Error()}
	}
	if _, err := a.tx.Exec(`INSERT INTO memory_roles (user_id, person_id, role, of_person_id, former, source_op_id, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		a.o.UserID, pid, op.To, of, former || op.Former, opID, a.now); err != nil {
		return OpResult{Reason: err.Error()}
	}
	return OpResult{Applied: true, Person: op.Person}
}

func (a *applier) addTrait(opID int64, op models.MemoryOp) OpResult {
	if op.Trait == "" {
		return OpResult{Reason: "no trait"}
	}
	if strings.EqualFold(op.Person, models.MemoryUser) || op.Person == "" {
		return a.insertFact(opID, op.Trait, 0, op.Evidence)
	}
	pid, _, _, _, ok := a.person(op.Person)
	if !ok {
		return OpResult{Reason: "no such person in this conversation"}
	}
	var n int
	a.tx.QueryRow(`SELECT COUNT(*) FROM memory_traits WHERE person_id = ? AND trait = ? COLLATE NOCASE`, pid, op.Trait).Scan(&n)
	if n > 0 {
		return OpResult{Applied: true, Person: op.Person, Reason: "already stored"}
	}
	if _, err := a.tx.Exec(`INSERT INTO memory_traits (user_id, person_id, trait, source_op_id, created_at) VALUES (?, ?, ?, ?, ?)`, a.o.UserID, pid, op.Trait, opID, a.now); err != nil {
		return OpResult{Reason: err.Error()}
	}
	return OpResult{Applied: true, Person: op.Person}
}

func (a *applier) addFact(opID int64, op models.MemoryOp) OpResult {
	if op.Fact == "" {
		return OpResult{Reason: "no fact"}
	}
	var pid int64
	if op.Person != "" && !strings.EqualFold(op.Person, models.MemoryUser) {
		id, _, _, _, ok := a.person(op.Person)
		if !ok {
			return OpResult{Reason: "no such person in this conversation"}
		}
		pid = id
	}
	return a.insertFact(opID, op.Fact, pid, op.Evidence)
}

func (a *applier) insertFact(opID int64, fact string, pid int64, evidence string) OpResult {
	var n int
	a.tx.QueryRow(`SELECT COUNT(*) FROM memory_facts WHERE conversation_id = ? AND IFNULL(person_id, 0) = ? AND fact = ? COLLATE NOCASE`, a.o.ConversationID, pid, fact).Scan(&n)
	if n > 0 {
		return OpResult{Applied: true, Reason: "already stored"}
	}
	var person interface{}
	if pid != 0 {
		person = pid
	}
	if _, err := a.tx.Exec(`INSERT INTO memory_facts (user_id, conversation_id, person_id, fact, evidence, source_op_id, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		a.o.UserID, a.o.ConversationID, person, fact, evidence, opID, a.now); err != nil {
		return OpResult{Reason: err.Error()}
	}
	return OpResult{Applied: true}
}

func (a *applier) setGoal(opID int64, op models.MemoryOp) OpResult {
	if op.Goal == "" {
		return OpResult{Reason: "no goal"}
	}
	var old string
	err := a.tx.QueryRow(`SELECT goal FROM memory_goals WHERE conversation_id = ?`, a.o.ConversationID).Scan(&old)
	if err == nil && !a.o.GoalSwitchConfirmed {
		if strings.EqualFold(old, op.Goal) {
			return OpResult{Applied: true, Reason: "already stored"}
		}
		return OpResult{Reason: "a goal exists: a switch must be proposed and confirmed first"}
	}
	a.tx.Exec(`UPDATE memory_ops SET before = ? WHERE id = ?`, old, opID)
	if _, err := a.tx.Exec(`INSERT INTO memory_goals (conversation_id, user_id, goal, source_op_id, updated_at) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(conversation_id) DO UPDATE SET goal = excluded.goal, source_op_id = excluded.source_op_id, updated_at = excluded.updated_at`,
		a.o.ConversationID, a.o.UserID, op.Goal, opID, a.now); err != nil {
		return OpResult{Reason: err.Error()}
	}
	return OpResult{Applied: true}
}

func (a *applier) proposeSwitch(opID int64, op models.MemoryOp) OpResult {
	if op.Goal == "" {
		return OpResult{Reason: "no goal proposed"}
	}
	var old string
	if err := a.tx.QueryRow(`SELECT goal FROM memory_goals WHERE conversation_id = ?`, a.o.ConversationID).Scan(&old); err != nil {
		return OpResult{Reason: "there is no goal to switch from: set_goal"}
	}
	if _, err := a.tx.Exec(`INSERT INTO memory_questions (user_id, conversation_id, kind, text, state, asked_message_id, created_at) VALUES (?, ?, 'confirm_goal', ?, 'open', ?, ?)`,
		a.o.UserID, a.o.ConversationID, op.Goal, a.o.MessageID, a.now); err != nil {
		return OpResult{Reason: err.Error()}
	}
	return OpResult{Applied: true}
}

func (a *applier) closeQuestion(opID int64, op models.MemoryOp) OpResult {
	qid, ok := parseKey("q", op.Question)
	if !ok {
		return OpResult{Reason: "no such question"}
	}
	switch op.State {
	case models.QuestionAnswered, models.QuestionDeclined, models.QuestionSkipped:
	default:
		return OpResult{Reason: "a question is closed as answered, declined or skipped"}
	}
	var state, kind string
	var about sql.NullInt64
	if err := a.tx.QueryRow(`SELECT state, kind, about_person_id FROM memory_questions WHERE id = ? AND conversation_id = ? AND user_id = ?`, qid, a.o.ConversationID, a.o.UserID).Scan(&state, &kind, &about); err != nil {
		return OpResult{Reason: "no such question in this conversation"}
	}
	if state == op.State {
		return OpResult{Applied: true, Reason: "already closed so"}
	}
	if state != models.QuestionOpen && state != models.QuestionSkipped {
		return OpResult{Reason: "the question is already " + state}
	}
	if _, err := a.tx.Exec(`UPDATE memory_questions SET state = ?, closed_op_id = ? WHERE id = ?`, op.State, opID, qid); err != nil {
		return OpResult{Reason: err.Error()}
	}
	if kind == "name" && op.State == models.QuestionDeclined && about.Valid {
		a.tx.Exec(`UPDATE memory_people SET name_status = 'declined', updated_at = ? WHERE id = ? AND name_status != 'named'`, a.now, about.Int64)
	}
	return OpResult{Applied: true}
}

// OpenQuestion records a question Moly asked in this conversation and returns its id ("q<number>"). about is a person id or "".
func (s *MemoryStore) OpenQuestion(userID, conversationID, messageID, kind, about, text, topic string) (string, error) {
	if userID == "" || conversationID == "" || strings.TrimSpace(text) == "" || kind == "" {
		return "", fmt.Errorf("userId, conversationId, kind and text required")
	}
	var person interface{}
	if about != "" {
		pid, ok := parseKey("p", about)
		if !ok {
			return "", fmt.Errorf("unknown person %q", about)
		}
		person = pid
		if kind == "name" {
			s.db.Exec(`UPDATE memory_people SET name_status = 'asked' WHERE id = ? AND user_id = ? AND name_status = 'unnamed'`, pid, userID)
		}
	}
	res, err := s.db.Exec(`INSERT INTO memory_questions (user_id, conversation_id, kind, about_person_id, text, topic, state, asked_message_id, created_at) VALUES (?, ?, ?, ?, ?, ?, 'open', ?, ?)`,
		userID, conversationID, kind, person, strings.TrimSpace(text), strings.TrimSpace(topic), messageID, time.Now().Unix())
	if err != nil {
		return "", err
	}
	id, _ := res.LastInsertId()
	return "q" + strconv.FormatInt(id, 10), nil
}

// Snapshot reads the memory of one conversation, in the form the update call is shown.
func (s *MemoryStore) Snapshot(userID, conversationID string) (models.Memory, error) {
	m := models.Memory{People: []models.MemoryPerson{}, Questions: []models.MemoryQuestion{}, Facts: []models.MemoryFact{}}
	rows, err := s.db.Query(`SELECT p.id, p.name, p.label, p.name_status FROM memory_people p
		WHERE p.user_id = ? AND p.status = 'active'
		AND (p.conversation_id = ? OR EXISTS (SELECT 1 FROM memory_mentions mm WHERE mm.person_id = p.id AND mm.conversation_id = ?))
		ORDER BY p.id`, userID, conversationID, conversationID)
	if err != nil {
		return m, err
	}
	index := map[int64]int{}
	for rows.Next() {
		var id int64
		var p models.MemoryPerson
		if err := rows.Scan(&id, &p.Name, &p.Label, &p.NameStatus); err != nil {
			rows.Close()
			return m, err
		}
		p.ID, p.Roles, p.Traits = personKey(id), []models.MemoryRole{}, []string{}
		index[id] = len(m.People)
		m.People = append(m.People, p)
	}
	rows.Close()
	rrows, err := s.db.Query(`SELECT person_id, role, of_person_id, former FROM memory_roles WHERE user_id = ? AND ended_op_id IS NULL ORDER BY id`, userID)
	if err != nil {
		return m, err
	}
	for rrows.Next() {
		var pid int64
		var role string
		var of sql.NullInt64
		var former bool
		if rrows.Scan(&pid, &role, &of, &former) != nil {
			continue
		}
		i, ok := index[pid]
		if !ok {
			continue
		}
		who := models.MemoryUser
		if of.Valid {
			if _, known := index[of.Int64]; !known {
				continue
			}
			who = personKey(of.Int64)
		}
		m.People[i].Roles = append(m.People[i].Roles, models.MemoryRole{Role: role, Of: who, Former: former})
	}
	rrows.Close()
	trows, err := s.db.Query(`SELECT person_id, trait FROM memory_traits WHERE user_id = ? ORDER BY id`, userID)
	if err != nil {
		return m, err
	}
	for trows.Next() {
		var pid int64
		var trait string
		if trows.Scan(&pid, &trait) == nil {
			if i, ok := index[pid]; ok {
				m.People[i].Traits = append(m.People[i].Traits, trait)
			}
		}
	}
	trows.Close()
	_ = s.db.QueryRow(`SELECT goal FROM memory_goals WHERE conversation_id = ? AND user_id = ?`, conversationID, userID).Scan(&m.Goal)
	qrows, err := s.db.Query(`SELECT id, kind, about_person_id, text, topic, state FROM memory_questions WHERE conversation_id = ? AND user_id = ? ORDER BY id`, conversationID, userID)
	if err != nil {
		return m, err
	}
	for qrows.Next() {
		var id int64
		var about sql.NullInt64
		q := models.MemoryQuestion{}
		if qrows.Scan(&id, &q.Kind, &about, &q.Text, &q.Topic, &q.State) == nil {
			q.ID = "q" + strconv.FormatInt(id, 10)
			if about.Valid {
				q.About = personKey(about.Int64)
			}
			m.Questions = append(m.Questions, q)
		}
	}
	qrows.Close()
	frows, err := s.db.Query(`SELECT person_id, fact, evidence FROM memory_facts WHERE conversation_id = ? AND user_id = ? ORDER BY id`, conversationID, userID)
	if err != nil {
		return m, err
	}
	for frows.Next() {
		var pid sql.NullInt64
		f := models.MemoryFact{Person: models.MemoryUser}
		if frows.Scan(&pid, &f.Fact, &f.Evidence) == nil {
			if pid.Valid {
				f.Person = personKey(pid.Int64)
			}
			m.Facts = append(m.Facts, f)
		}
	}
	frows.Close()
	return m, nil
}

// UndoMessage reverses every applied operation of one message, newest first. A merge cannot be undone and stops the undo.
func (s *MemoryStore) UndoMessage(userID, conversationID, messageID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	rows, err := tx.Query(`SELECT id, op, before FROM memory_ops WHERE user_id = ? AND conversation_id = ? AND message_id = ? AND status = 'applied' ORDER BY id DESC`, userID, conversationID, messageID)
	if err != nil {
		return err
	}
	type entry struct {
		id         int64
		op, before string
	}
	var entries []entry
	for rows.Next() {
		var e entry
		if rows.Scan(&e.id, &e.op, &e.before) == nil {
			entries = append(entries, e)
		}
	}
	rows.Close()
	// the people this message created have contact rows (the projection): they go too
	var contacts []int64
	crows, err := tx.Query(`SELECT contact_id FROM memory_people WHERE user_id = ? AND contact_id IS NOT NULL AND source_op_id IN
		(SELECT id FROM memory_ops WHERE user_id = ? AND conversation_id = ? AND message_id = ? AND status = 'applied')`, userID, userID, conversationID, messageID)
	if err != nil {
		return err
	}
	for crows.Next() {
		var cid int64
		if crows.Scan(&cid) == nil {
			contacts = append(contacts, cid)
		}
	}
	crows.Close()
	for _, e := range entries {
		var op models.MemoryOp
		if json.Unmarshal([]byte(e.op), &op) != nil {
			continue
		}
		if op.Op == models.OpMergePerson {
			return fmt.Errorf("a merge cannot be undone")
		}
		for _, q := range []string{
			`DELETE FROM memory_roles WHERE source_op_id = ?`,
			`DELETE FROM memory_traits WHERE source_op_id = ?`,
			`DELETE FROM memory_facts WHERE source_op_id = ?`,
			`DELETE FROM memory_people WHERE source_op_id = ?`,
			`UPDATE memory_roles SET ended_op_id = NULL WHERE ended_op_id = ?`,
			`UPDATE memory_questions SET state = 'open', closed_op_id = NULL WHERE closed_op_id = ?`,
		} {
			if _, err := tx.Exec(q, e.id); err != nil {
				return err
			}
		}
		switch op.Op {
		case models.OpNamePerson:
			var prev struct{ Name, Status string }
			var raw map[string]string
			if json.Unmarshal([]byte(e.before), &raw) == nil {
				prev.Name, prev.Status = raw["name"], raw["status"]
				if pid, ok := parseKey("p", op.Person); ok {
					tx.Exec(`UPDATE memory_people SET name = ?, name_status = ? WHERE id = ? AND user_id = ?`, prev.Name, prev.Status, pid, userID)
				}
			}
		case models.OpSetGoal:
			if e.before == "" {
				tx.Exec(`DELETE FROM memory_goals WHERE conversation_id = ? AND source_op_id = ?`, conversationID, e.id)
			} else {
				tx.Exec(`UPDATE memory_goals SET goal = ? WHERE conversation_id = ? AND source_op_id = ?`, e.before, conversationID, e.id)
			}
		case models.OpProposeGoalSwitch:
			tx.Exec(`DELETE FROM memory_questions WHERE conversation_id = ? AND kind = 'confirm_goal' AND asked_message_id = ? AND text = ?`, conversationID, messageID, op.Goal)
		}
		if _, err := tx.Exec(`UPDATE memory_ops SET status = ? WHERE id = ?`, opUndone, e.id); err != nil {
			return err
		}
	}
	// the questions this message asked, and the contact rows of the people it created
	if _, err := tx.Exec(`DELETE FROM memory_questions WHERE user_id = ? AND conversation_id = ? AND asked_message_id = ?`, userID, conversationID, messageID); err != nil {
		return err
	}
	for _, cid := range contacts {
		if _, err := tx.Exec(`DELETE FROM contacts WHERE id = ? AND user_id = ?`, cid, userID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// GapCountsByConversation counts the questions of a conversation that make its maturity (docs/ORCHESTRATOR_DESIGN.md: answered over answered
// plus still open): the gap, clarity and principle questions. A question the user answered or declined is resolved; one still open or
// skipped is a gap. Names, confirmations and goal switches are not counted.
func (s *MemoryStore) GapCountsByConversation(conversationID string) (answered, open int) {
	rows, err := s.db.Query(`SELECT state, COUNT(*) FROM memory_questions WHERE conversation_id = ? AND kind IN ('gap', 'clarity', 'principle') GROUP BY state`, conversationID)
	if err != nil {
		return 0, 0
	}
	defer rows.Close()
	for rows.Next() {
		var state string
		var n int
		if rows.Scan(&state, &n) != nil {
			continue
		}
		switch state {
		case models.QuestionAnswered, models.QuestionDeclined:
			answered += n
		default:
			open += n
		}
	}
	return answered, open
}

// SkipOpenQuestions marks every open gap, clarity and principle question of a conversation as skipped: the user pressed the skip button. They
// stay gaps (maturity) and may come back once, when they are still essential.
func (s *MemoryStore) SkipOpenQuestions(userID, conversationID string) error {
	_, err := s.db.Exec(`UPDATE memory_questions SET state = 'skipped' WHERE user_id = ? AND conversation_id = ? AND state = 'open' AND kind IN ('gap', 'clarity', 'principle')`, userID, conversationID)
	return err
}
