package tools

import (
	"fmt"
	"log"
	"time"
)

// Phase definitions - matches PoC design
const (
	PhaseDiscovery      = "discovery"      // 0.0-0.25: Learning about user
	PhaseAnalysis       = "analysis"       // 0.25-0.5: Understanding context
	PhaseDesign         = "design"         // 0.5-0.75: Refined understanding
	PhaseImplementation = "implementation" // 0.75-1.0: Full context, high confidence
)

var (
	PhaseRanges = map[string][2]float64{
		PhaseDiscovery:      {0.0, 0.25},
		PhaseAnalysis:       {0.25, 0.5},
		PhaseDesign:         {0.5, 0.75},
		PhaseImplementation: {0.75, 1.0},
	}

	PhaseOrder = []string{PhaseDiscovery, PhaseAnalysis, PhaseDesign, PhaseImplementation}

	// Thresholds (from PoC)
	ReadyThreshold     = 0.5  // Minimum maturity to proceed with full evaluation
	CompleteThreshold  = 0.9  // Maturity indicating excellent context
	WarningThreshold   = 0.2  // Maturity below which extra caution needed
)

// CategoryScore represents maturity of a specific context category
type CategoryScore struct {
	Category         string  `json:"category"`
	CurrentScore     float64 `json:"currentScore"`     // 0.0-1.0
	TargetScore      float64 `json:"targetScore"`      // Expected completion level
	Confidence       float64 `json:"confidence"`       // How confident are we in this score?
	SpecCount        int     `json:"specCount"`        // How many data points
	LastUpdated      int64   `json:"lastUpdated"`      // Unix timestamp
	UpdatedByMessage string  `json:"updatedByMessage"` // Which message/event updated this
}

// Percentage returns completion percentage (0-100)
func (cs *CategoryScore) Percentage() float64 {
	if cs.TargetScore == 0 {
		return 0.0
	}
	pct := (cs.CurrentScore / cs.TargetScore) * 100.0
	if pct > 100.0 {
		return 100.0
	}
	return pct
}

// IsComplete checks if category reached target
func (cs *CategoryScore) IsComplete() bool {
	return cs.CurrentScore >= cs.TargetScore
}

// PhaseMaturity represents complete maturity information for a conversation
type PhaseMaturity struct {
	Phase               string                    `json:"phase"`               // Current phase
	OverallScore        float64                   `json:"overallScore"`        // 0.0-1.0
	CategoryScores      map[string]*CategoryScore `json:"categoryScores"`      // Per-category breakdown
	TotalSpecs          int                       `json:"totalSpecs"`          // Total data points
	MissingCategories   []string                  `json:"missingCategories"`   // Categories below threshold
	StrongestCategories []string                  `json:"strongestCategories"`
	WeakestCategories   []string                  `json:"weakestCategories"`
	IsReadyToAdvance    bool                      `json:"isReadyToAdvance"`
	Warnings            []string                  `json:"warnings"`
	LastUpdated         int64                     `json:"lastUpdated"`
}

// MaturityEvent represents a maturity change in history
type MaturityEvent struct {
	Timestamp   int64                  `json:"timestamp"`
	Phase       string                 `json:"phase"`
	ScoreBefore float64                `json:"scoreBefore"`
	ScoreAfter  float64                `json:"scoreAfter"`
	Delta       float64                `json:"delta"`
	EventType   string                 `json:"eventType"`   // "clarification_answered", "context_extracted", "phase_advanced"
	Details     map[string]interface{} `json:"details"`     // Event-specific data
}

// MaturityCalculator manages maturity calculation and re-evaluation
type MaturityCalculator struct {
	// Context categories (from Moly's existing system)
	categories map[string]*CategoryScore
}

// NewMaturityCalculator creates a new calculator with default categories
func NewMaturityCalculator() *MaturityCalculator {
	return &MaturityCalculator{
		categories: initializeDefaultCategories(),
	}
}

