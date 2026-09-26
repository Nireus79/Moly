package database

import (
	"fmt"
	"log"

	"moly/models"
)

// AnalysisContextBuilder constructs bounded context for evaluators
// Combines: summary + recent messages + preferences + profile
// Total: ~700-800 tokens (scalable regardless of conversation length)
type AnalysisContextBuilder struct {
	summaryRepo      *ConversationSummaryRepository
	chatRepo         *ChatMessageRepository
	contextAttrRepo  *ContextAttributeRepository
	db               *Database
}

// NewAnalysisContextBuilder creates a new builder
func NewAnalysisContextBuilder(
	db *Database,
	summaryRepo *ConversationSummaryRepository,
	chatRepo *ChatMessageRepository,
	contextAttrRepo *ContextAttributeRepository,
) *AnalysisContextBuilder {
	return &AnalysisContextBuilder{
		summaryRepo:     summaryRepo,
		chatRepo:        chatRepo,
		contextAttrRepo: contextAttrRepo,
		db:              db,
	}
}

// BuildAnalysisContext builds complete context for evaluators
// Parameters:
//   - userID: user identifier
//   - conversationID: which conversation
//   - currentMessage: message being analyzed (from user)
//   - allMessages: all messages in conversation (for window extraction)
//   - userProfile: user's AboutMe (communication style, etc)
//   - recentContacts: contacts mentioned recently (for relevance)
// Returns: AnalysisContext with ~700-800 tokens total
func (b *AnalysisContextBuilder) BuildAnalysisContext(
	userID string,
	conversationID string,
	currentMessage string,
	allMessages []models.Message,
	userProfile *models.AboutMe,
) (*models.AnalysisContext, error) {

	if userID == "" || conversationID == "" || currentMessage == "" {
		return nil, fmt.Errorf("userID, conversationID, and currentMessage are required")
	}

	ctx := &models.AnalysisContext{
		CurrentMessage: currentMessage,
		ContextQuality: "minimal",
	}

	log.Printf("[AnalysisContextBuilder] Building context for conversation %s (message count: %d)",
		conversationID, len(allMessages))

	// 1. Load conversation summary (ALL history, compressed)
	summary, err := b.summaryRepo.GetSummary(userID, conversationID)
	if err != nil {
		log.Printf("[AnalysisContextBuilder] Warning: failed to load summary: %v", err)
	} else if summary != nil {
		ctx.ConversationSummary = summary
		ctx.TotalMessages = summary.MessageCount
		log.Printf("[AnalysisContextBuilder] ✓ Loaded summary (v%d, confidence=%.2f)",
			summary.SummaryVersion, summary.Confidence)
	} else {
		// No summary yet - capture total messages from input
		ctx.TotalMessages = len(allMessages)
		log.Printf("[AnalysisContextBuilder] No summary yet (conversation at %d messages)", len(allMessages))
	}

	// 2. Extract recent message window (last 2-3 messages)
	recentMessages := b.extractRecentMessageWindow(allMessages, 3) // Last 3 messages
	ctx.RecentMessages = recentMessages
	log.Printf("[AnalysisContextBuilder] ✓ Extracted %d recent messages for context", len(recentMessages))

	// 3. Load confirmed preferences (from Layer 3 clarifications)
	confirmedPrefs, err := b.loadConfirmedPreferences(userID, conversationID)
	if err != nil {
		log.Printf("[AnalysisContextBuilder] Warning: failed to load preferences: %v", err)
	} else if confirmedPrefs != nil {
		ctx.ConfirmedPreferences = confirmedPrefs
		log.Printf("[AnalysisContextBuilder] ✓ Loaded %d confirmed preferences", len(confirmedPrefs))
	}

	// 4. Load user profile (communication style, values, goals)
	if userProfile != nil {
		ctx.UserProfile = userProfile
		log.Printf("[AnalysisContextBuilder] ✓ Loaded user profile")
	}

	// 5. Extract relevant contacts from recent messages
	relevantContacts := b.extractRelevantContacts(allMessages, 2)
	ctx.RelevantContacts = relevantContacts
	log.Printf("[AnalysisContextBuilder] ✓ Extracted %d relevant contacts", len(relevantContacts))

	// 6. Determine context quality
	ctx.ContextQuality = b.assessContextQuality(ctx)

	log.Printf("[AnalysisContextBuilder] ✓ Context complete (quality: %s)", ctx.ContextQuality)
	return ctx, nil
}

