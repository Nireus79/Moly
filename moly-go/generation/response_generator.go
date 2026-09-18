package generation

import (
	"fmt"
	"log"
	"time"

	"moly/database"
	"moly/extraction"
	"moly/models"
)

// ResponseGenerator - Generates appropriate responses based on context
type ResponseGenerator struct {
	extractor  *extraction.MessageExtractor
	detector   *extraction.ConflictDetector
}

// NewResponseGenerator - Create generator
func NewResponseGenerator() *ResponseGenerator {
	return &ResponseGenerator{
		extractor: extraction.NewMessageExtractor(),
		detector:  extraction.NewConflictDetector(),
	}
}

// GenerateResponse - Generate full response for user message
func (rg *ResponseGenerator) GenerateResponse(msg string, ctx *database.UserContextSnapshot) (*models.ConversationResponse, error) {
	log.Printf("[ResponseGenerator] Generating response for message: %d chars", len(msg))

	response := &models.ConversationResponse{
		Phase:    "responding",
		Metadata: make(map[string]interface{}),
	}

	// Extract context from message
	extraction := rg.extractor.Extract(msg, ctx.AboutMe)
	log.Printf("[ResponseGenerator] Extracted: style=%s emotion=%s", extraction.Style, extraction.EmotionalTone)

	// Detect conflicts
	conflicts := rg.detector.Detect(extraction, ctx.AboutMe)
	if len(conflicts) > 0 {
		log.Printf("[ResponseGenerator] Found %d conflicts", len(conflicts))
	}

	// Decision: What should response be?
	if rg.detector.HasSignificantConflict(conflicts) {
		// User has conflicting preferences - ask about it
		question := rg.detector.ConflictToQuestion(conflicts[0])
		response.Response = question
		response.Metadata["type"] = "conflict_question"
		response.Metadata["conflict_field"] = conflicts[0].Field
		return response, nil
	}

	// Generate conversational response
	response.Response = rg.generateConversationalResponse(msg, extraction, ctx)
	response.Metadata["type"] = "conversational"

	log.Printf("[ResponseGenerator] ✓ Response generated: %d chars", len(response.Response))
	return response, nil
}

// generateConversationalResponse - Generate natural conversational response
func (rg *ResponseGenerator) generateConversationalResponse(msg string, extraction *extraction.ExtractedContext, ctx *database.UserContextSnapshot) string {
	// This is a placeholder that generates basic responses
	// Real implementation in Phase 2 will use LLM with adaptive prompts

	// Respect emotional tone
	baseResponse := ""
	switch extraction.EmotionalTone {
	case "very_negative":
		baseResponse = "I can see this is weighing on you. "
	case "negative":
		baseResponse = "That sounds challenging. "
	case "positive", "very_positive":
		baseResponse = "That's great to hear. "
	default:
		baseResponse = ""
	}

	// Add contextual acknowledgment
	if extraction.Topic != "general" {
		baseResponse += fmt.Sprintf("When it comes to %s, ", extraction.Topic)
	} else {
		baseResponse += "In this situation, "
	}

	// Add Socratic question
	baseResponse += "what do you think would be most important to consider? "

	// Respect communication style for phrasing
	if ctx.AboutMe != nil && ctx.AboutMe.CommunicationStyle == "formal" {
		baseResponse = "I understand your situation. " + baseResponse
	} else if ctx.AboutMe != nil && ctx.AboutMe.CommunicationStyle == "playful" {
		baseResponse = "Interesting! " + baseResponse
	} else {
		baseResponse = "That makes sense. " + baseResponse
	}

	return baseResponse
}

// GeneratePendingInputForConflict - Create pending input record for conflict
func (rg *ResponseGenerator) GeneratePendingInputForConflict(userID, conversationID string, conflict extraction.Conflict) *database.PendingInput {
	contextData := map[string]interface{}{
		"old_value": conflict.StoredValue,
		"new_value": conflict.ExtractedValue,
		"field":     conflict.Field,
		"conflict":  conflict.Context,
	}

	return &database.PendingInput{
		UserID:         userID,
		ConversationID: conversationID,
		Type:           "conflict",
		Subtype:        conflict.Type,
		Question:       rg.detector.ConflictToQuestion(conflict),
		Context:        marshalJSON(contextData),
		CreatedAt:      time.Now().Unix(),
	}
}

// GeneratePendingInputForClarification - Create pending input for clarification
func (rg *ResponseGenerator) GeneratePendingInputForClarification(userID, conversationID string, missingField string) *database.PendingInput {
	questions := map[string]string{
		"contact":     "Who are you messaging?",
		"intention":   "What are you trying to accomplish?",
		"style":       "How formal or casual do you want to be?",
		"tone":        "What tone would you like to set?",
	}

	question := questions[missingField]
	if question == "" {
		question = "Could you clarify a bit more about: " + missingField
	}

	contextData := map[string]interface{}{
		"missing_field": missingField,
		"prompt":        question,
	}

	return &database.PendingInput{
		UserID:         userID,
		ConversationID: conversationID,
		Type:           "clarification",
		Subtype:        "missing_" + missingField,
		Question:       question,
		Context:        marshalJSON(contextData),
		CreatedAt:      time.Now().Unix(),
	}
}

// GenerateInsightFromMessage - Create insight/reflection from message
func (rg *ResponseGenerator) GenerateInsightFromMessage(userID, conversationID string, msg string, extraction *extraction.ExtractedContext) *models.Reflection {
	// Placeholder: real implementation will use LLM to extract detailed insights
	return &models.Reflection{
		ConversationID: conversationID,
		Characteristics: []string{
			"expresses " + extraction.EmotionalTone + " emotion",
			"communication style: " + extraction.Style,
		},
		Interests: []string{},
		Intentions: []string{extraction.Intention},
		Status:     "pending_approval",
		CreatedAt:  time.Now().Unix(),
	}
}

// Helper: marshal to JSON
func marshalJSON(data interface{}) []byte {
	// Simple JSON marshaling - real implementation would handle errors
	switch v := data.(type) {
	case map[string]interface{}:
		// Return JSON-like representation
		result := "{"
		first := true
		for k, val := range v {
			if !first {
				result += ","
			}
			result += fmt.Sprintf(`"%s":"%v"`, k, val)
			first = false
		}
		result += "}"
		return []byte(result)
	default:
		return []byte("{}")
	}
}
