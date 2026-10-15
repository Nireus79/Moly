package database

import (
	"path/filepath"
	"testing"
	"time"

	"moly/models"
)

func scopeDB(t *testing.T) *Database {
	t.Helper()
	db, err := initWithKey(filepath.Join(t.TempDir(), "scope.db"), testKey(t))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	for _, u := range []string{"u1", "u2"} {
		db.GetConnection().Exec(`INSERT INTO users (id, created_at, last_active) VALUES (?, ?, ?)`, u, now, now)
	}
	return db
}

func person(name, status, rel string) *models.Contact {
	return &models.Contact{UserID: "u1", Name: name, NameStatus: status, Relationship: rel, Status: "active"}
}

func names(list []*models.Contact) []string {
	var out []string
	for _, c := range list {
		out = append(out, c.Name)
	}
	return out
}

// An unresolved person belongs to the conversation it came from: no other conversation, and no read without a conversation,
// can see it. A named person belongs to the user.
func TestAnUnnamedPersonIsSeenOnlyInItsOwnConversation(t *testing.T) {
	db := scopeDB(t)
	a := NewContactRepository(db).InConversation("convA")
	if err := a.Save(person("boss", "asked", "professional")); err != nil {
		t.Fatal(err)
	}
	if err := a.Save(person("Dana", "named", "professional")); err != nil {
		t.Fatal(err)
	}
	for conv, want := range map[string]int{"convA": 2, "convB": 1, "": 1} {
		repo := NewContactRepository(db)
		if conv != "" {
			repo = repo.InConversation(conv)
		}
		list, err := repo.GetByUserID("u1")
		if err != nil || len(list) != want {
			t.Fatalf("conversation %q sees %v (err %v), want %d people", conv, names(list), err, want)
		}
		if c, _ := repo.GetByName("u1", "boss"); (c != nil) != (conv == "convA") {
			t.Fatalf("conversation %q: the unnamed boss visible = %v", conv, c != nil)
		}
	}
	// no user sees another user's people
	if list, _ := NewContactRepository(db).InConversation("convA").GetByUserID("u2"); len(list) != 0 {
		t.Fatalf("another user sees %v", names(list))
	}
}

func TestEachConversationHasItsOwnBossAndOneDanaExistsPerUser(t *testing.T) {
	db := scopeDB(t)
	a, b := NewContactRepository(db).InConversation("convA"), NewContactRepository(db).InConversation("convB")
	if err := a.Save(person("boss", "asked", "professional")); err != nil {
		t.Fatal(err)
	}
	if err := b.Save(person("boss", "unnamed", "family")); err != nil {
		t.Fatalf("a second conversation must be able to have its own 'boss': %v", err)
	}
	ca, _ := a.GetByName("u1", "boss")
	cb, _ := b.GetByName("u1", "boss")
	if ca == nil || cb == nil || ca.ID == cb.ID || ca.Relationship != "professional" || cb.Relationship != "family" {
		t.Fatalf("the two bosses are different people: %+v %+v", ca, cb)
	}
	if err := a.Save(person("Dana", "named", "professional")); err != nil {
		t.Fatal(err)
	}
	other := person("Dana", "named", "friend")
	if err := b.Save(other); err != nil {
		t.Fatal(err)
	}
	if list, _ := NewContactRepository(db).GetByUserID("u1"); len(list) != 1 {
		t.Fatalf("one Dana per user, got %v", names(list))
	}
}

func TestAnUnnamedPersonCannotBeSavedWithoutAConversation(t *testing.T) {
	db := scopeDB(t)
	if err := NewContactRepository(db).Save(person("boss", "unnamed", "professional")); err == nil {
		t.Fatal("an unnamed person belongs to a conversation: saving without one must fail")
	}
	if err := NewContactRepository(db).Save(person("Dana", "named", "professional")); err != nil {
		t.Fatalf("a named person needs no conversation: %v", err)
	}
}

// The name answer of one conversation never closes another conversation's question.
func TestANameAnswerAppliesOnlyToItsOwnConversation(t *testing.T) {
	db := scopeDB(t)
	a, b := NewContactRepository(db).InConversation("convA"), NewContactRepository(db).InConversation("convB")
	a.Save(person("boss", "asked", "professional"))
	b.Save(person("sister", "asked", "family"))
	answered, err := b.ApplyNameAnswer("u1", "Mara", "", false)
	if err != nil || !answered {
		t.Fatalf("convB answer: %v %v", answered, err)
	}
	if c, _ := a.GetByName("u1", "boss"); c == nil || c.NameStatus != "asked" {
		t.Fatalf("convA's question must stay open, got %+v", c)
	}
	if list, _ := NewContactRepository(db).GetByUserID("u1"); len(list) != 1 || list[0].Name != "Mara" {
		t.Fatalf("Mara is now a person of the user, got %v", names(list))
	}
}

