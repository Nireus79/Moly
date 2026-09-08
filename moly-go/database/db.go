package database

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
	"sync"
	"time"

	_ "github.com/mutecomm/go-sqlcipher/v4"
)

//go:embed schema.sql
var schemaFS embed.FS

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
// If userID is provided, database is encrypted with AES-256 (V2.1)
// If userID is empty, database is unencrypted (V2.0 legacy)
func Init(dbPath string, userID ...string) (*Database, error) {
	var err error
	once.Do(func() {
		if len(userID) > 0 && userID[0] != "" {
			instance, err = initDatabaseEncrypted(dbPath, userID[0])
		} else {
			instance, err = initDatabase(dbPath)
		}
	})
	return instance, err
}

// initDatabaseEncrypted - Create encrypted connection (V2.1) and apply schema
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

// initDatabase - Create connection and apply schema (V2.0 legacy, unencrypted)
func initDatabase(dbPath string) (*Database, error) {
	conn, err := sql.Open("sqlite3", dbPath+"?cache=shared&mode=rwc&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &Database{conn: conn}

	// Apply schema
	if err := db.applySchema(); err != nil {
		return nil, fmt.Errorf("failed to apply schema: %w", err)
	}

	log.Printf("[Database] Initialized at %s", dbPath)
	return db, nil
}

// applySchema - Apply schema.sql to database
func (db *Database) applySchema() error {
	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read schema: %w", err)
	}

	_, err = db.conn.Exec(string(schema))
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	log.Printf("[Database] Schema applied successfully")
	return nil
}

// GetConnection - Get underlying SQL connection
func (db *Database) GetConnection() *sql.DB {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.conn
}

// Close - Close database connection
func (db *Database) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()
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
	tx, err := db.BeginTx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
