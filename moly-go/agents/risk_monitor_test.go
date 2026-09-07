package agents_test

import (
	"testing"

	"moly/agents"
	"moly/models"
)

func setupRiskMonitor(t *testing.T) models.RiskMonitoringAgent {
	monitor, err := agents.NewRiskMonitor("user123")
	if err != nil {
		t.Fatalf("Failed to create risk monitor: %v", err)
	}

	return monitor
}

func TestRiskMonitorCreation(t *testing.T) {
	monitor := setupRiskMonitor(t)

	if monitor == nil {
		t.Error("Expected risk monitor")
	}
}

func TestRiskMonitorEmptyUserID(t *testing.T) {
	_, err := agents.NewRiskMonitor("")
	if err == nil {
		t.Error("Expected error for empty userID")
	}
}

func TestAssessRisk(t *testing.T) {
	monitor := setupRiskMonitor(t)

	assessment, err := monitor.AssessRisk("user123", "I want to go for a walk")
	if err != nil {
		t.Fatalf("AssessRisk() error = %v", err)
	}

	if assessment == nil {
		t.Error("Expected risk assessment")
	}

	if assessment.RiskLevel == "" {
		t.Error("Expected RiskLevel to be set")
	}

	if assessment.Message == "" {
		t.Error("Expected Message to be set")
	}
}

func TestAssessRiskEmptyMessage(t *testing.T) {
	monitor := setupRiskMonitor(t)

	_, err := monitor.AssessRisk("user123", "")
	if err == nil {
		t.Error("Expected error for empty message")
	}
}

func TestRiskMonitorDetectPatterns(t *testing.T) {
	monitor := setupRiskMonitor(t)

	profile, err := monitor.DetectPatterns("user123")
	if err != nil {
		t.Fatalf("DetectPatterns() error = %v", err)
	}

	if profile == nil {
		t.Error("Expected risk profile")
	}

	if profile.UserID != "user123" {
		t.Errorf("UserID = %s, want user123", profile.UserID)
	}

	if profile.RiskPatterns == nil {
		t.Error("Expected RiskPatterns to be initialized")
	}
}

func TestGenerateEducationalResponse(t *testing.T) {
	monitor := setupRiskMonitor(t)

	risk := models.RiskAssessment{
		Message: "Please consider the impact",
	}

	responses, err := monitor.GenerateEducationalResponse(risk)
	if err != nil {
		t.Fatalf("GenerateEducationalResponse() error = %v", err)
	}

	if responses == nil {
		t.Error("Expected responses")
	}
}

func TestTrackPattern(t *testing.T) {
	monitor := setupRiskMonitor(t)

	pattern := &models.RiskPattern{
		PatternType: "manipulation",
		Severity:    5,
	}

	err := monitor.TrackPattern("user123", pattern)
	if err != nil {
		t.Fatalf("TrackPattern() error = %v", err)
	}
}

func TestTrackPatternNil(t *testing.T) {
	monitor := setupRiskMonitor(t)

	err := monitor.TrackPattern("user123", nil)
	if err == nil {
		t.Error("Expected error for nil pattern")
	}
}

func TestGetUserRiskProfile(t *testing.T) {
	monitor := setupRiskMonitor(t)

	profile, err := monitor.GetUserRiskProfile("user123")
	if err != nil {
		t.Fatalf("GetUserRiskProfile() error = %v", err)
	}

	if profile == nil {
		t.Error("Expected risk profile")
	}

	if profile.UserID != "user123" {
		t.Errorf("UserID = %s, want user123", profile.UserID)
	}

	if profile.UpdatedAt == 0 {
		t.Error("Expected UpdatedAt to be set")
	}
}
