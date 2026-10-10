package agents

import "testing"

// PHASE 3 wiring: the LLM answer "greeting" must map to IntentGreet. The prompt offers it and the parser accepts it.
func TestParseIntentGreeting(t *testing.T) {
	lid := &LLMIntentDetector{}
	got := lid.parseIntentResponseWithLLM(`{"intent": "greeting", "confidence": 0.9, "reasoning": "only a hello"}`, "Hello Moly")
	if got.Intent != IntentGreet {
		t.Fatalf("expected IntentGreet, got %v", got.Intent)
	}
}
