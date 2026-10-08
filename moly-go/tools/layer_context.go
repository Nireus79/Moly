package tools

import (
	"moly/database"
	"moly/models"
	"time"
)

// LayerContext is the unified context that flows through all 11 layers
// It wraps AnalysisContext and adds layer-specific processing results
type LayerContext struct {
	// Base analysis context (from ExtractionPhase)
	Analysis *models.AnalysisContext

	// Layer results - populated as context flows through layers
	Layer1  *Layer1Result
	Layer2  *Layer2Result
	Layer3  *Layer3Result
	Layer4  *Layer4Result
	Layer5  *Layer5Result
	Layer6  *Layer6Result
	Layer7  *Layer7Result
	Layer8  *Layer8Result
	Layer9  *Layer9Result
	Layer10 *Layer10Result
	Layer11 *Layer11Result

	// Control flow
	ShouldStop                 bool   // Set to true to skip remaining layers
	StopReason                 string // Why we stopped (e.g., "obvious_harm")
	StopAfterLayer4IfGapsFound bool   // ARCHITECTURAL FIX #1: Signal from Layer 3 to Layer 4: if immature, stop if gaps

	// Proportional gating (NEW: replaces hardcoded 0.5 threshold)
	MaturityPhase        string  // "discovery", "analysis", "design", "implementation"
	MaturitySeverityGate float64 // 0.3-1.0 based on phase (severity threshold for enforcement)

	// Metadata
	StartTime      int64 // Unix timestamp
	UserID         string
	MessageID      string
	ConversationID string

	// PHASE 5: Accumulated context across messages
	// Previous messages' extracted data - used by Layer 5+ for contradiction detection, etc.
	AccumulatedExtractedEntities []models.ExtractedEntity // All entities from previous messages
	PreviousGoal                 string                   // Goal from previous message(s)
	PreviousValues               []string                 // Values from previous message(s)

	// FIX #11: Phase 3 - Message summary cache for skipping redundant processing
	// Maps message IDs to their cached summaries from previous messages
	MessageSummaryCache map[string]interface{} // FIX #11: {messageID: MessageSummary}

	// FIX #2 (Session 34): Unified maturity context for all layers
	// Loaded once in main.go and passed to all layers to avoid duplicate saves
	MaturityContext *models.ConversationMaturity // Shared across all layers, NOT reloaded

	// FIX #4: Goal tracking - distinguish primary goal from current intent
	// Primary goal: Message 1's intention, never changes
	// Current intent: Fresh intention extracted this message
	PrimaryGoal          string   // Locked on Message 1, used for all comparisons
	CurrentMessageIntent string   // Fresh extraction each message
	GoalProgression      []string // Track evolution: M1 intent, M2 intent, M3 intent...
	IsMessageOne         bool     // True if this is Message 1 in conversation

	// FIX #72: Goal coherence analysis - how current goal relates to primary goal
	// Used by Layer 4 to determine which gaps are relevant
	GoalCoherence *models.GoalCoherence // "same", "related_subgoal", "different", etc.

	// FIX #22 & #23: Historical insights and reflections
	// Provide layers access to previous insights about contacts and conversation patterns
	RecentInsights      []models.Reflection // Previous insights/reflections about contacts
	RelevantReflections []models.Reflection // Reflections relevant to current conversation

	// FIX #5: User's goal for this message (extracted by Layer 1)
	// Used to guide response strategy and gap detection
	UserGoal string // "write_message", "decide_disclosure", etc.

	// FIX #6: Conversation topic/focus (extracted by Layer 1)
	// Used to validate response respects conversation focus
	ConversationTopic string // "girl", "Christine_sub", etc.

	// FIX #12: Pending clarifications from database
	// Loaded by orchestrator, used by Layer 4 to filter out already-asked gaps
	PendingClarifications []*database.ClarificationQuestion

	// FIX #14: Sentence analyses from this message
	// Loaded by orchestrator, used by Layer 5 for conflict detection
	SentenceAnalyses []*database.SentenceAnalysisData

	// FIX #52: Per-conversation context change tracker
	// Created fresh per conversation, NOT shared across conversations
	// Prevents data contamination between users/conversations
	ContextChangeTracker interface{} // agents.ContextChangeTracker (interface{} to avoid import cycle)

	// FIX #57: Clarification gaps from Layer 9 topic shift detection
	Layer9ClarificationGaps []Gap

	// PHASE 2: Contact Workflow State
	// Pre-layer-1 contact detection and disambiguation
	ClarificationNeeded      bool                   // True if contact ambiguity requires clarification
	ClarificationID          string                 // Unique ID for this clarification (for tracking)
	ClarificationFlag        string                 // Type: "contact_ambiguity", "group_membership", etc.
	ClarificationQuestion    string                 // Question to ask user
	ClarificationOptions     []string               // A/B/C options for user to select
	ClarificationConfidence  float64                // Confidence in the ambiguity (lower = more uncertain)
	ActiveContacts           []*models.Contact      // Resolved contacts for this message
	PronounResolutions       map[string]interface{} // Pronoun → Contact mapping (could use PronounResolution type)
	ContactContext           string                 // Context string for extraction: "Previous contacts were: ..."
}

