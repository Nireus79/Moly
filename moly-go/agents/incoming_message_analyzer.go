package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"

	"moly/tools"
)

// IncomingMessageAnalyzer analyzes incoming messages and generates suggestions
type IncomingMessageAnalyzer struct {
	llmClient tools.LLMProvider
}

// NewIncomingMessageAnalyzer creates a new analyzer
func NewIncomingMessageAnalyzer(llmClient tools.LLMProvider) *IncomingMessageAnalyzer {
	return &IncomingMessageAnalyzer{
		llmClient: llmClient,
	}
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
	// Detect message type and generate appropriate responses
	msg := strings.ToLower(incomingMessage)

	if strings.Contains(msg, "?") {
		// Question asked
		return []string{
			"That's a great question. Let me think about it.",
			"I appreciate you asking. Here's what I think...",
			"Good point - I hadn't considered that angle.",
		}
	} else if strings.Contains(msg, "thanks") || strings.Contains(msg, "thank you") {
		// Expression of gratitude
		return []string{
			"Of course! Always happy to help.",
			"Anytime - that's what I'm here for.",
			"Glad I could help!",
		}
	} else if strings.Contains(msg, "sorry") || strings.Contains(msg, "apologize") {
		// Apology
		return []string{
			"No worries, these things happen.",
			"It's okay - I appreciate you saying that.",
			"No need to apologize, let's move forward.",
		}
	} else {
		// Generic responses
		return []string{
			"Sounds good!",
			"I appreciate you sharing that.",
			"Thanks for letting me know.",
		}
	}
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
