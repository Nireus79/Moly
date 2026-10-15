package database

import (
	"strings"
	"testing"
)

func relationTexts(c interface{ RelationsText() string }) string { return c.RelationsText() }

// Whose father, whose boss: a person has several relations, each of someone (a person, or the user).
func TestAPersonHasSeveralRelationsEachOfSomeone(t *testing.T) {
	db := scopeDB(t)
	known := NewContactRepository(db)
	kyle := person("Kyle", "named", "professional")
	alex := person("Alex", "named", "family")
	known.Save(kyle)
	known.Save(alex)
	if err := known.AddRelation("u1", alex.ID, "father", kyle.ID, false); err != nil {
		t.Fatal(err)
	}
	if err := known.AddRelation("u1", alex.ID, "boss", 0, true); err != nil {
		t.Fatal(err)
	}
	got, _ := known.GetByName("u1", "Alex")
	if text := got.RelationsText(); text != "Kyle's father; the user's former boss" {
		t.Fatalf("both relations, each of the right person, got %q", text)
	}
	// the same relation again updates it, it does not repeat
	known.AddRelation("u1", alex.ID, "boss", 0, false)
	got, _ = known.GetByName("u1", "Alex")
	if text := got.RelationsText(); text != "Kyle's father; the user's boss" {
		t.Fatalf("the boss is no longer former, once, got %q", text)
	}
	list, _ := known.GetByUserID("u1")
	for _, c := range list {
		if c.Name == "Alex" && len(c.Relations) != 2 {
			t.Fatalf("a list carries the relations too, got %+v", c.Relations)
		}
	}
}

// A person is not their own relation, and a relation needs both people to be ones the repository sees.
func TestRelationsNeedRealPeopleAndNeverRelateAPersonToThemself(t *testing.T) {
	db := scopeDB(t)
	known := NewContactRepository(db)
	a := person("Dana", "named", "professional")
	known.Save(a)
	if known.AddRelation("u1", a.ID, "boss", a.ID, false) == nil {
		t.Fatal("a person cannot be their own boss")
	}
	if known.AddRelation("u1", a.ID, "boss", 9999, false) == nil {
		t.Fatal("no such person")
	}
	if known.AddRelation("u1", a.ID, "  ", 0, false) == nil {
		t.Fatal("a relation needs a word")
	}
	if known.AddRelation("other-user", a.ID, "boss", 0, false) == nil {
		t.Fatal("another user's person is not visible")
	}
}

// An unresolved person's relations stay in the conversation the person belongs to; a named person never shows a relation to an
// unresolved person of another conversation.
func TestRelationsFollowTheVisibilityOfThePeople(t *testing.T) {
	db := scopeDB(t)
	known := NewContactRepository(db)
	convA := known.InConversation("convA")
	convB := known.InConversation("convB")
	dana := person("Dana", "named", "professional")
	known.Save(dana)
	waiting := person("the neighbour", "asked", "other")
	convA.Save(waiting)
	if err := convA.AddRelation("u1", dana.ID, "friend", waiting.ID, false); err != nil {
		t.Fatal(err)
	}
	if err := convA.AddRelation("u1", waiting.ID, "neighbour", 0, false); err != nil {
		t.Fatal(err)
	}
	inA, _ := convA.GetByName("u1", "Dana")
	if !strings.Contains(inA.RelationsText(), "the neighbour's friend") {
		t.Fatalf("conversation A sees the relation, got %q", inA.RelationsText())
	}
	inB, _ := convB.GetByName("u1", "Dana")
	if inB.RelationsText() != "" {
		t.Fatalf("conversation B must not see a relation to A's unresolved person, got %q", inB.RelationsText())
	}
	if hidden, _ := convB.GetByName("u1", "the neighbour"); hidden != nil {
		t.Fatal("the unresolved person itself stays in A")
	}
}

