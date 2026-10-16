package database

import (
	"path/filepath"
	"testing"
	"time"

	"moly/models"
)

func memoryTestStore(t *testing.T) (*MemoryStore, ApplyOptions) {
	t.Helper()
	db, err := initWithKey(filepath.Join(t.TempDir(), "mem.db"), testKey(t))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	c := db.GetConnection()
	c.Exec(`INSERT INTO users (id, created_at, last_active) VALUES ('u1', ?, ?)`, now, now)
	c.Exec(`INSERT INTO conversations (id, user_id, name, created_at, updated_at) VALUES ('c1', 'u1', 'n', ?, ?)`, now, now)
	c.Exec(`INSERT INTO conversations (id, user_id, name, created_at, updated_at) VALUES ('c2', 'u1', 'n', ?, ?)`, now, now)
	return NewMemoryStore(c), ApplyOptions{UserID: "u1", ConversationID: "c1", MessageID: "m1"}
}

func applyOne(t *testing.T, s *MemoryStore, o ApplyOptions, sources string, op models.MemoryOp) OpResult {
	t.Helper()
	o.Sources = []string{sources}
	res, err := s.Apply(o, []models.MemoryOp{op})
	if err != nil {
		t.Fatal(err)
	}
	return res[0]
}

func TestAnOperationWithoutEvidenceInTheMessageIsRejected(t *testing.T) {
	s, o := memoryTestStore(t)
	msg := "Should I talk to his father first?"
	if r := applyOne(t, s, o, msg, models.MemoryOp{Op: models.OpAddPerson, Label: "father"}); r.Applied {
		t.Fatal("no evidence: rejected")
	}
	if r := applyOne(t, s, o, msg, models.MemoryOp{Op: models.OpAddPerson, Label: "father", Evidence: "his mother"}); r.Applied {
		t.Fatal("evidence that is not in the message: rejected")
	}
	r := applyOne(t, s, o, msg, models.MemoryOp{Op: models.OpAddPerson, Label: "father", Evidence: "his father"})
	if !r.Applied || r.Person != "p1" {
		t.Fatalf("a person with evidence is added, got %+v", r)
	}
	m, _ := s.Snapshot("u1", "c1")
	if len(m.People) != 1 || m.People[0].Label != "father" || m.People[0].NameStatus != "unnamed" {
		t.Fatalf("snapshot: %+v", m.People)
	}
}

func TestANameNeverMakesASecondPerson(t *testing.T) {
	s, o := memoryTestStore(t)
	msg := "His name is Kyle. He is my boss."
	first := applyOne(t, s, o, msg, models.MemoryOp{Op: models.OpAddPerson, Name: "Kyle", Evidence: "His name is Kyle"})
	if !first.Applied {
		t.Fatal(first.Reason)
	}
	again := applyOne(t, s, o, msg, models.MemoryOp{Op: models.OpAddPerson, Name: "kyle", Evidence: "Kyle"})
	if again.Applied {
		t.Fatal("the same name in the same conversation is rejected: use the person who is there")
	}
	// another conversation of the same user: the person is mentioned there, not duplicated
	o2 := o
	o2.ConversationID = "c2"
	other := applyOne(t, s, o2, "Kyle again", models.MemoryOp{Op: models.OpAddPerson, Name: "Kyle", Evidence: "Kyle"})
	if !other.Applied || other.Person != first.Person {
		t.Fatalf("a person the user named before is reused, got %+v", other)
	}
	m, _ := s.Snapshot("u1", "c2")
	if len(m.People) != 1 || m.People[0].Name != "Kyle" {
		t.Fatalf("the second conversation sees Kyle once: %+v", m.People)
	}
}

func TestANameGoesToTheWaitingPersonAndClosesItsQuestion(t *testing.T) {
	s, o := memoryTestStore(t)
	add := applyOne(t, s, o, "my supervisor", models.MemoryOp{Op: models.OpAddPerson, Label: "supervisor", Evidence: "supervisor"})
	q, err := s.OpenQuestion("u1", "c1", "m1", "name", add.Person, "Can you give me a name for the supervisor you mentioned?", "")
	if err != nil {
		t.Fatal(err)
	}
	named := applyOne(t, s, o, "Marta", models.MemoryOp{Op: models.OpNamePerson, Person: add.Person, Name: "Marta", Evidence: "Marta"})
	if !named.Applied {
		t.Fatal(named.Reason)
	}
	m, _ := s.Snapshot("u1", "c1")
	if m.People[0].Name != "Marta" || m.People[0].NameStatus != "named" {
		t.Fatalf("named: %+v", m.People[0])
	}
	if len(m.Questions) != 1 || m.Questions[0].ID != q || m.Questions[0].State != models.QuestionAnswered {
		t.Fatalf("the name question is answered by the name: %+v", m.Questions)
	}
}

