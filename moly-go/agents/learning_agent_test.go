package agents_test

import (
	"testing"

	"moly/agents"
	"moly/models"
)

func TestLearningAgentCreation(t *testing.T) {
	agent, err := agents.NewLearningAgent("user123")
	if err != nil {
		t.Fatalf("Failed to create learning agent: %v", err)
	}

	if agent == nil {
		t.Error("Expected learning agent, got nil")
	}
}

func TestLearningAgentEmptyUserID(t *testing.T) {
	_, err := agents.NewLearningAgent("")
	if err == nil {
		t.Error("Expected error for empty userID")
	}
}

func TestGetUserProfile(t *testing.T) {
	agent, _ := agents.NewLearningAgent("user123")

	profile, err := agent.GetUserProfile("user123")
	if err != nil {
		t.Fatalf("GetUserProfile() error = %v", err)
	}

	if profile == nil {
		t.Error("Expected profile, got nil")
	}

	if profile.UserID != "user123" {
		t.Errorf("UserID = %s, want user123", profile.UserID)
	}

	if profile.CommunicationProfile == nil {
		t.Error("Expected CommunicationProfile to be initialized")
	}
}

func TestRecordInteraction(t *testing.T) {
	agent, _ := agents.NewLearningAgent("user123")

	data := models.InteractionData{
		UserID:         "user123",
		ConversationID: "conv1",
		UserMessage:    "Hello, how are you?",
	}

	err := agent.RecordInteraction(data)
	if err != nil {
		t.Fatalf("RecordInteraction() error = %v", err)
	}
}

func TestRecordInteractionEmptyUserID(t *testing.T) {
	agent, _ := agents.NewLearningAgent("user123")

	data := models.InteractionData{
		UserID:         "",
		ConversationID: "conv1",
		UserMessage:    "Hello",
	}

	err := agent.RecordInteraction(data)
	if err == nil {
		t.Error("Expected error for empty userID")
	}
}

func TestRecordSuggestionChoice(t *testing.T) {
	agent, _ := agents.NewLearningAgent("user123")

	data := models.SuggestionChoiceData{
		UserID:          "user123",
		ConversationID:  "conv1",
		SuggestionIndex: 0,
		ModifiedText:    "Modified version",
		UserFeedback:    "positive",
	}

	err := agent.RecordSuggestionChoice(data)
	if err != nil {
		t.Fatalf("RecordSuggestionChoice() error = %v", err)
	}
}

func TestBuildBehavioralProfile(t *testing.T) {
	agent, _ := agents.NewLearningAgent("user123")

	profile, err := agent.BuildBehavioralProfile("user123")
	if err != nil {
		t.Fatalf("BuildBehavioralProfile() error = %v", err)
	}

	if profile == nil {
		t.Error("Expected profile, got nil")
	}

	if profile.UserID != "user123" {
		t.Errorf("UserID = %s, want user123", profile.UserID)
	}

	if profile.Confidence < 0 || profile.Confidence > 1 {
		t.Errorf("Confidence = %f, want 0-1", profile.Confidence)
	}
}

func TestDetectPatterns(t *testing.T) {
	agent, _ := agents.NewLearningAgent("user123")

	patterns, err := agent.DetectPatterns("user123")
	if err != nil {
		t.Fatalf("DetectPatterns() error = %v", err)
	}

	if patterns == nil {
		t.Error("Expected patterns, got nil")
	}

	if patterns.UserID != "user123" {
		t.Errorf("UserID = %s, want user123", patterns.UserID)
	}
}

func TestNoContactSurveillance(t *testing.T) {
	agent, _ := agents.NewLearningAgent("user123")

	profile, _ := agent.BuildBehavioralProfile("user123")

	if profile.CommunicationProfile == nil {
		t.Fatal("Expected CommunicationProfile to exist")
	}

	if profile.CommunicationGoals == nil {
		t.Fatal("Expected CommunicationGoals to exist")
	}

	if profile.SuggestionChoices == nil {
		t.Fatal("Expected SuggestionChoices to exist")
	}

	for key := range profile.CommunicationProfile {
		if key == "contact_behavior" {
			t.Error("Profile contains forbidden contact_behavior tracking")
		}
	}
}
