package agents

import (
	"context"
	"log"
	"strings"

	"moly/models"
	"moly/tools"
)

// A request for something illegal is refused (the product rule), but a person who is tempted by a shortcut or talks themselves into
// it is better served by one reflective question first (user decision, 2026-10-19: "some Socratic questioning would be good before
// refusal"). The model decides whether this is such a case; a request to hurt someone, a threat, or anything unclear is refused at
// once. The question is asked once per conversation: when the person goes on with the same request, the refusal follows. It is never
// skippable (the skip button would otherwise ask for the result), and it never helps with the act.

// refusalQuestionAsked reports whether Moly already put a reflective question in place of a refusal in this conversation.
func refusalQuestionAsked(history []models.Message) bool {
	for _, m := range history {
		if m.Role == "assistant" && m.Metadata != nil {
			if asked, _ := m.Metadata["refusalQuestion"].(bool); asked {
				return true
			}
		}
	}
	return false
}

// questionBeforeRefusal returns the reflective question to ask instead of the refusal, or "" when the refusal stands.
func (ca *conversationAgent) questionBeforeRefusal(ctx models.Context, userMessage string, alert *models.SafetyAlert) string {
	if alert == nil || alert.AlertType != "illegal" || ca.llmClient == nil || refusalQuestionAsked(ctx.ConversationHistory) {
		return ""
	}
	req := &tools.LLMRequest{
		Role:         tools.RoleReply,
		SystemPrompt: "You decide whether Moly, a communication thinking partner, may ask one reflective question before it refuses an illegal request, and you write that question." + readsNoPages,
		UserPrompt: "The user said: \"" + userMessage + "\"\n\nMoly will not help with this because it is illegal.\n\n" +
			"ask: true only when the person seems tempted by a shortcut, is talking themselves into it, or has not thought about the consequences for themselves " +
			"and for the people it affects, so that one question could make them think again. ask: false when the request is to hurt, threaten or stalk someone, " +
			"involves a child, is a plan that is already decided and detailed, or when you are not sure.\n" +
			"question: one short, calm question, in the language the user writes in, about the consequences or about the people it affects. " +
			"It must not help with the act in any way, must not describe how to do it or how to avoid being caught, and must not accuse the user.\n" +
			"Respond with ONLY JSON: {\"ask\": true|false, \"question\": \"...\"}",
		MaxTokens:   200,
		Temperature: 0.3,
	}
	resp, err := ca.llmClient.Call(context.Background(), req)
	if err != nil {
		log.Printf("[ConversationAgent] Question before refusal failed: %v (refusing)", err)
		return ""
	}
	var out struct {
		Ask      bool   `json:"ask"`
		Question string `json:"question"`
	}
	if err := tools.SafeJSONParse("RefusalQuestion", []byte(resp.Content), &out); err != nil || !out.Ask {
		return ""
	}
	q := strings.Trim(strings.TrimSpace(out.Question), "\"“”")
	if !meetsQuestionContract(q) {
		return ""
	}
	return q
}
