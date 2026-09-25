package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"moly/models"
)

// ConversationSummarizer - Generates compact, LLM-based conversation summaries
type ConversationSummarizer struct {
	llm LLMProvider
}

// NewConversationSummarizer creates a new summarizer
func NewConversationSummarizer(llm LLMProvider) *ConversationSummarizer {
	return &ConversationSummarizer{llm: llm}
}

// ShouldUpdateSummary checks if a summary needs updating
// Returns true if:
// - Summary doesn't exist (nil)
// - Messages since update >= threshold (default 10)
// - Summary is stale (hasn't been updated in > 1 hour)
func (cs *ConversationSummarizer) ShouldUpdateSummary(summary *models.ConversationSummary, threshold int) bool {
	if summary == nil {
		return true // Create new summary
	}

	if threshold <= 0 {
		threshold = 10 // Default threshold
	}

	// Check if enough new messages accumulated
	if summary.MessagesSinceUpdate >= threshold {
		log.Printf("[ConversationSummarizer] Update needed: %d messages since last update (threshold: %d)",
			summary.MessagesSinceUpdate, threshold)
		return true
	}

	// Check if summary is stale (not updated in > 1 hour)
	oneHourAgo := time.Now().Unix() - 3600
	if summary.LastUpdated < oneHourAgo {
		log.Printf("[ConversationSummarizer] Update needed: summary is stale (last updated %d seconds ago)",
			time.Now().Unix()-summary.LastUpdated)
		return true
	}

	return false
}

// SummarizeConversation generates or updates a summary from conversation messages
// messages: all messages in conversation
// previousSummary: existing summary (nil if creating new)
// Returns: updated or new ConversationSummary
func (cs *ConversationSummarizer) SummarizeConversation(
	ctx context.Context,
	conversationID string,
	userID string,
	messages []models.Message,
	previousSummary *models.ConversationSummary,
) (*models.ConversationSummary, error) {

	if conversationID == "" || userID == "" || len(messages) == 0 {
		return nil, fmt.Errorf("conversationID, userID, and messages are required")
	}

	if cs.llm == nil {
		return nil, fmt.Errorf("LLM provider not available")
	}

	log.Printf("[ConversationSummarizer] Summarizing %d messages for conversation %s",
		len(messages), conversationID)

	// Build conversation text for LLM
	conversationText := cs.buildConversationText(messages)

	// Build system prompt
	systemPrompt := cs.buildSystemPrompt(previousSummary)

	// Build user prompt
	userPrompt := cs.buildUserPrompt(conversationText, previousSummary)

	// Call LLM
	req := &LLMRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		MaxTokens:    800,
		Temperature:  0.3,
		Retries:      2,
	}

	log.Printf("[ConversationSummarizer] Calling LLM to generate summary")
	resp, err := cs.llm.Call(ctx, req)
	if err != nil {
		log.Printf("[ConversationSummarizer] LLM call failed: %v", err)
		return nil, fmt.Errorf("LLM summary generation failed: %w", err)
	}

	// Parse LLM response into summary
	summary, err := cs.parseAndValidateSummary(resp.Content, conversationID, userID, len(messages), previousSummary)
	if err != nil {
		log.Printf("[ConversationSummarizer] Failed to parse summary: %v", err)
		return nil, fmt.Errorf("summary parsing failed: %w", err)
	}

	log.Printf("[ConversationSummarizer] ✓ Summary generated: %d topics, %d patterns",
		len(summary.KeyTopics), len(summary.UserPatterns))

	return summary, nil
}

// buildConversationText formats messages for LLM analysis
func (cs *ConversationSummarizer) buildConversationText(messages []models.Message) string {
	var sb strings.Builder

	for _, msg := range messages {
		role := "User"
		if msg.Role == "assistant" {
			role = "Moly"
		}

		sb.WriteString(fmt.Sprintf("%s: %s\n\n", role, msg.Content))
	}

	return sb.String()
}

