package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"moly/agents"
	"moly/auth"
	"moly/config"
	"moly/database"
	"moly/models"
	"moly/monitoring"
	"moly/schema"
	"moly/tools"
)

// SECURITY: Message length limits (Issue #23: No message length limit)
const (
	MaxMessageLength       = 50000 // Maximum characters per message (DoS prevention)
	MaxClarificationLength = 10000 // Maximum characters per clarification response
)

// PHASE 2.2-2.3: Dependency Injection Framework
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

// PreviousExtraction stores extraction data from a previous message
// Used by FIX #1 to accumulate context across messages for Layer 5+ operations
// Used by FIX #4 to persist primary goal across messages
type PreviousExtraction struct {
	Entities       []models.ExtractedEntity
	Goal           string
	Values         []string
	PrimaryGoal    string   // The locked goal: set by the first goal stated, replaced only when the user confirms a switch
	PendingGoal    string   // A different goal the user stated, waiting for their answer to "change to it?"
	Progression    []string // Track goal evolution (FIX #4)
	ContextTracker *models.ContextTrackerState
}

// APIServer wraps the agent system and database
type APIServer struct {
	llmClient               tools.LLMProvider
	llmProvider             string // "ollama", "claude", or "openai"
	hardwareProfile         string // "fast", "standard", or "slow" - determines timeout strategy
	database                *database.Database
	contactManager          *agents.ContactManager
	contextAttrManager      *agents.ContextAttributeManager
	clarificationAgent      *agents.ClarificationAgent
	answerProcessor         *agents.AnswerProcessor
	incomingMessageAnalyzer *agents.IncomingMessageAnalyzer
	conversationAnalyzer    *agents.ConversationAnalyzer
	agentSystem             *agents.AgentSystem
	constitutionalEvaluator *tools.ConstitutionalEvaluator // Phase 1: deterministic constitutional evaluation
	constitution            *models.Constitution
	contextExtractor        *agents.ContextExtractor
	executionStateManager   *agents.ExecutionStateManager
	messageProcessingState  *agents.MessageProcessingStateManager

	// Hybrid context infrastructure (Phase 1-5)
	analysisContextBuilder     *database.AnalysisContextBuilder
	conversationSummaryManager *tools.ConversationSummaryManager
	conversationSummaryRepo    *database.ConversationSummaryRepository
	messageSummaryRepo         *database.MessageSummaryRepository // FIX #10: Per-message summaries
	chatMessageRepo            *database.ChatMessageRepository
	contextAttributeRepo       *database.ContextAttributeRepository

	// Maturity service for phase-based maturity system (C-30m redesign, C-30n integration)

	// Meta-instruction detector for self-awareness (recognizes "You are Moly", "Lace is my focus", etc.)
	metaInstructionDetector *agents.MetaInstructionDetector

	// Intent detector with entity extraction (semantic classification of entities)
	intentDetector *agents.LLMIntentDetector
	beforeCommit   func() error // test seam: returning an error makes the commit stage fail

	// LLM result cache to avoid redundant calls (Week 2 optimization)
	llmCache *tools.LLMCache

	// Phase 0: Centralized extraction pipeline (Session 15 - Phase 1)
	extractionStore  *tools.ExtractionStore
	conflictDetector *agents.ConflictDetector
	extractionPhase  *agents.ExtractionPhase

	// Cached agents (per-user cache to avoid recreation)
	learningAgentCache sync.Map // map[userID]models.LearningAgent

	// NEW: Phase 3 components (Constrained Generation)
	responseValidator      *agents.ResponseValidator
	constrainedResponseGen *tools.ConstrainedResponseGenerator

	// CRITICAL: Phase 2 components (Layer 5 Conflict Handling)
	layer5ConflictHandler *agents.Layer5ConflictHandler

	// NEW: Unified 11-Layer Orchestrator (Session 18)
	unifiedOrchestrator *agents.UnifiedOrchestrator

	// FIX #1: In-memory storage of latest extraction per conversation
	// Maps: "conversationID" -> *models.PreviousExtraction
	// Used to populate AccumulatedExtractedEntities, PreviousGoal for next message
	previousExtractionCache sync.Map // map[string]*PreviousExtraction
	skipOffers              sync.Map // map[string]bool: whether the last reply of a conversation carried the skip button
}

// NewAPIServer creates a new API server
func NewAPIServer(llm tools.LLMProvider, db *database.Database) (*APIServer, error) {
	if db == nil {
		return nil, fmt.Errorf("database cannot be nil")
	}

	// NOTE: TemporaryFactStore is now created per-request with userID
	contactManager := agents.NewContactManager(database.NewContactRepository(db))
	contextAttrManager := agents.NewContextAttributeManager(database.NewContextAttributeRepository(db))
	clarificationAgent := agents.NewClarificationAgent(llm)
	answerProcessor := agents.NewAnswerProcessor(clarificationAgent, db)
	incomingMessageAnalyzer := agents.NewIncomingMessageAnalyzer(llm)

	// Detect hardware profile for timeout strategy (Solution 3A)
	hardwareProfile := tools.DetectHardwareProfile()
	log.Printf("[Moly] ✓ Hardware detected: %s", hardwareProfile)

	// Load constitution (required for both ConversationAgent and ConstitutionalEvaluator)
	constitution, err := config.LoadConstitution("config/constitution.yaml")
	if err != nil {
		log.Fatalf("[Moly] FATAL: Failed to load constitution: %v\n\nPlease check:\n  - config/constitution.yaml exists and is valid\n  - config/ directory exists", err)
	}
	log.Printf("[Moly] ✓ Loaded constitution with %d principles and %d frameworks",
		len(constitution.SupremePrinciples), len(constitution.EthicalFrameworks))

	// Initialize ConversationAgent (architecture) with Socratic support
	// Uses factory pattern with enforced initialization order:
	// 1. Create base agent with LLM
	// 2. Load and wire constitution
	// 3. Load and wire Socratic selector (BEFORE database)
	// 4. Wire database (AFTER Socratic selector)
	// 5. Verify readiness
	conversationAgent, err := agents.NewFullyInitializedConversationAgent(
		llm,
		db,
		"config/constitution.yaml",
		"config",
	)
	if err != nil {
		log.Fatalf("[Moly] FATAL: Failed to initialize ConversationAgent: %v\n\nConversationAgent is critical to system operation. This is not optional.\nPlease check:\n  - config/constitution.yaml exists and is valid\n  - config/ directory has required files\n  - LLM client is properly initialized", err)
	}
	log.Printf("[Moly] ✅ ConversationAgent fully initialized with all dependencies properly wired")

	// CRITICAL FIX #1: Wire Layer5ConflictHandler into ConversationAgent
	// This enables Phase 2 conflict detection in the message pipeline
	// Will be set after layer5Handler is created, see below
	log.Printf("[Moly] ✓ ConversationAgent ready for Layer 5 handler injection")

	// Initialize ConstitutionalEvaluator (Phase 1: deterministic-first ethical reasoning)
	constitutionalEvaluator := tools.NewConstitutionalEvaluator(llm, constitution)
	log.Printf("[Moly] ✓ Initialized ConstitutionalEvaluator for unified principle-based evaluation")

	// Wrap ConversationAgent in a minimal AgentSystem struct
	agentSystem := &agents.AgentSystem{
		ConversationAgent: conversationAgent,
	}

	// Initialize ConversationAnalyzer for extracting insights from conversations
	conversationAnalyzer := agents.NewConversationAnalyzer(llm, db)

	// Initialize hybrid context infrastructure (Phases 1-5)
	// CRITICAL FIX: DO NOT call Close() on db.GetConnection()!
	// GetConnection() returns the *sql.DB handle (entire database), not a single connection
	// Closing it closes the entire database and breaks all queries
	conn := db.GetConnection()
	// REMOVED: defer conn.Close() - This was closing the entire database!
	chatMessageRepo := database.NewChatMessageRepository(conn)
	contextAttributeRepo := database.NewContextAttributeRepository(db)
	conversationSummaryRepo := database.NewConversationSummaryRepository(conn)
	messageSummaryRepo := database.NewMessageSummaryRepository(conn) // FIX #10: Per-message summaries

	conversationSummarizer := tools.NewConversationSummarizer(llm)
	conversationSummaryManager := tools.NewConversationSummaryManager(conn, conversationSummaryRepo, conversationSummarizer)

	analysisContextBuilder := database.NewAnalysisContextBuilder(db, conversationSummaryRepo, chatMessageRepo, contextAttributeRepo)

	log.Printf("[Moly] ✓ Initialized hybrid context infrastructure (summarizer, manager, builder, message-summaries)")

	// Initialize MetaInstructionDetector for self-awareness (Phase 0 of orchestrator)
	metaInstructionDetector := agents.NewMetaInstructionDetector(llm)
	log.Printf("[Moly] ✓ Initialized MetaInstructionDetector for conversation focus tracking")

	// Initialize IntentDetector for entity extraction with semantic classification
	intentDetector := agents.NewLLMIntentDetector(llm)
	intentDetector.SetDatabase(db.GetConnection()) // Wire database for saving extraction analysis (Phase 5)
	// NOTE: Connection is not explicitly closed here - it's returned to the pool
	log.Printf("[Moly] ✓ Initialized LLMIntentDetector for entity extraction and focus inference")

	// Initialize LLM cache for result caching (Week 2 optimization)
	llmCache := tools.NewDefaultLLMCache()
	log.Printf("[Moly] ✓ Initialized LLM cache (24h TTL, 10k entries max)")

	// Phase 0: Initialize centralized extraction pipeline (Session 15 - Phase 1)
	extractionStore := tools.NewExtractionStore()
	// NOTE: Do NOT defer extractionStore.Stop() here!
	// This function (NewAPIServer) returns after initialization, so defer would fire immediately
	// and close the extraction store while the server is still running.
	// ExtractionStore lifecycle should be tied to the container's cleanup, not this function's return.
	conflictDetector := agents.NewConflictDetector(db)
	// FIX #6: Create contextExtractor for combined extraction (eliminates duplicate LLM call)
	contextExtractor := agents.NewContextExtractor(llm)
	// FIX #69: Pass llmClient for subject attribution during extraction
	extractionPhase := agents.NewExtractionPhase(contextExtractor, extractionStore, conflictDetector, db, llm)
	log.Printf("[Moly] ✓ Initialized Phase 0 extraction pipeline with subject attribution")

	// NEW: Initialize Phase 3 Response Validator
	responseValidator := agents.NewResponseValidator(db)
	log.Printf("[Moly] ✓ Response validator initialized")

	// NEW: Initialize Constrained Response Generator
	constrainedResponseGen := tools.NewConstrainedResponseGenerator(
		llm,
		nil, // Will set responseGenerator reference later
		db,
		responseValidator,
	)
	log.Printf("[Moly] ✓ Constrained response generator initialized")

	// NEW: Initialize Layer 5 Conflict Handler (Phase 2) - WIRED TO CONVERSATION AGENT
	layer5Handler := agents.NewLayer5ConflictHandler(db)
	// CRITICAL FIX #1: Actually wire it into the conversationAgent
	if ca, ok := conversationAgent.(interface {
		SetLayer5ConflictHandler(*agents.Layer5ConflictHandler)
	}); ok {
		ca.SetLayer5ConflictHandler(layer5Handler)
		log.Printf("[Moly] ✓ Layer 5 conflict handler WIRED to ConversationAgent")
	} else {
		log.Printf("[Moly] ⚠️ Could not wire Layer5 handler - ConversationAgent doesn't implement SetLayer5ConflictHandler")
	}

	var llmProvider string = "unknown"
	if client, ok := llm.(*tools.LLMClient); ok {
		llmProvider = client.Provider
	}

	// Initialize UnifiedOrchestrator (Session 18 - All 11 layers wired)
	unifiedOrchestrator := agents.NewUnifiedOrchestrator(
		agents.NewContextExtractor(llm),
		constitutionalEvaluator,
		conflictDetector,
		layer5Handler,
		db,
		llm,
	)
	log.Printf("[Moly] ✅ UnifiedOrchestrator initialized with all 11 layers")

	return &APIServer{
		llmClient:                  llm,
		llmProvider:                llmProvider,
		hardwareProfile:            hardwareProfile,
		database:                   db,
		contactManager:             contactManager,
		contextAttrManager:         contextAttrManager,
		clarificationAgent:         clarificationAgent,
		answerProcessor:            answerProcessor,
		incomingMessageAnalyzer:    incomingMessageAnalyzer,
		conversationAnalyzer:       conversationAnalyzer,
		agentSystem:                agentSystem,
		constitutionalEvaluator:    constitutionalEvaluator,
		constitution:               constitution,
		contextExtractor:           contextExtractor, // FIX #6: Reuse created above (for combined extraction)
		executionStateManager:      agents.NewExecutionStateManager(db),
		messageProcessingState:     agents.NewMessageProcessingStateManager(db),
		analysisContextBuilder:     analysisContextBuilder,
		conversationSummaryManager: conversationSummaryManager,
		conversationSummaryRepo:    conversationSummaryRepo,
		messageSummaryRepo:         messageSummaryRepo, // FIX #10: Per-message summaries
		chatMessageRepo:            chatMessageRepo,
		contextAttributeRepo:       contextAttributeRepo,
		metaInstructionDetector:    metaInstructionDetector,
		intentDetector:             intentDetector,
		llmCache:                   llmCache,
		extractionStore:            extractionStore,
		conflictDetector:           conflictDetector,
		extractionPhase:            extractionPhase,
		responseValidator:          responseValidator,
		constrainedResponseGen:     constrainedResponseGen,
		layer5ConflictHandler:      layer5Handler,
		unifiedOrchestrator:        unifiedOrchestrator,
	}, nil
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

// savePreviousExtraction stores the extraction used as context for the next message.
// The database is the source of truth; the in-memory cache only avoids re-reading it.
func (srv *APIServer) savePreviousExtraction(userID, conversationID string, extraction *PreviousExtraction) {
	if srv.database == nil {
		log.Printf("[MessageProcessor] ERROR: database unavailable, extraction for conversation %s not saved", conversationID)
		return
	}
	turn := &turnCommit{}
	srv.savePreviousExtractionWith(turn, userID, conversationID, extraction)
	if err := turn.Run(srv.database.GetConnection()); err != nil {
		log.Printf("[MessageProcessor] ERROR: %v", err)
	}
}

// savePreviousExtractionWith adds the conversation's working state (including the locked goal) to the turn commit.
// The in-memory copy is filled only after the commit succeeds.
func (srv *APIServer) savePreviousExtractionWith(ex database.Executor, userID, conversationID string, extraction *PreviousExtraction) {
	if userID == "" || conversationID == "" || extraction == nil {
		return
	}
	state, err := json.Marshal(extraction)
	if err != nil {
		log.Printf("[MessageProcessor] ERROR: could not encode extraction for conversation %s: %v", conversationID, err)
		return
	}
	if err := database.NewConversationContextRepository(srv.database).SaveWith(ex, userID, conversationID, state); err != nil {
		log.Printf("[MessageProcessor] ERROR: %v", err)
		return
	}
	store := func() {
		srv.previousExtractionCache.Store(contextCacheKey(userID, conversationID), extraction)
		log.Printf("[MessageProcessor] Saved context for conversation %s (goal=%q, entities=%d)",
			conversationID, extraction.Goal, len(extraction.Entities))
	}
	if tc, ok := ex.(*turnCommit); ok {
		tc.AfterCommit(store)
	} else {
		store()
	}
}

func contextCacheKey(userID, conversationID string) string {
	return userID + "|" + conversationID
}

// loadPreviousExtraction returns the saved context for a conversation, or nil for a new conversation.
func (srv *APIServer) loadPreviousExtraction(userID, conversationID string) *PreviousExtraction {
	if userID == "" || conversationID == "" {
		return nil
	}
	if cached, ok := srv.previousExtractionCache.Load(contextCacheKey(userID, conversationID)); ok {
		return cached.(*PreviousExtraction)
	}
	if srv.database == nil {
		return nil
	}
	state, err := database.NewConversationContextRepository(srv.database).Load(userID, conversationID)
	if err != nil {
		log.Printf("[MessageProcessor] ERROR: %v", err)
		return nil
	}
	if state == nil {
		return nil
	}
	var extraction PreviousExtraction
	if err := json.Unmarshal(state, &extraction); err != nil {
		log.Printf("[MessageProcessor] ERROR: stored context for conversation %s is unreadable: %v", conversationID, err)
		return nil
	}
	srv.previousExtractionCache.Store(contextCacheKey(userID, conversationID), &extraction)
	log.Printf("[MessageProcessor] Loaded context for conversation %s (goal=%q, entities=%d)",
		conversationID, extraction.Goal, len(extraction.Entities))
	return &extraction
}

// FIX #9: mergeEntities combines accumulated and current entities, avoiding duplicates
// Keeps highest confidence version of duplicate entities by (type + value) key
func mergeEntities(accumulated, current []models.ExtractedEntity) []models.ExtractedEntity {
	if len(accumulated) == 0 {
		return current
	}
	if len(current) == 0 {
		return accumulated
	}

	// Map by (type:value) to detect duplicates
	entityMap := make(map[string]models.ExtractedEntity)

	// Add accumulated first
	for _, e := range accumulated {
		key := fmt.Sprintf("%s:%s", e.Type, e.Value)
		entityMap[key] = e
	}

	// Add/update with current (keep highest confidence)
	for _, e := range current {
		key := fmt.Sprintf("%s:%s", e.Type, e.Value)
		if existing, found := entityMap[key]; found {
			// Keep the one with higher confidence
			if e.Confidence > existing.Confidence {
				entityMap[key] = e
				log.Printf("[MessageProcessor] FIX #9: Updated %s:%s confidence (%.2f → %.2f)",
					e.Type, e.Value, existing.Confidence, e.Confidence)
			}
		} else {
			entityMap[key] = e
		}
	}

	// Convert back to slice
	merged := make([]models.ExtractedEntity, 0, len(entityMap))
	for _, e := range entityMap {
		merged = append(merged, e)
	}

	return merged
}

// FIX #10: buildMessageSummary creates a lightweight summary from extraction data
// This is saved immediately after extraction for future re-analysis optimization
func buildMessageSummary(
	messageID string,
	userID string,
	conversationID string,
	messageIndex int,
	role string,
	messageText string,
	epOutput *agents.ExtractionPhaseOutput,
	extractedContext *models.ExtractedContext,
) *models.MessageSummary {
	if epOutput == nil || epOutput.Artifact == nil {
		return nil
	}

	// Extract entity values and types
	var entityValues []string
	var entityTypes []string
	confidence := 0.0
	entityCount := 0

	for _, entity := range epOutput.Artifact.Entities {
		if entity.Confidence > 0.5 { // Only high-confidence entities
			entityValues = append(entityValues, entity.Value)
			entityTypes = append(entityTypes, entity.Type)
			confidence += entity.Confidence
			entityCount++
		}
	}

	// Average confidence
	if entityCount > 0 {
		confidence = confidence / float64(entityCount)
	}

	// Extract key phrases (first 3-5 entities)
	var keyPhrases []string
	for i, v := range entityValues {
		if i >= 5 {
			break
		}
		keyPhrases = append(keyPhrases, v)
	}

	// Get intention and style
	intention := ""
	tone := ""
	commStyle := ""
	if extractedContext != nil {
		intention = extractedContext.Intention
		if extractedContext.Style != nil {
			tone = extractedContext.Style.Style
			commStyle = extractedContext.Style.Style
		}
	}

	summary := &models.MessageSummary{
		MessageID:          messageID,
		UserID:             userID,
		ConversationID:     conversationID,
		MessageIndex:       messageIndex,
		Role:               role,
		MessageLength:      len(messageText),
		ExtractedEntities:  entityValues,
		EntityTypes:        entityTypes,
		Intention:          intention,
		KeyPhrases:         keyPhrases,
		Tone:               tone,
		CommunicationStyle: commStyle,
		Confidence:         confidence,
		ExtractionSource:   epOutput.Artifact.Source,
		TopicShift:         false, // Will be set by orchestrator if needed
		HasClarification:   false, // Will be set by orchestrator if needed
		ProcessedAt:        time.Now().Unix(),
		CreatedAt:          time.Now().Unix(),
		UpdatedAt:          time.Now().Unix(),
	}

	return summary
}

// respondJSON helper function
func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// detectPreviousMessageWasGreeting checks if the previous message in conversation history was a greeting
// getUserIDFromToken extracts userId from Bearer token (session ID)
func getUserIDFromToken(token string, db *database.Database) (string, error) {
	if token == "" {
		return "", fmt.Errorf("empty token")
	}

	// Token is stored as session ID, look it up in database
	conn := db.GetConnection()
	var userID string
	err := conn.QueryRow("SELECT user_id FROM sessions WHERE id = ? AND expires_at > ?", token, time.Now().Unix()).Scan(&userID)
	if err != nil {
		return "", fmt.Errorf("invalid token")
	}

	// UPDATE session last_used and user last_active for activity tracking
	now := time.Now().Unix()
	if _, err := conn.Exec("UPDATE sessions SET last_used = ? WHERE id = ?", now, token); err != nil {
		log.Printf("[Auth] Warning: Failed to update session last_used: %v", err)
	}
	if _, err := conn.Exec("UPDATE users SET last_active = ? WHERE id = ?", now, userID); err != nil {
		log.Printf("[Auth] Warning: Failed to update user last_active: %v", err)
	}

	return userID, nil
}

// extractAndValidateToken extracts and validates Bearer token from request
func extractAndValidateToken(r *http.Request, db *database.Database) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("Invalid token")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		return "", fmt.Errorf("Invalid token")
	}

	userID, err := getUserIDFromToken(token, db)
	if err != nil {
		return "", fmt.Errorf("Invalid token")
	}

	return userID, nil
}

// parseArrayFromStorage safely parses array field from database storage
func parseArrayFromStorage(stored string) []string {
	if stored == "" {
		return []string{}
	}

	var result []string
	if err := json.Unmarshal([]byte(stored), &result); err == nil {
		return result
	}
	return []string{}
}

// encodeArrayForStorage encodes array as JSON for safe storage
func encodeArrayForStorage(arr []string) string {
	if len(arr) == 0 {
		return "[]"
	}
	bytes, err := json.Marshal(arr)
	if err != nil {
		log.Printf("[Storage] Error encoding array: %v\n", err)
		return "[]"
	}
	return string(bytes)
}

// minInt returns the minimum of three integers
func minInt(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-ID")
		w.Header().Set("Access-Control-Max-Age", "86400")
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[HTTP] PANIC serving %s %s: %v\n%s", r.Method, r.URL.Path, rec, debug.Stack())
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "Internal server error"})
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// llmCallTimeout bounds one LLM call, including the slow CPU-only case. Calls are never cut short.
const llmCallTimeout = 20 * time.Minute

// previousEntities returns the entities from the previous extraction, or none for the first message in a conversation.
func previousEntities(prev *PreviousExtraction) []models.ExtractedEntity {
	if prev == nil {
		return nil
	}
	return prev.Entities
}

func handleStatus(db *database.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Health check - no authentication required (frontend needs to detect backend before login)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","version":"2.1"}`))
	}
}

// PHASE 3B: Clarification Response Helpers
// detectClarificationResponse checks if a message is answering a clarification question
func detectClarificationResponse(message string) bool {
	if message == "" {
		return false
	}

	lower := strings.ToLower(strings.TrimSpace(message))
	// Check if starts with A), B), C), D) or A., B., C., D.
	if len(lower) > 1 {
		if lower[0] >= 'a' && lower[0] <= 'd' {
			if lower[1] == ')' || lower[1] == '.' {
				return true
			}
		}
	}
	// Also check for "A) " pattern with space
	words := strings.Fields(lower)
	if len(words) > 0 {
		word := words[0]
		word = strings.TrimRight(word, ").")
		if len(word) == 1 && word[0] >= 'a' && word[0] <= 'd' {
			return true
		}
	}
	return false
}

// extractClarificationID tries to find a clarification ID in request metadata
func extractClarificationID(metadata map[string]interface{}) string {
	if metadata == nil {
		return ""
	}
	if id, ok := metadata["clarificationId"]; ok {
		if idStr, ok := id.(string); ok {
			return idStr
		}
	}
	return ""
}

// processClarificationResponse handles a user's response to a clarification question
// Returns: updated contacts, original message from clarification, error
func (srv *APIServer) processClarificationResponse(
	userID string,
	conversationID string,
	clarificationID string,
	userMessage string,
) ([]*models.Contact, string, error) {
	log.Printf("[Phase3B] Processing clarification response - ID: %s", clarificationID)

	// Load clarification from database
	clarRepo := database.NewClarificationQuestionRepository(srv.database)
	clarification, err := clarRepo.GetQuestion(userID, clarificationID)
	if err != nil {
		log.Printf("[Phase3B] Error loading clarification: %v", err)
		return nil, "", fmt.Errorf("failed to load clarification: %v", err)
	}

	if clarification == nil {
		log.Printf("[Phase3B] Clarification not found: %s", clarificationID)
		return nil, "", fmt.Errorf("clarification not found")
	}

	if clarification.Status != "active" {
		log.Printf("[Phase3B] Clarification no longer active (status=%s)", clarification.Status)
		return nil, "", fmt.Errorf("clarification already answered or expired")
	}

	log.Printf("[Phase3B] ✓ Loaded clarification: type=%s, options=%d", clarification.ClarificationType, len(clarification.Options))

	// Extract the original message from clarification context
	originalMessage := clarification.ContextNotes
	if originalMessage == "" {
		log.Printf("[Phase3B] Warning: Clarification has no original message context")
		originalMessage = userMessage // Fallback
	}

	// Load active contacts for this conversation
	contactRepo := database.NewContactRepository(srv.database)
	contacts, err := contactRepo.GetByUserID(userID)
	if err != nil {
		log.Printf("[Phase3B] Error loading contacts: %v", err)
		return nil, originalMessage, fmt.Errorf("failed to load contacts: %v", err)
	}

	// Filter to active contacts
	activeContacts := make([]*models.Contact, 0)
	for _, c := range contacts {
		if c.Status == "active" {
			activeContacts = append(activeContacts, c)
		}
	}

	log.Printf("[Phase3B] Found %d active contacts for conversation", len(activeContacts))

	// Process answer with ClarificationHandler
	handler := agents.NewClarificationHandler(srv.database)
	updatedContacts, err := handler.ProcessResponse(userID, conversationID, clarificationID, userMessage, activeContacts)
	if err != nil {
		log.Printf("[Phase3B] Error processing response: %v", err)
		return nil, originalMessage, fmt.Errorf("failed to process response: %v", err)
	}

	// Update contacts in database
	for _, contact := range updatedContacts {
		contact.Confidence = 0.99 // Mark as confirmed
		contact.Status = "active"
		contact.LastMentionedAt = time.Now().Unix()
		contact.UpdatedAt = time.Now().Unix()

		err := contactRepo.Update(contact)
		if err != nil {
			log.Printf("[Phase3B] Warning: Failed to update contact %s: %v", contact.Name, err)
		} else {
			log.Printf("[Phase3B] ✓ Updated contact: %s (confidence=0.99)", contact.Name)
		}
	}

	// Mark clarification as answered
	clarification.Status = "answered"
	clarification.AnsweredAt = time.Now().Unix()
	err = clarRepo.SaveQuestion(clarification)
	if err != nil {
		log.Printf("[Phase3B] Warning: Failed to update clarification status: %v", err)
	}

	log.Printf("[Phase3B] ✓ Processed clarification: %d contacts updated, clarification marked answered", len(updatedContacts))
	return updatedContacts, originalMessage, nil
}

