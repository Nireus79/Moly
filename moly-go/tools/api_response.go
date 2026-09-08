package tools

import (
	"fmt"
	"time"
)

// APIResponse - Standard response wrapper for all API endpoints
type APIResponse struct {
	Status    string                 `json:"status"`
	Code      int                    `json:"code"`
	Message   string                 `json:"message,omitempty"`
	Data      interface{}            `json:"data,omitempty"`
	Error     *APIError              `json:"error,omitempty"`
	Meta      map[string]interface{} `json:"meta,omitempty"`
	Timestamp int64                  `json:"timestamp"`
}

// APIError - Standard error structure
type APIError struct {
	Type    string   `json:"type"`
	Message string   `json:"message"`
	Details []string `json:"details,omitempty"`
	Code    string   `json:"code,omitempty"`
}

// NewSuccessResponse - Create successful response
func NewSuccessResponse(data interface{}) *APIResponse {
	return &APIResponse{
		Status:    "success",
		Code:      200,
		Data:      data,
		Timestamp: time.Now().Unix(),
	}
}

// NewErrorResponse - Create error response
func NewErrorResponse(statusCode int, errType string, message string) *APIResponse {
	return &APIResponse{
		Status:    "error",
		Code:      statusCode,
		Error: &APIError{
			Type:    errType,
			Message: message,
		},
		Timestamp: time.Now().Unix(),
	}
}

// NewBadRequestResponse - Create 400 response
func NewBadRequestResponse(message string) *APIResponse {
	return NewErrorResponse(400, "bad_request", message)
}

// NewUnauthorizedResponse - Create 401 response
func NewUnauthorizedResponse(message string) *APIResponse {
	return NewErrorResponse(401, "unauthorized", message)
}

// NewNotFoundResponse - Create 404 response
func NewNotFoundResponse(message string) *APIResponse {
	return NewErrorResponse(404, "not_found", message)
}

// NewInternalErrorResponse - Create 500 response
func NewInternalErrorResponse(message string) *APIResponse {
	return NewErrorResponse(500, "internal_error", message)
}

// WithMessage - Add message to response
func (r *APIResponse) WithMessage(msg string) *APIResponse {
	r.Message = msg
	return r
}

// WithMeta - Add metadata to response
func (r *APIResponse) WithMeta(key string, value interface{}) *APIResponse {
	if r.Meta == nil {
		r.Meta = make(map[string]interface{})
	}
	r.Meta[key] = value
	return r
}

