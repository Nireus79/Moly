package agents

import (
	"testing"
)

func TestSubjectAnalyzer_ExplicitUser(t *testing.T) {
	// Test clear identification of user as subject
	analyzer := NewSubjectAnalyzer()

	testCases := []string{"user", "I", "me", "my", "myself", "we", "us", "our"}

	for _, subject := range testCases {
		fact := ExtractedFact{
			Value:           "casual",
			FactType:        "style",
			ProposedSubject: subject,
			Evidence:        "I'm casual",
		}

		analysis := analyzer.Analyze(fact)

		if analysis.ClarityLevel != "clear" {
			t.Errorf("Subject '%s': expected clarity 'clear', got '%s'", subject, analysis.ClarityLevel)
		}
		if analysis.DetectedSubject != "user" {
			t.Errorf("Subject '%s': expected 'user', got '%s'", subject, analysis.DetectedSubject)
		}
		if analysis.RequiredQuestion != "" {
			t.Errorf("Subject '%s': expected no question, got '%s'", subject, analysis.RequiredQuestion)
		}
	}
}

func TestSubjectAnalyzer_ExplicitBoss(t *testing.T) {
	// Test clear identification of boss role
	analyzer := NewSubjectAnalyzer()

	testCases := []string{"boss", "manager", "director", "lead", "supervisor"}

	for _, subject := range testCases {
		fact := ExtractedFact{
			Value:           "formal",
			FactType:        "style",
			ProposedSubject: subject,
			Evidence:        "My boss is formal",
		}

		analysis := analyzer.Analyze(fact)

		if analysis.ClarityLevel != "clear" {
			t.Errorf("Subject '%s': expected clarity 'clear', got '%s'", subject, analysis.ClarityLevel)
		}
		if analysis.DetectedSubject != "contact_boss" {
			t.Errorf("Subject '%s': expected 'contact_boss', got '%s'", subject, analysis.DetectedSubject)
		}
	}
}

func TestSubjectAnalyzer_ExplicitColleague(t *testing.T) {
	// Test clear identification of colleague role
	analyzer := NewSubjectAnalyzer()

	testCases := []string{"colleague", "coworker", "teammate", "peer"}

	for _, subject := range testCases {
		fact := ExtractedFact{
			Value:           "collaborative",
			FactType:        "style",
			ProposedSubject: subject,
			Evidence:        "My colleague is collaborative",
		}

		analysis := analyzer.Analyze(fact)

		if analysis.ClarityLevel != "clear" {
			t.Errorf("Subject '%s': expected clarity 'clear', got '%s'", subject, analysis.ClarityLevel)
		}
		if analysis.DetectedSubject != "contact_colleague" {
			t.Errorf("Subject '%s': expected 'contact_colleague', got '%s'", subject, analysis.DetectedSubject)
		}
	}
}

func TestSubjectAnalyzer_ExplicitFriend(t *testing.T) {
	// Test clear identification of friend role
	analyzer := NewSubjectAnalyzer()

	testCases := []string{"friend", "buddy", "mate"}

	for _, subject := range testCases {
		fact := ExtractedFact{
			Value:           "outgoing",
			FactType:        "trait",
			ProposedSubject: subject,
			Evidence:        "My friend is outgoing",
		}

		analysis := analyzer.Analyze(fact)

		if analysis.ClarityLevel != "clear" {
			t.Errorf("Subject '%s': expected clarity 'clear', got '%s'", subject, analysis.ClarityLevel)
		}
		if analysis.DetectedSubject != "contact_friend" {
			t.Errorf("Subject '%s': expected 'contact_friend', got '%s'", subject, analysis.DetectedSubject)
		}
	}
}

func TestSubjectAnalyzer_ExplicitPartner(t *testing.T) {
	// Test clear identification of partner role
	analyzer := NewSubjectAnalyzer()

	testCases := []string{"partner", "spouse", "wife", "husband", "girlfriend", "boyfriend"}

	for _, subject := range testCases {
		fact := ExtractedFact{
			Value:           "understanding",
			FactType:        "trait",
			ProposedSubject: subject,
			Evidence:        "My wife is understanding",
		}

		analysis := analyzer.Analyze(fact)

		if analysis.ClarityLevel != "clear" {
			t.Errorf("Subject '%s': expected clarity 'clear', got '%s'", subject, analysis.ClarityLevel)
		}
		if analysis.DetectedSubject != "contact_partner" {
			t.Errorf("Subject '%s': expected 'contact_partner', got '%s'", subject, analysis.DetectedSubject)
		}
	}
}

