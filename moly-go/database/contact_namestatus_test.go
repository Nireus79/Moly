package database

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"moly/models"
)

// A real version 2 database has no contacts.name_status column. The upgrade adds it, keeps existing contacts,
// and marks them 'named' (they keep their names).
func TestSchemaUpgradeFromVersion2AddsNameStatus(t *testing.T) {
	db, err := initWithKey(filepath.Join(t.TempDir(), "v2.db"), testKey(t))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	c := db.GetConnection()
	c.Exec(`INSERT INTO users (id, created_at, last_active) VALUES ('u1', ?, ?)`, now, now)
	// Rebuild contacts exactly as version 2 had it (no name_status), then mark the database as version 2.
	if err := rebuildContactsWithoutNameStatus(db); err != nil {
		t.Fatalf("simulating version 2: %v", err)
	}
	if _, err := c.Exec(`INSERT INTO contacts (user_id, name, relationship, status, created_at, updated_at) VALUES ('u1', 'Ann', 'friend', 'active', ?, ?)`, now, now); err != nil {
		t.Fatal(err)
	}
	if err := db.applySchema(); err != nil {
		t.Fatalf("upgrade from v2: %v", err)
	}
	var version int
	c.QueryRow("PRAGMA user_version").Scan(&version)
	if version != SchemaVersion {
		t.Fatalf("expected version %d, got %d", SchemaVersion, version)
	}
	list, err := NewContactRepository(db).GetByUserID("u1")
	if err != nil || len(list) != 1 || list[0].Name != "Ann" || list[0].NameStatus != "named" {
		t.Fatalf("existing contact not kept as named: %v %+v", err, list)
	}
}

// name_status round-trips through Save and both reads (GetByUserID and GetByID).
func TestNameStatusRoundTrip(t *testing.T) {
	db, err := initWithKey(filepath.Join(t.TempDir(), "ns.db"), testKey(t))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	db.GetConnection().Exec(`INSERT INTO users (id, created_at, last_active) VALUES ('u1', ?, ?)`, now, now)
	repo := NewContactRepository(db).InConversation("c1")
	contact := &models.Contact{UserID: "u1", Name: "girl", Relationship: "other", Status: "active", NameStatus: "unnamed"}
	if err := repo.Save(contact); err != nil {
		t.Fatal(err)
	}
	byID, err := repo.GetByID("u1", contact.ID)
	if err != nil || byID == nil {
		t.Fatalf("GetByID failed (this used to fail on a column-count mismatch): %v", err)
	}
	if byID.NameStatus != "unnamed" {
		t.Fatalf("GetByID name status: %q", byID.NameStatus)
	}
	list, err := repo.GetByUserID("u1")
	if err != nil || len(list) != 1 || list[0].NameStatus != "unnamed" {
		t.Fatalf("GetByUserID name status: %v %+v", err, list)
	}
}

// rebuildContactsWithoutNameStatus recreates the contacts table from schema.sql without the name_status column,
// and sets the schema version to 2. It is the layout a version 2 database has.
func rebuildContactsWithoutNameStatus(db *Database) error {
	raw, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return err
	}
	re := regexp.MustCompile(`(?s)CREATE TABLE contacts \(.*?\n\);`)
	create := re.FindString(string(raw))
	if create == "" {
		return fmt.Errorf("contacts CREATE TABLE not found in schema.sql")
	}
	v2Create := strings.Replace(create, "CREATE TABLE contacts (", "CREATE TABLE contacts_v2 (", 1)
	v2Create = regexp.MustCompile(`(?m)^\s*name_status .*\n`).ReplaceAllString(v2Create, "")
	statements := "PRAGMA foreign_keys = OFF; DROP TABLE contacts; " + strings.Replace(v2Create, "contacts_v2 (", "contacts (", 1) + " PRAGMA user_version = 2; PRAGMA foreign_keys = ON;"
	_, err = db.GetConnection().Exec(statements)
	return err
}

func setupNameGateRepo(t *testing.T) *ContactRepository {
	t.Helper()
	db, err := initWithKey(filepath.Join(t.TempDir(), "gate.db"), testKey(t))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	db.GetConnection().Exec(`INSERT INTO users (id, created_at, last_active) VALUES ('u1', ?, ?)`, now, now)
	repo := NewContactRepository(db).InConversation("c1")
	if err := repo.Save(&models.Contact{UserID: "u1", Name: "girl", Relationship: "other", Status: "active", NameStatus: "asked"}); err != nil {
		t.Fatal(err)
	}
	return repo
}

// A given name renames the asked contact in place: one row, now named.
func TestApplyNameAnswerRenamesTheSameRow(t *testing.T) {
	repo := setupNameGateRepo(t)
	answered, err := repo.ApplyNameAnswer("u1", "Christine", "", false)
	if err != nil || !answered {
		t.Fatalf("answer not applied: %v %v", answered, err)
	}
	list, _ := repo.GetByUserID("u1")
	if len(list) != 1 || list[0].Name != "Christine" || list[0].NameStatus != "named" {
		t.Fatalf("expected one renamed contact, got %+v", list)
	}
}

