package main

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"moly/agents"
	"moly/config"
	"moly/database"
	"moly/models"
	"moly/tools"
)

// Dependency Injection Framework
// ⚠️ DEPRECATED: These globals will be replaced by ServiceContainer in Phase 2.3
// Future: config.GetContainer().GetDatabase() and container access pattern
var appDB *database.Database

var apiServer *APIServer

// takeSkipOffer reads and clears whether the last reply of the conversation offered the skip button. Clearing at the start of
// every message means a refusal, an error or any early exit leaves the offer off: a press is accepted only right after a
// reply that carried the button.
func (srv *APIServer) takeSkipOffer(userID, conversationID string) bool {
	v, ok := srv.skipOffers.LoadAndDelete(contextCacheKey(userID, conversationID))
	offered, _ := v.(bool)
	return ok && offered
}

func (srv *APIServer) setSkipOffer(userID, conversationID string, offered bool) {
	srv.skipOffers.Store(contextCacheKey(userID, conversationID), offered)
}

// llmParts are the components of the server that call a model. All of them are built from ONE provider, so a set of them
// belongs to one user and can only ever use that user's key (tools.LLMSource). The server embeds the set it is working with.
type llmParts struct {
	llmClient                  tools.LLMProvider
	clarificationAgent         *agents.ClarificationAgent
	answerProcessor            *agents.AnswerProcessor
	incomingMessageAnalyzer    *agents.IncomingMessageAnalyzer
	conversationAnalyzer       *agents.ConversationAnalyzer
	agentSystem                *agents.AgentSystem
	constitutionalEvaluator    *tools.ConstitutionalEvaluator // Phase 1: deterministic constitutional evaluation
	contextExtractor           *agents.ContextExtractor
	conversationSummaryManager *tools.ConversationSummaryManager
	metaInstructionDetector    *agents.MetaInstructionDetector // recognizes "You are Moly" and similar self-reference
	intentDetector             *agents.LLMIntentDetector       // semantic classification of what the user is doing
	extractionPhase            *agents.ExtractionPhase
	unifiedOrchestrator        *agents.UnifiedOrchestrator
}

// APIServer wraps the agent system and database
type APIServer struct {
	llmParts // the model-calling components of the user this server value is working for (see forUser)

	database               *database.Database
	contactManager         *agents.ContactManager
	contextAttrManager     *agents.ContextAttributeManager
	constitution           *models.Constitution
	executionStateManager  *agents.ExecutionStateManager
	messageProcessingState *agents.MessageProcessingStateManager

	// Hybrid context infrastructure (Phase 1-5)
	analysisContextBuilder  *database.AnalysisContextBuilder
	conversationSummaryRepo *database.ConversationSummaryRepository
	messageSummaryRepo      *database.MessageSummaryRepository // Per-message summaries
	contextAttributeRepo    *database.ContextAttributeRepository

	beforeCommit func() error // test seam: returning an error makes the commit stage fail

	// Phase 0: Centralized extraction pipeline (Session 15 - Phase 1)
	conflictDetector *agents.ConflictDetector

	// CRITICAL: Phase 2 components (Layer 5 Conflict Handling)

	// Shared by every user's server value (pointers: forUser copies the struct)
	learningAgentCache      *sync.Map // map[userID]models.LearningAgent
	previousExtractionCache *sync.Map // map[string]*PreviousExtraction: latest extraction per conversation 
	skipOffers              *sync.Map // map[string]bool: whether the last reply of a conversation carried the skip button

	// Where providers come from. Every user gets their own set of model-calling components, built on first use.
	source    tools.LLMSource
	userParts *sync.Map // map[userID]*llmParts
}

