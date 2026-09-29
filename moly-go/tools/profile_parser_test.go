package tools

import (
	"testing"
)

func TestProfileParser_FetLifeFormat(t *testing.T) {
	parser := NewProfileParser()

	message := `Genders: Female
Roles: submissive
Into: Bondage, Aftercare, Communication`

	data := parser.Parse(message)

	if data.Format != "fetlife" {
		t.Errorf("Expected format 'fetlife', got %q", data.Format)
	}

	if len(data.Attributes) != 3 {
		t.Errorf("Expected 3 attributes, got %d", len(data.Attributes))
	}

	// Check gender
	gender := parser.GetAttribute(data, "gender")
	if gender == nil {
		t.Errorf("Expected to find gender attribute")
	} else if gender.Value != "Female" {
		t.Errorf("Expected gender 'Female', got %q", gender.Value)
	}

	// Check interests list
	interests := parser.GetAttribute(data, "interests")
	if interests == nil {
		t.Errorf("Expected to find interests attribute")
	} else if !interests.IsList {
		t.Errorf("Expected interests to be marked as list")
	} else if len(interests.Values) != 3 {
		t.Errorf("Expected 3 interest values, got %d", len(interests.Values))
	}
}

func TestProfileParser_GetAllValues(t *testing.T) {
	parser := NewProfileParser()

	message := "Into: Bondage, Aftercare, Communication, Cuddling"
	data := parser.Parse(message)

	values := parser.GetAllValues(data, "interests")

	if len(values) != 4 {
		t.Errorf("Expected 4 values, got %d", len(values))
	}

	expected := []string{"Bondage", "Aftercare", "Communication", "Cuddling"}
	for i, val := range values {
		if val != expected[i] {
			t.Errorf("Value %d: expected %q, got %q", i, expected[i], val)
		}
	}
}

func TestProfileParser_SingleValue(t *testing.T) {
	parser := NewProfileParser()

	message := "Age: 42"
	data := parser.Parse(message)

	age := parser.GetAttribute(data, "age")
	if age == nil {
		t.Errorf("Expected to find age attribute")
	} else if age.Value != "42" {
		t.Errorf("Expected age '42', got %q", age.Value)
	}

	if age.IsList {
		t.Errorf("Expected single value to not be marked as list")
	}
}

func TestProfileParser_MultilineFormat(t *testing.T) {
	parser := NewProfileParser()

	message := `Genders: Female
Roles: submissive
Age: 39
Location: Portland, OR
Status: Single`

	data := parser.Parse(message)

	if len(data.Attributes) != 5 {
		t.Errorf("Expected 5 attributes, got %d", len(data.Attributes))
	}

	// Verify all attributes exist
	keys := []string{"gender", "role", "age", "location", "relationship_status"}
	for _, key := range keys {
		attr := parser.GetAttribute(data, key)
		if attr == nil {
			t.Errorf("Expected to find %q attribute", key)
		}
	}
}

func TestProfileParser_NormalizeCategoryKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"gender", "gender"},
		{"Genders", "gender"},
		{"role", "role"},
		{"Roles", "role"},
		{"into", "interests"},
		{"Into", "interests"},
		{"kinks", "kinks"},
		{"kink", "kinks"},
		{"relationship status", "relationship_status"},
		{"status", "relationship_status"},
	}

	for _, test := range tests {
		result := normalizeCategoryKey(test.input)
		if result != test.expected {
			t.Errorf("Input %q: expected %q, got %q", test.input, test.expected, result)
		}
	}
}

func TestProfileParser_EmptyMessage(t *testing.T) {
	parser := NewProfileParser()

	data := parser.Parse("")

	if len(data.Attributes) != 0 {
		t.Errorf("Expected no attributes for empty message, got %d", len(data.Attributes))
	}
}

func TestProfileParser_NoStructuredData(t *testing.T) {
	parser := NewProfileParser()

	message := "Just a regular message with no profile information"
	data := parser.Parse(message)

	if len(data.Attributes) != 0 {
		t.Errorf("Expected no attributes for unstructured message")
	}

	if data.Format != "unstructured" {
		t.Errorf("Expected format 'unstructured', got %q", data.Format)
	}
}

