package config

import "strings"

// Contact relationship types - must match Contact model validation
const (
	RelationshipRomantic    = "romantic"
	RelationshipProfessional = "professional"
	RelationshipFamily      = "family"
	RelationshipFriend      = "friend"
	RelationshipOther       = "other"
)

// Valid relationship types list
var ValidRelationships = []string{
	RelationshipRomantic,
	RelationshipProfessional,
	RelationshipFamily,
	RelationshipFriend,
	RelationshipOther,
}

// Context types for attributes
const (
	ContextWork     = "work"
	ContextPersonal = "personal"
	ContextGeneral  = "general"
	ContextSocial   = "social"
	ContextFamily   = "family"
)

// Valid context types list
var ValidContexts = []string{
	ContextWork,
	ContextPersonal,
	ContextGeneral,
	ContextSocial,
	ContextFamily,
}

// Risk severity levels
const (
	SeverityLow      = "low"
	SeverityMedium   = "medium"
	SeverityHigh     = "high"
	SeverityImmediate = "immediate"
)

// Valid severity levels
var ValidSeverities = []string{
	SeverityLow,
	SeverityMedium,
	SeverityHigh,
	SeverityImmediate,
}

// Risk levels
const (
	RiskLevelMinimal    = "minimal"
	RiskLevelLow        = "low"
	RiskLevelModerate   = "moderate"
	RiskLevelElevated   = "elevated"
	RiskLevelHigh       = "high"
	RiskLevelImmediate  = "immediate"
)

// Valid risk levels
var ValidRiskLevels = []string{
	RiskLevelMinimal,
	RiskLevelLow,
	RiskLevelModerate,
	RiskLevelElevated,
	RiskLevelHigh,
}

// Context quality levels
const (
	ContextQualityMinimal       = "minimal"
	ContextQualityPartial       = "partial"
	ContextQualityComprehensive = "comprehensive"
)

// Valid context quality levels
var ValidContextQualities = []string{
	ContextQualityMinimal,
	ContextQualityPartial,
	ContextQualityComprehensive,
}

// Severity thresholds for scoring (extracted from hardcoded values)
const (
	HighSeverityThreshold   = 60    // Severity score >= 60 is "high"
	MediumSeverityThreshold = 30    // Severity score >= 30 is "medium"
	DefaultLLMTimeoutSecs   = 30    // Default LLM call timeout
	MaxLLMTimeoutSecs       = 180   // Max timeout for long prompts
)

// MapKeywordToRelationship maps user keywords to valid relationship types
func MapKeywordToRelationship(keyword string) string {
	keyword = strings.ToLower(strings.TrimSpace(keyword))

	// Professional relationships
	if keyword == "boss" || keyword == "manager" || keyword == "supervisor" || keyword == "director" {
		return RelationshipProfessional
	}
	if keyword == "colleague" || keyword == "coworker" || keyword == "team member" || keyword == "work friend" {
		return RelationshipProfessional
	}

	// Friend relationships
	if keyword == "friend" || keyword == "buddy" || keyword == "pal" {
		return RelationshipFriend
	}

	// Family relationships
	if keyword == "family" || keyword == "parent" || keyword == "sibling" || keyword == "child" || keyword == "relative" {
		return RelationshipFamily
	}

	// Romantic relationships
	if keyword == "romantic" || keyword == "partner" || keyword == "boyfriend" || keyword == "girlfriend" || keyword == "spouse" || keyword == "lover" {
		return RelationshipRomantic
	}

	// Default
	return RelationshipOther
}

// MapKeywordToContext maps user keywords to valid context types
func MapKeywordToContext(keyword string) string {
	keyword = strings.ToLower(strings.TrimSpace(keyword))

	if keyword == "work" || keyword == "job" || keyword == "workplace" || keyword == "professional" {
		return ContextWork
	}
	if keyword == "personal" || keyword == "private" || keyword == "private life" {
		return ContextPersonal
	}
	if keyword == "social" || keyword == "friends" || keyword == "social life" || keyword == "out" || keyword == "partying" {
		return ContextSocial
	}
	if keyword == "family" || keyword == "home" || keyword == "parents" {
		return ContextFamily
	}

	return ContextGeneral
}
