package tools

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// MultiPersonSegment represents a section of text attributed to a specific person
type MultiPersonSegment struct {
	Subject string // "user", "she", "he", contact name, etc.
	Text    string // The text attributed to this person
	Start   int    // Position in original message
	End     int    // Position in original message
}

// ExtractionResult represents a single extracted entity with its subject and confidence
type ExtractionResult struct {
	Subject    string  // "user", "she", "he", "Christine", etc.
	Property   string  // The extracted property/attribute
	Type       string  // "preference", "characteristic", "negation", "structured", etc.
	Confidence float64 // 0.0-1.0
	RawMatch   string  // The matched phrase from the message
	IsNegated  bool    // true if the property is negated (e.g., "don't focus on X")
}

// Sentence represents a single sentence with its position and properties
type Sentence struct {
	Number   int    // 1-based sentence number
	Text     string // The sentence text
	StartPos int    // Character position in original message
	EndPos   int    // Character position in original message
}

// SentenceAnalysis represents the Subject-Verb-Object analysis of a sentence
type SentenceAnalysis struct {
	SentenceNumber int     // Which sentence in the message
	SentenceText   string  // The actual sentence
	Subject        string  // Raw subject from sentence: "I", "she", "Christine"
	SubjectType    string  // "pronoun", "name", "group"
	Verb           string  // The main verb: "am", "is", "like", "want"
	VerbType       string  // "copula", "transitive", "intransitive", "phrasal"
	VerbNegated    bool    // True if verb is negated
	Object         string  // The object: "dominant", "communication"
	ObjectType     string  // "adjective", "noun", "noun_phrase"
	Negated        bool    // Is the entire sentence negated?
	Confidence     float64 // 0.0-1.0 confidence in analysis
	ParsingMethod  string  // "regex", "llm", "hybrid"
	CreatedAt      int64   // Unix timestamp
}

// LinguisticParser handles grammar-based entity extraction
type LinguisticParser struct {
	// Compiled regex patterns for each rule
	ruleIsAdjective   *regexp.Regexp // "I am {adj}", "She is {adj}"
	ruleIsNegative    *regexp.Regexp // "I am NOT {adj}", "I'm not {adj}"
	ruleLikeDislike   *regexp.Regexp // "I like/love/want {noun}", "I don't like/want {noun}"
	ruleNotInterested *regexp.Regexp // "I'm not interested in {X}", "not interested in {X}"
	ruleNamedIs       *regexp.Regexp // "{Name} is {property}"
	ruleNamedVerb     *regexp.Regexp // "{Name} likes/wants {property}"
	ruleStructured    *regexp.Regexp // "Key: Value" format
	rulePrefer        *regexp.Regexp // "I prefer {noun}"
	ruleLookingFor    *regexp.Regexp // "I'm looking for {noun}"
}

