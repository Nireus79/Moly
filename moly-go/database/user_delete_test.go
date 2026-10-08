package database

import (
	"path/filepath"
	"testing"
	"time"
)

func TestDeleteUserRemovesAllTheirData(t *testing.T) {
	db, err := initWithKey(filepath.Join(t.TempDir(), "user.db"), testKey(t))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	c := db.GetConnection()
	for _, u := range []string{"u1", "u2"} {
		c.Exec(`INSERT INTO users (id, created_at, last_active) VALUES (?, ?, ?)`, u, now, now)
		c.Exec(`INSERT INTO conversations (id, user_id, name, created_at, updated_at) VALUES (?, ?, 'n', ?, ?)`, "c-"+u, u, now, now)
		c.Exec(`INSERT INTO chat_messages (id, user_id, conversation_id, role, content, created_at) VALUES (?, ?, ?, 'user', 'text', ?)`, "m-"+u, u, "c-"+u, now)
		c.Exec(`INSERT INTO conversation_maturity (id, user_id, conversation_id, current_phase, maturity, created_at, updated_at) VALUES (?, ?, ?, 'initial', 0.1, ?, ?)`, "mt-"+u, u, "c-"+u, now, now)
		c.Exec(`INSERT INTO contacts (id, user_id, name, created_at, updated_at) VALUES (?, ?, 'Ann', ?, ?)`, "ct-"+u, u, now, now)
	}

	if err := DeleteUserData(c, "u1"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	count := func(q string, args ...interface{}) int {
		var n int
		c.QueryRow(q, args...).Scan(&n)
		return n
	}
	if n := count(`SELECT COUNT(*) FROM users WHERE id = 'u1'`); n != 0 {
		t.Fatal("user row remains")
	}
	for _, q := range []string{
		`SELECT COUNT(*) FROM conversations WHERE user_id = 'u1'`,
		`SELECT COUNT(*) FROM chat_messages WHERE user_id = 'u1'`,
		`SELECT COUNT(*) FROM conversation_maturity WHERE user_id = 'u1'`,
		`SELECT COUNT(*) FROM contacts WHERE user_id = 'u1'`,
	} {
		if n := count(q); n != 0 {
			t.Fatalf("data left behind: %s -> %d", q, n)
		}
	}
	if n := count(`SELECT COUNT(*) FROM users WHERE id = 'u2'`); n != 1 {
		t.Fatal("another user was deleted")
	}
	if n := count(`SELECT COUNT(*) FROM chat_messages WHERE user_id = 'u2'`); n != 1 {
		t.Fatal("another user's messages were deleted")
	}
}
