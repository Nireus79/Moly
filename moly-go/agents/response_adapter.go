package agents

import (
	"log"
	"sync"
)

// ResponseAdapter adjusts response generation based on detected context changes
// FIX #47-48: Adapt response strategy to user's current state
// FIX #53: Singleton pattern - no state, reuse instance
type ResponseAdapter struct{}

var (
	responseAdapterInstance *ResponseAdapter
	responseAdapterOnce     sync.Once
)

// GetResponseAdapter returns singleton instance (FIX #53)
func GetResponseAdapter() *ResponseAdapter {
	responseAdapterOnce.Do(func() {
		responseAdapterInstance = &ResponseAdapter{}
	})
	return responseAdapterInstance
}

// AdaptToIntention modifies response tone/strategy based on detected user intent (FIX #47)
func (ra *ResponseAdapter) AdaptToIntention(
	intent string,
	currentMessage string,
) map[string]interface{} {
	adaptation := make(map[string]interface{})

	switch intent {
	case "ask":
		// User seeking advice: direct, practical, actionable
		adaptation["tone"] = "directive"
		adaptation["strategy"] = "provide_practical_advice"
		adaptation["include_steps"] = true
		adaptation["be_direct"] = true
		log.Printf("[ResponseAdapter] FIX #47: Adapting to 'ask' intent - providing practical advice")

	case "vent":
		// User venting: validating, empathetic, listening
		adaptation["tone"] = "empathetic"
		adaptation["strategy"] = "validate_and_listen"
		adaptation["include_steps"] = false
		adaptation["be_direct"] = false
		adaptation["acknowledge_feelings"] = true
		log.Printf("[ResponseAdapter] FIX #47: Adapting to 'vent' intent - validating feelings")

	case "help_seek":
		// User seeking help working through: collaborative, questioning
		adaptation["tone"] = "collaborative"
		adaptation["strategy"] = "socratic_questioning"
		adaptation["include_steps"] = false
		adaptation["ask_clarifying_questions"] = true
		log.Printf("[ResponseAdapter] FIX #47: Adapting to 'help_seek' intent - socratic approach")

	case "share":
		// User sharing: interested, engaged, reflective
		adaptation["tone"] = "engaged"
		adaptation["strategy"] = "active_listening"
		adaptation["ask_followup"] = true
		adaptation["show_understanding"] = true
		log.Printf("[ResponseAdapter] FIX #47: Adapting to 'share' intent - active listening")

	default:
		// Default: balanced approach
		adaptation["tone"] = "balanced"
		adaptation["strategy"] = "flexible"
		log.Printf("[ResponseAdapter] FIX #47: Using balanced tone for intent: %s", intent)
	}

	return adaptation
}
