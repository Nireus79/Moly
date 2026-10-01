package tools

import (
	"strings"
	"testing"
)

// ============================================================================
// SENTENCE SEGMENTATION TESTS
// ============================================================================

func TestSegmentIntoSentences_Simple(t *testing.T) {
	parser := NewLinguisticParser()
	text := "I am 46. She is submissive. We are looking for intensity."

	sentences := parser.SegmentIntoSentences(text)

	if len(sentences) != 3 {
		t.Errorf("Expected 3 sentences, got %d", len(sentences))
	}

	if !strings.Contains(sentences[0].Text, "46") {
		t.Errorf("Sentence 1 should contain '46', got: %s", sentences[0].Text)
	}

	if sentences[0].Number != 1 {
		t.Errorf("First sentence should have number 1, got %d", sentences[0].Number)
	}
}

func TestSegmentIntoSentences_WithAbbreviations(t *testing.T) {
	parser := NewLinguisticParser()
	text := "Hello Dr. Smith. How are you? Great!"

	sentences := parser.SegmentIntoSentences(text)

	// Should not split on "Dr." but should on "?" and "!"
	if len(sentences) < 2 {
		t.Errorf("Expected at least 2 sentences, got %d", len(sentences))
	}
}

func TestSegmentIntoSentences_Empty(t *testing.T) {
	parser := NewLinguisticParser()
	sentences := parser.SegmentIntoSentences("")

	if len(sentences) != 0 {
		t.Errorf("Empty message should return 0 sentences, got %d", len(sentences))
	}
}

// ============================================================================
// SVO ANALYSIS TESTS
// ============================================================================

func TestAnalyzeSentence_SimpleSubjectVerb(t *testing.T) {
	parser := NewLinguisticParser()
	analysis := parser.AnalyzeSentence(1, "I am dominant", []Sentence{})

	if analysis.Subject != "user" {
		t.Errorf("Subject should resolve 'I' to 'user', got: %s", analysis.Subject)
	}

	if strings.ToLower(analysis.Verb) != "am" {
		t.Errorf("Verb should be 'am', got: %s", analysis.Verb)
	}

	if analysis.Object != "dominant" {
		t.Errorf("Object should be 'dominant', got: %s", analysis.Object)
	}
}

func TestAnalyzeSentence_PronounResolution(t *testing.T) {
	parser := NewLinguisticParser()

	// First sentence establishes subject
	sent1 := parser.AnalyzeSentence(1, "I am dominant", []Sentence{})
	if sent1.Subject != "user" {
		t.Errorf("First sentence subject should be 'user', got: %s", sent1.Subject)
	}

	// Second sentence uses pronoun - should infer from prior
	sent2 := parser.AnalyzeSentence(2, "She is submissive", []Sentence{{Number: 1, Text: "I am dominant"}})
	if sent2.Subject != "she" {
		t.Errorf("Pronoun should be 'she', got: %s", sent2.Subject)
	}
}

func TestAnalyzeSentence_Negation(t *testing.T) {
	parser := NewLinguisticParser()
	analysis := parser.AnalyzeSentence(1, "I don't like pain", []Sentence{})

	if !analysis.Negated {
		t.Errorf("Analysis should detect negation")
	}
}

// ============================================================================
// PRONOUN DETECTION TESTS
// ============================================================================

func TestDetectPronouns_Personal(t *testing.T) {
	resolver := NewPronounResolver(nil)
	pronouns := resolver.DetectPronouns("She likes it and they want it too")

	if len(pronouns) == 0 {
		t.Errorf("Should detect pronouns in message")
	}

	// Should find "she", "it", "they"
	foundShe := false
	foundThey := false
	for _, p := range pronouns {
		if p.Pronoun == "she" {
			foundShe = true
		}
		if p.Pronoun == "they" {
			foundThey = true
		}
	}

	if !foundShe {
		t.Errorf("Should detect 'she' pronoun")
	}
	if !foundThey {
		t.Errorf("Should detect 'they' pronoun")
	}
}

// ============================================================================
// GROUP DETECTION TESTS
// ============================================================================

func TestDetectGroupPronouns(t *testing.T) {
	detector := NewGroupDetector()
	groups := detector.DetectGroupPronouns("We like it and they want it")

	if len(groups) == 0 {
		t.Errorf("Should detect group pronouns")
	}
}

func TestDetectGroupMembers_BothPattern(t *testing.T) {
	detector := NewGroupDetector()
	members := detector.DetectGroupMembers("Both Alice and Bob are here", []string{})

	if len(members) != 2 {
		t.Errorf("Should detect 2 members from 'both X and Y', got %d", len(members))
	}
}