// The name is asked first and compared after: the same name as a saved person is that person.
func TestTheNameGivenIsComparedWithTheSavedPeople(t *testing.T) {
	db := scopeDB(t)
	known := NewContactRepository(db)
	saved := person("Dana", "named", "professional")
	saved.Characteristics = []string{"task-giver"}
	known.Save(saved)

	conv := known.InConversation("convA")
	boss := person("boss", "asked", "")
	boss.ContactRole = "boss"
	boss.Characteristics = []string{"unfair"}
	conv.Save(boss)
	other := NewContactRepository(db).InConversation("convB")
	other.Save(person("boss", "asked", "professional")) // another conversation's boss stays as it is

	answered, err := conv.ApplyNameAnswer("u1", "Dana", "", false)
	if err != nil || !answered {
		t.Fatalf("name answer: %v %v", answered, err)
	}
	list, _ := known.GetByUserID("u1")
	if len(list) != 1 || list[0].Name != "Dana" {
		t.Fatalf("the boss is Dana: one person, got %v", names(list))
	}
	d := list[0]
	if d.ContactRole != "boss" || d.Relationship != "professional" || len(d.Characteristics) != 2 {
		t.Fatalf("known values kept, blanks filled, traits joined, role from the label; got %+v", d)
	}
	if c, _ := conv.GetByName("u1", "boss"); c != nil {
		t.Fatalf("the waiting person is gone after the merge: %+v", c)
	}
	if c, _ := other.GetByName("u1", "boss"); c == nil || c.NameStatus != "asked" {
		t.Fatalf("the other conversation's boss is untouched, got %+v", c)
	}
}

func TestANewNameRenamesTheSameRowAndMakesItAPersonOfTheUser(t *testing.T) {
	db := scopeDB(t)
	conv := NewContactRepository(db).InConversation("convA")
	boss := person("boss", "asked", "professional")
	boss.ContactRole = "boss"
	conv.Save(boss)
	if _, err := conv.ApplyNameAnswer("u1", "Dana", "", false); err != nil {
		t.Fatal(err)
	}
	c, _ := NewContactRepository(db).GetByName("u1", "Dana")
	if c == nil || c.ID != boss.ID || c.NameStatus != "named" || c.OriginConversationID != "" || c.ContactRole != "boss" {
		t.Fatalf("same row, named, a person of the user, role boss; got %+v", c)
	}
}

func TestSettingTheStatusNeverMakesAPersonNamed(t *testing.T) {
	db := scopeDB(t)
	conv := NewContactRepository(db).InConversation("convA")
	boss := person("boss", "unnamed", "professional")
	conv.Save(boss)
	if err := conv.SetNameStatus("u1", boss.ID, "named"); err == nil {
		t.Fatal("'named' only through NameContact, which compares the name with the saved people")
	}
}

// Unresolved conflicts belong to the conversation they came from.
func TestAnUnresolvedConflictIsAskedOnlyInItsOwnConversation(t *testing.T) {
	db := scopeDB(t)
	now := time.Now().Unix()
	for _, id := range []string{"convA", "convB"} {
		db.GetConnection().Exec(`INSERT INTO conversations (id, user_id, name, created_at, updated_at) VALUES (?, 'u1', 'x', ?, ?)`, id, now, now)
	}
	repo := db.GetContextConflictRepository()
	c := &ContextConflict{UserID: "u1", ConversationID: "convA", ConflictType: "contact_relationship", Severity: "low", Description: "d", Status: "unresolved", CreatedAt: now}
	if err := repo.Save(c); err != nil {
		t.Fatal(err)
	}
	if got, _ := repo.GetUnresolved("u1", "convA"); len(got) != 1 {
		t.Fatalf("convA should be asked, got %d", len(got))
	}
	if got, _ := repo.GetUnresolved("u1", "convB"); len(got) != 0 {
		t.Fatalf("convB must not be asked convA's conflict, got %d", len(got))
	}
	if got, _ := repo.GetUnresolved("u1", ""); len(got) != 0 {
		t.Fatalf("without a conversation nothing is asked, got %d", len(got))
	}
	if got, _ := repo.GetUnresolved("u2", "convA"); len(got) != 0 {
		t.Fatalf("another user is never asked, got %d", len(got))
	}
	if got, _ := repo.GetUnresolvedByID("u1", c.ID); got == nil {
		t.Fatal("a conflict can be fetched by its id")
	}
	if got, _ := repo.GetUnresolvedByID("u2", c.ID); got != nil {
		t.Fatal("another user's conflict is never found by id")
	}
}

