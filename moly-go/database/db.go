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

const SchemaVersion = 7

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

// schemaV3ToV4 adds app_settings (AI provider, model and API keys of the installation).
const schemaV3ToV4 = `
CREATE TABLE IF NOT EXISTS app_settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at INTEGER NOT NULL
);
PRAGMA user_version = 4;
`

// schemaV4ToV5 rebuilds contacts so that a person who has no name yet belongs to the conversation it came from
// (origin_conversation_id) and two conversations may each have their own "boss". The old UNIQUE(user_id, name) is replaced
// by two partial unique indexes; SQLite cannot drop a table constraint, hence the rebuild. Every existing row keeps its data
// and gets no origin: an old unnamed person is then visible in no conversation (it was never shown reliably) and a named
// person is unchanged.
const schemaV4ToV5 = `
CREATE TABLE contacts_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    relationship TEXT,
    age TEXT,
    characteristics TEXT,
    first_mentioned_at INTEGER,
    created_via TEXT,
    "status" TEXT DEFAULT 'active',
    version INTEGER DEFAULT 1,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL, confidence REAL DEFAULT 0.5, extraction_count INTEGER DEFAULT 1, last_mentioned_at INTEGER, pronouns TEXT, contact_role TEXT DEFAULT NULL, involved_intentions TEXT DEFAULT NULL, past_successes TEXT DEFAULT NULL, dependencies TEXT DEFAULT NULL,
    notes TEXT,
    name_status TEXT NOT NULL DEFAULT 'named',
    origin_conversation_id TEXT NOT NULL DEFAULT '',
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
INSERT INTO contacts_new (id, user_id, name, relationship, age, characteristics, first_mentioned_at, created_via, "status", version, created_at, updated_at, confidence, extraction_count, last_mentioned_at, pronouns, contact_role, involved_intentions, past_successes, dependencies, notes, name_status)
SELECT id, user_id, name, relationship, age, characteristics, first_mentioned_at, created_via, "status", version, created_at, updated_at, confidence, extraction_count, last_mentioned_at, pronouns, contact_role, involved_intentions, past_successes, dependencies, notes, name_status FROM contacts;
DROP TABLE contacts;
ALTER TABLE contacts_new RENAME TO contacts;
CREATE UNIQUE INDEX idx_contacts_named ON contacts(user_id, name) WHERE name_status = 'named';
CREATE UNIQUE INDEX idx_contacts_unresolved ON contacts(user_id, origin_conversation_id, name) WHERE name_status != 'named';
CREATE INDEX idx_contacts_contact_role ON contacts(user_id, contact_role) WHERE contact_role IS NOT NULL;
CREATE INDEX idx_contacts_involved_intentions ON contacts(user_id) WHERE involved_intentions IS NOT NULL;
CREATE INDEX idx_contacts_name ON contacts(name);
CREATE INDEX idx_contacts_user ON contacts(user_id);
CREATE INDEX idx_contacts_user_id ON contacts(user_id);
CREATE INDEX idx_contacts_user_mentioned ON contacts(user_id, last_mentioned_at);
PRAGMA user_version = 5;
`

// schemaV5ToV6 adds the AI provider settings and API keys OF A USER. Until now they were one set for the whole installation
// (app_settings); every user now has their own and no one else can read, use or replace them. The old rows of app_settings are
// kept only until the local profile claims them (see UserAI.ClaimLegacy).
const schemaV5ToV6 = `
CREATE TABLE IF NOT EXISTS user_ai_settings (
    user_id TEXT PRIMARY KEY,
    provider TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL DEFAULT '',
    ollama_endpoint TEXT NOT NULL DEFAULT '',
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS user_ai_keys (
    user_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    value TEXT NOT NULL,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY (user_id, provider),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
PRAGMA user_version = 6;
`

// schemaV6ToV7 adds what people are to each other and to the user (a role needs its "of whom").
const schemaV6ToV7 = `
CREATE TABLE IF NOT EXISTS contact_relations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    contact_id INTEGER NOT NULL,
    relation TEXT NOT NULL,
    of_contact_id INTEGER,
    former INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (contact_id) REFERENCES contacts(id) ON DELETE CASCADE,
    FOREIGN KEY (of_contact_id) REFERENCES contacts(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_contact_relations_unique ON contact_relations(contact_id, relation, IFNULL(of_contact_id, 0));
CREATE INDEX IF NOT EXISTS idx_contact_relations_user ON contact_relations(user_id, contact_id);
PRAGMA user_version = 7;
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
		HardenDataDir(filepath.Dir(dbPath))
		key, keyErr := LoadOrCreateKey(dbPath)
		if keyErr != nil {
			err = keyErr
			return
		}
		instance, err = initWithKey(dbPath, key)
	})
	return instance, err
}

// InitWithKey is Init with a key the caller already resolved (package keystore): it never touches the keychain.
func InitWithKey(dbPath string, key []byte) (*Database, error) {
	var err error
	once.Do(func() {
		if mkErr := os.MkdirAll(filepath.Dir(dbPath), 0700); mkErr != nil {
			err = fmt.Errorf("failed to create database directory: %w", mkErr)
			return
		}
		HardenDataDir(filepath.Dir(dbPath))
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
		log.Printf("[Database] Upgraded schema from version 2 to 3")
		version = 3
	}
	if version == 3 && tables > 0 {
		if _, err := db.conn.Exec(schemaV3ToV4); err != nil {
			return fmt.Errorf("failed to upgrade schema from version 3: %w", err)
		}
		log.Printf("[Database] Upgraded schema from version 3 to 4")
		version = 4
	}
	if version == 4 && tables > 0 {
		if err := db.execInTransaction(schemaV4ToV5); err != nil {
			return fmt.Errorf("failed to upgrade schema from version 4: %w", err)
		}
		log.Printf("[Database] Upgraded schema from version 4 to 5")
		version = 5
	}
	if version == 5 && tables > 0 {
		if err := db.execInTransaction(schemaV5ToV6); err != nil {
			return fmt.Errorf("failed to upgrade schema from version 5: %w", err)
		}
		log.Printf("[Database] Upgraded schema from version 5 to 6")
		version = 6
	}
	if version == 6 && tables > 0 {
		if err := db.execInTransaction(schemaV6ToV7); err != nil {
			return fmt.Errorf("failed to upgrade schema from version 6: %w", err)
		}
		log.Printf("[Database] Upgraded schema from version 6 to %d", SchemaVersion)
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

// execInTransaction runs a script all or nothing.
func (db *Database) execInTransaction(script string) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(script); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
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

// Shutdown closes the database on purpose, at the end of the program: it folds the write-ahead log into the file and closes
// the connection without the warning that Close prints (a Close during normal operation is a bug worth shouting about).
func (db *Database) Shutdown() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.conn == nil {
		return nil
	}
	if _, err := db.conn.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		log.Printf("[Database] could not fold the write-ahead log at shutdown: %v", err)
	}
	err := db.conn.Close()
	db.conn = nil
	return err
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
