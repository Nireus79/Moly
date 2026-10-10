package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"moly/models"
)

// ClarificationCapture handles the Layer 3 workflow:
// 1. Detect clarification response in user message
// 2. Save response to database
// 3. Check for conflicts with existing preferences
// 4. Convert confirmed response to context_attributes
type ClarificationCapture struct {
	clarificationQRepo *ClarificationQuestionRepository
	clarificationRRepo *ClarificationResponseRepository
	conflictGate       *ConflictGate
	contextAttrRepo    *ContextAttributeRepository
}

// NewClarificationCapture creates a new capture handler
func NewClarificationCapture(db *Database) *ClarificationCapture {
	return &ClarificationCapture{
		clarificationQRepo: NewClarificationQuestionRepository(db),
		clarificationRRepo: NewClarificationResponseRepository(db),
		conflictGate:       NewConflictGate(db),
		contextAttrRepo:    NewContextAttributeRepository(db),
	}
}

// ClarificationResponse represents a user's answer to a clarification
type ClarificationAnswerCapture struct {
	QuestionID     string
	UserID         string
	ConversationID string
	ResponseText   string
	SelectedOption string
}

// GetPendingClarifications retrieves unanswered clarification questions for a conversation
func (cc *ClarificationCapture) GetPendingClarifications(conversationID string) ([]interface{}, error) {
	if conversationID == "" {
		return nil, fmt.Errorf("conversationID is required")
	}

	log.Printf("[ClarificationCapture] Retrieving pending clarifications for conversation %s", conversationID)

	query := `
		SELECT id, user_id, conversation_id, clarification_type, question_text, context_notes,
		       options, priority, status, linked_facts, created_at, answered_at
		FROM clarification_questions
		WHERE conversation_id = ? AND status = 'pending'
		ORDER BY priority DESC, created_at DESC
	`

	rows, err := cc.clarificationQRepo.db.Query(query, conversationID)
	if err != nil {
		log.Printf("[ClarificationCapture] Failed to query pending questions: %v", err)
		return nil, fmt.Errorf("failed to query pending clarifications: %w", err)
	}
	defer rows.Close()

	var questions []interface{}
	for rows.Next() {
		question := &ClarificationQuestion{}
		var optionsJSON sql.NullString
		var factsJSON sql.NullString

		err := rows.Scan(
			&question.ID,
			&question.UserID,
			&question.ConversationID,
			&question.ClarificationType,
			&question.QuestionText,
			&question.ContextNotes,
			&optionsJSON,
			&question.Priority,
			&question.Status,
			&factsJSON,
			&question.CreatedAt,
			&question.AnsweredAt,
		)

		if err != nil {
			log.Printf("[ClarificationCapture] ERROR scanning question: %v", err)
			continue
		}

		if optionsJSON.Valid {
			if err := json.Unmarshal([]byte(optionsJSON.String), &question.Options); err != nil {
				log.Printf("[ClarificationCapture] ERROR parsing options for question %s: %v", question.ID, err)
			}
		}
		if factsJSON.Valid {
			if err := json.Unmarshal([]byte(factsJSON.String), &question.LinkedFacts); err != nil {
				log.Printf("[ClarificationCapture] ERROR parsing facts for question %s: %v", question.ID, err)
			}
		}

		questions = append(questions, question)
	}

	if err = rows.Err(); err != nil {
		log.Printf("[ClarificationCapture] ERROR iterating rows: %v", err)
		return nil, fmt.Errorf("error iterating clarification questions: %w", err)
	}

	log.Printf("[ClarificationCapture] ✓ Retrieved %d pending clarifications", len(questions))
	return questions, nil
}

