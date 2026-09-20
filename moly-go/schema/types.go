package schema

// AboutMeProfile represents a user's communication profile.
type AboutMeProfile struct {
	CommunicationStyle string   `json:"communicationStyle" validate:"max=100"`
	CoreValues         []string `json:"coreValues" validate:"max=20,dive,max=200"`
	TonePreference     string   `json:"tonePreference" validate:"max=100"`
	Preferences        string   `json:"preferences" validate:"max=1000"`
	Goals              []string `json:"goals" validate:"max=10,dive,max=500"`
	Patterns           []string `json:"patterns" validate:"max=10,dive,max=500"`
}

// ConversationMember represents a member in a conversation.
type ConversationMember struct {
	Name         string `json:"name" validate:"required"`
	Relationship string `json:"relationship"`
	Platform     string `json:"platform"`
	Notes        string `json:"notes" validate:"max=1000"`
}

// ConversationSettings represents conversation-specific settings.
type ConversationSettings struct {
	Mode         string `json:"mode"` // "socratic" or "direct"
	Context      string `json:"context"` // "formal", "friendly", "dating"
	LLMProvider  string `json:"llmProvider"`
}

// Conversation represents a single conversation.
type Conversation struct {
	ID          string                   `json:"id"`
	Name        string                   `json:"name" validate:"required,max=255"`
	Type        string                   `json:"type"`
	Purpose     string                   `json:"purpose" validate:"max=1000"`
	Description string                   `json:"description" validate:"max=5000"`
	Members     []ConversationMember     `json:"members" validate:"max=100,dive"`
	Settings    ConversationSettings     `json:"settings"`
	Notes       string                   `json:"notes" validate:"max=1000"`
	CreatedAt   int64                    `json:"createdAt"`
	UpdatedAt   int64                    `json:"updatedAt"`
}

// Contact represents a contact in the user's network.
type Contact struct {
	ID           string `json:"id"`
	Name         string `json:"name" validate:"required"`
	Relationship string `json:"relationship"`
	Notes        string `json:"notes" validate:"max=1000"`
	CreatedAt    int64  `json:"createdAt"`
}

// Phase5Request represents a message processing request for Phase5.
type Phase5Request struct {
	Message            string                 `json:"message" validate:"required,max=5000"`
	ConversationID     string                 `json:"conversationId" validate:"max=100"`
	AboutMe            map[string]interface{} `json:"aboutMe"`
	SelectedContactIds []string               `json:"selectedContactIds"`
	BrowserSessionId   string                 `json:"browserSessionId"` // Browser session ID for detecting new sessions
}

// ExtractedFact represents a fact extracted from a user message.
type ExtractedFact struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Value   string `json:"value"`
	Subject string `json:"subject"`
}

// SubjectShift represents a detected subject shift in conversation.
type SubjectShift struct {
	ID        string `json:"id"`
	FromID    string `json:"from_id"`
	ToID      string `json:"to_id"`
	Explicit  bool   `json:"explicit"`
	Timestamp int64  `json:"timestamp"`
}

// ClarificationQuestion represents a question needing clarification.
type ClarificationQuestion struct {
	ID               string   `json:"id"`
	Type             string   `json:"type"`               // "subject", "contact", "conflict", "socratic_exploration"
	Question         string   `json:"question"`
	Context          string   `json:"context"`            // Why we're asking
	Options          []string `json:"options"`            // Multiple choice (if applicable)
	Priority         int      `json:"priority"`           // 1=critical, 2=important, 3=nice-to-have
	LinkedFacts      []string `json:"linkedFacts"`        // Fact IDs waiting for clarification
	Status           string   `json:"status"`             // "pending", "answered", "skipped"
	CreatedAt        int64    `json:"createdAt"`
	// Socratic metadata
	SocraticApproach string   `json:"socraticApproach,omitempty"`     // e.g., "identifying_stakeholders"
	ExpectedInsights []string `json:"expectedInsights,omitempty"`     // Why this question helps
	TargetsPrinciple string   `json:"targetsPrinciple,omitempty"`     // Which principle it targets
	DepthLevel       int      `json:"depthLevel,omitempty"`           // 1=surface, 2=medium, 3=deep
}

// TemporaryFact represents a fact pending clarification.
type TemporaryFact struct {
	ID                string   `json:"id"`
	Type              string   `json:"type"`
	Value             string   `json:"value"`
	Subject           string   `json:"subject"`
	PendingQuestions  []string `json:"pendingQuestions"`
	Timestamp         int64    `json:"timestamp"`
	NeedsUserContext  bool     `json:"needsUserContext"`
	NeedsSubjectMatch bool     `json:"needsSubjectMatch"`
}

// ConflictDetection represents a detected conflict in stored data.
type ConflictDetection struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Field     string `json:"field"`
	OldValue  string `json:"oldValue"`
	NewValue  string `json:"newValue"`
	Timestamp int64  `json:"timestamp"`
}

// ContextAttribute represents an attributed fact stored in context.
type ContextAttribute struct {
	ID             string `json:"id"`
	Type           string `json:"type"`
	Value          string `json:"value"`
	AttributedTo   string `json:"attributedTo"`
	ConversationID string `json:"conversationId"`
	Confidence     float64 `json:"confidence"`
	Source         string `json:"source"`
	Timestamp      int64  `json:"timestamp"`
}

// Phase5Response represents the full response from message processing.
type Phase5Response struct {
	Success       bool                    `json:"success"`
	Phase1        Phase1Result            `json:"phase1"`
	Phase2        Phase2Result            `json:"phase2"`
	Phase3        Phase3Result            `json:"phase3"`
	Phase4        Phase4Result            `json:"phase4"`
	ActionRequired ActionRequiredResponse  `json:"action_required"`
}

// Phase1Result contains extraction results.
type Phase1Result struct {
	Facts  []ExtractedFact `json:"facts"`
	Shifts []SubjectShift  `json:"shifts"`
}

// Phase2Result contains clarification results.
type Phase2Result struct {
	Clarifications []ClarificationQuestion `json:"clarifications"`
	Resolved       int                     `json:"resolved"`
}

// Phase3Result contains contact handling results.
type Phase3Result struct {
	UnknownContacts []string `json:"unknown_contacts"`
	CreatedContacts []Contact `json:"created_contacts"`
}

// Phase4Result contains attribution results.
type Phase4Result struct {
	SavedAttributes []ContextAttribute `json:"saved_attributes"`
	Conflicts       []ConflictDetection `json:"conflicts"`
}

// ActionRequiredResponse describes what action the user needs to take.
type ActionRequiredResponse struct {
	NeedsClarification bool                   `json:"needsClarification"`
	ClarificationQs   []ClarificationQuestion `json:"clarificationQs"`
	TemporaryFacts    []TemporaryFact         `json:"temporaryFacts"`
	HasConflicts      bool                   `json:"hasConflicts"`
	Conflicts         []ConflictDetection    `json:"conflicts"`
}
