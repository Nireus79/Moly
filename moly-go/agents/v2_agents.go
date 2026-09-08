package agents

import (
	"errors"
	"log"

	"moly/database"
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

// NewAgentSystem - Create new agent system with database access
func NewAgentSystem(llm tools.LLMProvider, userID string, db *database.Database) (*AgentSystem, error) {
	log.Printf("[AgentSystem] Initializing for user %s", userID)

	// LLM client is optional - agents will have limited functionality without it
	if userID == "" {
		log.Printf("[AgentSystem] ERROR: userID cannot be empty")
		return nil, errors.New("userID cannot be empty")
	}

	log.Printf("[AgentSystem] Creating ConversationAgent (LLM available: %v)", llm != nil)
	conversationAgent, err := NewConversationAgent(llm)
	if err != nil {
		log.Printf("[AgentSystem] ERROR initializing ConversationAgent: %v", err)
		return nil, err
	}

	log.Printf("[AgentSystem] Creating LearningAgent")
	learningAgent, err := NewLearningAgentWithDB(userID, db)
	if err != nil {
		log.Printf("[AgentSystem] ERROR initializing LearningAgent: %v", err)
		return nil, err
	}

	log.Printf("[AgentSystem] Creating ContextManager")
	contextManager, err := NewContextManagerWithDB(userID, db)
	if err != nil {
		log.Printf("[AgentSystem] ERROR initializing ContextManager: %v", err)
		return nil, err
	}

	var riskMonitor models.RiskMonitoringAgent
	if llm != nil {
		log.Printf("[AgentSystem] Creating RiskMonitor with LLM")
		riskMonitorAgent, err := NewRiskMonitorWithLLM(userID, llm)
		if err != nil {
			log.Printf("[AgentSystem] ERROR initializing RiskMonitor with LLM: %v", err)
			return nil, err
		}
		riskMonitor = riskMonitorAgent
	} else {
		log.Printf("[AgentSystem] Creating RiskMonitor without LLM (heuristic mode)")
		riskMonitorAgent, err := NewRiskMonitor(userID)
		if err != nil {
			log.Printf("[AgentSystem] ERROR initializing RiskMonitor: %v", err)
			return nil, err
		}
		riskMonitor = riskMonitorAgent
	}

	log.Printf("[AgentSystem] Agent system initialized successfully for user %s", userID)
	return &AgentSystem{
		ConversationAgent:  conversationAgent,
		LearningAgent:      learningAgent,
		ContextManager:     contextManager,
		RiskMonitor:        riskMonitor,
	}, nil
}
