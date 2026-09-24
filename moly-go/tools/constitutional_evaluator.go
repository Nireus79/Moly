package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"moly/models"
)

// PrincipleMatch represents a matched principle violation
type PrincipleMatch struct {
	PrincipleID string // e.g., "harm_prevention"
	Name        string // e.g., "Harm Prevention"
	Severity    string // critical, high, medium, low (from constitution.yaml, not LLM)
	Evidence    string // quoted substring from the evaluated text
	Reasoning   string // why this principle was violated
}

// ConstitutionalVerdict is the result of evaluating a message against the constitution
type ConstitutionalVerdict struct {
	Allowed             bool              // true if no critical/high principle violations
	OverallSeverity     string            // critical, high, medium, low, clear
	MatchedPrinciples   []PrincipleMatch  // all principles that were violated
	Reasoning           string            // summary of the evaluation
	EvaluatedText       string            // the text that was evaluated
	Confidence          float64           // 0.0-1.0
	LLMReasoning        string            // full LLM response for debugging
}

// ConstitutionalEvaluator evaluates messages against constitutional principles
type ConstitutionalEvaluator struct {
	llm            LLMProvider
	constitution   *models.Constitution
}

// NewConstitutionalEvaluator creates a new constitutional evaluator
func NewConstitutionalEvaluator(llm LLMProvider, constitution *models.Constitution) *ConstitutionalEvaluator {
	if constitution == nil {
		log.Printf("[ConstitutionalEvaluator] WARNING: constitution is nil")
	}
	return &ConstitutionalEvaluator{
		llm:            llm,
		constitution:   constitution,
	}
}

// EvaluateDeterministic runs Tier 1a/1b checks only (no LLM, deterministic)
// Returns verdict if any principle is violated, nil if all clear
// This is fast and reliable for use before response generation
func (ce *ConstitutionalEvaluator) EvaluateDeterministic(text string) *ConstitutionalVerdict {
	if text == "" {
		return nil // No violation for empty
	}

	if ce.constitution == nil {
		log.Printf("[ConstitutionalEvaluator] WARNING: Cannot run deterministic check - constitution is nil")
		return nil
	}

	text = strings.TrimSpace(text)
	lowerText := strings.ToLower(text)

	verdict := &ConstitutionalVerdict{
		Allowed:         true,
		OverallSeverity: "clear",
		EvaluatedText:   text,
		Confidence:      1.0,
	}

	// Tier 1a: Check each principle's violation patterns (hard blocks)
	severityOrder := map[string]int{"critical": 4, "high": 3, "medium": 2, "low": 1, "clear": 0}
	maxSeverity := "clear"

	for _, principle := range ce.constitution.SupremePrinciples {
		// Check if any violation pattern matches
		for _, violationPattern := range principle.Violations {
			if strings.Contains(lowerText, strings.ToLower(violationPattern)) {
				match := PrincipleMatch{
					PrincipleID: principle.ID,
					Name:        principle.Name,
					Severity:    principle.Severity,
					Evidence:    violationPattern,
					Reasoning:   fmt.Sprintf("Matched violation pattern from %s", principle.Name),
				}
				verdict.MatchedPrinciples = append(verdict.MatchedPrinciples, match)

				if severityOrder[principle.Severity] > severityOrder[maxSeverity] {
					maxSeverity = principle.Severity
				}

				log.Printf("[ConstitutionalEvaluator.Deterministic] ✓ Tier 1a match: %s (%s)", principle.Name, principle.Severity)
				break // One violation per principle is enough
			}
		}
	}

	// Tier 1b: Scan for keyword signals (soft signals, only matters if no Tier 1a match)
	if maxSeverity == "clear" {
		for _, principle := range ce.constitution.SupremePrinciples {
			keywordCount := 0
			for _, keyword := range principle.CheckKeywords {
				if strings.Contains(lowerText, strings.ToLower(keyword)) {
					keywordCount++
				}
			}
			// If we found keywords, it's a signal but not a hard block
			if keywordCount > 0 {
				log.Printf("[ConstitutionalEvaluator.Deterministic] Tier 1b signal: %s (%d keywords matched)",
					principle.Name, keywordCount)
			}
		}
		// Tier 1b: Zero signals across all principles → definitely allow (no LLM needed)
		log.Printf("[ConstitutionalEvaluator.Deterministic] ✓ Tier 1b clear: no keyword signals detected")
		return nil
	}

	// Set verdict based on max severity
	verdict.OverallSeverity = maxSeverity
	if maxSeverity == "critical" || maxSeverity == "high" {
		verdict.Allowed = false
	}

	if len(verdict.MatchedPrinciples) > 0 {
		var reasons []string
		for _, m := range verdict.MatchedPrinciples {
			reasons = append(reasons, fmt.Sprintf("%s (%s)", m.Name, m.Severity))
		}
		verdict.Reasoning = fmt.Sprintf("Deterministic violations: %s", strings.Join(reasons, ", "))
		log.Printf("[ConstitutionalEvaluator.Deterministic] ✓ Verdict: allowed=%v, severity=%s, matches=%d",
			verdict.Allowed, verdict.OverallSeverity, len(verdict.MatchedPrinciples))
		return verdict
	}

	return nil
}

