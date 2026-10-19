package auth

import (
	"testing"

	_ "github.com/mutecomm/go-sqlcipher/v4"
)


// TestCodeFormat tests code format validation
func TestCodeFormat(t *testing.T) {
	tests := []struct {
		code  string
		valid bool
	}{
		{"moly-12345-abcde", true},
		{"moly-00000-00000", true},
		{"moly-fffff-fffff", true},
		{"moly-1234-abcde", false},   // Too short
		{"moly-123456-abcde", false}, // Too long
		{"MOLY-12345-abcde", false},  // Wrong case
		{"moly12345abcde", false},    // Missing dashes
		{"", false},
	}

	for _, test := range tests {
		result := IsValid(test.code)
		if result != test.valid {
			t.Errorf("IsValid(%q) = %v, expected %v", test.code, result, test.valid)
		}
	}

	t.Log("✓ Code format validation working")
}