// No given name keeps the label as the name: the person stays one of this conversation and is not asked again.
func TestApplyNameAnswerWithoutNameKeepsLabelAndStops(t *testing.T) {
	repo := setupNameGateRepo(t)
	if answered, err := repo.ApplyNameAnswer("u1", "", "", true); err != nil || !answered {
		t.Fatalf("first answer: %v %v", answered, err)
	}
	list, _ := repo.GetByUserID("u1")
	if len(list) != 1 || list[0].Name != "girl" || list[0].NameStatus != "nameless" {
		t.Fatalf("label not kept, nameless: %+v", list)
	}
	if answered, _ := repo.ApplyNameAnswer("u1", "", "", true); answered {
		t.Fatal("a contact that is no longer asked must not count as answered again")
	}
}

func TestGetByNameReturnsNameStatus(t *testing.T) {
	repo := setupNameGateRepo(t)
	c, err := repo.GetByName("u1", "girl")
	if err != nil || c == nil || c.NameStatus != "asked" {
		t.Fatalf("GetByName status: %v %+v", err, c)
	}
}

// What the message said the person is to the user ("manager") stays their role when the name answer renames the person. The label
// alone is not a role: "father" does not say whose father (see TestTheLabelAloneIsNotARole).
func TestNameAnswerKeepsTheOldLabelAsTheRole(t *testing.T) {
	db, err := initWithKey(filepath.Join(t.TempDir(), "role.db"), testKey(t))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	db.GetConnection().Exec(`INSERT INTO users (id, created_at, last_active) VALUES ('u1', ?, ?)`, now, now)
	repo := NewContactRepository(db).InConversation("c1")
	c := &models.Contact{UserID: "u1", Name: "manager", ContactRole: "manager", Relationship: "professional", Status: "active", NameStatus: "asked"}
	if err := repo.Save(c); err != nil {
		t.Fatal(err)
	}
	if answered, err := repo.ApplyNameAnswer("u1", "Dana", "", false); err != nil || !answered {
		t.Fatalf("name answer: %v %v", answered, err)
	}
	got, err := repo.GetByName("u1", "Dana")
	if err != nil || got == nil {
		t.Fatalf("renamed contact not found: %v", err)
	}
	if got.ContactRole != "manager" || got.NameStatus != "named" || got.ID != c.ID {
		t.Fatalf("same person, named Dana, role manager; got %+v", got)
	}
}

// Found live (2026-10-14): a person called only "father" got the role "father" when named, and was then shown as "the user's
// father" although he was someone else's father. A label says what the user called the person, not what the person is to them.
func TestTheLabelAloneIsNotARole(t *testing.T) {
	db, err := initWithKey(filepath.Join(t.TempDir(), "label.db"), testKey(t))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	db.GetConnection().Exec(`INSERT INTO users (id, created_at, last_active) VALUES ('u1', ?, ?)`, now, now)
	repo := NewContactRepository(db).InConversation("c1")
	c := &models.Contact{UserID: "u1", Name: "father", Relationship: "family", Status: "active", NameStatus: "asked"}
	if err := repo.Save(c); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ApplyNameAnswer("u1", "Alex", "", false); err != nil {
		t.Fatal(err)
	}
	got, _ := repo.GetByName("u1", "Alex")
	if got == nil || got.ContactRole != "" {
		t.Fatalf("named, with no role of the user, got %+v", got)
	}
}

// Found live (2026-10-15): a message that is not an answer closed the question, the person became nameless for good, and the name
// the user gave a turn later could not reach them. Only a name or a decline closes the question.
func TestAMessageThatIsNotAnAnswerLeavesTheNameQuestionOpen(t *testing.T) {
	repo := setupNameGateRepo(t)
	if answered, err := repo.ApplyNameAnswer("u1", "", "", false); err != nil || answered {
		t.Fatalf("no name and no decline answers nothing: %v %v", answered, err)
	}
	list, _ := repo.GetByUserID("u1")
	if len(list) != 1 || list[0].NameStatus != "asked" {
		t.Fatalf("the question stays open, got %+v", list)
	}
	if answered, err := repo.ApplyNameAnswer("u1", "Alex", "", false); err != nil || !answered {
		t.Fatalf("the name given later reaches the person: %v %v", answered, err)
	}
	if got, _ := repo.GetByName("u1", "Alex"); got == nil || got.NameStatus != "named" {
		t.Fatalf("named Alex, got %+v", got)
	}
}

// Two people wait for a name: a name goes to the one it is for, not to both.
func TestANameGoesOnlyToThePersonItIsFor(t *testing.T) {
	db, err := initWithKey(filepath.Join(t.TempDir(), "two.db"), testKey(t))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	db.GetConnection().Exec(`INSERT INTO users (id, created_at, last_active) VALUES ('u1', ?, ?)`, now, now)
	repo := NewContactRepository(db).InConversation("c1")
	for _, label := range []string{"boss", "father"} {
		if err := repo.Save(&models.Contact{UserID: "u1", Name: label, Relationship: "other", Status: "active", NameStatus: "asked"}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := repo.ApplyNameAnswer("u1", "Alex", "father", false); err != nil {
		t.Fatal(err)
	}
	boss, _ := repo.GetByName("u1", "boss")
	alex, _ := repo.GetByName("u1", "Alex")
	if boss == nil || boss.NameStatus != "asked" || alex == nil || alex.NameStatus != "named" {
		t.Fatalf("Alex is the father and the boss still waits: boss=%+v alex=%+v", boss, alex)
	}
}
