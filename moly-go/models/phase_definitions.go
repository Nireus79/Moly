package models

// PhaseDefinition defines accomplishments for each conversation phase
type PhaseDefinition struct {
	Name     string   // Phase name: "initial", "gathering", "analysis", "help"
	Required []string // Required accomplishments (must all be complete to advance)
	Optional []string // Optional accomplishments (nice to have)
}

// PhaseDefinitions maps each phase to its accomplishments
var PhaseDefinitions = map[string]*PhaseDefinition{
	"initial": {
		Name: "initial",
		Required: []string{
			"goal_extracted",       // User's stated goal/need extracted
			"contact_identified",   // Who the message/advice is for/about
		},
		Optional: []string{
			"style_preference_mentioned", // Initial style hints (smart, playful, etc.)
		},
	},

	"gathering": {
		Name: "gathering",
		Required: []string{
			"user_style_extracted",      // User's communication style identified
			"user_values_extracted",     // User's values/principles identified
			"contact_profile_known",     // Understanding of contact (who they are)
		},
		Optional: []string{
			"constraints_identified",  // Limitations or boundaries mentioned
			"previous_attempts_known", // Previous attempts or context
			"user_self_described",     // User described their approach in detail
		},
	},

	"analysis": {
		Name: "analysis",
		Required: []string{
			"strategy_designed",         // Strategy/approach decided
			"decision_points_identified", // Key decisions identified
			"concerns_surfaced",         // Potential concerns discussed
		},
		Optional: []string{
			"user_confirmed_direction", // User confirmed the direction is right
			"alternatives_considered",  // Alternatives were discussed
		},
	},

	"help": {
		Name: "help",
		Required: []string{
			"response_drafted",  // Response/message/advice drafted
			"help_provided",     // Help delivered to user
		},
		Optional: []string{
			"user_accepted",      // User accepted the help
			"refinement_requested", // Refinements made if requested
		},
	},
}

// NewConversationMaturity creates a new maturity context with all phases initialized
func NewConversationMaturity(userID, conversationID string) *ConversationMaturity {
	cm := &ConversationMaturity{
		UserID:          userID,
		ConversationID:  conversationID,
		Phases:          make(map[string]*ConversationPhase),
		CurrentPhase:    "initial",
		OverallScore:    0.0,
		LastUpdated:     now(),
		CompletedPhases: []string{},
	}

	// Initialize all phases
	for phaseName, definition := range PhaseDefinitions {
		phase := &ConversationPhase{
			Name:            phaseName,
			Required:        definition.Required,
			Optional:        definition.Optional,
			Accomplishments: make(map[string]*PhaseAccomplishment),
		}

		// Initialize all accomplishments as incomplete
		for _, accName := range definition.Required {
			phase.Accomplishments[accName] = &PhaseAccomplishment{
				Name:        accName,
				Completed:   false,
				Confidence:  0.0,
				LastUpdated: now(),
			}
		}

		for _, accName := range definition.Optional {
			phase.Accomplishments[accName] = &PhaseAccomplishment{
				Name:        accName,
				Completed:   false,
				Confidence:  0.0,
				LastUpdated: now(),
			}
		}

		cm.Phases[phaseName] = phase
	}

	return cm
}

// GetPhaseDefinition returns the definition for a phase
func GetPhaseDefinition(phaseName string) *PhaseDefinition {
	return PhaseDefinitions[phaseName]
}

// GetAllPhaseNames returns all phase names in order
func GetAllPhaseNames() []string {
	return []string{"initial", "gathering", "analysis", "help"}
}
