package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"moly/tools"
)

// Layer10PersistentQuestioning handles user insistence on potentially harmful requests
// REFACTOR: Uses LLM for adaptive iterative questioning based on context and previous answers
// FIX 3: Now with database persistence for multi-turn state tracking
type Layer10PersistentQuestioning struct {
	questioner *PersistentQuestioner
	llmClient  tools.LLMProvider
	db         interface{} // database.Database interface
}

// PersistenceSession tracks questioning state across conversation turns
// PHASE 7: Enables adaptive next-probe generation based on previous answers
type PersistenceSession struct {
	QuestionCount       int      // How many probes have been asked (0-4)
	PreviousAnswers     []string // What user said to each probe
	HasAcknowledgedHarm bool     // Did user acknowledge the harm?
}

// PersistentQuestioner generates adaptive probing questions
type PersistentQuestioner struct {
	maxTurns int // Maximum turns before giving up (per spec: 3-4)
}

// NewLayer10PersistentQuestioning creates persistent questioning layer with LLM + DB support
func NewLayer10PersistentQuestioning(llmClient tools.LLMProvider, db interface{}) *Layer10PersistentQuestioning {
	l10 := &Layer10PersistentQuestioning{
		questioner: &PersistentQuestioner{
			maxTurns: 4,
		},
		llmClient: llmClient,
		db:        db,
	}

	// Ensure table exists
	l10.createTableIfNotExists()

	return l10
}

// Name returns the layer identifier
func (l10 *Layer10PersistentQuestioning) Name() string {
	return "Layer10-PersistentQuestioning"
}

// Priority returns layer priority
func (l10 *Layer10PersistentQuestioning) Priority() int {
	return 45 // Lower priority - only after other layers
}

// CanSkip returns true if not insisting on harmful request
func (l10 *Layer10PersistentQuestioning) CanSkip(lc *tools.LayerContext) bool {
	// Skip if Layer 7 didn't find violations
	if lc.Layer7 == nil || !lc.Layer7.ViolationDetected {
		return true
	}

	return false
}

// createTableIfNotExists creates persistence_sessions table if it doesn't exist
func (l10 *Layer10PersistentQuestioning) createTableIfNotExists() {
	if l10.db == nil {
		return
	}

	dbInterface, ok := l10.db.(interface {
		Exec(query string, args ...interface{}) (interface{}, error)
	})
	if !ok {
		return
	}

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS persistence_sessions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		conversation_id TEXT NOT NULL,
		question_count INTEGER DEFAULT 0,
		previous_answers TEXT,
		has_acknowledged_harm INTEGER DEFAULT 0,
		created_at INTEGER,
		updated_at INTEGER,
		UNIQUE(user_id, conversation_id)
	);
	CREATE INDEX IF NOT EXISTS idx_persistence_sessions_user_conv
		ON persistence_sessions(user_id, conversation_id);
	`

	_, err := dbInterface.Exec(createTableSQL)
	if err != nil {
		log.Printf("[Layer10] Warning: Failed to create table: %v", err)
	}
}

// LoadOrCreateSession loads persistence session or creates new one
// FIX 3: Actual database implementation
func (l10 *Layer10PersistentQuestioning) LoadOrCreateSession(userID, conversationID string) *PersistenceSession {
	if l10.db == nil {
		log.Printf("[Layer10] No DB - new session")
		return &PersistenceSession{
			QuestionCount:       0,
			PreviousAnswers:     []string{},
			HasAcknowledgedHarm: false,
		}
	}

	// Try to query existing session
	dbInterface, ok := l10.db.(interface {
		QueryRow(query string, args ...interface{}) interface {
			Scan(dest ...interface{}) error
		}
	})
	if !ok {
		return &PersistenceSession{QuestionCount: 0, PreviousAnswers: []string{}}
	}

	var questionCount int
	var answersJSON string
	var acknowledged int

	row := dbInterface.QueryRow(
		"SELECT question_count, previous_answers, has_acknowledged_harm FROM persistence_sessions WHERE user_id=? AND conversation_id=?",
		userID, conversationID)

	err := row.Scan(&questionCount, &answersJSON, &acknowledged)
	if err != nil {
		// No existing session - return new one
		log.Printf("[Layer10] New session: user=%s, conv=%s", userID, conversationID)
		return &PersistenceSession{
			QuestionCount:       0,
			PreviousAnswers:     []string{},
			HasAcknowledgedHarm: false,
		}
	}

	// Unmarshal previous answers
	var previousAnswers []string
	json.Unmarshal([]byte(answersJSON), &previousAnswers)

	log.Printf("[Layer10] Loaded session: q=%d, answers=%d, ack=%v",
		questionCount, len(previousAnswers), acknowledged > 0)

	return &PersistenceSession{
		QuestionCount:       questionCount,
		PreviousAnswers:     previousAnswers,
		HasAcknowledgedHarm: acknowledged > 0,
	}
}

// SaveSession persists session state to database
// FIX 3: Actual database implementation
func (l10 *Layer10PersistentQuestioning) SaveSession(userID, conversationID string, session *PersistenceSession) error {
	if l10.db == nil {
		return nil
	}

	dbInterface, ok := l10.db.(interface {
		Exec(query string, args ...interface{}) (interface{}, error)
	})
	if !ok {
		return nil
	}

	answersJSON, _ := json.Marshal(session.PreviousAnswers)
	acknowledged := 0
	if session.HasAcknowledgedHarm {
		acknowledged = 1
	}

	now := time.Now().Unix()
	sessionID := fmt.Sprintf("%s_%s", userID, conversationID)

	_, err := dbInterface.Exec(
		`INSERT OR REPLACE INTO persistence_sessions
		 (id, user_id, conversation_id, question_count, previous_answers, has_acknowledged_harm, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		sessionID, userID, conversationID, session.QuestionCount, string(answersJSON), acknowledged, now, now)

	if err != nil {
		log.Printf("[Layer10] Error saving session: %v", err)
		return err
	}

	log.Printf("[Layer10] ✓ Session saved: q=%d, answers=%d", session.QuestionCount, len(session.PreviousAnswers))
	return nil
}

