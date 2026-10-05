package models

// MessageSummary - Lightweight per-message summary saved immediately after extraction
// Enables future message re-analysis to skip re-extraction of previous messages
type MessageSummary struct {
	ID                 int64    `json:"id"`
	MessageID          string   `json:"messageId"`
	UserID             string   `json:"userId"`
	ConversationID     string   `json:"conversationId"`
	MessageIndex       int      `json:"messageIndex"`       // 1st, 2nd, 3rd message in conversation
	Role               string   `json:"role"`               // "user" or "agent"
	MessageLength      int      `json:"messageLength"`      // Length of original message
	ExtractedEntities  []string `json:"extractedEntities"`  // Entity values only (Sarah, smart, etc)
	EntityTypes        []string `json:"entityTypes"`        // contact, characteristic, value, etc
	Intention          string   `json:"intention"`          // Main goal/purpose
	KeyPhrases         []string `json:"keyPhrases"`         // Important phrases for quick recall
	Tone               string   `json:"tone"`               // casual, formal, playful, mix
	CommunicationStyle string   `json:"communicationStyle"` // From extracted style
	Confidence         float64  `json:"confidence"`         // Average extraction confidence (0-1)
	ExtractionSource   string   `json:"extractionSource"`   // "llm" or "fallback"
	TopicShift         bool     `json:"topicShift"`         // Did topic change from previous message?
	HasClarification   bool     `json:"hasClarification"`   // Is this a clarification response?
	ProcessedAt        int64    `json:"processedAt"`        // Unix timestamp when processed
	CreatedAt          int64    `json:"createdAt"`
	UpdatedAt          int64    `json:"updatedAt"`
}

// GetSummaryForLayers returns a compact version suitable for layer processing
func (m *MessageSummary) GetSummaryForLayers() map[string]interface{} {
	return map[string]interface{}{
		"messageId":          m.MessageID,
		"intention":          m.Intention,
		"entities":           m.ExtractedEntities,
		"entityTypes":        m.EntityTypes,
		"tone":               m.Tone,
		"confidence":         m.Confidence,
		"keyPhrases":         m.KeyPhrases,
		"communicationStyle": m.CommunicationStyle,
	}
}
