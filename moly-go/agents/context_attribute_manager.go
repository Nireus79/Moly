package agents

import (
	"moly/database"
)

// ContextAttributeManager stores extracted facts with WHO attribution
type ContextAttributeManager struct {
	attrRepo *database.ContextAttributeRepository
}

// NewContextAttributeManager creates a new manager
func NewContextAttributeManager(repo *database.ContextAttributeRepository) *ContextAttributeManager {
	return &ContextAttributeManager{
		attrRepo: repo,
	}
}
