package agents

import (
	"fmt"
)

// ResponseValidator validates that generated responses align with extracted user characteristics
// PHASE 3: Prevents Moly from giving misaligned advice (e.g., "explore submissive" to dominant user)
type ResponseValidator struct {
	conflictDetector *ConflictDetector // Reuse antonym map from Phase 2
}

// ValidationResult represents the outcome of response validation
type ValidationResult struct {
	IsValid             bool            // Does response align with user characteristics?
	Contradictions      []Contradiction // List of detected contradictions
	Severity            string          // "high", "medium", "low"
	ShouldBlock         bool            // Should this response be blocked?
	Reason              string          // Human-readable explanation
	RecommendedQuestion string          // Suggested clarification question
}

// Contradiction represents a detected contradiction in the response
type Contradiction struct {
	ResponseCharacteristic string  // What the response suggests (e.g., "submissive")
	UserCharacteristic     string  // What user said (e.g., "dominant")
	IsAntonym              bool    // Are they antonyms?
	Evidence               string  // Where in response was this found?
	Confidence             float64 // How confident in this contradiction?
}

// NewResponseValidator creates a new response validator
func NewResponseValidator(db interface{}) *ResponseValidator {
	// Try to create a ConflictDetector if db is provided
	var detector *ConflictDetector
	if db != nil {
		// Type assertion would happen here if we had proper typing
		// For now, create a minimal detector with just the antonym map
	}

	if detector == nil {
		// Create minimal detector with default antonym map
		detector = &ConflictDetector{
			antonymMap: defaultAntonymMap,
		}
	}

	return &ResponseValidator{
		conflictDetector: detector,
	}
}

// GetSummary returns a human-readable summary of validation
func (rv *ResponseValidator) GetSummary(result *ValidationResult) string {
	if result == nil {
		return "Validation result is nil"
	}

	if result.IsValid {
		return "✓ Response validated: aligns with user characteristics"
	}

	return fmt.Sprintf("❌ Validation failed: %s (severity: %s)",
		result.Reason, result.Severity)
}
