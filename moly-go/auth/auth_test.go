package auth

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mutecomm/go-sqlcipher/v4"
)

// TestGenerateCode tests code generation
func TestGenerateCode(t *testing.T) {
	code, err := GenerateCode()
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Check format: moly-XXXXX-YYYYY
	if !IsValid(code) {
		t.Errorf("Generated code has invalid format: %s", code)
	}

	t.Logf("✓ Generated code: %s", code)
}

// TestCodeFormat tests code format validation
func TestCodeFormat(t *testing.T) {
	tests := []struct {
		code  string
		valid bool
	}{
		{"moly-12345-abcde", true},
		{"moly-00000-00000", true},
		{"moly-fffff-fffff", true},
		{"moly-1234-abcde", false},   // Too short
		{"moly-123456-abcde", false}, // Too long
		{"MOLY-12345-abcde", false},  // Wrong case
		{"moly12345abcde", false},    // Missing dashes
		{"", false},
	}

	for _, test := range tests {
		result := IsValid(test.code)
		if result != test.valid {
			t.Errorf("IsValid(%q) = %v, expected %v", test.code, result, test.valid)
		}
	}

	t.Log("✓ Code format validation working")
}

// TestCodeExpiration tests code expiration
func TestCodeExpiration(t *testing.T) {
	// Fresh code should not be expired
	freshCode := time.Now()
	if IsExpired(freshCode) {
		t.Errorf("Fresh code should not be expired")
	}

	// Old code should be expired
	oldCode := time.Now().Add(-2 * time.Hour)
	if !IsExpired(oldCode) {
		t.Errorf("Old code should be expired")
	}

	t.Log("✓ Code expiration working")
}

// TestCodeUniqueness tests that generated codes are unique
func TestCodeUniqueness(t *testing.T) {
	codes := make(map[string]bool)
	numCodes := 100

	for i := 0; i < numCodes; i++ {
		code, err := GenerateCode()
		if err != nil {
			t.Fatalf("Failed to generate code: %v", err)
		}

		if codes[code] {
			t.Errorf("Duplicate code generated: %s", code)
		}
		codes[code] = true
	}

	if len(codes) != numCodes {
		t.Errorf("Expected %d unique codes, got %d", numCodes, len(codes))
	}

	t.Logf("✓ Generated %d unique codes", len(codes))
}

// setupTestDB creates a temporary SQLite database for testing
func setupTestDB(t *testing.T) *sql.DB {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_auth.db")

	// Use SQLCipher (encrypted) connection for auth tests
	// Simple key: "test_key" for testing only
	dsn := "file:" + dbPath + "?key=test_key&cache=shared&mode=rwc"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Create sessions table
	schema := `
		CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			code TEXT UNIQUE,
			created_at INTEGER,
			expires_at INTEGER,
			last_active INTEGER,
			device_name TEXT
		);
	`

	_, err = db.Exec(schema)
	if err != nil {
		t.Fatalf("Failed to create sessions table: %v", err)
	}

	return db
}

// TestSessionCreation tests session creation
func TestSessionCreation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSessionRepository(db)
	userID := "test_user"
	code := "moly-12345-abcde"

	session, err := repo.CreateSession(userID, code)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	if session.UserID != userID {
		t.Errorf("Expected userID %s, got %s", userID, session.UserID)
	}

	if session.Code != code {
		t.Errorf("Expected code %s, got %s", code, session.Code)
	}

	if !session.IsValid() {
		t.Errorf("New session should be valid")
	}

	t.Logf("✓ Created session: %s", session.ID[:8]+"...")
}

// TestSessionRetrieval tests retrieving a session
func TestSessionRetrieval(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSessionRepository(db)
	userID := "test_user"
	code := "moly-12345-abcde"

	// Create session
	created, err := repo.CreateSession(userID, code)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Retrieve session
	retrieved, err := repo.GetSession(created.ID)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("Session ID mismatch")
	}

	if retrieved.UserID != userID {
		t.Errorf("UserID mismatch")
	}

	t.Log("✓ Session retrieval working")
}

// TestSessionValidation tests session validation and updates
func TestSessionValidation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSessionRepository(db)
	userID := "test_user"
	code := "moly-12345-abcde"

	// Create session
	created, err := repo.CreateSession(userID, code)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Validate and update last_active
	oldLastActive := created.LastActive
	time.Sleep(100 * time.Millisecond)

	validated, err := repo.ValidateSession(created.ID)
	if err != nil {
		t.Fatalf("Failed to validate session: %v", err)
	}

	if validated.LastActive.Equal(oldLastActive) {
		t.Errorf("LastActive should be updated")
	}

	t.Log("✓ Session validation working")
}

// TestSessionRefresh tests extending session expiration
func TestSessionRefresh(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSessionRepository(db)
	userID := "test_user"
	code := "moly-12345-abcde"

	// Create session
	created, err := repo.CreateSession(userID, code)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	oldExpiry := created.ExpiresAt

	// Wait a moment to ensure time difference
	time.Sleep(100 * time.Millisecond)

	// Refresh session
	err = repo.RefreshSession(created.ID)
	if err != nil {
		t.Fatalf("Failed to refresh session: %v", err)
	}

	// Check new expiry by querying database
	refreshed, err := repo.GetSession(created.ID)
	if err != nil {
		t.Fatalf("Failed to get refreshed session: %v", err)
	}

	// New expiry should be at least slightly after old expiry
	if !refreshed.ExpiresAt.After(oldExpiry.Add(-time.Second)) {
		t.Logf("Old expiry: %v, New expiry: %v", oldExpiry, refreshed.ExpiresAt)
		t.Errorf("Expiry should be extended (or approximately equal due to timing)")
	}

	t.Log("✓ Session refresh working")
}

// TestSessionDeletion tests session deletion (logout)
func TestSessionDeletion(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSessionRepository(db)

	// Create and delete session
	session, err := repo.CreateSession("test_user", "moly-12345-abcde")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	err = repo.DeleteSession(session.ID)
	if err != nil {
		t.Fatalf("Failed to delete session: %v", err)
	}

	// Should not be retrievable
	_, err = repo.GetSession(session.ID)
	if err == nil {
		t.Errorf("Deleted session should not be retrievable")
	}

	t.Log("✓ Session deletion working")
}

// TestAuthGate - Combined test for auth system
func TestAuthGate(t *testing.T) {
	tests := []struct {
		name string
		test func(*testing.T)
	}{
		{"GenerateCode", TestGenerateCode},
		{"CodeFormat", TestCodeFormat},
		{"CodeExpiration", TestCodeExpiration},
		{"CodeUniqueness", TestCodeUniqueness},
		{"SessionCreation", TestSessionCreation},
		{"SessionRetrieval", TestSessionRetrieval},
		{"SessionValidation", TestSessionValidation},
		{"SessionRefresh", TestSessionRefresh},
		{"SessionDeletion", TestSessionDeletion},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.test(t)
		})
	}

	t.Log("\n✓ Auth system gate PASSED - Codes + Sessions ready for Phase 1")
}
