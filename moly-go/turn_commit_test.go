package main

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/mutecomm/go-sqlcipher/v4"
)

func commitTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "commit.db"))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`CREATE TABLE t (id INTEGER PRIMARY KEY, v TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func rowCount(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM t`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestTurnCommitBuffersUntilRun(t *testing.T) {
	db := commitTestDB(t)
	c := &turnCommit{}
	c.Exec(`INSERT INTO t (v) VALUES (?)`, "a")
	if rowCount(t, db) != 0 || c.Len() != 1 {
		t.Fatal("nothing may be written before Run")
	}
	ran := false
	c.AfterCommit(func() { ran = true })
	if err := c.Run(db); err != nil {
		t.Fatal(err)
	}
	if rowCount(t, db) != 1 || !ran {
		t.Fatal("Run must write the buffered rows and then run the after-commit actions")
	}
}

func TestTurnCommitIsAllOrNothing(t *testing.T) {
	db := commitTestDB(t)
	c := &turnCommit{}
	c.Exec(`INSERT INTO t (v) VALUES (?)`, "a")
	c.Exec(`INSERT INTO t (v) VALUES (?)`, nil) // violates NOT NULL
	c.Exec(`INSERT INTO t (v) VALUES (?)`, "c")
	ran := false
	c.AfterCommit(func() { ran = true })
	if err := c.Run(db); err == nil {
		t.Fatal("a failing write must fail the commit")
	}
	if rowCount(t, db) != 0 {
		t.Fatalf("a failed commit must leave nothing behind, got %d rows", rowCount(t, db))
	}
	if ran {
		t.Fatal("after-commit actions must not run when the commit fails")
	}
}
