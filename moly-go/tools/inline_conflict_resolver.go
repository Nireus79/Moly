package tools

import (
	"fmt"
	"log"
	"strings"

	"moly/database"
)

// InlineConflictResolver detects and resolves conflicts within conversation flow
type InlineConflictResolver struct {
	db *database.Database
}

// NewInlineConflictResolver creates a new resolver
func NewInlineConflictResolver(db *database.Database) *InlineConflictResolver {
	return &InlineConflictResolver{db: db}
}

// ConflictAwareResponse wraps a response with conflict info
type ConflictAwareResponse struct {
	Response           string
	HasPendingConflict bool
	ConflictMessage    string // Question to ask user
	ConflictID         int64  // For tracking resolution
	ResolutionApplied  bool   // Whether a conflict was resolved
}

// CheckAndAskForConflicts detects pending conflicts and generates questions
func (r *InlineConflictResolver) CheckAndAskForConflicts(userID string) *ConflictAwareResponse {
	log.Printf("[InlineConflictResolver] Checking for pending conflicts for user %s", userID)

	conflictRepo := r.db.GetContextConflictRepository()
	conflicts, err := conflictRepo.GetUnresolved(userID)
	if err != nil {
		log.Printf("[InlineConflictResolver] Error loading conflicts: %v", err)
		return &ConflictAwareResponse{
			HasPendingConflict: false,
			ResolutionApplied:  false,
		}
	}

	if len(conflicts) == 0 {
		log.Printf("[InlineConflictResolver] No pending conflicts")
		return &ConflictAwareResponse{
			HasPendingConflict: false,
			ResolutionApplied:  false,
		}
	}

	// CRITICAL FIX: Filter out false "intention" conflicts
	// Different message intents (greet, ask, inform) are NOT conflicts
	// Real conflicts are VALUE/PREFERENCE mismatches (how you want to communicate, relationship types, etc.)
	validConflicts := []*database.ContextConflict{}
	for _, c := range conflicts {
		// SKIP: intention conflicts are NOT real conflicts (just different message intents)
		if c.ConflictType == "intention" {
			log.Printf("[InlineConflictResolver] Skipping false 'intention' conflict %d: different message intents are normal", c.ID)
			continue
		}
		validConflicts = append(validConflicts, c)
	}

	if len(validConflicts) == 0 {
		log.Printf("[InlineConflictResolver] No REAL conflicts found (skipped %d false intention conflicts)", len(conflicts))
		return &ConflictAwareResponse{
			HasPendingConflict: false,
			ResolutionApplied:  false,
		}
	}

	// Found real conflicts - generate question for the first one
	conflict := validConflicts[0]
	log.Printf("[InlineConflictResolver] Found REAL conflict %d: %s", conflict.ID, conflict.ConflictType)

	question := r.generateConflictQuestion(conflict)

	return &ConflictAwareResponse{
		HasPendingConflict: true,
		ConflictMessage:    question,
		ConflictID:         conflict.ID,
		ResolutionApplied:  false,
	}
}

// generateConflictQuestion creates a natural conversational question for the user
func (r *InlineConflictResolver) generateConflictQuestion(conflict *database.ContextConflict) string {
	contextExtractor := NewContextExtractorHelper()
	oldContextDesc := contextExtractor.GetContextDescription(r.getContextFromDetails(conflict, "oldContext"))
	newContextDesc := contextExtractor.GetContextDescription(r.getContextFromDetails(conflict, "newContext"))

	switch conflict.ConflictType {
	case "aboutme_communication_style":
		return fmt.Sprintf(
			"I remember you saying you prefer '%v' %s. "+
				"Now you're telling me '%v' %s. "+
				"Are both true for different situations, or has your preference changed?",
			conflict.SavedValue, oldContextDesc,
			conflict.ExtractedValue, newContextDesc,
		)

	case "contact_relationship":
		contactName := r.getContactFromDetails(conflict)
		return fmt.Sprintf(
			"I had noted that %s is your '%v'. "+
				"Now you're mentioning they're your '%v'. "+
				"Did that change, or does it depend on context?",
			contactName, conflict.SavedValue, conflict.ExtractedValue,
		)

	case "contact_characteristics":
		contactName := r.getContactFromDetails(conflict)
		oldChars := r.formatArray(conflict.SavedValue)
		newChars := r.formatArray(conflict.ExtractedValue)
		return fmt.Sprintf(
			"I used to know %s as: %s. "+
				"Now you're describing them as: %s. "+
				"Should I update how I see them?",
			contactName, oldChars, newChars,
		)

	case "intention":
		return fmt.Sprintf(
			"Earlier you mentioned your goal was '%v'. "+
				"Now you're saying '%v'. "+
				"Are you shifting priorities, or are these goals for different situations?",
			conflict.SavedValue, conflict.ExtractedValue,
		)

	default:
		return fmt.Sprintf(
			"I noticed you said '%v' before, but now '%v'. "+
				"Which is accurate?",
			conflict.SavedValue, conflict.ExtractedValue,
		)
	}
}

