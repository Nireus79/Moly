package agents

import (
	"context"
	"fmt"
	"log"
	"strings"

	"moly/database"
	"moly/models"
	"moly/tools"
)

// MessageAnalysis represents the LLM's analysis of a message
type MessageAnalysis struct {
	ClarityScore           float64             // 0-1: how clear is the message?
	CanProceed             bool                // true if we have enough info to respond meaningfully
	Priority               string              // "crisis", "high", "normal", "routine"
	RequiredClarifications []ClarificationNeed // What we need to understand better
	ResponseApproach       string              // How Moly should respond
	KeyConcerns            []string            // Main things the user is concerned about
	MessageQuality         string              // "clear", "ambiguous", "vague", "complex"
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

	// Call LLM to analyze
	return mca.analyzeLLM(userMessage, conversationSummary)
}

// AnalyzeWithAnalysisContext analyzes a message with the conversation around it (summary, people, recent messages). The model
// decides every time whether Moly can proceed. (A length-and-punctuation heuristic used to decide first and skip the model when
// it looked sure; it could not tell a short message in danger from a clear request, so it is gone.)
func (mca *MessageClarityAnalyzer) AnalyzeWithAnalysisContext(analysisCtx *models.AnalysisContext) *MessageAnalysis {
	if analysisCtx == nil || analysisCtx.CurrentMessage == "" {
		return &MessageAnalysis{
			ClarityScore:   0.0,
			CanProceed:     false,
			MessageQuality: "empty",
		}
	}
	log.Printf("[MessageClarityAnalyzer] Starting analysis with context (quality: %s)", analysisCtx.ContextQuality)

	if mca.llmClient == nil {
		log.Printf("[MessageClarityAnalyzer] No LLM available, cannot proceed")
		return &MessageAnalysis{
			ClarityScore:   0.5,
			CanProceed:     false,
			MessageQuality: "unable_to_analyze",
		}
	}

	return mca.analyzeLLM(analysisCtx.CurrentMessage, mca.buildContextFromAnalysisContext(analysisCtx))
}

// analyzeLLM - Common LLM analysis logic
func (mca *MessageClarityAnalyzer) analyzeLLM(userMessage string, contextSummary string) *MessageAnalysis {

	// Ask LLM to analyze: What does this person actually need?
	// IMPORTANT: Include guidance to NOT re-ask for clarifications already addressed
	prompt := `You are analyzing a conversation to understand what the person actually needs.

Context:
` + contextSummary + `

New message: "` + userMessage + `"

Analyze this message deeply. Think like a coach who listens:

CRISIS means: active harm/safety risk (suicide, abuse, violence, immediate danger). NOT relationship/intimacy content.
HIGH means: emotional distress, conflict, difficult decisions.
NORMAL means: requests for help, advice, clarifications (including about relationships, preferences, etc.).
ROUTINE means: casual updates, questions, or background info.

IMPORTANT: Do NOT ask for clarifications the user has already provided in previous messages.
- If user said "I want to write a message", they've clarified their intention
- If user mentioned "we have things in common", they've addressed shared interests
- If user named someone and said why they matter, they've clarified the person context
Only ask about truly missing or ambiguous information.

1. Is there an actual safety/emergency concern (crisis)?
2. What is this person's primary request or concern?
3. What do we need to understand better about their situation?
4. What clarifications STILL NEEDED - only things NOT already answered?
5. How urgent is this truly (crisis, high, normal, routine)?

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
  "response_approach": "how should Moly respond (e.g., acknowledge request, ask about situation, etc.)"
}

Respond with ONLY the JSON object.`

	req := &tools.LLMRequest{
		SystemPrompt: "You are a coach who listens deeply to understand what people actually need. Respond with only valid JSON. In any question you write for the user, never include a phone number, a hotline, a crisis line or a website; ask about their safety in plain words instead.",
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

// buildContextFromAnalysisContext - Build context from AnalysisContext (bounded, efficient)
func (mca *MessageClarityAnalyzer) buildContextFromAnalysisContext(analysisCtx *models.AnalysisContext) string {
	var sb strings.Builder

	// Include conversation summary if available
	if analysisCtx.ConversationSummary != nil && analysisCtx.ConversationSummary.Arc != "" {
		sb.WriteString("CONVERSATION ARC:\n")
		sb.WriteString(analysisCtx.ConversationSummary.Arc)
		sb.WriteString("\n\n")
	}

	// Include relevant contacts that have been established (what we know about them)
	if len(analysisCtx.RelevantContacts) > 0 {
		sb.WriteString("ESTABLISHED CONTACTS:\n")
		for _, contact := range analysisCtx.RelevantContacts {
			sb.WriteString(fmt.Sprintf("- %s (%s, confidence=%.1f%%)\n", contact.Name, contact.Relationship, contact.Confidence*100))
			if len(contact.Characteristics) > 0 {
				sb.WriteString(fmt.Sprintf("  Characteristics: %s\n", strings.Join(contact.Characteristics, ", ")))
			}
		}
		sb.WriteString("\n")
	}

	// Include recent exchange
	if len(analysisCtx.RecentMessages) > 0 {
		sb.WriteString("RECENT EXCHANGE:\n")
		for _, msg := range analysisCtx.RecentMessages {
			role := "User"
			if msg.Role == "assistant" {
				role = "Moly"
			}
			sb.WriteString(fmt.Sprintf("%s: %s\n", role, msg.Content))
		}
		sb.WriteString("\n")
	}

	// NOTE: User preferences (communication style, tone) are NOT included
	// Clarity analysis must be independent of user preferences
	// Preferences only affect how Moly responds, not message analysis

	return sb.String()
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
		ClarityScore   float64  `json:"clarity_score"`
		CanProceed     bool     `json:"can_proceed"`
		Priority       string   `json:"priority"`
		KeyConcerns    []string `json:"key_concerns"`
		MessageQuality string   `json:"message_quality"`
		Clarifications []struct {
			Priority    int    `json:"priority"`
			Type        string `json:"type"`
			Description string `json:"description"`
			Question    string `json:"question"`
		} `json:"clarifications"`
		ResponseApproach string `json:"response_approach"`
	}

	var llmResp LLMResponse
	if err := tools.SafeJSONParse("MessageClarity", []byte(jsonStr), &llmResp); err != nil {
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

// ModelPriority asks the model how urgent the message is ("crisis", "high", "normal", "routine"). The
// care checks use it on their own, before the rest of the reply is decided. An unavailable or failed model returns "normal".
func (mca *MessageClarityAnalyzer) ModelPriority(analysisCtx *models.AnalysisContext, message string) string {
	if mca == nil || mca.llmClient == nil || strings.TrimSpace(message) == "" {
		return "normal"
	}
	context := "(no earlier conversation)"
	if analysisCtx != nil {
		if built := mca.buildContextFromAnalysisContext(analysisCtx); strings.TrimSpace(built) != "" {
			context = built
		}
	}
	a := mca.analyzeLLM(message, context)
	if a == nil || a.Priority == "" {
		return "normal"
	}
	return a.Priority
}
