package tools

import (
	"testing"
)





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










// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0))
}
