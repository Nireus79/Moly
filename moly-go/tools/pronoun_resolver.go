package tools

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"moly/database"
)

// PronounReference represents a pronoun found in a sentence
type PronounReference struct {
	Pronoun     string // "she", "he", "they", "it", "both"
	PronounType string // "personal", "demonstrative", "relative", "possessive"
	SentenceNum int    // Which sentence this pronoun appears in
	Position    int    // Position in sentence
	Confidence  float64
}

// PronounResolution represents a pronoun's mapping to its antecedent
type PronounResolution struct {
	ID               int64
	UserID           string
	ConversationID   string
	Pronoun          string
	PronounType      string
	AntecedentType   string // "name", "contact", "group", "concept", "unknown"
	AntecedentValue  string // "Christine", "my boss", "both of them"
	AntecedentID     *int64 // FK to contacts if applicable
	MessageID        string
	SentencePosition int
	Confidence       float64
	EvidenceText     string
	ResolutionMethod string // "linguistic_match", "llm_reasoning", "user_clarification", "context"
	ScopeStartSeq    int    // Message sequence number when valid from
	ScopeEndSeq      *int   // Message sequence when invalid (NULL = ongoing)
	IsActive         bool
	CreatedAt        int64
}

// PronounResolver handles pronoun detection and antecedent resolution
type PronounResolver struct {
	db *database.Database
}

// NewPronounResolver creates a new pronoun resolver
func NewPronounResolver(db *database.Database) *PronounResolver {
	return &PronounResolver{db: db}
}

// ============================================================================
// PRONOUN DETECTION
// ============================================================================

// DetectPronouns finds all pronouns in a message and returns their details
func (pr *PronounResolver) DetectPronouns(message string) []PronounReference {
	var pronouns []PronounReference

	// Personal pronouns
	personalPronouns := map[string]bool{
		"i": true, "me": true, "my": true, "mine": true,
		"you": true, "your": true, "yours": true,
		"he": true, "him": true, "his": true,
		"she": true, "her": true, "hers": true,
		"it": true, "its": true,
		"we": true, "us": true, "our": true, "ours": true,
		"they": true, "them": true, "their": true, "theirs": true,
	}

	// Find all pronouns in message
	words := strings.FieldsFunc(message, func(r rune) bool {
		return r == ' ' || r == '\n' || r == '\t' || r == ',' || r == '.' || r == '!' || r == '?'
	})

	for _, word := range words {
		lower := strings.ToLower(word)
		if personalPronouns[lower] {
			pronouns = append(pronouns, PronounReference{
				Pronoun:     lower,
				PronounType: "personal",
				Confidence:  0.95,
			})
		}
	}

	// Demonstrative pronouns
	demonstrativePronouns := []string{"this", "that", "these", "those"}
	for _, dem := range demonstrativePronouns {
		if strings.Contains(strings.ToLower(message), dem) {
			pronouns = append(pronouns, PronounReference{
				Pronoun:     dem,
				PronounType: "demonstrative",
				Confidence:  0.85,
			})
		}
	}

	return pronouns
}

// ============================================================================
// ANTECEDENT RESOLUTION
// ============================================================================

