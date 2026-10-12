package agents

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	"moly/models"
	"moly/tools"
)

// Intent - What is the user doing with this message
type Intent string

const (
	IntentGreet   Intent = "greeting"   // User greets Moly
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
	ConfirmedStatement string  // what they're confirming (if Intent=="confirming")
	AnswersQuestion    bool    // the message only answers the question Moly just asked and states no aim of its own
	AdjustsDraft       bool    // the message asks to change, or comments on, a text Moly wrote for the user to send
}

// LLMIntentDetector uses LLM reasoning for intent detection
type LLMIntentDetector struct {
	llmClient    tools.LLMProvider
	constitution *models.Constitution
	db           *sql.DB // Database connection for saving analysis results
}

// NewLLMIntentDetector creates a new LLM-based intent detector
func NewLLMIntentDetector(llm tools.LLMProvider) *LLMIntentDetector {
	return &LLMIntentDetector{llmClient: llm}
}

// SetConstitution injects the loaded constitution (for principle-based prompts)
func (lid *LLMIntentDetector) SetConstitution(c *models.Constitution) {
	lid.constitution = c
}

// SetDatabase injects the database connection for saving analysis results
func (lid *LLMIntentDetector) SetDatabase(db *sql.DB) {
	lid.db = db
}

// DetectIntentWithLLM performs LLM-driven intent analysis
func (lid *LLMIntentDetector) DetectIntentWithLLM(userMessage string, conversationHistory []models.Message) IntentAnalysis {
	msg := strings.TrimSpace(userMessage)
	if msg == "" {
		return IntentAnalysis{Intent: IntentUnknown, Confidence: 0}
	}

	if lid.llmClient == nil {
		log.Printf("[IntentDetector] No LLM available, cannot detect intent")
		return IntentAnalysis{Intent: IntentUnknown, Confidence: 0}
	}

	log.Printf("[IntentDetector] Analyzing message intent with LLM")

	// Build conversation context for the LLM
	historyContext := buildIntentHistoryContext(conversationHistory)

	// LLM prompt to detect intent
	systemPrompt := `You are an intent analyzer. Analyze what the user is doing in their message.

Respond with ONLY a JSON object (no markdown, no explanation):
{
  "intent": "greeting|asking|sharing|reacting|venting|confirming|unknown",
  "confidence": 0.0-1.0,
  "answersQuestion": true|false,
  "adjustsDraft": true|false,
  "reasoning": "brief explanation of why"
}

Intent definitions:
- "greeting": User only greets Moly or opens the conversation with no request and no information (for example "Hello Moly", "Hi there"). A greeting that also states a request or a situation is NOT a greeting.
- "asking": User asks Moly a question or requests help/advice/assistance. Includes: "help me write", "how do I", "should I", "can you help", "I need help with", etc.
- "sharing": User provides information, context, experiences, clarifications, details, or answers to previous questions. Includes: describing relationships, providing profile info, sharing preferences, explaining situations
- "reacting": User responds directly to something Moly just said (agreement, disagreement, correction, follow-up to Moly's question)
- "venting": User expresses strong emotion (frustration, anger, fear, anxiety, sadness), even when the message also says what happened. Feelings stated as the point of the message ("I am so angry at my brother I could scream") are venting, not sharing.
- "confirming": User confirms, corrects, or clarifies their previous statement
- "unknown": No clear intent can be determined

"answersQuestion" is true ONLY when Moly's last message asked the user a question and this message does nothing except answer it (for example giving a requested name or detail) and states no request, aim or new situation of its own. It is false when there is no earlier question from Moly, or when the message adds a request, an aim or a new topic. A message that answers AND asks for something more is false, however short the request is (for example \"Her name is Dana, and also help me write to my brother\": false).

"adjustsDraft" is true ONLY when Moly's last message was a text written for the user to send (a draft) and this message asks to change it or comments on it (for example to make it shorter, warmer or more formal, or says what is wrong with it). It is false when Moly's last message was a question or an explanation, or when there is no draft.

Guidelines:
- "help me X" / "I want to write" / "don't know what to say" = ASKING (help-seeking)
- Detailed description of situation/relationship/context = SHARING, unless the user states strong feeling about it (furious, devastated, terrified): then VENTING
- Starting a new message after previous message = likely SHARING information
- Only use "unknown" if the message is truly ambiguous or unclear (rare)

Be generous: prefer a real intent over "unknown".
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
	analysis := lid.parseIntentResponseWithLLM(resp.Content, msg)
	log.Printf("[IntentDetector] Detected %s (confidence=%.2f)", analysis.Intent, analysis.Confidence)

	return analysis
}

// DetectIntentWithKnownContacts performs intent analysis with knowledge of known contacts
// This helps ensure pronouns are correctly interpreted and relationship types are accurate
func (lid *LLMIntentDetector) DetectIntentWithKnownContacts(userMessage string, conversationHistory []models.Message, knownContacts []*models.Contact) IntentAnalysis {
	if lid.llmClient == nil {
		log.Printf("[IntentDetector] No LLM available, cannot detect intent with contacts")
		return IntentAnalysis{Intent: IntentUnknown, Confidence: 0}
	}

	log.Printf("[IntentDetector] Analyzing message intent with %d known contacts", len(knownContacts))

	msg := strings.TrimSpace(userMessage)
	if msg == "" {
		return IntentAnalysis{Intent: IntentUnknown, Confidence: 0}
	}

	// Build conversation context and contact context
	historyContext := buildIntentHistoryContext(conversationHistory)
	contactsContext := ""
	if len(knownContacts) > 0 {
		contactsContext = "\nKnown contacts in this conversation:\n"
		for i, c := range knownContacts {
			if i >= 5 { // Limit to avoid token explosion
				break
			}
			relationship := c.Relationship
			if relationship == "" {
				relationship = "unspecified"
			}
			contactsContext += fmt.Sprintf("- %s (%s)\n", c.Name, relationship)
		}
	}

	// LLM prompt with contact context
	systemPrompt := `You are an intent analyzer. Analyze what the user is doing in their message.

Respond with ONLY a JSON object (no markdown, no explanation):
{
  "intent": "greeting|asking|sharing|reacting|venting|confirming|unknown",
  "confidence": 0.0-1.0,
  "reasoning": "brief explanation of why"
}

Intent definitions:
- "greeting": User only greets Moly or opens the conversation with no request and no information (for example "Hello Moly", "Hi there"). A greeting that also states a request or a situation is NOT a greeting.
- "asking": User asks Moly a question or requests information/advice
- "sharing": User provides information, context, experiences, or answers to previous questions
- "reacting": User responds directly to something Moly just said (agreement, disagreement, correction)
- "venting": User expresses strong emotion (frustration, anger, fear, anxiety, sadness)
- "confirming": User confirms, corrects, or clarifies their previous statement
- "unknown": No clear intent can be determined

CONTACT CONTEXT: Use the known contacts to properly interpret pronouns and relationship references.
If the user mentions "he" or "she" and you know their relationship type, consider that context.

Be generous with "sharing" - if user provides information, priorities, goals, or answers to implied questions, that's sharing.
Be specific with "reacting" - only if responding directly to Moly's words.
Use high confidence (0.8+) when intent is clear. Use medium (0.5-0.8) when there are mixed signals.`

	userPrompt := fmt.Sprintf(`User message: "%s"

Recent conversation context:
%s%s

What is the user's intent in this message?`, msg, historyContext, contactsContext)

	req := &tools.LLMRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Temperature:  0.3,
		MaxTokens:    200,
	}

	resp, err := lid.llmClient.Call(context.Background(), req)
	if err != nil {
		log.Printf("[IntentDetector] LLM call failed: %v, falling back to basic intent detection", err)
		return IntentAnalysis{Intent: IntentUnknown, Confidence: 0}
	}

	// Parse LLM response
	analysis := lid.parseIntentResponseWithLLM(resp.Content, msg)
	log.Printf("[IntentDetector] Detected %s (confidence=%.2f) with contact context", analysis.Intent, analysis.Confidence)

	return analysis
}

