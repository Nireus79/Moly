package main

// Step 0 of ORCHESTRATOR_DESIGN.md section 10: a scripted end-to-end test of the real message handler.
// The language model is scripted by prompt type, the database is real (temporary), and the handler is called directly.
// Expected results are written first; this test is expected to FAIL until the orchestration steps are done.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"moly/agents"
	"moly/database"
	"moly/models"
	"moly/tools"
)

// scriptedLLM answers by prompt type: the intent analyzer, the extraction, and anything else (reply text).
// A scenario can override a prompt type by setting the matching field.
type scriptedLLM struct {
	constitutional func(prompt string) (string, error) // nil: no violation
	reply          func(prompt string) (string, error) // nil: "Moly reply."
	gaps           func() (string, error)              // nil: no gaps
	principles     func() (string, error)              // nil: no principle engaged
	shift          func() (string, error)              // nil: no_shift
	seen           *[]string                           // when set, every prompt the model receives is appended
	persisting     bool                                // the persistence check answers "the user is pressing on"
}

func (l scriptedLLM) Call(ctx context.Context, req *tools.LLMRequest) (*tools.LLMResponse, error) {
	prompt := req.SystemPrompt + "\n" + req.UserPrompt
	if l.seen != nil {
		*l.seen = append(*l.seen, prompt)
	}
	switch {
	case strings.Contains(req.SystemPrompt, "You name conversations"):
		// the title writer (conversation_title.go)
		return &tools.LLMResponse{Content: "Opening message"}, nil
	case strings.Contains(req.SystemPrompt, "keep the memory of a conversation"):
		// the summary writer: the memory holds the newest user message, so a test can see which turns it has taken in
		start := strings.LastIndex(prompt, "\nUser: ")
		msg := ""
		if start >= 0 {
			msg = strings.SplitN(prompt[start+len("\nUser: "):], "\n", 2)[0]
		}
		return &tools.LLMResponse{Content: "MEMORY-AFTER[" + msg + "]"}, nil
	case strings.Contains(req.SystemPrompt, "who a phrase refers to"):
		// the person-reference judge: words about the text are no person; a pronoun or the kind of person ("she", "the girl")
		// points to the one person the conversation has brought up; with nobody brought up it is a new person
		switch {
		case strings.Contains(prompt, "Phrase taken from it: \"bit too formal\""):
			return &tools.LLMResponse{Content: `{"refersTo": "none", "name": ""}`}, nil
		}
		if strings.Contains(prompt, "Phrase taken from it: \"her\"") && strings.Contains(prompt, "People already brought up in this conversation: none") {
			return &tools.LLMResponse{Content: `{"refersTo": "none", "name": ""}`}, nil // a pronoun with nobody to point to
		}
		const marker = "People already brought up in this conversation: "
		if i := strings.Index(prompt, marker); i >= 0 {
			list := strings.SplitN(prompt[i+len(marker):], "\n", 2)[0]
			if list != "none" && !strings.Contains(list, ",") {
				phrase := strings.SplitN(prompt[strings.Index(prompt, "Phrase taken from it: \"")+len("Phrase taken from it: \""):], "\"", 2)[0]
				if phrase == "she" || phrase == "girl" || phrase == "her" || (phrase == "boss" && strings.Contains(list, "boss)")) {
					return &tools.LLMResponse{Content: `{"refersTo": "known", "name": "` + list + `"}`}, nil
				}
			}
		}
		return &tools.LLMResponse{Content: `{"refersTo": "new", "name": ""}`}, nil
	case strings.Contains(req.SystemPrompt, "whether a reply answers questions"):
		// the open-question judge: "ANSWERS-PENDING", or the style answer, answers the questions; it answers per question number
		n := strings.Count(prompt, "\n1. ") + strings.Count(prompt, "\n2. ") + strings.Count(prompt, "\n3. ")
		answers := strings.Contains(prompt, "ANSWERS-PENDING") || strings.Contains(prompt, "The user replied: \"Quite informal")
		var parts []string
		for i := 1; i <= n; i++ {
			parts = append(parts, fmt.Sprintf(`"%d": %t`, i, answers))
		}
		return &tools.LLMResponse{Content: "{" + strings.Join(parts, ", ") + "}"}, nil
	case strings.Contains(req.SystemPrompt, "intent analyzer"):
		intent := "asking"
		switch {
		case strings.Contains(prompt, "User message: \"Hi Moly\""):
			intent = "greeting"
		case strings.Contains(prompt, "User message: \"Her name is Christine"):
			intent = "sharing"
		}
		// The scripted model flags a bare name answer only when Moly's question is in the history it was shown.
		answers := (strings.Contains(prompt, "User message: \"Her name is Christine.\"") || strings.Contains(prompt, "User message: \"His name is Tom.\"") || strings.Contains(prompt, "User message: \"His name is Dana.\"")) && strings.Contains(prompt, "Moly: Can you give me a name")
		adjusts := strings.Contains(prompt, "User message: \"It is good but it is too formal.\"")
		confidence := 0.9
		if strings.Contains(prompt, "User message: \"Go ahead with what you have.\"") {
			confidence = 0.3 // the bare press looks unclear to the intent model (found live): it must not be asked about
		}
		return &tools.LLMResponse{Content: fmt.Sprintf(`{"intent": "%s", "confidence": %.1f, "answersQuestion": %t, "adjustsDraft": %t, "aboutMoly": %t, "reasoning": "scripted"}`, intent, confidence, answers, adjusts, strings.Contains(prompt, "User message: \"ABOUT-MOLY"))}, nil
	case strings.Contains(req.SystemPrompt, "would repeat something already asked"):
		// the repeat judge: a candidate whose text carries REPEAT is a rewording of an earlier question
		var parts []string
		for i, line := range strings.Split(prompt[strings.Index(prompt, "thinking of asking"):], "\n") {
			if i > 0 && len(line) > 2 && line[1] == '.' {
				parts = append(parts, fmt.Sprintf(`"%c": %t`, line[0], strings.Contains(line, "REPEAT")))
			}
		}
		return &tools.LLMResponse{Content: "{" + strings.Join(parts, ", ") + "}"}, nil
	case strings.Contains(req.SystemPrompt, "helping users achieve their goals through targeted questions"):
		if l.gaps != nil {
			out, err := l.gaps()
			return &tools.LLMResponse{Content: out}, err
		}
		return &tools.LLMResponse{Content: `[]`}, nil
	case strings.Contains(prompt, "userCharacteristics"):
		return &tools.LLMResponse{Content: withGoalEvidence(prompt, extractionFor(prompt))}, nil
	case strings.Contains(req.SystemPrompt, "conversation flow and topic changes"):
		if l.shift != nil {
			out, err := l.shift()
			return &tools.LLMResponse{Content: out}, err
		}
		return &tools.LLMResponse{Content: "no_shift"}, nil
	case strings.Contains(req.SystemPrompt, "ethical reasoning assistant"):
		if l.principles != nil {
			out, err := l.principles()
			return &tools.LLMResponse{Content: out}, err
		}
		return &tools.LLMResponse{Content: `{"engagedPrinciples": [], "primaryPrinciple": "", "evidence": "", "needsClarification": false}`}, nil
	case strings.Contains(req.SystemPrompt, "still worth asking"):
		// the relevance judge for pending questions: a message that holds IRRELEVANT-NOW makes every pending question moot
		n := strings.Count(prompt, "\n1. ") + strings.Count(prompt, "\n2. ") + strings.Count(prompt, "\n3. ")
		parts := []string{}
		for i := 1; i <= n; i++ {
			parts = append(parts, fmt.Sprintf(`"%d": %t`, i, !strings.Contains(prompt, "IRRELEVANT-NOW")))
		}
		return &tools.LLMResponse{Content: "{" + strings.Join(parts, ", ") + "}"}, nil
	case strings.Contains(req.SystemPrompt, "meta-instruction detector"):
		if strings.Contains(prompt, "CONTRADICTING-ORDERS") {
			return &tools.LLMResponse{Content: `{"isMetaInstruction": true, "type": "instruction", "confidence": 0.9, "contradictory": true, "reasoning": "scripted"}`}, nil
		}
		return &tools.LLMResponse{Content: `{"isMetaInstruction": false, "type": "none", "confidence": 0.9, "contradictory": false, "reasoning": "scripted"}`}, nil
	case strings.Contains(req.SystemPrompt, "agreed to change what a conversation is working on"):
		return &tools.LLMResponse{Content: `{"answer": "yes"}`}, nil
	case strings.Contains(req.SystemPrompt, "how a newly stated goal relates"):
		return &tools.LLMResponse{Content: `{"relation": "different"}`}, nil
	case strings.Contains(req.SystemPrompt, "keeps to the same course of action"):
		return &tools.LLMResponse{Content: fmt.Sprintf(`{"persisting": %t}`, l.persisting)}, nil
	case strings.Contains(req.SystemPrompt, "confirm whether flagged evidence"):
		return &tools.LLMResponse{Content: `{"confirmed": true, "reason": "scripted"}`}, nil
	case strings.Contains(req.SystemPrompt, "You are a safety expert"):
		return &tools.LLMResponse{Content: `{"risk_level": "clear", "severity": 0, "assessment": "scripted", "recommendation": "proceed"}`}, nil
	case strings.Contains(req.SystemPrompt, "listens deeply to understand") && strings.Contains(prompt, "DANGER-AT-HOME"):
		return &tools.LLMResponse{Content: `{"clarity_score": 0.9, "can_proceed": true, "priority": "crisis", "key_concerns": [], "message_quality": "clear", "clarifications": [], "response_approach": "scripted"}`}, nil
	case strings.Contains(req.SystemPrompt, "listens deeply to understand"):
		return &tools.LLMResponse{Content: `{"clarity_score": 0.9, "can_proceed": true, "priority": "normal", "key_concerns": [], "message_quality": "clear", "clarifications": [], "response_approach": "scripted"}`}, nil
	case strings.Contains(prompt, "violations"):
		// the constitutional evaluator: no principle violated
		if l.constitutional != nil {
			out, err := l.constitutional(prompt)
			return &tools.LLMResponse{Content: out}, err
		}
		return &tools.LLMResponse{Content: `{"violations": []}`}, nil
	default:
		if os.Getenv("MOLY_TRACE_PROMPTS") != "" {
			fmt.Printf("TRACE-PROMPT system=%.90q user=%.1500q\n", req.SystemPrompt, req.UserPrompt)
		}
		if l.reply != nil {
			out, err := l.reply(prompt)
			return &tools.LLMResponse{Content: out}, err
		}
		return &tools.LLMResponse{Content: "Moly reply."}, nil
	}
}

// withGoalEvidence gives a scripted goal the quote a real extractor must give: the message itself, unless the script set
// its own evidence (the scenarios about a goal that is not grounded do).
func withGoalEvidence(prompt, extraction string) string {
	var out map[string]interface{}
	if json.Unmarshal([]byte(extraction), &out) != nil {
		return extraction
	}
	if intention, _ := out["intention"].(string); intention == "" {
		return extraction
	}
	if _, set := out["intentionEvidence"]; set {
		return extraction
	}
	start := strings.Index(prompt, "Message: \"")
	end := messageEnd(prompt, start)
	if start < 0 || end < start {
		return extraction
	}
	out["intentionEvidence"] = prompt[start+len("Message: \"") : end]
	b, _ := json.Marshal(out)
	return string(b)
}

// messageEnd finds where the user's message ends in an extraction prompt: the long prompt continues with "Extract", the short
// one with the JSON form.
func messageEnd(prompt string, start int) int {
	if start < 0 {
		return -1
	}
	for _, d := range []string{"\"\n\nExtract", "\"\n\n{"} {
		if i := strings.Index(prompt[start:], d); i >= 0 {
			return start + i
		}
	}
	return -1
}

func extractionFor(prompt string) string {
	// The prompt also carries the recent exchange; the script reacts to the CURRENT message only.
	if start := strings.Index(prompt, "Message: \""); start >= 0 {
		if end := messageEnd(prompt, start); end >= 0 {
			prompt = prompt[start:end]
		}
	}
	switch {
	case strings.Contains(prompt, "Hi Moly"):
		return `{"intention": "", "intentionPrinciples": [], "contact": null, "userCharacteristics": [], "contactCharacteristics": {}}`
	case strings.Contains(prompt, "my mother instead"), strings.Contains(prompt, "I think she would be glad"):
		return `{"intention": "", "intentionPrinciples": [], "contact": null, "userCharacteristics": [], "contactCharacteristics": {}}`
	case strings.Contains(prompt, "Yes, let's do that."):
		return `{"intention": "", "intentionPrinciples": [], "contact": null, "userCharacteristics": [], "contactCharacteristics": {}}`
	case strings.Contains(prompt, "Maybe I should write to Anna."):
		return `{"intention": "", "intentionPrinciples": [], "contact": {"name": "Anna", "label": "Anna", "relationship": "other", "confidence": 0.9, "evidence": "Anna", "basis": "guessed", "traits": []}, "userCharacteristics": [], "contactCharacteristics": {}}`
	case strings.Contains(prompt, "Yes, I mean my colleague Anna."):
		return `{"intention": "", "intentionPrinciples": [], "contact": {"name": "Anna", "label": "my colleague Anna", "relationship": "professional", "confidence": 0.9, "evidence": "my colleague Anna", "basis": "stated", "traits": []}, "userCharacteristics": [], "contactCharacteristics": {}}`
	case strings.Contains(prompt, "thank-you note to my colleague Anna"):
		return `{"intention": "write a thank-you note to Anna", "intentionBasis": "stated", "intentionEvidence": "thank-you note to my colleague Anna", "intentionPrinciples": [], "contact": {"name": "Anna", "label": "my colleague Anna", "relationship": "professional", "confidence": 0.95, "evidence": "my colleague Anna", "basis": "stated", "traits": []}, "userCharacteristics": [], "contactCharacteristics": {}}`
	case strings.Contains(prompt, "split the shifts between Dana and Mark"):
		return `{"intention": "ask Dana to split the shifts fairly with Mark", "intentionBasis": "stated", "intentionEvidence": "split the shifts between Dana and Mark", "intentionPrinciples": [], "contact": {"name": "Dana", "label": "Dana", "role": "manager", "relationship": "professional", "confidence": 0.95, "evidence": "Dana", "basis": "stated", "traits": []}, "otherContacts": [{"name": "Mark", "label": "Mark", "role": "colleague", "relationship": "professional", "confidence": 0.95, "evidence": "Mark", "basis": "stated", "traits": []}], "userCharacteristics": [], "contactCharacteristics": {}}`
	case strings.Contains(prompt, "ZQXV unrelated words"):
		return `{"intention": "get advice about a friendship", "intentionBasis": "stated", "intentionEvidence": "my friend needs help", "intentionPrinciples": [], "contact": null, "userCharacteristics": [], "contactCharacteristics": {}}`
	case strings.Contains(prompt, "FORGERY"):
		return `{"intention": "forge my landlord's signature", "intentionBasis": "stated", "intentionPrinciples": [], "contact": null, "userCharacteristics": [], "contactCharacteristics": {}}`
	case strings.Contains(prompt, "Anna is my colleague."):
		return `{"intention": "", "intentionPrinciples": [], "contact": {"name": "Anna", "label": "Anna", "relationship": "professional", "confidence": 0.95, "evidence": "Anna", "basis": "stated", "traits": []}, "userCharacteristics": [], "contactCharacteristics": {}}`
	case strings.Contains(prompt, "She is always helping me."):
		return `{"intention": "", "intentionPrinciples": [], "contact": {"name": "", "label": "she", "relationship": "other", "confidence": 0.9, "evidence": "She", "basis": "stated", "traits": []}, "userCharacteristics": [], "contactCharacteristics": {}}`
	case strings.Contains(prompt, "Please make it less formal."):
		return `{"intention": "", "intentionPrinciples": [], "contact": {"name": "", "label": "a bit too formal", "relationship": "other", "confidence": 0.9, "evidence": "less formal", "basis": "stated", "traits": []}, "userCharacteristics": [], "contactCharacteristics": {}}`
	case strings.Contains(prompt, "Perhaps I need to say something."):
		return `{"intention": "reconcile with an old friend", "intentionBasis": "guessed", "intentionPrinciples": [], "contact": null, "userCharacteristics": [], "contactCharacteristics": {}}`
	case strings.Contains(prompt, "whether to tell her"):
		return `{"intention": "decide whether to tell her about my interests", "intentionPrinciples": [], "contact": null, "userCharacteristics": [], "contactCharacteristics": {}}`
	case strings.Contains(prompt, "CHAT-GRANDMA"):
		return `{"intention": "", "intentionPrinciples": [], "contact": {"name": "", "label": "grandmother", "relationship": "family", "confidence": 0.9, "evidence": "my grandmother", "basis": "stated", "traits": []}, "userCharacteristics": [], "contactCharacteristics": {}}`
	case strings.Contains(prompt, "CALL-GRANDMA"):
		return `{"intention": "call my grandmother", "intentionBasis": "stated", "intentionEvidence": "call my grandmother", "intentionPrinciples": [], "contact": {"name": "", "label": "grandmother", "relationship": "family", "confidence": 0.9, "evidence": "my grandmother", "basis": "stated", "traits": []}, "userCharacteristics": [], "contactCharacteristics": {}}`
	case strings.Contains(prompt, "SECRET-TEST"):
		return `{"intention": "write a complaint to my bank (password hunter2)", "intentionBasis": "stated", "intentionEvidence": "my password is hunter2", "intentionPrinciples": [], "contact": null, "userCharacteristics": ["uses the password hunter2"], "contactCharacteristics": {}, "sensitive": ["hunter2"]}`
	case strings.Contains(prompt, "DANGER-AT-HOME"):
		return `{"intention": "", "intentionPrinciples": [], "contact": {"name": "", "label": "husband", "relationship": "family", "confidence": 0.9, "evidence": "my husband", "basis": "stated", "traits": []}, "userCharacteristics": [], "contactCharacteristics": {}}`
	case strings.Contains(prompt, "SWITCH-WITH-NEW-PERSON"):
		return `{"intention": "write to my brother about the party", "intentionBasis": "stated", "intentionEvidence": "write to my brother about the party", "intentionPrinciples": [], "contact": {"name": "", "label": "brother", "relationship": "family", "confidence": 0.9, "evidence": "my brother", "basis": "stated", "traits": []}, "userCharacteristics": [], "contactCharacteristics": {}}`
	case strings.Contains(prompt, "WRITE-TO-BOSS"):
		return `{"intention": "write to my boss", "intentionBasis": "stated", "intentionEvidence": "write to my boss", "intentionPrinciples": [], "contact": {"name": "", "label": "boss", "role": "boss", "relationship": "professional", "confidence": 0.9, "evidence": "my boss", "basis": "stated", "traits": []}, "userCharacteristics": [], "contactCharacteristics": {}}`
	case strings.Contains(prompt, "His name is Dana."):
		return `{"intention": "", "intentionPrinciples": [], "contact": {"name": "Dana", "label": "boss", "relationship": "professional", "confidence": 0.95, "evidence": "Dana", "basis": "stated", "traits": []}, "userCharacteristics": [], "contactCharacteristics": {}}`
	case strings.Contains(prompt, "BOSS-AGAIN"):
		return `{"intention": "", "intentionPrinciples": [], "contact": {"name": "", "label": "boss", "relationship": "professional", "confidence": 0.9, "evidence": "my boss", "basis": "stated", "traits": []}, "userCharacteristics": [], "contactCharacteristics": {}}`
	case strings.Contains(prompt, "PRONOUN-AS-NAME"):
		return `{"intention": "", "intentionPrinciples": [], "contact": {"name": "her", "label": "her", "relationship": "professional", "confidence": 0.95, "evidence": "her", "basis": "stated", "traits": []}, "userCharacteristics": [], "contactCharacteristics": {}}`
	case strings.Contains(prompt, "His name is Tom."):
		return `{"intention": "", "intentionPrinciples": [], "contact": {"name": "Tom", "label": "brother", "relationship": "family", "nameKnown": true, "confidence": 0.95, "evidence": "Tom", "basis": "stated", "traits": []}, "userCharacteristics": [], "contactCharacteristics": {}}`
	case strings.Contains(prompt, "Her name is Christine"):
		// Hostile on purpose: the live model invented a goal from a bare name answer. The intent stage must drop it.
		return `{"intention": "identify who Christine is", "intentionPrinciples": [], "contact": {"name": "Christine", "label": "girl", "relationship": "other", "confidence": 0.9, "traits": []}, "userCharacteristics": [], "contactCharacteristics": {}}`
	default: // "I want to write a first message to a girl I saw on fetlife."
		return `{"intention": "initiate a conversation with the girl on fetlife", "intentionPrinciples": [], "contact": {"name": "", "label": "girl", "relationship": "other", "confidence": 0.85, "traits": []}, "userCharacteristics": [], "contactCharacteristics": {}}`
	}
}

