package database

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

// AppSettings stores the settings of the installation (not of a user) in the encrypted database: the AI provider, the
// model and the API keys.
type AppSettings struct{ db *Database }

func (db *Database) GetAppSettings() *AppSettings { return &AppSettings{db: db} }

// Get returns the value of key, and false when it is not set.
func (s *AppSettings) Get(key string) (string, bool, error) {
	var v string
	err := s.db.GetConnection().QueryRow(`SELECT value FROM app_settings WHERE key = ?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return v, true, nil
}

// Set stores value under key; an empty value removes the key.
func (s *AppSettings) Set(key, value string) error {
	if strings.TrimSpace(value) == "" {
		return s.Delete(key)
	}
	_, err := s.db.GetConnection().Exec(
		`INSERT INTO app_settings (key, value, updated_at) VALUES (?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, value, time.Now().Unix())
	return err
}

func (s *AppSettings) Delete(key string) error {
	_, err := s.db.GetConnection().Exec(`DELETE FROM app_settings WHERE key = ?`, key)
	return err
}
