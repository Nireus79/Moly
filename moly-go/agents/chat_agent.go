package agents

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"moly/models"
	"moly/tools"
)

// RunChat executes the chat flow for V2.1
// Handles multi-turn conversation with implicit context learning
func (ca *conversationAgent) RunChat(ctx context.Context, userMessage string, history []*models.ChatMessage) (*models.ChatResponse, error) {
	log.Printf("[ChatAgent] Processing message: %.80s...", userMessage)

	startTime := time.Now()
	response := &models.ChatResponse{
		Timestamp:      startTime.Unix(),
		ContextLearned: make(map[string]interface{}),
		AboutMeGaps:    []string{},
	}

	// Safety check first
	safetyInput := &tools.SafetyCheckInput{
		Message: userMessage,
	}
	safetyCheck, err := ca.safetyChecker.Check(ctx, safetyInput)
	if err != nil {
		log.Printf("[ChatAgent] Safety check failed: %v", err)
	} else {
		log.Printf("[ChatAgent] Safety check: alertType=%v, severity=%v", safetyCheck.AlertType, safetyCheck.Severity)

		if safetyCheck.AlertType == tools.SafetyAlertTypeCrisis {
			response.Response = safetyCheck.Message
			if len(safetyCheck.Resources) > 0 {
				response.Response += "\n\nResources available to help."
			}
			log.Printf("[ChatAgent] Crisis detected, providing safety resources")
			return response, nil
		}
	}

	// Detect contact mentions (simple keyword-based for now)
	contactMention := detectContactMention(userMessage)
	if contactMention != nil && contactMention.Detected {
		response.ContactMention = contactMention
		log.Printf("[ChatAgent] Contact mention detected: %s", contactMention.PersonName)
	}

	// Extract implicit AboutMe from message
	extractedAboutMe := extractImplicitAboutMe(userMessage)
	if len(extractedAboutMe) > 0 {
		response.ContextLearned["aboutMe"] = extractedAboutMe
		log.Printf("[ChatAgent] Learned from message: %+v", extractedAboutMe)
	}

	// Generate natural response using LLM
	llmResponse, err := ca.generateChatResponse(ctx, userMessage, history)
	if err != nil {
		log.Printf("[ChatAgent] LLM generation failed: %v, using fallback", err)
		llmResponse = generateFallbackChatResponse(userMessage)
	}

	response.Response = llmResponse

	// Suggest follow-up question if needed
	if shouldAskFollowUp(userMessage, history) {
		followUp := suggestFollowUpQuestion(userMessage, extractedAboutMe)
		response.SuggestedFollowUp = followUp
		log.Printf("[ChatAgent] Suggested follow-up: %s", followUp)
	}

	duration := time.Since(startTime)
	log.Printf("[ChatAgent] Chat processing completed in %v", duration)

	return response, nil
}

// generateChatResponse generates a natural language response using LLM
func (ca *conversationAgent) generateChatResponse(ctx context.Context, userMessage string, history []*models.ChatMessage) (string, error) {
	if ca.llmClient == nil {
		return generateFallbackChatResponse(userMessage), nil
	}

	// Build context from history (last 5 messages)
	historyContext := buildHistoryContext(history)

	// Create prompt for LLM
	prompt := fmt.Sprintf(`You are Moly, a personal communication coach. You help users navigate communication challenges with empathy and wisdom.

Previous conversation:
%s

User: %s

Respond naturally and warmly. Keep responses concise (2-3 sentences). If the user mentions a communication challenge, offer thoughtful guidance.`, historyContext, userMessage)

	systemPrompt := `You are Moly, a personal communication coach. Your role is to help users think through their communication challenges and improve their relationships. Be warm, empathetic, and wise. Ask Socratic questions when helpful. Remember what you learn about the user.`

	req := &tools.LLMRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   prompt,
		Temperature:  0.7,
		MaxTokens:    256,
	}

	resp, err := ca.llmClient.Call(ctx, req)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	return strings.TrimSpace(resp.Content), nil
}

// detectContactMention detects if a message mentions a contact/person
func detectContactMention(message string) *models.ContactMentionDetected {
	lowerMsg := strings.ToLower(message)

	// List of common contact references
	keywords := map[string]string{
		"boss":       "professional",
		"manager":    "professional",
		"colleague":  "professional",
		"coworker":   "professional",
		"partner":    "romantic",
		"spouse":     "romantic",
		"husband":    "romantic",
		"wife":       "romantic",
		"girlfriend": "romantic",
		"boyfriend":  "romantic",
		"friend":     "personal",
		"parent":     "family",
		"mom":        "family",
		"dad":        "family",
		"mother":     "family",
		"father":     "family",
		"sister":     "family",
		"brother":    "family",
		"family":     "family",
		"kid":        "family",
		"child":      "family",
	}

	for keyword, relationship := range keywords {
		if strings.Contains(lowerMsg, keyword) {
			return &models.ContactMentionDetected{
				Detected:     true,
				PersonName:   keyword,
				Relationship: relationship,
				Suggestion:   fmt.Sprintf("Would you like to save info about your %s to help me understand your relationships better?", keyword),
			}
		}
	}

	return nil
}

