package agents

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"moly/models"
	"moly/tools"
)

// SemanticExtractor performs linguistic/semantic extraction for all entity types
// Uses confidence-driven approach: high confidence → use directly, low confidence → clarify
type SemanticExtractor struct {
	patterns PatternRegistry
}

// NewSemanticExtractor creates a new semantic extractor with all patterns registered
func NewSemanticExtractor() *SemanticExtractor {
	return &SemanticExtractor{
		patterns: NewPatternRegistry(),
	}
}

// ExtractedEntity represents a single extracted piece of information with confidence
type ExtractedEntity struct {
	Type       string  // goal, contact, value, characteristic, style, concern
	Value      string  // The actual extracted value
	Confidence float64 // 0.0-1.0: how confident in this extraction
	Pattern    string  // Which pattern matched
	Evidence   string  // The phrase from the text that supports this
	Subject    string  // Who is doing/being (I, Sarah, they, etc.)
}

// Extract performs semantic extraction on a message, returning all entity types with confidence
func (se *SemanticExtractor) Extract(message string) *models.ExtractedContext {
	if message == "" {
		return &models.ExtractedContext{}
	}

	log.Printf("[SemanticExtractor] Starting extraction from: %.100s...", message)

	// Parse linguistic structure
	parsed := se.parseMessage(message)
	log.Printf("[SemanticExtractor] Parsed %d sentences", len(parsed.Sentences))

	// Extract each entity type
	extracted := &models.ExtractedContext{}

	// Extract goals
	goals := se.extractGoals(parsed)
	if len(goals) > 0 {
		bestGoal := selectBestExtraction(goals)
		extracted.Intention = bestGoal.Value
		extracted.IntentionConfidence = bestGoal.Confidence
		log.Printf("[SemanticExtractor] Goal (%.2f): %q", bestGoal.Confidence, bestGoal.Value)
	}

	// Extract contacts
	contacts := se.extractContacts(parsed)
	if len(contacts) > 0 {
		for _, contact := range contacts {
			extracted.ExtractedContacts = append(extracted.ExtractedContacts, &models.ExtractedContact{
				Name:           contact.Value,
				Relationship:   contact.Subject,
				Confidence:     contact.Confidence,
				Evidence:       contact.Evidence,
				DetectionMethod: "semantic",
			})
			log.Printf("[SemanticExtractor] Contact (%.2f): %s (%s)", contact.Confidence, contact.Value, contact.Subject)
		}
	}

	// Extract values
	values := se.extractValues(parsed)
	if len(values) > 0 {
		for _, v := range values {
			extracted.UserValues = append(extracted.UserValues, v.Value)
			log.Printf("[SemanticExtractor] Value (%.2f): %q", v.Confidence, v.Value)
		}
	}

	// Extract characteristics
	characteristics := se.extractCharacteristics(parsed)
	if len(characteristics) > 0 {
		for _, c := range characteristics {
			extracted.UserCharacteristics = append(extracted.UserCharacteristics, c.Value)
			log.Printf("[SemanticExtractor] Characteristic (%.2f): %q", c.Confidence, c.Value)
		}
	}

	// Extract style
	styles := se.extractStyle(parsed)
	if len(styles) > 0 {
		bestStyle := selectBestExtraction(styles)
		extracted.CommunicationStyle = &models.ExtractedStyle{
			Style:      bestStyle.Value,
			Confidence: bestStyle.Confidence,
			Evidence:   bestStyle.Evidence,
		}
		log.Printf("[SemanticExtractor] Style (%.2f): %q", bestStyle.Confidence, bestStyle.Value)
	}

	// Extract concerns
	concerns := se.extractConcerns(parsed)
	if len(concerns) > 0 {
		for _, concern := range concerns {
			extracted.IntentionPrinciples = append(extracted.IntentionPrinciples, concern.Value)
			log.Printf("[SemanticExtractor] Concern (%.2f): %q", concern.Confidence, concern.Value)
		}
	}

	log.Printf("[SemanticExtractor] ✓ Extraction complete")
	return extracted
}

// parseMessage breaks text into sentences and linguistic components
func (se *SemanticExtractor) parseMessage(message string) *ParsedMessage {
	// Split into sentences (simple regex approach)
	sentences := strings.Split(message, ".")
	parsed := &ParsedMessage{
		OriginalText: message,
		Sentences:    []string{},
	}

	for _, sent := range sentences {
		sent = strings.TrimSpace(sent)
		if sent != "" {
			parsed.Sentences = append(parsed.Sentences, sent)
		}
	}

	return parsed
}

