package main

import "testing"

func TestIncidentSeverityIsOneTheTableAccepts(t *testing.T) {
	for in, want := range map[string]string{"immediate": "critical", "critical": "critical", "high": "high", "warning": "medium", "": "medium", "low": "low"} {
		if got := incidentSeverity(in); got != want {
			t.Errorf("incidentSeverity(%q) = %q, want %q", in, got, want)
		}
	}
}
