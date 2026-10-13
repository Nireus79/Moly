package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// excludedFromExport are tables that hold credentials or session tokens, not the person's own data.
var excludedFromExport = map[string]bool{"sessions": true, "login_codes": true}

// ExportUserData collects everything stored about a user, as plain values ready to be written as JSON: the rows of every table
// that has a user_id, of every table keyed by conversation, and of the tables that hang from those through foreign keys. Like
// the deletion, it finds the tables from the schema, so a table added later is included without a change here. The password
// hash and the session tokens are left out.
func ExportUserData(conn *sql.DB, userID string) (map[string]any, error) {
	userTables, err := tablesWithColumn(conn, "user_id")
	if err != nil {
		return nil, err
	}
	convTables, err := tablesWithColumn(conn, "conversation_id")
	if err != nil {
		return nil, err
	}
	scoped := map[string]string{} // table -> condition with one "?" for the user id
	for _, t := range userTables {
		scoped[t] = "user_id = ?"
	}
	for _, t := range convTables {
		if _, ok := scoped[t]; !ok {
			scoped[t] = "conversation_id IN (SELECT id FROM conversations WHERE user_id = ?)"
		}
	}
	// Tables with neither column that point at a scoped table (answers to a user's questions, links between attributes).
	names, err := listTables(conn)
	if err != nil {
		return nil, err
	}
	for pass := 0; pass < 5; pass++ {
		found := false
		for _, child := range names {
			if _, ok := scoped[child]; ok || child == "users" {
				continue
			}
			rows, err := conn.Query(fmt.Sprintf(`SELECT "table", "from", "to" FROM pragma_foreign_key_list('%s')`, child))
			if err != nil {
				return nil, err
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
				if cond, ok := scoped[f.parent]; ok && f.to != "" {
					scoped[child] = fmt.Sprintf("%s IN (SELECT %s FROM %s WHERE %s)", f.from, f.to, f.parent, cond)
					found = true
					break
				}
			}
		}
		if !found {
			break
		}
	}

	out := map[string]any{}
	user, err := queryRows(conn, `SELECT id, name, email, created_at, last_active FROM users WHERE id = ?`, userID)
	if err != nil {
		return nil, err
	}
	if len(user) == 0 {
		return nil, sql.ErrNoRows
	}
	tables := map[string]any{}
	for _, t := range names {
		cond, ok := scoped[t]
		if !ok || t == "users" || excludedFromExport[t] {
			continue
		}
		rows, err := queryRows(conn, fmt.Sprintf(`SELECT * FROM %s WHERE %s`, t, cond), userID)
		if err != nil {
			return nil, fmt.Errorf("export of %s: %w", t, err)
		}
		if len(rows) > 0 {
			tables[t] = rows
		}
	}
	out["format"] = "moly-export-1"
	out["exportedAt"] = time.Now().UTC().Format(time.RFC3339)
	out["profile"] = user[0]
	out["tables"] = tables
	return out, nil
}

func listTables(conn *sql.DB) ([]string, error) {
	rows, err := conn.Query(`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		return nil, err
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

func queryRows(conn *sql.DB, query string, args ...any) ([]map[string]any, error) {
	rows, err := conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	var out []map[string]any
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := map[string]any{}
		for i, c := range cols {
			if strings.EqualFold(c, "password_hash") {
				continue
			}
			if b, ok := vals[i].([]byte); ok {
				row[c] = string(b)
			} else {
				row[c] = vals[i]
			}
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
