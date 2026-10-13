package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func TestScenarioExportHoldsMyDataAndOnlyMine(t *testing.T) {
	srv, db, userID, token := scenarioServer(t, scriptedLLM{
		gaps: func() (string, error) { return `[]`, nil },
	})
	conn := db.GetConnection()
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct horse"), bcrypt.MinCost)
	conn.Exec(`UPDATE users SET password_hash = ?, email = ? WHERE id = ?`, string(hash), "export-me@example.invalid", userID)
	convID := startNamedGoal(t, srv, token)
	sendMessage(t, srv, token, convID, "Her name is Christine and she is my sister.")
	time.Sleep(500 * time.Millisecond)

	otherID := "bystander_export_" + time.Now().Format("150405.000000")
	otherToken := "tok_" + otherID
	harnessUser(t, db, otherID, otherToken)
	sendMessage(t, srv, otherToken, "", "A secret sentence of somebody else, zebra-quartz-nine.")

	get := func(tok string) (*httptest.ResponseRecorder, map[string]any) {
		req := httptest.NewRequest(http.MethodGet, "/api/user/export", nil)
		if tok != "" {
			req.Header.Set("Authorization", "Bearer "+tok)
		}
		rec := httptest.NewRecorder()
		srv.ExportProfileHandler(rec, req)
		var out map[string]any
		json.Unmarshal(rec.Body.Bytes(), &out)
		return rec, out
	}
	if rec, _ := get(""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("an export needs a session, got %d", rec.Code)
	}
	rec, out := get(token)
	if rec.Code != http.StatusOK {
		t.Fatalf("export failed: %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("Content-Disposition"), "moly-export-") {
		t.Fatalf("the export is a file download: %q", rec.Header().Get("Content-Disposition"))
	}
	body := rec.Body.String()
	for _, want := range []string{"Christine", convID, "chat_messages", "moly-export-1"} {
		if !strings.Contains(body, want) {
			t.Errorf("the export lacks %q", want)
		}
	}
	for _, forbidden := range []string{"zebra-quartz-nine", otherID, string(hash), "password_hash"} {
		if strings.Contains(body, forbidden) {
			t.Errorf("the export must not contain %q", forbidden)
		}
	}
	tables, _ := out["tables"].(map[string]any)
	if _, has := tables["sessions"]; has {
		t.Error("session tokens are not part of the export")
	}
	if len(tables) < 2 {
		t.Errorf("expected several tables, got %d", len(tables))
	}
	if p, _ := out["profile"].(map[string]any); p["id"] != userID {
		t.Errorf("profile = %v", out["profile"])
	}
}
