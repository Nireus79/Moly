package database

import (
	"fmt"
	"log"
	"time"
)

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

// CheckBeforeSaving checks if saving this value conflicts with existing data
// Returns:
//   - conflict: The conflict object (nil if no conflict)
//   - shouldAsk: Whether user should be asked to confirm the change
func (cg *ConflictGate) CheckBeforeSaving(
	userID, conversationID, conflictType string,
	savedValue, newValue interface{},
) (*ContextConflict, bool) {

	if savedValue == nil || newValue == nil {
		// No conflict if either value is empty
		return nil, false
	}

	// Use repository's DetectConflict method
	conflict := cg.conflictRepo.DetectConflict(userID, conversationID, conflictType, savedValue, newValue)

	if conflict == nil {
		// No conflict detected
		log.Printf("[ConflictGate] ✓ No conflict for %s (saved=%v, new=%v)", conflictType, savedValue, newValue)
		return nil, false
	}

	// Conflict detected - need to ask user
	log.Printf("[ConflictGate] ⚠️  Conflict detected for %s: %s → %s", conflictType, savedValue, newValue)
	log.Printf("[ConflictGate]     Description: %s", conflict.Description)
	log.Printf("[ConflictGate]     Severity: %s", conflict.Severity)

	return conflict, true
}

// SaveWithConflictCheck checks for conflicts, saves if no conflict, or returns conflict if one exists
// If conflict exists, it's saved as "unresolved" for user to confirm
func (cg *ConflictGate) SaveWithConflictCheck(
	userID, conversationID, conflictType string,
	savedValue, newValue interface{},
) (*ContextConflict, error) {

	conflict, shouldAsk := cg.CheckBeforeSaving(userID, conversationID, conflictType, savedValue, newValue)

	if !shouldAsk {
		// No conflict, can proceed with save
		return nil, nil
	}

	// Conflict detected - save it for user resolution
	conflict.UserID = userID
	conflict.ConversationID = conversationID
	conflict.CreatedAt = time.Now().Unix()

	err := cg.conflictRepo.Save(conflict)
	if err != nil {
		log.Printf("[ConflictGate] ❌ Failed to save conflict: %v", err)
		return nil, fmt.Errorf("failed to save conflict: %w", err)
	}

	log.Printf("[ConflictGate] ✓ Conflict saved as unresolved (ID: %d)", conflict.ID)
	return conflict, nil
}

// GetUnresolvedForUser retrieves all unresolved conflicts for a user
func (cg *ConflictGate) GetUnresolvedForUser(userID string) ([]*ContextConflict, error) {
	return cg.conflictRepo.GetUnresolved(userID)
}

// ResolvConflict marks a conflict as resolved with user's decision
func (cg *ConflictGate) ResolveConflict(conflictID int64, resolution string, details map[string]interface{}) error {
	return cg.conflictRepo.Resolve(conflictID, resolution, details)
}
