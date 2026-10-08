package database

import (
	"path/filepath"
	"sync"
	"testing"
)

// Deleting a user must delete their messages on every pooled connection.
func TestUserDeleteCascadesAcrossConnections(t *testing.T) {
	db, err := initWithKey(filepath.Join(t.TempDir(), "cascade.db"), testKey(t))
	if err != nil {
		t.Fatal(err)
	}
	c := db.GetConnection()
	c.Exec(`INSERT INTO users (id, created_at, last_active) VALUES ('u1', 1, 1)`)
	c.Exec(`INSERT INTO conversations (id, user_id, name, created_at, updated_at) VALUES ('c1','u1','n',1,1)`)
	c.Exec(`INSERT INTO chat_messages (id, user_id, conversation_id, role, content, created_at) VALUES ('m1','u1','c1','user','text',1)`)

	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); c.Exec(`SELECT 1 FROM users`) }()
	}
	wg.Wait()

	c.Exec(`DELETE FROM users WHERE id = 'u1'`)
	var n int
	c.QueryRow(`SELECT COUNT(*) FROM chat_messages WHERE user_id = 'u1'`).Scan(&n)
	if n != 0 {
		t.Fatalf("%d chat messages left after deleting the user", n)
	}
}