// extractGoals uses infinitive complement patterns to find user goals
func (se *SemanticExtractor) extractGoals(parsed *ParsedMessage) []ExtractedEntity {
	var goals []ExtractedEntity

	for _, sentence := range parsed.Sentences {
		lower := strings.ToLower(sentence)

		// Pattern 1: "want/need/wish + to + VERB"
		goalVerbs := []string{"want", "need", "wish", "hope", "try", "aim", "plan"}
		for _, goalVerb := range goalVerbs {
			if strings.Contains(lower, goalVerb+" to ") {
				goal, conf := se.extractGoalPhrase(sentence, goalVerb)
				if goal != "" {
					goals = append(goals, ExtractedEntity{
						Type:       "goal",
						Value:      goal,
						Confidence: conf,
						Pattern:    fmt.Sprintf("%s + to", goalVerb),
						Evidence:   sentence,
					})
				}
			}
		}

		// Pattern 2: "I'm + going to + VERB"
		if strings.Contains(lower, "going to ") {
			goal, conf := se.extractGoalPhrase(sentence, "going to")
			if goal != "" {
				goals = append(goals, ExtractedEntity{
					Type:       "goal",
					Value:      goal,
					Confidence: conf,
					Pattern:    "going to",
					Evidence:   sentence,
				})
			}
		}

		// Pattern 3: Question with "how to" or "should I"
		if (strings.Contains(lower, "how") && strings.Contains(lower, "?")) ||
			(strings.Contains(lower, "should i") && strings.Contains(lower, "?")) {
			goal, conf := se.extractGoalFromQuestion(sentence)
			if goal != "" {
				goals = append(goals, ExtractedEntity{
					Type:       "goal",
					Value:      goal,
					Confidence: conf,
					Pattern:    "question",
					Evidence:   sentence,
				})
			}
		}
	}

	return goals
}

// extractGoalPhrase extracts the full goal phrase after goal verb
func (se *SemanticExtractor) extractGoalPhrase(sentence string, goalVerb string) (string, float64) {
	lower := strings.ToLower(sentence)
	idx := strings.Index(lower, goalVerb)

	if idx == -1 {
		return "", 0.0
	}

	// Skip past "to" if present
	afterVerb := strings.TrimSpace(sentence[idx+len(goalVerb):])
	if strings.HasPrefix(strings.ToLower(afterVerb), "to ") {
		afterVerb = strings.TrimSpace(afterVerb[3:])
	}

	// Extract until period, comma, or logical end
	goal := strings.FieldsFunc(afterVerb, func(r rune) bool {
		return r == '.' || r == ',' || r == '!' || r == '?'
	})[0]

	goal = strings.TrimSpace(goal)

	// Validate: should be meaningful (more than just preposition)
	words := strings.Fields(goal)
	if len(words) < 2 {
		// Single word goal - lower confidence
		return goal, 0.75
	}

	// Multi-word goal - high confidence
	return goal, 0.95
}

// extractGoalFromQuestion extracts goal from interrogative sentences
func (se *SemanticExtractor) extractGoalFromQuestion(sentence string) (string, float64) {
	lower := strings.ToLower(sentence)

	// Pattern: "how [do I / can I] VERB"
	howPattern := regexp.MustCompile(`(?i)how\s+(?:do\s+)?(?:i\s+)?(\w+)`)
	matches := howPattern.FindStringSubmatch(lower)
	if len(matches) > 1 {
		verb := matches[1]
		// Extract full phrase from sentence
		idx := strings.Index(lower, verb)
		if idx != -1 {
			phrase := strings.TrimSpace(sentence[idx:])
			phrase = strings.TrimRight(phrase, "?")
			return phrase, 0.85
		}
	}

	// Pattern: "should I VERB"
	shouldPattern := regexp.MustCompile(`(?i)should\s+i\s+(\w+)`)
	matches = shouldPattern.FindStringSubmatch(lower)
	if len(matches) > 1 {
		verb := matches[1]
		idx := strings.Index(lower, verb)
		if idx != -1 {
			phrase := strings.TrimSpace(sentence[idx:])
			phrase = strings.TrimRight(phrase, "?")
			return phrase, 0.85
		}
	}

	return "", 0.0
}

