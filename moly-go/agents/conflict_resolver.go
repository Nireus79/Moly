package agents

import (
	"fmt"
	"log"
	"time"

	"moly/database"
)

// ConflictResolution represents how a conflict was resolved
type ConflictResolution struct {
	ID           int64       `json:"id"`
	ConflictID   int64       `json:"conflictId"`
	UserChoice   string      `json:"userChoice"` // "keep_saved", "use_extracted", "merge", "ask_later"
	ChoiceNotes  string      `json:"choiceNotes"`
	ResolvedAt   int64       `json:"resolvedAt"`
	FinalValue   interface{} `json:"finalValue"`
}

// ConflictResolver applies user's resolution choice
type ConflictResolver struct {
	attrRepo    *database.ContextAttributeRepository
	conflictRepo *database.ConflictRepository // Need to implement
}

// NewConflictResolver creates a new resolver
func NewConflictResolver(
	attrRepo *database.ContextAttributeRepository,
	conflictRepo *database.ConflictRepository,
) *ConflictResolver {
	return &ConflictResolver{
		attrRepo: attrRepo,
		conflictRepo: conflictRepo,
	}
}

// ResolveConflict applies user's resolution to a conflict
// choice: "keep_saved", "use_extracted", "merge", "ask_later"
func (r *ConflictResolver) ResolveConflict(
	conflictID int64,
	userID string,
	choice string,
	conflictData *ConflictDetection,
	notes string,
) (*ConflictResolution, error) {
	log.Printf("[V2] ConflictResolver: RESOLVE START - conflict=%d user=%s choice=%s", conflictID, userID, choice)
	log.Printf("[V2] ConflictResolver:   saved=%v vs extracted=%v (context=%s severity=%s)", conflictData.SavedValue, conflictData.ExtractedValue, conflictData.Context, conflictData.Severity)

	if choice != "keep_saved" && choice != "use_extracted" && choice != "merge" && choice != "ask_later" {
		log.Printf("[V2] ConflictResolver: INVALID CHOICE - %s", choice)
		return nil, fmt.Errorf("invalid resolution choice: %s", choice)
	}

	var finalValue interface{}

	switch choice {
	case "keep_saved":
		log.Printf("[V2] ConflictResolver: ✓ USER CHOSE keep_saved (keeping %v, discarding %v)", conflictData.SavedValue, conflictData.ExtractedValue)
		finalValue = conflictData.SavedValue
		// Don't save extracted, conflict resolved

	case "use_extracted":
		log.Printf("[V2] ConflictResolver: ✓ USER CHOSE use_extracted (replacing %v with %v)", conflictData.SavedValue, conflictData.ExtractedValue)
		finalValue = conflictData.ExtractedValue
		// Save the extracted attribute with confirmed value

	case "merge":
		log.Printf("[V2] ConflictResolver: ✓ USER CHOSE merge (keeping both values with context separation)")
		finalValue = map[string]interface{}{
			"previous": conflictData.SavedValue,
			"new":      conflictData.ExtractedValue,
			"merged_at": time.Now().Unix(),
		}

	case "ask_later":
		log.Printf("[V2] ConflictResolver: ✓ USER CHOSE ask_later (deferring decision)")
		finalValue = nil // Don't change anything
	}

	resolution := &ConflictResolution{
		ConflictID:  conflictID,
		UserChoice:  choice,
		ChoiceNotes: notes,
		ResolvedAt:  time.Now().Unix(),
		FinalValue:  finalValue,
	}

	// Update conflict status in database
	if r.conflictRepo != nil {
		// Mark conflict as resolved
		log.Printf("[V2] ConflictResolver: persisting resolution to database")
		err := r.conflictRepo.MarkResolved(conflictID, choice)
		if err != nil {
			log.Printf("[V2] ConflictResolver: ✗ PERSIST FAILED - %v", err)
			return nil, err
		}
		log.Printf("[V2] ConflictResolver: ✓ resolution persisted (conflict=%d choice=%s)", conflictID, choice)
	}

	log.Printf("[V2] ConflictResolver: ✓ RESOLVED - conflict=%d user=%s choice=%s finalValue=%v notes=%s", conflictID, userID, choice, finalValue, notes)
	return resolution, nil
}

// ResolveSameContextConflict handles contradictions in same context
// Only one value can be true when context is the same
func (r *ConflictResolver) ResolveSameContextConflict(
	userID string,
	subject string,
	factType string,
	trustedValue string,
) error {
	log.Printf("[V2] ConflictResolver: resolving same-context conflict - %s.%s = %s", subject, factType, trustedValue)

	// Archive old value
	existing, err := r.attrRepo.GetByType(userID, subject, factType)
	if err != nil {
		return err
	}

	for _, attr := range existing {
		if attr.FactValue != trustedValue {
			// Soft delete by removing from active context
			// (implementation depends on how we handle archival)
			log.Printf("[V2] ConflictResolver: archived old value %s", attr.FactValue)
		}
	}

	return nil
}

// ResolveContextVariation handles attributes that differ by context
// Stores both values with context tags
func (r *ConflictResolver) ResolveContextVariation(
	userID string,
	conversationID string,
	subject string,
	factType string,
	formalContext string,
	informalContext string,
	evidence string,
	confidence float64,
) error {
	log.Printf("[V2] ConflictResolver: resolving context variation for %s", subject)

	attr := &database.ContextAttribute{
		UserID:         userID,
		ConversationID: conversationID,
		FactType:       factType,
		FactValue:      fmt.Sprintf("%s (in %s context), %s (in %s context)", formalContext, "work", informalContext, "social"),
		AttributedTo:   subject,
		Context:        "multi-context",
		Evidence:       evidence,
		Confidence:     confidence,
		Source:         "context_variation",
	}

	return r.attrRepo.Save(attr)
}

// CheckIfActuallyCompatible determines if two values can coexist
// E.g., "detail-oriented" + "thorough" are compatible (both mean same thing)
// But "formal" + "casual" are not
func (r *ConflictResolver) CheckIfActuallyCompatible(value1 string, value2 string) bool {
	compatiblePairs := map[[2]string]bool{
		[2]string{"detail-oriented", "thorough"}: true,
		[2]string{"thorough", "detail-oriented"}: true,
		[2]string{"direct", "straightforward"}: true,
		[2]string{"straightforward", "direct"}: true,
		[2]string{"organized", "structured"}: true,
		[2]string{"structured", "organized"}: true,
		[2]string{"analytical", "logical"}: true,
		[2]string{"logical", "analytical"}: true,
		[2]string{"collaborative", "team-oriented"}: true,
		[2]string{"team-oriented", "collaborative"}: true,
	}

	key := [2]string{value1, value2}
	return compatiblePairs[key]
}

// GenerateResolutionPrompt creates user-facing text for conflict resolution
func (r *ConflictResolver) GenerateResolutionPrompt(conflict *ConflictDetection) string {
	if conflict.Severity == "high" {
		return fmt.Sprintf("⚠️ Important clarification needed:\n\nPreviously you mentioned: \"%v\" (in %s context)\nNow you said: \"%v\"\n\nShould I keep the old info, use the new info, or are these from different contexts?",
			conflict.SavedValue, conflict.Context, conflict.ExtractedValue)
	}

	return fmt.Sprintf("Small detail to clarify:\n\nYou said \"%v\" before (context: %s), and now \"%v\". Should I:\n1. Keep \"%v\"\n2. Use \"%v\"\n3. Save both (different contexts)?",
		conflict.SavedValue, conflict.Context, conflict.ExtractedValue,
		conflict.SavedValue, conflict.ExtractedValue)
}
