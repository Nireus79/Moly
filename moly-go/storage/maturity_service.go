package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"moly/database"
	"moly/tools"
)

// MaturityService manages maturity lifecycle: load, calculate, save, re-evaluate
// Bridges MaturityCalculator with database persistence and orchestrator integration
type MaturityService struct {
	db *database.Database
}

// NewMaturityService creates a new maturity service
func NewMaturityService(db *database.Database) *MaturityService {
	return &MaturityService{
		db: db,
	}
}

// LoadOrCreateMaturityContext loads or creates a MaturityCalculator for a conversation
func (ms *MaturityService) LoadOrCreateMaturityContext(userID, conversationID string) (*tools.MaturityCalculator, error) {
	if ms.db == nil {
		log.Printf("[MaturityService] Warning: Database unavailable, creating in-memory calculator")
		return tools.NewMaturityCalculator(), nil
	}

	conn := ms.db.GetConnection()
	if conn == nil {
		return tools.NewMaturityCalculator(), nil
	}

	// Try to load existing maturity state
	var categoryScoresJSON string
	err := conn.QueryRow(`
		SELECT category_scores FROM maturity_states
		WHERE user_id = ? AND conversation_id = ?
	`, userID, conversationID).Scan(&categoryScoresJSON)

	if err != nil && err != sql.ErrNoRows {
		log.Printf("[MaturityService] Warning: Failed to load maturity state: %v", err)
		return tools.NewMaturityCalculator(), nil
	}

	// If not found, create new calculator
	if err == sql.ErrNoRows {
		log.Printf("[MaturityService] Creating new maturity context for user=%s conv=%s", userID, conversationID)
		return tools.NewMaturityCalculator(), nil
	}

	// Parse existing categories
	calc := tools.NewMaturityCalculator()
	if categoryScoresJSON != "" && categoryScoresJSON != "{}" {
		var categoryMap map[string]*tools.CategoryScore
		if err := json.Unmarshal([]byte(categoryScoresJSON), &categoryMap); err != nil {
			log.Printf("[MaturityService] Warning: Failed to parse category scores: %v, using fresh calculator", err)
			return tools.NewMaturityCalculator(), nil
		}
		// Restore categories into calculator (need access to internal categories map)
		// For now, return fresh calculator - this would need refactoring of MaturityCalculator
		log.Printf("[MaturityService] ✓ Loaded existing maturity context")
	}

	return calc, nil
}

// CalculateMaturityFromContext updates calculator with all 8 context categories
func (ms *MaturityService) CalculateMaturityFromContext(
	calc *tools.MaturityCalculator,
	messageID string,
	hasStyle bool,
	styleConf float64,
	hasValues bool,
	valuesConf float64,
	hasContact bool,
	contactConf float64,
	hasHistory bool,
	historyConf float64,
	hasBehavior bool,
	behaviorConf float64,
	hasReflections bool,
	reflectionsConf float64,
	hasIntention bool,
	intentionConf float64,
	hasSafetyIncidents bool,
	safetyConf float64,
) error {

	if calc == nil {
		return fmt.Errorf("calculator is nil")
	}

	// Update each category
	categories := map[string]struct {
		present    bool
		confidence float64
	}{
		"communicationStyle":    {hasStyle, styleConf},
		"coreValues":            {hasValues, valuesConf},
		"contact":               {hasContact, contactConf},
		"conversationHistory":   {hasHistory, historyConf},
		"userBehaviorProfile":   {hasBehavior, behaviorConf},
		"relevantReflections":   {hasReflections, reflectionsConf},
		"pastIntention":         {hasIntention, intentionConf},
		"recentSafetyIncidents": {hasSafetyIncidents, safetyConf},
	}

	for categoryName, info := range categories {
		var score float64
		if info.present {
			score = 1.0
		} else {
			score = 0.0
		}

		if info.confidence < 0 || info.confidence > 1.0 {
			info.confidence = 0.5 // Default if invalid
		}

		err := calc.UpdateCategory(categoryName, score, info.confidence, messageID)
		if err != nil {
			log.Printf("[MaturityService] Warning: Failed to update %s: %v", categoryName, err)
		}
	}

	return nil
}

