package agents

import (
	"context"
	"fmt"
	"log"
	"strings"

	"moly/models"
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

// ValidateResponse checks if a generated response contradicts extracted user characteristics
// Returns validation result with contradictions if any are found
// Optional: pass userProfile to validate against accumulated characteristics, not just this message's extraction
func (rv *ResponseValidator) ValidateResponse(
	ctx context.Context,
	extracted []models.ExtractedEntity,
	responseText string,
) *ValidationResult {
	return rv.ValidateResponseWithProfile(ctx, extracted, responseText, nil)
}

// ValidateResponseWithProfile is the full validation with optional AboutMe context
func (rv *ResponseValidator) ValidateResponseWithProfile(
	ctx context.Context,
	extracted []models.ExtractedEntity,
	responseText string,
	userProfile *models.AboutMe,
) *ValidationResult {

	log.Printf("[ResponseValidator] Validating response (length: %d chars)", len(responseText))

	result := &ValidationResult{
		IsValid:        true,
		Contradictions: []Contradiction{},
		Severity:       "low",
		ShouldBlock:    false,
	}

	if responseText == "" {
		log.Printf("[ResponseValidator] Empty response, skipping validation")
		return result
	}

	// Extract characteristics from the generated response
	responseCharacteristics := rv.extractCharacteristicsFromResponse(responseText)
	if len(responseCharacteristics) == 0 {
		log.Printf("[ResponseValidator] No characteristics found in response, validation passed")
		return result
	}

	// Get user characteristics - prefer accumulated AboutMe, fall back to this message's extraction
	var userCharacteristics []string
	if userProfile != nil && len(userProfile.Characteristics) > 0 {
		userCharacteristics = userProfile.Characteristics
		log.Printf("[ResponseValidator] Using accumulated user characteristics from AboutMe: %d traits", len(userCharacteristics))
	} else {
		userCharacteristics = rv.getUserCharacteristics(extracted)
		if len(userCharacteristics) > 0 {
			log.Printf("[ResponseValidator] Using current message's user characteristics: %d traits", len(userCharacteristics))
		}
	}

	if len(userCharacteristics) == 0 {
		log.Printf("[ResponseValidator] No user characteristics found")
		return result
	}

	log.Printf("[ResponseValidator] Checking %d response characteristics against %d user characteristics",
		len(responseCharacteristics), len(userCharacteristics))

	// Check for contradictions
	for _, respChar := range responseCharacteristics {
		for _, userChar := range userCharacteristics {
			// Check if they're antonyms
			if rv.conflictDetector.IsCharacteristicAntonym(respChar, userChar) {
				contradiction := Contradiction{
					ResponseCharacteristic: respChar,
					UserCharacteristic:     userChar,
					IsAntonym:              true,
					Evidence:               fmt.Sprintf("Response suggests '%s', user said '%s'", respChar, userChar),
					Confidence:             0.95, // High confidence for antonym pairs
				}

				log.Printf("[ResponseValidator] ⚠ Detected antonym contradiction: %s ↔ %s",
					respChar, userChar)

				result.Contradictions = append(result.Contradictions, contradiction)
				result.IsValid = false
				result.Severity = "high"
				result.ShouldBlock = true
			}
		}
	}

	if !result.IsValid {
		log.Printf("[ResponseValidator] ❌ Validation FAILED: %d contradictions found",
			len(result.Contradictions))
		result.Reason = rv.describeContradictions(result.Contradictions)
		result.RecommendedQuestion = rv.suggestClarificationQuestion(result.Contradictions)
	} else {
		log.Printf("[ResponseValidator] ✓ Validation PASSED: Response aligns with characteristics")
		result.Reason = "Response aligns with user characteristics"
	}

	return result
}

// extractCharacteristicsFromResponse extracts characteristics mentioned in the response
// Uses simple keyword matching for now (Phase 3.1)
// Could be enhanced with LLM analysis in future phases
func (rv *ResponseValidator) extractCharacteristicsFromResponse(responseText string) []string {
	var characteristics []string
	lowerText := strings.ToLower(responseText)

	// Get all characteristics from the antonym map
	for char := range rv.conflictDetector.antonymMap {
		// Check if characteristic appears in response (as whole word)
		// Simple heuristic: check if it appears surrounded by word boundaries
		patterns := []string{
			" " + char + " ",
			" " + char + ".",
			" " + char + ",",
			" " + char + "?",
			" " + char + "!",
			"\"" + char + "\"",
			"'" + char + "'",
		}

		for _, pattern := range patterns {
			if strings.Contains(lowerText, pattern) {
				// Avoid duplicates
				alreadyAdded := false
				for _, existing := range characteristics {
					if existing == char {
						alreadyAdded = true
						break
					}
				}
				if !alreadyAdded {
					characteristics = append(characteristics, char)
				}
				break
			}
		}
	}

	return characteristics
}

// getUserCharacteristics extracts characteristics from extracted entities
func (rv *ResponseValidator) getUserCharacteristics(extracted []models.ExtractedEntity) []string {
	var characteristics []string

	for _, entity := range extracted {
		if entity.Type == "characteristic" && entity.Confidence > 0.60 {
			// Avoid duplicates
			alreadyAdded := false
			for _, existing := range characteristics {
				if existing == entity.Value {
					alreadyAdded = true
					break
				}
			}
			if !alreadyAdded {
				characteristics = append(characteristics, entity.Value)
			}
		}
	}

	return characteristics
}

// describeContradictions creates a human-readable description of contradictions
func (rv *ResponseValidator) describeContradictions(contradictions []Contradiction) string {
	if len(contradictions) == 0 {
		return "No contradictions found"
	}

	var descriptions []string
	for _, c := range contradictions {
		desc := fmt.Sprintf("Response suggests '%s' but user said '%s'",
			c.ResponseCharacteristic, c.UserCharacteristic)
		descriptions = append(descriptions, desc)
	}

	return strings.Join(descriptions, "; ")
}

// suggestClarificationQuestion creates a clarification question for the contradiction
func (rv *ResponseValidator) suggestClarificationQuestion(contradictions []Contradiction) string {
	if len(contradictions) == 0 {
		return ""
	}

	// Use first contradiction to generate question
	c := contradictions[0]
	return fmt.Sprintf(
		"I notice I suggested exploring '%s' but you've described yourself as '%s'. Before I continue, could you clarify which better describes you right now?",
		c.ResponseCharacteristic,
		c.UserCharacteristic,
	)
}

// ShouldBlockResponse returns whether this response should be blocked
func (rv *ResponseValidator) ShouldBlockResponse(validationResult *ValidationResult) bool {
	if validationResult == nil {
		return false
	}
	return validationResult.ShouldBlock && validationResult.Severity == "high"
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
