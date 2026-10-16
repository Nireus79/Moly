package tools

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

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

// HandleContactRelationshipConflict checks if extracted relationship conflicts with saved relationship
func (h *ContextAwareConflictHandler) HandleContactRelationshipConflict(
	userID, conversationID, userMessage, contactName string,
	extractedRelationship string,
	extractedConfidence float64,
) *ConflictDecision {

	log.Printf("[ConflictHandler] Handling relationship conflict for contact %s", contactName)

	newContext := h.contextExtractor.ExtractContextFromMessage(userMessage)
	log.Printf("[ConflictHandler] Extracted context: '%s'", newContext)

	// Load current saved relationship
	loadedRel, err := h.dataLoader.LoadCurrentContactRelationship(userID, contactName)
	if err != nil {
		log.Printf("[ConflictHandler] Error loading current relationship: %v", err)
		return &ConflictDecision{
			HasConflict:   false,
			NeedsApproval: false,
			Action:        "save_new",
			SkipUpdate:    false,
		}
	}

	if !loadedRel.HasData {
		log.Printf("[ConflictHandler] No previous relationship, no conflict")
		return &ConflictDecision{
			HasConflict:   false,
			NeedsApproval: false,
			Action:        "save_new",
			SkipUpdate:    false,
		}
	}

	oldRel := loadedRel.Value.(string)
	oldContext := loadedRel.Context

	if oldRel == extractedRelationship {
		log.Printf("[ConflictHandler] Relationship unchanged: '%s'", oldRel)
		return &ConflictDecision{
			HasConflict:   false,
			NeedsApproval: false,
			Action:        "save_new",
			SkipUpdate:    false,
		}
	}

	// "other" says nothing of what the person is to the user: it is a gap, not a statement. A first real kind fills it (the identity
	// stage writes it), and "other" never replaces a real kind (found live, 2026-10-16: a father named only through his son was saved as
	// family, the user then said he is the ex boss, and Moly asked twice whether the relationship had changed).
	if oldRel == "other" || oldRel == "" || extractedRelationship == "other" || extractedRelationship == "" {
		log.Printf("[ConflictHandler] One of '%s' and '%s' is no statement: no conflict", oldRel, extractedRelationship)
		return &ConflictDecision{Action: "save_new"}
	}

	log.Printf("[ConflictHandler] Relationship changed: '%s' → '%s'", oldRel, extractedRelationship)

	// Different contexts = auto-merge for relationships (can have multiple roles)
	if !h.contextExtractor.AreContextsCompatible(oldContext, newContext) {
		log.Printf("[ConflictHandler] CONFLICT: Relationship changed in same context")

		conflict := &database.ContextConflict{
			UserID:         userID,
			ConversationID: conversationID,
			ConflictType:   "contact_relationship",
			Severity:       "high",
			SavedValue:     oldRel,
			ExtractedValue: extractedRelationship,
			Description: fmt.Sprintf("Contact relationship change: Was %s '%s', now seems to be '%s'",
				h.contextExtractor.GetContextDescription(oldContext),
				oldRel,
				extractedRelationship),
			Status: "unresolved",
			ResolutionDetails: map[string]interface{}{
				"oldContext": oldContext,
				"newContext": newContext,
				"contact":    contactName,
			},
			CreatedAt: h.getCurrentTimestamp(),
		}

		err := h.conflictRepo.Save(conflict)
		if err != nil {
			log.Printf("[ConflictHandler] Error saving conflict: %v", err)
			return &ConflictDecision{
				HasConflict:   false,
				NeedsApproval: false,
				Action:        "save_new",
				SkipUpdate:    false,
			}
		}

		return &ConflictDecision{
			HasConflict:   true,
			NeedsApproval: true,
			Action:        "queue_for_approval",
			SavedConflict: conflict,
			ConflictId:    conflict.ID,
			SkipUpdate:    true,
		}
	} else {
		log.Printf("[ConflictHandler] AUTO-MERGE: Different contexts for relationship")
		return &ConflictDecision{
			HasConflict:   false,
			NeedsApproval: false,
			Action:        "auto_merge",
			SkipUpdate:    false,
		}
	}
}

// QueueContactCharacteristicsConflict saves the question "should I update how I see this person?" for traits the model judged to
// describe a change (agents.JudgeTraitChange). Whether it IS a change is decided by the model, against what was stored before
// the message; this only records it, in the conversation the person was brought up in.
func (h *ContextAwareConflictHandler) QueueContactCharacteristicsConflict(userID, conversationID, contactName string, before, incoming []string) *ConflictDecision {
	log.Printf("[ConflictHandler] Queueing contact characteristics change for %s", contactName)
	conflict := &database.ContextConflict{
		UserID:         userID,
		ConversationID: conversationID,
		ConflictType:   "contact_characteristics",
		Severity:       "medium",
		SavedValue:     strings.Join(before, ", "),
		ExtractedValue: strings.Join(incoming, ", "),
		Description:    fmt.Sprintf("Contact characteristics changed: was [%s], now [%s]", strings.Join(before, ", "), strings.Join(incoming, ", ")),
		Status:         "unresolved",
		ResolutionDetails: map[string]interface{}{
			"contact": contactName,
		},
		CreatedAt: h.getCurrentTimestamp(),
	}
	if err := h.conflictRepo.Save(conflict); err != nil {
		log.Printf("[ConflictHandler] Error saving characteristics conflict: %v", err)
		return &ConflictDecision{Action: "save_new"}
	}
	return &ConflictDecision{HasConflict: true, NeedsApproval: true, Action: "queue_for_approval", SavedConflict: conflict, ConflictId: conflict.ID, SkipUpdate: true}
}

// Helper to get current timestamp (used locally; database will set its own on insert)
func (h *ContextAwareConflictHandler) getCurrentTimestamp() int64 {
	return time.Now().Unix()
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
