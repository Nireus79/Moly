package auth

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/mutecomm/go-sqlcipher/v4"
)


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
			device_id TEXT,
			created_at INTEGER,
			expires_at INTEGER,
			last_used INTEGER
		);
		CREATE TABLE IF NOT EXISTS login_codes (
			code TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			expires_at INTEGER NOT NULL,
			device_id TEXT,
			used BOOLEAN NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL
		);
	`

	_, err = db.Exec(schema)
	if err != nil {
		t.Fatalf("Failed to create sessions table: %v", err)
	}

	return db
}






