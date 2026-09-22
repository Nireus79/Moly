package agents

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"moly/database"
	"moly/models"
	"moly/tools"
)

// MessageAnalysis represents the LLM's analysis of a message
type MessageAnalysis struct {
	ClarityScore           float64              // 0-1: how clear is the message?
	CanProceed             bool                 // true if we have enough info to respond meaningfully
	Priority               string               // "crisis", "high", "normal", "routine"
	RequiredClarifications []ClarificationNeed  // What we need to understand better
	ResponseApproach       string               // How Moly should respond
	KeyConcerns            []string             // Main things the user is concerned about
	MessageQuality         string               // "clear", "ambiguous", "vague", "complex"
}

// ClarificationNeed represents a single thing we need to clarify
type ClarificationNeed struct {
	Priority    int    // 1=critical, 2=important, 3=helpful
	Type        string // human-readable description of what needs clarification
	Description string // What's unclear
	Question    string // What to ask
}

// MessageClarityAnalyzer - LLM-driven analysis of message clarity with no static rules
type MessageClarityAnalyzer struct {
	db        *database.Database
	llmClient tools.LLMProvider
}

// NewMessageClarityAnalyzer creates a new clarity analyzer
func NewMessageClarityAnalyzer(db *database.Database, llmClient tools.LLMProvider, socraticSelector *SocraticQuestionSelector) *MessageClarityAnalyzer {
	return &MessageClarityAnalyzer{
		db:        db,
		llmClient: llmClient,
	}
}

// Analyze - LLM-driven message analysis with no static rules
// Returns MessageAnalysis with LLM's reasoning about what clarifications are needed
func (mca *MessageClarityAnalyzer) Analyze(userMessage string, conversationHistory []models.Message) *MessageAnalysis {
	log.Printf("[MessageClarityAnalyzer] Starting LLM-driven analysis: msg_len=%d history_len=%d", len(userMessage), len(conversationHistory))

	if userMessage == "" {
		log.Printf("[MessageClarityAnalyzer] Empty message")
		return &MessageAnalysis{
			ClarityScore:   0.0,
			CanProceed:     false,
			MessageQuality: "empty",
		}
	}

	if mca.llmClient == nil {
		log.Printf("[MessageClarityAnalyzer] No LLM available, cannot proceed")
		return &MessageAnalysis{
			ClarityScore:   0.5,
			CanProceed:     false,
			MessageQuality: "unable_to_analyze",
		}
	}

	// Build conversation context for LLM
	conversationSummary := mca.buildConversationSummary(conversationHistory)

	// Ask LLM to analyze: What does this person actually need?
	prompt := `You are analyzing a conversation to understand what the person actually needs.

Previous conversation:
` + conversationSummary + `

New message: "` + userMessage + `"

Analyze this message deeply. Think like a coach who listens:
1. What is this person's PRIMARY concern or crisis?
2. What do we need to understand better about their situation?
3. What clarifications would help us respond better?
4. How urgent is this (crisis, high priority, normal, routine)?
5. What should we focus on in our response?

Return ONLY valid JSON (no other text):
{
  "clarity_score": 0.0-1.0,
  "can_proceed": true/false,
  "priority": "crisis" | "high" | "normal" | "routine",
  "key_concerns": ["concern1", "concern2"],
  "message_quality": "clear" | "ambiguous" | "vague" | "complex",
  "clarifications": [
    {
      "priority": 1-3,
      "type": "description of what's unclear",
      "description": "what needs clarification",
      "question": "what to ask the person"
    }
  ],
  "response_approach": "how should Moly respond (e.g., acknowledge crisis, ask about situation, etc.)"
}

Respond with ONLY the JSON object.`

	req := &tools.LLMRequest{
		SystemPrompt: "You are a coach who listens deeply to understand what people actually need. Respond with only valid JSON.",
		UserPrompt:   prompt,
		Temperature:  0.5,
		MaxTokens:    500,
	}

	resp, err := mca.llmClient.Call(context.Background(), req)
	if err != nil {
		log.Printf("[MessageClarityAnalyzer] LLM call failed: %v, returning default analysis", err)
		return &MessageAnalysis{
			ClarityScore:   0.5,
			CanProceed:     false,
			MessageQuality: "analysis_failed",
		}
	}

	// Parse LLM response
	analysis := mca.parseLLMAnalysis(resp.Content)
	log.Printf("[MessageClarityAnalyzer] ✓ Analysis complete: priority=%s clarity=%.2f can_proceed=%v clarifications=%d",
		analysis.Priority, analysis.ClarityScore, analysis.CanProceed, len(analysis.RequiredClarifications))

	return analysis
}

// buildConversationSummary - Create context from conversation history for LLM
func (mca *MessageClarityAnalyzer) buildConversationSummary(history []models.Message) string {
	if len(history) == 0 {
		return "(No previous conversation)"
	}

	var summary strings.Builder
	// Include last few messages for context
	start := len(history) - 4
	if start < 0 {
		start = 0
	}

	for i := start; i < len(history); i++ {
		msg := history[i]
		role := "User"
		if msg.Role == "assistant" {
			role = "Moly"
		}
		summary.WriteString(role + ": " + msg.Content + "\n")
	}

	return summary.String()
}

// parseLLMAnalysis - Parse the JSON response from LLM analysis
func (mca *MessageClarityAnalyzer) parseLLMAnalysis(responseText string) *MessageAnalysis {
	// Extract JSON from response
	startIdx := strings.Index(responseText, "{")
	endIdx := strings.LastIndex(responseText, "}")
	if startIdx == -1 || endIdx == -1 {
		log.Printf("[MessageClarityAnalyzer] Could not find JSON in response: %s", responseText)
		return &MessageAnalysis{
			ClarityScore:   0.5,
			CanProceed:     false,
			MessageQuality: "parse_error",
		}
	}

	jsonStr := responseText[startIdx : endIdx+1]

	type LLMResponse struct {
		ClarityScore      float64 `json:"clarity_score"`
		CanProceed        bool    `json:"can_proceed"`
		Priority          string  `json:"priority"`
		KeyConcerns       []string `json:"key_concerns"`
		MessageQuality    string  `json:"message_quality"`
		Clarifications    []struct {
			Priority    int    `json:"priority"`
			Type        string `json:"type"`
			Description string `json:"description"`
			Question    string `json:"question"`
		} `json:"clarifications"`
		ResponseApproach string `json:"response_approach"`
	}

	var llmResp LLMResponse
	if err := json.Unmarshal([]byte(jsonStr), &llmResp); err != nil {
		log.Printf("[MessageClarityAnalyzer] Failed to unmarshal response: %v", err)
		return &MessageAnalysis{
			ClarityScore:   0.5,
			CanProceed:     false,
			MessageQuality: "parse_error",
		}
	}

	analysis := &MessageAnalysis{
		ClarityScore:           llmResp.ClarityScore,
		CanProceed:             llmResp.CanProceed,
		Priority:               llmResp.Priority,
		KeyConcerns:            llmResp.KeyConcerns,
		MessageQuality:         llmResp.MessageQuality,
		ResponseApproach:       llmResp.ResponseApproach,
		RequiredClarifications: []ClarificationNeed{},
	}

	// Convert clarifications
	for _, clarif := range llmResp.Clarifications {
		analysis.RequiredClarifications = append(analysis.RequiredClarifications, ClarificationNeed{
			Priority:    clarif.Priority,
			Type:        clarif.Type,
			Description: clarif.Description,
			Question:    clarif.Question,
		})
	}

	return analysis
}
