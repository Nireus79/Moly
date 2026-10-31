package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// UserAuthServer handles user registration and login
type UserAuthServer struct {
	db    *sql.DB
	guard *attemptGuard
}

// NewUserAuthServer creates a new user auth server
func NewUserAuthServer(db *sql.DB) *UserAuthServer {
	return &UserAuthServer{db: db, guard: newAttemptGuard()}
}

// RegisterRequest for user registration
type RegisterRequest struct {
	Name     string `json:"name" validate:"required,max=255"`
	Email    string `json:"email"` // optional: a profile on one computer needs only a name and a password
	Password string `json:"password" validate:"required,min=8"`
}

// RegisterResponse returns user ID and token
type RegisterResponse struct {
	UserID    string `json:"userId"`
	Token     string `json:"token"`
	Email     string `json:"email"`
	ExpiresIn int    `json:"expiresIn"`
	// RecoveryCode is shown to the person once, here; only its hash is kept. It resets a forgotten password (/api/auth/reset).
	RecoveryCode string `json:"recoveryCode"`
}

// RegisterHandler - POST /api/auth/register
func (uas *UserAuthServer) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)

	// Validate email (only when one is given)
	if req.Email != "" && !ValidateEmail(req.Email) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid email format"})
		return
	}

	// Validate password strength
	if !ValidatePassword(req.Password) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Password must be at least 8 characters"})
		return
	}

	// Check if email already exists
	var exists bool
	var err error

	// Test Ping to see if database is alive
	pingErr := uas.db.Ping()
	if pingErr != nil {
	}

	// Try explicit connection from pool
	sqlConn, connErr := uas.db.Conn(r.Context())
	if connErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Database error: " + connErr.Error()})
		return
	}
	defer sqlConn.Close()

	if req.Email != "" {
		err = sqlConn.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM users WHERE email = ?)", req.Email).Scan(&exists)
	}
	if err != nil && err != sql.ErrNoRows {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Printf("[Auth] Database error checking email: %v\n", err)
		json.NewEncoder(w).Encode(map[string]string{"error": "Database error: " + err.Error()})
		return
	}

	if exists {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": "Email already registered"})
		return
	}

	// Validate name
	if req.Name == "" || len(req.Name) > 255 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Name is required (max 255 characters)"})
		return
	}
	// The name is what a person signs in with, so two profiles cannot share it.
	var nameTaken bool
	if err := sqlConn.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM users WHERE lower(name) = lower(?) AND password_hash IS NOT NULL)", req.Name).Scan(&nameTaken); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Database error"})
		return
	}
	if nameTaken {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": "That name is already used by another profile"})
		return
	}

	// Hash password
	passwordHash, err := HashPassword(req.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Registration failed"})
		return
	}

	recoveryCode, recoveryHash, err := newRecoveryCode()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Registration failed"})
		return
	}

	// Generate user ID
	userID := GenerateUserID(req.Name + req.Email)
	now := time.Now().Unix()
	var emailValue any // NULL without an address (the column is unique, and "" would be taken by the second profile)
	if req.Email != "" {
		emailValue = req.Email
	}

	// The first profile with a password on a computer that has used Moly before takes over the passwordless local profile: its
	// conversations, people and settings stay with the person who made them, and from then on need the password.
	claimed, claimErr := uas.claimLocalProfile(req.Name, emailValue, passwordHash, recoveryHash, now)
	if claimErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Registration failed"})
		return
	}
	if claimed {
		userID = LocalProfileID
	} else {
		// Create user with name
		_, err = uas.db.Exec(
			"INSERT INTO users (id, email, name, password_hash, recovery_hash, created_at, last_active, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			userID, emailValue, req.Name, passwordHash, recoveryHash, now, now, now,
		)
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Printf("[Auth] Failed to create user: %v\n", err)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create user: " + err.Error()})
		return
	}

	// Auto-create About Me with formal defaults (allows users to skip setup and customize later). A taken-over local profile has its own
	// already: OR IGNORE keeps what it had.
	formalDefaults := map[string]interface{}{
		"uiPreferences": map[string]bool{
			"showTooltips": true,
		},
	}
	formalPrefs, _ := json.Marshal(formalDefaults)
	coreValues := []string{"Respect", "Honesty", "Clarity"}
	coreValuesJSON, _ := json.Marshal(coreValues)
	userInstructions := []string{"I appreciate direct feedback", "I think best through examples", "Validate my feelings"}
	userInstructionsJSON, _ := json.Marshal(userInstructions)

	_, err = uas.db.Exec(
		"INSERT OR IGNORE INTO about_me (user_id, communication_style, core_values, tone_preference, preferences, goals, patterns, user_instructions, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		userID, "Formal & Respectful", string(coreValuesJSON), "Professional", string(formalPrefs), "[]", "[]", string(userInstructionsJSON), now, now,
	)
	if err != nil {
		log.Printf("[Auth] ERROR: Failed to create initial About Me for user %s: %v - registration blocked", userID, err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to initialize user profile. Please try again.",
		})
		return
	}
	log.Printf("[Auth] ✓ Created initial About Me record for user %s with formal defaults (customizable in Settings)", userID)

	// Create initial SystemContext with preset system instructions (for system self-awareness)
	// Preset on registration to match AboutMe pattern (no asymmetry)
	systemInstructions := []string{"Be Socratic - ask questions", "Help me think, don't decide for me", "Respect my autonomy"}
	systemInstructionsJSON, _ := json.Marshal(systemInstructions)

	_, err = uas.db.Exec(
		`INSERT OR IGNORE INTO system_context (user_id, user_feedback, user_directives, system_perceptions,
		                             preferred_interaction_style, system_instructions, helpfulness_rating, clarity_rating,
		                             created_at, updated_at, version)
		 VALUES (?, '[]', '[]', '[]', '', ?, 0.0, 0.0, ?, ?, 1)`,
		userID, string(systemInstructionsJSON), now, now,
	)
	if err != nil {
		log.Printf("[Auth] Warning: Failed to create initial SystemContext for user %s: %v", userID, err)
		// Don't fail registration - SystemContext will be created on-demand if needed
	} else {
		log.Printf("[Auth] ✓ Created initial SystemContext record for user %s (empty defaults)", userID)
	}

	// Generate JWT token (24 hour expiry)
	token := generateToken()
	expiresAt := time.Now().Add(24 * time.Hour).Unix()

	// Store token in sessions table
	_, err = uas.db.Exec(
		"INSERT INTO sessions (id, user_id, created_at, expires_at, last_used) VALUES (?, ?, ?, ?, ?)",
		token, userID, now, expiresAt, now,
	)
	if err != nil {
		// Token storage failed, but user was created - still return success
		fmt.Printf("[Auth] Failed to store session token: %v\n", err)
	}

	fmt.Printf("[Auth] User registered: %s\n", userID)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(RegisterResponse{
		UserID:    userID,
		Token:     token,
		Email:     req.Email,
		ExpiresIn: 86400, // 24 hours in seconds

		RecoveryCode: recoveryCode,
	})
}

