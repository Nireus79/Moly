package tools

import (
	"fmt"
	"regexp"
	"strings"
)

// ProfileAttribute represents a single extracted profile attribute
type ProfileAttribute struct {
	Key        string   // "gender", "role", "interest", "age", "location", etc.
	Value      string   // The extracted value
	Values     []string // Multiple values (for lists like "Into: Bondage, Aftercare")
	Confidence float64  // 0.0-1.0
	RawMatch   string   // Original matched text
	IsList     bool     // Whether this attribute contains multiple values
}

// ProfileData represents extracted profile information
type ProfileData struct {
	Attributes []ProfileAttribute
	RawText    string
	Format     string // "fetlife", "text", "structured"
	Subject    string // Who this profile is for (contact name or "user")
}

// ProfileParser handles extraction of profile data from messages
// Supports fetlife format (Genders: Female, Roles: submissive, Into: X, Y, Z)
type ProfileParser struct {
	// Regex patterns for different profile formats
	fetlifePattern  *regexp.Regexp
	keyValuePattern *regexp.Regexp
	colonPattern    *regexp.Regexp
}

// NewProfileParser creates a new profile parser
func NewProfileParser() *ProfileParser {
	return &ProfileParser{
		// FetLife format: "Key: Value" or "Key: Value1, Value2"
		fetlifePattern: regexp.MustCompile(`(?i)(Genders?|Roles?|Into|Interests|Kinks?|Traits?|Ages?|Locations?|Status|Body Type|Height|Build|Hair|Eyes|Ethnicity|Seeking|Looking for|Relationship Status):\s*([^\n]+?)(?:\n|$)`),

		// Generic key: value pattern
		keyValuePattern: regexp.MustCompile(`(?i)^([a-zA-Z\s]+?):\s*(.+?)$`),

		// Colon-based pattern (simple)
		colonPattern: regexp.MustCompile(`:\s*`),
	}
}

// Parse analyzes text and extracts profile attributes
func (pp *ProfileParser) Parse(text string) *ProfileData {
	data := &ProfileData{
		RawText:    text,
		Attributes: []ProfileAttribute{},
	}

	if text == "" {
		return data
	}

	// Try FetLife format first
	if pp.parseFetLifeFormat(text, data) {
		data.Format = "fetlife"
		return data
	}

	// Fall back to generic key-value format
	if pp.parseKeyValueFormat(text, data) {
		data.Format = "text"
		return data
	}

	// No structured format found
	data.Format = "unstructured"
	return data
}

// parseFetLifeFormat handles the FetLife profile format
func (pp *ProfileParser) parseFetLifeFormat(text string, data *ProfileData) bool {
	matches := pp.fetlifePattern.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return false
	}

	for _, match := range matches {
		if len(match) >= 6 {
			key := strings.ToLower(strings.TrimSpace(text[match[2]:match[3]]))
			value := strings.TrimSpace(text[match[4]:match[5]])

			// Normalize key names
			key = normalizeCategoryKey(key)

			// Split comma-separated values
			isList := false
			var values []string

			if strings.Contains(value, ",") {
				isList = true
				parts := strings.Split(value, ",")
				for _, part := range parts {
					trimmed := strings.TrimSpace(part)
					if trimmed != "" {
						values = append(values, trimmed)
					}
				}
			} else {
				values = []string{value}
			}

			attr := ProfileAttribute{
				Key:        key,
				Value:      value,
				Values:     values,
				Confidence: 0.95, // High confidence for structured format
				RawMatch:   strings.TrimSpace(text[match[0]:match[1]]),
				IsList:     isList,
			}

			data.Attributes = append(data.Attributes, attr)
		}
	}

	return len(data.Attributes) > 0
}

// parseKeyValueFormat handles generic key: value format
func (pp *ProfileParser) parseKeyValueFormat(text string, data *ProfileData) bool {
	lines := strings.Split(text, "\n")
	foundAny := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || !strings.Contains(trimmed, ":") {
			continue
		}

		matches := pp.keyValuePattern.FindStringSubmatchIndex(trimmed)
		if matches == nil || len(matches) < 4 {
			continue
		}

		key := strings.ToLower(strings.TrimSpace(trimmed[matches[2]:matches[3]]))
		value := strings.TrimSpace(trimmed[matches[4]:matches[5]])

		if key == "" || value == "" {
			continue
		}

		// Normalize key
		key = normalizeCategoryKey(key)

		// Split comma-separated values if present
		isList := false
		var values []string

		if strings.Contains(value, ",") {
			isList = true
			parts := strings.Split(value, ",")
			for _, part := range parts {
				trimmed := strings.TrimSpace(part)
				if trimmed != "" {
					values = append(values, trimmed)
				}
			}
		} else {
			values = []string{value}
		}

		attr := ProfileAttribute{
			Key:        key,
			Value:      value,
			Values:     values,
			Confidence: 0.75, // Slightly lower confidence for generic format
			RawMatch:   trimmed,
			IsList:     isList,
		}

		data.Attributes = append(data.Attributes, attr)
		foundAny = true
	}

	return foundAny
}