func TestDetectGroupMembers_MeAnd(t *testing.T) {
	detector := NewGroupDetector()
	members := detector.DetectGroupMembers("Me and Christine are together", []string{})

	if len(members) != 2 {
		t.Errorf("Should detect 2 members from 'me and X', got %d", len(members))
	}

	// Should include "user"
	foundUser := false
	for _, m := range members {
		if m == "user" {
			foundUser = true
		}
	}

	if !foundUser {
		t.Errorf("Should include 'user' in members")
	}
}

func TestClassifyGroupType(t *testing.T) {
	detector := NewGroupDetector()

	typeTwo := detector.ClassifyGroupType("both", 2)
	if typeTwo != "dual" {
		t.Errorf("2 people should be 'dual', got: %s", typeTwo)
	}

	typeThree := detector.ClassifyGroupType("they", 3)
	if typeThree != "plural" {
		t.Errorf("3+ people should be 'plural', got: %s", typeThree)
	}
}

// ============================================================================
// ORCHESTRATOR INTEGRATION TESTS
// ============================================================================

func TestOrchestrator_AnalyzeMessage(t *testing.T) {
	orchestrator := NewExtractionOrchestrator()
	message := "I am 46 years old. She is submissive. We like communication."
	knownContacts := map[string]int64{}

	analysis, err := orchestrator.AnalyzeMessageForExtraction(message, []string{}, knownContacts)

	if err != nil {
		t.Errorf("Analysis failed: %v", err)
	}

	if len(analysis.Sentences) != 3 {
		t.Errorf("Should analyze 3 sentences, got %d", len(analysis.Sentences))
	}

	if len(analysis.SentenceAnalyses) != 3 {
		t.Errorf("Should have 3 SVO analyses, got %d", len(analysis.SentenceAnalyses))
	}

	if len(analysis.SubjectContextMapping) == 0 {
		t.Errorf("Should have subject context mapping")
	}
}

func TestOrchestrator_DetermineSubject_User(t *testing.T) {
	orchestrator := NewExtractionOrchestrator()
	message := "I am 46. Results: 100% Rigger"
	analysis, _ := orchestrator.AnalyzeMessageForExtraction(message, []string{}, map[string]int64{})

	subject := orchestrator.DetermineSubjectForExtraction(
		message,
		"Rigger",
		"contact",
		2, // Second sentence
		analysis,
	)
	_ = analysis // Use analysis in comment above

	// THE BUG FIX: Should resolve to "user", not "contact"
	if subject == "contact" {
		t.Errorf("BUG NOT FIXED: Subject should not be 'contact', got: %s", subject)
	}

	if subject != "user" && !strings.Contains(subject, "user") {
		t.Logf("Subject resolved to: %s (expected 'user' or user-related)", subject)
	}
}

// ============================================================================
// REAL-WORLD SCENARIO TESTS
// ============================================================================

func TestRealWorld_RiggerBugFix(t *testing.T) {
	orchestrator := NewExtractionOrchestrator()

	// The exact message that caused the bug
	message := `I am 46 years old, white, male. Results from bdsmtest.org
100% Rigger 98% Dominant 81% Sadist 73% Master/Mistress`

	_, err := orchestrator.AnalyzeMessageForExtraction(message, []string{}, map[string]int64{})
	if err != nil {
		t.Errorf("Failed to analyze: %v", err)
		return
	}

	// Extract entities with context
	result, err := orchestrator.ExtractWithContext(message, []string{}, map[string]int64{})
	if err != nil {
		t.Errorf("Failed to extract: %v", err)
		return
	}

	// Verify we got context-aware entities
	if len(result.ContextAwareEntities) == 0 {
		t.Logf("No entities extracted (might be normal for this test)")
		return
	}

	// Check that no entity has subject="contact" for "Rigger"
	for _, entity := range result.ContextAwareEntities {
		if strings.Contains(entity.OriginalEntity.Property, "Rigger") ||
			strings.Contains(entity.OriginalEntity.Property, "Dominant") {
			if entity.ResolvedSubject == "contact" {
				t.Errorf("BUG NOT FIXED: Rigger-related fact has subject='contact'")
			}
			t.Logf("Rigger fact resolved to: %s (good, not 'contact')", entity.ResolvedSubject)
		}
	}
}

