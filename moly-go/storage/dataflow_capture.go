package storage

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"moly/database"
	"moly/models"
	"moly/tools"
)

// DataflowCapture handles persistence of all extracted/calculated data
// Fixes dataflow gaps: contacts, intentions, interests, needs, clarifications, context quality, verdict details, response linking
type DataflowCapture struct {
	db *database.Database
}

// NewDataflowCapture creates a new dataflow capture handler
func NewDataflowCapture(db *database.Database) *DataflowCapture {
	return &DataflowCapture{db: db}
}

// CaptureExtractedContact saves extracted contact from message (Gap 1)
func (dc *DataflowCapture) CaptureExtractedContact(userID, conversationID string, contact *models.ExtractedContact) error {
	if contact == nil || contact.Name == "" {
		return nil
	}

	contactRepo := dc.db.GetContactRepository()
	if contactRepo == nil {
		return fmt.Errorf("contact repository unavailable")
	}

	log.Printf("[DataflowCapture] Saving extracted contact: %s (confidence=%.2f)", contact.Name, contact.Confidence)

	// Check if contact exists
	existing, _ := contactRepo.GetByName(userID, contact.Name)

	now := time.Now().Unix()
	if existing != nil {
		// Update existing contact
		existing.LastMentionedAt = now
		existing.ExtractionCount = existing.ExtractionCount + 1
		if contact.Confidence > existing.Confidence {
			existing.Confidence = contact.Confidence
		}
		if existing.Relationship == "" {
			existing.Relationship = contact.Relationship
		}
		return contactRepo.Save(existing)
	}

	// Create new contact
	newContact := &models.Contact{
		UserID:           userID,
		Name:             contact.Name,
		Relationship:     contact.Relationship,
		CreatedVia:       "conversation",
		Status:           "active",
		Confidence:       contact.Confidence,
		FirstMentionedAt: now,
		LastMentionedAt:  now,
		ExtractionCount:  1,
	}

	return contactRepo.Save(newContact)
}

