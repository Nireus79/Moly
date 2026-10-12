package database

import (
	"database/sql"
	"fmt"
	"log"
)

// DeleteUserData removes a user and every row that belongs to them, in one transaction.
// Foreign-key checks are deferred to commit, so the delete order does not matter; if any
// row would be left pointing at a deleted parent, the whole deletion is rolled back.
func DeleteUserData(conn *sql.DB, userID string) error {
	tables, err := tablesWithColumn(conn, "user_id")
	if err != nil {
		return err
	}
	convTables, err := tablesWithColumn(conn, "conversation_id")
	if err != nil {
		return err
	}

	tx, err := conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to start profile deletion: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`PRAGMA defer_foreign_keys = ON`); err != nil {
		return fmt.Errorf("failed to defer foreign keys: %w", err)
	}

	// Rows keyed only by conversation must go before the conversations themselves.
	for _, t := range convTables {
		if contains(tables, t) {
			continue
		}
		q := fmt.Sprintf(`DELETE FROM %s WHERE conversation_id IN (SELECT id FROM conversations WHERE user_id = ?)`, t)
		if _, err := tx.Exec(q, userID); err != nil {
			return fmt.Errorf("failed to delete from %s: %w", t, err)
		}
	}

	// Rows with no user or conversation column that hang from a row about to be deleted (the answers to a user's clarification
	// questions, the links between a user's attributes and sentences) go first, found through the foreign keys.
	if err := deleteChildrenOfUserRows(tx, tables, convTables, userID); err != nil {
		return err
	}

	for _, t := range tables {
		if t == "users" {
			continue
		}
		if _, err := tx.Exec(fmt.Sprintf(`DELETE FROM %s WHERE user_id = ?`, t), userID); err != nil {
			return fmt.Errorf("failed to delete from %s: %w", t, err)
		}
	}

	if _, err := tx.Exec(`DELETE FROM users WHERE id = ?`, userID); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("profile deletion could not be completed: %w", err)
	}
	// Deleted rows must not linger in free pages or in the write-ahead log.
	ReclaimDeletedSpace(conn)
	return nil
}

// ReclaimDeletedSpace folds the write-ahead log into the database and rebuilds the file, so that the pages of deleted rows are
// gone from disk. Failures are logged and do not undo the deletion.
func ReclaimDeletedSpace(conn *sql.DB) {
	if _, err := conn.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		log.Printf("[Database] Warning: could not truncate the write-ahead log after a deletion: %v", err)
	}
	if _, err := conn.Exec(`VACUUM`); err != nil {
		log.Printf("[Database] Warning: could not compact the database after a deletion: %v", err)
	}
}

// deleteChildrenOfUserRows deletes the rows of tables that have neither user_id nor conversation_id but reference a table that does,
// through a foreign key, when the parent row belongs to the user. It repeats until nothing more is found, so chains are followed.
func deleteChildrenOfUserRows(tx *sql.Tx, userTables, convTables []string, userID string) error {
	scoped := map[string]string{} // table -> condition selecting the user's rows
	for _, t := range userTables {
		scoped[t] = "user_id = ?"
	}
	for _, t := range convTables {
		if _, ok := scoped[t]; !ok {
			scoped[t] = "conversation_id IN (SELECT id FROM conversations WHERE user_id = ?)"
		}
	}
	all, err := allTables(tx)
	if err != nil {
		return err
	}
	// Discover the chain top-down (parents first), then delete bottom-up, so that every deletion still finds its parent rows.
	type step struct{ query string }
	var steps []step
	for pass := 0; pass < 5; pass++ {
		found := false
		for _, child := range all {
			if _, ok := scoped[child]; ok || child == "users" {
				continue
			}
			rows, err := tx.Query(fmt.Sprintf(`SELECT "table", "from", "to" FROM pragma_foreign_key_list('%s')`, child))
			if err != nil {
				return fmt.Errorf("failed to read keys of %s: %w", child, err)
			}
			type fk struct{ parent, from, to string }
			var fks []fk
			for rows.Next() {
				var f fk
				rows.Scan(&f.parent, &f.from, &f.to)
				fks = append(fks, f)
			}
			rows.Close()
			for _, f := range fks {
				cond, ok := scoped[f.parent]
				if !ok || f.to == "" {
					continue
				}
				inParent := fmt.Sprintf("%s IN (SELECT %s FROM %s WHERE %s)", f.from, f.to, f.parent, cond)
				steps = append(steps, step{fmt.Sprintf(`DELETE FROM %s WHERE %s`, child, inParent)})
				if _, seen := scoped[child]; !seen {
					scoped[child] = inParent
					found = true
				}
			}
		}
		if !found {
			break
		}
	}
	for i := len(steps) - 1; i >= 0; i-- {
		if _, err := tx.Exec(steps[i].query, userID); err != nil {
			return fmt.Errorf("failed to delete dependent rows: %w", err)
		}
	}
	return nil
}

func allTables(tx *sql.Tx) ([]string, error) {
	rows, err := tx.Query(`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var n string
		rows.Scan(&n)
		out = append(out, n)
	}
	return out, nil
}

func tablesWithColumn(conn *sql.DB, column string) ([]string, error) {
	rows, err := conn.Query(`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			rows.Close()
			return nil, err
		}
		names = append(names, n)
	}
	rows.Close()

	var out []string
	for _, n := range names {
		cols, err := conn.Query(fmt.Sprintf(`SELECT name FROM pragma_table_info('%s')`, n))
		if err != nil {
			return nil, fmt.Errorf("failed to read columns of %s: %w", n, err)
		}
		for cols.Next() {
			var c string
			cols.Scan(&c)
			if c == column {
				out = append(out, n)
				break
			}
		}
		cols.Close()
	}
	return out, nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
