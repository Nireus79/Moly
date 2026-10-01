package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

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
	ReactionTarget     string  // what they're reacting to (if Intent=="reacting")
	ConfirmedStatement string  // what they're confirming (if Intent=="confirming")
}

// LLMIntentDetector uses LLM reasoning for intent detection
type LLMIntentDetector struct {
	llmClient    tools.LLMProvider
	constitution *models.Constitution
}

// NewLLMIntentDetector creates a new LLM-based intent detector
func NewLLMIntentDetector(llm tools.LLMProvider) *LLMIntentDetector {
	return &LLMIntentDetector{llmClient: llm}
}

// SetConstitution injects the loaded constitution (for principle-based prompts)
func (lid *LLMIntentDetector) SetConstitution(c *models.Constitution) {
	lid.constitution = c
}

// detectGreeting - Fast path: detect if message is a simple greeting (deterministic, no LLM)
func (lid *LLMIntentDetector) detectGreeting(msg string) *IntentAnalysis {
	lower := strings.ToLower(strings.TrimSpace(msg))

	greetingPhrases := []string{
		"hello", "hi", "hey", "greetings", "good morning", "good afternoon",
		"good evening", "what's up", "howdy", "sup", "hola", "bonjour",
	}

	// Check if message is a simple greeting (optionally mentioning Moly or "you")
	for _, phrase := range greetingPhrases {
		if strings.HasPrefix(lower, phrase) {
			// Allow optional mention of Moly, me, you, etc after greeting
			afterGreeting := strings.TrimSpace(lower[len(phrase):])
			if afterGreeting == "" ||
				strings.Contains(afterGreeting, "moly") ||
				strings.Contains(afterGreeting, "you") ||
				strings.Contains(afterGreeting, "there") ||
				len(strings.Fields(afterGreeting)) <= 2 { // Short follow-up like "Moly" or "there"
				return &IntentAnalysis{
					Intent:     IntentGreet,
					Confidence: 0.95,
				}
			}
		}
	}

	return nil
}