// Process executes persistent questioning
// PHASE 7 + FIX 2: Generates ONE adaptive question, tracks state across messages
func (l10 *Layer10PersistentQuestioning) Process(ctx context.Context, lc *tools.LayerContext) (*tools.LayerContext, error) {
	startTime := time.Now()

	// FIX 2: Load persistence session (future: from database)
	session := l10.LoadOrCreateSession(lc.UserID, lc.ConversationID)
	log.Printf("[Layer10] Loaded: qCount=%d, acknowledged=%v",
		session.QuestionCount, session.HasAcknowledgedHarm)

	var nextQuestion string
	var shouldContinue bool

	if session.QuestionCount < l10.questioner.maxTurns {
		// Get previous answer if available
		previousAnswer := ""
		if len(session.PreviousAnswers) > 0 {
			previousAnswer = session.PreviousAnswers[len(session.PreviousAnswers)-1]
		}

		nextQuestion = l10.GenerateNextProbe(session.QuestionCount, previousAnswer, lc)
		shouldContinue = true
		log.Printf("[Layer10] Generating probe %d/%d (prev: %q)",
			session.QuestionCount+1, l10.questioner.maxTurns, previousAnswer)
	} else {
		log.Printf("[Layer10] Max turns (%d) reached - user insisting", l10.questioner.maxTurns)
		shouldContinue = false
	}

	// FIX 2: Save session state (future: to database)
	session.QuestionCount++
	_ = l10.SaveSession(lc.UserID, lc.ConversationID, session)

	// Store results
	lc.Layer10 = &tools.Layer10Result{
		PersistentQuestions: []string{nextQuestion},
		QuestionCount:       1,
		AllowResponse:       shouldContinue,
	}

	if shouldContinue {
		log.Printf("[Layer10] 🔄 Probe %d/%d (duration=%.2fs)",
			session.QuestionCount, l10.questioner.maxTurns, time.Since(startTime).Seconds())
	} else {
		log.Printf("[Layer10] ✗ Max turns reached - to L11 denial (duration=%.2fs)",
			time.Since(startTime).Seconds())
	}

	return lc, nil
}