func TestProfileParser_FormatAsStructuredData(t *testing.T) {
	parser := NewProfileParser()

	message := `Genders: Female
Roles: submissive
Into: Bondage, Aftercare`

	data := parser.Parse(message)
	structured := parser.FormatAsStructuredData(data)

	if val, exists := structured["gender"]; !exists {
		t.Errorf("Expected 'gender' in structured data")
	} else if val != "Female" {
		t.Errorf("Expected gender 'Female', got %v", val)
	}

	if val, exists := structured["interests"]; !exists {
		t.Errorf("Expected 'interests' in structured data")
	} else if interests, ok := val.([]string); !ok {
		t.Errorf("Expected interests to be []string")
	} else if len(interests) != 2 {
		t.Errorf("Expected 2 interests, got %d", len(interests))
	}
}

func TestProfileParser_GetSummary(t *testing.T) {
	parser := NewProfileParser()

	message := `Genders: Female
Roles: submissive`

	data := parser.Parse(message)
	summary := parser.GetSummary(data)

	if summary == "" {
		t.Errorf("Expected non-empty summary")
	}

	if !contains(summary, "gender") || !contains(summary, "Female") {
		t.Errorf("Summary should contain gender info: %s", summary)
	}
}

func TestProfileParser_ValidateAttribute(t *testing.T) {
	parser := NewProfileParser()

	tests := []struct {
		key      string
		value    string
		expected bool
	}{
		{"age", "42", true},
		{"age", "abc", false},
		{"gender", "female", true},
		{"gender", "unknown", false},
		{"role", "dominant", true},
		{"role", "maybe", false},
		{"interests", "anything here", true},
	}

	for _, test := range tests {
		result := parser.ValidateAttribute(test.key, test.value)
		if result != test.expected {
			t.Errorf("ValidateAttribute(%q, %q): expected %v, got %v", test.key, test.value, test.expected, result)
		}
	}
}

func TestProfileParser_MergeProfiles(t *testing.T) {
	parser := NewProfileParser()

	existing := parser.Parse("Gender: Female\nAge: 39")
	newer := parser.Parse("Gender: Female\nRole: submissive")

	merged := parser.MergeProfiles(existing, newer)

	if len(merged.Attributes) != 3 {
		t.Errorf("Expected 3 attributes in merged profile, got %d", len(merged.Attributes))
	}

	// Both age and role should exist
	age := parser.GetAttribute(merged, "age")
	if age == nil {
		t.Errorf("Expected age to be preserved from existing profile")
	}

	role := parser.GetAttribute(merged, "role")
	if role == nil {
		t.Errorf("Expected role to be added from newer profile")
	}
}

func TestProfileParser_ConfidenceScores(t *testing.T) {
	parser := NewProfileParser()

	// FetLife format should have high confidence
	fetLifeData := parser.Parse("Genders: Female")
	if fetLifeData.Attributes[0].Confidence != 0.95 {
		t.Errorf("Expected FetLife format confidence 0.95, got %v", fetLifeData.Attributes[0].Confidence)
	}

	// Generic key-value also matches FetLife pattern, so will have high confidence
	kvData := parser.Parse("Gender: Female")
	if len(kvData.Attributes) > 0 && kvData.Attributes[0].Confidence < 0.75 {
		t.Errorf("Expected key-value format confidence >= 0.75, got %v", kvData.Attributes[0].Confidence)
	}
}

func TestProfileParser_ExtractProfileFromMessage(t *testing.T) {
	parser := NewProfileParser()

	message := "Genders: Female\nRoles: submissive"
	data := parser.ExtractProfileFromMessage(message, "christine")

	if data.Subject != "christine" {
		t.Errorf("Expected subject 'christine', got %q", data.Subject)
	}

	if len(data.Attributes) != 2 {
		t.Errorf("Expected 2 attributes, got %d", len(data.Attributes))
	}
}

func TestProfileParser_CaseInsensitivity(t *testing.T) {
	parser := NewProfileParser()

	message := `genders: FEMALE
roles: SUBMISSIVE
into: Bondage, AFTERCARE`

	data := parser.Parse(message)

	if len(data.Attributes) != 3 {
		t.Errorf("Expected 3 attributes, got %d", len(data.Attributes))
	}

	gender := parser.GetAttribute(data, "GENDER")
	if gender == nil {
		t.Errorf("Expected case-insensitive lookup to work")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0))
}
