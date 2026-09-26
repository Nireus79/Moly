package agents

import (
	"context"
	"encoding/json"
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
		// Parse LLM JSON response to extract new subject
		var shiftData map[string]interface{}
		if err := json.Unmarshal([]byte(resp.Content), &shiftData); err == nil {
			if newSubject, ok := shiftData["to"].(string); ok && newSubject != "" && newSubject != previousSubject {
				excerpt := extractExcerpt(message, 100)
				confidence := 0.85
				if conf, ok := shiftData["confidence"].(float64); ok {
					confidence = conf
				}
				return []SubjectShift{
					{
						From:           previousSubject,
						To:             newSubject,
						Trigger:        "llm",
						Explicit:       true,
						MessageExcerpt: excerpt,
						Confidence:     confidence,
					},
				}
			}
		}
	}

	return nil
}

// detectShiftsKeyword - DEPRECATED
// All keyword-based fallback detection has been removed
// Subject/contact shifts must be detected via LLM only (detectShiftsLLM above)
// If LLM unavailable or fails, return empty shifts instead of unreliable keyword matching
func (d *SubjectShiftDetector) detectShiftsKeyword(message string, previousSubject string) []SubjectShift {
	// REMOVED: No more keyword-based detection
	return []SubjectShift{}
}

// REMOVED: detectExplicitSubjectShifts method
// All subject shift detection is now LLM-based via detectShiftsLLM
// No more keyword pattern matching for role detection

// REMOVED: All hardcoded keyword extraction functions
// - extractSubjectFromMessage (role keyword matching)
// - extractSubjectFromSegment (legacy wrapper)
// - isExplicitSubject (hardcoded heuristics)
//
// Subject extraction now happens via:
// - LLM in detectShiftsLLM (structured JSON response)
// - ContextExtractor for contact information (LLM-based)

// extractExcerpt gets a portion of message showing the shift
func extractExcerpt(message string, maxLen int) string {
	if len(message) <= maxLen {
		return message
	}
	return message[:maxLen-3] + "..."
}

// sanitizeName already defined in subject_analyzer.go - reuse that