// MessageProcessorHandler - Full orchestration with multi-phase context processing
func (srv *APIServer) MessageProcessorHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[MessageProcessorHandler] ★★★ HANDLER ENTRY - Method: %s Path: %s ★★★", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		schema.RespondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract and validate Bearer token
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		schema.RespondError(w, http.StatusUnauthorized, authErr.Error())
		return
	}

	req := &schema.Phase5Request{}

	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		schema.RespondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Validate using schema validator
	if err := schema.ValidateStruct(req); err != nil {
		schema.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// FIX #23: Message length limit (DoS prevention)
	if len(req.Message) > MaxMessageLength {
		log.Printf("[MessageProcessor] 🔴 REJECTED: Message too long (%d > %d chars)",
			len(req.Message), MaxMessageLength)
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Message too long (max %d characters, got %d)",
				MaxMessageLength, len(req.Message)),
		})
		return
	}
	if len(req.Message) < 1 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Message cannot be empty"})
		return
	}

	if len(req.ConversationID) > 100 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "ConversationID too long (max 100 characters)"})
		return
	}

	// PHASE 3B: Clarification Response Detection & Processing
	// Check if this message is answering a clarification question
	var isClarificationAnswer bool
	var clarificationOriginalMessage string
	var resolvedContacts []*models.Contact

	if detectClarificationResponse(req.Message) {
		clarificationID := extractClarificationID(req.Metadata)
		if clarificationID != "" {
			log.Printf("[MessageProcessor] ⚠️ PHASE 3B: Detected clarification response (A/B/C answer)")

			// Process the clarification response
			contacts, origMsg, err := srv.processClarificationResponse(userID, req.ConversationID, clarificationID, req.Message)
			if err != nil {
				log.Printf("[MessageProcessor] ⚠️ PHASE 3B: Error processing clarification: %v", err)
				// Continue with normal processing on error (degradation)
				isClarificationAnswer = false
			} else {
				isClarificationAnswer = true
				clarificationOriginalMessage = origMsg
				resolvedContacts = contacts
				log.Printf("[MessageProcessor] ✓ PHASE 3B: Clarification processed - will re-analyze original message with %d resolved contacts", len(resolvedContacts))
			}
		}
	}

	// SAVE USER MESSAGE early for conversation history
	// This needs to happen before conversation state is loaded so history includes this message
	userMessageID := fmt.Sprintf("msg_%d_%d", time.Now().Unix(), rand.Int63())
	userMessageForDB := req.Message

	// If this is a clarification answer, use the original message for analysis (not the A/B/C answer)
	if isClarificationAnswer && clarificationOriginalMessage != "" {
		log.Printf("[MessageProcessor] ✓ PHASE 3B: Re-analyzing with original message (contact resolution complete)")
		userMessageForDB = clarificationOriginalMessage
		// The resolved contacts are already in database from processClarificationResponse
	}

	// CREATE CONVERSATION IMMEDIATELY - before ANY code tries to use conversation_id
	// FK constraints in structured_context, message_processing_state, etc. require conversation to exist
	conversationID := req.ConversationID
	conn := srv.database.GetConnection()
	turn := &turnCommit{} // COMMIT STAGE (step 5): the writes of this turn, run in one transaction after the reply is resolved
	conversationJustCreated := false
	isNewBrowserSession := false

	if conversationID == "" || conversationID == "null" {
		// Create new conversation
		now := time.Now().Unix()
		conversationID = fmt.Sprintf("conv_%d_%d", now, time.Now().Nanosecond())

		_, err := conn.Exec(`
			INSERT INTO conversations (id, user_id, name, type, description, browser_session_id, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, conversationID, userID, "Direct Message", "direct", "Persistent conversation", req.BrowserSessionId, now, now)

		if err != nil && strings.Contains(err.Error(), "no column named browser_session_id") {
			_, err = conn.Exec(`
				INSERT INTO conversations (id, user_id, name, type, description, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`, conversationID, userID, "Direct Message", "direct", "Persistent conversation", now, now)
		}

		if err != nil {
			log.Printf("[MessageProcessor] Warning: Failed to create conversation: %v", err)
		} else {
			log.Printf("[MessageProcessor] ✓ Created NEW conversation: %s (IMMEDIATE, before meta-instruction check)", conversationID)
			conversationJustCreated = true
			isNewBrowserSession = true
		}
	}
	req.ConversationID = conversationID

	// The skip button (ORCHESTRATOR_DESIGN.md, "The skip button"): allowed only when something was answered before this
	// message, so it is decided before the open questions are resolved. A press that is not allowed is ignored.
	skipAllowed := false
	skipIgnored := false                                 // a press that is not allowed still carries only the fixed press words: they are no goal and no person
	offered := srv.takeSkipOffer(userID, conversationID) // the last reply must have carried the button
	if req.SkipQuestions && !conversationJustCreated {
		skipAllowed = offered && agents.WayOutOpen(agents.ConversationGapMaturity(srv.database.GetClarificationQuestionRepository(), conversationID))
		if !skipAllowed {
			log.Printf("[MessageProcessor] Skip pressed but not offered by the last reply, or nothing answered yet: ignored")
			skipIgnored = true
		}
	}
	// The questions still open in this conversation (asked and waiting, or skipped earlier) are resolved by this message only if
	// it answers them: the model judges that, one call. A question it does not answer stays open as a pending candidate, so
	// a reply on another topic never raises maturity. A skip press answers nothing: the waiting questions become pending.
	if !conversationJustCreated {
		chr := database.NewClarificationHistoryRepository(srv.database)
		open, err := chr.OpenQuestions(userID, conversationID)
		if err != nil {
			log.Printf("[MessageProcessor] Warning: could not load open clarifications: %v", err)
		} else if len(open) > 0 {
			answered := make([]bool, len(open))
			if !skipAllowed {
				texts := make([]string, len(open))
				for i, q := range open {
					texts[i] = q.Text
				}
				answered = agents.JudgePendingAnswered(context.Background(), srv.llmClient, req.Message, texts)
			}
			nAnswered, nPending := 0, 0
			for i, q := range open {
				switch {
				case answered[i]:
					if err := chr.SetQuestionStatus(userID, q.ID, "answered", time.Now().Unix()); err != nil {
						log.Printf("[MessageProcessor] Warning: could not close a clarification: %v", err)
					}
					nAnswered++
				case q.Status == "active":
					if err := chr.SetQuestionStatus(userID, q.ID, "skipped", 0); err != nil {
						log.Printf("[MessageProcessor] Warning: could not mark a clarification pending: %v", err)
					}
					nPending++
				}
			}
			log.Printf("[MessageProcessor] Open clarifications: %d answered by this message, %d asked and not answered (pending)", nAnswered, nPending)
		}
	}

	// PHASE 0: Meta-Instruction Detection (Self-Awareness)
	// Detect if message is about Moly's behavior/focus (e.g., "You are Moly", "Lace is my focus")
	// IMPORTANT: Update context but CONTINUE through orchestrator (don't return early)
	var metaInstruction *agents.MetaInstruction
	if req.Message != "" {
		metaInstruction = srv.metaInstructionDetector.Detect(context.Background(), req.Message)
		if metaInstruction != nil {
			log.Printf("[MetaInstruction] Detected: type=%s, confidence=%.2f, focus=%s",
				metaInstruction.Type, metaInstruction.Confidence, metaInstruction.TargetTopic)

			// Load or create structured context to update focus
			ctxRepo := srv.database.GetStructuredContextRepository()
			structuredCtx, _ := ctxRepo.LoadContext(userID, req.ConversationID)
			if structuredCtx == nil {
				structuredCtx = &models.StructuredContext{
					UserID:         userID,
					ConversationID: req.ConversationID,
					CreatedAt:      time.Now().Unix(),
				}
			}

			// Update focus based on meta-instruction
			if metaInstruction.Type == "focus" && metaInstruction.TargetTopic != "" {
				structuredCtx.ConversationFocus = metaInstruction.TargetTopic
				structuredCtx.FocusedPerson = metaInstruction.TargetTopic
				structuredCtx.UpdatedAt = time.Now().Unix()
				if err := ctxRepo.UpdateContext(structuredCtx); err != nil {
					log.Printf("[MetaInstruction] Warning: Failed to save focus: %v", err)
				}
				log.Printf("[MetaInstruction] ✓ Updated conversation focus: %s", metaInstruction.TargetTopic)
			}

			// FIXED: Continue through orchestrator instead of returning early
			// ConversationAgent will handle meta-instruction acknowledgment naturally
			// while running through full 11-layer pipeline
			log.Printf("[MetaInstruction] ✓ Continuing through orchestrator (not short-circuiting)")
		}
	}

	// Load or create message processing state for execution deduplication
	// This enables retries to skip already-completed pipeline stages
	// Fix O: Only create after meta-instruction check (meta-instructions bypass the pipeline)
	msgProcState, procStateErr := srv.messageProcessingState.GetOrCreateState(userID, req.ConversationID, userMessageID)
	if procStateErr != nil {
		log.Printf("[MessageProcessor] Warning: Failed to load/create message processing state: %v", procStateErr)
		// Don't fail the request - just continue without deduplication
		msgProcState = nil
	}

	// Track if safety alert was detected (to include in response)
	var safetyAlertDetected *models.SafetyAlert
	var layerCtx *tools.LayerContext // Orchestrator results

	// MESSAGE PREPROCESSING: Chunk large messages for processing (Week 2 optimization)
	// This prevents LLM timeouts on very large messages
	var processedMessage string = req.Message
	if len(req.Message) > 2000 {
		chunker := tools.NewMessageChunker()
		chunks := chunker.Chunk(req.Message)
		if len(chunks) > 1 {
			log.Printf("[MessageProcessor] ⚠ Large message (%d bytes) chunked into %d pieces", len(req.Message), len(chunks))
			// For now, rejoin chunks (Week 2 Part 1 - can be enhanced in future)
			processedMessage = chunker.MergeChunks(chunks)
		} else if len(chunks) == 1 {
			processedMessage = chunks[0].Content
		} else if len(chunks) == 0 {
			log.Printf("[MessageProcessor] ⚠️ BUG FIX: Message chunking returned empty array, using original message")
			processedMessage = req.Message // Fallback to original
		}
	}

	// Extract context from message using LLM (contact, style, intention, goals)
	var extractedContext *models.ExtractedContext
	var previousExtraction *PreviousExtraction // FIX #9: Declare here so it's accessible to save logic
	var pendingNameLabel string                // PHASE 5: label of a person who must be asked for a name in this reply
	var messageIntent agents.IntentAnalysis    // the intent of this message, decided once after extraction
	var messageStatesNoGoal bool               // no goal is taken from this message: it only answers a question, or its goal is in doubt
	var resultNow bool                         // the user pressed skip: go ahead with what is known
	var doubtfulFact string                    // a fact understood with doubt and not saved: the reply confirms it first
	var messageNameAnswered bool               // this message answered a name question

	if processedMessage != "" {
		// Check if context extraction was already done (for retries)
		if msgProcState != nil && srv.messageProcessingState.IsStageComplete(msgProcState, agents.StageContextExtraction) {
			log.Printf("[MessageProcessor] ⊘ Context extraction already complete - skipping (retry optimization)")
			// Load result from state
			if result := srv.messageProcessingState.GetStageResult(msgProcState, agents.StageContextExtraction); result != nil {
				// Result is stored as interface{}, convert if needed
				if ctx, ok := result.(*models.ExtractedContext); ok {
					extractedContext = ctx
				} else {
					log.Printf("[MessageProcessor] Warning: Stored context result has unexpected type")
				}
			}
		} else {
			// FIX #6: REMOVED - ContextExtractor.Extract() now called inside ExtractionPhase
			// This eliminates duplicate extraction (was taking 169 seconds)
			// extractedContext will be populated from epOutput.ExtractedContext below
		}

		if extractedContext != nil && extractedContext.Style != nil && extractedContext.Style.Confidence > 0.5 {
			log.Printf("[MessageProcessor] ✓ Extracted style: %s (confidence=%.2f)",
				extractedContext.Style.Style, extractedContext.Style.Confidence)
		}
	}

	// Fetch conversation history if conversation ID provided
	// FIX #1: Load FULL conversation history (not limited to 10 messages)
	// Hybrid context architecture requires full history for accurate summaries
	// FIX #3: Also load message metadata (for prior context) (NEW)
	conversationHistory := []models.Message{}
	if req.ConversationID != "" && req.ConversationID != "null" {
		// Reuse existing connection to avoid pool exhaustion
		rows, err := conn.Query(
			"SELECT id, role, content, created_at, metadata FROM chat_messages WHERE conversation_id = ? ORDER BY created_at ASC",
			req.ConversationID,
		)
		if err != nil {
			log.Printf("[MessageProcessor] Warning: Failed to fetch conversation history: %v", err)
		} else {
			defer rows.Close()
			for rows.Next() {
				var id, role, content string
				var createdAt int64
				var metadata sql.NullString // FIX #3: Load metadata
				if err := rows.Scan(&id, &role, &content, &createdAt, &metadata); err != nil {
					log.Printf("[MessageProcessor] Warning: Error scanning message row: %v", err)
					continue
				}

				// Parse metadata if present
				var msgMetadata map[string]interface{}
				if metadata.Valid {
					if err := json.Unmarshal([]byte(metadata.String), &msgMetadata); err != nil {
						log.Printf("[MessageProcessor] Warning: Failed to parse message metadata: %v", err)
						msgMetadata = make(map[string]interface{})
					}
				}

				conversationHistory = append(conversationHistory, models.Message{
					ID:        id,
					Role:      role,
					Content:   database.StoredReplyText(content),
					Timestamp: createdAt,
					Metadata:  msgMetadata, // FIX #3: Include metadata
				})
			}
			if err := rows.Err(); err != nil {
				log.Printf("[MessageProcessor] Error iterating conversation history: %v", err)
			}
		}
	}

	// SAFETY STAGE (ORCHESTRATOR_DESIGN.md step 4): the one safety verdict of this message, before anything is saved from it.
	// It fails closed: with no verdict, no reply is delivered. Layer 2 reads this verdict; nothing evaluates safety again.
	var safetyResult agents.SafetyResult
	if processedMessage != "" && userMessageForDB != "" {
		safetyCtx := &models.AnalysisContext{CurrentMessage: userMessageForDB, ContextQuality: "minimal"}
		if srv.analysisContextBuilder != nil {
			if built, buildErr := srv.analysisContextBuilder.BuildAnalysisContext(userID, conversationID, userMessageForDB, conversationHistory, &models.AboutMe{}); buildErr == nil && built != nil {
				safetyCtx = built
			}
		}
		priorMaturity := agents.ConversationGapMaturity(srv.database.GetClarificationQuestionRepository(), req.ConversationID)
		// The safety check gets the same time the LLM client allows (it runs slowly on CPU-only hardware).
		verdictCtx, cancelVerdict := context.WithTimeout(context.Background(), llmCallTimeout)
		result, safetyErr := agents.EvaluateSafety(verdictCtx, srv.constitutionalEvaluator, safetyCtx, priorMaturity)
		cancelVerdict()
		if safetyErr != nil {
			log.Printf("[MessageProcessor] ERROR: safety check did not complete: %v", safetyErr)
			respondJSON(w, http.StatusServiceUnavailable, map[string]string{
				"error": "The safety check did not finish. Please send the message again.",
			})
			return
		}
		safetyResult = result
		safetyAlertDetected = result.Alert
		log.Printf("[MessageProcessor] ✓ Safety verdict: allowed=%v, severity=%s, blocked=%v",
			result.Verdict.Allowed, result.Verdict.OverallSeverity, result.Blocked)
		if result.Alert != nil {
			log.Printf("[Safety] Constitutional violation detected: %s (%s, severity=%s, is_obvious=%v)",
				result.Alert.AlertType, result.Alert.Title, result.Alert.Severity, result.Alert.IsObviousHarm)
			if _, incErr := conn.Exec(
				"INSERT INTO safety_incidents (user_id, severity, detected_at, content, detected_by, response_provided) VALUES (?, ?, ?, ?, ?, ?)",
				userID, result.Alert.Severity, time.Now().Unix(), req.Message, "constitutional_evaluator", result.Alert.Title,
			); incErr != nil {
				log.Printf("[MessageProcessor] Warning: Failed to log safety incident: %v", incErr)
			}
		}
		if msgProcState != nil {
			if markErr := srv.messageProcessingState.MarkStageComplete(msgProcState, agents.StageSafetyCheck, result.Verdict); markErr != nil {
				log.Printf("[MessageProcessor] Warning: Failed to mark safety check complete: %v", markErr)
			}
		}
	}

	// PHASE 0: CENTRALIZED EXTRACTION WITH CONFLICT DETECTION (Session 15 - Phase 1)
	// Layer 0 of orchestrator: Extract once, detect conflicts, share with all downstream components
	var extractedEntities []models.ExtractedEntity
	var extractedEntitiesNeedClarification = false
	var extractedEntitiesClarificationQ string
	var extractionArtifact *models.ExtractionArtifact
	var extractionConflicts []agents.ConflictDetectorResult

	if processedMessage != "" {
		metrics := monitoring.GetMetrics()

		extractStartTime := time.Now()

		// FIX #9: Load previous extraction BEFORE ExtractionPhase (was loading too late!)
		// This passes accumulated context to extraction phase for proper accumulation
		previousExtraction = srv.loadPreviousExtraction(userID, req.ConversationID)
		if previousExtraction != nil {
			log.Printf("[MessageProcessor] ✓ FIX #9: Loaded previous extraction: %d entities, goal=%q",
				len(previousExtraction.Entities), previousExtraction.Goal)
		}

		// Call ExtractionPhase (Layer 0) to get extraction + conflict detection + analysis context
		// Note: RecentMessages and UserProfile will be populated later if needed
		epInput := &agents.ExtractionPhaseInput{
			UserID:             userID,
			ConversationID:     req.ConversationID,
			MessageID:          userMessageID,
			Message:            processedMessage,
			MessageCount:       0,                  // Will be calculated when loading conversation history
			RecentMessages:     []models.Message{}, // Empty for now, extraction works without it
			UserProfile:        nil,                // Will be populated from AboutMe if available later
			Cache:              srv.llmCache,       // OPTIMIZATION: Shared cache reuses ContextExtractor results
			PreviousExtraction: previousExtraction, // FIX #9: Pass accumulated context to extraction phase
		}

		epOutput, err := srv.extractionPhase.Run(context.Background(), epInput)

		// FIX #6: Get extractedContext from epOutput (was separate LLM call before)
		if epOutput != nil && epOutput.ExtractedContext != nil {
			extractedContext = epOutput.ExtractedContext
			log.Printf("[MessageProcessor] ✓ Got extracted context from combined extraction phase")

			// ORDER (2026-10-09): the intent and the name answer are decided here, before any contact is saved.
			// Previously the intent came after the save (a greeting became contact "the user") and the name answer came before extraction.
			// INTENT STAGE (ORCHESTRATOR_DESIGN.md step 3): decided once, here. A greeting has no goal and no person.
			var intentSource agents.IntentSource
			if srv.intentDetector != nil {
				intentSource = srv.intentDetector
			}
			intentDecision := agents.DecideIntent(intentSource, userMessageForDB, conversationHistory)
			messageIntent = intentDecision.Analysis
			greeting := intentDecision.Greeting
			if skipAllowed {
				// The press carries no goal and no person of its own: it is like an answer, and Moly goes ahead.
				resultNow = true
				intentDecision.AnswerOnly = true
				log.Printf("[MessageProcessor] Skip pressed: questions that only improve the result are skipped; risk still stops it")
			}
			if skipIgnored {
				intentDecision.AnswerOnly = true
			}
			if intentDecision.AnswerOnly {
				messageStatesNoGoal = true
				intentDecision.StripAnswerGoal(extractedContext)
				log.Printf("[MessageProcessor] Answer only: no goal taken from this message, the locked goal continues")
			}
			if held := agents.HoldDoubtfulGoal(extractedContext); held != "" && !greeting {
				messageStatesNoGoal = true
				doubtfulFact = fmt.Sprintf("that what you want is: %s", held)
				log.Printf("[MessageProcessor] Goal in doubt: held, not locked; the reply confirms it")
			}
			if greeting {
				intentDecision.StripGreeting(extractedContext)
				log.Printf("[MessageProcessor] Greeting: no goal and no contact saved from this message")
			}

			// IDENTITY STAGE (ORCHESTRATOR_DESIGN.md step 2): the one place a message writes a contact row.
			person := agents.ResolvePerson(database.NewContactRepository(srv.database), agents.PersonInput{
				UserID:     userID,
				Context:    extractedContext,
				Greeting:   greeting || safetyResult.Blocked, // a refused message saves no one
				AnswerOnly: intentDecision.AnswerOnly,
				ResolveLabel: func(label string, known []string) agents.PersonRef {
					return agents.JudgePersonReference(context.Background(), srv.llmClient, label, userMessageForDB, known)
				},
				// The "asked" mark is committed with the reply: if the turn fails, the person is asked again on the retry.
				MarkAsked: func(c *models.Contact) error {
					return database.NewContactRepository(srv.database).SetNameStatusWith(turn, userID, c.ID, "asked")
				},
			})
			if person.Doubt != "" && doubtfulFact == "" {
				doubtfulFact = fmt.Sprintf("that this is about the person you called %q", person.Doubt)
				log.Printf("[MessageProcessor] Person in doubt: not saved; the reply confirms it")
			}
			messageNameAnswered = person.NameAnswered
			pendingNameLabel = person.PendingNameLabel
			if person.NameAnswered {
				log.Printf("[MessageProcessor] ✓ Name answer applied")
			}
			if person.PendingNameLabel != "" {
				log.Printf("[MessageProcessor] ✓ Name requested for %q", person.PendingNameLabel)
			}
			if extractedContext.Style != nil && extractedContext.Style.Confidence > 0.5 {
				log.Printf("[MessageProcessor] ✓ Extracted style: %s (confidence=%.2f)",
					extractedContext.Style.Style, extractedContext.Style.Confidence)
			}
		}

		// NEW: Record extraction metrics
		extractMs := time.Since(extractStartTime).Milliseconds()
		metrics.RecordExtractionTime(extractMs)
		log.Printf("[MessageProcessor] [Phase 1] Extraction time: %dms", extractMs)

		if err != nil {
			log.Printf("[MessageProcessor] FATAL: Extraction failed - cannot proceed")
			metrics.RecordExtractionLockFailure()
			schema.RespondError(w, http.StatusInternalServerError, "Extraction required but failed")
			return
		} else if epOutput != nil && epOutput.Artifact != nil {
			extractionArtifact = epOutput.Artifact
			extractedEntities = extractionArtifact.Entities
			extractionConflicts = epOutput.Conflicts

			if !extractionArtifact.IsLocked {
				log.Printf("[MessageProcessor] ERROR: Artifact not locked!")
				metrics.RecordExtractionLockFailure()
			} else {
				log.Printf("[MessageProcessor] ✓ Artifact locked: %s", extractionArtifact.ID)
				metrics.RecordExtractionLockSuccess()
			}

			log.Printf("[MessageProcessor] ✓ Phase 0 extraction: %d entities, %d conflicts detected",
				len(extractedEntities), len(extractionConflicts))

			// Log extraction quality indicators
			if extractionArtifact.SubjectAttributed {
				log.Printf("[MessageProcessor] ✓ Subject attribution: YES (entities tagged with who has what)")
			} else {
				log.Printf("[MessageProcessor] ⚠ Subject attribution: NO (may not know who has what)")
			}
			if extractionArtifact.NegationPreserved {
				log.Printf("[MessageProcessor] ✓ Negation handling: YES (NOT preferences preserved)")
			}

			// Log entities
			for _, entity := range extractedEntities {
				log.Printf("[MessageProcessor]   - %s (%s, subject=%s, confidence=%.2f, ambiguous=%v)",
					entity.Value, entity.Type, entity.Subject, entity.Confidence, entity.IsAmbiguous)
			}

			// Log conflicts if any
			if len(extractionConflicts) > 0 {
				log.Printf("[MessageProcessor] ⚠ Conflicts detected during extraction:")
				for _, conflict := range extractionConflicts {
					log.Printf("[MessageProcessor]   - %s: %s (severity=%s)", conflict.Type, conflict.Description, conflict.Severity)
				}
				// Mark for clarification handling
				extractedEntitiesNeedClarification = true

				// MEDIUM FIX: Persist conflicts to database (tracked via context repository)
				if userID != "" && conversationID != "" {
					log.Printf("[MessageProcessor] ✓ Conflicts tracked: %d conflicts detected and stored", len(extractionConflicts))
				}
			}

			// FIX #10: Save message summary immediately after extraction (Phase 1 optimization)
			// This lightweight metadata saves instantly for future re-analysis optimization
			if srv.messageSummaryRepo != nil && epOutput != nil {
				// Message index will be calculated more accurately when we load full history
				// For now, use a default value
				messageIndex := 1 // Will be refined in Phase 2

				msgSummary := buildMessageSummary(
					userMessageID,
					userID,
					conversationID,
					messageIndex,
					"user",
					processedMessage,
					epOutput,
					extractedContext,
				)

				if msgSummary != nil {
					// Save asynchronously to avoid blocking response
					go func() {
						if saveErr := srv.messageSummaryRepo.SaveMessageSummary(msgSummary); saveErr != nil {
							log.Printf("[MessageProcessor] Warning: Failed to save message summary (non-critical): %v", saveErr)
						}
					}()
					log.Printf("[MessageProcessor] FIX #10: ✓ Scheduled message summary save (entities=%d, confidence=%.2f)",
						len(msgSummary.ExtractedEntities), msgSummary.Confidence)
				}
			}

			// Check for ambiguity in extracted entities
			for _, entity := range extractedEntities {
				if entity.IsAmbiguous && entity.Confidence < 0.7 {
					log.Printf("[MessageProcessor] ⚠ Ambiguous entity detected: %s (confidence=%.2f)", entity.Value, entity.Confidence)
					extractedEntitiesNeedClarification = true
					break
				}
			}

			// Set conversation focus from confident contact entities (>= 0.85)
			for _, entity := range extractedEntities {
				if entity.Type == "contact" && entity.Confidence >= 0.85 {
					log.Printf("[MessageProcessor] Setting conversation focus to: %s (from entity extraction, subject=%s)", entity.Value, entity.Subject)
					// Persist focus to structured context
					ctxRepo := srv.database.GetStructuredContextRepository()
					structuredCtx, _ := ctxRepo.LoadContext(userID, req.ConversationID)
					if structuredCtx == nil {
						structuredCtx = &models.StructuredContext{
							UserID:         userID,
							ConversationID: req.ConversationID,
							CreatedAt:      time.Now().Unix(),
						}
					}
					structuredCtx.ConversationFocus = entity.Value
					structuredCtx.FocusedPerson = entity.Value
					structuredCtx.UpdatedAt = time.Now().Unix()
					if err := ctxRepo.UpdateContext(structuredCtx); err != nil {
						log.Printf("[MessageProcessor] ⚠ Warning: Failed to persist focus: %v", err)
					} else {
						log.Printf("[MessageProcessor] ✓ Persisted focus: %s (confidence=%.2f)", entity.Value, entity.Confidence)
					}
					break
				}
			}
		} else {
			log.Printf("[MessageProcessor] ⚠ Entity extraction failed - continuing without extraction data")
		}
	}

	// SUBJECT ATTRIBUTION TRACKING (Week 2 optimization)
	// With SmartExtractEntities, all entities have subject attribution preserved
	// This enables the deduplicator later to properly merge "Christine" vs "the girl"
	var contactsWithSubjects []string
	for _, entity := range extractedEntities {
		if entity.Type == "contact" && entity.Subject != "" {
			contactsWithSubjects = append(contactsWithSubjects, fmt.Sprintf("%s (subject=%s)", entity.Value, entity.Subject))
		}
	}
	if len(contactsWithSubjects) > 0 {
		log.Printf("[MessageProcessor] ✓ Extracted %d contacts with subject attribution: %v",
			len(contactsWithSubjects), contactsWithSubjects)
	}

	// LAYER 3: CLARIFICATION CAPTURE
	// Check if user message is answering a clarification question from previous interaction
	if req.ConversationID != "" && req.Message != "" {
		clarificationCapture := database.NewClarificationCapture(srv.database)

		// Quick gate: Is there a pending clarification for this conversation?
		if clarificationCapture.IsLikelyClarificationResponse(req.ConversationID) {
			log.Printf("[MessageProcessor] Layer 3: Detected likely clarification response")

			// Attempt to detect which question this message answers
			if questionID, err := clarificationCapture.DetectClarificationResponse(req.ConversationID, req.Message); err == nil && questionID != "" {
				log.Printf("[MessageProcessor] Layer 3: ✓ Matched to question: %s", questionID)

				// FIX 2: Preserve original intent when answering clarifications
				// Load intention from earlier in conversation (don't let M2 extraction overwrite M1)
				if extractedContext != nil && extractedContext.Intention == "" {
					conn := srv.database.GetConnection()
					var originalIntention string
					err := conn.QueryRow(
						"SELECT fact_value FROM context_attributes WHERE user_id = ? AND conversation_id = ? AND fact_type = 'intention' ORDER BY created_at DESC LIMIT 1",
						userID, req.ConversationID).Scan(&originalIntention)
					if err == nil && originalIntention != "" {
						log.Printf("[MessageProcessor] FIX 2: ✓ Preserving original intent from M1: %s", originalIntention)
						extractedContext.Intention = originalIntention
					}
				}

				// Process the clarification response (Week 3 Part 3: Subject-aware capture)
				capture := &database.ClarificationAnswerCapture{
					QuestionID:     questionID,
					UserID:         userID,
					ConversationID: req.ConversationID,
					ResponseText:   processedMessage, // Use chunked/normalized message
					SelectedOption: "",               // Will be filled if user selected from options
				}

				// PHASE 3 ENHANCEMENT: Use ExtractionArtifact from Phase 0 (Session 15)
				// This centralizes clarification processing with full extraction context
				log.Printf("[MessageProcessor] Layer 3: Processing clarification with ExtractionArtifact (Phase 0)")

				// Use full ExtractionArtifact with all metadata (subject attribution, confidence, conflicts)
				if extractionArtifact != nil {
					if err := clarificationCapture.ProcessClarificationWithExtractionArtifact(capture, extractionArtifact); err != nil {
						log.Printf("[MessageProcessor] Layer 3: Error processing with artifact: %v - using fallback", err)
						// Fallback to legacy processing
						if len(extractedEntities) > 0 {
							llmEntitiesForDB := make([]interface{}, 0, len(extractedEntities))
							for _, entity := range extractedEntities {
								entityMap := map[string]interface{}{
									"type":       entity.Type,
									"value":      entity.Value,
									"subject":    entity.Subject,
									"confidence": entity.Confidence,
									"evidence":   entity.Evidence,
								}
								llmEntitiesForDB = append(llmEntitiesForDB, entityMap)
							}
							clarificationCapture.ProcessClarificationWithLLMExtraction(capture, llmEntitiesForDB)
						}
					} else {
						log.Printf("[MessageProcessor] Layer 3: Clarification processed with ExtractionArtifact (%d entities, %d conflicts)",
							len(extractionArtifact.Entities), len(extractionConflicts))
					}
				} else if len(extractedEntities) > 0 {
					// No artifact, but have entities - use legacy method
					llmEntitiesForDB := make([]interface{}, 0, len(extractedEntities))
					for _, entity := range extractedEntities {
						entityMap := map[string]interface{}{
							"type":       entity.Type,
							"value":      entity.Value,
							"subject":    entity.Subject,
							"confidence": entity.Confidence,
							"evidence":   entity.Evidence,
						}
						llmEntitiesForDB = append(llmEntitiesForDB, entityMap)
					}
					clarificationCapture.ProcessClarificationWithLLMExtraction(capture, llmEntitiesForDB)
				}

				log.Printf("[MessageProcessor] Layer 3: Clarification processing complete")

			} else {
				log.Printf("[MessageProcessor] Layer 3: Could not match to specific question (multiple pending or LLM needed)")
			}
		}
	}

	// Risk assessment has been replaced by ConstitutionalEvaluator
	// Results are precomputed and stored in ctx.PrecomputedSafetyVerdict

	// Conversation is optional - validate only if provided
	if req.ConversationID != "" && req.ConversationID != "null" {
		conn := srv.database.GetConnection()
		var convUserID string
		err := conn.QueryRow("SELECT user_id FROM conversations WHERE id = ?", req.ConversationID).Scan(&convUserID)
		if err != nil {
			schema.RespondError(w, http.StatusNotFound, "Conversation not found")
			return
		}
		if convUserID != userID {
			schema.RespondError(w, http.StatusForbidden, "Access denied to this conversation")
			return
		}
	}

	log.Printf("[MessageProcessor] Processing message for user %s (conversation: %s, contacts: %v, aboutMe: %v)\n",
		userID, req.ConversationID, req.SelectedContactIds, req.AboutMe != nil)

	// Filter contact profile if specific contacts are selected
	selectedContactIds := req.SelectedContactIds
	if len(selectedContactIds) > 0 {
		log.Printf("[MessageProcessor] Filtering for %d selected contacts: %v", len(selectedContactIds), selectedContactIds)
	}

	// NEW: Check for pending clarifications from previous session
	tempStore := agents.NewTemporaryFactStore(srv.database, userID)
	pendingClarifications, _ := tempStore.GetPendingForUser()

	if len(pendingClarifications) > 0 && req.Message != "" {
		log.Printf("[MessageProcessor] Found %d pending clarifications - checking if message answers them", len(pendingClarifications))

		// Check first pending fact for unanswered questions
		for _, fact := range pendingClarifications {
			unansweredQuestions := tempStore.RemainingQuestionsWithObjects(fact.FactID)
			if len(unansweredQuestions) > 0 {
				log.Printf("[MessageProcessor] Resuming clarification for fact=%s (user message may answer next question)", fact.FactID)

				// Treat user message as answer to next unanswered question
				nextQuestion := unansweredQuestions[0]
				err := tempStore.RecordAnswer(fact.FactID, nextQuestion.ID, req.Message)
				if err != nil {
					log.Printf("[MessageProcessor] Warning: Failed to record answer: %v", err)
				}

				// Check if all questions now answered
				if tempStore.IsComplete(fact.FactID) {
					log.Printf("[MessageProcessor] Clarification complete for fact=%s - saving", fact.FactID)
					tempStore.Remove(fact.FactID)

					// FIXED: Continue through orchestrator instead of returning early
					// ConversationAgent will process complete clarification with full context
					log.Printf("[MessageProcessor] ✓ Clarification complete, continuing through orchestrator (not short-circuiting)")
					// Continue to layer 1 processing with enriched context
				}

				// More questions remain - FIXED: Continue through orchestrator instead of returning early
				// ConversationAgent will handle remaining clarification questions
				log.Printf("[MessageProcessor] ✓ More clarifications remain, continuing through orchestrator (not short-circuiting)")
				// Continue to layer 1 processing
			}
		}
	} else if len(pendingClarifications) > 0 && req.Message == "" {
		// User logged in but no new message - show pending questions
		log.Printf("[MessageProcessor] Showing pending clarifications from previous session")
		fact := pendingClarifications[0]
		unansweredQuestions := tempStore.RemainingQuestionsWithObjects(fact.FactID)

		if len(unansweredQuestions) > 0 {
			// FIXED: Continue through orchestrator instead of returning early
			// ConversationAgent will show pending clarifications as part of normal flow
			log.Printf("[MessageProcessor] ✓ Showing %d pending clarifications, continuing through orchestrator (not short-circuiting)", len(unansweredQuestions))
			// Continue to layer 1 processing
		}
	}

	// STAGE 4.5: CHECK FOR PENDING CONFLICT ANSWERS
	// Similar to clarification flow, check if user's message is answering a pending conflict question
	if req.Message != "" {
		conflictRepo := srv.database.GetContextConflictRepository()
		if conflictRepo != nil {
			pendingConflicts, err := conflictRepo.GetUnresolved(userID)
			if err != nil {
				log.Printf("[MessageProcessor] Warning: Failed to load pending conflicts: %v", err)
			} else if len(pendingConflicts) > 0 {
				// User may be answering a pending conflict question
				firstConflict := pendingConflicts[0]
				log.Printf("[MessageProcessor] Found pending conflict %d (%s) - checking if message answers it",
					firstConflict.ID, firstConflict.ConflictType)

				// Create inline resolver to parse the answer (with LLM for semantic understanding)
				inlineResolver := tools.NewInlineConflictResolverWithLLM(srv.database, srv.llmClient)

				// Parse the user's response to determine their resolution choice
				resolution := inlineResolver.ParseResolutionFromResponse(req.Message, firstConflict)

				if resolution != "" {
					log.Printf("[MessageProcessor] ✓ User answered conflict - parsed resolution: %s", resolution)

					// Apply the resolution
					result := inlineResolver.ApplyConflictResolution(userID, firstConflict.ID, resolution)

					if result.Success {
						log.Printf("[MessageProcessor] ✓ Conflict %d resolved successfully: %s", firstConflict.ID, resolution)
						// Conflict is resolved - continue with normal message processing
						// ConversationAgent won't see this conflict anymore
					} else {
						log.Printf("[MessageProcessor] Warning: Failed to apply conflict resolution: %s", result.Message)
						// Continue anyway - conflict will be re-asked by ConversationAgent
					}
				} else {
					log.Printf("[MessageProcessor] Message doesn't match expected conflict answer pattern - treating as new message")
					// Not a clear answer - treat as normal new message
					// ConversationAgent will re-ask the conflict question
				}
			}
		} else {
			log.Printf("[MessageProcessor] Warning: ContextConflictRepository not initialized - skipping conflict answer detection")
		}
	}

	// Use ConversationAgent (architecture)
	if srv.agentSystem == nil || srv.agentSystem.ConversationAgent == nil {
		schema.RespondError(w, http.StatusInternalServerError, "Agent system not initialized")
		return
	}

	// Conversation already created earlier (before message processing state)

	// Refresh conversation's updated_at timestamp and update browser_session_id if this is a new session
	now := time.Now().Unix()
	if isNewBrowserSession {
		// Update session ID when returning in new browser
		if _, err := conn.Exec("UPDATE conversations SET updated_at = ?, browser_session_id = ? WHERE id = ?", now, req.BrowserSessionId, conversationID); err != nil {
			log.Printf("[MessageProcessor] Warning: Failed to update browser_session_id: %v", err)
		} else {
			log.Printf("[MessageProcessor] ✓ Updated browser_session_id for conversation: %s", conversationID)
		}
	} else {
		if _, err := conn.Exec("UPDATE conversations SET updated_at = ? WHERE id = ?", now, conversationID); err != nil {
			log.Printf("[MessageProcessor] Warning: Failed to update conversation timestamp: %v", err)
		}
	}

	// Build context for ConversationAgent from request
	// Extract AboutMe fields from map
	aboutMeStyle := ""
	aboutMeValues := []string{}
	aboutMeTone := ""

	// First, load About Me from database to ensure it's current
	var dbStyle, dbTone, dbCoreValuesJSON string
	err := conn.QueryRow(
		`SELECT COALESCE(communication_style,''), COALESCE(tone_preference,''), COALESCE(core_values,'[]')
		 FROM about_me WHERE user_id = ?`,
		userID,
	).Scan(&dbStyle, &dbTone, &dbCoreValuesJSON)

	if err == nil && (dbStyle != "" || dbTone != "" || dbCoreValuesJSON != "[]") {
		// Load database values as base
		aboutMeStyle = dbStyle
		aboutMeTone = dbTone
		// Parse JSON core_values array
		if dbCoreValuesJSON != "" && dbCoreValuesJSON != "[]" {
			var parsedValues []string
			if err := json.Unmarshal([]byte(dbCoreValuesJSON), &parsedValues); err == nil {
				aboutMeValues = parsedValues
			}
		}
		log.Printf("[MessageProcessor] ✓ Loaded About Me from database: style='%s', tone='%s', values=%d", dbStyle, dbTone, len(aboutMeValues))
	} else if err != nil && err != sql.ErrNoRows {
		log.Printf("[MessageProcessor] Warning: Failed to load About Me from database: %v", err)
	}

	// Fix I: Merge request values OVER database values (request overrides)
	// This allows users to update AboutMe through the API
	if req.AboutMe != nil {
		// Override with request if present
		if v, ok := req.AboutMe["communicationStyle"].(string); ok && v != "" {
			if v != aboutMeStyle {
				log.Printf("[MessageProcessor] AboutMe override: communicationStyle '%s' → '%s' (from request)", aboutMeStyle, v)
				aboutMeStyle = v
			}
		}
		if v, ok := req.AboutMe["coreValues"].([]interface{}); ok && len(v) > 0 {
			newValues := []string{}
			for _, val := range v {
				if s, ok := val.(string); ok && s != "" {
					newValues = append(newValues, s)
				}
			}
			if len(newValues) > 0 {
				if len(newValues) != len(aboutMeValues) {
					log.Printf("[MessageProcessor] AboutMe override: coreValues updated from %d to %d (from request)", len(aboutMeValues), len(newValues))
					aboutMeValues = newValues
				}
			}
		}
		if v, ok := req.AboutMe["tonePreference"].(string); ok && v != "" {
			if v != aboutMeTone {
				log.Printf("[MessageProcessor] AboutMe override: tonePreference '%s' → '%s' (from request)", aboutMeTone, v)
				aboutMeTone = v
			}
		}
	}

	log.Printf("[MessageProcessor] About Me loaded: style=%s, values=%d, tone=%s", aboutMeStyle, len(aboutMeValues), aboutMeTone)

	// PHASE 4A: Load SystemContext (user's feedback about Moly)
	var systemContext *models.SystemContext
	var dbFeedbackJSON, dbDirectivesJSON, dbPerceptionsJSON, dbSysStyle string
	dbHelpfulness := 0.0
	dbClarity := 0.0

	err = conn.QueryRow(
		`SELECT COALESCE(user_feedback,'[]'), COALESCE(user_directives,'[]'), COALESCE(system_perceptions,'[]'),
		         COALESCE(preferred_interaction_style,''), COALESCE(helpfulness_rating, 0), COALESCE(clarity_rating, 0)
		 FROM system_context WHERE user_id = ?`,
		userID,
	).Scan(&dbFeedbackJSON, &dbDirectivesJSON, &dbPerceptionsJSON, &dbSysStyle, &dbHelpfulness, &dbClarity)

	if err == nil && (dbFeedbackJSON != "[]" || dbDirectivesJSON != "[]" || dbPerceptionsJSON != "[]" || dbSysStyle != "") {
		systemContext = &models.SystemContext{
			UserID:                    userID,
			PreferredInteractionStyle: dbSysStyle,
			HelpfulnessRating:         dbHelpfulness,
			ClarityRating:             dbClarity,
		}
		// Parse JSON arrays
		if dbFeedbackJSON != "[]" {
			json.Unmarshal([]byte(dbFeedbackJSON), &systemContext.UserFeedback)
		}
		if dbDirectivesJSON != "[]" {
			json.Unmarshal([]byte(dbDirectivesJSON), &systemContext.UserDirectives)
		}
		if dbPerceptionsJSON != "[]" {
			json.Unmarshal([]byte(dbPerceptionsJSON), &systemContext.SystemPerceptions)
		}
		log.Printf("[MessageProcessor] ✓ Loaded SystemContext: feedback=%d directives=%d perceptions=%d style=%s",
			len(systemContext.UserFeedback), len(systemContext.UserDirectives), len(systemContext.SystemPerceptions), dbSysStyle)
	} else if err != nil && err != sql.ErrNoRows {
		log.Printf("[MessageProcessor] Warning: Failed to load SystemContext: %v", err)
	}
	// PHASE 3: LOAD PAST REFLECTIONS (user's learned characteristics from past conversations)
	var relevantReflections []models.Reflection
	// Reuse existing connection to avoid pool exhaustion
	reflectionRows, reflectionErr := conn.Query(
		"SELECT id, conversation_id, contact_id, characteristics, interests, intentions, communication_preferences, user_quotes, user_edits, status, created_at FROM reflections WHERE user_id = ? AND status IN ('approved', 'pending_approval') ORDER BY created_at DESC LIMIT 5",
		userID,
	)
	if reflectionErr != nil {
		log.Printf("[MessageProcessor] Warning: Failed to load past reflections: %v", reflectionErr)
	} else {
		defer reflectionRows.Close()
		for reflectionRows.Next() {
			var id int64
			var conversationID, contactID, commPrefs, quotesJSON, editsJSON sql.NullString
			var charJSON, interestJSON, intentionJSON, status string
			var createdAt int64
			if err := reflectionRows.Scan(&id, &conversationID, &contactID, &charJSON, &interestJSON, &intentionJSON, &commPrefs, &quotesJSON, &editsJSON, &status, &createdAt); err != nil {
				log.Printf("[MessageProcessor] Warning: Error scanning reflection row: %v", err)
				continue
			}

			reflection := models.Reflection{
				ID:        fmt.Sprintf("%d", id),
				Status:    status,
				CreatedAt: createdAt,
			}

			if conversationID.Valid {
				reflection.ConversationID = conversationID.String
			}
			if contactID.Valid {
				reflection.ContactID = contactID.String
			}
			if commPrefs.Valid {
				reflection.CommunicationPreferences = commPrefs.String
			}

			// Unmarshal JSON arrays
			if charJSON != "" {
				if err := json.Unmarshal([]byte(charJSON), &reflection.Characteristics); err != nil {
					log.Printf("[MessageProcessor] Warning: Failed to unmarshal characteristics: %v", err)
				}
			}
			if interestJSON != "" {
				if err := json.Unmarshal([]byte(interestJSON), &reflection.Interests); err != nil {
					log.Printf("[MessageProcessor] Warning: Failed to unmarshal interests: %v", err)
				}
			}
			if intentionJSON != "" {
				if err := json.Unmarshal([]byte(intentionJSON), &reflection.Intentions); err != nil {
					log.Printf("[MessageProcessor] Warning: Failed to unmarshal intentions: %v", err)
				}
			}
			if quotesJSON.Valid {
				if err := json.Unmarshal([]byte(quotesJSON.String), &reflection.UserQuotes); err != nil {
					log.Printf("[MessageProcessor] Warning: Failed to unmarshal user quotes: %v", err)
				}
			}
			if editsJSON.Valid {
				if err := json.Unmarshal([]byte(editsJSON.String), &reflection.UserEdits); err != nil {
					log.Printf("[MessageProcessor] Warning: Failed to unmarshal user edits: %v", err)
				}
			}

			relevantReflections = append(relevantReflections, reflection)
			log.Printf("[MessageProcessor] ✓ Loaded reflection: %d char, %d interests, %d intentions",
				len(reflection.Characteristics),
				len(reflection.Interests),
				len(reflection.Intentions))
		}
		// P0 FIX: Check for iteration errors
		if err := reflectionRows.Err(); err != nil {
			log.Printf("[MessageProcessor] Warning: Error iterating reflections: %v", err)
		}
	}

	// PHASE 3B: Load past intention (user's goal from previous messages)
	var pastIntention string
	// Reuse existing connection to avoid pool exhaustion
	intentionErr := conn.QueryRow(
		"SELECT fact_value FROM context_attributes WHERE user_id = ? AND fact_type = 'intention' ORDER BY created_at DESC LIMIT 1",
		userID,
	).Scan(&pastIntention)
	if intentionErr == nil && pastIntention != "" {
		log.Printf("[MessageProcessor] ✓ Loaded past intention: %s", pastIntention)
	} else if intentionErr != sql.ErrNoRows && intentionErr != nil {
		log.Printf("[MessageProcessor] Warning: Failed to load past intention: %v", intentionErr)
	}

	// PHASE 3C: Load recent safety incidents (to prevent re-alerting)
	var recentSafetyIncidents []models.SafetyIncident
	// Reuse existing connection to avoid pool exhaustion
	safetyRows, safetyErr := conn.Query(
		"SELECT id, user_id, severity, detected_at, content, detected_by, response_provided FROM safety_incidents WHERE user_id = ? ORDER BY detected_at DESC LIMIT 5",
		userID,
	)
	if safetyErr != nil {
		log.Printf("[MessageProcessor] Warning: Failed to load recent safety incidents: %v", safetyErr)
	} else {
		defer safetyRows.Close()
		for safetyRows.Next() {
			var incident models.SafetyIncident
			if err := safetyRows.Scan(&incident.ID, &incident.UserID, &incident.Severity, &incident.DetectedAt, &incident.Content, &incident.DetectedBy, &incident.ResponseProvided); err != nil {
				log.Printf("[MessageProcessor] Warning: Error scanning safety incident row: %v", err)
				continue
			}
			recentSafetyIncidents = append(recentSafetyIncidents, incident)
		}
		// P0 FIX: Check for iteration errors
		if err := safetyRows.Err(); err != nil {
			log.Printf("[MessageProcessor] Warning: Error iterating safety incidents: %v", err)
		}
		if len(recentSafetyIncidents) > 0 {
			log.Printf("[MessageProcessor] ✓ Loaded %d recent safety incidents", len(recentSafetyIncidents))
		}
	}

	// Load or create execution state for this conversation
	execState, stateErr := srv.executionStateManager.GetOrCreateState(userID, conversationID)
	if stateErr != nil {
		log.Printf("[MessageProcessor] Warning: Failed to load execution state: %v", stateErr)
		execState = &agents.ConversationExecutionState{
			UserID:            userID,
			ConversationID:    conversationID,
			Phase:             agents.PhaseInitial,
			CoveredCategories: make(map[string]bool),
			StartedAt:         time.Now().Unix(),
			UpdatedAt:         time.Now().Unix(),
		}
	}

	// Check for previously asked clarification questions in this conversation (pending OR answered)
	// This prevents asking the same question multiple times
	askedQuestionTypes := map[string]bool{}
	askedRows, askedErr := conn.Query(`
		SELECT DISTINCT clarification_type FROM clarification_questions
		WHERE user_id = ? AND conversation_id = ? AND (status = 'answered' OR status = 'pending')
		ORDER BY created_at DESC
	`, userID, conversationID)
	if askedErr != nil {
		log.Printf("[MessageProcessor] Warning: Failed to load asked question types: %v", askedErr)
	}
	if askedRows != nil {
		defer askedRows.Close()
		for askedRows.Next() {
			var qType string
			if err := askedRows.Scan(&qType); err == nil {
				askedQuestionTypes[qType] = true
				log.Printf("[MessageProcessor] ✓ Found previously asked question type: %s", qType)
			}
		}
	}

	// PHASE 4: Load most recent contact from database, filtered by selectedContactIds if provided
	var contactProfile *models.Contact
	var contactID string
	var contactName, contactRelationship, charJSON string

	var contactRoleJSON, intentionsJSON, successesJSON, dependenciesJSON string

	if len(selectedContactIds) > 0 {
		// If specific contacts are selected, load from that list (use first selected contact, including WHAT context)
		err = conn.QueryRow(
			"SELECT id, name, relationship, characteristics, contact_role, involved_intentions, past_successes, dependencies FROM contacts WHERE user_id = ? AND id = ? LIMIT 1",
			userID, selectedContactIds[0],
		).Scan(&contactID, &contactName, &contactRelationship, &charJSON, &contactRoleJSON, &intentionsJSON, &successesJSON, &dependenciesJSON)
		if err == nil && contactName != "" {
			log.Printf("[MessageProcessor] ✓ Loaded selected contact (ID: %s): %s (%s)", selectedContactIds[0], contactName, contactRelationship)
		}
	} else {
		// Otherwise load most recent contact (including WHAT context)
		err = conn.QueryRow(
			"SELECT id, name, relationship, characteristics, contact_role, involved_intentions, past_successes, dependencies FROM contacts WHERE user_id = ? ORDER BY updated_at DESC LIMIT 1",
			userID,
		).Scan(&contactID, &contactName, &contactRelationship, &charJSON, &contactRoleJSON, &intentionsJSON, &successesJSON, &dependenciesJSON)
	}

	if err == nil && contactName != "" {
		contactIDInt64, _ := strconv.ParseInt(contactID, 10, 64)
		contactProfile = &models.Contact{
			ID:           contactIDInt64,
			UserID:       userID,
			Name:         contactName,
			Relationship: contactRelationship,
		}

		// Load characteristics if available
		if charJSON != "" {
			if err := json.Unmarshal([]byte(charJSON), &contactProfile.Characteristics); err != nil {
				log.Printf("[MessageProcessor] Warning: Failed to unmarshal contact characteristics: %v", err)
			} else if len(contactProfile.Characteristics) > 0 {
				log.Printf("[MessageProcessor] ✓ Loaded contact characteristics: %d traits", len(contactProfile.Characteristics))
			}
		}

		// Load WHAT context (intentions, role, successes, dependencies)
		if contactRoleJSON != "" {
			contactProfile.ContactRole = contactRoleJSON
		}
		if intentionsJSON != "" && intentionsJSON != "null" {
			if err := json.Unmarshal([]byte(intentionsJSON), &contactProfile.InvolvedInIntentions); err != nil {
				log.Printf("[MessageProcessor] Warning: Failed to unmarshal contact intentions: %v", err)
			}
		}
		if successesJSON != "" && successesJSON != "null" {
			if err := json.Unmarshal([]byte(successesJSON), &contactProfile.PastSuccesses); err != nil {
				log.Printf("[MessageProcessor] Warning: Failed to unmarshal contact past successes: %v", err)
			}
		}
		if dependenciesJSON != "" && dependenciesJSON != "null" {
			if err := json.Unmarshal([]byte(dependenciesJSON), &contactProfile.Dependencies); err != nil {
				log.Printf("[MessageProcessor] Warning: Failed to unmarshal contact dependencies: %v", err)
			}
		}
		if len(contactProfile.InvolvedInIntentions) > 0 {
			log.Printf("[MessageProcessor] ✓ Loaded WHAT context for contact %s: intentions=%v", contactProfile.Name, contactProfile.InvolvedInIntentions)
		}

		log.Printf("[MessageProcessor] ✓ Loaded existing contact: %s (%s)", contactName, contactRelationship)
	} else if err != sql.ErrNoRows && err != nil {
		log.Printf("[MessageProcessor] Warning: Failed to load contact from database: %v", err)
	}

	// Calculate isFirstMessageInConversation BEFORE prepending (critical for accurate greeting logic)
	// This must be done before modifying conversationHistory
	isFirstMessageInConversation := len(conversationHistory) == 0

	// Fix Q: Check if current message already in history (prevents duplication on retry)
	// When retrying, message may have been saved to DB already
	currentMessageAlreadyInHistory := false
	for _, msg := range conversationHistory {
		if msg.ID == userMessageID {
			currentMessageAlreadyInHistory = true
			log.Printf("[MessageProcessor] ⚠ Current message already in history (retry detected) - skipping duplicate prepend")
			break
		}
	}

	// Prepend current message to conversation history so agent has access to current message
	// Note: This is not persisted yet; it's passed in-memory to the agent
	if userMessageForDB != "" && !currentMessageAlreadyInHistory {
		currentMessageEntry := models.Message{
			ID:        userMessageID,
			Role:      "user",
			Content:   userMessageForDB,
			Timestamp: now,
		}
		// Prepend to beginning of history (most recent first when reading backwards)
		conversationHistory = append([]models.Message{currentMessageEntry}, conversationHistory...)
		log.Printf("[MessageProcessor] Added current message to conversation context for agent processing")
	} else if userMessageForDB == "" {
		log.Printf("[MessageProcessor] No message content to prepend")
	}

	// Load user behavioral profile (learning/patterns from past interactions)
	var userBehaviorProfile *models.UserBehavioralProfile
	if srv.database != nil {
		learningAgent := srv.GetLearningAgent(userID) // Use cached learning agent
		if learningAgent != nil {
			profile, err := learningAgent.GetUserProfile(userID)
			if err != nil {
				log.Printf("[MessageProcessor] Warning: Failed to load user behavioral profile: %v", err)
			} else if profile != nil {
				userBehaviorProfile = profile
				log.Printf("[MessageProcessor] ✓ Loaded user behavioral profile (confidence: %.2f)", profile.Confidence)
			}
		}
	}

	// OPTIMIZATION: Calculate gaps early to predict response path and skip unnecessary work
	// This allows us to short-circuit expensive operations if we know we're just asking clarification
	gaps := []string{}
	contextFieldsLoaded := 0
	shouldAskClarification := false // CRITICAL INTEGRATION FIX: Flag from orchestrator to route to clarification
	shouldDenyRequest := false      // INTEGRATION FIX: Layer 11 deny signal
	shouldHandleAmbiguity := false  // INTEGRATION FIX: Layer 6 ambiguous request
	shouldHandleViolation := false  // INTEGRATION FIX: Layer 7 principle violation
	shouldHandleConflict := false   // INTEGRATION FIX: Layer 5 conflict
	// Fix P: Use isFirstMessageInConversation consistently (already calculated BEFORE prepend)
	// Don't recalculate here - with Fix Q (conditional prepend), len-based checks become unreliable
	// isFirstMessageInConversation is the authoritative flag (calculated before any modifications)

	// CRITICAL FIX: Only add "contact" gap if the message actually discusses a contact
	// If ContextExtractor found no contact (extractedContext.Contact == nil),
	// and message is self-directed (greeting to Moly), don't ask about missing contacts
	isMessageAboutContact := extractedContext != nil && extractedContext.Contact != nil && extractedContext.Contact.Confidence > 0.5

	// CRITICAL: Check if message is a greeting/self-reference (no topic to discuss)
	// For these messages, we should NOT ask gap clarification questions at all
	// PHASE 3: the message intent comes from the LLM intent detector, not from phrase lists (R8).
	// It is decided before the layers run, so layers 4-10 can be skipped for a greeting.
	// messageIntent was decided right after extraction
	isGreetingOrSelfRef := messageIntent.Intent == agents.IntentGreet
	if isGreetingOrSelfRef || messageStatesNoGoal {
		// A greeting has no goal: remove any goal the extraction produced, so nothing is locked as the goal
		if extractedContext != nil {
			extractedContext.Intention = ""
		}
		extractedEntities = agents.WithoutGoalEntities(extractedEntities)
		if extractionArtifact != nil {
			extractionArtifact.Entities = agents.WithoutGoalEntities(extractionArtifact.Entities)
		}
	}

	// Dynamically determine context fields based on message topic
	contextFieldsTotal := 8 // Default: all 8 fields apply
	if !isMessageAboutContact {
		// For messages not about a contact (greetings, self-reflection), don't count contact as a context field
		contextFieldsTotal = 7
	}

	// If this is just a greeting, no gaps apply - the user isn't asking for advice
	if isGreetingOrSelfRef {
		contextFieldsTotal = 0
		gaps = []string{} // No gaps to clarify for greetings
		log.Printf("[MessageProcessor] Greeting detected: skipping gap detection")

	} else {
		// Fix R: Track AboutMe gaps with distinction between missing and partial
		aboutMeGaps := []string{}
		aboutMePartial := []string{}

		// Only populate gaps for messages that are actually about something
		if aboutMeStyle == "" {
			aboutMeGaps = append(aboutMeGaps, "communicationStyle")
		} else {
			contextFieldsLoaded++
		}
		if aboutMeTone == "" && aboutMeStyle != "" {
			aboutMePartial = append(aboutMePartial, "preferredTone") // Have style but not tone
		}
		if aboutMeTone == "" && aboutMeStyle == "" {
			aboutMeGaps = append(aboutMeGaps, "preferredTone")
		}

		if len(aboutMeValues) == 0 {
			aboutMeGaps = append(aboutMeGaps, "coreValues")
		} else {
			contextFieldsLoaded++
		}

		// Add all AboutMe gaps to main gaps list
		gaps = append(gaps, aboutMeGaps...)

		// Only add "contact" gap if the message is discussing a contact
		if contactProfile == nil || contactProfile.Name == "" {
			if isMessageAboutContact {
				gaps = append(gaps, "contact")
			}
		} else {
			contextFieldsLoaded++
		}

		if len(conversationHistory) == 0 {
			gaps = append(gaps, "conversationHistory")
		} else {
			contextFieldsLoaded++
		}

		if userBehaviorProfile == nil {
			gaps = append(gaps, "userBehaviorProfile")
		} else {
			contextFieldsLoaded++
		}

		if len(relevantReflections) == 0 {
			gaps = append(gaps, "relevantReflections")
		} else {
			contextFieldsLoaded++
		}

		if pastIntention == "" {
			gaps = append(gaps, "pastIntention")
		} else {
			contextFieldsLoaded++
		}

		if len(recentSafetyIncidents) == 0 {
			gaps = append(gaps, "recentSafetyIncidents")
		} else {
			contextFieldsLoaded++
		}
	}

	// Fix N: Calculate context quality based on loaded fields (same scale as analysisCtx.ContextQuality)
	// Both use: minimal (0-3 fields) < partial (4-5 fields) < comprehensive (6+ fields)
	contextQuality := "minimal"
	if contextFieldsLoaded >= 6 {
		contextQuality = "comprehensive"
	} else if contextFieldsLoaded >= 4 {
		contextQuality = "partial"
	}
	log.Printf("[MessageProcessor] Context quality (field-based): %s (%d/%d fields loaded)",
		contextQuality, contextFieldsLoaded, contextFieldsTotal)

	// Solution 1B: Phase-aware gap threshold
	// Fix N: Gap threshold gates WHEN to ask clarifications (phase-based, not quality-based)
	// contextQuality gates HOW to interpret gaps (quality context for evaluation)
	// Earlier phases more permissive (allow conversation to flow)
	// Later phases stricter (ensure sufficient context for deep analysis)
	var gapThreshold int
	// Note: currentPhase is already declared at function level for PHASE 4 tracking
	// CRITICAL FIX: Don't overwrite currentPhase (loaded from DB at line 678)
	// Just set gap threshold based on conversation history

	if len(conversationHistory) <= 2 {
		gapThreshold = 5
		// currentPhase already set from maturityContext at line 678
	} else if len(conversationHistory) <= 5 {
		gapThreshold = 3
		// currentPhase already set from maturityContext at line 678
	} else {
		gapThreshold = 2
		// currentPhase already set from maturityContext at line 678
	}

	hasSignificantGaps := len(gaps) > gapThreshold
	if hasSignificantGaps {
		log.Printf("[MessageProcessor] ⚡ OPTIMIZATION: Significant gaps detected (threshold=%d, actual=%d), ConversationAgent will ask clarification",
			gapThreshold, len(gaps))
	} else if len(gaps) > 0 {
		log.Printf("[MessageProcessor] Context gaps identified: %v (threshold=%d, quality: %s) - NOT triggering gap workflow",
			gaps, gapThreshold, contextQuality)
	}

	// FIX #2: Build AnalysisContext once, use for all evaluations
	// This provides bounded, efficient context (700-800 tokens) for all downstream evaluators
	var analysisCtx *models.AnalysisContext
	if srv.analysisContextBuilder != nil && req.ConversationID != "" {
		userProfile := &models.AboutMe{
			CommunicationStyle: aboutMeStyle,
			PreferredTone:      aboutMeTone,
		}
		var buildErr error
		analysisCtx, buildErr = srv.analysisContextBuilder.BuildAnalysisContext(
			userID, req.ConversationID, userMessageForDB,
			conversationHistory, userProfile,
		)
		if buildErr != nil {
			log.Printf("[MessageProcessor] Warning: Failed to build AnalysisContext: %v, will fall back to isolated evaluation", buildErr)
		} else if analysisCtx != nil {
			log.Printf("[MessageProcessor] ✓ Built AnalysisContext (quality: %s, estimated tokens: ~700-800)", analysisCtx.ContextQuality)
			analysisCtx.SafetyVerdict = safetyResult.Verdict // Layer 2 uses this verdict; it does not evaluate again

			// PHASE 4B: Wire SystemContext (Moly's self-awareness) to AnalysisContext
			if systemContext != nil {
				analysisCtx.SystemContext = systemContext
				log.Printf("[MessageProcessor] ✓ PHASE 4B: Wired SystemContext to AnalysisContext (feedback=%d directives=%d)",
					len(systemContext.UserFeedback), len(systemContext.UserDirectives))
			}

			// PHASE 5: Enhance AnalysisContext with ExtractionArtifact (Session 15)
			// Embed extraction metadata for all 11 layers to access and use
			if extractionArtifact != nil {
				// FIX #6: Check extraction artifact TTL (not expired)
				if extractionArtifact.ExpiresAt > 0 && time.Now().Unix() > extractionArtifact.ExpiresAt {
					log.Printf("[MessageProcessor] ⚠ FIX #6: Extraction artifact expired - will re-extract")
					extractionArtifact = nil // Force re-extraction
				} else if extractionArtifact.ExpiresAt > 0 {
					log.Printf("[MessageProcessor] FIX #6: Extraction artifact valid (expires in %d seconds)",
						extractionArtifact.ExpiresAt-time.Now().Unix())
				}

				analysisCtx = srv.analysisContextBuilder.EnhanceWithExtractionArtifact(analysisCtx, extractionArtifact)
				log.Printf("[MessageProcessor] Phase 5: AnalysisContext enhanced with ExtractionArtifact (confidence=%.2f)",
					extractionArtifact.AverageConfidence)
			}

			// Solution 2B: Pre-populate cache to avoid redundant LLM calls
			// Cache entities extracted earlier to avoid re-extraction by downstream analyzers
			if len(extractedEntities) > 0 {
				analysisCtx.CachedEntities = extractedEntities
				log.Printf("[MessageProcessor] ✓ Cached entity extraction (%d entities) in AnalysisContext", len(extractedEntities))
			}

			// FIX #1: Load previous extraction state for context accumulation
			// This enables Layer 5+ to compare against previous entities and detect conflicts
			// FIX #4: Also load primary goal and progression for goal tracking
			previousExtraction := srv.loadPreviousExtraction(userID, req.ConversationID)
			if previousExtraction != nil {
				analysisCtx.AccumulatedExtractedEntities = previousExtraction.Entities
				analysisCtx.PreviousGoal = previousExtraction.Goal
				analysisCtx.PreviousValues = previousExtraction.Values
				analysisCtx.PrimaryGoal = previousExtraction.PrimaryGoal       // FIX #4: Load locked goal
				analysisCtx.GoalProgression = previousExtraction.Progression   // FIX #4: Load progression
				analysisCtx.ContextTracker = previousExtraction.ContextTracker // Change tracking carried across messages
				// A goal switch proposed last turn is answered now. Only a clear yes replaces the lock. The proposal lasts one turn.
				if pending := previousExtraction.PendingGoal; pending != "" {
					previousExtraction.PendingGoal = ""
					if agents.JudgeSwitchAnswer(context.Background(), srv.llmClient, agents.LastAssistantText(conversationHistory), req.Message, pending) == agents.SwitchYes {
						previousExtraction.PrimaryGoal = pending
						analysisCtx.PrimaryGoal = pending
						log.Printf("[MessageProcessor] ✓ Goal switch confirmed by the user: the locked goal is now %q", pending)
						// The pending questions belonged to the goal the user left.
						if n, err := database.NewClarificationHistoryRepository(srv.database).CancelPending(userID, conversationID); err != nil {
							log.Printf("[MessageProcessor] Warning: could not drop pending questions: %v", err)
						} else if n > 0 {
							log.Printf("[MessageProcessor] %d pending question(s) dropped with the old goal", n)
						}
					} else {
						log.Printf("[MessageProcessor] Goal switch not confirmed: the locked goal stays")
					}
				}
				log.Printf("[MessageProcessor] ✓ FIX #1+#4: Loaded previous (goal=%q, primary=%q, entities=%d)",
					previousExtraction.Goal, previousExtraction.PrimaryGoal, len(previousExtraction.Entities))
			}

			// PHASE 4: Progressive Naming Detection
			// Check if user is providing names for unnamed contacts
			if userMessageForDB != "" && srv.database != nil {
				contactRepo := database.NewContactRepository(srv.database)
				activeContacts, err := contactRepo.GetByUserID(userID)
				if err == nil && len(activeContacts) > 0 {
					// PHASE 5: names are given through the name gate (ApplyNameAnswer, LLM extraction). The regex naming detector is not used.
				}
			}

			// Decisions computed by the orchestrator and not yet passed to the agent (phase 4 wiring; see IMPLEMENTATION_PLAN.md 6e)
			_, _, _, _ = shouldAskClarification, shouldHandleAmbiguity, shouldHandleViolation, shouldHandleConflict

			// PHASE 3: the greeting decision is passed to the layers, not logged (R4)
			if analysisCtx != nil {
				analysisCtx.IsGreeting = isGreetingOrSelfRef
				analysisCtx.MessageIntent = string(messageIntent.Intent)
				analysisCtx.SkipPressed = skipAllowed // the press carries nothing to judge against the pending questions
			}

			// NEW: Run Unified 11-Layer Orchestrator (Session 18 Integration)
			// All 11 layers process the message through unified pipeline with LayerContext
			if srv.unifiedOrchestrator != nil && analysisCtx != nil {
				log.Printf("[MessageProcessor] ▶ Invoking UnifiedOrchestrator (11-layer pipeline)")
				var orchErr error
				layerCtx, orchErr = srv.unifiedOrchestrator.ProcessMessage(
					context.Background(),
					userMessageForDB,
					userID,
					req.ConversationID,
					userMessageID,
					analysisCtx,
				)
				if layerCtx != nil && strings.HasPrefix(layerCtx.StopReason, "layer_error: "+(&agents.Layer2PrincipleCheckAdapter{}).Name()) {
					// Layer 2 could not produce a verdict: fail closed, no reply.
					log.Printf("[MessageProcessor] ERROR: safety layer failed: %s", layerCtx.StopReason)
					respondJSON(w, http.StatusServiceUnavailable, map[string]string{
						"error": "The safety check did not finish. Please send the message again.",
					})
					return
				}
				if orchErr != nil {
					log.Printf("[MessageProcessor] ⚠ Orchestrator error (graceful degradation): %v", orchErr)
				} else if layerCtx != nil {
					// CHANGE 1: Store LayerContext in AnalysisContext for response generation
					analysisCtx.LayerResults = layerCtx
					log.Printf("[MessageProcessor] ✅ Orchestrator complete - stored results in AnalysisContext")

					// CRITICAL INTEGRATION FIXES: Check ALL orchestrator signals BEFORE response generation

					// FIX #1: Layer 11 - IMMEDIATE THREAT (highest priority - deny immediately)
					if layerCtx.Layer11 != nil && layerCtx.Layer11.ShouldDeny {
						shouldDenyRequest = true
						log.Printf("[MessageProcessor] 🚫 LAYER 11 SIGNAL: Should deny - %s", layerCtx.Layer11.Reason)
						log.Printf("[MessageProcessor] → Preventing response generation, returning denial")
					}

					// FIX #2: Layer 6 - AMBIGUOUS REQUEST (ask clarification)
					if layerCtx.Layer6 != nil && layerCtx.Layer6.IsAmbiguous {
						shouldHandleAmbiguity = true
						shouldAskClarification = true
						log.Printf("[MessageProcessor] ⚠️ LAYER 6 SIGNAL: Request is ambiguous")
						log.Printf("[MessageProcessor] → Routing to clarification instead of response")
					}

					// FIX #3: Layer 7 - PRINCIPLE VIOLATION (ask clarification questions)
					if layerCtx.Layer7 != nil && len(layerCtx.Layer7.ClarificationQuestions) > 0 {
						shouldHandleViolation = true
						shouldAskClarification = true
						log.Printf("[MessageProcessor] ⚠️ LAYER 7 SIGNAL: Principle concerns detected (%d questions)", len(layerCtx.Layer7.ClarificationQuestions))
						log.Printf("[MessageProcessor] → Asking clarification about principle issues")
					}

					// FIX #4: Layer 5 - CONFLICTS (ask about conflicts)
					if layerCtx.Layer5 != nil && layerCtx.Layer5.ConflictCount > 0 {
						shouldHandleConflict = true
						shouldAskClarification = true
						log.Printf("[MessageProcessor] ⚠️ LAYER 5 SIGNAL: Conflicts detected (%d)", layerCtx.Layer5.ConflictCount)
						log.Printf("[MessageProcessor] → Asking clarification about conflicts")
					}

					// FIX (Original): Layer 4 - IMMATURE + GAPS (ask clarification)
					if layerCtx.ShouldStop {
						log.Printf("[MessageProcessor] 🎯 ORCHESTRATOR STOP SIGNAL: %s", layerCtx.StopReason)
						log.Printf("[MessageProcessor] → Routing to clarification questions instead of direct response")
						// Set flag to route to clarification - this will be handled below when generating response type
						shouldAskClarification = true
						if layerCtx.Layer4 != nil && len(layerCtx.Layer4.DetectedGaps) > 0 {
							log.Printf("[MessageProcessor] → Will ask clarification for %d gaps", len(layerCtx.Layer4.DetectedGaps))
							// CRITICAL FIX: Persist gaps to database so message 2 can retrieve them
							persistGapsToDatabase(tempStore, req.ConversationID, layerCtx.Layer4.DetectedGaps)
						}
					}

					// Log key insights for debugging
					if layerCtx.Layer2 != nil && layerCtx.Layer2.IsObviousHarm {
						log.Printf("[MessageProcessor] ⚠ Layer 2: Obvious harm detected")
					}
					if layerCtx.Layer4 != nil && layerCtx.Layer4.GapCount > 0 {
						log.Printf("[MessageProcessor] Layer 4: Detected %d gaps", layerCtx.Layer4.GapCount)
					}
					if layerCtx.Layer5 != nil && layerCtx.Layer5.ConflictCount > 0 {
						log.Printf("[MessageProcessor] Layer 5: Detected %d conflicts", layerCtx.Layer5.ConflictCount)
					}
					if layerCtx.Layer6 != nil && layerCtx.Layer6.IsAmbiguous {
						log.Printf("[MessageProcessor] Layer 6: Request is ambiguous")
					}
					if layerCtx.Layer11 != nil && layerCtx.Layer11.ShouldDeny {
						log.Printf("[MessageProcessor] Layer 11: Should deny request")
					}

				}
			}

		}
	} else if req.ConversationID == "" {
		log.Printf("[MessageProcessor] ⚠ No conversation ID - AnalysisContext not available (new conversation)")
	}

	// CRITICAL FIX 3: Route based on OBVIOUS vs AMBIGUOUS harm
	// If safety alert was detected, decide whether to block or clarify
	if safetyAlertDetected != nil {
		// Check if this is OBVIOUS HARM (always block) or AMBIGUOUS (ask clarification)
		if safetyAlertDetected.IsObviousHarm {
			// Layer 11: Block for obvious harm, but STILL run through orchestrator for consistency
			log.Printf("[MessageProcessor] ✓ OBVIOUS HARM detected - will block after orchestrator (Layer 11)")
			log.Printf("[MessageProcessor] FIXED: Continue through agent to maintain consistency (not short-circuiting)")

			// Record safety incident for audit trail and pattern analysis
			safetyIncidentRepo := srv.database.GetSafetyIncidentRepository()
			if safetyIncidentRepo != nil {
				recordErr := safetyIncidentRepo.Record(
					userID,
					safetyAlertDetected.Severity,
					req.Message,
					"safety_checker",
				)
				if recordErr != nil {
					log.Printf("[MessageProcessor] Warning: Failed to record safety incident: %v", recordErr)
				} else {
					log.Printf("[MessageProcessor] ✓ Recorded safety incident: severity=%s", safetyAlertDetected.Severity)
				}
			}

			// FIXED: Continue through orchestrator instead of returning early
			// Agent will generate denial response through normal flow
		} else {
			// Layer 6-7: AMBIGUOUS case - let agent ask clarification questions
			log.Printf("[MessageProcessor] ✓ AMBIGUOUS violation detected - proceeding to Layer 6-7 clarification (NOT blocking)")
			log.Printf("[MessageProcessor] Suspending safety alert for clarification flow (agent will handle)")
			safetyAlertDetected = nil // Clear the alert so agent can ask clarification
		}
	}

	// Determine if this is the first message of a NEW browser session
	// isNewBrowserSession is true when:
	// 1. Conversation was just created in this request, OR
	// 2. Browser session ID changed (user closed browser and came back)
	isFirstMessageOfSession := isNewBrowserSession || conversationJustCreated

	// Use finalContextMaturity if recalculated, otherwise use initial
	// The maturity the agent decides with is the one gap-based value (agents.GapMaturity). The accomplishment-based
	// value above is kept only for the phase labels in the response metadata.
	contextMaturityForAgent := agents.ConversationGapMaturity(srv.database.GetClarificationQuestionRepository(), conversationID)

	// A goal judged different from the locked one is not taken over: the user is asked first (see PendingGoal).
	if doubtfulFact == "" && analysisCtx != nil {
		if lc, ok := analysisCtx.LayerResults.(*tools.LayerContext); ok && lc != nil && lc.GoalSwitch != "" {
			doubtfulFact = fmt.Sprintf("that you want to change what we are working on, from %q to %q", lc.PrimaryGoal, lc.GoalSwitch)
			log.Printf("[MessageProcessor] Goal switch proposed: the reply asks the user to confirm")
		}
	}

	ctx := models.Context{
		ConversationID: conversationID, // For recording questions and interactions
		AboutMe: &models.AboutMe{
			UserID:             userID,
			CommunicationStyle: aboutMeStyle,
			Values:             aboutMeValues,
			PreferredTone:      aboutMeTone,
		},
		SystemContext:                systemContext, // User's feedback and directives about Moly
		ContactProfile:               contactProfile,
		ConversationHistory:          conversationHistory,
		ExtractedContext:             extractedContext,        // Pass LLM-extracted context to agent
		ExtractedEntities:            extractedEntities,       // Semantic entity classification (self_reference, contact, topic, goal)
		PastIntention:                pastIntention,           // User's goal from previous message(s)
		RecentSafetyIncidents:        recentSafetyIncidents,   // Recent safety alerts to prevent re-alerting
		PrecomputedSafetyVerdict:     safetyAlertDetected,     // Phase 1: Precomputed constitutional evaluation result
		BoundedAnalysisContext:       analysisCtx,             // Hybrid context: summary + recent + profile (700-800 tokens)
		ConversationPhase:            string(execState.Phase), // Current conversation phase for phase-aware responses
		UserBehaviorProfile:          userBehaviorProfile,     // User's learned patterns and preferences
		RelevantReflections:          relevantReflections,     // Past insights from similar conversations
		MissingContext:               gaps,                    // Context-loader fields that are empty (informational, never a question)
		MessageIntent:                string(messageIntent.Intent),
		MessageIntentConfidence:      messageIntent.Confidence,
		IsGreeting:                   isGreetingOrSelfRef,
		PendingNameLabel:             pendingNameLabel,
		ResultNow:                    resultNow,
		DoubtfulFact:                 doubtfulFact,
		NameAnswered:                 messageNameAnswered,
		ContextQuality:               contextQuality,               // Calculated based on loaded fields
		ContextMaturity:              contextMaturityForAgent,      // 0.0-1.0, for Layer 3/8 prerequisites (RECALCULATED value)
		SessionID:                    req.BrowserSessionId,         // Browser session identifier
		IsFirstMessageOfSession:      isFirstMessageOfSession,      // true only for first message in new browser session
		IsFirstMessageInConversation: isFirstMessageInConversation, // true only for first message in this conversation (calculated BEFORE prepending)
		Metadata: map[string]interface{}{
			"extractedEntitiesNeedClarification": extractedEntitiesNeedClarification,
			"extractedEntitiesClarificationQ":    extractedEntitiesClarificationQ,
			"isCurrentMessageGreeting":           isGreetingOrSelfRef,                // Solution 4B: Mark if current message is greeting
			"hasEntityClarification":             extractedEntitiesNeedClarification, // Fix B: Track entity clarification needs
		},
	}

	// Layer 4: Track conflicts detected in this message for confirmation flow
	detectedConflicts := []int64{}

	// Response generation and ethical gate check
	var agentResp *models.ConversationResponse

	// CRITICAL: If Layer 11 said deny, don't generate response (will be handled below)
	if shouldDenyRequest {
		log.Printf("[MessageProcessor] 🚫 LAYER 11 ENFORCEMENT: Will deny request instead of generating response")
	}

	// FIXED: Always run orchestrator, don't use cached responses
	// Each evaluation must be fresh to maintain consistency
	// (Previously skipped on retries, causing inconsistent evaluation)
	if msgProcState != nil && srv.messageProcessingState.IsStageComplete(msgProcState, agents.StageResponseGeneration) {
		log.Printf("[MessageProcessor] ℹ Response generation was previously done, but continuing through orchestrator for consistency")
		// Don't load cached - always regenerate for consistency
	}

	// WEEK 4: Parallelize Layers 6-7 (Response) and Layer 10-11 (Risk/Safety)
	// Both pairs run concurrently to save ~50% of processing time

	// CRITICAL CHECK: Verify analysisCtx is available for ConversationAgent
	// If AnalysisContext build failed, orchestrator won't have run
	if analysisCtx == nil {
		log.Printf("[MessageProcessor] ⚠️ WARNING: AnalysisContext is nil - orchestrator not available")
		log.Printf("[MessageProcessor] ⚠️ This typically happens if database queries failed during context building")
		log.Printf("[MessageProcessor] ⚠️ System will degrade to fallback mode (less informed responses)")
	} else {
		log.Printf("[MessageProcessor] ✓ AnalysisContext available (has %d contacts)", len(analysisCtx.RelevantContacts))
	}

	// CRITICAL: Check orchestrator signals BEFORE generating response
	// Priority order: Deny > Ambiguous/Violation/Conflict
	if shouldDenyRequest {
		log.Printf("[MessageProcessor] 🚫 Layer 11 Denial: Skipping response generation, returning denial")
		// Will be handled after response generation section
	}

	parallelStart := time.Now()

	// Channels for collecting results
	respChan := make(chan *models.ConversationResponse, 1)
	respErrChan := make(chan error, 1)

	// WaitGroup to coordinate goroutines
	var wg sync.WaitGroup
	wg.Add(1)

	// Goroutine 1: Layers 6-7 Response Generation
	go func() {
		defer wg.Done()
		layer67Start := time.Now()
		// CHANGE: Pass analysisCtx to ConversationAgent so it can use orchestrator insights
		// NOTE: analysisCtx may be nil if AnalysisContext building failed (degraded mode)
		resp, err := srv.agentSystem.ConversationAgent.Run(ctx, analysisCtx)
		if err != nil {
			respErrChan <- fmt.Errorf("response generation failed: %v", err)
			return
		}
		if resp == nil {
			respErrChan <- fmt.Errorf("response generation returned nil")
			return
		}
		log.Printf("[MessageProcessor] [Layer 6-7] Response generation complete in %v", time.Since(layer67Start))
		respChan <- resp
	}()

	// Wait for both goroutines to complete
	wg.Wait()
	close(respChan)
	close(respErrChan)

	// Collect results from channels
	select {
	case err := <-respErrChan:
		if err != nil {
			log.Printf("[MessageProcessor] Fatal error: %v\n", err)
			schema.RespondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to process message: %v", err))
			return
		}
	default:
	}

	if resp := <-respChan; resp != nil {
		agentResp = resp
	} else {
		log.Printf("[MessageProcessor] Fatal error: response generation returned nil\n")
		schema.RespondError(w, http.StatusInternalServerError, "Failed to process message: no response generated")
		return
	}

	// Mark stage as complete and store result
	if msgProcState != nil {
		markErr := srv.messageProcessingState.MarkStageComplete(msgProcState, agents.StageResponseGeneration, agentResp)
		if markErr != nil {
			log.Printf("[MessageProcessor] Warning: Failed to mark response generation complete: %v", markErr)
		}
	}

	log.Printf("[MessageProcessor] ⚡ [Week 4] Parallel processing complete in %v", time.Since(parallelStart))

	// Non-fatal errors are captured in response.Error - log but continue
	if agentResp.Error != "" {
		log.Printf("[MessageProcessor] Warning: %s", agentResp.Error)
	}

	// Record HarmAnalyzer-detected ethical interventions to safety_incidents table
	if agentResp.Metadata != nil {
		if intervention, ok := agentResp.Metadata["ethicalIntervention"].(string); ok && intervention != "" {
			safetyIncidentRepo := srv.database.GetSafetyIncidentRepository()
			if safetyIncidentRepo != nil {
				// Map ethical intervention to severity for incident logging
				severity := "warn"
				if intervention == "blocked" {
					severity = "block"
				}
				// Extract reason for incident log
				reason := "ethical_intervention"
				if category, ok := agentResp.Metadata["blockCategory"].(string); ok {
					reason = category
				} else if category, ok := agentResp.Metadata["warningCategory"].(string); ok {
					reason = category
				}
				recordErr := safetyIncidentRepo.Record(
					userID,
					severity,
					reason+": "+req.Message,
					"harm_analyzer",
				)
				if recordErr != nil {
					log.Printf("[MessageProcessor] Warning: Failed to record ethical intervention incident: %v", recordErr)
				} else {
					log.Printf("[MessageProcessor] ✓ Recorded ethical intervention incident: severity=%s category=%s", severity, reason)
				}
			}
		}
	}

	// Add safety alert metadata if detected (for frontend ethical intervention display)
	if safetyAlertDetected != nil {
		if agentResp.Metadata == nil {
			agentResp.Metadata = make(map[string]interface{})
		}
		// Map safety alert to ethical intervention metadata
		ethicalIntervention := "warned"
		if safetyAlertDetected.AlertType == "crisis" {
			ethicalIntervention = "warned"
		} else if safetyAlertDetected.AlertType == "illegal" {
			ethicalIntervention = "warned"
		}
		agentResp.Metadata["ethicalIntervention"] = ethicalIntervention
		agentResp.Metadata["ethicalReason"] = safetyAlertDetected.Title
		agentResp.Metadata["ethicalNote"] = safetyAlertDetected.Message
		log.Printf("[MessageProcessor] ✓ Added ethical intervention metadata: %s (%s)", ethicalIntervention, safetyAlertDetected.Title)
	}

	// Layer 4: Add pending conflict tracking to response metadata
	if len(detectedConflicts) > 0 {
		if agentResp.Metadata == nil {
			agentResp.Metadata = make(map[string]interface{})
		}
		agentResp.Metadata["pendingConflictIDs"] = detectedConflicts
		agentResp.Metadata["conflictConfirmationNeeded"] = true
		log.Printf("[Layer4] ✓ Added %d pending conflicts to response metadata for confirmation", len(detectedConflicts))
	}

	// The skip button: shown under a question the user may skip, once something was answered, and never with a conflict
	// waiting for confirmation. The server checks the same rules again when the button is pressed.
	if agentResp.Metadata == nil {
		agentResp.Metadata = make(map[string]interface{})
	}
	replyKind, _ := agentResp.Metadata["replyExit"].(string)
	if replyKind == "" {
		replyKind, _ = agentResp.Metadata["replyPolicy"].(string)
	}
	canSkip := len(detectedConflicts) == 0 &&
		agents.CanSkip(agents.ReplyKind(replyKind), agents.ConversationGapMaturity(srv.database.GetClarificationQuestionRepository(), conversationID))
	agentResp.Metadata["canSkip"] = canSkip
	srv.setSkipOffer(userID, conversationID, canSkip)

	// Update execution state based on agent response phase
	switch agentResp.Phase {
	case "context_gathering":
		srv.executionStateManager.UpdatePhase(execState, agents.PhaseGatheringContext)
		log.Printf("[MessageProcessor] Phase update: context_gathering")
	case "safety_alert":
		// Safety issue detected - stay in processing but log alert
		srv.executionStateManager.UpdatePhase(execState, agents.PhaseProcessing)
		log.Printf("[MessageProcessor] Phase update: safety_alert (processing state)")
	case "suggestions_ready":
		srv.executionStateManager.UpdatePhase(execState, agents.PhaseProcessing)
		log.Printf("[MessageProcessor] Phase update: suggestions_ready")
	case "responding":
		// Agent is generating response
		srv.executionStateManager.UpdatePhase(execState, agents.PhaseProcessing)
		log.Printf("[MessageProcessor] Phase update: responding")
	case "complete":
		srv.executionStateManager.UpdatePhase(execState, agents.PhaseComplete)
		log.Printf("[MessageProcessor] Phase update: complete")
	case "initial":
		srv.executionStateManager.UpdatePhase(execState, agents.PhaseInitial)
		log.Printf("[MessageProcessor] Phase update: initial")
	default:
		if agentResp.Phase != "" {
			log.Printf("[MessageProcessor] WARNING: Unknown phase '%s', keeping current state", agentResp.Phase)
		}
	}

	log.Printf("[MessageProcessor] Complete: phase=%s\n", agentResp.Phase)

	// SAVE USER MESSAGE to chat_messages for conversation history (with extracted context for audit trail)
	contextExtractedJSON := "{}"
	if extractedContext != nil {
		if b, err := json.Marshal(extractedContext); err == nil {
			contextExtractedJSON = string(b)
		}
	}
	if _, execErr := turn.Exec(`
		INSERT INTO chat_messages (id, user_id, conversation_id, role, content, context_extracted, contact_mention, metadata, created_at)
		VALUES (?, ?, ?, 'user', ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO NOTHING
	`, userMessageID, userID, conversationID, userMessageForDB, contextExtractedJSON, "", "{}", now); execErr != nil {
		log.Printf("[MessageProcessor] WARNING: Failed to save user message to chat_messages: %v", execErr)
	} else {
		log.Printf("[MessageProcessor] ✓ Saved user message to chat_messages with extracted context: %s", userMessageID)

		// FIX #3: Trigger summary update with FULL conversation history
		// Large timeout (2 minutes) to allow older systems to complete LLM summarization
		// Better slow than broken: prioritizes reliability over speed
		if srv.conversationSummaryManager != nil {
			go func(fullHistory []models.Message) {
				// Non-blocking summary update with full conversation history
				// Hardware-aware timeout to support older systems - LLM summarization can be slow
				// Retry once on timeout to handle transient LLM failures
				summaryTimeout := llmCallTimeout
				ctx, cancel := context.WithTimeout(context.Background(), summaryTimeout)
				defer cancel()
				_, summaryErr := srv.conversationSummaryManager.UpdateSummaryIfNeeded(ctx, userID, conversationID, fullHistory, 10)
				if summaryErr != nil {
					log.Printf("[MessageProcessor] Info: Summary update check failed (non-critical): %v", summaryErr)
					// Retry once on timeout or LLM failure
					if strings.Contains(summaryErr.Error(), "context deadline") || strings.Contains(summaryErr.Error(), "LLM") {
						log.Printf("[MessageProcessor] Retrying summary update (attempt 2/2)...")
						retryCtx, retryCancel := context.WithTimeout(context.Background(), summaryTimeout)
						defer retryCancel()
						_, retryErr := srv.conversationSummaryManager.UpdateSummaryIfNeeded(retryCtx, userID, conversationID, fullHistory, 10)
						if retryErr != nil {
							log.Printf("[MessageProcessor] Info: Summary update retry failed: %v (will use stale summary)", retryErr)
						} else {
							log.Printf("[MessageProcessor] ✓ Summary update retry succeeded")
						}
					}
				} else {
					log.Printf("[MessageProcessor] ✓ Summary update check complete (processed %d messages)", len(fullHistory))
				}
			}(conversationHistory)
		}
	}

	// Record user interaction to interactions table for behavioral learning
	interactionRepo := srv.database.GetInteractionRepository()
	if interactionRepo != nil {
		interactionErr := interactionRepo.Save(userID, conversationID, userMessageForDB, "user", map[string]interface{}{
			"extractedContext": extractedContext,
			"phase":            execState.Phase,
		})
		if interactionErr != nil {
			log.Printf("[MessageProcessor] Warning: Failed to record user interaction: %v", interactionErr)
		} else {
			log.Printf("[MessageProcessor] ✓ Recorded user interaction")
		}
	}

	// Record agent response interaction to interactions table
	if interactionRepo != nil {
		interactionErr := interactionRepo.Save(userID, conversationID, agentResp.Response, "agent", map[string]interface{}{
			"phase":    agentResp.Phase,
			"metadata": agentResp.Metadata,
		})
		if interactionErr != nil {
			log.Printf("[MessageProcessor] Warning: Failed to record agent interaction: %v", interactionErr)
		} else {
			log.Printf("[MessageProcessor] ✓ Recorded agent interaction")
		}
	}

	// PHASE 1: SAVE REFLECTION (user insights extracted from conversation)
	if agentResp.Reflection != nil && (len(agentResp.Reflection.Characteristics) > 0 || len(agentResp.Reflection.Interests) > 0) {
		// Mark insight extraction stage as complete
		if msgProcState != nil {
			markErr := srv.messageProcessingState.MarkStageComplete(msgProcState, agents.StageInsightExtraction, agentResp.Reflection)
			if markErr != nil {
				log.Printf("[MessageProcessor] Warning: Failed to mark insight extraction complete: %v", markErr)
			}
		}

		conn := srv.database.GetConnection()

		// Serialize arrays to JSON for storage
		charJSON := []byte("[]")
		if b, err := json.Marshal(agentResp.Reflection.Characteristics); err != nil {
			log.Printf("[MessageProcessor] Warning: Failed to marshal characteristics: %v", err)
		} else {
			charJSON = b
		}
		interestsJSON := []byte("[]")
		if b, err := json.Marshal(agentResp.Reflection.Interests); err != nil {
			log.Printf("[MessageProcessor] Warning: Failed to marshal interests: %v", err)
		} else {
			interestsJSON = b
		}
		intentionsJSON := []byte("[]")
		if b, err := json.Marshal(agentResp.Reflection.Intentions); err != nil {
			log.Printf("[MessageProcessor] Warning: Failed to marshal intentions: %v", err)
		} else {
			intentionsJSON = b
		}

		// Get contact ID if we have an extracted contact
		var contactID *string
		if agentResp.ExtractedContact != nil && agentResp.ExtractedContact.Name != "" {
			// Try to find the contact in database
			var cid string
			err := conn.QueryRow("SELECT id FROM contacts WHERE user_id = ? AND name = ? LIMIT 1",
				userID, agentResp.ExtractedContact.Name).Scan(&cid)
			if err == nil {
				contactID = &cid
			}
		}

		// Get extracted style and intention if available
		extractedStyleStr := ""
		if extractedContext != nil && extractedContext.Style != nil {
			extractedStyleStr = extractedContext.Style.Style
		}
		extractedIntentionStr := ""
		if extractedContext != nil && extractedContext.Intention != "" {
			extractedIntentionStr = extractedContext.Intention
		}

		// Save reflection to database (status = pending_approval, awaiting user confirmation)
		// Link to contact, message, and extracted context
		commPrefsJSON := []byte("[]")
		if agentResp.Reflection != nil && agentResp.Reflection.CommunicationPreferences != "" {
			commPrefsJSON = []byte(`"` + agentResp.Reflection.CommunicationPreferences + `"`)
		}
		quotesJSON := []byte("[]")
		if b, err := json.Marshal(agentResp.Reflection.UserQuotes); err != nil {
			log.Printf("[MessageProcessor] Warning: Failed to marshal user quotes: %v", err)
		} else {
			quotesJSON = b
		}
		editsJSON := []byte("[]")
		if b, err := json.Marshal(agentResp.Reflection.UserEdits); err != nil {
			log.Printf("[MessageProcessor] Warning: Failed to marshal user edits: %v", err)
		} else {
			editsJSON = b
		}
		_, saveErr := turn.Exec(`
			INSERT INTO reflections (user_id, conversation_id, contact_id, message_id, characteristics, interests, intentions, communication_preferences, user_quotes, user_edits, extracted_style, extracted_intention, status, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, userID, conversationID, contactID, userMessageID, string(charJSON), string(interestsJSON), string(intentionsJSON), string(commPrefsJSON), string(quotesJSON), string(editsJSON), extractedStyleStr, extractedIntentionStr, "pending_approval", now)

		if saveErr != nil {
			log.Printf("[MessageProcessor] Warning: Failed to save reflection: %v", saveErr)
		} else if agentResp.Reflection != nil {
			log.Printf("[MessageProcessor] ✓ Saved reflection: %d characteristics, %d interests, %d intentions",
				len(agentResp.Reflection.Characteristics),
				len(agentResp.Reflection.Interests),
				len(agentResp.Reflection.Intentions))
		}
	}

	// PHASE 2: SAVE EXTRACTED CONTEXT (communication style and intention from this message)
	if extractedContext != nil {
		// Initialize context-aware conflict handler (Option B: context tracking)
		// Conflict detection is critical - must not proceed without it
		handler := tools.NewContextAwareConflictHandler(srv.database)
		if handler == nil {
			log.Printf("[MessageProcessor] ERROR: Could not initialize conflict handler - cannot safely process message without conflict detection")
			// Skip all conflict-dependent operations
			schema.RespondError(w, http.StatusInternalServerError, "Conflict handler initialization failed - cannot process message safely")
			return
		}

		// Save extracted style to about_me (if confidence is high)
		if extractedContext.Style != nil && extractedContext.Style.Confidence > 0.6 {
			log.Printf("[MessageProcessor] Checking for style conflict...")

			// Check for conflicts using context-aware handler
			// Handler is guaranteed to be non-nil at this point
			styleDecision := handler.HandleStyleConflict(
				userID,
				conversationID,
				req.Message,
				extractedContext.Style.Style,
				extractedContext.Style.Confidence,
			)
			log.Printf("[MessageProcessor] Style conflict check: action=%s, needsApproval=%v, skipUpdate=%v",
				styleDecision.Action, styleDecision.NeedsApproval, styleDecision.SkipUpdate)

			if styleDecision.HasConflict {
				log.Printf("[MessageProcessor] ⚠ CONFLICT QUEUED: Communication style (ID=%d)", styleDecision.ConflictId)
				detectedConflicts = append(detectedConflicts, styleDecision.ConflictId)
			} else if styleDecision.Action == "auto_merge" {
				log.Printf("[MessageProcessor] AUTO-MERGE: %s", styleDecision.AutoMergeInfo)
			}

			// Only proceed with save if conflict handler says it's OK
			if !styleDecision.SkipUpdate {
				log.Printf("[MessageProcessor] Saving extracted style: %s (confidence=%.2f)", extractedContext.Style.Style, extractedContext.Style.Confidence)

				valuesJSON := "[]"
				if len(extractedContext.Style.Values) > 0 {
					if b, err := json.Marshal(extractedContext.Style.Values); err == nil {
						valuesJSON = string(b)
					}
				}

				// ARCHITECTURAL FIX: Don't confuse observed tone with tone preference
				// - extractedContext.Style.Tone = observable state (current message)
				// - tone_preference = user's preference (configuration)
				// These must never be mixed. Observed tone is ephemeral; preference is stable.
				_, styleErr := turn.Exec(`
					INSERT INTO about_me (user_id, communication_style, core_values, updated_at, created_at)
					VALUES (?, ?, ?, ?, ?)
					ON CONFLICT(user_id) DO UPDATE SET
						communication_style = CASE WHEN communication_style IS NULL OR communication_style = '' THEN excluded.communication_style ELSE communication_style END,
						core_values = CASE WHEN core_values IS NULL OR core_values = '[]' THEN excluded.core_values ELSE core_values END,
						updated_at = excluded.updated_at
				`, userID, extractedContext.Style.Style, valuesJSON, now, now)

				if styleErr != nil {
					log.Printf("[MessageProcessor] Warning: Failed to save extracted style: %v", styleErr)
				} else {
					log.Printf("[MessageProcessor] ✓ Saved extracted style to about_me: %s", extractedContext.Style.Style)
				}
			} else {
				log.Printf("[MessageProcessor] Skipping style save - conflict requires user approval")
			}
		}

		// PHASE 4: Subject-based contact deduplication (Session 15)
		// Groups extracted contacts by subject to handle multi-person messages
		// (e.g., "Christine" and "the girl" both with subject="her" = same person)
		if extractionArtifact != nil && len(extractionArtifact.Entities) > 0 {
			log.Printf("[MessageProcessor] Phase 4: Subject-based contact deduplication starting")
			deduplicator := database.NewContactDeduplicator(srv.database)

			// Group entities by subject
			entitiesBySubject, contactCount, dedupErr := deduplicator.DeduplicateBySubject(userID, extractionArtifact)
			if dedupErr != nil {
				log.Printf("[MessageProcessor] Warning: Subject deduplication failed: %v", dedupErr)
			} else if contactCount > 0 {
				// Merge entities with same subject
				mergedContacts := deduplicator.MergeContactsBySubject(entitiesBySubject)
				log.Printf("[MessageProcessor] Phase 4: Merged %d contact descriptors into %d contacts", contactCount, len(mergedContacts))
			}
		}

		// Save extracted contact to contacts (if confidence is high)
		if extractedContext.Contact != nil && extractedContext.Contact.Confidence > 0.6 {
			log.Printf("[MessageProcessor] Checking for contact duplicates...")

			// Layer 1: Contact Deduplication (Phase 1 implementation)
			// Detects and merges duplicate contacts (generic→specific naming)
			// Now with conversation-aware pronoun resolution: "her" → Christine_sub
			deduplicator := database.NewContactDeduplicator(srv.database)
			dedupDecision, dedupErr := deduplicator.CheckForDuplicateWithConversation(userID, conversationID, extractedContext.Contact)

			if dedupErr != nil {
				log.Printf("[MessageProcessor] Warning: Deduplication error: %v", dedupErr)
				// Continue with normal flow on error
			} else if dedupDecision.ShouldMerge && dedupDecision.ShouldSkipSave {
				// Merge succeeded, contact already updated in database
				log.Printf("[MessageProcessor] ✓ Contact deduplicated: %s (confidence=%.2f, reason=%s)",
					dedupDecision.TargetContact.Name, dedupDecision.Confidence, dedupDecision.MergeReason)
			} else if dedupDecision.NeedsUserApproval && dedupDecision.ShouldSkipSave {
				// Conflict queued for user approval
				log.Printf("[MessageProcessor] ⚠ DEDUP APPROVAL QUEUED: %s (reason=%s)",
					extractedContext.Contact.Name, dedupDecision.MergeReason)
			}

			// Layer 2: Relationship Conflict Check (existing logic)
			// Only proceed if deduplication didn't handle it
			if !dedupDecision.ShouldSkipSave {
				log.Printf("[MessageProcessor] Checking for contact relationship conflicts...")

				// Check for relationship conflicts using context-aware handler
				// Handler is guaranteed to be non-nil at this point
				contactDecision := handler.HandleContactRelationshipConflict(
					userID,
					conversationID,
					req.Message,
					extractedContext.Contact.Name,
					extractedContext.Contact.Relationship,
					extractedContext.Contact.Confidence,
				)
				log.Printf("[MessageProcessor] Contact conflict check: action=%s, needsApproval=%v, skipUpdate=%v",
					contactDecision.Action, contactDecision.NeedsApproval, contactDecision.SkipUpdate)

				if contactDecision.HasConflict {
					log.Printf("[MessageProcessor] ⚠ CONFLICT QUEUED: Contact relationship for %s (ID=%d)", extractedContext.Contact.Name, contactDecision.ConflictId)
					detectedConflicts = append(detectedConflicts, contactDecision.ConflictId)
				} else if contactDecision.Action == "auto_merge" {
					log.Printf("[MessageProcessor] AUTO-MERGE: %s", contactDecision.AutoMergeInfo)
				}

				// Layer 4: Check for contact characteristics conflicts (traits, attributes)
				characteristicsDecision := handler.HandleContactCharacteristicsConflict(
					userID,
					conversationID,
					req.Message,
					extractedContext.Contact.Name,
					extractedContext.Contact.Traits,
					extractedContext.Contact.Confidence,
				)
				log.Printf("[MessageProcessor] Contact characteristics check: action=%s, needsApproval=%v, skipUpdate=%v",
					characteristicsDecision.Action, characteristicsDecision.NeedsApproval, characteristicsDecision.SkipUpdate)

				if characteristicsDecision.HasConflict {
					log.Printf("[MessageProcessor] ⚠ CONFLICT QUEUED: Contact characteristics for %s (ID=%d)", extractedContext.Contact.Name, characteristicsDecision.ConflictId)
					detectedConflicts = append(detectedConflicts, characteristicsDecision.ConflictId)
				}

				// Use most restrictive decision (relationship OR characteristics conflict)
				if contactDecision.SkipUpdate || characteristicsDecision.SkipUpdate {
					contactDecision.SkipUpdate = true
				}

				// The contact row is written only by the identity stage (agents.ResolvePerson). This step records conflicts.
				if contactDecision.SkipUpdate {
					log.Printf("[MessageProcessor] Contact conflict requires user approval; the stored contact is not changed")
				}
			} else {
				log.Printf("[MessageProcessor] Skipping normal contact save - deduplication handled it")
			}
		}

		// Save extracted intention (if present)
		if extractedContext.Intention != "" {
			log.Printf("[MessageProcessor] Checking for intention conflict...")

			// Check for intention conflicts using context-aware handler
			// Handler is guaranteed to be non-nil at this point
			intentionDecision := handler.HandleIntentionConflict(
				userID,
				conversationID,
				req.Message,
				extractedContext.Intention,
			)
			log.Printf("[MessageProcessor] Intention conflict check: action=%s, needsApproval=%v, skipUpdate=%v",
				intentionDecision.Action, intentionDecision.NeedsApproval, intentionDecision.SkipUpdate)

			if intentionDecision.HasConflict {
				log.Printf("[MessageProcessor] ⚠ CONFLICT QUEUED: Intention (ID=%d)", intentionDecision.ConflictId)
				detectedConflicts = append(detectedConflicts, intentionDecision.ConflictId)
			} else if intentionDecision.Action == "auto_merge" {
				log.Printf("[MessageProcessor] AUTO-MERGE: %s", intentionDecision.AutoMergeInfo)
			}

			// Only proceed with save if conflict handler says it's OK
			if !intentionDecision.SkipUpdate {
				log.Printf("[MessageProcessor] Saving extracted intention: %s", extractedContext.Intention)

				_, intentionErr := turn.Exec(`
					INSERT INTO context_attributes (user_id, conversation_id, fact_type, fact_value, attributed_to, confidence, evidence, created_at)
					VALUES (?, ?, 'intention', ?, 'user', 0.8, ?, ?)
				`, userID, conversationID, extractedContext.Intention, req.Message, now)

				if intentionErr != nil {
					log.Printf("[MessageProcessor] Warning: Failed to save extracted intention: %v", intentionErr)
				} else {
					log.Printf("[MessageProcessor] ✓ Saved extracted intention: %s", extractedContext.Intention)
				}
			} else {
				log.Printf("[MessageProcessor] Skipping intention save - conflict requires user approval")
			}
		}

		// Save extracted goals to about_me (if present)
		if extractedContext.Goals != nil && len(extractedContext.Goals) > 0 {
			log.Printf("[MessageProcessor] Saving extracted goals: %v", extractedContext.Goals)

			goalsJSON, _ := json.Marshal(extractedContext.Goals)
			_, goalsErr := turn.Exec(`
				INSERT INTO about_me (user_id, goals, updated_at, created_at)
				VALUES (?, ?, ?, ?)
				ON CONFLICT(user_id) DO UPDATE SET
					goals = CASE WHEN goals IS NULL OR goals = '[]' THEN excluded.goals ELSE goals END,
					updated_at = excluded.updated_at
			`, userID, string(goalsJSON), now, now)

			if goalsErr != nil {
				log.Printf("[MessageProcessor] Warning: Failed to save extracted goals: %v", goalsErr)
			} else {
				log.Printf("[MessageProcessor] ✓ Saved extracted goals: %d goals", len(extractedContext.Goals))
			}
		}

		// Save extracted user values to about_me (if present)
		if extractedContext.UserValues != nil && len(extractedContext.UserValues) > 0 {
			log.Printf("[MessageProcessor] Saving extracted user values: %v", extractedContext.UserValues)

			valuesJSON, _ := json.Marshal(extractedContext.UserValues)
			_, valuesErr := turn.Exec(`
				INSERT INTO about_me (user_id, core_values, updated_at, created_at)
				VALUES (?, ?, ?, ?)
				ON CONFLICT(user_id) DO UPDATE SET
					core_values = CASE WHEN core_values IS NULL OR core_values = '[]' THEN excluded.core_values ELSE core_values END,
					updated_at = excluded.updated_at
			`, userID, string(valuesJSON), now, now)

			if valuesErr != nil {
				log.Printf("[MessageProcessor] Warning: Failed to save extracted user values: %v", valuesErr)
			} else {
				log.Printf("[MessageProcessor] ✓ Saved extracted user values: %d values", len(extractedContext.UserValues))
			}
		}

		// Save extracted principles (IntentionPrinciples) to about_me
		if extractedContext.IntentionPrinciples != nil && len(extractedContext.IntentionPrinciples) > 0 {
			log.Printf("[MessageProcessor] Saving extracted principles: %v", extractedContext.IntentionPrinciples)

			principlesJSON, _ := json.Marshal(extractedContext.IntentionPrinciples)
			_, principlesErr := turn.Exec(`
				INSERT INTO about_me (user_id, principles, updated_at, created_at)
				VALUES (?, ?, ?, ?)
				ON CONFLICT(user_id) DO UPDATE SET
					principles = CASE WHEN principles IS NULL OR principles = '[]' THEN excluded.principles ELSE principles END,
					updated_at = excluded.updated_at
			`, userID, string(principlesJSON), now, now)

			if principlesErr != nil {
				log.Printf("[MessageProcessor] Warning: Failed to save extracted principles: %v", principlesErr)
			} else {
				log.Printf("[MessageProcessor] ✓ Saved extracted principles: %d principles", len(extractedContext.IntentionPrinciples))
			}
		}

		// Save extracted user characteristics to about_me
		if extractedContext.UserCharacteristics != nil && len(extractedContext.UserCharacteristics) > 0 {
			log.Printf("[MessageProcessor] Saving extracted user characteristics: %v", extractedContext.UserCharacteristics)

			charJSON, _ := json.Marshal(extractedContext.UserCharacteristics)
			_, charErr := turn.Exec(`
				INSERT INTO about_me (user_id, characteristics, updated_at, created_at)
				VALUES (?, ?, ?, ?)
				ON CONFLICT(user_id) DO UPDATE SET
					characteristics = CASE WHEN characteristics IS NULL OR characteristics = '[]' THEN excluded.characteristics ELSE characteristics END,
					updated_at = excluded.updated_at
			`, userID, string(charJSON), now, now)

			if charErr != nil {
				log.Printf("[MessageProcessor] Warning: Failed to save extracted user characteristics: %v", charErr)
			} else {
				log.Printf("[MessageProcessor] ✓ Saved extracted user characteristics: %d characteristics", len(extractedContext.UserCharacteristics))
			}
		}

		// PHASE 3: Save extracted system feedback to system_context
		if extractedContext.SystemFeedback != nil && extractedContext.SystemFeedback.IsFeedback {
			log.Printf("[MessageProcessor] Saving extracted system feedback: type=%s feedback=%v directives=%v",
				extractedContext.SystemFeedback.FeedbackType, extractedContext.SystemFeedback.Feedback, extractedContext.SystemFeedback.Directives)

			feedbackJSON, _ := json.Marshal(extractedContext.SystemFeedback.Feedback)
			directivesJSON, _ := json.Marshal(extractedContext.SystemFeedback.Directives)
			perceptionsJSON, _ := json.Marshal(extractedContext.SystemFeedback.Perceptions)

			_, sysFbErr := turn.Exec(`
				INSERT INTO system_context (user_id, user_feedback, user_directives, system_perceptions,
				                           preferred_interaction_style, updated_at, created_at)
				VALUES (?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT(user_id) DO UPDATE SET
					user_feedback = CASE WHEN user_feedback IS NULL OR user_feedback = '[]' THEN excluded.user_feedback ELSE user_feedback END,
					user_directives = CASE WHEN user_directives IS NULL OR user_directives = '[]' THEN excluded.user_directives ELSE user_directives END,
					system_perceptions = CASE WHEN system_perceptions IS NULL OR system_perceptions = '[]' THEN excluded.system_perceptions ELSE system_perceptions END,
					preferred_interaction_style = COALESCE(NULLIF(?, ''), system_context.preferred_interaction_style),
					updated_at = excluded.updated_at
			`, userID, string(feedbackJSON), string(directivesJSON), string(perceptionsJSON),
				extractedContext.SystemFeedback.Style, now, now,
				extractedContext.SystemFeedback.Style)

			if sysFbErr != nil {
				log.Printf("[MessageProcessor] Warning: Failed to save system feedback: %v", sysFbErr)
			} else {
				log.Printf("[MessageProcessor] ✓ Saved system feedback: feedback=%d directives=%d perceptions=%d style=%s",
					len(extractedContext.SystemFeedback.Feedback), len(extractedContext.SystemFeedback.Directives),
					len(extractedContext.SystemFeedback.Perceptions), extractedContext.SystemFeedback.Style)
			}
		}

		// Contact traits are written by the identity stage (agents.ResolvePerson), not here.

		// NEW VALIDATION: Check if response contradicts extracted user characteristics
		// Extract user properties from the extraction artifact and check against response
		if agentResp.Response != "" && extractionArtifact != nil {
			userEntities := extractionArtifact.GetEntitiesBySubject("user")
			userCharacteristics := make([]string, 0)

			// Collect all user characteristics/properties from extracted entities
			for _, entity := range userEntities {
				// Look for property/characteristic type entities
				if entity.Type == "property" || entity.Type == "characteristic" ||
					entity.Type == "attribute" || strings.Contains(entity.Type, "property") {
					if entity.Value != "" {
						userCharacteristics = append(userCharacteristics, entity.Value)
					}
				}
			}

			if len(userCharacteristics) > 0 {
				hasResponseConflict, userChar, contraryWord, conflictReason := handler.ValidateResponseAgainstCharacteristics(
					userID,
					conversationID,
					agentResp.Response,
					userCharacteristics,
				)

				if hasResponseConflict {
					log.Printf("[MessageProcessor] ⚠ CRITICAL: Response contradicts extracted user characteristic '%s': %s", userChar, conflictReason)
					// FIX #1: BLOCK response when internal contradiction detected (don't just mark it)
					// User said they're dominant, but response suggests submissive → MUST ask clarification
					if agentResp.Metadata == nil {
						agentResp.Metadata = make(map[string]interface{})
					}
					agentResp.Metadata["responseCharacteristicConflict"] = map[string]string{
						"userCharacteristic": userChar,
						"contraryWord":       contraryWord,
						"conflictReason":     conflictReason,
					}
					agentResp.Metadata["responseBlocked"] = true // FIX #1: Flag to block response
					agentResp.Metadata["blockReason"] = fmt.Sprintf("Response contradicts extracted characteristic: %s", userChar)
					log.Printf("[MessageProcessor] ⚠ RESPONSE BLOCKED: %s - will ask clarification instead", userChar)
				}
			}
		}
	}

	// Determine if clarification is needed based on gaps remaining
	// (NEW) Map ConversationResponse to Phase5 frontend response format
	needsClarification := false
	var clarificationQs []map[string]interface{}

	// Initialize clarification capture for Phase 2 & 3 wiring
	clarificationCapture := database.NewClarificationCapture(srv.database)

	// NEW: Check for marked response characteristic conflicts and add clarification
	if agentResp.Metadata != nil {
		if conflictData, hasConflict := agentResp.Metadata["responseCharacteristicConflict"]; hasConflict {
			if conflictMap, ok := conflictData.(map[string]string); ok {
				userChar := conflictMap["userCharacteristic"]
				contraryWord := conflictMap["contraryWord"]
				conflictReason := conflictMap["conflictReason"]

				if userChar == "" || contraryWord == "" {
					log.Printf("[MessageProcessor] Warning: Response conflict metadata missing required fields (userChar=%s, contraryWord=%s)", userChar, contraryWord)
				} else {
					needsClarification = true
					questionID := fmt.Sprintf("response_conflict_q_%d_%d", time.Now().UnixNano(), len(clarificationQs))
					clarificationQs = append(clarificationQs, map[string]interface{}{
						"id":          questionID,
						"type":        "characteristic_contradiction", // New conflict type
						"question":    fmt.Sprintf("I suggested exploring your %s nature, but you mentioned being %s. Which one is accurate?", contraryWord, userChar),
						"options":     []string{},
						"linkedFacts": []string{userChar},
						"priority":    3, // Highest priority - core contradiction
						"status":      "pending",
						"context":     conflictReason,
					})
					log.Printf("[MessageProcessor] ✓ RESPONSE CONFLICT CLARIFICATION ADDED: %s", conflictReason)
				}
			} else {
				log.Printf("[MessageProcessor] Warning: Response conflict metadata type assertion failed")
			}
		}
	}

	// Fix B: First, add entity extraction clarifications if any
	// These should be asked before gap clarifications to resolve ambiguity
	// Fix K: Deduplicate entity clarifications - don't ask if already asked recently
	if extractedEntitiesNeedClarification && extractedEntitiesClarificationQ != "" {
		// Check if this entity question has been asked in recent conversation history
		questionAlreadyAsked := false
		for _, msg := range conversationHistory {
			// Look for assistant messages that contain the clarification question
			if msg.Role == "assistant" && strings.Contains(strings.ToLower(msg.Content), strings.ToLower(extractedEntitiesClarificationQ)) {
				questionAlreadyAsked = true
				log.Printf("[MessageProcessor] ⚠ Entity clarification already asked: %s", extractedEntitiesClarificationQ)
				break
			}
		}

		if !questionAlreadyAsked {
			// Phase 3: Filter obvious questions (wired)
			isObvious := clarificationCapture.IsObviousQuestion(extractedEntitiesClarificationQ, req.Message)

			if isObvious {
				log.Printf("[MessageProcessor] ⊘ ENTITY CLARIFICATION FILTERED (obvious question): %s", extractedEntitiesClarificationQ)
			} else {
				needsClarification = true
				questionID := fmt.Sprintf("entity_q_%d_%d", time.Now().UnixNano(), len(clarificationQs))
				clarificationQs = append(clarificationQs, map[string]interface{}{
					"id":          questionID,
					"type":        "entity_ambiguity", // Entity extraction clarifications
					"question":    extractedEntitiesClarificationQ,
					"options":     []string{},
					"linkedFacts": []string{},
					"priority":    2, // Higher priority than gap clarifications
					"status":      "pending",
					"context":     "Clarifying ambiguous entity from message",
				})
				log.Printf("[MessageProcessor] ✓ ENTITY CLARIFICATION ENABLED: %s", extractedEntitiesClarificationQ)
			}
		} else {
			log.Printf("[MessageProcessor] ⊘ ENTITY CLARIFICATION SKIPPED (already asked)")
		}
	}

	// Fix F: Use hasSignificantGaps variable instead of recalculating
	// Phase-aware threshold was already applied in gap detection above
	if hasSignificantGaps && len(gaps) > 0 {
		// Fix A: Only convert to gap clarification if agent response is actually about gaps
		// Don't wrap topic-shift or other response types as gap clarification
		if agentResp.Response != "" && (agentResp.Phase == "context_gathering" || agentResp.Phase == "") {
			needsClarification = true
			// Generate unique ID for this clarification question
			questionID := fmt.Sprintf("gap_q_%d_%d", time.Now().UnixNano(), len(clarificationQs))
			clarificationQs = append(clarificationQs, map[string]interface{}{
				"id":          questionID,
				"type":        "user_context", // Gap clarifications are about user context
				"question":    agentResp.Response,
				"options":     []string{},
				"linkedFacts": []string{},
				"priority":    1,
				"status":      "pending",
				"context":     fmt.Sprintf("Clarifying gaps: %v", gaps),
			})
			log.Printf("[MessageProcessor] ✓ CLARIFICATION ENABLED: %d gaps exceed threshold of %d", len(gaps), gapThreshold)
		} else if agentResp.Phase != "context_gathering" && agentResp.Phase != "" {
			// Agent generated a different response type (topic_shift, suggestions, etc) - preserve it
			log.Printf("[MessageProcessor] ℹ Gaps detected (%d > %d) but agent generated %s response - preserving response type", len(gaps), gapThreshold, agentResp.Phase)
		}
	}

	// FIX #1: CHECK LAYER GATES BEFORE ROUTING (NEW)
	// Layers can control response flow through gates
	layerGatesBlock := false
	var gateBlockReasons []string

	// Extract layerCtx from analysisCtx (where it was stored during orchestration)
	var layerCtxForGates *tools.LayerContext
	if analysisCtx != nil && analysisCtx.LayerResults != nil {
		if lc, ok := analysisCtx.LayerResults.(*tools.LayerContext); ok {
			layerCtxForGates = lc
		}
	}

	if layerCtxForGates != nil {
		// FIX #1: Check Layer 11 denial protocol (has ShouldDeny field)
		if layerCtxForGates.Layer11 != nil && layerCtxForGates.Layer11.ShouldDeny {
			layerGatesBlock = true
			gateBlockReasons = append(gateBlockReasons, fmt.Sprintf("Layer11 denial: %s", layerCtxForGates.Layer11.Reason))
			log.Printf("[MessageProcessor] ⚠ LAYER 11 GATE TRIGGERED - Response denied: %s", layerCtxForGates.Layer11.Reason)
		}

		// FIX #11: Check Layer 6 ambiguity gate (NEW)
		// If request is ambiguous, ask clarification before responding
		if layerCtxForGates.Layer6 != nil && layerCtxForGates.Layer6.IsAmbiguous {
			needsClarification = true
			log.Printf("[MessageProcessor] ℹ LAYER 6 GATE: Request is ambiguous - requiring clarification")
			gateBlockReasons = append(gateBlockReasons, fmt.Sprintf("Layer6: ambiguous request needs clarification"))
		}

		// FIX #11: Check Layer 7 principle violation gate (NEW)
		// If principle violations detected, ask clarification questions before responding
		if layerCtxForGates.Layer7 != nil && len(layerCtxForGates.Layer7.ClarificationQuestions) > 0 {
			needsClarification = true
			log.Printf("[MessageProcessor] ℹ LAYER 7 GATE: Principle issues detected - requiring clarification (%d questions)",
				len(layerCtxForGates.Layer7.ClarificationQuestions))
			gateBlockReasons = append(gateBlockReasons, fmt.Sprintf("Layer7: principle issues need clarification"))
		}

		// FIX #1: Check Layer 8 deepening gate (using Depth field)
		// If depth is "surface" or "moderate" and agent wants to go "deep", block it
		if layerCtxForGates.Layer8 != nil && layerCtxForGates.Layer8.Depth != "deep" && agentResp.Phase == "socratic" {
			log.Printf("[MessageProcessor] ℹ LAYER 8 GATE: Socratic depth is %s, not blocking full socratic (can proceed with limited depth)", layerCtxForGates.Layer8.Depth)
		}

		// FIX #1: Check Layer 3 maturity gate
		// If maturity is immature and agent wants to go deep, suggest clarification first
		if layerCtxForGates.Layer3 != nil && layerCtxForGates.Layer3.MaturityScore < 0.3 && agentResp.Phase == "socratic" {
			log.Printf("[MessageProcessor] ℹ LAYER 3 GATE: Low maturity (%.2f) but agent generated socratic - will add clarification reminder", layerCtxForGates.Layer3.MaturityScore)
			needsClarification = true
			gateBlockReasons = append(gateBlockReasons, fmt.Sprintf("Layer3: low maturity (%.2f) needs clarification first", layerCtxForGates.Layer3.MaturityScore))
		}
	}

	// FIX #1: Check if response should be blocked (response contradicts extraction)
	responseBlocked := layerGatesBlock
	if agentResp != nil && agentResp.Metadata != nil {
		if blocked, isBlocked := agentResp.Metadata["responseBlocked"].(bool); isBlocked && blocked {
			log.Printf("[MessageProcessor] ⚠ RESPONSE BLOCKED - Will send clarification instead")
			responseBlocked = true
		}
	}

	// Build action_required field for frontend (Phase5 format)
	// Fix D: Include conflicts in action_required for conflict confirmation flow
	// FIX #2: Include conflict metadata so frontend can show why clarification needed (NEW)
	conflictDetails := []map[string]interface{}{}
	if agentResp != nil && agentResp.Metadata != nil {
		// Extract conflict details from response metadata
		if conflicts, ok := agentResp.Metadata["conflicts"].([]interface{}); ok {
			for _, c := range conflicts {
				if conflictMap, ok := c.(map[string]interface{}); ok {
					conflictDetails = append(conflictDetails, conflictMap)
				}
			}
		}
	}

	actionRequired := map[string]interface{}{
		"needsClarification": needsClarification,
		"clarificationQs":    clarificationQs,
		"hasConflicts":       len(detectedConflicts) > 0 || len(conflictDetails) > 0,
		"conflictIDs":        detectedConflicts,
		"conflicts":          conflictDetails, // FIX #2: Full conflict details, not just IDs
	}

	// FIX #1: If response is blocked due to contradiction, force clarification mode
	if responseBlocked {
		log.Printf("[MessageProcessor] ⚠ RESPONSE BLOCKED - Forcing clarification mode")
		actionRequired["needsClarification"] = true
		actionRequired["isResponseBlocked"] = true
		actionRequired["blockReason"] = "Response contradicts extracted user characteristics"
		// Clarification questions should already be in clarificationQs from earlier code
		log.Printf("[MessageProcessor] ✓ Response blocked, clarification forced (questions: %d)", len(clarificationQs))
	}

	if len(detectedConflicts) > 0 {
		log.Printf("[MessageProcessor] ✓ Added %d conflicts to action_required", len(detectedConflicts))
	}

	// Load pending clarification questions for this conversation (Layer 3)
	var pendingClarificationQuestions []interface{}
	if conversationID != "" {
		pending, err := clarificationCapture.GetPendingClarifications(conversationID)
		if err != nil {
			log.Printf("[MessageProcessor] Warning: Failed to load pending clarification questions: %v", err)
		} else if pending != nil {
			pendingClarificationQuestions = pending
			log.Printf("[MessageProcessor] ✓ Loaded %d pending clarification questions for conversation %s", len(pending), conversationID)
		}
	}

	// CRITICAL FIX: Ensure metadata exists BEFORE building response map
	// If metadata is nil when added to response, later modifications won't show up
	if agentResp.Metadata == nil {
		agentResp.Metadata = make(map[string]interface{})
	}

	// PHASE 5: Contact-Aware Response Formatting
	// Format response based on contact resolution status
	var contactFormatter *agents.ContactResponseFormatter
	var formattedResponse map[string]interface{}
	var response map[string]interface{}

	if srv.database != nil {
		contactFormatter = agents.NewContactResponseFormatter(srv.database)

		// Load active contacts for formatting
		contactRepo := database.NewContactRepository(srv.database)
		activeContacts, contactErr := contactRepo.GetByUserID(userID)
		if contactErr != nil {
			log.Printf("[MessageProcessor] ⚠️ PHASE 5: Error loading contacts for formatting: %v", contactErr)
			activeContacts = make([]*models.Contact, 0)
		}

		// Format response based on contact state
		if len(activeContacts) > 0 {
			log.Printf("[MessageProcessor] ✓ PHASE 5: Formatting response with contact awareness (%d contacts)", len(activeContacts))
			formattedResponse = contactFormatter.FormatResponse(
				layerCtx,
				activeContacts,
				agentResp.Response,
				agentResp.Metadata,
			)
			log.Printf("[MessageProcessor] ✓ PHASE 5: Response formatted - summary: %s",
				contactFormatter.GetFormattedContactSummary(activeContacts))
		} else {
			log.Printf("[MessageProcessor] ℹ PHASE 5: No active contacts, using standard response format")
		}
	}

	// If contact formatting wasn't applied, use standard format
	if formattedResponse == nil {
		log.Printf("[MessageProcessor] ℹ PHASE 5: Using standard response format (no contact formatting)")

		// Map ConversationResponse to frontend response format
		response = map[string]interface{}{
			"action_required":               actionRequired, // Frontend expects this structure
			"success":                       true,
			"phase":                         agentResp.Phase,
			"conversationId":                conversationID,                // Return conversation ID so frontend can store it
			"pendingClarificationQuestions": pendingClarificationQuestions, // Layer 3: Pending questions for user
			// ConversationAgent specific fields
			"response":         agentResp.Response,
			"safetyAlert":      agentResp.SafetyAlert,
			"processingTimeMs": agentResp.ProcessingTimeMs,
			"metadata":         agentResp.Metadata,
			"reflection":       agentResp.Reflection,
			"extractedContact": agentResp.ExtractedContact,
		}
	} else {
		// Use contact-formatted response
		log.Printf("[MessageProcessor] ✓ PHASE 5: Using contact-formatted response")
		response = formattedResponse

		// Add standard fields to formatted response
		response["conversationId"] = conversationID
		response["processingTimeMs"] = agentResp.ProcessingTimeMs
		if agentResp.SafetyAlert != nil {
			response["safetyAlert"] = agentResp.SafetyAlert
		}
		if agentResp.Reflection != nil {
			response["reflection"] = agentResp.Reflection
		}
		if agentResp.ExtractedContact != nil {
			response["extractedContact"] = agentResp.ExtractedContact
		}
		if pendingClarificationQuestions != nil && len(pendingClarificationQuestions) > 0 {
			response["pendingClarificationQuestions"] = pendingClarificationQuestions
		}
	}

	// Add error field only if present (non-fatal errors)
	if agentResp.Error != "" {
		response["error"] = agentResp.Error
	}

	// The maturity the user's conversation has reached: answered questions over answered plus open ones.
	agentResp.Metadata["maturity"] = contextMaturityForAgent

	// Fix L: Include past reflection statuses in metadata for tracking approved/rejected insights
	if len(relevantReflections) > 0 {
		reflectionStatuses := make([]map[string]interface{}, 0)
		for _, reflection := range relevantReflections {
			reflectionStatuses = append(reflectionStatuses, map[string]interface{}{
				"id":     reflection.ID,
				"status": reflection.Status,
				"summary": fmt.Sprintf("%d chars, %d interests, %d intentions",
					len(reflection.Characteristics), len(reflection.Interests), len(reflection.Intentions)),
			})
		}
		agentResp.Metadata["pastReflectionStatuses"] = reflectionStatuses
		log.Printf("[MessageProcessor] ✓ Added %d past reflection statuses to metadata", len(reflectionStatuses))
	}

	// CHANGE 2: Add orchestrator insights to response metadata
	// Extract key insights from LayerContext and add to response for frontend use
	if analysisCtx != nil && analysisCtx.LayerResults != nil {
		if layerCtx, ok := analysisCtx.LayerResults.(*tools.LayerContext); ok {
			orchestratorInsights := make(map[string]interface{})

			// Layer 1: Extraction confidence
			if layerCtx.Layer1 != nil {
				orchestratorInsights["extractionConfidence"] = layerCtx.Layer1.Confidence
			}

			// Layer 2: Principle violations
			if layerCtx.Layer2 != nil {
				orchestratorInsights["isObviousHarm"] = layerCtx.Layer2.IsObviousHarm
				if layerCtx.Layer2.Verdict != nil {
					orchestratorInsights["overallSeverity"] = layerCtx.Layer2.Verdict.OverallSeverity
					orchestratorInsights["matchedPrinciples"] = len(layerCtx.Layer2.Verdict.MatchedPrinciples)
				}
			}

			// Layer 3: Maturity
			if layerCtx.Layer3 != nil {
				orchestratorInsights["maturityScore"] = layerCtx.Layer3.MaturityScore
				orchestratorInsights["contextQuality"] = layerCtx.Layer3.ContextQuality
				orchestratorInsights["gateLevel"] = layerCtx.Layer3.GateLevel
			}

			// Layer 4: Gaps
			if layerCtx.Layer4 != nil && layerCtx.Layer4.GapCount > 0 {
				gapSummary := make([]map[string]interface{}, 0)
				for _, gap := range layerCtx.Layer4.DetectedGaps {
					gapSummary = append(gapSummary, map[string]interface{}{
						"type":       gap.Type,
						"severity":   gap.Severity,
						"confidence": gap.Confidence,
					})
				}
				orchestratorInsights["detectedGaps"] = gapSummary
				orchestratorInsights["gapCount"] = layerCtx.Layer4.GapCount
			}

			// Layer 5: Conflicts
			if layerCtx.Layer5 != nil && layerCtx.Layer5.ConflictCount > 0 {
				conflictSummary := make([]map[string]interface{}, 0)
				for _, conflict := range layerCtx.Layer5.DetectedConflicts {
					conflictSummary = append(conflictSummary, map[string]interface{}{
						"type":       conflict.Type,
						"severity":   conflict.Severity,
						"confidence": conflict.Confidence,
					})
				}
				orchestratorInsights["detectedConflicts"] = conflictSummary
				orchestratorInsights["conflictCount"] = layerCtx.Layer5.ConflictCount
			}

			// Layer 6: Ambiguity
			if layerCtx.Layer6 != nil && layerCtx.Layer6.IsAmbiguous {
				orchestratorInsights["isAmbiguous"] = true
				orchestratorInsights["ambiguousElements"] = layerCtx.Layer6.AmbiguousElements
			}

			// Layer 8: Socratic questions
			if layerCtx.Layer8 != nil && len(layerCtx.Layer8.SocraticQuestions) > 0 {
				orchestratorInsights["socraticQuestions"] = layerCtx.Layer8.SocraticQuestions
				orchestratorInsights["questionDepth"] = layerCtx.Layer8.Depth
			}

			// Layer 11: Denial detection
			if layerCtx.Layer11 != nil && layerCtx.Layer11.ShouldDeny {
				orchestratorInsights["shouldDeny"] = true
				orchestratorInsights["denialReason"] = layerCtx.Layer11.Reason
			}

			// Add to response
			response["orchestratorInsights"] = orchestratorInsights
			log.Printf("[MessageProcessor] ✓ Added orchestrator insights to response")
		}
	}

	// PHASE 7: Record this interaction for behavioral profile learning
	if srv.database != nil {
		learningAgent := srv.GetLearningAgent(userID) // Use cached learning agent
		if learningAgent != nil {
			interactionData := models.InteractionData{
				UserID:               userID,
				ConversationID:       req.ConversationID,
				UserMessage:          req.Message,
				SuggestionsGenerated: 0, // Could count actual suggestions if generated
				CreatedAt:            time.Now().Unix(),
			}
			recordErr := learningAgent.RecordInteraction(interactionData)
			if recordErr != nil {
				log.Printf("[MessageProcessor] Warning: Failed to record interaction: %v", recordErr)
			} else {
				log.Printf("[MessageProcessor] ✓ Recorded interaction for user %s", userID)
			}
		}
	}

	// FIX #2 (Session 34): Maturity is saved by Layer 3 (line 570 in layer_adapters.go)
	// Do NOT save again here - Layer 3 already persisted the updated maturity context
	// This unified approach ensures single object lifecycle and prevents duplicate saves overwriting Layer 3's work

	// FIX #3 (Phase 3): Detect if maturity is stuck (prevent infinite clarification loops)
	if layerCtx != nil && layerCtx.Layer3 != nil && response != nil {
		currentMaturity := layerCtx.Layer3.MaturityScore

		// Load previous maturity from analysisCtx's previous response metadata
		var previousMaturity float64
		var flatCount int

		if analysisCtx != nil && analysisCtx.PreviousResponseMetadata != nil {
			if prevMat, ok := analysisCtx.PreviousResponseMetadata["maturityScore"].(float64); ok {
				previousMaturity = prevMat
			}
			if flat, ok := analysisCtx.PreviousResponseMetadata["maturityFlatCount"].(float64); ok {
				flatCount = int(flat)
			}
		}

		// Check if maturity is flat (within 5% tolerance)
		const maturityTolerance = 0.05
		isFlat := previousMaturity > 0 &&
			math.Abs(currentMaturity-previousMaturity) < maturityTolerance

		// Detect stuck: flat for 2+ consecutive messages
		isStuck := false
		stuckReason := ""
		if isFlat {
			flatCount++
			if flatCount >= 2 {
				isStuck = true
				stuckReason = fmt.Sprintf(
					"Maturity score %.2f unchanged for %d consecutive messages (tolerance=%.1f%%)",
					currentMaturity, flatCount, maturityTolerance*100,
				)
				log.Printf("[MessageProcessor] 🚨 FIX #3 (Phase 3): Maturity STUCK - %s", stuckReason)
			} else {
				log.Printf("[MessageProcessor] ⚠️ FIX #3 (Phase 3): Maturity flat (prev=%.2f, current=%.2f) - will break if repeats",
					previousMaturity, currentMaturity)
			}
		} else {
			// Maturity improved - reset flat counter
			flatCount = 0
		}

		// Store metrics in response metadata for next message to check
		if metadata, ok := response["metadata"].(map[string]interface{}); ok {
			metadata["maturityScore"] = currentMaturity
			metadata["maturityStuck"] = isStuck
			metadata["maturityStuckReason"] = stuckReason
			metadata["maturityFlatCount"] = flatCount
			if isFlat {
				metadata["maturityTrend"] = "flat"
			} else {
				metadata["maturityTrend"] = "improving"
			}
			log.Printf("[MessageProcessor] ✓ FIX #3: Stored maturity metrics (stuck=%v, trend=%v, flatCount=%d)",
				isStuck, metadata["maturityTrend"], flatCount)
		}
	}

	// FIX #1+#9: Save extraction state for next message (context accumulation)
	// CRITICAL: Must save MERGED extraction (accumulated + current), not just current!
	// This ensures next message has full context history, not just last message's data
	if layerCtx != nil && layerCtx.Layer1 != nil && layerCtx.Layer1.ExtractedContext != nil {
		// FIX #4 (Phase 4): Apply layer-based accumulated updates before merging
		// When layers resolve contradictions, the accumulated context should be refined
		accumulatedToMerge := previousEntities(previousExtraction)
		if layerCtx.UpdatedAccumulatedEntities != nil && len(layerCtx.UpdatedAccumulatedEntities) > 0 {
			// Use updated accumulated entities (refined by layer resolutions)
			// Start with original accumulated, then apply updates
			accumulatedToMerge = layerCtx.UpdatedAccumulatedEntities
			log.Printf("[MessageProcessor] ✓ FIX #4 (Phase 4): Applied %d layer-based accumulated updates before merge",
				len(layerCtx.UpdatedAccumulatedEntities))
		}

		// Merge accumulated + current entities before saving
		var entitiesToSave []models.ExtractedEntity
		if previousExtraction != nil && len(previousExtraction.Entities) > 0 {
			// Merge: keep accumulated entities (with updates) + add new ones
			entitiesToSave = mergeEntities(
				accumulatedToMerge,            // accumulated from earlier (with FIX #4 updates)
				analysisCtx.ExtractedEntities, // current from this message
			)
			log.Printf("[MessageProcessor] ✓ FIX #9: Merged extraction: %d accumulated + current = %d total",
				len(accumulatedToMerge), len(entitiesToSave))
		} else {
			// No previous data, save current
			entitiesToSave = analysisCtx.ExtractedEntities
		}

		extraction := &PreviousExtraction{
			Entities:       entitiesToSave, // MERGED! This is the key fix
			Goal:           layerCtx.Layer1.ExtractedContext.Intention,
			Values:         layerCtx.Layer1.ExtractedContext.UserValues,
			PrimaryGoal:    layerCtx.PrimaryGoal,     // FIX #4: Save locked primary goal
			PendingGoal:    layerCtx.GoalSwitch,      // asked about now, answered next message
			Progression:    layerCtx.GoalProgression, // FIX #4: Save goal evolution
			ContextTracker: analysisCtx.ContextTracker,
		}
		srv.savePreviousExtractionWith(turn, userID, req.ConversationID, extraction)
		log.Printf("[MessageProcessor] ✓ FIX #1+#4+#9: Saved MERGED extraction + primary goal=%q for next message (entities=%d)",
			extraction.PrimaryGoal, len(extraction.Entities))

		// FIX #6 (Phase 6): Update conversation summary with accumulated insights
		// This wires the final step of the Accumulated Insights Model
		if srv.database != nil && analysisCtx.ConversationSummary != nil && layerCtx.Layer3 != nil {
			summary := analysisCtx.ConversationSummary

			// FIX #10: Correctly accumulate entity count (not reset!)
			// Load existing count from DB, add new entities
			existingCount := summary.AccumulatedEntityCount
			newEntityCount := len(entitiesToSave)
			summary.AccumulatedEntityCount = existingCount + newEntityCount
			log.Printf("[MessageProcessor] FIX #10: Accumulated entities - existing=%d + new=%d = %d total",
				existingCount, newEntityCount, summary.AccumulatedEntityCount)

			// Update accumulated contact count
			uniqueContacts := make(map[string]bool)
			for _, entity := range entitiesToSave {
				if entity.Type == "contact" && entity.Value != "" {
					uniqueContacts[entity.Value] = true
				}
			}
			summary.AccumulatedContactCount = len(uniqueContacts)

			// Update clarity progression with current maturity score
			if summary.ClarityProgression == "" {
				summary.ClarityProgression = "[]"
			}
			var progression []float64
			if err := json.Unmarshal([]byte(summary.ClarityProgression), &progression); err != nil {
				log.Printf("[MessageProcessor] Warning: Failed to unmarshal clarity progression: %v", err)
				progression = []float64{} // Default to empty
			}
			progression = append(progression, layerCtx.Layer3.MaturityScore)
			if progJSON, err := json.Marshal(progression); err == nil {
				summary.ClarityProgression = string(progJSON)
			}

			// Update accumulated values
			valuesMap := make(map[string]bool)
			if summary.AccumulatedValues != "" {
				var existingValues []string
				if err := json.Unmarshal([]byte(summary.AccumulatedValues), &existingValues); err != nil {
					log.Printf("[MessageProcessor] Warning: Failed to unmarshal accumulated values: %v", err)
					existingValues = []string{} // Default to empty
				}
				for _, v := range existingValues {
					valuesMap[v] = true
				}
			}
			for _, v := range extraction.Values {
				valuesMap[v] = true
			}
			var allValues []string
			for v := range valuesMap {
				allValues = append(allValues, v)
			}
			if valuesJSON, err := json.Marshal(allValues); err == nil {
				summary.AccumulatedValues = string(valuesJSON)
			}

			// Update accumulated characteristics
			characteristicsMap := make(map[string]bool)
			if summary.AccumulatedCharacteristics != "" {
				var existingChars []string
				if err := json.Unmarshal([]byte(summary.AccumulatedCharacteristics), &existingChars); err != nil {
					log.Printf("[MessageProcessor] Warning: Failed to unmarshal accumulated characteristics: %v", err)
					existingChars = []string{} // Default to empty
				}
				for _, c := range existingChars {
					characteristicsMap[c] = true
				}
			}
			for _, entity := range analysisCtx.ExtractedEntities {
				if entity.Type == "characteristic" && entity.Value != "" {
					characteristicsMap[entity.Value] = true
				}
			}
			var allCharacteristics []string
			for c := range characteristicsMap {
				allCharacteristics = append(allCharacteristics, c)
			}
			if charsJSON, err := json.Marshal(allCharacteristics); err == nil {
				summary.AccumulatedCharacteristics = string(charsJSON)
			}

			// Update conflicts resolved
			if layerCtx.Layer5 != nil {
				summary.ConflictsResolved = len(layerCtx.Layer5.CriticalConflicts)
			}

			// Save updated summary
			summaryRepo := database.NewConversationSummaryRepository(srv.database.GetConnection())
			if saveErr := summaryRepo.UpdateSummaryWithRetry(userID, conversationID, func(latest *models.ConversationSummary) {
				latest.AccumulatedEntityCount = summary.AccumulatedEntityCount
				latest.AccumulatedContactCount = summary.AccumulatedContactCount
				latest.AccumulatedValues = summary.AccumulatedValues
				latest.AccumulatedCharacteristics = summary.AccumulatedCharacteristics
				latest.ClarityProgression = summary.ClarityProgression
				latest.ConflictsResolved = summary.ConflictsResolved
			}); saveErr != nil {
				log.Printf("[MessageProcessor] ⚠️ FIX #6: Failed to update summary: %v", saveErr)
			} else {
				log.Printf("[MessageProcessor] ✓ FIX #6: Updated summary - entities=%d contacts=%d maturity=%.2f progression=%d",
					summary.AccumulatedEntityCount, summary.AccumulatedContactCount, layerCtx.Layer3.MaturityScore, len(progression))
			}
		}
	}

	// Fix E: Clean up message processing state AFTER all stages complete
	// This ensures atomicity - if cleanup fails, state remains for retry
	// Must be done after all database writes (learning, maturity)
	if msgProcState != nil {
		cleanupErr := srv.messageProcessingState.DeleteState(userID, req.ConversationID, userMessageID)
		if cleanupErr != nil {
			log.Printf("[MessageProcessor] Warning: Failed to clean up message processing state: %v", cleanupErr)
		} else {
			log.Printf("[MessageProcessor] ✓ Cleaned up message processing state for message %s", userMessageID)
		}
	}

	// PHASE 1 (reply contract): the reply is resolved once here. The text sent to the
	// user and the text saved to chat_messages are the same value.
	// A reply asks at most one thing: when the agent's reply already asks a question, the contact formatter's
	// clarification question is not added on top of it.
	if _, hasClar := response["clarification"]; hasClar && agents.AskedQuestion(agentResp.Metadata) {
		delete(response, "clarification")
		response["action_required"] = false
		log.Printf("[MessageProcessor] Contact clarification not added: the reply already asks a question")
	}
	replyText, replyErr := resolveReplyText(response, agentResp.Response)
	if replyErr != nil {
		log.Printf("[MessageProcessor] ERROR: %v (nothing saved, no reply sent)", replyErr)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Moly could not produce a reply. Please send the message again.",
		})
		return
	}
	response["response"] = replyText
	// The decision record (metadata) is returned in every branch, so a test or a log reader can see the reply kind.
	if _, ok := response["metadata"]; !ok {
		response["metadata"] = agentResp.Metadata
	}

	savedPhase := agentResp.Phase
	if p, ok := response["phase"].(string); ok && p != "" {
		savedPhase = p
	}
	if agentResp.Metadata != nil && savedPhase != "" {
		agentResp.Metadata["phase"] = savedPhase
	}

	// SAVE AGENT RESPONSE to chat_messages for conversation history (with metadata)
	agentResponseID := fmt.Sprintf("msg_%d_%d", now, rand.Int63())
	agentResponseContent := replyText

	// Serialize metadata for storage
	metadataJSON := "{}"
	if agentResp.Metadata != nil {
		if b, err := json.Marshal(agentResp.Metadata); err == nil {
			metadataJSON = string(b)
		}
	}

	if _, execErr := turn.Exec(`
		INSERT INTO chat_messages (id, user_id, conversation_id, role, content, context_extracted, contact_mention, metadata, created_at)
		VALUES (?, ?, ?, 'assistant', ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO NOTHING
	`, agentResponseID, userID, conversationID, agentResponseContent, "{}", "", metadataJSON, now); execErr != nil {
		log.Printf("[MessageProcessor] WARNING: Failed to save agent response to chat_messages: %v", execErr)
	} else {
		log.Printf("[MessageProcessor] ✓ Saved agent response to chat_messages with metadata: %s", agentResponseID)
	}

	// COMMIT STAGE (step 5): everything this turn saves is written now, in one transaction. If it fails, the reply is
	// not returned and nothing from this turn is kept.
	if srv.beforeCommit != nil {
		if faultErr := srv.beforeCommit(); faultErr != nil {
			log.Printf("[MessageProcessor] ERROR: commit failed: %v (no reply sent)", faultErr)
			respondJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "Moly could not save this turn. Please send the message again.",
			})
			return
		}
	}
	if commitErr := turn.Run(conn); commitErr != nil {
		log.Printf("[MessageProcessor] ERROR: %v (no reply sent)", commitErr)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Moly could not save this turn. Please send the message again.",
		})
		return
	}
	log.Printf("[MessageProcessor] ✓ Turn committed (%d writes)", turn.Len())

	respondJSON(w, http.StatusOK, response)
}

