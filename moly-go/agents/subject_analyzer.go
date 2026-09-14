package agents

import (
	"fmt"
	"log"
	"strings"
)

// SubjectAnalysis contains the clarity analysis of a subject
type SubjectAnalysis struct {
	Fact              ExtractedFact
	ClarityLevel      string // "clear", "ambiguous", "missing"
	DetectedSubject   string // "user", "contact_boss", "unknown", "unknown_female", "unknown_male"
	RequiredQuestion  string // If clarity needed, what question to ask
}

// SubjectAnalyzer determines who/what each extracted fact is about
type SubjectAnalyzer struct{}

// NewSubjectAnalyzer creates a new subject analyzer
func NewSubjectAnalyzer() *SubjectAnalyzer {
	return &SubjectAnalyzer{}
}

// Analyze determines the clarity level and detected subject of a fact
func (a *SubjectAnalyzer) Analyze(fact ExtractedFact) SubjectAnalysis {
	log.Printf("[V2] SubjectAnalyzer.Analyze: fact=%s, subject=%s", fact.Value, fact.ProposedSubject)

	analysis := SubjectAnalysis{
		Fact: fact,
	}

	proposedLower := strings.ToLower(strings.TrimSpace(fact.ProposedSubject))

	// CLEAR: Explicit self-references (exact match only for short pronouns)
	if proposedLower == "user" || proposedLower == "i" || proposedLower == "me" || proposedLower == "my" ||
		proposedLower == "myself" || proposedLower == "we" || proposedLower == "us" || proposedLower == "our" ||
		proposedLower == "ourself" {
		analysis.ClarityLevel = "clear"
		analysis.DetectedSubject = "user"
		log.Printf("[V2]   → Clear: user (explicit self-reference)")
		return analysis
	}

	// CLEAR: Explicit relationship markers
	if isRole(proposedLower, "boss", "manager", "director", "lead", "ceo", "supervisor") {
		analysis.ClarityLevel = "clear"
		analysis.DetectedSubject = "contact_boss"
		log.Printf("[V2]   → Clear: contact_boss (explicit role)")
		return analysis
	}

	if isRole(proposedLower, "colleague", "coworker", "teammate", "peer", "work friend") {
		analysis.ClarityLevel = "clear"
		analysis.DetectedSubject = "contact_colleague"
		log.Printf("[V2]   → Clear: contact_colleague (explicit role)")
		return analysis
	}

	if isRole(proposedLower, "friend", "buddy", "mate", "pal") {
		analysis.ClarityLevel = "clear"
		analysis.DetectedSubject = "contact_friend"
		log.Printf("[V2]   → Clear: contact_friend (explicit role)")
		return analysis
	}

	if isRole(proposedLower, "partner", "spouse", "wife", "husband", "girlfriend", "boyfriend") {
		analysis.ClarityLevel = "clear"
		analysis.DetectedSubject = "contact_partner"
		log.Printf("[V2]   → Clear: contact_partner (explicit role)")
		return analysis
	}

	if isRole(proposedLower, "family", "parent", "mother", "father", "sister", "brother", "aunt", "uncle", "cousin") {
		analysis.ClarityLevel = "clear"
		analysis.DetectedSubject = "contact_family"
		log.Printf("[V2]   → Clear: contact_family (explicit role)")
		return analysis
	}

	// AMBIGUOUS: Pronouns that need clarification
	if proposedLower == "she" {
		analysis.ClarityLevel = "ambiguous"
		analysis.DetectedSubject = "unknown_female"
		analysis.RequiredQuestion = "Who is she? (e.g., boss, manager, colleague, friend, family member)"
		log.Printf("[V2]   → Ambiguous: female pronoun 'she'")
		return analysis
	}

	if proposedLower == "he" {
		analysis.ClarityLevel = "ambiguous"
		analysis.DetectedSubject = "unknown_male"
		analysis.RequiredQuestion = "Who is he? (e.g., boss, colleague, partner, friend, family member)"
		log.Printf("[V2]   → Ambiguous: male pronoun 'he'")
		return analysis
	}

	if proposedLower == "they" || proposedLower == "them" {
		analysis.ClarityLevel = "ambiguous"
		analysis.DetectedSubject = "unknown_group"
		analysis.RequiredQuestion = "Who are they? (e.g., team, department, friends, family)"
		log.Printf("[V2]   → Ambiguous: plural pronoun 'they'")
		return analysis
	}

	if strings.HasPrefix(proposedLower, "it") || strings.HasPrefix(proposedLower, "that") {
		analysis.ClarityLevel = "ambiguous"
		analysis.DetectedSubject = "unknown_thing"
		analysis.RequiredQuestion = "What are you referring to?"
		log.Printf("[V2]   → Ambiguous: non-person pronoun")
		return analysis
	}

	// AMBIGUOUS: Names or vague references without clear context
	// If it's a name or generic reference, we need clarification
	if isLikelyName(fact.ProposedSubject) {
		analysis.ClarityLevel = "ambiguous"
		analysis.DetectedSubject = fmt.Sprintf("contact_pending_%s", sanitizeName(fact.ProposedSubject))
		analysis.RequiredQuestion = fmt.Sprintf("What is %s's relationship to you? (e.g., boss, colleague, friend)", fact.ProposedSubject)
		log.Printf("[V2]   → Ambiguous: name without relationship context")
		return analysis
	}

	// MISSING: Empty or unrecognizable subject
	analysis.ClarityLevel = "missing"
	analysis.DetectedSubject = "unknown"
	analysis.RequiredQuestion = "Who are you referring to?"
	log.Printf("[V2]   → Missing: unrecognizable subject")
	return analysis
}

// containsAny checks if text contains any of the given substrings (case-insensitive)
func containsAny(text string, substrs ...string) bool {
	textLower := strings.ToLower(text)
	for _, sub := range substrs {
		if strings.Contains(textLower, strings.ToLower(sub)) {
			return true
		}
	}
	return false
}

// isRole checks if text matches or contains any role keyword
func isRole(text string, roles ...string) bool {
	// Exact match
	for _, role := range roles {
		if text == role {
			return true
		}
	}
	// Substring match (for phrases like "my boss" where "boss" is extracted)
	for _, role := range roles {
		if strings.Contains(text, " "+role) || strings.Contains(text, role+" ") || strings.Contains(text, " "+role+" ") {
			return true
		}
	}
	return false
}

// isLikelyName detects if a string looks like a proper name
func isLikelyName(s string) bool {
	lower := strings.ToLower(s)

	// Pronouns and common words are NOT names (exact match)
	pronouns := []string{"user", "i", "me", "my", "myself", "you", "your", "he", "she", "they", "it", "this", "that"}
	for _, p := range pronouns {
		if lower == p {
			return false
		}
	}

	// Names typically have capital letters or are in a reasonable length
	if len(s) == 0 || len(s) > 50 {
		return false
	}

	// Simple heuristic: if starts with capital or has mixed case, likely a name
	if len(s) > 0 && s[0] >= 'A' && s[0] <= 'Z' {
		return true
	}

	return false
}

// sanitizeName converts a name to a safe identifier
func sanitizeName(name string) string {
	lower := strings.ToLower(name)
	// Replace spaces and special chars with underscore
	safe := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, lower)
	return safe
}