func TestANameThatBelongsToAnotherPersonIsRefusedAndAsksForAMerge(t *testing.T) {
	s, o := memoryTestStore(t)
	applyOne(t, s, o, "Kyle", models.MemoryOp{Op: models.OpAddPerson, Name: "Kyle", Evidence: "Kyle"})
	boss := applyOne(t, s, o, "my boss", models.MemoryOp{Op: models.OpAddPerson, Label: "boss", Evidence: "boss"})
	r := applyOne(t, s, o, "My boss is Kyle", models.MemoryOp{Op: models.OpNamePerson, Person: boss.Person, Name: "Kyle", Evidence: "Kyle"})
	if r.Applied {
		t.Fatal("naming a nameless person with a name another person has is a merge")
	}
	merged := applyOne(t, s, o, "My boss is Kyle", models.MemoryOp{Op: models.OpMergePerson, Person: boss.Person, Into: "p1", Evidence: "My boss is Kyle"})
	if !merged.Applied {
		t.Fatal(merged.Reason)
	}
	m, _ := s.Snapshot("u1", "c1")
	if len(m.People) != 1 || m.People[0].Name != "Kyle" || m.People[0].Label != "boss" {
		t.Fatalf("after the merge there is one Kyle who is also the boss: %+v", m.People)
	}
}

func TestARoleIsInTheUsersWordsAndFormerIsNotPartOfIt(t *testing.T) {
	s, o := memoryTestStore(t)
	alex := applyOne(t, s, o, "Alex is my ex boss, father of Kyle", models.MemoryOp{Op: models.OpAddPerson, Name: "Alex", Evidence: "Alex"})
	kyle := applyOne(t, s, o, "Alex is my ex boss, father of Kyle", models.MemoryOp{Op: models.OpAddPerson, Name: "Kyle", Evidence: "Kyle"})
	msg := "Alex is my ex boss, father of Kyle"
	if r := applyOne(t, s, o, msg, models.MemoryOp{Op: models.OpSetRole, Person: alex.Person, Role: "supplier", Of: "user", Evidence: "my ex boss"}); r.Applied {
		t.Fatal("a role word the evidence does not hold is rejected")
	}
	if r := applyOne(t, s, o, msg, models.MemoryOp{Op: models.OpSetRole, Person: alex.Person, Role: "ex boss", Of: "user", Evidence: "my ex boss"}); !r.Applied {
		t.Fatal(r.Reason)
	}
	if r := applyOne(t, s, o, msg, models.MemoryOp{Op: models.OpSetRole, Person: alex.Person, Role: "father", Of: kyle.Person, Evidence: "father of Kyle"}); !r.Applied {
		t.Fatal(r.Reason)
	}
	if r := applyOne(t, s, o, msg, models.MemoryOp{Op: models.OpSetRole, Person: alex.Person, Role: "father", Of: alex.Person, Evidence: "father of Kyle"}); r.Applied {
		t.Fatal("a person is nobody's role-holder of themselves")
	}
	m, _ := s.Snapshot("u1", "c1")
	roles := m.People[0].Roles
	if len(roles) != 2 || roles[0].Role != "boss" || !roles[0].Former || roles[0].Of != "user" || roles[1].Role != "father" || roles[1].Of != kyle.Person {
		t.Fatalf("one former boss of the user and the father of Kyle, got %+v", roles)
	}
}

func TestACorrectedRoleReplacesTheOldOneAndCanBeUndone(t *testing.T) {
	s, o := memoryTestStore(t)
	maria := applyOne(t, s, o, "Maria is my sister", models.MemoryOp{Op: models.OpAddPerson, Name: "Maria", Evidence: "Maria"})
	applyOne(t, s, o, "Maria is my sister", models.MemoryOp{Op: models.OpSetRole, Person: maria.Person, Role: "sister", Of: "user", Evidence: "my sister"})
	o2 := o
	o2.MessageID = "m2"
	r := applyOne(t, s, o2, "No wait, she is my sister-in-law.", models.MemoryOp{Op: models.OpCorrectRole, Person: maria.Person, From: "sister", To: "sister-in-law", Evidence: "my sister-in-law"})
	if !r.Applied {
		t.Fatal(r.Reason)
	}
	m, _ := s.Snapshot("u1", "c1")
	if len(m.People[0].Roles) != 1 || m.People[0].Roles[0].Role != "sister-in-law" {
		t.Fatalf("only the corrected role remains: %+v", m.People[0].Roles)
	}
	if err := s.UndoMessage("u1", "c1", "m2"); err != nil {
		t.Fatal(err)
	}
	m, _ = s.Snapshot("u1", "c1")
	if len(m.People[0].Roles) != 1 || m.People[0].Roles[0].Role != "sister" {
		t.Fatalf("undoing the correction brings the sister back: %+v", m.People[0].Roles)
	}
}