// LoginRequest for user login (supports email or username)
type LoginRequest struct {
	EmailOrUsername string `json:"emailOrUsername"`
	Password        string `json:"password"`
}

// LoginResponse returns user ID and token
type LoginResponse struct {
	UserID    string `json:"userId"`
	Token     string `json:"token"`
	Email     string `json:"email"`
	ExpiresIn int    `json:"expiresIn"`
}

// LoginHandler - POST /api/auth/login
func (uas *UserAuthServer) LoginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}

	// Too many wrong tries for this name: refuse for a while (the answer is the same whether or not the name exists).
	loginKey := guardKey("login", req.EmailOrUsername)
	if d := uas.guard.wait(loginKey); d > 0 {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{"error": lockedMessage(d)})
		return
	}

	// Find user by email or username (name field)
	var user User
	var emailCol sql.NullString // accounts made without an address have none
	err := uas.db.QueryRow(
		"SELECT id, email, password_hash FROM users WHERE (email = ? OR lower(name) = lower(?)) AND password_hash IS NOT NULL",
		req.EmailOrUsername, req.EmailOrUsername,
	).Scan(&user.ID, &emailCol, &user.PasswordHash)
	user.Email = emailCol.String

	if err == sql.ErrNoRows {
		uas.guard.fail(loginKey)
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid email, username, or password"})
		return
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Database error"})
		return
	}

	// Verify password
	if !VerifyPassword(user.PasswordHash, req.Password) {
		uas.guard.fail(loginKey)
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid email, username, or password"})
		return
	}

	uas.guard.ok(loginKey)

	// Generate token
	token := generateToken()
	expiresAt := time.Now().Add(24 * time.Hour).Unix()
	now := time.Now().Unix()

	// Store session token
	_, err = uas.db.Exec(
		"INSERT INTO sessions (id, user_id, created_at, expires_at, last_used) VALUES (?, ?, ?, ?, ?)",
		token, user.ID, now, expiresAt, now,
	)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create session"})
		return
	}

	fmt.Printf("[Auth] User logged in: %s\n", user.ID)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(LoginResponse{
		UserID:    user.ID,
		Token:     token,
		Email:     user.Email,
		ExpiresIn: 86400, // 24 hours
	})
}

