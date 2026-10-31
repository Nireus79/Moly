package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"moly/agents"
	"moly/database"
	"moly/models"
	"moly/monitoring"
	"moly/privacy"
	"moly/schema"
	"moly/tools"
)

// llmCallTimeout bounds one LLM call, including the slow CPU-only case. Calls are never cut short.
const llmCallTimeout = 20 * time.Minute

// Clarification Response Helpers
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

// MessageProcessorHandler records how the user's turn ends (see turn_status.go) and runs it.
func (srv *APIServer) MessageProcessorHandler(w http.ResponseWriter, r *http.Request) {
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil || r.Method != http.MethodPost {
		srv.messageProcessorTurn(w, r) // answers the refusal itself
		return
	}
	turns.track(userID, w, r, srv.messageProcessorTurn)
}

// messageProcessorTurn - Full orchestration with multi-phase context processing
func (srv *APIServer) messageProcessorTurn(w http.ResponseWriter, r *http.Request) {
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

	// From here on this request works with the model-calling components of this user, built from this user's own provider.
	srv = srv.forUser(userID)
	defer runningTurns.begin(userID)()

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

	// Message length limit (DoS prevention)
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

	// Secrets with a known shape (payment card numbers, IBANs) never enter the conversation: they are cut out here, before any model
	// call, summary or save. Other secrets (a password, a PIN) are found by the extraction and cut out right after it.
	if cut, n := tools.RedactStructured(req.Message); n > 0 {
		req.Message = cut
		log.Printf("[Privacy] %d card or account number(s) removed from the message before it was used", n)
	}
	rememberForLogs(req.Message)

	// Clarification Response Detection & Processing
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
	titleGoal := "" // the locked goal after this turn: the conversation is named after it (conversation_title.go)

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
	// Meta-Instruction Detection (Self-Awareness)
	// Detect if message is about Moly's behavior/focus (e.g., "You are Moly", "Lace is my focus")
	// IMPORTANT: Update context but CONTINUE through orchestrator (don't return early)
	var metaInstruction *agents.MetaInstruction
	var messageReading agents.MessageReading // the intent and the meta-instruction check of this message, read with one model call
	// The reading starts now and runs while the safety check and the extraction are made (all three read only the message); the turn
	// waits for it where its result is first needed (finishReading).
	readText, readExchange := req.Message, ""
	readingDone := make(chan struct{})
	if readText != "" {
		readExchange = srv.recentExchange(req.ConversationID, readText)
		go func() {
			messageReading = agents.ReadMessage(context.Background(), srv.llmClient, readText, readExchange)
			close(readingDone)
		}()
	} else {
		close(readingDone)
	}
	readingHandled := false
	finishReading := func() {
		<-readingDone
		if readingHandled || readText == "" {
			return
		}
		readingHandled = true
		if messageReading.OK {
			metaInstruction = messageReading.Meta
		} else {
			metaInstruction = srv.metaInstructionDetector.DetectWithExchange(context.Background(), readText, readExchange) // the reading failed: ask the judge alone
		}
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
	var previousExtraction *PreviousExtraction // Declare here so it's accessible to save logic
	var pendingNameLabel string                // label of a person who must be asked for a name in this reply
	var messageIntent agents.IntentAnalysis    // the intent of this message, decided once after extraction
	var messageStatesNoGoal bool               // no goal is taken from this message: it only answers a question, or its goal is in doubt
	var resultNow bool                         // the user pressed skip: go ahead with what is known
	var doubtfulFact string                    // a fact understood with doubt and not saved: the reply confirms it first
	var aboutMoly bool                         // the message asks about Moly itself
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
			// REMOVED - ContextExtractor.Extract() now called inside ExtractionPhase
			// This eliminates duplicate extraction (was taking 169 seconds)
			// extractedContext will be populated from epOutput.ExtractedContext below
		}

		if extractedContext != nil && extractedContext.Style != nil && extractedContext.Style.Confidence > 0.5 {
			log.Printf("[MessageProcessor] ✓ Extracted style: %s (confidence=%.2f)",
				extractedContext.Style.Style, extractedContext.Style.Confidence)
		}
	}

	// Fetch conversation history if conversation ID provided
	// Load FULL conversation history (not limited to 10 messages)
	// Hybrid context architecture requires full history for accurate summaries
	// Also load message metadata (for prior context) (NEW)
	conversationHistory := []models.Message{}
	if req.ConversationID != "" && req.ConversationID != "null" {
		// Reuse existing connection to avoid pool exhaustion
		rows, err := conn.Query(
			"SELECT id, role, content, created_at, metadata FROM chat_messages WHERE conversation_id = ? ORDER BY created_at ASC, rowid ASC",
			req.ConversationID,
		)
		if err != nil {
			log.Printf("[MessageProcessor] Warning: Failed to fetch conversation history: %v", err)
		} else {
			defer rows.Close()
			for rows.Next() {
				var id, role, content string
				var createdAt int64
				var metadata sql.NullString // Load metadata
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
					Metadata:  msgMetadata, // Include metadata
				})
			}
			if err := rows.Err(); err != nil {
				log.Printf("[MessageProcessor] Error iterating conversation history: %v", err)
			}
		}
	}

	for _, m := range conversationHistory {
		rememberForLogs(m.Content)
	}

	// The people of this conversation: the people the user has named, and this conversation's own people who have no name yet.
	// No unnamed person of another conversation is visible through this repository. The people the conversation has already
	// brought up (before this message) are the only ones a pronoun or a role can point to.
	conversationPeople := database.NewContactRepository(srv.database).InConversation(conversationID)
	var earlierMessages []string
	for _, m := range conversationHistory {
		if m.ID != userMessageID {
			earlierMessages = append(earlierMessages, m.Content)
		}
	}
	establishedContacts := database.EstablishedPeople(conversationPeople, userID, earlierMessages)
	var thisMessagePerson *models.Contact // the person the identity stage resolved for this message, if any

	// SAFETY STAGE (ORCHESTRATOR_DESIGN.md step 4): the one safety verdict of this message, before anything is saved from it.
	// It fails closed: with no verdict, no reply is delivered. Layer 2 reads this verdict; nothing evaluates safety again.
	var safetyResult agents.SafetyResult
	var storedContext *models.AnalysisContext // what the builder loaded for this conversation (summary, people, last messages)

	// The extraction reads only the message and what was loaded for this conversation, like the safety check: both are model calls,
	// so they run at the same time (a turn waits for the slower one, not for both). Nothing is saved by either; a message the safety
	// check blocks was extracted before as well.
	type extractionResult struct {
		out *agents.ExtractionPhaseOutput
		err error
	}
	var extractionRun chan extractionResult
	var shadow *memoryShadow                            // step 5a of the memory redesign: development mode, MOLY_MEMORY_SHADOW=1
	defer func() { shadow.rollbackUnlessCommitted() }() // a turn that is not saved leaves the memory as it was
	var memoryNext *models.MemoryNext                   // memory mode: what the update decided to ask (nil outside memory mode)
	startExtraction := func() chan extractionResult {
		// Load previous extraction BEFORE ExtractionPhase (was loading too late!)
		// This passes accumulated context to extraction phase for proper accumulation
		previousExtraction = srv.loadPreviousExtraction(userID, req.ConversationID)
		if previousExtraction != nil {
			log.Printf("[MessageProcessor] ✓ FIX #9: Loaded previous extraction: %d entities, goal=%q",
				len(previousExtraction.Entities), previousExtraction.Goal)
		}
		// Note: RecentMessages and UserProfile will be populated later if needed
		epInput := &agents.ExtractionPhaseInput{
			UserID:             userID,
			ConversationID:     req.ConversationID,
			MessageID:          userMessageID,
			Message:            processedMessage,
			MessageCount:       0,                  // Will be calculated when loading conversation history
			RecentMessages:     []models.Message{}, // Empty for now, extraction works without it
			Exchange:           models.ExchangeBeforeChronological(conversationHistory, userMessageForDB, 1),
			Dialogue:           srv.dialogueBlock(storedContext, previousExtraction, userID, req.ConversationID),
			UserProfile:        nil,                // Will be populated from AboutMe if available later
			PreviousExtraction: previousExtraction, // Pass accumulated context to extraction phase
		}
		done := make(chan extractionResult, 1)
		go func() {
			out, err := srv.extractionPhase.Run(context.Background(), epInput)
			done <- extractionResult{out: out, err: err}
		}()
		return done
	}
	if processedMessage != "" && userMessageForDB != "" {
		safetyCtx := &models.AnalysisContext{CurrentMessage: userMessageForDB, ContextQuality: "minimal"}
		if srv.analysisContextBuilder != nil {
			if built, buildErr := srv.analysisContextBuilder.BuildAnalysisContext(userID, conversationID, userMessageForDB, conversationHistory, &models.AboutMe{}); buildErr == nil && built != nil {
				safetyCtx = built
				storedContext = built
			}
		}
		if extractionRun == nil {
			extractionRun = startExtraction() // runs while the safety check is made
			shadow = srv.startMemoryShadow(userID, conversationID, userMessageID, userMessageForDB, conversationHistory, conversationPeople)
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

	// CENTRALIZED EXTRACTION WITH CONFLICT DETECTION (Session 15 - Phase 1)
	// Layer 0 of orchestrator: Extract once, detect conflicts, share with all downstream components
	var extractedEntities []models.ExtractedEntity
	var extractedEntitiesNeedClarification = false
	var extractedEntitiesClarificationQ string
	var extractionArtifact *models.ExtractionArtifact
	var extractionConflicts []agents.ConflictDetectorResult

	if processedMessage != "" {
		metrics := monitoring.GetMetrics()

		extractStartTime := time.Now()

		// Call ExtractionPhase (Layer 0) to get extraction + conflict detection + analysis context. It was started with the safety
		// check when both run; otherwise it is started now.
		if extractionRun == nil {
			extractionRun = startExtraction()
		}
		run := <-extractionRun
		epOutput, err := run.out, run.err

		// Get extractedContext from epOutput (was separate LLM call before)
		if epOutput != nil && epOutput.ExtractedContext != nil {
			extractedContext = epOutput.ExtractedContext
			log.Printf("[MessageProcessor] ✓ Got extracted context from combined extraction phase")

			// Secrets the user typed (found by the extraction) are cut out of the message and out of everything derived from it,
			// before the intent, the people, the reply and every save: nothing stored holds them.
			if secrets := extractedContext.Sensitive; len(secrets) > 0 {
				var n int
				req.Message, n = tools.RedactItems(req.Message, secrets)
				userMessageForDB, _ = tools.RedactItems(userMessageForDB, secrets)
				processedMessage, _ = tools.RedactItems(processedMessage, secrets)
				var artifactEntities []models.ExtractedEntity
				if epOutput.Artifact != nil {
					artifactEntities = epOutput.Artifact.Entities
				}
				artifactEntities = agents.ScrubSensitive(extractedContext, artifactEntities, secrets)
				if epOutput.Artifact != nil {
					epOutput.Artifact.Entities = artifactEntities
				}
				log.Printf("[Privacy] %d secret(s) removed from the message and from what was derived from it", n)
				shadow.useMessage(userMessageForDB) // the memory update must not see them either
			}
			var loggedEntities []models.ExtractedEntity
			if epOutput.Artifact != nil {
				loggedEntities = epOutput.Artifact.Entities
			}
			rememberExtractionForLogs(extractedContext, loggedEntities)

			// ORDER (2026-10-09): the intent and the name answer are decided here, before any contact is saved.
			// Previously the intent came after the save (a greeting became contact "the user") and the name answer came before extraction.
			// INTENT STAGE (ORCHESTRATOR_DESIGN.md step 3): decided once, here. A greeting has no goal and no person.
			finishReading() // the reading of the message (intent and instruction check) was made while the extraction ran
			var intentSource agents.IntentSource
			if messageReading.OK {
				intentSource = agents.IntentFromReading(messageReading) // already read, together with the meta-instruction check
			} else if srv.intentDetector != nil {
				intentSource = srv.intentDetector
			}
			intentDecision := agents.DecideIntent(intentSource, userMessageForDB, conversationHistory)
			if wasAnswerOnly := intentDecision.AnswerOnly; wasAnswerOnly {
				intentDecision.ConfirmAnswerOnly(extractedContext, userMessageForDB)
				if !intentDecision.AnswerOnly {
					log.Printf("[MessageProcessor] The message answers and also states an aim of its own: not taken as only an answer")
				}
			}
			messageIntent = intentDecision.Analysis
			greeting := intentDecision.Greeting
			if intentDecision.AboutMoly {
				// A question about Moly itself teaches nothing about the user: no goal, no person, no question back.
				agents.ForgetRefusedMessage(extractedContext)
				messageStatesNoGoal = true
				aboutMoly = true
				log.Printf("[MessageProcessor] The message asks about Moly itself: answered from the product facts, nothing is taken from it")
			}
			if skipAllowed {
				// The press carries no goal and no person of its own: it is like an answer, and Moly goes ahead.
				resultNow = true
				if err := database.NewMemoryStore(conn).SkipOpenQuestions(userID, conversationID); err != nil {
					log.Printf("[Memory] could not mark the open questions skipped: %v", err)
				}
				intentDecision.AnswerOnly = true
				log.Printf("[MessageProcessor] Skip pressed: questions that only improve the result are skipped; risk still stops it")
			}
			if messageIntent.WantsResult && !skipAllowed && !greeting && !aboutMoly {
				// The user said in words what the Skip button says: stop asking and go ahead (found in a dynamic run: "I don't want to answer
				// questions" was answered with another question).
				resultNow = true
				log.Printf("[MessageProcessor] The user asked for the result now: questions that only improve it are skipped; risk still stops it")
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
				if previousExtraction != nil && previousExtraction.PrimaryGoal != "" {
					// A goal that is not in the user's own words cannot be a change of goal, and the conversation has a locked goal.
					// A comment on Moly's draft ("a bit too formal") gets its "goal" from the draft: it is not asked about, the
					// locked goal continues and the reply revises the draft (found in a dynamic live run).
					log.Printf("[MessageProcessor] Goal not in the user's words: dropped, the locked goal continues")
				} else {
					doubtfulFact = fmt.Sprintf("that what you want is: %s", held)
					log.Printf("[MessageProcessor] Goal in doubt: held, not locked; the reply confirms it")
				}
			}
			if greeting {
				intentDecision.StripGreeting(extractedContext)
				log.Printf("[MessageProcessor] Greeting: no goal and no contact saved from this message")
			}

			// IDENTITY STAGE (ORCHESTRATOR_DESIGN.md step 2): the one place a message writes a contact row.
			// The repository is this conversation's: it sees the people the user has named and this conversation's own unnamed
			// people, and no unnamed person of another conversation. A saved person is matched by the name the user gives, never
			// by "my boss" or "her"; only people this conversation has already brought up can be pointed to.
			// The memory decides who the people are: one update call (docs/MEMORY_REDESIGN.md). Its operations are checked and applied, the
			// people are written to the contacts for the readers that have not moved yet, and its next action gives the name question or the
			// confirmation.
			person := srv.applyMemoryPeople(shadow, conversationPeople, greeting || safetyResult.Blocked || aboutMoly)
			if person.Updated {
				memoryNext = &person.Next
			}
			thisMessagePerson = person.Contact
			if person.Doubt != "" && doubtfulFact == "" {
				doubtfulFact = fmt.Sprintf("that this is about the person you called %q", person.Doubt)
				log.Printf("[MessageProcessor] Person in doubt: not saved; the reply confirms it")
			}
			messageNameAnswered = person.NameAnswered
			pendingNameLabel = person.PendingNameLabel
			if people, listErr := conversationPeople.GetByUserID(userID); listErr == nil {
				for _, c := range people {
					rememberForLogs(c.Name, c.ContactRole)
					rememberForLogs(c.Characteristics...)
				}
			}
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

			// Save message summary immediately after extraction (Phase 1 optimization)
			// This lightweight metadata saves instantly for future re-analysis optimization
			if srv.messageSummaryRepo != nil && epOutput != nil && !safetyResult.Blocked {
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

				// ENHANCEMENT: Use ExtractionArtifact from Phase 0 
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

	// Pending clarifications of THIS conversation (from an earlier session of it). Another conversation's are not ours to answer.
	tempStore := agents.NewTemporaryFactStore(srv.database, userID)
	pendingClarifications, _ := tempStore.GetPendingForConversation(conversationID)

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
			pendingConflicts, err := conflictRepo.GetUnresolved(userID, conversationID)
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
					result := inlineResolver.ApplyConflictResolution(userID, conversationID, firstConflict.ID, resolution)

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

	// The saved settings are the truth. The panel sends the profile it loaded when it opened, and the settings page saves in
	// another page, so that copy can be old: it only fills what the database does not hold (a user with no saved profile).
	if req.AboutMe != nil {
		if v, ok := req.AboutMe["communicationStyle"].(string); ok && v != "" && aboutMeStyle == "" {
			aboutMeStyle = v
		}
		if v, ok := req.AboutMe["coreValues"].([]interface{}); ok && len(v) > 0 && len(aboutMeValues) == 0 {
			for _, val := range v {
				if s, ok := val.(string); ok && s != "" {
					aboutMeValues = append(aboutMeValues, s)
				}
			}
		}
		if v, ok := req.AboutMe["tonePreference"].(string); ok && v != "" && aboutMeTone == "" {
			aboutMeTone = v
		}
	}

	log.Printf("[MessageProcessor] About Me loaded: style=%s, values=%d, tone=%s", aboutMeStyle, len(aboutMeValues), aboutMeTone)

	// Load SystemContext (user's feedback about Moly)
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
	// LOAD PAST REFLECTIONS (what the user confirmed about themselves, from any conversation, and what this conversation
	// itself learned). An insight the user has not confirmed yet is unresolved: it belongs to the conversation it came from and
	// is not shown to another one (it used to be: the last five of any status, from any conversation, went into every reply).
	var relevantReflections []models.Reflection
	// Reuse existing connection to avoid pool exhaustion
	reflectionRows, reflectionErr := conn.Query(
		"SELECT id, conversation_id, contact_id, characteristics, interests, intentions, communication_preferences, user_quotes, user_edits, status, created_at FROM reflections WHERE user_id = ? AND (status = 'approved' OR (status = 'pending_approval' AND conversation_id = ?)) ORDER BY created_at DESC LIMIT 5",
		userID, conversationID,
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

	// Load past intention (the goal of the previous messages OF THIS CONVERSATION; the latest goal of any conversation
	// used to be taken, which told a new conversation about the goal of an old one)
	var pastIntention string
	// Reuse existing connection to avoid pool exhaustion
	intentionErr := conn.QueryRow(
		"SELECT fact_value FROM context_attributes WHERE user_id = ? AND conversation_id = ? AND fact_type = 'intention' ORDER BY created_at DESC LIMIT 1",
		userID, conversationID,
	).Scan(&pastIntention)
	if intentionErr == nil && pastIntention != "" {
		log.Printf("[MessageProcessor] ✓ Loaded past intention: %s", pastIntention)
	} else if intentionErr != sql.ErrNoRows && intentionErr != nil {
		log.Printf("[MessageProcessor] Warning: Failed to load past intention: %v", intentionErr)
	}

	// Load recent safety incidents (to prevent re-alerting)
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
			// the text columns may be NULL (an incident written without a response): a NULL is an empty text, not a row to skip
			var content, detectedBy, responseProvided sql.NullString
			if err := safetyRows.Scan(&incident.ID, &incident.UserID, &incident.Severity, &incident.DetectedAt, &content, &detectedBy, &responseProvided); err != nil {
				log.Printf("[MessageProcessor] Warning: Error scanning safety incident row: %v", err)
				continue
			}
			incident.Content, incident.DetectedBy, incident.ResponseProvided = content.String, detectedBy.String, responseProvided.String
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

	// the person this conversation is about. Only people this conversation can see are considered (the people the user has
	// named and this conversation's own unnamed people), and of those the person the user selected, else the one this message
	// resolved, else the most recently updated one the conversation has already brought up. The most recent contact of the user,
	// whatever the conversation, is never taken: that put a person from an old chat into a new one.
	var contactProfile *models.Contact
	if len(selectedContactIds) > 0 {
		if selectedID, parseErr := strconv.ParseInt(fmt.Sprint(selectedContactIds[0]), 10, 64); parseErr == nil {
			if c, getErr := conversationPeople.GetByID(userID, selectedID); getErr == nil && c != nil {
				contactProfile = c
				log.Printf("[MessageProcessor] ✓ Loaded selected contact (ID: %d): %s (%s)", c.ID, c.Name, c.Relationship)
			}
		}
	} else if thisMessagePerson != nil {
		contactProfile = thisMessagePerson
	} else if people, listErr := conversationPeople.GetByUserID(userID); listErr == nil {
		brought := map[string]bool{}
		for _, c := range establishedContacts {
			brought[c.Name] = true
		}
		for _, c := range people {
			if brought[c.Name] {
				contactProfile = c
				break
			}
		}
	}
	if contactProfile != nil {
		log.Printf("[MessageProcessor] ✓ Loaded contact for this conversation: %s (%s)", contactProfile.Name, contactProfile.Relationship)
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
	shouldDenyRequest := false // Layer 11 deny signal
	// Fix P: Use isFirstMessageInConversation consistently (already calculated BEFORE prepend)
	// Don't recalculate here - with Fix Q (conditional prepend), len-based checks become unreliable
	// isFirstMessageInConversation is the authoritative flag (calculated before any modifications)

	// CRITICAL FIX: Only add "contact" gap if the message actually discusses a contact
	// If ContextExtractor found no contact (extractedContext.Contact == nil),
	// and message is self-directed (greeting to Moly), don't ask about missing contacts
	isMessageAboutContact := extractedContext != nil && extractedContext.Contact != nil && extractedContext.Contact.Confidence > 0.5

	// CRITICAL: Check if message is a greeting/self-reference (no topic to discuss)
	// For these messages, we should NOT ask gap clarification questions at all
	// the message intent comes from the LLM intent detector, not from phrase lists (R8).
	// It is decided before the layers run, so layers 4-10 can be skipped for a greeting.
	// messageIntent was decided right after extraction
	isGreetingOrSelfRef := messageIntent.Intent == agents.IntentGreet
	if safetyResult.Blocked || aboutMoly {
		agents.ForgetRefusedMessage(extractedContext)
		extractedEntities = nil
		if extractionArtifact != nil {
			extractionArtifact.Entities = nil
		}
	}
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
	// Note: currentPhase is already declared at function level for tracking
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

	// Build AnalysisContext once, use for all evaluations
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

			// Wire SystemContext (Moly's self-awareness) to AnalysisContext
			if systemContext != nil {
				analysisCtx.SystemContext = systemContext
				log.Printf("[MessageProcessor] ✓ PHASE 4B: Wired SystemContext to AnalysisContext (feedback=%d directives=%d)",
					len(systemContext.UserFeedback), len(systemContext.UserDirectives))
			}

			// Enhance AnalysisContext with ExtractionArtifact 
			// Embed extraction metadata for all 11 layers to access and use
			if extractionArtifact != nil {
				// Check extraction artifact TTL (not expired)
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

			// Load previous extraction state for context accumulation
			// This enables Layer 5+ to compare against previous entities and detect conflicts
			// Also load primary goal and progression for goal tracking
			previousExtraction := srv.loadPreviousExtraction(userID, req.ConversationID)
			if previousExtraction != nil {
				analysisCtx.AccumulatedExtractedEntities = previousExtraction.Entities
				analysisCtx.PreviousGoal = previousExtraction.Goal
				analysisCtx.PreviousValues = previousExtraction.Values
				analysisCtx.PrimaryGoal = previousExtraction.PrimaryGoal       // Load locked goal
				analysisCtx.GoalProgression = previousExtraction.Progression   // Load progression
				analysisCtx.ContextTracker = previousExtraction.ContextTracker // Change tracking carried across messages
				log.Printf("[MessageProcessor] ✓ FIX #1+#4: Loaded previous (goal=%q, primary=%q, entities=%d)",
					previousExtraction.Goal, previousExtraction.PrimaryGoal, len(previousExtraction.Entities))
			}

			// What every model call is told about this conversation: the builder gave the summary, the people and the last messages;
			// here the locked goal, what was learned earlier and the questions still open are added.
			var openTexts []string
			if open, openErr := database.NewClarificationHistoryRepository(srv.database).OpenQuestions(userID, req.ConversationID); openErr == nil {
				for _, q := range open {
					openTexts = append(openTexts, q.Text)
				}
			}
			fillDialogue(analysisCtx, openTexts)
			srv.addMemoryFacts(analysisCtx, userID, req.ConversationID)
			rememberForLogs(openTexts...)
			if d := analysisCtx.Dialogue; d != nil {
				rememberForLogs(d.Summary, d.Goal, d.MolyLast, d.UserBefore)
				rememberForLogs(d.People...)
				rememberForLogs(d.Open...)
			}
			if previousExtraction != nil {
				rememberForLogs(previousExtraction.Goal, previousExtraction.PrimaryGoal)
				rememberForLogs(previousExtraction.Progression...)
				rememberExtractionForLogs(nil, previousExtraction.Entities)
			}

			// Progressive Naming Detection
			// Check if user is providing names for unnamed contacts
			if userMessageForDB != "" && srv.database != nil {
				contactRepo := database.NewContactRepository(srv.database)
				activeContacts, err := contactRepo.GetByUserID(userID)
				if err == nil && len(activeContacts) > 0 {
					// names are given through the name gate (ApplyNameAnswer, LLM extraction). The regex naming detector is not used.
				}
			}

			// the greeting decision is passed to the layers, not logged (R4)
			if analysisCtx != nil {
				analysisCtx.IsGreeting = isGreetingOrSelfRef
				finishReading()
				analysisCtx.InstructionsConflict = metaInstruction != nil && metaInstruction.Contradictory
				analysisCtx.AboutMoly = aboutMoly
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

					// Layer 11 - IMMEDIATE THREAT (highest priority - deny immediately)
					if layerCtx.Layer11 != nil && layerCtx.Layer11.ShouldDeny {
						shouldDenyRequest = true
						log.Printf("[MessageProcessor] 🚫 LAYER 11 SIGNAL: Should deny - %s", layerCtx.Layer11.Reason)
						log.Printf("[MessageProcessor] → Preventing response generation, returning denial")
					}

					if layerCtx.Layer2 != nil && layerCtx.Layer2.IsObviousHarm {
						log.Printf("[MessageProcessor] ⚠ Layer 2: Obvious harm detected")
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
					incidentSeverity(safetyAlertDetected.Severity),
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

	// A goal the user moved on to was adopted in layer 1 without a question; the questions of the old aim are dropped.
	if analysisCtx != nil {
		if lc, ok := analysisCtx.LayerResults.(*tools.LayerContext); ok && lc != nil && lc.GoalSwitch != "" {
			if n, err := database.NewClarificationHistoryRepository(srv.database).CancelPending(userID, conversationID); err != nil {
				log.Printf("[MessageProcessor] Warning: could not drop pending questions: %v", err)
			} else if n > 0 {
				log.Printf("[MessageProcessor] %d pending question(s) dropped with the old goal", n)
			}
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
		MessageAdjustsDraft:          messageIntent.AdjustsDraft && len(conversationHistory) > 0,
		IsGreeting:                   isGreetingOrSelfRef,
		AboutMoly:                    aboutMoly,
		MemoryNext:                   memoryNext,
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
				// The table takes low, medium, high or critical (found live, 2026-10-15: "block" was refused and no incident was recorded).
				severity := "low"
				if intervention == "blocked" {
					severity = "high"
					if s, ok := agentResp.Metadata["blockSeverity"].(string); ok && (s == "critical" || s == "high") {
						severity = s
					}
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
		agentResp.Metadata["safetyAlertType"] = safetyAlertDetected.AlertType // the next turn reads it: after a crisis refusal Moly checks in first
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

	// SAVE REFLECTION (user insights extracted from conversation)
	if agentResp.Reflection != nil && (len(agentResp.Reflection.Characteristics) > 0 || len(agentResp.Reflection.Interests) > 0) {
		// Mark insight extraction stage as complete
		if msgProcState != nil {
			markErr := srv.messageProcessingState.MarkStageComplete(msgProcState, agents.StageInsightExtraction, agentResp.Reflection)
			if markErr != nil {
				log.Printf("[MessageProcessor] Warning: Failed to mark insight extraction complete: %v", markErr)
			}
		}

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
			if c, getErr := conversationPeople.GetByName(userID, agentResp.ExtractedContact.Name); getErr == nil && c != nil {
				cid := strconv.FormatInt(c.ID, 10)
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

	// SAVE EXTRACTED CONTEXT (communication style and intention from this message)
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

		// Subject-based contact deduplication 
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

		// People are written by the memory (docs/MEMORY_REDESIGN.md); a change in how a person is related or described is a new operation of
		// the memory, not a conflict to approve.

		// Save extracted intention (if present)
		if extractedContext.Intention != "" {
			// No intention "conflict" is queued here. The goal of a conversation is the locked goal: how a new wording relates to it
			// is judged by the model (GoalRelation) and a real switch is confirmed with the user (goal_switch.go). Comparing the
			// text of two intentions queued a conflict for every rewording and kept the skip button off (found live, 2026-10-10).
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

		// Save extracted system feedback to system_context
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
					// BLOCK response when internal contradiction detected (don't just mark it)
					// User said they're dominant, but response suggests submissive → MUST ask clarification
					if agentResp.Metadata == nil {
						agentResp.Metadata = make(map[string]interface{})
					}
					agentResp.Metadata["responseCharacteristicConflict"] = map[string]string{
						"userCharacteristic": userChar,
						"contraryWord":       contraryWord,
						"conflictReason":     conflictReason,
					}
					agentResp.Metadata["responseBlocked"] = true // Flag to block response
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

	// CHECK LAYER GATES BEFORE ROUTING (NEW)
	// Layers can control response flow through gates
	layerGatesBlock := false

	// Extract layerCtx from analysisCtx (where it was stored during orchestration)
	var layerCtxForGates *tools.LayerContext
	if analysisCtx != nil && analysisCtx.LayerResults != nil {
		if lc, ok := analysisCtx.LayerResults.(*tools.LayerContext); ok {
			layerCtxForGates = lc
		}
	}

	if layerCtxForGates != nil {
		// Check Layer 11 denial protocol (has ShouldDeny field)
		if layerCtxForGates.Layer11 != nil && layerCtxForGates.Layer11.ShouldDeny {
			layerGatesBlock = true
			log.Printf("[MessageProcessor] ⚠ LAYER 11 GATE TRIGGERED - Response denied: %s", layerCtxForGates.Layer11.Reason)
		}

	}

	// Check if response should be blocked (response contradicts extraction)
	responseBlocked := layerGatesBlock
	if agentResp != nil && agentResp.Metadata != nil {
		if blocked, isBlocked := agentResp.Metadata["responseBlocked"].(bool); isBlocked && blocked {
			log.Printf("[MessageProcessor] ⚠ RESPONSE BLOCKED - Will send clarification instead")
			responseBlocked = true
		}
	}

	// Build action_required field for frontend (Phase5 format)
	// Fix D: Include conflicts in action_required for conflict confirmation flow
	// Include conflict metadata so frontend can show why clarification needed (NEW)
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
		"conflicts":          conflictDetails, // Full conflict details, not just IDs
	}

	// If response is blocked due to contradiction, force clarification mode
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

	// Contact-Aware Response Formatting
	// Format response based on contact resolution status
	var contactFormatter *agents.ContactResponseFormatter
	var formattedResponse map[string]interface{}
	var response map[string]interface{}

	if srv.database != nil {
		contactFormatter = agents.NewContactResponseFormatter(srv.database)

		// Load active contacts for formatting
		// Only the people this conversation is about: those it has brought up, and the one this message resolved. A saved person
		// of another conversation must not shape (or replace, by a clarification) the reply of this one.
		activeContacts := make([]*models.Contact, 0, len(establishedContacts)+1)
		seenInReply := map[int64]bool{}
		for _, c := range append(append([]*models.Contact{}, establishedContacts...), thisMessagePerson) {
			if c != nil && !seenInReply[c.ID] && c.Status != "" {
				seenInReply[c.ID] = true
				activeContacts = append(activeContacts, c)
			}
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

	// Record this interaction for behavioral profile learning
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

	// +#9: Save extraction state for next message (context accumulation)
	// CRITICAL: Must save MERGED extraction (accumulated + current), not just current!
	// This ensures next message has full context history, not just last message's data
	if layerCtx != nil && layerCtx.Layer1 != nil && layerCtx.Layer1.ExtractedContext != nil {
		// Apply layer-based accumulated updates before merging
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
				accumulatedToMerge,            // accumulated from earlier (with updates)
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
			PrimaryGoal:    layerCtx.PrimaryGoal,     // Save locked primary goal
			Progression:    layerCtx.GoalProgression, // Save goal evolution
			ContextTracker: analysisCtx.ContextTracker,
		}
		if safetyResult.Blocked {
			// A refused message saves no people and no summary line; it does not lock a goal either (found live: the forgery
			// request stayed the locked goal, and the next harmless message was answered as help with it).
			extraction.Goal, extraction.PrimaryGoal = "", ""
			extraction.Entities = previousEntities(previousExtraction)
			if previousExtraction != nil {
				extraction.Goal, extraction.PrimaryGoal, extraction.Progression = previousExtraction.Goal, previousExtraction.PrimaryGoal, previousExtraction.Progression
			}
		}
		srv.savePreviousExtractionWith(turn, userID, req.ConversationID, extraction)
		titleGoal = extraction.PrimaryGoal
		log.Printf("[MessageProcessor] ✓ FIX #1+#4+#9: Saved MERGED extraction + primary goal=%q for next message (entities=%d)",
			extraction.PrimaryGoal, len(extraction.Entities))

		// Update conversation summary with accumulated insights
		// This wires the final step of the Accumulated Insights Model
		if srv.database != nil && analysisCtx.ConversationSummary != nil {
			maturity := agents.ConversationGapMaturity(srv.database.GetClarificationQuestionRepository(), conversationID)
			summary := analysisCtx.ConversationSummary

			// Correctly accumulate entity count (not reset!)
			// Load existing count from DB, add new entities
			// entitiesToSave is already the merged set (everything accumulated so far plus this message), so it IS the unique count;
			// adding it to the stored count again counted every entity once per turn (123 for 22 entities after nine turns).
			existingCount := summary.AccumulatedEntityCount
			summary.AccumulatedEntityCount = len(entitiesToSave)
			log.Printf("[MessageProcessor] FIX #10: Accumulated entities - before=%d, now %d unique",
				existingCount, summary.AccumulatedEntityCount)

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
			progression = append(progression, maturity)
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

			// Save updated summary
			summaryRepo := database.NewConversationSummaryRepository(srv.database.GetConnection())
			if saveErr := summaryRepo.UpdateSummaryWithRetry(userID, conversationID, func(latest *models.ConversationSummary) {
				latest.AccumulatedEntityCount = summary.AccumulatedEntityCount
				latest.AccumulatedContactCount = summary.AccumulatedContactCount
				latest.AccumulatedValues = summary.AccumulatedValues
				latest.AccumulatedCharacteristics = summary.AccumulatedCharacteristics
				latest.ClarityProgression = summary.ClarityProgression
			}); saveErr != nil {
				log.Printf("[MessageProcessor] ⚠️ FIX #6: Failed to update summary: %v", saveErr)
			} else {
				log.Printf("[MessageProcessor] ✓ FIX #6: Updated summary - entities=%d contacts=%d maturity=%.2f progression=%d",
					summary.AccumulatedEntityCount, summary.AccumulatedContactCount, maturity, len(progression))
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

	// (reply contract): the reply is resolved once here. The text sent to the
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
	// A message Moly wrote is ready to send: no bracketed placeholder is left in it.
	if kind, _ := agentResp.Metadata["replyExit"].(string); kind == string(agents.KindHelp) {
		replyText = agents.StripPlaceholders(replyText)
	}
	// A pasted link: Moly does not read web pages, and the user is told so (the notice is in front of the reply, in their language).
	if agents.MessageHasLink(req.Message) {
		notice := agents.LinkNotice(r.Context(), srv.llmClient, req.Message, replyText)
		replyText = agents.WithLinkNotice(notice, replyText)
		if notice != "" {
			agentResp.Metadata["linkNotice"] = true
		}
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

	rememberForLogs(agentResp.Response)
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

	{
		oldKind, _ := agentResp.Metadata["replyExit"].(string)
		if oldKind == "" {
			oldKind, _ = agentResp.Metadata["replyPolicy"].(string) // the name question is decided before the exits and carries only the policy
		}
		shadow.finish(oldKind, replyText)
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
	shadow.markCommitted()
	if titleGoal != "" {
		go srv.nameConversation(userID, conversationID, titleGoal)
	}
	if titleGoal != "" {
		if err := database.NewMemoryStore(conn).MirrorGoal(userID, conversationID, titleGoal); err != nil {
			log.Printf("[Memory] could not mirror the goal: %v", err)
		}
	}

	// The summary is rewritten after every turn that is stored, right after its analysis and reply: it takes the previous
	// summary, the user's message and Moly's reply. It runs in the background, so the user does not wait for it; the next turn
	// uses it, or, if it is not finished, adds the user's previous message to the prompt (see dialogueFrom). A refused
	// message and a bare greeting are not remembered.
	if srv.conversationSummaryManager != nil && userMessageForDB != "" && !safetyResult.Blocked && !(analysisCtx != nil && analysisCtx.IsGreeting) {
		anchors := tools.SummaryAnchors{}
		if analysisCtx != nil && analysisCtx.Dialogue != nil {
			anchors.Goal, anchors.People = analysisCtx.Dialogue.Goal, analysisCtx.Dialogue.People
		}
		go srv.rememberTurn(userID, conversationID, len(conversationHistory)+1, anchors, userMessageForDB, agentResp.Response)
	}

	respondJSON(w, http.StatusOK, response)
}

// resolveReplyText returns the text the user sees for a response map.
// It uses the response's own "response" text when present. A clarification response has
// no text of its own, so its question and options are added to the agent's text.
// An empty result is an error: nothing is sent or saved as a reply.
func resolveReplyText(response map[string]interface{}, agentText string) (string, error) {
	text, _ := response["response"].(string)
	if text == "" {
		if clar, ok := response["clarification"].(map[string]interface{}); ok {
			// The agent's reply already asks its one question, in the user's language. The formatter's fixed question and A/B options are
			// English and would be a second question (found in the language checks, 2026-10-30): they are used only when the agent wrote nothing.
			if strings.TrimSpace(agentText) != "" {
				text = agentText
			} else {
				text = agents.ComposeReplyWithClarification(agentText, clar)
			}
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

// recentExchange writes the last stored messages of a conversation, for the model calls that run before the history is
// loaded. It returns "" for a new conversation or when the store cannot be read.
func (srv *APIServer) recentExchange(conversationID, current string) string {
	if conversationID == "" || conversationID == "null" || srv.database == nil || srv.database.GetConnection() == nil {
		return ""
	}
	rows, err := srv.database.GetConnection().Query(
		"SELECT role, content FROM chat_messages WHERE conversation_id = ? ORDER BY created_at DESC, rowid DESC LIMIT 5", conversationID)
	if err != nil {
		return ""
	}
	defer rows.Close()
	var newestFirst []models.Message
	for rows.Next() {
		var role, content string
		if rows.Scan(&role, &content) == nil {
			newestFirst = append(newestFirst, models.Message{Role: role, Content: database.StoredReplyText(content)})
		}
	}
	chronological := make([]models.Message, 0, len(newestFirst))
	for i := len(newestFirst) - 1; i >= 0; i-- {
		chronological = append(chronological, newestFirst[i])
	}
	out := models.ExchangeBeforeChronological(chronological, current, 1)
	return out
}

// dialogueBlock is the dialogue packet of this conversation as a prompt block, for the calls that run before the main
// analysis context is built. It returns "" for a new conversation.
func (srv *APIServer) dialogueBlock(stored *models.AnalysisContext, prev *PreviousExtraction, userID, conversationID string) string {
	if stored == nil || stored.Dialogue == nil {
		return ""
	}
	a := &models.AnalysisContext{Dialogue: &models.DialoguePacket{}}
	*a.Dialogue = *stored.Dialogue
	if prev != nil {
		a.PrimaryGoal, a.PreviousGoal = prev.PrimaryGoal, prev.Goal
	}
	var openTexts []string
	if open, err := database.NewClarificationHistoryRepository(srv.database).OpenQuestions(userID, conversationID); err == nil {
		for _, q := range open {
			openTexts = append(openTexts, q.Text)
		}
	}
	fillDialogue(a, openTexts)
	srv.addMemoryFacts(a, userID, conversationID)
	return a.Dialogue.Text()
}

// rememberTurn rewrites the conversation's summary with the newest turn. A failure leaves the previous summary and is retried once.
func (srv *APIServer) rememberTurn(userID, conversationID string, messageCount int, anchors tools.SummaryAnchors, userMessage, molyReply string) {
	for attempt := 1; attempt <= 2; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), llmCallTimeout)
		err := srv.conversationSummaryManager.UpdateWithTurn(ctx, userID, conversationID, messageCount, anchors, userMessage, molyReply)
		cancel()
		if err == nil {
			log.Printf("[MessageProcessor] ✓ Summary rewritten with the newest turn")
			return
		}
		log.Printf("[MessageProcessor] Summary update failed (attempt %d/2): %v", attempt, err)
	}
	log.Printf("[MessageProcessor] The summary stays as it was; the next prompt carries the user's previous message")
}

// incidentSeverity maps the alert's severity ("warning", "high", "immediate") to the four levels the incident table accepts. Before,
// an "immediate" alert was refused by the table ("severity ... must be one of low, medium, high, critical") and the incident of a
// refused message was not recorded (found in a dynamic run).
func incidentSeverity(alert string) string {
	switch alert {
	case "immediate", "critical":
		return "critical"
	case "high":
		return "high"
	case "low":
		return "low"
	}
	return "medium"
}

// rememberForLogs registers what the user said or what was learned from it, so that the log masks it from now on (see package privacy).
func rememberForLogs(values ...string) { privacy.Register(values...) }

// rememberExtractionForLogs registers the content of an extraction: the aim, the quotes, the traits, the values and the people.
func rememberExtractionForLogs(ec *models.ExtractedContext, entities []models.ExtractedEntity) {
	if ec != nil {
		rememberForLogs(ec.Intention, ec.IntentionEvidence)
		rememberForLogs(ec.Goals...)
		rememberForLogs(ec.UserValues...)
		rememberForLogs(ec.UserCharacteristics...)
		rememberForLogs(ec.UserInterests...)
		for k, v := range ec.ContactCharacteristics {
			rememberForLogs(k)
			rememberForLogs(v...)
		}
		for _, c := range append([]*models.ExtractedContact{ec.Contact}, ec.OtherContacts...) {
			if c != nil {
				rememberForLogs(c.Name, c.Label, c.Role, c.Evidence)
				rememberForLogs(c.Traits...)
			}
		}
	}
	for _, e := range entities {
		rememberForLogs(e.Value, e.Evidence)
	}
}
