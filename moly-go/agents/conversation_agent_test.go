package agents_test

import (
	"os"
	"testing"
	"time"

	"moly/agents"
	"moly/models"
	"moly/tools"
)

func setupTestAgent(t *testing.T) (models.ConversationAgent, *tools.LLMClient) {
	os.Setenv("CLAUDE_API_KEY", "test-key")

	llm, err := tools.NewLLMClient()
	if err != nil {
		t.Fatalf("Failed to create LLM client: %v", err)
	}

	agent, err := agents.NewConversationAgent(llm)
	if err != nil {
		t.Fatalf("Failed to create conversation agent: %v", err)
	}

	return agent, llm
}

func TestConversationAgentCreation(t *testing.T) {
	defer os.Unsetenv("CLAUDE_API_KEY")
	agent, _ := setupTestAgent(t)

	if agent == nil {
		t.Error("Expected conversation agent, got nil")
	}
}

func TestConversationAgentRunValidContext(t *testing.T) {
	defer os.Unsetenv("CLAUDE_API_KEY")
	agent, _ := setupTestAgent(t)

	context := models.Context{
		AboutMe: &models.AboutMe{
			UserID:             "user1",
			CommunicationStyle: "friendly",
			CreatedAt:          time.Now().Unix(),
		},
		ContextQuality: "partial",
	}

	response, err := agent.Run(context)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if response == nil {
		t.Error("Expected response, got nil")
	}

	if response.Phase == "" {
		t.Error("Expected phase to be set")
	}

	if response.ProcessingTimeMs < 0 {
		t.Error("Expected non-negative processing time")
	}
}

func TestConversationAgentRunMissingAboutMe(t *testing.T) {
	defer os.Unsetenv("CLAUDE_API_KEY")
	agent, _ := setupTestAgent(t)

	context := models.Context{
		ContextQuality: "minimal",
	}

	_, err := agent.Run(context)
	if err == nil {
		t.Error("Expected error for missing AboutMe")
	}
}

func TestConversationAgentResponseStructure(t *testing.T) {
	defer os.Unsetenv("CLAUDE_API_KEY")
	agent, _ := setupTestAgent(t)

	context := models.Context{
		AboutMe: &models.AboutMe{
			UserID: "user1",
		},
	}

	response, _ := agent.Run(context)

	if response.Suggestions == nil {
		t.Error("Expected Suggestions to be initialized")
	}

	if response.Phase == "" {
		t.Error("Expected Phase to be set")
	}
}

func TestConversationAgentGracefulFallback(t *testing.T) {
	defer os.Unsetenv("CLAUDE_API_KEY")
	agent, _ := setupTestAgent(t)

	context := models.Context{
		AboutMe: &models.AboutMe{
			UserID: "user1",
		},
		ContextQuality: "minimal",
		Gaps:           []string{"contact_profile", "history"},
	}

	response, err := agent.Run(context)

	if err != nil {
		t.Fatalf("Run() should handle errors gracefully, got %v", err)
	}

	if response.Phase == "" {
		t.Error("Response should still have a phase even with minimal context")
	}

	if response.Suggestions == nil {
		t.Error("Suggestions should be initialized even on error")
	}
}
