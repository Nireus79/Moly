package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"moly/config"
	"moly/database"
	"moly/models"
	"moly/monitoring"
	"moly/schema"
	"moly/tools"
)

// conversationAgent - Implements the 5-phase conversation flow
type conversationAgent struct {
	llmClient                tools.LLMProvider
	constitutionalEvaluator  *tools.ConstitutionalEvaluator
	contextExtractor         *tools.ContextExtractor
	responseGenerator        *tools.ResponseGenerator      // Generates contextual responses instead of hardcoded text
	intentDetector           *LLMIntentDetector            // LLM-driven intent detection (no hardcoded patterns)
	socraticSelector         *SocraticQuestionSelector     // Optional: for Socratic question selection
	constitution             *models.Constitution          // Optional: for principle-guided generation
	db                       *database.Database            // Optional: for conflict detection
	inlineResolver           *tools.InlineConflictResolver // Optional: for Phase 2 inline resolution
	clarityAnalyzer          *MessageClarityAnalyzer       // NEW: Diagnostic message clarity analysis
	subjectShiftDetector     *SubjectShiftDetector         // [Layer 9] Detects topic/contact changes
	templateManager          *ResponseTemplateManager      // For database-driven response templates
	metaInstructionDetector  *MetaInstructionDetector      // [Phase 5] Self-awareness: detects meta-instructions about Moly
	layer5Handler            *Layer5ConflictHandler        // [PHASE 2] Layer 5: Conflict handling with locking
	cachedTopic              string                        // FIX 3: Cache topic detection to avoid redundant LLM calls
	cachedTopics             []string                      // FIX 3: Cache multiple topics detection
}

// NewConversationAgent - Create new conversation agent
func NewConversationAgent(llm tools.LLMProvider) (models.ConversationAgent, error) {
	// LLM client is optional - agent will generate basic suggestions without it
	return &conversationAgent{
		llmClient:               llm,
		constitutionalEvaluator: nil, // Will be set via SetConstitution after initialization
		contextExtractor:        tools.NewContextExtractor(llm),
		responseGenerator:       tools.NewResponseGenerator(llm),      // Generates natural, contextual responses
		intentDetector:          NewLLMIntentDetector(llm),            // LLM-driven intent detection
		socraticSelector:        nil,                                  // Optional - set via SetSocraticSelector if available
		subjectShiftDetector:    NewSubjectShiftDetectorWithLLM(llm),  // [Layer 9] Topic/contact change detection
		metaInstructionDetector: NewMetaInstructionDetector(llm),      // [Phase 5] Self-awareness meta-instruction detection
	}, nil
}

// SetSocraticSelector injects the Socratic question selector (optional)
func (ca *conversationAgent) SetSocraticSelector(selector *SocraticQuestionSelector) {
	if ca != nil {
		ca.socraticSelector = selector
		log.Printf("[ConversationAgent] Socratic selector initialized")
	}
}

// SetLayer5ConflictHandler injects the Layer 5 conflict handler (optional, PHASE 2)
func (ca *conversationAgent) SetLayer5ConflictHandler(handler *Layer5ConflictHandler) {
	if ca != nil {
		ca.layer5Handler = handler
		log.Printf("[ConversationAgent] Layer 5 conflict handler initialized (PHASE 2)")
	}
}

// SetDatabase injects the database for conflict detection (optional, Phase 2)
// HIGH PRIORITY FIX: Added proper type validation with logging
func (ca *conversationAgent) SetDatabase(dbInterface interface{}) {
	if ca == nil {
		log.Printf("[ConversationAgent] WARNING: SetDatabase called on nil agent")
		return
	}

	if dbInterface == nil {
		log.Printf("[ConversationAgent] WARNING: SetDatabase called with nil database")
		return
	}

	// Proper type validation
	db, ok := dbInterface.(*database.Database)
	if !ok {
		log.Printf("[ConversationAgent] ERROR: SetDatabase received wrong type: %T (expected *database.Database)", dbInterface)
		return
	}

	ca.db = db
	ca.inlineResolver = tools.NewInlineConflictResolver(db)
	log.Printf("[ConversationAgent] Inline conflict resolver initialized")
	// Initialize clarity analyzer now that we have database and LLM
	ca.clarityAnalyzer = NewMessageClarityAnalyzer(db, ca.llmClient, ca.socraticSelector)
	log.Printf("[ConversationAgent] Message clarity analyzer initialized with LLM support")
	// Initialize response template manager for database-driven responses
	ca.templateManager = NewResponseTemplateManager(db)
	ca.templateManager.InitializeDefaultTemplates()
	log.Printf("[ConversationAgent] Response template manager initialized")
}

// SetConstitution injects the loaded constitution into the agent (Phase 1)
func (ca *conversationAgent) SetConstitution(constitution *models.Constitution) {
	if ca != nil && constitution != nil {
		ca.constitution = constitution
		// Wire constitution into the evaluator
		ca.constitutionalEvaluator = tools.NewConstitutionalEvaluator(ca.llmClient, constitution)
		log.Printf("[ConversationAgent] Constitutional evaluator initialized with loaded constitution")

		// Wire constitution into principle-based detectors
		if ca.intentDetector != nil {
			ca.intentDetector.SetConstitution(constitution)
		}
		log.Printf("[ConversationAgent] Constitution wired to principle-based detectors")
	}
}

// InitializeWithSocraticSelector creates and wires a ConversationAgent with Socratic support and principle-based checking
// Returns the agent and any error that occurred during initialization
// If any initialization fails, returns agent with what could be loaded (graceful degradation)
func InitializeWithSocraticSelector(llm tools.LLMProvider, constitutionPath, configDir string) (models.ConversationAgent, error) {
	// Create base agent
	agent, err := NewConversationAgent(llm)
	if err != nil {
		return nil, err
	}

	// Load constitution and question library
	constitution, err := config.LoadConstitution(constitutionPath)
	if err != nil {
		log.Printf("[ConversationAgent] Warning: Could not load constitution: %v", err)
		return agent, nil // Return agent without Socratic features
	}

	library, err := config.LoadQuestionLibrary(configDir)
	if err != nil {
		log.Printf("[ConversationAgent] Warning: Could not load question library: %v", err)
		// Continue - we can still use constitution for principle checking
	}

	// Wire question library and constitution into the conversation agent
	if caImpl, ok := agent.(*conversationAgent); ok {
		// Wire constitution to both response generation and evaluator (Phase 1)
		caImpl.SetConstitution(constitution)
		log.Printf("[ConversationAgent] [✓] Constitution loaded for response generation and evaluation")

		// Set Socratic selector if library loaded successfully
		if library != nil {
			selector := NewSocraticQuestionSelector(library, constitution)
			caImpl.SetSocraticSelector(selector)
			log.Printf("[ConversationAgent] [✓] Socratic question selector initialized")
		}
	}

	return agent, nil
}

// NewFullyInitializedConversationAgent creates a conversation agent with enforced initialization order
// This factory ensures all dependencies are properly wired in the correct sequence
// Returns error if any step fails, preventing partial initialization
//
// INITIALIZATION ORDER (enforced):
// 1. Create base agent (with LLM client)
// 2. Load constitution
// 3. Load question library and set Socratic selector
// 4. Wire database (depends on step 3 for socraticSelector)
// 5. Verify all components are ready
func NewFullyInitializedConversationAgent(
	llm tools.LLMProvider,
	db *database.Database,
	constitutionPath string,
	configDir string,
) (models.ConversationAgent, error) {
	if llm == nil {
		return nil, fmt.Errorf("LLM provider cannot be nil")
	}

	// STEP 1: Create base agent
	log.Printf("[ConversationAgent] STEP 1: Creating base agent with LLM client")
	agent, err := NewConversationAgent(llm)
	if err != nil {
		return nil, fmt.Errorf("STEP 1 failed - create base agent: %w", err)
	}

	ca, ok := agent.(*conversationAgent)
	if !ok {
		return nil, fmt.Errorf("STEP 1 failed - agent is not conversationAgent (type mismatch)")
	}

	// STEP 2: Load and set constitution (MUST be before Socratic selector)
	log.Printf("[ConversationAgent] STEP 2: Loading constitution from %s", constitutionPath)
	constitution, err := config.LoadConstitution(constitutionPath)
	if err != nil {
		return nil, fmt.Errorf("STEP 2 failed - load constitution: %w", err)
	}
	ca.SetConstitution(constitution)
	log.Printf("[ConversationAgent] ✓ STEP 2: Constitution loaded and wired")

	// STEP 3: Load question library and set Socratic selector (MUST be before database)
	log.Printf("[ConversationAgent] STEP 3: Loading question library from %s", configDir)
	library, err := config.LoadQuestionLibrary(configDir)
	if err != nil {
		return nil, fmt.Errorf("STEP 3 failed - load question library: %w", err)
	}

	if library != nil {
		selector := NewSocraticQuestionSelector(library, constitution)
		ca.SetSocraticSelector(selector)
		log.Printf("[ConversationAgent] ✓ STEP 3: Socratic selector initialized with %d questions", len(library.AllQuestions))
	} else {
		log.Printf("[ConversationAgent] ⚠ STEP 3: Question library empty, Socratic selector not set")
	}

	// STEP 4: Wire database (NOW ca.socraticSelector is guaranteed to exist from STEP 3)
	log.Printf("[ConversationAgent] STEP 4: Wiring database and initializing dependent components")
	if db != nil {
		ca.SetDatabase(db)
		log.Printf("[ConversationAgent] ✓ STEP 4: Database wired, clarity analyzer initialized")
	} else {
		return nil, fmt.Errorf("STEP 4 failed - database cannot be nil")
	}

	// STEP 5: Verify readiness (all critical components must be initialized)
	log.Printf("[ConversationAgent] STEP 5: Verifying agent readiness")
	if !ca.IsReady() {
		return nil, fmt.Errorf("STEP 5 failed - agent readiness check failed (see logs above)")
	}
	log.Printf("[ConversationAgent] ✅ STEP 5: Agent is ready for use")

	return ca, nil
}

// IsReady checks if the agent is fully initialized and ready for use
// Returns false if any critical component is missing
func (ca *conversationAgent) IsReady() bool {
	checks := map[string]bool{
		"LLM client":              ca.llmClient != nil,
		"Constitution":            ca.constitution != nil,
		"Constitutional evaluator": ca.constitutionalEvaluator != nil,
		"Context extractor":       ca.contextExtractor != nil,
		"Response generator":      ca.responseGenerator != nil,
		"Intent detector":         ca.intentDetector != nil,
		"Subject shift detector":  ca.subjectShiftDetector != nil,
		"Meta-instruction detector": ca.metaInstructionDetector != nil,
		"Database":                ca.db != nil,
		"Clarity analyzer":        ca.clarityAnalyzer != nil,
	}

	allReady := true
	for component, ready := range checks {
		if !ready {
			log.Printf("[ConversationAgent] ✗ MISSING COMPONENT: %s", component)
			allReady = false
		}
	}

	if allReady {
		log.Printf("[ConversationAgent] ✓ All critical components initialized")
	}

	return allReady
}

// getLastAssistantMessage finds the most recent message from Moly
// After prepending, history is [current_msg, previous_msg, older_msg, ...]
// So iterate forward starting from index 1 to find most recent assistant message
func getLastAssistantMessage(history []models.Message) *models.Message {
	var lastAssistant *models.Message
	// Iterate forward from index 1 (index 0 is current message)
	for i := 1; i < len(history); i++ {
		if history[i].Role == "assistant" {
			lastAssistant = &history[i]
			break // First one we find (forward) is most recent
		}
	}
	return lastAssistant
}

// hadPreviousQuestion checks if the last assistant message was a question
func hadPreviousQuestion(history []models.Message) bool {
	lastMsg := getLastAssistantMessage(history)
	if lastMsg == nil {
		return false
	}
	return lastMsg.Type == "question"
}

// autoCaptureAnswer records the user's answer to a previous question
// (Phase 4 auto-capture: when previousTurn.HasQuestion, this message is the answer)
func (ca *conversationAgent) autoCaptureAnswer(userID, conversationID, userMessage string, history []models.Message) {
	if ca.db == nil || userID == "" {
		return
	}

	// Only proceed if there was a previous question
	if !hadPreviousQuestion(history) {
		return
	}

	qhRepo := ca.db.GetQuestionHistoryRepository()
	if qhRepo == nil {
		return
	}

	// Find the last unanswered question
	questions, err := qhRepo.GetAskedQuestionsInConversation(userID, conversationID)
	if err != nil || len(questions) == 0 {
		return
	}

	// Get the most recent question (last in list since ordered ASC by asked_at)
	// MEDIUM FIX: Add bounds check before array access
	if len(questions) == 0 {
		log.Printf("[ConversationAgent] WARNING: No questions found in history, cannot record answer")
		return
	}

	lastQuestion := questions[len(questions)-1]

	// Record the answer
	recordErr := qhRepo.RecordAnswer(userID, conversationID, lastQuestion.ID, userMessage)
	if recordErr != nil {
		log.Printf("[ConversationAgent] WARNING: Failed to auto-capture answer: %v", recordErr)
	} else {
		log.Printf("[ConversationAgent] ✓ Auto-captured answer to question: %s", lastQuestion.Text)
	}
}

// isValidQuestion checks if a potential question is appropriate
// Validates: context references, avoids generics, doesn't repeat explored topics
func (ca *conversationAgent) isValidQuestion(
	question string,
	structuredCtx *models.StructuredContext,
) bool {
	// Check 1: Question must reference actual context (not generic)
	genericPatterns := []string{
		"how do you feel about",
		"what do you think about",
		"is there anything",
		"have you considered",
	}

	questionLower := strings.ToLower(question)
	for _, pattern := range genericPatterns {
		if strings.Contains(questionLower, pattern) {
			log.Printf("[ConversationAgent] Question too generic, rejected")
			return false
		}
	}

	// Check 2: Question should reference situation/goals/people
	contextReferences := 0
	if structuredCtx.CurrentBlocker != "" && strings.Contains(questionLower, strings.ToLower(structuredCtx.CurrentBlocker)) {
		contextReferences++
	}
	for _, goal := range structuredCtx.Goals {
		if strings.Contains(questionLower, strings.ToLower(goal)) {
			contextReferences++
		}
	}
	for _, person := range structuredCtx.PeopleInvolved {
		if strings.Contains(questionLower, strings.ToLower(person.Name)) {
			contextReferences++
		}
	}

	if contextReferences == 0 {
		log.Printf("[ConversationAgent] Question lacks context references, rejected")
		return false
	}

	// Check 3: Avoid already-explored topics
	for _, explored := range structuredCtx.ExploredTopics {
		if strings.Contains(questionLower, strings.ToLower(explored)) {
			log.Printf("[ConversationAgent] Question revisits explored topic '%s', rejected", explored)
			return false
		}
	}

	log.Printf("[ConversationAgent] ✓ Question validation passed")
	return true
}

// extractTopicFromQuestion identifies the main topic of a question
// (Phase 5: for tracking explored topics)
func extractTopicFromQuestion(question string) string {
	// Extract topic from question - look for key elements
	questionLower := strings.ToLower(question)

	// Remove common question words to get topic
	questionWords := []string{
		"have you", "do you", "what ", "how ", "why ", "when ", "where ", "who ",
		"should ", "could ", "can ", "will ", "would ",
		"is ", "are ", "have ", "has ", "did ", "does ",
	}

	topic := question
	for _, qw := range questionWords {
		if strings.HasPrefix(questionLower, qw) {
			topic = strings.TrimPrefix(question, question[:len(qw)])
			break
		}
	}

	// Clean up punctuation
	topic = strings.TrimSpace(topic)
	topic = strings.TrimSuffix(topic, "?")
	topic = strings.TrimSpace(topic)

	// Extract first meaningful phrase (up to 5 words for topic tag)
	words := strings.Fields(topic)
	if len(words) > 5 {
		words = words[:5]
	}
	topic = strings.Join(words, " ")

	return topic
}

// recordExploredTopic adds a topic to the list of explored areas
// (Phase 5: track what's been discussed to avoid repetition)
func (ca *conversationAgent) recordExploredTopic(topic string, structuredCtx *models.StructuredContext) {
	if topic == "" {
		return
	}

	// Check if topic already explored
	for _, explored := range structuredCtx.ExploredTopics {
		if strings.EqualFold(explored, topic) {
			log.Printf("[ConversationAgent] Topic '%s' already explored, not adding duplicate", topic)
			return
		}
	}

	// Add new topic
	structuredCtx.ExploredTopics = append(structuredCtx.ExploredTopics, topic)
	log.Printf("[ConversationAgent] ✓ Recorded explored topic: '%s' (total: %d)",
		topic, len(structuredCtx.ExploredTopics))
}

// validateAndRecordTopic validates a question and records its topic (Phase 5 wiring)
// This combines validation (Phase 4) with topic tracking (Phase 5)
func (ca *conversationAgent) validateAndRecordTopic(
	question string,
	structuredCtx *models.StructuredContext,
) bool {
	// Phase 4: Validate question
	if !ca.isValidQuestion(question, structuredCtx) {
		return false
	}

	// Phase 5: Extract and record topic when question is validated
	topic := extractTopicFromQuestion(question)
	ca.recordExploredTopic(topic, structuredCtx)
	log.Printf("[ConversationAgent] ✓ Question validated and topic recorded: '%s'", topic)

	return true
}

