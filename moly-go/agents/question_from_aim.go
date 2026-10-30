package agents

import (
	"context"
	"log"
	"strings"

	"moly/tools"
)

// writeQuestionFromAim asks the model for one question that serves an aim, in the language of the user's own words. This is how every
// question that used to be a fixed English sentence is written: the code says what the question is for, the model says it
// (docs/NEXT_VERSION.md, "Method for P1 and P1b"). The ladder: the reply model; if its text breaks the one-question contract, the fast
// model once with the same instruction; if that fails too the result is "", and the caller moves on to another kind of reply. There is
// no fixed sentence at the end of it.
func (ca *conversationAgent) writeQuestionFromAim(userMessage, dialogue, aim string) string {
	if ca.llmClient == nil || strings.TrimSpace(aim) == "" {
		return ""
	}
	for _, role := range []string{tools.RoleReply, tools.RoleFast} {
		req := &tools.LLMRequest{
			Role:         role,
			SystemPrompt: "You write one question for Moly, a communication thinking partner." + readsNoPages + tools.StatesOnlyWhatIsStated + tools.InUsersLanguage,
			UserPrompt: dialogue + "The user said: \"" + userMessage + "\"\n\n" +
				"The question has this aim: " + aim + ".\n" +
				"Write one short, warm question, in the language the user writes in (the language of their own words, never that of a name, a number or a quoted word), " +
				"that serves the aim in THIS situation, using the user's own words for their goal. You may begin with one short sentence saying what you understood. " +
				"One question only; no advice, no list. Output only the reply.",
			MaxTokens:   120,
			Temperature: 0.4,
		}
		resp, err := ca.llmClient.Call(context.Background(), req)
		if err != nil {
			log.Printf("[ConversationAgent] Question from an aim failed (%s): %v", role, err)
			continue
		}
		out := strings.Trim(strings.TrimSpace(resp.Content), "\"“”")
		if meetsQuestionContract(out) {
			return out
		}
		log.Printf("[ConversationAgent] Question from an aim broke the contract (%s: %d question marks, %d words)", role, countQuestionMarks(out), len(strings.Fields(out)))
	}
	return ""
}
