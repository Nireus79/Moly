package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// UserAuthServer handles user registration and login
type UserAuthServer struct {
	db *sql.DB
}

// NewUserAuthServer creates a new user auth server
func NewUserAuthServer(db *sql.DB) *UserAuthServer {
	return &UserAuthServer{db: db}
}

// RegisterRequest for user registration
type RegisterRequest struct {
	Name     string `json:"name" validate:"required,max=255"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

// RegisterResponse returns user ID and token
type RegisterResponse struct {
	UserID  string `json:"userId"`
	Token   string `json:"token"`
	Email   string `json:"email"`
	ExpiresIn int `json:"expiresIn"`
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

	// Validate email
	if !ValidateEmail(req.Email) {
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
	err := uas.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = ?)", req.Email).Scan(&exists)
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

	// Hash password
	passwordHash, err := HashPassword(req.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Registration failed"})
		return
	}

	// Generate user ID
	userID := GenerateUserID(req.Email)
	now := time.Now().Unix()

	// Create user with name
	_, err = uas.db.Exec(
		"INSERT INTO users (id, email, name, password_hash, created_at, last_active, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		userID, req.Email, req.Name, passwordHash, now, now, now,
	)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Printf("[Auth] Failed to create user: %v\n", err)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create user: " + err.Error()})
		return
	}

	// Auto-create empty About Me for context extraction (Issue #11 - Direct messaging support)
	_, err = uas.db.Exec(
		"INSERT INTO about_me (user_id, communication_style, core_values, tone_preference, preferences, goals, patterns, created_at, updated_at) VALUES (?, '', '[]', '', '{}', '[]', '[]', ?, ?)",
		userID, now, now,
	)
	if err != nil {
		log.Printf("[Auth] ERROR: Failed to create initial About Me for user %s: %v - registration blocked", userID, err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to initialize user profile. Please try again.",
		})
		return
	}
	log.Printf("[Auth] ✓ Created initial About Me record for user: %s", userID)

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

	// Find user by email or username (name field)
	var user User
	err := uas.db.QueryRow(
		"SELECT id, email, password_hash FROM users WHERE email = ? OR name = ?",
		req.EmailOrUsername, req.EmailOrUsername,
	).Scan(&user.ID, &user.Email, &user.PasswordHash)

	if err == sql.ErrNoRows {
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
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid email, username, or password"})
		return
	}

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
	var email string
	err = uas.db.QueryRow("SELECT email FROM users WHERE id = ?", userID).Scan(&email)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "User not found"})
		return
	}

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
