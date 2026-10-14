package database

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// UserAI stores the AI provider, the model and the API keys of ONE USER. It is the only place that reads or writes them.
// Every method takes the user id and touches only that user's rows; an empty id is refused, so a call that lost track of its
// user cannot fall back to anyone's settings. A key is kept exactly as package secrets wrote it ("plain:" or "sealed:") in the
// encrypted database. It is never exported (excludedFromExport) and goes with the profile (foreign key, ON DELETE CASCADE).
type UserAI struct{ db *Database }

func (db *Database) GetUserAI() *UserAI { return &UserAI{db: db} }

// UserAISettings are the choices of a user, without any key.
type UserAISettings struct {
	Provider       string
	Model          string
	OllamaEndpoint string
}

var errNoUser = errors.New("no user: the AI settings of nobody are not served")

func needUser(userID string) error {
	if strings.TrimSpace(userID) == "" {
		return errNoUser
	}
	return nil
}

// Settings returns the user's choices; the zero value when the user has made none.
func (r *UserAI) Settings(userID string) (UserAISettings, error) {
	if err := needUser(userID); err != nil {
		return UserAISettings{}, err
	}
	var s UserAISettings
	err := r.db.GetConnection().QueryRow(
		`SELECT provider, model, ollama_endpoint FROM user_ai_settings WHERE user_id = ?`, userID,
	).Scan(&s.Provider, &s.Model, &s.OllamaEndpoint)
	if errors.Is(err, sql.ErrNoRows) {
		return UserAISettings{}, nil
	}
	return s, err
}

// SaveSettings stores the user's choices.
func (r *UserAI) SaveSettings(userID string, s UserAISettings) error {
	if err := needUser(userID); err != nil {
		return err
	}
	_, err := r.db.GetConnection().Exec(
		`INSERT INTO user_ai_settings (user_id, provider, model, ollama_endpoint, updated_at) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(user_id) DO UPDATE SET provider = excluded.provider, model = excluded.model,
		   ollama_endpoint = excluded.ollama_endpoint, updated_at = excluded.updated_at`,
		userID, s.Provider, s.Model, s.OllamaEndpoint, time.Now().Unix())
	return err
}

// Key returns the stored form of the user's key for a provider ("plain:..." or "sealed:..."), and false when there is none.
func (r *UserAI) Key(userID, provider string) (string, bool, error) {
	if err := needUser(userID); err != nil {
		return "", false, err
	}
	var v string
	err := r.db.GetConnection().QueryRow(`SELECT value FROM user_ai_keys WHERE user_id = ? AND provider = ?`, userID, provider).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return v, true, nil
}

// SaveKey stores the stored form of the user's key for a provider.
func (r *UserAI) SaveKey(userID, provider, stored string) error {
	if err := needUser(userID); err != nil {
		return err
	}
	if strings.TrimSpace(provider) == "" || strings.TrimSpace(stored) == "" {
		return fmt.Errorf("a key needs a provider and a value")
	}
	_, err := r.db.GetConnection().Exec(
		`INSERT INTO user_ai_keys (user_id, provider, value, updated_at) VALUES (?, ?, ?, ?)
		 ON CONFLICT(user_id, provider) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		userID, provider, stored, time.Now().Unix())
	return err
}

// DeleteKey removes the user's key for a provider.
func (r *UserAI) DeleteKey(userID, provider string) error {
	if err := needUser(userID); err != nil {
		return err
	}
	_, err := r.db.GetConnection().Exec(`DELETE FROM user_ai_keys WHERE user_id = ? AND provider = ?`, userID, provider)
	return err
}

// legacy keys of app_settings, written before the settings belonged to a user
var legacyKeys = []string{"llm.provider", "llm.model", "llm.ollama_endpoint", "llm.api_key.claude", "llm.api_key.openai"}

// ClaimLegacy gives the user the installation-wide settings that older versions saved in app_settings, once, if the user has
// no settings of their own. The caller decides WHO may claim (only the local profile, which is the one person of an installed
// Moly); the rows are removed from app_settings in the same transaction, so they cannot be claimed twice or by anyone else.
// It reports whether anything was moved.
func (r *UserAI) ClaimLegacy(userID string) (bool, error) {
	if err := needUser(userID); err != nil {
		return false, err
	}
	conn := r.db.GetConnection()
	tx, err := conn.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	var has int
	if err := tx.QueryRow(`SELECT (SELECT COUNT(*) FROM user_ai_settings WHERE user_id = ?) + (SELECT COUNT(*) FROM user_ai_keys WHERE user_id = ?)`, userID, userID).Scan(&has); err != nil {
		return false, err
	}
	if has > 0 {
		return false, nil
	}
	old := map[string]string{}
	for _, k := range legacyKeys {
		var v string
		switch err := tx.QueryRow(`SELECT value FROM app_settings WHERE key = ?`, k).Scan(&v); {
		case errors.Is(err, sql.ErrNoRows):
		case err != nil:
			return false, err
		default:
			old[k] = v
		}
	}
	if len(old) == 0 {
		return false, nil
	}
	now := time.Now().Unix()
	if old["llm.provider"] != "" || old["llm.model"] != "" || old["llm.ollama_endpoint"] != "" {
		if _, err := tx.Exec(`INSERT INTO user_ai_settings (user_id, provider, model, ollama_endpoint, updated_at) VALUES (?, ?, ?, ?, ?)`,
			userID, old["llm.provider"], old["llm.model"], old["llm.ollama_endpoint"], now); err != nil {
			return false, err
		}
	}
	for _, p := range []string{"claude", "openai"} {
		if v := old["llm.api_key."+p]; v != "" {
			if _, err := tx.Exec(`INSERT INTO user_ai_keys (user_id, provider, value, updated_at) VALUES (?, ?, ?, ?)`, userID, p, v, now); err != nil {
				return false, err
			}
		}
	}
	for _, k := range legacyKeys {
		if _, err := tx.Exec(`DELETE FROM app_settings WHERE key = ?`, k); err != nil {
			return false, err
		}
	}
	return true, tx.Commit()
}
