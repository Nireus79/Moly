package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"moly/auth"
	"moly/database"
)

func TestLocalProfileSessionAndDeletion(t *testing.T) {
	db, err := database.Init(testDBPath)
	if err != nil {
		t.Fatal(err)
	}
	conn := db.GetConnection()
	conn.Exec(`DELETE FROM users WHERE id = ?`, auth.LocalProfileID)
	srv, err := NewAPIServer(scenarioSource{llm: scriptedLLM{}}, db)
	if err != nil {
		t.Fatal(err)
	}
	as := auth.NewUserAuthServer(conn)

	local := func() map[string]any {
		rec := httptest.NewRecorder()
		as.LocalProfileHandler(rec, httptest.NewRequest(http.MethodPost, "/api/auth/local", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("local profile: %d %s", rec.Code, rec.Body.String())
		}
		var out map[string]any
		json.Unmarshal(rec.Body.Bytes(), &out)
		return out
	}

	first := local()
	token1, _ := first["token"].(string)
	if first["userId"] != auth.LocalProfileID || token1 == "" || first["local"] != true {
		t.Fatalf("unexpected answer: %v", first)
	}
	var hashNull bool
	conn.QueryRow(`SELECT password_hash IS NULL FROM users WHERE id = ?`, auth.LocalProfileID).Scan(&hashNull)
	if !hashNull {
		t.Fatal("the local profile must have no password")
	}
	var expires int64
	conn.QueryRow(`SELECT expires_at FROM sessions WHERE id = ?`, token1).Scan(&expires)
	if days := time.Until(time.Unix(expires, 0)).Hours() / 24; days < 29 || days > 31 {
		t.Fatalf("the local session should last about 30 days, lasts %.1f", days)
	}
	if id, err := extractAndValidateToken(bearer(token1), srv.database); err != nil || id != auth.LocalProfileID {
		t.Fatalf("the session token must be accepted by the API: %v %v", id, err)
	}

	second := local()
	if second["userId"] != first["userId"] || second["token"] == first["token"] {
		t.Fatal("a second call gives the same profile and a new session")
	}
	var users int
	conn.QueryRow(`SELECT COUNT(*) FROM users WHERE id = ?`, auth.LocalProfileID).Scan(&users)
	if users != 1 {
		t.Fatalf("exactly one local profile, got %d", users)
	}

	// Nobody can log in to it by typing its name.
	rec := httptest.NewRecorder()
	as.LoginHandler(rec, httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"emailOrUsername":"You","password":"anything at all"}`)))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("logging in to the passwordless profile must fail with 401, got %d %s", rec.Code, rec.Body.String())
	}

	del := func(body, token string) int {
		req := httptest.NewRequest(http.MethodDelete, "/api/user/delete", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		srv.DeleteProfileHandler(w, req)
		return w.Code
	}
	for _, body := range []string{`{}`, `{"confirm":"delete"}`, `{"confirm":"yes"}`, `{"password":"x"}`} {
		if code := del(body, token1); code != http.StatusBadRequest {
			t.Errorf("%s: want 400, got %d", body, code)
		}
	}
	conn.QueryRow(`SELECT COUNT(*) FROM users WHERE id = ?`, auth.LocalProfileID).Scan(&users)
	if users != 1 {
		t.Fatal("a refused deletion deletes nothing")
	}
	if code := del(`{"confirm":"DELETE"}`, token1); code != http.StatusOK {
		t.Fatalf("typing DELETE must delete the local profile, got %d", code)
	}
	var sessions int
	conn.QueryRow(`SELECT COUNT(*) FROM users WHERE id = ?`, auth.LocalProfileID).Scan(&users)
	conn.QueryRow(`SELECT COUNT(*) FROM sessions WHERE user_id = ?`, auth.LocalProfileID).Scan(&sessions)
	if users != 0 || sessions != 0 {
		t.Fatalf("profile and sessions must be gone: users %d sessions %d", users, sessions)
	}
	if third := local(); third["userId"] != auth.LocalProfileID {
		t.Fatal("after a deletion the next start makes a fresh local profile")
	}
	conn.Exec(`DELETE FROM users WHERE id = ?`, auth.LocalProfileID)
}

func bearer(token string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	return r
}
