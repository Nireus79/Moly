package agents

import (
	"fmt"
	"log"
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

// WasRecentlyAsked checks if a conflict question was asked within the last 24 hours
// Returns true if question should be skipped
func (ch *ClarificationHistory) WasRecentlyAsked(
	userID string,
	conversationID string,
	entityValue string,
	conflictType string,
) (bool, error) {

	cacheKey := fmt.Sprintf("%s:%s:%s:%s", userID, conversationID, entityValue, conflictType)

	// Check in-memory cache first (24 hour TTL)
	ch.mu.RLock()
	if cachedTime, exists := ch.cache[cacheKey]; exists {
		ch.mu.RUnlock()
		if time.Since(cachedTime) < 24*time.Hour {
			log.Printf("[ClarificationHistory] Cache hit: Already asked about %s:%s",
				entityValue, conflictType)
			return true, nil
		}
	}
	ch.mu.RUnlock()

	// Check database for similar recent questions
	// Query clarification questions for this user from last 24 hours
	repo := database.NewClarificationQuestionRepository(ch.db)
	recentQuestions, err := repo.GetPendingQuestions(userID)
	if err != nil {
		log.Printf("[ClarificationHistory] Warning: Failed to query recent questions: %v", err)
		return false, err
	}

	// Check if any recent question is about the same conflict
	now := time.Now().Unix()
	for _, q := range recentQuestions {
		// Only check questions from last 24 hours
		if (now - q.CreatedAt) > (24 * 3600) {
			continue
		}

		// Check if this question is about the same entity/conflict type
		if ch.matchesConflict(q, entityValue, conflictType) {
			log.Printf("[ClarificationHistory] Database hit: Already asked about %s:%s",
				entityValue, conflictType)

			// Update cache
			ch.mu.Lock()
			ch.cache[cacheKey] = time.Now()
			ch.mu.Unlock()

			return true, nil
		}
	}

	// Not found - this conflict hasn't been asked about recently
	return false, nil
}

// RecordAsked records that we asked about a specific conflict
func (ch *ClarificationHistory) RecordAsked(
	questionID string,
	userID string,
	conversationID string,
	entityValue string,
	conflictType string,
) error {

	cacheKey := fmt.Sprintf("%s:%s:%s:%s", userID, conversationID, entityValue, conflictType)

	// Update in-memory cache
	ch.mu.Lock()
	ch.cache[cacheKey] = time.Now()
	ch.mu.Unlock()

	log.Printf("[ClarificationHistory] Recorded: Asked about %s:%s (Q:%s)",
		entityValue, conflictType, questionID)

	return nil
}

// matchesConflict checks if a question matches the given conflict type and entity
func (ch *ClarificationHistory) matchesConflict(
	question *database.ClarificationQuestion,
	entityValue string,
	conflictType string,
) bool {

	// Check if the question text mentions the entity
	if len(question.QuestionText) > 0 {
		// Simple check: does the question text contain the entity value?
		for _, linkedFact := range question.LinkedFacts {
			if linkedFact == fmt.Sprintf("extracted:%s", entityValue) {
				return true
			}
		}
	}

	return false
}

// CleanupOldRecords removes cached entries older than 24 hours
// Should be called periodically to prevent memory bloat
func (ch *ClarificationHistory) CleanupOldRecords() {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	now := time.Now()
	removed := 0

	for key, timestamp := range ch.cache {
		if now.Sub(timestamp) > 24*time.Hour {
			delete(ch.cache, key)
			removed++
		}
	}

	if removed > 0 {
		log.Printf("[ClarificationHistory] Cleanup: Removed %d old cache entries", removed)
	}
}

// GetCacheSize returns current cache size (for monitoring)
func (ch *ClarificationHistory) GetCacheSize() int {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	return len(ch.cache)
}
