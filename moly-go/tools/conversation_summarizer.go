package tools

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"moly/models"
)

// ConversationSummarizer - Generates compact, LLM-based conversation summaries
type ConversationSummarizer struct {
	llm LLMProvider
	db  interface{} // Optional database connection for loading confirmed preferences
}

// NewConversationSummarizer creates a new summarizer
func NewConversationSummarizer(llm LLMProvider) *ConversationSummarizer {
	return &ConversationSummarizer{llm: llm}
}

// SetDatabase allows the summarizer to load confirmed preferences from DB
func (cs *ConversationSummarizer) SetDatabase(db interface{}) {
	cs.db = db
}

// ShouldUpdateSummary checks if a summary needs updating
// Returns true if:
// - Summary doesn't exist (nil) AND we have >= 5 messages (early generation for better context)
// - Messages since update >= threshold (default 10 for updates)
// - Summary is stale (hasn't been updated in > 1 hour)
func (cs *ConversationSummarizer) ShouldUpdateSummary(summary *models.ConversationSummary, threshold int) bool {
	if summary == nil {
		// OPTIMIZATION: Generate first summary at 5 messages instead of 10 for better early context
		// Threshold passed will be the message count; if < 5, we wait for more
		if threshold >= 5 {
			log.Printf("[ConversationSummarizer] Early summary generation triggered at %d messages", threshold)
			return true
		}
		return false
	}

	if threshold <= 0 {
		threshold = 10 // Default threshold for updates
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

	// Build user prompt (include confirmed preferences from Layer 3)
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
  "confirmed_choices": ["choice1", "choice2"],
  "open_questions": ["question1", "question2"],
  "confidence": 0.95
}

GUIDELINES:
- arc: 1-2 sentence narrative of what happened, decisions made, topics explored
- key_topics: 3-5 tags describing what was discussed
- user_patterns: 3-5 observed communication patterns (e.g., "prefers_directness", "values_consent")
- confirmed_choices: Things the user has explicitly confirmed or clarified (e.g., "prefers_explicit_communication", "wants_structured_learning")
- open_questions: 2-4 unresolved questions or topics to explore
- confidence: 0-1 score of how complete/accurate the summary is

BE SPECIFIC AND CONCISE:
- Use lowercase with underscores for patterns/topics/choices
- Include actual observations, not generic descriptions
- Focus on USER's patterns and explicitly stated preferences, not Moly's responses
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
		Arc              string   `json:"arc"`
		KeyTopics        []string `json:"key_topics"`
		UserPatterns     []string `json:"user_patterns"`
		ConfirmedChoices []string `json:"confirmed_choices"`
		OpenQuestions    []string `json:"open_questions"`
		Confidence       float64  `json:"confidence"`
	}

	// FIX #33: Parse with validation
	if err := SafeJSONParse("ConversationSummarizer.buildSummary", []byte(llmResponse), &parsed); err != nil {
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
		UserID:           userID,
		ConversationID:   conversationID,
		Arc:              parsed.Arc,
		KeyTopics:        parsed.KeyTopics,
		UserPatterns:     parsed.UserPatterns,
		ConfirmedChoices: parsed.ConfirmedChoices,
		OpenQuestions:    parsed.OpenQuestions,
		Confidence:       parsed.Confidence,
		MessageCount:     messageCount,
		LastUpdated:      time.Now().Unix(),
		UpdatedAt:        time.Now().Unix(),
	}

	// If updating existing summary, merge confirmed choices
	if previousSummary != nil {
		summary.ID = previousSummary.ID
		summary.CreatedAt = previousSummary.CreatedAt
		summary.SummaryVersion = previousSummary.SummaryVersion + 1
		summary.MessagesSinceUpdate = 0 // Reset counter after update
		// Merge new confirmed choices with existing ones (avoid duplicates)
		choiceMap := make(map[string]bool)
		for _, choice := range previousSummary.ConfirmedChoices {
			choiceMap[choice] = true
		}
		for _, choice := range parsed.ConfirmedChoices {
			choiceMap[choice] = true
		}
		summary.ConfirmedChoices = make([]string, 0, len(choiceMap))
		for choice := range choiceMap {
			summary.ConfirmedChoices = append(summary.ConfirmedChoices, choice)
		}
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
	if len(summary.ConfirmedChoices) > 8 {
		summary.ConfirmedChoices = summary.ConfirmedChoices[:8]
	}
	if len(summary.OpenQuestions) > 5 {
		summary.OpenQuestions = summary.OpenQuestions[:5]
	}

	log.Printf("[ConversationSummarizer] Parsed summary: version=%d, confidence=%.2f",
		summary.SummaryVersion, summary.Confidence)

	return summary, nil
}


