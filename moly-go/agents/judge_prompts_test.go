package agents

import (
	"context"
	"strings"
	"testing"

	"moly/tools"
)

type recordingLLM struct {
	out    string
	prompt string
}

func (r *recordingLLM) Call(ctx context.Context, req *tools.LLMRequest) (*tools.LLMResponse, error) {
	r.prompt = req.UserPrompt
	return &tools.LLMResponse{Content: r.out}, nil
}

// Measured on Claude haiku (live run 2026-10-11): a prompt that only offers "yes / no / unclear" without saying what each
// means made the model answer "unclear" to a plain "Yes, switch." (3 of 8 replays) and "none" ("not a person") to
// "girlfriend". The prompts name what each answer means; these tests keep them from being shortened back.
func TestGoalSwitchJudgeNamesWhatEachAnswerMeans(t *testing.T) {
	llm := &recordingLLM{out: `{"answer": "yes"}`}
	if got := JudgeSwitchAnswer(context.Background(), llm, "Switch to X?", "Yes, switch.", "X"); got != SwitchYes {
		t.Fatalf("got %v", got)
	}
	for _, want := range []string{`"yes": the answer agrees`, `"no": the answer declines`, `"unclear": only when`} {
		if !strings.Contains(llm.prompt, want) {
			t.Errorf("the prompt lacks %q:\n%s", want, llm.prompt)
		}
	}
}

func TestPersonReferenceJudgeNamesWhatEachAnswerMeans(t *testing.T) {
	llm := &recordingLLM{out: `{"refersTo": "new", "name": ""}`}
	JudgePersonReference(context.Background(), llm, "girlfriend", "my girlfriend is moving", "", nil)
	for _, want := range []string{"girlfriend, boss, sister, neighbour", "a group (family, team, everyone", "a pronoun with nobody to point to"} {
		if !strings.Contains(llm.prompt, want) {
			t.Errorf("the prompt lacks %q:\n%s", want, llm.prompt)
		}
	}
}

// The meta-instruction answer was matched as text (`"isMetaInstruction":true`, no space), so a model that writes a space, a
// code fence or a null topic never produced a meta-instruction (live battery: 0 of 9 on Claude haiku).
func TestMetaInstructionAnswerIsParsedAsJSON(t *testing.T) {
	mid := &MetaInstructionDetector{}
	for _, in := range []string{
		`{"isMetaInstruction": true, "type": "scope", "confidence": 0.9, "targetTopic": null, "targetBehavior": "no advice", "reasoning": "x"}`,
		"```json\n{\n  \"isMetaInstruction\": true,\n  \"type\": \"scope\",\n  \"confidence\": 0.9\n}\n```",
		`Here you go: {"isMetaInstruction":true,"type":"scope","confidence":0.8}`,
	} {
		got := mid.parseMetaInstructionResponse(in, "Please don't give me advice")
		if got == nil || got.Type != "scope" || got.Confidence < 0.8 || got.Source != "llm" {
			t.Errorf("%q parsed as %+v", in, got)
		}
	}
	for _, in := range []string{`{"isMetaInstruction": false, "type": "none"}`, `{"isMetaInstruction": true, "type": "banana"}`, `no json`} {
		if got := mid.parseMetaInstructionResponse(in, "x"); got != nil {
			t.Errorf("%q should not be a meta-instruction, got %+v", in, got)
		}
	}
}

// Flags in a model answer are read as JSON, not found by searching the text for `"key": true`.
func TestJSONFlagsReadsTheAnswerAsJSON(t *testing.T) {
	for _, in := range []string{
		`{"transparency_engaged": true, "empathy_engaged": false}`,
		"```json\n{\n \"transparency_engaged\":true,\n \"empathy_engaged\": \"false\"\n}\n```",
	} {
		f := jsonFlags("t", in)
		if !f["transparency_engaged"] || f["empathy_engaged"] || f["missing"] {
			t.Errorf("%q read as %v", in, f)
		}
	}
	if f := jsonFlags("t", `the "transparency_engaged": true thing, no JSON object`); f["transparency_engaged"] {
		t.Error("text that is not JSON sets no flag")
	}
}

// An intent answer that is not JSON is not searched for words: the intent is unknown.
func TestIntentAnswerThatIsNotJSONIsUnknown(t *testing.T) {
	a := (&LLMIntentDetector{}).parseIntentResponseWithLLM(`I think the user is "asking" something, intent: asking`, "hello")
	if a.Intent != IntentUnknown || a.Confidence != 0 {
		t.Fatalf("got %+v", a)
	}
}

func TestMetaInstructionAnswerCarriesContradictory(t *testing.T) {
	mid := &MetaInstructionDetector{}
	got := mid.parseMetaInstructionResponse(`{"isMetaInstruction": true, "type": "instruction", "contradictory": true}`, "x")
	if got == nil || !got.Contradictory {
		t.Fatalf("got %+v", got)
	}
	got = mid.parseMetaInstructionResponse(`{"isMetaInstruction": true, "type": "instruction", "contradictory": "false"}`, "x")
	if got == nil || got.Contradictory {
		t.Fatalf("got %+v", got)
	}
}

func TestPrincipleIDAcceptsNamesAndIDs(t *testing.T) {
	for in, want := range map[string]string{
		"Stakeholder Consideration": "stakeholder_consideration", "stakeholder_consideration": "stakeholder_consideration",
		"Consent & Respect": "consent_and_respect", "consent and respect": "consent_and_respect",
		"Transparency": "transparency", " User Autonomy ": "user_autonomy", "Growth & Learning": "growth_and_learning",
		"Harm Prevention": "harm_prevention", "": "",
	} {
		if got := principleID(in); got != want {
			t.Errorf("principleID(%q) = %q, want %q", in, got, want)
		}
	}
}

type fixedCountingLLM struct {
	out   string
	calls int
}

func (c *fixedCountingLLM) Call(ctx context.Context, req *tools.LLMRequest) (*tools.LLMResponse, error) {
	c.calls++
	return &tools.LLMResponse{Content: c.out}, nil
}

// A "reacting" message ("Yes", "ok") costs one model call: the intent. It used to cost a second call to find "what they react to",
// and nothing ever read the result.
func TestReactingIntentCostsOneModelCall(t *testing.T) {
	llm := &fixedCountingLLM{out: `{"intent": "reacting", "confidence": 0.9, "answersQuestion": true, "adjustsDraft": false}`}
	a := NewLLMIntentDetector(llm).DetectIntentWithLLM("Yes", nil)
	if a.Intent != IntentReact || llm.calls != 1 {
		t.Fatalf("intent %v after %d calls", a.Intent, llm.calls)
	}
}