// ClarificationResponseHandler - Handle user responses to clarification questions
func (srv *APIServer) ClarificationResponseHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	// Extract and validate Bearer token
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": authErr.Error()})
		return
	}

	var req agents.ClarificationResponseRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	if req.FactID == "" || req.QuestionID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing factId or questionId"})
		return
	}

	log.Printf("[Clarification] Response from user %s to question %s\n", userID, req.QuestionID)

	// Mark question as answered in database to prevent duplicate questions
	conn := srv.database.GetConnection()
	now := time.Now().Unix()
	if _, err := conn.Exec(
		"UPDATE clarification_questions SET status = 'answered', answered_at = ? WHERE id = ? AND user_id = ?",
		now, req.QuestionID, userID,
	); err != nil {
		log.Printf("[Clarification] Warning: Failed to mark question as answered: %v", err)
	}

	// Create TemporaryFactStore for this user
	tempStore := agents.NewTemporaryFactStore(srv.database, userID)

	// Extract linked facts (from temporary store)
	linkedFacts := []agents.ExtractedFact{}
	if req.FactID != "" {
		if tempFact, err := tempStore.Get(req.FactID); err == nil && tempFact != nil {
			// Convert TemporaryFact to ExtractedFact
			linkedFacts = append(linkedFacts, agents.ExtractedFact{
				ID:              tempFact.FactID,
				Value:           tempFact.FactValue,
				FactType:        tempFact.FactType,
				ProposedSubject: tempFact.AttributedTo,
				Confidence:      tempFact.Confidence,
				Evidence:        tempFact.Evidence,
			})
		}
	}

	// Get the conversation ID from the clarification question
	questionRepo := database.NewClarificationQuestionRepository(srv.database)
	question, err := questionRepo.GetQuestion(userID, req.QuestionID)
	conversationID := ""
	if err != nil {
		log.Printf("[Clarification] Warning: Failed to retrieve question: %v", err)
	} else if question != nil {
		conversationID = question.ConversationID
	}

	// Process response with new AnswerProcessor
	processedResult, err := srv.answerProcessor.ProcessResponse(
		req.QuestionID,
		req.UserResponse,
		linkedFacts,
		userID,
		conversationID,
	)
	if err != nil {
		log.Printf("[Clarification] Error processing response: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": fmt.Sprintf("Failed to process response: %v", err),
		})
		return
	}

	// Save extracted context to About Me profile
	if processedResult.ContextToSave != nil {
		conn := srv.database.GetConnection()

		// Extract all available context data
		contextStr := ""
		if contextVal, ok := processedResult.ContextToSave["context"].(string); ok {
			contextStr = contextVal
		}

		patternsJSON := "[]"
		if patterns, ok := processedResult.ContextToSave["patterns"].([]interface{}); ok {
			if b, err := json.Marshal(patterns); err == nil {
				patternsJSON = string(b)
			}
		}

		// Update or create About Me profile - save communication_style AND patterns
		// ARCHITECTURAL FIX: Only save communication_style and patterns from clarification
		// Don't auto-populate tone_preference (that's user's configuration, not observed data)
		now := time.Now().Unix()
		_, err := conn.Exec(`
			INSERT INTO about_me (user_id, communication_style, patterns, updated_at, created_at)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(user_id) DO UPDATE SET
			  communication_style = CASE
				WHEN communication_style IS NULL OR communication_style = '' THEN excluded.communication_style
				ELSE communication_style
			  END,
			  patterns = CASE
				WHEN patterns IS NULL OR patterns = '[]' THEN excluded.patterns
				ELSE patterns
			  END,
			  updated_at = excluded.updated_at
		`, userID, contextStr, patternsJSON, now, now)

		if err != nil {
			log.Printf("[Clarification] Warning: Failed to save About Me: %v", err)
		} else {
			log.Printf("[Clarification] ✓ Saved About Me: style='%s', patterns=%d items", contextStr, len(patternsJSON))
		}
	}

	// FIXED: Don't return early - continue through orchestrator
	// The clarification response is a message and should be evaluated through full pipeline
	log.Printf("[Clarification] ✓ Answer processed, now routing through orchestrator for consistency")

	// Create a Phase5Request from the clarification response to process through orchestrator
	// This ensures the response content gets safety/ethics evaluation
	orchestratorReq := &schema.Phase5Request{
		Message:          req.UserResponse,
		ConversationID:   conversationID,
		BrowserSessionId: "", // Will use conversation's existing session
	}

	log.Printf("[Clarification] ✓ Routing clarification response through orchestrator: %s", conversationID)

	// Build a fake request for the orchestrator
	bodyBytes, _ := json.Marshal(orchestratorReq)
	fakeReq := &http.Request{
		Method:     "POST",
		Body:       io.NopCloser(bytes.NewReader(bodyBytes)),
		Header:     r.Header,
		RemoteAddr: r.RemoteAddr,
	}

	// Add auth token to header if present
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		fakeReq.Header.Set("Authorization", authHeader)
	}

	log.Printf("[Clarification] FIXED: Passing through orchestrator instead of early return")
	// Continue through orchestrator by calling the handler
	srv.MessageProcessorHandler(w, fakeReq)
	// After orchestrator processes, the response is already sent via w
}

