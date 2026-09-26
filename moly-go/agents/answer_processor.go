package agents

import (
	"encoding/json"
	"log"
)

// AnswerProcessor handles user responses to clarification questions
type AnswerProcessor struct {
	clarificationAgent *ClarificationAgent
}

// NewAnswerProcessor creates a new answer processor
func NewAnswerProcessor(clarificationAgent *ClarificationAgent) *AnswerProcessor {
	return &AnswerProcessor{
		clarificationAgent: clarificationAgent,
	}
}

// ProcessResponse handles a user's answer to a clarification question
func (ap *AnswerProcessor) ProcessResponse(
	questionID string,
	userAnswer string,
	linkedFacts []ExtractedFact,
	userID string,
) (*ProcessedAnswerResponse, error) {
	log.Printf("[AnswerProcessor] Processing response from user %s to question %s", userID, questionID)

	// Step 1: LLM processes the answer
	processingResult, err := ap.clarificationAgent.ProcessAnswer(
		"Clarifying your communication context",
		userAnswer,
		linkedFacts,
	)
	if err != nil {
		log.Printf("[AnswerProcessor] Error processing answer: %v", err)
		return nil, err
	}

	// Step 2: Parse LLM's response
	var extractedData map[string]interface{}
	if err := json.Unmarshal([]byte(processingResult.RawResponse), &extractedData); err != nil {
		log.Printf("[AnswerProcessor] WARNING: LLM response JSON parse failed: %v - response content: %s - treating user answer as plain text", err, processingResult.RawResponse)
		// Continue with text response (extractedData remains empty/nil)
	}

	// Step 3: Extract context to save
	contextToSave := ap.buildContextFromAnswer(
		userAnswer,
		extractedData,
		linkedFacts,
	)

	// Step 4: Check for conflicts with existing About Me (simplified for now)
	var conflicts []interface{} // Empty for now (TODO: implement full conflict detection)

	// Step 5: Generate Moly's acknowledgment
	molyReply, err := ap.clarificationAgent.ConfirmUnderstanding(
		"Your communication preference",
		userAnswer,
	)
	if err != nil {
		molyReply = "Got it, thanks for clarifying that!"
	}

	// Step 6: Determine if more clarification needed
	needsMore := false
	if extractedData != nil {
		if moreFlag, ok := extractedData["needsMoreClarification"].(bool); ok {
			needsMore = moreFlag
		}
	}

	response := &ProcessedAnswerResponse{
		Status:                   "processed",
		MolyReply:                molyReply,
		ContextToSave:            contextToSave,
		Conflicts:                conflicts,
		NeedsMoreClarification:   needsMore,
		ExtractedFacts:           extractedData,
	}

	log.Printf("[AnswerProcessor] ✓ Response processed")
	return response, nil
}

// buildContextFromAnswer extracts what we learned from the answer
func (ap *AnswerProcessor) buildContextFromAnswer(
	userAnswer string,
	extractedData map[string]interface{},
	linkedFacts []ExtractedFact,
) map[string]interface{} {
	context := make(map[string]interface{})

	// Extract structured data if available
	if extractedData != nil {
		if facts, ok := extractedData["factsExtracted"].([]interface{}); ok {
			context["facts"] = facts
		}
		if patterns, ok := extractedData["patterns"].([]interface{}); ok {
			context["patterns"] = patterns
		}
		if ctxStr, ok := extractedData["contextToSave"].(string); ok {
			context["context"] = ctxStr
		}
	}

	// Always include raw answer for reference
	context["rawAnswer"] = userAnswer

	// Link to facts this clarified
	if len(linkedFacts) > 0 {
		factIDs := make([]string, len(linkedFacts))
		for i, f := range linkedFacts {
			factIDs[i] = f.ID
		}
		context["clarifiedFacts"] = factIDs
	}

	return context
}

// detectConflicts checks if answer conflicts with existing About Me
func (ap *AnswerProcessor) detectConflicts(
	contextToSave map[string]interface{},
	userID string,
) []interface{} {
	// For now, skip conflict detection in answer processor
	// Conflicts will be detected at the About Me update level
	// This is a simplified version - full conflict detection can be added later
	return []interface{}{}
}

// ProcessedAnswerResponse is sent back to the frontend
type ProcessedAnswerResponse struct {
	Status                 string                 `json:"status"`
	MolyReply              string                 `json:"molyReply"`
	ContextToSave          map[string]interface{} `json:"contextToSave"`
	Conflicts              []interface{}          `json:"conflicts,omitempty"`
	NeedsMoreClarification bool                   `json:"needsMore"`
	ExtractedFacts         map[string]interface{} `json:"extractedFacts,omitempty"`
}