// NewAPIServer creates a new API server. src hands out the model provider of each user; the server value returned is the one
// for nobody (its provider fails every call) and the handlers call forUser as soon as they know who is asking.
func NewAPIServer(src tools.LLMSource, db *database.Database) (*APIServer, error) {
	if db == nil {
		return nil, fmt.Errorf("database cannot be nil")
	}
	if src == nil {
		return nil, fmt.Errorf("an LLM source is required")
	}

	// NOTE: TemporaryFactStore is now created per-request with userID
	contactManager := agents.NewContactManager(database.NewContactRepository(db))
	contextAttrManager := agents.NewContextAttributeManager(database.NewContextAttributeRepository(db))

	// Load constitution (required for both ConversationAgent and ConstitutionalEvaluator)
	constitution, err := config.LoadConstitution("config/constitution.yaml")
	if err != nil {
		log.Fatalf("[Moly] FATAL: Failed to load constitution: %v\n\nPlease check:\n  - config/constitution.yaml exists and is valid\n  - config/ directory exists", err)
	}
	log.Printf("[Moly] ✓ Loaded constitution with %d principles and %d frameworks",
		len(constitution.SupremePrinciples), len(constitution.EthicalFrameworks))

	// Initialize hybrid context infrastructure (Phases 1-5)
	// CRITICAL FIX: DO NOT call Close() on db.GetConnection()!
	// GetConnection() returns the *sql.DB handle (entire database), not a single connection
	// Closing it closes the entire database and breaks all queries
	conn := db.GetConnection()
	contextAttributeRepo := database.NewContextAttributeRepository(db)
	conversationSummaryRepo := database.NewConversationSummaryRepository(conn)
	messageSummaryRepo := database.NewMessageSummaryRepository(conn) // Per-message summaries
	analysisContextBuilder := database.NewAnalysisContextBuilder(db, conversationSummaryRepo, contextAttributeRepo)
	log.Printf("[Moly] ✓ Initialized hybrid context infrastructure (manager, builder, message-summaries)")

	// Phase 0: centralized extraction pipeline; Layer 5 conflict handling (Phase 2). Neither calls a model.
	conflictDetector := agents.NewConflictDetector(db)

	srv := &APIServer{
		database:                db,
		contactManager:          contactManager,
		contextAttrManager:      contextAttrManager,
		constitution:            constitution,
		executionStateManager:   agents.NewExecutionStateManager(db),
		messageProcessingState:  agents.NewMessageProcessingStateManager(db),
		analysisContextBuilder:  analysisContextBuilder,
		conversationSummaryRepo: conversationSummaryRepo,
		messageSummaryRepo:      messageSummaryRepo, // Per-message summaries
		contextAttributeRepo:    contextAttributeRepo,
		conflictDetector:        conflictDetector,
		learningAgentCache:      &sync.Map{},
		previousExtractionCache: &sync.Map{},
		skipOffers:              &sync.Map{},
		source:                  src,
		userParts:               &sync.Map{},
	}
	// The server value for nobody: every model call fails with tools.ErrNoUser until a handler binds it to a user.
	parts, err := srv.buildLLMParts(src.Unbound())
	if err != nil {
		return nil, err
	}
	srv.llmParts = parts
	installMemoryHooks(db) // memory mode reads the maturity from the memory's questions
	return srv, nil
}