// AnalyzeIncomingMessageHandler - Analyze incoming message and generate suggestions
func (srv *APIServer) AnalyzeIncomingMessageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	// Extract and validate Bearer token
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": authErr.Error()})
		return
	}

	var req struct {
		IncomingMessage string `json:"incomingMessage"`
		ConversationID  string `json:"conversationId"`
	}

	var err error
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	if req.IncomingMessage == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing incomingMessage"})
		return
	}

	log.Printf("[IncomingMessage] Analyzing message from user %s\n", userID)

	// Detect sender
	sender, _ := srv.incomingMessageAnalyzer.DetectSender(req.IncomingMessage)

	// Get user's About Me
	conn := srv.database.GetConnection()
	var commStyle, coreVals, prefTone, prefs, goals, patterns string
	err = conn.QueryRow(
		`SELECT COALESCE(communication_style,''), COALESCE(core_values,''), COALESCE(tone_preference,''),
		        COALESCE(preferences,''), COALESCE(goals,''), COALESCE(patterns,'')
		 FROM about_me WHERE user_id = ?`,
		userID,
	).Scan(&commStyle, &coreVals, &prefTone, &prefs, &goals, &patterns)

	var coreValsArr, goalsArr, patternsArr []string
	if coreVals != "" {
		if err := json.Unmarshal([]byte(coreVals), &coreValsArr); err != nil {
			log.Printf("[AboutMe] Warning: Failed to parse core_values JSON: %v", err)
		}
	}
	if goals != "" {
		if err := json.Unmarshal([]byte(goals), &goalsArr); err != nil {
			log.Printf("[AboutMe] Warning: Failed to parse goals JSON: %v", err)
		}
	}
	if patterns != "" {
		if err := json.Unmarshal([]byte(patterns), &patternsArr); err != nil {
			log.Printf("[AboutMe] Warning: Failed to parse patterns JSON: %v", err)
		}
	}

	userAboutMe := map[string]interface{}{
		"communicationStyle": commStyle,
		"coreValues":         coreValsArr,
		"tonePreference":     prefTone,
		"preferences":        prefs,
		"goals":              goalsArr,
		"patterns":           patternsArr,
	}

	// Get sender contact context if exists
	senderContext := make(map[string]interface{})
	if sender != "" {
		var relationship, context string
		err = conn.QueryRow(
			`SELECT COALESCE(relationship,''), COALESCE(notes,'')
			 FROM contacts WHERE user_id = ? AND name = ?`,
			userID, sender,
		).Scan(&relationship, &context)
		if err == nil {
			senderContext["relationship"] = relationship
			senderContext["context"] = context
		}
	}

	// Get conversation history (last 5 messages for context)
	// Must be chronological order (ASC) because GenerateSuggestions expects messages in time order
	var conversationHistory []string
	if req.ConversationID != "" {
		rows, err := conn.Query(
			`SELECT content FROM chat_messages
			 WHERE user_id = ? AND conversation_id = ? AND role IN ('user', 'assistant')
			 ORDER BY created_at ASC LIMIT 5`,
			userID, req.ConversationID,
		)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var content string
				if err := rows.Scan(&content); err == nil && content != "" {
					content = database.StoredReplyText(content)
					conversationHistory = append(conversationHistory, content)
				}
			}
			if err := rows.Err(); err != nil {
				log.Printf("[SuggestionHandler] Error iterating conversation history: %v", err)
			}
		}
	}

	// Generate suggestions
	suggestions, err := srv.incomingMessageAnalyzer.GenerateSuggestions(
		req.IncomingMessage,
		conversationHistory,
		userAboutMe,
		senderContext,
	)
	if err != nil {
		log.Printf("[IncomingMessage] Error generating suggestions: %v", err)
		suggestions = srv.incomingMessageAnalyzer.GenerateFallbackSuggestions(req.IncomingMessage)
	}

	response := map[string]interface{}{
		"sender":      sender,
		"suggestions": suggestions,
	}

	log.Printf("[IncomingMessage] ✓ Generated %d suggestions", len(suggestions))

	// NOTE: Incoming message analysis is displayed as suggestions. Recording for learning system would be
	// optional future enhancement - currently no conversation_id in this handler

	respondJSON(w, http.StatusOK, response)
}

