package agents

import (
	"context"
	"log"
	"time"

	"moly/tools"
)

// Layer10PersistentQuestioning handles user insistence on potentially harmful requests
// Uses persistent questioning to help user reconsider rather than immediate rejection
type Layer10PersistentQuestioning struct {
	questioner *PersistentQuestioner
}

// PersistentQuestioner generates probing questions
type PersistentQuestioner struct {
	maxAttempts int
}

// NewLayer10PersistentQuestioning creates persistent questioning layer
func NewLayer10PersistentQuestioning() *Layer10PersistentQuestioning {
	return &Layer10PersistentQuestioning{
		questioner: &PersistentQuestioner{
			maxAttempts: 3, // Try up to 3 times
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

// Process executes persistent questioning
func (l10 *Layer10PersistentQuestioning) Process(ctx context.Context, lc *tools.LayerContext) (*tools.LayerContext, error) {
	startTime := time.Now()

	// Generate deeper probing questions
	questions := l10.questioner.GenerateProbes(lc)

	// Store results
	lc.Layer10 = &tools.Layer10Result{
		PersistentQuestions: questions,
		QuestionCount:       len(questions),
		AllowResponse:       len(questions) > 0,
	}

	if len(questions) > 0 {
		log.Printf("[Layer10] 🔄 Persistent questioning initiated (probes=%d, duration=%.2fs)",
			len(questions), time.Since(startTime).Seconds())
	} else {
		log.Printf("[Layer10] ✓ No further questioning needed (duration=%.2fs)", time.Since(startTime).Seconds())
	}

	return lc, nil
}

// GenerateProbes creates deeper probing questions
func (pq *PersistentQuestioner) GenerateProbes(lc *tools.LayerContext) []string {
	questions := make([]string, 0)

	// Deeper probing based on the violation
	questions = append(questions, "I understand you want this, but let me ask: how would this affect the other person?")
	questions = append(questions, "What do you think their reaction would be, and why do you think that?")
	questions = append(questions, "Is there something you're hoping will change through this approach?")

	return questions
}