// initializeDefaultCategories sets up the 8 context categories from Moly
func initializeDefaultCategories() map[string]*CategoryScore {
	return map[string]*CategoryScore{
		"communicationStyle": {
			Category:    "communicationStyle",
			CurrentScore: 0.0,
			TargetScore: 1.0,
			Confidence:  0.0,
			SpecCount:   0,
		},
		"coreValues": {
			Category:    "coreValues",
			CurrentScore: 0.0,
			TargetScore: 1.0,
			Confidence:  0.0,
			SpecCount:   0,
		},
		"contact": {
			Category:    "contact",
			CurrentScore: 0.0,
			TargetScore: 1.0,
			Confidence:  0.0,
			SpecCount:   0,
		},
		"conversationHistory": {
			Category:    "conversationHistory",
			CurrentScore: 0.0,
			TargetScore: 1.0,
			Confidence:  0.0,
			SpecCount:   0,
		},
		"userBehaviorProfile": {
			Category:    "userBehaviorProfile",
			CurrentScore: 0.0,
			TargetScore: 1.0,
			Confidence:  0.0,
			SpecCount:   0,
		},
		"relevantReflections": {
			Category:    "relevantReflections",
			CurrentScore: 0.0,
			TargetScore: 1.0,
			Confidence:  0.0,
			SpecCount:   0,
		},
		"pastIntention": {
			Category:    "pastIntention",
			CurrentScore: 0.0,
			TargetScore: 1.0,
			Confidence:  0.0,
			SpecCount:   0,
		},
		"recentSafetyIncidents": {
			Category:    "recentSafetyIncidents",
			CurrentScore: 0.0,
			TargetScore: 1.0,
			Confidence:  0.0,
			SpecCount:   0,
		},
		"goalContextDeepening": {
			Category:    "goalContextDeepening",
			CurrentScore: 0.0,
			TargetScore: 1.0,
			Confidence:  0.0,
			SpecCount:   0,
		},
	}
}

// UpdateCategory updates a single category score (event-driven re-evaluation)
func (mc *MaturityCalculator) UpdateCategory(categoryName string, score float64, confidence float64, messageID string) error {
	category, exists := mc.categories[categoryName]
	if !exists {
		return fmt.Errorf("unknown category: %s", categoryName)
	}

	// Update with new score and confidence
	category.CurrentScore = score
	category.Confidence = confidence
	category.SpecCount++
	category.LastUpdated = time.Now().Unix()
	category.UpdatedByMessage = messageID

	log.Printf("[MaturityCalculator] Updated %s: score=%.2f confidence=%.2f (event: %s)", categoryName, score, confidence, messageID)

	return nil
}

// CalculateOverallMaturity computes overall maturity from category scores
// Uses weighted average, matching PoC design
func (mc *MaturityCalculator) CalculateOverallMaturity() float64 {
	if len(mc.categories) == 0 {
		return 0.0
	}

	// Sum scores for categories with data
	totalScore := 0.0
	activeCategories := 0

	for _, category := range mc.categories {
		if category.CurrentScore > 0 {
			// Weight by confidence: higher confidence = higher weight
			weightedScore := category.CurrentScore * category.Confidence
			totalScore += weightedScore
			activeCategories++
		}
	}

	if activeCategories == 0 {
		return 0.0
	}

	// Average of active categories, weighted by confidence
	// This avoids penalizing users for starting new categories
	overallMaturity := totalScore / float64(activeCategories)

	if overallMaturity > 1.0 {
		overallMaturity = 1.0
	}

	return overallMaturity
}

// EstimateCurrentPhase estimates current phase based on maturity
func (mc *MaturityCalculator) EstimateCurrentPhase(maturity float64) string {
	// Handle 0-100 scale
	if maturity > 1.0 {
		maturity = maturity / 100.0
	}

	for _, phase := range PhaseOrder {
		min, max := PhaseRanges[phase][0], PhaseRanges[phase][1]
		if maturity >= min && maturity < max {
			return phase
		}
	}

	// Default to implementation
	return PhaseImplementation
}

// GetPhaseCompletionPercentage returns completion % within current phase
func (mc *MaturityCalculator) GetPhaseCompletionPercentage(maturity float64) int {
	if maturity > 1.0 {
		maturity = maturity / 100.0
	}

	currentPhase := mc.EstimateCurrentPhase(maturity)
	minVal, maxVal := PhaseRanges[currentPhase][0], PhaseRanges[currentPhase][1]
	phaseRange := maxVal - minVal

	if phaseRange == 0 {
		return 0
	}

	positionInPhase := maturity - minVal
	return int((positionInPhase / phaseRange) * 100)
}

// IdentifyWeakCategories returns categories below confidence threshold
func (mc *MaturityCalculator) IdentifyWeakCategories(weakThreshold float64) []string {
	weak := []string{}
	for name, category := range mc.categories {
		if category.Confidence < weakThreshold {
			weak = append(weak, name)
		}
	}
	return weak
}

