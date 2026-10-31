package agents

import (
	"context"
	"fmt"
	"log"
	"strings"

	"moly/tools"
)

// confirmQuestion asks the user to confirm a fact Moly understood with doubt and did not save. The model writes it
// under the one-question contract: the reply model, then the fast model; the fixed English question only when both fail.
func (ca *conversationAgent) confirmQuestion(userMessage, fact, block string) string {
	fallback := fmt.Sprintf("Just to be sure, did you mean %s?", fact)
	if ca.llmClient == nil {
		return fallback
	}
	for _, role := range []string{tools.RoleReply, tools.RoleFast} {
		req := &tools.LLMRequest{
			Role:         role,
			SystemPrompt: "You write one confirming question for Moly, a communication thinking partner." + readsNoPages + tools.StatesOnlyWhatIsStated + tools.InUsersLanguage,
			UserPrompt: block + "The user said: \"" + userMessage + "\"\n\nMoly understood this, but is not sure of it: " + fact + "\n\n" +
				"Write one short, warm question, in the language the user writes in (the language of their own words, never that of a name, a number or a quoted word), that asks the user to confirm or correct exactly that. " +
				"Use the people and the conversation above: name a person as the user named them, give nobody a role the conversation does not state, and do not ask about people the conversation already settled. " +
				"The whole reply holds exactly one question mark. Do not give advice. Output only the question.",
			MaxTokens:   80,
			Temperature: 0.3,
		}
		resp, err := ca.llmClient.Call(context.Background(), req)
		if err != nil {
			log.Printf("[ConversationAgent] Confirm question generation failed (%s): %v", role, err)
			continue
		}
		out := strings.Trim(strings.TrimSpace(resp.Content), "\"“”")
		if meetsQuestionContract(out) {
			return out
		}
		log.Printf("[ConversationAgent] Confirm question broke the contract (%s: %d question marks, %d words)", role, countQuestionMarks(out), len(strings.Fields(out)))
	}
	return fallback
}