// asking is how the gap-question prompt introduces the gap. The dialogue block of other prompts also holds Moly's earlier
// questions, so the scripts must react to this phrase and not to the question text alone.
const asking = "missing important context about: "

const nameQuestion = "Can you give me a name for the girl you mentioned?"

// harnessDB returns the shared test database, with a fresh user and session for one scenario.
func harnessUser(t *testing.T, db *database.Database, userID, token string) {
	t.Helper()
	now := time.Now().Unix()
	c := db.GetConnection()
	if _, err := c.Exec(`INSERT INTO users (id, created_at, last_active) VALUES (?, ?, ?)`, userID, now, now); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if _, err := c.Exec(`INSERT INTO sessions (id, user_id, device_id, created_at, expires_at, last_used) VALUES (?, ?, 'harness', ?, ?, ?)`,
		token, userID, now, now+3600, now); err != nil {
		t.Fatalf("insert session: %v", err)
	}
}

func sendMessage(t *testing.T, srv *APIServer, token, conversationID, message string) (reply string, convID string, status int) {
	t.Helper()
	reply, convID, status, _ = sendFull(t, srv, token, conversationID, message, false)
	return reply, convID, status
}

// sendSkip presses the skip button: the extension sends a fixed message and skipQuestions.
func sendSkip(t *testing.T, srv *APIServer, token, conversationID string) (reply string, status int) {
	t.Helper()
	reply, _, status, _ = sendFull(t, srv, token, conversationID, "Go ahead with what you have.", true)
	return reply, status
}