// Layer1Result - Context extraction phase results
type Layer1Result struct {
	ExtractedContext  *models.ExtractedContext
	Confidence        float64
	Duration          float64
	ExtractedGoal     string // FIX #5: User's goal (e.g., "write_message", "decide_disclosure")
	ConversationTopic string // FIX #6: Conversation focus (e.g., "girl", "Christine_sub")
}

// Layer2Result - Principle evaluation results
type Layer2Result struct {
	Verdict           *ConstitutionalVerdict
	IsObviousHarm     bool
	MatchedPrinciples []string
	ShouldProceedToL6 bool
}

// Layer3Result - Maturity assessment results
type Layer3Result struct {
	MaturityScore   float64
	ContextQuality  string
	GateLevel       string // "immature", "developing", "mature"
	CanAccessL5Plus bool
}

// Layer4Result - Gap detection results
type Layer4Result struct {
	DetectedGaps           []Gap
	GapCount               int
	CriticalGaps           []Gap
	ShouldClarify          bool
	ClarificationQuestions []*database.ClarificationQuestion // FIX #3 Phase 3: Confidence-driven clarifications
}

// Gap represents a missing piece of context
// FIX #46: Extended to include detected context changes
type Gap struct {
	Type        string // "missing_profile", "vague_contact", "unclear_intention", "ambiguous_entity", "intention_changed", "goal_changed", "meta_instruction_conflict"
	Description string
	Severity    string // "critical", "medium", "low"
	Confidence  float64
	SourceFix   string // FIX #46: Track which fix created this gap (e.g., "FIX #43", "FIX #44")
	GoalTarget  string // FIX #72 Phase 2: Which goal does this gap relate to? "primary_goal", "current_goal", or "both"
}

// Layer5Result - Conflict detection results
type Layer5Result struct {
	DetectedConflicts      []Conflict
	ConflictCount          int
	CriticalConflicts      []Conflict
	ClarificationQuestions []*database.ClarificationQuestion
}

// Conflict represents a detected conflict between extracted and saved data
type Conflict struct {
	Type        string // "value_contradiction", "subject_mismatch", "new_entity"
	Severity    string // "critical", "high", "medium", "low"
	Confidence  float64
	Description string
	Resolution  string // "ask_clarification", "update_profile", "ignore"
}

// Layer6Result - Ambiguous request handling results
type Layer6Result struct {
	IsAmbiguous             bool
	AmbiguousElements       []string
	ClarificationQuestions  []string
	ShouldProceedToResponse bool
}

// Layer7Result - Principle violation clarification results
type Layer7Result struct {
	ViolationDetected      bool
	ClarificationQuestions []string
	ShouldAskBeforeReject  bool
}

// Layer8Result - Socratic deepening results
type Layer8Result struct {
	SocraticQuestions []string
	QuestionStrategy  string // "explore_values", "challenge_assumption", "expand_perspective"
	Depth             string // "surface", "moderate", "deep"
}

// TopicShift represents a detected topic or contact change
type TopicShift struct {
	Type       string // "contact_change", "topic_change"
	Severity   string // "high", "medium", "low"
	Confidence float64
}

// Layer9Result - Topic/contact shift detection results
type Layer9Result struct {
	DetectedShifts        []TopicShift
	ShiftCount            int
	RequiresContextSwitch bool
	TopicShifted          bool
	PreviousTopic         string
	CurrentTopic          string
	ContactShifted        bool
	PreviousContact       string
	CurrentContact        string
	ShouldResetContext    bool
}

// Layer10Result - Persistent questioning results
type Layer10Result struct {
	PersistentQuestions []string
	QuestionCount       int
	AllowResponse       bool
}

// Layer11Result - Denial protocol results
type Layer11Result struct {
	ShouldDeny    bool
	DenialMessage string
	Reason        string
	Resources     []string
	AltSuggestion string
}