// VerifyTokenHandler - POST /api/auth/verify
func (uas *UserAuthServer) VerifyTokenHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	token := r.Header.Get("Authorization")
	if token == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Missing token"})
		return
	}

	// Remove "Bearer " prefix if present
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	// Check if token is valid and not expired
	var userID string
	var expiresAt int64
	err := uas.db.QueryRow(
		"SELECT user_id, expires_at FROM sessions WHERE id = ?",
		token,
	).Scan(&userID, &expiresAt)

	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid token"})
		return
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Database error"})
		return
	}

	// Check if expired
	if expiresAt < time.Now().Unix() {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Token expired"})
		return
	}

	// Get user email
	// The local profile has no email address (NULL): that is a valid session, not an error.
	var emailCol sql.NullString
	err = uas.db.QueryRow("SELECT email FROM users WHERE id = ?", userID).Scan(&emailCol)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "User not found"})
		return
	}
	email := emailCol.String

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"userId":    userID,
		"email":     email,
		"expiresAt": expiresAt,
		"valid":     true,
	})
}

// LogoutHandler - POST /api/auth/logout
func (uas *UserAuthServer) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	token := r.Header.Get("Authorization")
	if token == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Missing token"})
		return
	}

	// Remove "Bearer " prefix if present
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	// Delete session
	_, err := uas.db.Exec("DELETE FROM sessions WHERE id = ?", token)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Logout failed"})
		return
	}

	fmt.Printf("[Auth] User logged out\n")

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"success": "true"})
}

// Helper function to generate secure token
func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// LocalProfileID is the one profile of a local installation (no email, no password).
const LocalProfileID = "user_local"

// localSessionLifetime is long on purpose: the profile protects nothing the API token does not already protect, and a
// session that ends in the middle of use would only be an interruption.
const localSessionLifetime = 30 * 24 * time.Hour

// claimLocalProfile gives the passwordless local profile its owner: a name, a password and a recovery code. It does so only when no profile
// with a password exists yet and the local profile holds a conversation or a person, so the second person who registers never takes it.
func (uas *UserAuthServer) claimLocalProfile(name string, email any, passwordHash, recoveryHash string, now int64) (bool, error) {
	var passworded int
	if err := uas.db.QueryRow("SELECT COUNT(*) FROM users WHERE password_hash IS NOT NULL").Scan(&passworded); err != nil || passworded > 0 {
		return false, err
	}
	var local int
	if err := uas.db.QueryRow("SELECT COUNT(*) FROM users WHERE id = ? AND password_hash IS NULL", LocalProfileID).Scan(&local); err != nil || local == 0 {
		return false, err
	}
	if _, err := uas.db.Exec(
		"UPDATE users SET name = ?, email = ?, password_hash = ?, recovery_hash = ?, updated_at = ? WHERE id = ?",
		name, email, passwordHash, recoveryHash, now, LocalProfileID,
	); err != nil {
		return false, err
	}
	// Sessions handed out while it had no password end now.
	uas.db.Exec("DELETE FROM sessions WHERE user_id = ?", LocalProfileID)
	return true, nil
}

