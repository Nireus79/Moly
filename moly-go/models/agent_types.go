package models

// Agent Types - Core type definitions for agents
// See: /MOLY_V2_ARCHITECTURE/05_API_SPECIFICATION.md
// See: /MOLY_V2_ARCHITECTURE/03_AGENT_PROMPTS.md

// ExtractedContext represents all structured data extracted from a message using LLM
type ExtractedContext struct {
	Contact                *ExtractedContact   `json:"contact,omitempty"`
	Style                  *ExtractedStyle     `json:"style,omitempty"`
	Intention              string              `json:"intention,omitempty"`
	IntentionConfidence    float64             `json:"intentionConfidence"`      // 0-1 confidence in extracted intention
	IntentionBasis         string              `json:"intentionBasis,omitempty"` // how the model reached the intention: stated, implied or guessed
	IntentionEvidence      string              `json:"intentionEvidence,omitempty"` // the exact words of the message that state the goal; a goal without them is not locked
	IntentionPrinciples    []string            `json:"intentionPrinciples"`      // LLM-identified principles engaged (transparency, autonomy, empathy, fairness, growth, stakeholder)
	Goals                  []string            `json:"goals,omitempty"`
	UserValues             []string            `json:"userValues,omitempty"`             // User's expressed values (for response constraint generation)
	UserCharacteristics    []string            `json:"userCharacteristics,omitempty"`    // FIX #5: Tagged characteristics about the user (format: "USER|trait|confidence")
	ContactCharacteristics map[string][]string `json:"contactCharacteristics,omitempty"` // FIX #5: Tagged characteristics about contacts (format: "CONTACT_name|trait|confidence")
	SystemFeedback         *SystemFeedbackInfo `json:"systemFeedback,omitempty"`         // Feedback about Moly itself (when user addresses system)
}

// SystemFeedbackInfo represents feedback directed at Moly
type SystemFeedbackInfo struct {
	IsFeedback   bool     `json:"isFeedback"`   // Is this feedback about the system?
	FeedbackType string   `json:"feedbackType"` // "positive", "negative", "directive", "perception"
	Feedback     []string `json:"feedback"`     // e.g., ["too verbose", "helpful", "confusing"]
	Directives   []string `json:"directives"`   // e.g., ["be more concise", "ask questions"]
	Perceptions  []string `json:"perceptions"`  // e.g., ["good at analysis", "lacks empathy"]
	Style        string   `json:"style"`        // "direct", "socratic", "collaborative" if directive
	Confidence   float64  `json:"confidence"`   // 0-1 confidence in feedback extraction
	Evidence     string   `json:"evidence"`     // Quote from message
}

// ExtractedContact represents a detected contact from message
type ExtractedContact struct {
	Name         string   `json:"name"`
	Label        string   `json:"label,omitempty"`                                                         // the user's own words for the person (used until a name is given)
	NameKnown    bool     `json:"nameKnown"`                                                               // true only when the user gave a real name
	Relationship string   `json:"relationship" validate:"oneof=romantic professional family friend other"` // Valid: romantic, professional, family, friend, other
	Traits       []string `json:"traits,omitempty"`
	Confidence   float64  `json:"confidence"`      // 0-1
	Evidence     string   `json:"evidence"`        // Quote from message
	Basis        string   `json:"basis,omitempty"` // how the model reached this person: stated, implied or guessed
}

// ExtractedStyle represents communication style preferences
type ExtractedStyle struct {
	Style      string   `json:"style"` // casual, formal, playful, mix
	Tone       string   `json:"tone"`  // friendly, professional, humorous, etc
	Values     []string `json:"values,omitempty"`
	Confidence float64  `json:"confidence"` // 0-1
}

