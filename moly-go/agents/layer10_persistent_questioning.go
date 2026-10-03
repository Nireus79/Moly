package agents

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"moly/tools"
)

// Layer10PersistentQuestioning handles user insistence on potentially harmful requests
// Uses persistent questioning to help user reconsider rather than immediate rejection
// PHASE 7: Implements adaptive iterative questioning, not single-shot probes
type Layer10PersistentQuestioning struct {
	questioner *PersistentQuestioner
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

// NewLayer10PersistentQuestioning creates persistent questioning layer
func NewLayer10PersistentQuestioning() *Layer10PersistentQuestioning {
	return &Layer10PersistentQuestioning{
		questioner: &PersistentQuestioner{
			maxTurns: 4, // Per spec: 3-4 question turns before giving up
		},
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

		nextQuestion = l10.questioner.GenerateNextProbe(session.QuestionCount, previousAnswer)
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

// GenerateNextProbe creates adaptive next question based on sequence
// PHASE 7: Each question targets different aspect per spec
// Q1: Intent/belief, Q2: Affected person, Q3: Consequences, Q4: Values
func (pq *PersistentQuestioner) GenerateNextProbe(questionNumber int, previousAnswer string) string {
	switch questionNumber {
	case 0:
		// Question 1: Clarify intent/belief
		return "Help me understand your thinking. What makes you believe this approach will work?"

	case 1:
		// Question 2: Affected person's perspective
		// Could be adaptive based on previousAnswer, but for now, static
		if strings.Contains(strings.ToLower(previousAnswer), "don't know") {
			return "That's honest. But how do you think the other person would actually react?"
		}
		return "How do you think the other person would feel about this?"

	case 2:
		// Question 3: Consequences
		if strings.Contains(strings.ToLower(previousAnswer), "okay") ||
		   strings.Contains(strings.ToLower(previousAnswer), "fine") {
			return "What if you're wrong about how they'd react? What if it damages your relationship?"
		}
		return "What do you think might happen as a result of this approach?"

	case 3:
		// Question 4: Values and alternatives
		return "What's more important to you - achieving this goal or keeping their trust?"

	default:
		return "I think we've explored this thoroughly. Let's take a step back and reconsider."
	}
}

// GenerateProbes creates initial probing questions (deprecated, use GenerateNextProbe)
// Kept for backward compatibility, but PHASE 7 moves to single-question-at-a-time approach
func (pq *PersistentQuestioner) GenerateProbes(lc *tools.LayerContext) []string {
	questions := make([]string, 0)

	// Generate first probe only
	// Subsequent probes generated on next turn based on user response
	q1 := pq.GenerateNextProbe(0, "")
	questions = append(questions, q1)

	return questions
}
