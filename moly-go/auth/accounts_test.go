package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"moly/database"
)

func accountServer(t *testing.T) *UserAuthServer {
	t.Helper()
	db, err := database.OpenIndependent(filepath.Join(t.TempDir(), "accounts.db"), make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	return NewUserAuthServer(db.GetConnection())
}

func post(t *testing.T, h http.HandlerFunc, body any) (int, map[string]any) {
	t.Helper()
	raw, _ := json.Marshal(body)
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(raw)))
	var out map[string]any
	json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func TestAccountWithoutEmailRegistersLogsInAndNamesAreUnique(t *testing.T) {
	uas := accountServer(t)
	code, out := post(t, uas.RegisterHandler, map[string]string{"name": "Anna", "password": "longenough1"})
	if code != http.StatusCreated || out["token"] == "" || out["recoveryCode"] == "" {
		t.Fatalf("a name and a password are enough: %d %v", code, out)
	}
	// a second profile without an address must not collide on the empty email
	if code, out = post(t, uas.RegisterHandler, map[string]string{"name": "Ben", "password": "longenough2"}); code != http.StatusCreated {
		t.Fatalf("second profile without email: %d %v", code, out)
	}
	if code, _ = post(t, uas.RegisterHandler, map[string]string{"name": "anna", "password": "longenough3"}); code != http.StatusConflict {
		t.Fatalf("the same name in other case must be refused, got %d", code)
	}
	if code, out = post(t, uas.LoginHandler, map[string]string{"emailOrUsername": "ANNA", "password": "longenough1"}); code != http.StatusOK || out["token"] == "" {
		t.Fatalf("login by name: %d %v", code, out)
	}
	if code, _ = post(t, uas.LoginHandler, map[string]string{"emailOrUsername": "Anna", "password": "wrongwrong"}); code != http.StatusUnauthorized {
		t.Fatalf("a wrong password must fail, got %d", code)
	}
}

func TestRecoveryCodeResetsThePasswordOnceAndMakesANewCode(t *testing.T) {
	uas := accountServer(t)
	_, reg := post(t, uas.RegisterHandler, map[string]string{"name": "Anna", "password": "longenough1"})
	codeText := reg["recoveryCode"].(string)
	if len(codeText) != 19 || strings.Count(codeText, "-") != 3 {
		t.Fatalf("recovery code looks wrong: %q", codeText)
	}
	oldToken := reg["token"].(string)

	if code, _ := post(t, uas.ResetHandler, map[string]string{"emailOrUsername": "Anna", "recoveryCode": "AAAA-AAAA-AAAA-AAAA", "newPassword": "brandnew123"}); code != http.StatusUnauthorized {
		t.Fatalf("a wrong code must be refused, got %d", code)
	}
	if code, _ := post(t, uas.ResetHandler, map[string]string{"emailOrUsername": "Nobody", "recoveryCode": codeText, "newPassword": "brandnew123"}); code != http.StatusUnauthorized {
		t.Fatalf("an unknown name must get the same refusal, got %d", code)
	}
	if code, _ := post(t, uas.ResetHandler, map[string]string{"emailOrUsername": "Anna", "recoveryCode": codeText, "newPassword": "short"}); code != http.StatusBadRequest {
		t.Fatalf("a short password must be refused, got %d", code)
	}
	// typed in lower case with spaces: the same code
	typed := strings.ToLower(strings.ReplaceAll(codeText, "-", " "))
	code, out := post(t, uas.ResetHandler, map[string]string{"emailOrUsername": "anna", "recoveryCode": typed, "newPassword": "brandnew123"})
	if code != http.StatusOK || out["recoveryCode"] == "" || out["recoveryCode"] == codeText {
		t.Fatalf("reset: %d %v", code, out)
	}
	if code, _ = post(t, uas.LoginHandler, map[string]string{"emailOrUsername": "Anna", "password": "longenough1"}); code != http.StatusUnauthorized {
		t.Fatalf("the old password must not work, got %d", code)
	}
	if code, _ = post(t, uas.LoginHandler, map[string]string{"emailOrUsername": "Anna", "password": "brandnew123"}); code != http.StatusOK {
		t.Fatalf("the new password must work, got %d", code)
	}
	if code, _ = post(t, uas.ResetHandler, map[string]string{"emailOrUsername": "Anna", "recoveryCode": codeText, "newPassword": "another12345"}); code != http.StatusUnauthorized {
		t.Fatalf("a used code must not work again, got %d", code)
	}
	var n int
	uas.db.QueryRow("SELECT COUNT(*) FROM sessions WHERE id = ?", oldToken).Scan(&n)
	if n != 0 {
		t.Fatal("sessions from before the reset must end")
	}
}