// Agent Interfaces
// ConversationAgent - Orchestrates the 5-phase conversation flow
type ConversationAgent interface {
	Run(ctx Context, analysisCtx *AnalysisContext) (*ConversationResponse, error)
	SetDatabase(db interface{}) // For Phase 2 inline conflict resolution
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

// Context Types
// Context - Relevant context for a conversation
type Context struct {
	ConversationID               string                 `json:"conversationId,omitempty"` // For recording questions and interactions
	AboutMe                      *AboutMe               `json:"aboutMe"`
	SystemContext                *SystemContext         `json:"systemContext,omitempty"` // User's feedback and directives about Moly
	ContactProfile               *Contact               `json:"contactProfile"`
	ConversationHistory          []Message              `json:"conversationHistory"`
	UserBehaviorProfile          *UserBehavioralProfile `json:"userBehaviorProfile"`
	RelevantReflections          []Reflection           `json:"relevantReflections"`
	ExtractedContext             *ExtractedContext      `json:"extractedContext,omitempty"`         // LLM-extracted contact, style, intention, goals
	ExtractedEntities            []ExtractedEntity      `json:"extractedEntities,omitempty"`        // Semantic entity classification (self_reference, contact, topic, goal)
	PastIntention                string                 `json:"pastIntention,omitempty"`            // User's goal from previous message(s)
	RecentSafetyIncidents        []SafetyIncident       `json:"recentSafetyIncidents,omitempty"`    // Recent safety alerts to prevent re-alerting
	PrecomputedSafetyVerdict     *SafetyAlert           `json:"precomputedSafetyVerdict,omitempty"` // Phase 1: Constitutional evaluator verdict (computed in main.go)
	BoundedAnalysisContext       *AnalysisContext       `json:"boundedAnalysisContext,omitempty"`   // Hybrid context: summary + recent messages + profile (700-800 tokens)
	ConversationPhase            string                 `json:"conversationPhase,omitempty"`        // "initial", "gathering", "processing", "complete"
	ContextQuality               string                 `json:"contextQuality"`                     // "complete", "partial", "minimal"
	ContextMaturity              float64                `json:"contextMaturity"`                    // 0.0-1.0, used for Layer 3 and Layer 8 prerequisites (deprecated, use Maturity)
	Maturity                     *ConversationMaturity  `json:"maturity,omitempty"`                 // Accomplishment-based maturity (phases + overall score)
	MissingContext               []string               `json:"missingContext"`                     // Context-loader fields that are empty (informational; never a question)
	Gaps                         []string               `json:"gaps"`                               // Layer 4 goal gaps only (questions come from here)
	MessageIntent                string                 `json:"messageIntent,omitempty"`            // LLM intent of this message (greeting, asking, sharing, ...), decided before the layers
	MessageIntentConfidence      float64                `json:"messageIntentConfidence"`            // Confidence of MessageIntent
	IsGreeting                   bool                   `json:"isGreeting"`
	NameAnswered                 bool                   `json:"nameAnswered"`                 // this message answered a name question
	ResultNow                    bool                   `json:"resultNow,omitempty"`          // the user chose to take the result now, with what is known
	DoubtfulFact                 string                 `json:"doubtfulFact,omitempty"`       // a fact is in doubt and was not saved: confirm it before anything else
	PendingNameLabel             string                 `json:"pendingNameLabel,omitempty"`   // a contact has no name yet: ask for it before anything else                         // True when MessageIntent is greeting: no goal, no gaps, greet back
	SessionID                    string                 `json:"sessionId,omitempty"`          // Browser session identifier
	IsFirstMessageOfSession      bool                   `json:"isFirstMessageOfSession"`      // true only for first message in new browser session
	IsFirstMessageInConversation bool                   `json:"isFirstMessageInConversation"` // true only for first message in this conversation (calculated before prepending)

	// Layer 3: Clarification Capture & Conflict Detection
	PendingClarifications      []interface{}          `json:"pendingClarifications,omitempty"`      // Unanswered clarification questions
	JustAnsweredClarifications []interface{}          `json:"justAnsweredClarifications,omitempty"` // Clarifications answered in this message
	ConfirmedUserPreferences   map[string]interface{} `json:"confirmedUserPreferences,omitempty"`   // User's confirmed preferences from past clarifications
	UnresolvedConflicts        []interface{}          `json:"unresolvedConflicts,omitempty"`        // Conflicts detected (need user resolution)

	// FIX #3: Track what was clarified by Tier 1 to avoid Tier 2 overlap
	Metadata map[string]interface{} `json:"metadata,omitempty"` // Gate information from clarification analysis
}

// SafetyIncident - Records of safety alerts triggered
type SafetyIncident struct {
	ID               int64  `json:"id"`
	UserID           string `json:"userId"`
	Severity         string `json:"severity"` // "low", "medium", "high", "critical"
	DetectedAt       int64  `json:"detectedAt"`
	Content          string `json:"content"`          // The message that triggered the alert
	DetectedBy       string `json:"detectedBy"`       // "heuristic", "llm", "manual"
	ResponseProvided string `json:"responseProvided"` // Alert title or response provided
}

// AboutMe - User's own communication profile
type AboutMe struct {
	UserID             string   `json:"userId"`
	CommunicationStyle string   `json:"communicationStyle"`         // e.g., "casual, direct, authentic"
	Values             []string `json:"values"`                     // e.g., ["authenticity", "loyalty"]
	PreferredTone      string   `json:"preferredTone"`              // "formal", "friendly", "dating"
	Goals              []string `json:"goals,omitempty"`            // e.g., ["improve communication", "build confidence"]
	Characteristics    []string `json:"characteristics,omitempty"`  // e.g., ["adventurous", "likes BDSM"] - extracted from user self-references
	UserInstructions   []string `json:"userInstructions,omitempty"` // e.g., ["I learn best through examples", "Validate my feelings first"] - how user wants to be understood
	Notes              string   `json:"notes"`
	CreatedAt          int64    `json:"createdAt"`
	UpdatedAt          int64    `json:"updatedAt"`
	Version            int      `json:"version"`
}

// SystemContext - User's feedback and preferences about Moly (system self-awareness)
type SystemContext struct {
	UserID                    string   `json:"userId"`
	UserFeedback              []string `json:"userFeedback,omitempty"`              // e.g., ["too verbose", "helpful", "confusing"]
	UserDirectives            []string `json:"userDirectives,omitempty"`            // e.g., ["be concise", "ask more", "skip family topics"]
	SystemPerceptions         []string `json:"systemPerceptions,omitempty"`         // e.g., ["can do legal advice?", "lacks empathy", "good at analysis"]
	PreferredInteractionStyle string   `json:"preferredInteractionStyle,omitempty"` // "direct", "socratic", "collaborative"
	SystemInstructions        []string `json:"systemInstructions,omitempty"`        // e.g., ["Be Socratic", "Ask questions", "Challenge assumptions"] - how system should behave
	HelpfulnessRating         float64  `json:"helpfulnessRating,omitempty"`         // 0-1, from explicit or implicit feedback
	ClarityRating             float64  `json:"clarityRating,omitempty"`             // 0-1, how clear user finds responses
	CreatedAt                 int64    `json:"createdAt"`
	UpdatedAt                 int64    `json:"updatedAt"`
	Version                   int      `json:"version"`
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

// ExtractedEntity - Entity extracted with semantic classification
type ExtractedEntity struct {
	Value                  string   `json:"value"`                            // "Moly", "Lace", "business", etc.
	Type                   string   `json:"type"`                             // "self_reference", "contact", "topic", "goal", "ambiguous"
	Evidence               string   `json:"evidence"`                         // Exact substring from message
	Confidence             float64  `json:"confidence"`                       // 0.0-1.0
	IsAmbiguous            bool     `json:"isAmbiguous"`                      // true if could be multiple types
	AmbiguousPossibilities []string `json:"ambiguousPossibilities,omitempty"` // ["self_reference", "contact"]
	Reasoning              string   `json:"reasoning"`                        // Why this classification
	Subject                string   `json:"subject,omitempty"`                // WHO has this property: "user", contact name, or pronoun (she/he/they)
	SourceType             string   `json:"sourceType,omitempty"`             // "extraction" or "clarification"
	Antonyms               []string `json:"antonyms,omitempty"`               // NEW: Opposite characteristics (for conflict detection)
}

// IntentAnalysis - User intent with entity extraction
type IntentAnalysis struct {
	Intent                string            `json:"intent"`                          // ask, share, help_seek, greet, vent, react, confirm
	Confidence            float64           `json:"confidence"`                      // 0.0-1.0
	Entities              []ExtractedEntity `json:"entities,omitempty"`              // Extracted entities with semantic classification
	NeedsClarification    bool              `json:"needsClarification"`              // true if ambiguous entity detected
	ClarificationQuestion string            `json:"clarificationQuestion,omitempty"` // Question to ask user if ambiguous
	AmbiguousEntity       string            `json:"ambiguousEntity,omitempty"`       // Which entity is ambiguous
}