// buildLLMParts builds every component that calls a model from one provider.
func (srv *APIServer) buildLLMParts(llm tools.LLMProvider) (llmParts, error) {
	db, constitution := srv.database, srv.constitution

	clarificationAgent := agents.NewClarificationAgent(llm)
	answerProcessor := agents.NewAnswerProcessor(clarificationAgent, db)
	incomingMessageAnalyzer := agents.NewIncomingMessageAnalyzer(llm)

	// Initialize ConversationAgent (architecture) with Socratic support
	// Uses factory pattern with enforced initialization order:
	// 1. Create base agent with LLM
	// 2. Load and wire constitution
	// 3. Load and wire Socratic selector (BEFORE database)
	// 4. Wire database (AFTER Socratic selector)
	// 5. Verify readiness
	conversationAgent, err := agents.NewFullyInitializedConversationAgent(llm, db, "config/constitution.yaml", "config")
	if err != nil {
		return llmParts{}, fmt.Errorf("the conversation agent could not be initialized: %w", err)
	}

	// ConstitutionalEvaluator (Phase 1: deterministic-first ethical reasoning)
	constitutionalEvaluator := tools.NewConstitutionalEvaluator(llm, constitution)

	conversationAnalyzer := agents.NewConversationAnalyzer(llm, db)

	conversationSummarizer := tools.NewConversationSummarizer(llm)
	conversationSummaryManager := tools.NewConversationSummaryManager(db.GetConnection(), srv.conversationSummaryRepo, conversationSummarizer)

	metaInstructionDetector := agents.NewMetaInstructionDetector(llm)

	intentDetector := agents.NewLLMIntentDetector(llm)
	intentDetector.SetDatabase(db.GetConnection()) // Wire database for saving extraction analysis (Phase 5)

	// Create contextExtractor for combined extraction (eliminates duplicate LLM call)
	contextExtractor := agents.NewContextExtractor(llm)
	// Pass llmClient for subject attribution during extraction
	extractionPhase := agents.NewExtractionPhase(contextExtractor, srv.conflictDetector, db)

	// Unified 11-layer orchestrator 
	unifiedOrchestrator := agents.NewUnifiedOrchestrator(
		agents.NewContextExtractor(llm),
		constitutionalEvaluator,
		db,
		llm,
	)

	return llmParts{
		llmClient:                  llm,
		clarificationAgent:         clarificationAgent,
		answerProcessor:            answerProcessor,
		incomingMessageAnalyzer:    incomingMessageAnalyzer,
		conversationAnalyzer:       conversationAnalyzer,
		agentSystem:                &agents.AgentSystem{ConversationAgent: conversationAgent},
		constitutionalEvaluator:    constitutionalEvaluator,
		contextExtractor:           contextExtractor,
		conversationSummaryManager: conversationSummaryManager,
		metaInstructionDetector:    metaInstructionDetector,
		intentDetector:             intentDetector,
		extractionPhase:            extractionPhase,
		unifiedOrchestrator:        unifiedOrchestrator,
	}, nil
}

// forUser returns the server value to use for one user's request: the same server, with the model-calling components that were
// built from that user's provider (built on first use, then kept). A handler calls it as soon as it knows who is asking and
// uses only the result afterwards. If the components cannot be built it returns the server for nobody, whose model calls fail.
func (srv *APIServer) forUser(userID string) *APIServer {
	if strings.TrimSpace(userID) == "" {
		return srv
	}
	v, ok := srv.userParts.Load(userID)
	if !ok {
		parts, err := srv.buildLLMParts(stopsWithTurn{inner: srv.source.For(userID), userID: userID})
		if err != nil {
			log.Printf("[Moly] ERROR: the components for a user could not be built: %v", err)
			return srv
		}
		v, _ = srv.userParts.LoadOrStore(userID, &parts)
	}
	bound := *srv
	bound.llmParts = *(v.(*llmParts))
	return &bound
}

// GetLearningAgent returns a cached learning agent for the user, creating if necessary
func (srv *APIServer) GetLearningAgent(userID string) models.LearningAgent {
	// Check cache first
	if cached, ok := srv.learningAgentCache.Load(userID); ok {
		log.Printf("[MessageProcessor] ✓ Using cached learning agent for user %s", userID)
		return cached.(models.LearningAgent)
	}

	// Create new learning agent and cache it
	learningAgent, err := agents.NewLearningAgentWithDB(userID, srv.database)
	if err != nil {
		log.Printf("[MessageProcessor] Warning: Failed to initialize learning agent: %v", err)
		return nil
	}

	if learningAgent != nil {
		srv.learningAgentCache.Store(userID, learningAgent)
		log.Printf("[MessageProcessor] ✓ Created and cached learning agent for user %s", userID)
	}

	return learningAgent
}
