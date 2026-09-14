package agents

import (
	"strings"
	"testing"
)

func TestClarificationEngine_SubjectClarification_She(t *testing.T) {
	engine := NewClarificationEngine()

	fact := ExtractedFact{
		ID:       "fact_1",
		Value:    "organized",
		Evidence: "She is organized",
	}

	q := engine.GenerateSubjectClarification(fact, "unknown_female")

	if q.Type != "subject_clarification" {
		t.Errorf("Expected type subject_clarification, got %s", q.Type)
	}
	if !strings.Contains(q.Question, "Who is she") {
		t.Errorf("Expected question to ask 'Who is she', got: %s", q.Question)
	}
	if q.Priority != 1 {
		t.Errorf("Expected priority 1 (critical), got %d", q.Priority)
	}
	if len(q.Options) == 0 {
		t.Error("Expected multiple choice options")
	}
	if !containsAny(q.Options[0], "boss", "manager", "colleague") {
		t.Errorf("Expected relationship options, got: %v", q.Options)
	}
}

func TestClarificationEngine_SubjectClarification_He(t *testing.T) {
	engine := NewClarificationEngine()

	fact := ExtractedFact{
		ID:       "fact_1",
		Value:    "detail-oriented",
		Evidence: "He is detail-oriented",
	}

	q := engine.GenerateSubjectClarification(fact, "unknown_male")

	if !strings.Contains(strings.ToLower(q.Question), "who is he") {
		t.Errorf("Expected question to ask 'Who is he', got: %s", q.Question)
	}
	if !containsAny(q.Context, "detail-oriented") {
		t.Errorf("Expected context to include evidence, got: %s", q.Context)
	}
}

func TestClarificationEngine_SubjectClarification_They(t *testing.T) {
	engine := NewClarificationEngine()

	fact := ExtractedFact{
		ID:       "fact_1",
		Value:    "collaborative",
		Evidence: "They work collaboratively",
	}

	q := engine.GenerateSubjectClarification(fact, "unknown_group")

	if !strings.Contains(strings.ToLower(q.Question), "who are they") {
		t.Errorf("Expected question to ask 'Who are they', got: %s", q.Question)
	}
	if !containsAny(q.Options[0], "team", "friends", "family") {
		t.Errorf("Expected group options, got: %v", q.Options)
	}
}

func TestClarificationEngine_SubjectClarification_Name(t *testing.T) {
	engine := NewClarificationEngine()

	fact := ExtractedFact{
		ID:       "fact_1",
		Value:    "organized",
		Evidence: "Sarah is organized",
	}

	q := engine.GenerateSubjectClarification(fact, "contact_pending_sarah")

	// Question should ask about relationship (name might be lowercased in extraction)
	if !strings.Contains(strings.ToLower(q.Question), "sarah") || !strings.Contains(strings.ToLower(q.Question), "relationship") {
		t.Errorf("Expected question to ask about sarah's relationship, got: %s", q.Question)
	}
	if !strings.Contains(q.Context, "Sarah") {
		t.Errorf("Expected context to include original evidence with proper casing, got: %s", q.Context)
	}
}

func TestClarificationEngine_ContactConfirmation(t *testing.T) {
	engine := NewClarificationEngine()

	traits := []string{"detail-oriented", "deadline-focused", "formal communication"}
	q := engine.GenerateContactConfirmation("John", "manager", traits)

	if q.Type != "contact_confirmation" {
		t.Errorf("Expected type contact_confirmation, got %s", q.Type)
	}
	if q.Priority != 2 {
		t.Errorf("Expected priority 2, got %d", q.Priority)
	}
	if !strings.Contains(q.Question, "John") {
		t.Errorf("Expected question to include contact name, got: %s", q.Question)
	}
	if !strings.Contains(q.Question, "manager") {
		t.Errorf("Expected question to include relationship, got: %s", q.Question)
	}
	if !strings.Contains(q.Context, "detail-oriented") {
		t.Errorf("Expected context to include discovered traits, got: %s", q.Context)
	}
	if len(q.Options) == 0 {
		t.Error("Expected confirmation options")
	}
}

func TestClarificationEngine_ContactConfirmation_NoTraits(t *testing.T) {
	engine := NewClarificationEngine()

	q := engine.GenerateContactConfirmation("Sarah", "colleague", []string{})

	if !strings.Contains(q.Question, "Sarah") {
		t.Error("Expected question to include name")
	}
	// Should still generate valid question even without traits
	if q.Question == "" {
		t.Error("Expected non-empty question")
	}
}