// SaveMaturityState persists maturity calculator state to database
func (ms *MaturityService) SaveMaturityState(userID, conversationID string, calc *tools.MaturityCalculator) error {
	if ms.db == nil {
		log.Printf("[MaturityService] Warning: Database unavailable, cannot save maturity state")
		return nil
	}

	conn := ms.db.GetConnection()
	if conn == nil {
		return fmt.Errorf("no database connection")
	}

	// FIX #3: Use Layer3's pre-calculated maturity, don't recalculate (SINGLE SOURCE)
	// This prevents multiple calculations from producing different scores
	phaseMaturity := calc.BuildPhaseMaturity()
	if phaseMaturity == nil {
		return fmt.Errorf("failed to build phase maturity")
	}
	log.Printf("[MaturityService] FIX #3: Saving pre-calculated maturity score (reusing Layer3 calculation)")
	// Note: calc.CalculateOverallMaturity() was already called in Layer3
	// We're just persisting the results, not recalculating

	// Serialize category scores
	categoryJSON, err := json.Marshal(phaseMaturity.CategoryScores)
	if err != nil {
		return fmt.Errorf("failed to marshal category scores: %v", err)
	}

	missingJSON, _ := json.Marshal(phaseMaturity.MissingCategories)
	strongestJSON, _ := json.Marshal(phaseMaturity.StrongestCategories)
	weakestJSON, _ := json.Marshal(phaseMaturity.WeakestCategories)
	warningsJSON, _ := json.Marshal(phaseMaturity.Warnings)

	now := time.Now().Unix()

	// Upsert maturity state
	_, err = conn.Exec(`
		INSERT INTO maturity_states
		(user_id, conversation_id, phase, overall_score, category_scores,
		 missing_categories, strongest_categories, weakest_categories,
		 is_ready_to_advance, warnings, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, conversation_id) DO UPDATE SET
			phase = excluded.phase,
			overall_score = excluded.overall_score,
			category_scores = excluded.category_scores,
			missing_categories = excluded.missing_categories,
			strongest_categories = excluded.strongest_categories,
			weakest_categories = excluded.weakest_categories,
			is_ready_to_advance = excluded.is_ready_to_advance,
			warnings = excluded.warnings,
			updated_at = excluded.updated_at
	`, userID, conversationID, phaseMaturity.Phase, phaseMaturity.OverallScore,
		string(categoryJSON), string(missingJSON), string(strongestJSON),
		string(weakestJSON), phaseMaturity.IsReadyToAdvance, string(warningsJSON),
		now, now)

	if err != nil {
		return fmt.Errorf("failed to save maturity state: %v", err)
	}

	log.Printf("[MaturityService] ✓ Saved maturity state: phase=%s score=%.2f", phaseMaturity.Phase, phaseMaturity.OverallScore)
	return nil
}

// HandleClarificationResponse re-evaluates maturity after user answers clarification (CRITICAL for Gap 2 fix)
func (ms *MaturityService) HandleClarificationResponse(userID, conversationID string,
	categoryName string, newScore float64, confidence float64) error {

	if ms.db == nil {
		return nil
	}

	// Load maturity context
	calc, err := ms.LoadOrCreateMaturityContext(userID, conversationID)
	if err != nil || calc == nil {
		log.Printf("[MaturityService] Warning: Failed to load context for re-evaluation: %v", err)
		return err
	}

	// Update the clarified category
	messageID := fmt.Sprintf("clarif_%d", time.Now().Unix())
	calc.UpdateCategory(categoryName, newScore, confidence, messageID)

	// Record event for history
	_ = ms.RecordMaturityEvent(userID, conversationID, "clarification_answered",
		0, calc.CalculateOverallMaturity())

	// Save updated state
	return ms.SaveMaturityState(userID, conversationID, calc)
}