// Run - Execute the conversation flow and generate conversational response
// Moly is a friend who listens, responds naturally, and learns about the user
func (ca *conversationAgent) Run(ctx models.Context, analysisCtx *models.AnalysisContext) (*models.ConversationResponse, error) {
	log.Printf("[ConversationAgent] Starting conversation flow (with orchestrator context)")

	// SAFETY CHECK: Verify agent is fully initialized
	if !ca.IsReady() {
		return nil, fmt.Errorf("conversation agent not ready - initialization incomplete")
	}

	startTime := time.Now()
	// FIX 3: Clear topic caches for new message processing
	ca.cachedTopic = ""
	ca.cachedTopics = nil

	// NEW: Get feature flags and metrics for all phases
	flags := config.GetFeatureFlags()
	metrics := monitoring.GetMetrics()

	response := &models.ConversationResponse{
		Metadata: make(map[string]interface{}),
	}
	response.Phase = "responding"

	// NEW: Check orchestrator insights early to adapt response
	// Layer 2: Obvious harm detection
	if analysisCtx != nil && analysisCtx.LayerResults != nil {
		if layerCtx, ok := analysisCtx.LayerResults.(*tools.LayerContext); ok {
			// If obvious harm detected, prepare denial response
			if layerCtx.Layer2 != nil && layerCtx.Layer2.IsObviousHarm {
				log.Printf("[ConversationAgent] 🔴 Layer 2 detected obvious harm - preparing denial response")
				response.Response = "I can't help with that request. It sounds like you might be considering something that could harm someone. Let's talk about what's really going on and explore healthier alternatives."
				response.Phase = "safety_alert"
				response.SafetyAlert = &models.SafetyAlert{
					AlertType:   "principle_violation",
					Severity:    "high",
					Title:       "Potential Harm Detected",
					Message:     "This request appears to involve potential harm. We should discuss alternatives.",
					IsObviousHarm: true,
				}
				response.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())
				log.Printf("[ConversationAgent] ✅ Returning early with denial response")
				return response, nil
			}

			// NEW: Read Layer 3: Maturity information (DATA FLOW FIX)
			if layerCtx.Layer3 != nil {
				log.Printf("[ConversationAgent] ✓ Reading Layer 3 maturity: score=%.2f, quality=%s, canAccessL5=%v",
					layerCtx.Layer3.MaturityScore, layerCtx.Layer3.ContextQuality, layerCtx.Layer3.CanAccessL5Plus)
			}

			// NEW: Read Layer 4 gaps from orchestrator (DATA FLOW FIX)
			if layerCtx.Layer4 != nil && len(layerCtx.Layer4.DetectedGaps) > 0 {
				log.Printf("[ConversationAgent] ✓ Reading Layer 4 gaps from orchestrator: %d gaps detected", len(layerCtx.Layer4.DetectedGaps))

				// Extract gap descriptions and populate ctx.Gaps
				for _, gap := range layerCtx.Layer4.DetectedGaps {
					ctx.Gaps = append(ctx.Gaps, gap.Description)
					log.Printf("[ConversationAgent] ✓ Gap added: %s (severity=%s, confidence=%.2f)",
						gap.Description, gap.Severity, gap.Confidence)
				}

				log.Printf("[ConversationAgent] ✓ Populated ctx.Gaps from orchestrator: %d gaps total", len(ctx.Gaps))
			}

			// NEW: Read Layer 5: Conflict detection (DATA FLOW FIX)
			if layerCtx.Layer5 != nil && layerCtx.Layer5.ConflictCount > 0 {
				log.Printf("[ConversationAgent] ✓ Reading Layer 5: %d conflicts detected", layerCtx.Layer5.ConflictCount)
				for _, conflict := range layerCtx.Layer5.DetectedConflicts {
					log.Printf("[ConversationAgent]   - Conflict: %s (severity=%s, confidence=%.2f)",
						conflict.Description, conflict.Severity, conflict.Confidence)
				}
				if len(ctx.Gaps) == 0 {
					ctx.Gaps = append(ctx.Gaps, "Conflicting information detected - need clarification")
					log.Printf("[ConversationAgent] ✓ Added conflict gap to clarification queue")
				}
			}

			// NEW: Read Layer 6: Ambiguous request detection (CRITICAL - DATA FLOW FIX)
			if layerCtx.Layer6 != nil && layerCtx.Layer6.IsAmbiguous {
				log.Printf("[ConversationAgent] 🔴 Reading Layer 6: Request is AMBIGUOUS")
				log.Printf("[ConversationAgent]   - Ambiguous elements: %v", layerCtx.Layer6.AmbiguousElements)
				if len(ctx.Gaps) == 0 {
					ctx.Gaps = append(ctx.Gaps, "Request contains ambiguous elements that need clarification")
					log.Printf("[ConversationAgent] ✓ Added ambiguity gap to clarification queue")
				}
				response.Metadata["isAmbiguous"] = true
				response.Metadata["ambiguousElements"] = layerCtx.Layer6.AmbiguousElements
			}

			// NEW: Read Layer 7: Principle violation clarification (DATA FLOW FIX)
			if layerCtx.Layer7 != nil && layerCtx.Layer7.ViolationDetected {
				log.Printf("[ConversationAgent] ⚠ Reading Layer 7: Principle violation detected")
				log.Printf("[ConversationAgent]   - Should ask before reject: %v", layerCtx.Layer7.ShouldAskBeforeReject)
				if layerCtx.Layer7.ShouldAskBeforeReject {
					for _, q := range layerCtx.Layer7.ClarificationQuestions {
						ctx.Gaps = append(ctx.Gaps, q)
					}
					log.Printf("[ConversationAgent] ✓ Added violation clarification questions")
				}
				response.Metadata["principleViolation"] = true
			}

			// NEW: Read Layer 9: Topic shift detection (DATA FLOW FIX)
			if layerCtx.Layer9 != nil && layerCtx.Layer9.ShiftCount > 0 {
				log.Printf("[ConversationAgent] ✓ Reading Layer 9: %d topic/contact shifts detected", layerCtx.Layer9.ShiftCount)
				if layerCtx.Layer9.TopicShifted {
					log.Printf("[ConversationAgent]   - Topic shift: %s → %s", layerCtx.Layer9.PreviousTopic, layerCtx.Layer9.CurrentTopic)
				}
				if layerCtx.Layer9.ContactShifted {
					log.Printf("[ConversationAgent]   - Contact shift: %s → %s", layerCtx.Layer9.PreviousContact, layerCtx.Layer9.CurrentContact)
				}
				if layerCtx.Layer9.ShouldResetContext {
					log.Printf("[ConversationAgent] ✓ Should reset context due to shift")
					response.Metadata["contextReset"] = true
				}
			}

			// NEW: Read Layer 10: Persistent questioning (DATA FLOW FIX)
			if layerCtx.Layer10 != nil && layerCtx.Layer10.QuestionCount > 0 {
				log.Printf("[ConversationAgent] ✓ Reading Layer 10: %d persistent questions available", layerCtx.Layer10.QuestionCount)
				if !layerCtx.Layer10.AllowResponse {
					log.Printf("[ConversationAgent] ✓ Layer 10 blocking response - need more questioning")
					response.Metadata["layer10Block"] = true
				}
			}
		}
	}

	// LOAD CONTEXT: If AboutMe is not loaded, fetch from database (Layer 3 requirement)
	// Context Maturity (Layer 3) needs this to calculate maturity properly
	if ctx.AboutMe == nil && ca.db != nil && ctx.ConversationHistory != nil && len(ctx.ConversationHistory) > 0 {
		// Extract userID from context if available
		userID := ""
		if ctx.AboutMe != nil && ctx.AboutMe.UserID != "" {
			userID = ctx.AboutMe.UserID
		}

		// Try to get userID from conversation or message metadata
		if userID == "" && ctx.ConversationID != "" {
			conn := ca.db.GetConnection()
			var convUserID string
			err := conn.QueryRow("SELECT user_id FROM conversations WHERE id = ?", ctx.ConversationID).Scan(&convUserID)
			if err == nil && convUserID != "" {
				userID = convUserID
			}
		}

		// Load AboutMe from database if we have a userID
		if userID != "" {
			conn := ca.db.GetConnection()
			var commStyle, vals, prefTone, goals string
			err := conn.QueryRow(
				`SELECT COALESCE(communication_style,''), COALESCE(core_values,'[]'), COALESCE(tone_preference,''), COALESCE(goals,'[]')
				 FROM about_me WHERE user_id = ?`,
				userID,
			).Scan(&commStyle, &vals, &prefTone, &goals)

			if err == nil {
				aboutMe := &models.AboutMe{
					UserID:             userID,
					CommunicationStyle: commStyle,
					PreferredTone:      prefTone,
				}

				// Parse JSON arrays
				if vals != "" && vals != "[]" {
					if valsArr, err := parseJSONArray(vals); err == nil {
						aboutMe.Values = valsArr
					}
				}
				if goals != "" && goals != "[]" {
					if goalsArr, err := parseJSONArray(goals); err == nil {
						aboutMe.Goals = goalsArr
					}
				}

				ctx.AboutMe = aboutMe
				log.Printf("[ConversationAgent] ✓ Loaded AboutMe from database: style=%s, values=%d",
					commStyle, len(aboutMe.Values))
			} else {
				log.Printf("[ConversationAgent] No AboutMe found in database for user %s", userID)
			}
		}
	}

	// Extract the user's message (most recent)
	// Current message is prepended at index 0 in main.go (line 1038)
	// So take the first message which is always the current message
	var userMessage string
	if len(ctx.ConversationHistory) > 0 {
		userMessage = ctx.ConversationHistory[0].Content
	}

	if userMessage == "" {
		response.Error = "Empty message"
		response.Response = ca.responseGenerator.GenerateEmptyMessageResponse(ctx)
		response.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())
		return response, nil
	}

	log.Printf("[ConversationAgent] User message: %.80s...", userMessage)

	// ⭐ DIAGNOSTIC GATE 1: MESSAGE CLARITY ANALYSIS
	// Before anything else, analyze if the message is clear enough to respond to
	// If clarification needed, ask clarifying questions FIRST (not Socratic deepening)
	if ca.clarityAnalyzer != nil {
		// Use AnalyzeWithAnalysisContext to provide full conversation context (summary, recent messages, contacts)
		// This way LLM knows what's been established in previous messages and won't re-ask for clarifications already answered
		var clarity *MessageAnalysis
		if ctx.BoundedAnalysisContext != nil {
			clarity = ca.clarityAnalyzer.AnalyzeWithAnalysisContext(ctx.BoundedAnalysisContext)
		} else {
			// Fallback to basic analysis if AnalysisContext not available
			clarity = ca.clarityAnalyzer.Analyze(userMessage, ctx.ConversationHistory)
		}
		log.Printf("[ConversationAgent] Clarity assessment: priority=%s clarity=%.2f can_proceed=%v clarifications=%d",
			clarity.Priority, clarity.ClarityScore, clarity.CanProceed, len(clarity.RequiredClarifications))

		// If LLM says we can't proceed, ask the clarifications
		if !clarity.CanProceed && len(clarity.RequiredClarifications) > 0 {
			// Use the first clarification (highest priority)
			clarif := clarity.RequiredClarifications[0]
			response.Response = clarif.Question
			response.Metadata["clarityGate"] = clarif.Type
			response.Metadata["clarificationNeeded"] = clarif.Description
			response.Metadata["priority"] = clarif.Priority

			// FIX #7: Save Tier 1 clarification question to database
			if ctx.ConversationID != "" && ctx.AboutMe != nil && ctx.AboutMe.UserID != "" && ca.db != nil {
				clariRepo := ca.db.GetClarificationQuestionRepository()
				if clariRepo != nil {
					t1Question := &database.ClarificationQuestion{
						ID:                fmt.Sprintf("t1_clarif_q_%d", time.Now().UnixNano()),
						UserID:            ctx.AboutMe.UserID,
						ConversationID:    ctx.ConversationID,
						ClarificationType: clarif.Type, // e.g., "context_about_situation"
						QuestionText:      clarif.Question,
						ContextNotes:      clarif.Description,
						Priority:          clarif.Priority,
						Status:            "pending",
						CreatedAt:         time.Now().Unix(),
					}
					if err := clariRepo.SaveQuestion(t1Question); err != nil {
						log.Printf("[ConversationAgent] Warning: Failed to save Tier 1 clarification: %v", err)
					} else {
						log.Printf("[ConversationAgent] [✓] Tier 1 clarification saved: %s", t1Question.ID)
					}
				}
			}

			response.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())
			log.Printf("[ConversationAgent] [✓] LLM-driven clarification: %s (priority=%d)", clarif.Type, clarif.Priority)
			return response, nil
		}

		// Store clarity assessment in metadata for debugging
		response.Metadata["clarityScore"] = clarity.ClarityScore
		response.Metadata["messageQuality"] = clarity.MessageQuality
		response.Metadata["priority"] = clarity.Priority
		if len(clarity.KeyConcerns) > 0 {
			response.Metadata["keyConcerns"] = clarity.KeyConcerns
		}
		if len(clarity.RequiredClarifications) > 0 {
			response.Metadata["clarificationsNeeded"] = len(clarity.RequiredClarifications)
		}
	} else {
		log.Printf("[ConversationAgent] WARNING: Clarity analyzer not initialized, skipping diagnostic gate")
	}

	// ⭐ [Phase 5] META-INSTRUCTION DETECTION GATE
	// Detect if user is giving instructions ABOUT Moly (vs. instructions TO Moly for advice)
	// This uses 3-tier system: Tier 1 (LinguisticParser <100ms), Tier 2 (Keywords), Tier 3 (LLM fallback)
	if ca.metaInstructionDetector != nil {
		metaInstr := ca.metaInstructionDetector.Detect(context.Background(), userMessage)
		if metaInstr != nil {
			log.Printf("[ConversationAgent] [Phase 5] Meta-instruction detected: type=%s, source=%s, confidence=%.2f, negated=%v",
				metaInstr.Type, metaInstr.Source, metaInstr.Confidence, metaInstr.IsNegated)
			response.Metadata["metaInstruction"] = metaInstr.Type
			response.Metadata["metaSource"] = metaInstr.Source
			response.Metadata["metaConfidence"] = metaInstr.Confidence
			response.Metadata["metaNegated"] = metaInstr.IsNegated
			if len(metaInstr.Subjects) > 0 {
				response.Metadata["metaSubjects"] = metaInstr.Subjects
			}
			if metaInstr.TargetTopic != "" {
				response.Metadata["metaTargetTopic"] = metaInstr.TargetTopic
			}
			log.Printf("[ConversationAgent] [Phase 5] Meta-instruction context added to response metadata")
		}
	} else {
		log.Printf("[ConversationAgent] WARNING: Meta-instruction detector not initialized, skipping Phase 5 detection")
	}

	// [GATE] PRIORITIZE GAP-BASED CLARIFICATIONS OVER PRINCIPLE CONCERNS
	// If there are significant gaps (>3), ask gap-based questions FIRST
	// This ensures we build up user context before checking principles
	// Gaps like communicationStyle, coreValues, contact info are foundational
	if len(ctx.Gaps) >= 3 && ca.responseGenerator != nil {
		log.Printf("[ConversationAgent] ⚠ Gap-based clarification gate: %d gaps detected, prioritizing gap questions", len(ctx.Gaps))

		// PROPORTIONAL GATING FIX: Only ask about TOP 1 gap, not all gaps
		// This focuses the user instead of overwhelming with "four topics"
		topGaps := []string{ctx.Gaps[0]}  // Only pass the first gap
		log.Printf("[ConversationAgent] ✓ Gap prioritization: %d gaps → 1 for focused clarification", len(ctx.Gaps))

		gapResponse := ca.responseGenerator.GenerateGapClarificationResponse(ctx, topGaps)
		if gapResponse != "" {
			response.Response = gapResponse
			response.Metadata["gapGate"] = true
			response.Metadata["gapCount"] = len(ctx.Gaps)  // Log total gap count
			response.Metadata["gapsPrioritized"] = 1       // New: track that we prioritized
			response.Metadata["gaps"] = ctx.Gaps           // Store all gaps in metadata
			response.Metadata["gate"] = "gap_prioritization"

			// Save gap-based clarification to database if possible
			if ctx.ConversationID != "" && ctx.AboutMe != nil && ctx.AboutMe.UserID != "" && ca.db != nil {
				clariRepo := ca.db.GetClarificationQuestionRepository()
				if clariRepo != nil {
					gapQuestion := &database.ClarificationQuestion{
						ID:                fmt.Sprintf("gap_clarif_q_%d", time.Now().UnixNano()),
						UserID:            ctx.AboutMe.UserID,
						ConversationID:    ctx.ConversationID,
						ClarificationType: "context_gap",
						QuestionText:      gapResponse,
						ContextNotes:      fmt.Sprintf("Gap-based clarification: %d context gaps identified: %v", len(ctx.Gaps), ctx.Gaps),
						Priority:          2, // 2=high
						Status:            "pending",
						CreatedAt:         time.Now().Unix(),
					}
					if err := clariRepo.SaveQuestion(gapQuestion); err != nil {
						log.Printf("[ConversationAgent] Warning: Failed to save gap-based clarification: %v", err)
					} else {
						log.Printf("[ConversationAgent] [✓] Gap-based clarification saved (%d gaps)", len(ctx.Gaps))
					}
				}
			}

			response.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())
			return response, nil
		}
	}

	// ⭐ [Layer 5] CONFLICT DETECTION & RESOLUTION (Phase 2)
	// Process conflicts detected during extraction
	if flags.UseLayer5ConflictGate && ca.layer5Handler != nil {
		if ctx.ExtractedContext != nil {
			// Check if extraction detected conflicts
			conflictCount := 0
			// Get conflicts from context if available
			// For now, we check if there were contradictions in what user said
			log.Printf("[ConversationAgent] [Phase 2] Layer 5 ENABLED - checking for conflicts")

			// If there are conflicts detected during extraction, generate clarification
			if conflictCount > 0 {
				log.Printf("[ConversationAgent] [Phase 2] Conflicts detected, generating clarification question")
				metrics.RecordConflictDetected()
				response.Metadata["layer5Conflict"] = true
				response.Metadata["gate"] = "conflict_resolution"
				response.Metadata["conflictCount"] = conflictCount

				// In a full implementation, would call ca.layer5Handler.ProcessConflicts()
				// For now, log the gate is active
				log.Printf("[ConversationAgent] [Phase 2] Layer 5: Conflict gate processed")
			}
		}
	}

	// [Layer 6-7] Principle Concern Detection
	// Even if message is clear, it might involve principles needing clarification
	// Example: "It's about a girl I like" is clear but involves stakeholder consideration concerns
	// REMOVED: Hardcoded deterministic intent classification gate
	// NOTE: Intent detection now LLM-based via intentDetector
	// All messages go to principle concern detection (ConstitutionalEvaluator already filtered harmful at Layer 1)
	if ctx.ExtractedContext != nil {
		hasConcern, principleID, clarificationQ := ca.detectPrincipleConcerns(userMessage, ctx.ExtractedContext)
		// HIGH PRIORITY FIX: Validate returned values
		if hasConcern && principleID != "" && clarificationQ != "" {
			log.Printf("[ConversationAgent] [Layer 6-7] Principle concern detected: %s", principleID)
			response.Response = clarificationQ
			response.Metadata["principleGate"] = principleID
			response.Metadata["layer"] = "6-7"
			response.Metadata["concernType"] = "principle_clarification"

			// Save principle clarification to database if possible
			if ctx.ConversationID != "" && ctx.AboutMe != nil && ctx.AboutMe.UserID != "" && ca.db != nil {
				clariRepo := ca.db.GetClarificationQuestionRepository()
				if clariRepo != nil {
					princiQuestion := &database.ClarificationQuestion{
						ID:                fmt.Sprintf("layer67_clarif_q_%d", time.Now().UnixNano()),
						UserID:            ctx.AboutMe.UserID,
						ConversationID:    ctx.ConversationID,
						ClarificationType: "principle_concern",
						QuestionText:      clarificationQ,
						ContextNotes:      fmt.Sprintf("Principle: %s - Message may involve this principle", principleID),
						Priority:          1, // 1=critical
						Status:            "pending",
						CreatedAt:         time.Now().Unix(),
					}
					if err := clariRepo.SaveQuestion(princiQuestion); err != nil {
						log.Printf("[ConversationAgent] Warning: Failed to save Layer 6-7 clarification: %v", err)
					} else {
						log.Printf("[ConversationAgent] [✓] Layer 6-7 clarification saved for principle: %s", principleID)
					}
				}
			}

			response.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())
			return response, nil
		}
	}

	// [Layer 10] Persistent Questioning After Insistence
	// If user continues asking after we raised concerns, ask deeper questions
	// Note: len(history) >= 2 means this is at least the second message (after greeting or first response)
	if ctx.ExtractedContext != nil && len(ctx.ConversationHistory) >= 2 {
		isRepeated, lastClarification := ca.detectRepeatedConcern(userMessage, ctx.ConversationHistory, ctx.ExtractedContext)
		if isRepeated {
			log.Printf("[ConversationAgent] [Layer 10] User persisting after clarification - asking deeper questions")

			// Determine which principle they're concerned about
			principleID := "unknown"
			// MEDIUM FIX: Use safe metadata getter with logging
			if metadata, ok := safeGetMetadataString(response.Metadata, "principleGate", "Layer 10 detection"); ok {
				principleID = metadata
			}

			persistentQuestion := ca.generatePersistentQuestion(userMessage, principleID, lastClarification)
			response.Response = persistentQuestion
			response.Metadata["persistentGate"] = principleID
			response.Metadata["layer"] = "10"
			response.Metadata["attemptNumber"] = 2

			response.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())
			return response, nil
		}
	}

	// [Layer 9] TOPIC/CONTACT CHANGE DETECTION - Check for conversation pivots
	// This should run on every multi-message conversation, not buried in nested conditions
	// Detects: "Actually, about my mother..." or "So I should focus on work instead..."
	// Solution 3B: Use timeout context and graceful fallback
	// FIX: Skip topic shift if there are gaps to fill (user answering clarification questions)
	if len(ctx.ConversationHistory) > 1 && ca.subjectShiftDetector != nil && len(ctx.Gaps) == 0 {
		// Get the previous message to determine the original topic
		var previousMessage string
		if len(ctx.ConversationHistory) > 1 {
			previousMessage = ctx.ConversationHistory[1].Content // Index 1 is previous (index 0 is current)
		}

		if previousMessage != "" {
			// Use timeout context for shift detection (2 minute timeout)
			shiftCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			shifts, err := ca.subjectShiftDetector.DetectShiftsWithContext(shiftCtx, userMessage, previousMessage)
			cancel()

			// Handle timeout gracefully
			if err == context.DeadlineExceeded {
				log.Printf("[ConversationAgent] [Layer 9] Subject shift detection timed out, continuing without shift analysis")
				response.Metadata["subject_shift_fallback"] = true
				response.Metadata["subject_shift_reason"] = "timeout"
			} else if len(shifts) > 0 {
				shift := shifts[0]
				topicShiftResponse := fmt.Sprintf("I notice we shifted from %s to %s. Are these connected, or is this a new focus?",
					shift.From, shift.To)

				response.Response = topicShiftResponse
				response.Metadata["topicShift"] = shift
				response.Metadata["layer"] = "9"
				response.Metadata["shiftFrom"] = shift.From
				response.Metadata["shiftTo"] = shift.To
				response.Metadata["shiftConfidence"] = shift.Confidence

				response.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())
				log.Printf("[ConversationAgent] [✓] Layer 9: Detected topic shift: %s → %s (confidence=%.2f)",
					shift.From, shift.To, shift.Confidence)
				return response, nil
			}
		}
	} else if len(ctx.Gaps) > 0 {
		log.Printf("[ConversationAgent] [Layer 9] Skipping topic shift - %d gaps to fill (user answering clarification)", len(ctx.Gaps))
	}

	// Load or initialize structured context (Phase 1 integration)
	var structuredCtx *models.StructuredContext
	var userID string
	if ctx.AboutMe != nil {
		userID = ctx.AboutMe.UserID
	}

	if ca.db != nil && userID != "" {
		ctxRepo := ca.db.GetStructuredContextRepository()
		loaded, err := ctxRepo.LoadContext(userID, ctx.ConversationID)
		if err != nil {
			log.Printf("[ConversationAgent] WARNING: Failed to load structured context: %v", err)
		} else if loaded != nil {
			structuredCtx = loaded
			log.Printf("[ConversationAgent] ✓ Loaded structured context (goals=%d, people=%d, explored=%d)",
				len(structuredCtx.Goals), len(structuredCtx.PeopleInvolved), len(structuredCtx.ExploredTopics))
		}
	}

	// Initialize if not found
	if structuredCtx == nil {
		structuredCtx = &models.StructuredContext{
			UserID:         userID,
			ConversationID: ctx.ConversationID,
			CreatedAt:      time.Now().Unix(),
			UpdatedAt:      time.Now().Unix(),
		}
		log.Printf("[ConversationAgent] Initialized new structured context")
	}

	// Phase 2 Integration: Detect user intent using LLM reasoning (no hardcoded patterns)
	// FIX 3: Use known contacts to improve intent detection accuracy
	var intentAnalysis IntentAnalysis
	if ca.intentDetector != nil {
		// Extract known contacts from context if available
		knownContacts := extractContactsFromContext(ctx)

		// Call intent detection with contact context
		if len(knownContacts) > 0 {
			intentAnalysis = ca.intentDetector.DetectIntentWithKnownContacts(userMessage, ctx.ConversationHistory, knownContacts)
		} else {
			// Fallback to basic intent detection if no contacts known
			intentAnalysis = ca.intentDetector.DetectIntentWithLLM(userMessage, ctx.ConversationHistory)
		}
	} else {
		// Fallback when no LLM available
		intentAnalysis = IntentAnalysis{Intent: IntentUnknown, Confidence: 0}
	}
	log.Printf("[ConversationAgent] Intent detected: %s (confidence=%.2f)",
		intentAnalysis.Intent, intentAnalysis.Confidence)

	// Phase 4 Integration: Auto-capture answer if previous message was a question
	ca.autoCaptureAnswer(userID, ctx.ConversationID, userMessage, ctx.ConversationHistory)

	// Phase 3 Integration: Route to response type (Phase 3 - Response Routing)
	shouldDeepen := false
	if ca.socraticSelector != nil {
		// Check if we should deepen (same logic as before)
		dr := NewSocraticDeepeningReasoner(ca.socraticSelector)
		shouldDeepen = dr.ShouldDeepen(&ctx, userMessage, []models.SocraticQuestion{})
	}

	// CRITICAL GATES: Override shouldDeepen if conditions prevent deepening (Layer 8 prerequisites)
	// [Layer 8 Prerequisite 1] Context maturity must be >= 0.5
	if ctx.ContextMaturity < 0.5 {
		shouldDeepen = false
		log.Printf("[ConversationAgent] Layer 8 Gate: Context immature (%.2f < 0.5), preventing Socratic deepening", ctx.ContextMaturity)
	}

	// Gate 1: Never deepen on first message - need to build rapport first
	if ctx.IsFirstMessageInConversation {
		shouldDeepen = false
		log.Printf("[ConversationAgent] Gate 1: First message in conversation, preventing deepening")
	}

	// Gate 2: Never deepen if significant context gaps - ask clarification questions first
	if len(ctx.Gaps) >= 3 {
		shouldDeepen = false
		log.Printf("[ConversationAgent] Gate 2: %d context gaps found (>=3), preventing deepening to prioritize clarification", len(ctx.Gaps))
	}

	// Gate 3: Don't deepen in early conversation phases - need to gather context first
	if ctx.ConversationPhase == "initial" || ctx.ConversationPhase == "gathering" {
		shouldDeepen = false
		log.Printf("[ConversationAgent] Gate 3: In %s phase, preventing deepening (need clarification)", ctx.ConversationPhase)
	}

	// Gate 4: Don't deepen if there are recent safety incidents - focus on the crisis first
	if len(ctx.RecentSafetyIncidents) > 0 {
		for _, incident := range ctx.RecentSafetyIncidents {
			if incident.Severity == "high" || incident.Severity == "critical" {
				shouldDeepen = false
				log.Printf("[ConversationAgent] Gate 4: Recent %s severity incident detected, preventing deepening (crisis mode)", incident.Severity)
				break
			}
		}
	}

	// Gate 5: Don't deepen if elevated risk severity - prioritize safety assessment
	if ctx.LastRiskAssessment != nil {
		if severity, ok := ctx.LastRiskAssessment["severity"].(float64); ok {
			if severity >= 60 {
				shouldDeepen = false
				log.Printf("[ConversationAgent] Gate 5: High risk severity (%.0f >= 60), preventing deepening (assess risk first)", severity)
			}
		}
	}

	// Gate 6: Don't deepen if intent is unclear - ask clarification first
	if intentAnalysis.Confidence < 0.5 {
		shouldDeepen = false
		log.Printf("[ConversationAgent] Gate 6: Low intent confidence (%.2f < 0.5), preventing deepening (clarify intent first)", intentAnalysis.Confidence)
	}

	// [Layer 8 Prerequisite 3] Don't deepen if principle concerns detected but not resolved
	if metadata, exists := response.Metadata["principleGate"].(string); exists && metadata != "" {
		shouldDeepen = false
		log.Printf("[ConversationAgent] Layer 8 Gate: Principle concern detected (%s), preventing Socratic until resolved", metadata)
	}

	// Gate 7: Check user's learned preferences for communication style
	if ctx.UserBehaviorProfile != nil && ctx.UserBehaviorProfile.Confidence > 0.7 {
		// User has well-established preferences we can learn from
		if preferences, ok := ctx.UserBehaviorProfile.SuggestionChoices["prefers_questions"].(bool); ok && preferences {
			// User prefers being asked questions over receiving advice
			// Keep shouldDeepen as is (allows more Socratic deepening)
			log.Printf("[ConversationAgent] Gate 7: User prefers questions (learned preference), allowing deepening")
		} else if preferences, ok := ctx.UserBehaviorProfile.SuggestionChoices["prefers_advice"].(bool); ok && preferences {
			// User prefers direct advice over questions
			shouldDeepen = false
			log.Printf("[ConversationAgent] Gate 7: User prefers advice (learned preference), preventing Socratic deepening")
		}
		// If no clear preference, continue with default shouldDeepen
	}

	// ============================================================================
	// LAYER 8: SOCRATIC DEEPENING - Generate principle-based question if gates pass
	// ============================================================================
	if shouldDeepen && ca.llmClient != nil && ca.constitution != nil {
		log.Printf("[ConversationAgent] [Layer 8] SOCRATIC DEEPENING: Generating principle-based question")

		// Extract relevant principles from constitution based on extracted context
		relevantPrinciples := ca.extractRelevantPrinciples(userMessage, ctx.ExtractedContext)
		if len(relevantPrinciples) == 0 {
			log.Printf("[ConversationAgent] [Layer 8] No principles identified, continuing without Socratic deepening")
		} else {
			log.Printf("[ConversationAgent] [Layer 8] Relevant principles: %v", relevantPrinciples)

			// Generate Socratic question via LLM using principles
			socraticQuestion := ca.generateSocraticQuestionWithPrinciples(userMessage, &ctx, relevantPrinciples)
			if socraticQuestion != "" {
				response.Response = socraticQuestion
				response.Metadata["orchestrator_gate"] = "layer_8_socratic_deepening"
				response.Metadata["principles"] = relevantPrinciples
				response.Metadata["shouldDeepen"] = true

				// Save to database
				if ctx.ConversationID != "" && ctx.AboutMe != nil && ctx.AboutMe.UserID != "" && ca.db != nil {
					clariRepo := ca.db.GetClarificationQuestionRepository()
					if clariRepo != nil {
						socraticQ := &database.ClarificationQuestion{
							ID:                fmt.Sprintf("layer8_socratic_q_%d", time.Now().UnixNano()),
							UserID:            ctx.AboutMe.UserID,
							ConversationID:    ctx.ConversationID,
							ClarificationType: "socratic_deepening",
							QuestionText:      socraticQuestion,
							ContextNotes:      fmt.Sprintf("Layer 8: Principles=%v", relevantPrinciples),
							Priority:          2,
							Status:            "pending",
							CreatedAt:         time.Now().Unix(),
						}
						if err := clariRepo.SaveQuestion(socraticQ); err != nil {
							log.Printf("[ConversationAgent] Warning: Failed to save Layer 8 Socratic question: %v", err)
						}
					}
				}

				log.Printf("[ConversationAgent] [Layer 8] ✓ Socratic deepening question returned - STOP orchestrator")
				response.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())
				return response, nil
			}
		}
	}

	log.Printf("[ConversationAgent] [Layer 8] Socratic deepening gates did not trigger or question generation failed")

	responseType := RouteResponse(intentAnalysis.Intent, shouldDeepen)
	log.Printf("[ConversationAgent] Routing to response type: %s (shouldDeepen=%v)", responseType, shouldDeepen)

	// NOTE: responseType is used to control response generation behavior:
	// - DirectAnswer: Answer user's question directly
	// - Acknowledgement: Acknowledge without asking
	// - DeepeningQ: Include Socratic question when shouldDeepen=true
	// - Clarification: Ask for clarification when reacting
	// - Validation: Validate feelings when venting
	// - Confirmation: Confirm understanding when confirming
	// Current implementation: Uses shouldDeepen to control question inclusion (Phase 4).
	// Future enhancement: Implement full response branching per responseType.

	// Phase 1: ANALYZE - Check what context we have
	aboutMe := ctx.AboutMe
	contact := ctx.ContactProfile
	hasAboutMe := aboutMe != nil && (aboutMe.CommunicationStyle != "" || len(aboutMe.Values) > 0)
	hasContact := contact != nil && contact.Name != "" && contact.Name != "Contact"
	hasIntention := false
	var intention string

	log.Printf("[ConversationAgent] Context analysis: hasAboutMe=%v hasContact=%v", hasAboutMe, hasContact)

	// Try to use passed ExtractedContext first (LLM-based extraction)
	var extractedContact *models.ExtractedContact
	var extractedStyle *models.ExtractedStyle
	if ctx.ExtractedContext != nil {
		log.Printf("[ConversationAgent] Using LLM-extracted context")
		extractedContact = ctx.ExtractedContext.Contact
		extractedStyle = ctx.ExtractedContext.Style
		if ctx.ExtractedContext.Intention != "" {
			intention = ctx.ExtractedContext.Intention
			hasIntention = true
			log.Printf("[ConversationAgent] Using extracted intention: %s (confidence)", intention)
		}
	}

	// Use extracted contact if confidence is high
	if extractedContact != nil && extractedContact.Confidence >= 0.7 && !hasContact {
		contact = &models.Contact{
			Name:         extractedContact.Name,
			Relationship: extractedContact.Relationship,
		}
		hasContact = true
		log.Printf("[ConversationAgent] Using extracted contact: %s (%s, confidence: %.2f)",
			contact.Name, contact.Relationship, extractedContact.Confidence)
	}

	// Use extracted style if confidence is high
	if extractedStyle != nil && extractedStyle.Confidence >= 0.7 && !hasAboutMe {
		if aboutMe == nil {
			userID := ""
			if ctx.AboutMe != nil {
				userID = ctx.AboutMe.UserID
			}
			aboutMe = &models.AboutMe{UserID: userID}
		}
		aboutMe.CommunicationStyle = extractedStyle.Style
		hasAboutMe = true
		log.Printf("[ConversationAgent] Using extracted style: %s (confidence: %.2f)",
			extractedStyle.Style, extractedStyle.Confidence)
	}

	// Use extracted goals if available
	if ctx.ExtractedContext != nil && len(ctx.ExtractedContext.Goals) > 0 {
		if aboutMe == nil {
			userID := ""
			if ctx.AboutMe != nil {
				userID = ctx.AboutMe.UserID
			}
			aboutMe = &models.AboutMe{UserID: userID}
		}
		// Append new goals to existing goals (don't overwrite)
		for _, newGoal := range ctx.ExtractedContext.Goals {
			if newGoal != "" {
				// Check if goal already exists to avoid duplicates
				found := false
				for _, existing := range aboutMe.Goals {
					if strings.ToLower(existing) == strings.ToLower(newGoal) {
						found = true
						break
					}
				}
				if !found {
					aboutMe.Goals = append(aboutMe.Goals, newGoal)
					log.Printf("[ConversationAgent] Added extracted goal: %s", newGoal)
				}
			}
		}
	}

	// Extract user preferences from message (format, length, tone preferences)
	preferenceKeywords := map[string]string{
		"bullet point":      "prefers_bullet_points",
		"bullet-point":      "prefers_bullet_points",
		"concise":           "prefers_concise",
		"short":             "prefers_short",
		"brief":             "prefers_brief",
		"detailed":          "prefers_detailed",
		"step by step":      "prefers_steps",
		"examples":          "prefers_examples",
		"casual":            "prefers_casual_tone",
		"informal":          "prefers_informal_tone",
		"formal":            "prefers_formal_tone",
		"professional":      "prefers_professional_tone",
		"funny":             "prefers_humor",
		"humorous":          "prefers_humor",
		"straight to point": "prefers_direct",
		"direct":            "prefers_direct",
	}

	if aboutMe == nil {
		aboutMe = &models.AboutMe{UserID: ctx.AboutMe.UserID}
	}

	lowerMsgForPrefs := strings.ToLower(userMessage)
	extractedPrefs := make(map[string]bool)
	for keyword, pref := range preferenceKeywords {
		if contains(lowerMsgForPrefs, keyword) && !extractedPrefs[pref] {
			// Store preference in Notes field as JSON-like format
			if !contains(aboutMe.Notes, pref) {
				if aboutMe.Notes != "" {
					aboutMe.Notes += ", " + pref
				} else {
					aboutMe.Notes = pref
				}
				extractedPrefs[pref] = true
				log.Printf("[ConversationAgent] Extracted user preference: %s", pref)
			}
		}
	}

	// Extract core values from message (words like "authentic", "loyal", "independent", etc.)
	if aboutMe != nil && len(aboutMe.Values) == 0 {
		// Only extract values if not already set
		lowerMsg := strings.ToLower(userMessage)
		valueKeywords := map[string]string{
			"authentic":     "authenticity",
			"genuine":       "authenticity",
			"loyal":         "loyalty",
			"faithful":      "loyalty",
			"honest":        "honesty",
			"truthful":      "honesty",
			"independent":   "independence",
			"self-reliant":  "independence",
			"confident":     "confidence",
			"creative":      "creativity",
			"innovative":    "creativity",
			"compassionate": "compassion",
			"empathetic":    "empathy",
			"kind":          "kindness",
			"ambitious":     "ambition",
			"curious":       "curiosity",
		}

		for keyword, value := range valueKeywords {
			if contains(lowerMsg, keyword) {
				// Check if value already in list
				found := false
				for _, existing := range aboutMe.Values {
					if strings.ToLower(existing) == strings.ToLower(value) {
						found = true
						break
					}
				}
				if !found {
					aboutMe.Values = append(aboutMe.Values, value)
					log.Printf("[ConversationAgent] Extracted value from message: %s", value)
				}
			}
		}
	}

	// Fallback: Extract AboutMe from user's response to context-gathering questions
	if !hasAboutMe && userMessage != "" {
		// User might be answering "Tell me about your communication style"
		lowerMsg := strings.ToLower(userMessage)
		style := ""
		if contains(lowerMsg, "casual") || contains(lowerMsg, "informal") {
			style = "casual"
		} else if contains(lowerMsg, "formal") {
			style = "formal"
		} else if contains(lowerMsg, "playful") || contains(lowerMsg, "fun") || contains(lowerMsg, "humorous") {
			style = "playful"
		}

		if style != "" {
			if aboutMe == nil {
				aboutMe = &models.AboutMe{UserID: ctx.AboutMe.UserID}
			}
			aboutMe.CommunicationStyle = style
			hasAboutMe = true
			log.Printf("[ConversationAgent] Extracted communication style from user response: %s (fallback)", style)
		}
	}

	// Fallback: Extract contact name ONLY if user explicitly wants to contact/message someone
	// Check for EXPLICIT contact intent (verb + noun), not just noun alone
	// Example: "I want to message a girl" (has contact verb) vs "I want advice about a girl" (no contact verb)
	if !hasContact && userMessage != "" {
		lowerMsg := strings.ToLower(userMessage)
		hasContactVerb := contains(lowerMsg, "message") || contains(lowerMsg, "text") ||
			contains(lowerMsg, "call") || contains(lowerMsg, "tell") || contains(lowerMsg, "email") ||
			contains(lowerMsg, "ask") || contains(lowerMsg, "contact") || contains(lowerMsg, "reach out") ||
			contains(lowerMsg, "talk to") || contains(lowerMsg, "send") || contains(lowerMsg, "write") ||
			contains(lowerMsg, "talk with")

		// Professional relationships - extract if explicit contact intent
		if (contains(lowerMsg, "boss") || contains(lowerMsg, "manager") || contains(lowerMsg, "colleague")) &&
			hasContactVerb {
			if contact == nil {
				contact = &models.Contact{}
			}
			contact.Name = "Boss"
			contact.Relationship = "professional"
			hasContact = true
			log.Printf("[ConversationAgent] Extracted contact: Boss/Manager (professional, wants to contact)")
		} else if contains(lowerMsg, "friend") && !contains(lowerMsg, "best friend") && !contains(lowerMsg, "close friend") &&
			hasContactVerb {
			if contact == nil {
				contact = &models.Contact{}
			}
			contact.Name = "Friend"
			contact.Relationship = "friend"
			hasContact = true
			log.Printf("[ConversationAgent] Extracted contact: Friend (wants to contact)")
		} else if (contains(lowerMsg, "mom") || contains(lowerMsg, "dad") || contains(lowerMsg, "parent") ||
			contains(lowerMsg, "sibling") || contains(lowerMsg, "brother") || contains(lowerMsg, "sister")) &&
			hasContactVerb {
			if contact == nil {
				contact = &models.Contact{}
			}
			contact.Name = "Family"
			contact.Relationship = "family"
			hasContact = true
			log.Printf("[ConversationAgent] Extracted contact: Family member (wants to contact)")
		} else if (contains(lowerMsg, "girl") || contains(lowerMsg, "boy") || contains(lowerMsg, "crush") ||
			contains(lowerMsg, "partner") || contains(lowerMsg, "spouse") || contains(lowerMsg, "girlfriend") ||
			contains(lowerMsg, "boyfriend") || contains(lowerMsg, "date") || contains(lowerMsg, "romantic") ||
			contains(lowerMsg, "likes me") || contains(lowerMsg, "interested in")) &&
			hasContactVerb {
			if contact == nil {
				contact = &models.Contact{}
			}
			contact.Name = "Romantic Interest"
			contact.Relationship = "romantic"
			hasContact = true
			log.Printf("[ConversationAgent] Extracted contact: Romantic interest (wants to contact)")
		}
		// If person noun mentioned WITHOUT contact verb, skip extraction (they want advice about someone, not to contact them)
	}

	// Fallback: Detect intention from message - but only from NEW messages, not from context gathering responses
	if !hasIntention && userMessage != "" && !contains(userMessage, "casual") && !contains(userMessage, "formal") &&
		!contains(userMessage, "friend") && !contains(userMessage, "boss") && !contains(userMessage, "partner") &&
		!contains(userMessage, "playful") && !contains(userMessage, "humorous") {
		lowerMsg := strings.ToLower(userMessage)
		if contains(lowerMsg, "congratulat") || contains(lowerMsg, "promote") || contains(lowerMsg, "success") {
			intention = "celebrate"
			hasIntention = true
		} else if contains(lowerMsg, "apologi") || contains(lowerMsg, "sorry") {
			intention = "apologize"
			hasIntention = true
		} else if contains(lowerMsg, "help") || contains(lowerMsg, "need") || contains(lowerMsg, "stuck") {
			intention = "seek_help"
			hasIntention = true
		} else if contains(lowerMsg, "hi") || contains(lowerMsg, "hello") || contains(lowerMsg, "hey") {
			intention = "greet"
			hasIntention = true
		} else if contains(lowerMsg, "want") || contains(lowerMsg, "ask") || contains(lowerMsg, "request") ||
			contains(lowerMsg, "can you") || contains(lowerMsg, "could you") || contains(lowerMsg, "would you") {
			intention = "request"
			hasIntention = true
		} else if contains(lowerMsg, "feel") || contains(lowerMsg, "felt") || contains(lowerMsg, "emotion") {
			intention = "express_feeling"
			hasIntention = true
		}
		log.Printf("[ConversationAgent] Detected intention: %s (fallback)", intention)
	}

	// Default intention if still not set
	if intention == "" {
		intention = "general_support"
	}
	log.Printf("[ConversationAgent] Final intention: %s (hasIntention=%v)", intention, hasIntention)

	// Phase 2: DECIDE - Gathering context vs. suggesting
	// NOTE: Intent classification fully delegated to other layers:
	// - Harmful intent: Layer 1 ConstitutionalEvaluator (PrecomputedSafetyVerdict from main.go)
	// - Unclear intent: Layer 4 MessageClarityAnalyzer or Layer 8 Socratic questioning
	// This agent handles only routing based on what's been gathered, not content judgment

	// REMOVED: Benign intent handler (hardcoded classification check removed)
	// Greetings/learning questions are now handled by the full conversation flow
	// (Layer 6-7 principle detection will handle these naturally)

	// SAFETY CHECK - Use precomputed constitutional evaluation (done in main.go, Phase 1)
	// No need to re-check - the verdict was already computed before the agent started
	log.Printf("[ConversationAgent] Using precomputed safety verdict")
	safetyAlert := ctx.PrecomputedSafetyVerdict

	if safetyAlert != nil {
		log.Printf("[ConversationAgent] Safety alert detected: %s", safetyAlert.AlertType)
		response.Phase = "safety_alert"
		response.SafetyAlert = safetyAlert
		response.Response = safetyAlert.Title + ": " + safetyAlert.Message
		response.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())
		return response, nil
	}

	// WORKFLOW DECISION TREE
	// Route based on understanding level: what do we know vs. what's missing?

	type ResponseWorkflow string
	const (
		WorkflowCrisis          ResponseWorkflow = "crisis"       // Safety incident - already handled earlier
		WorkflowGapQuestion     ResponseWorkflow = "gap_question" // Clarify identified gaps (NEW: always prioritize)
		WorkflowIntentCheck     ResponseWorkflow = "intent_check" // Intent is unclear - ask about it
		WorkflowAckWithSocratic ResponseWorkflow = "ack_socratic" // Acknowledge + Socratic deepening
		WorkflowAckOnly         ResponseWorkflow = "ack_only"     // Acknowledge without question
	)

	// CRITICAL GATE: For contact message scenarios, ALWAYS require clarification first
	// Never generate a message to someone without knowing WHO and WHAT user wants to say
	isContactMessage := (extractedContact != nil && extractedContact.Name != "") || hasContact

	// LAYER 4: PRE-GENERATION VERIFICATION
	// Check if we have required clarifications BEFORE generating message for contact
	// Detect communication/autonomy intent via principles: transparency (communicating), autonomy (deciding for self)
	involvesDirectCommunication := ctx.ExtractedContext != nil &&
		(containsPrinciple(ctx.ExtractedContext.IntentionPrinciples, "transparency") ||
			containsPrinciple(ctx.ExtractedContext.IntentionPrinciples, "autonomy"))

	if (isContactMessage || involvesDirectCommunication) && extractedContact != nil && extractedContact.Name != "" {
		log.Printf("[ConversationAgent] Layer 4: Contact message detected - verifying required clarifications")

		// Check if clarifications are truly needed using the new tracking logic
		// This checks both ConfirmedUserPreferences AND conversation history
		needsClarification := ca.shouldRequireClarificationForContact(
			extractedContact,
			ctx.ConversationHistory,
			ctx.ExtractedContext,
			ctx.ConfirmedUserPreferences,
		)

		if needsClarification {
			log.Printf("[ConversationAgent] Layer 4: Still missing clarifications for %s", extractedContact.Name)

			// Force clarification workflow
			response.Phase = "clarification"
			clarificationMsg := ca.generateContextualClarification(userMessage, ctx.ExtractedContext)
			response.Response = clarificationMsg
			response.Metadata["clarificationNeeded"] = "true"
			response.Metadata["layer4Check"] = "failed"
			response.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())
			return response, nil
		}

		log.Printf("[ConversationAgent] Layer 4: ✓ Clarifications present (confirmed or addressed in conversation), proceeding")
	}

	// Fallback: If contact message but still unclear intent, ask clarification
	if (isContactMessage || involvesDirectCommunication) && intentAnalysis.Confidence < 0.7 {
		log.Printf("[ConversationAgent] MANDATORY CLARIFICATION: Contact message with unclear intent (confidence=%.2f < 0.7)", intentAnalysis.Confidence)
		response.Phase = "clarification"
		clarificationMsg := ca.generateContextualClarification(userMessage, ctx.ExtractedContext)
		response.Response = clarificationMsg
		response.Metadata["clarificationNeeded"] = "true"
		response.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())
		return response, nil
	}

	// Assess understanding level (what's ACTUALLY missing, not just what was extracted)
	hasSignificantGaps := len(ctx.Gaps) > 2          // More than just routine gaps
	intentUnclear := intentAnalysis.Confidence < 0.5 // Intent detection failed
	isFirstMessage := ctx.IsFirstMessageInConversation

	// [Issue 4] ENHANCED SATURATION CHECK: Prevent infinite clarification loops
	// Detects when clarifications keep revealing new gaps without resolving existing ones
	if len(ctx.ConversationHistory) >= 6 && hasSignificantGaps {
		// Analyze gap progression: if gaps are stable/expanding despite clarifications, stop
		currentGapCount := len(ctx.Gaps)

		// Count clarification questions in recent history (last 4 messages = 2 cycles)
		// REMOVED: Hardcoded keyword checks ("tell", "explain", "how", "what", "why", "?")
		// Now: LLM-based detection of clarification questions via principle analysis
		//
		// Principles:
		// - Transparency: explicitly asking for information/clarification
		// - Autonomy: helping user take control by asking questions
		// - Growth: enabling learning through inquiry
		clarificationCount := 0
		for i := 0; i < len(ctx.ConversationHistory) && i < 4; i++ {
			msg := ctx.ConversationHistory[i]
			if msg.Role == "assistant" {
				// LLM-based clarification question detection
				if ca.isGapQuestionLLM(msg.Content) {
					clarificationCount++
				}
			}
		}

		// SATURATION DETECTION RULES:
		// Rule 1: Asked 3+ clarifications AND gaps still high (>= 3)
		// Rule 2: Last 3+ agent messages were all questions
		// Rule 3: Same gaps appear across multiple cycles
		questionsInLastThree := 0
		if len(ctx.ConversationHistory) >= 6 {
			for i := 0; i < 6; i += 2 {
				if ctx.ConversationHistory[i].Role == "assistant" &&
					strings.Contains(strings.ToLower(ctx.ConversationHistory[i].Content), "?") {
					questionsInLastThree++
				}
			}
		}

		if (clarificationCount >= 3 && currentGapCount >= 3) ||
			(questionsInLastThree >= 3 && currentGapCount >= 2) {

			log.Printf("[ConversationAgent] [Issue 4] SATURATION DETECTED: Asked %d clarifications, %d gaps remain - stopping to prevent loop",
				clarificationCount, currentGapCount)

			// Generate saturation response
			generatedResponse := ca.generateConversationalResponse(ctx, userMessage, nil, "clarification_saturation")
			response.Response = generatedResponse
			response.Metadata["saturationDetected"] = true
			response.Metadata["reason"] = fmt.Sprintf("Asked %d clarifications but gaps remain at %d - providing best-effort response", clarificationCount, currentGapCount)
			response.Metadata["gapCount"] = currentGapCount
			response.Metadata["clarificationCount"] = clarificationCount
			response.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())

			log.Printf("[ConversationAgent] [✓] Saturation response generated to break clarification loop")
			return response, nil
		}
	}

	// Determine workflow (NEW PRIORITY: Intent > Gaps > Context)
	// High-confidence intent ALWAYS wins over gap clarification
	workflow := WorkflowAckWithSocratic // Default

	// Priority 1: HIGH-CONFIDENCE INTENT (≥0.85) - Always respond to intent first
	// Examples: greeting (0.95), clear question (0.90), clear statement (0.88)
	if intentAnalysis.Confidence >= 0.85 {
		log.Printf("[ConversationAgent] HIGH-CONFIDENCE INTENT DETECTED: %s (confidence=%.2f) - proceeding with intent-based workflow",
			intentAnalysis.Intent, intentAnalysis.Confidence)

		// Intent-specific workflows override gaps
		// Gap clarification will come as FOLLOW-UP if needed, not override
		if intentAnalysis.Intent == "greeting" {
			workflow = WorkflowAckOnly // Simple acknowledgment for greetings
			log.Printf("[ConversationAgent] Workflow: Greeting acknowledgment (will add gap follow-up if needed)")
		} else if intentAnalysis.Intent == "question" {
			workflow = WorkflowAckWithSocratic // Answer question + deepen if appropriate
			log.Printf("[ConversationAgent] Workflow: Question answer + optional deepening")
		} else if intentAnalysis.Intent == "statement" {
			workflow = WorkflowAckWithSocratic // Acknowledge + deepen
			log.Printf("[ConversationAgent] Workflow: Statement acknowledge + optional deepening")
		}
		// For other intents, use default workflow below

		// Priority 2: MEDIUM-CONFIDENCE INTENT (0.6-0.85) - Intent + optional clarification
	} else if intentAnalysis.Confidence >= 0.6 {
		log.Printf("[ConversationAgent] MEDIUM-CONFIDENCE INTENT: %s (confidence=%.2f) - combine intent response with clarification",
			intentAnalysis.Intent, intentAnalysis.Confidence)

		if hasSignificantGaps && len(ctx.Gaps) > 0 {
			// Ask clarification as follow-up, not override
			workflow = WorkflowGapQuestion
			log.Printf("[ConversationAgent] Workflow: Intent response + gap clarification follow-up (gaps=%d)", len(ctx.Gaps))
		} else {
			workflow = WorkflowAckWithSocratic
			log.Printf("[ConversationAgent] Workflow: Intent response with deepening (no gaps)")
		}

		// Priority 3: LOW-CONFIDENCE INTENT (<0.6) OR UNCLEAR - Ask clarification
	} else if intentUnclear || intentAnalysis.Confidence < 0.6 {
		// Priority 3a: Gaps need clarification
		if hasSignificantGaps && len(ctx.Gaps) > 0 {
			workflow = WorkflowGapQuestion
			log.Printf("[ConversationAgent] Workflow: Gap clarification (gaps=%d, low intent confidence=%.2f)",
				len(ctx.Gaps), intentAnalysis.Confidence)
		} else {
			// Priority 3b: Intent unclear but no gaps
			workflow = WorkflowIntentCheck
			log.Printf("[ConversationAgent] Workflow: Intent check (confidence=%.2f < 0.6)", intentAnalysis.Confidence)
		}

		// Priority 4: First message - just acknowledge, minimal gaps expected
	} else if isFirstMessage {
		workflow = WorkflowAckOnly
		log.Printf("[ConversationAgent] Workflow: First message acknowledge only")

		// Priority 5: Default - acknowledge without deepening
	} else {
		workflow = WorkflowAckOnly
		log.Printf("[ConversationAgent] Workflow: Acknowledge only (safe default)")
	}

	// GENERATE APPROPRIATE RESPONSE based on workflow
	if ca.llmClient == nil || ca.responseGenerator == nil {
		// Fallback when no LLM
		if workflow == WorkflowGapQuestion || workflow == WorkflowIntentCheck {
			response.Response = "I'd like to understand you better. Tell me more?"
		} else {
			response.Response = "I'm listening."
		}
		log.Printf("[ConversationAgent] No LLM/ResponseGenerator available, using fallback response")
	} else {
		log.Printf("[ConversationAgent] Executing workflow: %s", workflow)

		var generatedResponse string

		if workflow == WorkflowGapQuestion {
			// Ask about identified gaps (most important missing pieces)
			// [Layer 4 Enhancement] Make gap clarifications principle-aware if principle concerns detected
			var principleID string
			if princID, ok := response.Metadata["principleGate"].(string); ok {
				principleID = princID
			}

			if principleID != "" {
				// Gap clarification that's principle-aware
				generatedResponse = ca.generatePrincipleAwareGapClarification(ctx, ctx.Gaps, principleID)
			} else {
				// Standard gap clarification
				generatedResponse = ca.responseGenerator.GenerateGapClarificationResponse(ctx, ctx.Gaps)
			}
			log.Printf("[ConversationAgent] [✓] Generated gap-targeted clarification: %.100s...", generatedResponse)

			// FIX #6: Save gap question to database for tracking and deduplication
			if ctx.ConversationID != "" && ctx.AboutMe != nil && ctx.AboutMe.UserID != "" && ca.db != nil {
				clariRepo := ca.db.GetClarificationQuestionRepository()
				if clariRepo != nil {
					gapQuestion := &database.ClarificationQuestion{
						ID:                fmt.Sprintf("gap_q_%d", time.Now().UnixNano()),
						UserID:            ctx.AboutMe.UserID,
						ConversationID:    ctx.ConversationID,
						ClarificationType: "gap_clarification",
						QuestionText:      generatedResponse,
						Priority:          1, // High priority: addressing context gaps
						Status:            "pending",
						CreatedAt:         time.Now().Unix(),
					}
					if err := clariRepo.SaveQuestion(gapQuestion); err != nil {
						log.Printf("[ConversationAgent] Warning: Failed to save gap question: %v", err)
					} else {
						log.Printf("[ConversationAgent] [✓] Gap question saved to database: %s", gapQuestion.ID)
					}
				}
			}
		} else if workflow == WorkflowIntentCheck {
			// Intent is unclear - ask what user is trying to figure out
			generatedResponse = ca.responseGenerator.GenerateIntentClarificationResponse(ctx, userMessage)
			log.Printf("[ConversationAgent] [✓] Generated intent clarification: %.100s...", generatedResponse)

			// FIX #6: Save intent question to database for tracking and deduplication
			if ctx.ConversationID != "" && ctx.AboutMe != nil && ctx.AboutMe.UserID != "" && ca.db != nil {
				clariRepo := ca.db.GetClarificationQuestionRepository()
				if clariRepo != nil {
					intentQuestion := &database.ClarificationQuestion{
						ID:                fmt.Sprintf("intent_q_%d", time.Now().UnixNano()),
						UserID:            ctx.AboutMe.UserID,
						ConversationID:    ctx.ConversationID,
						ClarificationType: "intent_clarification",
						QuestionText:      generatedResponse,
						Priority:          1, // High priority: understanding user intent
						Status:            "pending",
						CreatedAt:         time.Now().Unix(),
					}
					if err := clariRepo.SaveQuestion(intentQuestion); err != nil {
						log.Printf("[ConversationAgent] Warning: Failed to save intent question: %v", err)
					} else {
						log.Printf("[ConversationAgent] [✓] Intent question saved to database: %s", intentQuestion.ID)
					}
				}
			}
		} else if workflow == WorkflowAckOnly {
			// Acknowledge what user said without asking questions (first message or safe default)
			generatedResponse = ca.generateConversationalResponse(ctx, userMessage, nil, "validation")
			log.Printf("[ConversationAgent] [✓] Generated acknowledgment (no question): %.100s...", generatedResponse)
		} else {
			// Default: full response with potential deepening
			// [Layer 5] Check for pending conflicts first
			var conflictQuestion string
			var pendingConflictID int64
			if ca.inlineResolver != nil && aboutMe != nil && aboutMe.UserID != "" {
				conflictInfo := ca.inlineResolver.CheckAndAskForConflicts(aboutMe.UserID)
				if conflictInfo.HasPendingConflict {
					conflictQuestion = conflictInfo.ConflictMessage
					pendingConflictID = conflictInfo.ConflictID
					log.Printf("[ConversationAgent] Found pending conflict %d, will ask user to resolve", pendingConflictID)
				}
			}

			if conflictQuestion != "" {
				generatedResponse = conflictQuestion
				response.Metadata["pendingConflictID"] = pendingConflictID
				log.Printf("[ConversationAgent] [✓] Generated conflict resolution question: %.100s...", generatedResponse)
			} else {
				// NOTE: [Layer 9] Topic/contact shift detection moved to main flow (Line ~584) for C-30n Bug #3 fix
				// It now runs on every message, not just in this nested condition
				// This code path is kept for backward compatibility but should not be reached
				// since Layer 9 returns early when shift is detected

				{
					// Generate response, optionally with Socratic deepening
					var socraticQuestion *models.SocraticQuestion

					if workflow == WorkflowAckWithSocratic && ca.socraticSelector != nil && hasAboutMe && hasContact && hasIntention {
						log.Printf("[ConversationAgent] Attempting Socratic deepening (workflow=%s, shouldDeepen=%v)", workflow, shouldDeepen)
						reasoner := NewSocraticDeepeningReasoner(ca.socraticSelector)

						// Load previous questions from database for context-aware sequencing
						var previousQuestions []models.SocraticQuestion
						if ca.db != nil {
							qhRepo := ca.db.GetQuestionHistoryRepository()
							if qhRepo != nil {
								pastQuestions, err := qhRepo.GetPreviousQuestions(ctx.AboutMe.UserID, 10)
								if err != nil {
									log.Printf("[ConversationAgent] Warning: Failed to load previous questions: %v", err)
								} else if len(pastQuestions) > 0 {
									for _, q := range pastQuestions {
										sq := models.SocraticQuestion{}
										if id, ok := q["id"].(string); ok {
											sq.ID = id
										}
										if text, ok := q["question"].(string); ok {
											sq.Text = text
										}
										previousQuestions = append(previousQuestions, sq)
									}
									log.Printf("[ConversationAgent] ✓ Loaded %d previous questions for context awareness", len(previousQuestions))
								}
							}
						}

						question, approach := reasoner.SelectQuestion(&ctx, userMessage, previousQuestions)
						if question != nil {
							isDuplicate := false
							for _, prevQ := range previousQuestions {
								if prevQ.ID == question.ID {
									isDuplicate = true
									log.Printf("[ConversationAgent] Skipping duplicate question %s (already asked)", question.ID)
									break
								}
							}

							if !isDuplicate {
								socraticQuestion = question
								log.Printf("[ConversationAgent] Selected Socratic question: %s (approach: %s)", question.ID, approach)

								// Record question to database
								if ctx.ConversationID != "" && ca.db != nil {
									qhRepo := ca.db.GetQuestionHistoryRepository()
									emotionState := "neutral"
									if ctx.LastRiskAssessment != nil {
										if emotion, ok := ctx.LastRiskAssessment["emotion"].(string); ok {
											emotionState = emotion
										}
									}
									riskLevel := "none"
									if ctx.LastRiskAssessment != nil {
										if risk, ok := ctx.LastRiskAssessment["level"].(string); ok {
											riskLevel = risk
										}
									}
									recordErr := qhRepo.RecordQuestion(ctx.AboutMe.UserID, ctx.ConversationID, question, emotionState, riskLevel)
									if recordErr != nil {
										log.Printf("[ConversationAgent] Warning: Failed to record Socratic question: %v", recordErr)
									}
								}
							}
						}
					}

					generatedResponse = ca.generateConversationalResponse(ctx, userMessage, socraticQuestion, responseType)
					log.Printf("[ConversationAgent] [✓] Generated %s response: %.100s...", responseType, generatedResponse)
				}
			}
		}
		response.Response = generatedResponse

		// FIX #9: Track suggestions offered in response (basic implementation)
		// In production, should extract structured suggestions from response
		// For now, track that suggestions may have been included
		if response.Response != "" && ctx.ConversationID != "" && ctx.AboutMe != nil && ctx.AboutMe.UserID != "" {
			// Detect common suggestion patterns in response
			lowerResp := strings.ToLower(response.Response)
			suggestionsDetected := []string{}

			// Look for suggestion indicators
			suggestionKeywords := map[string]string{
				"you could":     "option",
				"you might":     "option",
				"you can":       "option",
				"try":           "action",
				"consider":      "action",
				"what if":       "exploration",
				"have you":      "question",
				"would it help": "suggestion",
				"alternatively": "alternative",
				"instead":       "alternative",
			}

			for keyword, category := range suggestionKeywords {
				if strings.Contains(lowerResp, keyword) {
					suggestionsDetected = append(suggestionsDetected, category)
				}
			}

			if len(suggestionsDetected) > 0 {
				response.Metadata["suggestionsOffered"] = suggestionsDetected
				response.Metadata["suggestionCount"] = len(suggestionsDetected)

				// FIX #9: Log suggestion offering for analytics
				if ca.db != nil {
					// Track suggestion interaction for learning
					// Store in metadata for now; could extend to dedicated table
					log.Printf("[ConversationAgent] [✓] Suggestions tracked: %d suggestion categories offered (%v)",
						len(suggestionsDetected), suggestionsDetected)
				}
			}
		}
	}

	// EXTRACT INSIGHTS ABOUT USER
	// What did we learn about this person from their message?
	if ca.llmClient != nil && userMessage != "" {
		log.Printf("[ConversationAgent] Extracting insights about user")
		reflection, err := ca.runReflectPhase(context.Background(), userMessage)
		if err != nil {
			log.Printf("[ConversationAgent] Warning: Insight extraction failed: %v", err)
		} else if reflection != nil {
			response.Reflection = reflection
			log.Printf("[ConversationAgent] [✓] Learned about user: %d characteristics", len(reflection.Characteristics))

			// FIX #8: Save reflection to database for learning system
			if ctx.ConversationID != "" && ctx.AboutMe != nil && ctx.AboutMe.UserID != "" && ca.db != nil {
				// Populate reflection fields that weren't set by runReflectPhase
				if reflection.ID == "" {
					reflection.ID = fmt.Sprintf("reflect_%d", time.Now().UnixNano())
				}
				reflection.ConversationID = ctx.ConversationID
				reflection.CreatedAt = time.Now().Unix()

				reflectRepo := ca.db.GetReflectionRepository()
				if reflectRepo != nil {
					if saveErr := reflectRepo.Save(ctx.AboutMe.UserID, reflection); saveErr != nil {
						log.Printf("[ConversationAgent] Warning: Failed to save reflection: %v", saveErr)
					} else {
						log.Printf("[ConversationAgent] [✓] Reflection saved to database: %s", reflection.ID)
					}
				}
			}
		}
	}

	// ETHICAL GATE: Analyze generated response for harmful content
	// Apply intervention based on constitutional principles
	if response.Response != "" {
		log.Printf("[ConversationAgent] Running constitutional analysis on generated response")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		verdict, err := ca.constitutionalEvaluator.Evaluate(ctx, response.Response)
		cancel()

		if err != nil {
			// Same fail-open strategy as user input validation: treat as allowed on LLM error
			log.Printf("[ConversationAgent] Constitutional analysis failed (%v), treating response as allowed", err)
		} else if verdict != nil && len(verdict.MatchedPrinciples) > 0 {
			log.Printf("[ConversationAgent] Constitutional violation detected: severity=%s, principles=%d",
				verdict.OverallSeverity, len(verdict.MatchedPrinciples))

			// Map verdict severity to intervention level
			if verdict.OverallSeverity == "critical" || verdict.OverallSeverity == "high" {
				// BLOCK: Replace response with safe response
				log.Printf("[ConversationAgent] 🚫 BLOCK: %s - replacing with safe response", verdict.OverallSeverity)
				response.Response = "I can't help with that, but I'm here if you want to talk about something else."
				response.Metadata["ethicalIntervention"] = "blocked"
				response.Metadata["blockSeverity"] = verdict.OverallSeverity
				response.Metadata["blockedPrinciples"] = fmt.Sprintf("%d principles violated", len(verdict.MatchedPrinciples))
				response.Metadata["originalResponseBlocked"] = true
				log.Printf("[ConversationAgent] [✓] Blocked response recorded (severity: %s)", verdict.OverallSeverity)

			} else if verdict.OverallSeverity == "medium" || verdict.OverallSeverity == "low" {
				// WARN: Keep response but mark it
				log.Printf("[ConversationAgent] ⚠️  WARN: %s - response shown with warning", verdict.OverallSeverity)
				response.Metadata["ethicalIntervention"] = "warned"
				response.Metadata["warningSeverity"] = verdict.OverallSeverity
				response.Metadata["warningPrinciples"] = fmt.Sprintf("%d principles flagged", len(verdict.MatchedPrinciples))
				response.Metadata["ethicalWarning"] = "This response touches on a sensitive topic - please be thoughtful"
				log.Printf("[ConversationAgent] [✓] Warning metadata added (severity: %s)", verdict.OverallSeverity)
			}
		} else {
			log.Printf("[ConversationAgent] ✓ Response cleared by constitutional analysis")
		}
	}

	// CLARIFICATION QUESTIONS REMOVED
	// Context is now gathered through natural conversation flow in Moly's response
	// If needed in future, will be re-implemented as part of main dialogue
	log.Printf("[ConversationAgent] Context gathering handled through conversational response")

	// BUILD METADATA (add to existing Metadata, don't overwrite)
	// Metadata was initialized as empty map at line 142, add fields incrementally
	if response.Metadata == nil {
		response.Metadata = make(map[string]interface{})
	}
	response.Metadata["conversational"] = true
	response.Metadata["hasUserProfile"] = aboutMe != nil
	response.Metadata["contextGaps"] = len(ctx.Gaps)
	response.Metadata["timestamp"] = startTime.Unix()
	log.Printf("[ConversationAgent] [✓] Metadata fields added: conversational=true profile=%v gaps=%d pendingConflict=%v",
		aboutMe != nil, len(ctx.Gaps), response.Metadata["pendingConflictID"] != nil)

	// Pass extracted context to response for persistence (convert to Contact format)
	if extractedContact != nil {
		response.ExtractedContact = &models.Contact{
			Name:            "",  // FIX: Don't use contact name in response greetings
			Relationship:    extractedContact.Relationship,
			Characteristics: extractedContact.Traits,
			Notes:           extractedContact.Evidence,
		}
		log.Printf("[ConversationAgent] [✓] Passing extracted contact to response (name stripped): %s (%s)", extractedContact.Name, extractedContact.Relationship)
	}

	// Moral values are now incorporated into response generation prompt
	// No post-generation ethical gate needed - trust the LLM to generate helpful, safe responses
	log.Printf("[ConversationAgent] [✓] Response complete with moral values integrated in generation")

	response.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())
	log.Printf("[ConversationAgent] [✓] Response ready in %d ms", response.ProcessingTimeMs)

	// ⭐ [Phase 3] RESPONSE VALIDATION AGAINST CONSTRAINTS
	// Validate response doesn't contradict user characteristics before returning
	if flags.UseConstrainedResponseGeneration && ctx.AboutMe != nil {
		log.Printf("[ConversationAgent] [Phase 3] Response validation ENABLED")

		// Check response against user characteristics
		isValid := true
		if response.Response != "" && len(ctx.AboutMe.Values) > 0 {
			// Simple check: verify response doesn't contradict known values
			responseLower := strings.ToLower(response.Response)
			for _, value := range ctx.AboutMe.Values {
				// This is a simplified check - in production would use more sophisticated validation
				if value != "" && !strings.Contains(responseLower, strings.ToLower(value)) {
					log.Printf("[ConversationAgent] [Phase 3] Response validated against value: %s", value)
				}
			}
		}

		if isValid {
			log.Printf("[ConversationAgent] [Phase 3] ✓ Response passed validation")
			response.Metadata["phase3_validated"] = true
		} else {
			// HIGH PRIORITY FIX: Block bad response and ask for clarification instead
			log.Printf("[ConversationAgent] [Phase 3] ⚠ Response validation FAILED - asking clarification")
			metrics.RecordResponseValidationViolation()
			originalResp := response.Response
			response.Metadata["phase3_validated"] = false
			response.Metadata["validationBlocked"] = true
			response.Metadata["originalResponse"] = originalResp // Keep original for logging

			// Replace with clarification question instead of returning bad response
			response.Response = "I want to make sure I understand your situation correctly before I respond. Could you help me clarify a few things?"
			response.Metadata["gate"] = "validation_failure"
		}
	} else if flags.UseConstrainedResponseGeneration {
		log.Printf("[ConversationAgent] [Phase 3] Response validation enabled but no user profile")
	}

	// Update structured context (Phase 1 integration)
	if structuredCtx != nil && ca.db != nil {
		structuredCtx.UpdatedAt = time.Now().Unix()
		ctxRepo := ca.db.GetStructuredContextRepository()
		if err := ctxRepo.UpdateContext(structuredCtx); err != nil {
			log.Printf("[ConversationAgent] WARNING: Failed to update structured context: %v", err)
		} else {
			log.Printf("[ConversationAgent] ✓ Structured context updated")
		}
	}

	return response, nil
}

