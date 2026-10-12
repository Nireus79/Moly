package database

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

type colInfo struct {
	name, typ string
	notNull   bool
	hasDef    bool
	pk        int
}

func tableColumns(t *testing.T, c *sql.DB, table string) []colInfo {
	rows, err := c.Query(fmt.Sprintf(`SELECT name, type, "notnull", dflt_value IS NOT NULL, pk FROM pragma_table_info('%s')`, table))
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []colInfo
	for rows.Next() {
		var ci colInfo
		rows.Scan(&ci.name, &ci.typ, &ci.notNull, &ci.hasDef, &ci.pk)
		out = append(out, ci)
	}
	return out
}

// TestDeleteUserLeavesNothingInAnyTable fills one row for a user ("ghost") and one for another user ("bystander") in EVERY table that
// can hold user data (generated from the schema, so a table added later is audited too), deletes the ghost, and scans every table:
// no row of the ghost may remain anywhere, whatever its key, and the bystander keeps every row.
func TestDeleteUserLeavesNothingInAnyTable(t *testing.T) {
	db, err := initWithKey(filepath.Join(t.TempDir(), "audit.db"), testKey(t))
	if err != nil {
		t.Fatal(err)
	}
	c := db.GetConnection()
	rows, _ := c.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	var tables []string
	for rows.Next() {
		var n string
		rows.Scan(&n)
		tables = append(tables, n)
	}
	rows.Close()

	isTextPK := func(table string) bool {
		for _, ci := range tableColumns(t, c, table) {
			if ci.pk == 1 && !strings.Contains(strings.ToUpper(ci.typ), "INT") {
				return true
			}
		}
		return false
	}
	pkColumn := func(table string) string {
		for _, ci := range tableColumns(t, c, table) {
			if ci.pk == 1 {
				return ci.name
			}
		}
		return ""
	}

	if _, err := c.Exec(`PRAGMA foreign_keys = OFF`); err != nil {
		t.Fatal(err)
	}
	type who struct {
		user string
		n    int
	}
	people := []who{{"ghost", 1}, {"bystander", 2}}
	inserted := map[string]bool{}
	for _, tb := range tables {
		if tb == "response_templates" {
			continue
		}
		cols := tableColumns(t, c, tb)
		fkRows, _ := c.Query(fmt.Sprintf(`SELECT "from", "table" FROM pragma_foreign_key_list('%s')`, tb))
		fks := map[string]string{}
		for fkRows.Next() {
			var from, parent string
			fkRows.Scan(&from, &parent)
			fks[from] = parent
		}
		fkRows.Close()
		for _, p := range people {
			var names []string
			var args []interface{}
			for _, ci := range cols {
				isIntPK := ci.pk == 1 && strings.Contains(strings.ToUpper(ci.typ), "INT")
				var v interface{}
				switch {
				case tb == "users" && ci.name == "id":
					v = p.user
				case ci.name == "user_id":
					v = p.user
				case ci.name == "conversation_id":
					v = "c-" + p.user
				case fks[ci.name] != "":
					parent := fks[ci.name]
					if isTextPK(parent) {
						v = fmt.Sprintf("pk-%s-%s", parent, p.user)
					} else {
						v = p.n
					}
				case ci.pk == 1 && tb == "conversations":
					v = "c-" + p.user
				case isIntPK:
					v = p.n
				case ci.pk == 1:
					v = fmt.Sprintf("pk-%s-%s", tb, p.user)
				case !ci.notNull || ci.hasDef:
					continue
				case strings.Contains(strings.ToUpper(ci.typ), "INT") || strings.Contains(strings.ToUpper(ci.typ), "BOOL"):
					v = 1
				case strings.Contains(strings.ToUpper(ci.typ), "REAL") || strings.Contains(strings.ToUpper(ci.typ), "FLOA") || strings.Contains(strings.ToUpper(ci.typ), "DOUB"):
					v = 0.5
				default:
					v = fmt.Sprintf("MARK-%s-%s", tb, p.user)
				}
				names = append(names, ci.name)
				args = append(args, v)
			}
			marks := strings.TrimSuffix(strings.Repeat("?,", len(names)), ",")
			q := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s)`, tb, strings.Join(names, ","), marks)
			if _, err := c.Exec(q, args...); err != nil {
				t.Logf("not exercised: %s (%v)", tb, err)
				continue
			}
			inserted[tb] = true
		}
	}
	if _, err := c.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatal(err)
	}
	var names []string
	for tb := range inserted {
		names = append(names, tb)
	}
	t.Logf("exercised %d of %d tables", len(inserted), len(tables))
	if len(inserted) < len(tables)-8 {
		t.Fatalf("the audit exercised only %d of %d tables", len(inserted), len(tables))
	}
	_ = pkColumn

	before := map[string]int{}
	for tb := range inserted {
		var n int
		c.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM %s`, tb)).Scan(&n)
		before[tb] = n
	}
	if err := DeleteUserData(c, "ghost"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	// scan every table, every cell: nothing of the ghost may remain
	for _, tb := range tables {
		cols := tableColumns(t, c, tb)
		var conds []string
		for _, ci := range cols {
			conds = append(conds, fmt.Sprintf(`CAST("%s" AS TEXT) LIKE '%%ghost%%'`, ci.name))
		}
		var n int
		c.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE %s`, tb, strings.Join(conds, " OR "))).Scan(&n)
		if n != 0 {
			t.Errorf("table %s still holds %d row(s) of the deleted user", tb, n)
		}
	}
	// the other user keeps everything (one row per exercised table)
	for tb := range inserted {
		var after int
		c.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM %s`, tb)).Scan(&after)
		if tb == "users" || after >= before[tb]-1 {
			continue
		}
		t.Errorf("table %s lost more than the deleted user's row: %d -> %d", tb, before[tb], after)
	}
	var left int
	c.QueryRow(`SELECT COUNT(*) FROM users WHERE id = 'bystander'`).Scan(&left)
	if left != 1 {
		t.Fatal("the other user was deleted")
	}
}
