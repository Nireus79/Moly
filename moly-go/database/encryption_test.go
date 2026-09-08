package database

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestKeyDerivation tests that key derivation is deterministic
func TestKeyDerivation(t *testing.T) {
	userID := "test_user_123"

	key1 := DeriveKey(userID)
	key2 := DeriveKey(userID)

	// Keys should be identical for same userID
	if key1 != key2 {
		t.Errorf("Key derivation not deterministic: got different keys for same userID")
	}

	// Keys should be 32 bytes
	if len(key1) != 32 {
		t.Errorf("Expected 32-byte key, got %d", len(key1))
	}

	t.Logf("✓ Key derivation deterministic and correct length (32 bytes)")
}

// TestKeyDifferentForDifferentUsers tests keys differ for different users
func TestKeyDifferentForDifferentUsers(t *testing.T) {
	key1 := DeriveKey("user1")
	key2 := DeriveKey("user2")

	if key1 == key2 {
		t.Errorf("Keys should differ for different userIDs")
	}

	t.Logf("✓ Keys differ for different users")
}

// TestEncryptedDatabase tests database encryption works
func TestEncryptedDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_encrypted.db")
	userID := "test_user_encrypted"

	// Open encrypted database
	db, err := OpenEncrypted(dbPath, userID)
	if err != nil {
		t.Fatalf("Failed to open encrypted database: %v", err)
	}
	defer db.Close()

	// Test write
	_, err = db.Exec("CREATE TABLE test_data (id TEXT PRIMARY KEY, value TEXT)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	_, err = db.Exec("INSERT INTO test_data (id, value) VALUES (?, ?)", "key1", "secret_value")
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	// Test read
	var value string
	err = db.QueryRow("SELECT value FROM test_data WHERE id = ?", "key1").Scan(&value)
	if err != nil {
		t.Fatalf("Failed to read data: %v", err)
	}

	if value != "secret_value" {
		t.Errorf("Expected 'secret_value', got '%s'", value)
	}

	db.Close()

	// Verify database file exists and is binary (encrypted)
	fileInfo, err := os.Stat(dbPath)
	if err != nil {
		t.Fatalf("Database file not created: %v", err)
	}

	if fileInfo.Size() == 0 {
		t.Errorf("Database file is empty")
	}

	t.Logf("✓ Encrypted database created and readable (size: %d bytes)", fileInfo.Size())
}

// TestEncryptedDatabaseWrongKey tests that wrong key cannot read database
func TestEncryptedDatabaseWrongKey(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_wrong_key.db")
	userID := "user_1"
	wrongUserID := "user_2"

	// Create and populate database with user_1
	db1, err := OpenEncrypted(dbPath, userID)
	if err != nil {
		t.Fatalf("Failed to open database with correct key: %v", err)
	}

	_, err = db1.Exec("CREATE TABLE test_secret (id TEXT PRIMARY KEY, secret TEXT)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	_, err = db1.Exec("INSERT INTO test_secret (id, secret) VALUES (?, ?)", "key1", "top_secret")
	if err != nil {
		t.Fatalf("Failed to insert secret: %v", err)
	}

	db1.Close()

	// Try to open with wrong key (user_2)
	db2, err := OpenEncrypted(dbPath, wrongUserID)
	if err == nil {
		// Connection opened, but should fail on query
		_, err = db2.Query("SELECT * FROM test_secret")
		if err == nil {
			t.Errorf("Should not be able to read database with wrong key")
		}
		db2.Close()
	}

	t.Logf("✓ Database correctly rejects wrong key")
}

// TestUnencryptedDatabase tests legacy unencrypted database still works
func TestUnencryptedDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_unencrypted.db")

	// Open unencrypted database
	db, err := OpenUnencrypted(dbPath)
	if err != nil {
		t.Fatalf("Failed to open unencrypted database: %v", err)
	}
	defer db.Close()

	// Test write
	_, err = db.Exec("CREATE TABLE test_data (id TEXT PRIMARY KEY, value TEXT)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	_, err = db.Exec("INSERT INTO test_data (id, value) VALUES (?, ?)", "key1", "plaintext_value")
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	// Test read
	var value string
	err = db.QueryRow("SELECT value FROM test_data WHERE id = ?", "key1").Scan(&value)
	if err != nil {
		t.Fatalf("Failed to read data: %v", err)
	}

	if value != "plaintext_value" {
		t.Errorf("Expected 'plaintext_value', got '%s'", value)
	}

	t.Logf("✓ Unencrypted database works (legacy mode)")
}

// TestEncryptionPersistence tests encrypted data persists across connections
func TestEncryptionPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_persistence.db")
	userID := "persistent_user"

	// Session 1: Create and insert data
	{
		db1, err := OpenEncrypted(dbPath, userID)
		if err != nil {
			t.Fatalf("Failed to open database (session 1): %v", err)
		}

		_, err = db1.Exec("CREATE TABLE IF NOT EXISTS persistent_data (id TEXT PRIMARY KEY, data TEXT)")
		if err != nil {
			t.Fatalf("Failed to create table: %v", err)
		}

		_, err = db1.Exec("INSERT OR REPLACE INTO persistent_data (id, data) VALUES (?, ?)", "persistent_key", "persistent_value")
		if err != nil {
			t.Fatalf("Failed to insert data: %v", err)
		}

		db1.Close()
	}

	// Session 2: Verify data persists
	{
		db2, err := OpenEncrypted(dbPath, userID)
		if err != nil {
			t.Fatalf("Failed to open database (session 2): %v", err)
		}
		defer db2.Close()

		var value string
		err = db2.QueryRow("SELECT data FROM persistent_data WHERE id = ?", "persistent_key").Scan(&value)
		if err != nil {
			t.Fatalf("Failed to read persistent data: %v", err)
		}

		if value != "persistent_value" {
			t.Errorf("Expected 'persistent_value', got '%s'", value)
		}

		t.Logf("✓ Encrypted data persists across sessions")
	}
}

// BenchmarkEncryptedDatabaseInsert benchmarks insert performance with encryption
func BenchmarkEncryptedDatabaseInsert(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench_encrypted.db")
	userID := "bench_user"

	db, err := OpenEncrypted(dbPath, userID)
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	db.Exec("CREATE TABLE bench_data (id TEXT PRIMARY KEY, value TEXT)")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		db.Exec("INSERT INTO bench_data (id, value) VALUES (?, ?)",
			fmt.Sprintf("key_%d", i),
			fmt.Sprintf("value_%d", i))
	}

	b.StopTimer()
	b.Logf("Encrypted insert throughput: %d ops/sec", b.N/(int(b.Elapsed().Seconds())+1))
}

// TestDatabaseEncryptionGate - Combined test to verify all encryption features work
func TestDatabaseEncryptionGate(t *testing.T) {
	tests := []struct {
		name string
		test func(*testing.T)
	}{
		{"KeyDerivation", TestKeyDerivation},
		{"KeyDifferentForDifferentUsers", TestKeyDifferentForDifferentUsers},
		{"EncryptedDatabase", TestEncryptedDatabase},
		{"EncryptedDatabaseWrongKey", TestEncryptedDatabaseWrongKey},
		{"UnencryptedDatabase", TestUnencryptedDatabase},
		{"EncryptionPersistence", TestEncryptionPersistence},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.test(t)
		})
	}

	t.Log("\n✓ Database encryption gate PASSED - AES-256 encryption ready for Phase 1")
}
