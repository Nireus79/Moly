package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// SessionRepository handles auth data operations
type SessionRepository struct {
	db *sql.DB
}

// NewSessionRepository creates a new session repository
func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// CreateSession creates a session from userID and code (for tests/auth flow)
func (sr *SessionRepository) CreateSession(userID, code string) (*SessionWithTime, error) {
	sessionID := generateSessionTokenFromUserID(userID)
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)

	query := `
		INSERT INTO sessions (id, user_id, device_id, created_at, expires_at, last_used)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := sr.db.Exec(query, sessionID, userID, "", now.Unix(), expiresAt.Unix(), now.Unix())
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &SessionWithTime{
		ID:         sessionID,
		UserID:     userID,
		Code:       code,
		DeviceID:   "",
		CreatedAt:  now,
		ExpiresAt:  expiresAt,
		LastActive: now,
	}, nil
}

// StoreLoginCode stores a login code in the database
func (sr *SessionRepository) StoreLoginCode(code string, expiresAt int64, deviceID string) error {
	query := `
		INSERT INTO login_codes (code, expires_at, device_id, used, created_at)
		VALUES (?, ?, ?, false, ?)
	`

	_, err := sr.db.Exec(query, code, expiresAt, deviceID, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("failed to store login code: %w", err)
	}

	return nil
}

// ValidateAndConsumeLoginCode validates a code and marks it as used
func (sr *SessionRepository) ValidateAndConsumeLoginCode(code string) (string, error) {
	var codeVal string
	var userID sql.NullString
	var expiresAt int64
	var used bool

	// Normalize code to lowercase for case-insensitive comparison
	code = strings.ToLower(code)

	query := `
		SELECT code, user_id, expires_at, used
		FROM login_codes
		WHERE LOWER(code) = LOWER(?)
		LIMIT 1
	`

	err := sr.db.QueryRow(query, code).Scan(
		&codeVal,
		&userID,
		&expiresAt,
		&used,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("code not found")
		}
		return "", fmt.Errorf("database error: %w", err)
	}

	// Check if code is expired
	if expiresAt < time.Now().Unix() {
		return "", fmt.Errorf("code expired")
	}

	// Check if code was already used
	if used {
		return "", fmt.Errorf("code already used")
	}

	// If user_id is NULL, create a new user
	if !userID.Valid {
		userID.String = generateUserID()
		userID.Valid = true

		// Create the user in database
		insertQuery := `INSERT INTO users (id, created_at, last_active) VALUES (?, ?, ?)`
		_, err = sr.db.Exec(insertQuery, userID.String, time.Now().Unix(), time.Now().Unix())
		if err != nil {
			return "", fmt.Errorf("failed to create user: %w", err)
		}

		// Update the code with the new user_id
		updateCodeQuery := `UPDATE login_codes SET user_id = ? WHERE code = ?`
		_, err = sr.db.Exec(updateCodeQuery, userID.String, code)
		if err != nil {
			return "", fmt.Errorf("failed to update code with user_id: %w", err)
		}
	}

	// Mark code as used
	updateQuery := `UPDATE login_codes SET used = true WHERE code = ?`
	_, err = sr.db.Exec(updateQuery, code)
	if err != nil {
		return "", fmt.Errorf("failed to mark code as used: %w", err)
	}

	return userID.String, nil
}

// generateUserID generates a unique user ID
func generateUserID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return "user_" + hex.EncodeToString(b)
}

// StoreSession stores a new session in the database (stores a pre-created session object)
func (sr *SessionRepository) StoreSession(session *Session) error {
	query := `
		INSERT INTO sessions (id, user_id, device_id, created_at, expires_at, last_used)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := sr.db.Exec(
		query,
		session.ID,
		session.UserID,
		session.DeviceID,
		session.CreatedAt,
		session.ExpiresAt,
		session.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// ValidateSession checks if a session is valid and returns session data with time.Time fields
func (sr *SessionRepository) ValidateSession(sessionID string) (*SessionWithTime, error) {
	sessionWithTime, err := sr.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	// Check if session is expired
	if sessionWithTime.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("session expired")
	}

	// Update last_used timestamp
	updateQuery := `UPDATE sessions SET last_used = ? WHERE id = ?`
	_, err = sr.db.Exec(updateQuery, time.Now().Unix(), sessionID)
	if err != nil {
		// Log but don't fail - updating last_used is not critical
		fmt.Printf("warning: failed to update session last_used: %v\n", err)
	}

	return sessionWithTime, nil
}

// InvalidateSession deletes a session
func (sr *SessionRepository) InvalidateSession(sessionID string) error {
	query := `DELETE FROM sessions WHERE id = ?`

	_, err := sr.db.Exec(query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to invalidate session: %w", err)
	}

	return nil
}

// CleanupExpiredCodes removes expired login codes
func (sr *SessionRepository) CleanupExpiredCodes() error {
	query := `DELETE FROM login_codes WHERE expires_at < ?`

	_, err := sr.db.Exec(query, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("failed to cleanup expired codes: %w", err)
	}

	return nil
}

// CleanupExpiredSessions removes expired sessions
func (sr *SessionRepository) CleanupExpiredSessions() error {
	query := `DELETE FROM sessions WHERE expires_at < ?`

	_, err := sr.db.Exec(query, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}

	return nil
}

// GetSession retrieves a session by ID and returns it with time.Time fields
func (sr *SessionRepository) GetSession(sessionID string) (*SessionWithTime, error) {
	var session SessionWithTime

	query := `
		SELECT id, user_id, device_id, created_at, expires_at, last_used
		FROM sessions
		WHERE id = ?
		LIMIT 1
	`

	var createdAt, expiresAt, lastUsed int64
	err := sr.db.QueryRow(query, sessionID).Scan(
		&session.ID,
		&session.UserID,
		&session.DeviceID,
		&createdAt,
		&expiresAt,
		&lastUsed,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	session.CreatedAt = time.Unix(createdAt, 0)
	session.ExpiresAt = time.Unix(expiresAt, 0)
	session.LastActive = time.Unix(lastUsed, 0)

	return &session, nil
}

// RefreshSession extends session expiration by 24 hours
func (sr *SessionRepository) RefreshSession(sessionID string) error {
	newExpiry := time.Now().Add(24 * time.Hour)
	query := `UPDATE sessions SET expires_at = ? WHERE id = ?`
	_, err := sr.db.Exec(query, newExpiry.Unix(), sessionID)
	if err != nil {
		return fmt.Errorf("failed to refresh session: %w", err)
	}
	return nil
}

// DeleteSession removes a session (logout)
func (sr *SessionRepository) DeleteSession(sessionID string) error {
	query := `DELETE FROM sessions WHERE id = ?`
	_, err := sr.db.Exec(query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// generateSessionTokenFromUserID creates a unique session ID from userID
func generateSessionTokenFromUserID(userID string) string {
	hasher := sha256.New()
	hasher.Write([]byte(userID + time.Now().String()))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}