// sendFull is sendMessage that can press skip and also returns the reply metadata.
func sendFull(t *testing.T, srv *APIServer, token, conversationID, message string, skip bool) (reply string, convID string, status int, metadata map[string]interface{}) {
	t.Helper()
	body, _ := json.Marshal(map[string]interface{}{
		"message":          message,
		"conversationId":   conversationID,
		"browserSessionId": "harness-session",
		"skipQuestions":    skip,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/message-processor", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	srv.MessageProcessorHandler(rec, req)
	var out map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &out)
	reply, _ = out["response"].(string)
	convID, _ = out["conversationId"].(string)
	metadata, _ = out["metadata"].(map[string]interface{})
	return reply, convID, rec.Code, metadata
}

type contactRow struct {
	Name, NameStatus string
}

func contactsFor(t *testing.T, db *database.Database, userID string) []contactRow {
	t.Helper()
	rows, err := db.GetConnection().Query(`SELECT name, name_status FROM contacts WHERE user_id = ? AND status = 'active' ORDER BY id`, userID)
	if err != nil {
		t.Fatalf("query contacts: %v", err)
	}
	defer rows.Close()
	var out []contactRow
	for rows.Next() {
		var r contactRow
		rows.Scan(&r.Name, &r.NameStatus)
		out = append(out, r)
	}
	return out
}

// lockedGoal returns the primary goal saved for a conversation ("" when none).
func lockedGoal(srv *APIServer, userID, convID string) string {
	prev := srv.loadPreviousExtraction(userID, convID)
	if prev == nil {
		return ""
	}
	return prev.PrimaryGoal
}

// TestScenarioS1 is the three-message scenario from the logs. Expected results are in the assertions.
func TestScenarioS1ThreeMessages(t *testing.T) {
	db, err := database.Init(testDBPath)
	if err != nil {
		t.Fatal(err)
	}
	srv, err := NewAPIServer(scenarioSource{llm: scriptedLLM{}}, db)
	if err != nil {
		t.Fatal(err)
	}
	userID := fmt.Sprintf("harness_%d", time.Now().UnixNano())
	token := "token_" + userID
	harnessUser(t, db, userID, token)

	// Step 1: greeting. No person, no goal, a greeting reply.
	reply, convID, code := sendMessage(t, srv, token, "", "Hi Moly")
	if code != http.StatusOK || strings.TrimSpace(reply) == "" {
		t.Fatalf("step 1: status %d, reply %q", code, reply)
	}
	if got := contactsFor(t, db, userID); len(got) != 0 {
		t.Fatalf("step 1: a greeting must not save a person, got %+v", got)
	}
	if reply == nameQuestion {
		t.Fatalf("step 1: a greeting must not ask for a name")
	}
	if g := lockedGoal(srv, userID, convID); g != "" {
		t.Fatalf("step 1: a greeting must not lock a goal, got %q", g)
	}

	// Step 2: the goal message. Exactly the name question, and one person waiting for a name.
	reply, convID, code = sendMessage(t, srv, token, convID, "I want to write a first message to a girl I saw on fetlife.")
	if code != http.StatusOK {
		t.Fatalf("step 2: status %d", code)
	}
	if reply != nameQuestion {
		t.Fatalf("step 2: reply must be exactly the name question\n got: %q\nwant: %q", reply, nameQuestion)
	}
	const goal = "initiate a conversation with the girl on fetlife"
	if g := lockedGoal(srv, userID, convID); g != goal {
		t.Fatalf("step 2: the goal must be locked once, want %q got %q", goal, g)
	}
	got := contactsFor(t, db, userID)
	if len(got) != 1 || got[0].Name != "girl" || got[0].NameStatus != "asked" {
		t.Fatalf("step 2: want one person named \"girl\" with name_status asked, got %+v", got)
	}

	// Step 3: the name is given. The same person is renamed (one row), and the goal continues.
	reply, _, code = sendMessage(t, srv, token, convID, "Her name is Christine.")
	if code != http.StatusOK {
		t.Fatalf("step 3: status %d", code)
	}
	if reply == nameQuestion {
		t.Fatalf("step 3: the name question must not be asked again")
	}
	if g := lockedGoal(srv, userID, convID); g != goal {
		t.Fatalf("step 3: a message with no goal must keep the locked goal %q, got %q", goal, g)
	}
	got = contactsFor(t, db, userID)
	if len(got) != 1 || got[0].Name != "Christine" || got[0].NameStatus != "named" {
		t.Fatalf("step 3: want one person named \"Christine\" with name_status named, got %+v", got)
	}
}

// scenarioServer builds a handler with a scripted model and a fresh user. It returns the server, database, user and token.
func scenarioServer(t *testing.T, llm scriptedLLM) (*APIServer, *database.Database, string, string) {
	t.Helper()
	db, err := database.Init(testDBPath)
	if err != nil {
		t.Fatal(err)
	}
	calls := &callLog{}
	t.Cleanup(func() {
		if n := calls.withoutUser(); n > 0 {
			t.Errorf("%d model call(s) were made for nobody: a component of the message flow is not bound to the user", n)
		}
	})
	srv, err := NewAPIServer(scenarioSource{llm: llm, calls: calls}, db)
	if err != nil {
		t.Fatal(err)
	}
	userID := fmt.Sprintf("harness_%d", time.Now().UnixNano())
	token := "token_" + userID
	harnessUser(t, db, userID, token)
	return srv, db, userID, token
}

func assistantRows(t *testing.T, db *database.Database, userID string) []string {
	t.Helper()
	rows, err := db.GetConnection().Query(`SELECT m.content FROM chat_messages m JOIN conversations c ON c.id = m.conversation_id WHERE c.user_id = ? AND m.role = 'assistant' ORDER BY m.rowid`, userID)
	if err != nil {
		t.Fatalf("query assistant rows: %v", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var c string
		rows.Scan(&c)
		out = append(out, c)
	}
	return out
}

// S2: the safety check fails. Rule: safety fails closed. The user gets an error status, and no assistant message is saved.
func TestScenarioS2SafetyFailureFailsClosed(t *testing.T) {
	srv, db, userID, token := scenarioServer(t, scriptedLLM{
		constitutional: func(string) (string, error) { return "", fmt.Errorf("model unavailable") },
	})
	reply, _, code := sendMessage(t, srv, token, "", "I want to write a first message to a girl I saw on fetlife.")
	if code != http.StatusServiceUnavailable {
		t.Fatalf("a failing safety check must fail closed with 503, got %d (reply %q)", code, reply)
	}
	if rows := assistantRows(t, db, userID); len(rows) != 0 {
		t.Fatalf("no assistant message may be saved when safety fails, got %q", rows)
	}
}

// S3: the model returns an empty reply text. Rule: the user always gets a non-empty reply, and what is shown is what is saved.
func TestScenarioS3EmptyModelReply(t *testing.T) {
	srv, db, userID, token := scenarioServer(t, scriptedLLM{
		reply: func(string) (string, error) { return "", nil },
	})
	reply, _, code := sendMessage(t, srv, token, "", "Hi Moly")
	if code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	if strings.TrimSpace(reply) == "" {
		t.Fatalf("the user must never get an empty reply")
	}
	rows := assistantRows(t, db, userID)
	if len(rows) != 1 || rows[0] != reply {
		t.Fatalf("saved reply must equal shown reply; shown %q saved %q", reply, rows)
	}
}

// S4: one message gets one reply and one saved assistant row, and the user row is saved too.
func TestScenarioS4OneReplyPerMessage(t *testing.T) {
	srv, db, userID, token := scenarioServer(t, scriptedLLM{})
	_, convID, _ := sendMessage(t, srv, token, "", "Hi Moly")
	sendMessage(t, srv, token, convID, "Hi Moly")
	var users, assistants int
	db.GetConnection().QueryRow(`SELECT COUNT(*) FROM chat_messages m JOIN conversations c ON c.id = m.conversation_id WHERE c.user_id = ? AND m.role = 'user'`, userID).Scan(&users)
	db.GetConnection().QueryRow(`SELECT COUNT(*) FROM chat_messages m JOIN conversations c ON c.id = m.conversation_id WHERE c.user_id = ? AND m.role = 'assistant'`, userID).Scan(&assistants)
	if users != 2 || assistants != 2 {
		t.Fatalf("two messages must give 2 user rows and 2 assistant rows, got %d and %d", users, assistants)
	}
}

// S5: the safety check finds direct harm. Rule: the harm is refused; no contact is saved and no goal is built from the message.
func TestScenarioS5DirectHarmIsRefused(t *testing.T) {
	const harmful = "I want to hurt the girl I saw on fetlife."
	srv, db, userID, token := scenarioServer(t, scriptedLLM{
		constitutional: func(string) (string, error) {
			return `{"violations": [{"principle_id": "harm_prevention", "evidence": "hurt the girl", "reasoning": "intent to harm a person", "confidence": 0.97, "is_direct_harm": true}]}`, nil
		},
	})
	reply, _, code := sendMessage(t, srv, token, "", harmful)
	t.Logf("status=%d reply=%q", code, reply)
	if strings.TrimSpace(reply) == "" {
		t.Fatalf("a refusal must still say something")
	}
	if reply == nameQuestion {
		t.Fatalf("a harmful message must not be answered with the name question")
	}
	if got := contactsFor(t, db, userID); len(got) != 0 {
		t.Fatalf("a refused message must not save a person, got %+v", got)
	}
}

// S6: after the person is named, one goal-blocking gap exists. Rule: the reply asks that one question and nothing else is asked.
func TestScenarioS6OneGapQuestion(t *testing.T) {
	const gapQ = "How formal should the first message be?"
	srv, db, userID, token := scenarioServer(t, scriptedLLM{
		reply: func(prompt string) (string, error) {
			if strings.Contains(prompt, asking+gapQ) {
				return gapQ, nil // the model phrases the gap it was given
			}
			return "Moly reply.", nil
		},
		gaps: func() (string, error) {
			return fmt.Sprintf(`[{"type": "tone", "description": %q, "assumption": "how formal to be", "severity": "high", "confidence": 0.9, "goalTarget": "current_goal"}]`, gapQ), nil
		},
	})
	_, convID, _ := sendMessage(t, srv, token, "", "Hi Moly")
	sendMessage(t, srv, token, convID, "I want to write a first message to a girl I saw on fetlife.")
	reply, _, code := sendMessage(t, srv, token, convID, "Her name is Christine.")
	t.Logf("status=%d reply=%q", code, reply)
	if code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	if !strings.Contains(reply, gapQ) {
		t.Fatalf("the reply must ask the open gap question %q, got %q", gapQ, reply)
	}
	if strings.Count(reply, "?") != 1 {
		t.Fatalf("the reply must ask exactly one question, got %q", reply)
	}
	rows, _ := db.GetConnection().Query(`SELECT status, clarification_type, question_text FROM clarification_questions WHERE user_id = ?`, userID)
	for rows.Next() {
		var st, ty, q string
		rows.Scan(&st, &ty, &q)
		t.Logf("stored question: status=%s type=%s text=%q", st, ty, q)
	}
	rows.Close()
	var active int
	db.GetConnection().QueryRow(`SELECT COUNT(*) FROM clarification_questions WHERE user_id = ? AND status = 'active' AND question_text = ?`, userID, gapQ).Scan(&active)
	if active != 1 {
		t.Fatalf("the gap question must be stored once as active, got %d", active)
	}
}

const stakeholderConcern = `{"engagedPrinciples": ["stakeholder_consideration"], "primaryPrinciple": "stakeholder_consideration", "evidence": "a girl", "needsClarification": true}`

// startNamedGoal runs the first three messages of S1, so that a named person and a goal exist.
func startNamedGoal(t *testing.T, srv *APIServer, token string) string {
	t.Helper()
	_, convID, _ := sendMessage(t, srv, token, "", "Hi Moly")
	sendMessage(t, srv, token, convID, "I want to write a first message to a girl I saw on fetlife.")
	return convID
}

// S7: a gap and a principle concern are both open. Rule: the gap is asked first, and the concern is not asked in the same reply.
func TestScenarioS7GapBeforePrincipleConcern(t *testing.T) {
	const gapQ = "How formal should the first message be?"
	srv, _, _, token := scenarioServer(t, scriptedLLM{
		reply: func(prompt string) (string, error) {
			if strings.Contains(prompt, asking+gapQ) {
				return gapQ, nil
			}
			return "Moly reply.", nil
		},
		gaps: func() (string, error) {
			return fmt.Sprintf(`[{"type": "tone", "description": %q, "assumption": "how formal to be", "severity": "high", "confidence": 0.9, "goalTarget": "current_goal"}]`, gapQ), nil
		},
		principles: func() (string, error) { return stakeholderConcern, nil },
	})
	convID := startNamedGoal(t, srv, token)
	reply, _, _ := sendMessage(t, srv, token, convID, "Her name is Christine.")
	t.Logf("reply=%q", reply)
	if reply != gapQ {
		t.Fatalf("the open gap must be asked, and only it; got %q", reply)
	}
}

// S8: a principle concern with no open gap. Rule: the concern is asked once; the next message does not repeat the same question.
func TestScenarioS8PrincipleConcernAskedOnce(t *testing.T) {
	srv, _, _, token := scenarioServer(t, scriptedLLM{
		principles: func() (string, error) { return stakeholderConcern, nil },
	})
	convID := startNamedGoal(t, srv, token)
	first, _, _ := sendMessage(t, srv, token, convID, "Her name is Christine.")
	second, _, _ := sendMessage(t, srv, token, convID, "I think she would be glad to hear from me.")
	t.Logf("first=%q\nsecond=%q", first, second)
	if first == "Moly reply." || first == nameQuestion {
		t.Fatalf("the concern must be asked when it is the only open item, got %q", first)
	}
	if second == first {
		t.Fatalf("the same concern question must not be asked twice in a row: %q", second)
	}
}

// S10: nothing is open (named person, goal, no gap, no concern). Rule: the reply is a reply, never a repeat of the name or a gap question.
func TestScenarioS10NothingOpen(t *testing.T) {
	srv, _, _, token := scenarioServer(t, scriptedLLM{})
	convID := startNamedGoal(t, srv, token)
	sendMessage(t, srv, token, convID, "Her name is Christine.")
	reply, _, code := sendMessage(t, srv, token, convID, "I think she would be glad to hear from me.")
	t.Logf("reply=%q", reply)
	if code != http.StatusOK || strings.TrimSpace(reply) == "" {
		t.Fatalf("status %d, reply %q", code, reply)
	}
	if reply == nameQuestion {
		t.Fatalf("the name is known; it must not be asked again")
	}
}

// S11: a later, different goal does not replace the locked goal.
func TestScenarioS11LaterGoalDoesNotReplaceLock(t *testing.T) {
	srv, _, userID, token := scenarioServer(t, scriptedLLM{})
	convID := startNamedGoal(t, srv, token)
	const goal = "initiate a conversation with the girl on fetlife"
	sendMessage(t, srv, token, convID, "Her name is Christine.")
	sendMessage(t, srv, token, convID, "Now I need to decide whether to tell her about my interests.")
	if g := lockedGoal(srv, userID, convID); g != goal {
		t.Fatalf("the locked goal must stay %q, got %q", goal, g)
	}
}

// S12: the safety verdict on the user's message is asked for once per message. (The agent separately checks its own reply.)
func TestScenarioS12SafetyAskedOncePerMessage(t *testing.T) {
	const m1, m2, m3 = "Hi Moly", "I want to write a first message to a girl I saw on fetlife.", "Her name is Christine."
	counts := map[string]int{}
	srv, _, _, token := scenarioServer(t, scriptedLLM{
		constitutional: func(prompt string) (string, error) {
			for _, m := range []string{m1, m2, m3} {
				if strings.Contains(prompt, "\""+m+"\"") && strings.Contains(prompt, "Analyze this message") {
					counts[m]++
				}
			}
			return `{"violations": []}`, nil
		},
	})
	_, convID, _ := sendMessage(t, srv, token, "", m1)
	sendMessage(t, srv, token, convID, m2)
	sendMessage(t, srv, token, convID, m3)
	for _, m := range []string{m1, m2, m3} {
		if counts[m] != 1 {
			t.Errorf("message %q must be evaluated once, got %d", m, counts[m])
		}
	}
}

func messageRows(t *testing.T, db *database.Database, userID, role string) int {
	t.Helper()
	var n int
	db.GetConnection().QueryRow(`SELECT COUNT(*) FROM chat_messages WHERE user_id = ? AND role = ?`, userID, role).Scan(&n)
	return n
}

// S13: the commit fails. Rule: no reply is returned and nothing from the turn is kept; the retry behaves like a first try.
func TestScenarioS13CommitFailureKeepsNothing(t *testing.T) {
	srv, db, userID, token := scenarioServer(t, scriptedLLM{})
	_, convID, _ := sendMessage(t, srv, token, "", "Hi Moly")
	usersBefore, assistantsBefore := messageRows(t, db, userID, "user"), messageRows(t, db, userID, "assistant")
	if usersBefore != 1 || assistantsBefore != 1 {
		t.Fatalf("setup: want 1 and 1 rows, got %d and %d", usersBefore, assistantsBefore)
	}

	srv.beforeCommit = func() error { return fmt.Errorf("disk full") }
	const goalMsg = "I want to write a first message to a girl I saw on fetlife."
	reply, _, code := sendMessage(t, srv, token, convID, goalMsg)
	if code != http.StatusInternalServerError || reply != "" {
		t.Fatalf("a failed commit must return an error and no reply: status %d, reply %q", code, reply)
	}
	if u, a := messageRows(t, db, userID, "user"), messageRows(t, db, userID, "assistant"); u != usersBefore || a != assistantsBefore {
		t.Fatalf("nothing from the failed turn may be saved: user rows %d (was %d), assistant rows %d (was %d)", u, usersBefore, a, assistantsBefore)
	}
	if g := lockedGoal(srv, userID, convID); g != "" {
		t.Fatalf("the goal of a failed turn must not be locked, got %q", g)
	}

	// The retry is a first try: the name question is asked, and the person is marked as asked.
	srv.beforeCommit = nil
	reply, _, code = sendMessage(t, srv, token, convID, goalMsg)
	if code != http.StatusOK || reply != nameQuestion {
		t.Fatalf("the retry must ask the name question: status %d, reply %q", code, reply)
	}
	got := contactsFor(t, db, userID)
	if len(got) != 1 || got[0].Name != "girl" || got[0].NameStatus != "asked" {
		t.Fatalf("after the retry: one person 'girl' asked, got %+v", got)
	}
	if u, a := messageRows(t, db, userID, "user"), messageRows(t, db, userID, "assistant"); u != usersBefore+1 || a != assistantsBefore+1 {
		t.Fatalf("the retry saves exactly one user and one assistant row: %d and %d", u, a)
	}
}

// S14: a bare name answer states no goal. The extractor invents one ("identify who Christine is"); the intent stage
// recognises the message as an answer, so the goal is dropped, nothing reports a goal change, and no later prompt carries it.
func TestScenarioS14NameAnswerStatesNoGoal(t *testing.T) {
	var seen []string
	srv, db, userID, token := scenarioServer(t, scriptedLLM{seen: &seen})
	convID := startNamedGoal(t, srv, token)
	before := len(seen)
	reply, _, code := sendMessage(t, srv, token, convID, "Her name is Christine.")
	if code != http.StatusOK || strings.TrimSpace(reply) == "" {
		t.Fatalf("status %d, reply %q", code, reply)
	}
	const invented = "identify who Christine is"
	for _, p := range seen[before:] {
		if strings.Contains(p, invented) {
			t.Fatalf("the invented goal reached a later prompt: %.300q", p)
		}
	}
	if g := lockedGoal(srv, userID, convID); g != "initiate a conversation with the girl on fetlife" {
		t.Fatalf("the locked goal must continue, got %q", g)
	}
	got := contactsFor(t, db, userID)
	if len(got) != 1 || got[0].Name != "Christine" || got[0].NameStatus != "named" {
		t.Fatalf("want one person named Christine, got %+v", got)
	}
}

// S15: explicit self-harm intent, judged by the one safety stage with the conversation in view, gets the short
// refusal that names a specialist: status 200, no help lines, no question, and nothing saved.
func TestScenarioS15SelfHarmGetsTheSpecialistRefusal(t *testing.T) {
	srv, db, userID, token := scenarioServer(t, scriptedLLM{
		constitutional: func(string) (string, error) {
			return `{"violations": [{"principle_id": "harm_prevention", "evidence": "end my life", "reasoning": "explicit intent to self-harm", "confidence": 0.97, "is_direct_harm": true, "harm_kind": "self_harm"}]}`, nil
		},
	})
	reply, _, code := sendMessage(t, srv, token, "", "I am going to end my life.")
	if code != http.StatusOK {
		t.Fatalf("a refusal must be a reply, got status %d", code)
	}
	if !strings.Contains(reply, "specialist") || strings.Contains(reply, "?") {
		t.Fatalf("want the specialist refusal with no question, got %q", reply)
	}
	if strings.Contains(strings.ToLower(reply), "helpline") || strings.Contains(reply, "988") {
		t.Fatalf("no help lines: %q", reply)
	}
	if got := contactsFor(t, db, userID); len(got) != 0 {
		t.Fatalf("nothing is saved from a refused message, got %+v", got)
	}
}

// S16: a reply asks one question. The model's gap wording holds three questions; the reply holds exactly one.
func TestScenarioS16ReplyAsksOneQuestion(t *testing.T) {
	const compound = "How formal should it be? What is your history with her? What do you want from her?"
	const one = "How formal should the first message be?"
	srv, _, _, token := scenarioServer(t, scriptedLLM{
		reply: func(prompt string) (string, error) {
			if strings.Contains(prompt, "Draft reply:") {
				return one, nil // the rewrite
			}
			return compound, nil // the first wording
		},
		gaps: func() (string, error) {
			return `[{"type": "tone", "description": "formality", "assumption": "how formal to be", "severity": "high", "confidence": 0.9, "goalTarget": "current_goal"}]`, nil
		},
	})
	convID := startNamedGoal(t, srv, token)
	reply, _, code := sendMessage(t, srv, token, convID, "Her name is Christine.")
	if code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	if reply != one || strings.Count(reply, "?") != 1 {
		t.Fatalf("want exactly one question %q, got %q", one, reply)
	}
}

// S17: persistence is judged by the model and only after Moly raised a principle concern. A message with the words
// "but" or "actually" after an ordinary question (S9) is not persistence; pressing on after a concern is.
func TestScenarioS17PersistenceNeedsARaisedConcern(t *testing.T) {
	srv, _, _, token := scenarioServer(t, scriptedLLM{
		principles: func() (string, error) { return stakeholderConcern, nil },
		persisting: true,
		reply: func(prompt string) (string, error) {
			if strings.Contains(prompt, "Draft reply:") {
				if strings.Contains(prompt, "I will do it anyway.") {
					return "Why does going ahead matter more to you than the concern?", nil
				}
				return "What would change for her if you went ahead?", nil
			}
			return "Moly reply.", nil
		},
	})
	convID := startNamedGoal(t, srv, token)
	first, _, _ := sendMessage(t, srv, token, convID, "Her name is Christine.")
	second, _, _ := sendMessage(t, srv, token, convID, "I will do it anyway.")
	t.Logf("first=%q\nsecond=%q", first, second)
	if second == first || strings.Count(second, "?") != 1 {
		t.Fatalf("pressing on after a concern gets one deeper question, got %q", second)
	}
}

// S18: a clear message has no gaps. The model returns none, or a gap that names no assumption (the model filling
// space); either way nothing blocks help and no question is asked.
func TestScenarioS18ClearMessageIsNotQuestioned(t *testing.T) {
	for name, gaps := range map[string]string{
		"no gaps":                `[]`,
		"a gap without a reason": `[{"type": "tone", "description": "How formal should it be?", "severity": "high", "confidence": 0.9, "goalTarget": "current_goal"}]`,
	} {
		t.Run(name, func(t *testing.T) {
			srv, _, _, token := scenarioServer(t, scriptedLLM{gaps: func() (string, error) { return gaps, nil }})
			convID := startNamedGoal(t, srv, token)
			reply, _, code := sendMessage(t, srv, token, convID, "Her name is Christine.")
			if code != http.StatusOK || strings.Contains(reply, "?") {
				t.Fatalf("status %d, nothing blocks help but the reply asks: %q", code, reply)
			}
		})
	}
}

// S19: a person the model only guessed is not saved. The reply is one question that asks the user to confirm.
func TestScenarioS19GuessedPersonIsConfirmedNotSaved(t *testing.T) {
	srv, db, userID, token := scenarioServer(t, scriptedLLM{})
	reply, _, code := sendMessage(t, srv, token, "", "Maybe I should write to Anna.")
	if code != http.StatusOK || strings.Count(reply, "?") != 1 {
		t.Fatalf("want one confirming question, status %d reply %q", code, reply)
	}
	if got := contactsFor(t, db, userID); len(got) != 0 {
		t.Fatalf("a guessed person must not be saved, got %+v", got)
	}
}

// S20: a goal the model only guessed is not locked. The reply is one question that asks the user to confirm.
func TestScenarioS20GuessedGoalIsConfirmedNotLocked(t *testing.T) {
	srv, _, userID, token := scenarioServer(t, scriptedLLM{})
	reply, convID, code := sendMessage(t, srv, token, "", "Perhaps I need to say something.")
	if code != http.StatusOK || strings.Count(reply, "?") != 1 {
		t.Fatalf("want one confirming question, status %d reply %q", code, reply)
	}
	if g := lockedGoal(srv, userID, convID); g != "" {
		t.Fatalf("a guessed goal must not be locked, got %q", g)
	}
}

// S21: a different goal is not taken over. The user is asked once; a clear yes replaces the lock, and not before.
func TestScenarioS21GoalSwitchNeedsConfirmation(t *testing.T) {
	srv, _, userID, token := scenarioServer(t, scriptedLLM{})
	convID := startNamedGoal(t, srv, token)
	const first = "initiate a conversation with the girl on fetlife"
	const second = "decide whether to tell her about my interests"
	sendMessage(t, srv, token, convID, "Her name is Christine.")

	ask, _, code := sendMessage(t, srv, token, convID, "Now I need to decide whether to tell her about my interests.")
	if code != http.StatusOK || strings.Count(ask, "?") != 1 {
		t.Fatalf("a different goal gets one confirming question, status %d reply %q", code, ask)
	}
	if g := lockedGoal(srv, userID, convID); g != first {
		t.Fatalf("the lock must not change before the user agrees, got %q", g)
	}

	sendMessage(t, srv, token, convID, "Yes, let's do that.")
	if g := lockedGoal(srv, userID, convID); g != second {
		t.Fatalf("after a clear yes the lock is the new goal %q, got %q", second, g)
	}
}

// S22: gap maturity follows the questions across messages. With one open gap it is 0; when the user answers it and
// nothing is left open it is 1, and Moly then helps instead of asking again.
func TestScenarioS22GapMaturityRisesWhenGapIsAnswered(t *testing.T) {
	const gapQ = "How formal should the first message be?"
	var answeredIt atomic.Bool
	srv, _, _, token := scenarioServer(t, scriptedLLM{
		reply: func(prompt string) (string, error) {
			if strings.Contains(prompt, asking+gapQ) {
				return gapQ, nil
			}
			return "Moly reply.", nil
		},
		gaps: func() (string, error) {
			if answeredIt.Load() {
				return `[]`, nil
			}
			return fmt.Sprintf(`[{"type": "tone", "description": %q, "assumption": "how formal to be", "severity": "high", "confidence": 0.9, "goalTarget": "current_goal"}]`, gapQ), nil
		},
	})
	convID := startNamedGoal(t, srv, token)
	ask, _, _ := sendMessage(t, srv, token, convID, "Her name is Christine.")
	if !strings.Contains(ask, gapQ) {
		t.Fatalf("the open gap must be asked, got %q", ask)
	}
	repo := srv.database.GetClarificationQuestionRepository()
	before := agents.ConversationGapMaturity(repo, convID)
	if before != 0 {
		t.Fatalf("with one gap open and none answered maturity is 0, got %v", before)
	}

	answeredIt.Store(true)
	reply, _, code := sendMessage(t, srv, token, convID, "Quite informal, she seems relaxed.")
	after := agents.ConversationGapMaturity(repo, convID)
	t.Logf("maturity %v -> %v; reply %q", before, after, reply)
	if code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	if after <= before {
		t.Fatalf("answering the gap must raise maturity: %v -> %v", before, after)
	}
	if strings.Contains(reply, "?") {
		t.Fatalf("goal clear, nothing open: Moly helps instead of asking, got %q", reply)
	}
}

// S23: confidence follows the conversation. A person the model only guessed is held (0.4, under the 0.6 doubt line)
// and confirmed with one question; when the user then states the person, it is stated and grounded, so it is saved.
func TestScenarioS23ConfirmedPersonIsSaved(t *testing.T) {
	srv, db, userID, token := scenarioServer(t, scriptedLLM{})
	reply, convID, _ := sendMessage(t, srv, token, "", "Maybe I should write to Anna.")
	if strings.Count(reply, "?") != 1 || len(contactsFor(t, db, userID)) != 0 {
		t.Fatalf("a guess is confirmed, not saved: reply %q, contacts %+v", reply, contactsFor(t, db, userID))
	}
	_, _, code := sendMessage(t, srv, token, convID, "Yes, I mean my colleague Anna.")
	got := contactsFor(t, db, userID)
	if code != http.StatusOK || len(got) != 1 {
		t.Fatalf("a stated person is saved once confirmed: status %d, contacts %+v", code, got)
	}
}

// S24: the skip button. One gap was answered (maturity above 0), a second is open, and the user presses skip. With no
// risk, Moly gives the result and asks nothing. The open gap is skipped, not answered.
func TestScenarioS24SkipTakesTheResultNow(t *testing.T) {
	const gapA = "How formal should the first message be?"
	const gapB = "What does she like to talk about?"
	var stage atomic.Int32
	srv, _, _, token := scenarioServer(t, scriptedLLM{
		reply: func(prompt string) (string, error) {
			switch {
			case strings.Contains(prompt, `pressed "go ahead"`) && strings.Contains(prompt, "a first message to a girl I saw on fetlife"):
				return "Here is your message.", nil // the result needs the request, which the press itself does not carry
			case strings.Contains(prompt, asking+gapB):
				return gapB, nil
			case strings.Contains(prompt, asking+gapA):
				return gapA, nil
			}
			return "Moly reply.", nil
		},
		gaps: func() (string, error) {
			q := gapA
			if stage.Load() >= 1 {
				q = gapB
			}
			return fmt.Sprintf(`[{"type": "gap%d", "description": %q, "assumption": "something to assume", "severity": "high", "confidence": 0.9, "goalTarget": "current_goal"}]`, stage.Load(), q), nil
		},
	})
	convID := startNamedGoal(t, srv, token)
	ask, _, _, meta := sendFull(t, srv, token, convID, "Her name is Christine.", false)
	if ask != gapA || meta["canSkip"] != false {
		t.Fatalf("the first gap is asked and nothing is answered yet, so no skip button: %q canSkip=%v", ask, meta["canSkip"])
	}
	stage.Store(1)
	ask, _, _, meta = sendFull(t, srv, token, convID, "Quite informal, she seems relaxed.", false)
	if ask != gapB || meta["canSkip"] != true {
		t.Fatalf("after one answer the second gap carries the skip button: %q canSkip=%v", ask, meta["canSkip"])
	}
	repo := srv.database.GetClarificationQuestionRepository()
	if got := agents.ConversationGapMaturity(repo, convID); got != 0.5 {
		t.Fatalf("one answered, one open: maturity 0.5, got %v", got)
	}
	reply, code := sendSkip(t, srv, token, convID)
	if code != http.StatusOK || reply != "Here is your message." {
		t.Fatalf("skip takes the result: status %d reply %q", code, reply)
	}
	if got := agents.ConversationGapMaturity(repo, convID); got != 0.5 {
		t.Fatalf("the skipped gap is pending, still a gap: maturity 0.5 (one answered, one pending), got %v", got)
	}
}

// S25: skip never removes risk. The skip is open (a skippable question carries the button); then a principle concern is
// engaged when the press arrives. The press is not enough: Moly still asks, and does not give the result.
func TestScenarioS25SkipDoesNotRemoveRisk(t *testing.T) {
	const gapA = "How formal should the first message be?"
	const gapB = "What does she like to talk about?"
	var stage atomic.Int32
	srv, _, _, token := scenarioServer(t, scriptedLLM{
		reply: func(prompt string) (string, error) {
			switch {
			case strings.Contains(prompt, `pressed "go ahead"`):
				return "Here is your message.", nil
			case strings.Contains(prompt, asking+gapB):
				return gapB, nil
			case strings.Contains(prompt, asking+gapA):
				return gapA, nil
			}
			return "Moly reply.", nil
		},
		gaps: func() (string, error) {
			q := gapA
			if stage.Load() >= 1 {
				q = gapB
			}
			return fmt.Sprintf(`[{"type": "g%d", "description": %q, "assumption": "x", "severity": "high", "confidence": 0.9, "goalTarget": "current_goal"}]`, stage.Load(), q), nil
		},
		principles: func() (string, error) {
			if stage.Load() >= 2 {
				return stakeholderConcern, nil // the concern is engaged when the press arrives
			}
			return "{}", nil
		},
	})
	convID := startNamedGoal(t, srv, token)
	sendMessage(t, srv, token, convID, "Her name is Christine.")
	stage.Store(1)
	if ask, _, _, meta := sendFull(t, srv, token, convID, "Quite informal, she seems relaxed.", false); ask != gapB || meta["canSkip"] != true {
		t.Fatalf("setup: the skip is open, got %q canSkip=%v", ask, meta["canSkip"])
	}
	stage.Store(2)
	reply, code := sendSkip(t, srv, token, convID)
	t.Logf("reply=%q", reply)
	if code != http.StatusOK || reply == "Here is your message." || strings.Count(reply, "?") != 1 {
		t.Fatalf("a principle concern is risk: Moly asks one question, got status %d reply %q", code, reply)
	}
}

// S26: a press at maturity 0 is ignored (a stale or forged button). Moly goes on with its question.
func TestScenarioS26SkipIgnoredAtMaturityZero(t *testing.T) {
	const gapQ = "How formal should the first message be?"
	srv, _, _, token := scenarioServer(t, scriptedLLM{
		reply: func(prompt string) (string, error) {
			if strings.Contains(prompt, "pressed \"go ahead\"") {
				return "Here is your message.", nil
			}
			if strings.Contains(prompt, asking+gapQ) {
				return gapQ, nil
			}
			return "Moly reply.", nil
		},
		gaps: func() (string, error) {
			return fmt.Sprintf(`[{"type": "tone", "description": %q, "assumption": "how formal to be", "severity": "high", "confidence": 0.9, "goalTarget": "current_goal"}]`, gapQ), nil
		},
	})
	convID := startNamedGoal(t, srv, token)
	sendMessage(t, srv, token, convID, "Her name is Christine.")
	reply, code := sendSkip(t, srv, token, convID)
	if code != http.StatusOK || reply == "Here is your message." {
		t.Fatalf("at maturity 0 the press is ignored, got status %d reply %q", code, reply)
	}
	if got := agents.ConversationGapMaturity(srv.database.GetClarificationQuestionRepository(), convID); got != 1 {
		t.Logf("maturity %v (the press counts as the reply it is)", got)
	}
}

// S27: a question about risk carries no skip button, however much has been answered.
func TestScenarioS27NoButtonUnderARiskQuestion(t *testing.T) {
	const gapQ = "How formal should the first message be?"
	var stage atomic.Int32
	srv, _, _, token := scenarioServer(t, scriptedLLM{
		reply: func(prompt string) (string, error) {
			if strings.Contains(prompt, asking+gapQ) {
				return gapQ, nil
			}
			return "Moly reply.", nil
		},
		gaps: func() (string, error) {
			if stage.Load() >= 1 {
				return `[]`, nil
			}
			return fmt.Sprintf(`[{"type": "tone", "description": %q, "assumption": "how formal to be", "severity": "high", "confidence": 0.9, "goalTarget": "current_goal"}]`, gapQ), nil
		},
		principles: func() (string, error) {
			if stage.Load() >= 1 {
				return stakeholderConcern, nil
			}
			return "{}", nil
		},
	})
	convID := startNamedGoal(t, srv, token)
	sendMessage(t, srv, token, convID, "Her name is Christine.")
	stage.Store(1)
	reply, _, _, meta := sendFull(t, srv, token, convID, "Quite informal, she seems relaxed.", false)
	t.Logf("reply=%q meta=%v", reply, meta["canSkip"])
	if meta["canSkip"] != false {
		t.Fatalf("a principle question carries no skip button, got %v (reply %q)", meta["canSkip"], reply)
	}
}

// pendingScenario runs the first part of the pending-question scenarios: one gap answered, a second gap asked and then
// skipped. It returns the server, the token, the conversation and the stage counter that drives the scripted gaps.
func pendingScenario(t *testing.T, extra func(stage int32) string) (*APIServer, string, string, *atomic.Int32) {
	t.Helper()
	const gapA = "How formal should the first message be?"
	const gapB = "What does she like to talk about?"
	var stage atomic.Int32
	gapJSON := func(sev, desc string) string {
		return fmt.Sprintf(`{"type": "t", "description": %q, "assumption": "x", "severity": %q, "confidence": 0.9, "goalTarget": "current_goal"}`, desc, sev)
	}
	srv, _, _, token := scenarioServer(t, scriptedLLM{
		reply: func(prompt string) (string, error) {
			switch {
			case strings.Contains(prompt, "pressed \"go ahead\""):
				return "Here is your message.", nil
			}
			for _, q := range []string{"Do you want to mention where you saw her?", gapB, gapA} {
				if strings.Contains(prompt, asking+q) {
					return q, nil
				}
			}
			return "Moly reply.", nil
		},
		gaps: func() (string, error) {
			switch stage.Load() {
			case 0:
				return "[" + gapJSON("high", gapA) + "]", nil
			case 1:
				return "[" + gapJSON("high", gapB) + "]", nil
			case 2: // after the skip the detector produces nothing new
				return "[]", nil
			}
			return "[" + extra(stage.Load()) + "]", nil
		},
	})
	convID := startNamedGoal(t, srv, token)
	sendMessage(t, srv, token, convID, "Her name is Christine.")
	stage.Store(1)
	if ask, _, _ := sendMessage(t, srv, token, convID, "Quite informal, she seems relaxed."); ask != gapB {
		t.Fatalf("setup: the second gap is asked, got %q", ask)
	}
	if reply, _ := sendSkip(t, srv, token, convID); reply != "Here is your message." {
		t.Fatalf("setup: skip gives the result, got %q", reply)
	}
	stage.Store(2)
	return srv, token, convID, &stage
}

// S28: a skipped question is saved as pending. The user goes on ("it is good but too formal"), the new message does not
// answer it, so it is a candidate again and, with nothing more important, it is asked. Asked again, the same row becomes
// active (no duplicate).
func TestScenarioS28SkippedQuestionIsPendingAndReturns(t *testing.T) {
	const gapB = "What does she like to talk about?"
	srv, token, convID, _ := pendingScenario(t, func(int32) string { return "" })
	repo := srv.database.GetClarificationQuestionRepository()
	var pending int
	srv.database.GetConnection().QueryRow(`SELECT COUNT(*) FROM clarification_questions WHERE conversation_id = ? AND status = 'skipped'`, convID).Scan(&pending)
	if pending != 1 {
		t.Fatalf("the skipped question is saved as pending, got %d", pending)
	}
	reply, _, _ := sendMessage(t, srv, token, convID, "I also saw her post about hiking.")
	if reply != gapB {
		t.Fatalf("the pending question is evaluated again and asked, got %q", reply)
	}
	var rows, active int
	srv.database.GetConnection().QueryRow(`SELECT COUNT(*) FROM clarification_questions WHERE conversation_id = ? AND question_text = ?`, convID, gapB).Scan(&rows)
	srv.database.GetConnection().QueryRow(`SELECT COUNT(*) FROM clarification_questions WHERE conversation_id = ? AND question_text = ? AND status = 'active'`, convID, gapB).Scan(&active)
	if rows != 1 || active != 1 {
		t.Fatalf("the same row is active again (rows %d, active %d)", rows, active)
	}
	if got := agents.ConversationGapMaturity(repo, convID); got != 0.5 {
		t.Fatalf("one answered, one open: 0.5, got %v", got)
	}
}

// S29: a pending question that the new message answers is closed as answered, and maturity rises.
func TestScenarioS29MessageThatAnswersAPendingQuestionClosesIt(t *testing.T) {
	srv, token, convID, _ := pendingScenario(t, func(int32) string { return "" })
	reply, _, _ := sendMessage(t, srv, token, convID, "ANSWERS-PENDING she likes hiking and old films.")
	if strings.Contains(reply, "?") {
		t.Fatalf("nothing is open any more, got %q", reply)
	}
	var skipped, answered int
	conn := srv.database.GetConnection()
	conn.QueryRow(`SELECT COUNT(*) FROM clarification_questions WHERE conversation_id = ? AND status = 'skipped'`, convID).Scan(&skipped)
	conn.QueryRow(`SELECT COUNT(*) FROM clarification_questions WHERE conversation_id = ? AND status = 'answered'`, convID).Scan(&answered)
	if skipped != 0 || answered != 2 {
		t.Fatalf("both questions are answered now (skipped %d, answered %d)", skipped, answered)
	}
	if got := agents.ConversationGapMaturity(srv.database.GetClarificationQuestionRepository(), convID); got != 1 {
		t.Fatalf("all answered: maturity 1, got %v", got)
	}
}

// S30: a pending question and a new gap are evaluated together. The one that blocks the goal more goes first.
func TestScenarioS30PendingAndNewGapsAreRankedTogether(t *testing.T) {
	const gapB = "What does she like to talk about?"
	const gapC = "Do you want to mention where you saw her?"
	// the pending question was high; a new medium gap does not outrank it
	srv, token, convID, stage := pendingScenario(t, func(int32) string {
		return fmt.Sprintf(`{"type": "t", "description": %q, "assumption": "x", "severity": "medium", "confidence": 0.9, "goalTarget": "current_goal"}`, gapC)
	})
	stage.Store(3) // the detector now produces a new, less important gap
	reply, _, _ := sendMessage(t, srv, token, convID, "Make it a bit shorter too.")
	if reply != gapB {
		t.Fatalf("a high pending question comes before a new medium gap, got %q", reply)
	}
}

// S31: a goal the user leaves takes its pending questions with it: they are cancelled, and not asked for the new goal.
func TestScenarioS31GoalSwitchDropsPendingQuestions(t *testing.T) {
	const gapB = "What does she like to talk about?"
	srv, token, convID, _ := pendingScenario(t, func(int32) string { return "" })
	sendMessage(t, srv, token, convID, "Now I need to decide whether to tell her about my interests.")
	reply, _, _ := sendMessage(t, srv, token, convID, "Yes, let's do that.")
	var cancelled int
	srv.database.GetConnection().QueryRow(`SELECT COUNT(*) FROM clarification_questions WHERE conversation_id = ? AND question_text = ? AND status = 'cancelled'`, convID, gapB).Scan(&cancelled)
	if cancelled != 1 || reply == gapB {
		t.Fatalf("the pending question of the old goal is cancelled and not asked (cancelled %d, reply %q)", cancelled, reply)
	}
}

// S32: words that do not refer to a person are not saved as one, so Moly never asks for the name of "the a bit too formal
// you mentioned" (found live). A real unnamed person is still saved.
func TestScenarioS32WordsThatAreNotAPersonAreNotSaved(t *testing.T) {
	srv, db, userID, token := scenarioServer(t, scriptedLLM{})
	_, convID, _ := sendMessage(t, srv, token, "", "Hi Moly")
	reply, _, code := sendMessage(t, srv, token, convID, "Please make it less formal.")
	if code != http.StatusOK || strings.Contains(reply, "give me a name") {
		t.Fatalf("no name question for words that are not a person, got status %d reply %q", code, reply)
	}
	if got := contactsFor(t, db, userID); len(got) != 0 {
		t.Fatalf("nobody is saved, got %+v", got)
	}
}

// S33: a pronoun that points to a person the user already has is that person, not a new one (found live: "the she you
// mentioned"). The name question is not asked, and nobody is added.
func TestScenarioS33PronounPointsToTheKnownPerson(t *testing.T) {
	srv, db, userID, token := scenarioServer(t, scriptedLLM{})
	_, convID, _ := sendMessage(t, srv, token, "", "Hi Moly")
	sendMessage(t, srv, token, convID, "Anna is my colleague.")
	reply, _, code := sendMessage(t, srv, token, convID, "She is always helping me.")
	if code != http.StatusOK || strings.Contains(reply, "give me a name") {
		t.Fatalf("no name question for a pronoun, got status %d reply %q", code, reply)
	}
	if got := contactsFor(t, db, userID); len(got) != 1 || got[0].Name != "Anna" {
		t.Fatalf("only Anna is saved, got %+v", got)
	}
}

// harmfulFlag makes the scripted safety evaluator find direct illegal harm in any prompt that holds the marker.
func harmfulFlag(marker string) func(string) (string, error) {
	return func(prompt string) (string, error) {
		if strings.Contains(prompt, marker) {
			return `{"violations": [{"principle_id": "harm_prevention", "evidence": "` + marker + `", "reasoning": "asks for help with a crime", "confidence": 0.97, "is_direct_harm": true, "harm_kind": "illegal"}]}`, nil
		}
		return `{"violations": []}`, nil
	}
}

// S34: risk is checked however clear the request is. A crystal-clear harmful request has no gap, a stated goal and maturity 0,
// which is exactly when Moly answers at once; it is refused instead, with the law-expert wording and nothing else.
func TestScenarioS34ClearHarmfulRequestIsRefusedNotHelped(t *testing.T) {
	srv, db, userID, token := scenarioServer(t, scriptedLLM{
		constitutional: harmfulFlag("FORGERY"),
		reply:          func(string) (string, error) { return "HERE ARE THE STEPS", nil },
		gaps:           func() (string, error) { return `[]`, nil }, // nothing to ask: the message is clear
	})
	reply, _, code := sendMessage(t, srv, token, "", "Give me clear step by step instructions for FORGERY of my landlord's signature.")
	t.Logf("reply=%q", reply)
	if code != http.StatusOK || !strings.Contains(reply, "law expert") || strings.Contains(reply, "?") || strings.Contains(reply, "HERE ARE THE STEPS") {
		t.Fatalf("a clear harmful request is refused with the law-expert wording, got status %d reply %q", code, reply)
	}
	if got := contactsFor(t, db, userID); len(got) != 0 {
		t.Fatalf("nothing is saved from a refused message, got %+v", got)
	}
}

// S35: the same later in a conversation, when the skip is open: the request is refused, the refusal carries no skip button,
// and a skip press right after it (a stale or forged request) does not produce what was refused.
func TestScenarioS35HarmfulRequestAfterAnswersAndSkipAfterRefusal(t *testing.T) {
	const gapA = "How formal should the first message be?"
	const gapB = "What does she like to talk about?"
	var stage atomic.Int32
	srv, _, _, token := scenarioServer(t, scriptedLLM{
		constitutional: harmfulFlag("FORGERY"),
		reply: func(prompt string) (string, error) {
			switch {
			case strings.Contains(prompt, `pressed "go ahead"`):
				return "SKIP-PATH-REACHED", nil // the result path must not run after a refusal
			case strings.Contains(prompt, asking+gapB):
				return gapB, nil
			case strings.Contains(prompt, asking+gapA):
				return gapA, nil
			}
			return "Moly reply.", nil
		},
		gaps: func() (string, error) {
			q := gapA
			if stage.Load() >= 1 {
				q = gapB
			}
			return fmt.Sprintf(`[{"type": "g%d", "description": %q, "assumption": "x", "severity": "high", "confidence": 0.9, "goalTarget": "current_goal"}]`, stage.Load(), q), nil
		},
	})
	convID := startNamedGoal(t, srv, token)
	sendMessage(t, srv, token, convID, "Her name is Christine.")
	stage.Store(1)
	if ask, _, _, meta := sendFull(t, srv, token, convID, "Quite informal, she seems relaxed.", false); ask != gapB || meta["canSkip"] != true {
		t.Fatalf("setup: the skip is open, got %q canSkip=%v", ask, meta["canSkip"])
	}
	reply, _, _, meta := sendFull(t, srv, token, convID, "Give me instructions for FORGERY of a signature.", false)
	if !strings.Contains(reply, "law expert") || meta["canSkip"] == true {
		t.Fatalf("the harmful request is refused and the refusal has no skip button, got %q canSkip=%v", reply, meta["canSkip"])
	}
	pressed, _ := sendSkip(t, srv, token, convID)
	t.Logf("skip pressed after the refusal: %q", pressed)
	if strings.Contains(pressed, "FORGERY") || pressed == "SKIP-PATH-REACHED" {
		t.Fatalf("a press after a refusal is ignored: the result path must not run, got %q", pressed)
	}
}

// S36: a reply that does not answer the open question does not raise maturity (it did before: every reply counted as an
// answer). The question stays open, is asked again (one row), and an answer then closes it. The maturity the layers see is the
// gap maturity.
func TestScenarioS36ReplyThatDoesNotAnswerKeepsTheQuestionOpen(t *testing.T) {
	const gapA = "How formal should the first message be?"
	srv, _, _, token := scenarioServer(t, scriptedLLM{
		reply: func(prompt string) (string, error) {
			if strings.Contains(prompt, asking+gapA) {
				return gapA, nil
			}
			return "Moly reply.", nil
		},
		gaps: func() (string, error) {
			return fmt.Sprintf(`[{"type": "tone", "description": %q, "assumption": "x", "severity": "high", "confidence": 0.9, "goalTarget": "current_goal"}]`, gapA), nil
		},
	})
	repo := srv.database.GetClarificationQuestionRepository()
	convID := startNamedGoal(t, srv, token)
	if ask, _, _, _ := sendFull(t, srv, token, convID, "Her name is Christine.", false); ask != gapA {
		t.Fatalf("setup: the gap is asked, got %q", ask)
	}
	ask, _, _, meta := sendFull(t, srv, token, convID, "My sister's wedding is next month.", false)
	if ask != gapA {
		t.Fatalf("the question was not answered, so it is asked again, got %q", ask)
	}
	var rows, answered int
	conn := srv.database.GetConnection()
	conn.QueryRow(`SELECT COUNT(*) FROM clarification_questions WHERE conversation_id = ? AND question_text = ?`, convID, gapA).Scan(&rows)
	conn.QueryRow(`SELECT COUNT(*) FROM clarification_questions WHERE conversation_id = ? AND status = 'answered'`, convID).Scan(&answered)
	if rows != 1 || answered != 0 {
		t.Fatalf("one open row, nothing answered (rows %d, answered %d)", rows, answered)
	}
	if got := agents.ConversationGapMaturity(repo, convID); got != 0 {
		t.Fatalf("an unrelated reply does not raise maturity, got %v", got)
	}
	if meta["maturityScore"] != float64(0) {
		t.Fatalf("the maturity the layers see is the gap maturity (0), got %v", meta["maturityScore"])
	}

	reply, _, _, meta := sendFull(t, srv, token, convID, "Quite informal, she seems relaxed.", false)
	if strings.Contains(reply, "?") {
		t.Fatalf("answered and nothing else open: Moly helps, got %q", reply)
	}
	if got := agents.ConversationGapMaturity(repo, convID); got != 1 {
		t.Fatalf("an answer closes the question: maturity 1, got %v", got)
	}
	if meta["maturityScore"] != float64(1) {
		t.Fatalf("the maturity the layers see is the gap maturity (1), got %v", meta["maturityScore"])
	}
}

// S37: a goal needs a quote. The model names a goal but the words it quotes are not in the message: the goal is held and
// confirmed, not locked (found live: goals invented from messages that state none).
func TestScenarioS37GoalWithoutAGroundedQuoteIsNotLocked(t *testing.T) {
	srv, _, userID, token := scenarioServer(t, scriptedLLM{})
	reply, convID, code := sendMessage(t, srv, token, "", "ZQXV unrelated words about the weather.")
	if code != http.StatusOK || strings.Count(reply, "?") != 1 {
		t.Fatalf("an ungrounded goal is confirmed with one question, got status %d reply %q", code, reply)
	}
	if g := lockedGoal(srv, userID, convID); g != "" {
		t.Fatalf("an ungrounded goal must not be locked, got %q", g)
	}
}

// S38: a goal worded differently in a later message is not an intention "conflict" to queue. The model judges how goals relate
// (S11, S31) and a real switch is confirmed with the user; a text difference must not queue a conflict, which also kept
// the skip button off (found live, 2026-10-10).
func TestScenarioS38DifferentGoalWordingQueuesNoIntentionConflict(t *testing.T) {
	srv, _, _, token := scenarioServer(t, scriptedLLM{})
	convID := startNamedGoal(t, srv, token)
	sendMessage(t, srv, token, convID, "Now I need to decide whether to tell her about my interests.")
	var n int
	if err := srv.database.GetConnection().QueryRow(`SELECT COUNT(*) FROM context_conflicts WHERE conflict_type = 'intention'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("no intention conflict is queued by a text difference, got %d", n)
	}
}

// S39: a bare "Yes" cannot be read alone. Every model call that carries the user's message must also carry Moly's last
// message (the question the "Yes" answers). The test finds, among the prompts of the turn, those that quote the reply and
// lack the question.
func TestScenarioS39EveryCallReadingAShortReplyCarriesTheQuestion(t *testing.T) {
	const gapQ = "How formal should the first message be?"
	var seen []string
	srv, _, _, token := scenarioServer(t, scriptedLLM{
		seen: &seen,
		reply: func(prompt string) (string, error) {
			if strings.Contains(prompt, asking+gapQ) {
				return gapQ, nil
			}
			return "Moly reply.", nil
		},
		gaps: func() (string, error) {
			return fmt.Sprintf(`[{"type": "tone", "description": %q, "assumption": "how formal to be", "severity": "high", "confidence": 0.9, "goalTarget": "current_goal"}]`, gapQ), nil
		},
	})
	convID := startNamedGoal(t, srv, token)
	if ask, _, _ := sendMessage(t, srv, token, convID, "Her name is Christine."); ask != gapQ {
		t.Fatalf("setup: Moly asks the gap question, got %q", ask)
	}
	before := len(seen)
	sendMessage(t, srv, token, convID, "Yes")
	var offenders []string
	for _, p := range seen[before:] {
		if regexp.MustCompile(`\bYes\b`).MatchString(p) && !strings.Contains(p, gapQ) {
			head := p
			if len(head) > 110 {
				head = head[:110]
			}
			offenders = append(offenders, strings.ReplaceAll(head, "\n", " "))
		}
	}
	if len(offenders) > 0 {
		t.Fatalf("%d model call(s) read the reply \"Yes\" without Moly's question:\n  %s", len(offenders), strings.Join(offenders, "\n  "))
	}
}

// S40: the reply model sees the dialogue. A comment on Moly's draft ("too formal") is answered with the draft in view, the
// goal, and the person with what they are to the user. Before, the reply prompt held only "you have prior conversation
// history" and none of it.
func TestScenarioS40ReplyModelSeesTheDraftTheGoalAndThePerson(t *testing.T) {
	var seen []string
	srv, _, _, token := scenarioServer(t, scriptedLLM{
		seen: &seen,
		reply: func(prompt string) (string, error) {
			if strings.Contains(prompt, `They just said: "Her name is Christine."`) {
				return "DRAFT-ALPHA Dear Christine, I would be delighted to meet.", nil
			}
			return "Moly reply.", nil
		},
		gaps: func() (string, error) { return "[]", nil },
	})
	convID := startNamedGoal(t, srv, token)
	if draft, _, _ := sendMessage(t, srv, token, convID, "Her name is Christine."); !strings.Contains(draft, "DRAFT-ALPHA") {
		t.Fatalf("setup: Moly writes the draft, got %q", draft)
	}
	before := len(seen)
	sendMessage(t, srv, token, convID, "It is good but a bit too formal.")
	var replyPrompt string
	for _, p := range seen[before:] {
		if strings.Contains(p, `They just said: "It is good but a bit too formal."`) {
			replyPrompt = p
		}
	}
	if replyPrompt == "" {
		t.Fatal("the reply prompt was not found")
	}
	for _, want := range []string{"DRAFT-ALPHA Dear Christine", "initiate a conversation with the girl on fetlife", "Christine", "conversation between the user and Moly"} {
		if !strings.Contains(replyPrompt, want) {
			t.Fatalf("the reply model must see %q:\n%.1500s", want, replyPrompt)
		}
	}
}

// summaryOf reads the stored summary text of a conversation.
func summaryOf(srv *APIServer, userID, convID string) string {
	var arc string
	srv.database.GetConnection().QueryRow(`SELECT arc FROM conversation_summaries WHERE user_id = ? AND conversation_id = ?`, userID, convID).Scan(&arc)
	return arc
}

func waitForSummary(t *testing.T, srv *APIServer, userID, convID, want string) {
	t.Helper()
	for i := 0; i < 60; i++ {
		if strings.Contains(summaryOf(srv, userID, convID), want) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("the summary never took in %q, it holds %q", want, summaryOf(srv, userID, convID))
}

// S41: every new user message updates the summary after its analysis and reply; the next prompt carries it, ahead of Moly's
// last message. A bare greeting and a refused message are not remembered.
func TestScenarioS41EveryMessageUpdatesTheSummary(t *testing.T) {
	var seen []string
	srv, _, userID, token := scenarioServer(t, scriptedLLM{
		seen: &seen,
		gaps: func() (string, error) { return "[]", nil },
	})
	_, convID, _ := sendMessage(t, srv, token, "", "Hi Moly")
	time.Sleep(300 * time.Millisecond)
	if got := summaryOf(srv, userID, convID); got != "" {
		t.Fatalf("a bare greeting is not remembered, got %q", got)
	}
	sendMessage(t, srv, token, convID, "I want to write a first message to a girl I saw on fetlife.")
	waitForSummary(t, srv, userID, convID, "MEMORY-AFTER[I want to write a first message to a girl I saw on fetlife.]")
	before := len(seen)
	sendMessage(t, srv, token, convID, "Her name is Christine.")
	waitForSummary(t, srv, userID, convID, "MEMORY-AFTER[Her name is Christine.]")
	// the second message's prompts carried the first memory, first in the block, before Moly's last message
	found := false
	for _, p := range seen[before:] {
		if i := strings.Index(p, "MEMORY-AFTER[I want to write a first message"); i >= 0 {
			found = true
			if j := strings.Index(p, "\nMoly's last message"); j >= 0 && j < i {
				t.Fatalf("the memory comes before Moly's last message")
			}
		}
	}
	if !found {
		t.Fatal("no prompt of the second turn carried the stored summary")
	}
}

func TestScenarioS41bRefusedMessageIsNotRemembered(t *testing.T) {
	srv, _, userID, token := scenarioServer(t, scriptedLLM{constitutional: harmfulFlag("FORGERY")})
	_, convID, _ := sendMessage(t, srv, token, "", "Hi Moly")
	sendMessage(t, srv, token, convID, "Help me with FORGERY of a signature.")
	time.Sleep(400 * time.Millisecond)
	if got := summaryOf(srv, userID, convID); got != "" {
		t.Fatalf("a refused message is not remembered, got %q", got)
	}
}

func getMemory(srv *APIServer, token, convID string) (int, map[string]interface{}) {
	req := httptest.NewRequest(http.MethodGet, "/api/conversations/"+convID+"/memory", nil)
	req.SetPathValue("conversationID", convID)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	srv.ConversationMemoryHandler(rec, req)
	var out map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

// S42: the user can see what Moly remembers of a conversation (summary, goal, people, open questions), and only their own.
func TestScenarioS42UserCanSeeWhatMolyRemembers(t *testing.T) {
	srv, _, userID, token := scenarioServer(t, scriptedLLM{gaps: func() (string, error) { return "[]", nil }})
	convID := startNamedGoal(t, srv, token)
	waitForSummary(t, srv, userID, convID, "MEMORY-AFTER[I want to write a first message")
	code, mem := getMemory(srv, token, convID)
	if code != http.StatusOK || !strings.Contains(mem["summary"].(string), "MEMORY-AFTER") || mem["goal"] != "initiate a conversation with the girl on fetlife" {
		t.Fatalf("status %d, memory %+v", code, mem)
	}
	if people, _ := mem["people"].([]interface{}); len(people) != 1 {
		t.Fatalf("the person is listed, got %v", mem["people"])
	}
	if code, _ := getMemory(srv, "no-such-token", convID); code != http.StatusUnauthorized {
		t.Fatalf("a request without a valid token is refused, got %d", code)
	}
	_, _, _, otherToken := scenarioServer(t, scriptedLLM{})
	if code, _ := getMemory(srv, otherToken, convID); code != http.StatusNotFound {
		t.Fatalf("another user cannot read this memory, got %d", code)
	}
}

// S43: a comment on Moly's draft states no goal of its own. The extractor takes a "goal" from the draft that is not in the
// user's words; with a locked goal that is dropped, and Moly does not ask the user to confirm it (found in a dynamic live run:
// "a bit too formal" was answered with "did you mean: request a meeting...?").
func TestScenarioS43CommentOnTheDraftIsNotAskedToConfirmAGoal(t *testing.T) {
	srv, _, userID, token := scenarioServer(t, scriptedLLM{gaps: func() (string, error) { return "[]", nil }})
	convID := startNamedGoal(t, srv, token)
	sendMessage(t, srv, token, convID, "Her name is Christine.")
	reply, _, _ := sendMessage(t, srv, token, convID, "ZQXV unrelated words, a bit too formal.")
	if strings.Contains(strings.ToLower(reply), "did you mean") || strings.Contains(strings.ToLower(reply), "to be sure") {
		t.Fatalf("no confirmation question about a goal the user did not state, got %q", reply)
	}
	if g := lockedGoal(srv, userID, convID); g != "initiate a conversation with the girl on fetlife" {
		t.Fatalf("the locked goal continues, got %q", g)
	}
}

// S44: a complete request to a named person is answered at maturity 0 (the user's decision "option 2"). The old rule asked a
// fixed question ("You mentioned Anna... What is the situation you are dealing with?") whenever a message was addressed to a named
// person, found in a dynamic run. Whether anything is missing is left to the model that finds gaps.
func TestScenarioS44ClearRequestToANamedPersonIsAnsweredAtOnce(t *testing.T) {
	var seen []string
	srv, _, _, token := scenarioServer(t, scriptedLLM{seen: &seen, gaps: func() (string, error) { return "[]", nil },
		reply: func(prompt string) (string, error) {
			if strings.Contains(prompt, "thank-you note to my colleague Anna") {
				return "Thank you, Anna, for covering my shift!", nil
			}
			return "Moly reply.", nil
		}})
	reply, _, status, _ := sendFull(t, srv, token, "", "Write a thank-you note to my colleague Anna for covering my shift yesterday.", false)
	if status != http.StatusOK || strings.Contains(reply, "?") || !strings.Contains(reply, "Anna") {
		t.Fatalf("the note is written, no fixed question, got %q (status %d)", reply, status)
	}
}

// S45: a message may mention several people; each is saved. Found in a dynamic run: "my colleague Mark and my manager Dana"
// saved one person, and the people list (and the summary's anchors) lacked the other.
func TestScenarioS45EveryPersonOfAMessageIsRemembered(t *testing.T) {
	srv, _, userID, token := scenarioServer(t, scriptedLLM{gaps: func() (string, error) { return "[]", nil }})
	_, convID, _ := sendMessage(t, srv, token, "", "Help me split the shifts between Dana and Mark, Dana is my manager.")
	_, mem := getMemory(srv, token, convID)
	_ = userID
	people, _ := mem["people"].([]interface{})
	text := fmt.Sprint(people)
	if len(people) != 2 || !strings.Contains(text, "Dana") || !strings.Contains(text, "Mark") {
		t.Fatalf("both people are remembered, got %v", people)
	}
}

// S46: a question that Moly already asked does not come back in other words. Found in a dynamic run: the same question about
// the fiance was asked in two turns in a row, because gap questions are free text and only exact text was compared.
func TestScenarioS46AQuestionAlreadyAskedIsNotAskedAgainInOtherWords(t *testing.T) {
	calls := 0
	first := "Have you talked to the person it is about, before writing?"
	second := "Did you speak with that person already? REPEAT"
	srv, _, _, token := scenarioServer(t, scriptedLLM{
		gaps: func() (string, error) {
			calls++
			q := first
			if calls > 1 {
				q = second
			}
			return fmt.Sprintf(`[{"type": "talked_before%d", "description": %q, "assumption": "whether they spoke", "severity": "high", "confidence": 0.9, "goalTarget": "current_goal"}]`, calls, q), nil
		},
		reply: func(prompt string) (string, error) {
			switch {
			case strings.Contains(prompt, "asking: "+second) || strings.Contains(prompt, "REPEAT"):
				return "REPEATED-QUESTION?", nil
			case strings.Contains(prompt, first):
				return first, nil
			}
			return "Here is some help.", nil
		},
	})
	convID := startNamedGoal(t, srv, token)
	calls = 0
	r1, _, _ := sendMessage(t, srv, token, convID, "Perhaps I need to say something.")
	r2, _, _ := sendMessage(t, srv, token, convID, "I am not sure, I keep going back and forth.")
	if !strings.Contains(r1, first) {
		t.Fatalf("the first question is asked, got %q", r1)
	}
	if strings.Contains(r2, "REPEATED-QUESTION") || strings.Contains(r2, "REPEAT") {
		t.Fatalf("the same question in other words is not asked again, got %q", r2)
	}
}

// S47: a message is checked for an instruction about Moly once. The reply agent used to check it again for metadata that
// nothing read (found in a dynamic run: two identical model calls per turn).
func TestScenarioS47MetaInstructionIsCheckedOncePerMessage(t *testing.T) {
	var seen []string
	srv, _, _, token := scenarioServer(t, scriptedLLM{seen: &seen, gaps: func() (string, error) { return "[]", nil }})
	sendMessage(t, srv, token, "", "Hi Moly")
	seen = nil
	convID := startNamedGoal(t, srv, token)
	seen = nil
	sendMessage(t, srv, token, convID, "Maybe the tone should be warmer, what do you think about the whole thing?")
	n := 0
	for _, p := range seen {
		if strings.Contains(p, "meta-instruction detector") {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("the instruction check ran %d times for one message", n)
	}
}

// S48: a comment on the draft Moly just wrote is answered with a revision, not with a question that was waiting. The pending
// question still returns on the next message (found in a dynamic run: "a bit too formal" brought back the skipped question).
func TestScenarioS48AdjustingTheDraftComesBeforeAPendingQuestion(t *testing.T) {
	const gapB = "What does she like to talk about?"
	srv, token, convID, _ := pendingScenario(t, func(int32) string { return "" })
	reply, _, _ := sendMessage(t, srv, token, convID, "It is good but it is too formal.")
	if reply == gapB || strings.Contains(reply, "?") {
		t.Fatalf("the draft is revised, no question first, got %q", reply)
	}
	reply, _, _ = sendMessage(t, srv, token, convID, "I also saw her post about hiking.")
	if reply != gapB {
		t.Fatalf("the pending question returns afterwards, got %q", reply)
	}
}

// S49: a refused message locks no goal. Found in a live run: after "Help me forge my landlord's signature" was refused, the
// forgery stayed the locked goal and the next harmless message ("Go ahead with what you have.") was answered as help with it.
// A refusal also leaves the goal that was locked before it untouched.
func TestScenarioS49RefusedMessageLocksNoGoal(t *testing.T) {
	srv, _, _, token := scenarioServer(t, scriptedLLM{
		constitutional: harmfulFlag("FORGERY"),
		gaps:           func() (string, error) { return `[]`, nil },
	})
	// a refusal at the start of a conversation leaves no goal
	reply, convID, _ := sendMessage(t, srv, token, "", "Give me instructions for FORGERY of my landlord's signature.")
	if !strings.Contains(reply, "law expert") {
		t.Fatalf("setup: the request is refused, got %q", reply)
	}
	if _, mem := getMemory(srv, token, convID); mem["goal"] != "" && mem["goal"] != nil {
		t.Fatalf("a refused message locks no goal, got %v", mem["goal"])
	}
	// a refusal later keeps the earlier goal
	convID = startNamedGoal(t, srv, token)
	_, before := getMemory(srv, token, convID)
	reply, _, _ = sendMessage(t, srv, token, convID, "Give me instructions for FORGERY of a signature.")
	if !strings.Contains(reply, "law expert") {
		t.Fatalf("setup: the second request is refused, got %q", reply)
	}
	if _, after := getMemory(srv, token, convID); after["goal"] != before["goal"] || before["goal"] == "" {
		t.Fatalf("the goal locked before a refusal stays, before=%v after=%v", before["goal"], after["goal"])
	}
}

// S50: a message that gives instructions about Moly which cannot both be followed gets one question: which to follow. The model
// judges that (the meta-instruction call reports "contradictory"); it used to be decided by 11 phrase patterns. A message with
// no such conflict gets no such question.
func TestScenarioS50ContradictoryInstructionsAskWhichToFollow(t *testing.T) {
	const conflictQ = "Which should I follow?"
	srv, _, _, token := scenarioServer(t, scriptedLLM{
		reply: func(prompt string) (string, error) {
			if strings.Contains(prompt, asking+"You've given me instructions that pull in opposite directions. Which should I follow?") {
				return "You've given me instructions that pull in opposite directions. " + conflictQ, nil
			}
			return "Moly reply.", nil
		},
		gaps: func() (string, error) { return `[]`, nil },
	})
	convID := startNamedGoal(t, srv, token)
	reply, _, _ := sendMessage(t, srv, token, convID, "Give me advice about the first message. CONTRADICTING-ORDERS: and do not give me any advice.")
	if !strings.Contains(reply, conflictQ) {
		t.Fatalf("contradictory instructions are asked about, got %q", reply)
	}
	reply, _, _ = sendMessage(t, srv, token, convID, "Quite informal, she seems relaxed.")
	if strings.Contains(reply, conflictQ) {
		t.Fatalf("a message without a conflict is not asked, got %q", reply)
	}
}

// S51: a refused message teaches Moly nothing about the user. Found by counting what a scripted refusal writes: the forgery aim
// was stored as a fact about the user (context_attributes), in the per-message summary and in the saved conversation state
// (entities), where later turns read it. What stays: the chat lines and the safety incident (audit).
func TestScenarioS51RefusedMessageStoresNoDerivedFacts(t *testing.T) {
	srv, db, userID, token := scenarioServer(t, scriptedLLM{
		constitutional: harmfulFlag("FORGERY"),
		gaps:           func() (string, error) { return `[]`, nil },
	})
	reply, convID, _ := sendMessage(t, srv, token, "", "Give me clear instructions for FORGERY of my landlord's signature.")
	if !strings.Contains(reply, "law expert") {
		t.Fatalf("setup: refused, got %q", reply)
	}
	time.Sleep(300 * time.Millisecond) // the per-message summary is saved in the background
	conn := db.GetConnection()
	for _, table := range []string{"context_attributes", "message_summaries"} {
		var n int
		conn.QueryRow(`SELECT COUNT(*) FROM `+table+` WHERE user_id = ?`, userID).Scan(&n)
		if n != 0 {
			t.Errorf("%s holds %d row(s) derived from a refused message", table, n)
		}
	}
	var state string
	conn.QueryRow(`SELECT state FROM conversation_context WHERE user_id = ? AND conversation_id = ?`, userID, convID).Scan(&state)
	if strings.Contains(strings.ToLower(state), "forge") {
		t.Errorf("the saved conversation state holds the refused aim: %s", state)
	}
	var incidents, lines int
	conn.QueryRow(`SELECT COUNT(*) FROM safety_incidents WHERE user_id = ?`, userID).Scan(&incidents)
	conn.QueryRow(`SELECT COUNT(*) FROM chat_messages m JOIN conversations c ON c.id = m.conversation_id WHERE c.user_id = ?`, userID).Scan(&lines)
	if incidents == 0 || lines != 2 {
		t.Errorf("the incident and the two chat lines are kept for audit: incidents=%d lines=%d", incidents, lines)
	}
}

// S52: a pending question comes back only while it is still worth asking. When what the user says next makes it moot, the model
// says so and the question is cancelled: it is not asked, it is no longer open, and it no longer holds maturity down. (Before,
// it returned until it was answered or the goal changed.)
func TestScenarioS52PendingQuestionThatIsMootIsCancelled(t *testing.T) {
	const gapB = "What does she like to talk about?"
	srv, token, convID, _ := pendingScenario(t, func(int32) string { return "" })
	reply, _, _ := sendMessage(t, srv, token, convID, "IRRELEVANT-NOW I already know what she likes, we talked yesterday.")
	if reply == gapB {
		t.Fatalf("a question made moot is not asked again, got %q", reply)
	}
	var skipped, cancelled, open int
	conn := srv.database.GetConnection()
	conn.QueryRow(`SELECT COUNT(*) FROM clarification_questions WHERE conversation_id = ? AND status = 'skipped'`, convID).Scan(&skipped)
	conn.QueryRow(`SELECT COUNT(*) FROM clarification_questions WHERE conversation_id = ? AND status = 'cancelled'`, convID).Scan(&cancelled)
	conn.QueryRow(`SELECT COUNT(*) FROM clarification_questions WHERE conversation_id = ? AND status IN ('active','skipped')`, convID).Scan(&open)
	if skipped != 0 || cancelled != 1 || open != 0 {
		t.Fatalf("the moot question is cancelled and no longer open (skipped %d, cancelled %d, open %d)", skipped, cancelled, open)
	}
	if got := agents.ConversationGapMaturity(srv.database.GetClarificationQuestionRepository(), convID); got != 1 {
		t.Fatalf("a cancelled question does not hold maturity down, got %v", got)
	}
}

// S53: a goal switch that was proposed in a reply where another question won (the name of the new person) is not lost. The name
// question is asked first; the switch is asked on the next turn; a yes then changes the goal. Before, the proposal was saved as
// if it had been asked, the next message was judged against the name question, and the switch vanished silently.
func TestScenarioS53GoalSwitchSurvivesANameQuestionThatWonTheReply(t *testing.T) {
	srv, _, userID, token := scenarioServer(t, scriptedLLM{gaps: func() (string, error) { return `[]`, nil }})
	convID := startNamedGoal(t, srv, token)
	const first = "initiate a conversation with the girl on fetlife"
	sendMessage(t, srv, token, convID, "Her name is Christine.")

	ask, _, _, meta := sendFull(t, srv, token, convID, "SWITCH-WITH-NEW-PERSON Now I need to write to my brother about the party.", false)
	if !strings.Contains(ask, "name for the brother") || meta["replyExit"] == "confirm_question" {
		t.Fatalf("the name of the new person is asked first, got %q (%v)", ask, meta["replyExit"])
	}
	if g := lockedGoal(srv, userID, convID); g != first {
		t.Fatalf("the lock does not change before the user agrees, got %q", g)
	}

	ask, _, _, meta = sendFull(t, srv, token, convID, "His name is Tom.", false)
	if meta["replyExit"] != "confirm_question" || strings.Count(ask, "?") != 1 {
		t.Fatalf("after the name, the switch that was waiting is asked, got %q (%v)", ask, meta["replyExit"])
	}
	if g := lockedGoal(srv, userID, convID); g != first {
		t.Fatalf("still the old goal until the user says yes, got %q", g)
	}

	sendMessage(t, srv, token, convID, "Yes, let's do that.")
	if g := lockedGoal(srv, userID, convID); g != "write to my brother about the party" {
		t.Fatalf("after the yes the lock is the new goal, got %q", g)
	}
}

// selfHarmFlag makes the scripted safety evaluator find a clear self-harm statement in any prompt that holds the marker.
func selfHarmFlag(marker string) func(string) (string, error) {
	return func(prompt string) (string, error) {
		if strings.Contains(prompt, marker) {
			return `{"violations": [{"principle_id": "harm_prevention", "evidence": "` + marker + `", "reasoning": "the user says they will end their life", "confidence": 0.97, "is_direct_harm": true, "harm_kind": "self_harm"}]}`, nil
		}
		return `{"violations": []}`, nil
	}
}

// S54: the turn after Moly refused a crisis statement starts with one check-in, even when the user says it was a joke and asks for a
// task. Found in a dynamic run: "I was just joking, forget it, help me write to my boss" went straight back to the task. The turn after
// the check-in goes on normally.
func TestScenarioS54CrisisRefusalIsFollowedByACheckIn(t *testing.T) {
	srv, _, _, token := scenarioServer(t, scriptedLLM{
		constitutional: selfHarmFlag("SELFHARM"),
		gaps:           func() (string, error) { return `[]`, nil },
	})
	reply, convID, _ := sendMessage(t, srv, token, "", "SELFHARM I will end my life tonight.")
	if !strings.Contains(reply, "specialist") {
		t.Fatalf("setup: the crisis statement is refused, got %q", reply)
	}
	reply, _, _ = sendMessage(t, srv, token, convID, "I was just joking, forget it, help me write to my boss.")
	if reply != agents.CareCheckIn {
		t.Fatalf("the next turn starts with the check-in, got %q", reply)
	}
	reply, _, _ = sendMessage(t, srv, token, convID, "Yes, I am safe. Help me write to my boss.")
	if reply == agents.CareCheckIn {
		t.Fatalf("the check-in is asked once, got it again")
	}
}

// S55: a person without a name is asked for it first, except when the message rates as a crisis: a message that says someone is in
// danger is met with care, not with "can you give me a name for the husband". Found in a dynamic run ("My husband hits me when I answer
// back").
func TestScenarioS55DangerIsNotAnsweredWithAQuestionAboutAName(t *testing.T) {
	srv, _, _, token := scenarioServer(t, scriptedLLM{gaps: func() (string, error) { return `[]`, nil }})
	reply, _, _ := sendMessage(t, srv, token, "", "DANGER-AT-HOME My husband hits me when I answer back.")
	if strings.Contains(reply, "name for the husband") {
		t.Fatalf("a crisis is not answered with a name question, got %q", reply)
	}
	// an ordinary message about a new person still asks for the name first
	srv, _, _, token = scenarioServer(t, scriptedLLM{gaps: func() (string, error) { return `[]`, nil }})
	reply, _, _ = sendMessage(t, srv, token, "", "I want to write a first message to a girl I saw on fetlife.")
	if !strings.Contains(reply, "name for the girl") {
		t.Fatalf("an ordinary message still asks for the name, got %q", reply)
	}
}

// S56: a question about Moly itself ("how do I use you?", "what does the Skip button do?") is answered from the product facts. It sets
// no goal (found in a dynamic run: "Learn how to use Moly" became the locked goal and the next question asked "switch focus?"),
// saves no person, and asks nothing back. The facts are in the reply prompt only for such a message, and the goal that comes next is
// taken as the first goal, with no switch question.
func TestScenarioS56QuestionAboutMolyIsAnsweredFromProductFacts(t *testing.T) {
	var seen []string
	srv, db, userID, token := scenarioServer(t, scriptedLLM{
		seen: &seen,
		reply: func(prompt string) (string, error) {
			if strings.Contains(prompt, "FACTS ABOUT MOLY") {
				return "The Skip button lets you go ahead with what Moly knows.", nil
			}
			return "Moly reply.", nil
		},
		gaps: func() (string, error) { return `[]`, nil },
	})
	reply, convID, _ := sendMessage(t, srv, token, "", "ABOUT-MOLY What does the Skip button do?")
	extractions := 0
	for _, p := range seen {
		if strings.Contains(p, "userCharacteristics") {
			extractions++
		}
	}
	if extractions != 1 {
		t.Fatalf("a question about Moly is extracted once, and Layer 1 does not look for a goal again, got %d extraction calls", extractions)
	}
	if reply != "The Skip button lets you go ahead with what Moly knows." {
		t.Fatalf("the answer is written from the product facts, got %q", reply)
	}
	if g := lockedGoal(srv, userID, convID); g != "" {
		t.Fatalf("a question about Moly locks no goal, got %q", g)
	}
	if got := contactsFor(t, db, userID); len(got) != 0 {
		t.Fatalf("a question about Moly saves no person, got %+v", got)
	}
	// the facts are not sent with every message
	seen = nil
	sendMessage(t, srv, token, convID, "I want to write a first message to a girl I saw on fetlife.")
	for _, p := range seen {
		if strings.Contains(p, "FACTS ABOUT MOLY") {
			t.Fatalf("the product facts belong to questions about Moly only")
		}
	}
	if g := lockedGoal(srv, userID, convID); g == "" {
		t.Fatalf("the next message sets the first goal, with no switch question")
	}
}

// S57: secrets the user types are not stored. A card number is cut out by its shape (Luhn) before anything uses the message; a
// password is found by the extraction and cut out of the message and of what was derived from it, before the intent, the reply and
// every save. Found in a dynamic run: a card number and a password stayed in the chat history.
func TestScenarioS57SecretsAreNotStored(t *testing.T) {
	var seen []string
	srv, db, userID, token := scenarioServer(t, scriptedLLM{seen: &seen, gaps: func() (string, error) { return `[]`, nil }})
	reply, convID, _ := sendMessage(t, srv, token, "", "SECRET-TEST My password is hunter2 and my card number is 4111 1111 1111 1111. Write a complaint to my bank.")
	time.Sleep(400 * time.Millisecond)
	for _, secret := range []string{"hunter2", "4111"} {
		if strings.Contains(reply, secret) {
			t.Fatalf("the reply repeats %q", secret)
		}
		conn := db.GetConnection()
		for table, column := range map[string]string{
			"chat_messages": "content", "interactions": "content", "user_interactions": "user_message",
			"message_summaries": "extracted_entities", "context_attributes": "fact_value", "conversation_context": "state",
		} {
			var n int
			q := `SELECT COUNT(*) FROM ` + table + ` WHERE user_id = ? AND ` + column + ` LIKE ?`
			if table == "chat_messages" {
				q = `SELECT COUNT(*) FROM chat_messages m JOIN conversations c ON c.id = m.conversation_id WHERE c.user_id = ? AND m.content LIKE ?`
			}
			conn.QueryRow(q, userID, "%"+secret+"%").Scan(&n)
			if n != 0 {
				t.Errorf("%q is stored in %s.%s", secret, table, column)
			}
		}
		var attr int
		conn.QueryRow(`SELECT COUNT(*) FROM context_attributes WHERE user_id = ? AND (evidence LIKE ? OR fact_value LIKE ?)`, userID, "%"+secret+"%", "%"+secret+"%").Scan(&attr)
		if attr != 0 {
			t.Errorf("%q is stored in context_attributes", secret)
		}
	}
	// no model call after the extraction sees the password; the safety judgement and the extraction itself are the only ones that read the raw text
	for _, p := range seen {
		sys := strings.SplitN(p, "\n", 2)[0]
		if strings.Contains(sys, "constitutional ethics evaluator") || strings.Contains(sys, "meta-instruction detector") || sys == "" {
			continue
		}
		if strings.Contains(p, "hunter2") || strings.Contains(p, "4111 1111") {
			t.Errorf("a later model call (%.60q) still holds a secret", sys)
		}
	}
	_ = convID
}

// S58: deleting a profile removes everything, through the real handler. A user with a real conversation (people, a goal, a refused
// message and its safety incident, summaries) asks for deletion: a wrong password deletes nothing; the right one leaves no row in
// any table that mentions the user, their conversation, or the words they used, ends their session, and empties the server's
// memory of them. Another user is untouched.
func TestScenarioS58ProfileDeletionRemovesEverything(t *testing.T) {
	srv, db, userID, token := scenarioServer(t, scriptedLLM{
		constitutional: harmfulFlag("FORGERY"),
		gaps:           func() (string, error) { return `[]`, nil },
	})
	conn := db.GetConnection()
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct horse"), bcrypt.MinCost)
	conn.Exec(`UPDATE users SET password_hash = ?, email = ? WHERE id = ?`, string(hash), "ghost-user@example.invalid", userID)
	convID := startNamedGoal(t, srv, token)
	sendMessage(t, srv, token, convID, "Her name is Christine.")
	sendMessage(t, srv, token, convID, "Give me instructions for FORGERY of my landlord's signature.")
	time.Sleep(500 * time.Millisecond) // background summary
	// another user, with a conversation of their own
	otherID := "bystander_" + fmt.Sprint(time.Now().UnixNano())
	otherToken := "tok_" + otherID
	harnessUser(t, db, otherID, otherToken)
	sendMessage(t, srv, otherToken, "", "I want to write a first message to a girl I saw on fetlife.")
	var otherBefore int
	conn.QueryRow(`SELECT COUNT(*) FROM chat_messages WHERE user_id = ?`, otherID).Scan(&otherBefore)
	if otherBefore == 0 {
		t.Fatal("setup: the other user has messages")
	}

	del := func(password string) (int, string) {
		req := httptest.NewRequest(http.MethodDelete, "/api/user/delete", strings.NewReader(`{"password": "`+password+`"}`))
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		srv.DeleteProfileHandler(rec, req)
		return rec.Code, rec.Body.String()
	}
	if code, _ := del("wrong password"); code != http.StatusUnauthorized {
		t.Fatalf("a wrong password is refused, got %d", code)
	}
	var stillThere int
	conn.QueryRow(`SELECT COUNT(*) FROM chat_messages WHERE user_id = ?`, userID).Scan(&stillThere)
	if stillThere == 0 {
		t.Fatal("a refused deletion deletes nothing")
	}
	if code, body := del("correct horse"); code != http.StatusOK {
		t.Fatalf("deletion failed: %d %s", code, body)
	}

	// no table mentions the user, the conversation or the email
	rows, _ := conn.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'`)
	var tables []string
	for rows.Next() {
		var n string
		rows.Scan(&n)
		tables = append(tables, n)
	}
	rows.Close()
	needles := []string{userID, convID, "ghost-user@example"} // unique to this user (the test database is shared with other scenarios)
	for _, tb := range tables {
		cr, _ := conn.Query(fmt.Sprintf(`SELECT name FROM pragma_table_info('%s')`, tb))
		var conds []string
		for cr.Next() {
			var c string
			cr.Scan(&c)
			for _, nd := range needles {
				conds = append(conds, fmt.Sprintf(`CAST("%s" AS TEXT) LIKE '%%%s%%'`, c, strings.ReplaceAll(nd, "'", "''")))
			}
		}
		cr.Close()
		var n int
		conn.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE %s`, tb, strings.Join(conds, " OR "))).Scan(&n)
		if n != 0 {
			t.Errorf("table %s still has %d row(s) that mention the deleted user, the conversation or their words", tb, n)
		}
	}
	var u, s int
	conn.QueryRow(`SELECT COUNT(*) FROM users WHERE id = ?`, userID).Scan(&u)
	conn.QueryRow(`SELECT COUNT(*) FROM sessions WHERE user_id = ?`, userID).Scan(&s)
	if u != 0 || s != 0 {
		t.Fatalf("the user and their sessions are gone, got users=%d sessions=%d", u, s)
	}
	if _, err := extractAndValidateToken(func() *http.Request {
		r := httptest.NewRequest(http.MethodGet, "/api/contacts", nil)
		r.Header.Set("Authorization", "Bearer "+token)
		return r
	}(), db); err == nil {
		t.Fatal("the old token no longer works")
	}
	srv.previousExtractionCache.Range(func(k, _ interface{}) bool {
		if strings.HasPrefix(k.(string), userID+"|") {
			t.Errorf("the server still holds the conversation state %v", k)
		}
		return true
	})
	var otherAfter int
	conn.QueryRow(`SELECT COUNT(*) FROM chat_messages WHERE user_id = ?`, otherID).Scan(&otherAfter)
	if otherAfter != otherBefore {
		t.Fatalf("the other user's messages are untouched: %d -> %d", otherBefore, otherAfter)
	}
}

// S59: a name is needed to write to someone, not to talk about them. A chat about a grandmother, with no aim, is not answered with
// "can you give me a name for the grandmother" (found in a dynamic run, and for a disclosure of abuse and for a figure of speech about a
// brother). The person is kept; the name is asked when the user states an aim that involves her.
func TestScenarioS59NameIsAskedWhenThereIsAnAim(t *testing.T) {
	srv, _, userID, token := scenarioServer(t, scriptedLLM{gaps: func() (string, error) { return `[]`, nil }})
	reply, convID, _ := sendMessage(t, srv, token, "", "CHAT-GRANDMA I have been thinking about my grandmother a lot lately.")
	if strings.Contains(reply, "name for the grandmother") {
		t.Fatalf("a chat without an aim gets no name question, got %q", reply)
	}
	reply, _, _ = sendMessage(t, srv, token, convID, "CALL-GRANDMA I want to call my grandmother this weekend.")
	if !strings.Contains(reply, "name for the grandmother") {
		t.Fatalf("with an aim that involves her, the name is asked, got %q", reply)
	}
	_ = userID
}

// conversationNameOf reads the stored name of a conversation.
func conversationNameOf(db *database.Database, convID string) string {
	var n string
	db.GetConnection().QueryRow(`SELECT name FROM conversations WHERE id = ?`, convID).Scan(&n)
	return n
}

// TestScenarioConversationIsNamedAfterItsGoal: a greeting leaves the name "Direct Message"; once the goal is locked, the
// conversation gets a short title written from it (in the background, after the reply).
func TestScenarioConversationIsNamedAfterItsGoal(t *testing.T) {
	srv, db, userID, token := scenarioServer(t, scriptedLLM{})
	_ = userID
	_, convID, code := sendMessage(t, srv, token, "", "Hi Moly")
	if code != http.StatusOK {
		t.Fatalf("greeting: status %d", code)
	}
	time.Sleep(300 * time.Millisecond)
	if n := conversationNameOf(db, convID); n != defaultConversationName {
		t.Fatalf("a greeting must not rename the conversation, got %q", n)
	}
	_, convID, code = sendMessage(t, srv, token, convID, "I want to write a first message to a girl I saw on fetlife.")
	if code != http.StatusOK {
		t.Fatalf("goal message: status %d", code)
	}
	deadline := time.Now().Add(5 * time.Second)
	for conversationNameOf(db, convID) == defaultConversationName && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if n := conversationNameOf(db, convID); n != "Opening message" {
		t.Fatalf("once the goal is locked the conversation must be named after it, got %q", n)
	}
}

// savedPerson saves a person with a name for the user, as an earlier conversation would have.
func savedPerson(t *testing.T, db *database.Database, userID, name, relationship string) {
	t.Helper()
	err := database.NewContactRepository(db).Save(&models.Contact{UserID: userID, Name: name, NameStatus: "named", Relationship: relationship, Status: "active", CreatedVia: "conversation"})
	if err != nil {
		t.Fatal(err)
	}
}

// S62 (found live, 2026-10-14): "Hi Moly" and a first message in a NEW conversation were answered with a question about "her" and
// "Dana" from a chat of days before. What is unresolved belongs to the conversation it came from: the unanswered conflict and the
// person waiting for a name stay in the old conversation, are not asked in the new one, and are not closed by what is said there.
func TestScenarioS62UnresolvedBelongsToItsConversation(t *testing.T) {
	srv, db, userID, token := scenarioServer(t, scriptedLLM{})
	_, oldConv, _ := sendMessage(t, srv, token, "", "Hi Moly")
	savedPerson(t, db, userID, "Dana", "professional")
	if err := database.NewContactRepository(db).InConversation(oldConv).Save(&models.Contact{UserID: userID, Name: "manager", NameStatus: "asked", Relationship: "professional", Status: "active"}); err != nil {
		t.Fatal(err)
	}
	const oldQuestion = "When you said 'her', who did you mean? 'her' (professional), 'Dana' (professional)"
	conflict := &database.ContextConflict{UserID: userID, ConversationID: oldConv, ConflictType: "contact_ambiguous_generic_name", Severity: "low",
		SavedValue: "'her' (professional), 'Dana' (professional)", ExtractedValue: "her", Description: oldQuestion, Status: "unresolved", CreatedAt: time.Now().Unix()}
	if err := db.GetContextConflictRepository().Save(conflict); err != nil {
		t.Fatal(err)
	}

	// the same course of messages in a new conversation
	greeting, newConv, _ := sendMessage(t, srv, token, "", "Hi Moly")
	first, _, _ := sendMessage(t, srv, token, newConv, "I want to write a first message to a girl I saw on fetlife.")
	reply, _, code := sendMessage(t, srv, token, newConv, "Perhaps I need to say something.")
	for _, said := range []string{greeting, first, reply} {
		if strings.Contains(said, "Dana") || strings.Contains(said, "who did you mean") || strings.Contains(said, "manager") {
			t.Fatalf("the new conversation was answered with the old conversation's open matters: %q", said)
		}
	}
	if code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	if left, _ := db.GetContextConflictRepository().GetUnresolved(userID, oldConv); len(left) != 1 {
		t.Fatalf("the old conflict stays open in its own conversation, got %d", len(left))
	}
	var status string
	db.GetConnection().QueryRow(`SELECT name_status FROM contacts WHERE user_id = ? AND name = 'manager'`, userID).Scan(&status)
	if status != "asked" {
		t.Fatalf("the old conversation's name question is not closed by another conversation, status %q", status)
	}

	// control: in its own conversation the question IS asked (the path is live in this scripted run)
	sendMessage(t, srv, token, oldConv, "I want to write a first message to a girl I saw on fetlife.")
	if asked, _, _ := sendMessage(t, srv, token, oldConv, "Perhaps I need to say something."); !strings.Contains(asked, "who did you mean") {
		t.Fatalf("control: the open conflict is asked in its own conversation, got %q", asked)
	}
}

// S63: the name first, the comparison after. The user has a saved boss, Dana. In a new conversation "my boss" is NOT taken for
// Dana: the name is asked. When the user gives the name, it is compared with the saved people, and the same name is the same person.
func TestScenarioS63TheNameIsAskedBeforeComparingWithSavedPeople(t *testing.T) {
	srv, db, userID, token := scenarioServer(t, scriptedLLM{})
	savedPerson(t, db, userID, "Dana", "professional")
	_, convID, _ := sendMessage(t, srv, token, "", "Hi Moly")
	reply, _, _ := sendMessage(t, srv, token, convID, "I need to write to my boss. WRITE-TO-BOSS")
	if !strings.Contains(reply, "give me a name for the boss") || strings.Contains(reply, "Dana") {
		t.Fatalf("the name of the boss is asked and Dana is not assumed, got %q", reply)
	}
	got := contactsFor(t, db, userID)
	if len(got) != 2 || got[0].Name != "Dana" || got[1].Name != "boss" || got[1].NameStatus != "asked" {
		t.Fatalf("Dana stays as she was and the boss is a separate person waiting for a name, got %+v", got)
	}

	reply, _, _ = sendMessage(t, srv, token, convID, "His name is Dana.")
	if strings.Contains(reply, "give me a name") {
		t.Fatalf("the name was given, got %q", reply)
	}
	got = contactsFor(t, db, userID)
	if len(got) != 1 || got[0].Name != "Dana" || got[0].NameStatus != "named" {
		t.Fatalf("the boss is Dana: one person, got %+v", got)
	}
	var role string
	db.GetConnection().QueryRow(`SELECT COALESCE(contact_role, '') FROM contacts WHERE user_id = ? AND name = 'Dana'`, userID).Scan(&role)
	if role != "boss" {
		t.Fatalf("what the user first called her is kept as her role, got %q", role)
	}
}

// S64: a pronoun in the name field is not a name and does not become a person.
func TestScenarioS64APronounIsNeverSavedAsAPerson(t *testing.T) {
	srv, db, userID, token := scenarioServer(t, scriptedLLM{})
	_, convID, _ := sendMessage(t, srv, token, "", "Hi Moly")
	_, _, code := sendMessage(t, srv, token, convID, "PRONOUN-AS-NAME")
	if code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	if got := contactsFor(t, db, userID); len(got) != 0 {
		t.Fatalf("a pronoun with nobody to point to is no one, got %+v", got)
	}
}

// S65: once the conversation has named the boss, "my boss" in a later message is that person; the name is not asked again.
func TestScenarioS65ARoleAlreadyNamedInTheConversationIsNotAskedAgain(t *testing.T) {
	srv, db, userID, token := scenarioServer(t, scriptedLLM{})
	_, convID, _ := sendMessage(t, srv, token, "", "Hi Moly")
	sendMessage(t, srv, token, convID, "I need to write to my boss. WRITE-TO-BOSS")
	sendMessage(t, srv, token, convID, "His name is Dana.")
	reply, _, code := sendMessage(t, srv, token, convID, "My boss is unfair. BOSS-AGAIN")
	if code != http.StatusOK || strings.Contains(reply, "give me a name") {
		t.Fatalf("the boss is Dana, no name question; status %d reply %q", code, reply)
	}
	if got := contactsFor(t, db, userID); len(got) != 1 || got[0].Name != "Dana" {
		t.Fatalf("still one person, Dana, got %+v", got)
	}
}

// S66: what the user confirmed about themselves is used in every conversation; an insight still waiting for the user's
// approval belongs to the conversation it came from and is not given to the reply model of another one. (Before, the last five
// reflections of any status from any conversation went into every reply prompt as "what you've learned about them".)
func TestScenarioS66UnconfirmedInsightsStayInTheirConversation(t *testing.T) {
	var prompts []string
	srv, db, userID, token := scenarioServer(t, scriptedLLM{seen: &prompts})
	_, oldConv, _ := sendMessage(t, srv, token, "", "Hi Moly")
	now := time.Now().Unix()
	insert := func(conv, status, trait string) {
		t.Helper()
		_, err := db.GetConnection().Exec(`INSERT INTO reflections (user_id, conversation_id, characteristics, interests, intentions, status, created_at) VALUES (?, ?, ?, '[]', '[]', ?, ?)`,
			userID, conv, `["`+trait+`"]`, status, now)
		if err != nil {
			t.Fatal(err)
		}
	}
	insert(oldConv, "approved", "ZZ-CONFIRMED-TRAIT")
	insert(oldConv, "pending_approval", "ZZ-PENDING-OF-OLD-CONVERSATION")

	_, newConv, _ := sendMessage(t, srv, token, "", "Hi Moly")
	sendMessage(t, srv, token, newConv, "I want to write a first message to a girl I saw on fetlife.")
	prompts = nil // the reply model is called on the next message (the first one asked for a name)
	sendMessage(t, srv, token, newConv, "Her name is Christine.")
	seen := strings.Join(prompts, "\n")
	if !strings.Contains(seen, "ZZ-CONFIRMED-TRAIT") {
		t.Fatalf("what the user confirmed is known in every conversation, but no prompt carried it")
	}
	if strings.Contains(seen, "ZZ-PENDING-OF-OLD-CONVERSATION") {
		t.Fatalf("an unconfirmed insight of another conversation reached the model")
	}
	// the conversation that learned it still has it
	sendMessage(t, srv, token, oldConv, "I want to write a first message to a girl I saw on fetlife.")
	prompts = nil
	sendMessage(t, srv, token, oldConv, "Her name is Christine.")
	if !strings.Contains(strings.Join(prompts, "\n"), "ZZ-PENDING-OF-OLD-CONVERSATION") {
		t.Fatalf("control: the conversation that learned the insight keeps it")
	}
}

// S67: the goal of another conversation is not carried into a new one ("Earlier they mentioned: ..." used to take the latest goal
// the user stated in any conversation).
func TestScenarioS67TheGoalOfAnotherConversationIsNotCarriedOver(t *testing.T) {
	var prompts []string
	srv, _, _, token := scenarioServer(t, scriptedLLM{seen: &prompts})
	_, oldConv, _ := sendMessage(t, srv, token, "", "Hi Moly")
	sendMessage(t, srv, token, oldConv, "I want to write a first message to a girl I saw on fetlife.")

	_, newConv, _ := sendMessage(t, srv, token, "", "Hi Moly")
	prompts = nil
	sendMessage(t, srv, token, newConv, "Yes, let's do that.") // states no goal; the reply model is called
	called := false
	for _, p := range prompts {
		if strings.Contains(p, "They just said: \"Yes, let's do that.\"") {
			called = true
			if strings.Contains(p, "Earlier they mentioned: initiate a conversation with the girl on fetlife") {
				t.Fatalf("a new conversation was told the goal of an old one")
			}
		}
	}
	if !called {
		t.Fatalf("setup: the reply model was not called for the message")
	}
}

// S68: a clarification that is waiting for an answer belongs to the conversation that raised it. The next message of ANOTHER
// conversation used to be recorded as the answer to it (the pending ones of the user, from any conversation, were loaded).
func TestScenarioS68PendingClarificationsBelongToTheirConversation(t *testing.T) {
	srv, db, userID, token := scenarioServer(t, scriptedLLM{})
	_, oldConv, _ := sendMessage(t, srv, token, "", "Hi Moly")
	store := agents.NewTemporaryFactStore(db, userID)
	if err := store.Store(&agents.TemporaryFact{FactID: "gap_" + oldConv + "_0", FactType: "gap", FactValue: "formality", ConversationID: oldConv}); err != nil {
		t.Fatal(err)
	}
	_, newConv, _ := sendMessage(t, srv, token, "", "Hi Moly")
	if got, _ := store.GetPendingForConversation(newConv); len(got) != 0 {
		t.Fatalf("a new conversation has no pending clarification of its own, got %d", len(got))
	}
	if got, _ := store.GetPendingForConversation(""); len(got) != 0 {
		t.Fatalf("without a conversation nothing is pending, got %d", len(got))
	}
	if got, _ := store.GetPendingForConversation(oldConv); len(got) != 1 {
		t.Fatalf("the conversation that raised it still has it, got %d", len(got))
	}
}

// scenarioSource is the model source of the scenario suite. A model call made through a user's components is forwarded to the
// scripted model (and written to calls, with the user it was made for); a call made through the components of nobody FAILS, as it
// does in the real program. So a scenario only passes if every model call of the message flow is made on behalf of the user.
type scenarioSource struct {
	llm   tools.LLMProvider
	calls *callLog
}

func (s scenarioSource) For(userID string) tools.LLMProvider {
	return userTagged{userID: userID, inner: s.llm, calls: s.calls}
}
func (s scenarioSource) Unbound() tools.LLMProvider {
	return unboundRecorder{calls: s.calls}
}

// unboundRecorder is the provider of nobody: it fails every call, as the real one does, and counts the attempts.
type unboundRecorder struct{ calls *callLog }

func (u unboundRecorder) Call(ctx context.Context, req *tools.LLMRequest) (*tools.LLMResponse, error) {
	u.calls.add("")
	return nil, tools.ErrNoUser
}

// callLog records which user each model call was made for.
type callLog struct {
	mu    sync.Mutex
	users []string
}

// withoutUser counts the calls that were made for nobody.
func (l *callLog) withoutUser() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := 0
	for _, u := range l.users {
		if u == "" {
			n++
		}
	}
	return n
}

func (l *callLog) add(user string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	l.users = append(l.users, user)
	l.mu.Unlock()
}

type userTagged struct {
	userID string
	inner  tools.LLMProvider
	calls  *callLog
}

func (u userTagged) Call(ctx context.Context, req *tools.LLMRequest) (*tools.LLMResponse, error) {
	u.calls.add(u.userID)
	return u.inner.Call(ctx, req)
}

// perUserSource gives each user their own scripted model; a user not in the map has no provider. Used to show that the
// message handler serves every user with that user's provider and no one else's.
type perUserSource struct {
	models map[string]tools.LLMProvider
	calls  *callLog
}

func (s perUserSource) For(userID string) tools.LLMProvider {
	if m, ok := s.models[userID]; ok {
		return userTagged{userID: userID, inner: m, calls: s.calls}
	}
	return userTagged{userID: userID, inner: noProvider{}, calls: s.calls}
}
func (s perUserSource) Unbound() tools.LLMProvider { return unboundRecorder{calls: s.calls} }

type noProvider struct{}

func (noProvider) Call(ctx context.Context, req *tools.LLMRequest) (*tools.LLMResponse, error) {
	return nil, tools.ErrNoProvider
}

// S69: two users, two providers, one running server. Each user's message is answered by that user's own model, every model call
// made while handling it is made for that user, and a user who has no provider is not served by anyone else's.
func TestScenarioS69EachUserIsServedByTheirOwnProvider(t *testing.T) {
	db, err := database.Init(testDBPath)
	if err != nil {
		t.Fatal(err)
	}
	calls := &callLog{}
	stamp := time.Now().UnixNano()
	userA, userB, userC := fmt.Sprintf("iso_a_%d", stamp), fmt.Sprintf("iso_b_%d", stamp), fmt.Sprintf("iso_c_%d", stamp)
	says := func(text string) scriptedLLM {
		return scriptedLLM{reply: func(string) (string, error) { return text, nil }}
	}
	srv, err := NewAPIServer(perUserSource{models: map[string]tools.LLMProvider{userA: says("REPLY-OF-A"), userB: says("REPLY-OF-B")}, calls: calls}, db)
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range []string{userA, userB, userC} {
		harnessUser(t, db, u, "token_"+u)
	}
	since := func(from int) (users map[string]int) {
		calls.mu.Lock()
		defer calls.mu.Unlock()
		users = map[string]int{}
		for _, u := range calls.users[from:] {
			users[u]++
		}
		return users
	}
	// A reply starts background work (the memory summary, the title); wait for it to end, so each step shows only its own calls.
	mark := func() int {
		last := -1
		for i := 0; i < 100; i++ {
			calls.mu.Lock()
			n := len(calls.users)
			calls.mu.Unlock()
			if n == last {
				return n
			}
			last = n
			time.Sleep(300 * time.Millisecond)
		}
		return last
	}

	from := mark()
	replyA, _, _ := sendMessage(t, srv, "token_"+userA, "", "Yes, let's do that.")
	if !strings.Contains(replyA, "REPLY-OF-A") {
		t.Fatalf("A is answered by A's model, got %q", replyA)
	}
	if got := since(from); len(got) != 1 || got[userA] == 0 {
		t.Fatalf("while A was served, model calls were made for %v", got)
	}

	from = mark()
	replyB, _, _ := sendMessage(t, srv, "token_"+userB, "", "Yes, let's do that.")
	if !strings.Contains(replyB, "REPLY-OF-B") || strings.Contains(replyB, "REPLY-OF-A") {
		t.Fatalf("B is answered by B's model, got %q", replyB)
	}
	mark()
	if got := since(from); len(got) != 1 || got[userB] == 0 {
		t.Fatalf("while B was served, model calls were made for %v", got)
	}

	from = mark()
	replyC, _, _ := sendMessage(t, srv, "token_"+userC, "", "Yes, let's do that.")
	if strings.Contains(replyC, "REPLY-OF-A") || strings.Contains(replyC, "REPLY-OF-B") {
		t.Fatalf("a user with no provider was served by another user's model: %q", replyC)
	}
	if got := since(from); got[userA] != 0 || got[userB] != 0 || got[""] != 0 {
		t.Fatalf("while C was served, model calls were made for %v", got)
	}
}
