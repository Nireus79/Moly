package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"moly/database"
)

func importInto(t *testing.T, srv *APIServer, token string, body any) (int, string) {
	t.Helper()
	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(body)
	req := httptest.NewRequest(http.MethodPost, "/api/user/import", &buf)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	srv.ImportProfileHandler(rec, req)
	return rec.Code, rec.Body.String()
}

func TestScenarioImportRestoresAnExportIntoAnEmptyProfile(t *testing.T) {
	srv, db, userID, token := scenarioServer(t, scriptedLLM{gaps: func() (string, error) { return `[]`, nil }})
	conn := db.GetConnection()
	convID := startNamedGoal(t, srv, token)
	sendMessage(t, srv, token, convID, "Her name is Christine and she is my sister.")
	time.Sleep(500 * time.Millisecond)
	// what a person is, and of whom, travels with the export too
	repo := database.NewContactRepository(db)
	if c, _ := repo.GetByName(userID, "Christine"); c != nil {
		if err := repo.AddRelation(userID, c.ID, "sister", 0, false); err != nil {
			t.Fatal(err)
		}
	} else {
		t.Fatal("setup: Christine is a person of the user")
	}

	var before int
	conn.QueryRow(`SELECT COUNT(*) FROM chat_messages WHERE user_id = ?`, userID).Scan(&before)
	if before == 0 {
		t.Fatal("setup: the user has messages")
	}
	exported, err := database.ExportUserData(conn, userID)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(exported) // as a file: numbers come back as floats
	var file map[string]any
	json.Unmarshal(raw, &file)

	// The person loses everything (a reinstall), then restores into a fresh profile.
	if err := database.DeleteUserData(conn, userID); err != nil {
		t.Fatal(err)
	}
	newID := fmt.Sprintf("restored_%d", time.Now().UnixNano())
	newToken := "tok_" + newID
	harnessUser(t, db, newID, newToken)

	code, body := importInto(t, srv, newToken, file)
	if code != http.StatusOK {
		t.Fatalf("import failed: %d %s", code, body)
	}
	var after int
	conn.QueryRow(`SELECT COUNT(*) FROM chat_messages WHERE user_id = ?`, newID).Scan(&after)
	if after != before {
		t.Fatalf("messages restored: %d, were %d", after, before)
	}
	var convs, withWords int
	conn.QueryRow(`SELECT COUNT(*) FROM conversations WHERE user_id = ?`, newID).Scan(&convs)
	conn.QueryRow(`SELECT COUNT(*) FROM chat_messages WHERE user_id = ? AND content LIKE '%Christine%'`, newID).Scan(&withWords)
	if convs == 0 || withWords == 0 {
		t.Fatalf("the conversation and the user's words must be back: conversations %d, messages with the name %d", convs, withWords)
	}
	var restoredRelations int
	conn.QueryRow(`SELECT COUNT(*) FROM contact_relations WHERE user_id = ?`, newID).Scan(&restoredRelations)
	if restoredRelations != 1 {
		t.Fatalf("the relation is restored with its person, got %d", restoredRelations)
	}
	var strayOwner int
	conn.QueryRow(`SELECT COUNT(*) FROM chat_messages WHERE user_id = ?`, userID).Scan(&strayOwner)
	if strayOwner != 0 {
		t.Fatal("no row may keep the old owner")
	}

	// A second import is refused and changes nothing.
	if code, _ := importInto(t, srv, newToken, file); code != http.StatusConflict {
		t.Fatalf("a profile with data must refuse an import with 409, got %d", code)
	}
	var again int
	conn.QueryRow(`SELECT COUNT(*) FROM chat_messages WHERE user_id = ?`, newID).Scan(&again)
	if again != before {
		t.Fatalf("a refused import must change nothing: %d messages, were %d", again, before)
	}
	// And a request without a session is refused.
	if code, _ := importInto(t, srv, "", file); code != http.StatusUnauthorized {
		t.Fatalf("import needs a session, got %d", code)
	}
}

func TestScenarioImportRefusesBadFilesAndChangesNothing(t *testing.T) {
	srv, db, _, _ := scenarioServer(t, scriptedLLM{gaps: func() (string, error) { return `[]`, nil }})
	conn := db.GetConnection()
	fresh := func() (string, string) {
		id := fmt.Sprintf("fresh_%d", time.Now().UnixNano())
		harnessUser(t, db, id, "tok_"+id)
		return id, "tok_" + id
	}
	msgRow := func(id string) map[string]any {
		return map[string]any{"id": float64(time.Now().UnixNano() % 1e9), "user_id": "someone-else", "conversation_id": "c-" + id, "role": "user", "content": "hello", "created_at": float64(time.Now().Unix())}
	}
	cases := map[string]map[string]any{
		"wrong format":              {"format": "something-else", "tables": map[string]any{}},
		"no tables":                 {"format": "moly-export-1"},
		"the users table":           {"format": "moly-export-1", "tables": map[string]any{"users": []any{map[string]any{"id": "x"}}}},
		"the sessions table":        {"format": "moly-export-1", "tables": map[string]any{"sessions": []any{map[string]any{"id": "x"}}}},
		"a table that is not":       {"format": "moly-export-1", "tables": map[string]any{"nothing_here": []any{map[string]any{"a": 1}}}},
		"a table name that is code": {"format": "moly-export-1", "tables": map[string]any{`chat_messages; DROP TABLE users; --`: []any{map[string]any{"a": 1}}}},
	}
	for name, file := range cases {
		id, tok := fresh()
		if code, body := importInto(t, srv, tok, file); code != http.StatusBadRequest {
			t.Errorf("%s: want 400, got %d %s", name, code, body)
		}
		var n int
		conn.QueryRow(`SELECT COUNT(*) FROM chat_messages WHERE user_id = ?`, id).Scan(&n)
		if n != 0 {
			t.Errorf("%s: nothing may be written", name)
		}
	}
	// One good table and one with an unknown column: the good one must not be written either.
	id, tok := fresh()
	file := map[string]any{"format": "moly-export-1", "tables": map[string]any{
		"chat_messages": []any{msgRow(id)},
		"contacts":      []any{map[string]any{"id": 1, "no_such_column": "x"}},
	}}
	if code, body := importInto(t, srv, tok, file); code != http.StatusBadRequest || !strings.Contains(body, "no_such_column") {
		t.Fatalf("an unknown column must be refused by name: %d %s", code, body)
	}
	var n int
	conn.QueryRow(`SELECT COUNT(*) FROM chat_messages WHERE user_id = ?`, id).Scan(&n)
	if n != 0 {
		t.Fatal("a file that is refused must write nothing, not even its good tables")
	}
	var users int
	conn.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&users)
	if users == 0 {
		t.Fatal("the users table must be intact")
	}
}