// NewLinguisticParser creates a new linguistic parser with compiled patterns
func NewLinguisticParser() *LinguisticParser {
	return &LinguisticParser{
		// Rule 1: Subject + "is" + Adjective
		// Matches: "I am dominant", "I'm submissive", "She is female", etc.
		ruleIsAdjective: regexp.MustCompile(
			`(?i)\b(I|I'm|i am|she|he|they|who)\s+(?:am\s+)?(?:is\s+)?` +
				`(dominant|submissive|active|passive|top|bottom|switch|bisexual|asexual|` +
				`pansexual|heterosexual|homosexual|experienced|inexperienced|monogamous|` +
				`polyamorous|interested|uninterested|open-minded|conservative|vanilla|` +
				`kinky|curious|direct|discreet|female|male|man|woman|non-binary|agender|` +
				`confident|cautious|adventurous|reserved|generous|selfish|caring|cold|` +
				`communicative|silent|engaged|detached|ready|hesitant)\b`,
		),

		// Rule 2: Negated preferences - "I am NOT", "I don't", "I'm not"
		// Preserves the negation
		ruleIsNegative: regexp.MustCompile(
			`(?i)\b(I|i|she|he|they)\s+(?:am\s+)?(?:is\s+)?` +
				`(?:not|n't)\s+(?:interested\s+in\s+)?` +
				`(interested in|want|like|seeking|looking for)?\s*([a-zA-Z0-9\s\-]+?)(?:\.|,|!|\?|$)`,
		),

		// Rule 3: Like/Love/Want + Object
		// Matches: "I like bondage", "I love cuddling", "I want casual sex"
		ruleLikeDislike: regexp.MustCompile(
			`(?i)\b(I|I'm|she|he|they|we)\s+` +
				`(like|love|want|enjoy|prefer|seek|seeking|looking for)\s+` +
				`([a-zA-Z0-9\s\-\.]+?)(?:\.|,|!|\?|and|or|but|$)`,
		),

		// Rule 4: "not interested in" - Explicit negation
		ruleNotInterested: regexp.MustCompile(
			`(?i)\b(I'm\s+)?(?:not\s+)?interested\s+in\s+([a-zA-Z0-9\s\-\.]+?)(?:\.|,|!|\?|$)`,
		),

		// Rule 5: Named subject + is + property
		// Matches: "Christine is 39", "Se is submissive", "John is experienced"
		ruleNamedIs: regexp.MustCompile(
			`(?i)\b([A-Z][a-z]+)\s+(?:is|are)\s+` +
				`([a-zA-Z0-9\s\-\.]+?)(?:\.|,|!|\?|$)`,
		),

		// Rule 6: Named subject + verb + property
		// Matches: "Christine likes bondage", "Se wants aftercare"
		ruleNamedVerb: regexp.MustCompile(
			`(?i)\b([A-Z][a-z]+)\s+` +
				`(like|love|want|enjoy|prefer|seek)\s+` +
				`([a-zA-Z0-9\s\-\.]+?)(?:\.|,|!|\?|$)`,
		),

		// Rule 7: Structured format - "Key: Value"
		// Matches: "Genders: Female", "Roles: submissive", "Into: Bondage"
		ruleStructured: regexp.MustCompile(
			`(?i)(Genders?|Roles?|Into|Interests|Kinks|Traits|Ages?|Locations?|Status):\s*([^\n]+?)(?:\n|$)`,
		),

		// Rule 8: "I prefer" + noun
		// Matches: "I prefer direct communication", "I prefer experienced partners"
		rulePrefer: regexp.MustCompile(
			`(?i)\b(I|I'm|we|we're)\s+prefer\s+([a-zA-Z0-9\s\-\.]+?)(?:\.|,|!|\?|$)`,
		),

		// Rule 9: "looking for" + noun
		// Matches: "I'm looking for a long-term relationship", "looking for experienced partners"
		ruleLookingFor: regexp.MustCompile(
			`(?i)\b(?:I'm|I am|looking for)\s+(?:a\s+)?([a-zA-Z0-9\s\-\.]+?)(?:\.|,|!|\?|$)`,
		),
	}
}

// Parse analyzes a message and extracts all entities with their subjects
func (lp *LinguisticParser) Parse(message string) []ExtractionResult {
	results := []ExtractionResult{}

	// Clean and normalize message
	cleaned := strings.TrimSpace(message)

	// Rule 1: Subject + "is" + Adjective
	results = append(results, lp.extractIsAdjective(cleaned)...)

	// Rule 2: Negated properties
	results = append(results, lp.extractNegated(cleaned)...)

	// Rule 3: Like/Dislike + Object
	results = append(results, lp.extractLikeDislike(cleaned)...)

	// Rule 4: "not interested in" (explicit)
	results = append(results, lp.extractNotInterested(cleaned)...)

	// Rule 5: Named subject + is
	results = append(results, lp.extractNamedIs(cleaned)...)

	// Rule 6: Named subject + verb
	results = append(results, lp.extractNamedVerb(cleaned)...)

	// Rule 7: Structured format
	results = append(results, lp.extractStructured(cleaned)...)

	// Rule 8: "I prefer"
	results = append(results, lp.extractPrefer(cleaned)...)

	// Rule 9: "looking for"
	results = append(results, lp.extractLookingFor(cleaned)...)

	// Rule 10: Focus directives
	results = append(results, lp.extractFocusDirective(cleaned)...)

	// Rule 11: Constraints
	results = append(results, lp.extractConstraint(cleaned)...)

	// Rule 12: Priorities
	results = append(results, lp.extractPriority(cleaned)...)

	// Rule 13: Negated directives (meta-instructions)
	results = append(results, lp.extractNegatedDirective(cleaned)...)

	// Dedup: remove duplicates (same subject + property)
	return lp.dedup(results)
}

