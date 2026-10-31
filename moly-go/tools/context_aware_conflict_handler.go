package tools

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"moly/database"
)

// ContextAwareConflictHandler decides whether conflicts need user approval based on context
// Core logic: Same field + same context = ask user. Same field + different context = auto-merge
type ContextAwareConflictHandler struct {
	db               *database.Database
	dataLoader       *DataLoaderHelper
	contextExtractor *ContextExtractorHelper
	conflictRepo     *database.ContextConflictRepository
}

// NewContextAwareConflictHandler creates a new handler
func NewContextAwareConflictHandler(db *database.Database) *ContextAwareConflictHandler {
	return &ContextAwareConflictHandler{
		db:               db,
		dataLoader:       NewDataLoaderHelper(db.GetConnection()),
		contextExtractor: NewContextExtractorHelper(),
		conflictRepo:     db.GetContextConflictRepository(),
	}
}

// ConflictDecision represents the decision about what to do with a conflict
type ConflictDecision struct {
	HasConflict   bool                      // Whether a real conflict exists
	NeedsApproval bool                      // Whether user needs to approve
	Action        string                    // "auto_merge", "queue_for_approval", "save_new"
	SavedConflict *database.ContextConflict // If queued, the saved conflict record
	AutoMergeInfo string                    // Info about auto-merge if applicable
	ConflictId    int64                     // ID of saved conflict if queued
	SkipUpdate    bool                      // Whether to skip the database update
}

// HandleStyleConflict checks if extracted style conflicts with saved style, considering context
func (h *ContextAwareConflictHandler) HandleStyleConflict(
	userID, conversationID, userMessage string,
	extractedStyle string,
	extractedConfidence float64,
) *ConflictDecision {

	log.Printf("[ConflictHandler] Handling style conflict for user %s", userID)

	// Extract context from current message
	newContext := h.contextExtractor.ExtractContextFromMessage(userMessage)
	log.Printf("[ConflictHandler] Extracted context from message: '%s'", newContext)

	// Load current saved style
	loadedStyle, err := h.dataLoader.LoadCurrentStyle(userID)
	if err != nil {
		log.Printf("[ConflictHandler] Error loading current style: %v", err)
		// If we can't load, don't create conflict, just save
		return &ConflictDecision{
			HasConflict:   false,
			NeedsApproval: false,
			Action:        "save_new",
			SkipUpdate:    false,
		}
	}

	// If no previous value, no conflict
	if !loadedStyle.HasData {
		log.Printf("[ConflictHandler] No previous style, no conflict")
		return &ConflictDecision{
			HasConflict:   false,
			NeedsApproval: false,
			Action:        "save_new",
			SkipUpdate:    false,
		}
	}

	oldStyle := loadedStyle.Value.(string)
	oldContext := loadedStyle.Context

	// Check if values are the same
	if oldStyle == extractedStyle {
		log.Printf("[ConflictHandler] Style unchanged: '%s', no conflict", oldStyle)
		return &ConflictDecision{
			HasConflict:   false,
			NeedsApproval: false,
			Action:        "save_new",
			SkipUpdate:    false,
		}
	}

	// Values differ - this is observed behavior, not a preference conflict
	// User's communication style varies by context and moment - this is normal learning, not a conflict
	log.Printf("[ConflictHandler] Observed style variation: '%s' (context: %s) → '%s' (context: %s)",
		oldStyle, oldContext, extractedStyle, newContext)
	log.Printf("[ConflictHandler] User behavior tracking (no conflict - single message tone is not a preference change)")

	// Don't create conflicts on observed user behavior
	// User can be formal in preferences but casual in a message - this is normal
	// Only Moly's behavior configuration should require approval
	return &ConflictDecision{
		HasConflict:   false,
		NeedsApproval: false,
		Action:        "save_new",
		SkipUpdate:    false,
	}
}

// ValidateResponseAgainstCharacteristics checks if Moly's response contradicts extracted user characteristics
// Returns: contradiction_found, user_characteristic, contrary_word_in_response, contradiction_reason
func (h *ContextAwareConflictHandler) ValidateResponseAgainstCharacteristics(
	userID, conversationID, response string,
	userCharacteristics []string,
) (bool, string, string, string) {

	if response == "" || len(userCharacteristics) == 0 {
		return false, "", "", ""
	}

	// Check if response mentions characteristics contradicting the extracted ones
	// Simple heuristic: if user extracted as "dominant" but response says "submissive", that's a contradiction
	contradictions := map[string][]string{
		"dominant":    []string{"submissive", "passive", "receptive", "obedient"},
		"assertive":   []string{"timid", "hesitant", "uncertain", "indecisive"},
		"submissive":  []string{"dominant", "assertive", "commanding", "controlling"},
		"passive":     []string{"active", "engaged", "dominant", "assertive"},
		"extroverted": []string{"introverted", "withdrawn", "reserved"},
		"introverted": []string{"extroverted", "outgoing", "talkative"},
		"confident":   []string{"insecure", "uncertain", "doubtful"},
		"insecure":    []string{"confident", "assured", "self-assured"},
	}

	lowerResponse := strings.ToLower(response)

	// Check each extracted characteristic
	for _, char := range userCharacteristics {
		if char == "" {
			continue // Skip empty characteristics
		}

		lowerChar := strings.ToLower(char)

		// Look for contradicting words in response
		if contraryWords, hasContrary := contradictions[lowerChar]; hasContrary {
			for _, contrary := range contraryWords {
				// Check if contrary word appears in response with word boundaries
				// Use regex to match whole words only, not substrings
				// Pattern: word boundary + word + word boundary
				pattern := fmt.Sprintf("\\b%s\\b", regexp.QuoteMeta(contrary))
				matched, err := regexp.MatchString(pattern, lowerResponse)
				if err != nil {
					log.Printf("[ConflictHandler] Warning: Regex match failed for pattern '%s': %v", pattern, err)
					continue
				}
				if matched {
					log.Printf("[ConflictHandler] ⚠ RESPONSE CONTRADICTION DETECTED: User extracted as '%s' but response suggests '%s'",
						char, contrary)
					return true, char, contrary, fmt.Sprintf("Response suggests '%s' but user was extracted as '%s'", contrary, char)
				}
			}
		}
	}

	return false, "", "", ""
}
