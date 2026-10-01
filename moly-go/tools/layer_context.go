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
	Layer1 *Layer1Result
	Layer2 *Layer2Result
	Layer3 *Layer3Result
	Layer4 *Layer4Result
	Layer5 *Layer5Result
	Layer6 *Layer6Result
	Layer7 *Layer7Result
	Layer8 *Layer8Result
	Layer9 *Layer9Result
	Layer10 *Layer10Result
	Layer11 *Layer11Result

	// Control flow
	ShouldStop bool   // Set to true to skip remaining layers
	StopReason string // Why we stopped (e.g., "obvious_harm")
	StopAfterLayer4IfGapsFound bool // ARCHITECTURAL FIX #1: Signal from Layer 3 to Layer 4: if immature, stop if gaps

	// Metadata
	StartTime   int64 // Unix timestamp
	UserID      string
	MessageID   string
	ConversationID string
}

// Layer1Result - Context extraction phase results
type Layer1Result struct {
	ExtractedContext *models.ExtractedContext
	Confidence       float64
	Duration         float64
}

// Layer2Result - Principle evaluation results
type Layer2Result struct {
	Verdict            *ConstitutionalVerdict
	IsObviousHarm      bool
	MatchedPrinciples  []string
	ShouldProceedToL6  bool
}

// Layer3Result - Maturity assessment results
type Layer3Result struct {
	MaturityScore      float64
	ContextQuality     string
	GateLevel          string // "immature", "developing", "mature"
	CanAccessL5Plus    bool
}

// Layer4Result - Gap detection results
type Layer4Result struct {
	DetectedGaps    []Gap
	GapCount        int
	CriticalGaps    []Gap
	ShouldClarify   bool
}

// Gap represents a missing piece of context
type Gap struct {
	Type        string  // "missing_profile", "vague_contact", "unclear_intention", "ambiguous_entity"
	Description string
	Severity    string  // "critical", "medium", "low"
	Confidence  float64
}

// Layer5Result - Conflict detection results
type Layer5Result struct {
	DetectedConflicts    []Conflict
	ConflictCount        int
	CriticalConflicts    []Conflict
	ClarificationQuestions []*database.ClarificationQuestion
}

// Conflict represents a detected conflict between extracted and saved data
type Conflict struct {
	Type        string  // "value_contradiction", "subject_mismatch", "new_entity"
	Severity    string  // "critical", "high", "medium", "low"
	Confidence  float64
	Description string
	Resolution  string // "ask_clarification", "update_profile", "ignore"
}

// Layer6Result - Ambiguous request handling results
type Layer6Result struct {
	IsAmbiguous            bool
	AmbiguousElements      []string
	ClarificationQuestions []string
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
	Depth            string // "surface", "moderate", "deep"
}

// TopicShift represents a detected topic or contact change
type TopicShift struct {
	Type       string  // "contact_change", "topic_change"
	Severity   string  // "high", "medium", "low"
	Confidence float64
}

// Layer9Result - Topic/contact shift detection results
type Layer9Result struct {
	DetectedShifts       []TopicShift
	ShiftCount           int
	RequiresContextSwitch bool
	TopicShifted         bool
	PreviousTopic        string
	CurrentTopic         string
	ContactShifted       bool
	PreviousContact      string
	CurrentContact       string
	ShouldResetContext   bool
}

// Layer10Result - Persistent questioning results
type Layer10Result struct {
	PersistentQuestions []string
	QuestionCount       int
	AllowResponse       bool
}

// Layer11Result - Denial protocol results
type Layer11Result struct {
	ShouldDeny       bool
	DenialMessage    string
	Reason           string
	Resources        []string
	AltSuggestion    string
}

// NewLayerContext creates a new context for a message
func NewLayerContext(
	analysisCtx *models.AnalysisContext,
	userID, messageID, conversationID string,
) *LayerContext {
	return &LayerContext{
		Analysis:       analysisCtx,
		StartTime:      time.Now().Unix(),
		UserID:         userID,
		MessageID:      messageID,
		ConversationID: conversationID,
		ShouldStop:     false,
	}
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