// buildPrincipleContext dynamically builds principle definitions from Constitution
func (lid *LLMIntentDetector) buildPrincipleContext() string {
	if lid.constitution == nil || len(lid.constitution.SupremePrinciples) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Constitutional principles:\n")

	// Include key principles for reaction context analysis
	relevantPrinciples := []string{"transparency", "stakeholder_consideration", "user_autonomy"}

	for _, princID := range relevantPrinciples {
		for _, principle := range lid.constitution.SupremePrinciples {
			if principle.ID == princID {
				sb.WriteString(fmt.Sprintf("- %s: %s\n", principle.Name, principle.Description))
				break
			}
		}
	}

	return sb.String()
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
		if msg.Role == "assistant" {
			role = "Moly"
		}
		context.WriteString(fmt.Sprintf("%s: %s\n", role, msg.Content))
	}

	return context.String()
}

// parseIntentResponseWithLLM - LLM-based intent parsing with reaction context detection
func (lid *LLMIntentDetector) parseIntentResponseWithLLM(llmResponse string, userMessage string) IntentAnalysis {
	analysis := IntentAnalysis{Intent: IntentUnknown, Confidence: 0.0}

	// Try to find the intent classification
	response := strings.ToLower(strings.TrimSpace(llmResponse))

	// First try JSON parsing for better reliability
	type intentResponse struct {
		Intent          string            `json:"intent"`
		Confidence      float64           `json:"confidence"`
		AnswersQuestion tools.LenientBool `json:"answersQuestion"`
		AdjustsDraft    tools.LenientBool `json:"adjustsDraft"`
	}

	var parsedResp intentResponse
	if err := tools.SafeJSONParse("IntentDetector", []byte(response), &parsedResp); err == nil {
		// Successfully parsed JSON - use the structured response
		intent := strings.ToLower(parsedResp.Intent)
		switch intent {
		case "greeting":
			analysis.Intent = IntentGreet
		case "asking":
			analysis.Intent = IntentAsk
			analysis.QuestionAsked = userMessage
		case "sharing":
			analysis.Intent = IntentShare
			analysis.InfoShared = userMessage
		case "reacting":
			analysis.Intent = IntentReact
		case "venting":
			analysis.Intent = IntentVent
			analysis.Emotional = true
		case "confirming":
			analysis.Intent = IntentConfirm
			analysis.ConfirmedStatement = userMessage
		case "unknown":
			analysis.Intent = IntentUnknown
		default:
			// Unrecognized intent, keep as unknown
			log.Printf("[IntentDetector] Unrecognized intent in response: %s", intent)
			analysis.Intent = IntentUnknown
		}

		analysis.AnswersQuestion = bool(parsedResp.AnswersQuestion)
		analysis.AdjustsDraft = bool(parsedResp.AdjustsDraft)

		// Use confidence from JSON if it's reasonable
		if parsedResp.Confidence >= 0.0 && parsedResp.Confidence <= 1.0 {
			analysis.Confidence = parsedResp.Confidence
		}
		return analysis
	}

	// An answer that is not JSON is not read for words: the intent is unknown, with no confidence.
	log.Printf("[IntentDetector] The answer is not readable JSON: intent unknown")
	return analysis
}
// ResponseType - The type of response to generate
type ResponseType string

