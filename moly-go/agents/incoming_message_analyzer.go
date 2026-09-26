package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"

	"moly/models"
	"moly/tools"
)

// IncomingMessageAnalyzer analyzes incoming messages and generates suggestions
type IncomingMessageAnalyzer struct {
	llmClient    tools.LLMProvider
	constitution *models.Constitution
}

// NewIncomingMessageAnalyzer creates a new analyzer
func NewIncomingMessageAnalyzer(llmClient tools.LLMProvider) *IncomingMessageAnalyzer {
	return &IncomingMessageAnalyzer{
		llmClient: llmClient,
	}
}

// SetConstitution injects the loaded constitution (for principle-based prompts)
func (ima *IncomingMessageAnalyzer) SetConstitution(c *models.Constitution) {
	ima.constitution = c
}

// DetectSender extracts sender name from incoming message
func (ima *IncomingMessageAnalyzer) DetectSender(incomingMessage string) (string, error) {
	log.Printf("[IncomingMessageAnalyzer] Detecting sender from incoming message")

	// Try "From: Name" format
	fromRegex := regexp.MustCompile(`^From:\s*([A-Za-z\s]+?)(?:\n|$)`)
	if matches := fromRegex.FindStringSubmatch(incomingMessage); matches != nil {
		sender := strings.TrimSpace(matches[1])
		if sender != "" && len(sender) < 50 {
			log.Printf("[IncomingMessageAnalyzer] ✓ Detected sender: %s (From: format)", sender)
			return sender, nil
		}
	}

	// Try "Name: message" format
	nameRegex := regexp.MustCompile(`^([A-Z][a-z]+(?:\s+[A-Z][a-z]+)?):\s+`)
	if matches := nameRegex.FindStringSubmatch(incomingMessage); matches != nil {
		sender := matches[1]
		if sender != "" && len(sender) < 50 {
			log.Printf("[IncomingMessageAnalyzer] ✓ Detected sender: %s (Name: format)", sender)
			return sender, nil
		}
	}

	log.Printf("[IncomingMessageAnalyzer] ⓘ Could not detect sender from message")
	return "", nil
}

// GenerateSuggestions creates response suggestions using LLM and user context
func (ima *IncomingMessageAnalyzer) GenerateSuggestions(
	incomingMessage string,
	conversationHistory []string,
	userAboutMe map[string]interface{},
	senderContext map[string]interface{},
) ([]string, error) {
	log.Printf("[IncomingMessageAnalyzer] Generating suggestions for incoming message")

	// Build conversation context
	historyStr := ""
	if len(conversationHistory) > 0 {
		for i, msg := range conversationHistory {
			if i >= len(conversationHistory)-5 { // Last 5 messages
				historyStr += fmt.Sprintf("• %s\n", msg)
			}
		}
	}

	// Build About Me summary
	aboutMeStr := ""
	if userAboutMe != nil {
		if comm, ok := userAboutMe["communicationStyle"].(string); ok && comm != "" {
			aboutMeStr += fmt.Sprintf("Communication style: %s\n", comm)
		}
		if vals, ok := userAboutMe["coreValues"].([]interface{}); ok && len(vals) > 0 {
			aboutMeStr += fmt.Sprintf("Values: %v\n", vals)
		}
		if tone, ok := userAboutMe["tonePreference"].(string); ok && tone != "" {
			aboutMeStr += fmt.Sprintf("Tone: %s\n", tone)
		}
	}

	// Build sender context
	senderStr := ""
	if senderContext != nil {
		if relationship, ok := senderContext["relationship"].(string); ok && relationship != "" {
			senderStr += fmt.Sprintf("Relationship: %s\n", relationship)
		}
		if context, ok := senderContext["context"].(string); ok && context != "" {
			senderStr += fmt.Sprintf("Context: %s\n", context)
		}
	}

	// Build prompt
	prompt := fmt.Sprintf(`You are Moly, a communication coach. Generate 3 response suggestions to this incoming message.

INCOMING MESSAGE:
"%s"

CONVERSATION CONTEXT:
%s

USER PROFILE:
%s

RECIPIENT CONTEXT:
%s

Generate exactly 3 natural response suggestions that:
- Match the user's communication style
- Reflect their values
- Are appropriate for this recipient
- Feel genuine, not robotic

Format as JSON:
{
  "suggestions": [
    "Response 1",
    "Response 2",
    "Response 3"
  ]
}`, incomingMessage, historyStr, aboutMeStr, senderStr)

	resp, err := ima.llmClient.Call(context.Background(), &tools.LLMRequest{
		UserPrompt:  prompt,
		MaxTokens:   500,
		Temperature: 0.8,
	})
	if err != nil {
		log.Printf("[IncomingMessageAnalyzer] Error generating suggestions: %v", err)
		// Fallback: generate basic suggestions without LLM
		return ima.GenerateFallbackSuggestions(incomingMessage), nil
	}

	response := resp.Content

	// Parse response and extract suggestions
	suggestions := ima.parseSuggestions(response)
	if len(suggestions) == 0 {
		suggestions = ima.GenerateFallbackSuggestions(incomingMessage)
	}

	log.Printf("[IncomingMessageAnalyzer] ✓ Generated %d suggestions", len(suggestions))
	return suggestions, nil
}

// parseSuggestions extracts suggestions from LLM response
func (ima *IncomingMessageAnalyzer) parseSuggestions(response string) []string {
	var result struct {
		Suggestions []string `json:"suggestions"`
	}

	// Try JSON parsing
	if err := unmarshalJSON(response, &result); err == nil && len(result.Suggestions) > 0 {
		return result.Suggestions
	}

	// Fallback: extract lines that look like suggestions
	lines := strings.Split(response, "\n")
	var suggestions []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 10 && strings.HasPrefix(line, `"`) && strings.HasSuffix(line, `"`) {
			suggestions = append(suggestions, strings.Trim(line, `"`))
		}
	}

	if len(suggestions) > 0 {
		return suggestions
	}
	return nil
}

