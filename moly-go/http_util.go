package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"moly/database"
)

// respondJSON helper function
func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// detectPreviousMessageWasGreeting checks if the previous message in conversation history was a greeting
// getUserIDFromToken extracts userId from Bearer token (session ID)
func getUserIDFromToken(token string, db *database.Database) (string, error) {
	if token == "" {
		return "", fmt.Errorf("empty token")
	}

	// Token is stored as session ID, look it up in database
	conn := db.GetConnection()
	var userID string
	err := conn.QueryRow("SELECT user_id FROM sessions WHERE id = ? AND expires_at > ?", token, time.Now().Unix()).Scan(&userID)
	if err != nil {
		return "", fmt.Errorf("invalid token")
	}

	// UPDATE session last_used and user last_active for activity tracking
	now := time.Now().Unix()
	if _, err := conn.Exec("UPDATE sessions SET last_used = ? WHERE id = ?", now, token); err != nil {
		log.Printf("[Auth] Warning: Failed to update session last_used: %v", err)
	}
	if _, err := conn.Exec("UPDATE users SET last_active = ? WHERE id = ?", now, userID); err != nil {
		log.Printf("[Auth] Warning: Failed to update user last_active: %v", err)
	}

	return userID, nil
}

// extractAndValidateToken extracts and validates Bearer token from request
func extractAndValidateToken(r *http.Request, db *database.Database) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("Invalid token")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		return "", fmt.Errorf("Invalid token")
	}

	userID, err := getUserIDFromToken(token, db)
	if err != nil {
		return "", fmt.Errorf("Invalid token")
	}

	return userID, nil
}

func corsMiddleware(next http.Handler) http.Handler {
	inner := recoverMiddleware(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-ID")
		w.Header().Set("Access-Control-Max-Age", "86400")
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		inner.ServeHTTP(w, r)
	})
}

// recoverMiddleware answers 500 instead of dropping the connection when a handler panics.
func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[HTTP] PANIC serving %s %s: %v\n%s", r.Method, r.URL.Path, rec, debug.Stack())
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "Internal server error"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func handleStatus(db *database.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Health check - no authentication required (frontend needs to detect backend before login)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","version":"2.1"}`))
	}
}

// handleFrontendErrors - Collect and log frontend errors
func handleFrontendErrors(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req struct {
		Errors  []interface{} `json:"errors"`
		Session string        `json:"session"`
		Version string        `json:"extension_version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	log.Printf("[Frontend] Error report: session=%s, version=%s, error_count=%d", req.Session, req.Version, len(req.Errors))

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"received": true,
		"count":    len(req.Errors),
	})
}
