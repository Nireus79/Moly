package agents

import (
	"context"
	"fmt"
	"log"
	"strings"

	"moly/models"
	"moly/tools"
)

// Intent - What is the user doing with this message
type Intent string

const (
	IntentAsk     Intent = "asking"     // User asks Moly a question
	IntentShare   Intent = "sharing"    // User shares information/context
	IntentReact   Intent = "reacting"   // User reacts to something Moly said
	IntentVent    Intent = "venting"    // User expresses emotion
	IntentConfirm Intent = "confirming" // User confirms/corrects understanding
	IntentUnknown Intent = "unknown"    // No clear intent
)

// IntentAnalysis - Result of intent detection
type IntentAnalysis struct {
	Intent             Intent
	Confidence         float64 // 0-1
	QuestionAsked      string  // if Intent=="asking"
	InfoShared         string  // if Intent=="sharing"
	Emotional          bool    // true if high emotional content
	ReactionTarget     string  // what they're reacting to (if Intent=="reacting")
	ConfirmedStatement string  // what they're confirming (if Intent=="confirming")
}

// LLMIntentDetector uses LLM reasoning for intent detection
type LLMIntentDetector struct {
	llmClient tools.LLMProvider
}

// NewLLMIntentDetector creates a new LLM-based intent detector
func NewLLMIntentDetector(llm tools.LLMProvider) *LLMIntentDetector {
	return &LLMIntentDetector{llmClient: llm}
}

// DetectIntent analyzes what the user is doing in this message using LLM reasoning
func DetectIntent(userMessage string, conversationHistory []models.Message) IntentAnalysis {
	log.Printf("[IntentDetector] Analyzing message intent")

	msg := strings.TrimSpace(userMessage)
	if msg == "" {
		return IntentAnalysis{Intent: IntentUnknown, Confidence: 0}
	}

	// For now, return unknown - the LLM version will be called from conversation_agent
	// where we have access to the LLM client
	return IntentAnalysis{Intent: IntentUnknown, Confidence: 0}
}

// DetectIntentWithLLM performs LLM-driven intent analysis
func (lid *LLMIntentDetector) DetectIntentWithLLM(userMessage string, conversationHistory []models.Message) IntentAnalysis {
	if lid.llmClient == nil {
		log.Printf("[IntentDetector] No LLM available, cannot detect intent")
		return IntentAnalysis{Intent: IntentUnknown, Confidence: 0}
	}

	log.Printf("[IntentDetector] Analyzing message intent with LLM")

	msg := strings.TrimSpace(userMessage)
	if msg == "" {
		return IntentAnalysis{Intent: IntentUnknown, Confidence: 0}
	}

	// Build conversation context for the LLM
	historyContext := buildIntentHistoryContext(conversationHistory)

	// LLM prompt to detect intent
	systemPrompt := `You are an intent analyzer. Analyze what the user is doing in their message.

Respond with ONLY a JSON object (no markdown, no explanation):
{
  "intent": "asking|sharing|reacting|venting|confirming|unknown",
  "confidence": 0.0-1.0,
  "reasoning": "brief explanation of why"
}

Intent definitions:
- "asking": User asks Moly a question or requests information/advice
- "sharing": User provides information, context, experiences, or answers to previous questions
- "reacting": User responds directly to something Moly just said (agreement, disagreement, correction)
- "venting": User expresses strong emotion (frustration, anger, fear, anxiety, sadness)
- "confirming": User confirms, corrects, or clarifies their previous statement
- "unknown": No clear intent can be determined

Be generous with "sharing" - if user provides information, priorities, goals, or answers to implied questions, that's sharing.
Be specific with "reacting" - only if responding directly to Moly's words.
Use high confidence (0.8+) when intent is clear. Use medium (0.5-0.8) when there are mixed signals.`

	userPrompt := fmt.Sprintf(`User message: "%s"

Recent conversation context:
%s

What is the user's intent in this message?`, msg, historyContext)

	req := &tools.LLMRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Temperature:  0.3,
		MaxTokens:    200,
	}

	resp, err := lid.llmClient.Call(context.Background(), req)
	if err != nil {
		log.Printf("[IntentDetector] LLM call failed: %v, returning unknown", err)
		return IntentAnalysis{Intent: IntentUnknown, Confidence: 0}
	}

	// Parse LLM response
	analysis := parseIntentResponse(resp.Content, msg)
	log.Printf("[IntentDetector] Detected %s (confidence=%.2f)", analysis.Intent, analysis.Confidence)

	return analysis
}

