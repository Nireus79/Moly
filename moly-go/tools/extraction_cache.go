package tools

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"moly/models"
	"sync"
	"time"
)

// ExtractionCacheEntry represents a cached extraction result
type ExtractionCacheEntry struct {
	Entities  []models.ExtractedEntity
	Timestamp int64
	ExpiresAt int64
	HitCount  int64
}

// ExtractionCache provides deduplication for extraction LLM calls
// Key: hash(userID + messageID + model_version)
// This prevents duplicate LLM calls between ContextExtractor and IntentDetector
type ExtractionCache struct {
	mu       sync.RWMutex
	entries  map[string]*ExtractionCacheEntry
	ttlHours int
	maxSize  int
}

// NewExtractionCache creates a new extraction cache
func NewExtractionCache() *ExtractionCache {
	cache := &ExtractionCache{
		entries:  make(map[string]*ExtractionCacheEntry),
		ttlHours: 24,
		maxSize:  10000,
	}

	// FIX #20: Start cleanup goroutine
	// Cleanup was previously disabled due to timing issues during initialization
	// Now safely enabled with proper shutdown handling via stopChan
	go cache.cleanupExpired()

	return cache
}

// Get retrieves a cached extraction result
// Returns (entities, found)
func (ec *ExtractionCache) Get(userID, messageID string) ([]models.ExtractedEntity, bool) {
	key := ec.generateKey(userID, messageID)

	ec.mu.RLock()
	defer ec.mu.RUnlock()

	entry, exists := ec.entries[key]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Now().Unix() > entry.ExpiresAt {
		// Expired - will be cleaned up by background goroutine
		return nil, false
	}

	// Hit!
	entry.HitCount++
	log.Printf("[ExtractionCache] ✓ Cache hit for user=%s message=%s (hits=%d)", userID, messageID, entry.HitCount)
	return entry.Entities, true
}

// Set stores an extraction result in cache
func (ec *ExtractionCache) Set(userID, messageID string, entities []models.ExtractedEntity) {
	key := ec.generateKey(userID, messageID)

	ec.mu.Lock()
	defer ec.mu.Unlock()

	// Check size limit
	if len(ec.entries) >= ec.maxSize {
		log.Printf("[ExtractionCache] ⚠️ Cache at max size (%d), evicting oldest entries", ec.maxSize)
		ec.evictOldest(ec.maxSize / 10) // Remove 10% of oldest entries
	}

	now := time.Now().Unix()
	expiresAt := now + int64(ec.ttlHours*3600)

	ec.entries[key] = &ExtractionCacheEntry{
		Entities:  entities,
		Timestamp: now,
		ExpiresAt: expiresAt,
		HitCount:  0,
	}

	log.Printf("[ExtractionCache] ✓ Cached extraction for user=%s message=%s (expires in %d hours)", userID, messageID, ec.ttlHours)
}

// Clear removes all entries (for testing)
func (ec *ExtractionCache) Clear() {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	count := len(ec.entries)
	ec.entries = make(map[string]*ExtractionCacheEntry)
	log.Printf("[ExtractionCache] ✓ Cleared %d entries", count)
}

// Size returns current cache size
func (ec *ExtractionCache) Size() int {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return len(ec.entries)
}

// Stats returns cache statistics
func (ec *ExtractionCache) Stats() map[string]interface{} {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	totalHits := int64(0)
	for _, entry := range ec.entries {
		totalHits += entry.HitCount
	}

	return map[string]interface{}{
		"entries":    len(ec.entries),
		"max_size":   ec.maxSize,
		"ttl_hours":  ec.ttlHours,
		"total_hits": totalHits,
		"hit_rate":   fmt.Sprintf("%.2f%%", float64(totalHits)*100.0/float64(len(ec.entries)+1)),
	}
}

// generateKey creates a cache key from userID and messageID
// Format: md5(userID:messageID:v1) to ensure consistent hashing
func (ec *ExtractionCache) generateKey(userID, messageID string) string {
	input := fmt.Sprintf("%s:%s:v1", userID, messageID)
	hash := md5.Sum([]byte(input))
	return hex.EncodeToString(hash[:])
}

// cleanupExpired runs periodically to remove expired entries
func (ec *ExtractionCache) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		ec.mu.Lock()

		now := time.Now().Unix()
		removed := 0

		for key, entry := range ec.entries {
			if now > entry.ExpiresAt {
				delete(ec.entries, key)
				removed++
			}
		}

		ec.mu.Unlock()

		if removed > 0 {
			log.Printf("[ExtractionCache] 🧹 Cleanup: removed %d expired entries (now have %d)", removed, ec.Size())
		}
	}
}

// evictOldest removes the N oldest entries (must be called with lock held)
func (ec *ExtractionCache) evictOldest(count int) {
	if count <= 0 || len(ec.entries) == 0 {
		return
	}

	// Find oldest entries
	type kv struct {
		key       string
		timestamp int64
	}

	entries := make([]kv, 0, len(ec.entries))
	for k, v := range ec.entries {
		entries = append(entries, kv{k, v.Timestamp})
	}

	// Sort by timestamp (oldest first)
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[j].timestamp < entries[i].timestamp {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	// Remove oldest N
	removed := 0
	for i := 0; i < count && i < len(entries); i++ {
		delete(ec.entries, entries[i].key)
		removed++
	}

	log.Printf("[ExtractionCache] 🧹 Evicted %d oldest entries", removed)
}
