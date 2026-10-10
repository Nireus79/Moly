package main

// Step 0 of ORCHESTRATOR_DESIGN.md section 10: a scripted end-to-end test of the real message handler.
// The language model is scripted by prompt type, the database is real (temporary), and the handler is called directly.
// Expected results are written first; this test is expected to FAIL until the orchestration steps are done.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"moly/agents"
	"moly/database"
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
	case strings.Contains(req.SystemPrompt, "who a phrase refers to"):
		// the person-reference judge: words about the text are no person; a pronoun points to the one person known
		switch {
		case strings.Contains(prompt, "Phrase taken from it: \"a bit too formal\""):
			return &tools.LLMResponse{Content: `{"refersTo": "none", "name": ""}`}, nil
		case strings.Contains(prompt, "Phrase taken from it: \"she\"") && strings.Contains(prompt, "People the user already has: Anna"):
			return &tools.LLMResponse{Content: `{"refersTo": "known", "name": "Anna"}`}, nil
		}
		return &tools.LLMResponse{Content: `{"refersTo": "new", "name": ""}`}, nil
	case strings.Contains(req.SystemPrompt, "answers questions Moly asked earlier"):
		// the open-question judge: "ANSWERS-PENDING", or the style answer, answers the questions
		n := strings.Count(prompt, "\n1. ") + strings.Count(prompt, "\n2. ") + strings.Count(prompt, "\n3. ")
		flags := make([]string, n)
		for i := range flags {
			// a message about the style answers the style question; ANSWERS-PENDING answers everything
			flags[i] = fmt.Sprintf("%t", strings.Contains(prompt, "ANSWERS-PENDING") || strings.Contains(prompt, "The user's message: \"Quite informal"))
		}
		return &tools.LLMResponse{Content: `{"answered": [` + strings.Join(flags, ",") + `]}`}, nil
	case strings.Contains(req.SystemPrompt, "intent analyzer"):
		intent := "asking"
		switch {
		case strings.Contains(prompt, "User message: \"Hi Moly\""):
			intent = "greeting"
		case strings.Contains(prompt, "User message: \"Her name is Christine"):
			intent = "sharing"
		}
		// The scripted model flags a bare name answer only when Moly's question is in the history it was shown.
		answers := strings.Contains(prompt, "User message: \"Her name is Christine.\"") && strings.Contains(prompt, "Moly: Can you give me a name")
		confidence := 0.9
		if strings.Contains(prompt, "User message: \"Go ahead with what you have.\"") {
			confidence = 0.3 // the bare press looks unclear to the intent model (found live): it must not be asked about
		}
		return &tools.LLMResponse{Content: fmt.Sprintf(`{"intent": "%s", "confidence": %.1f, "answersQuestion": %t, "reasoning": "scripted"}`, intent, confidence, answers)}, nil
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
	case strings.Contains(req.SystemPrompt, "meta-instruction detector"):
		return &tools.LLMResponse{Content: `{"isMetaInstruction": false, "type": "none", "confidence": 0.9, "reasoning": "scripted"}`}, nil
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
	end := strings.Index(prompt, "\"\n\nExtract")
	if start < 0 || end < start {
		return extraction
	}
	out["intentionEvidence"] = prompt[start+len("Message: \""):end]
	b, _ := json.Marshal(out)
	return string(b)
}

func extractionFor(prompt string) string {
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
	case strings.Contains(prompt, "Her name is Christine"):
		// Hostile on purpose: the live model invented a goal from a bare name answer. The intent stage must drop it.
		return `{"intention": "identify who Christine is", "intentionPrinciples": [], "contact": {"name": "Christine", "label": "girl", "relationship": "other", "confidence": 0.9, "traits": []}, "userCharacteristics": [], "contactCharacteristics": {}}`
	default: // "I want to write a first message to a girl I saw on fetlife."
		return `{"intention": "initiate a conversation with the girl on fetlife", "intentionPrinciples": [], "contact": {"name": "", "label": "girl", "relationship": "other", "confidence": 0.85, "traits": []}, "userCharacteristics": [], "contactCharacteristics": {}}`
	}
}

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
	srv, err := NewAPIServer(scriptedLLM{}, db)
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
	srv, err := NewAPIServer(llm, db)
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
			if strings.Contains(prompt, gapQ) {
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
			if strings.Contains(prompt, gapQ) {
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

// S9: the user changes subject with no open gap. Rule: one question asks whether it is connected or a new focus, and it is asked once.
func TestScenarioS9SubjectShiftAskedOnce(t *testing.T) {
	srv, _, _, token := scenarioServer(t, scriptedLLM{
		shift: func() (string, error) {
			return `{"shift": true, "from": "the girl on fetlife", "to": "your mother", "reason": "new person", "confidence": 0.9}`, nil
		},
	})
	convID := startNamedGoal(t, srv, token)
	sendMessage(t, srv, token, convID, "Her name is Christine.")
	first, _, _ := sendMessage(t, srv, token, convID, "Actually, let's talk about my mother instead.")
	t.Logf("first=%q", first)
	if !strings.Contains(first, "mother") || !strings.Contains(first, "?") {
		t.Fatalf("a subject change must be asked about as a question that names the new subject, got %q", first)
	}
	if strings.Count(first, "?") != 1 {
		t.Fatalf("exactly one question, got %q", first)
	}
	if strings.Contains(first, "Hi Moly") {
		t.Fatalf("the previous subject must be what the model named, not message text: %q", first)
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
			if strings.Contains(prompt, gapQ) {
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
			case strings.Contains(prompt, gapB):
				return gapB, nil
			case strings.Contains(prompt, gapA):
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
			case strings.Contains(prompt, gapB):
				return gapB, nil
			case strings.Contains(prompt, gapA):
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
			if strings.Contains(prompt, gapQ) {
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
			if strings.Contains(prompt, gapQ) {
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
				if strings.Contains(prompt, q) {
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
	reply, _, _ := sendMessage(t, srv, token, convID, "It is good but it is too formal.")
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
			case strings.Contains(prompt, gapB):
				return gapB, nil
			case strings.Contains(prompt, gapA):
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
			if strings.Contains(prompt, gapA) {
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