// CaptureExtractedIntention saves user intention (Gap 2)
// Also updates latest_intention in about_me for continuity across messages
func (dc *DataflowCapture) CaptureExtractedIntention(userID, conversationID, messageID string, intention string, confidence float64) error {
	if intention == "" {
		return nil
	}

	conn := dc.db.GetConnection()
	if conn == nil {
		return fmt.Errorf("database connection unavailable")
	}

	log.Printf("[DataflowCapture] Saving extracted intention: %s (confidence=%.2f)", intention, confidence)

	// 1. Insert into user_intentions table (historical tracking)
	_, err := conn.Exec(`
		INSERT INTO user_intentions (user_id, conversation_id, intention, confidence, extracted_from, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, userID, conversationID, intention, confidence, messageID, time.Now().Unix(), time.Now().Unix())

	if err != nil {
		log.Printf("[DataflowCapture] Warning: Failed to save intention history: %v", err)
		// Don't fail - still update about_me
	}

	// 2. Update latest_intention in about_me (for continuity in AnalysisContext)
	_, err = conn.Exec(`
		UPDATE about_me
		SET latest_intention = ?, latest_intention_at = ?
		WHERE user_id = ?
	`, intention, time.Now().Unix(), userID)

	if err != nil {
		log.Printf("[DataflowCapture] Warning: Failed to update about_me latest_intention: %v", err)
	}

	return nil
}

// CaptureExtractedInterestsAndNeeds saves interests and needs (Gap 3)
func (dc *DataflowCapture) CaptureExtractedInterestsAndNeeds(userID string, interests []string, needs []string) error {
	if len(interests) == 0 && len(needs) == 0 {
		return nil
	}

	conn := dc.db.GetConnection()
	if conn == nil {
		return fmt.Errorf("database connection unavailable")
	}

	interestsJSON, _ := json.Marshal(interests)
	needsJSON, _ := json.Marshal(needs)

	log.Printf("[DataflowCapture] Saving interests (%d) and needs (%d)", len(interests), len(needs))

	_, err := conn.Exec(`
		UPDATE about_me
		SET interests = ?, needs = ?
		WHERE user_id = ?
	`, string(interestsJSON), string(needsJSON), userID)

	return err
}

// CaptureClarificationQuestion saves generated clarification questions (Gap 4)
func (dc *DataflowCapture) CaptureClarificationQuestion(userID, conversationID string, interactionID int64, gapType, question string, priority int) error {
	if question == "" {
		return nil
	}

	conn := dc.db.GetConnection()
	if conn == nil {
		return fmt.Errorf("database connection unavailable")
	}

	log.Printf("[DataflowCapture] Saving clarification question for gap: %s", gapType)

	_, err := conn.Exec(`
		INSERT INTO extracted_clarifications (user_id, conversation_id, interaction_id, gap_type, question, priority, asked_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, userID, conversationID, interactionID, gapType, question, priority, time.Now().Unix(), time.Now().Unix())

	return err
}

// CaptureContextQuality saves context quality score (Gap 5)
func (dc *DataflowCapture) CaptureContextQuality(conversationID, userID string, quality float64) error {
	if quality < 0 || quality > 1 {
		return nil
	}

	conn := dc.db.GetConnection()
	if conn == nil {
		return fmt.Errorf("database connection unavailable")
	}

	log.Printf("[DataflowCapture] Saving context quality: %.2f", quality)

	// Update most recent interaction with context quality
	_, err := conn.Exec(`
		UPDATE interactions
		SET context_quality = ?
		WHERE conversation_id = ? AND user_id = ?
		ORDER BY timestamp DESC
		LIMIT 1
	`, quality, conversationID, userID)

	return err
}

// CaptureSafetyVerdictDetails saves complete verdict information (Gap 6)
func (dc *DataflowCapture) CaptureSafetyVerdictDetails(userID string, verdict *tools.ConstitutionalVerdict, responseText string) error {
	if verdict == nil {
		return nil
	}

	conn := dc.db.GetConnection()
	if conn == nil {
		return fmt.Errorf("database connection unavailable")
	}

	log.Printf("[DataflowCapture] Saving safety verdict details: severity=%s, confidence=%.2f", verdict.OverallSeverity, verdict.Confidence)

	// Prepare JSON fields from MatchedPrinciples
	violatedPrinciples := []string{}
	evidenceSnippets := []string{}
	if verdict.MatchedPrinciples != nil {
		for _, match := range verdict.MatchedPrinciples {
			violatedPrinciples = append(violatedPrinciples, match.PrincipleID)
			evidenceSnippets = append(evidenceSnippets, match.Evidence)
		}
	}
	principlesJSON, _ := json.Marshal(violatedPrinciples)
	evidenceJSON, _ := json.Marshal(evidenceSnippets)

	// Update most recent safety incident with detailed verdict
	_, err := conn.Exec(`
		UPDATE safety_incidents
		SET verdict_reasoning = ?, violated_principles = ?, evidence_snippets = ?, response_text = ?, context_maturity = ?
		WHERE user_id = ?
		ORDER BY detected_at DESC
		LIMIT 1
	`, verdict.Reasoning, string(principlesJSON), string(evidenceJSON), responseText, verdict.ContextMaturity, userID)

	if err != nil {
		log.Printf("[DataflowCapture] Warning: Failed to save verdict details: %v", err)
	}

	return nil
}

// LinkResponseToInteraction links agent response back to user interaction (Gap 7)
func (dc *DataflowCapture) LinkResponseToInteraction(conversationID, userID, responseID string, responseMetadata interface{}) error {
	if responseID == "" {
		return nil
	}

	conn := dc.db.GetConnection()
	if conn == nil {
		return fmt.Errorf("database connection unavailable")
	}

	log.Printf("[DataflowCapture] Linking response %s to interaction", responseID)

	// Prepare metadata JSON
	var metadataJSON string
	if responseMetadata != nil {
		b, _ := json.Marshal(responseMetadata)
		metadataJSON = string(b)
	}

	// Update most recent interaction with response link
	_, err := conn.Exec(`
		UPDATE interactions
		SET response_id = ?, phase_results = ?
		WHERE conversation_id = ? AND user_id = ?
		ORDER BY timestamp DESC
		LIMIT 1
	`, responseID, metadataJSON, conversationID, userID)

	return err
}

// CaptureInsightsFromFailure records learning even when safety blocks response (Gap 8)
func (dc *DataflowCapture) CaptureInsightsFromFailure(userID, conversationID string, blockReason string) error {
	conn := dc.db.GetConnection()
	if conn == nil {
		return fmt.Errorf("database connection unavailable")
	}

	log.Printf("[DataflowCapture] Recording safety block for learning: %s", blockReason)

	// Insert blocked interaction record
	now := time.Now().Unix()
	metadata := map[string]interface{}{
		"blocked_by_safety": true,
		"block_reason":      blockReason,
	}
	metadataJSON, _ := json.Marshal(metadata)

	_, err := conn.Exec(`
		INSERT INTO interactions (user_id, conversation_id, content, type, timestamp, metadata)
		VALUES (?, ?, ?, ?, ?, ?)
	`, userID, conversationID, "SAFETY_BLOCKED", "system_block", now, string(metadataJSON))

	return err
}

// DocumentRiskProfileSource documents origin of risk profile (Gap 9)
func (dc *DataflowCapture) DocumentRiskProfileSource(userID string, profileData map[string]interface{}, source string) error {
	conn := dc.db.GetConnection()
	if conn == nil {
		return fmt.Errorf("database connection unavailable")
	}

	log.Printf("[DataflowCapture] Documenting risk profile source: %s", source)

	profileJSON, _ := json.Marshal(profileData)

	_, err := conn.Exec(`
		INSERT INTO behavior_patterns (user_id, growth_trend, last_analyzed)
		VALUES (?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			growth_trend = ?, last_analyzed = ?
	`, userID, string(profileJSON), time.Now().Unix(), string(profileJSON), time.Now().Unix())

	return err
}

// CaptureAllDataflows is a batch operation for message processing
func (dc *DataflowCapture) CaptureAllDataflows(data *DataflowData) error {
	log.Printf("[DataflowCapture] Batch capturing %d dataflows", countDataflows(data))

	if data.ExtractedContact != nil {
		if err := dc.CaptureExtractedContact(data.UserID, data.ConversationID, data.ExtractedContact); err != nil {
			log.Printf("[DataflowCapture] Error capturing contact: %v", err)
		}
	}

	if data.ExtractedIntention != "" {
		if err := dc.CaptureExtractedIntention(data.UserID, data.ConversationID, data.MessageID, data.ExtractedIntention, data.IntentionConfidence); err != nil {
			log.Printf("[DataflowCapture] Error capturing intention: %v", err)
		}
	}

	if len(data.ExtractedInterests) > 0 || len(data.ExtractedNeeds) > 0 {
		if err := dc.CaptureExtractedInterestsAndNeeds(data.UserID, data.ExtractedInterests, data.ExtractedNeeds); err != nil {
			log.Printf("[DataflowCapture] Error capturing interests/needs: %v", err)
		}
	}

	if data.ContextQuality >= 0 && data.ContextQuality <= 1 {
		if err := dc.CaptureContextQuality(data.ConversationID, data.UserID, data.ContextQuality); err != nil {
			log.Printf("[DataflowCapture] Error capturing context quality: %v", err)
		}
	}

	if data.SafetyVerdict != nil {
		if err := dc.CaptureSafetyVerdictDetails(data.UserID, data.SafetyVerdict, data.ResponseText); err != nil {
			log.Printf("[DataflowCapture] Error capturing verdict details: %v", err)
		}
	}

	if data.ResponseID != "" {
		if err := dc.LinkResponseToInteraction(data.ConversationID, data.UserID, data.ResponseID, data.ResponseMetadata); err != nil {
			log.Printf("[DataflowCapture] Error linking response: %v", err)
		}
	}

	return nil
}

// DataflowData holds all data to be persisted in one batch
type DataflowData struct {
	UserID                string
	ConversationID        string
	MessageID             string
	ExtractedContact      *models.ExtractedContact
	ExtractedIntention    string
	IntentionConfidence   float64
	ExtractedInterests    []string
	ExtractedNeeds        []string
	ContextQuality        float64
	SafetyVerdict         *tools.ConstitutionalVerdict
	ResponseText          string
	ResponseID            string
	ResponseMetadata      interface{}
	ClarificationQuestion string
	ClarificationGap      string
	ClarificationPriority int
}

// countDataflows counts how many dataflows are being captured
func countDataflows(data *DataflowData) int {
	count := 0
	if data.ExtractedContact != nil {
		count++
	}
	if data.ExtractedIntention != "" {
		count++
	}
	if len(data.ExtractedInterests) > 0 || len(data.ExtractedNeeds) > 0 {
		count++
	}
	if data.ContextQuality >= 0 {
		count++
	}
	if data.SafetyVerdict != nil {
		count++
	}
	if data.ResponseID != "" {
		count++
	}
	if data.ClarificationQuestion != "" {
		count++
	}
	return count
}