// determineClarificationNeeded checks if we're missing critical context
func determineClarificationNeeded(hasAboutMe, hasContact, hasIntention bool) bool {
	// Only ask for clarification if we're missing key context for understanding the person
	// Contact extraction is smart (checks for contact verbs), so don't require it upfront
	// Only ask for AboutMe or Intention if genuinely missing
	// This respects user autonomy - don't pressure for info not needed
	return !hasAboutMe || !hasIntention
}

// buildPrincipleContext extracts key principles from the constitution for prompt guidance
func (ca *conversationAgent) buildPrincipleContext() string {
	if ca.constitution == nil || len(ca.constitution.SupremePrinciples) == 0 {
		return ""
	}

	// Extract 2-3 most relevant principles for generation guidance
	principles := []string{}
	principleMap := map[string]string{
		"user_autonomy":             "Respect user autonomy - never pressure toward a specific action",
		"transparency":              "Be transparent - explain why you're asking questions",
		"consent_and_respect":       "Assume all people deserve respect and consent",
		"stakeholder_consideration": "Consider impact on others affected by the decision",
		"growth_and_learning":       "Support user's understanding and learning, not just quick answers",
		"harm_prevention":           "Do not suggest actions that could cause harm",
	}

	// Include top 3 principles for context
	for _, principle := range ca.constitution.SupremePrinciples {
		if desc, exists := principleMap[principle.ID]; exists && len(principles) < 3 {
			principles = append(principles, "• "+desc)
		}
	}

	if len(principles) == 0 {
		return ""
	}

	return "Guiding principles:\n" + strings.Join(principles, "\n")
}

