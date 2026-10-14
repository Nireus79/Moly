package database

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func aiDB(t *testing.T) *Database {
	t.Helper()
	db, err := initWithKey(filepath.Join(t.TempDir(), "ai.db"), testKey(t))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	for _, u := range []string{"u1", "u2", "user_local"} {
		db.GetConnection().Exec(`INSERT INTO users (id, created_at, last_active) VALUES (?, ?, ?)`, u, now, now)
	}
	return db
}

// Each user's settings and keys are theirs alone; nobody, and no empty id, gets any.
func TestUserAISettingsBelongToTheUser(t *testing.T) {
	r := aiDB(t).GetUserAI()
	if err := r.SaveSettings("u1", UserAISettings{Provider: "claude", Model: "m1"}); err != nil {
		t.Fatal(err)
	}
	if err := r.SaveKey("u1", "claude", "plain:secret-of-u1"); err != nil {
		t.Fatal(err)
	}
	if s, _ := r.Settings("u2"); s.Provider != "" {
		t.Fatalf("u2 sees u1's settings: %+v", s)
	}
	if _, has, _ := r.Key("u2", "claude"); has {
		t.Fatal("u2 sees u1's key")
	}
	if v, has, _ := r.Key("u1", "claude"); !has || v != "plain:secret-of-u1" {
		t.Fatalf("u1 lost their key: %q %v", v, has)
	}
	// u2 removing "their" key removes nothing of u1's
	r.DeleteKey("u2", "claude")
	if _, has, _ := r.Key("u1", "claude"); !has {
		t.Fatal("u2 removed u1's key")
	}
	for _, id := range []string{"", "   "} {
		if _, err := r.Settings(id); err == nil {
			t.Errorf("settings served for the user %q", id)
		}
		if _, _, err := r.Key(id, "claude"); err == nil {
			t.Errorf("a key served for the user %q", id)
		}
		if err := r.SaveKey(id, "claude", "plain:x"); err == nil {
			t.Errorf("a key saved for the user %q", id)
		}
		if err := r.SaveSettings(id, UserAISettings{Provider: "claude"}); err == nil {
			t.Errorf("settings saved for the user %q", id)
		}
	}
}

// The profile takes its AI settings and keys with it, and an export never contains them.
func TestUserAIKeysGoWithTheProfileAndAreNeverExported(t *testing.T) {
	db := aiDB(t)
	r := db.GetUserAI()
	r.SaveSettings("u1", UserAISettings{Provider: "claude"})
	r.SaveKey("u1", "claude", "plain:EXPORT-ME-NOT")

	data, err := ExportUserData(db.GetConnection(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	tables, _ := data["tables"].(map[string]any)
	for _, name := range []string{"user_ai_keys", "user_ai_settings"} {
		if _, found := tables[name]; found {
			t.Errorf("the export contains %s", name)
		}
	}
	if strings.Contains(strings.ToLower(toJSON(t, data)), "export-me-not") {
		t.Fatal("the key is in the export")
	}

	if err := DeleteUserData(db.GetConnection(), "u1"); err != nil {
		t.Fatal(err)
	}
	var n int
	db.GetConnection().QueryRow(`SELECT (SELECT COUNT(*) FROM user_ai_settings WHERE user_id = 'u1') + (SELECT COUNT(*) FROM user_ai_keys WHERE user_id = 'u1')`).Scan(&n)
	if n != 0 {
		t.Fatalf("%d AI rows survived the deletion of the profile", n)
	}
}

// Settings saved for the installation (before they belonged to a user) go to the one who claims them, once.
func TestClaimLegacyMovesTheInstallationSettingsOnce(t *testing.T) {
	db := aiDB(t)
	legacy := db.GetAppSettings()
	legacy.Set("llm.provider", "claude")
	legacy.Set("llm.model", "claude-sonnet-5-5")
	legacy.Set("llm.api_key.claude", "sealed:v1:abc")
	r := db.GetUserAI()

	moved, err := r.ClaimLegacy("user_local")
	if err != nil || !moved {
		t.Fatalf("claim: %v %v", moved, err)
	}
	if s, _ := r.Settings("user_local"); s.Provider != "claude" || s.Model != "claude-sonnet-5-5" {
		t.Fatalf("settings: %+v", s)
	}
	if v, has, _ := r.Key("user_local", "claude"); !has || v != "sealed:v1:abc" {
		t.Fatalf("the key moved as it was stored: %q %v", v, has)
	}
	if _, ok, _ := legacy.Get("llm.api_key.claude"); ok {
		t.Fatal("the installation's key must leave app_settings")
	}
	if moved, _ := r.ClaimLegacy("u2"); moved {
		t.Fatal("the same settings were claimed twice")
	}
	if s, _ := r.Settings("u2"); s.Provider != "" {
		t.Fatal("the other user got the settings")
	}
	// a user who already has settings keeps them: nothing is overwritten
	legacy.Set("llm.provider", "openai")
	r.SaveSettings("u1", UserAISettings{Provider: "ollama"})
	if moved, _ := r.ClaimLegacy("u1"); moved {
		t.Fatal("a user with settings of their own was overwritten")
	}
}

// Upgrade of a version 5 database: nothing is lost and the per-user tables appear.
func TestSchemaUpgradeFromVersion5AddsTheUserAITables(t *testing.T) {
	db := aiDB(t)
	c := db.GetConnection()
	if _, err := c.Exec(`DROP TABLE user_ai_keys; DROP TABLE user_ai_settings; PRAGMA user_version = 5;`); err != nil {
		t.Fatal(err)
	}
	db.GetAppSettings().Set("llm.provider", "claude")
	if err := db.applySchema(); err != nil {
		t.Fatalf("upgrade from v5: %v", err)
	}
	var version int
	c.QueryRow(`PRAGMA user_version`).Scan(&version)
	if version != SchemaVersion {
		t.Fatalf("version %d, want %d", version, SchemaVersion)
	}
	if err := db.GetUserAI().SaveSettings("u1", UserAISettings{Provider: "claude"}); err != nil {
		t.Fatalf("the new table is not usable: %v", err)
	}
	if v, ok, _ := db.GetAppSettings().Get("llm.provider"); !ok || v != "claude" {
		t.Fatal("the installation's settings were lost in the upgrade; they wait for the local profile to claim them")
	}
}

func toJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// A user's key is not readable in the database file or its log, not even in its stored (plain) form: the file is encrypted.
func TestAUsersKeyIsNotReadableInTheDatabaseFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "k.db")
	db, err := initWithKey(path, testKey(t))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	db.GetConnection().Exec(`INSERT INTO users (id, created_at, last_active) VALUES ('u1', ?, ?)`, now, now)
	secret := "sk-ant-THIS-USERS-KEY-MUST-NOT-APPEAR-IN-THE-FILE-0123456789"
	if err := db.GetUserAI().SaveKey("u1", "claude", "plain:"+secret); err != nil {
		t.Fatal(err)
	}
	db.GetConnection().Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	for _, f := range []string{path, path + "-wal"} {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		if bytes.Contains(data, []byte(secret)) || bytes.Contains(data, []byte("user_ai_keys")) {
			t.Fatalf("%s holds the key or the table name in the clear", f)
		}
	}
}
