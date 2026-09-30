package models

// ExtractionArtifact: Central store for all extraction results from a message
// Single source of truth for entity extraction with rich metadata
type ExtractionArtifact struct {
	// Metadata
	ID             string // "extraction_msg_{timestamp}"
	MessageID      string // Which message this extraction is for
	UserID         string // Which user
	ConversationID string // Which conversation

	// Extraction results
	Entities []ExtractedEntity // The extracted entities (39+ typically from LLM)
	Source   string            // "llm" or "fallback" (LinguisticParser)
	LLMSuccess bool             // Did LLM succeed or did we fallback?
	Duration float64           // Extraction duration in milliseconds

	// Quality metrics
	SubjectAttributed bool    // Do all entities have subject info?
	NegationPreserved bool    // Are negations properly handled?
	AverageConfidence float64 // Average confidence across all entities

	// Timestamps
	CreatedAt int64 // Unix timestamp when extracted
	ExpiresAt int64 // Unix timestamp when this artifact should be cleaned up (TTL)

	// Metadata for debugging/auditing
	Metadata map[string]interface{} // Extra info (LLM model used, etc.)
}

// AmbiguousEntities returns entities that need clarification
func (ea *ExtractionArtifact) AmbiguousEntities() []ExtractedEntity {
	if ea == nil {
		return []ExtractedEntity{}
	}

	var ambiguous []ExtractedEntity
	for _, e := range ea.Entities {
		if e.IsAmbiguous && e.Confidence < 0.75 {
			ambiguous = append(ambiguous, e)
		}
	}
	return ambiguous
}

// HighConfidenceContacts returns contacts extracted with high confidence (>= 0.85)
func (ea *ExtractionArtifact) HighConfidenceContacts() []ExtractedEntity {
	if ea == nil {
		return []ExtractedEntity{}
	}

	var contacts []ExtractedEntity
	for _, e := range ea.Entities {
		if e.Type == "contact" && e.Confidence >= 0.85 {
			contacts = append(contacts, e)
		}
	}
	return contacts
}

// GetEntitiesBySubject returns all entities with a specific subject
func (ea *ExtractionArtifact) GetEntitiesBySubject(subject string) []ExtractedEntity {
	if ea == nil {
		return []ExtractedEntity{}
	}

	var entities []ExtractedEntity
	for _, e := range ea.Entities {
		if e.Subject == subject {
			entities = append(entities, e)
		}
	}
	return entities
}

// GetEntitiesByType returns all entities of a specific type
func (ea *ExtractionArtifact) GetEntitiesByType(entityType string) []ExtractedEntity {
	if ea == nil {
		return []ExtractedEntity{}
	}

	var entities []ExtractedEntity
	for _, e := range ea.Entities {
		if e.Type == entityType {
			entities = append(entities, e)
		}
	}
	return entities
}