// GetAttribute retrieves a specific attribute by key
func (pp *ProfileParser) GetAttribute(data *ProfileData, key string) *ProfileAttribute {
	key = normalizeCategoryKey(key)

	for i, attr := range data.Attributes {
		if attr.Key == key {
			return &data.Attributes[i]
		}
	}

	return nil
}

// GetAllValues gets all values for a specific key
func (pp *ProfileParser) GetAllValues(data *ProfileData, key string) []string {
	attr := pp.GetAttribute(data, key)
	if attr == nil {
		return []string{}
	}

	if attr.IsList {
		return attr.Values
	}

	return []string{attr.Value}
}

// normalizeCategoryKey standardizes profile category names
func normalizeCategoryKey(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))

	// Map variations to canonical names
	mappings := map[string]string{
		"gender":             "gender",
		"genders":            "gender",
		"role":               "role",
		"roles":              "role",
		"into":               "interests",
		"interest":           "interests",
		"interests":          "interests",
		"kink":               "kinks",
		"kinks":              "kinks",
		"trait":              "traits",
		"traits":             "traits",
		"age":                "age",
		"ages":               "age",
		"location":           "location",
		"locations":          "location",
		"status":             "relationship_status",
		"relationship status": "relationship_status",
		"body type":          "body_type",
		"height":             "height",
		"build":              "build",
		"hair":               "hair",
		"eyes":               "eyes",
		"eye color":          "eyes",
		"ethnicity":          "ethnicity",
		"seeking":            "seeking",
		"looking for":        "seeking",
	}

	if canonical, exists := mappings[key]; exists {
		return canonical
	}

	return key
}

// FormatAsStructuredData converts profile attributes to database format
// Returns a map suitable for storing in JSON columns
func (pp *ProfileParser) FormatAsStructuredData(data *ProfileData) map[string]interface{} {
	result := make(map[string]interface{})

	for _, attr := range data.Attributes {
		if attr.IsList {
			result[attr.Key] = attr.Values
		} else {
			result[attr.Key] = attr.Value
		}
	}

	return result
}

// ExtractProfileFromMessage parses a message and returns profile data
// This is a convenience method that combines parsing with subject detection
func (pp *ProfileParser) ExtractProfileFromMessage(message string, defaultSubject string) *ProfileData {
	data := pp.Parse(message)

	// Set subject (default to provided value, could be enhanced with ML detection)
	if data.Subject == "" {
		data.Subject = defaultSubject
	}

	return data
}

// MergeProfiles merges two profile data objects, with newer taking precedence
func (pp *ProfileParser) MergeProfiles(existing *ProfileData, newer *ProfileData) *ProfileData {
	if existing == nil {
		return newer
	}

	if newer == nil {
		return existing
	}

	merged := &ProfileData{
		Attributes: existing.Attributes,
		RawText:    existing.RawText + "\n---\n" + newer.RawText,
		Format:     newer.Format, // Use newer format
		Subject:    newer.Subject, // Use newer subject if set
	}

	// Merge attributes (newer overrides existing for same key)
	keyMap := make(map[string]ProfileAttribute)

	// First add existing attributes
	for _, attr := range existing.Attributes {
		keyMap[attr.Key] = attr
	}

	// Then add/override with newer attributes
	for _, attr := range newer.Attributes {
		keyMap[attr.Key] = attr
	}

	// Rebuild attributes list
	merged.Attributes = []ProfileAttribute{}
	for _, attr := range keyMap {
		merged.Attributes = append(merged.Attributes, attr)
	}

	return merged
}

// ValidateAttribute checks if an attribute value is reasonable
func (pp *ProfileParser) ValidateAttribute(key, value string) bool {
	key = normalizeCategoryKey(key)

	// Don't validate unknown keys
	if key == "" || value == "" {
		return false
	}

	// Basic validation rules
	switch key {
	case "age":
		// Age should be numeric
		for _, ch := range value {
			if ch < '0' || ch > '9' {
				return false
			}
		}
		return true

	case "gender", "role":
		// Should be one of known values
		validValues := map[string]bool{
			"male": true, "female": true, "non-binary": true,
			"dominant": true, "submissive": true, "switch": true,
			"top": true, "bottom": true,
		}
		return validValues[strings.ToLower(value)]

	default:
		// All other attributes are valid if non-empty
		return len(strings.TrimSpace(value)) > 0
	}
}

// GetSummary creates a human-readable summary of profile
func (pp *ProfileParser) GetSummary(data *ProfileData) string {
	if len(data.Attributes) == 0 {
		return "No profile information extracted"
	}

	var parts []string
	for _, attr := range data.Attributes {
		if attr.IsList {
			parts = append(parts, fmt.Sprintf("%s: %s", attr.Key, strings.Join(attr.Values, ", ")))
		} else {
			parts = append(parts, fmt.Sprintf("%s: %s", attr.Key, attr.Value))
		}
	}

	return strings.Join(parts, " | ")
}
