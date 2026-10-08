package database

import (
	"path/filepath"
	"testing"
	"time"
)

func seedUserAndConversation(t *testing.T, db *Database, userID, convID string) {
	t.Helper()
	now := time.Now().Unix()
	c := db.GetConnection()
	if _, err := c.Exec(`INSERT OR IGNORE INTO users (id, created_at, last_active) VALUES (?, ?, ?)`, userID, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Exec(`INSERT OR IGNORE INTO conversations (id, user_id, name, created_at, updated_at) VALUES (?, ?, 'n', ?, ?)`, convID, userID, now, now); err != nil {
		t.Fatal(err)
	}
}

func TestConversationContextRoundTripAndOwnership(t *testing.T) {
	db, err := initWithKey(filepath.Join(t.TempDir(), "ctx.db"), testKey(t))
	if err != nil {
		t.Fatal(err)
	}
	seedUserAndConversation(t, db, "u1", "c1")
	repo := NewConversationContextRepository(db)

	if err := repo.Save("u1", "c1", []byte(`{"goal":"x"}`)); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save("u1", "c1", []byte(`{"goal":"y"}`)); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Load("u1", "c1")
	if err != nil || string(got) != `{"goal":"y"}` {
		t.Fatalf("load after update: %v %q", err, got)
	}
	other, err := repo.Load("someone-else", "c1")
	if err != nil || other != nil {
		t.Fatalf("another user must not read the context: %v %q", err, other)
	}
	missing, err := repo.Load("u1", "no-such-conversation")
	if err != nil || missing != nil {
		t.Fatalf("missing conversation should return nil: %v %q", err, missing)
	}
}

func TestSchemaUpgradeFromVersion1KeepsData(t *testing.T) {
	db, err := initWithKey(filepath.Join(t.TempDir(), "up.db"), testKey(t))
	if err != nil {
		t.Fatal(err)
	}
	seedUserAndConversation(t, db, "u1", "c1")
	c := db.GetConnection()
	if _, err := c.Exec(`DROP TABLE conversation_context; PRAGMA user_version = 1;`); err != nil {
		t.Fatal(err)
	}
	if err := db.applySchema(); err != nil {
		t.Fatalf("upgrade from v1: %v", err)
	}
	var version int
	c.QueryRow(`PRAGMA user_version`).Scan(&version)
	if version != SchemaVersion {
		t.Fatalf("version after upgrade = %d, want %d", version, SchemaVersion)
	}
	var n int
	c.QueryRow(`SELECT COUNT(*) FROM conversations WHERE id = 'c1'`).Scan(&n)
	if n != 1 {
		t.Fatal("existing conversation data lost during upgrade")
	}
	if err := NewConversationContextRepository(db).Save("u1", "c1", []byte(`{}`)); err != nil {
		t.Fatalf("new table unusable after upgrade: %v", err)
	}
}