// DetectMultiPersonBoundaries detects and segments multi-person text
// Returns segments with identified subjects (user, she, he, contact names, etc)
// Sept 30, 2026: Added to handle "Here are some from her's" style messages
func (lp *LinguisticParser) DetectMultiPersonBoundaries(message string) []MultiPersonSegment {
	var segments []MultiPersonSegment
	lower := strings.ToLower(message)

	// Patterns for boundary markers
	// "Here are some from her's", "Her profile:", "His info:", etc.
	boundaryPatterns := []struct {
		pattern string
		subject string
	}{
		{`here\s+are\s+some\s+from\s+her`, "she"},
		{`here\s+are\s+some\s+from\s+his`, "he"},
		{`here\s+are\s+some\s+from\s+their`, "they"},
		{`her\s+profile\s*:`, "she"},
		{`her\s+info\s*:`, "she"},
		{`his\s+profile\s*:`, "he"},
		{`his\s+info\s*:`, "he"},
		{`their\s+profile\s*:`, "they"},
		{`their\s+info\s*:`, "they"},
		{`here\s+are\s+some\s+insights\s+from\s+my\s+profile`, "user"},
		{`my\s+profile\s*:`, "user"},
		{`my\s+info\s*:`, "user"},
	}

	positions := []struct {
		pos     int
		subject string
	}{}

	// Find all boundary markers
	for _, bp := range boundaryPatterns {
		re := regexp.MustCompile(bp.pattern)
		matches := re.FindAllStringIndex(lower, -1)
		for _, match := range matches {
			positions = append(positions, struct {
				pos     int
				subject string
			}{match[0], bp.subject})
		}
	}

	// If no boundaries detected, treat entire message as from one person
	if len(positions) == 0 {
		return []MultiPersonSegment{
			{
				Subject: "user",
				Text:    message,
				Start:   0,
				End:     len(message),
			},
		}
	}

	// Sort by position
	for i := 0; i < len(positions); i++ {
		for j := i + 1; j < len(positions); j++ {
			if positions[i].pos > positions[j].pos {
				positions[i], positions[j] = positions[j], positions[i]
			}
		}
	}

	// Create segments between boundaries
	currentPos := 0
	for i, pos := range positions {
		// Check if this is a starting boundary (not end of message)
		if pos.pos > currentPos {
			// Add segment BEFORE this boundary (attributed to previous subject or user)
			prevSubject := "user"
			if i > 0 {
				prevSubject = positions[i-1].subject
			}
			segments = append(segments, MultiPersonSegment{
				Subject: prevSubject,
				Text:    strings.TrimSpace(message[currentPos:pos.pos]),
				Start:   currentPos,
				End:     pos.pos,
			})
		}
		currentPos = pos.pos
	}

	// Add remaining text
	if currentPos < len(message) {
		lastSubject := "user"
		if len(positions) > 0 {
			lastSubject = positions[len(positions)-1].subject
		}
		remaining := strings.TrimSpace(message[currentPos:])
		if remaining != "" {
			segments = append(segments, MultiPersonSegment{
				Subject: lastSubject,
				Text:    remaining,
				Start:   currentPos,
				End:     len(message),
			})
		}
	}

	// Filter out empty segments
	var filtered []MultiPersonSegment
	for _, seg := range segments {
		if strings.TrimSpace(seg.Text) != "" {
			filtered = append(filtered, seg)
		}
	}

	if len(filtered) == 0 {
		return []MultiPersonSegment{
			{
				Subject: "user",
				Text:    message,
				Start:   0,
				End:     len(message),
			},
		}
	}

	return filtered
}

// extractIsAdjective handles "I am dominant", "She is submissive" patterns
func (lp *LinguisticParser) extractIsAdjective(message string) []ExtractionResult {
	var results []ExtractionResult
	matches := lp.ruleIsAdjective.FindAllStringSubmatchIndex(message, -1)

	for _, match := range matches {
		if len(match) >= 6 {
			subject := strings.ToLower(strings.TrimSpace(message[match[2]:match[3]]))
			property := strings.ToLower(strings.TrimSpace(message[match[4]:match[5]]))

			// Normalize subject
			subject = normalizeSubject(subject)

			results = append(results, ExtractionResult{
				Subject:    subject,
				Property:   property,
				Type:       "characteristic",
				Confidence: 0.90,
				RawMatch:   strings.TrimSpace(message[match[0]:match[1]]),
			})
		}
	}
	return results
}

// extractNegated handles "I am NOT interested", "I'm not submissive" patterns
func (lp *LinguisticParser) extractNegated(message string) []ExtractionResult {
	var results []ExtractionResult

	// Pattern: "I am not {property}" / "I'm not {property}"
	negPattern := regexp.MustCompile(`(?i)\b(I|i|she|he|they)\s+(?:am\s+)?(?:is\s+)?(?:n't|not)\s+([a-zA-Z0-9\s\-]+?)(?:\.|,|!|\?|$)`)
	matches := negPattern.FindAllStringSubmatchIndex(message, -1)

	for _, match := range matches {
		if len(match) >= 6 {
			subject := strings.ToLower(strings.TrimSpace(message[match[2]:match[3]]))
			property := strings.ToLower(strings.TrimSpace(message[match[4]:match[5]]))

			subject = normalizeSubject(subject)

			results = append(results, ExtractionResult{
				Subject:    subject,
				Property:   "NOT " + property,
				Type:       "negation",
				Confidence: 0.85,
				RawMatch:   strings.TrimSpace(message[match[0]:match[1]]),
			})
		}
	}

	return results
}