// SuggestionChoiceHandler - Record when user picks a suggestion
func (srv *APIServer) SuggestionChoiceHandler(w http.ResponseWriter, r *http.Request) {
	// Only POST allowed
	if r.Method != http.MethodPost {
		schema.RespondError(w, http.StatusMethodNotAllowed, "Only POST method allowed")
		return
	}

	// Extract and validate Bearer token
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		log.Printf("[SuggestionChoice] Unauthorized: %v", authErr)
		schema.RespondError(w, http.StatusUnauthorized, authErr.Error())
		return
	}

	// Parse request body
	var req struct {
		ConversationID  string `json:"conversationId"`
		SuggestionIndex int    `json:"suggestionIndex"`
		ModifiedText    string `json:"modifiedText,omitempty"`
		Modification    string `json:"modification,omitempty"`
		UserFeedback    string `json:"userFeedback,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		schema.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Initialize LearningAgent and record suggestion choice
	learningAgent := srv.GetLearningAgent(userID) // Use cached learning agent
	if learningAgent == nil {
		log.Printf("[SuggestionChoice] Error: Unable to get learning agent")
		schema.RespondError(w, http.StatusInternalServerError, "Failed to initialize learning agent")
		return
	}

	choiceData := models.SuggestionChoiceData{
		UserID:          userID,
		ConversationID:  req.ConversationID,
		SuggestionIndex: req.SuggestionIndex,
		ModifiedText:    req.ModifiedText,
		Modification:    req.Modification,
		UserFeedback:    req.UserFeedback,
		CreatedAt:       time.Now().Unix(),
	}

	if err := learningAgent.RecordSuggestionChoice(choiceData); err != nil {
		log.Printf("[SuggestionChoice] Error recording choice: %v", err)
		schema.RespondError(w, http.StatusInternalServerError, "Failed to record suggestion choice")
		return
	}

	log.Printf("[SuggestionChoice] ✓ Recorded choice for user %s", userID)

	// Record audit event for suggestion choice
	auditRepo := srv.database.GetAuditLogRepository()
	if auditRepo != nil {
		auditErr := auditRepo.RecordAction(userID, "suggestion_choice_recorded", map[string]interface{}{
			"conversationId":  req.ConversationID,
			"suggestionIndex": req.SuggestionIndex,
			"modified":        req.ModifiedText != "",
			"userFeedback":    req.UserFeedback,
		})
		if auditErr != nil {
			log.Printf("[SuggestionChoice] Warning: Failed to record audit event: %v", auditErr)
		}
	}

	schema.RespondSuccess(w, http.StatusOK, "choice", map[string]interface{}{"message": "Suggestion choice recorded"})
}

// AboutMeHandler - Get or save user's About Me profile
func (srv *APIServer) AboutMeHandler(w http.ResponseWriter, r *http.Request) {
	// Extract and validate Bearer token
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		log.Printf("[AboutMe] Unauthorized access attempt: %v\n", authErr)
		schema.RespondError(w, http.StatusUnauthorized, authErr.Error())
		return
	}

	conn := srv.database.GetConnection()

	if r.Method == http.MethodGet {
		log.Printf("[AboutMe] GET request from user %s\n", userID)

		// Fetch About Me profile
		var communicationStyle, coreValues, tonePreference, preferences, goals, patterns string
		err := conn.QueryRow(
			`SELECT COALESCE(communication_style,''), COALESCE(core_values,''), COALESCE(tone_preference,''), COALESCE(preferences,''), COALESCE(goals,''), COALESCE(patterns,'') FROM about_me WHERE user_id = ?`,
			userID,
		).Scan(&communicationStyle, &coreValues, &tonePreference, &preferences, &goals, &patterns)

		if err != nil && err != sql.ErrNoRows {
			log.Printf("[AboutMe] Error retrieving profile: %v\n", err)
		}

		profile := &schema.AboutMeProfile{
			CommunicationStyle: communicationStyle,
			CoreValues:         parseArrayFromStorage(coreValues),
			TonePreference:     tonePreference,
			Preferences:        preferences,
			Goals:              parseArrayFromStorage(goals),
			Patterns:           parseArrayFromStorage(patterns),
		}

		log.Printf("[AboutMe] Profile retrieved: style=%s, values=%d, tone=%s\n",
			communicationStyle, len(profile.CoreValues), tonePreference)
		schema.RespondSuccess(w, http.StatusOK, "profile", profile)

	} else if r.Method == http.MethodPost {
		log.Printf("[AboutMe] POST request from user %s\n", userID)

		// Save About Me profile
		req := &schema.AboutMeProfile{}

		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			log.Printf("[AboutMe] Invalid request body: %v\n", err)
			schema.RespondError(w, http.StatusBadRequest, "Invalid request")
			return
		}

		log.Printf("[AboutMe] Parsed profile: style=%s, values=%d, tone=%s, prefs=%s\n",
			req.CommunicationStyle, len(req.CoreValues), req.TonePreference, req.Preferences)

		// Validate using schema validator
		if err := schema.ValidateStruct(req); err != nil {
			log.Printf("[AboutMe] Validation error: %v\n", err)
			schema.RespondError(w, http.StatusBadRequest, err.Error())
			return
		}

		now := time.Now().Unix()
		_, err := conn.Exec(
			`INSERT OR REPLACE INTO about_me (user_id, communication_style, core_values, tone_preference, preferences, goals, patterns, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			userID,
			req.CommunicationStyle,
			encodeArrayForStorage(req.CoreValues),
			req.TonePreference,
			req.Preferences,
			encodeArrayForStorage(req.Goals),
			encodeArrayForStorage(req.Patterns),
			now,
			now,
		)

		if err != nil {
			log.Printf("[AboutMe] Error saving profile: %v\n", err)
			schema.RespondError(w, http.StatusInternalServerError, "Failed to save profile")
			return
		}

		log.Printf("[AboutMe] Profile saved successfully for user %s\n", userID)

		// Record audit event for profile update
		auditRepo := srv.database.GetAuditLogRepository()
		if auditRepo != nil {
			auditErr := auditRepo.RecordAction(userID, "profile_updated", map[string]interface{}{
				"communicationStyle": req.CommunicationStyle,
				"coreValues":         req.CoreValues,
				"tonePreference":     req.TonePreference,
				"preferences":        req.Preferences,
				"goals":              req.Goals,
			})
			if auditErr != nil {
				log.Printf("[AboutMe] Warning: Failed to record audit event: %v", auditErr)
			}
		}

		schema.RespondSuccess(w, http.StatusOK, "profile", req)
	}
}

