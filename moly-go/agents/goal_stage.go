package agents

// GoalState is the goal of the conversation after one message.
type GoalState struct {
	Locked     string // the conversation goal; set once, by the first message that states one
	Working    string // the goal every layer works on for this message
	JustLocked bool   // this message locked the goal
	Changed    bool   // this message states a goal different from the locked one
}

// ResolveGoal applies the goal rules (ORCHESTRATOR_DESIGN.md step 3). It is pure.
//   - A message that states no goal continues the locked goal.
//   - The first goal ever stated is locked; a greeting never states one, so it never locks.
//   - A later goal does not replace the locked goal. It is reported as Changed only when the model judged it a
//     different aim (relation); the same aim reworded, refined, or a step toward it is not a change.
func ResolveGoal(locked, stated string, relation GoalRelation) GoalState {
	switch {
	case stated == "":
		return GoalState{Locked: locked, Working: locked}
	case locked == "":
		return GoalState{Locked: stated, Working: stated, JustLocked: true}
	default:
		return GoalState{Locked: locked, Working: stated, Changed: relation == GoalDifferent}
	}
}
