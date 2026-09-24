package agents

import (
	"log"
	"strings"
)

type DeterministicIntentDetector struct{}

type DeterministicIntent string

const (
	IntentBenign               DeterministicIntent = "benign"
	IntentFraudAdvice          DeterministicIntent = "seek_advice_on_committing_fraud"
	IntentIllegalAdvice        DeterministicIntent = "seek_advice_on_illegal_activity"
	IntentCreateWeapon         DeterministicIntent = "create_weapon"
	IntentCreateExplosive       DeterministicIntent = "create_explosive"
	IntentAcquireIllegalDrug    DeterministicIntent = "acquire_illegal_drug"
	IntentSelfHarm             DeterministicIntent = "seek_advice_on_self_harm"
	IntentViolence             DeterministicIntent = "seek_advice_on_violence"
	IntentExploitation         DeterministicIntent = "seek_advice_on_exploitation"
)

func NewDeterministicIntentDetector() *DeterministicIntentDetector {
	return &DeterministicIntentDetector{}
}

// Detect analyzes message structure to extract intent deterministically (no LLM)
// Returns specific intent type based on request patterns
func (d *DeterministicIntentDetector) Detect(message string) DeterministicIntent {
	if message == "" {
		return IntentBenign
	}

	lower := strings.ToLower(message)

	// Check for fraud-related patterns
	if d.hasPattern(lower, []string{"fraud", "scam", "embezzle", "money laundering"}, []string{"how", "where", "help", "advice", "guide", "instruct"}) {
		log.Printf("[DeterministicIntentDetector] Detected fraud intent")
		return IntentFraudAdvice
	}

	// Check for weapon/bomb creation patterns
	if d.hasPattern(lower, []string{"bomb", "explosive", "weapon", "gun", "rifle", "knife"}, []string{"make", "create", "build", "construct", "how to"}) {
		log.Printf("[DeterministicIntentDetector] Detected weapon/explosive intent")
		return IntentCreateWeapon
	}

	// Check for drug acquisition patterns
	if d.hasPattern(lower, []string{"heroin", "cocaine", "meth", "fentanyl", "mdma", "lsd"}, []string{"where", "how to get", "buy", "acquire", "find"}) {
		log.Printf("[DeterministicIntentDetector] Detected illegal drug intent")
		return IntentAcquireIllegalDrug
	}

	// Check for self-harm patterns
	if d.hasPattern(lower, []string{"kill myself", "suicide", "self-harm", "cut myself", "hurt myself"}, []string{"how", "advice", "help", "methods", "ways"}) {
		log.Printf("[DeterministicIntentDetector] Detected self-harm intent")
		return IntentSelfHarm
	}

	// Check for violence patterns
	if d.hasPattern(lower, []string{"kill", "hurt", "beat up", "stab", "shoot", "attack"}, []string{"how to", "advice on", "help with", "way to"}) {
		log.Printf("[DeterministicIntentDetector] Detected violence intent")
		return IntentViolence
	}

	// Check for illegal activity patterns
	if d.hasPattern(lower, []string{"hack", "exploit", "steal", "rob", "break in", "bypass"}, []string{"how to", "help me", "way to", "can i"}) {
		log.Printf("[DeterministicIntentDetector] Detected illegal activity intent")
		return IntentIllegalAdvice
	}

	// Check for exploitation patterns
	if d.hasPattern(lower, []string{"exploit", "manipulate", "coerce", "abuse", "trap"}, []string{"how to", "way to", "help me", "advice on"}) {
		log.Printf("[DeterministicIntentDetector] Detected exploitation intent")
		return IntentExploitation
	}

	log.Printf("[DeterministicIntentDetector] Intent: benign")
	return IntentBenign
}

// hasPattern checks if message contains any harmful keyword AND any request pattern
// Both must be present to trigger (reduces false positives)
func (d *DeterministicIntentDetector) hasPattern(lower string, harmfulKeywords []string, requestPatterns []string) bool {
	hasKeyword := false
	for _, keyword := range harmfulKeywords {
		if strings.Contains(lower, keyword) {
			hasKeyword = true
			break
		}
	}

	if !hasKeyword {
		return false
	}

	hasPattern := false
	for _, pattern := range requestPatterns {
		if strings.Contains(lower, pattern) {
			hasPattern = true
			break
		}
	}

	return hasPattern
}

// IsBenign is a convenience helper
func (d *DeterministicIntentDetector) IsBenign(intent DeterministicIntent) bool {
	return intent == IntentBenign
}

// IsHarmful returns true if intent is any harmful category
func (d *DeterministicIntentDetector) IsHarmful(intent DeterministicIntent) bool {
	return intent != IntentBenign
}