// GenerateFallbackSuggestions creates basic suggestions when LLM fails
func (ima *IncomingMessageAnalyzer) GenerateFallbackSuggestions(incomingMessage string) []string {
	// LLM-based message type detection
	// REMOVED: Hardcoded keyword checks ("?", "thanks", "thank you")
	// Now: LLM detects message type via principle-based analysis
	//
	// Principles:
	// - Transparency: explicitly asking questions or clarifying
	// - Empathy: expressing gratitude or appreciation
	// - Growth: sharing progress or seeking feedback

	messageType := ima.detectMessageTypeLLM(incomingMessage)

	// Return suggestions based on detected message type
	switch messageType {
	case "question":
		return []string{
			"That's a great question. Let me think about it.",
			"I appreciate you asking. Here's what I think...",
			"Good point - I hadn't considered that angle.",
		}
	case "gratitude":
		return []string{
			"Of course! Always happy to help.",
			"Anytime - that's what I'm here for.",
			"Glad I could help!",
		}
	case "apology":
		return []string{
			"No worries, these things happen.",
			"It's okay - I appreciate you saying that.",
			"No need to apologize, let's move forward.",
		}
	default:
		// Generic fallback
		return []string{
			"I appreciate what you shared. Let me think about that.",
			"That's an interesting point. Here's my perspective...",
			"Thank you for sharing this with me.",
		}
	}
}

// detectMessageTypeLLM uses principle-based analysis to detect message type
func (ima *IncomingMessageAnalyzer) detectMessageTypeLLM(message string) string {
	if ima.llmClient == nil {
		return "unknown"
	}

	// Build principle context from Constitution
	principleContext := ima.buildMessageAnalysisPrincipleContext()
	if principleContext == "" {
		return "unknown"
	}

	// Principle-based analysis: which principles does this message engage with?
	prompt := fmt.Sprintf(`Analyze how this message engages with constitutional principles.

%s

Message: "%s"

Respond with ONLY a JSON object (no markdown):
{
  "transparency_engaged": boolean,
  "empathy_engaged": boolean,
  "autonomy_engaged": boolean,
  "growth_engaged": boolean
}`, principleContext, message)

	req := &tools.LLMRequest{
		SystemPrompt: `You analyze messages against constitutional principles.
Respond with only valid JSON, no other text.`,
		UserPrompt:  prompt,
		MaxTokens:   100,
		Temperature: 0.3,
		Retries:     1,
	}

	resp, err := ima.llmClient.Call(context.Background(), req)
	if err != nil {
		log.Printf("[IncomingMessageAnalyzer] detectMessageTypeLLM failed: %v, using default", err)
		return "unknown"
	}

	// Infer message type from principle engagement
	lower := strings.ToLower(resp.Content)

	// If transparency principle engaged (seeking clarity), it's a question
	if strings.Contains(lower, `"transparency_engaged": true`) || strings.Contains(lower, `"transparency_engaged":true`) {
		return "question"
	}

	// If empathy principle engaged, distinguish between gratitude and apology via LLM
	if strings.Contains(lower, `"empathy_engaged": true`) || strings.Contains(lower, `"empathy_engaged":true`) {
		// Ask LLM to analyze the empathy engagement type
		subPrompt := fmt.Sprintf(`Analyze the empathetic intent:
- Gratitude: expressing appreciation, thanks, positive acknowledgment
- Apology: acknowledging responsibility, seeking forgiveness, expressing regret

Respond with ONLY a JSON object:
{
  "gratitude_engaged": boolean,
  "apology_engaged": boolean
}

Message: "%s"`, message)

		subReq := &tools.LLMRequest{
			SystemPrompt: `Analyze empathetic intent via principles. Respond with only JSON.`,
			UserPrompt:   subPrompt,
			MaxTokens:    80,
			Temperature:  0.3,
			Retries:      1,
		}

		subResp, err := ima.llmClient.Call(context.Background(), subReq)
		if err == nil {
			subLower := strings.ToLower(subResp.Content)
			if strings.Contains(subLower, `"apology_engaged": true`) || strings.Contains(subLower, `"apology_engaged":true`) {
				return "apology"
			}
		}
		return "gratitude"
	}

	return "other"
}

// buildMessageAnalysisPrincipleContext dynamically builds principle definitions from Constitution
func (ima *IncomingMessageAnalyzer) buildMessageAnalysisPrincipleContext() string {
	if ima.constitution == nil || len(ima.constitution.SupremePrinciples) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Constitutional principles:\n")

	// Include key principles for message type analysis
	relevantPrinciples := []string{"transparency", "empathy_and_respect", "user_autonomy", "growth_and_learning"}

	for _, princID := range relevantPrinciples {
		for _, principle := range ima.constitution.SupremePrinciples {
			if principle.ID == princID {
				sb.WriteString(fmt.Sprintf("- %s: %s\n", principle.Name, principle.Description))
				break
			}
		}
	}

	return sb.String()
}

// Helper function to unmarshal JSON safely
func unmarshalJSON(data string, v interface{}) error {
	return json.Unmarshal([]byte(data), v)
}

// IncomingMessageAnalysisResult holds analysis of an incoming message
type IncomingMessageAnalysisResult struct {
	Sender      string   `json:"sender"`
	Suggestions []string `json:"suggestions"`
}
