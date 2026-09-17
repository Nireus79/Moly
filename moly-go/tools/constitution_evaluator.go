package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Principle - Ethical principle
type Principle struct {
	ID          string
	Name        string
	Description string
	Severity    string // "critical", "high", "medium"
	Questions   []string
}

// ConstitutionEvaluatorInput - Input for constitution evaluation
type ConstitutionEvaluatorInput struct {
	Message             string
	UserIntention       string
	ContactRelationship string
	HistoricalContext   string
}

// ConstitutionEvaluatorOutput - Output from constitution evaluator
type ConstitutionEvaluatorOutput struct {
	Violations             []PrincipleViolation
	AlignedPrinciples      []string
	OverallRiskLevel       string // "critical", "high", "medium", "low", "clear"
	CriticalConcerns       []string
	Recommendations        []string
	IsConstitutional       bool
	EducationalOpportunity string
}

// PrincipleViolation - Violation of a principle
type PrincipleViolation struct {
	PrincipleID string
	Principle   string
	Severity    string
	Description string
	Reasoning   string
}

// ConstitutionEvaluator - Evaluates ethical principles
type ConstitutionEvaluator struct {
	llm        LLMProvider
	principles []Principle
}

// NewConstitutionEvaluator - Create new constitution evaluator
func NewConstitutionEvaluator(llm LLMProvider) *ConstitutionEvaluator {
	return &ConstitutionEvaluator{
		llm:        llm,
		principles: getDefaultPrinciples(),
	}
}

// Evaluate - Evaluate message against ethical principles
func (ce *ConstitutionEvaluator) Evaluate(ctx context.Context, input *ConstitutionEvaluatorInput) (*ConstitutionEvaluatorOutput, error) {
	if input == nil {
		return nil, errors.New("input cannot be nil")
	}

	if input.Message == "" {
		return nil, errors.New("message cannot be empty")
	}

	output := &ConstitutionEvaluatorOutput{
		Violations:        []PrincipleViolation{},
		AlignedPrinciples: []string{},
		OverallRiskLevel:  "low",
		IsConstitutional:  true,
	}

	// Use LLM for nuanced analysis
	systemPrompt := ce.buildSystemPrompt()
	userPrompt := ce.buildUserPrompt(input)

	req := &LLMRequest{
		SystemPrompt:        systemPrompt,
		UserPrompt:          userPrompt,
		MaxTokens:           1200,
		Temperature:         0.4,
		UseExtendedThinking: true,
		Retries:             2,
	}

	resp, err := ce.llm.Call(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("constitution evaluation failed: %w", err)
	}

	// Parse LLM response for principle violations and recommendations
	if resp.Content != "" {
		parseConstitutionResponse(resp.Content, output)
	}

	// Determine overall risk level and constitutional status
	if len(output.Violations) > 0 {
		output.IsConstitutional = false
		for _, v := range output.Violations {
			if v.Severity == "critical" {
				output.OverallRiskLevel = "critical"
				break
			} else if v.Severity == "high" && output.OverallRiskLevel != "critical" {
				output.OverallRiskLevel = "high"
			} else if v.Severity == "medium" && output.OverallRiskLevel == "low" {
				output.OverallRiskLevel = "medium"
			}
		}
	}

	return output, nil
}

// buildSystemPrompt - Build system prompt for evaluation
func (ce *ConstitutionEvaluator) buildSystemPrompt() string {
	return `You help people think through whether their communication aligns with their values.

FIVE CORE PRINCIPLES:
1. AUTHENTICITY - Being genuine and honest, not strategic
2. RESPECT - Honoring boundaries and choices
3. GROWTH - Supporting development, not creating dependency
4. TRUST - Being truthful and reliable
5. AUTONOMY - Respecting their right to decide

Your job: Identify if anything might misalign with these principles, then turn it into a reflective question.

For each principle:
- If ALIGNED: Note it (can affirm later)
- If MISALIGNED: Frame as question for reflection ("Does this feel authentic to you?" or "Does this respect their choice?")
- If SERIOUS CONCERN: Flag it but don't judge

Never say "I noticed" or "This violates." Always ask questions that help them think.
Return structured analysis with questions, not judgments.`
}

// buildUserPrompt - Build user prompt for evaluation
func (ce *ConstitutionEvaluator) buildUserPrompt(input *ConstitutionEvaluatorInput) string {
	context := ""
	if input.HistoricalContext != "" {
		context = fmt.Sprintf("\nContext: %s", input.HistoricalContext)
	}

	intention := ""
	if input.UserIntention != "" {
		intention = fmt.Sprintf("\nTheir intention: %s", input.UserIntention)
	}

	return fmt.Sprintf(`Does this align with their core values?
"%s"

Relationship: %s%s%s

Check each of the 5 principles. If something might misalign, frame it as a reflective question they could ask themselves.
Never judge. Only help them think.`,
		input.Message, input.ContactRelationship, context, intention)
}

