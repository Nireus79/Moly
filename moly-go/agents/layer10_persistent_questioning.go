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
type Layer10PersistentQuestioning struct {
	questioner *PersistentQuestioner
	llmClient  tools.LLMProvider
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

// NewLayer10PersistentQuestioning creates persistent questioning layer with LLM support
func NewLayer10PersistentQuestioning(llmClient tools.LLMProvider) *Layer10PersistentQuestioning {
	return &Layer10PersistentQuestioning{
		questioner: &PersistentQuestioner{
			maxTurns: 4,
		},
		llmClient: llmClient,
	}
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

// LoadOrCreateSession loads persistence session or creates new one
// FIX 2: Database tracking for multi-turn state
func (l10 *Layer10PersistentQuestioning) LoadOrCreateSession(userID, conversationID string) *PersistenceSession {
	// Foundation for database tracking
	// Current: Return new session (stub)
	// Future: Query persistence_sessions table
	//   SELECT question_count, previous_answers FROM persistence_sessions
	//   WHERE user_id=? AND conversation_id=?

	log.Printf("[Layer10] Session: user=%s, conv=%s (DB stub)", userID, conversationID)
	return &PersistenceSession{
		QuestionCount:       0,
		PreviousAnswers:     []string{},
		HasAcknowledgedHarm: false,
	}
}

// SaveSession persists session state to database
// FIX 2: Database tracking for multi-turn state
func (l10 *Layer10PersistentQuestioning) SaveSession(userID, conversationID string, session *PersistenceSession) error {
	// Foundation for database tracking
	// Current: Log and return nil (stub)
	// Future: INSERT OR REPLACE INTO persistence_sessions

	answersJSON, _ := json.Marshal(session.PreviousAnswers)
	log.Printf("[Layer10] Session save: q=%d, answers=%d (DB stub)",
		session.QuestionCount, len(session.PreviousAnswers))
	_ = answersJSON // Use variable to avoid unused error
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