func TestSubjectAnalyzer_ExplicitFamily(t *testing.T) {
	// Test clear identification of family role
	analyzer := NewSubjectAnalyzer()

	testCases := []string{"family", "parent", "mother", "father", "sister", "brother"}

	for _, subject := range testCases {
		fact := ExtractedFact{
			Value:           "supportive",
			FactType:        "trait",
			ProposedSubject: subject,
			Evidence:        "My mother is supportive",
		}

		analysis := analyzer.Analyze(fact)

		if analysis.ClarityLevel != "clear" {
			t.Errorf("Subject '%s': expected clarity 'clear', got '%s'", subject, analysis.ClarityLevel)
		}
		if analysis.DetectedSubject != "contact_family" {
			t.Errorf("Subject '%s': expected 'contact_family', got '%s'", subject, analysis.DetectedSubject)
		}
	}
}

func TestSubjectAnalyzer_AmbiguousShe(t *testing.T) {
	// Test ambiguous "she" pronoun
	analyzer := NewSubjectAnalyzer()

	fact := ExtractedFact{
		Value:           "organized",
		FactType:        "trait",
		ProposedSubject: "she",
		Evidence:        "She is organized",
	}

	analysis := analyzer.Analyze(fact)

	if analysis.ClarityLevel != "ambiguous" {
		t.Errorf("Expected ambiguous, got %s", analysis.ClarityLevel)
	}
	if analysis.DetectedSubject != "unknown_female" {
		t.Errorf("Expected unknown_female, got %s", analysis.DetectedSubject)
	}
	if analysis.RequiredQuestion == "" {
		t.Error("Expected clarification question for ambiguous pronoun")
	}
	if !containsAny(analysis.RequiredQuestion, "who", "she") {
		t.Errorf("Question should ask about 'she', got: %s", analysis.RequiredQuestion)
	}
}

func TestSubjectAnalyzer_AmbiguousHe(t *testing.T) {
	// Test ambiguous "he" pronoun
	analyzer := NewSubjectAnalyzer()

	fact := ExtractedFact{
		Value:           "formal",
		FactType:        "style",
		ProposedSubject: "he",
		Evidence:        "He communicates formally",
	}

	analysis := analyzer.Analyze(fact)

	if analysis.ClarityLevel != "ambiguous" {
		t.Errorf("Expected ambiguous, got %s", analysis.ClarityLevel)
	}
	if analysis.DetectedSubject != "unknown_male" {
		t.Errorf("Expected unknown_male, got %s", analysis.DetectedSubject)
	}
	if analysis.RequiredQuestion == "" {
		t.Error("Expected clarification question for ambiguous pronoun")
	}
}

func TestSubjectAnalyzer_AmbiguousThey(t *testing.T) {
	// Test ambiguous "they" pronoun
	analyzer := NewSubjectAnalyzer()

	fact := ExtractedFact{
		Value:           "collaborative",
		FactType:        "style",
		ProposedSubject: "they",
		Evidence:        "They work collaboratively",
	}

	analysis := analyzer.Analyze(fact)

	if analysis.ClarityLevel != "ambiguous" {
		t.Errorf("Expected ambiguous, got %s", analysis.ClarityLevel)
	}
	if analysis.DetectedSubject != "unknown_group" {
		t.Errorf("Expected unknown_group, got %s", analysis.DetectedSubject)
	}
	if analysis.RequiredQuestion == "" {
		t.Error("Expected clarification question for ambiguous pronoun")
	}
}

func TestSubjectAnalyzer_Name(t *testing.T) {
	// Test proper name detection
	analyzer := NewSubjectAnalyzer()

	fact := ExtractedFact{
		Value:           "organized",
		FactType:        "trait",
		ProposedSubject: "Sarah",
		Evidence:        "Sarah is organized",
	}

	analysis := analyzer.Analyze(fact)

	if analysis.ClarityLevel != "ambiguous" {
		t.Errorf("Expected ambiguous for name, got %s", analysis.ClarityLevel)
	}
	if !containsAny(analysis.DetectedSubject, "sarah") {
		t.Errorf("Expected contact_pending_sarah or similar, got %s", analysis.DetectedSubject)
	}
	if analysis.RequiredQuestion == "" {
		t.Error("Expected clarification question for name without context")
	}
	if !containsAny(analysis.RequiredQuestion, "relationship") {
		t.Errorf("Question should ask about relationship, got: %s", analysis.RequiredQuestion)
	}
}

func TestSubjectAnalyzer_Missing(t *testing.T) {
	// Test missing/unrecognizable subject
	analyzer := NewSubjectAnalyzer()

	fact := ExtractedFact{
		Value:           "organized",
		FactType:        "trait",
		ProposedSubject: "xyz123",
		Evidence:        "xyz123 is organized",
	}

	analysis := analyzer.Analyze(fact)

	if analysis.ClarityLevel != "missing" {
		t.Errorf("Expected missing, got %s", analysis.ClarityLevel)
	}
	if analysis.DetectedSubject != "unknown" {
		t.Errorf("Expected unknown, got %s", analysis.DetectedSubject)
	}
}

