package tools

import (
	"fmt"
	"strings"
)

// ExtractionOrchestrator coordinates sentence analysis, pronoun resolution, and group detection
// This ties together all the Phase 2-5 components into a unified extraction flow
type ExtractionOrchestrator struct {
	parser          *LinguisticParser
	pronounResolver *PronounResolver
	groupDetector   *GroupDetector
}

// NewExtractionOrchestrator creates a new orchestrator
func NewExtractionOrchestrator() *ExtractionOrchestrator {
	return &ExtractionOrchestrator{
		parser:          NewLinguisticParser(),
		pronounResolver: NewPronounResolver(nil), // DB will be injected later
		groupDetector:   NewGroupDetector(),
	}
}

// ============================================================================
// ORCHESTRATION FLOW
// ============================================================================

// AnalyzeMessageForExtraction performs complete sentence-level analysis on a message
// Returns all analyses needed to properly attribute extracted facts
// Flow: Segment → Analyze → Resolve Pronouns → Detect Groups
func (eo *ExtractionOrchestrator) AnalyzeMessageForExtraction(
	message string,
	recentMessages []string,
	knownContacts map[string]int64, // contact name → contact ID
) (*MessageAnalysis, error) {

	analysis := &MessageAnalysis{
		OriginalMessage:        message,
		Sentences:              []Sentence{},
		SentenceAnalyses:       []SentenceAnalysis{},
		PronounResolutions:     make(map[string]*PronounResolution),
		GroupReferences:        make(map[string]*GroupReference),
		SubjectContextMapping:  make(map[int]string), // sentence number → resolved subject
	}

	// STEP 1: Segment message into sentences
	sentences := eo.parser.SegmentIntoSentences(message)
	if len(sentences) == 0 {
		return analysis, nil // Empty message
	}
	analysis.Sentences = sentences

	// STEP 2: Analyze each sentence (SVO parsing)
	for i, sentence := range sentences {
		priorSentences := sentences[:i]
		svoAnalysis := eo.parser.AnalyzeSentence(sentence.Number, sentence.Text, priorSentences)
		analysis.SentenceAnalyses = append(analysis.SentenceAnalyses, svoAnalysis)
	}

	// STEP 3: Resolve pronouns and groups
	// Build list of entities mentioned in this message
	mentionedEntities := eo.extractMentionedEntities(sentences)

	for i, svoAnalysis := range analysis.SentenceAnalyses {
		// Check if subject is a pronoun that needs resolution
		subject := svoAnalysis.Subject
		lower := strings.ToLower(subject)

		// Check if it's a group pronoun
		if eo.pronounResolver.IsGroupPronoun(lower) {
			// STEP 3A: Resolve group
			groupRef := eo.groupDetector.AnalyzeGroupContext(
				svoAnalysis.SentenceText,
				lower,
				mentionedEntities,
				knownContacts,
			)
			analysis.GroupReferences[lower] = groupRef

			// Update subject context mapping
			if len(groupRef.Members) > 0 {
				analysis.SubjectContextMapping[i+1] = fmt.Sprintf("group:%s", strings.Join(groupRef.Members, ","))
			} else {
				analysis.SubjectContextMapping[i+1] = lower // Unresolved group
			}
		} else if eo.pronounResolver.IsGroupPronoun(lower) == false &&
			(lower == "she" || lower == "he" || lower == "they" || lower == "it") {
			// STEP 3B: Resolve individual pronoun
			pronRes, err := eo.pronounResolver.ResolveAntecedent(
				"", // userID - will be set during extraction
				"", // conversationID - will be set during extraction
				lower,
				i+1,
				sentences,
				recentMessages,
				knownContacts,
			)
			if err == nil && pronRes != nil {
				analysis.PronounResolutions[lower] = pronRes

				// Update subject context mapping
				if pronRes.Confidence > 0 {
					analysis.SubjectContextMapping[i+1] = pronRes.AntecedentValue
				} else {
					analysis.SubjectContextMapping[i+1] = lower // Unresolved
				}
			}
		} else {
			// Not a pronoun - keep as-is
			analysis.SubjectContextMapping[i+1] = subject
		}
	}

	return analysis, nil
}