// extractRecentMessageWindow gets the last N messages from conversation
// Returns full message objects (not summarized) for immediate context
func (b *AnalysisContextBuilder) extractRecentMessageWindow(allMessages []models.Message, windowSize int) []models.Message {
	if len(allMessages) == 0 {
		return []models.Message{}
	}

	if len(allMessages) <= windowSize {
		return allMessages // Return all if fewer than window size
	}

	// Return last N messages
	return allMessages[len(allMessages)-windowSize:]
}

// loadConfirmedPreferences loads user's confirmed preferences from Layer 3
// Returns a map of preference_key -> preference_value
// Example: {"threesomes": true, "doggy_style": true, "anal": false}
func (b *AnalysisContextBuilder) loadConfirmedPreferences(userID, conversationID string) (map[string]interface{}, error) {
	if userID == "" || conversationID == "" {
		return nil, fmt.Errorf("userID and conversationID are required")
	}

	// Query context_attributes for source="clarification_response"
	// These are preferences confirmed in conversation
	query := `
		SELECT fact_type, fact_value
		FROM context_attributes
		WHERE user_id = ? AND conversation_id = ? AND source = 'clarification_response'
		ORDER BY created_at DESC
	`

	rows, err := b.db.conn.Query(query, userID, conversationID)
	if err != nil {
		log.Printf("[AnalysisContextBuilder] Failed to query preferences: %v", err)
		return nil, fmt.Errorf("failed to query preferences: %w", err)
	}
	defer rows.Close()

	prefs := make(map[string]interface{})

	for rows.Next() {
		var factType, factValue string
		if err := rows.Scan(&factType, &factValue); err != nil {
			log.Printf("[AnalysisContextBuilder] Failed to scan preference: %v", err)
			continue
		}
		prefs[factType] = factValue
	}

	if len(prefs) == 0 {
		return nil, nil
	}

	return prefs, nil
}

// extractRelevantContacts finds contacts mentioned in recent messages
// Looks at last N messages for contact names
func (b *AnalysisContextBuilder) extractRelevantContacts(allMessages []models.Message, lookbackMessages int) []models.Contact {
	if len(allMessages) == 0 {
		return []models.Contact{}
	}

	// Look at recent messages for contact mentions
	recentStart := len(allMessages) - lookbackMessages
	if recentStart < 0 {
		recentStart = 0
	}

	// In real implementation, would parse message metadata for contact mentions
	// For now, we return empty list since contact extraction is handled elsewhere
	// TODO: Query database for mentioned contacts from recent messages
	_ = recentStart // Avoid unused variable warning
	return []models.Contact{}
}

// assessContextQuality rates how complete/reliable the context is
// Factors:
// - Do we have a summary? (best case)
// - Do we have recent messages? (minimum)
// - Do we have user profile? (helpful)
// - Do we have confirmed preferences? (critical for some evaluations)
func (b *AnalysisContextBuilder) assessContextQuality(ctx *models.AnalysisContext) string {
	score := 0

	if ctx.ConversationSummary != nil {
		score += 3 // Summary is most valuable
	}

	if len(ctx.RecentMessages) > 0 {
		score += 2 // Recent messages are essential
	}

	if ctx.UserProfile != nil {
		score += 1 // Profile is helpful
	}

	if len(ctx.ConfirmedPreferences) > 0 {
		score += 2 // Confirmed preferences are valuable
	}

	if len(ctx.RelevantContacts) > 0 {
		score += 1 // Contact info is helpful
	}

	switch {
	case score >= 8:
		return "complete" // Have all major pieces
	case score >= 4:
		return "partial" // Have most pieces
	default:
		return "minimal" // Minimal context available
	}
}

