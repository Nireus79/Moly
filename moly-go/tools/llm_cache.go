package tools

import (
	"crypto/md5"
	"fmt"
	"sync"
	"time"
)

// CacheEntry represents a cached LLM result
type CacheEntry struct {
	Key       string        // Hash of input
	Input     string        // Original input (for debugging)
	Output    string        // LLM result
	Type      string        // "extraction", "intent", "topic", "analysis"
	Timestamp time.Time     // When cached
	Expires   time.Time     // When to evict
	HitCount  int           // Number of times accessed
}

// LLMCache provides in-memory caching for LLM results
// Prevents redundant LLM calls within a conversation
type LLMCache struct {
	entries map[string]*CacheEntry
	mu      sync.RWMutex
	ttl     time.Duration // Time to live for cache entries
	maxSize int           // Maximum number of entries before eviction
}

// NewLLMCache creates a new LLM result cache
// ttl: how long to keep entries (default 24h for session)
// maxSize: maximum entries before oldest are evicted
func NewLLMCache(ttl time.Duration, maxSize int) *LLMCache {
	if ttl == 0 {
		ttl = 24 * time.Hour // Default: 24 hour session lifetime
	}
	if maxSize == 0 {
		maxSize = 10000 // Default: 10k entries
	}

	cache := &LLMCache{
		entries: make(map[string]*CacheEntry),
		ttl:     ttl,
		maxSize: maxSize,
	}

	// FIX #20: Start cleanup goroutine
	// Cleanup was previously disabled due to timing issues during initialization
	// Now safely enabled with proper shutdown handling via stopChan
	go cache.cleanupExpired()

	return cache
}

// NewDefaultLLMCache creates cache with sensible defaults
func NewDefaultLLMCache() *LLMCache {
	return NewLLMCache(24*time.Hour, 10000)
}

// hash creates a deterministic key from input
func (lc *LLMCache) hash(input string, cacheType string) string {
	h := md5.Sum([]byte(input + "|" + cacheType))
	return fmt.Sprintf("%x", h)
}

// Set stores an LLM result in the cache
func (lc *LLMCache) Set(input, output, cacheType string) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	key := lc.hash(input, cacheType)
	now := time.Now()

	lc.entries[key] = &CacheEntry{
		Key:       key,
		Input:     input,
		Output:    output,
		Type:      cacheType,
		Timestamp: now,
		Expires:   now.Add(lc.ttl),
		HitCount:  0,
	}

	// Simple eviction: if we exceed max size, remove oldest entries
	if len(lc.entries) > lc.maxSize {
		lc.evictOldest(lc.maxSize / 10) // Remove 10% of oldest entries
	}
}

// Get retrieves a cached result, if it exists and hasn't expired
func (lc *LLMCache) Get(input, cacheType string) (string, bool) {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	key := lc.hash(input, cacheType)
	entry, exists := lc.entries[key]

	if !exists {
		return "", false
	}

	// Check expiration
	if time.Now().After(entry.Expires) {
		return "", false
	}

	// Increment hit count
	entry.HitCount++

	return entry.Output, true
}

// Has checks if a key is cached without retrieving it
func (lc *LLMCache) Has(input, cacheType string) bool {
	_, found := lc.Get(input, cacheType)
	return found
}

// evictOldest removes the oldest N entries
func (lc *LLMCache) evictOldest(count int) {
	if count <= 0 {
		count = 1
	}

	// Find oldest entries by timestamp
	type kv struct {
		key    string
		entry  *CacheEntry
		age    time.Duration
	}

	var entries []kv
	now := time.Now()

	for k, v := range lc.entries {
		entries = append(entries, kv{
			key:   k,
			entry: v,
			age:   now.Sub(v.Timestamp),
		})
	}

	// Sort by age (oldest first)
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].age < entries[j].age {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	// Remove oldest count entries
	if count > len(entries) {
		count = len(entries)
	}

	for i := 0; i < count; i++ {
		delete(lc.entries, entries[i].key)
	}
}

// Clear removes all entries
func (lc *LLMCache) Clear() {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.entries = make(map[string]*CacheEntry)
}

// ClearByType removes all entries of a specific type
func (lc *LLMCache) ClearByType(cacheType string) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	for key, entry := range lc.entries {
		if entry.Type == cacheType {
			delete(lc.entries, key)
		}
	}
}

// Size returns the number of cached entries
func (lc *LLMCache) Size() int {
	lc.mu.RLock()
	defer lc.mu.RUnlock()
	return len(lc.entries)
}

// Stats returns cache statistics
func (lc *LLMCache) Stats() map[string]interface{} {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	stats := map[string]interface{}{
		"total_entries":      len(lc.entries),
		"cache_ttl_hours":    lc.ttl.Hours(),
		"max_size":           lc.maxSize,
	}

	// Count by type
	typeCount := make(map[string]int)
	totalHits := 0

	for _, entry := range lc.entries {
		typeCount[entry.Type]++
		totalHits += entry.HitCount
	}

	stats["by_type"] = typeCount
	stats["total_hits"] = totalHits

	return stats
}

// cleanupExpired runs periodically to remove expired entries
func (lc *LLMCache) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		lc.mu.Lock()

		now := time.Now()
		for key, entry := range lc.entries {
			if now.After(entry.Expires) {
				delete(lc.entries, key)
			}
		}

		lc.mu.Unlock()
	}
}

// CacheStats holds detailed statistics about cache performance
type CacheStats struct {
	HitRate      float64           // Percentage of lookups that hit
	AvgAge       time.Duration     // Average age of cached entries
	ByType       map[string]int    // Entry count by type
	TotalSize    int               // Total entries
	Expired      int               // Expired but not yet cleaned
}

// GetDetailedStats returns detailed cache statistics
func (lc *LLMCache) GetDetailedStats() CacheStats {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	stats := CacheStats{
		ByType:   make(map[string]int),
		TotalSize: len(lc.entries),
	}

	if stats.TotalSize == 0 {
		return stats
	}

	now := time.Now()
	var totalAge time.Duration
	var totalHits int
	var expiredCount int

	for _, entry := range lc.entries {
		stats.ByType[entry.Type]++
		totalAge += now.Sub(entry.Timestamp)
		totalHits += entry.HitCount

		if now.After(entry.Expires) {
			expiredCount++
		}
	}

	stats.AvgAge = totalAge / time.Duration(stats.TotalSize)
	stats.Expired = expiredCount

	if stats.TotalSize > 0 {
		stats.HitRate = float64(totalHits) / float64(stats.TotalSize)
	}

	return stats
}

// GetTopHitters returns the N most-accessed cache entries
func (lc *LLMCache) GetTopHitters(limit int) []CacheEntry {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	if limit == 0 {
		limit = 10
	}

	// Convert to slice
	entries := make([]CacheEntry, 0, len(lc.entries))
	for _, v := range lc.entries {
		entries = append(entries, *v)
	}

	// Sort by hit count (descending)
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].HitCount < entries[j].HitCount {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	if limit > len(entries) {
		limit = len(entries)
	}

	return entries[:limit]
}