// ResolveAntecedent finds what a pronoun refers to based on context
// Checks: same sentence, previous sentence, recent messages, known contacts
func (pr *PronounResolver) ResolveAntecedent(
	userID string,
	conversationID string,
	pronoun string,
	currentSentenceNum int,
	allSentences []Sentence,
	recentMessages []string,
	knownContacts map[string]int64, // contact name → contact ID
) (*PronounResolution, error) {

	resolution := &PronounResolution{
		UserID:         userID,
		ConversationID: conversationID,
		Pronoun:        pronoun,
		PronounType:    "personal",
		Confidence:     0.0,
		IsActive:       true,
		CreatedAt:      time.Now().Unix(),
	}

	// Map pronouns to what they typically refer to
	pronounMapping := map[string]struct{ types []string }{
		"she":  {types: []string{"female", "woman", "girl"}},
		"her":  {types: []string{"female", "woman", "girl"}},
		"he":   {types: []string{"male", "man", "guy", "boy"}},
		"him":  {types: []string{"male", "man", "guy", "boy"}},
		"it":   {types: []string{"object", "concept"}},
		"they": {types: []string{"plural", "group"}},
		"them": {types: []string{"plural", "group"}},
		"both": {types: []string{"dual", "two people"}},
	}

	lower := strings.ToLower(pronoun)

	// Strategy 1: Check current sentence for explicit antecedent
	if currentSentenceNum > 0 && currentSentenceNum <= len(allSentences) {
		currentSentence := allSentences[currentSentenceNum-1]
		if antecedent := pr.extractAntecedentFromSentence(currentSentence.Text, lower, pronounMapping); antecedent != "" {
			resolution.AntecedentValue = antecedent
			resolution.AntecedentType = "name"
			resolution.Confidence = 0.90
			resolution.ResolutionMethod = "linguistic_match"
			resolution.EvidenceText = currentSentence.Text
			return resolution, nil
		}
	}

	// Strategy 2: Check immediately previous sentence
	if currentSentenceNum > 1 {
		prevSentence := allSentences[currentSentenceNum-2]
		if antecedent := pr.extractAntecedentFromSentence(prevSentence.Text, lower, pronounMapping); antecedent != "" {
			resolution.AntecedentValue = antecedent
			resolution.AntecedentType = "name"
			resolution.Confidence = 0.85
			resolution.ResolutionMethod = "linguistic_match"
			resolution.EvidenceText = prevSentence.Text
			return resolution, nil
		}
	}

	// Strategy 3: Check recent messages (last 2-3 messages)
	for i := len(recentMessages) - 1; i >= 0 && i >= len(recentMessages)-3; i-- {
		msg := recentMessages[i]
		if antecedent := pr.extractAntecedentFromMessage(msg, lower, pronounMapping); antecedent != "" {
			resolution.AntecedentValue = antecedent
			resolution.AntecedentType = "name"
			resolution.Confidence = 0.75
			resolution.ResolutionMethod = "linguistic_match"
			resolution.EvidenceText = msg
			return resolution, nil
		}
	}

	// Strategy 4: Check known contacts
	for contactName, contactID := range knownContacts {
		if pr.isLikelyAntecedent(contactName, lower, pronounMapping) {
			resolution.AntecedentValue = contactName
			resolution.AntecedentType = "contact"
			resolution.AntecedentID = &contactID
			resolution.Confidence = 0.70
			resolution.ResolutionMethod = "context"
			return resolution, nil
		}
	}

	// If we can't resolve, mark as unknown with low confidence
	resolution.AntecedentType = "unknown"
	resolution.AntecedentValue = "unknown"
	resolution.Confidence = 0.0
	resolution.ResolutionMethod = "unresolved"

	return resolution, nil
}

// extractAntecedentFromSentence looks for an antecedent in a sentence
// e.g., "Christine is submissive" + pronoun="she" → "Christine"
func (pr *PronounResolver) extractAntecedentFromSentence(
	sentence string,
	pronoun string,
	pronounMapping map[string]struct{ types []string },
) string {
	// Look for names (capitalized words) that match pronoun
	words := strings.Fields(sentence)
	for _, word := range words {
		// Skip pronouns themselves
		if strings.ToLower(word) == pronoun {
			continue
		}

		// Check if word looks like a name (starts with capital)
		if len(word) > 0 && word[0] >= 'A' && word[0] <= 'Z' {
			// Check if it matches the pronoun's semantic type
			if pronoun == "she" || pronoun == "her" {
				if pr.isFemaleName(word) {
					return word
				}
			}
			if pronoun == "he" || pronoun == "him" {
				if pr.isMaleName(word) {
					return word
				}
			}
			// For non-gendered pronouns, accept any name
			if pronoun == "it" || pronoun == "they" || pronoun == "them" {
				return word
			}
		}
	}

	return ""
}

// extractAntecedentFromMessage looks for an antecedent in a message
func (pr *PronounResolver) extractAntecedentFromMessage(
	message string,
	pronoun string,
	pronounMapping map[string]struct{ types []string },
) string {
	sentences := strings.Split(message, ".")
	if len(sentences) > 0 {
		// Use last sentence (most recent antecedent context)
		lastSentence := sentences[len(sentences)-1]
		return pr.extractAntecedentFromSentence(lastSentence, pronoun, pronounMapping)
	}
	return ""
}

// isLikelyAntecedent checks if a contact name is likely antecedent for a pronoun
func (pr *PronounResolver) isLikelyAntecedent(contactName string, pronoun string, pronounMapping map[string]struct{ types []string }) bool {
	if pronoun == "they" || pronoun == "them" {
		// Plural pronouns could refer to any group
		return true
	}

	if pronoun == "she" || pronoun == "her" {
		return pr.isFemaleName(contactName)
	}

	if pronoun == "he" || pronoun == "him" {
		return pr.isMaleName(contactName)
	}

	// For other pronouns, accept any contact
	return true
}

