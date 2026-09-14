package agents

import (
	"fmt"
	"log"
	"strings"
)

// SubjectResolver converts clarified responses into resolved subjects
// E.g., "Who is she?" + "Boss" → "contact_boss"
type SubjectResolver struct {
}

// NewSubjectResolver creates a new subject resolver
func NewSubjectResolver() *SubjectResolver {
	return &SubjectResolver{}
}

// ResolveSubject converts a response to a clarification question into a resolved subject
// ambiguousSubject: the original unclear subject (e.g., "unknown_female", "contact_pending_sarah")
// clarification: user's answer to the clarification question
func (r *SubjectResolver) ResolveSubject(ambiguousSubject string, clarification string) (string, error) {
	log.Printf("[V2] SubjectResolver: resolving %s with clarification: %s", ambiguousSubject, clarification)

	if clarification == "" {
		return "", fmt.Errorf("clarification required to resolve subject")
	}

	lower := strings.ToLower(clarification)

	// Map clarification to relationship
	relationship := ""
	if containsAny(lower, "boss", "manager", "director", "supervisor") {
		relationship = "boss"
	} else if containsAny(lower, "colleague", "coworker", "teammate") {
		relationship = "colleague"
	} else if containsAny(lower, "friend") {
		relationship = "friend"
	} else if containsAny(lower, "partner", "spouse", "wife", "husband") {
		relationship = "partner"
	} else if containsAny(lower, "family", "parent", "mother", "father", "sibling") {
		relationship = "family"
	} else {
		// Assume it's a new person to track
		return r.resolveName(clarification), nil
	}

	// Check if it's a name or just a role
	words := strings.Fields(clarification)

	// If single word like "Boss" or "Manager", just use relationship
	if len(words) == 1 {
		if relationship == "" {
			return "", fmt.Errorf("could not determine relationship from: %s", clarification)
		}
		resolved := fmt.Sprintf("contact_%s", relationship)
		log.Printf("[V2]   Resolved to: %s", resolved)
		return resolved, nil
	}

	// If multiple words like "My boss John" or "John who is my manager"
	name := r.extractName(clarification)
	if name != "" {
		resolved := fmt.Sprintf("contact_%s_%s", relationship, strings.ToLower(name))
		log.Printf("[V2]   Resolved to: %s", resolved)
		return resolved, nil
	}

	// Fallback to just relationship
	resolved := fmt.Sprintf("contact_%s", relationship)
	log.Printf("[V2]   Resolved to: %s (no name extracted)", resolved)
	return resolved, nil
}

// ResolveAmbiguousPronoun converts a pronoun with relationship info into a specific subject
// E.g., "unknown_female" + "Boss" → "contact_boss"
func (r *SubjectResolver) ResolveAmbiguousPronoun(pronounType string, clarification string) (string, error) {
	log.Printf("[V2] SubjectResolver: resolving pronoun %s with: %s", pronounType, clarification)

	// For pronouns, just use clarification as relationship indicator
	return r.ResolveSubject(pronounType, clarification)
}

// ResolvePendingName converts a pending contact into a known relationship
// E.g., "contact_pending_sarah" + "Boss" → "contact_boss_sarah"
func (r *SubjectResolver) ResolvePendingName(pendingSubject string, clarification string) (string, error) {
	log.Printf("[V2] SubjectResolver: resolving pending %s with: %s", pendingSubject, clarification)

	if !strings.HasPrefix(pendingSubject, "contact_pending_") {
		return "", fmt.Errorf("not a pending subject: %s", pendingSubject)
	}

	// Extract name from pending subject
	name := strings.TrimPrefix(pendingSubject, "contact_pending_")

	// Determine relationship from clarification
	lower := strings.ToLower(clarification)

	relationship := ""
	if containsAny(lower, "boss", "manager") {
		relationship = "boss"
	} else if containsAny(lower, "colleague", "coworker") {
		relationship = "colleague"
	} else if containsAny(lower, "friend") {
		relationship = "friend"
	} else if containsAny(lower, "partner", "spouse") {
		relationship = "partner"
	} else if containsAny(lower, "family", "parent") {
		relationship = "family"
	} else {
		// Unknown relationship, keep pending
		return pendingSubject, nil
	}

	resolved := fmt.Sprintf("contact_%s_%s", relationship, name)
	log.Printf("[V2]   Resolved pending to: %s", resolved)
	return resolved, nil
}

// resolveName extracts a name from free-text response and creates pending contact
func (r *SubjectResolver) resolveName(response string) string {
	name := r.extractName(response)
	if name != "" {
		resolved := fmt.Sprintf("contact_pending_%s", sanitizeName(name))
		log.Printf("[V2]   Resolved to pending contact: %s", resolved)
		return resolved
	}
	return "unknown"
}

// extractName tries to find a name in the response
// Heuristic: capitalized word that's not in the middle of a sentence
func (r *SubjectResolver) extractName(response string) string {
	words := strings.Fields(response)

	// Skip these known relationship keywords
	skipwords := map[string]bool{
		"boss": true, "manager": true, "director": true, "supervisor": true,
		"colleague": true, "coworker": true, "teammate": true,
		"friend": true, "partner": true, "spouse": true, "wife": true, "husband": true,
		"family": true, "parent": true, "mother": true, "father": true, "sibling": true,
		"i": true, "my": true, "the": true, "a": true, "and": true, "or": true,
		"is": true, "are": true, "was": true, "were": true,
	}

	// Look for capitalized words (likely names)
	for i, word := range words {
		// Skip punctuation
		cleaned := strings.TrimRight(word, ".,!?;:")
		cleanedLower := strings.ToLower(cleaned)

		// Capitalized word that's not a known skip word
		if len(cleaned) > 0 && cleaned[0] >= 'A' && cleaned[0] <= 'Z' {
			if !skipwords[cleanedLower] {
				// For names, we want the first word or proper noun sequence
				if i == 0 || (i > 0 && words[i-1] != "is" && words[i-1] != "like") {
					return cleaned
				}
			}
		}
	}

	return ""
}

// GetContactDisplayName formats a subject for display
// E.g., "contact_boss_sarah" → "Sarah (Boss)" or "contact_boss" → "Boss"
func (r *SubjectResolver) GetContactDisplayName(subject string) string {
	if !strings.HasPrefix(subject, "contact_") {
		return subject
	}

	parts := strings.Split(subject, "_")
	if len(parts) < 2 {
		return subject
	}

	// contact_boss → "Boss"
	if len(parts) == 2 {
		relationship := parts[1]
		return strings.ToTitle(strings.ToLower(relationship))
	}

	// contact_boss_sarah → "Sarah (Boss)"
	if len(parts) >= 3 {
		relationship := parts[1]
		name := strings.Join(parts[2:], " ")
		return fmt.Sprintf("%s (%s)", name, strings.ToTitle(relationship))
	}

	return subject
}
