package tools

import (
)



// ============================================================================
// ORCHESTRATION FLOW
// ============================================================================



// ============================================================================
// CONTEXT-AWARE SUBJECT ATTRIBUTION
// ============================================================================


// ============================================================================
// EXTRACTION WITH CONTEXT
// ============================================================================


// ============================================================================
// DATA STRUCTURES
// ============================================================================

// MessageAnalysis represents complete analysis of a message
type MessageAnalysis struct {
	OriginalMessage       string
	Sentences             []Sentence
	SentenceAnalyses      []SentenceAnalysis
	PronounResolutions    map[string]*PronounResolution
	GroupReferences       map[string]*GroupReference
	SubjectContextMapping map[int]string // sentence number → resolved subject
}



// ============================================================================
// DEBUGGING & INSPECTION
// ============================================================================