// ParseResolutionFromResponse analyzes user's response to determine their resolution choice
func (r *InlineConflictResolver) ParseResolutionFromResponse(userResponse string, conflict *database.ContextConflict) string {
	log.Printf("[InlineConflictResolver] Parsing resolution from response: %.100s...", userResponse)

	lowerResponse := strings.ToLower(userResponse)

	// Keywords indicating "keep old value"
	keepKeywords := []string{
		"first", "before", "original", "still", "still right", "was right", "correct",
		"keep", "that one", "old one", "earlier", "previous", "previous one",
	}

	// Keywords indicating "use new value"
	useKeywords := []string{
		"now", "changed", "new", "updated", "different", "shift", "current",
		"second", "that one", "now", "now i", "these days", "recent",
	}

	// Keywords indicating "both are true" / merge
	mergeKeywords := []string{
		"both", "depends", "depends on", "context", "situation", "different situations",
		"different times", "it depends", "sometimes", "both are", "both true",
		"work", "home", "social", "different places",
	}

	keepScore := r.countKeywords(lowerResponse, keepKeywords)
	useScore := r.countKeywords(lowerResponse, useKeywords)
	mergeScore := r.countKeywords(lowerResponse, mergeKeywords)

	log.Printf("[InlineConflictResolver] Scores - keep:%d use:%d merge:%d", keepScore, useScore, mergeScore)

	// Determine resolution based on highest score
	if mergeScore > keepScore && mergeScore > useScore {
		log.Printf("[InlineConflictResolver] Resolution: merge (both are true)")
		return "merge"
	} else if useScore > keepScore {
		log.Printf("[InlineConflictResolver] Resolution: use_extracted (changed)")
		return "use_extracted"
	} else if keepScore > 0 || keepScore == useScore {
		// If equal or keep wins, default to keeping old
		log.Printf("[InlineConflictResolver] Resolution: keep_saved (still right)")
		return "keep_saved"
	}

	// Default: if unclear, ask user to clarify
	log.Printf("[InlineConflictResolver] Unclear response, would need clarification")
	return ""
}

// ApplyConflictResolution applies the parsed resolution
func (r *InlineConflictResolver) ApplyConflictResolution(userID string, conflictID int64, resolution string) *ResolutionResult {
	if resolution == "" {
		// User's response wasn't clear enough
		return &ResolutionResult{
			Success: false,
			Message: "I couldn't tell from your response - could you clarify?",
		}
	}

	// Load the conflict
	conflictRepo := r.db.GetContextConflictRepository()
	conflicts, err := conflictRepo.GetUnresolved(userID)
	if err != nil {
		return &ResolutionResult{
			Success: false,
			Message: "Failed to load conflicts",
		}
	}

	var conflict *database.ContextConflict
	for _, c := range conflicts {
		if c.ID == conflictID {
			conflict = c
			break
		}
	}

	if conflict == nil {
		return &ResolutionResult{
			Success: false,
			Message: "Conflict not found",
		}
	}

	// Apply resolution
	handler := NewConflictResolutionHandler(r.db)
	result := handler.ApplyResolution(conflict, resolution)

	return result
}

// Helper functions

func (r *InlineConflictResolver) getContextFromDetails(conflict *database.ContextConflict, key string) string {
	if conflict.ResolutionDetails != nil {
		if ctx, ok := conflict.ResolutionDetails[key].(string); ok {
			return ctx
		}
	}
	return "general"
}

func (r *InlineConflictResolver) getContactFromDetails(conflict *database.ContextConflict) string {
	if conflict.ResolutionDetails != nil {
		if contact, ok := conflict.ResolutionDetails["contact"].(string); ok {
			return contact
		}
	}
	return "this person"
}

func (r *InlineConflictResolver) formatArray(value interface{}) string {
	if arrVal, ok := value.([]interface{}); ok {
		var items []string
		for _, v := range arrVal {
			items = append(items, fmt.Sprintf("%v", v))
		}
		return strings.Join(items, ", ")
	}
	return fmt.Sprintf("%v", value)
}

func (r *InlineConflictResolver) countKeywords(text string, keywords []string) int {
	count := 0
	for _, keyword := range keywords {
		// Count occurrences of keyword as whole words
		if strings.Contains(text, keyword) {
			count++
		}
	}
	return count
}
