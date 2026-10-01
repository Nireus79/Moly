package tools

import (
	"fmt"
	"regexp"
	"strings"
)

// GroupReference represents a reference to a group of people
type GroupReference struct {
	ID              int64
	UserID          string
	ConversationID  string
	ReferencePronoun string // "they", "we", "both", "all", "us"
	ReferenceType   string  // "dual" (2), "plural" (3+), "collection"
	Members         []string
	MemberIDs       map[string]int64 // name → contact ID
	IsUserInGroup   bool
	GroupContext    string // "couple", "trio", "group", "friends", "team"
	Confidence      float64
	EvidenceText    string
	DetectedFrom    string // Which sentence/message detected this
	CreatedAt       int64
}

// GroupDetector analyzes text to identify group references and membership
type GroupDetector struct{}

// NewGroupDetector creates a new group detector
func NewGroupDetector() *GroupDetector {
	return &GroupDetector{}
}

// ============================================================================
// GROUP PRONOUN DETECTION
// ============================================================================

// DetectGroupPronouns finds all group pronouns in a message
// Returns: List of group pronouns with their positions
func (gd *GroupDetector) DetectGroupPronouns(message string) []PronounReference {
	var groupPronouns []PronounReference

	groupPronounList := map[string]string{
		"they":     "personal_plural",
		"them":     "personal_plural",
		"their":    "personal_plural",
		"theirs":   "personal_plural",
		"we":       "personal_plural",
		"us":       "personal_plural",
		"our":      "personal_plural",
		"ours":     "personal_plural",
		"both":     "dual",
		"all":      "collection",
		"everyone": "collection",
	}

	lower := strings.ToLower(message)
	words := strings.FieldsFunc(lower, func(r rune) bool {
		return r == ' ' || r == '\n' || r == '\t' || r == ',' || r == '.' || r == '!' || r == '?'
	})

	for _, word := range words {
		if ptype, ok := groupPronounList[word]; ok {
			groupPronouns = append(groupPronouns, PronounReference{
				Pronoun:     word,
				PronounType: ptype,
				Confidence:  0.95,
			})
		}
	}

	return groupPronouns
}

// ============================================================================
// GROUP MEMBERSHIP DETECTION
// ============================================================================

// DetectGroupMembers analyzes text to determine who is in a group
// Uses patterns like "both X and Y", "me and X", "X, Y, and Z"
func (gd *GroupDetector) DetectGroupMembers(text string, knownEntities []string) []string {
	var members []string
	lower := strings.ToLower(text)

	// Pattern 1: "both X and Y"
	bothPattern := regexp.MustCompile(`(?i)\bboth\s+([a-zA-Z]+)\s+and\s+([a-zA-Z]+)\b`)
	if matches := bothPattern.FindStringSubmatch(lower); len(matches) > 2 {
		members = append(members, matches[1], matches[2])
		return members
	}

	// Pattern 2: "me and X" or "I and X"
	mePattern := regexp.MustCompile(`(?i)\b(me|i|myself)\s+and\s+([a-zA-Z]+)\b`)
	if matches := mePattern.FindStringSubmatch(lower); len(matches) > 2 {
		members = append(members, "user")
		members = append(members, matches[2])
		return members
	}

	// Pattern 3: "X and me" or "X and I"
	andMePattern := regexp.MustCompile(`(?i)\b([a-zA-Z]+)\s+and\s+(me|i|myself)\b`)
	if matches := andMePattern.FindStringSubmatch(lower); len(matches) > 2 {
		members = append(members, matches[1])
		members = append(members, "user")
		return members
	}

	// Pattern 4: "X, Y, and Z" (comma-separated list)
	commaPattern := regexp.MustCompile(`([a-zA-Z]+),\s*([a-zA-Z]+),?\s+and\s+([a-zA-Z]+)`)
	if matches := commaPattern.FindStringSubmatch(lower); len(matches) > 3 {
		members = append(members, matches[1], matches[2], matches[3])
		return members
	}

	// Pattern 5: "all of us" or "all of them"
	if strings.Contains(lower, "all of us") || strings.Contains(lower, "all of us") {
		// "All of us" usually means everyone mentioned so far
		return knownEntities
	}

	// Pattern 6: "both of us"
	if strings.Contains(lower, "both of us") {
		// Usually means user + one other known entity
		if len(knownEntities) > 0 {
			return []string{"user", knownEntities[len(knownEntities)-1]}
		}
		return []string{"user"}
	}

	// Pattern 7: "we three" or "we two"
	weNumPattern := regexp.MustCompile(`(?i)\bwe\s+(two|three|four|both)\b`)
	if matches := weNumPattern.FindStringSubmatch(lower); len(matches) > 1 {
		// Extract mentioned entities
		return knownEntities
	}

	return members
}