// ============================================================================
// NAME HEURISTICS (Simplified)
// ============================================================================

// isFemaleName uses simple heuristics to guess if a name is female
func (pr *PronounResolver) isFemaleName(name string) bool {
	lower := strings.ToLower(name)

	// Common female name endings
	femaleEndings := []string{"a", "e", "ie", "ia", "ine", "elle"}
	for _, ending := range femaleEndings {
		if strings.HasSuffix(lower, ending) {
			return true
		}
	}

	// Known female names
	femaleNames := map[string]bool{
		"christine": true, "sarah": true, "emma": true, "jessica": true,
		"ashley": true, "laura": true, "lisa": true, "karen": true,
		"deborah": true, "michelle": true, "susan": true, "kelly": true,
	}

	return femaleNames[lower]
}

// isMaleName uses simple heuristics to guess if a name is male
func (pr *PronounResolver) isMaleName(name string) bool {
	lower := strings.ToLower(name)

	// Common male name endings
	maleEndings := []string{"er", "or", "ar", "on"}
	for _, ending := range maleEndings {
		if strings.HasSuffix(lower, ending) {
			return true
		}
	}

	// Known male names
	maleNames := map[string]bool{
		"james": true, "robert": true, "michael": true, "john": true,
		"david": true, "charles": true, "richard": true, "joseph": true,
		"thomas": true, "christopher": true, "daniel": true, "matthew": true,
	}

	return maleNames[lower]
}

// ============================================================================
// SCOPE MANAGEMENT
// ============================================================================

// MarkScopeEnd marks when a pronoun resolution stops being valid
// Called when a new pronoun of the same type appears in a later message
func (pr *PronounResolver) MarkScopeEnd(resolution *PronounResolution, endMessageSeq int) error {
	resolution.ScopeEndSeq = &endMessageSeq
	resolution.IsActive = false

	// In Phase 5, this will be saved to database
	return nil
}

// IsResolutionValid checks if a pronoun resolution is still valid at a given message
func (pr *PronounResolver) IsResolutionValid(resolution *PronounResolution, currentMessageSeq int) bool {
	if !resolution.IsActive {
		return false
	}

	if currentMessageSeq < resolution.ScopeStartSeq {
		return false
	}

	if resolution.ScopeEndSeq != nil && currentMessageSeq >= *resolution.ScopeEndSeq {
		return false
	}

	return true
}

// ============================================================================
// DATABASE INTEGRATION (Phase 5)
// ============================================================================

