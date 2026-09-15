package agents

import (
	"context"
	"fmt"
	"log"
	"strings"

	"moly/tools"
)

// SubjectShift represents a detected change in subject mid-conversation
type SubjectShift struct {
	From           string  // Previous subject
	To             string  // New subject
	Explicit       bool    // Did user explicitly name the new subject?
	Trigger        string  // Detection method: "llm", "keyword_pattern", "explicit_mention"
	MessageExcerpt string  // Quote showing the shift
	Confidence     float64 // 0-1 confidence in this shift detection
}

// SubjectShiftDetector detects when a user switches topics or subjects
type SubjectShiftDetector struct {
	llmClient tools.LLMProvider
}

// NewSubjectShiftDetector creates a new subject shift detector (no LLM)
func NewSubjectShiftDetector() *SubjectShiftDetector {
	return &SubjectShiftDetector{
		llmClient: nil,
	}
}

// NewSubjectShiftDetectorWithLLM creates detector with LLM capability
func NewSubjectShiftDetectorWithLLM(llm tools.LLMProvider) *SubjectShiftDetector {
	return &SubjectShiftDetector{
		llmClient: llm,
	}
}

// DetectShifts analyzes a message and detects subject changes
// Uses LLM first (logic-based), falls back to keyword patterns
func (d *SubjectShiftDetector) DetectShifts(message string, previousSubject string) []SubjectShift {
	log.Printf("[SubjectShiftDetector] Detecting shifts: previous=%s, message=%.60s...", previousSubject, message)

	if message == "" || previousSubject == "" {
		return []SubjectShift{}
	}

	// Try LLM-based detection first (logic-based reasoning)
	if d.llmClient != nil {
		shifts := d.detectShiftsLLM(message, previousSubject)
		if len(shifts) > 0 {
			log.Printf("[SubjectShiftDetector] LLM detected %d shift(s)", len(shifts))
			return shifts
		}
	}

	// Fallback to keyword-based detection
	log.Printf("[SubjectShiftDetector] Using keyword-based fallback detection")
	return d.detectShiftsKeyword(message, previousSubject)
}

// detectShiftsLLM uses LLM reasoning to understand if topic changed
func (d *SubjectShiftDetector) detectShiftsLLM(message string, previousSubject string) []SubjectShift {
	req := &tools.LLMRequest{
		SystemPrompt: `You are an expert at understanding conversation flow and topic changes.
Analyze if the user is talking about a different person/subject than before.
Respond with ONLY "no_shift" if same subject, or JSON with shift details if changed.

If shift detected, respond with:
{"shift": true, "from": "<previous>", "to": "<new subject>", "reason": "<why>", "confidence": 0-1}`,
		UserPrompt: fmt.Sprintf(`Previous subject: "%s"
Current message: "%s"

Is the user talking about a different person/subject now?`, previousSubject, message),
		MaxTokens:   150,
		Temperature: 0.2,
		Retries:     1,
	}

	resp, err := d.llmClient.Call(context.Background(), req)
	if err != nil {
		log.Printf("[SubjectShiftDetector] LLM error: %v, falling back", err)
		return nil
	}

	lower := strings.ToLower(strings.TrimSpace(resp.Content))

	// Check for shift indication
	if strings.Contains(lower, "\"shift\": true") || strings.Contains(lower, "shift: true") {
		// Extract what the new subject is
		newSubject := extractSubjectFromMessage(message)
		if newSubject != "" && newSubject != previousSubject {
			excerpt := extractExcerpt(message, 100)
			return []SubjectShift{
				{
					From:           previousSubject,
					To:             newSubject,
					Trigger:        "llm",
					Explicit:       isExplicitSubject(newSubject),
					MessageExcerpt: excerpt,
					Confidence:     0.85, // LLM-detected shifts are high confidence
				},
			}
		}
	}

	return nil
}

