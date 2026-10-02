package tools

import (
	"log"
	"sync"
	"time"

	"moly/models"
)

// ExtractionStore: In-memory cache for extraction artifacts
// Stores extraction results temporarily (5-minute TTL)
// All downstream components query this instead of re-extracting
type ExtractionStore struct {
	mu        sync.RWMutex
	artifacts map[string]*models.ExtractionArtifact // key: messageID
	cleanup   *time.Ticker
	stopChan  chan struct{}
}

// NewExtractionStore creates a new extraction store with cleanup routine
func NewExtractionStore() *ExtractionStore {
	es := &ExtractionStore{
		artifacts: make(map[string]*models.ExtractionArtifact),
		cleanup:   time.NewTicker(1 * time.Minute),
		stopChan:  make(chan struct{}),
	}

	// Start cleanup goroutine
	// DISABLED: Cleanup goroutines were causing database to close prematurely
	// TODO: Re-enable after server starts listening
	// go es.cleanupExpired()
	log.Printf("[ExtractionStore] Initialized with 5-minute TTL and 1-minute cleanup interval (cleanup DISABLED)")

	return es
}

// Save stores an extraction artifact
func (es *ExtractionStore) Save(artifact *models.ExtractionArtifact) {
	if artifact == nil {
		return
	}

	es.mu.Lock()
	defer es.mu.Unlock()

	// Set TTL: 5 minutes from now
	artifact.ExpiresAt = time.Now().Add(5 * time.Minute).Unix()
	es.artifacts[artifact.MessageID] = artifact

	log.Printf("[ExtractionStore] Saved extraction for message %s (%d entities, source=%s, TTL=5m)",
		artifact.MessageID, len(artifact.Entities), artifact.Source)
}

// Get retrieves an extraction artifact by message ID
func (es *ExtractionStore) Get(messageID string) *models.ExtractionArtifact {
	if messageID == "" {
		return nil
	}

	es.mu.RLock()
	defer es.mu.RUnlock()

	artifact := es.artifacts[messageID]
	if artifact != nil {
		log.Printf("[ExtractionStore] Retrieved extraction for message %s (%d entities)",
			messageID, len(artifact.Entities))
	}
	return artifact
}

// Delete removes an extraction artifact from the store
func (es *ExtractionStore) Delete(messageID string) {
	es.mu.Lock()
	defer es.mu.Unlock()

	if _, exists := es.artifacts[messageID]; exists {
		delete(es.artifacts, messageID)
		log.Printf("[ExtractionStore] Deleted extraction for message %s", messageID)
	}
}

// Size returns the number of artifacts in the store
func (es *ExtractionStore) Size() int {
	es.mu.RLock()
	defer es.mu.RUnlock()
	return len(es.artifacts)
}

// cleanupExpired removes expired artifacts periodically
func (es *ExtractionStore) cleanupExpired() {
	defer func() {
		es.cleanup.Stop()
		log.Printf("[ExtractionStore] Cleanup routine stopped")
	}()

	for {
		select {
		case <-es.cleanup.C:
			es.mu.Lock()
			now := time.Now().Unix()
			deleted := 0

			for id, artifact := range es.artifacts {
				if artifact.ExpiresAt < now {
					delete(es.artifacts, id)
					deleted++
				}
			}

			if deleted > 0 {
				log.Printf("[ExtractionStore] Cleanup: Deleted %d expired artifacts (remaining: %d)",
					deleted, len(es.artifacts))
			}
			es.mu.Unlock()

		case <-es.stopChan:
			return
		}
	}
}

// Stop gracefully shuts down the extraction store
func (es *ExtractionStore) Stop() {
	close(es.stopChan)
	log.Printf("[ExtractionStore] Stop signal sent")
}