// extractLikeDislike handles "I like bondage", "I don't want casual sex" patterns
func (lp *LinguisticParser) extractLikeDislike(message string) []ExtractionResult {
	var results []ExtractionResult

	// Handle positive preferences: "I like X", "I want X"
	posPattern := regexp.MustCompile(`(?i)\b(I|I'm|she|he|they|we)\s+(like|love|want|enjoy|prefer|seek)\s+([a-zA-Z0-9\s\-\.]+?)(?:\.|,|!|\?|and|or|but|$)`)
	matches := posPattern.FindAllStringSubmatchIndex(message, -1)

	for _, match := range matches {
		if len(match) >= 8 {
			subject := strings.ToLower(strings.TrimSpace(message[match[2]:match[3]]))
			property := strings.ToLower(strings.TrimSpace(message[match[6]:match[7]]))

			subject = normalizeSubject(subject)

			results = append(results, ExtractionResult{
				Subject:    subject,
				Property:   property,
				Type:       "preference",
				Confidence: 0.85,
				RawMatch:   strings.TrimSpace(message[match[0]:match[1]]),
			})
		}
	}

	// Handle negative preferences: "I don't like X"
	negPattern := regexp.MustCompile(`(?i)\b(I|I'm|she|he|they|we)\s+(?:don't|doesn't|do not)\s+(like|want|enjoy|prefer)\s+([a-zA-Z0-9\s\-\.]+?)(?:\.|,|!|\?|$)`)
	negMatches := negPattern.FindAllStringSubmatchIndex(message, -1)

	for _, match := range negMatches {
		if len(match) >= 8 {
			subject := strings.ToLower(strings.TrimSpace(message[match[2]:match[3]]))
			property := strings.ToLower(strings.TrimSpace(message[match[6]:match[7]]))

			subject = normalizeSubject(subject)

			results = append(results, ExtractionResult{
				Subject:    subject,
				Property:   "NOT " + property,
				Type:       "negation",
				Confidence: 0.85,
				RawMatch:   strings.TrimSpace(message[match[0]:match[1]]),
			})
		}
	}

	return results
}

// extractNotInterested handles "not interested in X" patterns
func (lp *LinguisticParser) extractNotInterested(message string) []ExtractionResult {
	var results []ExtractionResult

	// Pattern: "not interested in {X}" or "I'm not interested in {X}"
	pattern := regexp.MustCompile(`(?i)(?:\b(?:I|I'm|I am)\s+)?(?:not\s+)?interested\s+in\s+([a-zA-Z0-9\s\-\.]+?)(?:\.|,|!|\?|$)`)
	matches := pattern.FindAllStringSubmatchIndex(message, -1)

	for _, match := range matches {
		if len(match) >= 4 {
			rawPhrase := strings.TrimSpace(message[match[0]:match[1]])

			// Only count as negation if it contains "not" or "n't"
			isNegated := strings.Contains(strings.ToLower(rawPhrase), "not") || strings.Contains(strings.ToLower(rawPhrase), "n't")

			property := strings.ToLower(strings.TrimSpace(message[match[2]:match[3]]))

			propType := "preference"
			propValue := property
			confidence := 0.80

			if isNegated {
				propValue = "NOT " + property
				propType = "negation"
				confidence = 0.85
			}

			results = append(results, ExtractionResult{
				Subject:    "user",
				Property:   propValue,
				Type:       propType,
				Confidence: confidence,
				RawMatch:   rawPhrase,
			})
		}
	}

	return results
}

// extractNamedIs handles "Christine is 39", "Se is submissive" patterns
func (lp *LinguisticParser) extractNamedIs(message string) []ExtractionResult {
	var results []ExtractionResult
	matches := lp.ruleNamedIs.FindAllStringSubmatchIndex(message, -1)

	for _, match := range matches {
		if len(match) >= 6 {
			subject := strings.ToLower(strings.TrimSpace(message[match[2]:match[3]]))
			property := strings.ToLower(strings.TrimSpace(message[match[4]:match[5]]))

			results = append(results, ExtractionResult{
				Subject:    subject,
				Property:   property,
				Type:       "characteristic",
				Confidence: 0.90,
				RawMatch:   strings.TrimSpace(message[match[0]:match[1]]),
			})
		}
	}
	return results
}

