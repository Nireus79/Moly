package tools

import (
	"context"
	"fmt"
	"log"
	"strings"

	"moly/models"
)

// ResponseGenerator creates natural, contextual LLM-generated responses
type ResponseGenerator struct {
	llmClient         LLMProvider
	responseValidator interface{} // Response validation (set via SetResponseValidator)
}

// NewResponseGenerator creates a new generator
func NewResponseGenerator(llm LLMProvider) *ResponseGenerator {
	return &ResponseGenerator{llmClient: llm}
}

// callLLM is a helper to make LLM calls with consistent formatting
func (rg *ResponseGenerator) callLLM(systemPrompt, userPrompt string) (string, error) {
	if rg.llmClient == nil {
		return "", fmt.Errorf("no LLM client available")
	}

	req := &LLMRequest{
		Role:         RoleReply,
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Temperature:  0.7,
	}

	resp, err := rg.llmClient.Call(context.Background(), req)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(resp.Content), nil
}

// StatesOnlyWhatIsStated is added to the instructions of the models that write replies and questions. They get the conversation's facts
// (summary, goal, people with what they are) and must stay inside them: the premise check that used to read every reply afterwards
// asked the same thing of the same facts, one to four calls later (docs/NEXT_VERSION.md, D9 step A2).
const StatesOnlyWhatIsStated = " Use only what the conversation states: do not say or hint that an event happened, that someone feels something, or that a person is something to anyone, unless the user or the list of people says so; when you need to know, ask. A person belongs to the user only as stated: a relation between two other people is not a relation to the user. You speak to the user: the people they mention are third persons, so never use a person's name to address the user (no \"Name, ...\" opening); use a name only to speak about that person."

// InUsersLanguage is added next to StatesOnlyWhatIsStated: every text a model writes for the user is in the language of the user's own words.
// Found live (2026-10-30): the help reply to a short Greek message came back in English, because the main reply writer was never told.
const InUsersLanguage = " Write in the language of the user's own words (never that of a name, a number or a quoted word), and follow them if they change language. In a language with a formal and an informal \"you\", use the form the user uses for themselves and for you; when their words do not show it, use the informal singular, the way a friend speaks, and do not change to the formal or the plural form later in the conversation."

// GenerateEmptyMessageResponse generates a response for empty user input
func (rg *ResponseGenerator) GenerateEmptyMessageResponse(ctx models.Context) string {
	if rg.llmClient == nil {
		return "What's on your mind?"
	}

	systemPrompt := "You are Moly, a communication coach. Generate a brief, warm prompt asking the user to share. One sentence only. Be conversational, not robotic."
	userPrompt := buildEmptyMessagePrompt(ctx)

	response, err := rg.callLLM(systemPrompt, userPrompt)
	if err != nil {
		log.Printf("[ResponseGenerator] Warning: Failed to generate empty message response: %v", err)
		return "What's on your mind?"
	}

	return response
}

// GenerateGapClarificationResponse generates targeted clarification questions for specific identified gaps
//
// dialogue is the conversation as the models read it (the people with what they are and of whom, the memory, the last messages),
// and userMessage the message just received: a question written without them takes people for the user's own (found live,
// 2026-10-14: "your father" for someone else's father).
func (rg *ResponseGenerator) GenerateGapClarificationResponse(ctx models.Context, gaps []string, dialogue, userMessage string) string {
	// No canned greeting is put in front of the question: a gap question opens, at most, with the model's own sentence of
	// understanding (found in a dynamic run: "Hi! I'd love to help you with that." was followed by "Thanks for telling me that.").
	greeting := ""

	// Validate gaps array before use
	if rg.llmClient == nil || len(gaps) == 0 {
		return greeting + "I'd like to understand you better."
	}

	// Sanitize gaps before passing to LLM (prevent echoing full message)
	validGaps := make([]string, 0)
	for _, gap := range gaps {
		if gap != "" && len(gap) < 200 { // Skip empty or excessively long gaps
			validGaps = append(validGaps, gap)
		}
	}

	if len(validGaps) == 0 {
		log.Printf("[ResponseGenerator] FIX #29: All gaps invalid or empty, using fallback")
		return greeting + "I'd like to understand you better."
	}

	systemPrompt := `You are Moly, a communication coach. You've identified some missing context to better understand the user's situation.
Generate ONE natural, warm clarifying question about the most important missing piece.
Ask for exactly one thing: never join two requests in one sentence ("when ... and what ..."). One to two sentences. Be conversational and specific.` + StatesOnlyWhatIsStated + InUsersLanguage

	userPrompt := buildGapClarificationPrompt(ctx, validGaps, dialogue, userMessage)

	response, err := rg.callLLM(systemPrompt, userPrompt)
	if err != nil {
		log.Printf("[ResponseGenerator] Warning: Failed to generate gap clarification response: %v", err)
		// Safely access first gap with length check
		if len(validGaps) > 0 {
			return greeting + gapToDefaultQuestion(validGaps[0])
		}
		return greeting + "I'd like to understand you better."
	}

	return greeting + response
}

