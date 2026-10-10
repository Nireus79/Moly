package tools

import (
)











// ConversationResponse - Response for conversation endpoint
type ConversationResponse struct {
	Phase       string                 `json:"phase"`
	Suggestions []SuggestionItem       `json:"suggestions,omitempty"`
	Questions   []string               `json:"questions,omitempty"`
	Context     ContextItem            `json:"context"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	SafetyAlert *SafetyAlertItem       `json:"safetyAlert,omitempty"`
}

// SuggestionItem - Individual suggestion in response
type SuggestionItem struct {
	ID         string  `json:"id"`
	Text       string  `json:"text"`
	Reasoning  string  `json:"reasoning,omitempty"`
	Tone       string  `json:"tone,omitempty"`
	Confidence float64 `json:"confidence"`
}

// ContextItem - Context information in response
type ContextItem struct {
	Level           string                 `json:"level"` // "minimal", "partial", "comprehensive"
	Quality         float64                `json:"quality"`
	Items           map[string]interface{} `json:"items,omitempty"`
	Recommendations []string               `json:"recommendations,omitempty"`
}

// SafetyAlertItem - Safety alert in response
type SafetyAlertItem struct {
	Title     string   `json:"title"`
	Severity  string   `json:"severity"`
	Message   string   `json:"message"`
	Resources []string `json:"resources,omitempty"`
}



// ExtractedAboutMe - Extracted about me info
type ExtractedAboutMe struct {
	CommunicationStyle string   `json:"communicationStyle,omitempty"`
	Values             []string `json:"values,omitempty"`
	PreferredTone      string   `json:"preferredTone,omitempty"`
}

// ExtractedContact - Extracted contact info
type ExtractedContact struct {
	Name            string   `json:"name,omitempty"`
	Relationship    string   `json:"relationship,omitempty"`
	Age             string   `json:"age,omitempty"`
	Characteristics []string `json:"characteristics,omitempty"`
}




// ContextResponse - Response for context endpoint
type ContextResponse struct {
	UserID            string        `json:"userId"`
	ContextLevel      string        `json:"contextLevel"`
	Quality           float64       `json:"quality"`
	LastUpdated       int64         `json:"lastUpdated"`
	AboutMe           *AboutMeItem  `json:"aboutMe,omitempty"`
	Contact           *ContactItem  `json:"contact,omitempty"`
	History           []HistoryItem `json:"history,omitempty"`
	BehavioralProfile *ProfileItem  `json:"behavioralProfile,omitempty"`
	Gaps              []string      `json:"gaps,omitempty"`
}

// AboutMeItem - About me in response
type AboutMeItem struct {
	CommunicationStyle string   `json:"communicationStyle"`
	Values             []string `json:"values"`
	PreferredTone      string   `json:"preferredTone"`
	CreatedAt          int64    `json:"createdAt"`
}

// ContactItem - Contact in response
type ContactItem struct {
	Name            string   `json:"name"`
	Relationship    string   `json:"relationship"`
	Age             string   `json:"age"`
	Characteristics []string `json:"characteristics"`
	CreatedAt       int64    `json:"createdAt"`
}

// HistoryItem - History entry in response
type HistoryItem struct {
	Timestamp int64  `json:"timestamp"`
	Content   string `json:"content"`
	Type      string `json:"type"`
}

// ProfileItem - Profile in response
type ProfileItem struct {
	Confidence           float64  `json:"confidence"`
	CommunicationStyle   string   `json:"communicationStyle"`
	PreferredTones       []string `json:"preferredTones"`
	InteractionFrequency string   `json:"interactionFrequency"`
	GrowthTrend          string   `json:"growthTrend"`
	Insights             []string `json:"insights"`
}