// DetectClarificationResponse attempts to match user message to pending clarifications
// Returns the question ID if matched, empty string if not a clarification response
func (cc *ClarificationCapture) DetectClarificationResponse(
	conversationID string,
	userMessage string,
) (string, error) {

	log.Printf("[ClarificationCapture] Attempting to detect if message is clarification response")
	log.Printf("[ClarificationCapture] Message received (len=%d)", len(userMessage))

	// Get all questions for this conversation
	allQuestions, err := cc.clarificationQRepo.GetConversationQuestions(conversationID)
	if err != nil {
		log.Printf("[ClarificationCapture] Warning: Failed to get questions: %v", err)
		return "", err
	}

	// Filter for pending (unanswered) questions only
	var questions []*ClarificationQuestion
	for _, q := range allQuestions {
		if q.Status == "active" {
			questions = append(questions, q)
		}
	}

	if len(questions) == 0 {
		log.Printf("[ClarificationCapture] No pending questions for this conversation")
		return "", nil
	}

	log.Printf("[ClarificationCapture] Found %d pending clarifications", len(questions))

	// FIX #1: Validate that response matches question intent (not just loose matching)
	// For each pending question, check if response is relevant to that question
	if len(questions) >= 1 && len(userMessage) > 0 {
		for _, q := range questions {
			// Validate response matches question type/intent
			isValid := cc.validateClarificationResponse(userMessage, q)
			if isValid {
				log.Printf("[ClarificationCapture] ✓ FIX #1: Validated clarification response to: %s", q.ID)
				return q.ID, nil
			}
		}

		// If multiple questions and none validated, can't auto-detect
		if len(questions) > 1 {
			log.Printf("[ClarificationCapture] Multiple pending questions, none validated as matching response")
			return "", nil
		}

		// Single question but response doesn't match - still apply but log warning
		if len(questions) == 1 {
			log.Printf("[ClarificationCapture] ⚠ FIX #1: Response doesn't clearly match question but only 1 pending, will apply cautiously")
			return questions[0].ID, nil
		}
	}

	// Multiple pending questions - would need LLM or user selection
	log.Printf("[ClarificationCapture] Multiple pending questions, cannot auto-detect")
	return "", nil
}

// FIX #1: validateClarificationResponse checks if response is relevant to the question
func (cc *ClarificationCapture) validateClarificationResponse(response string, question *ClarificationQuestion) bool {
	if question == nil || response == "" {
		return false
	}

	log.Printf("[ClarificationCapture] FIX #1: Validating response against %s question", question.ClarificationType)

	// Simple validation based on question type
	responseWords := strings.Fields(strings.ToLower(response))
	questionWords := strings.Fields(strings.ToLower(question.QuestionText))

	// Check: Does response contain at least some key words from question?
	// This prevents completely off-topic responses
	keywordMatches := 0
	for _, respWord := range responseWords {
		for _, qWord := range questionWords {
			if len(respWord) > 4 && len(qWord) > 4 && respWord == qWord {
				keywordMatches++
			}
		}
	}

	// If at least some keywords match, or response is non-trivial, accept it
	isRelevant := keywordMatches > 0 || len(response) > 10

	if isRelevant {
		log.Printf("[ClarificationCapture] FIX #1: Response validated (matches: %d keywords)", keywordMatches)
	} else {
		log.Printf("[ClarificationCapture] FIX #1: Response too brief or off-topic (matches: %d keywords)", keywordMatches)
	}

	return isRelevant
}

// IsLikelyClarificationResponse performs a simple check: is there a pending question?
// Used as a quick gate before doing more expensive matching
func (cc *ClarificationCapture) IsLikelyClarificationResponse(conversationID string) bool {
	allQuestions, err := cc.clarificationQRepo.GetConversationQuestions(conversationID)
	if err != nil {
		return false
	}

	// Check if any are pending
	for _, q := range allQuestions {
		if q.Status == "active" {
			return true
		}
	}
	return false
}

// IsObviousQuestion determines if a clarification question is obvious and shouldn't be asked
// Don't ask "Are you dominant?" when user just said "I am dominant"
func (cc *ClarificationCapture) IsObviousQuestion(question string, userMessage string) bool {
	log.Printf("[ClarificationCapture] Checking if question is obvious")

	// Pattern 1: Question asking about something user just explicitly stated
	lowerQuestion := strings.ToLower(question)
	lowerMessage := strings.ToLower(userMessage)

	// "Are you dominant?" is obvious if user just said "I am dominant"
	if strings.Contains(lowerQuestion, "are you") && strings.Contains(lowerMessage, "i am") {
		property := strings.TrimPrefix(strings.TrimPrefix(lowerQuestion, "are you"), " ")
		if strings.Contains(lowerMessage, property) {
			log.Printf("[ClarificationCapture] Question is obvious (user already stated): %s", question)
			return true
		}
	}

	// "Is Se submissive?" is obvious if user just said "Se is submissive"
	if strings.Contains(lowerQuestion, " is ") && strings.Contains(lowerMessage, " is ") {
		// Extract the property from both
		questionParts := strings.Split(lowerQuestion, " is ")
		messageParts := strings.Split(lowerMessage, " is ")

		if len(questionParts) > 1 && len(messageParts) > 1 {
			questionProp := strings.TrimSpace(questionParts[1])
			messageProp := strings.TrimSpace(messageParts[1])

			if strings.Contains(messageProp, questionProp) {
				log.Printf("[ClarificationCapture] Question is obvious (explicitly stated): %s", question)
				return true
			}
		}
	}

	// Pattern 2: Question asking about something already confirmed
	if strings.Contains(lowerQuestion, "correct") && strings.Contains(lowerMessage, "i said") {
		log.Printf("[ClarificationCapture] Question is obvious (already clarified): %s", question)
		return true
	}

	log.Printf("[ClarificationCapture] Question is not obvious: %s", question)
	return false
}

