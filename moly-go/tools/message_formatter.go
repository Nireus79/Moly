package tools

import (
	"fmt"
	"strings"

	"moly/models"
)

// MessageFormatter - Formats and structures messages
type MessageFormatter struct{}

// NewMessageFormatter - Create new message formatter
func NewMessageFormatter() *MessageFormatter {
	return &MessageFormatter{}
}

// FormatSuggestion - Format suggestion for display
func (mf *MessageFormatter) FormatSuggestion(suggestion *models.Suggestion) string {
	if suggestion == nil {
		return ""
	}

	formatted := fmt.Sprintf("**Option %d**: %s\n", suggestion.Index+1, suggestion.Text)
	if suggestion.Reasoning != "" {
		formatted += fmt.Sprintf("*Why*: %s\n", suggestion.Reasoning)
	}
	if suggestion.Confidence > 0 {
		formatted += fmt.Sprintf("*Confidence*: %.0f%%\n", suggestion.Confidence*100)
	}

	return formatted
}

// FormatQuestion - Format question for display
func (mf *MessageFormatter) FormatQuestion(question string) string {
	if question == "" {
		return ""
	}
	return fmt.Sprintf("> %s", question)
}

// FormatContext - Format context for display
func (mf *MessageFormatter) FormatContext(ctx *models.Context) string {
	if ctx == nil {
		return "No context available"
	}

	var parts []string

	if ctx.AboutMe != nil && ctx.AboutMe.CommunicationStyle != "" {
		parts = append(parts, fmt.Sprintf("**Your style**: %s", ctx.AboutMe.CommunicationStyle))
	}

	if ctx.ContactProfile != nil && ctx.ContactProfile.Name != "" {
		parts = append(parts, fmt.Sprintf("**Talking to**: %s (%s)", ctx.ContactProfile.Name, ctx.ContactProfile.Relationship))
	}

	if len(ctx.ConversationHistory) > 0 {
		parts = append(parts, fmt.Sprintf("**History**: %d messages", len(ctx.ConversationHistory)))
	}

	if parts == nil {
		return "Context being gathered..."
	}

	return strings.Join(parts, " • ")
}

// FormatReflection - Format reflection for display
func (mf *MessageFormatter) FormatReflection(reflection *models.Reflection) string {
	if reflection == nil {
		return ""
	}

	formatted := "**What I noticed about you:**\n"

	if len(reflection.Characteristics) > 0 {
		formatted += fmt.Sprintf("- **Characteristics**: %s\n", strings.Join(reflection.Characteristics, ", "))
	}

	if len(reflection.Interests) > 0 {
		formatted += fmt.Sprintf("- **Interests**: %s\n", strings.Join(reflection.Interests, ", "))
	}

	if len(reflection.Intentions) > 0 {
		formatted += fmt.Sprintf("- **Intentions**: %s\n", strings.Join(reflection.Intentions, ", "))
	}

	if reflection.Status == "pending_approval" {
		formatted += "\n*Does this feel right? You can approve or edit.*"
	}

	return formatted
}

// FormatSafetyAlert - Format safety alert for display
func (mf *MessageFormatter) FormatSafetyAlert(alert *models.SafetyAlert) string {
	if alert == nil {
		return ""
	}

	formatted := fmt.Sprintf("⚠️ **%s** (%s)\n", alert.Title, alert.Severity)
	formatted += fmt.Sprintf("%s\n", alert.Message)

	if len(alert.Resources) > 0 {
		formatted += "\n**Resources that might help:**\n"
		for _, resource := range alert.Resources {
			formatted += fmt.Sprintf("- [%s](%s): %s\n", resource.Name, resource.URL, resource.Description)
		}
	}

	return formatted
}

// StructureResponse - Convert response to structured output
func (mf *MessageFormatter) StructureResponse(
	phase string,
	suggestions []models.Suggestion,
	questions []string,
	alert *models.SafetyAlert,
) map[string]interface{} {

	response := map[string]interface{}{
		"phase": phase,
	}

	if alert != nil {
		response["safety_alert"] = alert
	}

	if len(suggestions) > 0 {
		formattedSuggestions := make([]map[string]interface{}, len(suggestions))
		for i, s := range suggestions {
			formattedSuggestions[i] = map[string]interface{}{
				"index":      i,
				"text":       s.Text,
				"reasoning":  s.Reasoning,
				"confidence": s.Confidence,
				"tone":       s.Tone,
			}
		}
		response["suggestions"] = formattedSuggestions
	}

	if len(questions) > 0 {
		response["questions"] = questions
	}

	return response
}

// ExtractUserMessage - Extract and clean user message
func (mf *MessageFormatter) ExtractUserMessage(input string) string {
	if input == "" {
		return ""
	}

	// Clean up whitespace
	cleaned := strings.TrimSpace(input)

	// Remove multiple spaces
	cleaned = strings.Join(strings.Fields(cleaned), " ")

	return cleaned
}

// BreakIntoSentences - Break text into sentences
func (mf *MessageFormatter) BreakIntoSentences(text string) []string {
	if text == "" {
		return []string{}
	}

	sentences := strings.Split(text, ". ")
	for i, s := range sentences {
		sentences[i] = strings.TrimSpace(s)
		if !strings.HasSuffix(sentences[i], ".") && !strings.HasSuffix(sentences[i], "?") && !strings.HasSuffix(sentences[i], "!") {
			sentences[i] += "."
		}
	}

	return sentences
}

// HighlightKeywords - Highlight important keywords in text
func (mf *MessageFormatter) HighlightKeywords(text string, keywords []string) string {
	result := text

	for _, keyword := range keywords {
		placeholder := fmt.Sprintf("**%s**", keyword)
		result = strings.ReplaceAll(result, keyword, placeholder)
	}

	return result
}