func TestSubjectAnalyzer_CaseInsensitive(t *testing.T) {
	// Test that analysis is case-insensitive
	analyzer := NewSubjectAnalyzer()

	testCases := []struct {
		subject string
		expect  string
	}{
		{"BOSS", "contact_boss"},
		{"Boss", "contact_boss"},
		{"COLLEAGUE", "contact_colleague"},
		{"Friend", "contact_friend"},
		{"SHE", "unknown_female"},
		{"She", "unknown_female"},
	}

	for _, tc := range testCases {
		fact := ExtractedFact{
			Value:           "test",
			FactType:        "trait",
			ProposedSubject: tc.subject,
			Evidence:        "test",
		}

		analysis := analyzer.Analyze(fact)

		if analysis.DetectedSubject != tc.expect {
			t.Errorf("Subject '%s': expected '%s', got '%s'", tc.subject, tc.expect, analysis.DetectedSubject)
		}
	}
}

func TestSubjectAnalyzer_PreservesFactData(t *testing.T) {
	// Test that analysis preserves original fact data
	analyzer := NewSubjectAnalyzer()

	originalFact := ExtractedFact{
		Value:           "organized",
		FactType:        "trait",
		ProposedSubject: "she",
		Evidence:        "She is organized",
		Confidence:      0.95,
		CreatedAt:       12345,
	}

	analysis := analyzer.Analyze(originalFact)

	if analysis.Fact.Value != originalFact.Value {
		t.Error("Fact value changed")
	}
	if analysis.Fact.FactType != originalFact.FactType {
		t.Error("Fact type changed")
	}
	if analysis.Fact.Confidence != originalFact.Confidence {
		t.Error("Fact confidence changed")
	}
	if analysis.Fact.Evidence != originalFact.Evidence {
		t.Error("Fact evidence changed")
	}
}

func TestSubjectAnalyzer_AllFactTypes(t *testing.T) {
	// Test that analyzer works with all fact types
	analyzer := NewSubjectAnalyzer()

	factTypes := []string{"trait", "style", "value", "preference", "goal"}

	for _, ftype := range factTypes {
		fact := ExtractedFact{
			Value:           "test",
			FactType:        ftype,
			ProposedSubject: "boss",
			Evidence:        "test",
		}

		analysis := analyzer.Analyze(fact)

		if analysis.ClarityLevel != "clear" {
			t.Errorf("Fact type '%s': expected clear, got %s", ftype, analysis.ClarityLevel)
		}
	}
}

func TestSubjectAnalyzer_Pronouns_AllVariants(t *testing.T) {
	// Test all pronoun variants
	analyzer := NewSubjectAnalyzer()

	testCases := []struct {
		subject   string
		expected  string
		clearness string
	}{
		{"she", "unknown_female", "ambiguous"},
		{"he", "unknown_male", "ambiguous"},
		{"they", "unknown_group", "ambiguous"},
		{"them", "unknown_group", "ambiguous"},
		{"it", "unknown_thing", "ambiguous"},
		{"that", "unknown_thing", "ambiguous"},
	}

	for _, tc := range testCases {
		fact := ExtractedFact{
			Value:           "test",
			FactType:        "trait",
			ProposedSubject: tc.subject,
			Evidence:        "test",
		}

		analysis := analyzer.Analyze(fact)

		if analysis.ClarityLevel != tc.clearness {
			t.Errorf("Subject '%s': expected clarity '%s', got '%s'", tc.subject, tc.clearness, analysis.ClarityLevel)
		}
		if analysis.DetectedSubject != tc.expected {
			t.Errorf("Subject '%s': expected '%s', got '%s'", tc.subject, tc.expected, analysis.DetectedSubject)
		}
	}
}

func TestSubjectAnalyzer_NameSanitization(t *testing.T) {
	// Test that names are properly sanitized in detected subject
	analyzer := NewSubjectAnalyzer()

	testCases := []struct {
		name     string
		contains string
	}{
		{"John", "john"},
		{"Sarah Smith", "sarah"},
		{"Dr. Smith", "dr"},
	}

	for _, tc := range testCases {
		fact := ExtractedFact{
			Value:           "test",
			FactType:        "trait",
			ProposedSubject: tc.name,
			Evidence:        "test",
		}

		analysis := analyzer.Analyze(fact)

		if !contains(analysis.DetectedSubject, tc.contains) {
			t.Errorf("Name '%s': expected '%s' in subject, got '%s'", tc.name, tc.contains, analysis.DetectedSubject)
		}
	}
}
