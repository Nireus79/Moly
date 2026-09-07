package agents

import (
	"errors"

	"moly/models"
	"moly/tools"
)

// AgentSystem - Orchestrator for all 4 agents
type AgentSystem struct {
	ConversationAgent   models.ConversationAgent
	LearningAgent       models.LearningAgent
	ContextManager      models.ContextManagerAgent
	RiskMonitor         models.RiskMonitoringAgent
}

// NewAgentSystem - Create new agent system
func NewAgentSystem(llm *tools.LLMClient, userID string) (*AgentSystem, error) {
	// LLM client is optional - agents will have limited functionality without it
	if userID == "" {
		return nil, errors.New("userID cannot be empty")
	}

	conversationAgent, err := NewConversationAgent(llm)
	if err != nil {
		return nil, err
	}

	learningAgent, err := NewLearningAgent(userID)
	if err != nil {
		return nil, err
	}

	contextManager, err := NewContextManager(userID)
	if err != nil {
		return nil, err
	}

	riskMonitor, err := NewRiskMonitor(userID)
	if err != nil {
		return nil, err
	}

	return &AgentSystem{
		ConversationAgent:  conversationAgent,
		LearningAgent:      learningAgent,
		ContextManager:     contextManager,
		RiskMonitor:        riskMonitor,
	}, nil
}