// extractContacts finds named entities and their relationships
func (se *SemanticExtractor) extractContacts(parsed *ParsedMessage) []ExtractedEntity {
	var contacts []ExtractedEntity

	for _, sentence := range parsed.Sentences {
		lower := strings.ToLower(sentence)

		// Pattern 1: "[Name] is my [relationship]"
		isPattern := regexp.MustCompile(`(?i)([A-Z][a-z]+)\s+is\s+(?:my\s+)?(\w+)`)
		matches := isPattern.FindAllStringSubmatch(sentence, -1)
		for _, match := range matches {
			if len(match) > 2 {
				name := match[1]
				relationship := match[2]
				contacts = append(contacts, ExtractedEntity{
					Type:       "contact",
					Value:      name,
					Subject:    relationship,
					Confidence: 0.95,
					Pattern:    "[Name] is [relationship]",
					Evidence:   sentence,
				})
			}
		}

		// Pattern 2: "message/talk to [Name]"
		toPattern := regexp.MustCompile(`(?i)(?:to|with)\s+([A-Z][a-z]+)`)
		matches = toPattern.FindAllStringSubmatch(sentence, -1)
		for _, match := range matches {
			if len(match) > 1 {
				name := match[1]
				contacts = append(contacts, ExtractedEntity{
					Type:       "contact",
					Value:      name,
					Confidence: 0.90,
					Pattern:    "to/with [Name]",
					Evidence:   sentence,
				})
			}
		}

		// Pattern 3: "my [relationship] [Name]"
		possessivePattern := regexp.MustCompile(`(?i)my\s+(\w+)\s+([A-Z][a-z]+)`)
		matches = possessivePattern.FindAllStringSubmatch(sentence, -1)
		for _, match := range matches {
			if len(match) > 2 {
				relationship := match[1]
				name := match[2]
				contacts = append(contacts, ExtractedEntity{
					Type:       "contact",
					Value:      name,
					Subject:    relationship,
					Confidence: 0.92,
					Pattern:    "my [relationship] [Name]",
					Evidence:   sentence,
				})
			}
		}
	}

	return contacts
}

// extractValues finds what user cares about
func (se *SemanticExtractor) extractValues(parsed *ParsedMessage) []ExtractedEntity {
	var values []ExtractedEntity

	for _, sentence := range parsed.Sentences {
		lower := strings.ToLower(sentence)

		// Pattern 1: "value/care about/believe in [NOUN]"
		valueVerbs := []string{"value", "care about", "believe in", "prioritize"}
		for _, verb := range valueVerbs {
			if strings.Contains(lower, verb) {
				value, conf := se.extractAfterPattern(sentence, verb)
				if value != "" {
					values = append(values, ExtractedEntity{
						Type:       "value",
						Value:      value,
						Confidence: conf,
						Pattern:    fmt.Sprintf("%s + [object]", verb),
						Evidence:   sentence,
					})
				}
			}
		}

		// Pattern 2: "it's important to me that [CLAUSE]"
		if strings.Contains(lower, "important") {
			value, conf := se.extractAfterPattern(sentence, "that")
			if value != "" {
				values = append(values, ExtractedEntity{
					Type:       "value",
					Value:      value,
					Confidence: conf,
					Pattern:    "important + that",
					Evidence:   sentence,
				})
			}
		}
	}

	return values
}

// extractCharacteristics finds adjectives describing the user
func (se *SemanticExtractor) extractCharacteristics(parsed *ParsedMessage) []ExtractedEntity {
	var characteristics []ExtractedEntity

	for _, sentence := range parsed.Sentences {
		lower := strings.ToLower(sentence)

		// Pattern 1: "I am/I'm [ADJ]" or "I am [ADJ]"
		iAmPattern := regexp.MustCompile(`(?i)(?:I\s+am|I'm|i\s+am)\s+([a-z\s,and]+?)(?:,|and|$)`)
		matches := iAmPattern.FindAllStringSubmatch(sentence, -1)
		for _, match := range matches {
			if len(match) > 1 {
				adj := strings.TrimSpace(match[1])
				adj = strings.Trim(adj, ",")
				if adj != "" {
					characteristics = append(characteristics, ExtractedEntity{
						Type:       "characteristic",
						Value:      adj,
						Confidence: 0.95,
						Pattern:    "I am [ADJ]",
						Evidence:   sentence,
					})
				}
			}
		}

		// Pattern 2: "I describe myself as [ADJ]"
		if strings.Contains(lower, "describe myself as") {
			value, conf := se.extractAfterPattern(sentence, "as")
			if value != "" {
				characteristics = append(characteristics, ExtractedEntity{
					Type:       "characteristic",
					Value:      value,
					Confidence: conf,
					Pattern:    "describe myself as",
					Evidence:   sentence,
				})
			}
		}
	}

	return characteristics
}

