package database

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mutecomm/go-sqlcipher/v4"
)

// EncryptionConfig holds encryption settings
type EncryptionConfig struct {
	Enabled bool
	UserID  string
	Salt    string // Static salt for key derivation
}

// DefaultSalt is a hardcoded salt used for key derivation
// In production, consider storing this securely or deriving from hardware ID
const DefaultSalt = "moly-v2.1-encryption-salt-2026"

// DeriveKey derives a 32-byte encryption key from userID and salt
// Uses SHA256(userID + salt)
func DeriveKey(userID string) [32]byte {
	hasher := sha256.New()
	hasher.Write([]byte(userID + DefaultSalt))
	key := [32]byte{}
	copy(key[:], hasher.Sum(nil))
	return key
}

// OpenEncrypted opens a SQLite database with SQLCipher encryption
// The database is encrypted with AES-256, key derived from userID
func OpenEncrypted(dbPath string, userID string) (*sql.DB, error) {
	key := DeriveKey(userID)

	// SQLCipher connection string format:
	// file:path?key=hex(key_bytes)&cache=shared&mode=rwc&_journal_mode=WAL
	// Using PRAGMA key='...' approach via DSN
	dsn := fmt.Sprintf("file:%s?key=%s&cache=shared&mode=rwc&_journal_mode=WAL&_timeout=5000",
		dbPath,
		fmt.Sprintf("%x", key[:]))

	conn, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open encrypted database: %w", err)
	}

	// Configure connection pool
	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(5)

	// Test connection (will fail if key is wrong)
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping encrypted database (wrong key?): %w", err)
	}

	log.Printf("[Encryption] Database opened with AES-256 encryption (userID: %s)", userID)
	return conn, nil
}

// OpenUnencrypted opens a standard SQLite database (legacy, V2.0 compat)
func OpenUnencrypted(dbPath string) (*sql.DB, error) {
	conn, err := sql.Open("sqlite3", dbPath+"?cache=shared&mode=rwc&_journal_mode=WAL&_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(5)

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("[Encryption] Database opened WITHOUT encryption (legacy mode)")
	return conn, nil
}
