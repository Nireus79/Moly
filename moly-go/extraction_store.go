package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"moly/agents"
	"moly/database"
	"moly/models"
)

// PreviousExtraction stores extraction data from a previous message
// Used by FIX #1 to accumulate context across messages for Layer 5+ operations
// Used by FIX #4 to persist primary goal across messages
type PreviousExtraction struct {
	Entities       []models.ExtractedEntity
	Goal           string
	Values         []string
	PrimaryGoal    string   // The goal of the conversation: set by the first goal stated, replaced when the user moves on to another aim
	Progression    []string // Track goal evolution (FIX #4)
	ContextTracker *models.ContextTrackerState
}

// savePreviousExtraction stores the extraction used as context for the next message.
// The database is the source of truth; the in-memory cache only avoids re-reading it.
func (srv *APIServer) savePreviousExtraction(userID, conversationID string, extraction *PreviousExtraction) {
	if srv.database == nil {
		log.Printf("[MessageProcessor] ERROR: database unavailable, extraction for conversation %s not saved", conversationID)
		return
	}
	turn := &turnCommit{}
	srv.savePreviousExtractionWith(turn, userID, conversationID, extraction)
	if err := turn.Run(srv.database.GetConnection()); err != nil {
		log.Printf("[MessageProcessor] ERROR: %v", err)
	}
}

// savePreviousExtractionWith adds the conversation's working state (including the locked goal) to the turn commit.
// The in-memory copy is filled only after the commit succeeds.
func (srv *APIServer) savePreviousExtractionWith(ex database.Executor, userID, conversationID string, extraction *PreviousExtraction) {
	if userID == "" || conversationID == "" || extraction == nil {
		return
	}
	state, err := json.Marshal(extraction)
	if err != nil {
		log.Printf("[MessageProcessor] ERROR: could not encode extraction for conversation %s: %v", conversationID, err)
		return
	}
	if err := database.NewConversationContextRepository(srv.database).SaveWith(ex, userID, conversationID, state); err != nil {
		log.Printf("[MessageProcessor] ERROR: %v", err)
		return
	}
	store := func() {
		srv.previousExtractionCache.Store(contextCacheKey(userID, conversationID), extraction)
		log.Printf("[MessageProcessor] Saved context for conversation %s (goal=%q, entities=%d)",
			conversationID, extraction.Goal, len(extraction.Entities))
	}
	if tc, ok := ex.(*turnCommit); ok {
		tc.AfterCommit(store)
	} else {
		store()
	}
}

func contextCacheKey(userID, conversationID string) string {
	return userID + "|" + conversationID
}

// loadPreviousExtraction returns the saved context for a conversation, or nil for a new conversation.
func (srv *APIServer) loadPreviousExtraction(userID, conversationID string) *PreviousExtraction {
	if userID == "" || conversationID == "" {
		return nil
	}
	if cached, ok := srv.previousExtractionCache.Load(contextCacheKey(userID, conversationID)); ok {
		return cached.(*PreviousExtraction)
	}
	if srv.database == nil {
		return nil
	}
	state, err := database.NewConversationContextRepository(srv.database).Load(userID, conversationID)
	if err != nil {
		log.Printf("[MessageProcessor] ERROR: %v", err)
		return nil
	}
	if state == nil {
		return nil
	}
	var extraction PreviousExtraction
	if err := json.Unmarshal(state, &extraction); err != nil {
		log.Printf("[MessageProcessor] ERROR: stored context for conversation %s is unreadable: %v", conversationID, err)
		return nil
	}
	srv.previousExtractionCache.Store(contextCacheKey(userID, conversationID), &extraction)
	log.Printf("[MessageProcessor] Loaded context for conversation %s (goal=%q, entities=%d)",
		conversationID, extraction.Goal, len(extraction.Entities))
	return &extraction
}