func TestAGoalIsReplacedOnlyWhenTheSwitchIsConfirmed(t *testing.T) {
	s, o := memoryTestStore(t)
	if r := applyOne(t, s, o, "I need to secure my position at the company", models.MemoryOp{Op: models.OpSetGoal, Goal: "secure my position", Evidence: "secure my position"}); !r.Applied {
		t.Fatal(r.Reason)
	}
	msg := "Actually, I need to write a letter to my landlord"
	if r := applyOne(t, s, o, msg, models.MemoryOp{Op: models.OpSetGoal, Goal: "write to my landlord", Evidence: "write a letter to my landlord"}); r.Applied {
		t.Fatal("a goal exists: replace only after a confirmed switch")
	}
	if r := applyOne(t, s, o, msg, models.MemoryOp{Op: models.OpProposeGoalSwitch, Goal: "write to my landlord", Evidence: "write a letter to my landlord"}); !r.Applied {
		t.Fatal(r.Reason)
	}
	m, _ := s.Snapshot("u1", "c1")
	if m.Goal != "secure my position" || len(m.Questions) != 1 || m.Questions[0].Kind != "confirm_goal" {
		t.Fatalf("the goal stays and the proposal waits as a question: %+v", m)
	}
	o.GoalSwitchConfirmed = true
	if r := applyOne(t, s, o, "Yes", models.MemoryOp{Op: models.OpSetGoal, Goal: "write to my landlord", Evidence: "Yes"}); !r.Applied {
		t.Fatal(r.Reason)
	}
	m, _ = s.Snapshot("u1", "c1")
	if m.Goal != "write to my landlord" {
		t.Fatalf("the confirmed goal: %q", m.Goal)
	}
}

func TestDecliningANameQuestionClosesItAndMarksThePerson(t *testing.T) {
	s, o := memoryTestStore(t)
	sup := applyOne(t, s, o, "a supplier", models.MemoryOp{Op: models.OpAddPerson, Label: "supplier", Evidence: "supplier"})
	q, _ := s.OpenQuestion("u1", "c1", "m1", "name", sup.Person, "Can you give me a name for the supplier you mentioned?", "")
	r := applyOne(t, s, o, "It doesn't matter the name.", models.MemoryOp{Op: models.OpCloseQuestion, Question: q, State: models.QuestionDeclined, Evidence: "It doesn't matter the name"})
	if !r.Applied {
		t.Fatal(r.Reason)
	}
	m, _ := s.Snapshot("u1", "c1")
	if m.People[0].NameStatus != "declined" || m.Questions[0].State != models.QuestionDeclined {
		t.Fatalf("declined: %+v %+v", m.People[0], m.Questions[0])
	}
}

func TestEveryOperationIsRecordedWithItsReason(t *testing.T) {
	s, o := memoryTestStore(t)
	o.Sources = []string{"my boss"}
	res, _ := s.Apply(o, []models.MemoryOp{
		{Op: models.OpAddPerson, Label: "boss", Evidence: "my boss"},
		{Op: models.OpAddPerson, Label: "boss", Evidence: "nothing like this"},
	})
	if !res[0].Applied || res[1].Applied {
		t.Fatalf("one applied and one rejected: %+v", res)
	}
	var applied, rejected int
	s.db.QueryRow(`SELECT COUNT(*) FROM memory_ops WHERE status = 'applied'`).Scan(&applied)
	s.db.QueryRow(`SELECT COUNT(*) FROM memory_ops WHERE status = 'rejected' AND reason != ''`).Scan(&rejected)
	if applied != 1 || rejected != 1 {
		t.Fatalf("audit rows: %d applied, %d rejected with a reason", applied, rejected)
	}
}