// LocalProfileHandler - POST /api/auth/local. Gives the caller a session for the local profile, creating it on first use.
// It is registered only in production mode, where every /api call already needs the per-run token (see security.go), so
// only the extension of this computer's user can reach it. The profile has no password: there is nothing to forget.
func (uas *UserAuthServer) LocalProfileHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	now := time.Now().Unix()

	// Once a profile has a password, nobody gets in without it: this door serves the phone (one owner) and an installation nobody has
	// given a password yet.
	var passworded int
	uas.db.QueryRow("SELECT COUNT(*) FROM users WHERE password_hash IS NOT NULL").Scan(&passworded)
	if passworded > 0 {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "This computer's profiles have passwords. Sign in."})
		return
	}

	if _, err := uas.db.Exec(
		`INSERT OR IGNORE INTO users (id, email, name, password_hash, created_at, last_active, updated_at)
		 VALUES (?, NULL, 'You', NULL, ?, ?, ?)`,
		LocalProfileID, now, now, now,
	); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Could not create the local profile"})
		return
	}

	token := generateToken()
	expires := time.Now().Add(localSessionLifetime).Unix()
	if _, err := uas.db.Exec(
		"INSERT INTO sessions (id, user_id, created_at, expires_at, last_used) VALUES (?, ?, ?, ?, ?)",
		token, LocalProfileID, now, expires, now,
	); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create session"})
		return
	}
	// Old sessions of the local profile are of no use any more.
	uas.db.Exec("DELETE FROM sessions WHERE user_id = ? AND expires_at < ?", LocalProfileID, now)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"userId":    LocalProfileID,
		"token":     token,
		"email":     "",
		"expiresIn": int(localSessionLifetime.Seconds()),
		"local":     true,
	})
}

// recoveryAlphabet has no look-alike characters (0/O, 1/I/L).
const recoveryAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

// newRecoveryCode makes a code of 16 characters in four groups (about 79 bits) and the hash to keep. The code is shown once.
func newRecoveryCode() (code, hash string, err error) {
	b := make([]byte, 16)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	var sb strings.Builder
	for i, v := range b {
		if i > 0 && i%4 == 0 {
			sb.WriteByte('-')
		}
		sb.WriteByte(recoveryAlphabet[int(v)%len(recoveryAlphabet)])
	}
	code = sb.String()
	h, err := HashPassword(normalizeRecoveryCode(code))
	return code, h, err
}

// normalizeRecoveryCode ignores case, spaces and dashes, so the code can be typed the way it was read.
func normalizeRecoveryCode(c string) string {
	return strings.Map(func(r rune) rune {
		if r == '-' || r == ' ' || r == '\t' || r == '\n' {
			return -1
		}
		return r
	}, strings.ToUpper(c))
}

// ResetRequest asks for a new password with the recovery code.
type ResetRequest struct {
	EmailOrUsername string `json:"emailOrUsername"`
	RecoveryCode    string `json:"recoveryCode"`
	NewPassword     string `json:"newPassword"`
}

// ResetHandler - POST /api/auth/reset. A forgotten password is replaced with the help of the recovery code shown at registration. The code
// works once: a new one is made and returned, the old sessions end, and the person is signed in. The answer for a wrong name and for a
// wrong code is the same, so the call does not tell which names exist.
func (uas *UserAuthServer) ResetHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req ResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}
	if !ValidatePassword(req.NewPassword) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Password must be at least 8 characters"})
		return
	}
	resetKey := guardKey("reset", req.EmailOrUsername)
	if d := uas.guard.wait(resetKey); d > 0 {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{"error": lockedMessage(d)})
		return
	}
	bad := func() {
		uas.guard.fail(resetKey)
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "The name or the recovery code is not right"})
	}
	var id string
	var email, recovery sql.NullString
	err := uas.db.QueryRow(
		"SELECT id, email, recovery_hash FROM users WHERE (email = ? OR lower(name) = lower(?)) AND password_hash IS NOT NULL",
		req.EmailOrUsername, req.EmailOrUsername,
	).Scan(&id, &email, &recovery)
	if err != nil || !recovery.Valid || recovery.String == "" || !VerifyPassword(recovery.String, normalizeRecoveryCode(req.RecoveryCode)) {
		bad()
		return
	}
	uas.guard.ok(resetKey)
	passwordHash, err := HashPassword(req.NewPassword)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Reset failed"})
		return
	}
	newCode, newHash, err := newRecoveryCode()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Reset failed"})
		return
	}
	now := time.Now().Unix()
	token := generateToken()
	tx, err := uas.db.Begin()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Reset failed"})
		return
	}
	defer tx.Rollback()
	if _, err = tx.Exec("UPDATE users SET password_hash = ?, recovery_hash = ?, updated_at = ? WHERE id = ?", passwordHash, newHash, now, id); err == nil {
		_, err = tx.Exec("DELETE FROM sessions WHERE user_id = ?", id)
	}
	if err == nil {
		_, err = tx.Exec("INSERT INTO sessions (id, user_id, created_at, expires_at, last_used) VALUES (?, ?, ?, ?, ?)", token, id, now, time.Now().Add(24*time.Hour).Unix(), now)
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Reset failed"})
		return
	}
	log.Printf("[Auth] Password reset with the recovery code: %s", id)
	json.NewEncoder(w).Encode(RegisterResponse{UserID: id, Token: token, Email: email.String, ExpiresIn: 86400, RecoveryCode: newCode})
}

