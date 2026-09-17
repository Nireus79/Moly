package tools

import (
	"fmt"
	"log"
	"time"

	"moly/database"
)

// ContextAwareConflictHandler decides whether conflicts need user approval based on context
// Core logic: Same field + same context = ask user. Same field + different context = auto-merge
type ContextAwareConflictHandler struct {
	db                  *database.Database
	dataLoader          *DataLoaderHelper
	contextExtractor    *ContextExtractorHelper
	conflictRepo        *database.ContextConflictRepository
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
	HasConflict      bool                       // Whether a real conflict exists
	NeedsApproval    bool                       // Whether user needs to approve
	Action           string                     // "auto_merge", "queue_for_approval", "save_new"
	SavedConflict    *database.ContextConflict  // If queued, the saved conflict record
	AutoMergeInfo    string                     // Info about auto-merge if applicable
	ConflictId       int64                      // ID of saved conflict if queued
	SkipUpdate       bool                       // Whether to skip the database update
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

	// Values differ - check contexts
	log.Printf("[ConflictHandler] Style changed: '%s' (context: %s) → '%s' (context: %s)",
		oldStyle, oldContext, extractedStyle, newContext)

	// Different contexts and both are specific (not general)?
	if !h.contextExtractor.AreContextsCompatible(oldContext, newContext) {
		// Same context, different values = USER CONFLICT
		log.Printf("[ConflictHandler] CONFLICT: Same context (%s), different values", oldContext)

		conflict := &database.ContextConflict{
			UserID:         userID,
			ConversationID: conversationID,
			ConflictType:   "aboutme_communication_style",
			Severity:       "medium",
			SavedValue:     oldStyle,
			ExtractedValue: extractedStyle,
			Description:    fmt.Sprintf("Communication style conflict: Previously %s '%s', now %s seems '%s'",
				h.contextExtractor.GetContextDescription(oldContext),
				oldStyle,
				h.contextExtractor.GetContextDescription(newContext),
				extractedStyle),
			Status:    "unresolved",
			ResolutionDetails: map[string]interface{}{
				"oldContext": oldContext,
				"newContext": newContext,
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
			HasConflict:    true,
			NeedsApproval:  true,
			Action:         "queue_for_approval",
			SavedConflict:   conflict,
			ConflictId:      conflict.ID,
			SkipUpdate:      true, // Don't update database yet
			AutoMergeInfo:   "User approval required",
		}
	} else {
		// Different contexts = AUTO-MERGE (save both)
		log.Printf("[ConflictHandler] AUTO-MERGE: Different contexts (%s vs %s), saving both", oldContext, newContext)

		return &ConflictDecision{
			HasConflict:    false,
			NeedsApproval:  false,
			Action:         "auto_merge",
			SkipUpdate:     false, // Update database with new context
			AutoMergeInfo:   fmt.Sprintf("Merged: '%s' (%s) + '%s' (%s)", oldStyle, oldContext, extractedStyle, newContext),
		}
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
			Description:    fmt.Sprintf("Contact relationship change: Was %s '%s', now seems to be '%s'",
				h.contextExtractor.GetContextDescription(oldContext),
				oldRel,
				extractedRelationship),
			Status:    "unresolved",
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
			SavedConflict:  conflict,
			ConflictId:     conflict.ID,
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

// HandleIntentionConflict checks if extracted intention conflicts with saved intention
func (h *ContextAwareConflictHandler) HandleIntentionConflict(
	userID, conversationID, userMessage string,
	extractedIntention string,
) *ConflictDecision {

	log.Printf("[ConflictHandler] Handling intention conflict for user %s", userID)

	newContext := h.contextExtractor.ExtractContextFromMessage(userMessage)
	log.Printf("[ConflictHandler] Extracted context: '%s'", newContext)

	// Load all intentions to check for conflicts
	allIntentions, err := h.dataLoader.LoadAllIntentionsForUser(userID)
	if err != nil {
		log.Printf("[ConflictHandler] Error loading intentions: %v", err)
		return &ConflictDecision{
			HasConflict:   false,
			NeedsApproval: false,
			Action:        "save_new",
			SkipUpdate:    false,
		}
	}

	if len(allIntentions) == 0 {
		log.Printf("[ConflictHandler] No previous intentions, no conflict")
		return &ConflictDecision{
			HasConflict:   false,
			NeedsApproval: false,
			Action:        "save_new",
			SkipUpdate:    false,
		}
	}

	// Check if intention already exists in same context
	if oldIntention, exists := allIntentions[newContext]; exists && oldIntention == extractedIntention {
		log.Printf("[ConflictHandler] Intention unchanged in context %s: '%s'", newContext, oldIntention)
		return &ConflictDecision{
			HasConflict:   false,
			NeedsApproval: false,
			Action:        "save_new",
			SkipUpdate:    false,
		}
	}

	// Check for contradictions in same context
	if oldIntention, exists := allIntentions[newContext]; exists && oldIntention != extractedIntention {
		log.Printf("[ConflictHandler] CONFLICT: Different intention in same context %s", newContext)

		conflict := &database.ContextConflict{
			UserID:         userID,
			ConversationID: conversationID,
			ConflictType:   "intention",
			Severity:       "medium",
			SavedValue:     oldIntention,
			ExtractedValue: extractedIntention,
			Description:    fmt.Sprintf("Intention conflict %s: Previously '%s', now '%s'",
				h.contextExtractor.GetContextDescription(newContext),
				oldIntention,
				extractedIntention),
			Status:    "unresolved",
			ResolutionDetails: map[string]interface{}{
				"context": newContext,
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
			SavedConflict:  conflict,
			ConflictId:     conflict.ID,
			SkipUpdate:    true,
		}
	}

	// Different context = auto-merge
	if _, exists := allIntentions[newContext]; !exists {
		log.Printf("[ConflictHandler] AUTO-MERGE: New context %s, adding alongside existing intentions", newContext)
		return &ConflictDecision{
			HasConflict:   false,
			NeedsApproval: false,
			Action:        "auto_merge",
			SkipUpdate:    false,
			AutoMergeInfo: fmt.Sprintf("Added intention in new context: %s", newContext),
		}
	}

	return &ConflictDecision{
		HasConflict:   false,
		NeedsApproval: false,
		Action:        "save_new",
		SkipUpdate:    false,
	}
}

// Helper to get current timestamp (used locally; database will set its own on insert)
func (h *ContextAwareConflictHandler) getCurrentTimestamp() int64 {
	return time.Now().Unix()
}
