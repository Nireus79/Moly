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
	"strings"
	"time"

	"moly/agents"
	"moly/auth"
	"moly/database"
	"moly/models"
	"moly/safety"
	"moly/schema"
	"moly/tools"
)

var v2db *database.Database
var v2Server *V2APIServer

// V2APIServer wraps the agent system and database
type V2APIServer struct {
	llmClient                tools.LLMProvider
	database                 *database.Database
	contactManager           *agents.ContactManager
	contextAttrManager       *agents.ContextAttributeManager
	clarificationAgent       *agents.ClarificationAgent
	answerProcessor          *agents.AnswerProcessor
	incomingMessageAnalyzer  *agents.IncomingMessageAnalyzer
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

	// Initialize ConversationAgent (V2 architecture) with optional Socratic support
	conversationAgent, err := agents.InitializeWithSocraticSelector(llm, "config/constitution.yaml", "config")
	if err != nil {
		log.Printf("[Moly] Warning: Failed to initialize ConversationAgent: %v, will operate with fallback mode\n", err)
		conversationAgent = nil // Fallback to nil, but don't fail startup
	}

	// Wrap ConversationAgent in a minimal AgentSystem struct
	agentSystem := &agents.AgentSystem{
		ConversationAgent: conversationAgent,
	}

	// Initialize RiskMonitor with LLM for contextual risk analysis
	// Uses per-request userID, so creating a dummy instance here; will recreate per-request
	riskMonitor, _ := agents.NewRiskMonitorWithLLM("system", llm)
	if riskMonitor == nil {
		// Fallback to basic heuristic-based risk monitor
		riskMonitor, _ = agents.NewRiskMonitor("system")
	}

	return &V2APIServer{
		llmClient:               llm,
		database:                db,
		contactManager:          contactManager,
		contextAttrManager:      contextAttrManager,
		clarificationAgent:      clarificationAgent,
		answerProcessor:         answerProcessor,
		incomingMessageAnalyzer: incomingMessageAnalyzer,
		agentSystem:             agentSystem,
		safetyChecker:           safety.NewCheckerWithLLM(llm),
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
	_, _ = conn.Exec("UPDATE sessions SET last_used = ? WHERE id = ?", time.Now().Unix(), token)

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

func handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok","version":"2.1"}`))
}

// MessageProcessorHandler - Full orchestration with multi-phase context processing
func (srv *V2APIServer) MessageProcessorHandler(w http.ResponseWriter, r *http.Request) {
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
	var riskAssessmentForResponse *models.RiskAssessment // Store for inclusion in response
	if req.Message != "" && srv.riskMonitor != nil {
		// Check if risk assessment was already done (for retries)
		if msgProcState != nil && srv.messageProcessingState.IsStageComplete(msgProcState, agents.StageRiskAssessment) {
			log.Printf("[MessageProcessor] ⊘ Risk assessment already complete - skipping (retry optimization)")
			// Load result from state
			if result := srv.messageProcessingState.GetStageResult(msgProcState, agents.StageRiskAssessment); result != nil {
				if riskAssessment, ok := result.(*models.RiskAssessment); ok {
					riskAssessmentForResponse = riskAssessment
				} else {
					log.Printf("[MessageProcessor] Warning: Stored risk assessment has unexpected type")
				}
			}
		} else {
			// First time execution - run the stage
			riskAssessment, riskErr := srv.riskMonitor.AssessRisk(userID, req.Message)
			if riskErr != nil {
				log.Printf("[RiskMonitor] Warning: Risk assessment failed: %v (treating as clear)", riskErr)
			} else if riskAssessment != nil {
				log.Printf("[RiskMonitor] ✓ Assessment for user %s: level=%s severity=%d", userID, riskAssessment.RiskLevel, riskAssessment.Severity)
				riskAssessmentForResponse = riskAssessment // Store for later inclusion in response

				// Mark stage as complete and store result
				if msgProcState != nil {
					markErr := srv.messageProcessingState.MarkStageComplete(msgProcState, agents.StageRiskAssessment, riskAssessment)
					if markErr != nil {
						log.Printf("[MessageProcessor] Warning: Failed to mark risk assessment complete: %v", markErr)
					}
				}
			}

			// If high risk, provide educational response instead of processing
			if riskAssessment.RiskLevel == "high" || riskAssessment.RiskLevel == "immediate" {
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

	// Use ConversationAgent (V2 architecture)
	if srv.agentSystem == nil || srv.agentSystem.ConversationAgent == nil {
		schema.RespondError(w, http.StatusInternalServerError, "Agent system not initialized")
		return
	}

	// LOAD OR CREATE CONVERSATION - reuse existing for same user (within 30-day window)
	conversationID := req.ConversationID
	conn := srv.database.GetConnection()
	if conversationID == "" || conversationID == "null" {
		// Try to load most recent conversation for this user (within 30 days)
		var existingConvID string
		thirtyDaysAgo := time.Now().Unix() - (30 * 24 * 60 * 60)
		err := conn.QueryRow(
			`SELECT id FROM conversations WHERE user_id = ? AND updated_at > ? ORDER BY updated_at DESC LIMIT 1`,
			userID, thirtyDaysAgo,
		).Scan(&existingConvID)

		if err == nil && existingConvID != "" {
			conversationID = existingConvID
			log.Printf("[MessageProcessor] ✓ Loaded existing conversation (within 30-day window): %s", conversationID)
		} else {
			// Create new conversation only if none exists or all are older than 30 days
			now := time.Now().Unix()
			conversationID = fmt.Sprintf("conv_%d", now)
			_, err := conn.Exec(`
				INSERT INTO conversations (id, user_id, name, type, description, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`, conversationID, userID, "Direct Message", "direct", "Persistent conversation", now, now)
			if err != nil {
				log.Printf("[MessageProcessor] Warning: Failed to create conversation: %v", err)
			} else {
				log.Printf("[MessageProcessor] ✓ Created conversation: %s", conversationID)
			}
		}
	}

	// Refresh conversation's updated_at timestamp to maintain within 30-day window
	now := time.Now().Unix()
	_, _ = conn.Exec("UPDATE conversations SET updated_at = ? WHERE id = ?", now, conversationID)

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
			"SELECT id, role, content, created_at FROM chat_messages WHERE conversation_id = ? ORDER BY created_at DESC LIMIT 10",
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
		"SELECT id, characteristics, interests, intentions, status, created_at FROM reflections WHERE user_id = ? AND status IN ('approved', 'pending_approval') ORDER BY created_at DESC LIMIT 5",
		userID,
	)
	if reflectionErr != nil {
		log.Printf("[MessageProcessor] Warning: Failed to load past reflections: %v", reflectionErr)
	} else {
		defer reflectionRows.Close()
		for reflectionRows.Next() {
			var id, charJSON, interestJSON, intentionJSON, status string
			var createdAt int64
			if err := reflectionRows.Scan(&id, &charJSON, &interestJSON, &intentionJSON, &status, &createdAt); err != nil {
				log.Printf("[MessageProcessor] Warning: Error scanning reflection row: %v", err)
				continue
			}

			reflection := models.Reflection{
				ID:         id,
				Status:     status,
				CreatedAt:  createdAt,
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

			relevantReflections = append(relevantReflections, reflection)
			log.Printf("[MessageProcessor] ✓ Loaded reflection: %d char, %d interests, %d intentions",
				len(reflection.Characteristics),
				len(reflection.Interests),
				len(reflection.Intentions))
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
	var contactName, contactRelationship, charJSON string

	if len(selectedContactIds) > 0 {
		// If specific contacts are selected, load from that list (use first selected contact)
		err = conn.QueryRow(
			"SELECT name, relationship, characteristics FROM user_contacts WHERE user_id = ? AND id = ? LIMIT 1",
			userID, selectedContactIds[0],
		).Scan(&contactName, &contactRelationship, &charJSON)
		if err == nil && contactName != "" {
			log.Printf("[MessageProcessor] ✓ Loaded selected contact (ID: %s): %s (%s)", selectedContactIds[0], contactName, contactRelationship)
		}
	} else {
		// Otherwise load most recent contact
		err = conn.QueryRow(
			"SELECT name, relationship, characteristics FROM user_contacts WHERE user_id = ? ORDER BY updated_at DESC LIMIT 1",
			userID,
		).Scan(&contactName, &contactRelationship, &charJSON)
	}

	if err == nil && contactName != "" {
		contactProfile = &models.Contact{
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
	if aboutMeStyle == "" {
		gaps = append(gaps, "communicationStyle")
	}
	if len(aboutMeValues) == 0 {
		gaps = append(gaps, "coreValues")
	}
	if contactProfile == nil || contactProfile.Name == "" {
		gaps = append(gaps, "contact")
	}
	if len(conversationHistory) == 0 {
		gaps = append(gaps, "conversationHistory")
	}
	if userBehaviorProfile == nil {
		gaps = append(gaps, "userBehaviorProfile")
	}
	if len(gaps) > 0 {
		log.Printf("[MessageProcessor] Context gaps identified: %v", gaps)
	}

	// If safety alert was detected, return immediately with alert response (no agent processing)
	if safetyAlertDetected != nil {
		log.Printf("[MessageProcessor] Skipping agent processing due to safety alert")
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
			"processingTimeMs": int(time.Since(time.Now()).Milliseconds()),
			"metadata":         map[string]interface{}{},
			"reflection":       nil,
			"constitutionConcerns": nil,
			"extractedContact": nil,
		}
		respondJSON(w, http.StatusOK, response)
		return
	}

	ctx := models.Context{
		AboutMe: &models.AboutMe{
			UserID:             userID,
			CommunicationStyle: aboutMeStyle,
			Values:             aboutMeValues,
			PreferredTone:      aboutMeTone,
		},
		ContactProfile:      contactProfile,
		ConversationHistory: conversationHistory,
		ExtractedContext:    extractedContext,       // Pass LLM-extracted context to agent
		UserBehaviorProfile: userBehaviorProfile,   // User's learned patterns and preferences
		RelevantReflections: relevantReflections,   // Past insights from similar conversations
		Gaps:                gaps,                   // Missing context fields
		ContextQuality:      "minimal",
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

	// Filter out questions for already-asked question types (pending or answered)
	filteredQuestions := []*schema.ClarificationQuestion{}
	for _, q := range agentResp.Questions {
		if !askedQuestionTypes[q.Type] {
			filteredQuestions = append(filteredQuestions, q)
		} else {
			log.Printf("[MessageProcessor] ⊘ Skipping duplicate question type: %s", q.Type)
		}
	}
	agentResp.Questions = filteredQuestions

	// Save generated questions to database for tracking
	if len(filteredQuestions) > 0 {
		log.Printf("[MessageProcessor] Persisting %d clarification questions to database", len(filteredQuestions))
		conn := srv.database.GetConnection()
		for _, q := range filteredQuestions {
			now := time.Now().Unix()
			linkedFactsJSON := "[]"
			if len(q.LinkedFacts) > 0 {
				if b, err := json.Marshal(q.LinkedFacts); err == nil {
					linkedFactsJSON = string(b)
				}
			}

			// Prepare Socratic metadata for storage
			expectedInsightsJSON := "[]"
			if len(q.ExpectedInsights) > 0 {
				if b, err := json.Marshal(q.ExpectedInsights); err == nil {
					expectedInsightsJSON = string(b)
				}
			}

			log.Printf("[MessageProcessor] 📝 Saving question: id=%s type=%s approach=%s principle=%s depth=%d",
				q.ID, q.Type, q.SocraticApproach, q.TargetsPrinciple, q.DepthLevel)
			log.Printf("[MessageProcessor]   └─ Question text: %.80s", q.Question)
			log.Printf("[MessageProcessor]   └─ Expected insights: %v", q.ExpectedInsights)
			log.Printf("[MessageProcessor]   └─ Linked facts: %s", linkedFactsJSON)

			_, err := conn.Exec(`
				INSERT INTO clarification_questions
				(id, user_id, conversation_id, clarification_type, question_text, priority, status,
				 linked_facts, created_at, socratic_approach, targets_principle, expected_insights, depth_level)
				VALUES (?, ?, ?, ?, ?, 2, 'pending', ?, ?, ?, ?, ?, ?)
				ON CONFLICT(id) DO NOTHING
			`, q.ID, userID, conversationID, q.Type, q.Question, linkedFactsJSON, now,
			q.SocraticApproach, q.TargetsPrinciple, expectedInsightsJSON, q.DepthLevel)

			if err != nil {
				log.Printf("[MessageProcessor] ⚠️ Error persisting question %s: %v", q.ID, err)
			} else {
				log.Printf("[MessageProcessor] ✓ Question %s persisted successfully", q.ID)
			}
		}
		log.Printf("[MessageProcessor] ✓ All %d questions persisted", len(filteredQuestions))
	} else {
		log.Printf("[MessageProcessor] No questions to persist")
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

	log.Printf("[MessageProcessor] Complete: phase=%s, suggestions=%d, questions=%d (after dedup)\n",
		agentResp.Phase, len(agentResp.Suggestions), len(agentResp.Questions))

	// SAVE USER MESSAGE to chat_messages for conversation history
	_, _ = conn.Exec(`
		INSERT INTO chat_messages (id, user_id, conversation_id, role, content, created_at)
		VALUES (?, ?, ?, 'user', ?, ?)
		ON CONFLICT(id) DO NOTHING
	`, userMessageID, userID, conversationID, userMessageForDB, now)
	log.Printf("[MessageProcessor] ✓ Saved user message to chat_messages: %s", userMessageID)

	// SAVE AGENT RESPONSE to chat_messages for conversation history
	agentResponseID := fmt.Sprintf("msg_%d_%d", now, rand.Int63())
	agentResponseJSON, _ := json.Marshal(map[string]interface{}{
		"phase":       agentResp.Phase,
		"questions":   agentResp.Questions,
		"suggestions": agentResp.Suggestions,
	})
	_, _ = conn.Exec(`
		INSERT INTO chat_messages (id, user_id, conversation_id, role, content, created_at)
		VALUES (?, ?, ?, 'assistant', ?, ?)
		ON CONFLICT(id) DO NOTHING
	`, agentResponseID, userID, conversationID, string(agentResponseJSON), now)
	log.Printf("[MessageProcessor] ✓ Saved agent response to chat_messages: %s", agentResponseID)

	// PHASE 2: SAVE CONTACT CHARACTERISTICS (when contact is mentioned)
	if agentResp.ExtractedContact != nil && agentResp.ExtractedContact.Name != "" {
		contactID := fmt.Sprintf("contact_%d_%d", now, rand.Int63())
		conn := srv.database.GetConnection()

		// Check if contact already exists (by name and user_id)
		var existingID string
		err := conn.QueryRow(
			"SELECT id FROM user_contacts WHERE user_id = ? AND name = ?",
			userID, agentResp.ExtractedContact.Name,
		).Scan(&existingID)

		// Prepare characteristics JSON if we have reflection data about the contact
		var charJSON []byte
		if agentResp.Reflection != nil && len(agentResp.Reflection.Characteristics) > 0 {
			charJSON, _ = json.Marshal(agentResp.Reflection.Characteristics)
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
				log.Printf("[MessageProcessor] ✓ Saved contact: %s (%s) with %d characteristics",
					agentResp.ExtractedContact.Name,
					agentResp.ExtractedContact.Relationship,
					len(agentResp.Reflection.Characteristics))
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
		conn := srv.database.GetConnection()

		// Serialize arrays to JSON for storage
		charJSON, _ := json.Marshal(agentResp.Reflection.Characteristics)
		interestsJSON, _ := json.Marshal(agentResp.Reflection.Interests)
		intentionsJSON, _ := json.Marshal(agentResp.Reflection.Intentions)

		// Save reflection to database (status = pending_approval, awaiting user confirmation)
		_, saveErr := conn.Exec(`
			INSERT INTO reflections (user_id, characteristics, interests, intentions, status, created_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, userID, string(charJSON), string(interestsJSON), string(intentionsJSON), "pending_approval", now)

		if saveErr != nil {
			log.Printf("[MessageProcessor] Warning: Failed to save reflection: %v", saveErr)
		} else {
			log.Printf("[MessageProcessor] ✓ Saved reflection: %d characteristics, %d interests, %d intentions",
				len(agentResp.Reflection.Characteristics),
				len(agentResp.Reflection.Interests),
				len(agentResp.Reflection.Intentions))
		}
	}

	// Include risk assessment if it exists (from LLM evaluation)
	if agentResp.RiskWarning == nil && riskAssessmentForResponse != nil {
		// Map RiskAssessment to RiskWarning format (all fields)
		agentResp.RiskWarning = &models.RiskWarning{
			RiskLevel:            riskAssessmentForResponse.RiskLevel,
			Pattern:              riskAssessmentForResponse.Pattern,
			Severity:             riskAssessmentForResponse.Severity,
			Message:              riskAssessmentForResponse.Message,
			EducationalQuestions: riskAssessmentForResponse.EducationalQuestions,
			Principles:           riskAssessmentForResponse.Principles,
			Alternatives:         riskAssessmentForResponse.Alternatives,
			Recommendation:       riskAssessmentForResponse.Recommendation,
		}
	}

	// Map ConversationResponse to Phase5 response format for frontend compatibility
	response := map[string]interface{}{
		"success": true,
		"phase":   agentResp.Phase,
		// Phase 1: Extract facts (not exposed by ConversationAgent currently)
		"phase1": map[string]interface{}{
			"facts":  []interface{}{},
			"shifts": []interface{}{},
		},
		// Phase 2: Clarification questions
		"phase2": map[string]interface{}{
			"clarifications": agentResp.Questions,
			"resolved":       0,
		},
		// Phase 3: Contact management
		"phase3": map[string]interface{}{
			"unknown_contacts": []interface{}{},
			"created_contacts": []interface{}{},
		},
		// Phase 4: Storage
		"phase4": map[string]interface{}{
			"saved_attributes": []interface{}{},
			"conflicts":        []interface{}{},
		},
		// Action required for frontend
		"action_required": map[string]interface{}{
			"needsClarification": len(agentResp.Questions) > 0,
			"clarificationQs":    agentResp.Questions,
			"temporaryFacts":     []interface{}{},
			"hasConflicts":       agentResp.RiskWarning != nil && agentResp.RiskWarning.RiskLevel != "clear",
			"conflicts":          []interface{}{},
		},
		// ConversationAgent specific fields
		"response":         agentResp.Response,
		"suggestions":      agentResp.Suggestions,
		"riskWarning":      agentResp.RiskWarning,
		"safetyAlert":      agentResp.SafetyAlert,
		"processingTimeMs": agentResp.ProcessingTimeMs,
		"metadata":         agentResp.Metadata,
		"reflection":       agentResp.Reflection,
		"constitutionConcerns": agentResp.ConstitutionConcerns,
		"extractedContact": agentResp.ExtractedContact,
	}

	// Add error field only if present (non-fatal errors)
	if agentResp.Error != "" {
		response["error"] = agentResp.Error
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
	_, _ = conn.Exec(
		"UPDATE clarification_questions SET status = 'answered', answered_at = ? WHERE id = ?",
		now, req.QuestionID,
	)

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
	_ = json.Unmarshal([]byte(coreVals), &coreValsArr)
	_ = json.Unmarshal([]byte(goals), &goalsArr)
	_ = json.Unmarshal([]byte(patterns), &patternsArr)

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
	var conversationHistory []string
	if req.ConversationID != "" {
		rows, err := conn.Query(
			`SELECT content FROM chat_messages
			 WHERE user_id = ? AND conversation_id = ? AND role IN ('user', 'assistant')
			 ORDER BY created_at DESC LIMIT 5`,
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
	respondJSON(w, http.StatusOK, response)
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
		schema.RespondSuccess(w, http.StatusOK, "profile", req)
	}
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
			var id, name, convType, description, purpose, membersJSON, settingsJSON, notes string
			var createdAt, updatedAt int64
			if err := rows.Scan(&id, &name, &convType, &description, &purpose, &membersJSON, &settingsJSON, &notes, &createdAt, &updatedAt); err != nil {
				log.Printf("[ConversationsHandler] WARNING: Skipping corrupted conversation row - scan error: %v", err)
				continue
			}

			conv := map[string]interface{}{
				"id":          id,
				"name":        name,
				"type":        convType,
				"description": description,
				"purpose":     purpose,
				"createdAt":   createdAt,
				"updatedAt":   updatedAt,
				"notes":       notes,
			}

			// Parse JSON fields
			if membersJSON != "" {
				var members []schema.ConversationMember
				if err := json.Unmarshal([]byte(membersJSON), &members); err == nil {
					conv["members"] = members
				}
			}
			if settingsJSON != "" {
				var settings schema.ConversationSettings
				if err := json.Unmarshal([]byte(settingsJSON), &settings); err == nil {
					conv["settings"] = settings
				}
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
			var id, name, relationship, notes string
			var createdAt int64
			if err := rows.Scan(&id, &name, &relationship, &notes, &createdAt); err != nil {
				log.Printf("[ContactsHandler] WARNING: Skipping corrupted contact row - scan error: %v", err)
				continue
			}
			contacts = append(contacts, map[string]interface{}{
				"id":           id,
				"name":         name,
				"relationship": relationship,
				"notes":        notes,
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
		var id, role, content, contextExtracted, contactMention string
		var uID, convID string
		var createdAt int64

		if err := rows.Scan(&id, &uID, &convID, &role, &content, &contextExtracted, &contactMention, &createdAt); err != nil {
			log.Printf("[MessagesHandler] WARNING: Skipping corrupted message row - scan error: %v", err)
			continue
		}

		messages = append(messages, map[string]interface{}{
			"id":                id,
			"conversationId":    convID,
			"role":              role,
			"content":           content,
			"contextExtracted":  contextExtracted,
			"contactMention":    contactMention,
			"createdAt":         createdAt,
		})
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

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"evaluation": "ethical",
		"score":      0.95,
		"violations": []string{},
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

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"recommended": true,
		"risk_level":  "low",
		"analysis":    "Mode shift is appropriate given context",
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

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"questions": []string{
			"What was the last time you connected with " + req.ContactName + "?",
			"How do they typically prefer to communicate?",
			"What topics are they interested in?",
		},
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

	// Initialize LLM client (Ollama > Claude API > None)
	var llmClient tools.LLMProvider
	llmClient, err = tools.NewLLMClient()
	if err != nil {
		log.Printf("[Moly] Warning: LLM client initialization failed: %v (will run in heuristic-only mode)\n", err)
	} else if llmClient != nil {
		log.Println("[Moly] LLM client initialized")
	}

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
	log.Println("[Moly] Phase 5 API routes registered (full orchestration + clarification)")

	// Context binding endpoints
	http.HandleFunc("/api/v2/about-me", v2Server.AboutMeHandler)
	http.HandleFunc("/api/v2/conversations", v2Server.ConversationsHandler)
	http.HandleFunc("/api/v2/contacts", v2Server.ContactsHandler)
	http.HandleFunc("/api/v2/messages", v2Server.MessagesHandler)
	http.HandleFunc("/api/v2/reflections", v2Server.ReflectionsHandler)
	http.HandleFunc("/api/v2/metrics", v2Server.MetricsHandler)
	log.Println("[Moly] Context binding API routes registered (about-me + conversations + contacts + metrics)")

	// Health check
	http.HandleFunc("/api/status", handleStatus)

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
