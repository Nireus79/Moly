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

// ConversationResponse - Response with metadata
type ConversationResponse struct {
	Phase            string                        `json:"phase"` // "responding", "context_gathering", "safety_alert", "error"
	Response         string                        `json:"response"` // Moly's conversational response to user
	Reflection       *Reflection                   `json:"reflection,omitempty"` // Insights about the user
	SafetyAlert      *SafetyAlert                  `json:"safetyAlert,omitempty"`
	ExtractedContact *Contact                      `json:"extractedContact,omitempty"` // Contact detected from message
	ProcessingTimeMs int                           `json:"processingTimeMs"`
	Metadata         map[string]interface{}        `json:"metadata,omitempty"`
	Error            string                        `json:"error,omitempty"`
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
	ID                       int64        `json:"id"`
	UserID                   string       `json:"userId"`
	Name                     string       `json:"name"`
	Relationship             string       `json:"relationship" validate:"oneof=romantic professional family friend other"` // Valid: romantic, professional, family, friend, other
	Age                      string       `json:"age,omitempty"`
	Characteristics          []string     `json:"characteristics"`
	Interests                []string     `json:"interests"`
	CommunicationPreferences string       `json:"communicationPreferences"`
	Notes                    string       `json:"notes"`
	FirstMentionedAt         int64        `json:"firstMentionedAt,omitempty"`
	CreatedVia               string       `json:"createdVia"` // "conversation", "manual", "import"
	Status                   string       `json:"status"`     // "active", "archived"
	Version                  int64        `json:"version"`    // For optimistic locking
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

// PersonInvolved - Person in the structured context
type PersonInvolved struct {
	Name         string `json:"name"`
	Relationship string `json:"relationship"` // "boss", "partner", "friend", "family", "colleague"
	Role         string `json:"role,omitempty"`
}

// StructuredContext - Tracked understanding of the situation
type StructuredContext struct {
	ID              int64             `json:"id"`
	UserID          string            `json:"userId"`
	ConversationID  string            `json:"conversationId"`
	Situation       string            `json:"situation"`
	Topic           string            `json:"topic"`
	PeopleInvolved  []PersonInvolved  `json:"peopleInvolved"`
	Goals           []string          `json:"goals"`
	Values          []string          `json:"values"`
	Constraints     []string          `json:"constraints"`
	PastAttempts    []string          `json:"pastAttempts"`
	CurrentBlocker  string            `json:"currentBlocker"`
	EmotionalTone   string            `json:"emotionalTone"`
	RemainingGaps   []string          `json:"remainingGaps"`
	ExploredTopics  []string          `json:"exploredTopics"`
	CreatedAt       int64             `json:"createdAt"`
	UpdatedAt       int64             `json:"updatedAt"`
}

// ConversationSummary - Hybrid context: compact summary of all messages + metadata
type ConversationSummary struct {
	ID                  int64    `json:"id"`
	UserID              string   `json:"userId"`
	ConversationID      string   `json:"conversationId"`
	Arc                 string   `json:"arc"`                  // Narrative summary of conversation flow
	KeyTopics           []string `json:"keyTopics"`            // Tags: what was discussed
	UserPatterns        []string `json:"userPatterns"`         // Observed communication patterns
	ConfirmedChoices    []string `json:"confirmedChoices"`     // From Layer 3 clarifications
	OpenQuestions       []string `json:"openQuestions"`        // Unresolved questions
	MessageCount        int      `json:"messageCount"`         // Total messages in conversation
	MessagesSinceUpdate int      `json:"messagesSinceUpdate"`  // How many new messages since last update
	SummaryVersion      int      `json:"summaryVersion"`       // Track summary iterations
	Confidence          float64  `json:"confidence"`           // 0-1: accuracy/completeness
	LastUpdated         int64    `json:"lastUpdated"`          // Unix timestamp
	CreatedAt           int64    `json:"createdAt"`
	UpdatedAt           int64    `json:"updatedAt"`
}

// AnalysisContext - Context passed to evaluators (hybrid: summary + recent messages + data)
type AnalysisContext struct {
	ConversationSummary  *ConversationSummary        `json:"conversationSummary"`  // Compact summary of full history
	RecentMessages       []Message                   `json:"recentMessages"`       // Last 2-3 full messages
	ConfirmedPreferences map[string]interface{}      `json:"confirmedPreferences"` // From Layer 3
	UserProfile          *AboutMe                    `json:"userProfile"`          // Communication style
	RelevantContacts     []Contact                   `json:"relevantContacts"`     // Contacts mentioned
	CurrentMessage       string                      `json:"currentMessage"`       // Message being analyzed
	TotalMessages        int                         `json:"totalMessages"`        // Full conversation length
	ContextQuality       string                      `json:"contextQuality"`       // "complete", "partial", "minimal"
}
