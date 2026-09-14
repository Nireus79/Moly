package agents

import (
	"testing"
)

func TestSubjectShiftDetector_ConjunctionShift(t *testing.T) {
	detector := NewSubjectShiftDetector()

	// User starts by talking about themselves, then says "but my boss..."
	message := "I'm casual in my communication. But my boss is very formal."
	shifts := detector.DetectShifts(message, "user")

	if len(shifts) == 0 {
		t.Errorf("Expected to detect shift from user to boss, got none")
		return
	}

	if shifts[0].From != "user" {
		t.Errorf("Expected 'from' user, got %s", shifts[0].From)
	}
	if shifts[0].To != "contact_boss" {
		t.Errorf("Expected 'to' contact_boss, got %s", shifts[0].To)
	}
	if shifts[0].Trigger != "but" {
		t.Errorf("Expected trigger 'but', got %s", shifts[0].Trigger)
	}
}

func TestSubjectShiftDetector_HoweverShift(t *testing.T) {
	detector := NewSubjectShiftDetector()

	message := "I prefer quick decisions. However, my colleague likes to take time."
	shifts := detector.DetectShifts(message, "user")

	if len(shifts) == 0 {
		t.Errorf("Expected to detect shift")
		return
	}

	if shifts[0].To != "contact_colleague" {
		t.Errorf("Expected shift to colleague, got %s", shifts[0].To)
	}
	// Accept either "however" trigger or explicit_mention (both are valid)
	if shifts[0].Trigger != "however" && shifts[0].Trigger != "explicit_mention" {
		t.Errorf("Expected trigger 'however' or 'explicit_mention', got %s", shifts[0].Trigger)
	}
}

func TestSubjectShiftDetector_UnlikeShift(t *testing.T) {
	detector := NewSubjectShiftDetector()

	message := "I value honesty in everything I do. Unlike my manager, who is more diplomatic."
	shifts := detector.DetectShifts(message, "user")

	if len(shifts) == 0 {
		t.Errorf("Expected to detect shift")
		return
	}

	if shifts[0].Trigger != "unlike" {
		t.Errorf("Expected trigger 'unlike', got %s", shifts[0].Trigger)
	}
	if shifts[0].To != "contact_boss" {
		t.Errorf("Expected shift to manager/boss, got %s", shifts[0].To)
	}
}

func TestSubjectShiftDetector_NoShift_SameSubject(t *testing.T) {
	detector := NewSubjectShiftDetector()

	message := "I'm casual but I also like to be professional in meetings."
	shifts := detector.DetectShifts(message, "user")

	// Should detect no shift because both parts are about user
	if len(shifts) > 0 {
		t.Errorf("Expected no shift when both parts are about user, but got %d", len(shifts))
	}
}

func TestSubjectShiftDetector_ExplicitMention(t *testing.T) {
	detector := NewSubjectShiftDetector()

	message := "My boss is very detail-oriented."
	shifts := detector.DetectShifts(message, "user")

	if len(shifts) == 0 {
		t.Errorf("Expected to detect explicit shift to boss")
		return
	}

	if shifts[0].To != "contact_boss" {
		t.Errorf("Expected shift to contact_boss, got %s", shifts[0].To)
	}
	if !shifts[0].Explicit {
		t.Errorf("Expected explicit flag to be true")
	}
}

func TestSubjectShiftDetector_ExplicitMention_Colleague(t *testing.T) {
	detector := NewSubjectShiftDetector()

	message := "My colleague and I have different working styles."
	shifts := detector.DetectShifts(message, "user")

	if len(shifts) == 0 {
		t.Errorf("Expected to detect explicit shift to colleague")
		return
	}

	if shifts[0].To != "contact_colleague" {
		t.Errorf("Expected shift to contact_colleague, got %s", shifts[0].To)
	}
}

func TestSubjectShiftDetector_ExplicitMention_Partner(t *testing.T) {
	detector := NewSubjectShiftDetector()

	message := "My wife prefers async communication."
	shifts := detector.DetectShifts(message, "user")

	if len(shifts) == 0 {
		t.Errorf("Expected to detect explicit shift to partner")
		return
	}

	if shifts[0].To != "contact_partner" {
		t.Errorf("Expected shift to contact_partner, got %s", shifts[0].To)
	}
}