// extractNamedVerb handles "Christine likes bondage" patterns
func (lp *LinguisticParser) extractNamedVerb(message string) []ExtractionResult {
	var results []ExtractionResult
	matches := lp.ruleNamedVerb.FindAllStringSubmatchIndex(message, -1)

	for _, match := range matches {
		if len(match) >= 8 {
			subject := strings.TrimSpace(message[match[2]:match[3]])
			property := strings.ToLower(strings.TrimSpace(message[match[6]:match[7]]))

			results = append(results, ExtractionResult{
				Subject:    subject,
				Property:   property,
				Type:       "preference",
				Confidence: 0.85,
				RawMatch:   strings.TrimSpace(message[match[0]:match[1]]),
			})
		}
	}
	return results
}

// extractStructured handles "Genders: Female", "Roles: submissive" patterns
func (lp *LinguisticParser) extractStructured(message string) []ExtractionResult {
	var results []ExtractionResult
	matches := lp.ruleStructured.FindAllStringSubmatchIndex(message, -1)

	for _, match := range matches {
		if len(match) >= 6 {
			key := strings.ToLower(strings.TrimSpace(message[match[2]:match[3]]))
			value := strings.ToLower(strings.TrimSpace(message[match[4]:match[5]]))

			// Map key names to normalized types
			propType := key
			if strings.Contains(key, "gender") {
				propType = "gender"
			} else if strings.Contains(key, "role") {
				propType = "role"
			} else if strings.Contains(key, "into") || strings.Contains(key, "interest") {
				propType = "interest"
			}

			// Clean up value (remove extra formatting)
			value = strings.Trim(value, " ,-")

			results = append(results, ExtractionResult{
				Subject:    "contact", // Structured format usually applies to the contact being described
				Property:   fmt.Sprintf("%s:%s", propType, value),
				Type:       "structured",
				Confidence: 0.90,
				RawMatch:   strings.TrimSpace(message[match[0]:match[1]]),
			})
		}
	}
	return results
}

// extractPrefer handles "I prefer direct communication" patterns
func (lp *LinguisticParser) extractPrefer(message string) []ExtractionResult {
	var results []ExtractionResult
	matches := lp.rulePrefer.FindAllStringSubmatchIndex(message, -1)

	for _, match := range matches {
		if len(match) >= 6 {
			subject := strings.ToLower(strings.TrimSpace(message[match[2]:match[3]]))
			property := strings.ToLower(strings.TrimSpace(message[match[4]:match[5]]))

			subject = normalizeSubject(subject)

			results = append(results, ExtractionResult{
				Subject:    subject,
				Property:   property,
				Type:       "preference",
				Confidence: 0.75,
				RawMatch:   strings.TrimSpace(message[match[0]:match[1]]),
			})
		}
	}
	return results
}

// extractLookingFor handles "I'm looking for X" patterns
// Sept 30, 2026: Improved to check for explicit subject prefix
func (lp *LinguisticParser) extractLookingFor(message string) []ExtractionResult {
	var results []ExtractionResult

	// Match with explicit subject capture
	pattern := regexp.MustCompile(`(?i)\b((?:I'm|I am)?)\s*looking\s+for\s+(?:a\s+)?([a-zA-Z0-9\s\-\.]+?)(?:\.|,|!|\?|$)`)
	matches := pattern.FindAllStringSubmatchIndex(message, -1)

	for _, match := range matches {
		if len(match) >= 6 {
			subjectPrefix := strings.ToLower(strings.TrimSpace(message[match[2]:match[3]]))
			property := strings.ToLower(strings.TrimSpace(message[match[4]:match[5]]))

			// Determine subject and confidence based on prefix
			subject := "user"
			confidence := 0.80

			// If no "I'm" or "I am" prefix, confidence is lower (ambiguous subject)
			if subjectPrefix == "" {
				confidence = 0.55 // Low confidence - no explicit subject marker
			}

			results = append(results, ExtractionResult{
				Subject:    subject,
				Property:   fmt.Sprintf("seeking:%s", property),
				Type:       "preference",
				Confidence: confidence,
				RawMatch:   strings.TrimSpace(message[match[0]:match[1]]),
			})
		}
	}
	return results
}

// dedup removes duplicate extractions (same subject + property)
func (lp *LinguisticParser) dedup(results []ExtractionResult) []ExtractionResult {
	seen := make(map[string]bool)
	var deduped []ExtractionResult

	for _, result := range results {
		key := fmt.Sprintf("%s:%s", result.Subject, result.Property)
		if !seen[key] {
			seen[key] = true
			deduped = append(deduped, result)
		}
	}

	return deduped
}

// ============================================================================
// SENTENCE-LEVEL ANALYSIS FUNCTIONS (NEW)
// ============================================================================

