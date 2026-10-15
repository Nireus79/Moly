package database

import (
	"fmt"
	"log"
	"strings"
	"unicode"

	"moly/models"
)

// AnalysisContextBuilder constructs bounded context for evaluators
// Combines: summary + recent messages + preferences + profile
// Total: ~700-800 tokens (scalable regardless of conversation length)
type AnalysisContextBuilder struct {
	summaryRepo        *ConversationSummaryRepository
	messageSummaryRepo *MessageSummaryRepository // FIX #11: For loading cached summaries
	chatRepo           *ChatMessageRepository
	contextAttrRepo    *ContextAttributeRepository
	db                 *Database
}

// NewAnalysisContextBuilder creates a new builder
func NewAnalysisContextBuilder(
	db *Database,
	summaryRepo *ConversationSummaryRepository,
	chatRepo *ChatMessageRepository,
	contextAttrRepo *ContextAttributeRepository,
) *AnalysisContextBuilder {
	return &AnalysisContextBuilder{
		summaryRepo:        summaryRepo,
		messageSummaryRepo: NewMessageSummaryRepository(db.GetConnection()), // FIX #11
		chatRepo:           chatRepo,
		contextAttrRepo:    contextAttrRepo,
		db:                 db,
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
//
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

	// 2. Extract recent message window (dynamic sizing to prevent message gaps)
	// Ensure no messages fall between summary end and recent window start
	windowSize := 3 // Minimum window
	if summary != nil && summary.MessageCount > 0 {
		// If summary covers N messages, recent window must start at or before N+1
		messagesSinceSummary := len(allMessages) - summary.MessageCount
		if messagesSinceSummary > windowSize {
			windowSize = messagesSinceSummary
		}
	}
	recentMessages := b.extractRecentMessageWindow(allMessages, windowSize)
	ctx.RecentMessages = recentMessages
	log.Printf("[AnalysisContextBuilder] ✓ Extracted %d recent messages (window=%d) for context", len(recentMessages), windowSize)

	// FIX #11: Load message summaries for recent messages (Phase 2 optimization)
	// Enables layers to skip re-extraction of known entities
	if b.messageSummaryRepo != nil {
		recentSummaries, err := b.messageSummaryRepo.GetRecentMessageSummaries(conversationID, windowSize)
		if err != nil {
			log.Printf("[AnalysisContextBuilder] Warning: failed to load message summaries: %v", err)
		} else if len(recentSummaries) > 0 {
			// Convert to interface{} slice for storage
			var summaryInterfaces []interface{}
			for _, s := range recentSummaries {
				summaryInterfaces = append(summaryInterfaces, s)
			}
			ctx.RecentMessageSummaries = summaryInterfaces
			log.Printf("[AnalysisContextBuilder] FIX #11: ✓ Loaded %d recent message summaries for re-extraction optimization", len(recentSummaries))
		}
	}

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
	relevantContacts := b.extractRelevantContacts(userID, conversationID, allMessages)
	ctx.RelevantContacts = relevantContacts
	log.Printf("[AnalysisContextBuilder] ✓ Extracted %d relevant contacts", len(relevantContacts))
	ctx.Dialogue = dialogueFrom(ctx, allMessages, currentMessage)

	// 6. Determine context quality
	ctx.ContextQuality = b.assessContextQuality(ctx)

	log.Printf("[AnalysisContextBuilder] ✓ Context complete (quality: %s)", ctx.ContextQuality)
	return ctx, nil
}

// EnhanceWithExtractionArtifact adds Phase 0 extraction data to AnalysisContext
// Phase 5 integration: Embeds extraction metadata for all layers to use
func (b *AnalysisContextBuilder) EnhanceWithExtractionArtifact(
	ctx *models.AnalysisContext,
	artifact *models.ExtractionArtifact,
) *models.AnalysisContext {

	if artifact == nil {
		return ctx
	}

	if ctx == nil {
		ctx = &models.AnalysisContext{}
	}

	// Populate extraction data from artifact
	ctx.ExtractedEntities = artifact.Entities
	ctx.ExtractedSource = artifact.Source
	ctx.ExtractedConfidence = artifact.AverageConfidence
	ctx.ExtractionDuration = artifact.Duration

	// Extract contacts from high-confidence contact entities
	// Use ExtractedContacts field to avoid duplicating RelevantContacts
	// (Phase 5 integration: separate field for extraction-derived contacts)
	extractedContactMap := make(map[string]*models.Contact)
	for _, entity := range artifact.Entities {
		if entity.Type == "contact" && entity.Confidence >= 0.7 {
			key := entity.Value // Use name as dedup key
			existing, found := extractedContactMap[key]
			if !found || entity.Confidence > existing.Confidence {
				contact := &models.Contact{
					Name:       entity.Value,
					Confidence: entity.Confidence,
				}
				extractedContactMap[key] = contact
			}
		}
	}

	// Add deduplicated extraction contacts (don't add to ctx.Contacts to avoid duplicating RelevantContacts)
	// Instead populate the separate extraction fields
	for _, contact := range extractedContactMap {
		ctx.Contacts = append(ctx.Contacts, *contact)
	}

	// Extract preferences and characteristics
	var prefs []string
	var chars []string
	for _, entity := range artifact.Entities {
		if entity.Confidence >= 0.70 {
			switch entity.Type {
			case "preference", "negation":
				prefs = append(prefs, entity.Value)
			case "characteristic":
				chars = append(chars, entity.Value)
			}
		}
	}

	if len(prefs) > 0 {
		ctx.ExtractedPreferences = prefs
	}
	if len(chars) > 0 {
		ctx.ExtractedCharacteristics = chars
	}

	// Set extraction quality indicators
	ctx.ExtractionQuality = &models.ExtractionQuality{
		SubjectAttributed: artifact.SubjectAttributed,
		NegationPreserved: artifact.NegationPreserved,
		LLMExtraction:     artifact.LLMSuccess,
	}

	log.Printf("[AnalysisContextBuilder] ✓ Enhanced with ExtractionArtifact (Phase 5)")
	log.Printf("[AnalysisContextBuilder]   - %d entities (source=%s, confidence=%.2f)",
		len(artifact.Entities), artifact.Source, artifact.AverageConfidence)
	log.Printf("[AnalysisContextBuilder]   - Subject: %v, Negation: %v, LLM: %v",
		artifact.SubjectAttributed, artifact.NegationPreserved, artifact.LLMSuccess)

	return ctx
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

	// Check for errors from row iteration
	if err := rows.Err(); err != nil {
		log.Printf("[AnalysisContextBuilder] Error iterating preferences: %v", err)
		return nil, fmt.Errorf("error iterating preferences: %w", err)
	}

	if len(prefs) == 0 {
		return nil, nil
	}

	return prefs, nil
}

// extractRelevantContacts returns the active contacts that the conversation is about (the user's named people and this conversation's
// own unnamed people, never another conversation's): those whose name (or the label the user
// first used for them) appears in any message of the conversation, Moly's included. It compares the names the user already
// has with the text; it does not guess names from capital letters, and it does not look at only the last two messages (found
// live: on the answer "Yes" no name was in the last two messages, so the people block of the gap prompt was empty).
func (b *AnalysisContextBuilder) extractRelevantContacts(userID, conversationID string, allMessages []models.Message) []models.Contact {
	if len(allMessages) == 0 {
		return []models.Contact{}
	}
	contacts, err := NewContactRepository(b.db).InConversation(conversationID).GetByUserID(userID)
	if err != nil {
		log.Printf("[AnalysisContextBuilder] Warning: Failed to load contacts: %v", err)
		return []models.Contact{}
	}
	var relevant []models.Contact
	for _, c := range contacts {
		if c.Status != "active" || strings.TrimSpace(c.Name) == "" {
			continue
		}
		for _, m := range allMessages {
			if mentionsName(m.Content, c.Name) {
				relevant = append(relevant, *c)
				break
			}
		}
	}
	return relevant
}

// EstablishedPeople are the people a conversation has already brought up: the people the user has named whose name appears in
// the given messages, and the conversation's own people who have no name yet (by label). repo must be the conversation's
// repository (ContactRepository.InConversation), so no other conversation's unnamed person can be among them. A saved person
// the conversation has not mentioned is not established: only the name the user gives can match them.
func EstablishedPeople(repo *ContactRepository, userID string, messages []string) []*models.Contact {
	contacts, err := repo.GetByUserID(userID)
	if err != nil {
		log.Printf("[Established] Could not load people: %v", err)
		return nil
	}
	var people []*models.Contact
	for _, c := range contacts {
		if c.Status != "active" || strings.TrimSpace(c.Name) == "" {
			continue
		}
		if c.NameStatus != NameStatusNamed {
			people = append(people, c)
			continue
		}
		for _, m := range messages {
			if mentionsName(m, c.Name) {
				people = append(people, c)
				break
			}
		}
	}
	return people
}

// JudgeLines write the established people for the model that decides who a phrase refers to: the name, and the role the user
// gave ("Dana (the user's boss)"), so that "my boss" can be recognised as Dana once the conversation has said so.
func JudgeLines(people []*models.Contact) []string {
	lines := make([]string, 0, len(people))
	for _, c := range people {
		if c.NameStatus == NameStatusNamed && c.RelationsText() != "" {
			lines = append(lines, c.Name+" ("+c.RelationsText()+")")
		} else if c.NameStatus == NameStatusNamed && c.ContactRole != "" {
			lines = append(lines, c.Name+" (the user's "+c.ContactRole+")")
		} else {
			lines = append(lines, c.Name)
		}
	}
	return lines
}

// mentionsName reports whether text contains name as a whole word or phrase, ignoring case.
func mentionsName(text, name string) bool {
	t, n := strings.ToLower(text), strings.ToLower(strings.TrimSpace(name))
	if n == "" {
		return false
	}
	for from := 0; ; {
		i := strings.Index(t[from:], n)
		if i < 0 {
			return false
		}
		start, end := from+i, from+i+len(n)
		before := start == 0 || !isWordRune(rune(t[start-1]))
		after := end >= len(t) || !isWordRune(rune(t[end]))
		if before && after {
			return true
		}
		from = start + 1
	}
}

func isWordRune(r rune) bool { return unicode.IsLetter(r) || unicode.IsNumber(r) }

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

	// Fix H: Align with main.go contextQuality names for consistency
	switch {
	case score >= 8:
		return "comprehensive" // Have all major pieces
	case score >= 4:
		return "partial" // Have most pieces
	default:
		return "minimal" // Minimal context available
	}
}

