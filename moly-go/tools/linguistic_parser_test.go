package tools

import (
	"testing"
)

func TestLinguisticParser_SimpleIsAdjective(t *testing.T) {
	parser := NewLinguisticParser()

	tests := []struct {
		message          string
		expectedSub      string
		expectedProp     string
		shouldContain    bool
	}{
		{
			message:       "I am dominant",
			expectedSub:   "user",
			expectedProp:  "dominant",
			shouldContain: true,
		},
		{
			message:       "She is submissive",
			expectedSub:   "she",
			expectedProp:  "submissive",
			shouldContain: true,
		},
		{
			message:       "I'm experienced",
			expectedSub:   "user",
			expectedProp:  "experienced",
			shouldContain: true,
		},
		{
			message:       "He is bisexual",
			expectedSub:   "he",
			expectedProp:  "bisexual",
			shouldContain: true,
		},
	}

	for _, test := range tests {
		results := parser.Parse(test.message)

		if len(results) == 0 {
			t.Errorf("Message: %q\nExpected results, got none", test.message)
			continue
		}

		found := false
		for _, result := range results {
			if result.Subject == test.expectedSub && result.Property == test.expectedProp {
				found = true
				break
			}
		}

		if !found && test.shouldContain {
			t.Errorf("Message: %q\nExpected to find subject=%q, property=%q", test.message, test.expectedSub, test.expectedProp)
			for _, r := range results {
				t.Logf("  Got: subject=%q, property=%q, type=%q", r.Subject, r.Property, r.Type)
			}
		}
	}
}

func TestLinguisticParser_NegatedPreferences(t *testing.T) {
	parser := NewLinguisticParser()

	tests := []struct {
		message        string
		expectedProps  []string // Should contain "NOT"
		shouldContain  bool
	}{
		{
			message:       "I am not submissive",
			expectedProps: []string{"NOT submissive"},
			shouldContain: true,
		},
		{
			message:       "I don't want casual sex",
			expectedProps: []string{"NOT casual sex"},
			shouldContain: true,
		},
		{
			message:       "I'm not interested in bondage",
			expectedProps: []string{"NOT bondage"},
			shouldContain: true,
		},
	}

	for _, test := range tests {
		results := parser.Parse(test.message)

		if len(results) == 0 {
			t.Errorf("Message: %q\nExpected results, got none", test.message)
			continue
		}

		found := false
		for _, result := range results {
			for _, expectedProp := range test.expectedProps {
				if result.Property == expectedProp {
					found = true
					break
				}
			}
		}

		if !found && test.shouldContain {
			t.Errorf("Message: %q\nExpected to find property with NOT", test.message)
			for _, r := range results {
				t.Logf("  Got: property=%q", r.Property)
			}
		}
	}
}

func TestLinguisticParser_MultiPersonMessage(t *testing.T) {
	parser := NewLinguisticParser()

	message := "I am dominant. She is submissive. I don't want casual sex."
	results := parser.Parse(message)

	if len(results) < 3 {
		t.Errorf("Expected at least 3 results, got %d", len(results))
		for _, r := range results {
			t.Logf("  Got: subject=%q, property=%q", r.Subject, r.Property)
		}
		return
	}

	// Check for user:dominant
	found := false
	for _, r := range results {
		if r.Subject == "user" && r.Property == "dominant" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected to find user:dominant in results")
	}

	// Check for she:submissive
	found = false
	for _, r := range results {
		if r.Subject == "she" && r.Property == "submissive" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected to find she:submissive in results")
	}

	// Check for user:NOT casual sex
	found = false
	for _, r := range results {
		if r.Subject == "user" && r.Property == "NOT casual sex" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected to find user:NOT casual sex in results")
	}
}

