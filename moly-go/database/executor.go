package database

import "database/sql"

// Executor runs a write. A *sql.DB, a *sql.Tx and the turn commit buffer all satisfy it, so a repository write can
// be part of a transaction chosen by the caller.
type Executor interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}
