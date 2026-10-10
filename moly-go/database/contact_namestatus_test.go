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
	if version != 3 {
		t.Fatalf("expected version 3, got %d", version)
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
	repo := NewContactRepository(db)
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
	repo := NewContactRepository(db)
	if err := repo.Save(&models.Contact{UserID: "u1", Name: "girl", Relationship: "other", Status: "active", NameStatus: "asked"}); err != nil {
		t.Fatal(err)
	}
	return repo
}

// A given name renames the asked contact in place: one row, now named.
func TestApplyNameAnswerRenamesTheSameRow(t *testing.T) {
	repo := setupNameGateRepo(t)
	answered, err := repo.ApplyNameAnswer("u1", "Christine")
	if err != nil || !answered {
		t.Fatalf("answer not applied: %v %v", answered, err)
	}
	list, _ := repo.GetByUserID("u1")
	if len(list) != 1 || list[0].Name != "Christine" || list[0].NameStatus != "named" {
		t.Fatalf("expected one renamed contact, got %+v", list)
	}
}

// No given name keeps the label as the name, and the contact is not asked again.
func TestApplyNameAnswerWithoutNameKeepsLabelAndStops(t *testing.T) {
	repo := setupNameGateRepo(t)
	if answered, err := repo.ApplyNameAnswer("u1", ""); err != nil || !answered {
		t.Fatalf("first answer: %v %v", answered, err)
	}
	list, _ := repo.GetByUserID("u1")
	if len(list) != 1 || list[0].Name != "girl" || list[0].NameStatus != "named" {
		t.Fatalf("label not kept as named: %+v", list)
	}
	if answered, _ := repo.ApplyNameAnswer("u1", ""); answered {
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

// The words the user first used ("my manager") are kept as the role when the name answer renames the person.
func TestNameAnswerKeepsTheOldLabelAsTheRole(t *testing.T) {
	db, err := initWithKey(filepath.Join(t.TempDir(), "role.db"), testKey(t))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	db.GetConnection().Exec(`INSERT INTO users (id, created_at, last_active) VALUES ('u1', ?, ?)`, now, now)
	repo := NewContactRepository(db)
	c := &models.Contact{UserID: "u1", Name: "manager", Relationship: "professional", Status: "active", NameStatus: "asked"}
	if err := repo.Save(c); err != nil {
		t.Fatal(err)
	}
	if answered, err := repo.ApplyNameAnswer("u1", "Dana"); err != nil || !answered {
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
