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

// GetMetaInstructionRespect returns constraints on response generation (FIX #48)
// FIX #56: Changed parameter type from map[string]int to map[string]bool
// (ContextChangeTracker now uses bool flags, not cumulative counters)
func (ra *ResponseAdapter) GetMetaInstructionRespect(
	metaInstructionLog map[string]bool,
) map[string]interface{} {
	constraints := make(map[string]interface{})

	// Build instruction set from current message instructions
	if metaInstructionLog["keep it focused"] && !metaInstructionLog["don't focus"] {
		constraints["stay_focused"] = true
		constraints["avoid_tangents"] = true
		log.Printf("[ResponseAdapter] FIX #56: Respecting 'keep focused' meta-instruction")
	}

	if metaInstructionLog["give advice"] && !metaInstructionLog["don't give advice"] {
		constraints["give_advice"] = true
		constraints["be_prescriptive"] = true
		log.Printf("[ResponseAdapter] FIX #56: Respecting 'give advice' meta-instruction")
	} else if metaInstructionLog["don't give advice"] {
		constraints["give_advice"] = false
		constraints["listen_only"] = true
		log.Printf("[ResponseAdapter] FIX #56: Respecting 'don't give advice' meta-instruction")
	}

	if metaInstructionLog["be direct"] && !metaInstructionLog["be casual"] {
		constraints["directness"] = "high"
		log.Printf("[ResponseAdapter] FIX #56: Respecting 'be direct' meta-instruction")
	} else if metaInstructionLog["be casual"] {
		constraints["directness"] = "low"
		log.Printf("[ResponseAdapter] FIX #56: Respecting 'be casual' meta-instruction")
	}

	if metaInstructionLog["just listen"] {
		constraints["listen_only"] = true
		constraints["validate"] = true
		constraints["give_advice"] = false
		log.Printf("[ResponseAdapter] FIX #56: Respecting 'just listen' meta-instruction")
	}

	return constraints
}