func TestLinguisticParser_NamedContact(t *testing.T) {
	parser := NewLinguisticParser()

	tests := []struct {
		message      string
		expectedSub  string
		expectedProp string
	}{
		{
			message:      "Christine is 39 female",
			expectedSub:  "christine",
			expectedProp: "39 female",
		},
		{
			message:      "Se is submissive",
			expectedSub:  "se",
			expectedProp: "submissive",
		},
	}

	for _, test := range tests {
		results := parser.Parse(test.message)

		if len(results) == 0 {
			t.Errorf("Message: %q\nExpected results, got none", test.message)
			continue
		}

		// Check that the named subject result exists
		found := false
		for _, result := range results {
			if result.Subject == test.expectedSub && result.Property == test.expectedProp {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Message: %q\nExpected to find subject=%q, property=%q", test.message, test.expectedSub, test.expectedProp)
			for _, r := range results {
				t.Logf("  Got: subject=%q, property=%q", r.Subject, r.Property)
			}
		}
	}
}

func TestLinguisticParser_StructuredFormat(t *testing.T) {
	parser := NewLinguisticParser()

	message := "Genders: Female\nRoles: submissive\nInto: Bondage, Aftercare"
	results := parser.Parse(message)

	// Should find structured entries
	if len(results) < 3 {
		t.Errorf("Expected at least 3 results from structured format, got %d", len(results))
	}

	// Check for gender
	found := false
	for _, r := range results {
		if r.Type == "structured" && r.Property == "gender:female" {
			found = true
			break
		}
	}
	if !found {
		t.Logf("Results found:")
		for _, r := range results {
			t.Logf("  subject=%q, property=%q, type=%q", r.Subject, r.Property, r.Type)
		}
		t.Errorf("Expected to find gender:female in structured results")
	}
}

func TestLinguisticParser_Confidence(t *testing.T) {
	parser := NewLinguisticParser()

	// Test that different rule types have expected confidence levels
	results := parser.Parse("I am dominant. I prefer direct communication. I don't like games.")

	if len(results) > 0 {
		for _, r := range results {
			if r.Type == "characteristic" && r.Confidence != 0.90 {
				t.Errorf("Characteristic should have 0.90 confidence, got %v", r.Confidence)
			}
			if r.Type == "preference" && r.Confidence < 0.75 {
				t.Errorf("Preference confidence seems low: %v", r.Confidence)
			}
		}
	}
}

func TestLinguisticParser_EmptyMessage(t *testing.T) {
	parser := NewLinguisticParser()

	results := parser.Parse("")
	if len(results) != 0 {
		t.Errorf("Empty message should produce no results, got %d", len(results))
	}
}

func TestLinguisticParser_NoMatches(t *testing.T) {
	parser := NewLinguisticParser()

	message := "Hello there!"
	results := parser.Parse(message)

	if len(results) > 0 {
		t.Errorf("Generic greeting should produce no results, got %d", len(results))
		for _, r := range results {
			t.Logf("  Got: subject=%q, property=%q", r.Subject, r.Property)
		}
	}
}

func TestLinguisticParser_LikeDislike(t *testing.T) {
	parser := NewLinguisticParser()

	tests := []struct {
		message       string
		expectedProps []string
	}{
		{
			message:       "I like bondage",
			expectedProps: []string{"bondage"},
		},
		{
			message:       "I love cuddling",
			expectedProps: []string{"cuddling"},
		},
		{
			message:       "I don't like games",
			expectedProps: []string{"NOT games"},
		},
	}

	for _, test := range tests {
		results := parser.Parse(test.message)

		if len(results) < 1 {
			t.Errorf("Message: %q\nExpected at least 1 result, got %d", test.message, len(results))
			continue
		}

		found := false
		for _, r := range results {
			for _, expectedProp := range test.expectedProps {
				if r.Property == expectedProp {
					found = true
					break
				}
			}
		}

		if !found {
			t.Errorf("Message: %q\nExpected to find %v in results", test.message, test.expectedProps)
			for _, r := range results {
				t.Logf("  Got: property=%q", r.Property)
			}
		}
	}
}

func TestExtractFocusDirective(t *testing.T) {
	parser := NewLinguisticParser()

	// Test 1: "Lace is my focus"
	results := parser.extractFocusDirective("Lace is my focus")
	if len(results) == 0 {
		t.Errorf("Expected focus extraction from 'Lace is my focus'")
	} else if results[0].Property != "lace" {
		t.Errorf("Expected property 'lace', got '%s'", results[0].Property)
	} else if results[0].Confidence < 0.90 {
		t.Errorf("Expected confidence > 0.90, got %.2f", results[0].Confidence)
	}

	// Test 2: "focus on communication"
	results = parser.extractFocusDirective("Let's focus on communication")
	if len(results) == 0 {
		t.Errorf("Expected focus extraction from 'focus on communication'")
	}

	// Test 3: "don't focus on drama"
	results = parser.extractFocusDirective("I don't focus on drama")
	if len(results) == 0 {
		t.Errorf("Expected focus extraction from negated focus")
	} else if !results[0].IsNegated {
		t.Errorf("Expected IsNegated=true for 'don't focus on drama'")
	}
}

func TestExtractConstraint(t *testing.T) {
	parser := NewLinguisticParser()

	// Test 1: "remember to be patient"
	results := parser.extractConstraint("Remember to be patient with me")
	if len(results) == 0 {
		t.Errorf("Expected constraint extraction from 'remember to be patient'")
	}

	// Test 2: "keep in mind I'm nervous"
	results = parser.extractConstraint("Keep in mind I'm nervous about this")
	if len(results) == 0 {
		t.Errorf("Expected constraint extraction from 'keep in mind'")
	}
}

func TestExtractPriority(t *testing.T) {
	parser := NewLinguisticParser()

	// Test 1: "my priority is intimacy"
	results := parser.extractPriority("My priority is intimacy in our relationship")
	if len(results) == 0 {
		t.Errorf("Expected priority extraction from 'my priority is intimacy'")
	} else if results[0].Confidence < 0.90 {
		t.Errorf("Expected high confidence, got %.2f", results[0].Confidence)
	}

	// Test 2: "prioritize communication"
	results = parser.extractPriority("Please prioritize communication")
	if len(results) == 0 {
		t.Errorf("Expected priority extraction from 'prioritize'")
	}
}

func TestExtractNegatedDirective(t *testing.T) {
	parser := NewLinguisticParser()

	// Test 1: "not interested in casual"
	results := parser.extractNegatedDirective("I'm not interested in casual encounters")
	if len(results) == 0 {
		t.Errorf("Expected negated directive extraction")
	} else if !results[0].IsNegated {
		t.Errorf("Expected IsNegated=true")
	}

	// Test 2: "don't want drama"
	results = parser.extractNegatedDirective("I don't want drama in this")
	if len(results) == 0 {
		t.Errorf("Expected negated directive from 'don't want'")
	}
}

func TestMetaInstructionExtraction(t *testing.T) {
	parser := NewLinguisticParser()

	// Test: Full meta-instruction message
	message := "Lace is my focus and remember to be patient"
	results := parser.Parse(message)

	// Should extract both focus and constraint
	hasFocus := false
	hasConstraint := false

	for _, r := range results {
		if r.Type == "focus" {
			hasFocus = true
		}
		if r.Type == "constraint" {
			hasConstraint = true
		}
	}

	if !hasFocus {
		t.Errorf("Expected to find focus directive in compound instruction")
	}
	if !hasConstraint {
		t.Errorf("Expected to find constraint in compound instruction")
	}
}

func TestNegationPreservation(t *testing.T) {
	parser := NewLinguisticParser()

	// Negated and non-negated should be different
	posResults := parser.extractConstraint("remember patience")
	negResults := parser.extractNegatedDirective("not interested in X")

	if len(posResults) == 0 || len(negResults) == 0 {
		t.Errorf("Failed to extract in negation test")
	}

	if negResults[0].IsNegated != true {
		t.Errorf("Expected negated to be true for negated directive")
	}
}

func TestDetectMultiPersonBoundaries(t *testing.T) {
	parser := NewLinguisticParser()

	tests := []struct {
		name           string
		message        string
		expectedCount  int
		expectedSubject string
	}{
		{
			name:            "Single person message",
			message:         "I am dominant and looking for a submissive partner",
			expectedCount:   1,
			expectedSubject: "user",
		},
		{
			name:            "Multi-person with boundary marker",
			message:         "I am dominant. Here are some from her's She is submissive",
			expectedCount:   2,
			expectedSubject: "she",
		},
		{
			name:            "Profile boundary",
			message:         "My profile: I'm dominant. Her profile: She's submissive",
			expectedCount:   2,
			expectedSubject: "she",
		},
	}

	for _, test := range tests {
		segments := parser.DetectMultiPersonBoundaries(test.message)

		if len(segments) != test.expectedCount {
			t.Errorf("Test %q: Expected %d segments, got %d", test.name, test.expectedCount, len(segments))
			continue
		}

		// Check that last segment has expected subject
		if len(segments) > 0 && segments[len(segments)-1].Subject != test.expectedSubject {
			t.Errorf("Test %q: Expected last subject %q, got %q",
				test.name, test.expectedSubject, segments[len(segments)-1].Subject)
		}
	}
}