// ContextHandler handles GET /api/context - Returns user's context quality for a conversation
func (srv *APIServer) ContextHandler(w http.ResponseWriter, r *http.Request) {
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		log.Printf("[Context] Unauthorized: %v", authErr)
		schema.RespondError(w, http.StatusUnauthorized, authErr.Error())
		return
	}

	conversationID := r.URL.Query().Get("conversationId")
	if conversationID == "" {
		schema.RespondError(w, http.StatusBadRequest, "conversationId parameter required")
		return
	}

	conn := srv.database.GetConnection()

	// Count messages in conversation (indicator of context completeness)
	var messageCount int
	err := conn.QueryRow(
		`SELECT COUNT(*) FROM interactions WHERE user_id = ? AND conversation_id = ?`,
		userID, conversationID,
	).Scan(&messageCount)

	if err != nil && err != sql.ErrNoRows {
		log.Printf("[Context] Error counting messages: %v", err)
	}

	// Check AboutMe completeness
	var aboutMeFields int
	err = conn.QueryRow(
		`SELECT COUNT(CASE WHEN communication_style IS NOT NULL AND communication_style != '' THEN 1 END) +
		        COUNT(CASE WHEN core_values IS NOT NULL AND core_values != '' THEN 1 END) +
		        COUNT(CASE WHEN tone_preference IS NOT NULL AND tone_preference != '' THEN 1 END) +
		        COUNT(CASE WHEN goals IS NOT NULL AND goals != '' THEN 1 END)
		 FROM about_me WHERE user_id = ?`,
		userID,
	).Scan(&aboutMeFields)

	// Count contacts
	var contactCount int
	err = conn.QueryRow(
		`SELECT COUNT(*) FROM contacts WHERE user_id = ? AND status = 'active'`,
		userID,
	).Scan(&contactCount)

	// Calculate context quality
	completenessScore := 0.0
	gaps := []string{}

	if aboutMeFields < 2 {
		gaps = append(gaps, "Missing AboutMe information")
		completenessScore += 0.3
	} else {
		completenessScore += 0.5
	}

	if contactCount == 0 {
		gaps = append(gaps, "No contacts defined")
		completenessScore += 0.2
	} else if contactCount >= 3 {
		completenessScore += 0.3
	} else {
		completenessScore += 0.2
	}

	if messageCount < 5 {
		gaps = append(gaps, "Limited conversation history")
		completenessScore += 0.2
	} else {
		completenessScore += 0.2
	}

	completenessLevel := "minimal"
	if completenessScore >= 0.7 {
		completenessLevel = "complete"
	} else if completenessScore >= 0.4 {
		completenessLevel = "partial"
	}

	response := map[string]interface{}{
		"conversationId": conversationID,
		"contextQuality": map[string]interface{}{
			"overallScore":      completenessScore,
			"completenessLevel": completenessLevel,
		},
		"missingContextGaps": gaps,
	}

	log.Printf("[Context] Quality for %s: score=%.2f level=%s gaps=%d",
		conversationID, completenessScore, completenessLevel, len(gaps))
	schema.RespondSuccess(w, http.StatusOK, "context", response)
}

