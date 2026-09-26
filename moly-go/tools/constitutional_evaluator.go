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
	Allowed             bool              // true if allowed based on maturity-aware decision logic
	OverallSeverity     string            // critical, high, medium, low, clear
	MatchedPrinciples   []PrincipleMatch  // all principles that were violated
	Reasoning           string            // summary of the evaluation
	EvaluatedText       string            // the text that was evaluated
	Confidence          float64           // 0.0-1.0
	LLMReasoning        string            // full LLM response for debugging
	ContextMaturity     float64           // 0.0-1.0 - context maturity level
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

// NO Tier 1a/1b - Everything is principle-based with LLM reasoning and validation

// Evaluate analyzes a message against constitutional principles
// Returns a verdict that indicates whether the message violates any principles
func (ce *ConstitutionalEvaluator) Evaluate(ctx context.Context, text string) (*ConstitutionalVerdict, error) {
	return ce.EvaluateWithContextAndMaturity(ctx, text, "", 1.0)
}

// EvaluateWithContext analyzes a message with conversation context
// prevMessage provides context about what this message is responding to
// DEPRECATED: Use EvaluateWithContextAndMaturity instead for proper context-maturity gating
func (ce *ConstitutionalEvaluator) EvaluateWithContext(ctx context.Context, text string, prevMessage string) (*ConstitutionalVerdict, error) {
	return ce.EvaluateWithContextAndMaturity(ctx, text, prevMessage, 1.0)
}

