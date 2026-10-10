package agents

import (
	"moly/database"
)

// ContactManager handles contact lifecycle: create, update, retrieve
type ContactManager struct {
	contactRepo *database.ContactRepository
}

// NewContactManager creates a new contact manager
func NewContactManager(repo *database.ContactRepository) *ContactManager {
	return &ContactManager{
		contactRepo: repo,
	}
}
