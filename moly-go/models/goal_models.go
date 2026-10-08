package models

// GoalProgression tracks how user's goals evolve across messages
// Enables multi-message understanding and prevents repeat questions
type GoalProgression struct {
	PrimaryGoal  string          `json:"primaryGoal"`            // Message 1 goal (locked, never changes)
	CurrentGoal  string          `json:"currentGoal"`            // This message's goal
	MessageGoals []string        `json:"messageGoals,omitempty"` // All goals by message index
	GoalAnswered map[string]bool `json:"goalAnswered,omitempty"` // Which goals were addressed
	CreatedAt    int64           `json:"createdAt,omitempty"`
	UpdatedAt    int64           `json:"updatedAt,omitempty"`
}

// GoalCoherence analyzes how current goal relates to primary goal
// Used by response generation to determine response strategy
// FIX #13: Wired to response strategy determination for multi-message goal tracking
type GoalCoherence struct {
	PrimaryGoal      string  `json:"primaryGoal"`      // Message 1 goal (locked)
	CurrentGoal      string  `json:"currentGoal"`      // This message's goal
	IsSameGoal       bool    `json:"isSameGoal"`       // Is user still pursuing Message 1 goal?
	GoalProgression  string  `json:"goalProgression"`  // "same", "related_subgoal", "new_subgoal", "different"
	ShouldAddressNew bool    `json:"shouldAddressNew"` // Help with new goal or focus on primary?
	Relationship     string  `json:"relationship"`     // How does current goal relate to primary?
	Confidence       float64 `json:"confidence"`       // How confident in this analysis?
}

// Gap represents missing information needed for decision-making
// FIX #72 Phase 2: Added GoalTarget to enable goal-aligned gap filtering
type Gap struct {
	Type        string  `json:"type"`        // gap type identifier
	Description string  `json:"description"` // human-readable description
	Severity    string  `json:"severity"`    // "high", "medium", "low"
	Confidence  float64 `json:"confidence"`  // 0-1
	Impact      float64 `json:"impact"`      // 0-1: how much does this block the goal?
	GoalTarget  string  `json:"goalTarget"`  // FIX #72 Phase 2: Which goal does this gap relate to? "primary_goal", "current_goal", or "both"
}
