package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"moly/agents"
	"moly/auth"
	"moly/database"
	"moly/models"
	"moly/orchestration"
	"moly/safety"
	"moly/schema"
	"moly/tools"
)

var v2db *database.Database
var v2Server *V2APIServer

// V2APIServer wraps the agent system and database
type V2APIServer struct {
	llmClient                tools.LLMProvider
	llmProvider              string // "ollama", "claude", or "openai"
	database                 *database.Database
	pipeline                 *orchestration.MessagePipeline // New: greenfield pipeline
	contactManager           *agents.ContactManager
	contextAttrManager       *agents.ContextAttributeManager
	clarificationAgent       *agents.ClarificationAgent
	answerProcessor          *agents.AnswerProcessor
	incomingMessageAnalyzer  *agents.IncomingMessageAnalyzer
	conversationAnalyzer     *agents.ConversationAnalyzer
	agentSystem              *agents.AgentSystem
	safetyChecker            *safety.Checker
	riskMonitor              models.RiskMonitoringAgent
	contextExtractor         *agents.ContextExtractor
	executionStateManager    *agents.ExecutionStateManager
	messageProcessingState   *agents.MessageProcessingStateManager
}

// NewV2APIServer creates a new V2 API server
func NewV2APIServer(llm tools.LLMProvider, db *database.Database) (*V2APIServer, error) {
	if db == nil {
		return nil, fmt.Errorf("database cannot be nil")
	}

	// NOTE: TemporaryFactStore is now created per-request with userID
	contactManager := agents.NewContactManager(database.NewContactRepository(db))
	contextAttrManager := agents.NewContextAttributeManager(database.NewContextAttributeRepository(db))
	clarificationAgent := agents.NewClarificationAgent(llm)
	answerProcessor := agents.NewAnswerProcessor(clarificationAgent)
	incomingMessageAnalyzer := agents.NewIncomingMessageAnalyzer(llm)

	// Initialize ConversationAgent (V2 architecture) with Socratic support
	// ConversationAgent is critical to the system - must not fail silently
	conversationAgent, err := agents.InitializeWithSocraticSelector(llm, "config/constitution.yaml", "config")
	if err != nil {
		log.Fatalf("[Moly] FATAL: Failed to initialize ConversationAgent: %v\n\nConversationAgent is critical to system operation. This is not optional.\nPlease check:\n  - config/constitution.yaml exists and is valid\n  - config/ directory has required files\n  - LLM client is properly initialized", err)
	}

	// Wire database for Phase 2 inline conflict resolution
	conversationAgent.SetDatabase(db)
	log.Printf("[Moly] Database wired to ConversationAgent for Phase 2 conflict resolution")

	// Wrap ConversationAgent in a minimal AgentSystem struct
	agentSystem := &agents.AgentSystem{
		ConversationAgent: conversationAgent,
	}

	// Initialize RiskMonitor with LLM for contextual risk analysis
	// Uses per-request userID, so creating a dummy instance here; will recreate per-request
	riskMonitor, err := agents.NewRiskMonitorWithLLM("system", llm)
	if err != nil {
		log.Fatalf("[Moly] Failed to initialize RiskMonitor: %v", err)
	}

	// Initialize ConversationAnalyzer for extracting insights from conversations
	conversationAnalyzer := agents.NewConversationAnalyzer(llm, db)

	// Initialize greenfield pipeline with LLM and safety checker
	safetyChecker := safety.NewCheckerWithLLM(llm)

	// Type assert to get the concrete client for pipeline
	var llmClient *tools.LLMClient
	var llmProvider string = "unknown"
	if client, ok := llm.(*tools.LLMClient); ok {
		llmClient = client
		llmProvider = client.Provider
	}

	pipeline := orchestration.NewMessagePipeline(db, llmClient).WithSafetyChecker(safetyChecker)
	log.Printf("[Moly] Initialized greenfield pipeline with LLM provider: %s", llmProvider)

	return &V2APIServer{
		llmClient:               llm,
		llmProvider:             llmProvider,
		database:                db,
		pipeline:                pipeline,
		contactManager:          contactManager,
		contextAttrManager:      contextAttrManager,
		clarificationAgent:      clarificationAgent,
		answerProcessor:         answerProcessor,
		incomingMessageAnalyzer: incomingMessageAnalyzer,
		conversationAnalyzer:    conversationAnalyzer,
		agentSystem:             agentSystem,
		safetyChecker:           safetyChecker,
		riskMonitor:             riskMonitor,
		contextExtractor:        agents.NewContextExtractor(llm),
		executionStateManager:   agents.NewExecutionStateManager(db),
		messageProcessingState:  agents.NewMessageProcessingStateManager(db),
	}, nil
}