// SegmentIntoSentences breaks a message into individual sentences
// Handles: periods, question marks, exclamation marks
// Returns: Sentence structs with number, text, and positions
func (lp *LinguisticParser) SegmentIntoSentences(message string) []Sentence {
	if strings.TrimSpace(message) == "" {
		return []Sentence{}
	}

	var sentences []Sentence
	sentenceNum := 1
	startPos := 0

	runes := []rune(message)
	for i := 0; i < len(runes); i++ {
		char := runes[i]

		// Check for sentence boundaries
		if char == '.' || char == '?' || char == '!' {
			// Skip if this is an abbreviation (Dr., Mr., etc.)
			if char == '.' && i > 0 && i < len(runes)-1 {
				nextChar := runes[i+1]
				if nextChar != ' ' && nextChar != '\n' && nextChar != '\t' {
					continue
				}
			}

			// Extract sentence text
			endPos := i + 1
			sentenceText := strings.TrimSpace(string(runes[startPos:endPos]))

			if sentenceText != "" {
				sentences = append(sentences, Sentence{
					Number:   sentenceNum,
					Text:     sentenceText,
					StartPos: startPos,
					EndPos:   endPos,
				})
				sentenceNum++
			}

			// Move to next sentence
			startPos = i + 1
			// Skip whitespace
			for startPos < len(runes) && (runes[startPos] == ' ' || runes[startPos] == '\n' || runes[startPos] == '\t') {
				startPos++
			}
			i = startPos - 1
		}
	}

	// Handle remaining text (last sentence if no final punctuation)
	remaining := strings.TrimSpace(string(runes[startPos:]))
	if remaining != "" {
		sentences = append(sentences, Sentence{
			Number:   sentenceNum,
			Text:     remaining,
			StartPos: startPos,
			EndPos:   len(runes),
		})
	}

	return sentences
}

// AnalyzeSentence performs Subject-Verb-Object analysis on a sentence
func (lp *LinguisticParser) AnalyzeSentence(sentenceNum int, sentenceText string, priorSentences []Sentence) SentenceAnalysis {
	analysis := SentenceAnalysis{
		SentenceNumber: sentenceNum,
		SentenceText:   sentenceText,
		Confidence:     0.75,
		ParsingMethod:  "regex",
		CreatedAt:      time.Now().Unix(),
	}

	lower := strings.ToLower(strings.TrimSpace(sentenceText))

	// Detect negation
	analysis.Negated = strings.Contains(lower, "not ") || strings.Contains(lower, "don't ") ||
		strings.Contains(lower, "n't")

	// Extract subject
	subjPattern := regexp.MustCompile(`(?i)^(i|i'm|i am|she|he|they|we|you|it)\s+`)
	subjMatch := subjPattern.FindStringSubmatch(lower)
	if len(subjMatch) > 0 {
		analysis.Subject = normalizeSubject(subjMatch[1])
		analysis.SubjectType = "pronoun"
	} else {
		namePattern := regexp.MustCompile(`(?i)^([a-z]+)\s+(?:is|are|was|were|like|likes|want|wants)\s+`)
		nameMatch := namePattern.FindStringSubmatch(lower)
		if len(nameMatch) > 0 {
			analysis.Subject = nameMatch[1]
			analysis.SubjectType = "name"
		} else {
			analysis.Subject = lp.inferSubjectFromContext(priorSentences)
			analysis.SubjectType = "inferred"
		}
	}

	// Extract verb
	verbPatterns := map[string]string{
		`\b(am|is|are|was|were)\b`:     "copula",
		`\b(like|love|enjoy|prefer)\b`: "transitive",
		`\b(want|need)\b`:              "transitive",
	}

	for pattern, vtype := range verbPatterns {
		re := regexp.MustCompile("(?i)" + pattern)
		if matches := re.FindStringSubmatch(lower); len(matches) > 0 {
			analysis.Verb = matches[1]
			analysis.VerbType = vtype
			analysis.VerbNegated = analysis.Negated
			break
		}
	}

	// Extract object (simplified)
	if analysis.Verb != "" {
		verbIdx := strings.Index(lower, analysis.Verb)
		if verbIdx != -1 {
			afterVerb := strings.TrimSpace(lower[verbIdx+len(analysis.Verb):])
			afterVerb = strings.TrimRight(afterVerb, ".!?")
			words := strings.Fields(afterVerb)
			if len(words) > 0 {
				analysis.Object = words[0]
				if len(words) > 1 {
					analysis.ObjectType = "noun_phrase"
				} else {
					analysis.ObjectType = "noun"
				}
			}
		}
	}

	return analysis
}

