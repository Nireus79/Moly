package tools

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ContextExtractorInput - Input for context extraction
type ContextExtractorInput struct {
	Message                    string
	ConversationHistory        []string
	ExistingCharacteristics    []string
	ExistingInterests          []string
	ExistingCommunicationPrefs string
}

// ContextExtractorOutput - Output from context extraction
type ContextExtractorOutput struct {
	NewCharacteristics        []string
	NewInterests              []string
	UpdatedCommunicationPrefs string
	UserQuotes                []string
	Intentions                []string
	RelationshipPhase         string
	Confidence                float64 // 0-1
	ExtractedAt               int64
}

// InsightType - Type of extracted insight
type InsightType string

const (
	InsightTypeCharacteristic InsightType = "characteristic"
	InsightTypeInterest       InsightType = "interest"
	InsightTypePref           InsightType = "preference"
	InsightTypeIntention      InsightType = "intention"
	InsightTypePhase          InsightType = "phase"
)

// ContextExtractor - Extracts insights from conversations
type ContextExtractor struct {
	llm LLMProvider
}

// NewContextExtractor - Create new context extractor
func NewContextExtractor(llm LLMProvider) *ContextExtractor {
	return &ContextExtractor{
		llm: llm,
	}
}

// Extract - Extract insights from conversation
func (ce *ContextExtractor) Extract(ctx context.Context, input *ContextExtractorInput) (*ContextExtractorOutput, error) {
	if input == nil {
		return nil, errors.New("input cannot be nil")
	}

	if input.Message == "" {
		return nil, errors.New("message cannot be empty")
	}

	output := &ContextExtractorOutput{
		NewCharacteristics:        []string{},
		NewInterests:              []string{},
		UpdatedCommunicationPrefs: input.ExistingCommunicationPrefs,
		UserQuotes:                []string{},
		Intentions:                []string{},
		Confidence:                0.0,
	}

	// Use LLM for nuanced extraction if available
	if ce.llm != nil {
		systemPrompt := ce.buildSystemPrompt()
		userPrompt := ce.buildUserPrompt(input)

		req := &LLMRequest{
			SystemPrompt:        systemPrompt,
			UserPrompt:          userPrompt,
			MaxTokens:           1000,
			Temperature:         0.5,
			UseExtendedThinking: true,
			Retries:             2,
		}

		resp, err := ce.llm.Call(ctx, req)
		if err == nil && resp.Content != "" {
			parseContextResponse(resp.Content, output)
		}
	} else {
		// Fallback to direct extraction
		directOutput := ce.ExtractDirectly(input)
		output.NewCharacteristics = directOutput.NewCharacteristics
		output.NewInterests = directOutput.NewInterests
		output.UserQuotes = directOutput.UserQuotes
	}

	// Calculate confidence based on data extracted
	if len(output.NewCharacteristics) > 0 && len(output.NewInterests) > 0 {
		output.Confidence = 0.85
	} else if len(output.NewCharacteristics) > 0 || len(output.NewInterests) > 0 {
		output.Confidence = 0.7
	} else {
		output.Confidence = 0.5
	}

	return output, nil
}

// buildSystemPrompt - Build system prompt for extraction
func (ce *ContextExtractor) buildSystemPrompt() string {
	return `You are an expert at extracting insights from conversation.
Your job is to identify what the user reveals about the contact they're talking to.

IMPORTANT RULES:
1. Only extract what was explicitly stated
2. Don't infer or guess beyond the words
3. Keep the user's exact language where possible
4. Look for:
   - Characteristics (personality traits, behaviors)
   - Interests (things they're passionate about)
   - Communication preferences (how they like to be talked to)
   - User's intentions (what they want to achieve)
   - Relationship phase (beginning, deepening, etc.)

5. Flag what you're confident about vs. uncertain
6. Always include relevant user quotes

Format your response clearly with sections for each category.`
}

