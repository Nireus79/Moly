package database

import (
	"crypto/rand"
	"database/sql"
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	_ "github.com/mutecomm/go-sqlcipher/v4"
)

//go:embed schema.sql
var schemaFS embed.FS

const SchemaVersion = 3

// schemaV1ToV2 adds the conversation_context table to a version 1 database without touching existing data.
const schemaV1ToV2 = `
CREATE TABLE IF NOT EXISTS conversation_context (
    conversation_id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    state TEXT NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
);
PRAGMA user_version = 2;
`

// schemaV2ToV3 adds contacts.name_status to a version 2 database. Existing contacts are 'named': they keep their names.
const schemaV2ToV3 = `
ALTER TABLE contacts ADD COLUMN name_status TEXT NOT NULL DEFAULT 'named';
PRAGMA user_version = 3;
`

// contactsHasNameStatus reports whether the contacts table already has the name_status column.
func (db *Database) contactsHasNameStatus() (bool, error) {
	var count int
	if err := db.conn.QueryRow("SELECT COUNT(*) FROM pragma_table_info('contacts') WHERE name = 'name_status'").Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

// Database - Main database connection handler
type Database struct {
	conn *sql.DB
	mu   sync.RWMutex
}

var (
	instance *Database
	once     sync.Once
)

// Init opens the encrypted database at dbPath, creating its key on first use,
// and creates the schema if the database is empty.
func Init(dbPath string) (*Database, error) {
	var err error
	once.Do(func() {
		if dbPath == ":memory:" {
			instance, err = initWithKey(dbPath, nil)
			return
		}
		if mkErr := os.MkdirAll(filepath.Dir(dbPath), 0700); mkErr != nil {
			err = fmt.Errorf("failed to create database directory: %w", mkErr)
			return
		}
		key, keyErr := LoadOrCreateKey(dbPath)
		if keyErr != nil {
			err = keyErr
			return
		}
		instance, err = initWithKey(dbPath, key)
	})
	return instance, err
}

func initWithKey(dbPath string, key []byte) (*Database, error) {
	if key == nil {
		key = make([]byte, keyBytes)
		if _, err := rand.Read(key); err != nil {
			return nil, fmt.Errorf("failed to generate ephemeral key: %w", err)
		}
	}
	conn, err := OpenEncrypted(dbPath, key)
	if err != nil {
		return nil, err
	}
	db := &Database{conn: conn}
	if err := db.applySchema(); err != nil {
		return nil, fmt.Errorf("failed to apply schema: %w", err)
	}
	return db, nil
}

// applySchema creates the schema in an empty database and refuses to touch any other.
func (db *Database) applySchema() error {
	if _, err := db.conn.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	var version int
	if err := db.conn.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("failed to read schema version: %w", err)
	}
	if version == SchemaVersion {
		return nil
	}

	var tables int
	if err := db.conn.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table'").Scan(&tables); err != nil {
		return fmt.Errorf("failed to inspect database: %w", err)
	}
	if version == 1 && tables > 0 {
		if _, err := db.conn.Exec(schemaV1ToV2); err != nil {
			return fmt.Errorf("failed to upgrade schema from version 1: %w", err)
		}
		log.Printf("[Database] Upgraded schema from version 1 to 2")
		version = 2
	}
	if version == 2 && tables > 0 {
		hasColumn, err := db.contactsHasNameStatus()
		if err != nil {
			return fmt.Errorf("failed to inspect contacts table: %w", err)
		}
		if hasColumn {
			// Already present (for example when the schema was created by the current file): only bump the version
			if _, err := db.conn.Exec("PRAGMA user_version = 3;"); err != nil {
				return fmt.Errorf("failed to set schema version 3: %w", err)
			}
		} else if _, err := db.conn.Exec(schemaV2ToV3); err != nil {
			return fmt.Errorf("failed to upgrade schema from version 2: %w", err)
		}
		log.Printf("[Database] Upgraded schema from version 2 to %d", SchemaVersion)
		return nil
	}
	if version != 0 || tables > 0 {
		return fmt.Errorf("database schema version %d does not match expected %d; move the database file aside to create a new one", version, SchemaVersion)
	}

	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read schema: %w", err)
	}
	if _, err := db.conn.Exec(string(schema)); err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	log.Printf("[Database] Schema version %d created", SchemaVersion)
	return nil
}

// GetConnection - Get underlying SQL connection
func (db *Database) GetConnection() *sql.DB {
	db.mu.RLock()
	defer db.mu.RUnlock()
	log.Printf("[GetConnection] db.conn=%p, nil=%v", db.conn, db.conn == nil)
	if db.conn != nil {
		// Test if the connection is actually valid
		if err := db.conn.Ping(); err != nil {
			log.Printf("[GetConnection] WARNING: Ping on returned connection failed: %v", err)
		}
	}
	return db.conn
}

// GetAboutMeRepository - Get about me repository
func (db *Database) GetAboutMeRepository() *AboutMeRepository {
	return NewAboutMeRepository(db)
}

// GetReflectionRepository - Get reflection repository
func (db *Database) GetReflectionRepository() *ReflectionRepository {
	return NewReflectionRepository(db)
}

// GetInteractionRepository - Get interaction repository
func (db *Database) GetInteractionRepository() *InteractionRepository {
	return NewInteractionRepository(db)
}

// GetContextConflictRepository - Get context conflict repository
func (db *Database) GetContextConflictRepository() *ContextConflictRepository {
	return NewContextConflictRepository(db)
}

// GetQuestionHistoryRepository - Get question history repository
func (db *Database) GetQuestionHistoryRepository() *QuestionHistoryRepository {
	return NewQuestionHistoryRepository(db)
}

// GetClarificationQuestionRepository - Get clarification question repository
func (db *Database) GetClarificationQuestionRepository() *ClarificationQuestionRepository {
	return NewClarificationQuestionRepository(db)
}

// GetMetricsRepository - Get metrics repository for learning analytics
func (db *Database) GetMetricsRepository() *MetricsRepository {
	return NewMetricsRepository(db)
}

// GetAuditLogRepository - Get audit log repository
func (db *Database) GetAuditLogRepository() *AuditLogRepository {
	return NewAuditLogRepository(db)
}

// GetSafetyIncidentRepository - Get safety incident repository
func (db *Database) GetSafetyIncidentRepository() *SafetyIncidentRepository {
	return NewSafetyIncidentRepository(db)
}

// GetPendingInputRepository - Get unified pending input repository
func (db *Database) GetPendingInputRepository() *PendingInputRepository {
	return NewPendingInputRepository(db)
}

// GetStructuredContextRepository - Get structured context repository
func (db *Database) GetStructuredContextRepository() *StructuredContextRepository {
	return NewStructuredContextRepository(db)
}

// Close - Close database connection
func (db *Database) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// TRACKING: Log who is closing the database
	log.Printf("[DATABASE CLOSE] WARNING: Database.Close() called! This should NOT happen during server lifetime!")
	log.Printf("[DATABASE CLOSE] Stack trace:")
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	log.Printf("%s", buf[:n])

	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}

// Exec - Execute query
func (db *Database) Exec(query string, args ...interface{}) (sql.Result, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.conn.Exec(query, args...)
}

// QueryRow - Query single row
func (db *Database) QueryRow(query string, args ...interface{}) *sql.Row {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.conn.QueryRow(query, args...)
}

// Query - Query multiple rows
func (db *Database) Query(query string, args ...interface{}) (*sql.Rows, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.conn.Query(query, args...)
}
