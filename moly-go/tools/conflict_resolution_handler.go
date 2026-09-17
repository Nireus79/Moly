package tools

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"moly/database"
)

// ConflictResolutionHandler applies user's conflict resolution to the database
type ConflictResolutionHandler struct {
	db *database.Database
}

// NewConflictResolutionHandler creates a new handler
func NewConflictResolutionHandler(db *database.Database) *ConflictResolutionHandler {
	return &ConflictResolutionHandler{db: db}
}

// ResolutionResult describes what was done to resolve the conflict
type ResolutionResult struct {
	Success       bool
	Message       string
	UpdatedTable  string
	UpdatedField  string
	OldValue      interface{}
	NewValue      interface{}
	ResolutionID  int64
	AppliedAt     int64
}

// ApplyResolution takes a conflict and a user's choice, updates database accordingly
// resolution: "keep_saved" | "use_extracted" | "merge"
func (h *ConflictResolutionHandler) ApplyResolution(
	conflict *database.ContextConflict,
	resolution string,
) *ResolutionResult {

	if conflict == nil {
		return &ResolutionResult{
			Success: false,
			Message: "Conflict is nil",
		}
	}

	log.Printf("[ConflictResolution] Applying resolution for conflict %d: %s (resolution=%s)",
		conflict.ID, conflict.ConflictType, resolution)

	// Validate resolution choice
	if resolution != "keep_saved" && resolution != "use_extracted" && resolution != "merge" {
		return &ResolutionResult{
			Success: false,
			Message: fmt.Sprintf("Invalid resolution: %s", resolution),
		}
	}

	// Route to appropriate handler based on conflict type
	switch conflict.ConflictType {
	case "aboutme_communication_style":
		return h.resolveStyleConflict(conflict, resolution)
	case "contact_relationship":
		return h.resolveContactRelationshipConflict(conflict, resolution)
	case "contact_characteristics":
		return h.resolveContactCharacteristicsConflict(conflict, resolution)
	case "intention":
		return h.resolveIntentionConflict(conflict, resolution)
	default:
		return &ResolutionResult{
			Success: false,
			Message: fmt.Sprintf("Unknown conflict type: %s", conflict.ConflictType),
		}
	}
}

// resolveStyleConflict handles communication style conflicts
func (h *ConflictResolutionHandler) resolveStyleConflict(
	conflict *database.ContextConflict,
	resolution string,
) *ResolutionResult {

	log.Printf("[ConflictResolution] Resolving style conflict: %s", resolution)

	conn := h.db.GetConnection()
	now := time.Now().Unix()

	var newValue interface{}

	switch resolution {
	case "keep_saved":
		// Keep the old value, don't update
		log.Printf("[ConflictResolution] Keeping saved style")
		newValue = conflict.SavedValue
	case "use_extracted":
		// Update to new extracted value
		log.Printf("[ConflictResolution] Updating to extracted style: %v", conflict.ExtractedValue)
		newValue = conflict.ExtractedValue
	case "merge":
		// For style, merge means format as "context1: style1, context2: style2"
		// Extract context info from resolution_details if available
		newValue = h.buildMergedStyle(conflict)
		log.Printf("[ConflictResolution] Merged style: %v", newValue)
	}

	// Update about_me table
	_, err := conn.Exec(
		"UPDATE about_me SET communication_style = ?, updated_at = ? WHERE user_id = ?",
		fmt.Sprintf("%v", newValue),
		now,
		conflict.UserID,
	)

	if err != nil {
		log.Printf("[ConflictResolution] ERROR updating style: %v", err)
		return &ResolutionResult{
			Success: false,
			Message: fmt.Sprintf("Failed to update style: %v", err),
		}
	}

	// Mark conflict as resolved
	conflictRepo := h.db.GetContextConflictRepository()
	resolveErr := conflictRepo.Resolve(conflict.ID, resolution, map[string]interface{}{
		"applied_at":   now,
		"new_value":    newValue,
		"resolution":   resolution,
	})

	if resolveErr != nil {
		log.Printf("[ConflictResolution] ERROR marking conflict resolved: %v", resolveErr)
		return &ResolutionResult{
			Success: false,
			Message: fmt.Sprintf("Failed to mark conflict resolved: %v", resolveErr),
		}
	}

	log.Printf("[ConflictResolution] ✓ Style conflict resolved: %s → %v", resolution, newValue)

	return &ResolutionResult{
		Success:      true,
		Message:      fmt.Sprintf("Style conflict resolved: %s", resolution),
		UpdatedTable: "about_me",
		UpdatedField: "communication_style",
		OldValue:     conflict.SavedValue,
		NewValue:     newValue,
		ResolutionID: conflict.ID,
		AppliedAt:    now,
	}
}

