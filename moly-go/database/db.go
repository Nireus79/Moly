package database

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	_ "github.com/mutecomm/go-sqlcipher/v4"
)

//go:embed schema.sql
var schemaFS embed.FS

const SchemaVersion = 1

// Database - Main database connection handler
type Database struct {
	conn *sql.DB
	mu   sync.RWMutex
}

var (
	instance *Database
	once     sync.Once
)

// Init - Initialize database connection and run migrations
// If userID is provided, database is encrypted with AES-256
// If userID is empty, database is unencrypted
func Init(dbPath string, userID ...string) (*Database, error) {
	var err error
	once.Do(func() {
		if mkErr := os.MkdirAll(filepath.Dir(dbPath), 0700); mkErr != nil {
			err = fmt.Errorf("failed to create database directory: %w", mkErr)
			return
		}
		if len(userID) > 0 && userID[0] != "" {
			instance, err = initDatabaseEncrypted(dbPath, userID[0])
		} else {
			instance, err = initDatabase(dbPath)
		}
	})
	return instance, err
}

// initDatabaseEncrypted - Create encrypted connection and apply schema
func initDatabaseEncrypted(dbPath string, userID string) (*Database, error) {
	conn, err := OpenEncrypted(dbPath, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to open encrypted database: %w", err)
	}

	db := &Database{conn: conn}

	// Apply schema
	if err := db.applySchema(); err != nil {
		return nil, fmt.Errorf("failed to apply schema: %w", err)
	}

	log.Printf("[Database] Initialized at %s (encrypted)", dbPath)
	return db, nil
}

// initDatabase - Create connection and apply schema (unencrypted)
func initDatabase(dbPath string) (*Database, error) {
	// Use the proven working OpenUnencrypted function directly
	conn, err := OpenUnencrypted(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open unencrypted database: %w", err)
	}

	db := &Database{conn: conn}

	// Apply schema
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

// GetChatMessageRepository - Get chat message repository
func (db *Database) GetChatMessageRepository() *ChatMessageRepository {
	return NewChatMessageRepository(db.GetConnection())
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

// GetSuggestionChoiceRepository - Get suggestion choice repository
func (db *Database) GetSuggestionChoiceRepository() *SuggestionChoiceRepository {
	return NewSuggestionChoiceRepository(db)
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

// GetClarificationResponseRepository - Get clarification response repository
func (db *Database) GetClarificationResponseRepository() *ClarificationResponseRepository {
	return NewClarificationResponseRepository(db)
}

// GetContactRepository - Get contact repository
func (db *Database) GetContactRepository() *ContactRepository {
	return NewContactRepository(db)
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

// GetInstance - Get singleton database instance
func GetInstance() *Database {
	return instance
}

// BeginTx - Start a transaction
func (db *Database) BeginTx() (*sql.Tx, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.conn.Begin()
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

// CreateUser - Create or update user record
func (db *Database) CreateUser(userID string) error {
	// FIX #32: Validate user before save
	if userID == "" || len(userID) > 255 {
		return fmt.Errorf("userId required and must be <= 255 chars")
	}

	now := time.Now().Unix()
	query := `
		INSERT INTO users (id, created_at, last_active)
		VALUES (?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET last_active = excluded.last_active
	`
	_, err := db.Exec(query, userID, now, now)
	return err
}

// UpdateUserContextLevel - Update user's context completeness level
func (db *Database) UpdateUserContextLevel(userID string, level string) error {
	query := `UPDATE users SET context_level = ? WHERE id = ?`
	_, err := db.Exec(query, level, userID)
	return err
}

// Health - Check database health
func (db *Database) Health() (bool, string) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.conn == nil {
		return false, "database not initialized"
	}

	if err := db.conn.Ping(); err != nil {
		return false, fmt.Sprintf("ping failed: %v", err)
	}

	return true, "healthy"
}

// Transaction - Helper for running transactional code
func (db *Database) Transaction(fn func(*sql.Tx) error) error {
	log.Printf("[Database] Transaction: Beginning...")
	tx, err := db.BeginTx()
	if err != nil {
		log.Printf("[Database] Transaction: ❌ Failed to begin: %v", err)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	log.Printf("[Database] Transaction: Executing function...")
	if err := fn(tx); err != nil {
		log.Printf("[Database] Transaction: ❌ Function failed, rolling back: %v", err)
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			log.Printf("[Database] Transaction: ❌ Rollback also failed: %v", rollbackErr)
		}
		return err
	}

	log.Printf("[Database] Transaction: Committing...")
	if err := tx.Commit(); err != nil {
		log.Printf("[Database] Transaction: ❌ Commit failed: %v (type: %T)", err, err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("[Database] Transaction: ✅ Committed successfully")
	return nil
}
