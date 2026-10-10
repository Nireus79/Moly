package models

// ContactMentionDetected indicates a contact was mentioned
type ContactMentionDetected struct {
	Detected     bool   `json:"detected"`
	PersonName   string `json:"person,omitempty"`
	Relationship string `json:"relationship,omitempty"`
	Suggestion   string `json:"suggestion,omitempty"`
}

// ChatMessage represents a message in the conversation
type ChatMessage struct {
	ID               string                  `json:"id"`
	UserID           string                  `json:"userId"`
	ConversationID   string                  `json:"conversationId"`
	Role             string                  `json:"role"` // "user" or "assistant"
	Content          string                  `json:"content"`
	ContextExtracted map[string]interface{}  `json:"contextExtracted,omitempty"`
	ContactMention   *ContactMentionDetected `json:"contactMention,omitempty"`
	CreatedAt        int64                   `json:"createdAt"`
}

// Note: Conversation type is already defined in conversation_types.go