// extractImplicitAboutMe extracts AboutMe fields from natural conversation
func extractImplicitAboutMe(message string) map[string]interface{} {
	extracted := make(map[string]interface{})
	lowerMsg := strings.ToLower(message)

	// Extract communication style
	if strings.Contains(lowerMsg, "direct") || strings.Contains(lowerMsg, "straight to the point") {
		extracted["communicationStyle"] = "direct"
	} else if strings.Contains(lowerMsg, "gentle") || strings.Contains(lowerMsg, "soft") {
		extracted["communicationStyle"] = "gentle"
	}

	// Extract preferences
	preferences := []string{}
	if strings.Contains(lowerMsg, "hate small talk") || strings.Contains(lowerMsg, "hate talking about weather") {
		preferences = append(preferences, "no_small_talk")
	}
	if strings.Contains(lowerMsg, "no small talk") || strings.Contains(lowerMsg, "skip pleasantries") {
		preferences = append(preferences, "direct_communication")
	}
	if len(preferences) > 0 {
		extracted["preferences"] = preferences
	}

	// Extract values
	values := []string{}
	if strings.Contains(lowerMsg, "honesty") || strings.Contains(lowerMsg, "honest") {
		values = append(values, "honesty")
	}
	if strings.Contains(lowerMsg, "respect") {
		values = append(values, "respect")
	}
	if strings.Contains(lowerMsg, "clear") || strings.Contains(lowerMsg, "clarity") {
		values = append(values, "clarity")
	}
	if len(values) > 0 {
		extracted["values"] = values
	}

	return extracted
}

// shouldAskFollowUp determines if a follow-up question is appropriate
func shouldAskFollowUp(message string, history []*models.ChatMessage) bool {
	// Ask follow-up if:
	// - User mentioned a person but didn't provide details
	// - Message is ambiguous
	// - We're building profile

	lowerMsg := strings.ToLower(message)

	// Don't ask if message is just a greeting
	if message == "hi" || message == "hello" || message == "hey" {
		return false
	}

	// Ask if contact mentioned but relationship not clear
	hasContact := strings.ContainsAny(lowerMsg, "boss manager colleague friend")
	hasDetails := strings.ContainsAny(lowerMsg, "years long time recently always")

	return hasContact && !hasDetails
}

// suggestFollowUpQuestion generates a Socratic follow-up question
func suggestFollowUpQuestion(message string, extractedAboutMe map[string]interface{}) string {
	lowerMsg := strings.ToLower(message)

	// Socratic questions based on context
	if strings.Contains(lowerMsg, "boss") || strings.Contains(lowerMsg, "manager") {
		return "How long have you worked together, and how would you describe your working relationship?"
	}

	if strings.Contains(lowerMsg, "friend") {
		return "How close are you to this friend, and what makes your friendship meaningful?"
	}

	if strings.Contains(lowerMsg, "family") {
		return "How would you describe your relationship with them?"
	}

	if strings.Contains(lowerMsg, "difficult") || strings.Contains(lowerMsg, "challenge") {
		return "What's making this situation difficult for you?"
	}

	return "Tell me more about this situation and what's important to you here."
}

// buildHistoryContext creates a summary of conversation history
func buildHistoryContext(history []*models.ChatMessage) string {
	if len(history) == 0 {
		return "(This is the start of our conversation)"
	}

	// Take last 4 messages (2 exchanges)
	start := 0
	if len(history) > 4 {
		start = len(history) - 4
	}

	var context strings.Builder
	for _, msg := range history[start:] {
		if msg.Role == "user" {
			context.WriteString(fmt.Sprintf("User: %s\n", msg.Content))
		} else {
			context.WriteString(fmt.Sprintf("Moly: %s\n", msg.Content))
		}
	}

	return context.String()
}

// generateFallbackChatResponse provides a response when LLM is unavailable
func generateFallbackChatResponse(userMessage string) string {
	lowerMsg := strings.ToLower(userMessage)

	if strings.Contains(lowerMsg, "help") || strings.Contains(lowerMsg, "advice") {
		return "I'm here to help. Tell me more about what's going on, and I'll do my best to support you."
	}

	if strings.Contains(lowerMsg, "difficult") || strings.Contains(lowerMsg, "challenge") {
		return "That sounds challenging. What would help you navigate this better?"
	}

	if strings.Contains(lowerMsg, "sorry") || strings.Contains(lowerMsg, "apologi") {
		return "It's thoughtful that you want to make things right. Let's think through the best way to approach this."
	}

	if strings.Contains(lowerMsg, "congratul") || strings.Contains(lowerMsg, "success") {
		return "That's wonderful! You should be proud of yourself."
	}

	return "I hear you. Help me understand better—what's really important here?"
}
