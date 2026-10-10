package util

// Contains checks if a string contains a substring
// AUDIT FIX: Consolidated from 4 duplicate implementations
func Contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(substr) > 0 && compareSubstring(s, substr)))
}

// compareSubstring is a helper for substring comparison
func compareSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