// FIX #9: mergeEntities combines accumulated and current entities, avoiding duplicates
// Keeps highest confidence version of duplicate entities by (type + value) key
func mergeEntities(accumulated, current []models.ExtractedEntity) []models.ExtractedEntity {
	if len(accumulated) == 0 {
		return current
	}
	if len(current) == 0 {
		return accumulated
	}

	// Map by (type:value) to detect duplicates
	entityMap := make(map[string]models.ExtractedEntity)

	// Add accumulated first
	for _, e := range accumulated {
		key := fmt.Sprintf("%s:%s", e.Type, e.Value)
		entityMap[key] = e
	}

	// Add/update with current (keep highest confidence)
	for _, e := range current {
		key := fmt.Sprintf("%s:%s", e.Type, e.Value)
		if existing, found := entityMap[key]; found {
			// Keep the one with higher confidence
			if e.Confidence > existing.Confidence {
				entityMap[key] = e
				log.Printf("[MessageProcessor] FIX #9: Updated %s:%s confidence (%.2f → %.2f)",
					e.Type, e.Value, existing.Confidence, e.Confidence)
			}
		} else {
			entityMap[key] = e
		}
	}

	// Convert back to slice
	merged := make([]models.ExtractedEntity, 0, len(entityMap))
	for _, e := range entityMap {
		merged = append(merged, e)
	}

	return merged
}

// FIX #10: buildMessageSummary creates a lightweight summary from extraction data
// This is saved immediately after extraction for future re-analysis optimization
func buildMessageSummary(
	messageID string,
	userID string,
	conversationID string,
	messageIndex int,
	role string,
	messageText string,
	epOutput *agents.ExtractionPhaseOutput,
	extractedContext *models.ExtractedContext,
) *models.MessageSummary {
	if epOutput == nil || epOutput.Artifact == nil {
		return nil
	}

	// Extract entity values and types
	var entityValues []string
	var entityTypes []string
	confidence := 0.0
	entityCount := 0

	for _, entity := range epOutput.Artifact.Entities {
		if entity.Confidence > 0.5 { // Only high-confidence entities
			entityValues = append(entityValues, entity.Value)
			entityTypes = append(entityTypes, entity.Type)
			confidence += entity.Confidence
			entityCount++
		}
	}

	// Average confidence
	if entityCount > 0 {
		confidence = confidence / float64(entityCount)
	}

	// Extract key phrases (first 3-5 entities)
	var keyPhrases []string
	for i, v := range entityValues {
		if i >= 5 {
			break
		}
		keyPhrases = append(keyPhrases, v)
	}

	// Get intention and style
	intention := ""
	tone := ""
	commStyle := ""
	if extractedContext != nil {
		intention = extractedContext.Intention
		if extractedContext.Style != nil {
			tone = extractedContext.Style.Style
			commStyle = extractedContext.Style.Style
		}
	}

	summary := &models.MessageSummary{
		MessageID:          messageID,
		UserID:             userID,
		ConversationID:     conversationID,
		MessageIndex:       messageIndex,
		Role:               role,
		MessageLength:      len(messageText),
		ExtractedEntities:  entityValues,
		EntityTypes:        entityTypes,
		Intention:          intention,
		KeyPhrases:         keyPhrases,
		Tone:               tone,
		CommunicationStyle: commStyle,
		Confidence:         confidence,
		ExtractionSource:   epOutput.Artifact.Source,
		TopicShift:         false, // Will be set by orchestrator if needed
		HasClarification:   false, // Will be set by orchestrator if needed
		ProcessedAt:        time.Now().Unix(),
		CreatedAt:          time.Now().Unix(),
		UpdatedAt:          time.Now().Unix(),
	}

	return summary
}

// previousEntities returns the entities from the previous extraction, or none for the first message in a conversation.
func previousEntities(prev *PreviousExtraction) []models.ExtractedEntity {
	if prev == nil {
		return nil
	}
	return prev.Entities
}

// parseArrayFromStorage safely parses array field from database storage
func parseArrayFromStorage(stored string) []string {
	if stored == "" {
		return []string{}
	}

	var result []string
	if err := json.Unmarshal([]byte(stored), &result); err == nil {
		return result
	}
	return []string{}
}

// encodeArrayForStorage encodes array as JSON for safe storage
func encodeArrayForStorage(arr []string) string {
	if len(arr) == 0 {
		return "[]"
	}
	bytes, err := json.Marshal(arr)
	if err != nil {
		log.Printf("[Storage] Error encoding array: %v\n", err)
		return "[]"
	}
	return string(bytes)
}
