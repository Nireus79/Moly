package models

import "time"

// PhaseAccomplishment represents a single accomplishment within a conversation phase
type PhaseAccomplishment struct {
	Name        string  // "goal_extracted", "values_identified", etc.
	Completed   bool    // Has this accomplishment been achieved?
	Confidence  float64 // 0.0-1.0: how confident are we this is complete?
	LastUpdated int64   // Unix timestamp of last update
	Evidence    string  // What proves this is complete? (optional)
}

// ConversationPhase represents maturity for one phase
type ConversationPhase struct {
	Name            string                          // "initial", "gathering", "analysis", "help"
	Required        []string                        // Required accomplishment names
	Optional        []string                        // Optional accomplishment names
	Accomplishments map[string]*PhaseAccomplishment // Map of accomplishment name -> status
}

// ConversationMaturity tracks overall conversation maturity across all phases
type ConversationMaturity struct {
	UserID          string
	ConversationID  string
	Phases          map[string]*ConversationPhase // Map phase name -> phase state
	CurrentPhase    string                        // Which phase is user in? ("initial", "gathering", "analysis", "help")
	OverallScore    float64                       // 0.0-1.0: average of active phases
	LastUpdated     int64                         // Unix timestamp
	CompletedPhases []string                      // Phases that have been completed
}

// CalculatePhaseMaturity returns completion percentage for a phase
func (cp *ConversationPhase) CalculateMaturity() float64 {
	if cp == nil || len(cp.Accomplishments) == 0 {
		return 0.0
	}

	completed := 0
	total := len(cp.Required) + len(cp.Optional)

	if total == 0 {
		return 0.0
	}

	for _, acc := range cp.Accomplishments {
		if acc.Completed {
			completed++
		}
	}

	return float64(completed) / float64(total)
}

// GetCompletedCount returns number of completed accomplishments
func (cp *ConversationPhase) GetCompletedCount() int {
	if cp == nil {
		return 0
	}

	count := 0
	for _, acc := range cp.Accomplishments {
		if acc.Completed {
			count++
		}
	}

	return count
}

// GetTotalCount returns total number of accomplishments (required + optional)
func (cp *ConversationPhase) GetTotalCount() int {
	if cp == nil {
		return 0
	}

	return len(cp.Required) + len(cp.Optional)
}

// CalculateOverallMaturity returns average maturity of active phases (non-penalizing)
// Active phases = phases with at least one accomplishment tracked
func (cm *ConversationMaturity) CalculateOverallMaturity() float64 {
	if cm == nil || len(cm.Phases) == 0 {
		return 0.0
	}

	activePhases := []float64{}

	for _, phase := range cm.Phases {
		if phase != nil && len(phase.Accomplishments) > 0 {
			maturity := phase.CalculateMaturity()
			if maturity > 0 {
				activePhases = append(activePhases, maturity)
			}
		}
	}

	if len(activePhases) == 0 {
		return 0.0
	}

	sum := 0.0
	for _, m := range activePhases {
		sum += m
	}

	return sum / float64(len(activePhases))
}

// EstimateCurrentPhase returns which phase user is in based on overall maturity
func (cm *ConversationMaturity) EstimateCurrentPhase() string {
	if cm == nil {
		return "initial"
	}

	overall := cm.CalculateOverallMaturity()

	// Phase ranges (mirroring project phases):
	// initial: 0.0-0.25
	// gathering: 0.25-0.5
	// analysis: 0.5-0.75
	// help: 0.75-1.0

	if overall < 0.25 {
		return "initial"
	}
	if overall < 0.5 {
		return "gathering"
	}
	if overall < 0.75 {
		return "analysis"
	}
	return "help"
}

// MarkAccomplished marks an accomplishment as complete
func (cm *ConversationMaturity) MarkAccomplished(phaseName, accomplishmentName string) error {
	if cm == nil {
		return nil
	}

	phase, exists := cm.Phases[phaseName]
	if !exists {
		return nil // Silently skip if phase doesn't exist
	}

	if phase.Accomplishments == nil {
		phase.Accomplishments = make(map[string]*PhaseAccomplishment)
	}

	acc, exists := phase.Accomplishments[accomplishmentName]
	if !exists {
		// Create new accomplishment
		acc = &PhaseAccomplishment{
			Name:       accomplishmentName,
			Completed:  true,
			Confidence: 1.0,
		}
		phase.Accomplishments[accomplishmentName] = acc
	} else {
		// Update existing accomplishment
		acc.Completed = true
		acc.Confidence = 1.0
	}

	acc.LastUpdated = now()
	cm.LastUpdated = now()

	return nil
}

// Now returns current Unix timestamp
func now() int64 {
	return time.Now().Unix()
}
