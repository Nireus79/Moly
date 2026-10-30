package schema

import (
	"bytes"
	"encoding/json"
)

// FlexText is a text that is also accepted as an object. The first-start form of versions up to 0.1.1 sent {"notes": "..."} where
// the field is a text, and the whole profile was refused (found by a tester, 2026-10-29). An object is read as its "notes" text.
type FlexText string

// UnmarshalJSON reads a string, or an object with a "notes" text; null and anything else is the empty text.
func (f *FlexText) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) > 0 && b[0] == '{' {
		var o struct {
			Notes string `json:"notes"`
		}
		if err := json.Unmarshal(b, &o); err != nil {
			return err
		}
		*f = FlexText(o.Notes)
		return nil
	}
	var s string
	if len(b) > 0 && b[0] == '"' {
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
	}
	*f = FlexText(s)
	return nil
}

// AboutMeProfile represents a user's communication profile.
type AboutMeProfile struct {
	CommunicationStyle string   `json:"communicationStyle" validate:"max=100"`
	CoreValues         []string `json:"coreValues" validate:"max=20,dive,max=200"`
	TonePreference     string   `json:"tonePreference" validate:"max=100"`
	Preferences        FlexText `json:"preferences" validate:"max=1000"`
	Goals              []string `json:"goals" validate:"max=10,dive,max=500"`
	Patterns           []string `json:"patterns" validate:"max=10,dive,max=500"`
}

// ConversationMember represents a member in a conversation.
type ConversationMember struct {
	Name         string `json:"name" validate:"required"`
	Relationship string `json:"relationship"`
	Platform     string `json:"platform"`
	Notes        string `json:"notes" validate:"max=1000"`
}

// ConversationSettings represents conversation-specific settings.
type ConversationSettings struct {
	Mode        string `json:"mode"`    // "socratic" or "direct"
	Context     string `json:"context"` // "formal", "friendly", "dating"
	LLMProvider string `json:"llmProvider"`
}

// Conversation represents a single conversation.
type Conversation struct {
	ID          string               `json:"id"`
	Name        string               `json:"name" validate:"required,max=255"`
	Type        string               `json:"type"`
	Purpose     string               `json:"purpose" validate:"max=1000"`
	Description string               `json:"description" validate:"max=5000"`
	Members     []ConversationMember `json:"members" validate:"max=100,dive"`
	Settings    ConversationSettings `json:"settings"`
	Notes       string               `json:"notes" validate:"max=1000"`
	CreatedAt   int64                `json:"createdAt"`
	UpdatedAt   int64                `json:"updatedAt"`
}

// Contact represents a contact in the user's network.
type Contact struct {
	ID           string `json:"id"`
	Name         string `json:"name" validate:"required"`
	Relationship string `json:"relationship"`
	Notes        string `json:"notes" validate:"max=1000"`
	CreatedAt    int64  `json:"createdAt"`
}

// Phase5Request represents a message processing request for Phase5.
type Phase5Request struct {
	Message            string                 `json:"message" validate:"required,max=50000"` // the same limit as MaxMessageLength in main.go: a pasted chat is long
	ConversationID     string                 `json:"conversationId" validate:"max=100"`
	AboutMe            map[string]interface{} `json:"aboutMe"`
	SelectedContactIds []string               `json:"selectedContactIds"`
	BrowserSessionId   string                 `json:"browserSessionId"` // Browser session ID for detecting new sessions
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	SkipQuestions      bool                   `json:"skipQuestions,omitempty"` // the skip button: go ahead with what is known
}

// ExtractedFact represents a fact extracted from a user message.
type ExtractedFact struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Value   string `json:"value"`
	Subject string `json:"subject"`
}

// SubjectShift represents a detected subject shift in conversation.
type SubjectShift struct {
	ID        string `json:"id"`
	FromID    string `json:"from_id"`
	ToID      string `json:"to_id"`
	Explicit  bool   `json:"explicit"`
	Timestamp int64  `json:"timestamp"`
}

// ClarificationQuestion represents a question needing clarification.
type ClarificationQuestion struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"` // "subject", "contact", "conflict", "socratic_exploration"
	Question    string   `json:"question"`
	Context     string   `json:"context"`     // Why we're asking
	Options     []string `json:"options"`     // Multiple choice (if applicable)
	Priority    int      `json:"priority"`    // 1=critical, 2=important, 3=nice-to-have
	LinkedFacts []string `json:"linkedFacts"` // Fact IDs waiting for clarification
	Status      string   `json:"status"`      // "active", "answered", "skipped", "cancelled"
	CreatedAt   int64    `json:"createdAt"`
	// Socratic metadata
	SocraticApproach string   `json:"socraticApproach,omitempty"` // e.g., "identifying_stakeholders"
	ExpectedInsights []string `json:"expectedInsights,omitempty"` // Why this question helps
	TargetsPrinciple string   `json:"targetsPrinciple,omitempty"` // Which principle it targets
	DepthLevel       int      `json:"depthLevel,omitempty"`       // 1=surface, 2=medium, 3=deep
}

// TemporaryFact represents a fact pending clarification.
type TemporaryFact struct {
	ID                string   `json:"id"`
	Type              string   `json:"type"`
	Value             string   `json:"value"`
	Subject           string   `json:"subject"`
	PendingQuestions  []string `json:"pendingQuestions"`
	Timestamp         int64    `json:"timestamp"`
	NeedsUserContext  bool     `json:"needsUserContext"`
	NeedsSubjectMatch bool     `json:"needsSubjectMatch"`
}

// ContextAttribute represents an attributed fact stored in context.
type ContextAttribute struct {
	ID             string  `json:"id"`
	Type           string  `json:"type"`
	Value          string  `json:"value"`
	AttributedTo   string  `json:"attributedTo"`
	ConversationID string  `json:"conversationId"`
	Confidence     float64 `json:"confidence"`
	Source         string  `json:"source"`
	Timestamp      int64   `json:"timestamp"`
}