// resolveContactRelationshipConflict handles contact relationship conflicts
func (h *ConflictResolutionHandler) resolveContactRelationshipConflict(
	conflict *database.ContextConflict,
	resolution string,
) *ResolutionResult {

	log.Printf("[ConflictResolution] Resolving contact relationship conflict: %s", resolution)

	conn := h.db.GetConnection()
	now := time.Now().Unix()

	// Extract contact name from resolution_details if available
	contactName := ""
	if conflict.ResolutionDetails != nil {
		if name, ok := conflict.ResolutionDetails["contact"].(string); ok {
			contactName = name
		}
	}

	if contactName == "" {
		return &ResolutionResult{
			Success: false,
			Message: "Contact name not found in conflict details",
		}
	}

	var newValue interface{}

	switch resolution {
	case "keep_saved":
		log.Printf("[ConflictResolution] Keeping saved relationship")
		newValue = conflict.SavedValue
	case "use_extracted":
		log.Printf("[ConflictResolution] Updating to extracted relationship: %v", conflict.ExtractedValue)
		newValue = conflict.ExtractedValue
	case "merge":
		newValue = h.buildMergedValue(conflict.SavedValue, conflict.ExtractedValue)
		log.Printf("[ConflictResolution] Merged relationship: %v", newValue)
	}

	// Update user_contacts table
	_, err := conn.Exec(
		"UPDATE user_contacts SET relationship = ?, updated_at = ? WHERE user_id = ? AND name = ?",
		fmt.Sprintf("%v", newValue),
		now,
		conflict.UserID,
		contactName,
	)

	if err != nil {
		log.Printf("[ConflictResolution] ERROR updating contact relationship: %v", err)
		return &ResolutionResult{
			Success: false,
			Message: fmt.Sprintf("Failed to update contact: %v", err),
		}
	}

	// Mark conflict as resolved
	conflictRepo := h.db.GetContextConflictRepository()
	resolveErr := conflictRepo.Resolve(conflict.ID, resolution, map[string]interface{}{
		"applied_at":   now,
		"new_value":    newValue,
		"contact":      contactName,
		"resolution":   resolution,
	})

	if resolveErr != nil {
		log.Printf("[ConflictResolution] ERROR marking conflict resolved: %v", resolveErr)
		return &ResolutionResult{
			Success: false,
			Message: fmt.Sprintf("Failed to mark conflict resolved: %v", resolveErr),
		}
	}

	log.Printf("[ConflictResolution] ✓ Contact conflict resolved: %s → %v", resolution, newValue)

	return &ResolutionResult{
		Success:      true,
		Message:      fmt.Sprintf("Contact conflict resolved: %s", resolution),
		UpdatedTable: "user_contacts",
		UpdatedField: "relationship",
		OldValue:     conflict.SavedValue,
		NewValue:     newValue,
		ResolutionID: conflict.ID,
		AppliedAt:    now,
	}
}

// resolveContactCharacteristicsConflict handles contact characteristics conflicts
func (h *ConflictResolutionHandler) resolveContactCharacteristicsConflict(
	conflict *database.ContextConflict,
	resolution string,
) *ResolutionResult {

	log.Printf("[ConflictResolution] Resolving contact characteristics conflict: %s", resolution)

	conn := h.db.GetConnection()
	now := time.Now().Unix()

	// Extract contact name
	contactName := ""
	if conflict.ResolutionDetails != nil {
		if name, ok := conflict.ResolutionDetails["contact"].(string); ok {
			contactName = name
		}
	}

	if contactName == "" {
		return &ResolutionResult{
			Success: false,
			Message: "Contact name not found in conflict details",
		}
	}

	var newValue interface{}

	switch resolution {
	case "keep_saved":
		log.Printf("[ConflictResolution] Keeping saved characteristics")
		newValue = conflict.SavedValue
	case "use_extracted":
		log.Printf("[ConflictResolution] Updating to extracted characteristics: %v", conflict.ExtractedValue)
		newValue = conflict.ExtractedValue
	case "merge":
		newValue = h.mergeCharacteristics(conflict.SavedValue, conflict.ExtractedValue)
		log.Printf("[ConflictResolution] Merged characteristics: %v", newValue)
	}

	// Convert to JSON if needed
	charJSON := fmt.Sprintf("%v", newValue)
	if charArray, ok := newValue.([]interface{}); ok {
		b, _ := json.Marshal(charArray)
		charJSON = string(b)
	}

	// Update user_contacts table
	_, err := conn.Exec(
		"UPDATE user_contacts SET characteristics = ?, updated_at = ? WHERE user_id = ? AND name = ?",
		charJSON,
		now,
		conflict.UserID,
		contactName,
	)

	if err != nil {
		log.Printf("[ConflictResolution] ERROR updating characteristics: %v", err)
		return &ResolutionResult{
			Success: false,
			Message: fmt.Sprintf("Failed to update characteristics: %v", err),
		}
	}

	// Mark conflict as resolved
	conflictRepo := h.db.GetContextConflictRepository()
	resolveErr := conflictRepo.Resolve(conflict.ID, resolution, map[string]interface{}{
		"applied_at":   now,
		"new_value":    newValue,
		"contact":      contactName,
		"resolution":   resolution,
	})

	if resolveErr != nil {
		return &ResolutionResult{
			Success: false,
			Message: fmt.Sprintf("Failed to mark conflict resolved: %v", resolveErr),
		}
	}

	return &ResolutionResult{
		Success:      true,
		Message:      fmt.Sprintf("Characteristics conflict resolved: %s", resolution),
		UpdatedTable: "user_contacts",
		UpdatedField: "characteristics",
		OldValue:     conflict.SavedValue,
		NewValue:     newValue,
		ResolutionID: conflict.ID,
		AppliedAt:    now,
	}
}