const (
	ResponseGreeting        ResponseType = "greeting"        // Simple greeting acknowledgment
	ResponseDirectAnswer    ResponseType = "direct_answer"   // Answer their question directly
	ResponseAcknowledgement ResponseType = "acknowledgement" // Acknowledge what they shared
	ResponseDeepeningQ      ResponseType = "deepening_q"     // Acknowledgement + Socratic question
	ResponseClarification   ResponseType = "clarification"   // Clarify what they meant
	ResponseValidation      ResponseType = "validation"      // Validate their feelings
	ResponseConfirmation    ResponseType = "confirmation"    // Confirm understanding
)

// RouteResponse determines what type of response to generate
// Based on intent and context
func RouteResponse(intent Intent, shouldDeepen bool) ResponseType {
	switch intent {
	case IntentGreet:
		return ResponseGreeting
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

// jsonFlags reads a model answer that is a JSON object of true/false flags. A key that is missing, or an answer that is not
// readable JSON, is false. (The flags used to be found by searching the text for `"key": true`.)
func jsonFlags(label, content string) map[string]bool {
	var raw map[string]tools.LenientBool
	flags := map[string]bool{}
	if err := tools.SafeJSONParse(label, []byte(content), &raw); err != nil {
		return flags
	}
	for k, v := range raw {
		flags[k] = bool(v)
	}
	return flags
}
