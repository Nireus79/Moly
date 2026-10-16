package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"moly/models"
	"moly/tools"
)

// MessageReading is what one reading of a user's message gives: what the user is doing with it (the intent judge) and whether it
// is an instruction about Moly itself (the meta-instruction judge). Both read the same message and the same last exchange, so one
// model call answers both (plan "models by task", step 3, 2026-10-15): the two prompts are the judges' own, unchanged, in one.
type MessageReading struct {
	Intent IntentAnalysis
	Meta   *MetaInstruction // nil: not a meta-instruction
	OK     bool             // the model answered and the answer was read; false: the caller falls back to the separate judges
}

// readingSystemPrompt is built from the two judges' own prompts, so that they cannot drift apart: the definitions of the intent
// judge and the definitions of the meta-instruction judge, with one answer format for both.
func readingSystemPrompt() string {
	intentDefs := intentSystemPrompt[strings.Index(intentSystemPrompt, "Intent definitions:"):]
	metaDefs := metaInstructionSystemPrompt[strings.Index(metaInstructionSystemPrompt, "Meta-instructions include:"):strings.Index(metaInstructionSystemPrompt, "RESPOND WITH ONLY JSON")] +
		metaInstructionSystemPrompt[strings.Index(metaInstructionSystemPrompt, `"contradictory" is true ONLY`):]
	return `You read one message that a user wrote to Moly, a communication coach, in two ways, and answer both in ONE JSON object.

Respond with ONLY a JSON object (no markdown, no explanation):
{
  "intent": {
    "intent": "greeting|asking|sharing|reacting|venting|confirming|unknown",
    "confidence": 0.0-1.0,
    "answersQuestion": true|false,
    "adjustsDraft": true|false,
    "aboutMoly": true|false,
    "wantsResult": true|false,
    "reasoning": "brief explanation of why"
  },
  "meta": {
    "isMetaInstruction": true/false,
    "type": "identity|focus|instruction|scope|constraint|none",
    "confidence": 0.0-1.0,
    "targetTopic": "extracted focus topic or null",
    "targetBehavior": "extracted behavior or null",
    "contradictory": true/false,
    "reasoning": "brief explanation"
  }
}

READING 1, "intent": analyze what the user is doing in their message.

` + intentDefs + `

READING 2, "meta": decide whether this message is an instruction ABOUT Moly's behavior or focus, not TO Moly for advice.

` + metaDefs
}

// ReadMessage reads a message once for both judges. exchange is the last exchange of the conversation (models.WithExchange). It
// returns OK=false when the model failed or its answer could not be read; the caller then asks the two judges separately.
func ReadMessage(ctx context.Context, llm tools.LLMProvider, message, exchange string) MessageReading {
	msg := strings.TrimSpace(message)
	if llm == nil || msg == "" {
		return MessageReading{}
	}
	if strings.TrimSpace(exchange) == "" {
		exchange = "(Beginning of conversation)\n"
	}
	resp, err := llm.Call(ctx, &tools.LLMRequest{
		SystemPrompt: readingSystemPrompt(),
		UserPrompt:   fmt.Sprintf("User message: %q\n\nRecent conversation context:\n%s\nRead the message in both ways.", msg, exchange),
		Temperature:  0.3,
		MaxTokens:    400,
	})
	if err != nil {
		log.Printf("[MessageReading] model call failed: %v (the judges are asked separately)", err)
		return MessageReading{}
	}
	var parts struct {
		Intent json.RawMessage `json:"intent"`
		Meta   json.RawMessage `json:"meta"`
	}
	if err := tools.SafeJSONParse("MessageReading", []byte(resp.Content), &parts); err != nil || len(parts.Intent) == 0 {
		log.Printf("[MessageReading] answer unreadable (the judges are asked separately)")
		return MessageReading{}
	}
	reading := MessageReading{OK: true}
	reading.Intent = (&LLMIntentDetector{}).parseIntentResponseWithLLM(string(parts.Intent), msg)
	if len(parts.Meta) > 0 {
		reading.Meta = (&MetaInstructionDetector{}).parseMetaInstructionResponse(string(parts.Meta), msg)
	}
	return reading
}

// readIntent is an IntentSource that returns an analysis that was already made (by ReadMessage).
type readIntent struct{ analysis IntentAnalysis }

// DetectIntentWithLLM returns the analysis that was made; the arguments are not used again.
func (r readIntent) DetectIntentWithLLM(string, []models.Message) IntentAnalysis { return r.analysis }

// IntentFromReading makes an IntentSource of a reading, for the intent stage (DecideIntent).
func IntentFromReading(r MessageReading) IntentSource { return readIntent{analysis: r.Intent} }
