package config


// Contact relationship types - must match Contact model validation
const (
	RelationshipRomantic     = "romantic"
	RelationshipProfessional = "professional"
	RelationshipFamily       = "family"
	RelationshipFriend       = "friend"
	RelationshipOther        = "other"
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
	SeverityLow       = "low"
	SeverityMedium    = "medium"
	SeverityHigh      = "high"
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
	RiskLevelMinimal   = "minimal"
	RiskLevelLow       = "low"
	RiskLevelModerate  = "moderate"
	RiskLevelElevated  = "elevated"
	RiskLevelHigh      = "high"
	RiskLevelImmediate = "immediate"
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
	HighSeverityThreshold   = 60  // Severity score >= 60 is "high"
	MediumSeverityThreshold = 30  // Severity score >= 30 is "medium"
	DefaultLLMTimeoutSecs   = 30  // Default LLM call timeout
	MaxLLMTimeoutSecs       = 180 // Max timeout for long prompts
)




