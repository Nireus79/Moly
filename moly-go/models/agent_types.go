package models

// Agent Types - Core type definitions for v2 agents
// See: /MOLY_V2_ARCHITECTURE/05_API_SPECIFICATION.md
// See: /MOLY_V2_ARCHITECTURE/03_AGENT_PROMPTS.md

// Agent Interfaces
// ConversationAgent - Orchestrates the 5-phase conversation flow
type ConversationAgent interface {
	Run(ctx Context) (*ConversationResponse, error)
}

// LearningAgent - Builds and maintains user behavioral profile
type LearningAgent interface {
	GetUserProfile(userID string) (*UserBehavioralProfile, error)
	RecordInteraction(data InteractionData) error
	RecordSuggestionChoice(data SuggestionChoiceData) error
	BuildBehavioralProfile(userID string) (*UserBehavioralProfile, error)
	DetectPatterns(userID string) (*UserPatterns, error)
}

// ContextManagerAgent - Intelligent knowledge base management
type ContextManagerAgent interface {
	GetAboutMe(userID string) (*AboutMe, error)
	SetAboutMe(userID string, aboutMe *AboutMe) error
	GetContact(userID, contactID string) (*Contact, error)
	GetContacts(userID string) ([]Contact, error)
	CreateContact(userID string, contact *Contact) (*Contact, error)
	UpdateContact(userID, contactID string, updates Contact) error
	GetRelevantContext(conversationID, userID string) (*Context, error)
	SaveReflection(conversationID string, reflection *Reflection) error
	ApproveReflection(conversationID string, reflection *Reflection) error
	AppendMessage(conversationID string, message *Message) error
}

// RiskMonitoringAgent - Pattern detection and educational safety
type RiskMonitoringAgent interface {
	AssessRisk(userID string, message string) (*RiskAssessment, error)
	DetectPatterns(userID string) (*UserRiskProfile, error)
	GenerateEducationalResponse(risk RiskAssessment) ([]string, error)
	TrackPattern(userID string, pattern *RiskPattern) error
	GetUserRiskProfile(userID string) (*UserRiskProfile, error)
}

// Context Types
// Context - Relevant context for a conversation
type Context struct {
	AboutMe             *AboutMe               `json:"aboutMe"`
	ContactProfile      *Contact               `json:"contactProfile"`
	ConversationHistory []Message              `json:"conversationHistory"`
	UserBehaviorProfile *UserBehavioralProfile `json:"userBehaviorProfile"`
	RelevantReflections []Reflection           `json:"relevantReflections"`
	ContextQuality      string                 `json:"contextQuality"` // "complete", "partial", "minimal"
	Gaps                []string               `json:"gaps"`           // Missing context fields
}

// AboutMe - User's own communication profile
type AboutMe struct {
	UserID             string   `json:"userId"`
	CommunicationStyle string   `json:"communicationStyle"` // e.g., "casual, direct, authentic"
	Values             []string `json:"values"`             // e.g., ["authenticity", "loyalty"]
	PreferredTone      string   `json:"preferredTone"`      // "formal", "friendly", "dating"
	Notes              string   `json:"notes"`
	CreatedAt          int64    `json:"createdAt"`
	UpdatedAt          int64    `json:"updatedAt"`
	Version            int      `json:"version"`
}

// UserBehavioralProfile - What Moly learns about the user (NOT contacts)
type UserBehavioralProfile struct {
	UserID               string                 `json:"userId"`
	CommunicationProfile map[string]interface{} `json:"communicationProfile"`
	CommunicationGoals   map[string]int         `json:"communicationGoals"`
	SuggestionChoices    map[string]interface{} `json:"suggestionChoices"`
	SuccessMetrics       map[string]interface{} `json:"successMetrics"`
	EmergingPersonality  []string               `json:"emergingPersonality"`
	GrowthTrajectory     map[string]interface{} `json:"growthTrajectory"`
	CreatedAt            int64                  `json:"createdAt"`
	UpdatedAt            int64                  `json:"updatedAt"`
	Version              int                    `json:"version"`
	Confidence           float64                `json:"confidence"` // 0-1
}