// buildSystemPrompt creates the system prompt for summary generation
func (cs *ConversationSummarizer) buildSystemPrompt(previousSummary *models.ConversationSummary) string {
	var sb strings.Builder

	sb.WriteString(`You are a conversation summarizer. Your task is to create a concise, structured summary of a conversation.

RESPONSE FORMAT:
Return ONLY valid JSON (no markdown, no explanation):
{
  "arc": "brief narrative of conversation flow",
  "key_topics": ["topic1", "topic2"],
  "user_patterns": ["pattern1", "pattern2"],
  "open_questions": ["question1", "question2"],
  "confidence": 0.95
}

GUIDELINES:
- arc: 1-2 sentence narrative of what happened, decisions made, topics explored
- key_topics: 3-5 tags describing what was discussed
- user_patterns: 3-5 observed communication patterns (e.g., "prefers_directness", "values_consent")
- open_questions: 2-4 unresolved questions or topics to explore
- confidence: 0-1 score of how complete/accurate the summary is

BE SPECIFIC AND CONCISE:
- Use lowercase with underscores for patterns/topics
- Include actual observations, not generic descriptions
- Focus on USER's patterns, not Moly's responses
`)

	if previousSummary != nil && previousSummary.Arc != "" {
		sb.WriteString("\nPREVIOUS SUMMARY (use to update, not replace):\n")
		sb.WriteString(fmt.Sprintf("Arc: %s\n", previousSummary.Arc))
		sb.WriteString(fmt.Sprintf("Topics: %v\n", previousSummary.KeyTopics))
		sb.WriteString(fmt.Sprintf("Patterns: %v\n", previousSummary.UserPatterns))
		sb.WriteString(fmt.Sprintf("Open questions: %v\n", previousSummary.OpenQuestions))
		sb.WriteString("\nIf this is an UPDATE (not replacement), preserve patterns and topics still relevant, add new ones, retire outdated ones.\n")
	}

	return sb.String()
}

// buildUserPrompt creates the user prompt for summary generation
func (cs *ConversationSummarizer) buildUserPrompt(conversationText string, previousSummary *models.ConversationSummary) string {
	var sb strings.Builder

	if previousSummary != nil && previousSummary.Arc != "" {
		sb.WriteString(fmt.Sprintf("UPDATE the existing summary with these new messages:\n\n"))
	} else {
		sb.WriteString("SUMMARIZE this conversation:\n\n")
	}

	sb.WriteString(conversationText)

	return sb.String()
}

