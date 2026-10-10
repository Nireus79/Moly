package tools

import (
	"moly/models"
	"testing"
)

func TestNewLayerContext(t *testing.T) {
	analysisCtx := &models.AnalysisContext{
		UserID:         "user1",
		ConversationID: "conv1",
		CurrentMessage: "Test message",
	}

	lc := NewLayerContext(analysisCtx, "user1", "msg1", "conv1")

	if lc == nil {
		t.Fatal("NewLayerContext returned nil")
	}

	if lc.UserID != "user1" {
		t.Errorf("Expected UserID='user1', got '%s'", lc.UserID)
	}

	if lc.GetMessage() != "Test message" {
		t.Errorf("Expected message='Test message', got '%s'", lc.GetMessage())
	}

	if lc.ShouldStop {
		t.Error("ShouldStop should be false initially")
	}
}

func TestLayerContextGetters(t *testing.T) {
	profile := &models.AboutMe{
		UserID:             "user1",
		CommunicationStyle: "direct",
	}

	contacts := []models.Contact{
		{
			Name:         "Alice",
			Relationship: "friend",
		},
	}

	analysisCtx := &models.AnalysisContext{
		UserID:              "user1",
		ConversationID:      "conv1",
		CurrentMessage:      "Hello Alice",
		UserProfile:         profile,
		RelevantContacts:    contacts,
		ExtractedConfidence: 0.95,
	}

	lc := NewLayerContext(analysisCtx, "user1", "msg1", "conv1")

	// Test getters
	if lc.GetMessage() != "Hello Alice" {
		t.Error("GetMessage failed")
	}

	if lc.GetUserProfile() != profile {
		t.Error("GetUserProfile failed")
	}

	if len(lc.GetRelevantContacts()) != 1 {
		t.Error("GetRelevantContacts failed")
	}

	if lc.GetExtractionConfidence() != 0.95 {
		t.Error("GetExtractionConfidence failed")
	}
}

func TestLayerContextMaturityScore(t *testing.T) {
	lc := NewLayerContext(&models.AnalysisContext{}, "user1", "msg1", "conv1")

	// Initially no Layer 3 result
	if lc.GetMaturityScore() != 0 {
		t.Error("Maturity score should be 0 initially")
	}

	// Set Layer 3 result
	lc.Layer3 = &Layer3Result{
		MaturityScore: 0.75,
	}

	if lc.GetMaturityScore() != 0.75 {
		t.Error("GetMaturityScore failed")
	}
}


func TestLayerContextStop(t *testing.T) {
	lc := NewLayerContext(&models.AnalysisContext{}, "user1", "msg1", "conv1")

	if lc.ShouldStop {
		t.Error("ShouldStop should be false initially")
	}

	// Stop execution
	lc.ShouldStop = true
	lc.StopReason = "test_stop"

	if !lc.ShouldStop {
		t.Error("ShouldStop should be true after setting")
	}

	if lc.StopReason != "test_stop" {
		t.Error("StopReason mismatch")
	}
}

func TestObviousHarm(t *testing.T) {
	lc := NewLayerContext(&models.AnalysisContext{}, "user1", "msg1", "conv1")

	if lc.IsObviousHarm() {
		t.Error("IsObviousHarm should be false initially")
	}

	// Set Layer 2 result with obvious harm
	lc.Layer2 = &Layer2Result{
		IsObviousHarm: true,
	}

	if !lc.IsObviousHarm() {
		t.Error("IsObviousHarm should be true")
	}
}

