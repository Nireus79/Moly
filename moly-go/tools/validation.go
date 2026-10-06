package tools

import (
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

// SanitizeIntention ensures intention is 2-5 words (FIX #25 + generalized)
func SanitizeIntention(intention string) string {
	if intention == "" {
		return ""
	}

	words := strings.Fields(strings.TrimSpace(intention))
	if len(words) > 5 {
		log.Printf("[Validation] Intention too long (%d words), truncating to first 5", len(words))
		return strings.Join(words[:5], " ")
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
