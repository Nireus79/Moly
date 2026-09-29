package tools

import (
	"fmt"
	"regexp"
	"strings"
)

// ExtractionResult represents a single extracted entity with its subject and confidence
type ExtractionResult struct {
	Subject    string  // "user", "she", "he", "Christine", etc.
	Property   string  // The extracted property/attribute
	Type       string  // "preference", "characteristic", "negation", "structured", etc.
	Confidence float64 // 0.0-1.0
	RawMatch   string  // The matched phrase from the message
}

// LinguisticParser handles grammar-based entity extraction
type LinguisticParser struct {
	// Compiled regex patterns for each rule
	ruleIsAdjective    *regexp.Regexp // "I am {adj}", "She is {adj}"
	ruleIsNegative     *regexp.Regexp // "I am NOT {adj}", "I'm not {adj}"
	ruleLikeDislike    *regexp.Regexp // "I like/love/want {noun}", "I don't like/want {noun}"
	ruleNotInterested  *regexp.Regexp // "I'm not interested in {X}", "not interested in {X}"
	ruleNamedIs        *regexp.Regexp // "{Name} is {property}"
	ruleNamedVerb      *regexp.Regexp // "{Name} likes/wants {property}"
	ruleStructured     *regexp.Regexp // "Key: Value" format
	rulePrefer         *regexp.Regexp // "I prefer {noun}"
	ruleLookingFor     *regexp.Regexp // "I'm looking for {noun}"
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

	// Dedup: remove duplicates (same subject + property)
	return lp.dedup(results)
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
func (lp *LinguisticParser) extractLookingFor(message string) []ExtractionResult {
	var results []ExtractionResult

	// Only match if "looking for" or "looking" is present
	pattern := regexp.MustCompile(`(?i)\b(?:I'm|I am)?\s*looking\s+for\s+(?:a\s+)?([a-zA-Z0-9\s\-\.]+?)(?:\.|,|!|\?|$)`)
	matches := pattern.FindAllStringSubmatchIndex(message, -1)

	for _, match := range matches {
		if len(match) >= 4 {
			property := strings.ToLower(strings.TrimSpace(message[match[2]:match[3]]))

			results = append(results, ExtractionResult{
				Subject:    "user",
				Property:   fmt.Sprintf("seeking:%s", property),
				Type:       "preference",
				Confidence: 0.80,
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
