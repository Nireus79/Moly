package tools

import (
	"context"
	"errors"
	"fmt"
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
	Message               string
	UserIntention         string
	ContactRelationship   string
	HistoricalContext     string
}

// ConstitutionEvaluatorOutput - Output from constitution evaluator
type ConstitutionEvaluatorOutput struct {
	Violations           []PrincipleViolation
	AlignedPrinciples    []string
	OverallRiskLevel     string // "critical", "high", "medium", "low", "clear"
	CriticalConcerns     []string
	Recommendations      []string
	IsConstitutional     bool
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
	llm        *LLMClient
	principles []Principle
}

// NewConstitutionEvaluator - Create new constitution evaluator
func NewConstitutionEvaluator(llm *LLMClient) *ConstitutionEvaluator {
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

	// Parse LLM response
	// TODO: Implement parsing logic when actual LLM integration is done
	if resp.Content != "" {
		output.EducationalOpportunity = resp.Content
	}

	return output, nil
}

// buildSystemPrompt - Build system prompt for evaluation
func (ce *ConstitutionEvaluator) buildSystemPrompt() string {
	return `You are an ethics expert evaluating communication against five core principles.

PRINCIPLES:
1. AUTHENTICITY - Be genuine and honest, not strategic or manipulative
2. RESPECT - Honor boundaries, preferences, and the person's autonomy
3. GROWTH - Support the person's development, not create dependency
4. TRUST - Be truthful, reliable, and follow through on commitments
5. AUTONOMY - Respect their choices and right to decide

For each principle, determine:
- ALIGNED: The message supports this principle
- NEUTRAL: The message is neutral to this principle
- VIOLATED: The message violates this principle
- CRITICAL: The message severely violates this principle

Respond with analysis and recommendations.
Remember: We educate, never block. Our goal is growth and learning.`
}

// buildUserPrompt - Build user prompt for evaluation
func (ce *ConstitutionEvaluator) buildUserPrompt(input *ConstitutionEvaluatorInput) string {
	context := ""
	if input.HistoricalContext != "" {
		context = fmt.Sprintf("\nContext: %s", input.HistoricalContext)
	}

	intention := ""
	if input.UserIntention != "" {
		intention = fmt.Sprintf("\nIntention: %s", input.UserIntention)
	}

	return fmt.Sprintf(`Evaluate this message:
"%s"

Relationship: %s%s%s

Analyze against the five principles.
Identify any violations or concerns.
Suggest how to align with our principles.`,
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
