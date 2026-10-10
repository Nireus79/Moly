package agents

import "testing"

// ORDERING: a Socratic question is asked only after the deepening gates have passed.
// The ack_socratic workflow used to ask one even when the gates said "need clarification first".
func TestSocraticNotAskedWhenGatesSayWait(t *testing.T) {
	if socraticAllowed(true, false, true, true, true, true) {
		t.Fatal("Socratic question asked although shouldDeepen is false")
	}
}

func TestSocraticAskedWhenAllowed(t *testing.T) {
	if !socraticAllowed(true, true, true, true, true, true) {
		t.Fatal("Socratic question refused although every condition holds")
	}
}

func TestSocraticNotAskedForOtherWorkflows(t *testing.T) {
	if socraticAllowed(false, true, true, true, true, true) {
		t.Fatal("Socratic question asked outside the ack_socratic workflow")
	}
}