// Evaluate analyzes a message against constitutional principles
// Returns a verdict that indicates whether the message violates any principles
func (ce *ConstitutionalEvaluator) Evaluate(ctx context.Context, text string) (*ConstitutionalVerdict, error) {
	if text == "" {
		return &ConstitutionalVerdict{
			Allowed:         true,
			OverallSeverity: "clear",
			Reasoning:       "Empty text",
			Confidence:      1.0,
		}, nil
	}

	if ce.llm == nil {
		log.Printf("[ConstitutionalEvaluator] ERROR: No LLM available")
		return nil, fmt.Errorf("LLM provider is nil")
	}

	if ce.constitution == nil {
		log.Printf("[ConstitutionalEvaluator] ERROR: No constitution loaded")
		return nil, fmt.Errorf("constitution is nil")
	}

	text = strings.TrimSpace(text)
	log.Printf("[ConstitutionalEvaluator] Evaluating message (%d chars) against %d principles",
		len(text), len(ce.constitution.SupremePrinciples))

	// Build system prompt from constitution
	systemPrompt := ce.buildSystemPrompt()

	// Build user prompt
	userPrompt := ce.buildUserPrompt(text)

	// Call LLM
	req := &LLMRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		MaxTokens:    500,
		Temperature:  0.3,
		Retries:      2,
	}

	log.Printf("[ConstitutionalEvaluator] ▶ LLM call: temp=%.1f, max_tokens=%d, retries=%d",
		req.Temperature, req.MaxTokens, req.Retries)

	resp, err := ce.llm.Call(ctx, req)
	if err != nil {
		log.Printf("[ConstitutionalEvaluator] ✗ LLM call failed: %v", err)
		return nil, fmt.Errorf("LLM evaluation failed: %w", err)
	}

	log.Printf("[ConstitutionalEvaluator] ◄ LLM response: %d chars", len(resp.Content))

	// Validate and parse response
	verdict, err := ce.validateAndParse(resp.Content, text)
	if err != nil {
		log.Printf("[ConstitutionalEvaluator] ✗ Validation failed: %v", err)
		return nil, fmt.Errorf("response validation failed: %w", err)
	}

	verdict.EvaluatedText = text
	verdict.LLMReasoning = resp.Content

	// Log result
	log.Printf("[ConstitutionalEvaluator] ✓ Evaluation complete: allowed=%v, severity=%s, matches=%d",
		verdict.Allowed, verdict.OverallSeverity, len(verdict.MatchedPrinciples))

	for _, m := range verdict.MatchedPrinciples {
		log.Printf("  - %s (%s): %s", m.Name, m.Severity, m.Evidence)
	}

	return verdict, nil
}