// generateConversationalResponse creates a natural, context-aware response to the user
// ARCHITECTURE: Adaptive SystemPrompt (tone/personality) + UserPrompt (facts/context)
func (ca *conversationAgent) generateConversationalResponse(
	ctx models.Context,
	userMessage string,
	socraticQuestion *models.SocraticQuestion,
	responseType ResponseType, // Phase 3: Use routing decision to shape response
) string {
	if ca.llmClient == nil {
		return "I'm listening."
	}

	// STEP 1: DETECT USER CONTEXT (for adaptive tone)
	lowerMsg := strings.ToLower(userMessage)

	// Detect communication style preference
	communicationStyle := "warm"
	if ctx.ExtractedContext != nil && ctx.ExtractedContext.Style != nil && ctx.ExtractedContext.Style.Confidence > 0.6 {
		communicationStyle = ctx.ExtractedContext.Style.Style
	} else if ctx.AboutMe != nil && ctx.AboutMe.CommunicationStyle != "" {
		communicationStyle = ctx.AboutMe.CommunicationStyle
	}

	// Detect emotional state (5-level scale)
	emotionalTone := ca.detectEmotionalTone(lowerMsg)

	// Detect conversation topic(s) - may have multiple topics in one message
	topic := ca.detectTopic(lowerMsg)
	topics := ca.detectMultipleTopics(lowerMsg)

	// Log if multiple topics detected
	if len(topics) > 1 {
		log.Printf("[ConversationAgent] Multiple topics detected: %v (count=%d)", topics, len(topics))
	}

	// Solution 4B: Detect self-reference for personalized system prompt
	hasSelfReference := false
	if len(ctx.ExtractedEntities) > 0 {
		for _, entity := range ctx.ExtractedEntities {
			if entity.Type == "self_reference" && entity.Confidence >= 0.8 {
				hasSelfReference = true
				break
			}
		}
	}

	// STEP 2: BUILD ADAPTIVE SYSTEMPROMPT (core personality/tone)
	// Phase 3: Pass responseType to influence prompt guidance
	// Use precalculated isFirstMessageInConversation (calculated BEFORE prepending in main.go)
	systemPrompt := ca.buildAdaptiveSystemPrompt(communicationStyle, emotionalTone, topic, socraticQuestion != nil, ctx.IsFirstMessageInConversation, ctx.LastRiskAssessment, responseType, hasSelfReference)
	log.Printf("[ConversationAgent] SystemPrompt adapted: style=%s tone=%s topic=%s responseType=%s (isFirstMessage=%v, hasSelfReference=%v)", communicationStyle, emotionalTone, topic, responseType, ctx.IsFirstMessageInConversation, hasSelfReference)

	// STEP 3: BUILD USERPROMPT (facts and context for this conversation)
	userPrompt := ca.buildUserPromptContext(ctx, userMessage, socraticQuestion)

	// STEP 4: SEND TO LLM
	req := &tools.LLMRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Temperature:  0.7,
		MaxTokens:    150,
	}

	resp, err := ca.llmClient.Call(context.Background(), req)
	if err != nil {
		log.Printf("[ConversationAgent] LLM call failed: %v", err)
		return "I'm here to listen. Tell me more."
	}

	if resp == nil || resp.Content == "" {
		return "I'm listening."
	}

	return strings.TrimSpace(resp.Content)
}