// EvaluateWithContextAndMaturity evaluates a message considering context maturity
// maturity (0.0-1.0): 0 = immature (new user, insufficient context), 1.0 = mature
// According to architecture:
// - maturity < 0.5: Ask clarification questions, then re-evaluate
// - maturity >= 0.5: Full principle evaluation with LLM
func (ce *ConstitutionalEvaluator) EvaluateWithContextAndMaturity(ctx context.Context, text string, prevMessage string, maturity float64) (*ConstitutionalVerdict, error) {
	if text == "" {
		return &ConstitutionalVerdict{
			Allowed:         true,
			OverallSeverity: "clear",
			Reasoning:       "Empty text",
			Confidence:      1.0,
			ContextMaturity: maturity,
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
	log.Printf("[ConstitutionalEvaluator] Evaluating message (%d chars), maturity=%.2f", len(text), maturity)

	// Always call LLM for principle-based evaluation
	// NO hardcoded phrases, keywords, or tiers - only principle-based reasoning

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

	log.Printf("[ConstitutionalEvaluator] ▶ Constitutional evaluation (LLM): temp=%.1f, max_tokens=%d",
		req.Temperature, req.MaxTokens)

	resp, err := ce.llm.Call(ctx, req)
	if err != nil {
		log.Printf("[ConstitutionalEvaluator] ✗ LLM evaluation failed: %v", err)
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
	verdict.ContextMaturity = maturity

	// Apply maturity-based decision logic
	// If immature: allow even if violations detected (will ask clarification instead)
	// If mature: apply normal blocking logic
	if maturity < 0.5 && (verdict.OverallSeverity == "high" || verdict.OverallSeverity == "critical") {
		log.Printf("[ConstitutionalEvaluator] Context immature (%.2f < 0.5): deferring principle enforcement, will ask clarifications", maturity)
		verdict.Allowed = true // Allow through for clarification phase
		verdict.Reasoning = fmt.Sprintf("Potential principle concern detected (%s), but context insufficient. Clarification questions will be asked.", verdict.OverallSeverity)
	}

	// Log result
	log.Printf("[ConstitutionalEvaluator] ✓ Evaluation complete: allowed=%v, severity=%s, maturity=%.2f, matches=%d",
		verdict.Allowed, verdict.OverallSeverity, maturity, len(verdict.MatchedPrinciples))

	for _, m := range verdict.MatchedPrinciples {
		log.Printf("  - %s (%s): %s", m.Name, m.Severity, m.Evidence)
	}

	return verdict, nil
}

// EvaluateWithAnalysisContext analyzes with rich conversation context
// analysisCtx provides: summary, recent messages, preferences, profile
// This is the primary method - provides maximum context for accurate evaluation
// DEPRECATED: Use EvaluateWithAnalysisContextAndMaturity for proper maturity gating
func (ce *ConstitutionalEvaluator) EvaluateWithAnalysisContext(ctx context.Context, analysisCtx *models.AnalysisContext) (*ConstitutionalVerdict, error) {
	return ce.EvaluateWithAnalysisContextAndMaturity(ctx, analysisCtx, 1.0)
}

// EvaluateWithAnalysisContextAndMaturity analyzes with rich conversation context and maturity consideration
// analysisCtx provides: summary, recent messages, preferences, profile
// maturity (0.0-1.0): used to gate whether violations block or trigger clarification
func (ce *ConstitutionalEvaluator) EvaluateWithAnalysisContextAndMaturity(ctx context.Context, analysisCtx *models.AnalysisContext, maturity float64) (*ConstitutionalVerdict, error) {
	if analysisCtx == nil || analysisCtx.CurrentMessage == "" {
		return nil, fmt.Errorf("analysisCtx with currentMessage is required")
	}

	if ce.llm == nil {
		log.Printf("[ConstitutionalEvaluator] ERROR: No LLM available")
		return nil, fmt.Errorf("LLM provider is nil")
	}

	if ce.constitution == nil {
		log.Printf("[ConstitutionalEvaluator] ERROR: No constitution loaded")
		return nil, fmt.Errorf("constitution is nil")
	}

	text := strings.TrimSpace(analysisCtx.CurrentMessage)
	log.Printf("[ConstitutionalEvaluator] Evaluating message (%d chars) with analysis context (quality: %s, maturity: %.2f)",
		len(text), analysisCtx.ContextQuality, maturity)

	// Always call LLM for principle-based evaluation
	// NO hardcoded phrases, keywords, or tiers - only principle-based reasoning with rich context

	// Build system prompt from constitution
	systemPrompt := ce.buildSystemPrompt()

	// Build user prompt WITH rich analysis context
	userPrompt := ce.buildUserPromptWithAnalysisContext(analysisCtx)

	// Call LLM
	req := &LLMRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		MaxTokens:    500,
		Temperature:  0.3,
		Retries:      2,
	}

	log.Printf("[ConstitutionalEvaluator] ▶ Constitutional evaluation (LLM with analysis context): temp=%.1f, max_tokens=%d",
		req.Temperature, req.MaxTokens)

	resp, err := ce.llm.Call(ctx, req)
	if err != nil {
		log.Printf("[ConstitutionalEvaluator] ✗ LLM evaluation failed: %v", err)
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
	verdict.ContextMaturity = maturity

	// Apply maturity-based decision logic
	// If immature: allow even if violations detected (will ask clarification instead)
	// If mature: apply normal blocking logic
	if maturity < 0.5 && (verdict.OverallSeverity == "high" || verdict.OverallSeverity == "critical") {
		log.Printf("[ConstitutionalEvaluator] Context immature (%.2f < 0.5): deferring principle enforcement, will ask clarifications", maturity)
		verdict.Allowed = true // Allow through for clarification phase
		verdict.Reasoning = fmt.Sprintf("Potential principle concern detected (%s), but context insufficient. Clarification questions will be asked.", verdict.OverallSeverity)
	}

	// Log result
	log.Printf("[ConstitutionalEvaluator] ✓ Evaluation complete: allowed=%v, severity=%s, maturity=%.2f, context_quality=%s, matches=%d",
		verdict.Allowed, verdict.OverallSeverity, maturity, analysisCtx.ContextQuality, len(verdict.MatchedPrinciples))

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
	sb.WriteString("- Consider what the user is actually ASKING FOR, not just keywords present\n")
	sb.WriteString("- Refinement requests (e.g., 'make it more playful', 'be friendlier') are NOT violations\n")
	sb.WriteString("- Innocent requests for tone/style adjustments are NOT principle violations\n")
	sb.WriteString("- Only flag messages that ACTUALLY VIOLATE a principle, not ones that mention related topics\n")
	sb.WriteString("- CRITICAL: Presence of a keyword ≠ violation. Example:\n")
	sb.WriteString("  ✗ WRONG: 'fetlife mentioned' = Harm Prevention violation\n")
	sb.WriteString("  ✓ RIGHT: 'asking help manipulating without consent' = User Autonomy violation\n")
	sb.WriteString("- Avoid false positives by understanding INTENT, not just PRESENCE\n")
	sb.WriteString("- If user is innocently sharing information, mentioning a platform, or asking for advice: NOT a violation\n")

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

// buildUserPromptWithAnalysisContext creates a prompt with rich conversation context
// Uses conversation summary, recent messages, preferences, and user profile
func (ce *ConstitutionalEvaluator) buildUserPromptWithAnalysisContext(analysisCtx *models.AnalysisContext) string {
	var prompt strings.Builder

	prompt.WriteString("Analyze this message for principle violations.\n\n")

	// Include conversation summary if available
	if analysisCtx.ConversationSummary != nil {
		prompt.WriteString("CONVERSATION CONTEXT:\n")
		prompt.WriteString(fmt.Sprintf("Arc: %s\n", analysisCtx.ConversationSummary.Arc))
		if len(analysisCtx.ConversationSummary.KeyTopics) > 0 {
			prompt.WriteString(fmt.Sprintf("Topics: %v\n", analysisCtx.ConversationSummary.KeyTopics))
		}
		if len(analysisCtx.ConversationSummary.UserPatterns) > 0 {
			prompt.WriteString(fmt.Sprintf("Patterns: %v\n", analysisCtx.ConversationSummary.UserPatterns))
		}
		prompt.WriteString("\n")
	}

	// Include recent messages for immediate context
	if len(analysisCtx.RecentMessages) > 0 {
		prompt.WriteString("RECENT EXCHANGE:\n")
		for _, msg := range analysisCtx.RecentMessages {
			role := "User"
			if msg.Role == "assistant" {
				role = "Moly"
			}
			prompt.WriteString(fmt.Sprintf("%s: %s\n", role, msg.Content))
		}
		prompt.WriteString("\n")
	}

	// Include user profile (communication style)
	if analysisCtx.UserProfile != nil && analysisCtx.UserProfile.CommunicationStyle != "" {
		prompt.WriteString(fmt.Sprintf("USER PROFILE: Communication style: %s\n", analysisCtx.UserProfile.CommunicationStyle))
		if analysisCtx.UserProfile.PreferredTone != "" {
			prompt.WriteString(fmt.Sprintf("Preferred tone: %s\n", analysisCtx.UserProfile.PreferredTone))
		}
		prompt.WriteString("\n")
	}

	// Current message to evaluate
	prompt.WriteString("CURRENT MESSAGE TO EVALUATE:\n")
	prompt.WriteString(fmt.Sprintf("\"%s\"\n\n", analysisCtx.CurrentMessage))

	// Add evaluation guidance (refined for context-aware evaluation)
	prompt.WriteString("EVALUATION GUIDANCE:\n")
	prompt.WriteString("- Evaluate this message IN FULL CONTEXT, not in isolation\n")
	prompt.WriteString("- Consider the conversation arc and user's established patterns\n")
	prompt.WriteString("- Refinement requests ('make it more playful', 'be friendlier') are NOT violations\n")
	prompt.WriteString("- Innocent requests for tone/style adjustments are NOT principle violations\n")
	prompt.WriteString("- ONLY flag messages that ACTUALLY violate principles\n")
	prompt.WriteString("- Avoid false positives by understanding context and intent\n")

	return prompt.String()
}

// validateAndParse validates and parses the LLM response
// CRITICAL FIX: Not just checking if evidence is a substring.
// Must verify that the evidence ACTUALLY constitutes a principle violation in context.
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
		// 1. Validate principle ID exists
		principle, exists := principleMap[v.PrincipleID]
		if !exists {
			log.Printf("[ConstitutionalEvaluator] ✗ Dropping unknown principle: %s", v.PrincipleID)
			continue
		}

		// 2. Validate evidence is actually in the message (substring check)
		if !strings.Contains(strings.ToLower(originalText), strings.ToLower(v.Evidence)) {
			log.Printf("[ConstitutionalEvaluator] ✗ Dropping unverified evidence: %q not in message", v.Evidence)
			continue
		}

		// 3. CRITICAL: Validate that evidence is CONTEXTUAL, not just PRESENT
		// A keyword existing ≠ a violation. We need to verify the violation is real.
		// Conservative approach: only accept violations that clearly violate the principle
		// Reject: single word evidence, common words, platform names
		if !ce.isContextualViolation(v.PrincipleID, v.Evidence, v.Reasoning, originalText) {
			log.Printf("[ConstitutionalEvaluator] ✗ Dropping keyword-only evidence: %q (not contextual violation)", v.Evidence)
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

	// Generate principle-specific, context-aware message (Layer 11)
	message := "I can't help with that."

	if len(v.MatchedPrinciples) > 0 {
		primaryPrinciple := v.MatchedPrinciples[0]
		evidence := strings.ToLower(v.MatchedPrinciples[0].Evidence)

		// Layer 11: Context-specific guidance based on principle
		switch primaryPrinciple.PrincipleID {
		case "harm_prevention":
			if strings.Contains(evidence, "myself") || strings.Contains(evidence, "self") ||
			   strings.Contains(evidence, "suicide") || strings.Contains(evidence, "kill myself") {
				// Self-harm: offer support resources
				message = "I can't help with that. If you're in crisis, please reach out to a mental health professional or contact a crisis helpline."
			} else {
				// Harm to others: firm refusal
				message = "I can't help with that. Deliberately harming someone goes against everything I stand for."
			}

		case "consent_and_respect":
			message = "I can't help with that. I don't support approaches that bypass someone's wishes or boundaries."

		case "stakeholder_consideration":
			message = "I can't help with that. I can't assist with plans that ignore how others might be affected."

		case "user_autonomy":
			message = "I can't help by pressuring you into a specific choice. This decision is yours to make."

		case "transparency":
			message = "I can't help with that. Honesty and transparency are important to me."

		default:
			message = "I can't help with that. This falls outside what I'm able to assist with."
		}
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

// isContextualViolation checks if evidence is a REAL violation, not just a keyword present
// Filters out false positives from keyword-matching:
// - Single words or platform names (fetlife, reddit, twitter, etc.)
// - Common words that appear in innocent contexts (girl, help, want, like, etc.)
// - Content mentions that aren't requesting harm
func (ce *ConstitutionalEvaluator) isContextualViolation(principleID, evidence, reasoning, fullText string) bool {
	evidence = strings.TrimSpace(evidence)
	lower := strings.ToLower(evidence)

	// Reject: single words (too vague, likely keyword-matching)
	if !strings.Contains(evidence, " ") && len(evidence) < 15 {
		// Exception: multi-word harmful phrases like "kill myself" are OK
		if !ce.isMultiwordHarmPhrase(evidence) {
			log.Printf("[ConstitutionalEvaluator] ✗ Rejecting single-word evidence: %q", evidence)
			return false
		}
	}

	// Reject: Platform names and site mentions
	platformNames := []string{"fetlife", "reddit", "twitter", "instagram", "facebook", "tiktok", "website", "site", "profile"}
	for _, platform := range platformNames {
		if lower == platform {
			log.Printf("[ConstitutionalEvaluator] ✗ Rejecting platform name as violation: %q", evidence)
			return false
		}
	}

	// Reject: Common context words that aren't violations
	contextWords := []string{"girl", "boy", "man", "woman", "person", "friend", "help", "want", "like", "need", "tell", "ask", "say"}
	if len(strings.Fields(evidence)) == 1 {
		for _, word := range contextWords {
			if lower == word {
				log.Printf("[ConstitutionalEvaluator] ✗ Rejecting common context word: %q", evidence)
				return false
			}
		}
	}

	// Accept: Evidence that shows clear action/intent violation
	// Examples: "I want to manipulate", "convince her to do", "without asking", "trick them"
	harmfulPatterns := []string{
		"manipulate", "trick", "coerce", "pressure", "force", "hurt", "harm",
		"without asking", "without consent", "kill", "suicide", "self-harm",
		"exploit", "abuse", "violate",
	}
	for _, pattern := range harmfulPatterns {
		if strings.Contains(lower, pattern) {
			log.Printf("[ConstitutionalEvaluator] ✓ Accepting action-based violation: %q contains %q", evidence, pattern)
			return true
		}
	}

	// For other evidence: require reasoning to explain WHY it's a violation
	// Don't accept violations without clear reasoning about the principle
	if reasoning == "" || len(reasoning) < 20 {
		log.Printf("[ConstitutionalEvaluator] ✗ Rejecting evidence without sufficient reasoning: %q", evidence)
		return false
	}

	// Evidence must directly relate to the principle, not just mention a topic
	// Example: mentioning "fetlife" isn't a Harm Prevention violation just because sexual content exists there
	if !ce.reasoningExplainsViolation(principleID, reasoning) {
		log.Printf("[ConstitutionalEvaluator] ✗ Rejecting: reasoning doesn't explain principle violation")
		return false
	}

	log.Printf("[ConstitutionalEvaluator] ✓ Accepting contextual violation: %q (%s)", evidence, reasoning)
	return true
}

// isMultiwordHarmPhrase checks if evidence is a known harmful multi-word phrase
func (ce *ConstitutionalEvaluator) isMultiwordHarmPhrase(evidence string) bool {
	harmPhrases := []string{
		"kill myself", "kill myself", "hurt myself", "harm myself",
		"kill someone", "hurt someone", "make a bomb", "build a weapon",
	}
	lower := strings.ToLower(evidence)
	for _, phrase := range harmPhrases {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

// reasoningExplainsViolation checks if the reasoning actually explains a principle violation
// Rejects: "mentions fetlife" as reasoning for Harm Prevention
// Accepts: "requesting user to manipulate without consent" for User Autonomy
func (ce *ConstitutionalEvaluator) reasoningExplainsViolation(principleID, reasoning string) bool {
	lower := strings.ToLower(reasoning)

	// Reasoning must mention ACTION or INTENT, not just PRESENCE
	actionWords := []string{"request", "asking", "want", "ask", "tell", "instruct", "require", "demand", "force", "pressure", "manipulate"}
	hasAction := false
	for _, action := range actionWords {
		if strings.Contains(lower, action) {
			hasAction = true
			break
		}
	}

	if !hasAction {
		log.Printf("[ConstitutionalEvaluator] ✗ Reasoning lacks action/intent: %s", reasoning)
		return false
	}

	// Reasoning must connect to the principle, not just describe the content
	principleKeywords := map[string][]string{
		"harm_prevention": {"harm", "hurt", "damage", "injury", "safety", "wellbeing", "risk"},
		"user_autonomy": {"pressure", "coerce", "force", "manipulate", "choice", "decision", "autonomy"},
		"transparency": {"honest", "deceiv", "truth", "transparent", "clear", "hiding", "withhold"},
		"consent_and_respect": {"consent", "agree", "permission", "respect", "boundaries", "ask", "without"},
		"empathy_and_respect": {"respect", "consider", "empathy", "feelings", "impact", "perspectives"},
	}

	keywords := principleKeywords[principleID]
	hasRelevance := false
	for _, keyword := range keywords {
		if strings.Contains(lower, keyword) {
			hasRelevance = true
			break
		}
	}

	if !hasRelevance {
		log.Printf("[ConstitutionalEvaluator] ✗ Reasoning doesn't connect to principle %s", principleID)
		return false
	}

	return true
}