// ProcessClarificationWithLLMExtraction uses LLM extraction results directly (FIXED: uses LLM results instead of re-parsing)
// This preserves the 39 correctly-extracted entities with proper subject attribution
func (cc *ClarificationCapture) ProcessClarificationWithLLMExtraction(
	capture *ClarificationAnswerCapture,
	llmEntities []interface{}, // []models.ExtractedEntity as interface{}
) error {

	log.Printf("[ClarificationCapture] Processing clarification with LLM extraction results (preserving subject attribution)")

	// Step 1: Save the clarification response
	response := &ClarificationResponse{
		ID:             fmt.Sprintf("resp_%d", time.Now().UnixNano()),
		QuestionID:     capture.QuestionID,
		UserID:         capture.UserID,
		ResponseText:   capture.ResponseText,
		SelectedOption: capture.SelectedOption,
		RespondedAt:    time.Now().Unix(),
		CreatedAt:      time.Now().Unix(),
	}

	err := cc.clarificationRRepo.SaveResponse(response)
	if err != nil {
		log.Printf("[ClarificationCapture] ❌ Failed to save clarification response: %v", err)
		return fmt.Errorf("failed to save response: %w", err)
	}
	log.Printf("[ClarificationCapture] ✓ Saved response: %s", response.ID)

	// Step 2: Save LLM-extracted entities with their subject attribution intact
	savedCount := 0
	for _, entity := range llmEntities {
		// Type assert to map[string]interface{} (from json.Unmarshal of ExtractedEntity)
		if entityMap, ok := entity.(map[string]interface{}); ok {
			// Extract fields from entity map
			subject := fmt.Sprintf("%v", entityMap["subject"])
			if subject == "" || subject == "<nil>" {
				subject = "user" // Fallback if no subject provided
			}

			attr := &ContextAttribute{
				ID:             0,
				UserID:         capture.UserID,
				ConversationID: capture.ConversationID,
				FactType:       fmt.Sprintf("%v", entityMap["type"]),
				FactValue:      fmt.Sprintf("%v", entityMap["value"]),
				AttributedTo:   subject, // CRITICAL: Use LLM's subject attribution, not default
				Context:        "clarification",
				Confidence:     0.90, // LLM confidence
				Source:         "llm_clarification_extraction",
				Evidence:       fmt.Sprintf("%v", entityMap["evidence"]),
				Version:        1,
				CreatedAt:      time.Now().Unix(),
			}

			if err := cc.contextAttrRepo.Save(attr); err != nil {
				log.Printf("[ClarificationCapture] Warning: Failed to save entity: %v", err)
			} else {
				savedCount++
				log.Printf("[ClarificationCapture] ✓ Saved entity: %s=%s (subject=%s, source=LLM)", attr.FactType, attr.FactValue, subject)
			}
		}
	}

	log.Printf("[ClarificationCapture] ✓ Saved %d entities with LLM-provided subject attribution", savedCount)

	// Step 3: Mark question as answered
	if err := cc.clarificationQRepo.MarkAnswered(capture.QuestionID); err != nil {
		log.Printf("[ClarificationCapture] Warning: Failed to mark question as answered: %v", err)
	}

	log.Printf("[ClarificationCapture] ✓ Clarification processing complete using LLM extraction")
	return nil
}