// dialogueFrom writes the packet from what the builder loaded: the stored summary, the people the conversation is about with
// what they are to the user, and Moly's last message. The user's message before it is added only while the summary has not
// caught up with it (it covers fewer messages than came before this one). The caller adds the goal and the open questions.
func dialogueFrom(ctx *models.AnalysisContext, allMessages []models.Message, current string) *models.DialoguePacket {
	p := &models.DialoguePacket{}
	older := models.MessagesBefore(allMessages, current)
	p.MolyLast, p.UserBefore = models.LastMolyMessage(older)
	p.SetUserSaid(older)
	covered := 0
	if s := ctx.ConversationSummary; s != nil {
		p.Summary = s.Arc
		covered = s.MessageCount
	}
	if covered >= len(older) {
		p.UserBefore = "" // the summary already holds it
	}
	for _, c := range ctx.RelevantContacts {
		p.People = append(p.People, PersonLine(&c))
	}
	return p
}

// PersonLine says who a contact is to the user: the name, and the role the user gave ("the user's manager"), or the relationship.
func PersonLine(c *models.Contact) string {
	switch {
	case c.NameStatus == NameStatusNameless:
		return c.Name + " (the user gave no name)"
	case c.NameStatus != "" && c.NameStatus != NameStatusNamed:
		return c.Name + " (name not given yet)"
	case c.RelationsText() != "":
		return c.Name + " (" + c.RelationsText() + ")"
	case c.ContactRole != "":
		return c.Name + " (the user's " + c.ContactRole + ")"
	case c.Relationship != "" && c.Relationship != "other":
		return c.Name + " (" + c.Relationship + ")"
	}
	return c.Name
}