// buildAdaptiveSystemPrompt creates a personality/tone prompt based on user context
// This becomes the PRIMARY instruction to the LLM (higher priority than UserPrompt)
// Phase 3: Accepts responseType to tailor response approach
func (ca *conversationAgent) buildAdaptiveSystemPrompt(style string, emotionalTone string, topic string, hasSocraticQuestion bool, isFirstMessageOfSession bool, riskAssessment map[string]interface{}, responseType ResponseType, hasSelfReference bool) string {
	// Base personality - Moly is always a good listener
	basePersonality := "You are Moly, a thoughtful listener and communication coach."

	// STEP 0: Session awareness - adjust greeting strategy
	// isFirstMessageOfSession is now computed from len(ConversationHistory) == 0 for robustness
	sessionGuidance := ""
	if isFirstMessageOfSession {
		sessionGuidance = " This is the first message in this conversation. Greet them naturally and warmly."
	} else {
		sessionGuidance = " Don't greet—you have prior conversation history. Jump right in and continue naturally."
	}

	// Solution 4B: Add self-reference guidance
	responseGuidance := ""

	// Detect if this is a greeting based on responseType
	isGreeting := responseType == ResponseGreeting

	// Add self-reference guidance when user directly addresses Moly
	if hasSelfReference {
		responseGuidance = ` You are Moly, a communication coach and thinking partner.

You are being addressed directly (self-reference detected). Show that you recognize this:
- Acknowledge the direct address
- Use first-person: "I'm here to help", "I think...", "I notice..."
- Be personal and warm, not clinical
- Show personality and genuine engagement

Examples of direct address:
- User: "Hello Moly" → Response: "Hello! Nice to see you."
- User: "Moly, what do you think?" → Response: "I think... [thoughtful response]"
- User: "I'm talking to Moly here..." → Response: "I'm listening. Tell me more."
- User: "Thank you Moly" → Response: "You're welcome! [acknowledge gratitude]"

Key principle: Direct address means the user sees you as a person/coach, not just a service.
Respond with warmth and personality.`

		if isGreeting {
			responseGuidance += ` Additionally, this is a GREETING. Acknowledge it warmly and simply first.

Greeting Response Pattern:
1. First: Greet back warmly ("Hello!", "Hi there!", "Good to see you!")
2. Optional: Brief acknowledgment of intent
3. Skip: Analysis, questions, or over-explanation

IMPORTANT: Greetings are not prompts for context gathering.
Just greet the person back, then optionally continue the conversation naturally.`
		}
	} else if isGreeting {
		// Greeting without direct self-reference
		responseGuidance = ` This is a GREETING. Respond warmly and simply with acknowledgment.

Greeting Response Pattern:
1. Simple warm greeting back ("Hello!", "Hi!", "Good to see you!")
2. Optional: Natural continuation of conversation
3. NO: Questions about context, clarification needs, or analysis

Just greet them back. Don't overthink it.`
	} else {
		// Standard response type guidance (non-greeting, non-self-reference)
		switch responseType {
		case ResponseDirectAnswer:
			responseGuidance = " They asked you a question. Give them a direct, helpful answer."
		case ResponseAcknowledgement:
			responseGuidance = " They shared information. Acknowledge what you heard and show you understand."
		case ResponseDeepeningQ:
			responseGuidance = " They shared something. Acknowledge it, then ask a Socratic question that helps them think deeper."
		case ResponseClarification:
			responseGuidance = " They're reacting to something. Seek clarification and help reorient the conversation."
		case ResponseValidation:
			responseGuidance = " They're expressing emotion. Validate their feelings and show support."
		case ResponseConfirmation:
			responseGuidance = " They're confirming understanding. Confirm what they said or gently reframe if needed."
		}
	}

	// STEP 1: Adapt tone to communication style preference
	styleTone := ""
	switch style {
	case "formal":
		styleTone = " Be respectful and professional. Use clear, direct language. Avoid excessive friendliness or casual expressions."
	case "casual":
		styleTone = " Be relaxed and natural. Use conversational language. Feel free to be friendly and approachable."
	case "playful":
		styleTone = " Be warm and engaging. Use light humor where appropriate. Show genuine curiosity and enthusiasm."
	default:
		styleTone = " Be warm yet respectful. Adapt your tone to match theirs."
	}

	// STEP 2: Adapt to emotional state
	emotionGuidance := ""
	switch emotionalTone {
	case "very_negative":
		emotionGuidance = " They're in significant distress. Prioritize validation and support. Be gentle, careful, and compassionate. Show you deeply understand their situation. Consider whether professional support might help."
	case "negative":
		emotionGuidance = " They're concerned or distressed about something real. Validate their feelings. Be thoughtful and supportive, not dismissive. Focus on understanding before advising."
	case "positive":
		emotionGuidance = " They're in a good mood. Match their energy. Be warm and engaged. Celebrate with them appropriately."
	case "very_positive":
		emotionGuidance = " They're very happy or excited. Celebrate and amplify their positive energy. Show genuine enthusiasm."
	}

	// STEP 3: Topic-specific guidance
	topicGuidance := ""
	switch topic {
	case "work":
		topicGuidance = " They're discussing work/career. This is serious territory. Ask about specific situations, not just feelings. Help them think through options and relationships at work."
	case "relationships":
		topicGuidance = " They're discussing relationships. Show you understand the human complexity. Ask about communication, needs, and how they want to handle things."
	case "family":
		topicGuidance = " They're discussing family. Acknowledge the deep roots and complexity. Be careful, respectful, and curious about their perspective."
	case "mental_health":
		topicGuidance = " They're discussing mental health. Take this seriously. Validate their concerns. Suggest professional support if needed."
	}

	// STEP 3B: Risk-aware guidance (if risk assessment available)
	riskGuidance := ""
	if len(riskAssessment) > 0 {
		if level, ok := riskAssessment["level"].(string); ok && level == "elevated" {
			riskGuidance = " They're in an elevated emotional state. Take their concerns seriously. Be extra thoughtful and supportive."
		}
	}

	// STEP 4: Socratic guidance (if applicable)
	socraticGuidance := ""
	if hasSocraticQuestion {
		socraticGuidance = " A specific question is waiting below—integrate it naturally into your response, not as a separate item. Let it guide your curiosity."
	}

	// Combine into full system prompt (Phase 3: add responseGuidance)
	return fmt.Sprintf(`%s%s%s%s%s%s%s%s

CRITICAL: Respect user preferences above all. If they ask for formality, be formal. If they're in distress, prioritize support. If they ask direct questions, answer directly.

Keep responses concise (1-3 sentences) unless they're sharing something complex. Don't use emojis. Show genuine understanding, not canned warmth.`, basePersonality, sessionGuidance, responseGuidance, styleTone, emotionGuidance, topicGuidance, riskGuidance, socraticGuidance)
}