// ConversationResponse - Response for conversation endpoint
type ConversationResponse struct {
	Phase        string                 `json:"phase"`
	Suggestions  []SuggestionItem       `json:"suggestions,omitempty"`
	Questions    []string               `json:"questions,omitempty"`
	Context      ContextItem            `json:"context"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	SafetyAlert  *SafetyAlertItem       `json:"safetyAlert,omitempty"`
}

// SuggestionItem - Individual suggestion in response
type SuggestionItem struct {
	ID        string  `json:"id"`
	Text      string  `json:"text"`
	Reasoning string  `json:"reasoning,omitempty"`
	Tone      string  `json:"tone,omitempty"`
	Confidence float64 `json:"confidence"`
}

// ContextItem - Context information in response
type ContextItem struct {
	Level            string                 `json:"level"` // "minimal", "partial", "comprehensive"
	Quality          float64                `json:"quality"`
	Items            map[string]interface{} `json:"items,omitempty"`
	Recommendations  []string               `json:"recommendations,omitempty"`
}

// SafetyAlertItem - Safety alert in response
type SafetyAlertItem struct {
	Title       string `json:"title"`
	Severity    string `json:"severity"`
	Message     string `json:"message"`
	Resources   []string `json:"resources,omitempty"`
}

// FeedbackResponse - Response for feedback endpoint
type FeedbackResponse struct {
	Status           string            `json:"status"`
	Recorded         bool              `json:"recorded"`
	Extracted        ExtractedData     `json:"extracted"`
	ProfileUpdated   bool              `json:"profileUpdated"`
	NextPhase        string            `json:"nextPhase"`
}

// ExtractedData - Data extracted from feedback
type ExtractedData struct {
	AboutMe         *ExtractedAboutMe    `json:"aboutMe,omitempty"`
	Contact         *ExtractedContact    `json:"contact,omitempty"`
	Intention       string               `json:"intention,omitempty"`
	Confidence      float64              `json:"confidence"`
}

// ExtractedAboutMe - Extracted about me info
type ExtractedAboutMe struct {
	CommunicationStyle string   `json:"communicationStyle,omitempty"`
	Values            []string `json:"values,omitempty"`
	PreferredTone     string   `json:"preferredTone,omitempty"`
}

// ExtractedContact - Extracted contact info
type ExtractedContact struct {
	Name          string `json:"name,omitempty"`
	Relationship  string `json:"relationship,omitempty"`
	Age           string `json:"age,omitempty"`
	Characteristics []string `json:"characteristics,omitempty"`
}

// HealthResponse - Response for health check endpoint
type HealthResponse struct {
	Status        string                 `json:"status"`
	Version       string                 `json:"version"`
	BuildTime     string                 `json:"buildTime,omitempty"`
	Uptime        int64                  `json:"uptime"`
	Components    map[string]interface{} `json:"components"`
	Database      DatabaseHealth         `json:"database"`
	LLM           LLMHealth              `json:"llm"`
}

// DatabaseHealth - Database health status
type DatabaseHealth struct {
	Status   string `json:"status"` // "healthy", "degraded", "error"
	Message  string `json:"message,omitempty"`
	Latency  int64  `json:"latency"` // milliseconds
}

// LLMHealth - LLM health status
type LLMHealth struct {
	Status    string `json:"status"` // "ready", "unavailable", "degraded"
	Provider  string `json:"provider,omitempty"`
	Message   string `json:"message,omitempty"`
	Latency   int64  `json:"latency"` // milliseconds
}

// ContextResponse - Response for context endpoint
type ContextResponse struct {
	UserID           string                 `json:"userId"`
	ContextLevel     string                 `json:"contextLevel"`
	Quality          float64                `json:"quality"`
	LastUpdated      int64                  `json:"lastUpdated"`
	AboutMe          *AboutMeItem           `json:"aboutMe,omitempty"`
	Contact          *ContactItem           `json:"contact,omitempty"`
	History          []HistoryItem          `json:"history,omitempty"`
	BehavioralProfile *ProfileItem          `json:"behavioralProfile,omitempty"`
	Gaps             []string               `json:"gaps,omitempty"`
}

// AboutMeItem - About me in response
type AboutMeItem struct {
	CommunicationStyle string   `json:"communicationStyle"`
	Values            []string `json:"values"`
	PreferredTone     string   `json:"preferredTone"`
	CreatedAt         int64    `json:"createdAt"`
}

// ContactItem - Contact in response
type ContactItem struct {
	Name              string   `json:"name"`
	Relationship      string   `json:"relationship"`
	Age               string   `json:"age"`
	Characteristics   []string `json:"characteristics"`
	CreatedAt         int64    `json:"createdAt"`
}

// HistoryItem - History entry in response
type HistoryItem struct {
	Timestamp int64  `json:"timestamp"`
	Content   string `json:"content"`
	Type      string `json:"type"`
}

// ProfileItem - Profile in response
type ProfileItem struct {
	Confidence           float64              `json:"confidence"`
	CommunicationStyle   string               `json:"communicationStyle"`
	PreferredTones       []string             `json:"preferredTones"`
	InteractionFrequency string               `json:"interactionFrequency"`
	GrowthTrend          string               `json:"growthTrend"`
	Insights            []string             `json:"insights"`
}

// PaginatedResponse - Response with pagination
type PaginatedResponse struct {
	Items      interface{} `json:"items"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"pageSize"`
	HasMore    bool        `json:"hasMore"`
	NextCursor string      `json:"nextCursor,omitempty"`
}

// ErrorDetail - Individual error detail
type ErrorDetail struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// ValidationErrorResponse - Response for validation errors
func ValidationErrorResponse(details []ErrorDetail) *APIResponse {
	resp := NewBadRequestResponse("Validation failed")
	resp.Error = &APIError{
		Type:    "validation_error",
		Message: "One or more validation errors occurred",
	}
	resp.Meta = map[string]interface{}{
		"details": details,
	}
	return resp
}

// RateLimitResponse - Response when rate limited
func RateLimitResponse(retryAfter int) *APIResponse {
	resp := NewErrorResponse(429, "rate_limit_exceeded", fmt.Sprintf("Rate limit exceeded. Retry after %d seconds", retryAfter))
	resp.Meta = map[string]interface{}{
		"retryAfter": retryAfter,
	}
	return resp
}
