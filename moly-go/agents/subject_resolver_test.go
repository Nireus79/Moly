package agents

import (
	"strings"
	"testing"
)

func TestSubjectResolver_ResolvePronoun_She_Boss(t *testing.T) {
	resolver := NewSubjectResolver()

	result, err := resolver.ResolveAmbiguousPronoun("unknown_female", "Boss")

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if !strings.Contains(result, "boss") {
		t.Errorf("Expected result to contain 'boss', got: %s", result)
	}
}

func TestSubjectResolver_ResolvePronoun_He_Manager(t *testing.T) {
	resolver := NewSubjectResolver()

	result, err := resolver.ResolveAmbiguousPronoun("unknown_male", "Manager")

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if !strings.Contains(result, "boss") {
		t.Errorf("Expected manager to resolve to boss, got: %s", result)
	}
}

func TestSubjectResolver_ResolvePronoun_They_Colleagues(t *testing.T) {
	resolver := NewSubjectResolver()

	result, err := resolver.ResolveAmbiguousPronoun("unknown_group", "Colleague")

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if !strings.Contains(result, "colleague") {
		t.Errorf("Expected result to contain 'colleague', got: %s", result)
	}
}

func TestSubjectResolver_ResolveSubject_RoleOnly(t *testing.T) {
	resolver := NewSubjectResolver()

	result, err := resolver.ResolveSubject("unknown_female", "Boss")

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if result != "contact_boss" {
		t.Errorf("Expected contact_boss, got: %s", result)
	}
}

func TestSubjectResolver_ResolveSubject_RoleAndName(t *testing.T) {
	resolver := NewSubjectResolver()

	result, err := resolver.ResolveSubject("unknown_female", "My manager Sarah")

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if !strings.Contains(result, "boss") || !strings.Contains(result, "sarah") {
		t.Errorf("Expected contact_boss_sarah or similar, got: %s", result)
	}
}

func TestSubjectResolver_ResolveSubject_JustName(t *testing.T) {
	resolver := NewSubjectResolver()

	result, err := resolver.ResolveSubject("unknown_female", "Sarah")

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if !strings.Contains(result, "pending") {
		t.Errorf("Expected pending contact, got: %s", result)
	}
}

func TestSubjectResolver_ResolvePendingName_BossRole(t *testing.T) {
	resolver := NewSubjectResolver()

	result, err := resolver.ResolvePendingName("contact_pending_sarah", "Boss")

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if result != "contact_boss_sarah" {
		t.Errorf("Expected contact_boss_sarah, got: %s", result)
	}
}

func TestSubjectResolver_ResolvePendingName_ColleagueRole(t *testing.T) {
	resolver := NewSubjectResolver()

	result, err := resolver.ResolvePendingName("contact_pending_john", "Colleague")

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if result != "contact_colleague_john" {
		t.Errorf("Expected contact_colleague_john, got: %s", result)
	}
}

func TestSubjectResolver_ResolvePendingName_FriendRole(t *testing.T) {
	resolver := NewSubjectResolver()

	result, err := resolver.ResolvePendingName("contact_pending_mike", "Friend")

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if result != "contact_friend_mike" {
		t.Errorf("Expected contact_friend_mike, got: %s", result)
	}
}

func TestSubjectResolver_ResolvePendingName_InvalidSubject(t *testing.T) {
	resolver := NewSubjectResolver()

	_, err := resolver.ResolvePendingName("contact_boss_sarah", "Friend")

	if err == nil {
		t.Error("Expected error for non-pending subject")
	}
}

func TestSubjectResolver_ResolveSubject_Relationship_Boss(t *testing.T) {
	resolver := NewSubjectResolver()

	testCases := []string{
		"Boss",
		"Manager",
		"Director",
		"Supervisor",
	}

	for _, tc := range testCases {
		result, _ := resolver.ResolveSubject("unknown_female", tc)
		if !strings.Contains(result, "boss") {
			t.Errorf("Input '%s': expected contact_boss, got %s", tc, result)
		}
	}
}

func TestSubjectResolver_ResolveSubject_Relationship_Colleague(t *testing.T) {
	resolver := NewSubjectResolver()

	testCases := []string{
		"Colleague",
		"Coworker",
		"Teammate",
	}

	for _, tc := range testCases {
		result, _ := resolver.ResolveSubject("unknown_male", tc)
		if !strings.Contains(result, "colleague") {
			t.Errorf("Input '%s': expected contact_colleague, got %s", tc, result)
		}
	}
}

func TestSubjectResolver_ResolveSubject_Relationship_Friend(t *testing.T) {
	resolver := NewSubjectResolver()

	result, _ := resolver.ResolveSubject("unknown_female", "Friend")

	if !strings.Contains(result, "friend") {
		t.Errorf("Expected contact_friend, got %s", result)
	}
}

func TestSubjectResolver_ResolveSubject_Relationship_Partner(t *testing.T) {
	resolver := NewSubjectResolver()

	testCases := []string{
		"Partner",
		"Spouse",
		"Wife",
		"Husband",
	}

	for _, tc := range testCases {
		result, _ := resolver.ResolveSubject("unknown_male", tc)
		if !strings.Contains(result, "partner") {
			t.Errorf("Input '%s': expected contact_partner, got %s", tc, result)
		}
	}
}

