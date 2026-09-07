package models

// ContextRequest - Request to retrieve conversation context
type ContextRequest struct {
	ConversationID string `json:"conversationId" binding:"required"`
	UserID         string `json:"userId" binding:"required"`
	IncludeHistory bool   `json:"includeHistory"`
	IncludeProfile bool   `json:"includeProfile"`
	IncludeContact bool   `json:"includeContact"`
}

// ContextResponse - Comprehensive context for conversation
type ContextResponse struct {
	ConversationID       string                 `json:"conversationId"`
	AboutMe              *AboutMe               `json:"aboutMe,omitempty"`
	ContactProfile       *ContactProfile        `json:"contactProfile,omitempty"`
	ConversationHistory  []Message              `json:"conversationHistory,omitempty"`
	UserBehaviorProfile  *UserBehavioralProfile `json:"userBehaviorProfile,omitempty"`
	RelevantReflections  []Reflection           `json:"relevantReflections,omitempty"`
	ContextQuality       ContextQualityMetrics  `json:"contextQuality"`
	MissingContextGaps   []string               `json:"missingContextGaps"`
	Error                string                 `json:"error,omitempty"`
}

// ContactProfile - Enhanced contact profile with metadata
type ContactProfile struct {
	ID                       string                 `json:"id"`
	UserID                   string                 `json:"userId"`
	Name                     string                 `json:"name"`
	Relationship             string                 `json:"relationship"`
	Characteristics          []string               `json:"characteristics"`
	Interests                []string               `json:"interests"`
	CommunicationPreferences string                 `json:"communicationPreferences"`
	Notes                    string                 `json:"notes"`
	ReflectionCount          int                    `json:"reflectionCount"`
	LastInteractionTime      int64                  `json:"lastInteractionTime"`
	ConversationCount        int                    `json:"conversationCount"`
	CreatedAt                int64                  `json:"createdAt"`
	UpdatedAt                int64                  `json:"updatedAt"`
}

// ContextQualityMetrics - Assessment of context completeness
type ContextQualityMetrics struct {
	OverallScore      float64            `json:"overallScore"`      // 0-1
	HasAboutMe        bool               `json:"hasAboutMe"`
	HasContactProfile bool               `json:"hasContactProfile"`
	HasHistory        bool               `json:"hasHistory"`
	HasBehaviorProfile bool              `json:"hasBehaviorProfile"`
	HasReflections    bool               `json:"hasReflections"`
	HistoryLength     int                `json:"historyLength"`
	ReflectionCount   int                `json:"reflectionCount"`
	CompletenessLevel string             `json:"completenessLevel"` // "complete", "partial", "minimal"
	Recommendations   []string           `json:"recommendations"`
}

// ContactsListResponse - Response with all user contacts
type ContactsListResponse struct {
	UserID   string           `json:"userId"`
	Contacts []ContactProfile `json:"contacts"`
	Total    int              `json:"total"`
	Error    string           `json:"error,omitempty"`
}

// AboutMeRequest - Request to save/update AboutMe profile
type AboutMeRequest struct {
	UserID           string   `json:"userId" binding:"required"`
	CommunicationStyle string  `json:"communicationStyle"`
	Values           []string `json:"values"`
	PreferredTone    string   `json:"preferredTone"`
	Notes            string   `json:"notes"`
}

// AboutMeResponse - Response with updated AboutMe
type AboutMeResponse struct {
	AboutMe *AboutMe `json:"aboutMe"`
	Error   string   `json:"error,omitempty"`
}

// CreateContactRequest - Request to create new contact
type CreateContactRequest struct {
	UserID                   string   `json:"userId" binding:"required"`
	Name                     string   `json:"name" binding:"required"`
	Relationship             string   `json:"relationship"`
	Characteristics          []string `json:"characteristics"`
	Interests                []string `json:"interests"`
	CommunicationPreferences string   `json:"communicationPreferences"`
	Notes                    string   `json:"notes"`
}

// CreateContactResponse - Response with created contact
type CreateContactResponse struct {
	Contact *ContactProfile `json:"contact"`
	Error   string          `json:"error,omitempty"`
}

// UpdateContactRequest - Request to update contact
type UpdateContactRequest struct {
	ContactID                string   `json:"contactId" binding:"required"`
	UserID                   string   `json:"userId" binding:"required"`
	Name                     string   `json:"name,omitempty"`
	Relationship             string   `json:"relationship,omitempty"`
	Characteristics          []string `json:"characteristics,omitempty"`
	Interests                []string `json:"interests,omitempty"`
	CommunicationPreferences string   `json:"communicationPreferences,omitempty"`
	Notes                    string   `json:"notes,omitempty"`
}

// UpdateContactResponse - Response after update
type UpdateContactResponse struct {
	Contact *ContactProfile `json:"contact"`
	Error   string          `json:"error,omitempty"`
}

// ReflectionRequest - Request to save extracted reflection
type ReflectionRequest struct {
	ConversationID        string                 `json:"conversationId" binding:"required"`
	UserID                string                 `json:"userId" binding:"required"`
	ContactID             string                 `json:"contactId,omitempty"`
	Characteristics       []string               `json:"characteristics"`
	Interests             []string               `json:"interests"`
	CommunicationPreferences string               `json:"communicationPreferences"`
	Intentions            []string               `json:"intentions"`
	UserQuotes            []string               `json:"userQuotes,omitempty"`
	Metadata              map[string]interface{} `json:"metadata,omitempty"`
}

// ReflectionResponse - Response with pending reflection
type ReflectionResponse struct {
	Reflection *Reflection `json:"reflection"`
	Error      string      `json:"error,omitempty"`
}

// ApproveReflectionRequest - Request to approve reflection
type ApproveReflectionRequest struct {
	ReflectionID  string                 `json:"reflectionId" binding:"required"`
	UserID        string                 `json:"userId" binding:"required"`
	UserEdits     map[string]interface{} `json:"userEdits,omitempty"`
	ApprovalNotes string                 `json:"approvalNotes,omitempty"`
}

// ApproveReflectionResponse - Response after approval
type ApproveReflectionResponse struct {
	Reflection *Reflection `json:"reflection"`
	Error      string      `json:"error,omitempty"`
}

// ContextInsight - Single extracted insight from conversation
type ContextInsight struct {
	ID          string `json:"id"`
	Type        string `json:"type"` // "characteristic", "interest", "goal", "communication_style"
	Text        string `json:"text"`
	Confidence  float64 `json:"confidence"` // 0-1
	Source      string `json:"source"`      // "message_text", "tone_analysis", "implicit"
	UserQuote   string `json:"userQuote,omitempty"`
}

// ContextExtractionResult - Result of context extraction
type ContextExtractionResult struct {
	ConversationID string            `json:"conversationId"`
	Insights       []ContextInsight  `json:"insights"`
	Quality        float64           `json:"quality"` // 0-1
	Completeness   float64           `json:"completeness"` // 0-1
	RecommendedApproval bool         `json:"recommendedApproval"`
	ProcessingMs   int               `json:"processingMs"`
}
