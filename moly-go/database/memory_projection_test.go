package database

import (
	"testing"

	"moly/models"
)

func projectionSetup(t *testing.T) (*MemoryStore, *ContactRepository, ApplyOptions) {
	s, o := memoryTestStore(t)
	repo := NewContactRepository(&Database{conn: s.db}).InConversation("c1")
	return s, repo, o
}

func TestTheMemoryIsProjectedIntoContacts(t *testing.T) {
	s, repo, o := projectionSetup(t)
	msg := "My boss Kyle, the son of my old boss Alex, avoids me. Someone, a supplier, offered me a deal."
	o.Sources = []string{msg}
	res, _ := s.Apply(o, []models.MemoryOp{
		{Op: models.OpAddPerson, Name: "Kyle", Evidence: "Kyle"},
		{Op: models.OpAddPerson, Name: "Alex", Evidence: "Alex"},
		{Op: models.OpAddPerson, Label: "supplier", Evidence: "supplier"},
		{Op: models.OpSetRole, Person: "new1", Role: "boss", Of: "user", Evidence: "My boss"},
		{Op: models.OpSetRole, Person: "new2", Role: "boss", Of: "user", Former: true, Evidence: "old boss"},
		{Op: models.OpSetRole, Person: "new2", Role: "father", Of: "new1", Evidence: "the son of my old boss"},
		{Op: models.OpAddTrait, Person: "new1", Trait: "avoids the user", Evidence: "avoids me"},
	})
	for i, r := range res {
		if !r.Applied {
			t.Fatalf("operation %d rejected: %s", i, r.Reason)
		}
	}
	if err := s.ProjectToContacts(repo, "u1", "c1"); err != nil {
		t.Fatal(err)
	}
	kyle, _ := repo.GetNamedByName("u1", "Kyle")
	alex, _ := repo.GetNamedByName("u1", "Alex")
	sup, _ := repo.GetUnresolvedByLabel("u1", "supplier")
	if kyle == nil || alex == nil || sup == nil {
		t.Fatalf("three contacts: %v %v %v", kyle, alex, sup)
	}
	if kyle.ContactRole != "boss" || len(kyle.Characteristics) != 1 || kyle.Characteristics[0] != "avoids the user" {
		t.Fatalf("Kyle: %+v", kyle)
	}
	if alex.RelationsText() != "the user's former boss; Kyle's father" && alex.RelationsText() != "Kyle's father; the user's former boss" {
		t.Fatalf("Alex's relations: %q", alex.RelationsText())
	}
	if sup.NameStatus != NameStatusUnnamed || sup.ContactRole != "" {
		t.Fatalf("the supplier has no name and no role: %+v", sup)
	}
	// running it again changes nothing
	if err := s.ProjectToContacts(repo, "u1", "c1"); err != nil {
		t.Fatal(err)
	}
	all, _ := repo.GetByUserID("u1")
	if len(all) != 3 {
		t.Fatalf("still three people after a second projection, got %d", len(all))
	}
}

func TestANamedPersonKeepsTheirRowWhenTheMemoryNamesThem(t *testing.T) {
	s, repo, o := projectionSetup(t)
	o.Sources = []string{"the supplier"}
	s.Apply(o, []models.MemoryOp{{Op: models.OpAddPerson, Label: "supplier", Evidence: "supplier"}})
	s.ProjectToContacts(repo, "u1", "c1")
	before, _ := repo.GetUnresolvedByLabel("u1", "supplier")
	o2 := o
	o2.MessageID = "m2"
	o2.Sources = []string{"His name is Rosa"}
	if res, _ := s.Apply(o2, []models.MemoryOp{{Op: models.OpNamePerson, Person: "p1", Name: "Rosa", Evidence: "Rosa"}}); !res[0].Applied {
		t.Fatal(res[0].Reason)
	}
	s.ProjectToContacts(repo, "u1", "c1")
	after, _ := repo.GetNamedByName("u1", "Rosa")
	if after == nil || after.ID != before.ID || after.NameStatus != NameStatusNamed {
		t.Fatalf("the same row takes the name: before %+v after %+v", before, after)
	}
}

func TestAMergedNamelessPersonLeavesNoContact(t *testing.T) {
	s, repo, o := projectionSetup(t)
	o.Sources = []string{"Kyle is my boss"}
	s.Apply(o, []models.MemoryOp{
		{Op: models.OpAddPerson, Name: "Kyle", Evidence: "Kyle"},
		{Op: models.OpAddPerson, Label: "boss", Evidence: "boss"},
	})
	s.ProjectToContacts(repo, "u1", "c1")
	o2 := o
	o2.MessageID = "m2"
	if res, _ := s.Apply(o2, []models.MemoryOp{{Op: models.OpMergePerson, Person: "p2", Into: "p1", Evidence: "Kyle is my boss"}}); !res[0].Applied {
		t.Fatal(res[0].Reason)
	}
	s.ProjectToContacts(repo, "u1", "c1")
	all, _ := repo.GetByUserID("u1")
	if len(all) != 1 || all[0].Name != "Kyle" {
		t.Fatalf("one contact, Kyle, after the merge: %+v", all)
	}
}