// resolveIntentionConflict handles intention conflicts
func (h *ConflictResolutionHandler) resolveIntentionConflict(
	conflict *database.ContextConflict,
	resolution string,
) *ResolutionResult {

	log.Printf("[ConflictResolution] Resolving intention conflict: %s", resolution)

	conn := h.db.GetConnection()
	now := time.Now().Unix()

	// Extract context from resolution_details
	context := "general"
	if conflict.ResolutionDetails != nil {
		if c, ok := conflict.ResolutionDetails["context"].(string); ok {
			context = c
		}
	}

	var newValue interface{}

	switch resolution {
	case "keep_saved":
		log.Printf("[ConflictResolution] Keeping saved intention")
		newValue = conflict.SavedValue
	case "use_extracted":
		log.Printf("[ConflictResolution] Updating to extracted intention: %v", conflict.ExtractedValue)
		newValue = conflict.ExtractedValue
	case "merge":
		// For intentions, merge means adding to context_attributes (new entry)
		newValue = conflict.ExtractedValue
		log.Printf("[ConflictResolution] Adding new intention for context: %s", context)
	}

	// For merge, we add a new entry rather than update
	if resolution == "merge" {
		_, err := conn.Exec(
			"INSERT INTO context_attributes (user_id, fact_type, fact_value, context, confidence, created_at) VALUES (?, ?, ?, ?, ?, ?)",
			conflict.UserID,
			"intention",
			fmt.Sprintf("%v", newValue),
			context,
			0.8,
			now,
		)

		if err != nil {
			log.Printf("[ConflictResolution] ERROR adding intention: %v", err)
			return &ResolutionResult{
				Success: false,
				Message: fmt.Sprintf("Failed to add intention: %v", err),
			}
		}
	} else {
		// Keep_saved or use_extracted: update existing
		_, err := conn.Exec(
			"UPDATE context_attributes SET fact_value = ?, updated_at = ? WHERE user_id = ? AND fact_type = 'intention' AND context = ?",
			fmt.Sprintf("%v", newValue),
			now,
			conflict.UserID,
			context,
		)

		if err != nil {
			log.Printf("[ConflictResolution] ERROR updating intention: %v", err)
			return &ResolutionResult{
				Success: false,
				Message: fmt.Sprintf("Failed to update intention: %v", err),
			}
		}
	}

	// Mark conflict as resolved
	conflictRepo := h.db.GetContextConflictRepository()
	resolveErr := conflictRepo.Resolve(conflict.ID, resolution, map[string]interface{}{
		"applied_at":   now,
		"new_value":    newValue,
		"context":      context,
		"resolution":   resolution,
	})

	if resolveErr != nil {
		return &ResolutionResult{
			Success: false,
			Message: fmt.Sprintf("Failed to mark conflict resolved: %v", resolveErr),
		}
	}

	log.Printf("[ConflictResolution] ✓ Intention conflict resolved: %s → %v", resolution, newValue)

	return &ResolutionResult{
		Success:      true,
		Message:      fmt.Sprintf("Intention conflict resolved: %s", resolution),
		UpdatedTable: "context_attributes",
		UpdatedField: "fact_value",
		OldValue:     conflict.SavedValue,
		NewValue:     newValue,
		ResolutionID: conflict.ID,
		AppliedAt:    now,
	}
}

// Helper functions

func (h *ConflictResolutionHandler) buildMergedStyle(conflict *database.ContextConflict) string {
	oldCtx := "general"
	newCtx := "general"

	if conflict.ResolutionDetails != nil {
		if ctx, ok := conflict.ResolutionDetails["oldContext"].(string); ok {
			oldCtx = ctx
		}
		if ctx, ok := conflict.ResolutionDetails["newContext"].(string); ok {
			newCtx = ctx
		}
	}

	return fmt.Sprintf("%s (%s) + %s (%s)",
		conflict.SavedValue, oldCtx,
		conflict.ExtractedValue, newCtx)
}

func (h *ConflictResolutionHandler) buildMergedValue(old, new interface{}) string {
	return fmt.Sprintf("%v + %v", old, new)
}

func (h *ConflictResolutionHandler) mergeCharacteristics(old, new interface{}) []interface{} {
	merged := []interface{}{}

	// Add old characteristics
	if oldArray, ok := old.([]interface{}); ok {
		merged = append(merged, oldArray...)
	}

	// Add new characteristics (avoiding duplicates)
	if newArray, ok := new.([]interface{}); ok {
		for _, newChar := range newArray {
			isDuplicate := false
			for _, oldChar := range merged {
				if oldChar == newChar {
					isDuplicate = true
					break
				}
			}
			if !isDuplicate {
				merged = append(merged, newChar)
			}
		}
	}

	return merged
}
