package database

import (
	"path/filepath"
	"testing"
	"time"
)

func TestDeleteConversationRemovesAllItsData(t *testing.T) {
	db, err := initWithKey(filepath.Join(t.TempDir(), "del.db"), testKey(t))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	c := db.GetConnection()
	c.Exec(`INSERT INTO users (id, created_at, last_active) VALUES ('u1', ?, ?)`, now, now)
	c.Exec(`INSERT INTO conversations (id, user_id, name, created_at, updated_at) VALUES ('c1', 'u1', 'n', ?, ?)`, now, now)
	c.Exec(`INSERT INTO chat_messages (id, user_id, conversation_id, role, content, created_at) VALUES ('m1', 'u1', 'c1', 'user', 'text', ?)`, now)
	c.Exec(`INSERT INTO conversation_maturity (id, user_id, conversation_id, current_phase, maturity, created_at, updated_at) VALUES ('mt1', 'u1', 'c1', 'initial', 0.1, ?, ?)`, now, now)

	if err := DeleteConversationData(c, "u1", "c1"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	var n int
	c.QueryRow(`SELECT COUNT(*) FROM conversations WHERE id = 'c1'`).Scan(&n)
	if n != 0 {
		t.Fatal("conversation still present")
	}
	c.QueryRow(`SELECT COUNT(*) FROM chat_messages WHERE conversation_id = 'c1'`).Scan(&n)
	if n != 0 {
		t.Fatal("messages left behind")
	}
}

func TestDeleteConversationOnlyForOwner(t *testing.T) {
	db, err := initWithKey(filepath.Join(t.TempDir(), "own.db"), testKey(t))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	c := db.GetConnection()
	c.Exec(`INSERT INTO users (id, created_at, last_active) VALUES ('u1', ?, ?)`, now, now)
	c.Exec(`INSERT INTO conversations (id, user_id, name, created_at, updated_at) VALUES ('c1', 'u1', 'n', ?, ?)`, now, now)
	if err := DeleteConversationData(c, "someone-else", "c1"); err != nil {
		t.Fatal(err)
	}
	var n int
	c.QueryRow(`SELECT COUNT(*) FROM conversations WHERE id = 'c1'`).Scan(&n)
	if n != 1 {
		t.Fatal("another user deleted this conversation")
	}
}
