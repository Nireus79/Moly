package agents

import (
	"context"
	"log"
	"strconv"

	"moly/models"
	"moly/tools"
)

// Gap questions are free text written by the model, so the same question comes back in other words: exact-text and type
// matching cannot catch it (found in a dynamic run: Moly asked whether the user had spoken to the fiancé, got a reply about
// feelings, and asked the same thing again; "thank her only, or raise what happened?" was asked twice in a row).
// DropRepeatedGaps asks the model which candidates repeat something Moly already asked or the user already said.

// DropRepeatedGaps returns the gaps that are new. alreadyAsked holds the question texts asked earlier in the conversation
// (answered or still open). Nothing is judged on the first message, when there is nothing to repeat. A judgement that fails
// or cannot be read keeps every gap: a repeated question is better than a silence about something that blocks the goal.
func DropRepeatedGaps(ctx context.Context, llm tools.LLMProvider, gaps []tools.Gap, dialogue *models.DialoguePacket, alreadyAsked []string) []tools.Gap {
	if llm == nil || len(gaps) == 0 {
		return gaps
	}
	block := dialogue.Text()
	if block == "" && len(alreadyAsked) == 0 {
		return gaps
	}
	if dialogue != nil && dialogue.MolyLast == "" && len(dialogue.Open) == 0 && len(alreadyAsked) == 0 {
		return gaps
	}
	list := ""
	for i, g := range gaps {
		list += strconv.Itoa(i+1) + ". " + g.Description + "\n"
	}
	asked := ""
	if len(alreadyAsked) > 0 {
		asked = "Questions Moly asked earlier in this conversation (any of them counts, not only the last):\n"
		for i, q := range alreadyAsked {
			asked += "Q" + strconv.Itoa(i+1) + ". " + q + "\n"
		}
		asked += "\n"
	}
	resp, err := llm.Call(ctx, &tools.LLMRequest{
		Role:         tools.RoleReply, // the small model missed about one paraphrase in seven; this judgement decides whether Moly nags
		SystemPrompt: "You judge whether a question would repeat something already asked or already told.",
		UserPrompt: block + dialogue.SaidText() + asked + "Moly is thinking of asking the user these questions:\n" + list + "\n" +
			"For each question, go through the earlier questions Q1, Q2, ... and Moly's last message one by one and ask what each of them asks FOR (the information wanted). Is it a repeat? It is a repeat when the answer to it would be the same as the answer to a question Moly already asked " +
			"(whatever the words: if answering one question would also answer the other, they are a repeat, even when one is more specific or one is a yes/no question and the other asks for a detail; for instance \"How soon do you need an answer?\" and \"By when must this be decided?\" ask for the same thing), including Moly's last message, " +
			"or when the user has already told the answer. Respond with ONLY JSON that has one entry per question number, " +
			"true for a repeat and false for a question that is new, for example {\"1\": true, \"2\": false}",
		MaxTokens:   80,
		Temperature: 0.1,
	})
	if err != nil {
		log.Printf("[GapNovelty] judgement failed: %v (all kept)", err)
		return gaps
	}
	var repeat map[string]tools.LenientBool
	if err := tools.SafeJSONParse("GapNovelty", []byte(resp.Content), &repeat); err != nil {
		log.Printf("[GapNovelty] judgement not readable (all kept): %v", err)
		return gaps
	}
	var kept []tools.Gap
	for i, g := range gaps {
		if bool(repeat[strconv.Itoa(i+1)]) {
			log.Printf("[GapNovelty] gap %q repeats an earlier question or answer: dropped", g.Type)
			continue
		}
		kept = append(kept, g)
	}
	return kept
}

// askedBeforeLine is what the gap prompt says about questions already asked.
const askedBeforeLine = "Do NOT ask a question that Moly already asked (it is listed as open, or it is in Moly's last message), even in other words. " +
	"If the user did not answer it, they chose to talk about something else.\n"