func TestClarificationEngine_ConflictClarification(t *testing.T) {
	engine := NewClarificationEngine()

	q := engine.GenerateConflictClarification("user", "casual", "formal")

	if q.Type != "conflict_clarification" {
		t.Errorf("Expected type conflict_clarification, got %s", q.Type)
	}
	if q.Priority != 1 {
		t.Errorf("Expected priority 1 (critical), got %d", q.Priority)
	}
	if !strings.Contains(q.Question, "casual") {
		t.Errorf("Expected question to reference previous value, got: %s", q.Question)
	}
	if !strings.Contains(q.Question, "formal") {
		t.Errorf("Expected question to reference new value, got: %s", q.Question)
	}
	if len(q.Options) < 3 {
		t.Errorf("Expected at least 3 options, got %d", len(q.Options))
	}
}

func TestClarificationEngine_ContextClarification(t *testing.T) {
	engine := NewClarificationEngine()

	q := engine.GenerateContextClarification("boss", "but")

	if q.Type != "context_clarification" {
		t.Errorf("Expected type context_clarification, got %s", q.Type)
	}
	if !strings.Contains(strings.ToLower(q.Question), "boss") {
		t.Errorf("Expected question to mention previous subject, got: %s", q.Question)
	}
	if !strings.Contains(q.Context, "but") {
		t.Errorf("Expected context to mention trigger word, got: %s", q.Context)
	}
	if len(q.Options) == 0 {
		t.Error("Expected clarification options")
	}
}

func TestClarificationEngine_QuestionUniqueness(t *testing.T) {
	engine := NewClarificationEngine()

	fact1 := ExtractedFact{ID: "fact_1", Value: "organized", Evidence: "She is organized"}
	fact2 := ExtractedFact{ID: "fact_2", Value: "formal", Evidence: "He is formal"}

	q1 := engine.GenerateSubjectClarification(fact1, "unknown_female")
	q2 := engine.GenerateSubjectClarification(fact2, "unknown_male")

	if q1.ID == q2.ID {
		t.Errorf("Expected unique question IDs, but both are: %s", q1.ID)
	}
}

func TestClarificationEngine_QuestionStatus(t *testing.T) {
	engine := NewClarificationEngine()

	fact := ExtractedFact{ID: "fact_1", Value: "test", Evidence: "test"}
	q := engine.GenerateSubjectClarification(fact, "unknown_female")

	if q.Status != "pending" {
		t.Errorf("Expected status 'pending', got %s", q.Status)
	}
}

func TestClarificationEngine_ProcessResponse_SelectedOption(t *testing.T) {
	engine := NewClarificationEngine()

	response := ClarificationResponse{
		QuestionID:     "q_1",
		SelectedOption: "Boss / Manager",
		Timestamp:      1234567890,
	}

	result, err := engine.ProcessResponse(response)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if result != "Boss / Manager" {
		t.Errorf("Expected result to be selected option, got: %s", result)
	}
}

func TestClarificationEngine_ProcessResponse_TextResponse(t *testing.T) {
	engine := NewClarificationEngine()

	response := ClarificationResponse{
		QuestionID:   "q_1",
		UserResponse: "My colleague at work",
		Timestamp:    1234567890,
	}

	result, err := engine.ProcessResponse(response)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if result != "My colleague at work" {
		t.Errorf("Expected result to be text response, got: %s", result)
	}
}

func TestClarificationEngine_ProcessResponse_NoResponse(t *testing.T) {
	engine := NewClarificationEngine()

	response := ClarificationResponse{
		QuestionID: "q_1",
		Timestamp:  1234567890,
	}

	result, err := engine.ProcessResponse(response)

	if err == nil {
		t.Error("Expected error for no response")
	}
	if result != "" {
		t.Errorf("Expected empty result on error, got: %s", result)
	}
}

func TestClarificationEngine_ExtractClarifiedSubject_BossOption(t *testing.T) {
	engine := NewClarificationEngine()

	testCases := []string{
		"Boss / Manager",
		"boss",
		"Manager",
		"Director",
	}

	for _, tc := range testCases {
		result := engine.ExtractClarifiedSubject(tc)
		if result != "contact_boss" {
			t.Errorf("Input '%s': expected contact_boss, got %s", tc, result)
		}
	}
}

