package tools

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// ValidationSpec defines what valid data looks like
type ValidationSpec struct {
	FieldName        string
	Type             string // "string", "stringArray", "float64", "int"
	MinLength        int    // For strings
	MaxLength        int
	MinArrayLen      int    // For arrays
	MaxArrayLen      int
	MinValue         float64 // For floats
	MaxValue         float64
	AllowedValues    []string // For enums
	Required         bool
}

// ValidationResult holds what was wrong
type ValidationResult struct {
	IsValid bool
	Errors  []string
}

// ValidateExtractedContextField validates a single field from extraction
func ValidateExtractedContextField(fieldName string, value interface{}, spec ValidationSpec) ValidationResult {
	result := ValidationResult{IsValid: true, Errors: []string{}}

	switch spec.Type {
	case "string":
		str, ok := value.(string)
		if !ok {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: not a string (got %T)", fieldName, value))
			result.IsValid = false
			break
		}

		if spec.Required && str == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: required but empty", fieldName))
			result.IsValid = false
		}

		if len(str) < spec.MinLength {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: too short (got %d, min %d)", fieldName, len(str), spec.MinLength))
			result.IsValid = false
		}

		if spec.MaxLength > 0 && len(str) > spec.MaxLength {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: too long (got %d, max %d)", fieldName, len(str), spec.MaxLength))
			result.IsValid = false
		}

		if len(spec.AllowedValues) > 0 {
			found := false
			for _, allowed := range spec.AllowedValues {
				if str == allowed {
					found = true
					break
				}
			}
			if !found {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: invalid value %q (allowed: %v)", fieldName, str, spec.AllowedValues))
				result.IsValid = false
			}
		}

	case "stringArray":
		arr, ok := value.([]string)
		if !ok {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: not a string array (got %T)", fieldName, value))
			result.IsValid = false
			break
		}

		if spec.MinArrayLen > 0 && len(arr) < spec.MinArrayLen {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: too few items (got %d, min %d)", fieldName, len(arr), spec.MinArrayLen))
			result.IsValid = false
		}

		if spec.MaxArrayLen > 0 && len(arr) > spec.MaxArrayLen {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: too many items (got %d, max %d)", fieldName, len(arr), spec.MaxArrayLen))
			result.IsValid = false
		}

	case "float64":
		f, ok := value.(float64)
		if !ok {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: not a float64 (got %T)", fieldName, value))
			result.IsValid = false
			break
		}

		if f < spec.MinValue {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: below minimum (got %.2f, min %.2f)", fieldName, f, spec.MinValue))
			result.IsValid = false
		}

		if spec.MaxValue > 0 && f > spec.MaxValue {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: above maximum (got %.2f, max %.2f)", fieldName, f, spec.MaxValue))
			result.IsValid = false
		}
	}

	return result
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

	// Word limit (20 words for semantic completeness)
	words := strings.Fields(intention)
	if len(words) > 20 {
		log.Printf("[Validation] Intention very long (%d words), truncating to first 20", len(words))
		return strings.Join(words[:20], " ")
	}

	return intention
}

// SanitizeStringField ensures string isn't empty or too long
func SanitizeStringField(fieldName string, value string, maxLen int) string {
	if len(value) == 0 {
		log.Printf("[Validation] %s is empty, using default", fieldName)
		return ""
	}

	if maxLen > 0 && len(value) > maxLen {
		log.Printf("[Validation] %s exceeded %d chars, truncating", fieldName, maxLen)
		return value[:maxLen]
	}

	return value
}

// EnsureArrayNotEmpty returns empty slice if nil
func EnsureArrayNotEmpty(arr []string) []string {
	if arr == nil {
		return []string{}
	}
	return arr
}

// ValidateBeforeUseInTemplate checks if string is safe for sprintf
func ValidateBeforeUseInTemplate(fieldName string, value string) string {
	if value == "" {
		log.Printf("[Validation] Template field %s is empty, using fallback", fieldName)
		return "[missing]"
	}

	// Prevent format string injection
	if strings.Contains(value, "%") {
		log.Printf("[Validation] Template field %s contains format specifiers, escaping")
		value = strings.ReplaceAll(value, "%", "%%")
	}

	return value
}

// SafeArrayLength returns length of array, with nil protection (FIX #28)
func SafeArrayLength(arr interface{}) int {
	if arr == nil {
		return 0
	}

	switch v := arr.(type) {
	case []string:
		return len(v)
	case []interface{}:
		return len(v)
	default:
		return 0
	}
}

