package schema

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// ValidationError holds custom error messages for specific fields.
type ValidationError struct {
	Field   string
	Message string
}

// ValidateStruct validates a struct and returns custom error messages matching the golden spec.
func ValidateStruct(s interface{}) error {
	errs := validate.Struct(s)
	if errs == nil {
		return nil
	}

	// Extract the first validation error and convert to golden-spec error message
	for _, err := range errs.(validator.ValidationErrors) {
		return convertValidationError(err)
	}
	return nil
}

// convertValidationError converts a validator.FieldError to the exact error message
// expected by the golden spec.
func convertValidationError(err validator.FieldError) error {
	fieldName := err.Field()
	tag := err.Tag()

	switch {
	// Required field missing
	case tag == "required":
		// Convert field names to expected error messages
		switch fieldName {
		case "Name":
			return fmt.Errorf("Name is required")
		case "Message":
			return fmt.Errorf("Missing message or conversationId")
		case "ConversationID":
			return fmt.Errorf("Missing message or conversationId")
		default:
			return fmt.Errorf("%s is required", fieldName)
		}

	// Max length violations
	case tag == "max":
		param := err.Param()
		switch fieldName {
		case "Message":
			return fmt.Errorf("Message too long (max %s characters)", param)
		case "ConversationID":
			return fmt.Errorf("ConversationID too long (max %s characters)", param)
		case "CommunicationStyle":
			return fmt.Errorf("Communication style too long (max %s characters)", param)
		case "TonePreference":
			return fmt.Errorf("Tone preference too long (max %s characters)", param)
		case "Preferences":
			return fmt.Errorf("Preferences too long (max %s characters)", param)
		case "Notes":
			return fmt.Errorf("Notes too long (max %s characters)", param)
		default:
			return fmt.Errorf("%s too long (max %s characters)", fieldName, param)
		}

	// Array/slice max violations
	case tag == "max" && strings.Contains(fieldName, "Values"):
		param := err.Param()
		return fmt.Errorf("Core values exceeds max %s items", param)

	// Dive (nested validation) failures
	case tag == "dive":
		return fmt.Errorf("Invalid value in %s", fieldName)

	default:
		return fmt.Errorf("Invalid %s: %s", fieldName, tag)
	}
}
