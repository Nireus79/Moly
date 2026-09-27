package agents

import (
	"context"
	"encoding/json"
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

	// Include user communication style
	if analysisCtx.UserProfile != nil && analysisCtx.UserProfile.CommunicationStyle != "" {
		sb.WriteString(fmt.Sprintf("USER COMMUNICATION STYLE: %s\n", analysisCtx.UserProfile.CommunicationStyle))
	}

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
		analysis.ReactionTarget = lid.getReactionContextLLM(userMessage)
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

// ExtractEntitiesWithClassification extracts entities with semantic classification (self-ref vs contact vs topic)
func (lid *LLMIntentDetector) ExtractEntitiesWithClassification(ctx context.Context, message string) ([]models.ExtractedEntity, error) {
	if lid.llmClient == nil {
		log.Printf("[IntentDetector] No LLM available, cannot extract entities")
		return []models.ExtractedEntity{}, nil
	}

	log.Printf("[IntentDetector] Extracting entities with semantic classification")

	systemPrompt := `You are an entity classifier. Extract entities from the message and classify them.

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
      "ambiguous_possibilities": ["self_reference", "contact"]
    }
  ]
}`

	userPrompt := fmt.Sprintf(`Classify entities in this message:
Message: "%s"

Extract all persons, topics, and goals mentioned. For each entity, determine:
1. What it is (person, topic, goal)
2. Whether it's self-reference (Moly) or external
3. Confidence level (0.0-1.0)
4. Is it ambiguous? (could be multiple types)`, message)

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

		// 2. Validate evidence is substring from message
		if !strings.Contains(message, entity.Evidence) {
			log.Printf("[IntentDetector] Evidence not found in message: %s", entity.Evidence)
			continue
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

	// Check for ambiguous entities
	for _, entity := range entities {
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