// parseAndValidateSummary parses and validates the LLM response
func (cs *ConversationSummarizer) parseAndValidateSummary(
	llmResponse string,
	conversationID string,
	userID string,
	messageCount int,
	previousSummary *models.ConversationSummary,
) (*models.ConversationSummary, error) {

	// Parse JSON response
	var parsed struct {
		Arc           string   `json:"arc"`
		KeyTopics     []string `json:"key_topics"`
		UserPatterns  []string `json:"user_patterns"`
		OpenQuestions []string `json:"open_questions"`
		Confidence    float64  `json:"confidence"`
	}

	if err := json.Unmarshal([]byte(llmResponse), &parsed); err != nil {
		log.Printf("[ConversationSummarizer] Failed to parse JSON: %v", err)
		return nil, fmt.Errorf("invalid JSON response: %w", err)
	}

	// Validate and sanitize
	if parsed.Arc == "" {
		return nil, fmt.Errorf("arc cannot be empty")
	}

	// Ensure confidence is in valid range
	if parsed.Confidence < 0 || parsed.Confidence > 1 {
		parsed.Confidence = 0.75 // Default reasonable value
	}

	// Build summary
	summary := &models.ConversationSummary{
		UserID:         userID,
		ConversationID: conversationID,
		Arc:            parsed.Arc,
		KeyTopics:      parsed.KeyTopics,
		UserPatterns:   parsed.UserPatterns,
		OpenQuestions:  parsed.OpenQuestions,
		Confidence:     parsed.Confidence,
		MessageCount:   messageCount,
		LastUpdated:    time.Now().Unix(),
		UpdatedAt:      time.Now().Unix(),
	}

	// If updating existing summary
	if previousSummary != nil {
		summary.ID = previousSummary.ID
		summary.CreatedAt = previousSummary.CreatedAt
		summary.SummaryVersion = previousSummary.SummaryVersion + 1
		summary.MessagesSinceUpdate = 0 // Reset counter after update
		// Preserve confirmed choices from Layer 3
		summary.ConfirmedChoices = previousSummary.ConfirmedChoices
	} else {
		summary.CreatedAt = time.Now().Unix()
		summary.SummaryVersion = 1
		summary.MessagesSinceUpdate = 0
	}

	// Validate arrays aren't too large
	if len(summary.KeyTopics) > 10 {
		summary.KeyTopics = summary.KeyTopics[:10]
	}
	if len(summary.UserPatterns) > 10 {
		summary.UserPatterns = summary.UserPatterns[:10]
	}
	if len(summary.OpenQuestions) > 5 {
		summary.OpenQuestions = summary.OpenQuestions[:5]
	}

	log.Printf("[ConversationSummarizer] Parsed summary: version=%d, confidence=%.2f",
		summary.SummaryVersion, summary.Confidence)

	return summary, nil
}

// SummarizeMessageWindow takes last N messages and creates a compact window summary
// Used for context building when messages before the window are already summarized
// Returns formatted string for inclusion in evaluator context
func (cs *ConversationSummarizer) SummarizeMessageWindow(messages []models.Message) string {
	if len(messages) == 0 {
		return ""
	}

	var sb strings.Builder

	for _, msg := range messages {
		role := "User"
		if msg.Role == "assistant" {
			role = "Moly"
		}

		sb.WriteString(fmt.Sprintf("%s: %s\n", role, msg.Content))
	}

	return sb.String()
}

// DetectTopicBoundary checks if there's a topic shift in recent messages
// Returns true if conversation appears to have shifted to a new topic
func (cs *ConversationSummarizer) DetectTopicBoundary(ctx context.Context, messages []models.Message) (bool, error) {
	if len(messages) < 3 {
		return false, nil // Not enough messages to detect boundary
	}

	// Get last 3 messages
	recentMessages := messages
	if len(messages) > 3 {
		recentMessages = messages[len(messages)-3:]
	}

	systemPrompt := `You are a conversation analyzer. Determine if the last message(s) represent a topic shift.
A topic shift is when the conversation moves from one subject to a distinctly different subject.

Return ONLY JSON:
{"is_topic_shift": true/false, "reasoning": "brief explanation"}

Ignore minor elaborations or follow-ups within the same topic.`

	conversationText := cs.buildConversationText(recentMessages)

	userPrompt := fmt.Sprintf(`Has there been a topic shift in this conversation segment?

%s

Is this a topic shift? (true/false)`, conversationText)

	req := &LLMRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		MaxTokens:    150,
		Temperature:  0.3,
		Retries:      1,
	}

	resp, err := cs.llm.Call(ctx, req)
	if err != nil {
		log.Printf("[ConversationSummarizer] Topic boundary detection failed: %v", err)
		return false, nil // Fail open - don't force update
	}

	var parsed struct {
		IsTopicShift bool   `json:"is_topic_shift"`
		Reasoning    string `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(resp.Content), &parsed); err != nil {
		log.Printf("[ConversationSummarizer] Failed to parse topic boundary response: %v", err)
		return false, nil
	}

	if parsed.IsTopicShift {
		log.Printf("[ConversationSummarizer] Topic shift detected: %s", parsed.Reasoning)
	}

	return parsed.IsTopicShift, nil
}
