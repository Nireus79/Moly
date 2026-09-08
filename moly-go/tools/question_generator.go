package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// QuestionType - Type of Socratic question
type QuestionType string

const (
	QuestionTypeSocratic      QuestionType = "socratic"
	QuestionTypeEducational   QuestionType = "educational"
	QuestionTypeContextGather QuestionType = "context_gathering"
)

// QuestionGeneratorInput - Input for question generation
type QuestionGeneratorInput struct {
	Type                   QuestionType
	UserMessage            string
	UserCommunicationStyle string
	ContactRelationship    string
	RiskContext            string
	MissingContext         []string
}

// QuestionGeneratorOutput - Output from question generator
type QuestionGeneratorOutput struct {
	Questions   []string
	Type        QuestionType
	Reasoning   string
	GeneratedAt int64
}

// QuestionGenerator - Generates contextual questions
type QuestionGenerator struct {
	llm LLMProvider
}

// NewQuestionGenerator - Create new question generator
func NewQuestionGenerator(llm LLMProvider) *QuestionGenerator {
	return &QuestionGenerator{
		llm: llm,
	}
}

// Generate - Generate contextual questions
func (qg *QuestionGenerator) Generate(ctx context.Context, input *QuestionGeneratorInput) (*QuestionGeneratorOutput, error) {
	if input == nil {
		return nil, errors.New("input cannot be nil")
	}

	if input.Type == "" {
		input.Type = QuestionTypeSocratic
	}

	systemPrompt := qg.buildSystemPrompt(input)
	userPrompt := qg.buildUserPrompt(input)

	useThinking := true
	if input.Type == QuestionTypeContextGather {
		useThinking = false
	}

	req := &LLMRequest{
		SystemPrompt:        systemPrompt,
		UserPrompt:          userPrompt,
		MaxTokens:           800,
		Temperature:         0.6,
		UseExtendedThinking: useThinking,
		Retries:             2,
	}

	resp, err := qg.llm.Call(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate questions: %w", err)
	}

	output := &QuestionGeneratorOutput{
		Type:      input.Type,
		Reasoning: "Questions generated to promote reflection and understanding",
		Questions: parseQuestions(resp.Content),
	}

	return output, nil
}

// buildSystemPrompt - Build system prompt for question generation
func (qg *QuestionGenerator) buildSystemPrompt(input *QuestionGeneratorInput) string {
	switch input.Type {
	case QuestionTypeSocratic:
		return `You are a thoughtful coach using Socratic questioning.
Your goal is to help users explore their own thoughts and values.

Guidelines:
- Ask open-ended questions that promote reflection
- Help users discover their own answers
- Show genuine curiosity about their perspective
- Avoid being preachy or judgmental
- Questions should feel natural and conversational
- Every question should be fresh, not templated

Generate 2-4 Socratic questions that help the user think deeper about their situation.`

	case QuestionTypeEducational:
		return `You are an ethics educator responding to a concerning pattern.
Your role is to help users understand the impact of their actions through questions.

Guidelines:
- Ask questions that help users consider consequences
- Help them think about the other person's perspective
- Guide reflection without lecturing
- Use empathy and understanding
- Questions should invite learning, not shame
- Every question should be specific to the situation

Generate 2-3 educational questions that help the user reflect on the concern.`

	case QuestionTypeContextGather:
		return `You are gathering context to better help the user.
Generate brief, natural questions that feel like conversation.

Guidelines:
- Ask what's missing to give better help
- Keep questions open but focused
- Questions should feel natural and not intrusive
- Every question should be fresh and specific

Generate 1-2 context-gathering questions.`

	default:
		return "You are a helpful assistant generating questions."
	}
}

// buildUserPrompt - Build user prompt for question generation
func (qg *QuestionGenerator) buildUserPrompt(input *QuestionGeneratorInput) string {
	switch input.Type {
	case QuestionTypeSocratic:
		msg := input.UserMessage
		if msg == "" {
			msg = "The user wants to think more deeply about their relationship and communication"
		}
		return fmt.Sprintf(`Help the user explore: "%s"

User's style: %s

Generate Socratic questions that help them reflect on their thoughts and values.`, msg, input.UserCommunicationStyle)

	case QuestionTypeEducational:
		return fmt.Sprintf(`The user has shown this pattern: "%s"
Relationship: %s

Generate educational questions that help them understand the concern.
Help them think about impact and consequences.`, input.RiskContext, input.ContactRelationship)

	case QuestionTypeContextGather:
		missing := ""
		if len(input.MissingContext) > 0 {
			missing = fmt.Sprintf("Missing context: %v. ", input.MissingContext)
		}
		return fmt.Sprintf(`%sAsk natural follow-up questions to better understand the user's situation.
Keep it conversational and brief.`, missing)

	default:
		return fmt.Sprintf("Context: %s", input.UserMessage)
	}
}

// parseQuestions - Parse LLM response into individual questions
func parseQuestions(content string) []string {
	if content == "" {
		return []string{}
	}

	var questions []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		line = strings.TrimPrefix(line, "- ")
		line = strings.TrimPrefix(line, "* ")
		line = strings.TrimPrefix(line, "1. ")
		line = strings.TrimPrefix(line, "2. ")
		line = strings.TrimPrefix(line, "3. ")
		line = strings.TrimPrefix(line, "4. ")

		if strings.HasSuffix(line, "?") && len(line) > 10 {
			questions = append(questions, line)
		}
	}

	return questions
}
