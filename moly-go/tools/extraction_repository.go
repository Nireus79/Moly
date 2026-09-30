package tools

import (
	"fmt"
	"log"
	"moly/models"
	"sync"
	"time"
)

// ExtractionRepository enforces immutability of locked extractions
// PHASE 1: Prevents modifications to locked extraction artifacts
type ExtractionRepository struct {
	mu          sync.RWMutex
	extractions map[string]*models.ExtractionArtifact // Key: extraction ID
}

// NewExtractionRepository creates a new repository with lock enforcement
func NewExtractionRepository() *ExtractionRepository {
	return &ExtractionRepository{
		extractions: make(map[string]*models.ExtractionArtifact),
	}
}

// SaveLocked saves an extraction only if it's locked
// Returns error if extraction is not locked
func (er *ExtractionRepository) SaveLocked(artifact *models.ExtractionArtifact) error {
	if artifact == nil {
		return fmt.Errorf("cannot save nil extraction artifact")
	}

	// PHASE 1: Enforce lock requirement
	if !artifact.IsLocked {
		return fmt.Errorf("cannot save unlocked extraction (ID: %s) - must lock before persisting", artifact.ID)
	}

	er.mu.Lock()
	defer er.mu.Unlock()

	er.extractions[artifact.ID] = artifact
	log.Printf("[ExtractionRepository] Saved locked extraction: id=%s, locked_at=%d, reason=%s",
		artifact.ID, artifact.LockedAt, artifact.LockReason)

	return nil
}

// GetLocked retrieves a locked extraction or errors if not locked
// Returns error if:
// 1. Extraction not found
// 2. Extraction exists but is not locked
func (er *ExtractionRepository) GetLocked(extractionID string) (*models.ExtractionArtifact, error) {
	if extractionID == "" {
		return nil, fmt.Errorf("cannot get extraction: empty ID")
	}

	er.mu.RLock()
	defer er.mu.RUnlock()

	artifact, exists := er.extractions[extractionID]
	if !exists {
		return nil, fmt.Errorf("extraction not found: %s", extractionID)
	}

	// PHASE 1: Enforce lock requirement on retrieval
	if !artifact.IsLocked {
		return nil, fmt.Errorf("extraction found but not locked (ID: %s) - this indicates incomplete initialization", extractionID)
	}

	return artifact, nil
}

// TryModify returns error if extraction is locked (enforces immutability)
// Used by downstream code that wants to modify an extraction
func (er *ExtractionRepository) TryModify(extractionID string, operation string) error {
	if extractionID == "" {
		return fmt.Errorf("cannot modify extraction: empty ID")
	}

	er.mu.RLock()
	defer er.mu.RUnlock()

	artifact, exists := er.extractions[extractionID]
	if !exists {
		return fmt.Errorf("cannot %s: extraction not found", operation)
	}

	// PHASE 1: Prevent any modifications to locked extraction
	if artifact.IsLocked {
		return fmt.Errorf("cannot %s on locked extraction (ID: %s, locked_at: %d, reason: %s)",
			operation, extractionID, artifact.LockedAt, artifact.LockReason)
	}

	return nil
}

// Get retrieves an extraction without lock enforcement
// Used only for read-only operations (querying, not modifying)
func (er *ExtractionRepository) Get(extractionID string) (*models.ExtractionArtifact, error) {
	if extractionID == "" {
		return nil, fmt.Errorf("cannot get extraction: empty ID")
	}

	er.mu.RLock()
	defer er.mu.RUnlock()

	artifact, exists := er.extractions[extractionID]
	if !exists {
		return nil, fmt.Errorf("extraction not found: %s", extractionID)
	}

	return artifact, nil
}

// GetAll returns all extractions (locked or not)
// Used for cleanup/monitoring, not for processing
func (er *ExtractionRepository) GetAll() []*models.ExtractionArtifact {
	er.mu.RLock()
	defer er.mu.RUnlock()

	artifacts := make([]*models.ExtractionArtifact, 0, len(er.extractions))
	for _, artifact := range er.extractions {
		artifacts = append(artifacts, artifact)
	}
	return artifacts
}

// GetLocked extractions only (enforces lock state)
func (er *ExtractionRepository) GetAllLocked() []*models.ExtractionArtifact {
	er.mu.RLock()
	defer er.mu.RUnlock()

	locked := make([]*models.ExtractionArtifact, 0)
	for _, artifact := range er.extractions {
		if artifact.IsLocked {
			locked = append(locked, artifact)
		}
	}
	return locked
}

// CleanupExpired removes extractions older than TTL
// Only considers locked extractions (prevents partial cleanup)
func (er *ExtractionRepository) CleanupExpired() int {
	er.mu.Lock()
	defer er.mu.Unlock()

	now := time.Now().Unix()
	removed := 0

	for id, artifact := range er.extractions {
		// Only cleanup if locked (completed processing)
		if !artifact.IsLocked {
			continue
		}

		// Check TTL
		if artifact.ExpiresAt > 0 && artifact.ExpiresAt < now {
			delete(er.extractions, id)
			removed++
			log.Printf("[ExtractionRepository] Cleaned up expired extraction: id=%s, expired_at=%d",
				id, artifact.ExpiresAt)
		}
	}

	if removed > 0 {
		log.Printf("[ExtractionRepository] Cleanup complete: removed %d expired extractions", removed)
	}

	return removed
}

// DeleteIfUnlocked removes an extraction only if it's not locked
// Used for cleanup of incomplete/failed extractions
func (er *ExtractionRepository) DeleteIfUnlocked(extractionID string) error {
	if extractionID == "" {
		return fmt.Errorf("cannot delete extraction: empty ID")
	}

	er.mu.Lock()
	defer er.mu.Unlock()

	artifact, exists := er.extractions[extractionID]
	if !exists {
		return fmt.Errorf("extraction not found: %s", extractionID)
	}

	if artifact.IsLocked {
		return fmt.Errorf("cannot delete locked extraction (ID: %s, locked_at: %d)",
			extractionID, artifact.LockedAt)
	}

	delete(er.extractions, extractionID)
	log.Printf("[ExtractionRepository] Deleted unlocked extraction: id=%s", extractionID)
	return nil
}

// Size returns total number of extractions stored
func (er *ExtractionRepository) Size() int {
	er.mu.RLock()
	defer er.mu.RUnlock()
	return len(er.extractions)
}

// Stats returns statistics about stored extractions
type ExtractionStats struct {
	Total           int
	Locked          int
	Unlocked        int
	AverageAge      float64 // seconds
	OldestExtractionAge float64 // seconds
}

func (er *ExtractionRepository) Stats() ExtractionStats {
	er.mu.RLock()
	defer er.mu.RUnlock()

	stats := ExtractionStats{
		Total:    len(er.extractions),
		Locked:   0,
		Unlocked: 0,
	}

	if stats.Total == 0 {
		return stats
	}

	now := time.Now().Unix()
	var totalAge int64
	var maxAge int64

	for _, artifact := range er.extractions {
		if artifact.IsLocked {
			stats.Locked++
		} else {
			stats.Unlocked++
		}

		age := now - artifact.CreatedAt
		totalAge += age
		if age > maxAge {
			maxAge = age
		}
	}

	stats.AverageAge = float64(totalAge) / float64(stats.Total)
	stats.OldestExtractionAge = float64(maxAge)

	return stats
}