// extractStyle finds communication preferences
func (se *SemanticExtractor) extractStyle(parsed *ParsedMessage) []ExtractedEntity {
	var styles []ExtractedEntity

	for _, sentence := range parsed.Sentences {
		lower := strings.ToLower(sentence)

		// Pattern 1: "I prefer [STYLE]" or "I like [STYLE]"
		styleVerbs := []string{"prefer", "like", "want to be"}
		for _, verb := range styleVerbs {
			if strings.Contains(lower, verb) {
				value, conf := se.extractAfterPattern(sentence, verb)
				if value != "" {
					styles = append(styles, ExtractedEntity{
						Type:       "style",
						Value:      value,
						Confidence: conf,
						Pattern:    fmt.Sprintf("%s + [style]", verb),
						Evidence:   sentence,
					})
				}
			}
		}

		// Pattern 2: "casual/formal/playful" adjectives
		styleAdjectives := []string{"casual", "formal", "playful", "direct", "gentle", "honest", "sincere"}
		for _, adj := range styleAdjectives {
			if strings.Contains(lower, adj) {
				styles = append(styles, ExtractedEntity{
					Type:       "style",
					Value:      adj,
					Confidence: 0.85,
					Pattern:    fmt.Sprintf("style adjective: %s", adj),
					Evidence:   sentence,
				})
			}
		}
	}

	return styles
}

// extractConcerns finds worries and safety considerations
func (se *SemanticExtractor) extractConcerns(parsed *ParsedMessage) []ExtractedEntity {
	var concerns []ExtractedEntity

	for _, sentence := range parsed.Sentences {
		lower := strings.ToLower(sentence)

		// Pattern 1: "worried/afraid/concerned about [NOUN]"
		concernVerbs := []string{"worried about", "afraid of", "concerned about", "scared of"}
		for _, verb := range concernVerbs {
			if strings.Contains(lower, verb) {
				value, conf := se.extractAfterPattern(sentence, verb)
				if value != "" {
					concerns = append(concerns, ExtractedEntity{
						Type:       "concern",
						Value:      value,
						Confidence: conf,
						Pattern:    fmt.Sprintf("%s + [concern]", verb),
						Evidence:   sentence,
					})
				}
			}
		}

		// Pattern 2: "I shouldn't/don't want to [ACTION]"
		if strings.Contains(lower, "shouldn't") || strings.Contains(lower, "don't want to") {
			value, conf := se.extractGoalPhrase(sentence, "to")
			if value != "" {
				concerns = append(concerns, ExtractedEntity{
					Type:       "concern",
					Value:      fmt.Sprintf("avoid %s", value),
					Confidence: conf,
					Pattern:    "negation - avoid [action]",
					Evidence:   sentence,
				})
			}
		}
	}

	return concerns
}

// extractAfterPattern extracts noun phrase after a pattern marker
func (se *SemanticExtractor) extractAfterPattern(sentence string, pattern string) (string, float64) {
	lower := strings.ToLower(sentence)
	idx := strings.Index(lower, pattern)

	if idx == -1 {
		return "", 0.0
	}

	afterPattern := strings.TrimSpace(sentence[idx+len(pattern):])
	// Remove leading "to", "that", "in", etc.
	afterPattern = strings.TrimLeft(afterPattern, " to that in of ")

	// Extract until punctuation
	words := strings.FieldsFunc(afterPattern, func(r rune) bool {
		return r == '.' || r == ',' || r == '!' || r == '?'
	})

	if len(words) == 0 {
		return "", 0.0
	}

	result := strings.Join(words, " ")
	result = strings.TrimSpace(result)

	if len(words) > 2 {
		return result, 0.90
	}
	return result, 0.80
}

// selectBestExtraction picks the highest confidence extraction
func selectBestExtraction(extractions []ExtractedEntity) ExtractedEntity {
	if len(extractions) == 0 {
		return ExtractedEntity{}
	}

	best := extractions[0]
	for _, e := range extractions[1:] {
		if e.Confidence > best.Confidence {
			best = e
		}
	}
	return best
}

// ParsedMessage represents a parsed message structure
type ParsedMessage struct {
	OriginalText string
	Sentences    []string
}

// PatternRegistry holds patterns for each entity type
type PatternRegistry struct {
	goalPatterns            []Pattern
	contactPatterns         []Pattern
	valuePatterns           []Pattern
	characteristicPatterns  []Pattern
	stylePatterns           []Pattern
	concernPatterns         []Pattern
}

// Pattern defines a linguistic pattern for extraction
type Pattern struct {
	Name             string
	EntityType       string
	LinguisticRule   string
	ConfidenceBase   float64
	ExtractionLogic  func(string) (string, float64)
	Example          string
}

// NewPatternRegistry creates a new pattern registry
func NewPatternRegistry() PatternRegistry {
	return PatternRegistry{
		// Patterns populated on demand
	}
}