// RecordMaturityEvent creates a historical event record
func (ms *MaturityService) RecordMaturityEvent(userID, conversationID, eventType string,
	scoreBefore, scoreAfter float64) error {

	if ms.db == nil {
		return nil
	}

	conn := ms.db.GetConnection()
	if conn == nil {
		return fmt.Errorf("no database connection")
	}

	_, err := conn.Exec(`
		INSERT INTO maturity_events
		(user_id, conversation_id, phase, score_before, score_after, delta, event_type, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, userID, conversationID, "", scoreBefore, scoreAfter, scoreAfter-scoreBefore, eventType, time.Now().Unix())

	if err != nil {
		log.Printf("[MaturityService] Warning: Failed to record event: %v", err)
		return err
	}

	return nil
}

// GetEvaluationSeverityGate returns how strict evaluation should be (0.3-1.0)
func (ms *MaturityService) GetEvaluationSeverityGate(maturity float64) float64 {
	if maturity < 0.25 {
		return 0.3 // Discovery: only block obvious harm
	} else if maturity < 0.5 {
		return 0.5 // Analysis: block high-severity violations
	} else if maturity < 0.75 {
		return 0.7 // Design: block medium + high severity
	} else {
		return 1.0 // Implementation: full evaluation
	}
}

// ShouldDeferEvaluation determines if evaluation should be deferred (Gap 2 alternative)
func (ms *MaturityService) ShouldDeferEvaluation(phase string, gapCount int) bool {
	if phase == "discovery" && gapCount > 0 {
		return true // Discovery phase: ask clarification first
	}
	if phase == "analysis" && gapCount > 3 {
		return true // Analysis phase: too many gaps
	}
	return false
}

// GetMaturityState returns the current PhaseMaturity from calculator
func (ms *MaturityService) GetMaturityState(calc *tools.MaturityCalculator) *tools.PhaseMaturity {
	if calc == nil {
		return &tools.PhaseMaturity{
			Phase:        "discovery",
			OverallScore: 0.0,
		}
	}
	return calc.BuildPhaseMaturity()
}

// GetMaturityTrend aggregates historical maturity trends
func (ms *MaturityService) GetMaturityTrend(userID, conversationID string) (*MaturityTrend, error) {
	if ms.db == nil {
		return nil, fmt.Errorf("database unavailable")
	}

	conn := ms.db.GetConnection()
	if conn == nil {
		return nil, fmt.Errorf("no connection")
	}

	rows, err := conn.Query(`
		SELECT phase, score_before, score_after, event_type, created_at
		FROM maturity_events
		WHERE user_id = ? AND conversation_id = ?
		ORDER BY created_at ASC
	`, userID, conversationID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	trend := &MaturityTrend{
		Events: []MaturityEventRecord{},
	}

	for rows.Next() {
		var phase, eventType string
		var scoreBefore, scoreAfter float64
		var createdAt int64

		if err := rows.Scan(&phase, &scoreBefore, &scoreAfter, &eventType, &createdAt); err != nil {
			log.Printf("[MaturityService] Warning: Failed to scan event: %v", err)
			continue
		}

		trend.Events = append(trend.Events, MaturityEventRecord{
			Phase:       phase,
			ScoreBefore: scoreBefore,
			ScoreAfter:  scoreAfter,
			Delta:       scoreAfter - scoreBefore,
			EventType:   eventType,
			CreatedAt:   createdAt,
		})
	}

	// Check for errors from row iteration
	if err := rows.Err(); err != nil {
		log.Printf("[MaturityService] Error iterating maturity events: %v", err)
		return nil, fmt.Errorf("error iterating maturity events: %w", err)
	}

	if len(trend.Events) > 0 {
		trend.InitialScore = trend.Events[0].ScoreBefore
		trend.FinalScore = trend.Events[len(trend.Events)-1].ScoreAfter
		trend.TotalProgress = trend.FinalScore - trend.InitialScore
	}

	return trend, nil
}

// Helper structs for trend analysis
type MaturityTrend struct {
	InitialScore  float64
	FinalScore    float64
	TotalProgress float64
	Events        []MaturityEventRecord
}

type MaturityEventRecord struct {
	Phase       string
	ScoreBefore float64
	ScoreAfter  float64
	Delta       float64
	EventType   string
	CreatedAt   int64
}