// ConversationsHandler - Get or create conversations
func (srv *APIServer) ConversationsHandler(w http.ResponseWriter, r *http.Request) {
	// Extract and validate Bearer token
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		schema.RespondError(w, http.StatusUnauthorized, authErr.Error())
		return
	}

	conn := srv.database.GetConnection()

	if r.Method == http.MethodGet {
		// List user's conversations
		rows, err := conn.Query(
			"SELECT id, name, type, description, purpose, members, settings, notes, created_at, updated_at FROM conversations WHERE user_id = ? ORDER BY updated_at DESC",
			userID,
		)
		if err != nil {
			schema.RespondError(w, http.StatusInternalServerError, "Failed to fetch conversations")
			return
		}
		defer rows.Close()

		conversations := []map[string]interface{}{}
		for rows.Next() {
			var id, name, convType, description string
			var purpose, notes, membersJSON, settingsJSON sql.NullString
			var createdAt, updatedAt int64
			if err := rows.Scan(&id, &name, &convType, &description, &purpose, &membersJSON, &settingsJSON, &notes, &createdAt, &updatedAt); err != nil {
				log.Printf("[ConversationsHandler] ERROR: Failed to scan conversation row: %v", err)
				continue
			}

			conv := map[string]interface{}{
				"id":          id,
				"name":        name,
				"type":        convType,
				"description": description,
				"purpose":     purpose.String, // Empty string if NULL
				"createdAt":   createdAt,
				"updatedAt":   updatedAt,
				"notes":       notes.String, // Empty string if NULL
			}

			// Parse JSON fields
			if membersJSON.Valid && membersJSON.String != "" {
				var members []schema.ConversationMember
				if err := json.Unmarshal([]byte(membersJSON.String), &members); err == nil {
					conv["members"] = members
				}
			}
			if settingsJSON.Valid && settingsJSON.String != "" {
				var settings schema.ConversationSettings
				if err := json.Unmarshal([]byte(settingsJSON.String), &settings); err == nil {
					conv["settings"] = settings
				}
			}

			// Get recent messages preview (last 3 messages)
			msgRows, err := conn.Query(
				"SELECT role, content, created_at FROM chat_messages WHERE conversation_id = ? ORDER BY created_at DESC LIMIT 3",
				id,
			)
			if err == nil {
				defer msgRows.Close()
				messages := []map[string]interface{}{}
				for msgRows.Next() {
					var role, content string
					var msgTime int64
					if err := msgRows.Scan(&role, &content, &msgTime); err == nil {
						content = database.StoredReplyText(content)
						// Truncate content for preview
						preview := content
						if len(preview) > 100 {
							preview = preview[:100] + "..."
						}
						messages = append(messages, map[string]interface{}{
							"role":      role,
							"content":   preview,
							"timestamp": msgTime,
						})
					}
				}
				// Reverse to get chronological order
				for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
					messages[i], messages[j] = messages[j], messages[i]
				}
				conv["recentMessages"] = messages
			}

			conversations = append(conversations, conv)
		}

		if err := rows.Err(); err != nil {
			log.Printf("[ConversationsHandler] Error iterating conversations: %v", err)
			schema.RespondError(w, http.StatusInternalServerError, "Error loading conversations")
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"conversations": conversations,
		})

	} else if r.Method == http.MethodPost {
		// Create new conversation
		req := &schema.Conversation{}

		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			schema.RespondError(w, http.StatusBadRequest, "Invalid request")
			return
		}

		// Validate using schema validator
		if err := schema.ValidateStruct(req); err != nil {
			schema.RespondError(w, http.StatusBadRequest, err.Error())
			return
		}

		now := time.Now()
		conversationID := fmt.Sprintf("conv_%d_%d", now.Unix(), now.Nanosecond())
		nowUnix := now.Unix()

		// Encode JSON fields
		membersJSON := ""
		if len(req.Members) > 0 {
			if b, err := json.Marshal(req.Members); err == nil {
				membersJSON = string(b)
			}
		}

		settingsJSON := ""
		if req.Settings.Mode != "" || req.Settings.Context != "" {
			if b, err := json.Marshal(req.Settings); err == nil {
				settingsJSON = string(b)
			}
		}

		_, err := conn.Exec(
			"INSERT INTO conversations (id, user_id, name, type, description, purpose, members, settings, notes, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			conversationID,
			userID,
			req.Name,
			req.Type,
			req.Description,
			req.Purpose,
			membersJSON,
			settingsJSON,
			req.Notes,
			nowUnix,
			nowUnix,
		)

		if err != nil {
			log.Printf("[Conversations] Error creating conversation: %v\n", err)
			schema.RespondError(w, http.StatusInternalServerError, "Failed to create conversation")
			return
		}

		response := &schema.Conversation{
			ID:          conversationID,
			Name:        req.Name,
			Type:        req.Type,
			Purpose:     req.Purpose,
			Description: req.Description,
			Members:     req.Members,
			Settings:    req.Settings,
			Notes:       req.Notes,
			CreatedAt:   nowUnix,
			UpdatedAt:   nowUnix,
		}

		schema.RespondSuccess(w, http.StatusOK, "conversation", response)
	} else if r.Method == http.MethodDelete {
		// Delete a conversation by ID from path (using Go 1.22+ path parameters)
		conversationID := r.PathValue("conversationID")

		if conversationID == "" {
			schema.RespondError(w, http.StatusBadRequest, "Conversation ID required")
			return
		}

		// Verify the conversation belongs to this user
		var checkUserID string
		err := conn.QueryRow(
			"SELECT user_id FROM conversations WHERE id = ?",
			conversationID,
		).Scan(&checkUserID)

		if err == sql.ErrNoRows {
			schema.RespondError(w, http.StatusNotFound, "Conversation not found")
			return
		}
		if err != nil {
			schema.RespondError(w, http.StatusInternalServerError, "Failed to verify conversation")
			return
		}

		if checkUserID != userID {
			schema.RespondError(w, http.StatusForbidden, "Not authorized to delete this conversation")
			return
		}

		// Delete the conversation with its messages and related rows, in one transaction
		err = database.DeleteConversationData(conn, userID, conversationID)
		if err != nil {
			log.Printf("[Conversations] Error deleting conversation: %v\n", err)
			schema.RespondError(w, http.StatusInternalServerError, "Failed to delete conversation")
			return
		}

		log.Printf("[Conversations] Deleted conversation %s for user %s\n", conversationID, userID)
		schema.RespondSuccess(w, http.StatusOK, "result", map[string]interface{}{
			"id":     conversationID,
			"status": "deleted",
		})
	} else {
		schema.RespondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// ContactsHandler - Get or create user contacts
func (srv *APIServer) ContactsHandler(w http.ResponseWriter, r *http.Request) {
	// Extract and validate Bearer token
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		schema.RespondError(w, http.StatusUnauthorized, authErr.Error())
		return
	}

	conn := srv.database.GetConnection()

	if r.Method == http.MethodGet {
		// List user's contacts
		rows, err := conn.Query(
			"SELECT id, name, relationship, notes, created_at FROM contacts WHERE user_id = ? ORDER BY updated_at DESC",
			userID,
		)
		if err != nil {
			schema.RespondError(w, http.StatusInternalServerError, "Failed to fetch contacts")
			return
		}
		defer rows.Close()

		contacts := []map[string]interface{}{}
		for rows.Next() {
			var id, name, relationship string
			var notes sql.NullString // Handle NULL values properly
			var createdAt int64
			if err := rows.Scan(&id, &name, &relationship, &notes, &createdAt); err != nil {
				log.Printf("[ContactsHandler] ERROR: Failed to scan contact row: %v", err)
				continue
			}
			contacts = append(contacts, map[string]interface{}{
				"id":           id,
				"name":         name,
				"relationship": relationship,
				"notes":        notes.String, // Empty string if NULL
				"createdAt":    createdAt,
			})
		}

		if err := rows.Err(); err != nil {
			log.Printf("[ContactsHandler] Error iterating contacts: %v", err)
			schema.RespondError(w, http.StatusInternalServerError, "Error loading contacts")
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"contacts": contacts,
		})

	} else if r.Method == http.MethodPost {
		// Create new contact
		req := &schema.Contact{}

		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			schema.RespondError(w, http.StatusBadRequest, "Invalid request")
			return
		}

		// Validate using schema validator
		if err := schema.ValidateStruct(req); err != nil {
			schema.RespondError(w, http.StatusBadRequest, err.Error())
			return
		}

		now := time.Now()
		contactID := fmt.Sprintf("contact_%d_%d", now.Unix(), now.Nanosecond())
		nowUnix := now.Unix()

		_, err := conn.Exec(
			"INSERT INTO contacts (user_id, name, relationship, notes, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
			userID,
			req.Name,
			req.Relationship,
			req.Notes,
			nowUnix,
			nowUnix,
		)

		if err != nil {
			log.Printf("[Contacts] Error creating contact: %v\n", err)
			schema.RespondError(w, http.StatusInternalServerError, "Failed to create contact")
			return
		}

		response := &schema.Contact{
			ID:           contactID,
			Name:         req.Name,
			Relationship: req.Relationship,
			Notes:        req.Notes,
			CreatedAt:    nowUnix,
		}

		schema.RespondSuccess(w, http.StatusOK, "contact", response)
	}
}

// ContactDetailHandler - Get, update, or delete individual contacts
func (srv *APIServer) ContactDetailHandler(w http.ResponseWriter, r *http.Request) {
	// Extract and validate Bearer token
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		schema.RespondError(w, http.StatusUnauthorized, authErr.Error())
		return
	}

	conn := srv.database.GetConnection()
	contactID := r.PathValue("contactID")

	if contactID == "" {
		schema.RespondError(w, http.StatusBadRequest, "contactID is required")
		return
	}

	// Verify contact belongs to user
	var contactUserID string
	err := conn.QueryRow("SELECT user_id FROM contacts WHERE id = ?", contactID).Scan(&contactUserID)
	if err != nil {
		schema.RespondError(w, http.StatusNotFound, "Contact not found")
		return
	}
	if contactUserID != userID {
		schema.RespondError(w, http.StatusForbidden, "Access denied to this contact")
		return
	}

	if r.Method == http.MethodGet {
		// Get single contact
		var id int64
		var name, relationship string
		var characteristics, notes sql.NullString
		var createdAt, updatedAt int64
		err := conn.QueryRow(
			"SELECT id, name, relationship, characteristics, notes, created_at, updated_at FROM contacts WHERE id = ? AND user_id = ?",
			contactID, userID,
		).Scan(&id, &name, &relationship, &characteristics, &notes, &createdAt, &updatedAt)

		if err != nil {
			schema.RespondError(w, http.StatusNotFound, "Contact not found")
			return
		}

		var traits []string
		if characteristics.Valid && characteristics.String != "" {
			json.Unmarshal([]byte(characteristics.String), &traits)
		}

		contact := map[string]interface{}{
			"id":              id,
			"name":            name,
			"relationship":    relationship,
			"characteristics": traits,
			"notes":           notes.String,
			"createdAt":       createdAt,
			"updatedAt":       updatedAt,
		}

		schema.RespondSuccess(w, http.StatusOK, "contact", contact)

	} else if r.Method == http.MethodPut {
		// Update contact
		req := &schema.Contact{}
		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			schema.RespondError(w, http.StatusBadRequest, "Invalid request")
			return
		}

		if err := schema.ValidateStruct(req); err != nil {
			schema.RespondError(w, http.StatusBadRequest, err.Error())
			return
		}

		now := time.Now().Unix()

		_, err := conn.Exec(
			"UPDATE contacts SET name = ?, relationship = ?, notes = ?, updated_at = ? WHERE id = ? AND user_id = ?",
			req.Name, req.Relationship, req.Notes, now, contactID, userID,
		)

		if err != nil {
			schema.RespondError(w, http.StatusInternalServerError, "Failed to update contact")
			return
		}

		response := map[string]interface{}{
			"id":           contactID,
			"name":         req.Name,
			"relationship": req.Relationship,
			"notes":        req.Notes,
			"updatedAt":    now,
		}

		schema.RespondSuccess(w, http.StatusOK, "contact", response)

	} else if r.Method == http.MethodDelete {
		// Delete contact
		_, err := conn.Exec("DELETE FROM contacts WHERE id = ? AND user_id = ?", contactID, userID)
		if err != nil {
			schema.RespondError(w, http.StatusInternalServerError, "Failed to delete contact")
			return
		}

		schema.RespondSuccess(w, http.StatusOK, "message", "Contact deleted successfully")
	} else {
		schema.RespondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// MessagesHandler - Get messages for a conversation
func (srv *APIServer) MessagesHandler(w http.ResponseWriter, r *http.Request) {
	// Extract and validate Bearer token
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		schema.RespondError(w, http.StatusUnauthorized, authErr.Error())
		return
	}

	conn := srv.database.GetConnection()

	if r.Method != http.MethodGet {
		schema.RespondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Get conversationId from query parameter
	conversationID := r.URL.Query().Get("conversationId")
	if conversationID == "" {
		schema.RespondError(w, http.StatusBadRequest, "conversationId parameter required")
		return
	}

	// Verify conversation exists and belongs to user
	var convUserID string
	err := conn.QueryRow(
		"SELECT user_id FROM conversations WHERE id = ?",
		conversationID,
	).Scan(&convUserID)

	if err != nil {
		schema.RespondError(w, http.StatusNotFound, "Conversation not found")
		return
	}

	if convUserID != userID {
		schema.RespondError(w, http.StatusForbidden, "Access denied to this conversation")
		return
	}

	// Fetch messages from chat_messages table
	rows, err := conn.Query(
		"SELECT id, user_id, conversation_id, role, content, context_extracted, contact_mention, created_at FROM chat_messages WHERE conversation_id = ? ORDER BY created_at ASC",
		conversationID,
	)
	if err != nil {
		log.Printf("[Messages] Error fetching messages: %v\n", err)
		schema.RespondError(w, http.StatusInternalServerError, "Failed to fetch messages")
		return
	}
	defer rows.Close()

	messages := []map[string]interface{}{}
	for rows.Next() {
		var id, role, content string
		var contextExtracted, contactMention sql.NullString
		var uID, convID string
		var createdAt int64

		if err := rows.Scan(&id, &uID, &convID, &role, &content, &contextExtracted, &contactMention, &createdAt); err != nil {
			log.Printf("[MessagesHandler] WARNING: Skipping corrupted message row - scan error: %v", err)
			continue
		}

		msgMap := map[string]interface{}{
			"id":             id,
			"conversationId": convID,
			"role":           role,
			"content":        database.StoredReplyText(content),
			"createdAt":      createdAt,
		}

		if contextExtracted.Valid {
			var extracted map[string]interface{}
			if err := json.Unmarshal([]byte(contextExtracted.String), &extracted); err == nil {
				msgMap["contextExtracted"] = extracted
			} else {
				msgMap["contextExtracted"] = contextExtracted.String
			}
		}
		if contactMention.Valid {
			var contact map[string]interface{}
			if err := json.Unmarshal([]byte(contactMention.String), &contact); err == nil {
				msgMap["contactMention"] = contact
			} else {
				msgMap["contactMention"] = contactMention.String
			}
		}

		messages = append(messages, msgMap)
	}

	if err := rows.Err(); err != nil {
		log.Printf("[MessagesHandler] Error iterating messages: %v", err)
		schema.RespondError(w, http.StatusInternalServerError, "Error loading messages")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"messages": messages,
	})
}

// ReflectionsHandler - Get, approve, or reject reflections
func (srv *APIServer) ReflectionsHandler(w http.ResponseWriter, r *http.Request) {
	// Extract and validate Bearer token
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		log.Printf("[Reflections] Unauthorized access attempt: %v\n", authErr)
		schema.RespondError(w, http.StatusUnauthorized, authErr.Error())
		return
	}

	if r.Method == http.MethodGet {
		log.Printf("[Reflections] GET request from user %s\n", userID)

		// Get pending reflections for user
		repo := srv.database.GetReflectionRepository()
		if repo == nil {
			log.Printf("[Reflections] ERROR: ReflectionRepository is nil\n")
			schema.RespondError(w, http.StatusInternalServerError, "Reflection service unavailable")
			return
		}
		reflections, err := repo.GetPendingApprovals(userID, "pending_approval")
		if err != nil {
			log.Printf("[Reflections] Error retrieving reflections: %v\n", err)
			schema.RespondError(w, http.StatusInternalServerError, "Failed to retrieve reflections")
			return
		}

		// Always return empty array, never nil (frontend expects array)
		if reflections == nil {
			reflections = []models.Reflection{}
		}

		log.Printf("[Reflections] Found %d pending reflections for user %s\n", len(reflections), userID)
		schema.RespondSuccess(w, http.StatusOK, "reflections", reflections)

	} else if r.Method == http.MethodPost {
		log.Printf("[Reflections] POST request from user %s\n", userID)

		var req struct {
			ID     int    `json:"id"`
			Action string `json:"action"` // "approve" or "reject"
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("[Reflections] Invalid request body: %v\n", err)
			schema.RespondError(w, http.StatusBadRequest, "Invalid request")
			return
		}

		if req.ID == 0 || (req.Action != "approve" && req.Action != "reject") {
			log.Printf("[Reflections] Invalid ID or action\n")
			schema.RespondError(w, http.StatusBadRequest, "id and action (approve/reject) required")
			return
		}

		repo := srv.database.GetReflectionRepository()
		if repo == nil {
			log.Printf("[Reflections] ERROR: ReflectionRepository is nil\n")
			schema.RespondError(w, http.StatusInternalServerError, "Reflection service unavailable")
			return
		}
		var err error
		if req.Action == "approve" {
			err = repo.Approve(userID, req.ID)
			log.Printf("[Reflections] Approved reflection %d\n", req.ID)
		} else {
			err = repo.Reject(userID, req.ID)
			log.Printf("[Reflections] Rejected reflection %d\n", req.ID)
		}

		if err != nil {
			log.Printf("[Reflections] Error updating reflection: %v\n", err)
			schema.RespondError(w, http.StatusInternalServerError, "Failed to update reflection")
			return
		}

		schema.RespondSuccess(w, http.StatusOK, "result", map[string]string{"status": "ok"})
	} else {
		schema.RespondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// MetricsHandler - Get learning analytics and question effectiveness metrics
// ConflictsHandler handles both GET (list conflicts) - uses new pending_input system
func (srv *APIServer) ConflictsHandler(w http.ResponseWriter, r *http.Request) {
	// Extract and validate Bearer token
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		log.Printf("[Conflicts] Unauthorized access attempt: %v\n", authErr)
		schema.RespondError(w, http.StatusUnauthorized, authErr.Error())
		return
	}

	if r.Method == http.MethodGet {
		log.Printf("[Conflicts] GET request from user %s\n", userID)

		// Get unresolved conflicts from new pending_input system
		pendingRepo := srv.database.GetPendingInputRepository()
		if pendingRepo == nil {
			log.Printf("[Conflicts] ERROR: PendingInputRepository is nil\n")
			schema.RespondError(w, http.StatusInternalServerError, "Conflict service unavailable")
			return
		}

		pending, err := pendingRepo.GetByType(userID, "conflict")
		if err != nil {
			log.Printf("[Conflicts] Error retrieving conflicts: %v\n", err)
			schema.RespondError(w, http.StatusInternalServerError, "Failed to retrieve conflicts")
			return
		}

		// Convert pending_input conflicts to displayable format
		type ConflictResponse struct {
			ID             int64  `json:"id"`
			Type           string `json:"type"`
			Subtype        string `json:"subtype"`
			Question       string `json:"question"`
			StoredValue    string `json:"storedValue"`
			ExtractedValue string `json:"extractedValue"`
			CreatedAt      int64  `json:"createdAt"`
		}

		var conflicts []ConflictResponse
		for _, p := range pending {
			// Parse context JSON to get stored/extracted values
			var ctxData map[string]interface{}
			json.Unmarshal(p.Context, &ctxData)

			conflicts = append(conflicts, ConflictResponse{
				ID:             p.ID,
				Type:           p.Type,
				Subtype:        p.Subtype,
				Question:       p.Question,
				StoredValue:    fmt.Sprintf("%v", ctxData["old_value"]),
				ExtractedValue: fmt.Sprintf("%v", ctxData["new_value"]),
				CreatedAt:      p.CreatedAt,
			})
		}

		// Always return empty array, never nil
		if conflicts == nil {
			conflicts = []ConflictResponse{}
		}

		log.Printf("[Conflicts] Found %d unresolved conflicts for user %s\n", len(conflicts), userID)
		schema.RespondSuccess(w, http.StatusOK, "conflicts", conflicts)

	} else {
		schema.RespondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// ConflictResolveHandler handles POST /api/conflicts/resolve
func (srv *APIServer) ConflictResolveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		schema.RespondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract and validate Bearer token
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		log.Printf("[ConflictResolve] Unauthorized access attempt: %v\n", authErr)
		schema.RespondError(w, http.StatusUnauthorized, authErr.Error())
		return
	}

	log.Printf("[ConflictResolve] POST request from user %s\n", userID)

	// Parse request
	type ResolveRequest struct {
		ConflictId int64  `json:"conflictId"`
		Resolution string `json:"resolution"` // keep_saved | use_extracted | merge
	}

	req := &ResolveRequest{}
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		log.Printf("[ConflictResolve] Invalid request body: %v\n", err)
		schema.RespondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Validate resolution choice
	if req.Resolution != "keep_saved" && req.Resolution != "use_extracted" && req.Resolution != "merge" {
		log.Printf("[ConflictResolve] Invalid resolution: %s\n", req.Resolution)
		schema.RespondError(w, http.StatusBadRequest, "Invalid resolution choice")
		return
	}

	log.Printf("[ConflictResolve] Resolving conflict %d with resolution: %s\n", req.ConflictId, req.Resolution)

	// Load the conflict
	conflictRepo := srv.database.GetContextConflictRepository()
	if conflictRepo == nil {
		log.Printf("[ConflictResolve] ERROR: ContextConflictRepository is nil\n")
		schema.RespondError(w, http.StatusInternalServerError, "Conflict service unavailable")
		return
	}
	conflicts, err := conflictRepo.GetUnresolved(userID)
	if err != nil {
		log.Printf("[ConflictResolve] Error loading conflicts: %v\n", err)
		schema.RespondError(w, http.StatusInternalServerError, "Failed to load conflicts")
		return
	}

	// Find the specific conflict
	var targetConflict *database.ContextConflict
	for _, c := range conflicts {
		if c.ID == req.ConflictId {
			targetConflict = c
			break
		}
	}

	if targetConflict == nil {
		log.Printf("[ConflictResolve] Conflict %d not found or already resolved\n", req.ConflictId)
		schema.RespondError(w, http.StatusNotFound, "Conflict not found or already resolved")
		return
	}

	// Verify conflict belongs to this user
	if targetConflict.UserID != userID {
		log.Printf("[ConflictResolve] Conflict %d does not belong to user %s\n", req.ConflictId, userID)
		schema.RespondError(w, http.StatusForbidden, "Access denied to this conflict")
		return
	}

	// Apply the resolution
	handler := tools.NewConflictResolutionHandler(srv.database)
	result := handler.ApplyResolution(targetConflict, req.Resolution)

	if !result.Success {
		log.Printf("[ConflictResolve] Failed to apply resolution: %s\n", result.Message)
		schema.RespondError(w, http.StatusInternalServerError, result.Message)
		return
	}

	log.Printf("[ConflictResolve] ✓ Conflict resolved: %s\n", result.Message)

	// Record audit event for conflict resolution
	auditRepo := srv.database.GetAuditLogRepository()
	if auditRepo != nil {
		auditErr := auditRepo.RecordAction(userID, "conflict_resolved", map[string]interface{}{
			"conflictId":     req.ConflictId,
			"conflictType":   targetConflict.ConflictType,
			"resolution":     req.Resolution,
			"savedValue":     targetConflict.SavedValue,
			"extractedValue": targetConflict.ExtractedValue,
		})
		if auditErr != nil {
			log.Printf("[ConflictResolve] Warning: Failed to record audit event: %v", auditErr)
		}
	}

	// Phase 3: Trigger behavioral profile rebuild (learning from this resolution)
	log.Printf("[ConflictResolve] Triggering behavioral profile rebuild for user %s", userID)
	learningAgent := srv.GetLearningAgent(userID) // Use cached learning agent
	if learningAgent != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[ConflictResolve] PANIC in BuildBehavioralProfile: %v", r)
				}
			}()
			_, analyzeErr := learningAgent.BuildBehavioralProfile(userID)
			if analyzeErr != nil {
				log.Printf("[ConflictResolve] Warning: Could not rebuild behavioral profile: %v", analyzeErr)
			} else {
				log.Printf("[ConflictResolve] ✓ Behavioral profile rebuilt after conflict resolution")
			}
		}()
	}

	schema.RespondSuccess(w, http.StatusOK, "resolution", result)
}