// buildSystemPrompt creates a system prompt from the constitution
func (ce *ConstitutionalEvaluator) buildSystemPrompt() string {
	var sb strings.Builder

	sb.WriteString("You are a constitutional ethics evaluator. Your task is to analyze a message ")
	sb.WriteString("against these core principles and determine if any are violated.\n\n")

	sb.WriteString("PRINCIPLES:\n")
	sb.WriteString("============\n")

	for i, p := range ce.constitution.SupremePrinciples {
		sb.WriteString(fmt.Sprintf("\n%d. %s (id: %s, severity: %s)\n", i+1, p.Name, p.ID, p.Severity))
		sb.WriteString(fmt.Sprintf("   %s\n", p.Description))
		sb.WriteString("   Violations:\n")
		for _, v := range p.Violations {
			sb.WriteString(fmt.Sprintf("   - %s\n", v))
		}
	}

	sb.WriteString("\n\nRESPONSE FORMAT:\n")
	sb.WriteString("================\n")
	sb.WriteString("Respond with ONLY a JSON object (no markdown, no explanation):\n")
	sb.WriteString(`{"violations": [{"principle_id": "id", "evidence": "exact quote from message", "reasoning": "why"}]}`)
	sb.WriteString("\n\n")
	sb.WriteString("GUIDELINES:\n")
	sb.WriteString("- Only include principles that are actually violated by the message\n")
	sb.WriteString("- 'evidence' MUST be an exact substring from the message (quote the words used)\n")
	sb.WriteString("- If no principles are violated, return: {\"violations\": []}\n")
	sb.WriteString("- Be precise and specific, not overly strict\n")

	return sb.String()
}

// buildUserPrompt creates a user prompt for evaluation
func (ce *ConstitutionalEvaluator) buildUserPrompt(text string) string {
	return fmt.Sprintf("Analyze this message for principle violations:\n\n\"%s\"", text)
}

// validateAndParse validates and parses the LLM response
func (ce *ConstitutionalEvaluator) validateAndParse(rawResponse string, originalText string) (*ConstitutionalVerdict, error) {
	// Parse JSON response
	var parsed struct {
		Violations []struct {
			PrincipleID string `json:"principle_id"`
			Evidence    string `json:"evidence"`
			Reasoning   string `json:"reasoning"`
		} `json:"violations"`
	}

	if err := json.Unmarshal([]byte(rawResponse), &parsed); err != nil {
		log.Printf("[ConstitutionalEvaluator] Failed to parse JSON: %v", err)
		return nil, fmt.Errorf("invalid JSON response: %w", err)
	}

	verdict := &ConstitutionalVerdict{
		Allowed:           true,
		OverallSeverity:   "clear",
		MatchedPrinciples: []PrincipleMatch{},
		Confidence:        1.0,
	}

	if len(parsed.Violations) == 0 {
		verdict.Reasoning = "No principle violations detected"
		return verdict, nil
	}

	// Build a map of principle IDs to Principle objects for fast lookup
	principleMap := make(map[string]*models.Principle)
	for i, p := range ce.constitution.SupremePrinciples {
		principleMap[p.ID] = &ce.constitution.SupremePrinciples[i]
	}

	// Validate each violation
	maxSeverity := "low"
	severityOrder := map[string]int{"critical": 4, "high": 3, "medium": 2, "low": 1, "clear": 0}

	for _, v := range parsed.Violations {
		// Validate principle ID exists
		principle, exists := principleMap[v.PrincipleID]
		if !exists {
			log.Printf("[ConstitutionalEvaluator] ✗ Dropping unknown principle: %s", v.PrincipleID)
			continue
		}

		// Validate evidence is a substring (hallucination guard)
		if !strings.Contains(strings.ToLower(originalText), strings.ToLower(v.Evidence)) {
			log.Printf("[ConstitutionalEvaluator] ✗ Dropping unverified evidence: %q not in message", v.Evidence)
			continue
		}

		// Severity is ALWAYS from the principle, never from LLM
		match := PrincipleMatch{
			PrincipleID: v.PrincipleID,
			Name:        principle.Name,
			Severity:    principle.Severity, // Force to principle's declared severity
			Evidence:    v.Evidence,
			Reasoning:   v.Reasoning,
		}

		verdict.MatchedPrinciples = append(verdict.MatchedPrinciples, match)

		// Update max severity
		if severityOrder[principle.Severity] > severityOrder[maxSeverity] {
			maxSeverity = principle.Severity
		}

		log.Printf("[ConstitutionalEvaluator] ✓ Validated: %s (%s)", principle.Name, principle.Severity)
	}

	// Set overall severity and allowed flag
	verdict.OverallSeverity = maxSeverity

	// Decision logic: critical or high severity → not allowed; others → allowed but flagged
	if maxSeverity == "critical" || maxSeverity == "high" {
		verdict.Allowed = false
		verdict.Confidence = 0.95 // High confidence for validated violations
	} else if maxSeverity == "medium" || maxSeverity == "low" {
		verdict.Allowed = true
		verdict.Confidence = 0.90 // Medium confidence for soft violations
	} else {
		verdict.Allowed = true
		verdict.Confidence = 1.0 // Full confidence for no violations
	}

	// Build reasoning summary
	if len(verdict.MatchedPrinciples) > 0 {
		var reasons []string
		for _, m := range verdict.MatchedPrinciples {
			reasons = append(reasons, fmt.Sprintf("%s (%s)", m.Name, m.Severity))
		}
		verdict.Reasoning = fmt.Sprintf("Violations detected: %s", strings.Join(reasons, ", "))
	} else {
		verdict.Reasoning = "No principle violations detected"
	}

	return verdict, nil
}

