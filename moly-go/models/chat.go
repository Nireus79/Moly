package models

// ChatRequest is the request to send a chat message
type ChatRequest struct {
	Message        string `json:"message"`
	ConversationID string `json:"conversationId"`
}

// ContactMentionDetected indicates a contact was mentioned
type ContactMentionDetected struct {
	Detected    bool   `json:"detected"`
	PersonName  string `json:"person,omitempty"`
	Relationship string `json:"relationship,omitempty"`
	Suggestion  string `json:"suggestion,omitempty"`
}

// ChatResponse is the response from a chat message
type ChatResponse struct {
	MessageID       string                   `json:"messageId"`
	Response        string                   `json:"response"`
	ContactMention  *ContactMentionDetected  `json:"contactMention,omitempty"`
	AboutMeGaps     []string                 `json:"aboutMeGaps,omitempty"`
	SuggestedFollowUp string                 `json:"suggestedFollowUp,omitempty"`
	ContextLearned  map[string]interface{}   `json:"contextLearned,omitempty"`
	Timestamp       int64                    `json:"timestamp"`
}

// ChatMessage represents a message in the conversation
type ChatMessage struct {
	ID              string                 `json:"id"`
	UserID          string                 `json:"userId"`
	ConversationID  string                 `json:"conversationId"`
	Role            string                 `json:"role"` // "user" or "assistant"
	Content         string                 `json:"content"`
	ContextExtracted map[string]interface{} `json:"contextExtracted,omitempty"`
	ContactMention  *ContactMentionDetected `json:"contactMention,omitempty"`
	CreatedAt       int64                  `json:"createdAt"`
}

// Note: Conversation type is already defined in conversation_types.go