func TestSubjectResolver_ResolveSubject_Relationship_Family(t *testing.T) {
	resolver := NewSubjectResolver()

	testCases := []string{
		"Family",
		"Parent",
		"Mother",
		"Father",
		"Sibling",
	}

	for _, tc := range testCases {
		result, _ := resolver.ResolveSubject("unknown_female", tc)
		if !strings.Contains(result, "family") {
			t.Errorf("Input '%s': expected contact_family, got %s", tc, result)
		}
	}
}

func TestSubjectResolver_GetDisplayName_Boss(t *testing.T) {
	resolver := NewSubjectResolver()

	result := resolver.GetContactDisplayName("contact_boss")

	if !strings.Contains(strings.ToLower(result), "boss") {
		t.Errorf("Expected display name to contain 'boss', got: %s", result)
	}
}

func TestSubjectResolver_GetDisplayName_BossWithName(t *testing.T) {
	resolver := NewSubjectResolver()

	result := resolver.GetContactDisplayName("contact_boss_sarah")

	// Name is stored in lowercase, so check for that
	if !strings.Contains(strings.ToLower(result), "sarah") {
		t.Errorf("Expected name in display, got: %s", result)
	}
	if !strings.Contains(strings.ToLower(result), "boss") {
		t.Errorf("Expected relationship in display, got: %s", result)
	}
}

func TestSubjectResolver_GetDisplayName_ColleagueWithName(t *testing.T) {
	resolver := NewSubjectResolver()

	result := resolver.GetContactDisplayName("contact_colleague_john")

	// Name is stored in lowercase
	if !strings.Contains(strings.ToLower(result), "john") {
		t.Errorf("Expected name john in display, got: %s", result)
	}
	if !strings.Contains(strings.ToLower(result), "colleague") {
		t.Errorf("Expected relationship in display, got: %s", result)
	}
}

func TestSubjectResolver_GetDisplayName_NonContact(t *testing.T) {
	resolver := NewSubjectResolver()

	result := resolver.GetContactDisplayName("user")

	if result != "user" {
		t.Errorf("Expected non-contact subject unchanged, got: %s", result)
	}
}

func TestSubjectResolver_CaseInsensitive(t *testing.T) {
	resolver := NewSubjectResolver()

	testCases := []struct {
		input  string
		expect string
	}{
		{"BOSS", "boss"},
		{"Boss / Manager", "boss"},
		{"COLLEAGUE", "colleague"},
		{"Friend", "friend"},
	}

	for _, tc := range testCases {
		result, _ := resolver.ResolveSubject("unknown_female", tc.input)
		if !strings.Contains(result, tc.expect) {
			t.Errorf("Input '%s': expected %s in result, got %s", tc.input, tc.expect, result)
		}
	}
}

func TestSubjectResolver_NameExtraction_Simple(t *testing.T) {
	resolver := NewSubjectResolver()

	testCases := []struct {
		input  string
		expect string // Expected substring in result
	}{
		{"Sarah", "sarah"},
		{"My boss John", "john"},
		{"John Smith", "john"},
	}

	for _, tc := range testCases {
		result, _ := resolver.ResolveSubject("unknown_female", tc.input)
		if !strings.Contains(strings.ToLower(result), tc.expect) {
			t.Errorf("Input '%s': expected to extract %s, got %s", tc.input, tc.expect, result)
		}
	}
}

func TestSubjectResolver_EmptyResponse(t *testing.T) {
	resolver := NewSubjectResolver()

	_, err := resolver.ResolveSubject("unknown_female", "")

	if err == nil {
		t.Error("Expected error for empty clarification")
	}
}

func TestSubjectResolver_MultipleNames(t *testing.T) {
	resolver := NewSubjectResolver()

	// When there's a compound name, should extract the main one
	result, _ := resolver.ResolveSubject("unknown_female", "John Smith is my boss")

	if !strings.Contains(result, "john") {
		t.Errorf("Expected to find john in result, got: %s", result)
	}
}

func TestSubjectResolver_SkipsCommonWords(t *testing.T) {
	resolver := NewSubjectResolver()

	// Should skip common starting words like "My", "The", etc.
	result, _ := resolver.ResolveSubject("unknown_female", "My friend Sarah")

	if !strings.Contains(result, "sarah") {
		t.Errorf("Expected to find sarah, got: %s", result)
	}
}

func TestSubjectResolver_PendingWithUnknownRole(t *testing.T) {
	resolver := NewSubjectResolver()

	result, err := resolver.ResolvePendingName("contact_pending_alice", "Random Role")

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	// Should stay pending if role not recognized
	if !strings.Contains(result, "pending") {
		// Or it might extract the name and resolve with a default
		if !strings.Contains(result, "alice") {
			t.Errorf("Expected alice in result, got: %s", result)
		}
	}
}

func TestSubjectResolver_MultiWordName(t *testing.T) {
	resolver := NewSubjectResolver()

	result, _ := resolver.ResolvePendingName("contact_pending_sarah_smith", "Boss")

	// Should handle multi-word names in pending
	if !strings.Contains(result, "boss") {
		t.Errorf("Expected boss in resolved subject, got: %s", result)
	}
}
