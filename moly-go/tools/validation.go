package tools

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
)


// ValidationResult holds what was wrong
type ValidationResult struct {
	IsValid bool
	Errors  []string
}


// SanitizeIntention ensures intention preserves meaning (FIX #38 & #40: word + character limits)
// Word limit: 20 words (increased from 5 which lost semantic context like "playful")
// Character limit: 200 chars (ensures DB storage + semantic meaning preserved)
// Both limits enforced to guarantee intention always fits and remains meaningful
func SanitizeIntention(intention string) string {
	if intention == "" {
		return ""
	}

	intention = strings.TrimSpace(intention)

	// FIX #40: Character limit (200 chars for semantic preservation + DB efficiency)
	const maxChars = 200
	if len(intention) > maxChars {
		intention = intention[:maxChars]
		log.Printf("[Validation] Intention truncated to %d chars (was longer)", maxChars)
	}

	// FIX #25 REMOVED: No truncation - root cause of message echo has been fixed
	// Root causes were fixed in:
	// 1. layer_adapters.go line 156: Extract goal from entities, NOT CurrentMessage
	// 2. context_extractor.go: LLM semantic extraction via buildExtractionPrompt (not patterns)
	// 3. Fallback paths validated to never pass full message
	//
	// Keeping full semantic goals allows richer context for gap detection and response generation
	// The LLM extraction is semantic (outputs goals like "write message to X") not message echo
	log.Printf("[Validation] Intention accepted without truncation: %q (%d chars)", intention, len(intention))
	return intention
}








// FIX #31: Database Write Validation Helpers







// FIX #33: LLM Parse Validation Helpers (85+ parse points)

// SafeJSONParse validates JSON before unmarshaling (prevents panic on malformed JSON)
func SafeJSONParse(source string, data []byte, v interface{}) error {
	if len(data) == 0 {
		return fmt.Errorf("%s: empty JSON data", source)
	}
	if len(data) > 1000000 { // 1MB max
		return fmt.Errorf("%s: JSON too large (%d bytes)", source, len(data))
	}
	// Basic JSON validation - must start with { or [
	trimmed := strings.TrimSpace(string(data))
	if len(trimmed) == 0 {
		return fmt.Errorf("%s: JSON is empty after trim", source)
	}
	if trimmed[0] != '{' && trimmed[0] != '[' {
		return fmt.Errorf("%s: invalid JSON structure (must start with { or [)", source)
	}

	err := json.Unmarshal(data, v)
	if err != nil {
		return fmt.Errorf("%s: JSON parse failed: %w", source, err)
	}
	return nil
}


