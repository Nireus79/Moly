package main

import (
	"database/sql"
	"fmt"
)

// turnCommit is the commit stage of a message (ORCHESTRATOR_DESIGN.md step 5). During the turn, the writes that make
// up the turn are buffered here, not executed. After the reply is resolved they run in one transaction: all of them
// or none. If the transaction fails, no reply is returned.
//
// It has the same Exec shape as *sql.DB, so a write site changes only the receiver.
type turnCommit struct {
	writes []bufferedWrite
	after  []func()
}

type bufferedWrite struct {
	query string
	args  []interface{}
}

type bufferedResult struct{}

func (bufferedResult) LastInsertId() (int64, error) { return 0, nil }
func (bufferedResult) RowsAffected() (int64, error) { return 0, nil }

// Exec buffers a write. It never fails; failures surface when the commit runs.
func (c *turnCommit) Exec(query string, args ...interface{}) (sql.Result, error) {
	c.writes = append(c.writes, bufferedWrite{query: query, args: args})
	return bufferedResult{}, nil
}

// AfterCommit registers something to run only if the transaction succeeded (for example filling an in-memory cache).
func (c *turnCommit) AfterCommit(f func()) {
	c.after = append(c.after, f)
}

// Len is the number of buffered writes.
func (c *turnCommit) Len() int { return len(c.writes) }

// Run executes every buffered write in one transaction. On any error the transaction is rolled back and
// nothing is kept; the after-commit actions do not run.
func (c *turnCommit) Run(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("commit: begin: %w", err)
	}
	for i, w := range c.writes {
		if _, err := tx.Exec(w.query, w.args...); err != nil {
			tx.Rollback()
			return fmt.Errorf("commit: write %d of %d failed: %w", i+1, len(c.writes), err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	for _, f := range c.after {
		f()
	}
	return nil
}
