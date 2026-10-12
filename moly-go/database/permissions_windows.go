//go:build windows

package database

// HardenDataDir does nothing on Windows: the user profile directory is private to the account by default.
func HardenDataDir(dir string) {}
