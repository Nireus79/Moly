package generation

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"moly/database"
	"moly/extraction"
	"moly/models"
	"moly/tools"
)

// LLMResponseGenerator - LLM-powered response generation
type LLMResponseGenerator struct {
	llm *tools.LLMClient
}

// NewLLMResponseGenerator - Create LLM-based generator
func NewLLMResponseGenerator(llm *tools.LLMClient) *LLMResponseGenerator {
	return &LLMResponseGenerator{llm: llm}
}

// GenerateResponse - Generate response using LLM with adaptive prompts
func (lrg *LLMResponseGenerator) GenerateResponse(ctx context.Context, msg string, extraction *extraction.ExtractedContext, userContext *database.UserContextSnapshot) (*models.ConversationResponse, error) {
	log.Printf("[LLMResponseGenerator] Generating response with adaptive prompt")

	if lrg.llm == nil {
		log.Printf("[LLMResponseGenerator] WARNING: LLM client is nil, cannot generate response")
		return nil, fmt.Errorf("LLM client is nil")
	}

	// Build adaptive system prompt based on user's style and emotional state
	systemPrompt := lrg.buildAdaptiveSystemPrompt(extraction, userContext.AboutMe)

	// Build user prompt with context
	userPrompt := lrg.buildUserPrompt(msg, extraction, userContext)

	req := &tools.LLMRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Temperature:  0.7, // Higher temperature for natural responses
		MaxTokens:    1000,
	}

	resp, err := lrg.llm.Call(context.Background(), req)
	if err != nil {
		log.Printf("[LLMResponseGenerator] ERROR: LLM call failed: %v", err)
		return nil, fmt.Errorf("failed to generate response: %w", err)
	}

	response := &models.ConversationResponse{
		Phase:    "responding",
		Response: strings.TrimSpace(resp.Content),
		Metadata: map[string]interface{}{
			"type":            "conversational",
			"generated_by":    "llm",
			"tokens_used":     resp.TokensUsed,
			"processing_time": resp.ProcessingTimeMs,
		},
	}

	log.Printf("[LLMResponseGenerator] ✓ Response generated: %d chars", len(response.Response))
	return response, nil
}

// buildAdaptiveSystemPrompt - Build system prompt that adapts to user's style and emotional state
func (lrg *LLMResponseGenerator) buildAdaptiveSystemPrompt(extraction *extraction.ExtractedContext, aboutMe *models.AboutMe) string {
	// Start with role definition
	prompt := "You are Moly, a thoughtful conversational AI that helps people think through their communication and relationships.\n\n"

	// Add communication style instructions
	if aboutMe != nil && aboutMe.CommunicationStyle != "" {
		switch aboutMe.CommunicationStyle {
		case "formal":
			prompt += "The user prefers formal, professional communication. Use proper grammar and professional tone.\n"
		case "casual":
			prompt += "The user prefers casual, friendly communication. Use conversational language and a warm tone.\n"
		case "playful":
			prompt += "The user appreciates playful, light communication. Use humor and a friendly approach.\n"
		}
	}

	// Add emotional tone guidance
	switch extraction.EmotionalTone {
	case "very_negative":
		prompt += "The user is expressing significant distress or concern. Prioritize validation of their feelings before asking deeper questions.\n"
	case "negative":
		prompt += "The user is expressing frustration or concern. Acknowledge their feelings and show understanding.\n"
	case "positive", "very_positive":
		prompt += "The user is expressing positive emotions. Match their energy while still asking thoughtful questions.\n"
	default:
		prompt += "The user seems neutral. Ask clarifying Socratic questions.\n"
	}

	// Add core guidelines
	prompt += `
Guidelines:
1. VALIDATE: Acknowledge what the user shared before asking questions
2. SOCRATIC: Ask open-ended questions that help them think deeper
3. CONTEXTUAL: Reference their specific situation, not generic advice
4. CONCISE: Keep responses focused and natural (not robotic)
5. AUTHENTIC: Be genuine, not fake-friendly

DO NOT:
- Give unsolicited advice or tell them what to do
- Use platitudes or canned responses
- Ignore their emotional state or communication preferences
- Ask more than one question at a time`

	return prompt
}