// BuildPhaseMaturity builds complete phase maturity information
func (mc *MaturityCalculator) BuildPhaseMaturity() *PhaseMaturity {
	overallScore := mc.CalculateOverallMaturity()
	currentPhase := mc.EstimateCurrentPhase(overallScore)

	// Identify strong and weak categories
	strongestCategories := []string{}
	weakestCategories := mc.IdentifyWeakCategories(0.6)
	missingCategories := []string{}

	for name, category := range mc.categories {
		if category.CurrentScore >= 0.8 {
			strongestCategories = append(strongestCategories, name)
		}
		if category.CurrentScore == 0 {
			missingCategories = append(missingCategories, name)
		}
	}

	// Check if ready to advance
	isReady := overallScore >= ReadyThreshold && len(weakestCategories) <= 2

	warnings := []string{}
	if overallScore < WarningThreshold {
		warnings = append(warnings, "Very low maturity - user likely new or context sparse")
	}
	if len(missingCategories) > 3 {
		warnings = append(warnings, "Multiple critical gaps - consider targeted clarifications")
	}

	return &PhaseMaturity{
		Phase:               currentPhase,
		OverallScore:        overallScore,
		CategoryScores:      mc.categories,
		TotalSpecs:          mc.countTotalSpecs(),
		MissingCategories:   missingCategories,
		StrongestCategories: strongestCategories,
		WeakestCategories:   weakestCategories,
		IsReadyToAdvance:    isReady,
		Warnings:            warnings,
		LastUpdated:         time.Now().Unix(),
	}
}

// GetCategoryImprovement calculates improvement from before to after
func (mc *MaturityCalculator) GetCategoryImprovement(before map[string]float64) map[string]float64 {
	improvements := make(map[string]float64)
	for name, beforeScore := range before {
		if category, exists := mc.categories[name]; exists {
			improvements[name] = category.CurrentScore - beforeScore
		}
	}
	return improvements
}

// Snapshot captures current state for before/after comparison
func (mc *MaturityCalculator) Snapshot() map[string]float64 {
	snapshot := make(map[string]float64)
	for name, category := range mc.categories {
		snapshot[name] = category.CurrentScore
	}
	return snapshot
}

// CreateMaturityEvent creates an event record for history
func (mc *MaturityCalculator) CreateMaturityEvent(eventType string, scoreBefore, scoreAfter float64, details map[string]interface{}) *MaturityEvent {
	currentPhase := mc.EstimateCurrentPhase(scoreAfter)
	return &MaturityEvent{
		Timestamp: time.Now().Unix(),
		Phase:     currentPhase,
		ScoreBefore: scoreBefore,
		ScoreAfter:  scoreAfter,
		Delta:       scoreAfter - scoreBefore,
		EventType:   eventType,
		Details:     details,
	}
}

// countTotalSpecs returns total spec/data points across all categories
func (mc *MaturityCalculator) countTotalSpecs() int {
	total := 0
	for _, category := range mc.categories {
		total += category.SpecCount
	}
	return total
}

// GetReadinessLevel returns human-readable readiness level based on maturity
func (mc *MaturityCalculator) GetReadinessLevel(maturity float64) string {
	if maturity < WarningThreshold {
		return "insufficient"
	} else if maturity < ReadyThreshold {
		return "emerging"
	} else if maturity < CompleteThreshold {
		return "ready"
	} else {
		return "complete"
	}
}

// ShouldDeferEvaluation determines if evaluation should be deferred based on maturity
// (Replaces hardcoded gate at 0.5)
func (mc *MaturityCalculator) ShouldDeferEvaluation(maturity float64, gapCount int, currentPhase string) bool {
	// Defer if in discovery phase
	if currentPhase == PhaseDiscovery && maturity < 0.15 {
		return true
	}

	// Defer if in analysis phase with many gaps
	if currentPhase == PhaseAnalysis && gapCount > 3 {
		return true
	}

	// Otherwise, proceed with evaluation (but use maturity for gating severity)
	return false
}

// GetEvaluationSeverityGate returns how strict evaluation should be based on phase
// Lower maturity = more lenient (ask clarification instead of blocking)
func (mc *MaturityCalculator) GetEvaluationSeverityGate(maturity float64) float64 {
	// Returns a multiplier for how strict the evaluator should be
	// < 0.25: Very lenient (discovery)
	// 0.25-0.5: Lenient (analysis)
	// 0.5-0.75: Moderate (design)
	// > 0.75: Strict (implementation)

	if maturity < 0.25 {
		return 0.3 // Only block obvious harm
	} else if maturity < 0.5 {
		return 0.5 // Block high-severity violations
	} else if maturity < 0.75 {
		return 0.7 // Block medium + high severity
	} else {
		return 1.0 // Full evaluation
	}
}
