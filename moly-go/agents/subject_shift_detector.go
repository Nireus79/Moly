package agents

import (
	"fmt"
	"log"
	"strings"
)

// SubjectShift represents a detected change in subject mid-conversation
type SubjectShift struct {
	From           string // Previous subject
	To             string // New subject
	Explicit       bool   // Did user explicitly name the new subject?
	Trigger        string // "but", "however", "unlike", explicit name
	MessageExcerpt string // Quote showing the shift
}

// SubjectShiftDetector detects when a user switches topics or subjects
type SubjectShiftDetector struct{}

// NewSubjectShiftDetector creates a new subject shift detector
func NewSubjectShiftDetector() *SubjectShiftDetector {
	return &SubjectShiftDetector{}
}

// DetectShifts analyzes a message and detects subject changes
// previousSubject: what the user was talking about before
// message: the current user message
func (d *SubjectShiftDetector) DetectShifts(message string, previousSubject string) []SubjectShift {
	log.Printf("[V2] SubjectShiftDetector.DetectShifts: previous=%s, message=%.60s...", previousSubject, message)

	if message == "" || previousSubject == "" {
		return []SubjectShift{}
	}

	shifts := []SubjectShift{}

	// Shift triggers: conjunctions that often precede subject changes
	triggers := []string{"but", "however", "unlike", "although", "though", "whereas", "while", "in contrast"}

	messageLower := strings.ToLower(message)
	for _, trigger := range triggers {
		// Try various patterns: " trigger ", " trigger,", "\ntrigger "
		var parts []string
		patterns := []string{
			" " + trigger + " ",
			" " + trigger + ",",
			"\n" + trigger + " ",
			"\n" + trigger + ",",
		}
		for _, pattern := range patterns {
			if strings.Contains(messageLower, pattern) {
				parts = strings.Split(messageLower, pattern)
				break
			}
		}
		if len(parts) == 0 {
			continue
		}

		if len(parts) > 1 {
			// Found a trigger word
			for i := 1; i < len(parts); i++ {
				afterTrigger := strings.TrimSpace(parts[i])
				newSubject := extractSubjectFromSegment(afterTrigger)

				if newSubject != "" && newSubject != "unknown" && newSubject != previousSubject {
					// Extract excerpt from original message (case-preserved)
					excerptStart := strings.Index(strings.ToLower(message), " "+trigger)
					if excerptStart == -1 {
						excerptStart = strings.Index(strings.ToLower(message), "\n"+trigger)
					}
					excerpt := ""
					if excerptStart != -1 {
						excerpt = strings.TrimSpace(message[excerptStart:])
						if len(excerpt) > 100 {
							excerpt = excerpt[:97] + "..."
						}
					}

					shift := SubjectShift{
						From:           previousSubject,
						To:             newSubject,
						Trigger:        trigger,
						Explicit:       isExplicitName(newSubject),
						MessageExcerpt: excerpt,
					}
					shifts = append(shifts, shift)

					log.Printf("[V2]   Detected shift: %s → %s (trigger: %s)", previousSubject, newSubject, trigger)
				}
			}
		}
	}

	// Also check for explicit name mentions that might indicate a subject shift
	// Look for common patterns like "John says..." or "My colleague..."
	explicitShifts := detectExplicitSubjectShifts(message, previousSubject)
	shifts = append(shifts, explicitShifts...)

	log.Printf("[V2] SubjectShiftDetector: detected %d shifts", len(shifts))
	return shifts
}

// extractSubjectFromSegment tries to identify the subject from a message segment
func extractSubjectFromSegment(segment string) string {
	lower := segment

	// Look for explicit role keywords (ordered list for consistent detection)
	roles := []struct {
		keyword string
		subject string
	}{
		{"boss", "contact_boss"},
		{"manager", "contact_boss"},
		{"director", "contact_boss"},
		{"colleague", "contact_colleague"},
		{"coworker", "contact_colleague"},
		{"teammate", "contact_colleague"},
		{"friend", "contact_friend"},
		{"partner", "contact_partner"},
		{"wife", "contact_partner"},
		{"husband", "contact_partner"},
		{"family", "contact_family"},
		{"mother", "contact_family"},
		{"father", "contact_family"},
		{"sister", "contact_family"},
		{"brother", "contact_family"},
	}

	for _, role := range roles {
		if strings.Contains(lower, role.keyword) {
			return role.subject
		}
	}

	// Look for pronouns (handle various punctuation)
	if strings.Contains(lower, "she ") || strings.Contains(lower, "she'") || strings.Contains(lower, " she") {
		return "unknown_female"
	}
	if strings.Contains(lower, "he ") || strings.Contains(lower, "he'") || strings.Contains(lower, " he") || strings.Contains(lower, ", he") {
		return "unknown_male"
	}
	if strings.Contains(lower, "they ") || strings.Contains(lower, "them ") {
		return "unknown_group"
	}

	// Look for proper names (capitalized words at start of segment)
	words := strings.Fields(segment)
	if len(words) > 0 && len(words[0]) > 1 {
		first := words[0]
		// Remove punctuation
		first = strings.Trim(first, ".,!?;:")
		if len(first) > 0 && first[0] >= 'A' && first[0] <= 'Z' {
			// Likely a name
			return fmt.Sprintf("contact_pending_%s", sanitizeName(first))
		}
	}

	return "unknown"
}

// isExplicitName checks if a subject reference is an explicit name (not a pronoun or generic term)
func isExplicitName(subject string) bool {
	// Explicit names start with "contact_" followed by something other than "pending"
	if strings.HasPrefix(subject, "contact_") && !strings.Contains(subject, "pending") {
		return false // These are already-known relationships
	}

	// Check if it's a pronoun or generic
	pronouns := []string{"unknown_female", "unknown_male", "unknown_group", "unknown_thing", "unknown"}
	for _, p := range pronouns {
		if subject == p {
			return false
		}
	}

	// If it has "pending_" in it, it's a name without known relationship
	return strings.Contains(subject, "pending_")
}

// detectExplicitSubjectShifts finds subject shifts based on explicit mentions
func detectExplicitSubjectShifts(message string, previousSubject string) []SubjectShift {
	shifts := []SubjectShift{}

	// Pattern: "My [role]" or "[Name] is"
	patterns := []struct {
		prefix string
		role   string
	}{
		{"my boss", "contact_boss"},
		{"my manager", "contact_boss"},
		{"my colleague", "contact_colleague"},
		{"my coworker", "contact_colleague"},
		{"my friend", "contact_friend"},
		{"my partner", "contact_partner"},
		{"my wife", "contact_partner"},
		{"my husband", "contact_partner"},
		{"my family", "contact_family"},
		{"my mother", "contact_family"},
		{"my father", "contact_family"},
	}

	lower := strings.ToLower(message)

	for _, pattern := range patterns {
		if strings.Contains(lower, pattern.prefix) && pattern.role != previousSubject {
			// Found a shift
			idx := strings.Index(lower, pattern.prefix)
			excerpt := message[idx:]
			if len(excerpt) > 100 {
				excerpt = excerpt[:97] + "..."
			}

			shift := SubjectShift{
				From:           previousSubject,
				To:             pattern.role,
				Trigger:        "explicit_mention",
				Explicit:       true,
				MessageExcerpt: excerpt,
			}
			shifts = append(shifts, shift)
			log.Printf("[V2]   Detected explicit shift: %s → %s", previousSubject, pattern.role)
			break // Only report one shift per message
		}
	}

	return shifts
}