// inferSubjectFromContext uses prior sentences to infer subject
func (lp *LinguisticParser) inferSubjectFromContext(priorSentences []Sentence) string {
	if len(priorSentences) == 0 {
		return "unknown"
	}

	lastSentence := priorSentences[len(priorSentences)-1]
	lower := strings.ToLower(lastSentence.Text)

	if strings.Contains(lower, "i am") || strings.Contains(lower, "i'm") || strings.Contains(lower, "my ") {
		return "user"
	}
	if strings.Contains(lower, "she ") || strings.Contains(lower, "her ") {
		return "she"
	}
	if strings.Contains(lower, "he ") || strings.Contains(lower, "his ") {
		return "he"
	}
	if strings.Contains(lower, "they ") || strings.Contains(lower, "them ") {
		return "they"
	}

	return "unknown"
}

// DetectSubjectContext examines preceding text to determine subject attribution
func (lp *LinguisticParser) DetectSubjectContext(message string, currentPos int) string {
	precedingText := message[:currentPos]
	lower := strings.ToLower(precedingText)

	if strings.Contains(lower, "i am") || strings.Contains(lower, "i'm") ||
		strings.Contains(lower, "my profile") || strings.Contains(lower, "my results") {
		return "user"
	}

	if strings.Contains(lower, "her profile") || strings.Contains(lower, "his profile") ||
		strings.Contains(lower, "she ") || strings.Contains(lower, "he ") {
		return "contact"
	}

	return "ambiguous"
}

// normalizeSubject converts pronoun references to standard form
func normalizeSubject(subject string) string {
	subject = strings.ToLower(strings.TrimSpace(subject))

	switch subject {
	case "i", "i'm", "i am", "me":
		return "user"
	case "she", "her", "herself":
		return "she"
	case "he", "him", "himself":
		return "he"
	case "they", "them", "themselves", "we", "us", "ourselves":
		return "they"
	default:
		return subject
	}
}

// extractFocusDirective handles patterns like "X is my focus", "focus on X", "don't focus on X"
// Returns: {subject: "user", property: "X", type: "focus", confidence: 0.90-0.95, is_negated: bool}
func (lp *LinguisticParser) extractFocusDirective(message string) []ExtractionResult {
	var results []ExtractionResult
	lower := strings.ToLower(message)

	// Pattern 3: "don't focus on X" or "can't focus on X" (check FIRST to avoid matching Pattern 2)
	if strings.Contains(lower, "don't focus") || strings.Contains(lower, "can't focus") ||
		strings.Contains(lower, "won't focus") {
		pattern := regexp.MustCompile(`(?:don't|can't|won't)\s+focus\s+on\s+([a-z]+)`)
		matches := pattern.FindStringSubmatch(lower)
		if len(matches) > 1 {
			property := strings.TrimSpace(matches[1])
			results = append(results, ExtractionResult{
				Subject:    "user",
				Property:   property,
				Type:       "focus",
				Confidence: 0.92,
				IsNegated:  true,
				RawMatch:   matches[0],
			})
		}
	}

	// Pattern 1: "X is my focus" or "my focus is X"
	if strings.Contains(lower, "is my focus") || strings.Contains(lower, "my focus is") {
		// Extract X (word before "is my focus")
		pattern := regexp.MustCompile(`(.+?)\s+is\s+my\s+focus`)
		matches := pattern.FindStringSubmatch(lower)
		if len(matches) > 1 {
			property := strings.TrimSpace(matches[1])
			// Get last word if multiple
			words := strings.Fields(property)
			if len(words) > 0 {
				property = words[len(words)-1]
			}
			results = append(results, ExtractionResult{
				Subject:    "user",
				Property:   property,
				Type:       "focus",
				Confidence: 0.95,
				RawMatch:   matches[0],
			})
		}
	}

	// Pattern 2: "focus on X" or "let's focus on X" (only if not negated)
	if strings.Contains(lower, "focus on ") && !strings.Contains(lower, "don't focus") &&
		!strings.Contains(lower, "can't focus") && !strings.Contains(lower, "won't focus") {
		pattern := regexp.MustCompile(`focus\s+on\s+([a-z\s]+?)(?:\.|,|$)`)
		matches := pattern.FindStringSubmatch(lower)
		if len(matches) > 1 {
			property := strings.TrimSpace(matches[1])
			words := strings.Fields(property)
			if len(words) > 0 {
				// Take first 1-2 words
				if len(words) == 1 {
					property = words[0]
				} else if len(words) >= 2 && (words[0] == "my" || words[0] == "the") {
					property = words[1]
				} else {
					property = words[0]
				}
			}
			results = append(results, ExtractionResult{
				Subject:    "user",
				Property:   property,
				Type:       "focus",
				Confidence: 0.90,
				RawMatch:   matches[0],
			})
		}
	}

	return results
}

