package database

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestAppSettingsRoundTrip(t *testing.T) {
	db, err := initWithKey(filepath.Join(t.TempDir(), "s.db"), testKey(t))
	if err != nil {
		t.Fatal(err)
	}
	s := db.GetAppSettings()
	if _, ok, _ := s.Get("llm.provider"); ok {
		t.Fatal("empty database must have no settings")
	}
	if err := s.Set("llm.provider", "claude"); err != nil {
		t.Fatal(err)
	}
	s.Set("llm.provider", "openai") // overwrite
	if v, ok, _ := s.Get("llm.provider"); !ok || v != "openai" {
		t.Fatalf("got %q %v, want openai", v, ok)
	}
	s.Set("llm.provider", "  ") // empty value removes
	if _, ok, _ := s.Get("llm.provider"); ok {
		t.Fatal("an empty value must remove the key")
	}
}

// The point of keeping keys here: the file on disk must not contain them in the clear.
func TestApiKeyIsNotReadableInTheDatabaseFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.db")
	db, err := initWithKey(path, testKey(t))
	if err != nil {
		t.Fatal(err)
	}
	secret := "sk-ant-THIS-MUST-NOT-APPEAR-IN-THE-FILE-0123456789"
	if err := db.GetAppSettings().Set("llm.api_key.claude", secret); err != nil {
		t.Fatal(err)
	}
	db.GetConnection().Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	for _, f := range []string{path, path + "-wal"} {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		if bytes.Contains(data, []byte(secret)) || bytes.Contains(data, []byte("llm.api_key")) {
			t.Fatalf("%s holds the key or its name in the clear", f)
		}
	}
}

func TestSchemaV3UpgradesWithAppSettings(t *testing.T) {
	db, err := initWithKey(filepath.Join(t.TempDir(), "up.db"), testKey(t))
	if err != nil {
		t.Fatal(err)
	}
	c := db.GetConnection()
	c.Exec(`DROP TABLE app_settings`)
	c.Exec(`PRAGMA user_version = 3`)
	if err := db.applySchema(); err != nil {
		t.Fatalf("upgrade from version 3 failed: %v", err)
	}
	var version int
	c.QueryRow(`PRAGMA user_version`).Scan(&version)
	if version != SchemaVersion {
		t.Fatalf("version = %d, want %d", version, SchemaVersion)
	}
	if err := db.GetAppSettings().Set("llm.provider", "ollama"); err != nil {
		t.Fatalf("app_settings must exist after the upgrade: %v", err)
	}
}

func TestShutdownClosesCleanlyAndIsRepeatable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.db")
	db, err := initWithKey(path, testKey(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.GetAppSettings().Set("llm.provider", "ollama"); err != nil {
		t.Fatal(err)
	}
	conn := db.conn
	if err := db.Shutdown(); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if err := conn.Ping(); err == nil {
		t.Fatal("the connection must be closed after Shutdown")
	}
	if err := db.Shutdown(); err != nil {
		t.Fatalf("a second Shutdown must be harmless: %v", err)
	}
	// The data is in the file, not in a leftover write-ahead log.
	reopened, err := OpenEncrypted(path, testKey(t))
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	var v string
	if err := reopened.QueryRow(`SELECT value FROM app_settings WHERE key = 'llm.provider'`).Scan(&v); err != nil || v != "ollama" {
		t.Fatalf("data lost across shutdown: %q %v", v, err)
	}
}