// ToSafetyAlert adapts the verdict to the existing SafetyAlert shape
func (v *ConstitutionalVerdict) ToSafetyAlert() *models.SafetyAlert {
	if v.Allowed {
		return nil // No alert if allowed
	}

	alertType := "none"
	if v.OverallSeverity == "critical" || v.OverallSeverity == "high" {
		// Determine alert type based on principles
		for _, m := range v.MatchedPrinciples {
			if m.PrincipleID == "harm_prevention" {
				alertType = "crisis"
				break
			}
		}
		if alertType == "none" {
			alertType = "illegal"
		}
	}

	sev := "warning"
	if v.OverallSeverity == "critical" || v.OverallSeverity == "high" {
		sev = "high"
		if v.OverallSeverity == "critical" {
			sev = "immediate"
		}
	}

	alert := &models.SafetyAlert{
		AlertType:   alertType,
		Severity:    sev,
		Title:       "Content Review Required",
		Message:     v.Reasoning,
		Indicators:  []string{}, // Could populate from MatchedPrinciples if needed
		Resources:   []models.CrisisResource{},
		Recommendations: []string{"Please reconsider this message"},
	}

	return alert
}

// ToRiskAssessment adapts the verdict to the existing RiskAssessment shape
func (v *ConstitutionalVerdict) ToRiskAssessment() *models.RiskAssessment {
	riskLevel := "clear"
	recommendation := "proceed"

	if v.OverallSeverity == "critical" {
		riskLevel = "crisis"
		recommendation = "alert"
	} else if v.OverallSeverity == "high" {
		riskLevel = "elevated"
		recommendation = "caution"
	} else if v.OverallSeverity == "medium" {
		riskLevel = "elevated"
		recommendation = "caution"
	}

	return &models.RiskAssessment{
		RiskLevel:            riskLevel,
		Severity:             int(v.Confidence * 100),
		EducationalQuestions: []string{},
		Principles:           []models.CommunicationPrinciple{},
		Alternatives:         []string{},
		Recommendation:       recommendation,
		Message:              v.Reasoning,
	}
}
