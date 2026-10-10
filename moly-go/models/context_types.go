package models

// ContextResponse - Comprehensive context for conversation
type ContextResponse struct {
	ConversationID      string                 `json:"conversationId"`
	AboutMe             *AboutMe               `json:"aboutMe,omitempty"`
	ContactProfile      *Contact               `json:"contactProfile,omitempty"` // Uses Contact from conversation_types
	ConversationHistory []Message              `json:"conversationHistory,omitempty"`
	UserBehaviorProfile *UserBehavioralProfile `json:"userBehaviorProfile,omitempty"`
	RelevantReflections []Reflection           `json:"relevantReflections,omitempty"`
	ContextQuality      ContextQualityMetrics  `json:"contextQuality"`
	MissingContextGaps  []string               `json:"missingContextGaps"`
	Error               string                 `json:"error,omitempty"`
}

// ContextQualityMetrics - Assessment of context completeness
type ContextQualityMetrics struct {
	OverallScore       float64  `json:"overallScore"` // 0-1
	HasAboutMe         bool     `json:"hasAboutMe"`
	HasContactProfile  bool     `json:"hasContactProfile"`
	HasHistory         bool     `json:"hasHistory"`
	HasBehaviorProfile bool     `json:"hasBehaviorProfile"`
	HasReflections     bool     `json:"hasReflections"`
	HistoryLength      int      `json:"historyLength"`
	ReflectionCount    int      `json:"reflectionCount"`
	CompletenessLevel  string   `json:"completenessLevel"` // "complete", "partial", "minimal"
	Recommendations    []string `json:"recommendations"`
}
