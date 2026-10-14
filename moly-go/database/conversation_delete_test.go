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

// What is unresolved belongs to its conversation and goes with it; a named person belongs to the user and stays.
func TestDeleteConversationTakesItsUnresolvedMattersAndKeepsNamedPeople(t *testing.T) {
	db := scopeDB(t)
	c := db.GetConnection()
	now := time.Now().Unix()
	c.Exec(`INSERT INTO conversations (id, user_id, name, created_at, updated_at) VALUES ('c1', 'u1', 'n', ?, ?), ('c2', 'u1', 'n', ?, ?)`, now, now, now, now)
	NewContactRepository(db).Save(person("Dana", "named", "professional"))
	NewContactRepository(db).InConversation("c1").Save(person("boss", "asked", "professional"))
	NewContactRepository(db).InConversation("c2").Save(person("sister", "asked", "family"))
	for _, conv := range []string{"c1", "c2"} {
		db.GetContextConflictRepository().Save(&ContextConflict{UserID: "u1", ConversationID: conv, ConflictType: "contact_relationship", Severity: "low", Description: "d", Status: "unresolved", CreatedAt: now})
	}
	for _, r := range []struct{ conv, status string }{{"c1", "pending_approval"}, {"c1", "approved"}, {"c2", "pending_approval"}} {
		c.Exec(`INSERT INTO reflections (user_id, conversation_id, characteristics, status, created_at) VALUES ('u1', ?, '[]', ?, ?)`, r.conv, r.status, now)
	}

	if err := DeleteConversationData(c, "u1", "c1"); err != nil {
		t.Fatal(err)
	}
	var people, conflicts int
	c.QueryRow(`SELECT COUNT(*) FROM contacts WHERE name = 'boss'`).Scan(&people)
	c.QueryRow(`SELECT COUNT(*) FROM context_conflicts WHERE conversation_id = 'c1'`).Scan(&conflicts)
	if people != 0 || conflicts != 0 {
		t.Fatalf("the conversation's unresolved matters must go with it: %d people, %d conflicts left", people, conflicts)
	}
	var pendingGone, approvedKept, otherKept int
	c.QueryRow(`SELECT COUNT(*) FROM reflections WHERE conversation_id = 'c1' AND status = 'pending_approval'`).Scan(&pendingGone)
	c.QueryRow(`SELECT COUNT(*) FROM reflections WHERE conversation_id = 'c1' AND status = 'approved'`).Scan(&approvedKept)
	c.QueryRow(`SELECT COUNT(*) FROM reflections WHERE conversation_id = 'c2'`).Scan(&otherKept)
	if pendingGone != 0 || approvedKept != 1 || otherKept != 1 {
		t.Fatalf("unconfirmed insights go with their conversation, confirmed ones and other conversations' stay: %d, %d, %d", pendingGone, approvedKept, otherKept)
	}
	c.QueryRow(`SELECT COUNT(*) FROM contacts WHERE name IN ('Dana', 'sister')`).Scan(&people)
	c.QueryRow(`SELECT COUNT(*) FROM context_conflicts WHERE conversation_id = 'c2'`).Scan(&conflicts)
	if people != 2 || conflicts != 1 {
		t.Fatalf("a named person and another conversation's matters stay: %d people, %d conflicts", people, conflicts)
	}
}