// parseIntentResponse parses the LLM's JSON response
func parseIntentResponse(llmResponse string, userMessage string) IntentAnalysis {
	analysis := IntentAnalysis{Intent: IntentUnknown, Confidence: 0}

	// Try to find the intent classification
	response := strings.ToLower(strings.TrimSpace(llmResponse))

	// Extract intent from response
	switch {
	case strings.Contains(response, `"intent":"asking"`):
		analysis.Intent = IntentAsk
		analysis.QuestionAsked = userMessage
	case strings.Contains(response, `"intent":"sharing"`):
		analysis.Intent = IntentShare
		analysis.InfoShared = userMessage
	case strings.Contains(response, `"intent":"reacting"`):
		analysis.Intent = IntentReact
		analysis.ReactionTarget = extractReactionContext(userMessage)
	case strings.Contains(response, `"intent":"venting"`):
		analysis.Intent = IntentVent
		analysis.Emotional = true
	case strings.Contains(response, `"intent":"confirming"`):
		analysis.Intent = IntentConfirm
		analysis.ConfirmedStatement = userMessage
	}

	// Extract confidence score
	if confidenceStart := strings.Index(response, `"confidence":`); confidenceStart >= 0 {
		confidenceStart += len(`"confidence":`)
		if confidenceEnd := strings.Index(response[confidenceStart:], ","); confidenceEnd > 0 {
			confStr := strings.TrimSpace(response[confidenceStart : confidenceStart+confidenceEnd])
			var conf float64
			if _, err := fmt.Sscanf(confStr, "%f", &conf); err == nil {
				analysis.Confidence = conf
			}
		} else if confidenceEnd := strings.Index(response[confidenceStart:], "}"); confidenceEnd > 0 {
			confStr := strings.TrimSpace(response[confidenceStart : confidenceStart+confidenceEnd])
			var conf float64
			if _, err := fmt.Sscanf(confStr, "%f", &conf); err == nil {
				analysis.Confidence = conf
			}
		}
	}

	return analysis
}

// buildIntentHistoryContext creates a brief conversation context for intent detection
func buildIntentHistoryContext(history []models.Message) string {
	if len(history) == 0 {
		return "(Beginning of conversation)"
	}

	// Take last 6 messages (3 exchanges) for context
	start := 0
	if len(history) > 6 {
		start = len(history) - 6
	}

	var context strings.Builder
	for _, msg := range history[start:] {
		role := "User"
		if msg.Role == "assistant" || msg.Role == "moly" {
			role = "Moly"
		}
		context.WriteString(fmt.Sprintf("%s: %s\n", role, msg.Content))
	}

	return context.String()
}

// extractReactionContext tries to extract what the user is reacting to
func extractReactionContext(msg string) string {
	msg = strings.ToLower(strings.TrimSpace(msg))

	if strings.Contains(msg, "that") || strings.Contains(msg, "what you said") {
		return "previous_moly_statement"
	}
	if strings.Contains(msg, "this") {
		return "recent_context"
	}

	return "previous_message"
}

// ResponseType - The type of response to generate
type ResponseType string

const (
	ResponseDirectAnswer    ResponseType = "direct_answer"    // Answer their question directly
	ResponseAcknowledgement ResponseType = "acknowledgement"   // Acknowledge what they shared
	ResponseDeepeningQ      ResponseType = "deepening_q"      // Acknowledgement + Socratic question
	ResponseClarification   ResponseType = "clarification"    // Clarify what they meant
	ResponseValidation      ResponseType = "validation"       // Validate their feelings
	ResponseConfirmation    ResponseType = "confirmation"     // Confirm understanding
)

// RouteResponse determines what type of response to generate
// Based on intent and context
func RouteResponse(intent Intent, shouldDeepen bool) ResponseType {
	switch intent {
	case IntentAsk:
		return ResponseDirectAnswer
	case IntentShare:
		if shouldDeepen {
			return ResponseDeepeningQ
		}
		// When not deepening (insufficient context), ask clarification instead of just acknowledging
		return ResponseClarification
	case IntentReact:
		return ResponseClarification
	case IntentVent:
		return ResponseValidation
	case IntentConfirm:
		return ResponseConfirmation
	default:
		// When intent is unknown, ask clarification instead of just acknowledging
		return ResponseClarification
	}
}
