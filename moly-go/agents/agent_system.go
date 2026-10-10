package agents

import (
	"moly/models"
)

// AgentSystem - Orchestrator for all 4 agents
type AgentSystem struct {
	ConversationAgent models.ConversationAgent
	LearningAgent     models.LearningAgent
	ContextManager    models.ContextManagerAgent
}