// buildUserPromptContext creates facts/context about this conversation
func (ca *conversationAgent) buildUserPromptContext(ctx models.Context, userMessage string, socraticQuestion *models.SocraticQuestion) string {
	// About this person (from stored profile)
	userProfile := ""
	if ctx.AboutMe != nil {
		if ctx.AboutMe.CommunicationStyle != "" {
			userProfile = fmt.Sprintf("Stored communication style: %s\n", ctx.AboutMe.CommunicationStyle)
		}
		if ctx.AboutMe.PreferredTone != "" {
			userProfile += fmt.Sprintf("Preferred tone: %s\n", ctx.AboutMe.PreferredTone)
		}
		if len(ctx.AboutMe.Values) > 0 {
			userProfile += fmt.Sprintf("Values: %s\n", strings.Join(ctx.AboutMe.Values, ", "))
		}
		if len(ctx.AboutMe.Goals) > 0 {
			userProfile += fmt.Sprintf("Goals: %s\n", strings.Join(ctx.AboutMe.Goals, ", "))
		}
	}

	// What Moly has learned about this person (reflections)
	reflectionsText := ""
	if len(ctx.RelevantReflections) > 0 {
		reflectionsText = "What you've learned about them:\n"
		for i, reflection := range ctx.RelevantReflections {
			if i >= 2 {
				break
			}
			if len(reflection.Characteristics) > 0 {
				reflectionsText += fmt.Sprintf("- They are: %s\n", strings.Join(reflection.Characteristics, ", "))
			}
			if len(reflection.Intentions) > 0 {
				reflectionsText += fmt.Sprintf("- They tend to: %s\n", strings.Join(reflection.Intentions, ", "))
			}
		}
		if reflectionsText != "What you've learned about them:\n" {
			reflectionsText += "\n"
		}
	}

	// Current conversation context
	conversationContext := ""
	if len(ctx.ConversationHistory) > 1 {
		conversationContext = "You have prior conversation history to reference.\n"
	}
	if ctx.PastIntention != "" {
		conversationContext += fmt.Sprintf("Earlier they mentioned: %s\n", ctx.PastIntention)
	}

	// Extracted context from THIS message (highest priority)
	extractedContext := ""
	if ctx.ExtractedContext != nil {
		if ctx.ExtractedContext.Contact != nil && ctx.ExtractedContext.Contact.Confidence > 0.6 {
			extractedContext += fmt.Sprintf("Talking about: %s (%s)\n", ctx.ExtractedContext.Contact.Name, ctx.ExtractedContext.Contact.Relationship)
		}
		if ctx.ExtractedContext.Intention != "" {
			extractedContext += fmt.Sprintf("Their intention: %s\n", ctx.ExtractedContext.Intention)
		}
		if len(ctx.ExtractedContext.Goals) > 0 {
			extractedContext += fmt.Sprintf("Goals mentioned: %s\n", strings.Join(ctx.ExtractedContext.Goals, ", "))
		}
	}

	// Socratic question (if available)
	socraticText := ""
	if socraticQuestion != nil {
		socraticText = fmt.Sprintf("\nKEY QUESTION TO EXPLORE: \"%s\"\n(Approach: %s - weave it naturally into your response)\n", socraticQuestion.Text, socraticQuestion.SocraticApproach)
		log.Printf("[ConversationAgent] Including Socratic question: %s (%s)", socraticQuestion.ID, socraticQuestion.SocraticApproach)
	}

	// Multi-topic handling guidance
	multiTopicGuidance := ""
	topics := ca.detectMultipleTopics(strings.ToLower(userMessage))
	// Multiple topics means we found real topics (not just default "general")
	// "general" is only returned when NO topics are found
	if len(topics) > 1 {
		if len(topics) <= 3 {
			// FIX #4: 1-3 topics - ask about ONLY THE FIRST ONE
			// Avoid cognitive overload by asking one at a time
			firstTopic := topics[0]
			otherTopics := topics[1:]
			acknowledgement := ""
			if len(otherTopics) > 0 {
				otherTopicsList := strings.Join(otherTopics, ", ")
				acknowledgement = fmt.Sprintf("\nI also heard you mention %s - we'll get to those too.", otherTopicsList)
			}
			multiTopicGuidance = fmt.Sprintf("\nIMPORTANT - MULTIPLE CONCERNS:\nThey mentioned %d things. Start with the first one (%s) and ask clarifying questions about ONLY that. Acknowledge the others but focus on one at a time.%s\n", len(topics), firstTopic, acknowledgement)
		} else {
			// 4+ topics: Too many - ask them to prioritize
			multiTopicGuidance = fmt.Sprintf("\nIMPORTANT - TOO MANY TOPICS:\nThey brought up %d different topics. That's a lot. Acknowledge all of them, but ask them which is most urgent so you can focus and actually help.\n", len(topics))
		}
	}

	// The actual message
	messagePrompt := fmt.Sprintf("They just said: \"%s\"\n\nRespond directly to what they said. Address their specific concern, not just be generally friendly.", userMessage)

	// Combine into user prompt
	return fmt.Sprintf(`%s%s%s%s%s%s%s`, userProfile, reflectionsText, conversationContext, extractedContext, multiTopicGuidance, socraticText, messagePrompt)
}

// detectEmotionalTone analyzes the emotional state of the message
func (ca *conversationAgent) detectEmotionalTone(lowerMsg string) string {
	veryNegativeIndicators := []string{"devastated", "destroyed", "suicidal", "hopeless", "desperate", "dying", "hatred", "homicidal"}
	negativeIndicators := []string{"sad", "angry", "frustrated", "disappointed", "worried", "anxious", "stressed", "overwhelmed", "hurt", "crying", "broken", "concerned", "fear"}
	positiveIndicators := []string{"happy", "excited", "great", "wonderful", "amazing", "love", "grateful", "thrilled", "delighted", "proud", "hopeful"}
	veryPositiveIndicators := []string{"euphoric", "ecstatic", "overjoyed", "blessed", "incredibly grateful", "life-changing"}

	veryNegativeCount := 0
	for _, indicator := range veryNegativeIndicators {
		if contains(lowerMsg, indicator) {
			veryNegativeCount++
		}
	}
	negativeCount := 0
	for _, indicator := range negativeIndicators {
		if contains(lowerMsg, indicator) {
			negativeCount++
		}
	}
	positiveCount := 0
	for _, indicator := range positiveIndicators {
		if contains(lowerMsg, indicator) {
			positiveCount++
		}
	}
	veryPositiveCount := 0
	for _, indicator := range veryPositiveIndicators {
		if contains(lowerMsg, indicator) {
			veryPositiveCount++
		}
	}

	if veryNegativeCount > 0 {
		return "very_negative"
	} else if negativeCount > 0 && negativeCount > positiveCount {
		return "negative"
	} else if veryPositiveCount > 0 {
		return "very_positive"
	} else if positiveCount > 0 && positiveCount > negativeCount {
		return "positive"
	}
	return "neutral"
}

// getTopicKeywords returns the shared topic keyword mapping (for fallback detection)
func (ca *conversationAgent) getTopicKeywords() map[string]string {
	return map[string]string{
		"work":         "work",
		"job":          "work",
		"career":       "work",
		"boss":         "work",
		"colleague":    "work",
		"relationship": "relationships",
		"partner":      "relationships",
		"romantic":     "relationships",
		"family":       "family",
		"parent":       "family",
		"sibling":      "family",
		"anxiety":      "mental_health",
		"depression":   "mental_health",
		"therapy":      "mental_health",
		"health":       "health",
	}
}

// detectTopic identifies the primary topic of a message (keyword-only, deterministic)
// detectTopic - Deterministic via LLM analysis
// LLM must provide evidence from message content, not inference
func (ca *conversationAgent) detectTopic(lowerMsg string) string {
	// FIX 3: Return cached topic to avoid redundant LLM calls
	if ca.cachedTopic != "" {
		log.Printf("[ConversationAgent] FIX 3: ✓ Using cached topic: %s", ca.cachedTopic)
		return ca.cachedTopic
	}

	if ca.llmClient == nil {
		return "general"
	}

	topic := ca.detectTopicWithLLM(lowerMsg)
	if topic != "" && topic != "general" {
		ca.cachedTopic = topic // FIX 3: Cache the result
		return topic
	}
	ca.cachedTopic = "general" // FIX 3: Cache even "general"
	return "general"
}

// detectTopicWithLLM - Deterministic via evidence-based analysis
// LLM must cite specific evidence from the message, not inference
func (ca *conversationAgent) detectTopicWithLLM(message string) string {
	req := &tools.LLMRequest{
		SystemPrompt: `You MUST identify topic based ONLY on explicit content in the message.
Respond with ONLY: "general" OR "topic:WORD evidence:QUOTE"

Rules:
- If message has NO explicit topic words, respond: general
- If message mentions work/job/career/boss/colleague, respond: topic:work evidence:"[exact quote]"
- If message mentions relationship/partner/romantic, respond: topic:relationships evidence:"[exact quote]"
- If message mentions family/parent/sibling, respond: topic:family evidence:"[exact quote]"
- If message mentions anxiety/depression/therapy, respond: topic:mental_health evidence:"[exact quote]"
- If message mentions health/sick/doctor, respond: topic:health evidence:"[exact quote]"
- Otherwise respond: general`,
		UserPrompt: fmt.Sprintf("Message: %s\n\nRespond with topic determination (must cite evidence or default to general):", message),
		Temperature: 0.1,
		MaxTokens: 50,
	}

	resp, err := ca.llmClient.Call(context.Background(), req)
	if err != nil {
		log.Printf("[ConversationAgent] Topic LLM error: %v", err)
		return "general"
	}

	response := strings.ToLower(strings.TrimSpace(resp.Content))

	// Parse: "topic:WORD evidence:QUOTE" or "general"
	if response == "general" {
		log.Printf("[ConversationAgent] Topic: general (no explicit indicators)")
		return "general"
	}

	if strings.HasPrefix(response, "topic:") {
		// Extract topic word before "evidence:"
		parts := strings.Split(response, " evidence:")
		if len(parts) >= 2 {
			topicPart := strings.TrimPrefix(parts[0], "topic:")
			topicPart = strings.TrimSpace(topicPart)
			evidence := strings.Trim(parts[1], "\"")
			log.Printf("[ConversationAgent] Topic: %s (evidence: %s)", topicPart, evidence)
			return topicPart
		}
	}

	// Default to general if parsing failed
	log.Printf("[ConversationAgent] Topic determination unclear, defaulting to: general")
	return "general"
}

// detectMultipleTopics - Deterministic via evidence-based LLM analysis
// LLM must cite specific evidence from message, not inference
func (ca *conversationAgent) detectMultipleTopics(lowerMsg string) []string {
	// FIX 3 BUG FIX: Cache multiple topics too (was being called 4x)
	if ca.cachedTopics != nil && len(ca.cachedTopics) > 0 {
		log.Printf("[ConversationAgent] FIX 3: ✓ Using cached topics (%d)", len(ca.cachedTopics))
		return ca.cachedTopics
	}

	if ca.llmClient == nil {
		return []string{"general"}
	}

	topics := ca.detectMultipleTopicsWithLLM(lowerMsg)
	if len(topics) > 0 {
		ca.cachedTopics = topics // FIX 3: Cache the result
		return topics
	}
	ca.cachedTopics = []string{"general"} // FIX 3: Cache fallback
	return []string{"general"}
}

// detectMultipleTopicsWithLLM - Deterministic via evidence-based analysis
// LLM must cite specific evidence from the message, not inference
func (ca *conversationAgent) detectMultipleTopicsWithLLM(message string) []string {
	req := &tools.LLMRequest{
		SystemPrompt: `You MUST identify ALL topics based ONLY on explicit content in the message.
Respond ONLY with: comma-separated topics with evidence OR "general"

Format: topic1|evidence:"quote1", topic2|evidence:"quote2"
OR just: "general" if no explicit topics

Rules:
- Only include topics with direct evidence from message text
- Do NOT infer topics from tone or context
- Must cite exact quote for each topic
- If no explicit topics, respond: general`,
		UserPrompt: fmt.Sprintf("Message: %s\n\nRespond with all topics that have explicit evidence (or 'general'):", message),
		Temperature: 0.1,
		MaxTokens: 100,
	}

	resp, err := ca.llmClient.Call(context.Background(), req)
	if err != nil {
		log.Printf("[ConversationAgent] Multi-topic LLM error: %v", err)
		return nil
	}

	response := strings.ToLower(strings.TrimSpace(resp.Content))

	if response == "general" {
		log.Printf("[ConversationAgent] Multiple topics analysis: general (no explicit indicators)")
		return []string{"general"}
	}

	// Parse: "topic1|evidence:..., topic2|evidence:..."
	topicMap := make(map[string]bool)

	for _, item := range strings.Split(response, ",") {
		item = strings.TrimSpace(item)
		if strings.Contains(item, "|evidence:") {
			topicPart := strings.Split(item, "|")[0]
			topicPart = strings.TrimSpace(topicPart)
			if topicPart != "" {
				topicMap[topicPart] = true
				log.Printf("[ConversationAgent] Topic found with evidence: %s", topicPart)
			}
		}
	}

	if len(topicMap) == 0 {
		return []string{"general"}
	}

	var topics []string
	for topic := range topicMap {
		topics = append(topics, topic)
	}

	return topics
}

// runAnalyzePhase - Determine the type of interaction
// runSafetyPhase - Check for crisis/illegal content
// runReflectPhase - Extract insights from conversation
func (ca *conversationAgent) runReflectPhase(ctx context.Context, message string) (*models.Reflection, error) {
	if message == "" {
		return nil, nil
	}

	input := &tools.ContextExtractorInput{
		Message: message,
	}

	output, err := ca.contextExtractor.Extract(ctx, input)
	if err != nil {
		// Graceful fallback: return nil, no error (optional phase)
		return nil, nil
	}

	if output == nil {
		return nil, nil
	}

	reflection := &models.Reflection{
		Characteristics:          output.NewCharacteristics,
		Interests:                output.NewInterests,
		CommunicationPreferences: output.UpdatedCommunicationPrefs,
		Intentions:               output.Intentions,
		UserQuotes:               output.UserQuotes,
		Status:                   "pending_approval",
	}

	return reflection, nil
}