// EstimateTokenCount estimates how many LLM tokens this context will use
// Used for budgeting and verification that context stays bounded
func (b *AnalysisContextBuilder) EstimateTokenCount(ctx *models.AnalysisContext) int {
	tokens := 0

	// Rough estimation: ~4 characters = 1 token
	// Summary: ~150-200 tokens
	if ctx.ConversationSummary != nil {
		summaryChars := len(ctx.ConversationSummary.Arc)
		for _, topic := range ctx.ConversationSummary.KeyTopics {
			summaryChars += len(topic)
		}
		for _, pattern := range ctx.ConversationSummary.UserPatterns {
			summaryChars += len(pattern)
		}
		tokens += (summaryChars / 4)
	}

	// Recent messages: ~200-250 tokens
	for _, msg := range ctx.RecentMessages {
		tokens += (len(msg.Content) / 4)
	}

	// Preferences: ~80-100 tokens
	for k, v := range ctx.ConfirmedPreferences {
		tokens += (len(k) / 4) + 5 // key + value + overhead
		if valStr, ok := v.(string); ok {
			tokens += (len(valStr) / 4)
		}
	}

	// Profile: ~20-50 tokens
	if ctx.UserProfile != nil {
		tokens += len(ctx.UserProfile.CommunicationStyle)/4 +
			len(ctx.UserProfile.PreferredTone)/4 +
			(5 * len(ctx.UserProfile.Goals))
	}

	// Contacts: ~30-50 tokens (if present)
	for _, contact := range ctx.RelevantContacts {
		tokens += (len(contact.Name) / 4) + len(contact.Relationship) + 10
	}

	return tokens
}

// BuildForConstitutionalEvaluation builds context optimized for constitutional evaluation
// Includes: summary + recent messages for understanding intent in context
func (b *AnalysisContextBuilder) BuildForConstitutionalEvaluation(
	userID string,
	conversationID string,
	currentMessage string,
	allMessages []models.Message,
) (*models.AnalysisContext, error) {
	// Use standard build - constitutional evaluation needs full context
	return b.BuildAnalysisContext(userID, conversationID, currentMessage, allMessages, nil)
}

// BuildForRiskAssessment builds context optimized for risk evaluation
// Includes: summary + patterns for detecting behavioral trends
func (b *AnalysisContextBuilder) BuildForRiskAssessment(
	userID string,
	conversationID string,
	currentMessage string,
	allMessages []models.Message,
	userProfile *models.AboutMe,
) (*models.AnalysisContext, error) {
	// Risk assessment benefits from user profile (patterns) and summary (trends)
	return b.BuildAnalysisContext(userID, conversationID, currentMessage, allMessages, userProfile)
}

// BuildForMessageClarity builds context optimized for clarity analysis
// Includes: summary + recent messages + user profile for tone understanding
func (b *AnalysisContextBuilder) BuildForMessageClarity(
	userID string,
	conversationID string,
	currentMessage string,
	allMessages []models.Message,
	userProfile *models.AboutMe,
) (*models.AnalysisContext, error) {
	// Clarity analysis needs user profile (communication style) and recent context
	return b.BuildAnalysisContext(userID, conversationID, currentMessage, allMessages, userProfile)
}

// BuildForIntentDetection builds context optimized for intent analysis
// Includes: summary + recent messages + confirmed preferences
func (b *AnalysisContextBuilder) BuildForIntentDetection(
	userID string,
	conversationID string,
	currentMessage string,
	allMessages []models.Message,
) (*models.AnalysisContext, error) {
	// Intent detection needs recent context and confirmed preferences
	return b.BuildAnalysisContext(userID, conversationID, currentMessage, allMessages, nil)
}

// VerifyContextBounds checks that context stays within token budget
// Budget: ~700-800 tokens total
// Returns true if within bounds, false if exceeds
func (b *AnalysisContextBuilder) VerifyContextBounds(ctx *models.AnalysisContext) (bool, int) {
	tokens := b.EstimateTokenCount(ctx)
	budget := 850 // 800 + 50 token buffer

	if tokens > budget {
		log.Printf("[AnalysisContextBuilder] ⚠️  Context exceeds budget: %d tokens (budget: %d)",
			tokens, budget)
		return false, tokens
	}

	log.Printf("[AnalysisContextBuilder] ✓ Context within budget: %d / %d tokens",
		tokens, budget)
	return true, tokens
}