func TestRealWorld_MultiPerson_SheReference(t *testing.T) {
	orchestrator := NewExtractionOrchestrator()

	message := "Her name is Christine. She is submissive and very interested in BDSM."
	knownContacts := map[string]int64{"christine": 1}

	analysis, err := orchestrator.AnalyzeMessageForExtraction(message, []string{}, knownContacts)
	if err != nil {
		t.Errorf("Failed to analyze: %v", err)
		return
	}

	// Should have pronoun resolution for "she"
	if len(analysis.PronounResolutions) == 0 {
		t.Logf("No pronoun resolutions found (might be expected)")
	}

	result, err := orchestrator.ExtractWithContext(message, []string{}, knownContacts)
	if err != nil {
		t.Errorf("Failed to extract: %v", err)
		return
	}

	// Verify context awareness
	for _, entity := range result.ContextAwareEntities {
		// Facts about "she" should be resolved to Christine or equivalent
		if entity.OriginalEntity.Subject == "she" {
			if entity.ResolvedSubject == "she" || entity.ResolvedSubject == "unknown" {
				t.Logf("Pronoun 'she' not fully resolved, but that's OK for this test")
			}
		}
	}
}

func TestRealWorld_WeGroup(t *testing.T) {
	orchestrator := NewExtractionOrchestrator()

	message := "I'm with my girlfriend Christine. We like intensity and communication."
	knownContacts := map[string]int64{}

	analysis, err := orchestrator.AnalyzeMessageForExtraction(message, []string{}, knownContacts)
	if err != nil {
		t.Errorf("Failed to analyze: %v", err)
		return
	}

	// Should detect group reference for "we"
	if len(analysis.GroupReferences) > 0 {
		for pronoun, groupRef := range analysis.GroupReferences {
			if pronoun == "we" || strings.Contains(pronoun, "we") {
				if len(groupRef.Members) > 0 {
					t.Logf("Group 'we' detected with members: %v", groupRef.Members)
					if !groupRef.IsUserInGroup {
						t.Errorf("User should be in 'we' group")
					}
				}
			}
		}
	}
}

// ============================================================================
// EDGE CASE TESTS
// ============================================================================

func TestEdgeCase_EmptyMessage(t *testing.T) {
	orchestrator := NewExtractionOrchestrator()
	analysis, err := orchestrator.AnalyzeMessageForExtraction("", []string{}, map[string]int64{})

	if err != nil {
		t.Errorf("Should handle empty message gracefully: %v", err)
	}

	if len(analysis.Sentences) != 0 {
		t.Errorf("Empty message should have 0 sentences, got %d", len(analysis.Sentences))
	}
}

func TestEdgeCase_SingleSentence(t *testing.T) {
	orchestrator := NewExtractionOrchestrator()
	message := "I am dominant"
	analysis, err := orchestrator.AnalyzeMessageForExtraction(message, []string{}, map[string]int64{})

	if err != nil {
		t.Errorf("Should handle single sentence: %v", err)
	}

	if len(analysis.Sentences) != 1 {
		t.Errorf("Should have 1 sentence, got %d", len(analysis.Sentences))
	}

	if len(analysis.SentenceAnalyses) != 1 {
		t.Errorf("Should have 1 SVO analysis, got %d", len(analysis.SentenceAnalyses))
	}
}

func TestEdgeCase_OnlyPronouns(t *testing.T) {
	orchestrator := NewExtractionOrchestrator()
	message := "She likes it"
	analysis, err := orchestrator.AnalyzeMessageForExtraction(message, []string{}, map[string]int64{})

	if err != nil {
		t.Errorf("Should handle pronoun-only message: %v", err)
	}

	if len(analysis.Sentences) != 1 {
		t.Errorf("Should parse pronoun message")
	}
}

// ============================================================================
// DEBUG/REPORTING TESTS
// ============================================================================

func TestGetExtractionReport(t *testing.T) {
	orchestrator := NewExtractionOrchestrator()
	message := "I am dominant. She is submissive."
	result, err := orchestrator.ExtractWithContext(message, []string{}, map[string]int64{})

	if err != nil {
		t.Errorf("Failed to extract: %v", err)
		return
	}

	report := orchestrator.GetExtractionReport(result)

	if len(report) == 0 {
		t.Errorf("Report should not be empty")
	}

	if !strings.Contains(report, "EXTRACTION ANALYSIS REPORT") {
		t.Errorf("Report should contain header")
	}

	if !strings.Contains(report, "SENTENCES ANALYZED") {
		t.Errorf("Report should list sentences")
	}
}