// GenerateNextProbe creates adaptive next question via LLM
// REFACTOR: Dynamic questions based on sequence, previous answer, and user goal
func (l10 *Layer10PersistentQuestioning) GenerateNextProbe(questionNumber int, previousAnswer string, lc *tools.LayerContext) string {
	if l10.llmClient == nil {
		return l10.fallbackProbe(questionNumber, previousAnswer)
	}

	// Determine what aspect this question should focus on
	aspects := []string{
		"intent_and_belief",
		"affected_person_perspective",
		"consequences_and_impact",
		"values_and_alternatives",
	}

	aspect := ""
	if questionNumber < len(aspects) {
		aspect = aspects[questionNumber]
	}

	// Build context for LLM
	message := ""
	if lc.Analysis != nil {
		message = lc.Analysis.CurrentMessage
	}

	userGoal := ""
	if lc.Layer1 != nil && lc.Layer1.ExtractedContext != nil {
		userGoal = lc.Layer1.ExtractedContext.Intention
	}

	contextStr := ""
	if userGoal != "" {
		contextStr += fmt.Sprintf("User goal: %s\n", userGoal)
	}
	if previousAnswer != "" {
		contextStr += fmt.Sprintf("Previous answer: %s\n", previousAnswer)
	}
	contextStr += fmt.Sprintf("Question number: %d of 4\n", questionNumber+1)

	prompt := fmt.Sprintf(`You are Moly, a communication coach using Socratic method to help someone reconsider a decision.

User said: "%s"

%s
Task: Generate question #%d focusing on %s.

The goal is to help them think more deeply, not to judge. Build on their previous answer if available.

Requirements:
- Conversational and empathetic (1-2 sentences)
- Show genuine curiosity
- Ask about ONE thing only
- No brackets, no lectures
- Adapt to their previous answer
- No preamble

Generate ONLY the question.`, message, contextStr, questionNumber+1, aspect)

	resp, err := l10.llmClient.Call(context.Background(), &tools.LLMRequest{
		UserPrompt:  prompt,
		MaxTokens:   100,
		Temperature: 0.7,
	})

	if err != nil {
		log.Printf("[Layer10] LLM error for Q%d: %v - fallback", questionNumber+1, err)
		return l10.fallbackProbe(questionNumber, previousAnswer)
	}

	return resp.Content
}

// fallbackProbe returns static question if LLM unavailable
func (l10 *Layer10PersistentQuestioning) fallbackProbe(questionNumber int, previousAnswer string) string {
	switch questionNumber {
	case 0:
		return "Help me understand your thinking. What makes you believe this approach will work?"

	case 1:
		if strings.Contains(strings.ToLower(previousAnswer), "don't know") {
			return "That's honest. But how do you think the other person would actually react?"
		}
		return "How do you think the other person would feel about this?"

	case 2:
		if strings.Contains(strings.ToLower(previousAnswer), "okay") ||
		   strings.Contains(strings.ToLower(previousAnswer), "fine") {
			return "What if you're wrong about how they'd react? What if it damages your relationship?"
		}
		return "What do you think might happen as a result of this approach?"

	case 3:
		return "What's more important to you - achieving this goal or keeping their trust?"

	default:
		return "I think we've explored this thoroughly. Let's take a step back and reconsider."
	}
}

// GenerateProbes creates initial probing questions (deprecated)
func (pq *PersistentQuestioner) GenerateProbes(lc *tools.LayerContext) []string {
	questions := make([]string, 0)
	q1 := pq.GenerateNextProbe(0, "")
	questions = append(questions, q1)
	return questions
}

// GenerateNextProbe legacy method (kept for compatibility)
func (pq *PersistentQuestioner) GenerateNextProbe(questionNumber int, previousAnswer string) string {
	switch questionNumber {
	case 0:
		return "Help me understand your thinking. What makes you believe this approach will work?"
	case 1:
		if strings.Contains(strings.ToLower(previousAnswer), "don't know") {
			return "That's honest. But how do you think the other person would actually react?"
		}
		return "How do you think the other person would feel about this?"
	case 2:
		if strings.Contains(strings.ToLower(previousAnswer), "okay") ||
		   strings.Contains(strings.ToLower(previousAnswer), "fine") {
			return "What if you're wrong about how they'd react? What if it damages your relationship?"
		}
		return "What do you think might happen as a result of this approach?"
	case 3:
		return "What's more important to you - achieving this goal or keeping their trust?"
	default:
		return "I think we've explored this thoroughly. Let's take a step back and reconsider."
	}
}
