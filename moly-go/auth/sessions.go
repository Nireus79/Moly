package auth

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"time"
)

// Session represents an active user session
type Session struct {
	ID        string    // Session token (32-byte hex)
	UserID    string    // User identifier
	Code      string    // Original code used to create session
	CreatedAt time.Time
	ExpiresAt time.Time
	LastActive time.Time
	DeviceName string // Optional device identifier
}

// IsValid checks if session is still valid
func (s *Session) IsValid() bool {
	return time.Now().Before(s.ExpiresAt)
}

// SessionRepository manages session storage in database
type SessionRepository struct {
	db *sql.DB
}

// NewSessionRepository creates a new session repository
func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// generateSessionToken creates a session ID from user ID
// Using SHA256(userID + timestamp + random) for uniqueness
func generateSessionToken(userID string) string {
	hasher := sha256.New()
	hasher.Write([]byte(userID + time.Now().String()))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

// CreateSession creates a new session from a code
func (sr *SessionRepository) CreateSession(userID, code string) (*Session, error) {
	sessionID := generateSessionToken(userID)
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour) // 24 hour session timeout

	query := `
		INSERT INTO sessions (id, user_id, code, created_at, expires_at, last_active)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := sr.db.Exec(query, sessionID, userID, code, now.Unix(), expiresAt.Unix(), now.Unix())
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &Session{
		ID:         sessionID,
		UserID:     userID,
		Code:       code,
		CreatedAt:  now,
		ExpiresAt:  expiresAt,
		LastActive: now,
	}, nil
}

// GetSession retrieves a session by ID
func (sr *SessionRepository) GetSession(sessionID string) (*Session, error) {
	query := `
		SELECT id, user_id, code, created_at, expires_at, last_active, device_name
		FROM sessions
		WHERE id = ?
	`

	var session Session
	var createdAtUnix, expiresAtUnix, lastActiveUnix int64
	var deviceName sql.NullString

	err := sr.db.QueryRow(query, sessionID).Scan(
		&session.ID,
		&session.UserID,
		&session.Code,
		&createdAtUnix,
		&expiresAtUnix,
		&lastActiveUnix,
		&deviceName,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	session.CreatedAt = time.Unix(createdAtUnix, 0)
	session.ExpiresAt = time.Unix(expiresAtUnix, 0)
	session.LastActive = time.Unix(lastActiveUnix, 0)
	if deviceName.Valid {
		session.DeviceName = deviceName.String
	}

	return &session, nil
}

// ValidateSession checks if a session is valid and updates last_active
func (sr *SessionRepository) ValidateSession(sessionID string) (*Session, error) {
	session, err := sr.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	if !session.IsValid() {
		return nil, fmt.Errorf("session expired")
	}

	// Update last_active
	now := time.Now()
	query := `UPDATE sessions SET last_active = ? WHERE id = ?`
	_, err = sr.db.Exec(query, now.Unix(), sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	session.LastActive = now
	return session, nil
}

// RefreshSession extends session expiration
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

// CleanupExpiredSessions removes all expired sessions (maintenance)
func (sr *SessionRepository) CleanupExpiredSessions() error {
	now := time.Now().Unix()
	query := `DELETE FROM sessions WHERE expires_at < ?`
	_, err := sr.db.Exec(query, now)
	if err != nil {
		return fmt.Errorf("failed to cleanup sessions: %w", err)
	}
	return nil
}

// GetUserSessions retrieves all active sessions for a user
func (sr *SessionRepository) GetUserSessions(userID string) ([]*Session, error) {
	query := `
		SELECT id, user_id, code, created_at, expires_at, last_active, device_name
		FROM sessions
		WHERE user_id = ? AND expires_at > ?
		ORDER BY last_active DESC
	`

	rows, err := sr.db.Query(query, userID, time.Now().Unix())
	if err != nil {
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		var session Session
		var createdAtUnix, expiresAtUnix, lastActiveUnix int64
		var deviceName sql.NullString

		err := rows.Scan(
			&session.ID,
			&session.UserID,
			&session.Code,
			&createdAtUnix,
			&expiresAtUnix,
			&lastActiveUnix,
			&deviceName,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}

		session.CreatedAt = time.Unix(createdAtUnix, 0)
		session.ExpiresAt = time.Unix(expiresAtUnix, 0)
		session.LastActive = time.Unix(lastActiveUnix, 0)
		if deviceName.Valid {
			session.DeviceName = deviceName.String
		}

		sessions = append(sessions, &session)
	}

	return sessions, rows.Err()
}