// When the name makes a waiting person the same person as one the user already has, what was said of either holds for the one left.
func TestRelationsMoveToTheSavedPersonWhenTheNameIsTheirs(t *testing.T) {
	db := scopeDB(t)
	known := NewContactRepository(db)
	kyle := person("Kyle", "named", "professional")
	alex := person("Alex", "named", "family")
	known.Save(kyle)
	known.Save(alex)
	conv := known.InConversation("convA")
	father := person("father", "asked", "family")
	conv.Save(father)
	if err := conv.AddRelation("u1", father.ID, "father", kyle.ID, false); err != nil {
		t.Fatal(err)
	}
	if err := conv.AddRelation("u1", father.ID, "boss", 0, true); err != nil {
		t.Fatal(err)
	}
	known.AddRelation("u1", alex.ID, "boss", 0, true) // the saved Alex already was the former boss: not repeated
	if _, err := conv.ApplyNameAnswer("u1", "Alex"); err != nil {
		t.Fatal(err)
	}
	got, _ := known.GetByName("u1", "Alex")
	if text := got.RelationsText(); text != "Kyle's father; the user's former boss" {
		t.Fatalf("Alex has both, once, got %q", text)
	}
	if gone, _ := conv.GetByName("u1", "father"); gone != nil {
		t.Fatal("the waiting person is gone after the merge")
	}
}

// A renamed person keeps the relations (the same row).
func TestRelationsStayWhenAWaitingPersonIsNamed(t *testing.T) {
	db := scopeDB(t)
	known := NewContactRepository(db)
	kyle := person("Kyle", "named", "professional")
	known.Save(kyle)
	conv := known.InConversation("convA")
	father := person("father", "asked", "family")
	conv.Save(father)
	conv.AddRelation("u1", father.ID, "father", kyle.ID, false)
	if _, err := conv.ApplyNameAnswer("u1", "Alex"); err != nil {
		t.Fatal(err)
	}
	got, _ := known.GetByName("u1", "Alex")
	if got == nil || got.RelationsText() != "Kyle's father" {
		t.Fatalf("got %+v", got)
	}
}

// The migration from version 6 adds the table and keeps the data.
func TestUpgradeFromVersion6AddsRelations(t *testing.T) {
	db := scopeDB(t)
	c := db.GetConnection()
	if _, err := c.Exec(`DROP TABLE contact_relations; PRAGMA user_version = 6;`); err != nil {
		t.Fatal(err)
	}
	if err := db.applySchema(); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := c.QueryRow(`SELECT COUNT(*) FROM contact_relations`).Scan(&n); err != nil {
		t.Fatalf("the table must exist after the upgrade: %v", err)
	}
}

// Found live (2026-10-14): saving a person again deleted the row and made a new one (a new id each time). A person keeps their row
// and their id, and what points at them (their relations) survives a save.
func TestSavingAPersonAgainKeepsTheRowAndTheRelations(t *testing.T) {
	db := scopeDB(t)
	known := NewContactRepository(db)
	kyle := person("Kyle", "named", "professional")
	known.Save(kyle)
	alex := person("Alex", "named", "family")
	known.Save(alex)
	first := alex.ID
	known.AddRelation("u1", alex.ID, "father", kyle.ID, false)

	again, _ := known.GetByName("u1", "Alex")
	again.Characteristics = []string{"fair"}
	if err := known.Save(again); err != nil {
		t.Fatal(err)
	}
	fresh := person("Alex", "named", "family") // saved by a caller that only knows the name
	fresh.Characteristics = []string{"kind"}
	if err := known.Save(fresh); err != nil {
		t.Fatal(err)
	}
	got, _ := known.GetByName("u1", "Alex")
	if got.ID != first || fresh.ID != first {
		t.Fatalf("one row, one id (%d): got %d and %d", first, got.ID, fresh.ID)
	}
	if got.RelationsText() != "Kyle's father" {
		t.Fatalf("the relation survives a save, got %q", got.RelationsText())
	}
	list, _ := known.GetByUserID("u1")
	if len(list) != 2 {
		t.Fatalf("still two people, got %d", len(list))
	}
}
