package database

import ()

// ConflictGate checks for conflicts before saving user preferences
// Implements the principle: "Ask for confirmation when preferences change"
type ConflictGate struct {
	conflictRepo *ContextConflictRepository
}

// NewConflictGate creates a new conflict gate
func NewConflictGate(db *Database) *ConflictGate {
	return &ConflictGate{
		conflictRepo: NewContextConflictRepository(db),
	}
}