// buildUserPrompt - Build user prompt with context
func (lrg *LLMResponseGenerator) buildUserPrompt(msg string, extraction *extraction.ExtractedContext, userContext *database.UserContextSnapshot) string {
	prompt := fmt.Sprintf(`User's message: "%s"

Context about this user:
- Communication style: %s
- Emotional tone: %s
- Topic: %s
- Intention: %s
- Contact (if mentioned): %s`, msg, extraction.Style, extraction.EmotionalTone, extraction.Topic, extraction.Intention, extraction.Contact)

	// Add recent conversation context if available
	if len(userContext.RecentMessages) > 0 {
		prompt += "\n\nRecent conversation history (for context):"
		for i, msg := range userContext.RecentMessages {
			if i >= 3 { // Limit to last 3 messages for context
				break
			}
			prompt += fmt.Sprintf("\n- [%s]: %s", msg.Role, msg.Content)
		}
	}

	// Add known goals if available
	if userContext.AboutMe != nil && len(userContext.AboutMe.Goals) > 0 {
		prompt += "\n\nUser's known goals:"
		for _, goal := range userContext.AboutMe.Goals {
			prompt += fmt.Sprintf("\n- %s", goal)
		}
	}

	prompt += "\n\nGenerate a single, authentic response that validates their message and asks a Socratic question to deepen their thinking."

	return prompt
}

// GenerateInsight - Extract insights from message using LLM
func (lrg *LLMResponseGenerator) GenerateInsight(ctx context.Context, msg string, extraction *extraction.ExtractedContext, userID, conversationID string) (*models.Reflection, error) {
	log.Printf("[LLMResponseGenerator] Extracting insights from message")

	systemPrompt := `You are an expert at understanding what messages reveal about a person's patterns, values, and communication style.
Extract structured insights and return ONLY valid JSON:
{
  "characteristics": ["what this reveals about them"],
  "interests": ["topics they care about"],
  "intentions": ["what they're trying to accomplish"]
}`

	userPrompt := fmt.Sprintf(`Analyze this message and extract insights:
"%s"

The user's current:
- Communication style: %s
- Emotional state: %s
- Topic focus: %s

Return only valid JSON.`, msg, extraction.Style, extraction.EmotionalTone, extraction.Topic)

	req := &tools.LLMRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Temperature:  0.5,
		MaxTokens:    500,
	}

	resp, err := lrg.llm.Call(context.Background(), req)
	if err != nil {
		log.Printf("[LLMResponseGenerator] WARNING: Failed to extract insight: %v", err)
		return nil, err
	}

	var insightData map[string]interface{}
	if err := json.Unmarshal([]byte(resp.Content), &insightData); err != nil {
		log.Printf("[LLMResponseGenerator] WARNING: Failed to parse insight JSON: %v", err)
		return nil, err
	}

	// Convert to Reflection struct
	insight := &models.Reflection{
		ConversationID: conversationID,
		Status:         "pending_approval",
	}

	// Parse characteristics
	if chars, ok := insightData["characteristics"].([]interface{}); ok {
		for _, c := range chars {
			if str, ok := c.(string); ok {
				insight.Characteristics = append(insight.Characteristics, str)
			}
		}
	}

	// Parse interests
	if ints, ok := insightData["interests"].([]interface{}); ok {
		for _, i := range ints {
			if str, ok := i.(string); ok {
				insight.Interests = append(insight.Interests, str)
			}
		}
	}

	// Parse intentions
	if inds, ok := insightData["intentions"].([]interface{}); ok {
		for _, ind := range inds {
			if str, ok := ind.(string); ok {
				insight.Intentions = append(insight.Intentions, str)
			}
		}
	}

	log.Printf("[LLMResponseGenerator] ✓ Insight extracted: %d characteristics", len(insight.Characteristics))
	return insight, nil
}