// respondJSON helper function
func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

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

	// UPDATE session last_used for activity tracking
	if _, err := conn.Exec("UPDATE sessions SET last_used = ? WHERE id = ?", time.Now().Unix(), token); err != nil {
		log.Printf("[Auth] Warning: Failed to update session last_used: %v", err)
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
// Supports both JSON (preferred) and legacy CSV formats for backward compatibility
func parseArrayFromStorage(stored string) []string {
	if stored == "" {
		return []string{}
	}

	// Try JSON format first (new format)
	var result []string
	if err := json.Unmarshal([]byte(stored), &result); err == nil {
		return result
	}

	// Fallback to CSV format (legacy, for backward compatibility)
	parts := strings.Split(stored, ",")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	// Filter out empty strings
	filtered := []string{}
	for _, part := range parts {
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	return filtered
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

// levenshteinDistance calculates the edit distance between two strings
// Returns distance: 0 = identical, higher = more different
func levenshteinDistance(a, b string) int {
	a = strings.ToLower(strings.TrimSpace(a))
	b = strings.ToLower(strings.TrimSpace(b))

	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Create distance matrix
	d := make([][]int, len(a)+1)
	for i := range d {
		d[i] = make([]int, len(b)+1)
		d[i][0] = i
	}
	for j := range d[0] {
		d[0][j] = j
	}

	// Calculate distances
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			d[i][j] = minInt(
				d[i-1][j]+1,    // deletion
				d[i][j-1]+1,    // insertion
				d[i-1][j-1]+cost, // substitution
			)
		}
	}
	return d[len(a)][len(b)]
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

// isNameSimilar checks if two names refer to the same person
// Returns true if similarity is high enough (distance <= threshold)
func isNameSimilar(name1, name2 string) bool {
	if name1 == "" || name2 == "" {
		return false
	}

	// Exact match (after normalization)
	n1 := strings.ToLower(strings.TrimSpace(name1))
	n2 := strings.ToLower(strings.TrimSpace(name2))
	if n1 == n2 {
		return true
	}

	// Edit distance check
	distance := levenshteinDistance(name1, name2)
	maxLen := len(n1)
	if len(n2) > maxLen {
		maxLen = len(n2)
	}

	// Allow up to 2 character differences or 20% of max length
	tolerance := (maxLen + 4) / 5 // ~20%
	if tolerance < 2 {
		tolerance = 2
	}
	if distance <= tolerance {
		return true
	}

	// Check if one is subset of other (partial match)
	if strings.Contains(n1, n2) || strings.Contains(n2, n1) {
		return true
	}

	return false
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-ID")
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func handleStatus(db *database.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Health check - no authentication required (frontend needs to detect backend before login)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","version":"2.1"}`))
	}
}

// MessageProcessorHandler - Full orchestration with multi-phase context processing
func (srv *V2APIServer) MessageProcessorHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

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
	if len(req.ConversationID) > 100 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "ConversationID too long (max 100 characters)"})
		return
	}

	// SAVE USER MESSAGE early for conversation history
	// This needs to happen before conversation state is loaded so history includes this message
	userMessageID := fmt.Sprintf("msg_%d_%d", time.Now().Unix(), rand.Int63())
	userMessageForDB := req.Message

	// Load or create message processing state for execution deduplication
	// This enables retries to skip already-completed pipeline stages
	msgProcState, procStateErr := srv.messageProcessingState.GetOrCreateState(userID, req.ConversationID, userMessageID)
	if procStateErr != nil {
		log.Printf("[MessageProcessor] Warning: Failed to load/create message processing state: %v", procStateErr)
		// Don't fail the request - just continue without deduplication
		msgProcState = nil
	}

	// Track if safety alert was detected (to include in response)
	var safetyAlertDetected *models.SafetyAlert

	// Safety check: detect crisis or illegal content
	if req.Message != "" {
		// Check if safety check was already done (for retries)
		if msgProcState != nil && srv.messageProcessingState.IsStageComplete(msgProcState, agents.StageSafetyCheck) {
			log.Printf("[MessageProcessor] ⊘ Safety check already complete - skipping (retry optimization)")
		} else {
			// First time execution - run the stage
			alert := srv.safetyChecker.CheckMessage(req.Message)
			if alert != nil {
				log.Printf("[Safety] Alert detected for user %s: %s (%s)", userID, alert.AlertType, alert.Title)
				// Log the incident to database
				conn := srv.database.GetConnection()
				_, err := conn.Exec(
					"INSERT INTO safety_incidents (user_id, severity, detected_at, content, detected_by, response_provided) VALUES (?, ?, ?, ?, ?, ?)",
					userID, alert.Severity, time.Now().Unix(), req.Message, "pattern_match", alert.Title,
				)
				if err != nil {
					log.Printf("[MessageProcessor] Warning: Failed to log safety incident: %v", err)
				}
				// Convert safety.SafetyAlert to models.SafetyAlert
				modelAlert := &models.SafetyAlert{
					AlertType:       string(alert.AlertType),
					Severity:        string(alert.Severity),
					Title:           alert.Title,
					Message:         alert.Message,
					Indicators:      alert.Indicators,
					Recommendations: alert.Recommendations,
				}
				// Convert resources
				for _, r := range alert.Resources {
					modelAlert.Resources = append(modelAlert.Resources, models.CrisisResource{
						Name:        r.Name,
						Description: r.Description,
						Number:      r.Number,
						URL:         r.URL,
					})
				}
				// Store for inclusion in response (don't exit early)
				safetyAlertDetected = modelAlert
				log.Printf("[MessageProcessor] Safety alert detected - will include in response")

				// Mark stage as complete
				if msgProcState != nil {
					markErr := srv.messageProcessingState.MarkStageComplete(msgProcState, agents.StageSafetyCheck, alert)
					if markErr != nil {
						log.Printf("[MessageProcessor] Warning: Failed to mark safety check complete: %v", markErr)
					}
				}
			} else if msgProcState != nil {
				// Mark as complete even if no alert (still checked, just clean)
				markErr := srv.messageProcessingState.MarkStageComplete(msgProcState, agents.StageSafetyCheck, map[string]string{"status": "clean"})
				if markErr != nil {
					log.Printf("[MessageProcessor] Warning: Failed to mark safety check complete: %v", markErr)
				}
			}
		}
	}

	// Extract context from message using LLM (contact, style, intention, goals)
	var extractedContext *models.ExtractedContext
	if req.Message != "" {
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
			// First time execution - run the stage
			var extractErr error
			extractedContext, extractErr = srv.contextExtractor.Extract(context.Background(), req.Message)
			if extractErr != nil {
				log.Printf("[MessageProcessor] Warning: Context extraction failed: %v (falling back to database)", extractErr)
			}

			// Mark stage as complete and store result
			if extractedContext != nil && msgProcState != nil {
				markErr := srv.messageProcessingState.MarkStageComplete(msgProcState, agents.StageContextExtraction, extractedContext)
				if markErr != nil {
					log.Printf("[MessageProcessor] Warning: Failed to mark context extraction complete: %v", markErr)
				}
			}
		}

		if extractedContext != nil && extractedContext.Contact != nil && extractedContext.Contact.Confidence > 0.5 {
			log.Printf("[MessageProcessor] ✓ Extracted contact: %s (%s, confidence=%.2f)",
				extractedContext.Contact.Name, extractedContext.Contact.Relationship, extractedContext.Contact.Confidence)
		}
		if extractedContext != nil && extractedContext.Style != nil && extractedContext.Style.Confidence > 0.5 {
			log.Printf("[MessageProcessor] ✓ Extracted style: %s (confidence=%.2f)",
				extractedContext.Style.Style, extractedContext.Style.Confidence)
		}
	}

	// Risk assessment: LLM-based contextual risk analysis
	// Only run if SafetyChecker didn't trigger (crisis/illegal are escalated above this level)
	var currentRiskAssessment *models.RiskAssessment
	if req.Message != "" && srv.riskMonitor != nil {
		// Check if risk assessment was already done (for retries)
		if msgProcState != nil && srv.messageProcessingState.IsStageComplete(msgProcState, agents.StageRiskAssessment) {
			log.Printf("[MessageProcessor] ⊘ Risk assessment already complete - skipping (retry optimization)")
		} else {
			// First time execution - run the stage
			riskAssessment, riskErr := srv.riskMonitor.AssessRisk(userID, req.Message)
			if riskErr != nil {
				log.Printf("[RiskMonitor] Warning: Risk assessment failed: %v (treating as clear)", riskErr)
			} else if riskAssessment != nil {
				currentRiskAssessment = riskAssessment
				log.Printf("[RiskMonitor] ✓ Assessment for user %s: level=%s severity=%d", userID, riskAssessment.RiskLevel, riskAssessment.Severity)

				// Mark stage as complete and store result
				if msgProcState != nil {
					markErr := srv.messageProcessingState.MarkStageComplete(msgProcState, agents.StageRiskAssessment, riskAssessment)
					if markErr != nil {
						log.Printf("[MessageProcessor] Warning: Failed to mark risk assessment complete: %v", markErr)
					}
				}
			}

			// If high risk, provide educational response instead of processing
			if riskAssessment != nil && (riskAssessment.RiskLevel == "high" || riskAssessment.RiskLevel == "immediate") {
				log.Printf("[RiskMonitor] High risk detected - providing educational response")
				// Log risk assessment to database
				conn := srv.database.GetConnection()
				_, err := conn.Exec(
					"INSERT INTO safety_incidents (user_id, severity, detected_at, content, detected_by, response_provided) VALUES (?, ?, ?, ?, ?, ?)",
					userID, riskAssessment.Severity, time.Now().Unix(), req.Message, "risk_assessment", riskAssessment.Message,
				)
				if err != nil {
					log.Printf("[MessageProcessor] Warning: Failed to log risk assessment: %v", err)
				}

				// Return educational response with questions instead of processing
				response := map[string]interface{}{
					"risk_assessment": riskAssessment,
					"phase":           "risk_education",
					"message":         riskAssessment.Message,
					"questions":       riskAssessment.EducationalQuestions,
				}
				schema.RespondSuccess(w, http.StatusOK, "response", response)
				return
			}
		}
	}

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

					response := map[string]interface{}{
						"success": true,
						"phase":   "clarification_complete",
						"message": "Great! I've gathered all the context I need about this.",
						"factId":  fact.FactID,
					}
					schema.RespondSuccess(w, http.StatusOK, "response", response)
					return
				}

				// More questions remain
				remainingQs := tempStore.RemainingQuestionsWithObjects(fact.FactID)
				response := map[string]interface{}{
					"success": true,
					"phase":   "context_gathering",
					"message": "Thanks! One more thing:",
					"questions": remainingQs,
					"factId":    fact.FactID,
				}
				schema.RespondSuccess(w, http.StatusOK, "response", response)
				return
			}
		}
	} else if len(pendingClarifications) > 0 && req.Message == "" {
		// User logged in but no new message - show pending questions
		log.Printf("[MessageProcessor] Showing pending clarifications from previous session")
		fact := pendingClarifications[0]
		unansweredQuestions := tempStore.RemainingQuestionsWithObjects(fact.FactID)

		if len(unansweredQuestions) > 0 {
			response := map[string]interface{}{
				"success": true,
				"phase":   "context_gathering",
				"message": "You have unfinished clarifications from last time:",
				"questions": unansweredQuestions,
				"factId":    fact.FactID,
			}
			schema.RespondSuccess(w, http.StatusOK, "response", response)
			return
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

				// Create inline resolver to parse the answer
				inlineResolver := tools.NewInlineConflictResolver(srv.database)

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
			log.Printf("[MessageProcessor] Warning: ConflictRepository not available for conflict answer handling")
		}
	}

	// Use ConversationAgent (V2 architecture)
	if srv.agentSystem == nil || srv.agentSystem.ConversationAgent == nil {
		schema.RespondError(w, http.StatusInternalServerError, "Agent system not initialized")
		return
	}

	// LOAD OR CREATE CONVERSATION - reuse existing for same user (within 30-day window)
	conversationID := req.ConversationID
	conn := srv.database.GetConnection()
	conversationJustCreated := false
	isNewBrowserSession := false

	if conversationID == "" || conversationID == "null" {
		// Try to load most recent conversation for this user (within 30 days)
		var existingConvID, existingSessionID string
		thirtyDaysAgo := time.Now().Unix() - (30 * 24 * 60 * 60)
		err := conn.QueryRow(
			`SELECT id, COALESCE(browser_session_id, '') FROM conversations WHERE user_id = ? AND updated_at > ? ORDER BY updated_at DESC LIMIT 1`,
			userID, thirtyDaysAgo,
		).Scan(&existingConvID, &existingSessionID)

		if err == nil && existingConvID != "" {
			conversationID = existingConvID
			// Check if this is a new browser session
			isNewBrowserSession = (existingSessionID != "" && existingSessionID != req.BrowserSessionId)
			log.Printf("[MessageProcessor] ✓ Loaded existing conversation (within 30-day window): %s", conversationID)
			if isNewBrowserSession {
				log.Printf("[MessageProcessor] ✓ Detected new browser session (was: %s, now: %s)", existingSessionID, req.BrowserSessionId)
			}
		} else {
			// Create new conversation only if none exists or all are older than 30 days
			now := time.Now().Unix()
			conversationID = fmt.Sprintf("conv_%d", now)
			// Try with browser_session_id first, fall back if column doesn't exist
			_, err := conn.Exec(`
				INSERT INTO conversations (id, user_id, name, type, description, browser_session_id, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`, conversationID, userID, "Direct Message", "direct", "Persistent conversation", req.BrowserSessionId, now, now)

			if err != nil && strings.Contains(err.Error(), "no column named browser_session_id") {
				// Fallback for older schemas without browser_session_id column
				_, err = conn.Exec(`
					INSERT INTO conversations (id, user_id, name, type, description, created_at, updated_at)
					VALUES (?, ?, ?, ?, ?, ?, ?)
				`, conversationID, userID, "Direct Message", "direct", "Persistent conversation", now, now)
				log.Printf("[MessageProcessor] ⚠ Created conversation without browser_session_id (old schema)")
			}

			if err != nil {
				log.Printf("[MessageProcessor] Warning: Failed to create conversation: %v", err)
			} else {
				log.Printf("[MessageProcessor] ✓ Created conversation: %s (sessionId: %s)", conversationID, req.BrowserSessionId)
				conversationJustCreated = true
				isNewBrowserSession = true
			}
		}
	}

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

	// Update req.ConversationID for later use
	req.ConversationID = conversationID

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
		// Use database values as source of truth
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

	// Supplement with request values if database doesn't have them
	if req.AboutMe != nil {
		if aboutMeStyle == "" {
			if v, ok := req.AboutMe["communicationStyle"].(string); ok {
				aboutMeStyle = v
			}
		}
		if len(aboutMeValues) == 0 {
			if v, ok := req.AboutMe["coreValues"].([]interface{}); ok {
				for _, val := range v {
					if s, ok := val.(string); ok {
						aboutMeValues = append(aboutMeValues, s)
					}
				}
			}
		}
		if aboutMeTone == "" {
			if v, ok := req.AboutMe["tonePreference"].(string); ok {
				aboutMeTone = v
			}
		}
	}

	log.Printf("[MessageProcessor] About Me loaded: style=%s, values=%d, tone=%s", aboutMeStyle, len(aboutMeValues), aboutMeTone)

	// Fetch conversation history if conversation ID provided
	conversationHistory := []models.Message{}
	if req.ConversationID != "" && req.ConversationID != "null" {
		conn := srv.database.GetConnection()
		rows, err := conn.Query(
			"SELECT id, role, content, created_at FROM chat_messages WHERE conversation_id = ? ORDER BY created_at ASC LIMIT 10",
			req.ConversationID,
		)
		if err != nil {
			log.Printf("[MessageProcessor] Warning: Failed to fetch conversation history: %v", err)
		} else {
			defer rows.Close()
			for rows.Next() {
				var id, role, content string
				var createdAt int64
				if err := rows.Scan(&id, &role, &content, &createdAt); err != nil {
					log.Printf("[MessageProcessor] Warning: Error scanning message row: %v", err)
					continue
				}
				conversationHistory = append(conversationHistory, models.Message{
					ID:        id,
					Role:      role,
					Content:   content,
					Timestamp: createdAt,
				})
			}
		}
	}

	// PHASE 3: LOAD PAST REFLECTIONS (user's learned characteristics from past conversations)
	var relevantReflections []models.Reflection
	conn = srv.database.GetConnection()
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
				ID:         fmt.Sprintf("%d", id),
				Status:     status,
				CreatedAt:  createdAt,
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
	}

	// PHASE 3B: Load past intention (user's goal from previous messages)
	var pastIntention string
	conn = srv.database.GetConnection()
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
	conn = srv.database.GetConnection()
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
		if len(recentSafetyIncidents) > 0 {
			log.Printf("[MessageProcessor] ✓ Loaded %d recent safety incidents", len(recentSafetyIncidents))
		}
	}

	// PHASE 3D: Use current risk assessment (for context awareness)
	// Use the current message's risk assessment if available, otherwise default to empty
	var lastRiskAssessment map[string]interface{}
	if currentRiskAssessment != nil {
		lastRiskAssessment = map[string]interface{}{
			"level":    currentRiskAssessment.RiskLevel,
			"severity": float64(currentRiskAssessment.Severity),
			"emotion":  "unknown",
		}
		log.Printf("[MessageProcessor] ✓ Using current risk assessment: level=%s severity=%d", currentRiskAssessment.RiskLevel, currentRiskAssessment.Severity)
	} else {
		log.Printf("[MessageProcessor] ⊘ No current risk assessment available")
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
	askedRows, _ := conn.Query(`
		SELECT DISTINCT clarification_type FROM clarification_questions
		WHERE user_id = ? AND conversation_id = ? AND (status = 'answered' OR status = 'pending')
		ORDER BY created_at DESC
	`, userID, conversationID)
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

	if len(selectedContactIds) > 0 {
		// If specific contacts are selected, load from that list (use first selected contact)
		err = conn.QueryRow(
			"SELECT id, name, relationship, characteristics FROM user_contacts WHERE user_id = ? AND id = ? LIMIT 1",
			userID, selectedContactIds[0],
		).Scan(&contactID, &contactName, &contactRelationship, &charJSON)
		if err == nil && contactName != "" {
			log.Printf("[MessageProcessor] ✓ Loaded selected contact (ID: %s): %s (%s)", selectedContactIds[0], contactName, contactRelationship)
		}
	} else {
		// Otherwise load most recent contact
		err = conn.QueryRow(
			"SELECT id, name, relationship, characteristics FROM user_contacts WHERE user_id = ? ORDER BY updated_at DESC LIMIT 1",
			userID,
		).Scan(&contactID, &contactName, &contactRelationship, &charJSON)
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

		log.Printf("[MessageProcessor] ✓ Loaded existing contact: %s (%s)", contactName, contactRelationship)
	} else if err != sql.ErrNoRows && err != nil {
		log.Printf("[MessageProcessor] Warning: Failed to load contact from database: %v", err)
	}

	// Calculate isFirstMessageInConversation BEFORE prepending (critical for accurate greeting logic)
	// This must be done before modifying conversationHistory
	isFirstMessageInConversation := len(conversationHistory) == 0

	// Prepend current message to conversation history so agent has access to current message
	// Note: This is not persisted yet; it's passed in-memory to the agent
	if userMessageForDB != "" {
		currentMessageEntry := models.Message{
			ID:        userMessageID,
			Role:      "user",
			Content:   userMessageForDB,
			Timestamp: now,
		}
		// Prepend to beginning of history (most recent first when reading backwards)
		conversationHistory = append([]models.Message{currentMessageEntry}, conversationHistory...)
		log.Printf("[MessageProcessor] Added current message to conversation context for agent processing")
	}

	// Load user behavioral profile (learning/patterns from past interactions)
	var userBehaviorProfile *models.UserBehavioralProfile
	if srv.database != nil {
		learningAgent, err := agents.NewLearningAgentWithDB(userID, srv.database)
		if err != nil {
			log.Printf("[MessageProcessor] Warning: Failed to initialize learning agent: %v", err)
		} else if learningAgent != nil {
			profile, err := learningAgent.GetUserProfile(userID)
			if err != nil {
				log.Printf("[MessageProcessor] Warning: Failed to load user behavioral profile: %v", err)
			} else if profile != nil {
				userBehaviorProfile = profile
				log.Printf("[MessageProcessor] ✓ Loaded user behavioral profile (confidence: %.2f)", profile.Confidence)
			}
		}
	}

	// Identify missing context gaps (after all loading complete)
	gaps := []string{}
	contextFieldsLoaded := 0
	contextFieldsTotal := 8

	if aboutMeStyle == "" {
		gaps = append(gaps, "communicationStyle")
	} else {
		contextFieldsLoaded++
	}

	if len(aboutMeValues) == 0 {
		gaps = append(gaps, "coreValues")
	} else {
		contextFieldsLoaded++
	}

	if contactProfile == nil || contactProfile.Name == "" {
		gaps = append(gaps, "contact")
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

	// Calculate context quality
	contextQuality := "minimal"
	if contextFieldsLoaded >= 6 {
		contextQuality = "comprehensive"
	} else if contextFieldsLoaded >= 4 {
		contextQuality = "partial"
	}

	if len(gaps) > 0 {
		log.Printf("[MessageProcessor] Context gaps identified: %v (%d/%d fields loaded, quality: %s)", gaps, contextFieldsLoaded, contextFieldsTotal, contextQuality)
	}

	// If safety alert was detected, return immediately with alert response (no agent processing)
	if safetyAlertDetected != nil {
		log.Printf("[MessageProcessor] Skipping agent processing due to safety alert")

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

		response := map[string]interface{}{
			"success": true,
			"phase":   "safety_alert",
			// Phase structures (empty, since no agent processed)
			"phase1": map[string]interface{}{
				"facts":  []interface{}{},
				"shifts": []interface{}{},
			},
			"phase2": map[string]interface{}{
				"clarifications": []interface{}{},
				"resolved":       0,
			},
			"phase3": map[string]interface{}{
				"unknown_contacts": []interface{}{},
				"created_contacts": []interface{}{},
			},
			"phase4": map[string]interface{}{
				"saved_attributes": []interface{}{},
				"conflicts":        []interface{}{},
			},
			"action_required": map[string]interface{}{
				"needsClarification": false,
				"clarificationQs":    []interface{}{},
				"temporaryFacts":     []interface{}{},
				"hasConflicts":       false,
				"conflicts":          []interface{}{},
			},
			// Response fields for safety alert
			"response":         safetyAlertDetected.Title + ": " + safetyAlertDetected.Message,
			"suggestions":      []interface{}{},
			"riskWarning":      nil,
			"safetyAlert":      safetyAlertDetected,
			"processingTimeMs": int(time.Since(startTime).Milliseconds()),
			"metadata":         map[string]interface{}{},
			"reflection":       nil,
			"constitutionConcerns": nil,
			"extractedContact": nil,
		}
		respondJSON(w, http.StatusOK, response)
		return
	}

	// Determine if this is the first message of a NEW browser session
	// isNewBrowserSession is true when:
	// 1. Conversation was just created in this request, OR
	// 2. Browser session ID changed (user closed browser and came back)
	isFirstMessageOfSession := isNewBrowserSession || conversationJustCreated

	ctx := models.Context{
		ConversationID: conversationID,                   // For recording questions and interactions
		AboutMe: &models.AboutMe{
			UserID:             userID,
			CommunicationStyle: aboutMeStyle,
			Values:             aboutMeValues,
			PreferredTone:      aboutMeTone,
		},
		ContactProfile:          contactProfile,
		ConversationHistory:     conversationHistory,
		ExtractedContext:        extractedContext,           // Pass LLM-extracted context to agent
		PastIntention:           pastIntention,              // User's goal from previous message(s)
		RecentSafetyIncidents:   recentSafetyIncidents,      // Recent safety alerts to prevent re-alerting
		LastRiskAssessment:      lastRiskAssessment,         // Most recent risk assessment result
		ConversationPhase:       string(execState.Phase),    // Current conversation phase for phase-aware responses
		UserBehaviorProfile:     userBehaviorProfile,        // User's learned patterns and preferences
		RelevantReflections:     relevantReflections,        // Past insights from similar conversations
		Gaps:                    gaps,                       // Missing context fields
		ContextQuality:          contextQuality,            // Calculated based on loaded fields
		SessionID:               req.BrowserSessionId,       // Browser session identifier
		IsFirstMessageOfSession: isFirstMessageOfSession,    // true only for first message in new browser session
		IsFirstMessageInConversation: isFirstMessageInConversation, // true only for first message in this conversation (calculated BEFORE prepending)
	}

	// Response generation and ethical gate check
	var agentResp *models.ConversationResponse

	// Check if response generation was already done (for retries)
	if msgProcState != nil && srv.messageProcessingState.IsStageComplete(msgProcState, agents.StageResponseGeneration) {
		log.Printf("[MessageProcessor] ⊘ Response generation already complete - skipping (retry optimization)")
		// Load result from state
		if result := srv.messageProcessingState.GetStageResult(msgProcState, agents.StageResponseGeneration); result != nil {
			if resp, ok := result.(*models.ConversationResponse); ok {
				agentResp = resp
			} else {
				log.Printf("[MessageProcessor] Warning: Stored response has unexpected type, regenerating")
			}
		}
	}

	if agentResp == nil {
		// First time execution - run the stage
		var respErr error
		agentResp, respErr = srv.agentSystem.ConversationAgent.Run(ctx)
		if respErr != nil || agentResp == nil {
			// Fatal error - unable to generate any response
			log.Printf("[MessageProcessor] Fatal error: %v\n", respErr)
			schema.RespondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to process message: %v", respErr))
			return
		}

		// Mark stage as complete and store result
		if msgProcState != nil {
			markErr := srv.messageProcessingState.MarkStageComplete(msgProcState, agents.StageResponseGeneration, agentResp)
			if markErr != nil {
				log.Printf("[MessageProcessor] Warning: Failed to mark response generation complete: %v", markErr)
			}
		}
	}

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
	if _, execErr := conn.Exec(`
		INSERT INTO chat_messages (id, user_id, conversation_id, role, content, context_extracted, metadata, created_at)
		VALUES (?, ?, ?, 'user', ?, ?, ?, ?)
		ON CONFLICT(id) DO NOTHING
	`, userMessageID, userID, conversationID, userMessageForDB, contextExtractedJSON, "{}", now); execErr != nil {
		log.Printf("[MessageProcessor] WARNING: Failed to save user message to chat_messages: %v", execErr)
	} else {
		log.Printf("[MessageProcessor] ✓ Saved user message to chat_messages with extracted context: %s", userMessageID)
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

	// SAVE AGENT RESPONSE to chat_messages for conversation history (with metadata)
	agentResponseID := fmt.Sprintf("msg_%d_%d", now, rand.Int63())
	agentResponseJSON, _ := json.Marshal(map[string]interface{}{
		"phase":    agentResp.Phase,
		"response": agentResp.Response,
	})

	// Serialize metadata for storage
	metadataJSON := "{}"
	if agentResp.Metadata != nil {
		if b, err := json.Marshal(agentResp.Metadata); err == nil {
			metadataJSON = string(b)
		}
	}

	if _, execErr := conn.Exec(`
		INSERT INTO chat_messages (id, user_id, conversation_id, role, content, context_extracted, metadata, created_at)
		VALUES (?, ?, ?, 'assistant', ?, ?, ?, ?)
		ON CONFLICT(id) DO NOTHING
	`, agentResponseID, userID, conversationID, string(agentResponseJSON), "{}", metadataJSON, now); execErr != nil {
		log.Printf("[MessageProcessor] WARNING: Failed to save agent response to chat_messages: %v", execErr)
	} else {
		log.Printf("[MessageProcessor] ✓ Saved agent response to chat_messages with metadata: %s", agentResponseID)
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

	// PHASE 2: SAVE CONTACT CHARACTERISTICS (when contact is mentioned)
	if agentResp.ExtractedContact != nil && agentResp.ExtractedContact.Name != "" {
		contactID := fmt.Sprintf("contact_%d_%d", now, rand.Int63())
		conn := srv.database.GetConnection()

		// First try exact match, then fall back to fuzzy matching for similar names
		var existingID string
		err := conn.QueryRow(
			"SELECT id FROM user_contacts WHERE user_id = ? AND name = ?",
			userID, agentResp.ExtractedContact.Name,
		).Scan(&existingID)

		// If exact match not found, try fuzzy matching
		if err == sql.ErrNoRows {
			// Get all contacts for this user and check for similar names
			rows, queryErr := conn.Query(
				"SELECT id, name FROM user_contacts WHERE user_id = ? ORDER BY updated_at DESC LIMIT 20",
				userID,
			)
			if queryErr == nil {
				defer rows.Close()
				for rows.Next() {
					var cid, cname string
					if scanErr := rows.Scan(&cid, &cname); scanErr == nil {
						if isNameSimilar(agentResp.ExtractedContact.Name, cname) {
							existingID = cid
							log.Printf("[MessageProcessor] ✓ Found similar contact via fuzzy match: '%s' matches existing '%s'", agentResp.ExtractedContact.Name, cname)
							err = nil // Reset err to indicate match found
							break
						}
					}
				}
			}
		}

		// Prepare characteristics JSON if we have reflection data about the contact
		var charJSON []byte
		if agentResp.Reflection != nil && len(agentResp.Reflection.Characteristics) > 0 {
			if b, err := json.Marshal(agentResp.Reflection.Characteristics); err != nil {
				log.Printf("[MessageProcessor] Warning: Failed to marshal contact characteristics: %v", err)
			} else {
				charJSON = b
			}
		}

		if err == sql.ErrNoRows {
			// Contact doesn't exist, insert it
			_, insertErr := conn.Exec(`
				INSERT INTO user_contacts (id, user_id, name, relationship, characteristics, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`, contactID, userID, agentResp.ExtractedContact.Name, agentResp.ExtractedContact.Relationship, string(charJSON), now, now)

			if insertErr != nil {
				log.Printf("[MessageProcessor] Warning: Failed to save contact: %v", insertErr)
			} else {
				charCount := 0
				if agentResp.Reflection != nil {
					charCount = len(agentResp.Reflection.Characteristics)
				}
				log.Printf("[MessageProcessor] ✓ Saved contact: %s (%s) with %d characteristics",
					agentResp.ExtractedContact.Name,
					agentResp.ExtractedContact.Relationship,
					charCount)
			}
		} else if err != nil {
			log.Printf("[MessageProcessor] Warning: Failed to check existing contact: %v", err)
		} else {
			// Contact exists, update relationship and characteristics
			// Always update relationship if present
			if agentResp.ExtractedContact.Relationship != "" {
				_, err := conn.Exec(
					"UPDATE user_contacts SET relationship = ?, updated_at = ? WHERE id = ?",
					agentResp.ExtractedContact.Relationship, now, existingID,
				)
				if err != nil {
					log.Printf("[MessageProcessor] Warning: Failed to update contact relationship: %v", err)
				}
			}

			// Update characteristics if we have them from reflection
			if len(charJSON) > 0 {
				_, err := conn.Exec(
					"UPDATE user_contacts SET characteristics = ?, updated_at = ? WHERE id = ?",
					string(charJSON), now, existingID,
				)
				if err != nil {
					log.Printf("[MessageProcessor] Warning: Failed to update contact characteristics: %v", err)
				} else {
					log.Printf("[MessageProcessor] ✓ Updated contact %s characteristics: %d traits",
						agentResp.ExtractedContact.Name,
						len(agentResp.Reflection.Characteristics))
				}
			}
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
			err := conn.QueryRow("SELECT id FROM user_contacts WHERE user_id = ? AND name = ? LIMIT 1",
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
		_, saveErr := conn.Exec(`
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
		conn := srv.database.GetConnection()

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

				_, styleErr := conn.Exec(`
					INSERT INTO about_me (user_id, communication_style, core_values, tone_preference, updated_at, created_at)
					VALUES (?, ?, ?, ?, ?, ?)
					ON CONFLICT(user_id) DO UPDATE SET
						communication_style = CASE WHEN communication_style IS NULL OR communication_style = '' THEN excluded.communication_style ELSE communication_style END,
						core_values = CASE WHEN core_values IS NULL OR core_values = '[]' THEN excluded.core_values ELSE core_values END,
						tone_preference = CASE WHEN tone_preference IS NULL OR tone_preference = '' THEN excluded.tone_preference ELSE tone_preference END,
						updated_at = excluded.updated_at
				`, userID, extractedContext.Style.Style, valuesJSON, extractedContext.Style.Tone, now, now)

				if styleErr != nil {
					log.Printf("[MessageProcessor] Warning: Failed to save extracted style: %v", styleErr)
				} else {
					log.Printf("[MessageProcessor] ✓ Saved extracted style to about_me: %s", extractedContext.Style.Style)
				}
			} else {
				log.Printf("[MessageProcessor] Skipping style save - conflict requires user approval")
			}
		}

		// Save extracted contact to user_contacts (if confidence is high)
		if extractedContext.Contact != nil && extractedContext.Contact.Confidence > 0.6 {
			log.Printf("[MessageProcessor] Checking for contact conflicts...")

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
			} else if contactDecision.Action == "auto_merge" {
				log.Printf("[MessageProcessor] AUTO-MERGE: %s", contactDecision.AutoMergeInfo)
			}

			// Only proceed with save if conflict handler says it's OK
			if !contactDecision.SkipUpdate {
				log.Printf("[MessageProcessor] Saving extracted contact: %s (confidence=%.2f)", extractedContext.Contact.Name, extractedContext.Contact.Confidence)

				traitsJSON := "[]"
				if len(extractedContext.Contact.Traits) > 0 {
					if b, err := json.Marshal(extractedContext.Contact.Traits); err == nil {
						traitsJSON = string(b)
					}
				}

				contactID := fmt.Sprintf("contact_%d_%d", now, rand.Int63())
				_, contactErr := conn.Exec(`
					INSERT INTO user_contacts (id, user_id, name, relationship, characteristics, updated_at, created_at)
					VALUES (?, ?, ?, ?, ?, ?, ?)
					ON CONFLICT(user_id, name) DO UPDATE SET
						relationship = CASE WHEN relationship IS NULL OR relationship = '' THEN excluded.relationship ELSE relationship END,
						characteristics = CASE WHEN characteristics IS NULL OR characteristics = '[]' THEN excluded.characteristics ELSE characteristics END,
						updated_at = excluded.updated_at
				`, contactID, userID, extractedContext.Contact.Name, extractedContext.Contact.Relationship, traitsJSON, now, now)

				if contactErr != nil {
					log.Printf("[MessageProcessor] Warning: Failed to save extracted contact: %v", contactErr)
				} else {
					log.Printf("[MessageProcessor] ✓ Saved extracted contact to user_contacts: %s (%s)", extractedContext.Contact.Name, extractedContext.Contact.Relationship)
				}
			} else {
				log.Printf("[MessageProcessor] Skipping contact save - conflict requires user approval")
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
			} else if intentionDecision.Action == "auto_merge" {
				log.Printf("[MessageProcessor] AUTO-MERGE: %s", intentionDecision.AutoMergeInfo)
			}

			// Only proceed with save if conflict handler says it's OK
			if !intentionDecision.SkipUpdate {
				log.Printf("[MessageProcessor] Saving extracted intention: %s", extractedContext.Intention)

				_, intentionErr := conn.Exec(`
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
			_, goalsErr := conn.Exec(`
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
	}

	// Map ConversationResponse to frontend response format
	response := map[string]interface{}{
		"success":        true,
		"phase":          agentResp.Phase,
		"conversationId": conversationID, // Return conversation ID so frontend can store it
		// ConversationAgent specific fields
		"response":         agentResp.Response,
		"safetyAlert":      agentResp.SafetyAlert,
		"processingTimeMs": agentResp.ProcessingTimeMs,
		"metadata":         agentResp.Metadata,
		"reflection":       agentResp.Reflection,
		"extractedContact": agentResp.ExtractedContact,
	}

	// Add error field only if present (non-fatal errors)
	if agentResp.Error != "" {
		response["error"] = agentResp.Error
	}

	// Ensure metadata exists for frontend ethical intervention display
	if agentResp.Metadata == nil {
		agentResp.Metadata = make(map[string]interface{})
	}

	// Clean up message processing state now that message has been fully processed
	// This allows the space to be reclaimed for the next message
	if msgProcState != nil {
		cleanupErr := srv.messageProcessingState.DeleteState(userID, req.ConversationID, userMessageID)
		if cleanupErr != nil {
			log.Printf("[MessageProcessor] Warning: Failed to clean up message processing state: %v", cleanupErr)
		} else {
			log.Printf("[MessageProcessor] ✓ Cleaned up message processing state for message %s", userMessageID)
		}
	}

	// PHASE 7: Record this interaction for behavioral profile learning
	if srv.database != nil {
		learningAgent, learningErr := agents.NewLearningAgentWithDB(userID, srv.database)
		if learningErr != nil {
			log.Printf("[MessageProcessor] Warning: Failed to initialize learning agent for recording: %v", learningErr)
		} else if learningAgent != nil {
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

	respondJSON(w, http.StatusOK, response)
}

// ClarificationResponseHandler - Handle user responses to clarification questions
func (srv *V2APIServer) ClarificationResponseHandler(w http.ResponseWriter, r *http.Request) {
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
		"UPDATE clarification_questions SET status = 'answered', answered_at = ? WHERE id = ?",
		now, req.QuestionID,
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

	// Process response with new AnswerProcessor
	processedResult, err := srv.answerProcessor.ProcessResponse(
		req.QuestionID,
		req.UserResponse,
		linkedFacts,
		userID,
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
		now := time.Now().Unix()
		_, err := conn.Exec(`
			INSERT INTO about_me (user_id, communication_style, preferences, patterns, tone_preference, updated_at, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(user_id) DO UPDATE SET
			  communication_style = CASE
				WHEN communication_style IS NULL OR communication_style = '' THEN excluded.communication_style
				ELSE communication_style
			  END,
			  preferences = CASE
				WHEN preferences IS NULL OR preferences = '' THEN excluded.preferences
				ELSE preferences
			  END,
			  patterns = CASE
				WHEN patterns IS NULL OR patterns = '[]' THEN excluded.patterns
				ELSE patterns
			  END,
			  tone_preference = CASE
				WHEN tone_preference IS NULL OR tone_preference = '' THEN excluded.tone_preference
				ELSE tone_preference
			  END,
			  updated_at = excluded.updated_at
		`, userID, contextStr, contextStr, patternsJSON, contextStr, now, now)

		if err != nil {
			log.Printf("[Clarification] Warning: Failed to save About Me: %v", err)
		} else {
			log.Printf("[Clarification] ✓ Saved About Me: style='%s', patterns=%d items", contextStr, len(patternsJSON))
		}
	}

	// Convert ProcessedAnswerResponse to format expected by frontend
	response := map[string]interface{}{
		"status":     processedResult.Status,
		"molyReply":  processedResult.MolyReply,
		"contextToSave": processedResult.ContextToSave,
		"conflicts": processedResult.Conflicts,
		"needsMore": processedResult.NeedsMoreClarification,
	}

	respondJSON(w, http.StatusOK, response)
}

// AnalyzeIncomingMessageHandler - Analyze incoming message and generate suggestions
func (srv *V2APIServer) AnalyzeIncomingMessageHandler(w http.ResponseWriter, r *http.Request) {
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
			 FROM user_contacts WHERE user_id = ? AND name = ?`,
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
					conversationHistory = append(conversationHistory, content)
				}
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

	// Record incoming message analysis for learning
	suggestionsJSON, _ := json.Marshal(suggestions)
	_, recordErr := conn.Exec(`
		INSERT INTO context_attributes (user_id, fact_type, fact_value, confidence, evidence, created_at)
		VALUES (?, 'incoming_message_sender', ?, 0.8, ?, ?)
	`, userID, sender, string(suggestionsJSON), time.Now().Unix())

	if recordErr != nil {
		log.Printf("[IncomingMessage] Warning: Failed to record incoming message analysis: %v", recordErr)
	} else {
		log.Printf("[IncomingMessage] ✓ Recorded incoming message analysis: sender=%s, suggestions=%d", sender, len(suggestions))
	}

	respondJSON(w, http.StatusOK, response)
}

// SuggestionChoiceHandler - Record when user picks a suggestion
func (srv *V2APIServer) SuggestionChoiceHandler(w http.ResponseWriter, r *http.Request) {
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
	learningAgent, err := agents.NewLearningAgentWithDB(userID, srv.database)
	if err != nil {
		log.Printf("[SuggestionChoice] Error initializing LearningAgent: %v", err)
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
			"conversationId":   req.ConversationID,
			"suggestionIndex":  req.SuggestionIndex,
			"modified":         req.ModifiedText != "",
			"userFeedback":     req.UserFeedback,
		})
		if auditErr != nil {
			log.Printf("[SuggestionChoice] Warning: Failed to record audit event: %v", auditErr)
		}
	}

	schema.RespondSuccess(w, http.StatusOK, "choice", map[string]interface{}{"message": "Suggestion choice recorded"})
}

// AboutMeHandler - Get or save user's About Me profile
func (srv *V2APIServer) AboutMeHandler(w http.ResponseWriter, r *http.Request) {
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

// ContextHandler handles GET /api/v2/context - Returns user's context quality for a conversation
func (srv *V2APIServer) ContextHandler(w http.ResponseWriter, r *http.Request) {
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
			"overallScore":     completenessScore,
			"completenessLevel": completenessLevel,
		},
		"missingContextGaps": gaps,
	}

	log.Printf("[Context] Quality for %s: score=%.2f level=%s gaps=%d",
		conversationID, completenessScore, completenessLevel, len(gaps))
	schema.RespondSuccess(w, http.StatusOK, "context", response)
}

// ConversationsHandler - Get or create conversations
func (srv *V2APIServer) ConversationsHandler(w http.ResponseWriter, r *http.Request) {
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
	}
}

// ContactsHandler - Get or create user contacts
func (srv *V2APIServer) ContactsHandler(w http.ResponseWriter, r *http.Request) {
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
			"SELECT id, name, relationship, notes, created_at FROM user_contacts WHERE user_id = ? ORDER BY updated_at DESC",
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
			"INSERT INTO user_contacts (id, user_id, name, relationship, notes, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			contactID,
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

// MessagesHandler - Get messages for a conversation
func (srv *V2APIServer) MessagesHandler(w http.ResponseWriter, r *http.Request) {
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
			"content":        content,
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

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"messages": messages,
	})
}

// ReflectionsHandler - Get, approve, or reject reflections
func (srv *V2APIServer) ReflectionsHandler(w http.ResponseWriter, r *http.Request) {
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
			err = repo.Approve(req.ID)
			log.Printf("[Reflections] Approved reflection %d\n", req.ID)
		} else {
			err = repo.Reject(req.ID)
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
func (srv *V2APIServer) ConflictsHandler(w http.ResponseWriter, r *http.Request) {
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

// ConflictResolveHandler handles POST /api/v2/conflicts/resolve
func (srv *V2APIServer) ConflictResolveHandler(w http.ResponseWriter, r *http.Request) {
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
	learningAgent, err := agents.NewLearningAgentWithDB(userID, srv.database)
	if err == nil && learningAgent != nil {
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

func (srv *V2APIServer) MetricsHandler(w http.ResponseWriter, r *http.Request) {
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
func (srv *V2APIServer) GetPreviousQuestionsHandler(w http.ResponseWriter, r *http.Request) {
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
func (srv *V2APIServer) QuestionEffectivenessHandler(w http.ResponseWriter, r *http.Request) {
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
		QuestionID            string `json:"questionId"`
		SocraticApproach      string `json:"socraticApproach"`
		QuestionText          string `json:"questionText"`
		UserResponse          string `json:"userResponse"`
		ReducedAmbiguity      bool   `json:"reducedAmbiguity"`
		InsightGained         string `json:"insightGained"`
		DepthLevelAdvanced    bool   `json:"depthLevelAdvanced"`
		PrincipleClarified    string `json:"principleClarified"`
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
func (srv *V2APIServer) AnalyzeConversationHandler(w http.ResponseWriter, r *http.Request) {
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
			Content:   content,
			Timestamp: createdAt,
		})
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
		"conversationId":     req.ConversationID,
		"aboutMeUpdates":     result.AboutMeUpdates,
		"patternDetections":  result.PatternDetections,
		"contactMentions":    result.ContactMentions,
		"goalProgressUpdates": result.GoalProgressUpdates,
		"confidence":         result.ConfidenceScore,
		"extractedAt":        result.ExtractedAt,
	})
}

// ReflectionApprovalHandler handles POST /api/v2/reflections/approve and /reject
func (srv *V2APIServer) ReflectionApprovalHandler(w http.ResponseWriter, r *http.Request) {
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
		err = reflectionRepo.Approve(int(req.ReflectionID))
	} else {
		err = reflectionRepo.Reject(int(req.ReflectionID))
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

// handleCheckSafety - Check message for safety issues (crisis/illegal language)
func handleCheckSafety(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	// Simple implementation: check for crisis keywords
	crisisKeywords := []string{"suicide", "kill myself", "self harm", "hurt myself"}
	alertType := "none"
	for _, keyword := range crisisKeywords {
		if strings.Contains(strings.ToLower(req.Message), keyword) {
			alertType = "crisis"
			break
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"alert_type": alertType,
		"severity":   "none",
		"message":    "Message safe",
	})
}

// handleEvaluateConstitution - Evaluate message against ethical principles
func handleEvaluateConstitution(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	if req.Message == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Message is required"})
		return
	}

	// Use Checker to evaluate message
	checker := safety.NewChecker()
	alert := checker.CheckMessage(req.Message)

	// Determine evaluation result
	var evaluation string
	var score float64
	var violations []string

	if alert != nil && alert.Severity != "" {
		evaluation = "flagged"
		score = 0.3 // Lower score for flagged content
		violations = []string{string(alert.AlertType)}
	} else {
		evaluation = "ethical"
		score = 0.95
		violations = []string{}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"evaluation": evaluation,
		"score":      score,
		"violations": violations,
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

// handleGenerateAuthCode - Generate authentication code
func handleGenerateAuthCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req struct {
		DeviceID string `json:"deviceId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	if req.DeviceID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "deviceId required"})
		return
	}

	// Generate 6-digit code
	code := fmt.Sprintf("%06d", rand.Intn(1000000))
	expiresAt := time.Now().Add(10 * time.Minute).Unix()

	// Store code in login_codes table
	conn := v2db.GetConnection()
	_, err := conn.Exec(
		"INSERT INTO login_codes (code, expires_at, device_id, created_at) VALUES (?, ?, ?, ?)",
		code, expiresAt, req.DeviceID, time.Now().Unix(),
	)
	if err != nil {
		log.Printf("[Auth] Failed to store code: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to generate code"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"code":      code,
		"expiresIn": 600,
	})
}

// handleCodeBasedLogin - Login with code (replaces email/password login)
func handleCodeBasedLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req struct {
		Code     string `json:"code"`
		DeviceID string `json:"deviceId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	if req.Code == "" || req.DeviceID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "code and deviceId required"})
		return
	}

	// Look up code in login_codes table (with expiration check in query)
	conn := v2db.GetConnection()
	var userID *string // Use pointer for nullable column

	err := conn.QueryRow(
		"SELECT user_id FROM login_codes WHERE code = ? AND used = 0 AND expires_at > ?",
		req.Code,
		time.Now().Unix(),
	).Scan(&userID)

	if err != nil {
		if err == sql.ErrNoRows {
			respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid or expired code"})
		} else {
			log.Printf("[Auth] Code lookup failed: %v", err)
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Login failed"})
		}
		return
	}

	// If no user_id in code (first-time login), create new user
	var finalUserID string
	if userID == nil {
		finalUserID = "user_" + fmt.Sprintf("%d", time.Now().UnixNano())
		log.Printf("[Auth] Creating new user: %s", finalUserID)
		_, err := conn.Exec(
			"INSERT INTO users (id, created_at, last_active) VALUES (?, ?, ?)",
			finalUserID, time.Now().Unix(), time.Now().Unix(),
		)
		if err != nil {
			log.Printf("[Auth] Failed to create user: %v", err)
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Login failed: " + err.Error()})
			return
		}
		// Update code with user_id
		_, err = conn.Exec(
			"UPDATE login_codes SET user_id = ? WHERE code = ?",
			finalUserID, req.Code,
		)
		if err != nil {
			log.Printf("[Auth] Failed to update code: %v", err)
		}
	} else {
		finalUserID = *userID
	}

	// Create session
	sessionID := "sess_" + fmt.Sprintf("%d", time.Now().UnixNano())
	sessionExpiresAt := time.Now().Add(30 * 24 * time.Hour).Unix() // 30 days
	now := time.Now().Unix()

	log.Printf("[Auth] Creating session for user: %s", finalUserID)
	_, err = conn.Exec(
		"INSERT INTO sessions (id, user_id, device_id, created_at, expires_at, last_used) VALUES (?, ?, ?, ?, ?, ?)",
		sessionID, finalUserID, req.DeviceID, now, sessionExpiresAt, now,
	)
	if err != nil {
		log.Printf("[Auth] Failed to create session: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Login failed: " + err.Error()})
		return
	}

	// Mark code as used
	_, err = conn.Exec("UPDATE login_codes SET used = 1 WHERE code = ?", req.Code)
	if err != nil {
		log.Printf("[Auth] Failed to mark code as used: %v", err)
	}

	log.Printf("[Auth] ✓ Login successful - sessionId: %s, userId: %s", sessionID, finalUserID)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"sessionId": sessionID,
		"userId":   finalUserID,
		"expiresIn": 2592000, // 30 days in seconds
	})
}

// handleValidateAuthCode - Validate current session token
func handleValidateAuthCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	// Extract Bearer token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Missing authorization header"})
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid authorization format"})
		return
	}

	// Look up session
	conn := v2db.GetConnection()
	var userID string
	var expiresAt int64

	err := conn.QueryRow(
		"SELECT user_id, expires_at FROM sessions WHERE id = ?",
		token,
	).Scan(&userID, &expiresAt)

	if err != nil {
		if err == sql.ErrNoRows {
			respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid token"})
		} else {
			log.Printf("[Auth] Session lookup failed: %v", err)
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Validation failed"})
		}
		return
	}

	// Check if session is expired
	if expiresAt < time.Now().Unix() {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Session expired"})
		return
	}

	// Update last_used timestamp
	_, err = conn.Exec(
		"UPDATE sessions SET last_used = ? WHERE id = ?",
		time.Now().Unix(), token,
	)
	if err != nil {
		log.Printf("[Auth] Failed to update session: %v", err)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"valid":   true,
		"userId":  userID,
		"expiresIn": expiresAt - time.Now().Unix(),
	})
}

func main() {
	// Initialize V2 database
	v2dbPath := filepath.Join(os.TempDir(), "moly-v2.db")
	var err error
	v2db, err = database.Init(v2dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize V2 database: %v", err)
	}
	defer v2db.Close()

	log.Println("[Moly] V2 Database initialized at", v2dbPath)

	// Initialize auth tables
	conn := v2db.GetConnection()
	if err := conn.Ping(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Create users table if it doesn't exist
	_, err = conn.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE,
			password_hash TEXT,
			created_at INTEGER NOT NULL,
			last_active INTEGER NOT NULL,
			updated_at INTEGER DEFAULT 0,
			context_level TEXT DEFAULT 'minimal',
			safety_tier TEXT DEFAULT 'standard'
		)
	`)
	if err != nil {
		log.Printf("[Moly] Warning creating users table: %v\n", err)
	}

	// Create sessions table if it doesn't exist
	_, err = conn.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			device_id TEXT,
			created_at INTEGER NOT NULL,
			expires_at INTEGER NOT NULL,
			last_used INTEGER NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		log.Printf("[Moly] Warning creating sessions table: %v\n", err)
	}

	// All tables (about_me, conversations, user_contacts, etc.) are created by schema.sql
	// No inline CREATE TABLE statements - schema.sql is the single source of truth
	log.Println("[Moly] Auth and context binding tables initialized (via schema.sql)")

	// Initialize LLM client (Ollama > Claude API > Fail)
	// Moly requires an LLM provider - no fallback
	var llmClient tools.LLMProvider
	llmClient, err = tools.NewLLMClient()
	if err != nil {
		log.Fatalf("\n%v\n", err)
	}
	log.Println("[Moly] LLM client initialized and ready")

	// Initialize V2 API Server with agents and orchestration
	v2Server, err = NewV2APIServer(llmClient, v2db)
	if err != nil {
		log.Fatalf("Failed to initialize V2 API server: %v", err)
	}
	log.Println("[Moly] V2 API Server initialized")

	// Auth API routes
	userAuthServer := auth.NewUserAuthServer(v2db.GetConnection())
	http.HandleFunc("/api/auth/register", userAuthServer.RegisterHandler)
	http.HandleFunc("/api/auth/login", userAuthServer.LoginHandler)
	http.HandleFunc("/api/auth/verify", userAuthServer.VerifyTokenHandler)
	http.HandleFunc("/api/auth/logout", userAuthServer.LogoutHandler)
	log.Println("[Moly] Auth API routes registered (Email/Password authentication)")

	// Phase 5: Full orchestration endpoints
	http.HandleFunc("/api/v2/message-processor", v2Server.MessageProcessorHandler)
	http.HandleFunc("/api/v2/clarification/respond", v2Server.ClarificationResponseHandler)
	http.HandleFunc("/api/v2/incoming-message/analyze", v2Server.AnalyzeIncomingMessageHandler)
	http.HandleFunc("/api/v2/suggestion/choice", v2Server.SuggestionChoiceHandler)
	log.Println("[Moly] Phase 5 API routes registered (full orchestration + clarification + suggestion tracking)")

	// Context binding endpoints
	http.HandleFunc("/api/v2/about-me", v2Server.AboutMeHandler)
	http.HandleFunc("/api/v2/context", v2Server.ContextHandler)
	http.HandleFunc("/api/v2/conversations", v2Server.ConversationsHandler)
	http.HandleFunc("/api/v2/contacts", v2Server.ContactsHandler)
	http.HandleFunc("/api/v2/messages", v2Server.MessagesHandler)
	http.HandleFunc("/api/v2/reflections", v2Server.ReflectionsHandler)
	http.HandleFunc("/api/v2/conflicts", v2Server.ConflictsHandler)
	http.HandleFunc("/api/v2/conflicts/resolve", v2Server.ConflictResolveHandler)
	http.HandleFunc("/api/v2/reflections/approval", v2Server.ReflectionApprovalHandler)
	http.HandleFunc("/api/v2/questions", v2Server.GetPreviousQuestionsHandler)
	http.HandleFunc("/api/v2/questions/effectiveness", v2Server.QuestionEffectivenessHandler)
	http.HandleFunc("/api/v2/conversations/analyze", v2Server.AnalyzeConversationHandler)
	http.HandleFunc("/api/v2/metrics", v2Server.MetricsHandler)
	log.Println("[Moly] Context binding API routes registered (about-me + conversations + contacts + metrics + analysis)")

	// Greenfield Pipeline Routes (new 4-stage architecture)
	http.HandleFunc("/api/v2/message-processor/pipeline", v2Server.MessageProcessorHandlerPipeline)
	http.HandleFunc("/api/v2/pipeline/health", v2Server.PipelineHealthCheckHandler)
	log.Println("[Moly] Greenfield pipeline routes registered (message-processor/pipeline + pipeline/health)")

	// Health check
	http.HandleFunc("/api/status", handleStatus(v2Server.database))

	// Safety & Ethics Endpoints
	http.HandleFunc("/api/check-safety", handleCheckSafety)
	http.HandleFunc("/api/evaluate-constitution", handleEvaluateConstitution)
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

	log.Println("[Moly] Server starting on http://localhost:8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// respondError - Helper to return error responses (used by legacy chat handlers)
func respondError(w http.ResponseWriter, statusCode int, message string) {
	respondJSON(w, statusCode, map[string]string{"error": message})
}

// getConfigPath - Helper to get config file path (used by legacy config handlers)
func getConfigPath() string {
	return filepath.Join(os.TempDir(), "moly-config.json")
}

// MessageProcessorHandlerPipeline - New handler using greenfield 4-stage pipeline
func (srv *V2APIServer) MessageProcessorHandlerPipeline(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

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

	// Validate request
	if err := schema.ValidateStruct(req); err != nil {
		schema.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if len(req.ConversationID) > 100 {
		schema.RespondError(w, http.StatusBadRequest, "ConversationID too long (max 100 characters)")
		return
	}

	// Use pipeline to process message
	log.Printf("[MessageProcessorPipeline] Processing message: user=%s conv=%s len=%d", userID, req.ConversationID, len(req.Message))

	response, err := srv.pipeline.ProcessMessage(userID, req.ConversationID, req.Message)
	if err != nil {
		log.Printf("[MessageProcessorPipeline] ERROR: %v", err)
		schema.RespondError(w, http.StatusInternalServerError, "Failed to process message")
		return
	}

	elapsed := time.Since(startTime)
	log.Printf("[MessageProcessorPipeline] ✓ Complete: %dms response=%d chars", elapsed.Milliseconds(), len(response.Response))

	// Return response
	schema.RespondSuccess(w, http.StatusOK, "response", response)
}

// PipelineHealthCheckHandler - Check if pipeline is ready
func (srv *V2APIServer) PipelineHealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	if srv.pipeline == nil {
		schema.RespondError(w, http.StatusServiceUnavailable, "Pipeline not initialized")
		return
	}

	healthData := map[string]string{
		"status": "ready",
		"provider": srv.llmProvider,
	}

	schema.RespondSuccess(w, http.StatusOK, "health", healthData)
}