// GenerateIntentClarificationResponse generates a question when user's intent is unclear
func (rg *ResponseGenerator) GenerateIntentClarificationResponse(ctx models.Context, userMessage, dialogue string) string {
	if rg.llmClient == nil {
		return "I want to make sure I understand what you're looking for. What's most important to you right now?"
	}

	systemPrompt := `You are Moly, a communication coach. The user's message is a bit unclear about what they're actually trying to figure out.
Generate ONE natural, warm clarifying question to understand their actual intent or goal.
One to two sentences. Ask what they're really looking for or what matters most.` + StatesOnlyWhatIsStated + InUsersLanguage

	userPrompt := buildIntentClarificationPrompt(ctx, userMessage, dialogue)

	response, err := rg.callLLM(systemPrompt, userPrompt)
	if err != nil {
		log.Printf("[ResponseGenerator] Warning: Failed to generate intent clarification: %v", err)
		return "What's most important to you in this situation?"
	}

	return response
}

// Helper functions to build prompts

func buildEmptyMessagePrompt(ctx models.Context) string {
	return `The user sent an empty message.
Generate a brief, warm prompt asking them to share what's on their mind.
Be conversational and inviting.

Context:
- Communication style: ` + ctx.AboutMe.CommunicationStyle + `
- Message history: ` + fmt.Sprintf("%d messages", len(ctx.ConversationHistory)) + `

Generate ONLY the prompt, nothing else.`
}

// gapToQuestion maps identified context gaps to natural clarifying questions
func gapToQuestion(gap string) string {
	gapQuestions := map[string]string{
		"communicationStyle":    "How do you usually communicate when something's important? Are you more direct and to-the-point, or do you prefer taking time to explain?",
		"coreValues":            "What matters most to you in a situation like this? What's really important here for you?",
		"contact":               "Tell me more about this person — who are they to you, and what's your relationship like?",
		"relevantReflections":   "Have you been in a situation like this before? What happened then, and how did it turn out?",
		"pastIntention":         "What are you really trying to figure out here? What would a good resolution look like for you?",
		"recentSafetyIncidents": "I want to make sure you're okay. Can you tell me more about what you're dealing with?",
	}

	if q, ok := gapQuestions[gap]; ok {
		return q
	}
	return "Tell me more about this situation."
}

// gapToDefaultQuestion returns a safe fallback question for a gap
func gapToDefaultQuestion(gap string) string {
	return gapToQuestion(gap)
}

