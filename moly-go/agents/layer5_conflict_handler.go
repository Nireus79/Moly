package agents

import (
	"moly/database"
)

// Layer5ConflictHandler handles Layer 5: Conflict Detection & Clarification
// When user says something that conflicts with what they said before,
// ask a clarification question instead of giving potentially misaligned advice
type Layer5ConflictHandler struct {
	conflictDetector     *ConflictDetector
	clarificationEngine  *ClarificationEngine
	clarificationRepo    *database.ClarificationQuestionRepository
	clarificationHistory *ClarificationHistory
	db                   *database.Database
}

// NewLayer5ConflictHandler creates a new Layer 5 handler
func NewLayer5ConflictHandler(db *database.Database) *Layer5ConflictHandler {
	return &Layer5ConflictHandler{
		conflictDetector:     NewConflictDetector(db),
		clarificationEngine:  NewClarificationEngine(),
		clarificationRepo:    database.NewClarificationQuestionRepository(db),
		clarificationHistory: NewClarificationHistory(db),
		db:                   db,
	}
}