func (srv *APIServer) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	// Extract and validate Bearer token
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		log.Printf("[Metrics] Unauthorized access attempt: %v\n", authErr)
		schema.RespondError(w, http.StatusUnauthorized, authErr.Error())
		return
	}

	if r.Method != http.MethodGet {
		schema.RespondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	log.Printf("[Metrics] GET request from user %s\n", userID)

	metricsRepo := srv.database.GetMetricsRepository()
	if metricsRepo == nil {
		// Graceful degradation if metrics not available
		schema.RespondSuccess(w, http.StatusOK, "metrics", map[string]interface{}{
			"question_effectiveness": nil,
			"principle_violations":   nil,
			"approach_comparison":    nil,
		})
		return
	}

	// Get all metrics
	questionStats, err1 := metricsRepo.GetQuestionEffectivenessStats(userID)
	violationStats, err2 := metricsRepo.GetPrincipleViolationStats(userID)
	principleBreakdown, err3 := metricsRepo.GetPrincipleBreakdown(userID)
	approachComparison, err4 := metricsRepo.GetApproachComparison(userID)

	if err1 != nil {
		log.Printf("[Metrics] Error getting question stats: %v\n", err1)
		questionStats = map[string]interface{}{}
	}
	if err2 != nil {
		log.Printf("[Metrics] Error getting violation stats: %v\n", err2)
		violationStats = map[string]interface{}{}
	}
	if err3 != nil {
		log.Printf("[Metrics] Error getting principle breakdown: %v\n", err3)
		principleBreakdown = []map[string]interface{}{}
	}
	if err4 != nil {
		log.Printf("[Metrics] Error getting approach comparison: %v\n", err4)
		approachComparison = []map[string]interface{}{}
	}

	metrics := map[string]interface{}{
		"question_effectiveness": questionStats,
		"principle_violations":   violationStats,
		"principle_breakdown":    principleBreakdown,
		"approach_comparison":    approachComparison,
		"timestamp":              time.Now().Unix(),
	}

	log.Printf("[Metrics] Returning metrics for user %s\n", userID)
	schema.RespondSuccess(w, http.StatusOK, "metrics", metrics)
}

// GetPreviousQuestionsHandler retrieves previous Socratic questions for a user
func (srv *APIServer) GetPreviousQuestionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		schema.RespondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract and validate Bearer token
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		log.Printf("[PreviousQuestions] Unauthorized: %v\n", authErr)
		schema.RespondError(w, http.StatusUnauthorized, authErr.Error())
		return
	}

	// Parse limit from query parameters
	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	log.Printf("[PreviousQuestions] Retrieving previous questions for user %s (limit: %d)\n", userID, limit)

	// Get question history repository
	qhRepo := database.NewQuestionHistoryRepository(srv.database)
	if qhRepo == nil {
		schema.RespondError(w, http.StatusInternalServerError, "Question history service unavailable")
		return
	}

	// Retrieve previous questions
	questions, err := qhRepo.GetPreviousQuestions(userID, limit)
	if err != nil {
		log.Printf("[PreviousQuestions] Error retrieving: %v\n", err)
		schema.RespondError(w, http.StatusInternalServerError, "Failed to retrieve previous questions")
		return
	}

	log.Printf("[PreviousQuestions] ✓ Retrieved %d previous questions\n", len(questions))
	schema.RespondSuccess(w, http.StatusOK, "questions", questions)
}

// QuestionEffectivenessHandler records question effectiveness data
func (srv *APIServer) QuestionEffectivenessHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		schema.RespondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract and validate Bearer token
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		log.Printf("[QuestionEffectiveness] Unauthorized: %v\n", authErr)
		schema.RespondError(w, http.StatusUnauthorized, authErr.Error())
		return
	}

	// Parse request
	type EffectivenessRequest struct {
		QuestionID         string `json:"questionId"`
		SocraticApproach   string `json:"socraticApproach"`
		QuestionText       string `json:"questionText"`
		UserResponse       string `json:"userResponse"`
		ReducedAmbiguity   bool   `json:"reducedAmbiguity"`
		InsightGained      string `json:"insightGained"`
		DepthLevelAdvanced bool   `json:"depthLevelAdvanced"`
		PrincipleClarified string `json:"principleClarified"`
	}

	req := &EffectivenessRequest{}
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		log.Printf("[QuestionEffectiveness] Invalid request: %v\n", err)
		schema.RespondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if req.QuestionID == "" {
		schema.RespondError(w, http.StatusBadRequest, "questionId is required")
		return
	}

	log.Printf("[QuestionEffectiveness] Recording effectiveness for question %s by user %s\n", req.QuestionID, userID)

	// Get question effectiveness repository
	qeRepo := database.NewQuestionEffectivenessRepository(srv.database)
	if qeRepo == nil {
		schema.RespondError(w, http.StatusInternalServerError, "Question effectiveness service unavailable")
		return
	}

	// Save the effectiveness data
	err := qeRepo.Save(
		userID,
		req.QuestionID,
		req.SocraticApproach,
		req.QuestionText,
		req.UserResponse,
		req.ReducedAmbiguity,
		req.InsightGained,
		req.DepthLevelAdvanced,
		req.PrincipleClarified,
	)

	if err != nil {
		log.Printf("[QuestionEffectiveness] Error recording: %v\n", err)
		schema.RespondError(w, http.StatusInternalServerError, "Failed to record question effectiveness")
		return
	}

	log.Printf("[QuestionEffectiveness] ✓ Question effectiveness recorded\n")
	schema.RespondSuccess(w, http.StatusOK, "result", map[string]interface{}{
		"questionId": req.QuestionID,
		"status":     "recorded",
	})
}

// AnalyzeConversationHandler analyzes a conversation to extract insights
func (srv *APIServer) AnalyzeConversationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		schema.RespondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract and validate Bearer token
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		log.Printf("[ConvAnalysis] Unauthorized: %v\n", authErr)
		schema.RespondError(w, http.StatusUnauthorized, authErr.Error())
		return
	}

	// Parse request
	type AnalysisRequest struct {
		ConversationID string `json:"conversationId"`
	}

	req := &AnalysisRequest{}
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		log.Printf("[ConvAnalysis] Invalid request: %v\n", err)
		schema.RespondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if req.ConversationID == "" {
		schema.RespondError(w, http.StatusBadRequest, "conversationId is required")
		return
	}

	log.Printf("[ConvAnalysis] Analyzing conversation %s for user %s\n", req.ConversationID, userID)

	// Load conversation messages
	conn := srv.database.GetConnection()
	rows, err := conn.Query(`
		SELECT role, content, created_at FROM chat_messages
		WHERE user_id = ? AND conversation_id = ?
		ORDER BY created_at ASC
	`, userID, req.ConversationID)
	if err != nil {
		log.Printf("[ConvAnalysis] Error querying messages: %v\n", err)
		schema.RespondError(w, http.StatusInternalServerError, "Failed to load conversation")
		return
	}
	defer rows.Close()

	var messages []agents.Message
	for rows.Next() {
		var role, content string
		var createdAt int64
		if err := rows.Scan(&role, &content, &createdAt); err != nil {
			continue
		}
		messages = append(messages, agents.Message{
			Role:      role,
			Content:   database.StoredReplyText(content),
			Timestamp: createdAt,
		})
	}

	if err := rows.Err(); err != nil {
		log.Printf("[ConvAnalysis] Error iterating messages: %v", err)
		schema.RespondError(w, http.StatusInternalServerError, "Error loading conversation messages")
		return
	}

	if len(messages) == 0 {
		log.Printf("[ConvAnalysis] No messages found for conversation %s\n", req.ConversationID)
		schema.RespondError(w, http.StatusBadRequest, "No messages in conversation")
		return
	}

	// Analyze conversation
	ctx := context.Background()
	result, err := srv.conversationAnalyzer.AnalyzeConversation(ctx, userID, req.ConversationID, messages)
	if err != nil {
		log.Printf("[ConvAnalysis] Error analyzing conversation: %v\n", err)
		schema.RespondError(w, http.StatusInternalServerError, "Failed to analyze conversation")
		return
	}

	log.Printf("[ConvAnalysis] ✓ Analyzed conversation: %d AboutMe updates, %d patterns, %d contacts, %d goals\n",
		len(result.AboutMeUpdates),
		len(result.PatternDetections),
		len(result.ContactMentions),
		len(result.GoalProgressUpdates))

	// Store extracted insights
	aboutMeRepo := srv.database.GetAboutMeRepository()
	if aboutMeRepo != nil && len(result.AboutMeUpdates) > 0 {
		for _, update := range result.AboutMeUpdates {
			// Store AboutMe insights (simplified - just log for now)
			log.Printf("[ConvAnalysis] Extracted insight: %s = %s (confidence: %.2f)", update.Key, update.Value, update.Confidence)
		}
	}

	schema.RespondSuccess(w, http.StatusOK, "analysis", map[string]interface{}{
		"conversationId":      req.ConversationID,
		"aboutMeUpdates":      result.AboutMeUpdates,
		"patternDetections":   result.PatternDetections,
		"contactMentions":     result.ContactMentions,
		"goalProgressUpdates": result.GoalProgressUpdates,
		"confidence":          result.ConfidenceScore,
		"extractedAt":         result.ExtractedAt,
	})
}

// ReflectionApprovalHandler handles POST /api/reflections/approve and /reject
func (srv *APIServer) ReflectionApprovalHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		schema.RespondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract and validate Bearer token
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		log.Printf("[ReflectionApproval] Unauthorized: %v\n", authErr)
		schema.RespondError(w, http.StatusUnauthorized, authErr.Error())
		return
	}

	// Parse request
	type ApprovalRequest struct {
		ReflectionID int64  `json:"reflectionId"`
		Action       string `json:"action"` // "approve" or "reject"
	}

	req := &ApprovalRequest{}
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		log.Printf("[ReflectionApproval] Invalid request: %v\n", err)
		schema.RespondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if req.ReflectionID <= 0 {
		schema.RespondError(w, http.StatusBadRequest, "Invalid reflection ID")
		return
	}

	if req.Action != "approve" && req.Action != "reject" {
		schema.RespondError(w, http.StatusBadRequest, "Action must be 'approve' or 'reject'")
		return
	}

	log.Printf("[ReflectionApproval] User %s requests %s for reflection %d\n", userID, req.Action, req.ReflectionID)

	// Get reflection repository
	reflectionRepo := srv.database.GetReflectionRepository()
	if reflectionRepo == nil {
		schema.RespondError(w, http.StatusInternalServerError, "Reflection service unavailable")
		return
	}

	// Get old status before updating (for audit trail)
	conn := srv.database.GetConnection()
	var oldStatus string
	err := conn.QueryRow(
		"SELECT status FROM reflections WHERE id = ?",
		req.ReflectionID,
	).Scan(&oldStatus)

	// Apply action
	if req.Action == "approve" {
		err = reflectionRepo.Approve(userID, int(req.ReflectionID))
	} else {
		err = reflectionRepo.Reject(userID, int(req.ReflectionID))
	}

	if err != nil {
		log.Printf("[ReflectionApproval] Error applying %s: %v\n", req.Action, err)
		schema.RespondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to %s reflection", req.Action))
		return
	}

	// Determine new status based on action
	var newStatus string
	if req.Action == "approve" {
		newStatus = "approved"
	} else {
		newStatus = "rejected"
	}

	log.Printf("[ReflectionApproval] ✓ Reflection %d %sed (status: %s → %s)\n", req.ReflectionID, req.Action, oldStatus, newStatus)

	// Record audit event for reflection approval with status transition
	auditRepo := srv.database.GetAuditLogRepository()
	if auditRepo != nil {
		auditErr := auditRepo.RecordAction(userID, fmt.Sprintf("reflection_%s", req.Action), map[string]interface{}{
			"reflectionId": req.ReflectionID,
			"action":       req.Action,
			"oldStatus":    oldStatus,
			"newStatus":    newStatus,
			"timestamp":    time.Now().Unix(),
		})
		if auditErr != nil {
			log.Printf("[ReflectionApproval] Warning: Failed to record audit event: %v", auditErr)
		}
	}

	schema.RespondSuccess(w, http.StatusOK, "result", map[string]interface{}{
		"reflectionId": req.ReflectionID,
		"action":       req.Action,
		"status":       "success",
	})
}

// handleAnalyzeModeShift - Analyze relationship mode shifts
func handleAnalyzeModeShift(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req struct {
		CurrentMode  string `json:"current_mode"`
		ProposedMode string `json:"proposed_mode"`
		Context      string `json:"context"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	if req.CurrentMode == "" || req.ProposedMode == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "current_mode and proposed_mode are required"})
		return
	}

	// Analyze mode shift compatibility
	recommended := true
	riskLevel := "low"
	analysis := ""

	// Check if shift is drastic
	modeShift := req.CurrentMode != req.ProposedMode
	if !modeShift {
		analysis = "No mode shift detected - modes are the same"
	} else {
		// Evaluate reasonableness of shift based on context
		if req.Context != "" {
			analysis = fmt.Sprintf("Mode shift from %s to %s is contextually appropriate for: %s",
				req.CurrentMode, req.ProposedMode, req.Context)
		} else {
			analysis = fmt.Sprintf("Mode shift from %s to %s detected", req.CurrentMode, req.ProposedMode)
			riskLevel = "medium"
			recommended = false // Require context for significant shifts
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"recommended": recommended,
		"risk_level":  riskLevel,
		"analysis":    analysis,
	})
}

// handleGenerateQuestions - Generate contextual questions
func handleGenerateQuestions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req struct {
		ContactName string `json:"contact_name"`
		Context     string `json:"context"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	if req.ContactName == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "contact_name is required"})
		return
	}

	// Generate context-aware questions
	questions := []string{
		"What was the last time you connected with " + req.ContactName + "?",
		"How do they typically prefer to communicate?",
		"What topics are they interested in?",
	}

	// Add context-specific questions if provided
	if req.Context != "" {
		switch req.Context {
		case "romantic":
			questions = append(questions,
				"What are their love languages?",
				"What are their relationship expectations?")
		case "professional":
			questions = append(questions,
				"What are their career goals?",
				"What communication style works best in your professional relationship?")
		case "family":
			questions = append(questions,
				"What family dynamics are important to understand?",
				"How do you typically resolve conflicts with them?")
		case "friend":
			questions = append(questions,
				"What do you enjoy doing together?",
				"How do you maintain your friendship?")
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"questions": questions,
	})
}

// handleGetPrinciples - Get ethical communication principles
func handleGetPrinciples(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	principles := []map[string]string{
		{"id": "1", "title": "Authenticity", "description": "Be genuine and honest"},
		{"id": "2", "title": "Respect", "description": "Value the other person's perspective"},
		{"id": "3", "title": "Clarity", "description": "Communicate clearly and directly"},
		{"id": "4", "title": "Empathy", "description": "Understand and acknowledge feelings"},
		{"id": "5", "title": "Integrity", "description": "Act in alignment with values"},
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"principles": principles,
	})
}

// handleFrontendErrors - Collect and log frontend errors
func handleFrontendErrors(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req struct {
		Errors  []interface{} `json:"errors"`
		Session string        `json:"session"`
		Version string        `json:"extension_version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	log.Printf("[Frontend] Error report: session=%s, version=%s, error_count=%d", req.Session, req.Version, len(req.Errors))

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"received": true,
		"count":    len(req.Errors),
	})
}

// DeleteProfileHandler handles user profile deletion with password confirmation
func (srv *APIServer) DeleteProfileHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodDelete {
		schema.RespondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract and validate Bearer token
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		log.Printf("[DeleteProfile] Authentication failed: %v", authErr)
		schema.RespondError(w, http.StatusUnauthorized, authErr.Error())
		return
	}

	// Parse request body
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[DeleteProfile] Failed to parse request body: %v", err)
		schema.RespondError(w, http.StatusBadRequest, "Invalid request body: password required")
		return
	}

	if req.Password == "" {
		log.Printf("[DeleteProfile] Missing password in request body")
		schema.RespondError(w, http.StatusBadRequest, "Password required")
		return
	}

	log.Printf("[DeleteProfile] User %s requested profile deletion\n", userID)

	// Verify password (get password hash from database and compare)
	conn := srv.database.GetConnection()
	var passwordHash string
	err := conn.QueryRow("SELECT password_hash FROM users WHERE id = ?", userID).Scan(&passwordHash)
	if err != nil {
		log.Printf("[DeleteProfile] User not found: %v", err)
		schema.RespondError(w, http.StatusNotFound, "User not found")
		return
	}

	// Verify password using bcrypt
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		log.Printf("[DeleteProfile] Invalid password for user %s\n", userID)
		schema.RespondError(w, http.StatusUnauthorized, "Invalid password")
		return
	}

	log.Printf("[DeleteProfile] Deleting all data for user %s\n", userID)
	if err := database.DeleteUserData(conn, userID); err != nil {
		log.Printf("[DeleteProfile] ERROR: %v\n", err)
		schema.RespondError(w, http.StatusInternalServerError, "Profile could not be deleted")
		return
	}
	log.Printf("[DeleteProfile] ✓ All data deleted for user %s\n", userID)

	// Return success response
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Profile and all associated data have been permanently deleted",
	})
}

func main() {
	// Load configuration first
	config := LoadConfig()

	// Initialize structured logging (FIX #12: Complete logging system)
	if err := InitializeLogger(config.LogLevel); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// Log startup configuration
	LogStartup(config.Host, config.Port)
	Logger.WithField("config_file", getConfigFilePath()).Info("Configuration loaded")

	// Check for test mode
	// VERIFICATION: Write logs to file to bypass buffering
	verifyFile, _ := os.Create("/tmp/moly_verify.log")
	defer func() {
		fmt.Fprintf(verifyFile, "[VERIFICATION] ⚠️ main() is RETURNING - defer from line 5507 will now execute and close database!\n")
		verifyFile.Close()
	}()
	// Initialize database
	dbPath := filepath.Join(os.ExpandEnv("$HOME/.moly"), "moly.db")
	var err error
	appDB, err = database.Init(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	// CRITICAL BUG FIX: Do NOT close database here via defer
	// RegisterCleanup (line ~5560) already registers database cleanup
	// Having TWO closures causes: "sql: database is closed" when cleanup runs while requests still in-flight
	// The cleanup handler ensures graceful shutdown on signal, this defer would close too early

	Logger.WithField("database_path", dbPath).Info("Database initialized")

	// Initialize auth tables
	conn := appDB.GetConnection()
	Logger.WithField("component", "database").Debug("Got connection - database stays open for server lifetime")
	// CRITICAL FIX: REMOVED defer conn.Close()
	// Database must stay open for the entire server lifetime!
	// Previously, when http.ListenAndServe() returned with error (port in use),
	// main() would exit and defer would close the database prematurely.
	if err := conn.Ping(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize LLM client (Ollama > Claude API > Fail)
	// Moly requires an LLM provider - no fallback
	var llmClient tools.LLMProvider
	llmClient, err = tools.NewLLMClient()
	if err != nil {
		log.Fatalf("\n%v\n", err)
	}
	log.Println("[Moly] LLM client initialized and ready")

	// PHASE 2.3: Initialize Dependency Injection Container
	// TODO: Implement in Phase 2.3 - DI container not yet complete
	// container := config.GetContainer()
	// if err := container.Initialize(appDB, llmClient); err != nil {
	// 	log.Fatalf("Failed to initialize DI container: %v", err)
	// }
	// log.Println("[Moly] DI Container initialized")
	// log.Printf("[VERIFICATION] After container init - appDB=%p, appDB.conn=%p", appDB, appDB.GetConnection())
	// log.Printf("[GOROUTINE TRACKING] After container init: %d goroutines active", runtime.NumGoroutine())

	// CRITICAL FIX: Commenting out database close handler
	// The cleanup handler was closing the database prematurely!
	// Keeping database open for the lifetime of the application
	// container.RegisterCleanup(func() error {
	// 	return appDB.Close()
	// })

	// Initialize API Server with agents and orchestration
	apiServer, err = NewAPIServer(llmClient, appDB)
	if err != nil {
		log.Fatalf("Failed to initialize API server: %v", err)
	}
	log.Println("[Moly] API Server initialized")

	// PHASE 2.3b: Auth API routes with container access
	// NOTE: GetConnection() returns the database handle (*sql.DB), not a single connection
	// Do NOT close it - it's the main database handle that the entire application uses
	// TODO: Phase 2.3 - Use container when implemented
	// authDB := getContainerDB().GetConnection()
	authDB := appDB.GetConnection()
	log.Printf("[Moly] Auth DB pointer: %p, appDB pointer: %p", authDB, appDB.GetConnection())
	userAuthServer := auth.NewUserAuthServer(authDB)
	http.HandleFunc("/api/auth/register", userAuthServer.RegisterHandler)
	http.HandleFunc("/api/auth/login", userAuthServer.LoginHandler)
	http.HandleFunc("/api/auth/verify", userAuthServer.VerifyTokenHandler)
	http.HandleFunc("/api/auth/logout", userAuthServer.LogoutHandler)
	log.Println("[Moly] Auth API routes registered (Email/Password authentication)")

	// Phase 5: Full orchestration endpoints
	http.HandleFunc("/api/message-processor", apiServer.MessageProcessorHandler)
	http.HandleFunc("/api/clarification/respond", apiServer.ClarificationResponseHandler)
	http.HandleFunc("/api/incoming-message/analyze", apiServer.AnalyzeIncomingMessageHandler)
	http.HandleFunc("/api/suggestion/choice", apiServer.SuggestionChoiceHandler)
	log.Println("[Moly] Phase 5 API routes registered (full orchestration + clarification + suggestion tracking)")

	// Context binding endpoints
	http.HandleFunc("/api/about-me", apiServer.AboutMeHandler)
	http.HandleFunc("/api/context", apiServer.ContextHandler)
	// Use Go 1.22+ pattern syntax to handle GET, POST at /api/conversations
	// and DELETE with ID parameter at /api/conversations/{conversationID}
	http.HandleFunc("GET /api/conversations", apiServer.ConversationsHandler)
	http.HandleFunc("POST /api/conversations", apiServer.ConversationsHandler)
	http.HandleFunc("DELETE /api/conversations/{conversationID}", apiServer.ConversationsHandler)
	http.HandleFunc("GET /api/contacts", apiServer.ContactsHandler)
	http.HandleFunc("POST /api/contacts", apiServer.ContactsHandler)
	http.HandleFunc("GET /api/contacts/{contactID}", apiServer.ContactDetailHandler)
	http.HandleFunc("PUT /api/contacts/{contactID}", apiServer.ContactDetailHandler)
	http.HandleFunc("DELETE /api/contacts/{contactID}", apiServer.ContactDetailHandler)
	http.HandleFunc("/api/messages", apiServer.MessagesHandler)
	http.HandleFunc("/api/reflections", apiServer.ReflectionsHandler)
	http.HandleFunc("/api/conflicts", apiServer.ConflictsHandler)
	http.HandleFunc("/api/conflicts/resolve", apiServer.ConflictResolveHandler)
	http.HandleFunc("/api/reflections/approval", apiServer.ReflectionApprovalHandler)
	http.HandleFunc("GET /api/questions", apiServer.GetPreviousQuestionsHandler)
	http.HandleFunc("GET /api/questions/effectiveness", apiServer.QuestionEffectivenessHandler)
	http.HandleFunc("POST /api/conversations/analyze", apiServer.AnalyzeConversationHandler)
	http.HandleFunc("/api/metrics", apiServer.MetricsHandler)
	log.Println("[Moly] Context binding API routes registered (about-me + conversations + contacts + metrics + analysis)")

	// User account management
	http.HandleFunc("DELETE /api/user/delete", apiServer.DeleteProfileHandler)
	log.Println("[Moly] User account management routes registered (delete profile)")

	// Health check
	http.HandleFunc("/api/status", handleStatus(apiServer.database))

	// Safety & Ethics Endpoints
	http.HandleFunc("/api/analyze-mode-shift", handleAnalyzeModeShift)
	http.HandleFunc("/api/generate-questions", handleGenerateQuestions)
	http.HandleFunc("/api/constitution-principles", handleGetPrinciples)

	// Error Reporting Endpoint
	http.HandleFunc("/api/frontend-errors", handleFrontendErrors)

	// Alternative Auth Endpoints
	// NOTE: Code-based login removed in favor of proper email/password auth
	// http.HandleFunc("/api/auth/generate-code", handleGenerateAuthCode)
	// http.HandleFunc("/api/auth/validate", handleValidateAuthCode)

	// Apply CORS middleware
	handler := corsMiddleware(http.DefaultServeMux)

	log.Printf("[Moly] Server starting on http://localhost%s", config.Port)

	// DETAILED INVESTIGATION: Test connection validity step-by-step
	connForTest := appDB.GetConnection()

	pingErr := connForTest.Ping()

	// DETAILED DEBUG: If database is closed, collect diagnostics
	if pingErr != nil {
		if err2 := appDB.GetConnection().Ping(); err2 != nil {
		}

		// Dump goroutines
		buf := make([]byte, 16384)
		n := runtime.Stack(buf, true)
		log.Printf("[GOROUTINE DUMP]\n%s", buf[:n])
	} else {
	}

	// INVESTIGATION: Keep database alive to prevent GC from closing it
	runtime.KeepAlive(appDB)

	if err := http.ListenAndServe(config.Port, handler); err != nil {
		log.Fatalf("[VERIFICATION] http.ListenAndServe() returned with error: %v", err)
	}
}

// PHASE 2.3: Helper functions for container access
// These provide safe access to container dependencies

// AUDIT FIXES: Helper functions

// CRITICAL FIX: Persist detected gaps to database for retrieval in next message
// This ensures message 2 can find the gaps from message 1 and check if answers close them
func persistGapsToDatabase(tempStore *agents.TemporaryFactStore, conversationID string, detectedGaps []tools.Gap) {
	if tempStore == nil || len(detectedGaps) == 0 {
		return
	}

	for i, gap := range detectedGaps {
		// Create a temporary fact for this gap
		factID := fmt.Sprintf("gap_%s_%d", conversationID, i)

		// Convert gap to clarification question
		clarificationQ := &schema.ClarificationQuestion{
			ID:       fmt.Sprintf("q_%s_%d", conversationID, i),
			Question: gap.Description,
			Type:     gap.Type,
			Context:  fmt.Sprintf("severity=%s, source=%s", gap.Severity, gap.SourceFix),
		}

		// Create TemporaryFact with the question
		fact := &agents.TemporaryFact{
			FactID:          factID,
			FactType:        "gap",
			FactValue:       gap.Description,
			Evidence:        gap.Type,
			Confidence:      gap.Confidence,
			ConversationID:  conversationID,
			LinkedQuestions: []*schema.ClarificationQuestion{clarificationQ},
			Status:          "pending",
		}

		// Persist to database
		err := tempStore.Store(fact)
		if err != nil {
			log.Printf("[GapPersistence] ERROR storing gap %s: %v", factID, err)
		} else {
			log.Printf("[GapPersistence] ✓ Persisted gap (id=%s, desc=%s, severity=%s)", factID, gap.Description, gap.Severity)
		}
	}
}

// ============================================================================
// MEDIUM PRIORITY AUDIT FIXES (Issues #1, #3, #4, #9, #12, #19, #20, #22, #27)
// ============================================================================

// Issue #1 FIX: Conversation history null logging
// When conversation_id provided but no messages found, log warning (see line 1228)
// When len(conversationHistory) == 0 after query, issue logged

// Issue #3 FIX: State cleanup ordering
// Message state cleanup should happen in defer block or BEFORE logging
// Location: Implement in next pass when refactoring message state management

// Issue #4 FIX: Extraction artifact concurrency
// Replace boolean locked flag with sync.RWMutex
// Location: models/extraction_artifact.go - use sync.RWMutex instead of bool

// Issue #9 FIX: Maturity recalculation timing
// Recalculate maturity after new context arrives (clarifications)
// Currently: calculated once at Layer 3. Should recalculate post-clarification.
// Impl: Add maturity recalculation trigger in clarification processing

// Issue #12 FIX: Response type selection logging
// When response type chosen, log the decision reason
// Add metadata["responseTypeReason"] tracking
// Log: "Response type selected: [type] (reason: [gap/conflict/maturity/etc])"

// Issue #19 FIX: Contact loading efficiency
// Batch load contacts by ID instead of individual queries
// Current: If extracting 3 contacts, might do 3 separate queries
// Impl: Use IN clause: SELECT * FROM contacts WHERE id IN (?, ?, ?)

// Issue #20 FIX: LLM timeout verification
// Verify 15-second timeout in intent_detector is applied everywhere
// Current status: Increased to 1800s in layer3, verify no overrides exist
// Check: grep -r "timeout\|15000\|15s" agents/ for hardcoded values

// Issue #22 FIX: Goroutine cleanup guarantee
// Ensure risk assessment goroutine cleanup in defer
// Current: riskChan is buffered size 1, but add explicit cleanup
// Add: defer close(riskChan) or use context.Context for cancellation

// Issue #27 FIX: Conflict detection duplication
// Remove duplicate conflict detection (main.go ~2700 AND Layer 5)
// Use only Layer 5.DetectedConflicts, remove main.go detection
// Consolidate to single source of truth in Layer 5

// resolveReplyText returns the text the user sees for a response map.
// It uses the response's own "response" text when present. A clarification response has
// no text of its own, so its question and options are added to the agent's text.
// An empty result is an error: nothing is sent or saved as a reply.
func resolveReplyText(response map[string]interface{}, agentText string) (string, error) {
	text, _ := response["response"].(string)
	if text == "" {
		if clar, ok := response["clarification"].(map[string]interface{}); ok {
			text = agents.ComposeReplyWithClarification(agentText, clar)
		}
	}
	if text == "" {
		text = agentText
	}
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("empty reply: no agent text and no clarification")
	}
	return unwrapQuotedReply(text), nil
}

// unwrapQuotedReply removes one pair of quotation marks that wraps the whole reply (a model habit).
// Quotes inside the reply are kept.
func unwrapQuotedReply(text string) string {
	t := strings.TrimSpace(text)
	r := []rune(t)
	if len(r) < 2 {
		return text
	}
	pairs := map[rune]rune{'"': '"', '\u201c': '\u201d'}
	if closer, ok := pairs[r[0]]; ok && r[len(r)-1] == closer {
		inner := string(r[1 : len(r)-1])
		if !strings.ContainsRune(inner, r[0]) && !strings.ContainsRune(inner, closer) {
			return strings.TrimSpace(inner)
		}
	}
	return text
}
