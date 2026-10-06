package agents

import (
	"log"
	"moly/database"
	"moly/tools"
)

// AnswerProcessor handles user responses to clarification questions
type AnswerProcessor struct {
	clarificationAgent  *ClarificationAgent
	contextAttrRepo     *database.ContextAttributeRepository
}

// NewAnswerProcessor creates a new answer processor
func NewAnswerProcessor(clarificationAgent *ClarificationAgent, db *database.Database) *AnswerProcessor {
	return &AnswerProcessor{
		clarificationAgent: clarificationAgent,
		contextAttrRepo:    database.NewContextAttributeRepository(db),
	}
}

// ProcessResponse handles a user's answer to a clarification question
func (ap *AnswerProcessor) ProcessResponse(
	questionID string,
	userAnswer string,
	linkedFacts []ExtractedFact,
	userID string,
	conversationID string,
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
	if err := tools.SafeJSONParse("AnswerProcessor", []byte(processingResult.RawResponse), &extractedData); err != nil {
		log.Printf("[AnswerProcessor] WARNING: LLM response JSON parse failed: %v - response content: %s - treating user answer as plain text", err, processingResult.RawResponse)
		// Continue with text response (extractedData remains empty/nil)
	}

	// Step 3: Extract context to save
	contextToSave := ap.buildContextFromAnswer(
		userAnswer,
		extractedData,
		linkedFacts,
	)

	// Step 4: Check for conflicts with existing context attributes
	conflicts := ap.detectConflicts(contextToSave, userID, conversationID)

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
		Status:                 "processed",
		MolyReply:              molyReply,
		ContextToSave:          contextToSave,
		Conflicts:              conflicts,
		NeedsMoreClarification: needsMore,
		ExtractedFacts:         extractedData,
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

// detectConflicts checks if answer conflicts with existing context attributes
func (ap *AnswerProcessor) detectConflicts(
	contextToSave map[string]interface{},
	userID string,
	conversationID string,
) []interface{} {
	if userID == "" {
		return []interface{}{}
	}

	log.Printf("[AnswerProcessor] Checking for conflicts in context to save for user %s", userID)

	var conflicts []interface{}

	// Extract fact types from context to save
	// Look for common fields: "facts", "patterns", "context", "style", "tone", "values", "goals"
	if facts, ok := contextToSave["facts"].([]interface{}); ok {
		for _, factRaw := range facts {
			if factMap, ok := factRaw.(map[string]interface{}); ok {
				if factType, ok := factMap["type"].(string); ok {
					if factValue, ok := factMap["value"].(string); ok {
						// Query existing attributes with this fact type for the user subject
						existing, err := ap.contextAttrRepo.GetByType(userID, "user", factType)
						if err != nil {
							log.Printf("[AnswerProcessor] Warning: failed to check existing attributes for %s: %v", factType, err)
							continue
						}

						// Check each existing attribute for conflicts
						for _, attr := range existing {
							if attr.FactValue != factValue {
								conflict := map[string]interface{}{
									"type":        "context_fact_conflict",
									"factType":    factType,
									"savedValue":  attr.FactValue,
									"newValue":    factValue,
									"severity":    "medium",
									"description": "New answer differs from previously stated preference",
									"confidence":  attr.Confidence,
								}
								conflicts = append(conflicts, conflict)
								log.Printf("[AnswerProcessor] ⚠️  Conflict detected: %s changed from '%s' to '%s'", factType, attr.FactValue, factValue)
							}
						}
					}
				}
			}
		}
	}

	// Check direct fields like "style", "tone", "values"
	fieldTypes := []string{"style", "tone", "values", "goals", "context"}
	for _, fieldType := range fieldTypes {
		if newValue, ok := contextToSave[fieldType].(string); ok && newValue != "" {
			existing, err := ap.contextAttrRepo.GetByType(userID, "user", fieldType)
			if err != nil {
				log.Printf("[AnswerProcessor] Warning: failed to check existing %s: %v", fieldType, err)
				continue
			}

			for _, attr := range existing {
				if attr.FactValue != newValue {
					conflict := map[string]interface{}{
						"type":        "preference_conflict",
						"factType":    fieldType,
						"savedValue":  attr.FactValue,
						"newValue":    newValue,
						"severity":    "medium",
						"description": "New preference differs from previously stated",
					}
					conflicts = append(conflicts, conflict)
					log.Printf("[AnswerProcessor] ⚠️  Conflict detected: %s changed from '%s' to '%s'", fieldType, attr.FactValue, newValue)
				}
			}
		}
	}

	if len(conflicts) > 0 {
		log.Printf("[AnswerProcessor] ⚠️  Found %d conflicts", len(conflicts))
	} else {
		log.Printf("[AnswerProcessor] ✓ No conflicts detected")
	}

	return conflicts
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