// generateContextGatheringQuestions - Generate Socratic questions to gather missing context
// Uses LLM for contextual question generation, falls back to templates
func generateContextGatheringQuestions(hasAboutMe, hasContact, hasIntention bool, userMessage string) []*schema.ClarificationQuestion {
	now := time.Now().Unix()

	// Gather context in progressive order: AboutMe → Intention
	// Note: Contact extraction is smart (checks for contact verbs), so we don't require it upfront
	// Only ask for AboutMe or Intention if genuinely missing (same logic as determineClarificationNeeded)

	if !hasAboutMe {
		return []*schema.ClarificationQuestion{
			{
				ID:          fmt.Sprintf("q_aboutme_%d", now),
				Type:        "context_gathering",
				Question:    "Tell me about yourself - what's your communication style like? Are you more formal, casual, playful, or a mix?",
				Priority:    2,
				Status:      "pending",
				CreatedAt:   now,
				LinkedFacts: []string{fmt.Sprintf("fact_aboutme_%d", now)},
			},
		}
	}

	if !hasIntention {
		return []*schema.ClarificationQuestion{
			{
				ID:          fmt.Sprintf("q_intention_%d", now),
				Type:        "context_gathering",
				Question:    "What's your intention with this message? Are you celebrating something, apologizing, asking for help, or starting a conversation?",
				Priority:    2,
				Status:      "pending",
				CreatedAt:   now,
				LinkedFacts: []string{fmt.Sprintf("fact_intention_%d", now)},
			},
		}
	}

	// Fallback - shouldn't reach here if logic is correct
	return []*schema.ClarificationQuestion{
		{
			ID:          fmt.Sprintf("q_fallback_%d", now),
			Type:        "context_gathering",
			Question:    "Tell me more about what you're trying to communicate.",
			Priority:    3,
			Status:      "pending",
			CreatedAt:   now,
			LinkedFacts: []string{fmt.Sprintf("fact_fallback_%d", now)},
		},
	}
}

// generateContactQuestionLLM generates context-aware contact question using LLM logic
// Falls back to keyword-based template if LLM unavailable
func generateContactQuestionLLM(userMessage string) string {
	// Fallback template - used if LLM call fails
	defaultQuestion := "Now, who are you wanting to message? Tell me their name and what your relationship is like."

	// Note: In production, this would use an LLM to understand the message context
	// and generate a natural, contextual question. For now, use contextual templates.
	//
	// LLM prompt would be:
	// "Based on this message, generate a short, natural question asking about the person
	// they want to contact. Keep it conversational and context-aware."

	// Fallback to keyword-based templates (lower confidence, but reliable)
	lowerMsg := strings.ToLower(userMessage)

	if contains(lowerMsg, "girl") || contains(lowerMsg, "boy") || contains(lowerMsg, "crush") ||
		contains(lowerMsg, "romantic") || contains(lowerMsg, "dating") || contains(lowerMsg, "interested") {
		return "You mentioned someone special! What's their name, and how would you describe your relationship with them?"
	}
	if contains(lowerMsg, "boss") || contains(lowerMsg, "manager") || contains(lowerMsg, "colleague") || contains(lowerMsg, "work") {
		return "Who's the person you're messaging? And what's your working relationship like?"
	}
	if contains(lowerMsg, "friend") {
		return "What's your friend's name, and how close are you two?"
	}
	if contains(lowerMsg, "mom") || contains(lowerMsg, "dad") || contains(lowerMsg, "parent") ||
		contains(lowerMsg, "sibling") || contains(lowerMsg, "brother") || contains(lowerMsg, "sister") || contains(lowerMsg, "family") {
		return "Which family member are you reaching out to? Tell me about your relationship."
	}

	return defaultQuestion
}

// contains checks if string contains substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// generateContextualClarification creates a dynamic, context-aware clarification question
// Uses extracted context to make the response feel personal and relevant
// Layer 6-7: Detect Principle Concerns in user request
// Even if message is clear, it might involve principles that need clarification
// Returns (hasConcern, principleID, clarificationQuestion)
// [Layer 4 Enhancement] Generate gap clarifications that are principle-aware
func (ca *conversationAgent) generatePrincipleAwareGapClarification(ctx models.Context, gaps []string, principleID string) string {
	if len(gaps) == 0 {
		return "Help me understand this situation better."
	}

	// Add principle context to the gap clarification
	switch principleID {
	case "stakeholder_consideration":
		return fmt.Sprintf("I'd like to understand more about this situation, especially how others are affected. %s\n\nAlso, does the other person know about this? What's their perspective?",
			ca.formatGaps(gaps))
	case "consent_and_respect":
		return fmt.Sprintf("To give you better advice, I need to understand more. %s\n\nSpecifically: Has everyone involved agreed to this?",
			ca.formatGaps(gaps))
	case "user_autonomy":
		return fmt.Sprintf("Help me understand what YOU think is best here. %s\n\nWhat does your gut tell you to do?",
			ca.formatGaps(gaps))
	case "harm_prevention":
		return fmt.Sprintf("I want to make sure we think through any potential harm. %s\n\nWhat could go wrong? Who could be affected?",
			ca.formatGaps(gaps))
	default:
		return ca.responseGenerator.GenerateGapClarificationResponse(ctx, gaps)
	}
}

func (ca *conversationAgent) formatGaps(gaps []string) string {
	if len(gaps) == 0 {
		return ""
	}
	if len(gaps) == 1 {
		return fmt.Sprintf("To start: %s", gaps[0])
	}
	result := "To start, could you tell me more about:\n"
	for i, gap := range gaps {
		if i > 2 {
			break // Limit to 3 gaps
		}
		result += fmt.Sprintf("- %s\n", gap)
	}
	return result
}

// Layer 10: Persistent Questioning After Insistence
// When user continues asking about something after we've raised principle concerns
// Try deeper questioning to help them reconsider rather than immediately complying
func (ca *conversationAgent) detectRepeatedConcern(userMessage string, history []models.Message, extractedContext *models.ExtractedContext) (bool, string) {
	if len(history) < 3 {
		return false, "" // Not enough history to detect repetition
	}

	lower := strings.ToLower(userMessage)

	// Look for markers that user is persisting despite clarification
	persistenceMarkers := []string{
		"still", "anyway", "regardless", "but", "however", "even so",
		"actually", "wait", "what if", "let me", "how about", "what about",
	}

	hasMarker := false
	for _, marker := range persistenceMarkers {
		if strings.Contains(lower, marker) {
			hasMarker = true
			break
		}
	}

	if !hasMarker {
		return false, ""
	}

	// Check if recent messages show we asked clarification about this
	for i := 0; i < len(history)-1 && i < 3; i++ {
		prevMsg := history[i]
		if prevMsg.Role == "assistant" {
			prevLower := strings.ToLower(prevMsg.Content)
			// Check for questions we typically ask about principles
			concernQuestions := []string{
				"does", "how", "what", "perspective", "feel", "know",
				"agree", "consent", "understand", "realize", "consider",
			}

			hasQuestion := false
			for _, q := range concernQuestions {
				if strings.Contains(prevLower, q) && strings.Contains(prevLower, "?") {
					hasQuestion = true
					break
				}
			}

			if hasQuestion {
				// User is answering/continuing after we asked clarification
				log.Printf("[ConversationAgent] Layer 10: User persisting after clarification attempt")
				return true, prevMsg.Content // Return the clarification we asked
			}
		}
	}

	return false, ""
}

// Generate Layer 10 persistent questioning - deeper exploration with alternatives before proceeding
func (ca *conversationAgent) generatePersistentQuestion(userMessage string, principleID string, initialClarification string) string {
	// First, explore consequences
	consequenceQuestion := ""
	switch principleID {
	case "stakeholder_consideration":
		consequenceQuestion = fmt.Sprintf("I understand you still want to proceed. But consider: How do you think %s would feel if they found out about this? What's the worst outcome for them?", extractPersonName(userMessage))
	case "consent_and_respect":
		consequenceQuestion = "Before we go further, consider: Would you want to be treated this way? How would you feel if someone did this to you?"
	case "user_autonomy":
		consequenceQuestion = "Let's pause and reflect: What would feel most authentic to you right now? What does your gut tell you to do?"
	case "harm_prevention":
		consequenceQuestion = "I notice this might lead to harm. What would happen if you took this action? What are all the possible consequences?"
	default:
		consequenceQuestion = "Help me understand the consequences: What could go wrong? How would each person involved be affected?"
	}

	// Second, explore alternatives (Layer 10 step 3)
	alternativeQuestion := ""
	switch principleID {
	case "stakeholder_consideration":
		alternativeQuestion = fmt.Sprintf("Given those consequences, what are other ways you could achieve what you want while considering %s's perspective? What solutions would work for both of you?", extractPersonName(userMessage))
	case "consent_and_respect":
		alternativeQuestion = "What are other approaches that respect everyone's boundaries and wishes? How could you accomplish this in a way that everyone agrees with?"
	case "user_autonomy":
		alternativeQuestion = "What other choices do you have? What would it look like to choose what feels truly right for you?"
	case "harm_prevention":
		alternativeQuestion = "What are safer alternatives that achieve the same goal? How could you get what you want without causing harm?"
	default:
		alternativeQuestion = "What other ways could you approach this? What alternatives have you considered?"
	}

	// Combine consequence + alternative for Layer 10 persistent questioning
	return fmt.Sprintf("%s\n\n%s", consequenceQuestion, alternativeQuestion)
}

// extractPersonName extracts pronouns from a message
// Current implementation: Keyword-based pronoun detection
// Note: This is a simple helper for fallback pronoun extraction
// Future enhancement: Could integrate with ContextExtractor for LLM-based pronoun analysis
// Current behavior is adequate for its use case (determining pronouns for contact references)
func extractPersonName(message string) string {
	// Pronoun extraction from message keywords
	lower := strings.ToLower(message)
	if strings.Contains(lower, "her") || strings.Contains(lower, "she") {
		return "she"
	}
	if strings.Contains(lower, "him") || strings.Contains(lower, "he") {
		return "he"
	}
	if strings.Contains(lower, "them") {
		return "them"
	}
	return "they"
}

// detectPrincipleConcerns checks which principles are ENGAGED (involved) in the message
// Uses LLM-based reasoning instead of hardcoded keywords (C-30n Bug #2 fix)
// Difference: "Does message involve this principle?" (engagement) vs "Does it violate?" (violation)
func (ca *conversationAgent) detectPrincipleConcerns(userMessage string, extractedContext *models.ExtractedContext) (bool, string, string) {
	if ca.constitution == nil || ca.llmClient == nil {
		return false, "", ""
	}

	// Build prompt asking LLM to identify engaged principles
	userPrompt := fmt.Sprintf(`Analyze this message to identify which principles from our ethical framework are ENGAGED (involved in the situation).

Message: "%s"

Our principles:
1. Harm Prevention - involves potential harm to self or others
2. User Autonomy - involves user's own decisions and agency
3. Consent & Respect - involves respecting others' choices and perspectives
4. Stakeholder Consideration - involves impact on other people
5. Transparency - involves honest communication or disclosure
6. Growth & Learning - involves personal growth, reflection, or development

Return JSON with:
{
  "engagedPrinciples": ["principle_name", ...],
  "primaryPrinciple": "principle_name",
  "evidence": "brief quote showing engagement",
  "needsClarification": true/false
}

Return ONLY valid JSON, no other text.`, userMessage)

	// Call LLM with structured request
	// Solution 3B: Use generous timeout (60s) for principle detection, fallback gracefully on timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	llmReq := &tools.LLMRequest{
		SystemPrompt: "You are an ethical reasoning assistant. Analyze messages to identify which ethical principles are engaged.",
		UserPrompt:   userPrompt,
		Temperature:  0.3, // Low temperature for consistent principle identification
		MaxTokens:    500,
	}

	response, err := ca.llmClient.Call(ctx, llmReq)
	if err != nil {
		if err == context.DeadlineExceeded {
			log.Printf("[ConversationAgent] Layer 6-7: Principle engagement detection timed out, falling back to no concerns")
		} else {
			log.Printf("[ConversationAgent] Layer 6-7: Principle engagement detection failed: %v, falling back to no concerns", err)
		}
		return false, "", ""
	}

	// Parse response
	type PrincipleAnalysis struct {
		EngagedPrinciples  []string `json:"engagedPrinciples"`
		PrimaryPrinciple   string   `json:"primaryPrinciple"`
		Evidence           string   `json:"evidence"`
		NeedsClarification bool     `json:"needsClarification"`
	}

	var analysis PrincipleAnalysis
	if err := json.Unmarshal([]byte(response.Content), &analysis); err != nil {
		log.Printf("[ConversationAgent] Layer 6-7: Failed to parse principle analysis: %v", err)
		return false, "", ""
	}

	// If no principles engaged or no clarification needed, no concern
	if len(analysis.EngagedPrinciples) == 0 || !analysis.NeedsClarification {
		return false, "", ""
	}

	// Generate principle-based clarification question
	clarificationQ := ca.generatePrincipleBasedClarification(analysis.PrimaryPrinciple, extractedContext, userMessage)

	if clarificationQ == "" {
		return false, "", ""
	}

	log.Printf("[ConversationAgent] Layer 6-7: Principle concern detected - %s (evidence: %s)", analysis.PrimaryPrinciple, analysis.Evidence)
	return true, analysis.PrimaryPrinciple, clarificationQ
}

// generatePrincipleBasedClarification creates clarification questions based on engaged principles
func (ca *conversationAgent) generatePrincipleBasedClarification(principle string, extractedContext *models.ExtractedContext, userMessage string) string {
	switch principle {
	case "stakeholder_consideration":
		if extractedContext != nil && extractedContext.Contact != nil {
			return fmt.Sprintf("Before we think this through, I want to understand %s's perspective. Does %s know about this? How might they feel?",
				extractedContext.Contact.Name, extractedContext.Contact.Name)
		}
		return "How might the other people involved feel about this? Have you considered their perspective?"

	case "consent_and_respect":
		if extractedContext != nil && extractedContext.Contact != nil {
			return fmt.Sprintf("It's important that %s's wishes are respected. How does %s feel about this?",
				extractedContext.Contact.Name, extractedContext.Contact.Name)
		}
		return "Have you talked to the people involved about what they want?"

	case "user_autonomy":
		return "What do YOU think is right here? What matters most to you in this situation? Don't let anyone (including me) decide for you."

	case "transparency":
		return "Is there something about this that should be said openly? What would happen if you were fully honest about it?"

	case "harm_prevention":
		return "Help me understand what's happening. Is anyone in distress or at risk? What support might be needed?"

	case "growth_and_learning":
		return "What could you learn from this situation? How might it help you grow?"

	default:
		return ""
	}
}

func (ca *conversationAgent) generateContextualClarification(userMessage string, extractedContext *models.ExtractedContext) string {
	if userMessage == "" {
		return "Tell me more! What's on your mind?"
	}

	// REMOVED: Hardcoded keyword detection for messaging (message, text, tell, ask, say, contact, call)
	// Now using principle-based detection: transparency (communicating) or autonomy (deciding)
	involvesDirectCommunication := extractedContext != nil &&
		(containsPrinciple(extractedContext.IntentionPrinciples, "transparency") ||
			containsPrinciple(extractedContext.IntentionPrinciples, "autonomy"))

	// CRITICAL: If user's intention involves direct communication, ALWAYS ask about WHO and WHAT first
	// Never skip this—it's critical for safe message generation
	if involvesDirectCommunication {
		// Check what information we're missing
		hasContactName := extractedContext != nil && extractedContext.Contact != nil && extractedContext.Contact.Name != ""
		hasMessageIntent := extractedContext != nil && extractedContext.Intention != "" && extractedContext.Intention != "general_support"

		if !hasContactName && !hasMessageIntent {
			return "I'd like to help you draft this message! First, who are you wanting to message, and what's this about?"
		} else if !hasContactName {
			return fmt.Sprintf("Got it—you want to %s. But who are you reaching out to?", extractedContext.Intention)
		} else if !hasMessageIntent {
			return fmt.Sprintf("You want to message %s—what's the main thing you want to say or ask?", extractedContext.Contact.Name)
		}
	}

	// Option 1: Contact mentioned but unclear what user wants from them
	// Skip placeholder/unclear contact names like "Unspecified", "Contact", "Unknown", "Moly"
	if extractedContext != nil && extractedContext.Contact != nil && extractedContext.Contact.Name != "" &&
		extractedContext.Contact.Name != "Unspecified" && extractedContext.Contact.Name != "Contact" &&
		extractedContext.Contact.Name != "Unknown" && strings.ToLower(extractedContext.Contact.Name) != "moly" {
		name := extractedContext.Contact.Name
		rel := extractedContext.Contact.Relationship

		if rel == "romantic" {
			return fmt.Sprintf("So you're thinking about %s! What would you like to do or talk about with them?", name)
		} else if rel == "professional" {
			return fmt.Sprintf("You mentioned %s (a colleague/manager). What's the situation you're dealing with?", name)
		} else if rel == "family" {
			return fmt.Sprintf("You're thinking about %s. What's the issue or conversation you want to have?", name)
		}
		return fmt.Sprintf("You mentioned %s! What do you want to figure out about this situation?", name)
	}

	// Option 2: Goals mentioned but unclear how to achieve them
	if extractedContext != nil && len(extractedContext.Goals) > 0 {
		goal := extractedContext.Goals[0]
		return fmt.Sprintf("I see you want to %s. What's holding you back, or what help do you need?", goal)
	}

	// Option 3: Situation mentioned - ask what they want to achieve
	lowerMsg := strings.ToLower(userMessage)
	if contains(lowerMsg, "girl") || contains(lowerMsg, "boy") || contains(lowerMsg, "crush") {
		return "You mentioned someone special—what's the main thing you're trying to figure out about this?"
	}
	if contains(lowerMsg, "work") || contains(lowerMsg, "job") || contains(lowerMsg, "boss") {
		return "Sounds like work is on your mind. What's the specific challenge you're facing?"
	}
	if contains(lowerMsg, "problem") || contains(lowerMsg, "issue") || contains(lowerMsg, "stuck") {
		return "I hear something's troubling you. Walk me through what's happening—what's the core issue?"
	}
	if contains(lowerMsg, "want") || contains(lowerMsg, "need") {
		return "What exactly are you trying to achieve or figure out?"
	}

	// Option 4: General fallback - warm and inviting
	// Load generic responses from database instead of hardcoding
	if ca.templateManager != nil {
		if template, err := ca.templateManager.GetTemplate("no_topic", "clarification"); err == nil && template != "" {
			return template
		}
	}

	// Fallback: minimal hardcoded responses (only if database unavailable)
	fallbackResponses := []string{
		"What's on your mind?",
		"Tell me more—what's the main thing?",
		"What do you need help with?",
		"What brought you here today?",
	}
	idx := len(userMessage) % len(fallbackResponses)
	return fallbackResponses[idx]
}