// extractConstraint handles "remember X", "keep in mind X", "don't forget X"
// Returns: {subject: "user", property: "X", type: "constraint", confidence: 0.85-0.90}
func (lp *LinguisticParser) extractConstraint(message string) []ExtractionResult {
	var results []ExtractionResult
	lower := strings.ToLower(message)

	// Pattern: "remember X" or "keep in mind X" or "don't forget X"
	var pattern *regexp.Regexp

	if strings.Contains(lower, "remember ") {
		pattern = regexp.MustCompile(`remember\s+(?:to\s+)?(.+?)(?:\.|,|$)`)
	} else if strings.Contains(lower, "keep in mind") {
		pattern = regexp.MustCompile(`keep\s+in\s+mind\s+(.+?)(?:\.|,|$)`)
	} else if strings.Contains(lower, "don't forget") {
		pattern = regexp.MustCompile(`don't\s+forget\s+(.+?)(?:\.|,|$)`)
	}

	if pattern != nil {
		matches := pattern.FindStringSubmatch(lower)
		if len(matches) > 1 {
			property := strings.TrimSpace(matches[1])
			// Take just the first word/phrase for cleaner extraction
			words := strings.Fields(property)
			if len(words) > 0 {
				// Keep first 1-2 words for clarity
				if len(words) == 1 {
					property = words[0]
				} else {
					property = words[0]
				}
			}
			results = append(results, ExtractionResult{
				Subject:    "user",
				Property:   property,
				Type:       "constraint",
				Confidence: 0.85,
				RawMatch:   matches[0],
			})
		}
	}

	return results
}

// extractPriority handles "my priority is X" or "prioritize X"
// Returns: {subject: "user", property: "X", type: "priority", confidence: 0.90-0.95}
func (lp *LinguisticParser) extractPriority(message string) []ExtractionResult {
	var results []ExtractionResult
	lower := strings.ToLower(message)

	// Pattern: "my priority is X" or "priority is X"
	if strings.Contains(lower, "priority is ") {
		pattern := regexp.MustCompile(`priority\s+is\s+([a-z\s]+?)(?:\.|,|$)`)
		matches := pattern.FindStringSubmatch(lower)
		if len(matches) > 1 {
			property := strings.TrimSpace(matches[1])
			words := strings.Fields(property)
			if len(words) > 0 {
				property = words[0]
			}
			results = append(results, ExtractionResult{
				Subject:    "user",
				Property:   property,
				Type:       "priority",
				Confidence: 0.95,
				RawMatch:   matches[0],
			})
		}
	}

	// Pattern: "prioritize X"
	if strings.Contains(lower, "prioritize ") {
		pattern := regexp.MustCompile(`prioritize\s+([a-z\s]+?)(?:\.|,|$)`)
		matches := pattern.FindStringSubmatch(lower)
		if len(matches) > 1 {
			property := strings.TrimSpace(matches[1])
			words := strings.Fields(property)
			if len(words) > 0 {
				property = words[0]
			}
			results = append(results, ExtractionResult{
				Subject:    "user",
				Property:   property,
				Type:       "priority",
				Confidence: 0.90,
				RawMatch:   matches[0],
			})
		}
	}

	return results
}

// extractNegatedDirective handles "not interested in X", "don't want X"
// Returns: {subject: "user", property: "X", type: "interest", confidence: 0.85-0.90, is_negated: true}
func (lp *LinguisticParser) extractNegatedDirective(message string) []ExtractionResult {
	var results []ExtractionResult
	lower := strings.ToLower(message)

	// Pattern: "not interested in X"
	if strings.Contains(lower, "not interested in") {
		pattern := regexp.MustCompile(`not\s+interested\s+in\s+([a-z\s]+?)(?:\.|,|$)`)
		matches := pattern.FindStringSubmatch(lower)
		if len(matches) > 1 {
			property := strings.TrimSpace(matches[1])
			results = append(results, ExtractionResult{
				Subject:    "user",
				Property:   property,
				Type:       "interest",
				Confidence: 0.90,
				IsNegated:  true,
				RawMatch:   matches[0],
			})
		}
	}

	// Pattern: "don't want X"
	if strings.Contains(lower, "don't want") {
		pattern := regexp.MustCompile(`don't\s+want\s+([a-z\s]+?)(?:\.|,|$)`)
		matches := pattern.FindStringSubmatch(lower)
		if len(matches) > 1 {
			property := strings.TrimSpace(matches[1])
			results = append(results, ExtractionResult{
				Subject:    "user",
				Property:   property,
				Type:       "preference",
				Confidence: 0.88,
				IsNegated:  true,
				RawMatch:   matches[0],
			})
		}
	}

	return results
}