// SafeArrayAccess returns element at index with bounds checking (FIX #28)
func SafeArrayAccess(fieldName string, arr []string, index int) string {
	if arr == nil || index < 0 || index >= len(arr) {
		log.Printf("[Validation] Array %s bounds check failed (len=%d, index=%d)", fieldName, len(arr), index)
		return ""
	}
	return arr[index]
}

// SafeArrayIterate returns array, ensuring it's not nil (FIX #28)
func SafeArrayIterate(fieldName string, arr []string) []string {
	if arr == nil {
		log.Printf("[Validation] Array %s is nil, using empty slice", fieldName)
		return []string{}
	}
	return arr
}

// ValidateArrayBeforeIteration ensures array is safe to iterate (FIX #28)
func ValidateArrayBeforeIteration(fieldName string, arr []string, minItems int) bool {
	if arr == nil {
		log.Printf("[Validation] Array %s is nil, skipping iteration", fieldName)
		return false
	}
	if len(arr) < minItems {
		log.Printf("[Validation] Array %s has insufficient items (%d < %d), skipping iteration", fieldName, len(arr), minItems)
		return false
	}
	return true
}

// FIX #31: Database Write Validation Helpers

// ValidateRequiredField ensures a string field is not empty
func ValidateRequiredField(fieldName string, value string) error {
	if value == "" {
		return fmt.Errorf("%s is required (cannot be empty)", fieldName)
	}
	if len(value) > 255 {
		return fmt.Errorf("%s exceeds max length (got %d, max 255)", fieldName, len(value))
	}
	return nil
}

// ValidateEnumField ensures value is one of allowed options
func ValidateEnumField(fieldName string, value string, allowedValues []string) error {
	if value == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	for _, allowed := range allowedValues {
		if value == allowed {
			return nil
		}
	}
	return fmt.Errorf("%s has invalid value %q (allowed: %v)", fieldName, value, allowedValues)
}

// ValidateSeverityLevel ensures severity is valid
func ValidateSeverityLevel(severity string) error {
	validLevels := []string{"low", "medium", "high", "critical"}
	return ValidateEnumField("severity", severity, validLevels)
}

// ValidateJSONField validates JSON can be marshaled without error
func ValidateJSONField(fieldName string, value interface{}) error {
	if value == nil {
		return nil // nil is ok for optional JSON
	}
	_, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("%s failed JSON validation: %w", fieldName, err)
	}
	return nil
}

// ValidateTimestamp ensures timestamp is reasonable
func ValidateTimestamp(fieldName string, timestamp int64) error {
	if timestamp == 0 {
		return fmt.Errorf("%s is required (cannot be zero)", fieldName)
	}
	if timestamp > 9999999999 { // Year ~2286
		return fmt.Errorf("%s appears invalid (too far in future: %d)", fieldName, timestamp)
	}
	return nil
}

// ValidateID ensures ID field meets requirements
func ValidateID(fieldName string, id string) error {
	if id == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	if len(id) > 255 {
		return fmt.Errorf("%s exceeds max length (%d > 255)", fieldName, len(id))
	}
	return nil
}

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

// ValidateParsedLLMResponse checks common LLM response field patterns
func ValidateParsedLLMResponse(source string, response map[string]interface{}, requiredFields []string) error {
	if response == nil {
		return fmt.Errorf("%s: parsed response is nil", source)
	}
	if len(response) == 0 {
		return fmt.Errorf("%s: parsed response is empty", source)
	}

	// Check required fields
	for _, field := range requiredFields {
		val, exists := response[field]
		if !exists {
			return fmt.Errorf("%s: missing required field %q", source, field)
		}
		// Check for empty strings or nil
		if val == nil {
			return fmt.Errorf("%s: field %q is nil", source, field)
		}
		if str, ok := val.(string); ok && str == "" {
			return fmt.Errorf("%s: field %q is empty string", source, field)
		}
	}

	return nil
}

// ParseLLMResponseWithValidation is a one-liner for safe JSON parsing + validation
func ParseLLMResponseWithValidation(source string, jsonStr string, v interface{}, requiredFields []string) error {
	// First, basic structure validation
	if err := SafeJSONParse(source, []byte(jsonStr), v); err != nil {
		return err
	}

	// Then, content validation for maps
	if m, ok := v.(*map[string]interface{}); ok {
		if err := ValidateParsedLLMResponse(source, *m, requiredFields); err != nil {
			return err
		}
	}

	return nil
}
