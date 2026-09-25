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

// Evaluate analyzes a message against constitutional principles
// Returns a verdict that indicates whether the message violates any principles
func (ce *ConstitutionalEvaluator) Evaluate(ctx context.Context, text string) (*ConstitutionalVerdict, error) {
	return ce.EvaluateWithContext(ctx, text, "")
}

// EvaluateWithContext analyzes a message with conversation context
// prevMessage provides context about what this message is responding to
func (ce *ConstitutionalEvaluator) EvaluateWithContext(ctx context.Context, text string, prevMessage string) (*ConstitutionalVerdict, error) {
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

	// Build user prompt WITH context
	userPrompt := ce.buildUserPromptWithContext(text, prevMessage)

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

	sb.WriteString("\n\nEVALUATION GUIDANCE:\n")
	sb.WriteString("==================\n")
	sb.WriteString("- Evaluate messages IN CONTEXT, not in isolation\n")
	sb.WriteString("- Consider what the user is actually asking for, not just the words\n")
	sb.WriteString("- Refinement requests (e.g., 'make it more playful', 'be friendlier') are NOT violations\n")
	sb.WriteString("- Innocent requests for tone/style adjustments are NOT principle violations\n")
	sb.WriteString("- Only flag messages that ACTUALLY violate a principle's core intent\n")
	sb.WriteString("- Avoid false positives by understanding context and intent\n")

	sb.WriteString("\n\nRESPONSE FORMAT:\n")
	sb.WriteString("================\n")
	sb.WriteString("Respond with ONLY a JSON object (no markdown, no explanation):\n")
	sb.WriteString(`{"violations": [{"principle_id": "id", "evidence": "exact quote from message", "reasoning": "why"}]}`)
	sb.WriteString("\n\n")
	sb.WriteString("GUIDELINES:\n")
	sb.WriteString("- Only include principles that are ACTUALLY violated by the message\n")
	sb.WriteString("- 'evidence' MUST be an exact substring from the message (quote the words used)\n")
	sb.WriteString("- If no principles are violated, return: {\"violations\": []}\n")
	sb.WriteString("- Be precise and specific, not overly strict\n")
	sb.WriteString("- Avoid false positives - only report real violations\n")

	return sb.String()
}

// buildUserPrompt creates a user prompt for evaluation (without context)
func (ce *ConstitutionalEvaluator) buildUserPrompt(text string) string {
	return fmt.Sprintf("Analyze this message for principle violations:\n\n\"%s\"", text)
}

// buildUserPromptWithContext creates a user prompt with conversation context
// This prevents false positives by evaluating messages in context, not in isolation
func (ce *ConstitutionalEvaluator) buildUserPromptWithContext(currentMessage, prevMessage string) string {
	var prompt strings.Builder

	prompt.WriteString("Analyze this message for principle violations.\n\n")

	// Include context if available
	if prevMessage != "" {
		prompt.WriteString("CONTEXT (previous message):\n")
		prompt.WriteString(fmt.Sprintf("\"%s\"\n\n", prevMessage))
		prompt.WriteString("CURRENT MESSAGE TO EVALUATE:\n")
	} else {
		prompt.WriteString("MESSAGE TO EVALUATE:\n")
	}

	prompt.WriteString(fmt.Sprintf("\"%s\"\n\n", currentMessage))

	// Add evaluation guidance
	prompt.WriteString("IMPORTANT:\n")
	prompt.WriteString("- Evaluate this message IN CONTEXT of what it's responding to\n")
	prompt.WriteString("- A refinement request (e.g., 'make it more playful') is NOT a violation by itself\n")
	prompt.WriteString("- Only flag actual principle violations, not innocent requests for adjustments\n")
	prompt.WriteString("- Consider: What is the user actually asking for?\n")

	return prompt.String()
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

	// Generate simple, clear message based on principle violated
	message := "I can't help with that."

	if len(v.MatchedPrinciples) > 0 {
		// Check evidence to distinguish self-harm from harming others
		evidence := strings.ToLower(v.MatchedPrinciples[0].Evidence)
		if strings.Contains(evidence, "myself") || strings.Contains(evidence, "self") ||
		   strings.Contains(evidence, "suicide") || strings.Contains(evidence, "kill myself") {
			// Self-harm: offer support
			message = "I can't help with that. If you're in crisis, please reach out to a mental health professional."
		} else if v.MatchedPrinciples[0].PrincipleID != "harm_prevention" {
			// Other violations
			message = message + " I have no knowledge on that matter."
		}
		// For harm_prevention (harming others): just the firm refusal
	}

	alert := &models.SafetyAlert{
		AlertType:   alertType,
		Severity:    sev,
		Title:       "I can't help with that",
		Message:     message,
		Indicators:  []string{},
		Resources:   []models.CrisisResource{},
		Recommendations: []string{},
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