func TestClarificationEngine_ExtractClarifiedSubject_ColleagueOption(t *testing.T) {
	engine := NewClarificationEngine()

	testCases := []string{
		"Colleague / Coworker",
		"colleague",
		"Coworker",
		"Teammate",
	}

	for _, tc := range testCases {
		result := engine.ExtractClarifiedSubject(tc)
		if result != "contact_colleague" {
			t.Errorf("Input '%s': expected contact_colleague, got %s", tc, result)
		}
	}
}

func TestClarificationEngine_ExtractClarifiedSubject_FriendOption(t *testing.T) {
	engine := NewClarificationEngine()

	result := engine.ExtractClarifiedSubject("Friend")
	if result != "contact_friend" {
		t.Errorf("Expected contact_friend, got %s", result)
	}
}

func TestClarificationEngine_ExtractClarifiedSubject_PartnerOption(t *testing.T) {
	engine := NewClarificationEngine()

	testCases := []string{
		"Partner",
		"Spouse",
		"Wife",
		"Husband",
	}

	for _, tc := range testCases {
		result := engine.ExtractClarifiedSubject(tc)
		if result != "contact_partner" {
			t.Errorf("Input '%s': expected contact_partner, got %s", tc, result)
		}
	}
}

func TestClarificationEngine_ExtractClarifiedSubject_FamilyOption(t *testing.T) {
	engine := NewClarificationEngine()

	testCases := []string{
		"Family member",
		"Parent",
		"Mother",
		"Father",
		"Sibling",
	}

	for _, tc := range testCases {
		result := engine.ExtractClarifiedSubject(tc)
		if result != "contact_family" {
			t.Errorf("Input '%s': expected contact_family, got %s", tc, result)
		}
	}
}

func TestClarificationEngine_ExtractClarifiedSubject_Name(t *testing.T) {
	engine := NewClarificationEngine()

	testCases := []struct {
		input  string
		expect string
	}{
		{"Sarah", "contact_pending_sarah"},
		{"John Smith", "contact_pending_john"},
		{"Dr. Patterson", "contact_pending_dr"},
	}

	for _, tc := range testCases {
		result := engine.ExtractClarifiedSubject(tc.input)
		if !strings.HasPrefix(result, "contact_pending_") {
			t.Errorf("Input '%s': expected contact_pending_*, got %s", tc.input, result)
		}
	}
}

func TestClarificationEngine_LinkedFacts(t *testing.T) {
	engine := NewClarificationEngine()

	fact := ExtractedFact{
		ID:       "fact_123",
		Value:    "organized",
		Evidence: "She is organized",
	}

	q := engine.GenerateSubjectClarification(fact, "unknown_female")

	if len(q.LinkedFacts) == 0 {
		t.Error("Expected linked facts")
	}
	if q.LinkedFacts[0] != "fact_123" {
		t.Errorf("Expected fact ID in linked facts, got: %v", q.LinkedFacts)
	}
}

func TestClarificationEngine_Timestamps(t *testing.T) {
	engine := NewClarificationEngine()

	fact := ExtractedFact{ID: "fact_1", Value: "test", Evidence: "test"}
	q := engine.GenerateSubjectClarification(fact, "unknown_female")

	if q.CreatedAt == 0 {
		t.Error("Expected non-zero timestamp")
	}
}

func TestClarificationEngine_ContextInclusion(t *testing.T) {
	engine := NewClarificationEngine()

	fact := ExtractedFact{
		ID:       "fact_1",
		Value:    "organized",
		Evidence: "She handles everything with precision",
	}

	q := engine.GenerateSubjectClarification(fact, "unknown_female")

	// Context should include the evidence
	if !strings.Contains(q.Context, "precision") {
		t.Errorf("Expected context to include evidence, got: %s", q.Context)
	}
}

func TestClarificationEngine_CaseInsensitive(t *testing.T) {
	engine := NewClarificationEngine()

	testCases := []struct {
		input  string
		expect string
	}{
		{"BOSS", "contact_boss"},
		{"Boss / Manager", "contact_boss"},
		{"COLLEAGUE", "contact_colleague"},
		{"Friend", "contact_friend"},
	}

	for _, tc := range testCases {
		result := engine.ExtractClarifiedSubject(tc.input)
		if result != tc.expect {
			t.Errorf("Input '%s': expected %s, got %s", tc.input, tc.expect, result)
		}
	}
}
