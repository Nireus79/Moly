package database

import (
	"bytes"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/zalando/go-keyring"
)

func testKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, keyBytes)
	for i := range key {
		key[i] = byte(i + 1)
	}
	return key
}

func TestOpenEncryptedRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "enc.db")
	key := testKey(t)

	db, err := OpenEncrypted(path, key)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("CREATE TABLE t(x TEXT); INSERT INTO t VALUES ('PLAINTEXT_MARKER')"); err != nil {
		t.Fatal(err)
	}
	db.Close()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("PLAINTEXT_MARKER")) || bytes.HasPrefix(raw, []byte("SQLite format 3")) {
		t.Fatal("database file is not encrypted")
	}

	db, err = OpenEncrypted(path, key)
	if err != nil {
		t.Fatalf("reopen with correct key: %v", err)
	}
	var x string
	if err := db.QueryRow("SELECT x FROM t").Scan(&x); err != nil || x != "PLAINTEXT_MARKER" {
		t.Fatalf("read back: %v %q", err, x)
	}
	db.Close()

	wrong := testKey(t)
	wrong[0] ^= 0xff
	if _, err := OpenEncrypted(path, wrong); err == nil {
		t.Fatal("opened with wrong key")
	}
}

func TestLoadOrCreateKeyUsesKeychain(t *testing.T) {
	keyring.MockInit()
	t.Setenv(dbKeyEnv, "")
	path := filepath.Join(t.TempDir(), "k.db")

	first, err := LoadOrCreateKey(path)
	if err != nil {
		t.Fatal(err)
	}
	second, err := LoadOrCreateKey(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) || len(first) != keyBytes {
		t.Fatal("key not stable across loads")
	}
	if _, err := os.Stat(path + ".key"); !os.IsNotExist(err) {
		t.Fatal("key must not be written to a file")
	}
}

func TestLoadOrCreateKeyEnvOverride(t *testing.T) {
	keyring.MockInit()
	want := testKey(t)
	t.Setenv(dbKeyEnv, hex.EncodeToString(want))

	got, err := LoadOrCreateKey(filepath.Join(t.TempDir(), "e.db"))
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("env key not used: %v", err)
	}

	t.Setenv(dbKeyEnv, "not-hex")
	if _, err := LoadOrCreateKey(filepath.Join(t.TempDir(), "e2.db")); err == nil {
		t.Fatal("invalid env key accepted")
	}
}

func TestLoadOrCreateKeyFailsWithoutKeychain(t *testing.T) {
	keyring.MockInitWithError(keyring.ErrUnsupportedPlatform)
	t.Setenv(dbKeyEnv, "")
	if _, err := LoadOrCreateKey(filepath.Join(t.TempDir(), "nk.db")); err == nil {
		t.Fatal("expected error when keychain is unavailable")
	}
}