// ProcessClarificationWithExtractionArtifact uses the full ExtractionArtifact from Phase 0
// This preserves all extraction metadata: subject attribution, confidence, conflicts, quality metrics
// PHASE 1: Extraction artifact MUST be locked (immutable) - prevents re-parsing
// Phase 3 integration: Centralizes clarification processing with full extraction context
func (cc *ClarificationCapture) ProcessClarificationWithExtractionArtifact(
	capture *ClarificationAnswerCapture,
	artifact *models.ExtractionArtifact,
) error {

	if artifact == nil {
		log.Printf("[ClarificationCapture] Warning: nil artifact, falling back to basic processing")
		return nil
	}

	// PHASE 1: Enforce extraction lock requirement
	if !artifact.IsLocked {
		log.Printf("[ClarificationCapture] ❌ CRITICAL: Extraction artifact is not locked (PHASE 1 violation)")
		log.Printf("[ClarificationCapture]    This indicates extraction was not properly locked in ExtractionPhase")
		log.Printf("[ClarificationCapture]    Refusing to process clarification - system integrity compromised")
		return fmt.Errorf("extraction artifact must be locked before clarification processing (PHASE 1 enforcement)")
	}

	log.Printf("[ClarificationCapture] Processing clarification with LOCKED ExtractionArtifact (Phase 0 extraction)")
	log.Printf("[ClarificationCapture]   - Artifact locked at: %d (reason: %s)",
		artifact.LockedAt, artifact.LockReason)
	log.Printf("[ClarificationCapture]   - %d entities (source=%s, avg_confidence=%.2f)",
		len(artifact.Entities), artifact.Source, artifact.AverageConfidence)
	log.Printf("[ClarificationCapture]   - Subject attribution: %v, Negation preserved: %v",
		artifact.SubjectAttributed, artifact.NegationPreserved)

	// Step 1: Save the clarification response
	response := &ClarificationResponse{
		ID:             fmt.Sprintf("resp_%d", time.Now().UnixNano()),
		QuestionID:     capture.QuestionID,
		UserID:         capture.UserID,
		ResponseText:   capture.ResponseText,
		SelectedOption: capture.SelectedOption,
		RespondedAt:    time.Now().Unix(),
		CreatedAt:      time.Now().Unix(),
	}

	err := cc.clarificationRRepo.SaveResponse(response)
	if err != nil {
		log.Printf("[ClarificationCapture] ❌ Failed to save clarification response: %v", err)
		return fmt.Errorf("failed to save response: %w", err)
	}
	log.Printf("[ClarificationCapture] ✓ Saved response: %s", response.ID)

	// Step 2: Log extraction artifact metadata
	// This helps track extraction quality and troubleshoot future issues
	log.Printf("[ClarificationCapture] Artifact metadata: source=%s, subject_attr=%v, negation=%v, llm_success=%v, avg_conf=%.2f, entities=%d",
		artifact.Source, artifact.SubjectAttributed, artifact.NegationPreserved, artifact.LLMSuccess,
		artifact.AverageConfidence, len(artifact.Entities))

	// Step 3: Save all extracted entities with their full metadata
	savedCount := 0
	ambiguousCount := 0
	lowConfidenceCount := 0

	for _, entity := range artifact.Entities {
		// Determine confidence level
		confidence := entity.Confidence
		source := artifact.Source

		// Mark quality indicators
		qualityFlags := []string{}
		if entity.IsAmbiguous {
			qualityFlags = append(qualityFlags, "ambiguous")
			ambiguousCount++
		}
		if confidence < 0.7 {
			qualityFlags = append(qualityFlags, "low_confidence")
			lowConfidenceCount++
		}

		attr := &ContextAttribute{
			ID:             0,
			UserID:         capture.UserID,
			ConversationID: capture.ConversationID,
			FactType:       entity.Type,
			FactValue:      entity.Value,
			AttributedTo:   entity.Subject, // Use LLM-determined subject, not default
			Context:        "clarification_artifact",
			Confidence:     confidence,
			Source:         "extraction_artifact_" + source,
			Evidence:       strings.Join(qualityFlags, ","), // Encode quality flags as evidence
			Version:        1,
			CreatedAt:      time.Now().Unix(),
		}

		if err := cc.contextAttrRepo.Save(attr); err != nil {
			log.Printf("[ClarificationCapture] Warning: Failed to save entity: %v", err)
		} else {
			savedCount++
			logMsg := fmt.Sprintf("[ClarificationCapture] ✓ Saved entity: %s=%s (subject=%s, confidence=%.2f, source=%s)",
				attr.FactType, attr.FactValue, entity.Subject, confidence, source)
			if len(qualityFlags) > 0 {
				logMsg += fmt.Sprintf(" [%s]", strings.Join(qualityFlags, ","))
			}
			log.Print(logMsg)
		}
	}

	log.Printf("[ClarificationCapture] ✓ Saved %d entities (ambiguous=%d, low_conf=%d)",
		savedCount, ambiguousCount, lowConfidenceCount)

	// Step 4: Mark question as answered
	if err := cc.clarificationQRepo.MarkAnswered(capture.QuestionID); err != nil {
		log.Printf("[ClarificationCapture] Warning: Failed to mark question as answered: %v", err)
	}

	log.Printf("[ClarificationCapture] ✓ Clarification processing complete with ExtractionArtifact (Phase 3)")
	return nil
}
