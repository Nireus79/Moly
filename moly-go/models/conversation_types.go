package models

// ConversationRequest - Request to generate conversation response
type ConversationRequest struct {
	ConversationID string                 `json:"conversationId" binding:"required"`
	UserID         string                 `json:"userId" binding:"required"`
	UserMessage    string                 `json:"userMessage" binding:"required"`
	Mode           string                 `json:"mode"` // "socratic", "direct"
	Tone           string                 `json:"tone"` // "formal", "friendly", "dating"
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// ConversationResponse - Response with suggestions and metadata
type ConversationResponse struct {
	Phase                string                 `json:"phase"` // "suggestions_ready", "context_gathering", "safety_alert", "error"
	Suggestions          []Suggestion           `json:"suggestions,omitempty"`
	Questions            []string               `json:"questions,omitempty"`
	Reflection           *Reflection            `json:"reflection,omitempty"`
	RiskWarning          *RiskWarning           `json:"riskWarning,omitempty"`
	SafetyAlert          *SafetyAlert           `json:"safetyAlert,omitempty"`
	ConstitutionConcerns *ConstitutionAnalysis  `json:"constitutionConcerns,omitempty"`
	ProcessingTimeMs     int                    `json:"processingTimeMs"`
	Metadata             map[string]interface{} `json:"metadata,omitempty"`
	Error                string                 `json:"error,omitempty"`
}

// Suggestion - Generated communication suggestion
type Suggestion struct {
	Index      int     `json:"index"`
	Text       string  `json:"text"`
	Tone       string  `json:"tone"` // "formal", "friendly", "dating"
	Reasoning  string  `json:"reasoning"`
	Confidence float64 `json:"confidence"` // 0-1
}

// Message - Message in conversation (user or assistant)
type Message struct {
	ID             string                 `json:"id"`
	ConversationID string                 `json:"conversationId"`
	Role           string                 `json:"role"` // "user", "assistant"
	Content        string                 `json:"content"`
	Type           string                 `json:"type"` // "message", "question", "suggestion"
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	Timestamp      int64                  `json:"timestamp"`
}

// Reflection - Extracted insights from conversation
type Reflection struct {
	ID                       string                 `json:"id"`
	ConversationID           string                 `json:"conversationId"`
	ContactID                string                 `json:"contactId,omitempty"`
	Characteristics          []string               `json:"characteristics"`
	Interests                []string               `json:"interests"`
	CommunicationPreferences string                 `json:"communicationPreferences"`
	Intentions               []string               `json:"intentions"`
	UserQuotes               []string               `json:"userQuotes,omitempty"`
	Status                   string                 `json:"status"` // "pending_approval", "approved", "rejected"
	UserEdits                map[string]interface{} `json:"userEdits,omitempty"`
	CreatedAt                int64                  `json:"createdAt"`
	ApprovedAt               int64                  `json:"approvedAt,omitempty"`
}

// RiskWarning - Risk pattern detection result
type RiskWarning struct {
	RiskLevel            string                   `json:"riskLevel"` // "immediate", "high", "medium", "low", "clear"
	Pattern              string                   `json:"pattern,omitempty"`
	Severity             int                      `json:"severity"` // 0-10
	EducationalQuestions []string                 `json:"educationalQuestions"`
	Principles           []CommunicationPrinciple `json:"principles"`
	Alternatives         []string                 `json:"alternatives"`
	Recommendation       string                   `json:"recommendation"` // "proceed", "educate_first", "escalate"
	Message              string                   `json:"message"`
}

// SafetyAlert - Crisis/illegal content detected
type SafetyAlert struct {
	AlertType       string           `json:"alert_type"` // "crisis", "illegal", "none"
	Severity        string           `json:"severity"`   // "immediate", "high", "warning"
	Title           string           `json:"title"`
	Message         string           `json:"message"`
	Indicators      []string         `json:"indicators"`
	Resources       []CrisisResource `json:"resources"`
	Recommendations []string         `json:"recommendations"`
}

// CrisisResource - Resource for crisis situations
type CrisisResource struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Number      string `json:"number"`
	URL         string `json:"url"`
	Region      string `json:"region"`
}

// ConstitutionAnalysis - Ethical principles analysis
type ConstitutionAnalysis struct {
	AnalyzedAction    string                  `json:"analyzed_action"`
	Violations        []ConstitutionViolation `json:"violations"`
	AlignedPrinciples []string                `json:"aligned_principles"`
	OverallRiskLevel  string                  `json:"overall_risk_level"`
	CriticalConcerns  []string                `json:"critical_concerns"`
	Recommendations   []string                `json:"recommendations"`
	IsConstitutional  bool                    `json:"is_constitutional"`
}

// ConstitutionViolation - Violation of ethical principle
type ConstitutionViolation struct {
	PrincipleID string `json:"principle_id"`
	Principle   string `json:"principle"`
	Severity    string `json:"severity"` // "critical", "high", "medium"
	Description string `json:"description"`
	Reasoning   string `json:"reasoning"`
}

// ConversationFeedback - User feedback after suggestions
type ConversationFeedback struct {
	ConversationID      string                 `json:"conversationId" binding:"required"`
	UserID              string                 `json:"userId" binding:"required"`
	SuggestionChosen    int                    `json:"suggestionChosen"`
	SuggestionText      string                 `json:"suggestionText"`
	UserModified        bool                   `json:"userModified"`
	ModificationRequest string                 `json:"modificationRequest,omitempty"`
	ReflectionApproved  bool                   `json:"reflectionApproved"`
	ReflectionEdits     map[string]interface{} `json:"reflectionEdits,omitempty"`
	Timestamp           int64                  `json:"timestamp"`
}

// Contact - User's knowledge of a contact (user's observations only)
type Contact struct {
	ID                       string       `json:"id"`
	UserID                   string       `json:"userId"`
	Name                     string       `json:"name"`
	Relationship             string       `json:"relationship"` // "close_friend", "family", "work", "romantic", "new"
	Characteristics          []string     `json:"characteristics"`
	Interests                []string     `json:"interests"`
	CommunicationPreferences string       `json:"communicationPreferences"`
	Notes                    string       `json:"notes"`
	Reflections              []Reflection `json:"reflections,omitempty"`
	CreatedAt                int64        `json:"createdAt"`
	UpdatedAt                int64        `json:"updatedAt"`
}

// Conversation - Metadata about a conversation
type Conversation struct {
	ID              string `json:"id"`
	UserID          string `json:"userId"`
	ContactID       string `json:"contactId"`
	ContactName     string `json:"contactName"`
	MessageCount    int    `json:"messageCount"`
	LastMessage     string `json:"lastMessage"`
	LastMessageTime int64  `json:"lastMessageTime"`
	CreatedAt       int64  `json:"createdAt"`
}
