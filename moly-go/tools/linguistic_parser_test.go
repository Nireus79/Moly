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