// NewLayerContext creates a new context for a message
func NewLayerContext(
	analysisCtx *models.AnalysisContext,
	userID, messageID, conversationID string,
	maturityContext *models.ConversationMaturity,
) *LayerContext {
	// FIX #22 & #23: Extract insights/reflections from analysis context
	insights := []models.Reflection{}
	reflections := []models.Reflection{}
	if analysisCtx != nil {
		// RecentInsights are previous conversation insights
		if analysisCtx.RecentMessages != nil {
			// Would be populated from database via AnalysisContext
		}
	}

	return &LayerContext{
		Analysis:                     analysisCtx,
		StartTime:                    time.Now().Unix(),
		UserID:                       userID,
		MessageID:                    messageID,
		ConversationID:               conversationID,
		ShouldStop:                   false,
		MaturityContext:              maturityContext, // FIX #2 (Session 34): Pass pre-loaded maturity
		AccumulatedExtractedEntities: make([]models.ExtractedEntity, 0),
		MessageSummaryCache:          make(map[string]interface{}), // FIX #11: Initialize cache
		RecentInsights:               insights,                     // FIX #22: Insights from previous messages
		RelevantReflections:          reflections,                  // FIX #23: Reflections for current contacts
	}
}

// SetAccumulatedContext sets context from previous messages
// PHASE 5: Provides previous message data to current layers for comparison
func (lc *LayerContext) SetAccumulatedContext(
	previousEntities []models.ExtractedEntity,
	previousGoal string,
	previousValues []string,
) {
	lc.AccumulatedExtractedEntities = previousEntities
	lc.PreviousGoal = previousGoal
	lc.PreviousValues = previousValues
}

// FIX #11: GetMessageSummary retrieves cached summary for a message
// Phase 3 optimization: Layers can use cached summaries instead of re-processing
func (lc *LayerContext) GetMessageSummary(messageID string) interface{} {
	if lc.MessageSummaryCache == nil {
		return nil
	}
	return lc.MessageSummaryCache[messageID]
}

// FIX #11: HasMessageSummary checks if a message has a cached summary
func (lc *LayerContext) HasMessageSummary(messageID string) bool {
	if lc.MessageSummaryCache == nil {
		return false
	}
	_, exists := lc.MessageSummaryCache[messageID]
	return exists
}

// GetMessage returns the current message being analyzed
func (lc *LayerContext) GetMessage() string {
	if lc.Analysis != nil {
		return lc.Analysis.CurrentMessage
	}
	return ""
}

// GetUserProfile returns the user's stored profile
func (lc *LayerContext) GetUserProfile() *models.AboutMe {
	if lc.Analysis != nil {
		return lc.Analysis.UserProfile
	}
	return nil
}

// GetRelevantContacts returns contacts mentioned in conversation
func (lc *LayerContext) GetRelevantContacts() []models.Contact {
	if lc.Analysis != nil {
		return lc.Analysis.RelevantContacts
	}
	return []models.Contact{}
}

// GetExtractionConfidence returns extraction confidence from Layer 1
func (lc *LayerContext) GetExtractionConfidence() float64 {
	if lc.Analysis != nil {
		return lc.Analysis.ExtractedConfidence
	}
	return 0
}

// GetMaturityScore returns maturity score from Layer 3
func (lc *LayerContext) GetMaturityScore() float64 {
	if lc.Layer3 != nil {
		return lc.Layer3.MaturityScore
	}
	return 0
}

// GetGaps returns detected gaps from Layer 4
func (lc *LayerContext) GetGaps() []Gap {
	if lc.Layer4 != nil {
		return lc.Layer4.DetectedGaps
	}
	return []Gap{}
}

// GetConflicts returns detected conflicts from Layer 5
func (lc *LayerContext) GetConflicts() []Conflict {
	if lc.Layer5 != nil {
		return lc.Layer5.DetectedConflicts
	}
	return []Conflict{}
}

// IsObviousHarm returns whether Layer 2 detected obvious harm
func (lc *LayerContext) IsObviousHarm() bool {
	if lc.Layer2 != nil {
		return lc.Layer2.IsObviousHarm
	}
	return false
}

// CanProceedToLayer5 checks if maturity allows Layer 5+ operations
func (lc *LayerContext) CanProceedToLayer5() bool {
	if lc.Layer3 != nil {
		return lc.Layer3.CanAccessL5Plus
	}
	return false
}

// HasCriticalGaps checks if there are critical gaps
func (lc *LayerContext) HasCriticalGaps() bool {
	if lc.Layer4 != nil {
		return len(lc.Layer4.CriticalGaps) > 0
	}
	return false
}