// buildGapClarificationPrompt builds a prompt that targets specific identified gaps
func buildGapClarificationPrompt(ctx models.Context, gaps []string, dialogue, userMessage string) string {
	if len(gaps) == 0 {
		return "Generate a question asking the user to tell you more about their situation."
	}

	// Prioritize which gap to ask about (some are more important than others)
	primaryGap := gaps[0]
	for _, gap := range gaps {
		// Prioritize certain gaps
		if gap == "pastIntention" || gap == "contact" {
			primaryGap = gap
			break
		}
	}

	gapDescription := gapToDescription(primaryGap)
	relatedContext := buildContextSummary(ctx)
	lastUserMessage := userMessage // the message just received (it used to be the first message of the history)
	if lastUserMessage == "" && len(ctx.ConversationHistory) > 0 {
		lastUserMessage = ctx.ConversationHistory[0].Content
	}

	return dialogue + fmt.Sprintf(`The user just said: "%s"

I've understood some parts of their situation, but I'm missing important context about: %s

Their current context:
%s

You are Moly, speaking TO the user. Never write as the user and never claim their thanks, feelings or situation as your own: address them as "you".
Use only what the conversation above states. A person belongs to the user ("your ...") only if the conversation says so; a relation to another person is not a relation to the user. Do not assume events, facts or roles that were not stated.
Generate a natural, warm clarifying question about what's missing. Ask about %s specifically.
Make it conversational and reference what they've already told you. If they gave several things at once, you may open with one short sentence saying what you understood, then ask only about this one thing; the rest can wait.
Ask exactly ONE question.

Generate ONLY the reply, nothing else.`,
		lastUserMessage,
		gapDescription,
		relatedContext,
		gapDescription,
	)
}

// gapToDescription provides human-readable descriptions of gaps
func gapToDescription(gap string) string {
	descriptions := map[string]string{
		// Legacy gap types
		"communicationStyle":    "their communication style and preferences",
		"coreValues":            "what really matters to them",
		"contact":               "who they're talking about and their relationship",
		"relevantReflections":   "whether they've experienced something similar",
		"pastIntention":         "what they're ultimately trying to figure out",
		"recentSafetyIncidents": "their safety and wellbeing",

		// NEW: Extracted data gap types (message-specific, higher priority)
		"extracted_preference_needs_context":     "how their stated preference applies to this specific situation",
		"extracted_characteristic_needs_context": "how their characteristic or experience informs their approach here",
		"extracted_negation_needs_clarification": "what they would prefer instead of what they've ruled out",
	}

	if desc, ok := descriptions[gap]; ok {
		return desc
	}
	// Not a legacy gap code: the gap is already a description written by the gap detector. Use it as written.
	if strings.TrimSpace(gap) != "" {
		return gap
	}
	return "more details"
}

// buildIntentClarificationPrompt builds a prompt for clarifying unclear intent
func buildIntentClarificationPrompt(ctx models.Context, userMessage, dialogue string) string {
	return dialogue + fmt.Sprintf(`The user just said: "%s"

I'm not quite sure what they're really trying to figure out or what matters most to them.

Generate a natural question that asks them to clarify their actual goal or intent.
Make it warm and conversational, not interrogative.

Generate ONLY the question, nothing else.`,
		userMessage,
	)
}

// buildContextSummary creates a brief summary of what we already know
func buildContextSummary(ctx models.Context) string {
	var summary []string

	if ctx.AboutMe != nil {
		if ctx.AboutMe.CommunicationStyle != "" {
			summary = append(summary, fmt.Sprintf("- Communication style: %s", ctx.AboutMe.CommunicationStyle))
		}
		if len(ctx.AboutMe.Values) > 0 {
			summary = append(summary, fmt.Sprintf("- Values: %s", strings.Join(ctx.AboutMe.Values, ", ")))
		}
	}

	if ctx.ContactProfile != nil && ctx.ContactProfile.Name != "" {
		// what the person is comes from the relations (whose father, whose boss); the bare kind ("family") reads as the user's own
		what := ctx.ContactProfile.RelationsText()
		if what == "" {
			what = ctx.ContactProfile.Relationship
		}
		summary = append(summary, fmt.Sprintf("- Talking about: %s (%s)", ctx.ContactProfile.Name, what))
	}

	if ctx.PastIntention != "" {
		summary = append(summary, fmt.Sprintf("- Goal: %s", ctx.PastIntention))
	}

	if len(summary) == 0 {
		return "- (just starting to understand their situation)"
	}

	return strings.Join(summary, "\n")
}