// ============================================================================
// GROUP TYPE CLASSIFICATION
// ============================================================================

// ClassifyGroupType determines if group is dual (2), plural (3+), or collection
func (gd *GroupDetector) ClassifyGroupType(pronoun string, memberCount int) string {
	lower := strings.ToLower(pronoun)

	if lower == "both" {
		return "dual"
	}

	if memberCount == 2 || (lower == "we" && memberCount == 2) {
		return "dual"
	}

	if memberCount >= 3 || lower == "all" || lower == "everyone" {
		return "plural"
	}

	if memberCount > 3 {
		return "plural"
	}

	// Generic plural
	return "plural"
}

// ============================================================================
// GROUP CONTEXT INFERENCE
// ============================================================================

// InferGroupContext analyzes text to determine the nature of the group
// Returns: "couple", "trio", "group", "friends", "team", "family", "unknown"
func (gd *GroupDetector) InferGroupContext(text string, memberCount int) string {
	lower := strings.ToLower(text)

	// Keywords indicating relationship/couple
	relationshipKeywords := []string{
		"romantic", "dating", "relationship", "partner", "wife", "husband",
		"boyfriend", "girlfriend", "spouse", "couple", "together",
	}
	for _, keyword := range relationshipKeywords {
		if strings.Contains(lower, keyword) {
			if memberCount == 2 {
				return "couple"
			}
		}
	}

	// Keywords indicating friends
	friendKeywords := []string{
		"friend", "friends", "buddy", "mate", "pal", "pals", "companions",
		"hang out", "party", "group",
	}
	for _, keyword := range friendKeywords {
		if strings.Contains(lower, keyword) {
			if memberCount >= 3 {
				return "friends"
			}
			return "group"
		}
	}

	// Keywords indicating work/team
	teamKeywords := []string{
		"team", "work", "workplace", "colleague", "coworker", "boss", "manager",
		"company", "office",
	}
	for _, keyword := range teamKeywords {
		if strings.Contains(lower, keyword) {
			return "team"
		}
	}

	// Keywords indicating family
	familyKeywords := []string{
		"family", "parent", "sibling", "brother", "sister", "mother", "father",
		"mom", "dad", "children", "kids",
	}
	for _, keyword := range familyKeywords {
		if strings.Contains(lower, keyword) {
			return "family"
		}
	}

	// Default based on count
	if memberCount == 2 {
		return "couple"
	}
	if memberCount == 3 {
		return "trio"
	}

	return "group"
}

// ============================================================================
// GROUP CONTEXT ANALYSIS
// ============================================================================

// AnalyzeGroupContext performs comprehensive group analysis from text
func (gd *GroupDetector) AnalyzeGroupContext(
	text string,
	groupPronoun string,
	knownEntities []string,
	knownContacts map[string]int64,
) *GroupReference {

	ref := &GroupReference{
		ReferencePronoun: groupPronoun,
		EvidenceText:     text,
		Confidence:       0.75,
	}

	// Detect members
	members := gd.DetectGroupMembers(text, knownEntities)
	if len(members) == 0 {
		// If no explicit members found, use known entities
		members = knownEntities
	}

	ref.Members = members

	// Check if user is in group
	lower := strings.ToLower(text)
	if strings.Contains(lower, "we") || strings.Contains(lower, "us") ||
		strings.Contains(lower, "our") || strings.Contains(lower, "me and") ||
		strings.Contains(lower, "i and") {
		ref.IsUserInGroup = true
	}

	// Classify type
	ref.ReferenceType = gd.ClassifyGroupType(groupPronoun, len(members))

	// Infer context
	ref.GroupContext = gd.InferGroupContext(text, len(members))

	// Build member ID map
	ref.MemberIDs = make(map[string]int64)
	for memberName, contactID := range knownContacts {
		for _, member := range members {
			if strings.ToLower(member) == strings.ToLower(memberName) {
				ref.MemberIDs[member] = contactID
			}
		}
	}

	// Adjust confidence based on clarity
	if len(members) == 0 {
		ref.Confidence = 0.30 // Low confidence - no explicit members
	} else if len(members) <= 2 && ref.ReferenceType == "dual" {
		ref.Confidence = 0.90 // High confidence - explicit "both X and Y"
	} else if len(members) > 2 {
		ref.Confidence = 0.75 // Medium-high confidence - explicit list
	}

	return ref
}

