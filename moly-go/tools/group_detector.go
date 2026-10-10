package tools

import (
	"strings"
)

// GroupReference represents a reference to a group of people
type GroupReference struct {
	ID               int64
	UserID           string
	ConversationID   string
	ReferencePronoun string // "they", "we", "both", "all", "us"
	ReferenceType    string // "dual" (2), "plural" (3+), "collection"
	Members          []string
	MemberIDs        map[string]int64 // name → contact ID
	IsUserInGroup    bool
	GroupContext     string // "couple", "trio", "group", "friends", "team"
	Confidence       float64
	EvidenceText     string
	DetectedFrom     string // Which sentence/message detected this
	CreatedAt        int64
}

// GroupDetector analyzes text to identify group references and membership
type GroupDetector struct{}


// ============================================================================
// GROUP PRONOUN DETECTION
// ============================================================================


// ============================================================================
// GROUP MEMBERSHIP DETECTION
// ============================================================================


// ============================================================================
// GROUP TYPE CLASSIFICATION
// ============================================================================


// ============================================================================
// GROUP CONTEXT INFERENCE
// ============================================================================


// ============================================================================
// GROUP CONTEXT ANALYSIS
// ============================================================================


// ============================================================================
// AMBIGUITY DETECTION
// ============================================================================

// IsAmbiguous returns true if group membership is unclear
// Group is ambiguous if:
// - No explicit members mentioned
// - "they" without recent context
// - "we" but unclear who is included
func (gd *GroupDetector) IsAmbiguous(ref *GroupReference) bool {
	if len(ref.Members) == 0 {
		return true
	}

	if strings.ToLower(ref.ReferencePronoun) == "they" || strings.ToLower(ref.ReferencePronoun) == "them" {
		// "They" is ambiguous unless we have recent context
		return ref.Confidence < 0.70
	}

	return ref.Confidence < 0.50
}

// ============================================================================
// GROUP RELATIONSHIP MAPPING
// ============================================================================


// ============================================================================
// SCOPE TRACKING
// ============================================================================


// ============================================================================
// GENDER COMPOSITION ANALYSIS
// ============================================================================