// ProfilesHandler - GET /api/auth/profiles. The names of the profiles that have a password, so the start screen can offer them. The local
// profile is not listed (it has no password and is always there). Names only: no email, no id.
func (uas *UserAuthServer) ProfilesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	rows, err := uas.db.Query("SELECT name FROM users WHERE password_hash IS NOT NULL ORDER BY created_at")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Database error"})
		return
	}
	defer rows.Close()
	names := []string{}
	for rows.Next() {
		var n string
		if rows.Scan(&n) == nil {
			names = append(names, n)
		}
	}
	json.NewEncoder(w).Encode(map[string]any{"profiles": names})
}

// ChangePasswordRequest asks for a new password; the current one proves the person at the keyboard is the owner of the signed-in profile.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

// ChangePasswordHandler - POST /api/auth/password. The caller is signed in (Authorization: Bearer <session>). The current password is checked
// (with the same limit on wrong tries as the sign-in), the password is replaced, and every other session of the profile ends. The recovery code
// stays as it is.
func (uas *UserAuthServer) ChangePasswordHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fail := func(code int, msg string) {
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(map[string]string{"error": msg})
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	session := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	var userID string
	var expiresAt int64
	if session == "" || uas.db.QueryRow("SELECT user_id, expires_at FROM sessions WHERE id = ?", session).Scan(&userID, &expiresAt) != nil || expiresAt < time.Now().Unix() {
		fail(http.StatusUnauthorized, "Sign in first")
		return
	}
	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fail(http.StatusBadRequest, "Invalid request")
		return
	}
	if !ValidatePassword(req.NewPassword) {
		fail(http.StatusBadRequest, "Password must be at least 8 characters")
		return
	}
	var current sql.NullString
	if err := uas.db.QueryRow("SELECT password_hash FROM users WHERE id = ?", userID).Scan(&current); err != nil || !current.Valid || current.String == "" {
		fail(http.StatusBadRequest, "This profile has no password")
		return
	}
	key := guardKey("password", userID)
	if d := uas.guard.wait(key); d > 0 {
		fail(http.StatusTooManyRequests, lockedMessage(d))
		return
	}
	if !VerifyPassword(current.String, req.CurrentPassword) {
		uas.guard.fail(key)
		fail(http.StatusUnauthorized, "The current password is not right")
		return
	}
	uas.guard.ok(key)
	hash, err := HashPassword(req.NewPassword)
	if err != nil {
		fail(http.StatusInternalServerError, "The password could not be changed")
		return
	}
	tx, err := uas.db.Begin()
	if err != nil {
		fail(http.StatusInternalServerError, "The password could not be changed")
		return
	}
	defer tx.Rollback()
	if _, err = tx.Exec("UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?", hash, time.Now().Unix(), userID); err == nil {
		_, err = tx.Exec("DELETE FROM sessions WHERE user_id = ? AND id <> ?", userID, session)
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		fail(http.StatusInternalServerError, "The password could not be changed")
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}
