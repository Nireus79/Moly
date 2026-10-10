package agents

import (
	"sync"
	"time"

	"moly/database"
)

// ClarificationHistory tracks which clarification questions have been asked
// to prevent asking the same question twice (deduplication)
type ClarificationHistory struct {
	db *database.Database
	// In-memory cache to avoid database hits for recent queries
	// Key: "userID:conversationID:entityValue:conflictType"
	cache map[string]time.Time
	mu    sync.RWMutex
}

// NewClarificationHistory creates a new clarification history tracker
func NewClarificationHistory(db *database.Database) *ClarificationHistory {
	return &ClarificationHistory{
		db:    db,
		cache: make(map[string]time.Time),
	}
}
