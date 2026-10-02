package database

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	_ "github.com/mutecomm/go-sqlcipher/v4"
)

//go:embed schema.sql
var schemaFS embed.FS

//go:embed migrations/*.sql
var migrationsFS embed.FS

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

// applySchema - Apply schema.sql to database
func (db *Database) applySchema() error {
	// Enable foreign key constraints (required for schema integrity)
	_, err := db.conn.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		log.Printf("[Database] WARNING: Failed to enable foreign keys: %v", err)
	}

	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read schema: %w", err)
	}

	_, err = db.conn.Exec(string(schema))
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	// Apply schema migrations (handle errors gracefully for optional columns)
	db.applyMigrations()

	log.Printf("[Database] Schema applied successfully (foreign keys enabled)")
	return nil
}

// applyMigrations - Apply all migrations from migrations/ directory in order
func (db *Database) applyMigrations() {
	// First ensure migrations_applied tracking table exists
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS migrations_applied (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			migration_name TEXT UNIQUE NOT NULL,
			applied_at INTEGER NOT NULL
		)
	`)
	if err != nil {
		log.Printf("[Database] WARNING: Failed to create migrations_applied table: %v", err)
		return
	}

	// Read migration files from embedded directory
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		log.Printf("[Database] WARNING: Failed to read migrations directory: %v", err)
		return
	}

	// Sort and execute migration files in order
	for _, entry := range entries {
		if !entry.IsDir() && len(entry.Name()) > 4 && entry.Name()[len(entry.Name())-4:] == ".sql" {
			migrationName := entry.Name()

			// Check if migration already applied
			var count int
			err := db.conn.QueryRow("SELECT COUNT(*) FROM migrations_applied WHERE migration_name = ?", migrationName).Scan(&count)
			if err == nil && count > 0 {
				log.Printf("[Database] Migration already applied: %s", migrationName)
				continue
			}

			// Read and execute migration
			content, err := migrationsFS.ReadFile("migrations/" + migrationName)
			if err != nil {
				log.Printf("[Database] WARNING: Failed to read migration file %s: %v", migrationName, err)
				continue
			}

			// Execute migration (split by semicolon for multiple statements)
			_, err = db.conn.Exec(string(content))
			if err != nil {
				// Some migrations may fail if already applied (e.g., CREATE TABLE IF NOT EXISTS)
				// Log as warning but continue
				log.Printf("[Database] Migration %s: %v (may already be applied)", migrationName, err)
			} else {
				log.Printf("[Database] ✓ Applied migration: %s", migrationName)
			}

			// Mark migration as applied
			_, err = db.conn.Exec("INSERT OR IGNORE INTO migrations_applied (migration_name, applied_at) VALUES (?, ?)", migrationName, time.Now().Unix())
			if err != nil {
				log.Printf("[Database] WARNING: Failed to mark migration %s as applied: %v", migrationName, err)
			}
		}
	}

	// Also run legacy hardcoded migrations for backward compatibility
	legacyMigrations := []string{
		"ALTER TABLE reflections ADD COLUMN contact_id TEXT",
		"ALTER TABLE reflections ADD COLUMN message_id TEXT",
		"ALTER TABLE reflections ADD COLUMN extracted_style TEXT",
		"ALTER TABLE reflections ADD COLUMN extracted_intention TEXT",
		"ALTER TABLE reflections ADD COLUMN conversation_id TEXT",
		"ALTER TABLE reflections ADD COLUMN communication_preferences TEXT",
		"ALTER TABLE reflections ADD COLUMN user_quotes TEXT",
		"ALTER TABLE reflections ADD COLUMN user_edits TEXT",
		"ALTER TABLE chat_messages ADD COLUMN metadata TEXT",
	}

	for _, migration := range legacyMigrations {
		_, err := db.conn.Exec(migration)
		if err != nil {
			if !contains(err.Error(), "duplicate column") && !contains(err.Error(), "already exists") {
				log.Printf("[Database] Legacy migration optional (may already exist): %v", err)
			}
		}
	}
}

// contains - Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findSubstring(s, substr)))
}

// findSubstring - Helper to find substring
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
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
