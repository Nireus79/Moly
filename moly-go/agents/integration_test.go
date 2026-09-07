package agents_test

import (
	"os"
	"testing"
	"time"

	"moly/agents"
	"moly/models"
	"moly/tools"
)

func setupAgentSystem(t *testing.T) *agents.AgentSystem {
	os.Setenv("CLAUDE_API_KEY", "test-key")

	llm, err := tools.NewLLMClient()
	if err != nil {
		t.Fatalf("Failed to create LLM client: %v", err)
	}

	system, err := agents.NewAgentSystem(llm, "user123")
	if err != nil {
		t.Fatalf("Failed to create agent system: %v", err)
	}

	return system
}

func TestHappyPath(t *testing.T) {
	defer os.Unsetenv("CLAUDE_API_KEY")
	system := setupAgentSystem(t)

	if system == nil {
		t.Fatal("Expected agent system")
	}

	if system.ConversationAgent == nil {
		t.Error("ConversationAgent not initialized")
	}

	if system.LearningAgent == nil {
		t.Error("LearningAgent not initialized")
	}

	if system.ContextManager == nil {
		t.Error("ContextManager not initialized")
	}

	if system.RiskMonitor == nil {
		t.Error("RiskMonitor not initialized")
	}
}

func TestMissingContext(t *testing.T) {
	defer os.Unsetenv("CLAUDE_API_KEY")
	system := setupAgentSystem(t)

	context := models.Context{
		ContextQuality: "minimal",
		Gaps:           []string{"about_me", "contact_profile"},
	}

	response, err := system.ConversationAgent.Run(context)
	if err == nil {
		t.Error("Expected error for missing AboutMe")
	}

	if response != nil {
		t.Error("Expected nil response on error")
	}
}

func TestContextWithAboutMe(t *testing.T) {
	defer os.Unsetenv("CLAUDE_API_KEY")
	system := setupAgentSystem(t)

	context := models.Context{
		AboutMe: &models.AboutMe{
			UserID:            "user123",
			CommunicationStyle: "friendly",
			CreatedAt:          time.Now().Unix(),
		},
		ContextQuality: "partial",
	}

	response, err := system.ConversationAgent.Run(context)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if response == nil {
		t.Error("Expected response")
	}

	if response.Phase == "" {
		t.Error("Response missing phase")
	}
}

func TestLearningAgentIntegration(t *testing.T) {
	defer os.Unsetenv("CLAUDE_API_KEY")
	system := setupAgentSystem(t)

	profile, err := system.LearningAgent.GetUserProfile("user123")
	if err != nil {
		t.Fatalf("GetUserProfile() error = %v", err)
	}

	if profile == nil {
		t.Error("Expected profile")
	}

	data := models.InteractionData{
		UserID:         "user123",
		ConversationID: "conv1",
		UserMessage:    "Hello",
	}

	err = system.LearningAgent.RecordInteraction(data)
	if err != nil {
		t.Fatalf("RecordInteraction() error = %v", err)
	}
}

func TestContextManagerIntegration(t *testing.T) {
	defer os.Unsetenv("CLAUDE_API_KEY")
	system := setupAgentSystem(t)

	aboutMe := &models.AboutMe{
		UserID:            "user123",
		CommunicationStyle: "friendly",
	}

	err := system.ContextManager.SetAboutMe("user123", aboutMe)
	if err != nil {
		t.Fatalf("SetAboutMe() error = %v", err)
	}

	retrieved, err := system.ContextManager.GetAboutMe("user123")
	if err != nil {
		t.Fatalf("GetAboutMe() error = %v", err)
	}

	if retrieved == nil {
		t.Error("Expected AboutMe to be retrieved")
	}
}

func TestRiskMonitorIntegration(t *testing.T) {
	defer os.Unsetenv("CLAUDE_API_KEY")
	system := setupAgentSystem(t)

	riskProfile, err := system.RiskMonitor.GetUserRiskProfile("user123")
	if err != nil {
		t.Fatalf("GetUserRiskProfile() error = %v", err)
	}

	if riskProfile == nil {
		t.Error("Expected risk profile")
	}

	if riskProfile.UserID != "user123" {
		t.Errorf("UserID = %s, want user123", riskProfile.UserID)
	}
}