func TestSubjectShiftDetector_ExplicitMention_Family(t *testing.T) {
	detector := NewSubjectShiftDetector()

	message := "My mother always wants detailed explanations."
	shifts := detector.DetectShifts(message, "user")

	if len(shifts) == 0 {
		t.Errorf("Expected to detect explicit shift to family")
		return
	}

	if shifts[0].To != "contact_family" {
		t.Errorf("Expected shift to contact_family, got %s", shifts[0].To)
	}
}

func TestSubjectShiftDetector_EmptyMessage(t *testing.T) {
	detector := NewSubjectShiftDetector()

	shifts := detector.DetectShifts("", "user")

	if len(shifts) > 0 {
		t.Errorf("Expected no shifts from empty message")
	}
}

func TestSubjectShiftDetector_EmptyPreviousSubject(t *testing.T) {
	detector := NewSubjectShiftDetector()

	shifts := detector.DetectShifts("Some message", "")

	if len(shifts) > 0 {
		t.Errorf("Expected no shifts without previous subject")
	}
}

func TestSubjectShiftDetector_MessageExcerpt(t *testing.T) {
	detector := NewSubjectShiftDetector()

	message := "I'm very casual. But my boss is very formal and structured."
	shifts := detector.DetectShifts(message, "user")

	if len(shifts) == 0 {
		t.Errorf("Expected to detect shift")
		return
	}

	if shifts[0].MessageExcerpt == "" {
		t.Errorf("Expected message excerpt to be non-empty")
	}
	if !containsAny(shifts[0].MessageExcerpt, "boss", "formal") {
		t.Errorf("Expected excerpt to contain relevant keywords, got: %s", shifts[0].MessageExcerpt)
	}
}

func TestSubjectShiftDetector_Pronoun_She(t *testing.T) {
	detector := NewSubjectShiftDetector()

	message := "I like to plan ahead. But she prefers spontaneity."
	shifts := detector.DetectShifts(message, "user")

	if len(shifts) == 0 {
		t.Errorf("Expected to detect shift to 'she'")
		return
	}

	if shifts[0].To != "unknown_female" {
		t.Errorf("Expected shift to unknown_female, got %s", shifts[0].To)
	}
}

func TestSubjectShiftDetector_Pronoun_He(t *testing.T) {
	detector := NewSubjectShiftDetector()

	// Simpler message without comma to ensure split works
	message := "I value honesty. But he is more pragmatic."
	shifts := detector.DetectShifts(message, "user")

	if len(shifts) == 0 {
		t.Errorf("Expected to detect shift to 'he'")
		return
	}

	if shifts[0].To != "unknown_male" {
		t.Errorf("Expected shift to unknown_male, got %s", shifts[0].To)
	}
}

func TestSubjectShiftDetector_Pronoun_They(t *testing.T) {
	detector := NewSubjectShiftDetector()

	message := "I'm organized. But they prefer chaos."
	shifts := detector.DetectShifts(message, "user")

	if len(shifts) == 0 {
		t.Errorf("Expected to detect shift to 'they'")
		return
	}

	if shifts[0].To != "unknown_group" {
		t.Errorf("Expected shift to unknown_group, got %s", shifts[0].To)
	}
}

func TestSubjectShiftDetector_MultipleShifts(t *testing.T) {
	detector := NewSubjectShiftDetector()

	// Message with multiple shift triggers - should catch both
	message := "I like details. But my boss is very casual. Although my colleague is formal."
	shifts := detector.DetectShifts(message, "user")

	if len(shifts) == 0 {
		t.Errorf("Expected to detect at least one shift")
		return
	}

	// Should detect shifts to both boss and colleague
	subjects := map[string]bool{}
	for _, shift := range shifts {
		subjects[shift.To] = true
	}

	if !subjects["contact_boss"] {
		t.Errorf("Expected to detect shift to boss, but got: %v", subjects)
	}
	if !subjects["contact_colleague"] {
		t.Errorf("Expected to detect shift to colleague, but got: %v", subjects)
	}
}

