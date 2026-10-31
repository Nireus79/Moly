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

// Gap represents missing information needed for decision-making
// Phase 2: Added GoalTarget to enable goal-aligned gap filtering
type Gap struct {
	Type        string  `json:"type"`        // gap type identifier
	Description string  `json:"description"` // human-readable description
	Severity    string  `json:"severity"`    // "high", "medium", "low"
	Confidence  float64 `json:"confidence"`  // 0-1
	Impact      float64 `json:"impact"`      // 0-1: how much does this block the goal?
	GoalTarget  string  `json:"goalTarget"`  // Phase 2: Which goal does this gap relate to? "primary_goal", "current_goal", or "both"
}