// The people a conversation can point to are those it has brought up: a saved person it never mentioned is not one of them.
func TestEstablishedPeopleAreThoseTheConversationBroughtUp(t *testing.T) {
	db := scopeDB(t)
	known := NewContactRepository(db)
	known.Save(person("Dana", "named", "professional"))
	known.Save(person("Anna", "named", "friend"))
	known.InConversation("convA").Save(person("boss", "asked", "professional"))
	known.InConversation("convB").Save(person("sister", "asked", "family"))

	got := names(EstablishedPeople(known.InConversation("convA"), "u1", []string{"Hi Moly", "My friend Anna called."}))
	want := map[string]bool{"Anna": true, "boss": true}
	if len(got) != len(want) {
		t.Fatalf("established people %v, want Anna and boss (not Dana, not convB's sister)", got)
	}
	for _, n := range got {
		if !want[n] {
			t.Fatalf("unexpected established person %q in %v", n, got)
		}
	}
}

// Upgrade of a version 4 database: every person is kept, the two uniqueness rules replace UNIQUE(user_id, name), and a second
// conversation can have its own unnamed person with the same label.
func TestSchemaUpgradeFromVersion4ScopesUnnamedPeople(t *testing.T) {
	db := scopeDB(t)
	c := db.GetConnection()
	now := time.Now().Unix()
	if _, err := c.Exec(`
		DROP INDEX IF EXISTS idx_contacts_named; DROP INDEX IF EXISTS idx_contacts_unresolved;
		DROP TABLE contacts;
		CREATE TABLE contacts (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id TEXT NOT NULL, name TEXT NOT NULL, relationship TEXT, age TEXT,
			characteristics TEXT, first_mentioned_at INTEGER, created_via TEXT, "status" TEXT DEFAULT 'active', version INTEGER DEFAULT 1,
			created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL, confidence REAL DEFAULT 0.5, extraction_count INTEGER DEFAULT 1,
			last_mentioned_at INTEGER, pronouns TEXT, contact_role TEXT DEFAULT NULL, involved_intentions TEXT DEFAULT NULL,
			past_successes TEXT DEFAULT NULL, dependencies TEXT DEFAULT NULL, notes TEXT, name_status TEXT NOT NULL DEFAULT 'named',
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE, UNIQUE(user_id, name));
		PRAGMA user_version = 4;`); err != nil {
		t.Fatalf("simulating version 4: %v", err)
	}
	for _, row := range []struct{ name, status string }{{"Dana", "named"}, {"boss", "asked"}} {
		if _, err := c.Exec(`INSERT INTO contacts (user_id, name, relationship, status, name_status, created_at, updated_at) VALUES ('u1', ?, 'professional', 'active', ?, ?, ?)`, row.name, row.status, now, now); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.applySchema(); err != nil {
		t.Fatalf("upgrade from v4: %v", err)
	}
	var version int
	c.QueryRow("PRAGMA user_version").Scan(&version)
	if version != SchemaVersion {
		t.Fatalf("version %d, want %d", version, SchemaVersion)
	}
	list, _ := NewContactRepository(db).GetByUserID("u1")
	if len(list) != 1 || list[0].Name != "Dana" {
		t.Fatalf("the named person is kept and the old unnamed one (no conversation) is seen by nobody, got %v", names(list))
	}
	var rows int
	c.QueryRow(`SELECT COUNT(*) FROM contacts`).Scan(&rows)
	if rows != 2 {
		t.Fatalf("no row is lost in the upgrade, got %d", rows)
	}
	if err := NewContactRepository(db).InConversation("convA").Save(person("boss", "unnamed", "professional")); err != nil {
		t.Fatalf("a new conversation can have its own 'boss' after the upgrade: %v", err)
	}
}

func TestJudgeLinesCarryTheRoleOfANamedPerson(t *testing.T) {
	lines := JudgeLines([]*models.Contact{
		{Name: "Dana", NameStatus: "named", ContactRole: "boss"},
		{Name: "Anna", NameStatus: "named"},
		{Name: "sister", NameStatus: "asked", ContactRole: "should-not-show"},
	})
	if len(lines) != 3 || lines[0] != "Dana (the user's boss)" || lines[1] != "Anna" || lines[2] != "sister" {
		t.Fatalf("got %v", lines)
	}
}