// detectShiftsKeyword uses keyword patterns as fallback when LLM unavailable
func (d *SubjectShiftDetector) detectShiftsKeyword(message string, previousSubject string) []SubjectShift {
	shifts := []SubjectShift{}

	// First check for explicit mentions (highest confidence)
	if explicit := d.detectExplicitSubjectShifts(message, previousSubject); len(explicit) > 0 {
		return explicit
	}

	// Then check for trigger words that indicate topic change
	triggers := []string{"but", "however", "unlike", "although", "though", "whereas", "while", "in contrast"}

	messageLower := strings.ToLower(message)
	for _, trigger := range triggers {
		patterns := []string{
			" " + trigger + " ",
			" " + trigger + ",",
			"\n" + trigger + " ",
			"\n" + trigger + ",",
		}

		for _, pattern := range patterns {
			if strings.Contains(messageLower, pattern) {
				parts := strings.Split(messageLower, pattern)
				if len(parts) > 1 {
					afterTrigger := strings.TrimSpace(parts[1])
					newSubject := extractSubjectFromSegment(afterTrigger)

					if newSubject != "" && newSubject != "unknown" && newSubject != previousSubject {
						excerpt := extractExcerpt(message, 100)
						shift := SubjectShift{
							From:           previousSubject,
							To:             newSubject,
							Trigger:        "keyword_pattern",
							Explicit:       isExplicitSubject(newSubject),
							MessageExcerpt: excerpt,
							Confidence:     0.65, // Keyword-based is lower confidence
						}
						shifts = append(shifts, shift)
						log.Printf("[SubjectShiftDetector] Keyword detected shift: %s → %s", previousSubject, newSubject)
						return shifts // Return first detected shift
					}
				}
			}
		}
	}

	return shifts
}

// detectExplicitSubjectShifts finds shifts based on explicit mentions (highest confidence)
func (d *SubjectShiftDetector) detectExplicitSubjectShifts(message string, previousSubject string) []SubjectShift {
	shifts := []SubjectShift{}

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
				Confidence:     0.95, // Explicit mentions are very high confidence
			}
			shifts = append(shifts, shift)
			log.Printf("[SubjectShiftDetector] Explicit shift detected: %s → %s", previousSubject, pattern.role)
			break // Only one explicit shift per message
		}
	}

	return shifts
}

// extractSubjectFromMessage identifies the subject/person in a message using keyword fallback
func extractSubjectFromMessage(message string) string {
	lower := strings.ToLower(message)

	// Check for explicit role keywords
	roles := []struct {
		keyword string
		subject string
	}{
		{"boss", "contact_boss"},
		{"manager", "contact_boss"},
		{"colleague", "contact_colleague"},
		{"friend", "contact_friend"},
		{"partner", "contact_partner"},
		{"wife", "contact_partner"},
		{"husband", "contact_partner"},
		{"mother", "contact_family"},
		{"father", "contact_family"},
		{"family", "contact_family"},
	}

	for _, role := range roles {
		if strings.Contains(lower, role.keyword) {
			return role.subject
		}
	}

	// Check for pronouns
	if strings.Contains(lower, "she") {
		return "unknown_female"
	}
	if strings.Contains(lower, "he") {
		return "unknown_male"
	}
	if strings.Contains(lower, "they") {
		return "unknown_group"
	}

	// Check for capitalized names
	words := strings.Fields(message)
	for _, word := range words {
		clean := strings.Trim(word, ".,!?;:")
		if len(clean) > 1 && clean[0] >= 'A' && clean[0] <= 'Z' {
			return fmt.Sprintf("contact_pending_%s", sanitizeName(clean))
		}
	}

	return "unknown"
}

// extractSubjectFromSegment tries to identify subject from message segment (legacy)
func extractSubjectFromSegment(segment string) string {
	return extractSubjectFromMessage(segment)
}

// isExplicitSubject checks if subject is explicitly named vs inferred
func isExplicitSubject(subject string) bool {
	if strings.Contains(subject, "pending_") {
		return true // Named person
	}
	if strings.HasPrefix(subject, "contact_") && !strings.Contains(subject, "pending") {
		return true // Known relationship
	}
	if strings.HasPrefix(subject, "unknown_") {
		return false // Inferred from pronoun
	}
	return false
}

// extractExcerpt gets a portion of message showing the shift
func extractExcerpt(message string, maxLen int) string {
	if len(message) <= maxLen {
		return message
	}
	return message[:maxLen-3] + "..."
}

// sanitizeName already defined in subject_analyzer.go - reuse that
