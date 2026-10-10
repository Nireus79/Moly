package main

import (
	"os"
	"path/filepath"
	"testing"
)

// testDBPath is the one database file of this package's tests. database.Init is a process-wide singleton,
// so the first caller fixes the path for the whole run; it must outlive every test, not belong to one.
var testDBPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "moly-main-test-")
	if err != nil {
		panic(err)
	}
	testDBPath = filepath.Join(dir, "test.db")
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