// ============================================================================
// AMBIGUITY DETECTION
// ============================================================================

// IsAmbiguous returns true if group membership is unclear
// Group is ambiguous if:
// - No explicit members mentioned
// - "they" without recent context
// - "we" but unclear who is included
func (gd *GroupDetector) IsAmbiguous(ref *GroupReference) bool {
	if len(ref.Members) == 0 {
		return true
	}

	if strings.ToLower(ref.ReferencePronoun) == "they" || strings.ToLower(ref.ReferencePronoun) == "them" {
		// "They" is ambiguous unless we have recent context
		return ref.Confidence < 0.70
	}

	return ref.Confidence < 0.50
}

// ============================================================================
// GROUP RELATIONSHIP MAPPING
// ============================================================================

// MapGroupRelationships returns what the group relationships are
// e.g., if group is "me and Christine", returns "user_with_christine"
func (gd *GroupDetector) MapGroupRelationships(ref *GroupReference) []string {
	var relationships []string

	if len(ref.Members) == 2 {
		member1 := ref.Members[0]
		member2 := ref.Members[1]

		if strings.ToLower(member1) == "user" || ref.IsUserInGroup {
			other := member2
			if strings.ToLower(member2) == "user" {
				other = member1
			}
			relationships = append(relationships, fmt.Sprintf("user_with_%s", other))
		} else {
			relationships = append(relationships, fmt.Sprintf("%s_with_%s", member1, member2))
		}
	} else if len(ref.Members) > 2 {
		// For groups > 2, create "group_context" relationship
		relationships = append(relationships, fmt.Sprintf("%s_%s", strings.Join(ref.Members, "_"), ref.GroupContext))
	}

	return relationships
}

// ============================================================================
// SCOPE TRACKING
// ============================================================================

// IsValidForContext checks if this group reference is still valid in current context
// A group reference expires when:
// - A completely new group is mentioned
// - Context shifts significantly
// - Explicit scope-ending phrase appears
func (gd *GroupDetector) IsValidForContext(ref *GroupReference, newText string) bool {
	lower := strings.ToLower(newText)

	// Scope-ending phrases
	scopeEnders := []string{
		"without", "except", "but not", "alone", "solo", "just me",
		"separately", "different", "other people",
	}

	for _, ender := range scopeEnders {
		if strings.Contains(lower, ender) {
			return false
		}
	}

	// If new group mentioned, old one is invalid
	if gd.DetectGroupPronouns(newText) != nil {
		// New group reference - old one may be superseded
		// This is conservative; actual logic would compare members
		return false
	}

	return true
}

// ============================================================================
// GENDER COMPOSITION ANALYSIS
// ============================================================================

// AnalyzeGenderComposition determines likely gender composition of group
// Returns: "mixed", "male", "female", "unknown"
func (gd *GroupDetector) AnalyzeGenderComposition(members []string, nameGuesser func(string) string) string {
	if len(members) == 0 {
		return "unknown"
	}

	maleCount := 0
	femaleCount := 0

	for _, member := range members {
		if strings.ToLower(member) == "user" {
			// Skip "user" - we don't know gender from pronoun alone
			continue
		}

		gender := nameGuesser(member)
		if gender == "male" {
			maleCount++
		} else if gender == "female" {
			femaleCount++
		}
	}

	if maleCount > 0 && femaleCount > 0 {
		return "mixed"
	}
	if maleCount > 0 {
		return "male"
	}
	if femaleCount > 0 {
		return "female"
	}

	return "unknown"
}
