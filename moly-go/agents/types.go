package agents

// ExtractedFact represents a fact extracted from user message
type ExtractedFact struct {
	ID              string  `json:"id"`
	Value           string  `json:"value"`           // "casual", "detail-oriented"
	FactType        string  `json:"factType"`        // "trait", "style", "value", "preference", "goal"
	ProposedSubject string  `json:"proposedSubject"` // "user", "she", "boss", "they"
	Confidence      float64 `json:"confidence"`      // 0-1
	Evidence        string  `json:"evidence"`        // Exact quote from message
	CreatedAt       int64   `json:"createdAt"`
}
