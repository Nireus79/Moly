package agents

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"moly/database"
)

// setupTestDatabase creates a temporary SQLite database for testing
func setupTestDatabase() *database.Database {
	// Create temporary directory for test databases
	tmpDir := filepath.Join(os.TempDir(), "moly-tests")
	os.MkdirAll(tmpDir, 0755)

	// Create unique test database file per test
	// Since Init uses sync.Once (singleton), we need to use unique paths
	// and rely on them being initialized independently
	timestamp := time.Now().UnixNano()
	random := rand.Int63()
	dbFile := filepath.Join(tmpDir, fmt.Sprintf("test-%d-%d.db", timestamp, random))

	// Make sure file doesn't exist
	os.Remove(dbFile)

	// Initialize the database - each unique path creates a unique instance
	db, err := database.Init(dbFile)
	if err != nil {
		panic(err)
	}

	return db
}

// setupTestAttributeRepo creates a repository with test database
func setupTestAttributeRepo() *database.ContextAttributeRepository {
	db := setupTestDatabase()
	return database.NewContextAttributeRepository(db)
}