// extractMentionedEntities finds all entity names mentioned in sentences
func (eo *ExtractionOrchestrator) extractMentionedEntities(sentences []Sentence) []string {
	var entities []string
	seen := make(map[string]bool)

	for _, sentence := range sentences {
		words := strings.Fields(sentence.Text)
		for _, word := range words {
			// Look for capitalized words (likely names)
			if len(word) > 0 && word[0] >= 'A' && word[0] <= 'Z' {
				lower := strings.ToLower(word)
				if !seen[lower] && lower != "i" && lower != "the" {
					entities = append(entities, lower)
					seen[lower] = true
				}
			}
		}
	}

	return entities
}

// ============================================================================
// CONTEXT-AWARE SUBJECT ATTRIBUTION
// ============================================================================

// DetermineSubjectForExtraction uses context to determine the proper subject for an extracted fact
// This is where we fix the "100% Rigger" bug by checking preceding text
func (eo *ExtractionOrchestrator) DetermineSubjectForExtraction(
	message string,
	extractedProperty string,
	rawSubject string,
	sentenceNum int,
	analysis *MessageAnalysis,
) string {

	// If we have a subject context mapping from SVO analysis, use that
	if resolvedSubject, ok := analysis.SubjectContextMapping[sentenceNum]; ok && resolvedSubject != "" {
		return resolvedSubject
	}

	// Fallback: Check preceding text for context clues
	contextSubject := eo.parser.DetectSubjectContext(message, len(message)/2) // Approximate position

	// If raw subject is a pronoun, try to resolve it from known resolutions
	if resolution, ok := analysis.PronounResolutions[rawSubject]; ok {
		if resolution.Confidence > 0.5 {
			return resolution.AntecedentValue
		}
	}

	// If raw subject is a group pronoun, return group reference
	if groupRef, ok := analysis.GroupReferences[rawSubject]; ok {
		if len(groupRef.Members) > 0 {
			return fmt.Sprintf("group:%s", strings.Join(groupRef.Members, ","))
		}
	}

	// Default: Use context or raw subject
	if contextSubject != "ambiguous" {
		return contextSubject
	}

	return rawSubject
}

// ============================================================================
// EXTRACTION WITH CONTEXT
// ============================================================================

// ExtractWithContext performs entity extraction with full context awareness
// Returns entities with properly resolved subjects
func (eo *ExtractionOrchestrator) ExtractWithContext(
	message string,
	recentMessages []string,
	knownContacts map[string]int64,
) (*ContextAwareExtractionResult, error) {

	// First, perform comprehensive analysis
	analysis, err := eo.AnalyzeMessageForExtraction(message, recentMessages, knownContacts)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze message: %w", err)
	}

	result := &ContextAwareExtractionResult{
		OriginalMessage:      message,
		SentenceAnalyses:     analysis.SentenceAnalyses,
		PronounResolutions:   analysis.PronounResolutions,
		GroupReferences:      analysis.GroupReferences,
		ExtractedEntities:    []ExtractionResult{},
		ContextAwareEntities: []ContextAwareEntity{},
	}

	// Now extract entities using existing parser but with better subject attribution
	existingEntities := eo.parser.Parse(message)

	// For each extracted entity, enhance with context information
	for _, entity := range existingEntities {
		// Find which sentence this came from (approximate)
		sentenceNum := 1 // Default to first sentence
		for _, sent := range analysis.Sentences {
			if strings.Contains(sent.Text, entity.RawMatch) {
				sentenceNum = sent.Number
				break
			}
		}

		// Determine proper subject using context
		resolvedSubject := eo.DetermineSubjectForExtraction(
			message,
			entity.Property,
			entity.Subject,
			sentenceNum,
			analysis,
		)

		// Calculate confidence based on context
		confidence := entity.Confidence
		if resolvedSubject != entity.Subject && resolvedSubject != "ambiguous" {
			// Subject was corrected by context - might lower confidence slightly
			confidence = confidence * 0.95 // Small penalty for correction
		}

		// Store both original and context-aware versions
		result.ExtractedEntities = append(result.ExtractedEntities, entity)

		result.ContextAwareEntities = append(result.ContextAwareEntities, ContextAwareEntity{
			OriginalEntity:    entity,
			ResolvedSubject:   resolvedSubject,
			SubjectSentence:   sentenceNum,
			ContextConfidence: confidence,
		})
	}

	return result, nil
}

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