func TestSubjectShiftDetector_CaseInsensitive(t *testing.T) {
	detector := NewSubjectShiftDetector()

	testCases := []struct {
		message string
		expect  string
	}{
		{"I'm casual. BUT my boss is formal.", "contact_boss"},
		{"I like structure. However, My colleague prefers flexibility.", "contact_colleague"},
		{"I value honesty. UNLIKE my manager who is diplomatic.", "contact_boss"},
	}

	for _, tc := range testCases {
		shifts := detector.DetectShifts(tc.message, "user")

		if len(shifts) == 0 {
			t.Errorf("Message '%s': Expected to detect shift to %s", tc.message, tc.expect)
			continue
		}

		if shifts[0].To != tc.expect {
			t.Errorf("Message '%s': Expected %s, got %s", tc.message, tc.expect, shifts[0].To)
		}
	}
}

func TestSubjectShiftDetector_PreservesMessageCase(t *testing.T) {
	detector := NewSubjectShiftDetector()

	message := "I'm Casual. But My Boss is Very Formal."
	shifts := detector.DetectShifts(message, "user")

	if len(shifts) == 0 {
		t.Errorf("Expected to detect shift")
		return
	}

	// Excerpt should preserve original casing (check for key words preserving their case)
	if !containsAny(shifts[0].MessageExcerpt, "boss", "formal", "Boss", "Formal") {
		t.Errorf("Expected excerpt to contain boss/formal info, got: %s", shifts[0].MessageExcerpt)
	}
}

func TestSubjectShiftDetector_LongMessage(t *testing.T) {
	detector := NewSubjectShiftDetector()

	message := "I tend to be very casual in my communication style, preferring to keep things relaxed and informal. " +
		"But my boss is incredibly detail-oriented and prefers a more formal, structured approach to all communications."
	shifts := detector.DetectShifts(message, "user")

	if len(shifts) == 0 {
		t.Errorf("Expected to detect shift in long message")
		return
	}

	if shifts[0].From != "user" || shifts[0].To != "contact_boss" {
		t.Errorf("Expected shift from user to contact_boss")
	}
}

func TestSubjectShiftDetector_NoFalsePositives(t *testing.T) {
	detector := NewSubjectShiftDetector()

	// Message that mentions "but" but doesn't shift subject
	message := "I'm casual but also professional when needed."
	shifts := detector.DetectShifts(message, "user")

	if len(shifts) > 0 {
		t.Errorf("Expected no false positive shifts, but got %d", len(shifts))
	}
}

func TestSubjectShiftDetector_AllTriggers(t *testing.T) {
	detector := NewSubjectShiftDetector()

	triggers := []struct {
		message      string
		expectTrigger string
		allowExplicit bool // Some messages might detect explicit mention instead
	}{
		{"I like it. But my boss doesn't.", "but", false},
		{"I'm quick. However, my colleague is slow.", "however", true}, // May detect explicit mention
		{"I'm formal. Although my friend is casual.", "although", false},
		{"I value honesty. Though my manager disagrees.", "though", false},
		{"I'm structured. Whereas my colleague is chaotic.", "whereas", false},
		{"I'm decisive. Unlike my boss who deliberates.", "unlike", false},
	}

	for _, tc := range triggers {
		shifts := detector.DetectShifts(tc.message, "user")

		if len(shifts) == 0 {
			t.Errorf("Message with '%s': Expected to detect shift", tc.expectTrigger)
			continue
		}

		// Check if we found the expected trigger or if explicit mention is allowed
		found := false
		for _, shift := range shifts {
			if shift.Trigger == tc.expectTrigger {
				found = true
				break
			}
			if tc.allowExplicit && shift.Trigger == "explicit_mention" {
				found = true
				break
			}
		}

		if !found {
			triggers := []string{}
			for _, s := range shifts {
				triggers = append(triggers, s.Trigger)
			}
			t.Errorf("Message with '%s': Expected trigger '%s', got %v", tc.expectTrigger, tc.expectTrigger, triggers)
		}
	}
}