func TestTheMemoryOfAConversationLeavesWithTheProfile(t *testing.T) {
	s, o := memoryTestStore(t)
	applyOne(t, s, o, "my boss Kyle", models.MemoryOp{Op: models.OpAddPerson, Name: "Kyle", Evidence: "Kyle"})
	s.OpenQuestion("u1", "c1", "m1", "gap", "", "How long?", "")
	if err := DeleteUserData(s.db, "u1"); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"memory_people", "memory_ops", "memory_questions", "memory_roles", "memory_mentions"} {
		var n int
		s.db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n)
		if n != 0 {
			t.Fatalf("%s still holds %d rows after the profile was deleted", table, n)
		}
	}
}

func TestAPersonAddedInTheBatchCanBeNamedByItsOrderAfterwards(t *testing.T) {
	s, o := memoryTestStore(t)
	o.Sources = []string{"My boss Kyle is the son of my old boss Alex"}
	res, err := s.Apply(o, []models.MemoryOp{
		{Op: models.OpAddPerson, Name: "Kyle", Evidence: "Kyle"},
		{Op: models.OpAddPerson, Name: "Alex", Evidence: "Alex"},
		{Op: models.OpSetRole, Person: "new1", Role: "boss", Of: "user", Evidence: "My boss"},
		{Op: models.OpSetRole, Person: "new2", Role: "boss", Of: "user", Former: true, Evidence: "old boss"},
		{Op: models.OpSetRole, Person: "new2", Role: "father", Of: "new1", Evidence: "the son of my old boss"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for i, r := range res {
		if !r.Applied {
			t.Fatalf("operation %d rejected: %s", i, r.Reason)
		}
	}
	m, _ := s.Snapshot("u1", "c1")
	if len(m.People) != 2 || len(m.People[0].Roles) != 1 || len(m.People[1].Roles) != 2 {
		t.Fatalf("each person has the roles given by the batch's own order: %+v", m.People)
	}
}

func TestDeletingAConversationKeepsANamedPersonAnotherConversationMentions(t *testing.T) {
	s, o := memoryTestStore(t)
	s.db.Exec(`PRAGMA foreign_keys = ON`)
	o.Sources = []string{"my boss Kyle and the supplier"}
	s.Apply(o, []models.MemoryOp{
		{Op: models.OpAddPerson, Name: "Kyle", Evidence: "Kyle"},
		{Op: models.OpAddPerson, Label: "supplier", Evidence: "supplier"},
		{Op: models.OpSetGoal, Goal: "keep my job", Evidence: "my boss"},
	})
	o2 := o
	o2.ConversationID = "c2"
	o2.Sources = []string{"Kyle again"}
	s.Apply(o2, []models.MemoryOp{{Op: models.OpAddPerson, Name: "Kyle", Evidence: "Kyle"}})
	if err := DeleteConversationData(s.db, "u1", "c1"); err != nil {
		t.Fatal(err)
	}
	m2, _ := s.Snapshot("u1", "c2")
	if len(m2.People) != 1 || m2.People[0].Name != "Kyle" {
		t.Fatalf("Kyle stays for the conversation that mentions him: %+v", m2.People)
	}
	var left int
	s.db.QueryRow(`SELECT COUNT(*) FROM memory_people WHERE label = 'supplier'`).Scan(&left)
	var goals int
	s.db.QueryRow(`SELECT COUNT(*) FROM memory_goals WHERE conversation_id = 'c1'`).Scan(&goals)
	if left != 0 || goals != 0 {
		t.Fatalf("the unnamed supplier and the goal go with the conversation, left %d people, %d goals", left, goals)
	}
}

func TestANameThatBelongsToAPersonOfAnotherConversationIsThatPerson(t *testing.T) {
	s, o := memoryTestStore(t)
	o2 := o
	o2.ConversationID = "c2"
	o2.Sources = []string{"my boss Kyle"}
	s.Apply(o2, []models.MemoryOp{{Op: models.OpAddPerson, Name: "Kyle", Evidence: "Kyle"}})
	o.Sources = []string{"my boss"}
	add, _ := s.Apply(o, []models.MemoryOp{{Op: models.OpAddPerson, Label: "boss", Evidence: "boss"}})
	o.MessageID = "m2"
	o.Sources = []string{"His name is Kyle"}
	res, _ := s.Apply(o, []models.MemoryOp{{Op: models.OpNamePerson, Person: add[0].Person, Name: "Kyle", Evidence: "Kyle"}})
	if !res[0].Applied {
		t.Fatalf("a name the user gave another conversation is that person: %s", res[0].Reason)
	}
	m, _ := s.Snapshot("u1", "c1")
	if len(m.People) != 1 || m.People[0].Name != "Kyle" {
		t.Fatalf("this conversation has Kyle once, got %+v", m.People)
	}
}
