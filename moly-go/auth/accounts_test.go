package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func postAuth(t *testing.T, h http.HandlerFunc, token string, body any) (int, map[string]any) {
	t.Helper()
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(raw))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h(rec, req)
	var out map[string]any
	json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func TestFiveWrongPasswordsLockTheNameForAWhileAndARightOneClearsTheCount(t *testing.T) {
	uas := accountServer(t)
	clock := time.Now()
	uas.guard.now = func() time.Time { return clock }
	post(t, uas.RegisterHandler, map[string]string{"name": "Anna", "password": "longenough1"})
	bad := map[string]string{"emailOrUsername": "Anna", "password": "wrongwrong"}
	for i := 0; i < 4; i++ {
		if code, _ := post(t, uas.LoginHandler, bad); code != http.StatusUnauthorized {
			t.Fatalf("try %d: %d", i, code)
		}
	}
	// a right password before the fifth wrong one clears the count
	if code, _ := post(t, uas.LoginHandler, map[string]string{"emailOrUsername": "Anna", "password": "longenough1"}); code != http.StatusOK {
		t.Fatalf("right password: %d", code)
	}
	for i := 0; i < 5; i++ {
		post(t, uas.LoginHandler, bad)
	}
	code, out := post(t, uas.LoginHandler, map[string]string{"emailOrUsername": "ANNA", "password": "longenough1"})
	if code != http.StatusTooManyRequests || !strings.Contains(out["error"].(string), "Wait 5 minutes") {
		t.Fatalf("locked, even for the right password and another spelling of the name: %d %v", code, out)
	}
	clock = clock.Add(6 * time.Minute)
	if code, _ = post(t, uas.LoginHandler, map[string]string{"emailOrUsername": "Anna", "password": "longenough1"}); code != http.StatusOK {
		t.Fatalf("after the wait the right password works: %d", code)
	}
	// the second round of wrong tries waits twice as long
	for i := 0; i < 5; i++ {
		post(t, uas.LoginHandler, bad)
	}
	clock = clock.Add(6 * time.Minute)
	for i := 0; i < 5; i++ {
		post(t, uas.LoginHandler, bad)
	}
	if code, out = post(t, uas.LoginHandler, bad); code != http.StatusTooManyRequests || !strings.Contains(out["error"].(string), "Wait 10 minutes") {
		t.Fatalf("second round: %d %v", code, out)
	}
}

func TestAnUnknownNameIsLockedTheSameWay(t *testing.T) {
	uas := accountServer(t)
	for i := 0; i < 5; i++ {
		post(t, uas.LoginHandler, map[string]string{"emailOrUsername": "nobody", "password": "wrongwrong"})
	}
	if code, _ := post(t, uas.LoginHandler, map[string]string{"emailOrUsername": "nobody", "password": "wrongwrong"}); code != http.StatusTooManyRequests {
		t.Fatalf("a name that does not exist must answer like one that does, got %d", code)
	}
}

func TestWrongRecoveryCodesAreLockedToo(t *testing.T) {
	uas := accountServer(t)
	post(t, uas.RegisterHandler, map[string]string{"name": "Anna", "password": "longenough1"})
	for i := 0; i < 5; i++ {
		post(t, uas.ResetHandler, map[string]string{"emailOrUsername": "Anna", "recoveryCode": "AAAA-AAAA-AAAA-AAAA", "newPassword": "brandnew123"})
	}
	if code, _ := post(t, uas.ResetHandler, map[string]string{"emailOrUsername": "Anna", "recoveryCode": "AAAA-AAAA-AAAA-AAAA", "newPassword": "brandnew123"}); code != http.StatusTooManyRequests {
		t.Fatalf("got %d", code)
	}
}

func TestChangingThePassword(t *testing.T) {
	uas := accountServer(t)
	_, reg := post(t, uas.RegisterHandler, map[string]string{"name": "Anna", "password": "longenough1"})
	here := reg["token"].(string)
	_, other := post(t, uas.LoginHandler, map[string]string{"emailOrUsername": "Anna", "password": "longenough1"})
	elsewhere := other["token"].(string)

	if code, _ := postAuth(t, uas.ChangePasswordHandler, "", map[string]string{"currentPassword": "longenough1", "newPassword": "brandnew123"}); code != http.StatusUnauthorized {
		t.Fatalf("signed out must be refused, got %d", code)
	}
	if code, _ := postAuth(t, uas.ChangePasswordHandler, here, map[string]string{"currentPassword": "not-it-not-it", "newPassword": "brandnew123"}); code != http.StatusUnauthorized {
		t.Fatalf("a wrong current password must be refused, got %d", code)
	}
	if code, _ := postAuth(t, uas.ChangePasswordHandler, here, map[string]string{"currentPassword": "longenough1", "newPassword": "short"}); code != http.StatusBadRequest {
		t.Fatalf("a short new password must be refused, got %d", code)
	}
	if code, out := postAuth(t, uas.ChangePasswordHandler, here, map[string]string{"currentPassword": "longenough1", "newPassword": "brandnew123"}); code != http.StatusOK || out["success"] != true {
		t.Fatalf("change: %d %v", code, out)
	}
	if code, _ := post(t, uas.LoginHandler, map[string]string{"emailOrUsername": "Anna", "password": "longenough1"}); code != http.StatusUnauthorized {
		t.Fatalf("the old password must stop working, got %d", code)
	}
	if code, _ := post(t, uas.LoginHandler, map[string]string{"emailOrUsername": "Anna", "password": "brandnew123"}); code != http.StatusOK {
		t.Fatalf("the new password must work, got %d", code)
	}
	var n int
	uas.db.QueryRow("SELECT COUNT(*) FROM sessions WHERE id = ?", here).Scan(&n)
	if n != 1 {
		t.Fatal("the session that changed the password stays")
	}
	uas.db.QueryRow("SELECT COUNT(*) FROM sessions WHERE id = ?", elsewhere).Scan(&n)
	if n != 0 {
		t.Fatal("every other session of the profile must end")
	}
}

func TestChangingThePasswordIsLockedAfterWrongTries(t *testing.T) {
	uas := accountServer(t)
	_, reg := post(t, uas.RegisterHandler, map[string]string{"name": "Anna", "password": "longenough1"})
	here := reg["token"].(string)
	for i := 0; i < 5; i++ {
		postAuth(t, uas.ChangePasswordHandler, here, map[string]string{"currentPassword": "guessguess", "newPassword": "brandnew123"})
	}
	if code, _ := postAuth(t, uas.ChangePasswordHandler, here, map[string]string{"currentPassword": "longenough1", "newPassword": "brandnew123"}); code != http.StatusTooManyRequests {
		t.Fatalf("got %d", code)
	}
}

func TestTheLocalProfileOfAPhoneHasNoPasswordToChange(t *testing.T) {
	uas := accountServer(t)
	_, out := post(t, uas.LocalProfileHandler, map[string]string{})
	if code, _ := postAuth(t, uas.ChangePasswordHandler, out["token"].(string), map[string]string{"currentPassword": "x", "newPassword": "brandnew123"}); code != http.StatusBadRequest {
		t.Fatalf("got %d", code)
	}
}
