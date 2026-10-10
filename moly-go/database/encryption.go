package database

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/mutecomm/go-sqlcipher/v4"
	"github.com/zalando/go-keyring"
)

const (
	dbKeyEnv     = "MOLY_DB_KEY"
	keyBytes     = 32
	keychainSvc  = "moly"
	keychainUser = "database-key"
)

// LoadOrCreateKey returns the 256-bit database key. MOLY_DB_KEY (64 hex chars)
// overrides everything. Otherwise the key is read from the OS keychain, and
// generated and stored there on first use. If no keychain is available the
// error is returned; the key is never written to a plain file.
func LoadOrCreateKey(dbPath string) ([]byte, error) {
	if env := strings.TrimSpace(os.Getenv(dbKeyEnv)); env != "" {
		return decodeKey(env)
	}

	user := keychainUser + ":" + dbPath
	stored, err := keyring.Get(keychainSvc, user)
	if err == nil {
		return decodeKey(stored)
	}
	if !errors.Is(err, keyring.ErrNotFound) {
		return nil, fmt.Errorf("OS keychain unavailable (%v); set %s to provide the database key", err, dbKeyEnv)
	}

	key := make([]byte, keyBytes)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate database key: %w", err)
	}
	if err := keyring.Set(keychainSvc, user, hex.EncodeToString(key)); err != nil {
		return nil, fmt.Errorf("failed to store database key in OS keychain (%v); set %s to provide it", err, dbKeyEnv)
	}
	log.Printf("[Encryption] Created new database key in the OS keychain")
	return key, nil
}

func decodeKey(s string) ([]byte, error) {
	key, err := hex.DecodeString(s)
	if err != nil || len(key) != keyBytes {
		return nil, fmt.Errorf("database key must be %d hex characters", keyBytes*2)
	}
	return key, nil
}

// OpenEncrypted opens a SQLCipher database (AES-256) using the given key.
func OpenEncrypted(dbPath string, key []byte) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma_key=x'%s'&mode=rwc&_journal_mode=WAL&_timeout=5000&_foreign_keys=1",
		dbPath, hex.EncodeToString(key))

	conn, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(10)
	conn.SetConnMaxLifetime(5 * time.Minute)

	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open database (wrong key or not an encrypted database?): %w", err)
	}

	log.Printf("[Encryption] Database opened with AES-256 encryption")
	return conn, nil
}
