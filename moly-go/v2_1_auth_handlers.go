package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"moly/auth"
	"moly/database"
)

// AuthResponse is the response from auth endpoints
type AuthResponse struct {
	Success   bool   `json:"success"`
	Code      string `json:"code,omitempty"`
	UserID    string `json:"userId,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
	ExpiresIn int64  `json:"expiresIn,omitempty"`
	IsNewUser bool   `json:"isNewUser,omitempty"`
	Error     string `json:"error,omitempty"`
}

// VerifyResponse is the response from verify endpoint
type VerifyResponse struct {
	Valid      bool   `json:"valid"`
	UserID     string `json:"userId,omitempty"`
	IsNewUser  bool   `json:"isNewUser,omitempty"`
	LastActive int64  `json:"lastActive,omitempty"`
	Error      string `json:"error,omitempty"`
}

// AuthServer handles authentication for V2.1
type AuthServer struct {
	db          *database.Database
	sessionRepo *auth.SessionRepository
}

// NewAuthServer creates a new auth server
func NewAuthServer(db *database.Database) *AuthServer {
	return &AuthServer{
		db:          db,
		sessionRepo: auth.NewSessionRepository(db.GetConnection()),
	}
}

// GenerateCodeHandler generates a new authentication code
// GET /api/v2.1/auth/generate
func (as *AuthServer) GenerateCodeHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)

	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != http.MethodGet {
		respondJSON(w, http.StatusMethodNotAllowed, AuthResponse{
			Success: false,
			Error:   "Method not allowed",
		})
		return
	}

	// Generate code
	code, err := auth.GenerateCode()
	if err != nil {
		Logger.WithError(err).Error("[Auth] Failed to generate code")
		respondJSON(w, http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to generate code",
		})
		return
	}

	Logger.WithField("code", code[:8]+"...").Info("[Auth] Generated new authentication code")

	respondJSON(w, http.StatusOK, AuthResponse{
		Success:   true,
		Code:      code,
		ExpiresIn: int64(auth.DefaultCodeConfig.ExpirationTTL.Seconds()),
	})
}

// LoginHandler logs in a user with a code
// POST /api/v2.1/auth/login
func (as *AuthServer) LoginHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)

	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, AuthResponse{
			Success: false,
			Error:   "Method not allowed",
		})
		return
	}

	// Parse request
	var req struct {
		Code string `json:"code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Invalid request format",
		})
		return
	}

	// Validate code format
	if req.Code == "" {
		respondJSON(w, http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Code is required",
		})
		return
	}

	if !auth.IsValid(req.Code) {
		respondJSON(w, http.StatusBadRequest, AuthResponse{
			Success: false,
			Error:   "Invalid code format",
		})
		return
	}

	// Generate userID from code (deterministic)
	// In V2.1, userID is derived from the code for security
	userID := generateUserIDFromCode(req.Code)

	// Create user in database if doesn't exist
	isNewUser := false
	if err := as.db.CreateUser(userID); err != nil {
		// Check if this is a new user by looking at context level
		// For now, assume new if creation succeeds
		isNewUser = true
	}

	// Create session
	session, err := as.sessionRepo.CreateSession(userID, req.Code)
	if err != nil {
		Logger.WithError(err).Error("[Auth] Failed to create session")
		respondJSON(w, http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to create session",
		})
		return
	}

	Logger.WithField("userId", userID[:8]+"...").Info("[Auth] User logged in successfully")

	respondJSON(w, http.StatusOK, AuthResponse{
		Success:   true,
		UserID:    userID,
		SessionID: session.ID,
		ExpiresIn: int64(session.ExpiresAt.Sub(session.CreatedAt).Seconds()),
		IsNewUser: isNewUser,
	})
}

// VerifyHandler verifies a session
// GET /api/v2.1/auth/verify
// Header: Authorization: Bearer {sessionId}
func (as *AuthServer) VerifyHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)

	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != http.MethodGet {
		respondJSON(w, http.StatusMethodNotAllowed, VerifyResponse{
			Valid: false,
			Error: "Method not allowed",
		})
		return
	}

	// Extract session ID from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		respondJSON(w, http.StatusUnauthorized, VerifyResponse{
			Valid: false,
			Error: "Missing Authorization header",
		})
		return
	}

	// Parse "Bearer {sessionId}"
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		respondJSON(w, http.StatusUnauthorized, VerifyResponse{
			Valid: false,
			Error: "Invalid Authorization header format",
		})
		return
	}

	sessionID := parts[1]

	// Validate session
	session, err := as.sessionRepo.ValidateSession(sessionID)
	if err != nil {
		Logger.WithError(err).Debug("[Auth] Session validation failed")
		respondJSON(w, http.StatusUnauthorized, VerifyResponse{
			Valid: false,
			Error: "Invalid or expired session",
		})
		return
	}

	Logger.WithField("userId", session.UserID[:8]+"...").Debug("[Auth] Session verified")

	respondJSON(w, http.StatusOK, VerifyResponse{
		Valid:      true,
		UserID:     session.UserID,
		LastActive: session.LastActive.Unix(),
		IsNewUser:  false, // Would need to check in database
	})
}

// LogoutHandler logs out a user
// POST /api/v2.1/auth/logout
// Header: Authorization: Bearer {sessionId}
func (as *AuthServer) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)

	if handleCORSPreflight(w, r) {
		return
	}

	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, AuthResponse{
			Success: false,
			Error:   "Method not allowed",
		})
		return
	}

	// Extract session ID from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		respondJSON(w, http.StatusUnauthorized, AuthResponse{
			Success: false,
			Error:   "Missing Authorization header",
		})
		return
	}

	// Parse "Bearer {sessionId}"
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		respondJSON(w, http.StatusUnauthorized, AuthResponse{
			Success: false,
			Error:   "Invalid Authorization header format",
		})
		return
	}

	sessionID := parts[1]

	// Delete session
	if err := as.sessionRepo.DeleteSession(sessionID); err != nil {
		Logger.WithError(err).Error("[Auth] Failed to logout")
		respondJSON(w, http.StatusInternalServerError, AuthResponse{
			Success: false,
			Error:   "Failed to logout",
		})
		return
	}

	Logger.Info("[Auth] User logged out successfully")

	respondJSON(w, http.StatusOK, AuthResponse{
		Success: true,
	})
}

// generateUserIDFromCode generates a deterministic userID from a code
// This ensures the same code always maps to the same user
func generateUserIDFromCode(code string) string {
	// Use SHA256 hash of code to generate stable userID
	// Format: user_{hash} where hash is first 16 chars of code hash
	hasher := sha256.New()
	hasher.Write([]byte(code))
	hash := fmt.Sprintf("%x", hasher.Sum(nil))
	return "user_" + hash[:16]
}

// SessionMiddleware extracts and validates session from request
// Returns userID if valid, empty string if invalid
func (as *AuthServer) SessionMiddleware(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("missing Authorization header")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", fmt.Errorf("invalid Authorization header format")
	}

	sessionID := parts[1]

	// Validate session
	session, err := as.sessionRepo.ValidateSession(sessionID)
	if err != nil {
		return "", fmt.Errorf("invalid or expired session")
	}

	return session.UserID, nil
}