func TestLocalProfileCannotBeResetOrListed(t *testing.T) {
	uas := accountServer(t)
	rec := httptest.NewRecorder()
	uas.LocalProfileHandler(rec, httptest.NewRequest(http.MethodPost, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("local profile: %d", rec.Code)
	}
	if code, _ := post(t, uas.ResetHandler, map[string]string{"emailOrUsername": "You", "recoveryCode": "AAAA-AAAA-AAAA-AAAA", "newPassword": "brandnew123"}); code != http.StatusUnauthorized {
		t.Fatalf("the local profile has no password to reset, got %d", code)
	}
	post(t, uas.RegisterHandler, map[string]string{"name": "Anna", "password": "longenough1"})
	rec = httptest.NewRecorder()
	uas.ProfilesHandler(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	var out struct{ Profiles []string }
	json.Unmarshal(rec.Body.Bytes(), &out)
	if len(out.Profiles) != 1 || out.Profiles[0] != "Anna" {
		t.Fatalf("only accounts with a password are listed, got %v", out.Profiles)
	}
}

// A computer that used Moly before 0.1.2 has its conversations in the passwordless local profile. The first profile with a password takes it
// over (the data stays with its owner and from then on needs the password); nobody gets in through the local door afterwards.
func TestFirstProfileClaimsTheLocalProfileAndTheLocalDoorCloses(t *testing.T) {
	uas := accountServer(t)
	if code, out := post(t, uas.LocalProfileHandler, map[string]string{}); code != http.StatusOK || out["userId"] != LocalProfileID {
		t.Fatalf("before any password the local door opens: %d %v", code, out)
	}
	code, out := post(t, uas.RegisterHandler, map[string]string{"name": "Anna", "password": "longenough1"})
	if code != http.StatusCreated || out["userId"] != LocalProfileID {
		t.Fatalf("the first profile takes the local data over: %d %v", code, out)
	}
	if code, _ = post(t, uas.LocalProfileHandler, map[string]string{}); code != http.StatusForbidden {
		t.Fatalf("the local door must close once a profile has a password, got %d", code)
	}
	if code, out = post(t, uas.LoginHandler, map[string]string{"emailOrUsername": "Anna", "password": "longenough1"}); code != http.StatusOK || out["userId"] != LocalProfileID {
		t.Fatalf("Anna signs in to the old data: %d %v", code, out)
	}
	// the second person gets a profile of their own
	code, out = post(t, uas.RegisterHandler, map[string]string{"name": "Ben", "password": "longenough2"})
	if code != http.StatusCreated || out["userId"] == LocalProfileID || out["userId"] == "" {
		t.Fatalf("the second profile must be new: %d %v", code, out)
	}
}

func TestFreshComputerFirstProfileIsNewAndClosesTheDoor(t *testing.T) {
	uas := accountServer(t)
	code, out := post(t, uas.RegisterHandler, map[string]string{"name": "Anna", "password": "longenough1"})
	if code != http.StatusCreated || out["userId"] == LocalProfileID {
		t.Fatalf("nothing to claim: %d %v", code, out)
	}
	if code, _ = post(t, uas.LocalProfileHandler, map[string]string{}); code != http.StatusForbidden {
		t.Fatalf("local door must be closed, got %d", code)
	}
}