// DetectIntentWithLLM performs LLM-driven intent analysis
func (lid *LLMIntentDetector) DetectIntentWithLLM(userMessage string, conversationHistory []models.Message) IntentAnalysis {
	msg := strings.TrimSpace(userMessage)
	if msg == "" {
		return IntentAnalysis{Intent: IntentUnknown, Confidence: 0}
	}

	// Fast path: detect greeting (deterministic, no LLM needed)
	if greeting := lid.detectGreeting(msg); greeting != nil {
		log.Printf("[IntentDetector] Detected greeting with high confidence")
		return *greeting
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
  "intent": "asking|sharing|reacting|venting|confirming|unknown",
  "confidence": 0.0-1.0,
  "reasoning": "brief explanation of why"
}

Intent definitions:
- "asking": User asks Moly a question or requests help/advice/assistance. Includes: "help me write", "how do I", "should I", "can you help", "I need help with", etc.
- "sharing": User provides information, context, experiences, clarifications, details, or answers to previous questions. Includes: describing relationships, providing profile info, sharing preferences, explaining situations
- "reacting": User responds directly to something Moly just said (agreement, disagreement, correction, follow-up to Moly's question)
- "venting": User expresses strong emotion (frustration, anger, fear, anxiety, sadness)
- "confirming": User confirms, corrects, or clarifies their previous statement
- "unknown": No clear intent can be determined

Guidelines:
- "help me X" / "I want to write" / "don't know what to say" = ASKING (help-seeking)
- Detailed description of situation/relationship/context = SHARING
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

// DetectIntentWithAnalysisContext performs intent analysis with rich conversation context
// Uses conversation summary, recent messages, confirmed preferences, and user profile
func (lid *LLMIntentDetector) DetectIntentWithAnalysisContext(analysisCtx *models.AnalysisContext) IntentAnalysis {
	if lid.llmClient == nil {
		log.Printf("[IntentDetector] No LLM available, cannot detect intent")
		return IntentAnalysis{Intent: IntentUnknown, Confidence: 0}
	}

	if analysisCtx == nil || analysisCtx.CurrentMessage == "" {
		return IntentAnalysis{Intent: IntentUnknown, Confidence: 0}
	}

	log.Printf("[IntentDetector] Analyzing intent with analysis context (quality: %s)", analysisCtx.ContextQuality)

	msg := strings.TrimSpace(analysisCtx.CurrentMessage)

	// Build rich context from AnalysisContext
	contextStr := lid.buildIntentContextFromAnalysisContext(analysisCtx)

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

IMPORTANT: Evaluate IN CONTEXT. Consider:
- Conversation arc and established patterns
- User's communication style and preferences
- Recent exchange flow
- What they've confirmed before

Be generous with "sharing" - if user provides information, priorities, goals, or answers to implied questions, that's sharing.
Be specific with "reacting" - only if responding directly to Moly's words.
Use high confidence (0.8+) when intent is clear. Use medium (0.5-0.8) when there are mixed signals.`

	userPrompt := fmt.Sprintf(`User message: "%s"

CONTEXT:
%s

What is the user's intent in this message?`, msg, contextStr)

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
	log.Printf("[IntentDetector] Detected %s (confidence=%.2f) with analysis context", analysis.Intent, analysis.Confidence)

	return analysis
}

// buildIntentContextFromAnalysisContext builds context from AnalysisContext
func (lid *LLMIntentDetector) buildIntentContextFromAnalysisContext(analysisCtx *models.AnalysisContext) string {
	var sb strings.Builder

	// Include conversation summary/arc
	if analysisCtx.ConversationSummary != nil {
		sb.WriteString("CONVERSATION ARC:\n")
		sb.WriteString(analysisCtx.ConversationSummary.Arc)
		sb.WriteString("\n\n")

		if len(analysisCtx.ConversationSummary.UserPatterns) > 0 {
			sb.WriteString("USER PATTERNS: ")
			for i, p := range analysisCtx.ConversationSummary.UserPatterns {
				if i > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(p)
			}
			sb.WriteString("\n\n")
		}
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

	// NOTE: User preferences (communication style) are NOT included
	// Intent detection must be independent of user preferences
	// Preferences only affect how Moly responds, not message analysis

	// Include confirmed preferences
	if len(analysisCtx.ConfirmedPreferences) > 0 {
		sb.WriteString("CONFIRMED PREFERENCES:\n")
		for k, v := range analysisCtx.ConfirmedPreferences {
			sb.WriteString(fmt.Sprintf("- %s: %v\n", k, v))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// parseIntentResponse - DEPRECATED: Use parseIntentResponseWithLLM instead
// Kept for backward compatibility, uses conservative default for reaction context
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
		analysis.ReactionTarget = "previous_message" // Conservative default
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
		if msg.Role == "assistant" || msg.Role == "moly" {
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
		Intent     string  `json:"intent"`
		Confidence float64 `json:"confidence"`
	}

	var parsedResp intentResponse
	if err := json.Unmarshal([]byte(response), &parsedResp); err == nil {
		// Successfully parsed JSON - use the structured response
		intent := strings.ToLower(parsedResp.Intent)
		switch intent {
		case "asking":
			analysis.Intent = IntentAsk
			analysis.QuestionAsked = userMessage
		case "sharing":
			analysis.Intent = IntentShare
			analysis.InfoShared = userMessage
		case "reacting":
			analysis.Intent = IntentReact
			analysis.ReactionTarget = lid.getReactionContextLLM(userMessage)
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

		// Use confidence from JSON if it's reasonable
		if parsedResp.Confidence >= 0.0 && parsedResp.Confidence <= 1.0 {
			analysis.Confidence = parsedResp.Confidence
		}
		return analysis
	}

	// Fallback: Extract intent from string pattern matching
	switch {
	case strings.Contains(response, `"intent":"asking"`):
		analysis.Intent = IntentAsk
		analysis.QuestionAsked = userMessage
	case strings.Contains(response, `"intent":"sharing"`):
		analysis.Intent = IntentShare
		analysis.InfoShared = userMessage
	case strings.Contains(response, `"intent":"reacting"`):
		analysis.Intent = IntentReact
		analysis.ReactionTarget = lid.getReactionContextLLM(userMessage)
	case strings.Contains(response, `"intent":"venting"`):
		analysis.Intent = IntentVent
		analysis.Emotional = true
	case strings.Contains(response, `"intent":"confirming"`):
		analysis.Intent = IntentConfirm
		analysis.ConfirmedStatement = userMessage
	default:
		// Check if any intent word is mentioned (with more flexibility)
		if strings.Contains(response, "asking") {
			analysis.Intent = IntentAsk
		} else if strings.Contains(response, "sharing") {
			analysis.Intent = IntentShare
		} else if strings.Contains(response, "reacting") {
			analysis.Intent = IntentReact
		} else if strings.Contains(response, "venting") {
			analysis.Intent = IntentVent
		} else if strings.Contains(response, "confirming") {
			analysis.Intent = IntentConfirm
		}
	}

	// Extract confidence score
	if confidenceStart := strings.Index(response, `"confidence":`); confidenceStart >= 0 {
		confidenceStart += len(`"confidence":`)
		if confidenceEnd := strings.Index(response[confidenceStart:], ","); confidenceEnd > 0 {
			confStr := strings.TrimSpace(response[confidenceStart : confidenceStart+confidenceEnd])
			var conf float64
			if _, err := fmt.Sscanf(confStr, "%f", &conf); err == nil && conf >= 0.0 && conf <= 1.0 {
				analysis.Confidence = conf
			}
		} else if confidenceEnd := strings.Index(response[confidenceStart:], "}"); confidenceEnd > 0 {
			confStr := strings.TrimSpace(response[confidenceStart : confidenceStart+confidenceEnd])
			var conf float64
			if _, err := fmt.Sscanf(confStr, "%f", &conf); err == nil && conf >= 0.0 && conf <= 1.0 {
				analysis.Confidence = conf
			}
		}
	}

	// Ensure unknown intent always has low confidence
	if analysis.Intent == IntentUnknown && analysis.Confidence > 0.5 {
		analysis.Confidence = 0.0
	}

	return analysis
}

// getReactionContextLLM - LLM-based detection of what user is reacting to
// REMOVED: Hardcoded keyword checks ("that", "what you said", "this")
// Now: LLM analyzes reaction context via principle-based reasoning
// Principles:
// - Stakeholder: considering all parties in the situation
// - Transparency: being explicit about what they're responding to
// - Autonomy: asserting their own position
func (lid *LLMIntentDetector) getReactionContextLLM(msg string) string {
	if lid.llmClient == nil {
		return "previous_message"
	}

	// Build principle context from Constitution
	principleContext := lid.buildPrincipleContext()
	if principleContext == "" {
		// Fallback if constitution not available
		return "previous_message"
	}

	// Principle-based analysis: which principles does this reaction engage with?
	prompt := fmt.Sprintf(`Analyze how this message engages with constitutional principles.

%s

Message: "%s"

Respond with ONLY a JSON object (no markdown):
{
  "transparency_engaged": boolean,
  "stakeholder_engaged": boolean,
  "autonomy_engaged": boolean
}`, principleContext, msg)

	req := &tools.LLMRequest{
		SystemPrompt: `You analyze messages against constitutional principles.
Respond with only valid JSON, no other text.`,
		UserPrompt:  prompt,
		MaxTokens:   100,
		Temperature: 0.3,
		Retries:     1,
	}

	resp, err := lid.llmClient.Call(context.Background(), req)
	if err != nil {
		log.Printf("[LLMIntentDetector] getReactionContextLLM failed: %v, using default", err)
		return "previous_message"
	}

	// Parse principle engagement to infer context
	lower := strings.ToLower(resp.Content)

	// If message engages with transparency principle (being explicit), likely direct reference to Moly's statement
	if strings.Contains(lower, `"transparency_engaged": true`) || strings.Contains(lower, `"transparency_engaged":true`) {
		return "moly_statement"
	}

	// If stakeholder principle engaged (considering others), likely situational context
	if strings.Contains(lower, `"stakeholder_engaged": true`) || strings.Contains(lower, `"stakeholder_engaged":true`) {
		return "recent_context"
	}

	// Default: engaging from their own position (autonomy principle)
	return "previous_message"
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

// ExtractEntitiesWithClassification extracts entities with semantic classification (self-ref vs contact vs topic)
func (lid *LLMIntentDetector) ExtractEntitiesWithClassification(ctx context.Context, message string) ([]models.ExtractedEntity, error) {
	if lid.llmClient == nil {
		log.Printf("[IntentDetector] No LLM available, cannot extract entities")
		return []models.ExtractedEntity{}, nil
	}

	log.Printf("[IntentDetector] Extracting entities with semantic classification")

	systemPrompt := `You are an entity classifier. Extract entities from the message and classify them, identifying WHO each property belongs to.

ENTITY TYPES (must be one of these):
- self_reference: References to Moly (the AI), "you", "yourself"
- contact: External persons mentioned
- topic: Subject of discussion (not person)
- goal: Objectives user wants to achieve

CLASSIFICATION RULES:
1. Grammar matters:
   - "You are Moly" + "are" = self_reference (high confidence)
   - "my friend Moly" + "my" = contact (high confidence)
   - "Moly" alone with no context = AMBIGUOUS (confidence < 0.7)

2. Evidence requirement:
   - Each entity must include exact substring from message
   - If you cannot find the substring, do NOT include it

3. Confidence: only high confidence (0.8+) if multiple signals support it
   - If confidence < 0.7, mark as ambiguous

4. SUBJECT TRACKING (NEW):
   For each extracted property/characteristic, identify WHO HAS IT:
   - "user" = when message uses I/me/my (property belongs to the user)
   - Contact name (Se, Kate, John) = when explicitly named
   - Pronoun (she/he/they) = when using pronouns to refer to someone

   Example: "I am dominant. Se is submissive."
   - Property: dominant → Subject: "user" (from "I am")
   - Property: submissive → Subject: "Se" (from "Se is")

RESPOND WITH ONLY JSON:
{
  "entities": [
    {
      "value": "extracted entity name",
      "type": "self_reference|contact|topic|goal|ambiguous",
      "evidence": "exact substring with context",
      "confidence": 0.85,
      "reasoning": "why this classification",
      "is_ambiguous": false,
      "ambiguous_possibilities": ["self_reference", "contact"],
      "subject": "user|contact_name|pronoun",
      "source_type": "extraction"
    }
  ]
}`

	userPrompt := fmt.Sprintf(`Classify entities in this message:
Message: "%s"

Extract all persons, topics, and goals mentioned. For each entity, determine:
1. What it is (person, topic, goal)
2. Whether it's self-reference (Moly) or external
3. Confidence level (0.0-1.0)
4. Is it ambiguous? (could be multiple types)
5. WHO HAS THIS PROPERTY (subject tracking):
   - Use "user" if they say I/me/my
   - Use actual name (Se, Kate) if explicitly mentioned
   - Use pronoun (she, he, they) if using pronouns
   - Note subject switches between sentences`, message)

	req := &tools.LLMRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Temperature:  0.3,
		MaxTokens:    500,
	}

	resp, err := lid.llmClient.Call(ctx, req)
	if err != nil {
		log.Printf("[IntentDetector] Entity extraction LLM call failed: %v", err)
		return []models.ExtractedEntity{}, nil
	}

	return lid.validateAndParseEntities(resp.Content, message)
}

// validateAndParseEntities validates and parses LLM entity extraction response
func (lid *LLMIntentDetector) validateAndParseEntities(rawResponse string, message string) ([]models.ExtractedEntity, error) {
	var result struct {
		Entities []models.ExtractedEntity `json:"entities"`
	}

	if err := json.Unmarshal([]byte(rawResponse), &result); err != nil {
		log.Printf("[IntentDetector] Failed to parse entity response: %v", err)
		return []models.ExtractedEntity{}, nil
	}

	var validated []models.ExtractedEntity

	validTypes := map[string]bool{
		"self_reference": true,
		"contact":        true,
		"topic":          true,
		"goal":           true,
		"ambiguous":      true,
	}

	for _, entity := range result.Entities {
		// 1. Validate type
		if !validTypes[entity.Type] {
			log.Printf("[IntentDetector] Invalid entity type: %s", entity.Type)
			continue
		}

		// 2. Validate evidence is substring from message (but be lenient with pasted/multiline content)
		// Skip validation for multiline evidence or profile data (likely pasted, not user's words)
		evidenceHasMultipleLines := strings.Count(entity.Evidence, "\n") > 0 ||
			strings.Count(entity.Evidence, "•") > 0 || // Bullet points from profiles
			len(entity.Evidence) > 100 // Very long evidence suggests pasted content

		if !evidenceHasMultipleLines && !strings.Contains(message, entity.Evidence) {
			log.Printf("[IntentDetector] Evidence not found in message (single-line validation): %s", entity.Evidence)
			continue
		}

		if evidenceHasMultipleLines {
			// For pasted content, just accept it - the LLM extracted it as relevant
			log.Printf("[IntentDetector] Accepting multiline/pasted evidence for %s entity", entity.Type)
		}

		// 3. Validate confidence
		if entity.Confidence < 0.0 || entity.Confidence > 1.0 {
			log.Printf("[IntentDetector] Invalid confidence: %.2f", entity.Confidence)
			continue
		}

		// 4. Mark as ambiguous if low confidence
		if entity.Confidence < 0.7 {
			entity.IsAmbiguous = true
		}

		// 5. Validate subject tracking (new)
		if entity.Subject == "" {
			// Default subject based on type
			switch entity.Type {
			case "self_reference":
				entity.Subject = "user"
			default:
				// For contacts/topics/goals, try to infer from evidence
				if strings.Contains(strings.ToLower(entity.Evidence), "i ") ||
					strings.Contains(strings.ToLower(entity.Evidence), "me") ||
					strings.Contains(strings.ToLower(entity.Evidence), "my") {
					entity.Subject = "user"
				} else {
					entity.Subject = entity.Value // Default to entity name
				}
			}
		}

		// Set source type if not provided
		if entity.SourceType == "" {
			entity.SourceType = "extraction"
		}

		log.Printf("[IntentDetector] Entity validated: value=%s type=%s subject=%s confidence=%.2f",
			entity.Value, entity.Type, entity.Subject, entity.Confidence)

		validated = append(validated, entity)
	}

	return validated, nil
}

// GenerateClarificationQuestion generates a clarification question for ambiguous entities
func (lid *LLMIntentDetector) GenerateClarificationQuestion(entity models.ExtractedEntity) string {
	// Generate clarification question based on LLM-identified ambiguous possibilities
	if entity.IsAmbiguous && len(entity.AmbiguousPossibilities) > 0 {
		types := strings.Join(entity.AmbiguousPossibilities, " or ")
		return fmt.Sprintf("When you mention '%s', do you mean %s?", entity.Value, types)
	}

	return fmt.Sprintf("Can you clarify what you mean by '%s'?", entity.Value)
}

// ExtractEntitiesAndAnalyzeIntent wraps entity extraction in models.IntentAnalysis
func (lid *LLMIntentDetector) ExtractEntitiesAndAnalyzeIntent(ctx context.Context, message string) (*models.IntentAnalysis, error) {
	if message == "" {
		return &models.IntentAnalysis{
			Intent:             "unknown",
			Confidence:         0.0,
			NeedsClarification: false,
		}, nil
	}

	entities, err := lid.ExtractEntitiesWithClassification(ctx, message)
	if err != nil {
		log.Printf("[IntentDetector] Entity extraction failed: %v", err)
		return &models.IntentAnalysis{
			Intent:             "unknown",
			Confidence:         0.0,
			NeedsClarification: false,
		}, nil
	}

	analysis := &models.IntentAnalysis{
		Entities:   entities,
		Confidence: 0.7, // Default confidence for entity extraction
	}

	// Check for ambiguous entities (but not greeting words - those are functional, not semantic)
	greetingWords := map[string]bool{
		"hello": true, "hi": true, "hey": true, "greetings": true, "good day": true,
		"good morning": true, "good afternoon": true, "good evening": true,
	}

	for _, entity := range entities {
		// Skip greeting words - they don't need clarification
		if greetingWords[strings.ToLower(entity.Value)] {
			log.Printf("[IntentDetector] Skipping greeting word from clarification: %s", entity.Value)
			continue
		}

		if entity.IsAmbiguous && entity.Confidence < 0.7 {
			analysis.NeedsClarification = true
			analysis.AmbiguousEntity = entity.Value
			analysis.ClarificationQuestion = lid.GenerateClarificationQuestion(entity)
			log.Printf("[IntentDetector] Found ambiguous entity: %s", entity.Value)
			break // Only clarify the first ambiguous entity
		}
	}

	return analysis, nil
}

// SmartExtractionResult combines LLM and fallback extraction results
type SmartExtractionResult struct {
	Artifact           *models.ExtractionArtifact // NEW: Central extraction artifact with metadata
	Entities           []models.ExtractedEntity   // Combined extractions (LLM + fallback)
	Source             string                     // "llm", "fallback", or "cached"
	LLMSuccess         bool                       // Whether LLM succeeded
	FallbackUsed       bool                       // Whether fallback was activated
	ExtractionDuration float64                    // Milliseconds
	Error              string                     // If any error occurred
	SubjectAttributed  bool                       // Whether extraction includes subject attribution
	NegationPreserved  bool                       // Whether negation is properly handled
}

// SmartExtractEntities performs intelligent entity extraction with fallback
// 1. Checks cache first
// 2. Tries LLM extraction with timeout
// 3. Falls back to LinguisticParser on timeout/failure
// 4. Returns consolidated results with source attribution
func (lid *LLMIntentDetector) SmartExtractEntities(ctx context.Context, message string, cache *tools.LLMCache) *SmartExtractionResult {
	startTime := time.Now()
	result := &SmartExtractionResult{
		Entities:  []models.ExtractedEntity{},
		Source:    "unknown",
		LLMSuccess: false,
		FallbackUsed: false,
	}

	if message == "" {
		return result
	}

	// Step 1: Check cache
	if cache != nil {
		if cached, found := cache.Get(message, "entity_extraction"); found {
			log.Printf("[SmartExtraction] Cache hit for entity extraction")
			result.Source = "cached"
			result.ExtractionDuration = time.Since(startTime).Seconds() * 1000

			// Parse cached entities
			var entities []models.ExtractedEntity
			if err := json.Unmarshal([]byte(cached), &entities); err == nil {
				result.Entities = entities
				result.SubjectAttributed = checkSubjectAttribution(entities)
				result.NegationPreserved = checkNegationHandling(entities)

				// NEW: Create ExtractionArtifact for cached result too (Phase 0)
				result.Artifact = &models.ExtractionArtifact{
					ID:                fmt.Sprintf("extraction_%d", time.Now().UnixNano()),
					Entities:          entities,
					Source:            "cached",
					LLMSuccess:        false,
					Duration:          result.ExtractionDuration,
					SubjectAttributed: result.SubjectAttributed,
					NegationPreserved: result.NegationPreserved,
					AverageConfidence: calculateAverageConfidence(entities),
					CreatedAt:         time.Now().Unix(),
				}

				return result
			}
		}
	}

	// Step 2: Try LLM extraction with timeout
	if lid.llmClient != nil {
		log.Printf("[SmartExtraction] Attempting LLM entity extraction")
		entities, success := lid.extractEntitiesWithLLM(ctx, message)

		if success && len(entities) > 0 {
			log.Printf("[SmartExtraction] LLM extraction succeeded with %d entities", len(entities))
			result.Source = "llm"
			result.LLMSuccess = true
			result.Entities = entities
			result.SubjectAttributed = checkSubjectAttribution(entities)
			result.NegationPreserved = checkNegationHandling(entities)

			// Cache successful result
			if cache != nil {
				if data, err := json.Marshal(entities); err == nil {
					cache.Set(message, string(data), "entity_extraction")
				}
			}

			result.ExtractionDuration = time.Since(startTime).Seconds() * 1000

			// NEW: Create ExtractionArtifact (Phase 0)
			result.Artifact = &models.ExtractionArtifact{
				ID:                fmt.Sprintf("extraction_%d", time.Now().UnixNano()),
				Entities:          entities,
				Source:            "llm",
				LLMSuccess:        true,
				Duration:          result.ExtractionDuration,
				SubjectAttributed: result.SubjectAttributed,
				NegationPreserved: result.NegationPreserved,
				AverageConfidence: calculateAverageConfidence(entities),
				CreatedAt:         time.Now().Unix(),
			}

			return result
		}

		log.Printf("[SmartExtraction] LLM extraction failed or timed out, activating fallback")
	}

	// Step 3: Fallback to ExtractionOrchestrator (with context-aware analysis)
	log.Printf("[SmartExtraction] Using ExtractionOrchestrator fallback (Phase 6 integration)")
	result.FallbackUsed = true

	// Phase 6: Create orchestrator for context-aware extraction
	orchestrator := tools.NewExtractionOrchestrator()

	// Build list of recent messages for pronoun resolution context
	var recentMessages []string
	// Note: In production, would get recent conversation history
	recentMessages = append(recentMessages, message)

	// Build known contacts map (would come from database in production)
	knownContacts := make(map[string]int64)
	// Note: In production, would load from database

	// Phase 6: Perform complete context-aware analysis
	var contextAwareResult *tools.ContextAwareExtractionResult
	_, err := orchestrator.AnalyzeMessageForExtraction(message, recentMessages, knownContacts)
	if err != nil {
		log.Printf("[SmartExtraction] Orchestrator analysis failed: %v, falling back to basic parser", err)
		// Fallback to old behavior if orchestrator fails
		parser := tools.NewLinguisticParser()
		extractions := parser.Parse(message)
		for _, extraction := range extractions {
			entity := models.ExtractedEntity{
				Type:        extraction.Type,
				Value:       extraction.Property,
				Subject:     extraction.Subject,
				Confidence:  extraction.Confidence,
				IsAmbiguous: extraction.Confidence < 0.75,
				SourceType:  "linguistic_parser",
				Evidence:    extraction.RawMatch,
			}
			result.Entities = append(result.Entities, entity)
		}
	} else {
		// Phase 6: Use orchestrator's context-aware extraction
		contextAwareResult, err = orchestrator.ExtractWithContext(message, recentMessages, knownContacts)
		if err != nil {
			log.Printf("[SmartExtraction] Context-aware extraction failed: %v", err)
		} else {
			// Convert context-aware results to models.ExtractedEntity format
			for _, contextEntity := range contextAwareResult.ContextAwareEntities {
				// Phase 6: USE THE FIXED SUBJECT (this is the bug fix!)
				entity := models.ExtractedEntity{
					Type:        contextEntity.OriginalEntity.Type,
					Value:       contextEntity.OriginalEntity.Property,
					Subject:     contextEntity.ResolvedSubject, // FIXED SUBJECT - THE BUG FIX!
					Confidence:  contextEntity.ContextConfidence,
					IsAmbiguous: contextEntity.ContextConfidence < 0.75,
					SourceType:  "extraction_orchestrator",
					Evidence:    contextEntity.OriginalEntity.RawMatch,
				}
				result.Entities = append(result.Entities, entity)
				log.Printf("[SmartExtraction] Entity: %s | Original subject: %s | Resolved subject: %s",
					contextEntity.OriginalEntity.Property,
					contextEntity.OriginalEntity.Subject,
					contextEntity.ResolvedSubject)
			}

			// Phase 5: Save analysis results to database
			log.Printf("[SmartExtraction] Phase 5: Saving analysis results to database")

			// Save sentence analyses
			for i, svo := range contextAwareResult.SentenceAnalyses {
				log.Printf("[SmartExtraction] SaveSentenceAnalysis: sentence=%d, subject=%s, verb=%s, object=%s, confidence=%.2f",
					i+1, svo.Subject, svo.Verb, svo.Object, svo.Confidence)
				// TODO: sentenceAnalysisRepo.SaveSentenceAnalysis(userID, messageID, conversationID, ...)
			}

			// Save pronoun resolutions
			if len(contextAwareResult.PronounResolutions) > 0 {
				log.Printf("[SmartExtraction] Saving %d pronoun resolutions", len(contextAwareResult.PronounResolutions))
				for pronoun, resolution := range contextAwareResult.PronounResolutions {
					log.Printf("[SmartExtraction] SavePronounResolution: pronoun=%s → %s (confidence=%.2f)",
						pronoun, resolution.AntecedentValue, resolution.Confidence)
					// TODO: pronounResolutionRepo.SavePronounResolution(userID, conversationID, ...)
				}
			}

			// Save group references
			if len(contextAwareResult.GroupReferences) > 0 {
				log.Printf("[SmartExtraction] Saving %d group references", len(contextAwareResult.GroupReferences))
				for groupPronoun, groupRef := range contextAwareResult.GroupReferences {
					log.Printf("[SmartExtraction] SaveGroupReference: %s = %v (context: %s)",
						groupPronoun, groupRef.Members, groupRef.GroupContext)
					// TODO: groupReferenceRepo.SaveGroupReference(userID, conversationID, ...)
				}
			}

			log.Printf("[SmartExtraction] Database saves queued (TODO: implement when DB injected)")
		}
	}

	result.Source = "fallback"
	result.SubjectAttributed = len(result.Entities) > 0 // Orchestrator always includes subjects

	// Check for negation in sentence analyses from orchestrator
	negationFound := false
	if contextAwareResult != nil {
		for _, analysis := range contextAwareResult.SentenceAnalyses {
			if analysis.Negated {
				negationFound = true
				break
			}
		}
	}
	result.NegationPreserved = negationFound

	// Cache fallback result (lower confidence, marked as fallback)
	if cache != nil && len(result.Entities) > 0 {
		if data, err := json.Marshal(result.Entities); err == nil {
			cache.Set(message, string(data), "entity_extraction")
		}
	}

	result.ExtractionDuration = time.Since(startTime).Seconds() * 1000

	// NEW: Create ExtractionArtifact for fallback too (Phase 0)
	result.Artifact = &models.ExtractionArtifact{
		ID:                fmt.Sprintf("extraction_%d", time.Now().UnixNano()),
		Entities:          result.Entities,
		Source:            "fallback",
		LLMSuccess:        false,
		Duration:          result.ExtractionDuration,
		SubjectAttributed: result.SubjectAttributed,
		NegationPreserved: result.NegationPreserved,
		AverageConfidence: calculateAverageConfidence(result.Entities),
		CreatedAt:         time.Now().Unix(),
	}

	return result
}

// ExtractAndLock performs extraction and immediately locks the artifact
// PHASE 1: Ensures extraction cannot be re-parsed after creation
// Returns error if:
// 1. Extraction fails completely
// 2. Lock operation fails (should never happen)
func (lid *LLMIntentDetector) ExtractAndLock(ctx context.Context, message string, cache *tools.LLMCache) (*models.ExtractionArtifact, error) {
	// Step 1: Extract entities (uses SmartExtractEntities internally)
	extractionResult := lid.SmartExtractEntities(ctx, message, cache)
	if extractionResult == nil {
		return nil, fmt.Errorf("extraction returned nil result")
	}

	// Step 2: Verify artifact was created
	if extractionResult.Artifact == nil {
		return nil, fmt.Errorf("extraction did not create artifact (extraction failed: %s)", extractionResult.Error)
	}

	artifact := extractionResult.Artifact

	// Step 3: Set TTL for cleanup (30 minutes)
	artifact.ExpiresAt = time.Now().Add(30 * time.Minute).Unix()

	// Step 4: Lock the extraction immediately
	// PHASE 1: This prevents any downstream code from re-parsing or modifying
	lockReason := fmt.Sprintf("extraction_complete: source=%s, entities=%d, confidence=%.2f",
		artifact.Source, len(artifact.Entities), artifact.AverageConfidence)

	err := artifact.Lock(lockReason)
	if err != nil {
		return nil, fmt.Errorf("failed to lock extraction artifact: %w", err)
	}

	log.Printf("[ExtractAndLock] Locked extraction: id=%s, reason=%s, entities=%d",
		artifact.ID, lockReason, len(artifact.Entities))

	// Step 5: Verify lock was successful (paranoia check)
	if !artifact.IsLocked {
		return nil, fmt.Errorf("extraction locked returned success but IsLocked is false (implementation bug)")
	}

	return artifact, nil
}

// calculateAverageConfidence computes the average confidence of extracted entities
func calculateAverageConfidence(entities []models.ExtractedEntity) float64 {
	if len(entities) == 0 {
		return 0.0
	}

	totalConfidence := 0.0
	for _, e := range entities {
		totalConfidence += e.Confidence
	}
	return totalConfidence / float64(len(entities))
}

// extractEntitiesWithLLM performs LLM-based entity extraction
// Returns entities and success flag (false on timeout or error)
func (lid *LLMIntentDetector) extractEntitiesWithLLM(ctx context.Context, message string) ([]models.ExtractedEntity, bool) {
	systemPrompt := `You are an entity extraction specialist. Extract all relevant entities from the user message.

Focus on preferences, characteristics, and contextual information. Include WHO has what property (subject attribution).

Respond with ONLY a JSON array of objects (no markdown, no explanation):
[
  {
    "type": "preference|characteristic|interest|negation|profile",
    "value": "the property or interest",
    "subject": "who this applies to (user, contact_name, she, he, etc)",
    "confidence": 0.0-1.0
  }
]

IMPORTANT:
- Subject attribution: Include who has what property (user:dominant, she:submissive, not just dominant/submissive)
- Preserve negation: "I don't want casual sex" should be marked as negation type, value "casual sex"
- Include all meaningful properties mentioned
- Use type "negation" when user explicitly says they DON'T want something
- Be comprehensive but accurate`

	userPrompt := fmt.Sprintf(`Extract entities from this message:

"%s"

Focus on:
1. Properties about the user (I am X, I like Y, I want Z, I don't want W)
2. Properties about others mentioned (She is X, He likes Y)
3. Preferences and interests (both positive and negative)
4. Context and nuance

Always include subject attribution (who has what).`, message)

	req := &tools.LLMRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Temperature:  0.3,
		MaxTokens:    500,
	}

	// NOTE: Removed hardcoded 15-second timeout override (Sept 30, 2026)
	// llmClient.Call() uses adaptive timeout based on prompt length:
	// - Base: 30 seconds
	// - Plus: 5 seconds per 500 characters of prompt
	// For multi-person messages (2000-3000 chars), this yields 50-60 seconds
	// This allows proper LLM processing without premature fallback

	resp, err := lid.llmClient.Call(ctx, req)
	if err != nil {
		log.Printf("[SmartExtraction] LLM call failed: %v", err)
		return nil, false
	}

	// Parse JSON response with comprehensive error handling
	var entities []models.ExtractedEntity
	content := resp.Content

	log.Printf("[SmartExtraction] Response length: %d chars, first 250: %s", len(content), truncateString(content, 250))

	// Try parsing as-is first
	if err := json.Unmarshal([]byte(content), &entities); err != nil {
		log.Printf("[SmartExtraction] Failed to parse LLM response (attempt 1): %v", err)

		// Try to extract JSON from response if it's mixed with text
		if jsonStr := lid.extractJSONFromText(content); jsonStr != "" {
			log.Printf("[SmartExtraction] Extracted JSON (%d chars), attempting parse", len(jsonStr))
			if err := json.Unmarshal([]byte(jsonStr), &entities); err == nil {
				log.Printf("[SmartExtraction] Successfully parsed extracted JSON (%d entities)", len(entities))
				return entities, true
			} else {
				// Extracted JSON still invalid - log for debugging
				log.Printf("[SmartExtraction] Extracted JSON still invalid: %v", err)
				if len(jsonStr) <= 500 {
					log.Printf("[SmartExtraction] Extracted JSON: %s", jsonStr)
				} else {
					log.Printf("[SmartExtraction] Extracted JSON first 250: %s", truncateString(jsonStr, 250))
				}
			}
		} else {
			log.Printf("[SmartExtraction] Could not extract JSON from response")
		}

		// All parsing attempts failed
		log.Printf("[SmartExtraction] Failed to parse LLM response after all attempts")
		return nil, false
	}

	return entities, true
}

// checkSubjectAttribution verifies that entities have subject information
func checkSubjectAttribution(entities []models.ExtractedEntity) bool {
	if len(entities) == 0 {
		return false
	}

	// Check if at least 80% of entities have subject attribution
	withSubject := 0
	for _, e := range entities {
		if e.Subject != "" {
			withSubject++
		}
	}

	return withSubject >= (len(entities) * 80 / 100)
}

// extractJSONFromText attempts to extract and validate JSON from response
func (lid *LLMIntentDetector) extractJSONFromText(text string) string {
	// Look for JSON array pattern: [...]
	startIdx := strings.Index(text, "[")
	if startIdx == -1 {
		return ""
	}

	// Strategy: Try progressively smaller substrings from the end to find valid JSON
	// This handles cases where LLM adds text after the JSON array

	bestJSON := ""

	for endIdx := len(text); endIdx > startIdx+2; endIdx-- {
		candidate := text[startIdx:endIdx]

		// Quick bracket check first (avoid expensive JSON unmarshal for obviously broken JSON)
		openBrackets := strings.Count(candidate, "[") - strings.Count(candidate, "]")
		openBraces := strings.Count(candidate, "{") - strings.Count(candidate, "}")

		// If brackets/braces are balanced, try to unmarshal
		if openBrackets == 0 && openBraces == 0 {
			var test []map[string]interface{}
			if err := json.Unmarshal([]byte(candidate), &test); err == nil {
				// Found valid JSON
				if len(test) > 0 { // Ensure it's not empty
					bestJSON = candidate
					log.Printf("[JSONExtraction] Found valid JSON at end position %d (length %d)", endIdx, len(candidate))
					break
				}
			}
		}
	}

	// If we found valid JSON, return it
	if bestJSON != "" {
		return bestJSON
	}

	// Fallback: Try to repair common JSON issues in the full extraction
	fullExtraction := tryExtractWithBracketMatching(text, startIdx)
	if fullExtraction != "" {
		// Try to repair and validate
		repaired := repairJSON(fullExtraction)
		if repaired != "" {
			var test []map[string]interface{}
			if err := json.Unmarshal([]byte(repaired), &test); err == nil {
				log.Printf("[JSONExtraction] Repaired JSON is valid (%d objects)", len(test))
				return repaired
			}
		}
	}

	return "" // Could not extract valid JSON
}

// tryExtractWithBracketMatching extracts JSON with proper bracket matching
func tryExtractWithBracketMatching(text string, startIdx int) string {
	bracketDepth := 0
	braceDepth := 0
	inString := false
	escaped := false
	endIdx := startIdx

	for i := startIdx; i < len(text); i++ {
		ch := text[i]

		// Handle string escaping
		if ch == '\\' && !escaped {
			escaped = true
			continue
		}
		if escaped {
			escaped = false
			continue
		}

		// Track string state
		if ch == '"' {
			inString = !inString
			continue
		}

		if !inString {
			if ch == '[' {
				bracketDepth++
			} else if ch == ']' {
				bracketDepth--
				if bracketDepth == 0 {
					endIdx = i + 1
					break
				}
			} else if ch == '{' {
				braceDepth++
			} else if ch == '}' {
				braceDepth--
			}
		}
	}

	if endIdx == startIdx {
		return ""
	}

	return text[startIdx:endIdx]
}

// repairJSON attempts to fix common JSON issues
func repairJSON(jsonStr string) string {
	// Remove trailing commas
	jsonStr = strings.ReplaceAll(jsonStr, ",]", "]")
	jsonStr = strings.ReplaceAll(jsonStr, ",}", "}")

	// Unescape common issues
	// Replace \" with " when it appears to be double-escaped
	jsonStr = strings.ReplaceAll(jsonStr, "\\\"", "\"")

	// Try to parse - if valid, return
	var test []map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &test); err == nil {
		return jsonStr
	}

	// If still invalid, return empty
	return ""
}

// truncateString safely truncates a string for logging
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// checkNegationHandling verifies that negations are properly handled
func checkNegationHandling(entities []models.ExtractedEntity) bool {
	// If any entities have type "negation", we're handling negation
	for _, e := range entities {
		if e.Type == "negation" {
			return true
		}
	}
	return false
}

// checkNegationInExtractions verifies negation in extraction results
func checkNegationInExtractions(extractions []tools.ExtractionResult) bool {
	for _, e := range extractions {
		if strings.Contains(e.Property, "NOT ") {
			return true
		}
	}
	return false
}
