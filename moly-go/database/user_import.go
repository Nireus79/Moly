package database

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
)

// ErrProfileNotEmpty is returned when an import is attempted into a profile that already has data.
var ErrProfileNotEmpty = errors.New("the profile already has data: an import restores into an empty profile (delete the profile first, or use a fresh installation)")

// ErrBadExport is returned for a file that is not a Moly export or does not fit this version.
type ErrBadExport struct{ Reason string }

func (e ErrBadExport) Error() string { return "this is not a usable Moly export: " + e.Reason }

// ImportResult says how many rows were restored.
type ImportResult struct {
	Tables map[string]int `json:"tables"`
	Rows   int            `json:"rows"`
}

func quoteIdent(name string) string { return `"` + strings.ReplaceAll(name, `"`, `""`) + `"` }

// ImportUserData restores a file made by ExportUserData into the profile userID, which must be empty. Everything is checked
// before anything is written, and the writing is one transaction, so a bad file changes nothing. Table and column names come
// from the file and are accepted only if the schema has them (and only for tables that belong to a user). The user_id of every
// restored row becomes userID. The export's own profile row is ignored: the profile keeps its identity.
func ImportUserData(conn *sql.DB, userID string, data map[string]any) (*ImportResult, error) {
	if f, _ := data["format"].(string); f != "moly-export-1" {
		return nil, ErrBadExport{"unknown format"}
	}
	tables, ok := data["tables"].(map[string]any)
	if !ok {
		return nil, ErrBadExport{"no tables"}
	}
	names, scoped, err := userScopes(conn)
	if err != nil {
		return nil, err
	}

	// The profile must be empty.
	for _, t := range names {
		cond, ok := scoped[t]
		if !ok || t == "users" || excludedFromExport[t] {
			continue
		}
		var n int
		if err := conn.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE %s`, quoteIdent(t), cond), userID).Scan(&n); err != nil {
			return nil, err
		}
		if n > 0 {
			return nil, ErrProfileNotEmpty
		}
	}

	// Check everything first.
	type plan struct {
		table string
		cols  []string
		rows  []map[string]any
	}
	var plans []plan
	keys := make([]string, 0, len(tables))
	for t := range tables {
		keys = append(keys, t)
	}
	sort.Strings(keys)
	for _, t := range keys {
		if _, ok := scoped[t]; !ok || t == "users" || excludedFromExport[t] {
			return nil, ErrBadExport{fmt.Sprintf("table %q is not a table of a user's data", t)}
		}
		known, err := columnSet(conn, t)
		if err != nil {
			return nil, err
		}
		list, ok := tables[t].([]any)
		if !ok {
			return nil, ErrBadExport{fmt.Sprintf("table %q is not a list of rows", t)}
		}
		p := plan{table: t}
		colSeen := map[string]bool{}
		for _, item := range list {
			row, ok := item.(map[string]any)
			if !ok {
				return nil, ErrBadExport{fmt.Sprintf("a row of %q is not an object", t)}
			}
			for c := range row {
				if !known[c] {
					return nil, ErrBadExport{fmt.Sprintf("table %q has no column %q in this version of Moly", t, c)}
				}
				colSeen[c] = true
			}
			p.rows = append(p.rows, row)
		}
		for c := range colSeen {
			p.cols = append(p.cols, c)
		}
		sort.Strings(p.cols)
		if len(p.rows) > 0 {
			plans = append(plans, p)
		}
	}

	tx, err := conn.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`PRAGMA defer_foreign_keys = ON`); err != nil {
		return nil, err
	}
	res := &ImportResult{Tables: map[string]int{}}
	for _, p := range plans {
		hasUser := false
		quoted := make([]string, len(p.cols))
		marks := make([]string, len(p.cols))
		for i, c := range p.cols {
			quoted[i] = quoteIdent(c)
			marks[i] = "?"
			if c == "user_id" {
				hasUser = true
			}
		}
		stmt := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s)`, quoteIdent(p.table), strings.Join(quoted, ", "), strings.Join(marks, ", "))
		for _, row := range p.rows {
			args := make([]any, len(p.cols))
			for i, c := range p.cols {
				args[i] = sqlValue(row[c])
				if hasUser && c == "user_id" {
					args[i] = userID
				}
			}
			if _, err := tx.Exec(stmt, args...); err != nil {
				return nil, fmt.Errorf("restoring %s: %w", p.table, err)
			}
			res.Tables[p.table]++
			res.Rows++
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("the import could not be completed: %w", err)
	}
	return res, nil
}

func columnSet(conn *sql.DB, table string) (map[string]bool, error) {
	rows, err := conn.Query(fmt.Sprintf(`SELECT name FROM pragma_table_info('%s')`, strings.ReplaceAll(table, "'", "''")))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var n string
		rows.Scan(&n)
		out[n] = true
	}
	return out, nil
}

// sqlValue turns a value read from JSON back into what SQLite stores: whole numbers become integers.
func sqlValue(v any) any {
	if f, ok := v.(float64); ok && f == math.Trunc(f) && math.Abs(f) < 1<<53 {
		return int64(f)
	}
	return v
}
