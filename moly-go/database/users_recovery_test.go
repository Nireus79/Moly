package database

import (
	"database/sql"
	"testing"
)

// The upgrade script from version 10 adds the recovery hash and keeps the accounts. (SQLite cannot drop a column, so the old shape is made
// by hand: a users table as version 10 had it.)
func TestSchemaV10ToV11AddsTheRecoveryHash(t *testing.T) {
	c, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.SetMaxOpenConns(1)
	if _, err := c.Exec(`CREATE TABLE users (id TEXT PRIMARY KEY, email TEXT UNIQUE, name TEXT NOT NULL DEFAULT 'User', password_hash TEXT, created_at INTEGER NOT NULL, last_active INTEGER NOT NULL);
		INSERT INTO users (id, name, password_hash, created_at, last_active) VALUES ('acc1', 'Anna', 'x', 1, 1);
		PRAGMA user_version = 10;`); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Exec(schemaV10ToV11); err != nil {
		t.Fatalf("upgrade script: %v", err)
	}
	var version int
	c.QueryRow(`PRAGMA user_version`).Scan(&version)
	if version != 11 {
		t.Fatalf("version %d, want 11", version)
	}
	var name string
	var hash sql.NullString
	if err := c.QueryRow(`SELECT name, recovery_hash FROM users WHERE id = 'acc1'`).Scan(&name, &hash); err != nil || name != "Anna" || hash.Valid {
		t.Fatalf("account after upgrade: %v %q %v", err, name, hash)
	}
}
