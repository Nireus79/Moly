package schema

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse wraps an error message in the standard error format.
type ErrorResponse struct {
	Error string `json:"error"`
}

// SuccessResponse is a generic container for successful responses.
type SuccessResponse map[string]interface{}

// RespondError sends a standardized error response.
func RespondError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	resp := ErrorResponse{Error: message}
	json.NewEncoder(w).Encode(resp)
}

// RespondSuccess sends a standardized success response with the given resource.
// The key parameter is the resource name (e.g., "profile", "conversation", "contact").
// The resource parameter is the data to be included (e.g., an AboutMeProfile, Conversation, etc.).
func RespondSuccess(w http.ResponseWriter, statusCode int, key string, resource interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	resp := SuccessResponse{
		"success": true,
		key:      resource,
	}

	json.NewEncoder(w).Encode(resp)
}
