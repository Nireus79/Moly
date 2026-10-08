package database

import (
	"database/sql"
	"fmt"
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
	return nil
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