// SaveResolution saves a pronoun resolution to the database
// (Implemented in Phase 5 when database integration is wired)
// FIX #19: SaveResolution now persists pronoun mappings to database
func (pr *PronounResolver) SaveResolution(resolution *PronounResolution) error {
	if pr.db == nil {
		return fmt.Errorf("database not initialized")
	}

	if resolution == nil {
		return fmt.Errorf("resolution cannot be nil")
	}

	conn := pr.db.GetConnection()
	if conn == nil {
		return fmt.Errorf("database connection unavailable")
	}

	// Set timestamp if not already set
	if resolution.CreatedAt == 0 {
		resolution.CreatedAt = time.Now().Unix()
	}

	// Insert or update pronoun resolution
	query := `
		INSERT INTO pronoun_resolutions (
			user_id, conversation_id, pronoun, pronoun_type,
			antecedent_type, antecedent_value, antecedent_id,
			message_id, sentence_position, confidence,
			evidence_text, resolution_method, scope_start_seq,
			scope_end_seq, is_active, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := conn.Exec(query,
		resolution.UserID,
		resolution.ConversationID,
		resolution.Pronoun,
		resolution.PronounType,
		resolution.AntecedentType,
		resolution.AntecedentValue,
		resolution.AntecedentID,
		resolution.MessageID,
		resolution.SentencePosition,
		resolution.Confidence,
		resolution.EvidenceText,
		resolution.ResolutionMethod,
		resolution.ScopeStartSeq,
		resolution.ScopeEndSeq,
		resolution.IsActive,
		resolution.CreatedAt,
	)

	if err != nil {
		log.Printf("[PronounResolver] Error saving resolution: %v", err)
		return fmt.Errorf("failed to save pronoun resolution: %w", err)
	}

	id, err := result.LastInsertId()
	if err == nil && resolution.ID == 0 {
		resolution.ID = id
	}

	log.Printf("[PronounResolver] ✓ FIX #19: Saved pronoun resolution (pronoun=%s, antecedent=%s, confidence=%.2f)",
		resolution.Pronoun, resolution.AntecedentValue, resolution.Confidence)

	return nil
}

// FIX #19: GetResolutionsForPronoun now queries database for pronoun mappings
// Retrieves all active resolutions for a pronoun (ordered by recency)
func (pr *PronounResolver) GetResolutionsForPronoun(userID string, pronoun string) ([]*PronounResolution, error) {
	if pr.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	conn := pr.db.GetConnection()
	if conn == nil {
		return nil, fmt.Errorf("database connection unavailable")
	}

	// Query active pronoun resolutions for this user, ordered by most recent first
	query := `
		SELECT
			id, user_id, conversation_id, pronoun, pronoun_type,
			antecedent_type, antecedent_value, antecedent_id,
			message_id, sentence_position, confidence,
			evidence_text, resolution_method, scope_start_seq,
			scope_end_seq, is_active, created_at
		FROM pronoun_resolutions
		WHERE user_id = ? AND pronoun = ? AND is_active = true
		ORDER BY created_at DESC
		LIMIT 10
	`

	rows, err := conn.Query(query, userID, pronoun)
	if err != nil {
		log.Printf("[PronounResolver] Error querying pronouns: %v", err)
		return nil, fmt.Errorf("failed to query pronoun resolutions: %w", err)
	}
	defer rows.Close()

	var resolutions []*PronounResolution
	for rows.Next() {
		resolution := &PronounResolution{}
		err := rows.Scan(
			&resolution.ID,
			&resolution.UserID,
			&resolution.ConversationID,
			&resolution.Pronoun,
			&resolution.PronounType,
			&resolution.AntecedentType,
			&resolution.AntecedentValue,
			&resolution.AntecedentID,
			&resolution.MessageID,
			&resolution.SentencePosition,
			&resolution.Confidence,
			&resolution.EvidenceText,
			&resolution.ResolutionMethod,
			&resolution.ScopeStartSeq,
			&resolution.ScopeEndSeq,
			&resolution.IsActive,
			&resolution.CreatedAt,
		)
		if err != nil {
			log.Printf("[PronounResolver] Error scanning resolution row: %v", err)
			continue
		}
		resolutions = append(resolutions, resolution)
	}

	if len(resolutions) > 0 {
		log.Printf("[PronounResolver] ✓ FIX #19: Retrieved %d active resolutions for pronoun '%s'",
			len(resolutions), pronoun)
	}

	return resolutions, nil
}

// ============================================================================
// HELPER: Detect group pronouns
// ============================================================================

// IsGroupPronoun checks if a pronoun refers to a group
func (pr *PronounResolver) IsGroupPronoun(pronoun string) bool {
	lower := strings.ToLower(pronoun)
	groupPronouns := map[string]bool{
		"they": true, "them": true, "their": true, "theirs": true,
		"we": true, "us": true, "our": true, "ours": true,
		"both": true, "all": true,
	}
	return groupPronouns[lower]
}

// DetectGroupMembers analyzes text to determine group membership
// Returns: list of entity names that should be in the group
func (pr *PronounResolver) DetectGroupMembers(text string, knownEntities []string) []string {
	var members []string

	lower := strings.ToLower(text)

	// Check for "both" - implies exactly 2 people
	if strings.Contains(lower, "both") || strings.Contains(lower, "we") {
		// Look for "me and X" or "X and I" patterns
		mePattern := regexp.MustCompile(`(?i)(me|I|myself)\s+and\s+([a-zA-Z]+)`)
		if matches := mePattern.FindStringSubmatch(lower); len(matches) > 2 {
			members = append(members, "user")
			members = append(members, matches[2])
		}

		// Look for "X and Y" patterns
		andPattern := regexp.MustCompile(`(?i)([a-zA-Z]+)\s+and\s+([a-zA-Z]+)`)
		if matches := andPattern.FindStringSubmatch(lower); len(matches) > 2 {
			members = append(members, matches[1])
			members = append(members, matches[2])
		}
	}

	// Check for "all" - implies everyone/all entities
	if strings.Contains(lower, "all of") || strings.Contains(lower, "all of us") {
		members = knownEntities
	}

	return members
}
