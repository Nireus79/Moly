package tools

import (
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
	ReadyThreshold    = 0.5 // Minimum maturity to proceed with full evaluation
	CompleteThreshold = 0.9 // Maturity indicating excellent context
	WarningThreshold  = 0.2 // Maturity below which extra caution needed
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


// IsComplete checks if category reached target
func (cs *CategoryScore) IsComplete() bool {
	return cs.CurrentScore >= cs.TargetScore
}

// PhaseMaturity represents complete maturity information for a conversation
type PhaseMaturity struct {
	Phase               string                    `json:"phase"`             // Current phase
	OverallScore        float64                   `json:"overallScore"`      // 0.0-1.0
	CategoryScores      map[string]*CategoryScore `json:"categoryScores"`    // Per-category breakdown
	TotalSpecs          int                       `json:"totalSpecs"`        // Total data points
	MissingCategories   []string                  `json:"missingCategories"` // Categories below threshold
	StrongestCategories []string                  `json:"strongestCategories"`
	WeakestCategories   []string                  `json:"weakestCategories"`
	IsReadyToAdvance    bool                      `json:"isReadyToAdvance"`
	Warnings            []string                  `json:"warnings"`
	LastUpdated         int64                     `json:"lastUpdated"`
}


// MaturityCalculator manages maturity calculation and re-evaluation
type MaturityCalculator struct {
	// Context categories (from Moly's existing system)
	categories map[string]*CategoryScore

	// PHASE 5: Track accumulated context - don't lose previous scores
	// Keep best score achieved so maturity actually improves across messages
	accumulatedBestScores map[string]float64 // Category -> highest score seen
}

// NewMaturityCalculator creates a new calculator with default categories
func NewMaturityCalculator() *MaturityCalculator {
	mc := &MaturityCalculator{
		categories:            initializeDefaultCategories(),
		accumulatedBestScores: make(map[string]float64),
	}
	// Initialize accumulated scores to 0
	for categoryName := range mc.categories {
		mc.accumulatedBestScores[categoryName] = 0.0
	}
	return mc
}

// initializeDefaultCategories sets up the 8 context categories from Moly
func initializeDefaultCategories() map[string]*CategoryScore {
	return map[string]*CategoryScore{
		"communicationStyle": {
			Category:     "communicationStyle",
			CurrentScore: 0.0,
			TargetScore:  1.0,
			Confidence:   0.0,
			SpecCount:    0,
		},
		"coreValues": {
			Category:     "coreValues",
			CurrentScore: 0.0,
			TargetScore:  1.0,
			Confidence:   0.0,
			SpecCount:    0,
		},
		"contact": {
			Category:     "contact",
			CurrentScore: 0.0,
			TargetScore:  1.0,
			Confidence:   0.0,
			SpecCount:    0,
		},
		"conversationHistory": {
			Category:     "conversationHistory",
			CurrentScore: 0.0,
			TargetScore:  1.0,
			Confidence:   0.0,
			SpecCount:    0,
		},
		"userBehaviorProfile": {
			Category:     "userBehaviorProfile",
			CurrentScore: 0.0,
			TargetScore:  1.0,
			Confidence:   0.0,
			SpecCount:    0,
		},
		"relevantReflections": {
			Category:     "relevantReflections",
			CurrentScore: 0.0,
			TargetScore:  1.0,
			Confidence:   0.0,
			SpecCount:    0,
		},
		"pastIntention": {
			Category:     "pastIntention",
			CurrentScore: 0.0,
			TargetScore:  1.0,
			Confidence:   0.0,
			SpecCount:    0,
		},
		"recentSafetyIncidents": {
			Category:     "recentSafetyIncidents",
			CurrentScore: 0.0,
			TargetScore:  1.0,
			Confidence:   0.0,
			SpecCount:    0,
		},
		"goalContextDeepening": {
			Category:     "goalContextDeepening",
			CurrentScore: 0.0,
			TargetScore:  1.0,
			Confidence:   0.0,
			SpecCount:    0,
		},
	}
}


// PHASE 4: Four-factor maturity calculation per MOLY_11_LAYER_SYSTEM.md spec
// These methods calculate maturity based on:
// 1. User profile completeness (AboutMe fields)
// 2. Contact/relationship clarity (contacts defined and clear)
// 3. Conversation depth (message count)
// 4. Extracted entities (count + confidence)

// CalculateProfileCompleteness scores user profile completeness
// Returns 0.0 (empty) to 1.0 (complete)
func (mc *MaturityCalculator) CalculateProfileCompleteness(profile interface{}) float64 {
	// If no profile, score is 0
	if profile == nil {
		return 0.0
	}

	// For now, check if profile exists and has data
	// Full implementation would check AboutMe fields:
	// - CommunicationStyle (filled? 0.25 points)
	// - Values (filled? 0.25 points)
	// - Approach (filled? 0.25 points)
	// - Preferences (filled? 0.25 points)

	// Return partial score if profile exists
	// Full implementation: count filled fields / 4
	return 0.5 // Placeholder: assumes profile exists but incomplete
}

// CalculateContactsClarity scores contact/relationship clarity
// Returns 0.0 (no contacts) to 1.0 (clear contacts)
func (mc *MaturityCalculator) CalculateContactsClarity(contactCount int, clearContactCount int) float64 {
	if contactCount == 0 {
		return 0.0
	}

	if clearContactCount == 0 {
		return 0.0
	}

	// Score based on clarity ratio
	clarity := float64(clearContactCount) / float64(contactCount)
	if clarity > 1.0 {
		clarity = 1.0
	}

	return clarity
}

// CalculateConversationDepth scores conversation depth by message count
// Returns 0.0 (1 message) to 1.0 (5+ messages)
// NOTE: Depth=0 on M1 is intentional - ensures Layer 8 only runs when conversation has evolved
func (mc *MaturityCalculator) CalculateConversationDepth(messageCount int) float64 {
	if messageCount <= 1 {
		return 0.0 // First message has no depth - enforces gap clarification before Socratic
	}

	if messageCount >= 5 {
		return 1.0
	}

	// Linear scale: (messageCount - 1) / 4
	return float64(messageCount-1) / 4.0
}

// CalculateEntityConfidence scores extracted entity quality
// Returns 0.0 (no entities) to 1.0 (high confidence entities)
func (mc *MaturityCalculator) CalculateEntityConfidence(entityCount int, averageConfidence float64) float64 {
	if entityCount == 0 {
		return 0.0
	}

	// FIX #71: Entity saturation was too aggressive
	// Old: Saturated at 10 entities, blocking maturity improvement
	// New: Saturate at 30 entities, allowing growth through message 2, 3, etc.
	// Scale: 10 entities = 0.33, 20 = 0.67, 30+ = 1.0
	countScore := float64(entityCount) / 30.0 // Saturate at 30 entities (was 10)
	if countScore > 1.0 {
		countScore = 1.0
	}

	// Weight by average confidence (0.5 = weak, 0.9+ = strong)
	return (countScore + averageConfidence) / 2.0
}

// CalculateOverallMaturityWithFactors computes maturity from 4 factors per spec
// PHASE 4: Implements 4-factor calculation
// Factors: profile + contacts + depth + entities (average)
func (mc *MaturityCalculator) CalculateOverallMaturityWithFactors(
	profile interface{},
	contactCount int,
	clearContactCount int,
	messageCount int,
	entityCount int,
	averageEntityConfidence float64,
) float64 {
	// Calculate each factor
	profileFactor := mc.CalculateProfileCompleteness(profile)
	contactsFactor := mc.CalculateContactsClarity(contactCount, clearContactCount)
	depthFactor := mc.CalculateConversationDepth(messageCount)
	entityFactor := mc.CalculateEntityConfidence(entityCount, averageEntityConfidence)

	// Log factor calculation
	log.Printf("[MaturityCalculator] PHASE 4: 4-factor calculation")
	log.Printf("[MaturityCalculator]   profile=%.2f contacts=%.2f depth=%.2f entities=%.2f",
		profileFactor, contactsFactor, depthFactor, entityFactor)

	// Average of 4 factors
	overall := (profileFactor + contactsFactor + depthFactor + entityFactor) / 4.0

	if overall > 1.0 {
		overall = 1.0
	}

	log.Printf("[MaturityCalculator] Overall maturity (4-factor): %.2f", overall)

	return overall
}

// CalculateOverallMaturity computes overall maturity from category scores
// PHASE 5: Uses accumulated best scores - maturity improves across messages
// Never loses previous context understanding, only gains
func (mc *MaturityCalculator) CalculateOverallMaturity() float64 {
	if len(mc.categories) == 0 {
		return 0.0
	}

	// For each category, use the BEST score achieved (accumulated)
	totalScore := 0.0
	activeCategories := 0

	for categoryName, category := range mc.categories {
		// Keep best score ever achieved, don't let it drop
		if category.CurrentScore > mc.accumulatedBestScores[categoryName] {
			mc.accumulatedBestScores[categoryName] = category.CurrentScore
			log.Printf("[MaturityCalculator] Accumulated: %s improved to %.2f", categoryName, category.CurrentScore)
		}

		// Use accumulated best score for calculation
		bestScore := mc.accumulatedBestScores[categoryName]
		if bestScore > 0 {
			// Weight by confidence: higher confidence = higher weight
			weightedScore := bestScore * category.Confidence
			totalScore += weightedScore
			activeCategories++
		}
	}

	if activeCategories == 0 {
		return 0.0
	}

	// Average of active categories, weighted by confidence
	// This accumulates - maturity grows as more categories are understood
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

// BuildPhaseMaturityWithFactors builds phase maturity using 4-factor calculation
// PHASE 4 FIX: Integrated 4-factor maturity calculation
func (mc *MaturityCalculator) BuildPhaseMaturityWithFactors(
	profile interface{},
	contactCount int,
	clearContactCount int,
	messageCount int,
	entityCount int,
	averageEntityConfidence float64,
) *PhaseMaturity {
	// Use 4-factor calculation per spec
	overallScore := mc.CalculateOverallMaturityWithFactors(
		profile,
		contactCount,
		clearContactCount,
		messageCount,
		entityCount,
		averageEntityConfidence,
	)

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





// countTotalSpecs returns total spec/data points across all categories
func (mc *MaturityCalculator) countTotalSpecs() int {
	total := 0
	for _, category := range mc.categories {
		total += category.SpecCount
	}
	return total
}