// ContextAwareExtractionResult contains extraction with full context
type ContextAwareExtractionResult struct {
	OriginalMessage      string
	SentenceAnalyses     []SentenceAnalysis
	PronounResolutions   map[string]*PronounResolution
	GroupReferences      map[string]*GroupReference
	ExtractedEntities    []ExtractionResult
	ContextAwareEntities []ContextAwareEntity
}

// ContextAwareEntity wraps an extraction with context information
type ContextAwareEntity struct {
	OriginalEntity    ExtractionResult
	ResolvedSubject   string  // After context resolution
	SubjectSentence   int     // Which sentence subject came from
	ContextConfidence float64 // Confidence after context application
}

// ============================================================================
// DEBUGGING & INSPECTION
// ============================================================================

// GetExtractionReport generates a human-readable report of extraction with context
func (eo *ExtractionOrchestrator) GetExtractionReport(result *ContextAwareExtractionResult) string {
	var report strings.Builder

	report.WriteString("=== EXTRACTION ANALYSIS REPORT ===\n\n")

	// Sentence analyses
	report.WriteString("SENTENCES ANALYZED:\n")
	for _, svo := range result.SentenceAnalyses {
		report.WriteString(fmt.Sprintf("  [%d] %s\n", svo.SentenceNumber, svo.SentenceText))
		report.WriteString(fmt.Sprintf("      Subject: %s | Verb: %s | Object: %s\n",
			svo.Subject, svo.Verb, svo.Object))
	}
	report.WriteString("\n")

	// Pronoun resolutions
	if len(result.PronounResolutions) > 0 {
		report.WriteString("PRONOUN RESOLUTIONS:\n")
		for pronoun, resolution := range result.PronounResolutions {
			if resolution.Confidence > 0 {
				report.WriteString(fmt.Sprintf("  %s → %s (confidence: %.2f)\n",
					pronoun, resolution.AntecedentValue, resolution.Confidence))
			}
		}
		report.WriteString("\n")
	}

	// Group references
	if len(result.GroupReferences) > 0 {
		report.WriteString("GROUP REFERENCES:\n")
		for pronoun, groupRef := range result.GroupReferences {
			report.WriteString(fmt.Sprintf("  %s (%s): %v\n",
				pronoun, groupRef.ReferenceType, groupRef.Members))
		}
		report.WriteString("\n")
	}

	// Extracted entities with context
	report.WriteString("EXTRACTED ENTITIES (WITH CONTEXT):\n")
	for _, entity := range result.ContextAwareEntities {
		report.WriteString(fmt.Sprintf("  Property: %s\n", entity.OriginalEntity.Property))
		report.WriteString(fmt.Sprintf("    Original subject: %s\n", entity.OriginalEntity.Subject))
		report.WriteString(fmt.Sprintf("    Resolved subject: %s (from sentence %d)\n",
			entity.ResolvedSubject, entity.SubjectSentence))
		report.WriteString(fmt.Sprintf("    Confidence: %.2f\n", entity.ContextConfidence))
	}

	return report.String()
}