// getDefaultPrinciples - Get default ethical principles
func getDefaultPrinciples() []Principle {
	return []Principle{
		{
			ID:          "authenticity",
			Name:        "Authenticity",
			Description: "Be genuine and honest, not strategic or manipulative",
			Severity:    "high",
			Questions: []string{
				"Am I being genuine in this message?",
				"Would I say this to their face?",
				"Am I hiding any important truth?",
			},
		},
		{
			ID:          "respect",
			Name:        "Respect",
			Description: "Honor boundaries, preferences, and the person's autonomy",
			Severity:    "high",
			Questions: []string{
				"Am I respecting their boundaries?",
				"Have they indicated what they want?",
				"Am I overstepping?",
			},
		},
		{
			ID:          "growth",
			Name:        "Growth",
			Description: "Support their development, not create dependency",
			Severity:    "medium",
			Questions: []string{
				"Am I helping them grow?",
				"Could this create unhealthy dependency?",
				"Am I respecting their ability to figure things out?",
			},
		},
		{
			ID:          "trust",
			Name:        "Trust",
			Description: "Be truthful, reliable, and follow through",
			Severity:    "high",
			Questions: []string{
				"Am I being truthful?",
				"Can I follow through on this?",
				"Am I being reliable?",
			},
		},
		{
			ID:          "autonomy",
			Name:        "Autonomy",
			Description: "Respect choices and the right to decide",
			Severity:    "critical",
			Questions: []string{
				"Am I respecting their right to choose?",
				"Am I trying to control the outcome?",
				"Have I left room for their decision?",
			},
		},
	}
}

// parseConstitutionResponse - Parse LLM response for violations and aligned principles
func parseConstitutionResponse(content string, output *ConstitutionEvaluatorOutput) {
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check for violations
		if strings.Contains(strings.ToUpper(line), "VIOLATED") || strings.Contains(strings.ToUpper(line), "CRITICAL") {
			violation := parseViolationLine(line)
			if violation != nil {
				output.Violations = append(output.Violations, *violation)
				if !contains(output.CriticalConcerns, violation.Description) {
					output.CriticalConcerns = append(output.CriticalConcerns, violation.Description)
				}
			}
		}

		// Check for aligned principles
		if strings.Contains(strings.ToUpper(line), "ALIGNED") {
			principle := extractPrinciple(line)
			if principle != "" && !contains(output.AlignedPrinciples, principle) {
				output.AlignedPrinciples = append(output.AlignedPrinciples, principle)
			}
		}

		// Check for recommendations
		if strings.HasPrefix(strings.ToUpper(line), "RECOMMEND") || strings.HasPrefix(strings.ToUpper(line), "SUGGESTION") {
			rec := extractRecommendation(line)
			if rec != "" && !contains(output.Recommendations, rec) {
				output.Recommendations = append(output.Recommendations, rec)
			}
		}
	}

	// If no violations found, set educational opportunity
	if len(output.Violations) == 0 && len(output.AlignedPrinciples) > 0 {
		output.EducationalOpportunity = fmt.Sprintf("Message aligns with principles: %s", strings.Join(output.AlignedPrinciples, ", "))
	}
}

func parseViolationLine(line string) *PrincipleViolation {
	parts := strings.Split(line, "-")
	if len(parts) < 2 {
		return nil
	}

	v := &PrincipleViolation{
		Severity:    "medium",
		Description: strings.TrimSpace(parts[len(parts)-1]),
	}

	for _, part := range parts {
		if strings.Contains(strings.ToUpper(part), "AUTHENTICITY") {
			v.Principle = "Authenticity"
		} else if strings.Contains(strings.ToUpper(part), "RESPECT") {
			v.Principle = "Respect"
		} else if strings.Contains(strings.ToUpper(part), "GROWTH") {
			v.Principle = "Growth"
		} else if strings.Contains(strings.ToUpper(part), "TRUST") {
			v.Principle = "Trust"
		} else if strings.Contains(strings.ToUpper(part), "AUTONOMY") {
			v.Principle = "Autonomy"
		}

		if strings.Contains(strings.ToUpper(part), "CRITICAL") {
			v.Severity = "critical"
		} else if strings.Contains(strings.ToUpper(part), "HIGH") {
			v.Severity = "high"
		}
	}

	return v
}

func extractPrinciple(line string) string {
	principles := []string{"Authenticity", "Respect", "Growth", "Trust", "Autonomy"}
	for _, p := range principles {
		if strings.Contains(line, p) {
			return p
		}
	}
	return ""
}

func extractRecommendation(line string) string {
	parts := strings.Split(line, ":")
	if len(parts) > 1 {
		return strings.TrimSpace(parts[len(parts)-1])
	}
	return strings.TrimSpace(line)
}

func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}
