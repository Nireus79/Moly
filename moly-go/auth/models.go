package auth

import "time"


// Session represents an authenticated user session
type Session struct {
	ID        string `db:"id"`
	UserID    string `db:"user_id"`
	DeviceID  string `db:"device_id"`
	CreatedAt int64  `db:"created_at"`
	ExpiresAt int64  `db:"expires_at"`
	LastUsed  int64  `db:"last_used"`
}

// SessionWithTime represents a session with time.Time fields (for external APIs)
type SessionWithTime struct {
	ID         string
	UserID     string
	Code       string // Original login code
	DeviceID   string
	CreatedAt  time.Time
	ExpiresAt  time.Time
	LastActive time.Time
}

// IsValid checks if session is still valid
func (s *SessionWithTime) IsValid() bool {
	return time.Now().Before(s.ExpiresAt)
}


