package tools

import (
	"fmt"
	"moly/models"
	"sync"
)

// ExtractionRepository enforces immutability of locked extractions
// PHASE 1: Prevents modifications to locked extraction artifacts
type ExtractionRepository struct {
	mu          sync.RWMutex
	extractions map[string]*models.ExtractionArtifact // Key: extraction ID
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





// Size returns total number of extractions stored
func (er *ExtractionRepository) Size() int {
	er.mu.RLock()
	defer er.mu.RUnlock()
	return len(er.extractions)
}


