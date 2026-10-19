package tools

import (
	"context"
	"fmt"
	"log"
	"strings"
)

// The summary is Moly's memory of a conversation: a short piece of plain prose, rewritten after every turn from the previous
// summary and the newest turn. It is free text on purpose: it can hold several people, how they relate to each other and to
// the user, and why things matter, and every model reads prose, whatever its size. The facts that the code depends on (the
// locked goal, the people) live in the database and are given to the writer as fixed facts.

// maxSummaryRunes keeps the summary small enough to travel with every prompt (about 150 words).
const maxSummaryRunes = 1100

// SummaryAnchors are the facts the summary must not contradict. They come from the database.
type SummaryAnchors struct {
	Goal   string
	People []string // each with what the person is to the user
}

// ConversationSummarizer rewrites the summary of a conversation.
type ConversationSummarizer struct {
	llm LLMProvider
}

// NewConversationSummarizer creates a summarizer.
func NewConversationSummarizer(llm LLMProvider) *ConversationSummarizer {
	return &ConversationSummarizer{llm: llm}
}

// SummarizeTurn returns the new summary: the previous one corrected and extended with the newest turn. An empty or unusable answer
// is an error, and the caller keeps the previous summary.
func (cs *ConversationSummarizer) SummarizeTurn(ctx context.Context, previous string, anchors SummaryAnchors, userMessage, molyReply string) (string, error) {
	if cs.llm == nil {
		return "", fmt.Errorf("no model available")
	}
	if strings.TrimSpace(userMessage) == "" {
		return "", fmt.Errorf("no message to summarize")
	}
	var b strings.Builder
	b.WriteString("Current memory of the conversation:\n")
	if strings.TrimSpace(previous) == "" {
		b.WriteString("(nothing yet)\n")
	} else {
		b.WriteString(strings.TrimSpace(previous) + "\n")
	}
	if anchors.Goal != "" || len(anchors.People) > 0 {
		b.WriteString("\nFixed facts (do not contradict them):\n")
		if anchors.Goal != "" {
			b.WriteString("- The goal the user is working on: " + anchors.Goal + "\n")
		}
		for _, p := range anchors.People {
			b.WriteString("- " + p + "\n")
		}
	}
	b.WriteString("\nThe newest turn:\nUser: " + strings.TrimSpace(userMessage) + "\n")
	if strings.TrimSpace(molyReply) != "" {
		b.WriteString("Moly: " + strings.TrimSpace(molyReply) + "\n")
	}
	b.WriteString("\nRewrite the memory so that it includes this turn. Keep what is still true, correct what changed, add what is new. " +
		"Cover: who the people are, how they relate to each other and to the user, what the user wants and why, what has been decided or confirmed, " +
		"what is still open, and what Moly last asked, proposed or wrote (describe it in a few words; do not copy a draft). " +
		"Write plain prose of at most 120 words, in short statements of fact. " +
		"Do not write that things are \"established\" or \"known\", do not repeat the fixed facts as a list, and do not describe the conversation itself. " +
		"Use only what the user or Moly said; invent nothing and do not guess what the user prefers. " +
		"Do not connect two statements the user made separately, and do not rank, compare or weigh the user's worries or feelings unless the user did. " +
		"Write \"the user\" or \"they\" for the user, and use he or she for another person only when the conversation gave that person's pronoun; never guess anyone's gender. Output only the memory.")

	resp, err := cs.llm.Call(ctx, &LLMRequest{
		SystemPrompt: "You keep the memory of a conversation between a user and Moly, a communication coach. You write it as short, plain prose.",
		UserPrompt:   b.String(),
		// 120 words of prose is about 200 tokens on older models and about 260 on Claude 5 (its tokenizer counts about 30% more); the
		// limit was 320 and the memory was cut off in the middle of a sentence (found live, 2026-10-15)
		MaxTokens: 700,
		Temperature:  0.2,
		Retries:      2,
	})
	if err != nil {
		return "", fmt.Errorf("summary call failed: %w", err)
	}
	out := cleanSummary(resp.Content)
	if out == "" {
		return "", fmt.Errorf("the summary came back empty or unusable")
	}
	log.Printf("[ConversationSummarizer] ✓ Summary rewritten (%d characters)", len([]rune(out)))
	return out, nil
}

// cleanSummary turns the model's answer into the stored text: no code fence, no label, no JSON, and not longer than the limit
// (cut at the end of a sentence when possible).
func cleanSummary(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)
	for _, label := range []string{"Memory:", "Summary:", "New memory:", "Updated memory:"} {
		if strings.HasPrefix(strings.ToLower(s), strings.ToLower(label)) {
			s = strings.TrimSpace(s[len(label):])
		}
	}
	s = strings.Trim(s, "\"")
	if s == "" || strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[") {
		return ""
	}
	runes := []rune(s)
	if len(runes) > maxSummaryRunes {
		cut := string(runes[:maxSummaryRunes])
		if i := strings.LastIndexAny(cut, ".!?"); i > maxSummaryRunes/2 {
			cut = cut[:i+1]
		}
		s = cut
	}
	return strings.TrimSpace(s)
}