// buildUserPrompt - Build user prompt for extraction
func (ce *ContextExtractor) buildUserPrompt(input *ContextExtractorInput) string {
	history := ""
	if len(input.ConversationHistory) > 0 {
		history = "\nPrevious exchanges:\n"
		for i, msg := range input.ConversationHistory {
			if i >= 3 {
				break
			}
			history += fmt.Sprintf("- %s\n", msg)
		}
	}

	existing := ""
	if len(input.ExistingCharacteristics) > 0 {
		existing = fmt.Sprintf("\nAlready known: %v\n", input.ExistingCharacteristics)
	}

	return fmt.Sprintf(`Extract insights from this message:
"%s"%s%s
What does this reveal about who they are and what they want?
Which quotes best capture these insights?
How confident are you in these insights?`,
		input.Message, history, existing)
}

// ExtractDirectly - Direct extraction without LLM (fast path)
func (ce *ContextExtractor) ExtractDirectly(input *ContextExtractorInput) *ContextExtractorOutput {
	output := &ContextExtractorOutput{
		NewCharacteristics:        []string{},
		NewInterests:              []string{},
		UpdatedCommunicationPrefs: input.ExistingCommunicationPrefs,
		UserQuotes:                []string{},
		Intentions:                []string{},
		RelationshipPhase:         "developing",
		Confidence:                0.5,
	}

	// Simple keyword-based extraction
	if len(input.Message) > 10 {
		output.UserQuotes = []string{input.Message}
	}

	return output
}

// parseContextResponse - Parse LLM response into structured context
func parseContextResponse(content string, output *ContextExtractorOutput) {
	sections := strings.Split(content, "\n\n")

	for _, section := range sections {
		section = strings.TrimSpace(section)
		if section == "" {
			continue
		}

		lowerSection := strings.ToLower(section)

		if strings.Contains(lowerSection, "characteristic") {
			parseCharacteristics(section, output)
		} else if strings.Contains(lowerSection, "interest") {
			parseInterests(section, output)
		} else if strings.Contains(lowerSection, "communication") || strings.Contains(lowerSection, "preference") {
			parsePreferences(section, output)
		} else if strings.Contains(lowerSection, "intention") || strings.Contains(lowerSection, "goal") {
			parseIntentions(section, output)
		} else if strings.Contains(lowerSection, "quote") {
			parseQuotes(section, output)
		}
	}
}

func parseCharacteristics(section string, output *ContextExtractorOutput) {
	re := regexp.MustCompile(`[-*]\s*(.+?)(?:\n|$)`)
	matches := re.FindAllStringSubmatch(section, -1)
	for _, match := range matches {
		if len(match) > 1 {
			char := strings.TrimSpace(match[1])
			if char != "" && !stringInSlice(char, output.NewCharacteristics) {
				output.NewCharacteristics = append(output.NewCharacteristics, char)
			}
		}
	}
}

func parseInterests(section string, output *ContextExtractorOutput) {
	re := regexp.MustCompile(`[-*]\s*(.+?)(?:\n|$)`)
	matches := re.FindAllStringSubmatch(section, -1)
	for _, match := range matches {
		if len(match) > 1 {
			interest := strings.TrimSpace(match[1])
			if interest != "" && !stringInSlice(interest, output.NewInterests) {
				output.NewInterests = append(output.NewInterests, interest)
			}
		}
	}
}

func parsePreferences(section string, output *ContextExtractorOutput) {
	lines := strings.Split(section, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "Communication") {
			output.UpdatedCommunicationPrefs = line
			break
		}
	}
}

func parseIntentions(section string, output *ContextExtractorOutput) {
	re := regexp.MustCompile(`[-*]\s*(.+?)(?:\n|$)`)
	matches := re.FindAllStringSubmatch(section, -1)
	for _, match := range matches {
		if len(match) > 1 {
			intent := strings.TrimSpace(match[1])
			if intent != "" && !stringInSlice(intent, output.Intentions) {
				output.Intentions = append(output.Intentions, intent)
			}
		}
	}
}

func parseQuotes(section string, output *ContextExtractorOutput) {
	re := regexp.MustCompile(`"(.+?)"`)
	matches := re.FindAllStringSubmatch(section, -1)
	for _, match := range matches {
		if len(match) > 1 {
			quote := strings.TrimSpace(match[1])
			if quote != "" && !stringInSlice(quote, output.UserQuotes) {
				output.UserQuotes = append(output.UserQuotes, quote)
			}
		}
	}
}

// stringInSlice - Helper to check if string is in slice
func stringInSlice(s string, slice []string) bool {
	for _, item := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}