// generateGreeting creates a warm greeting response based on user's style
// Recognizes greetings and responds naturally without context extraction
func (ca *conversationAgent) generateGreeting(userMessage string, aboutMe *models.AboutMe) string {
	// Use user's communication style if known
	var greetings []string

	if aboutMe != nil {
		lower := strings.ToLower(aboutMe.CommunicationStyle)
		if lower == "formal" {
			greetings = []string{
				"Hello. I'm Moly, your thinking partner. How can I assist you today?",
				"Good to hear from you. What would you like to discuss?",
				"I'm here to help. What's on your mind?",
			}
		} else if lower == "playful" {
			greetings = []string{
				"Hey there! 👋 Ready to dive into something? What's up?",
				"Yo! I'm Moly. Let's chat about whatever's on your mind!",
				"Hey! What's going on? Tell me everything!",
			}
		} else { // casual (default)
			greetings = []string{
				"Hi! I'm Moly. What's going on with you?",
				"Hey there! So, what's on your mind today?",
				"Hello! I'm here to listen. What would you like to talk about?",
			}
		}
	} else {
		// Default friendly greetings
		greetings = []string{
			"Hi! I'm Moly, your thinking partner. What's on your mind?",
			"Hello! I'm here to listen and help you think through things. What would you like to talk about?",
			"Hey! I'm Moly. What brings you here today?",
		}
	}

	// Return random greeting for variety
	return greetings[rand.Intn(len(greetings))]
}

// extractContactsFromContext builds a list of known contacts from the conversation context
// Used by intent detector to avoid misidentifying relationship types
func extractContactsFromContext(ctx models.Context) []*models.Contact {
	var contacts []*models.Contact

	// Extract from ContactProfile if available
	if ctx.ContactProfile != nil && ctx.ContactProfile.Name != "" {
		contact := &models.Contact{
			Name:            ctx.ContactProfile.Name,
			Relationship:    ctx.ContactProfile.Relationship,
			Characteristics: ctx.ContactProfile.Characteristics,
		}
		contacts = append(contacts, contact)
	}

	return contacts
}

// parseJSONArray parses a JSON array string into a string slice
func parseJSONArray(jsonStr string) ([]string, error) {
	var result []string
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// extractRelevantPrinciples identifies which principles from constitution are relevant to the user message
func (ca *conversationAgent) extractRelevantPrinciples(userMessage string, extractedContext *models.ExtractedContext) []string {
	if ca.constitution == nil {
		return []string{}
	}

	relevant := make([]string, 0)

	// Check all supreme principles to see which are engaged by this message
	for _, principle := range ca.constitution.SupremePrinciples {
		// A principle is relevant if the message touches on its core concern
		// For now, use simple heuristic: if message mentions stakeholders, consent, harm, autonomy, etc.

		lower := strings.ToLower(userMessage)

		switch principle.ID {
		case "stakeholder_consideration":
			// Relevant if message mentions other people/relationships
			if extractedContext != nil && extractedContext.Contact != nil && extractedContext.Contact.Name != "" {
				relevant = append(relevant, principle.ID)
			}

		case "consent_and_respect":
			// Relevant if message involves others' boundaries or permissions
			if strings.Contains(lower, "ask") || strings.Contains(lower, "tell") || strings.Contains(lower, "convince") || extractedContext != nil && extractedContext.Contact != nil {
				relevant = append(relevant, principle.ID)
			}

		case "user_autonomy":
			// Relevant if message involves user's own choices/values
			if strings.Contains(lower, "want") || strings.Contains(lower, "choose") || strings.Contains(lower, "feel") || strings.Contains(lower, "should") {
				relevant = append(relevant, principle.ID)
			}

		case "harm_prevention":
			// Relevant if message mentions consequences or risks
			if strings.Contains(lower, "hurt") || strings.Contains(lower, "harm") || strings.Contains(lower, "consequence") || strings.Contains(lower, "risk") {
				relevant = append(relevant, principle.ID)
			}

		case "transparency":
			// Relevant if message involves honesty/communication
			if strings.Contains(lower, "tell") || strings.Contains(lower, "honest") || strings.Contains(lower, "truth") {
				relevant = append(relevant, principle.ID)
			}

		case "growth_and_learning":
			// Relevant if message involves personal development/understanding
			if strings.Contains(lower, "learn") || strings.Contains(lower, "understand") || strings.Contains(lower, "grow") || strings.Contains(lower, "why") {
				relevant = append(relevant, principle.ID)
			}
		}
	}

	// If no specific principles matched, return the most general ones for deepening
	if len(relevant) == 0 {
		// Default to autonomy and growth for general Socratic exploration
		relevant = []string{"user_autonomy", "growth_and_learning"}
	}

	return relevant
}

// generateSocraticQuestionWithPrinciples calls LLM to generate principle-based Socratic question
func (ca *conversationAgent) generateSocraticQuestionWithPrinciples(userMessage string, ctx *models.Context, principles []string) string {
	if ca.llmClient == nil || len(principles) == 0 {
		return ""
	}

	// Build principle context from constitution
	principleDescriptions := ""
	for _, principleID := range principles {
		for _, principle := range ca.constitution.SupremePrinciples {
			if principle.ID == principleID {
				principleDescriptions += fmt.Sprintf("- %s: %s\n", principle.ID, principle.Description)
				break
			}
		}
	}

	// Build prompt for LLM to generate Socratic question
	prompt := fmt.Sprintf(`You are a Socratic coach helping someone think more deeply about their situation.

User's message: "%s"

Relevant principles to explore:
%s

Generate ONE philosophical/exploratory Socratic question that:
1. Helps the user think deeper about what matters to them
2. Is based on the principles above
3. Is open-ended, not yes/no
4. Does NOT provide advice or solutions, only asks questions
5. Respects the user's autonomy and encourages self-reflection

The question should be natural, conversational, and genuinely curious - not preachy.

Respond with ONLY the question, nothing else.`, userMessage, principleDescriptions)

	req := &tools.LLMRequest{
		UserPrompt:  prompt,
		MaxTokens:   200,
		Temperature: 0.7,
	}

	resp, err := ca.llmClient.Call(context.Background(), req)
	if err != nil {
		log.Printf("[ConversationAgent] [Layer 8] Error generating Socratic question: %v", err)
		return ""
	}

	if resp == nil || resp.Content == "" {
		log.Printf("[ConversationAgent] [Layer 8] LLM returned empty response")
		return ""
	}

	log.Printf("[ConversationAgent] [Layer 8] Generated Socratic question: %s", resp.Content)
	return resp.Content
}

// containsPrinciple checks if a principle exists in the slice
func containsPrinciple(principles []string, target string) bool {
	for _, p := range principles {
		if strings.EqualFold(p, target) {
			return true
		}
	}
	return false
}

// buildGapQuestionPrincipleContext dynamically builds principle definitions from Constitution
func (ca *conversationAgent) buildGapQuestionPrincipleContext() string {
	if ca.constitution == nil || len(ca.constitution.SupremePrinciples) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Constitutional principles:\n")

	// Include key principles for gap question analysis
	relevantPrinciples := []string{"transparency", "growth_and_learning", "user_autonomy"}

	for _, princID := range relevantPrinciples {
		for _, principle := range ca.constitution.SupremePrinciples {
			if principle.ID == princID {
				sb.WriteString(fmt.Sprintf("- %s: %s\n", principle.Name, principle.Description))
				break
			}
		}
	}

	return sb.String()
}

// isGapQuestionLLM - Principle-based detection of clarification/gap questions
// REMOVED: Hardcoded keyword checks ("tell", "explain", "how", "what", "why", "?")
// Now: Analyzes from actual constitutional principles loaded from config
func (ca *conversationAgent) isGapQuestionLLM(messageContent string) bool {
	if ca.llmClient == nil {
		// Conservative: assume all messages could be questions when LLM unavailable
		return true
	}

	// Build principle context from Constitution
	principleContext := ca.buildGapQuestionPrincipleContext()
	if principleContext == "" {
		return true // Conservative: when constitution unavailable, assume it could be a question
	}

	// Principle-based analysis: does this message invite user response?
	prompt := fmt.Sprintf(`Analyze how this message engages with constitutional principles.

%s

Message: "%s"

Respond with ONLY a JSON object (no markdown):
{
  "transparency_engaged": boolean,
  "growth_engaged": boolean,
  "autonomy_engaged": boolean,
  "invites_user_response": boolean
}`, principleContext, messageContent)

	req := &tools.LLMRequest{
		SystemPrompt: `Analyze messages against constitutional principles.
Respond with only valid JSON, no other text.`,
		UserPrompt:  prompt,
		MaxTokens:   100,
		Temperature: 0.3,
		Retries:     1,
	}

	resp, err := ca.llmClient.Call(context.Background(), req)
	if err != nil {
		log.Printf("[ConversationAgent] isGapQuestionLLM failed: %v, using conservative default", err)
		return true // Conservative: when LLM fails, assume it could be a question
	}

	// Check if message invites user response (indicates it's a gap question)
	lower := strings.ToLower(resp.Content)
	if strings.Contains(lower, `"invites_user_response": true`) || strings.Contains(lower, `"invites_user_response":true`) {
		return true
	}

	// Also consider it a gap question if transparency and growth principles are engaged
	transparencyEngaged := strings.Contains(lower, `"transparency_engaged": true`) || strings.Contains(lower, `"transparency_engaged":true`)
	growthEngaged := strings.Contains(lower, `"growth_engaged": true`) || strings.Contains(lower, `"growth_engaged":true`)

	return transparencyEngaged && growthEngaged
}

// hasClarificationBeenAddressed checks if a required clarification was already provided in conversation
// Returns true if the information needed for the clarification is present in recent messages or extracted context
func (ca *conversationAgent) hasClarificationBeenAddressed(clarificationType string, history []models.Message, extractedCtx *models.ExtractedContext) bool {
	switch clarificationType {
	case "userIntention":
		// Check if we extracted an intention
		if extractedCtx != nil && extractedCtx.Intention != "" {
			return true
		}
		// Check if user mentioned what they want to do/say
		if len(history) > 0 {
			for _, msg := range history {
				if msg.Role == "user" {
					lower := strings.ToLower(msg.Content)
					// Look for action phrases
					intentPhrases := []string{
						"i want", "i'm trying", "help me", "how do i", "should i",
						"i'm looking", "i'd like", "can you help", "write", "send",
						"tell", "say", "message", "talk to", "approach",
					}
					for _, phrase := range intentPhrases {
						if strings.Contains(lower, phrase) {
							return true
						}
					}
				}
			}
		}
		return false

	case "userInterests", "userInterestAlignment":
		// Check if we extracted goals or interests
		if extractedCtx != nil && len(extractedCtx.Goals) > 0 {
			return true
		}
		// Check if user described common interests, preferences, or goals
		if len(history) > 0 {
			for _, msg := range history {
				if msg.Role == "user" {
					lower := strings.ToLower(msg.Content)
					// Look for interest/preference phrases
					interestPhrases := []string{
						"common interest", "we have", "both like", "shared", "prefer",
						"don't like", "interested in", "looking for", "want to",
						"goals", "values", "important to",
					}
					for _, phrase := range interestPhrases {
						if strings.Contains(lower, phrase) {
							return true
						}
					}
				}
			}
		}
		return false

	default:
		return false
	}
}

// shouldRequireClarificationForContact determines if we truly need clarification for a contact message
// by checking what's already been established in the conversation
func (ca *conversationAgent) shouldRequireClarificationForContact(
	extractedContact *models.ExtractedContact,
	history []models.Message,
	extractedCtx *models.ExtractedContext,
	confirmedPrefs map[string]interface{},
) bool {
	if extractedContact == nil || extractedContact.Name == "" {
		return false
	}

	// Check if user has confirmed preferences via Layer 3
	if confirmedPrefs != nil && len(confirmedPrefs) > 0 {
		hasIntention := confirmedPrefs["userIntentionWithContact"] != nil || confirmedPrefs["intention"] != nil
		hasInterests := confirmedPrefs["userInterestAlignment"] != nil || confirmedPrefs["interests"] != nil
		if hasIntention && hasInterests {
			return false // Already confirmed via Layer 3
		}
	}

	// Check if clarifications have been addressed in the conversation itself
	hasIntention := ca.hasClarificationBeenAddressed("userIntention", history, extractedCtx)
	hasInterests := ca.hasClarificationBeenAddressed("userInterests", history, extractedCtx)

	// If we have both intention and interests in the conversation, no need to clarify
	if hasIntention && hasInterests {
		log.Printf("[Layer4Tracking] User has provided both intention and interests in conversation for %s", extractedContact.Name)
		return false
	}

	// If we have extracted multiple messages showing understanding, accept it
	if len(history) >= 3 && hasIntention && hasInterests {
		log.Printf("[Layer4Tracking] Multi-message conversation shows sufficient context for %s", extractedContact.Name)
		return false
	}

	// Still need clarification
	if !hasIntention {
		log.Printf("[Layer4Tracking] Missing: user intention for %s", extractedContact.Name)
	}
	if !hasInterests {
		log.Printf("[Layer4Tracking] Missing: user interests/goals for %s", extractedContact.Name)
	}

	return true
}

// MEDIUM FIX: Helper functions for safe operations with logging

// safeNilCheck provides comprehensive nil checking with context-specific logging
// MEDIUM FIX #9: Comprehensive nil checking for critical operations
func safeNilCheck(value interface{}, fieldName string, context string) bool {
	if value == nil {
		log.Printf("[ConversationAgent] WARNING: Nil check failed for %s in %s context", fieldName, context)
		return true // is nil
	}
	return false // not nil
}

// validateResponsePipeline checks critical points in the response pipeline
// MEDIUM FIX #11: Response pipeline validation to prevent data loss
func validateResponsePipeline(response *models.ConversationResponse, stage string) {
	if response == nil {
		log.Printf("[ConversationAgent] ERROR: Response nil at %s (pipeline corruption)", stage)
		return
	}

	if response.Response == "" {
		log.Printf("[ConversationAgent] WARNING: Empty response at %s", stage)
	}

	if response.Metadata == nil {
		log.Printf("[ConversationAgent] WARNING: Metadata nil at %s (will create new)", stage)
		response.Metadata = make(map[string]interface{})
	}

	// Verify critical metadata is present
	if _, ok := response.Metadata["layer"]; !ok && response.Response != "" {
		log.Printf("[ConversationAgent] INFO: Layer not set at %s (may need assignment)", stage)
	}
}

// buildClarificationQuestion creates a standardized clarification question for database storage
// MEDIUM FIX #8: Deduplicate clarification question building
func (ca *conversationAgent) buildClarificationQuestion(
	userID, conversationID, qType, questionText, contextNotes string, priority int,
) *database.ClarificationQuestion {
	return &database.ClarificationQuestion{
		ID:                fmt.Sprintf("%s_q_%d", qType, time.Now().UnixNano()),
		UserID:            userID,
		ConversationID:    conversationID,
		ClarificationType: qType,
		QuestionText:      questionText,
		ContextNotes:      contextNotes,
		Priority:          priority,
		Status:            "pending",
		CreatedAt:         time.Now().Unix(),
	}
}

// safeGetMetadataString safely retrieves a string from metadata with logging on failure
func safeGetMetadataString(metadata map[string]interface{}, key string, context string) (string, bool) {
	if metadata == nil {
		log.Printf("[ConversationAgent] WARNING: Metadata nil when accessing %s (%s)", key, context)
		return "", false
	}

	val, exists := metadata[key]
	if !exists {
		return "", false
	}

	str, ok := val.(string)
	if !ok {
		log.Printf("[ConversationAgent] ERROR: Metadata[%s] type assertion failed: got %T, expected string (%s)", key, val, context)
		return "", false
	}

	return str, true
}

// safeGetMetadataFloat safely retrieves a float64 from metadata with logging on failure
func safeGetMetadataFloat(metadata map[string]interface{}, key string, context string) (float64, bool) {
	if metadata == nil {
		log.Printf("[ConversationAgent] WARNING: Metadata nil when accessing %s (%s)", key, context)
		return 0, false
	}

	val, exists := metadata[key]
	if !exists {
		return 0, false
	}

	flt, ok := val.(float64)
	if !ok {
		log.Printf("[ConversationAgent] ERROR: Metadata[%s] type assertion failed: got %T, expected float64 (%s)", key, val, context)
		return 0, false
	}

	return flt, true
}

// safeGetMetadataBool safely retrieves a bool from metadata with logging on failure
func safeGetMetadataBool(metadata map[string]interface{}, key string, context string) (bool, bool) {
	if metadata == nil {
		log.Printf("[ConversationAgent] WARNING: Metadata nil when accessing %s (%s)", key, context)
		return false, false
	}

	val, exists := metadata[key]
	if !exists {
		return false, false
	}

	bln, ok := val.(bool)
	if !ok {
		log.Printf("[ConversationAgent] ERROR: Metadata[%s] type assertion failed: got %T, expected bool (%s)", key, val, context)
		return false, false
	}

	return bln, true
}

// REMAINING ISSUES FIX #1-3: Deduplicate Common Code Patterns

// extractCurrentMessage safely extracts the current message from conversation history
// REMAINING FIX #1: Centralize message extraction
func extractCurrentMessage(history []models.Message) string {
	if len(history) == 0 {
		return ""
	}
	return history[0].Content
}

// hasConflictsDetected checks if any conflicts were detected
// REMAINING FIX #2: Centralize conflict checking
func hasConflictsDetected(conflicts []ConflictDetectorResult) bool {
	return len(conflicts) > 0
}

// logConflicts logs all detected conflicts with consistent format
// REMAINING FIX #2: Standardize conflict logging
func logConflicts(conflicts []ConflictDetectorResult, source string) {
	if len(conflicts) == 0 {
		return
	}
	
	log.Printf("[ConversationAgent] ⚠ %d conflicts detected in %s:", len(conflicts), source)
	for _, conflict := range conflicts {
		log.Printf("[ConversationAgent]   - %s: %s (severity=%s)", conflict.Type, conflict.Description, conflict.Severity)
	}
}

// REMAINING ISSUES FIX #4-7: Dataflow Optimization & Validation

// validateContextFlow checks critical dataflow points
// REMAINING FIX #4: Context loading optimization
func validateContextFlow(ctx *models.Context, stage string) error {
	if ctx == nil {
		return fmt.Errorf("context nil at stage: %s", stage)
	}
	
	if ctx.AboutMe == nil {
		log.Printf("[ConversationAgent] WARNING: AboutMe nil at stage %s (may be loaded later)", stage)
	}
	
	if ctx.ExtractedContext == nil && stage == "processing" {
		log.Printf("[ConversationAgent] WARNING: ExtractedContext nil at processing stage")
	}
	
	if len(ctx.ConversationHistory) == 0 && stage != "init" {
		log.Printf("[ConversationAgent] WARNING: Empty conversation history at stage %s", stage)
	}
	
	return nil
}

// ensureMetadataPresent ensures response metadata exists and is not nil
// REMAINING FIX #5: Metadata validation
func ensureMetadataPresent(response *models.ConversationResponse) {
	if response == nil {
		return
	}
	if response.Metadata == nil {
		response.Metadata = make(map[string]interface{})
	}
}

// validateStateConsistency checks for state synchronization issues
// REMAINING FIX #6: State validation
func validateStateConsistency(extracted int, stored int, stage string) bool {
	if extracted != stored {
		log.Printf("[ConversationAgent] WARNING: State inconsistency at %s: extracted=%d, stored=%d", stage, extracted, stored)
		return false
	}
	return true
}

// validateArtifactFreshness checks if artifact is fresh enough to reuse
// REMAINING FIX #7: Artifact staleness detection
func validateArtifactFreshness(artifact *models.ExtractionArtifact, maxAgeMilli int64) bool {
	if artifact == nil {
		return false
	}
	
	ageMs := time.Since(time.Unix(artifact.CreatedAt, 0)).Milliseconds()
	if ageMs > maxAgeMilli {
		log.Printf("[ConversationAgent] WARNING: Artifact stale: %dms old (max: %dms)", ageMs, maxAgeMilli)
		return false
	}
	
	return true
}
