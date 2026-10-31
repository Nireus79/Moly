package agents

import "strings"

// fallbackBlockedReplyQuestion is the last resort: used only when the model failed twice (reply model, then fast model). The draft that was
// flagged cannot be sent, and failing safe matters more than the language here.
const fallbackBlockedReplyQuestion = "What would you like to focus on first?"

// replacementForBlockedReply is what the user sees when the output gate flags Moly's own draft. The user's message
// already passed the safety stage, so the gate is a second opinion on the draft, not a verdict on the user: the
// draft is dropped and Moly asks one question instead of refusing (a refusal belongs to the safety stage only).
// The question is written by the model in the language of the user's words (writeQuestionFromAim).
func (ca *conversationAgent) replacementForBlockedReply(userMessage string) string {
	q := ca.writeQuestionFromAim(userMessage, "", "ask what the user would like to work on or say first; it may begin with one short sentence of what you understood if they mentioned several things; do not give advice and do not refuse")
	if strings.TrimSpace(q) == "" {
		return fallbackBlockedReplyQuestion
	}
	return q
}