// UserPatterns - Detected patterns from behavioral analysis
type UserPatterns struct {
	UserID              string             `json:"userId"`
	CommunicationStyle  string             `json:"communicationStyle"`
	PreferredTone       map[string]float64 `json:"preferredTone"` // "formal", "friendly", "dating" percentages
	SuggestionPickRate  float64            `json:"suggestionPickRate"`
	ModificationRate    float64            `json:"modificationRate"`
	CommunicationGoals  map[string]int     `json:"communicationGoals"`
	EmergingPersonality []string           `json:"emergingPersonality"`
	ConfidenceLevel     string             `json:"confidenceLevel"` // "high", "medium", "low"
}

// InteractionData - Recorded when user interacts with Moly
type InteractionData struct {
	UserID               string `json:"userId"`
	ConversationID       string `json:"conversationId"`
	UserMessage          string `json:"userMessage"`
	SuggestionsGenerated int    `json:"suggestionsGenerated"`
	CreatedAt            int64  `json:"createdAt"`
}

// SuggestionChoiceData - Recorded when user picks a suggestion
type SuggestionChoiceData struct {
	UserID          string `json:"userId"`
	ConversationID  string `json:"conversationId"`
	SuggestionIndex int    `json:"suggestionIndex"`
	ModifiedText    string `json:"modifiedText,omitempty"`
	Modification    string `json:"modification,omitempty"` // e.g., "make more casual"
	UserFeedback    string `json:"userFeedback,omitempty"` // "positive", "neutral", "negative"
	CreatedAt       int64  `json:"createdAt"`
}

// RiskAssessment - Result of risk detection
type RiskAssessment struct {
	RiskLevel            string                   `json:"riskLevel"` // "immediate", "high", "medium", "low", "clear"
	Pattern              string                   `json:"pattern,omitempty"`
	Severity             int                      `json:"severity"` // 0-10
	EducationalQuestions []string                 `json:"educationalQuestions"`
	Principles           []CommunicationPrinciple `json:"principles"`
	Alternatives         []string                 `json:"alternatives"`
	Recommendation       string                   `json:"recommendation"` // "proceed", "educate_first", "escalate"
	Message              string                   `json:"message"`
}

// UserRiskProfile - Tracked risk patterns for a user
type UserRiskProfile struct {
	UserID          string         `json:"userId"`
	RiskPatterns    []RiskPattern  `json:"riskPatterns"`
	HighestRisk     string         `json:"highestRisk"`     // Most concerning pattern
	InterventionLog []Intervention `json:"interventionLog"` // What worked
	UpdatedAt       int64          `json:"updatedAt"`
}

// RiskPattern - Detected concerning pattern in user behavior
type RiskPattern struct {
	PatternType         string                 `json:"patternType"` // "manipulation", "boundary", "scam", "harm", "insincerity"
	Severity            int                    `json:"severity"`    // 0-10
	FirstOccurrence     int64                  `json:"firstOccurrence"`
	LastOccurrence      int64                  `json:"lastOccurrence"`
	OccurrenceCount     int                    `json:"occurrenceCount"`
	Interventions       []Intervention         `json:"interventions"`
	Trend               string                 `json:"trend"` // "increasing", "stable", "decreasing"
	RootCauseHypothesis string                 `json:"rootCauseHypothesis"`
	UserResponsePattern map[string]interface{} `json:"userResponsePattern"`
}

// Intervention - Educational response to risk pattern
type Intervention struct {
	Date    int64  `json:"date"`
	Type    string `json:"type"`    // "socratic_questions", "principle_education", "alternative_suggestion"
	Outcome string `json:"outcome"` // "adjusted", "proceeded", "unknown"
}

// CommunicationPrinciple - Ethical principle for communication
type CommunicationPrinciple struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Severity    string   `json:"severity"` // "critical", "high", "medium"
	Description string   `json:"description"`
	Questions   []string `json:"questions"`
}
