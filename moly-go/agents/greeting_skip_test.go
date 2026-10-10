package agents

import (
	"testing"

	"moly/tools"
)

// PHASE 3: for a greeting, layers 4-10 have nothing to analyse (no goal, no gaps) and are skipped.
// Layer 2 (safety) and Layer 3 (maturity) are not in this list and still run.
func TestLayersSkipForGreeting(t *testing.T) {
	greeting := &tools.LayerContext{IsGreeting: true}
	skipping := map[string]bool{
		"Layer4":  NewLayer4GapDetector(nil).CanSkip(greeting),
		"Layer5":  NewLayer5UnifiedConflictDetection(nil, nil).CanSkip(greeting),
		"Layer6":  NewLayer6AmbiguousRequestHandler(nil).CanSkip(greeting),
		"Layer7":  NewLayer7PrincipleViolationClarification(nil).CanSkip(greeting),
		"Layer8":  NewLayer8SocraticDeepening().CanSkip(greeting),
		"Layer9":  NewLayer9TopicShiftDetection().CanSkip(greeting),
		"Layer10": NewLayer10PersistentQuestioning(nil, nil).CanSkip(greeting),
	}
	for name, skip := range skipping {
		if !skip {
			t.Errorf("%s must be skipped for a greeting", name)
		}
	}
}

func TestLayer4DoesNotSkipNonGreeting(t *testing.T) {
	if NewLayer4GapDetector(nil).CanSkip(&tools.LayerContext{}) {
		t.Error("Layer 4 must still run for a message that is not a greeting")
	}
}
