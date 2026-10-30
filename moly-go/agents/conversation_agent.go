package agents

import (
	"context"
	"fmt"
	"log"
	"moly/privacy"
	"strings"
	"time"

	"moly/config"
	"moly/database"
	"moly/models"
	"moly/tools"
)

// conversationAgent - Implements the 5-phase conversation flow
type conversationAgent struct {
	llmClient               tools.LLMProvider
	constitutionalEvaluator *tools.ConstitutionalEvaluator
	responseGenerator       *tools.ResponseGenerator      // Generates contextual responses instead of hardcoded text
	intentDetector          *LLMIntentDetector            // LLM-driven intent detection (no hardcoded patterns)
	socraticSelector        *SocraticQuestionSelector     // Optional: for Socratic question selection
	constitution            *models.Constitution          // Optional: for principle-guided generation
	db                      *database.Database            // Optional: for conflict detection
	inlineResolver          *tools.InlineConflictResolver // Optional: for Phase 2 inline resolution
	clarityAnalyzer         *MessageClarityAnalyzer       // NEW: Diagnostic message clarity analysis
	templateManager         *ResponseTemplateManager      // For database-driven response templates
}

// NewConversationAgent - Create new conversation agent
func NewConversationAgent(llm tools.LLMProvider) (models.ConversationAgent, error) {
	// LLM client is optional - agent will generate basic suggestions without it
	return &conversationAgent{
		llmClient:               llm,
		constitutionalEvaluator: nil,                             // Will be set via SetConstitution after initialization
		responseGenerator:       tools.NewResponseGenerator(llm), // Generates natural, contextual responses
		intentDetector:          NewLLMIntentDetector(llm),       // LLM-driven intent detection
		socraticSelector:        nil,                             // Optional - set via SetSocraticSelector if available
	}, nil
}

// SetSocraticSelector injects the Socratic question selector (optional)
func (ca *conversationAgent) SetSocraticSelector(selector *SocraticQuestionSelector) {
	if ca != nil {
		ca.socraticSelector = selector
		log.Printf("[ConversationAgent] Socratic selector initialized")
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
		"LLM client":               ca.llmClient != nil,
		"Constitution":             ca.constitution != nil,
		"Constitutional evaluator": ca.constitutionalEvaluator != nil,
		"Response generator":       ca.responseGenerator != nil,
		"Intent detector":          ca.intentDetector != nil,
		"Database":                 ca.db != nil,
		"Clarity analyzer":         ca.clarityAnalyzer != nil,
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

	// Check array length before indexing
	if len(questions) == 0 {
		log.Printf("[ConversationAgent] FIX #66: No questions to record answer for")
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
				// The refusal is the safety stage's own alert: short, naming who to ask, no help lines, no question.
				alert := ctx.PrecomputedSafetyVerdict
				if alert == nil || alert.Message == "" {
					alert = &models.SafetyAlert{
						AlertType:     "illegal",
						Severity:      "high",
						Title:         "I can't help with that",
						Message:       "I can't help with that.",
						IsObviousHarm: true,
					}
				}
				current := ""
				if len(ctx.ConversationHistory) > 0 {
					current = ctx.ConversationHistory[0].Content
				}
				if q := ca.questionBeforeRefusal(ctx, current, alert); q != "" {
					response.Response = q
					response.Phase = "clarification"
					response.Metadata["refusalQuestion"] = true
					response.Metadata["replyExit"] = "refusal_question"
					response.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())
					log.Printf("[ConversationAgent] ✅ Returning early with a reflective question in place of the refusal")
					return response, nil
				}
				localized := *alert // the alert the user sees carries the refusal in the language of their words
				localized.Message = ca.refusalInUsersLanguage(current, alert.Message)
				response.Response = localized.Message
				response.Phase = "safety_alert"
				response.SafetyAlert = &localized
				response.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())
				log.Printf("[ConversationAgent] ✅ Returning early with denial response")
				return response, nil
			}

			// Care comes before everything that only improves the result. The turn after a refusal of a crisis statement starts
			// with one check-in, whatever the next message says (found in a dynamic run: "I was just joking, help me write to my
			// boss" went straight back to the task). A message that rates as a crisis is not answered with a question about a name.
			if prev := LastAssistantMessage(ctx.ConversationHistory); prev != nil && !ctx.IsGreeting {
				if t, _ := prev.Metadata["safetyAlertType"].(string); t == "crisis" {
					response.Response = CareCheckIn
					response.Phase = "context_gathering"
					response.Metadata["careCheckIn"] = true
					response.Metadata["replyExit"] = string(KindClarity)
					response.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())
					log.Printf("[ConversationAgent] Care check-in: the last reply refused a crisis statement")
					return response, nil
				}
			}
			skipNameForCrisis := false
			if ctx.PendingNameLabel != "" && ca.clarityAnalyzer != nil && analysisCtx != nil {
				if ca.clarityAnalyzer.ModelPriority(ctx.BoundedAnalysisContext, analysisCtx.CurrentMessage) == "crisis" {
					skipNameForCrisis = true
					log.Printf("[ConversationAgent] The message rates as a crisis: the name question waits")
				}
			}

			// (name gate): a person without a name is asked for it first. The question is the
			// only content of this reply; advice and gap questions wait until the name is known or declined.
			// Safety (Layer 2) has already run above.
			// STEP 1 (ORCHESTRATOR_DESIGN.md): the kind of reply is decided by DecideReply, from explicit facts.
			// The facts known here are: greeting, unanswered name, name answered, open gap. Deepening is not known yet
			// (it is decided later), so the plan recorded here is definitive only for deny, greeting and name.
			plan := DecideReply(ReplyFacts{
				Greeting:       ctx.IsGreeting,
				AboutMoly:      ctx.AboutMoly,
				NameUnanswered: ctx.PendingNameLabel != "" && !skipNameForCrisis,
				NameAnswered:   ctx.NameAnswered,
				ConfirmFact:    ctx.DoubtfulFact != "",
				OpenGap:        len(ctx.Gaps) > 0,
				Bypass:         ctx.ResultNow,
			})
			response.Metadata["replyPolicy"] = string(plan.Kind)
			response.Metadata["replyPolicyReason"] = plan.Reason
			if plan.Kind == KindNameQuestion {
				molyLast := ""
				if prev := LastAssistantMessage(ctx.ConversationHistory); prev != nil {
					molyLast = prev.Content
				}
				response.Response = ca.nameQuestion(analysisCtx.CurrentMessage, molyLast, ctx.PendingNameLabel)
				response.Phase = "context_gathering"
				response.Metadata["nameQuestion"] = true
				response.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())
				log.Printf("[ConversationAgent] Reply policy: %s (%s)", plan.Kind, plan.Reason)
				return response, nil
			}

			// ENHANCE: Read Layer 2 full verdict details (violation principles, severity, reasoning)
			if layerCtx.Layer2 != nil && layerCtx.Layer2.Verdict != nil {
				log.Printf("[ConversationAgent] ✓ Reading Layer 2 evaluation: severity=%s, confidence=%.2f, obvious_harm=%v",
					layerCtx.Layer2.Verdict.OverallSeverity, layerCtx.Layer2.Verdict.Confidence, layerCtx.Layer2.IsObviousHarm)

				// Add principle violation details to metadata
				if layerCtx.Layer2.Verdict != nil && len(layerCtx.Layer2.Verdict.MatchedPrinciples) > 0 {
					principleNames := make([]string, 0)
					for _, pm := range layerCtx.Layer2.Verdict.MatchedPrinciples {
						principleNames = append(principleNames, pm.PrincipleID)
					}
					response.Metadata["violatedPrinciples"] = principleNames
					response.Metadata["principleViolationDetails"] = layerCtx.Layer2.Verdict.MatchedPrinciples
					log.Printf("[ConversationAgent]   - Violated principles: %v", principleNames)
				}

				response.Metadata["evaluationSeverity"] = layerCtx.Layer2.Verdict.OverallSeverity
				response.Metadata["evaluationConfidence"] = layerCtx.Layer2.Verdict.Confidence
				response.Metadata["evaluationReasoning"] = layerCtx.Layer2.Verdict.Reasoning
				response.Metadata["evaluationLLMReasoning"] = layerCtx.Layer2.Verdict.LLMReasoning
			}

			// Check Layer 2 ShouldProceedToL6 gate
			if layerCtx.Layer2 != nil && !layerCtx.Layer2.ShouldProceedToL6 {
				log.Printf("[ConversationAgent] 🚫 Layer 2 gates L5/6 access: ShouldProceedToL6=false")
				response.Metadata["layer2Gate"] = "blocked"
			}

			// NEW: Read Layer 4 gaps from orchestrator (DATA FLOW FIX)
			if layerCtx.Layer4 != nil && len(layerCtx.Layer4.DetectedGaps) > 0 {
				log.Printf("[ConversationAgent] ✓ Reading Layer 4 gaps from orchestrator: %d gaps detected", len(layerCtx.Layer4.DetectedGaps))

				// Check previous gap count to avoid resurfacing same gaps
				if analysisCtx != nil && analysisCtx.PreviousResponseMetadata != nil {
					if prevGapCount, ok := analysisCtx.PreviousResponseMetadata["gapCount"].(float64); ok && prevGapCount > 0 {
						log.Printf("[ConversationAgent] FIX #24: Previous message had %.0f gaps, current has %d - tracking for pattern",
							prevGapCount, len(layerCtx.Layer4.DetectedGaps))
						response.Metadata["previousGapCount"] = prevGapCount
					}
				}

				// Extract gap descriptions and populate ctx.Gaps
				for _, gap := range layerCtx.Layer4.DetectedGaps {
					if gap.Severity == "low" {
						// "nice to know": the answer would only be less detailed without it, so it is not asked (the gap generator
						// is told to leave these out; one that still arrives is dropped here, not asked in place of a better one)
						log.Printf("[ConversationAgent] Gap left unasked (severity=low)")
						continue
					}
					ctx.Gaps = append(ctx.Gaps, gap.Description)
					log.Printf("[ConversationAgent] ✓ Gap added: %s (severity=%s, confidence=%.2f)",
						gap.Description, gap.Severity, gap.Confidence)
				}

				log.Printf("[ConversationAgent] ✓ Populated ctx.Gaps from orchestrator: %d gaps total", len(ctx.Gaps))

				// Check Layer 4 ShouldClarify gate
				// Layer 4 signals whether to ask clarifications based on maturity
				if !layerCtx.Layer4.ShouldClarify {
					log.Printf("[ConversationAgent] ✓ FIX #18: Layer 4 gate ShouldClarify=false - bypassing gap clarification")
					response.Metadata["layer4SkipClarification"] = true
					// Continue without asking gaps - context too immature
				} else {
					response.Metadata["layer4ClarifyAllowed"] = true
				}
			}

			// NEW: Read Layer 11: Denial protocol (DATA FLOW FIX)
			if layerCtx.Layer11 != nil && layerCtx.Layer11.ShouldDeny {
				log.Printf("[ConversationAgent] 🚫 Reading Layer 11: DENIAL PROTOCOL TRIGGERED")
				log.Printf("[ConversationAgent]   - Reason: %s", layerCtx.Layer11.Reason)
				log.Printf("[ConversationAgent]   - Resources: %v", layerCtx.Layer11.Resources)

				// COMPLETE Reference previous resources for consistency
				if analysisCtx != nil && analysisCtx.PreviousResponseMetadata != nil {
					if prevResources, ok := analysisCtx.PreviousResponseMetadata["denialResources"].([]interface{}); ok && len(prevResources) > 0 {
						log.Printf("[ConversationAgent] COMPLETE FIX #21: Previous denial offered %d resources, maintaining consistency",
							len(prevResources))
						response.Metadata["previousDenialResources"] = prevResources
					}
					if prevAlt, ok := analysisCtx.PreviousResponseMetadata["denialAltSuggestion"].(string); ok && prevAlt != "" {
						log.Printf("[ConversationAgent] COMPLETE FIX #21: Previous denial suggested: %s", prevAlt)
						response.Metadata["previousDenialSuggestion"] = prevAlt
					}
				}

				response.Metadata["shouldDeny"] = true
				response.Metadata["denialReason"] = layerCtx.Layer11.Reason
				response.Metadata["denialResources"] = layerCtx.Layer11.Resources
				response.Metadata["denialAltSuggestion"] = layerCtx.Layer11.AltSuggestion
				log.Printf("[ConversationAgent] ✓ Denial protocol will be applied")
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

	log.Printf("[ConversationAgent] User message received (len=%d)", len(userMessage))

	// Meta-instructions about Moly are detected once per message, in the request handler (main.go), which updates the focus.

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
	if ctx.MessageIntent != "" {
		// the intent was decided before the layers ran (main.go); it is not decided twice
		intentAnalysis = IntentAnalysis{Intent: Intent(ctx.MessageIntent), Confidence: ctx.MessageIntentConfidence, AdjustsDraft: ctx.MessageAdjustsDraft}
	} else if ca.intentDetector != nil {
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

	// The profile, contact and intention come from the stored profile and the model-based extraction above.
	// Keyword fallbacks for preferences, values, style, contact and intention were removed: meaning is not decided by word lists.
	if aboutMe == nil {
		aboutMe = &models.AboutMe{UserID: ctx.AboutMe.UserID}
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

	// STEP 1 (ORCHESTRATOR_DESIGN.md): every reply kind is one decision. DecideReply picks the kind from facts that
	// probes find in precedence order; reply_exits.go renders the question kinds. Nothing below may ask a question
	// that the decision did not choose: what follows only writes the help text.
	contactFacts := exitInputs{
		intent: intentAnalysis,
	}
	done, exitDeepen := ca.replyByExits(ctx, analysisCtx, userMessage, contactFacts, response, startTime)
	if done {
		return response, nil
	}
	shouldDeepen := exitDeepen
	responseType := RouteResponse(intentAnalysis.Intent, shouldDeepen)
	log.Printf("[ConversationAgent] Routing to response type: %s (shouldDeepen=%v)", responseType, shouldDeepen)

	// WORKFLOW DECISION TREE
	// Route based on understanding level: what do we know vs. what's missing?

	type ResponseWorkflow string
	const (
		WorkflowCrisis          ResponseWorkflow = "crisis"       // Safety incident - already handled earlier
		WorkflowAckWithSocratic ResponseWorkflow = "ack_socratic" // Acknowledge + Socratic deepening
		WorkflowAckOnly         ResponseWorkflow = "ack_only"     // Acknowledge without question
	)

	// Assess understanding level (what's ACTUALLY missing, not just what was extracted)
	intentUnclear := intentAnalysis.Confidence < 0.5 // Intent detection failed
	isFirstMessage := ctx.IsFirstMessageInConversation

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

		workflow = WorkflowAckWithSocratic
		log.Printf("[ConversationAgent] Workflow: Intent response with deepening")

		// Priority 3: LOW-CONFIDENCE INTENT (<0.6) OR UNCLEAR - Ask clarification
	} else if intentUnclear || intentAnalysis.Confidence < 0.6 {
		// Unclear intent is a question kind (KindIntentCheck); if it did not answer, acknowledge only.
		workflow = WorkflowAckOnly
		log.Printf("[ConversationAgent] Workflow: Acknowledge only (intent confidence=%.2f)", intentAnalysis.Confidence)

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
		response.Response = "I'm listening."
		log.Printf("[ConversationAgent] No LLM/ResponseGenerator available, using fallback response")
	} else {
		log.Printf("[ConversationAgent] Executing workflow: %s", workflow)

		var generatedResponse string

		if workflow == WorkflowAckOnly {
			// Acknowledge what user said without asking questions (first message or safe default)
			var genErr error
			if generatedResponse, genErr = ca.generateConversationalResponse(ctx, analysisCtx, userMessage, nil, "validation"); genErr != nil {
				return nil, genErr
			}
			log.Printf("[ConversationAgent] [✓] Generated acknowledgment (no question): %.100s...", generatedResponse)
		} else {
			// Default: full response with potential deepening
			// [Layer 5] Check for pending conflicts first
			var conflictQuestion string
			var pendingConflictID int64
			if ca.inlineResolver != nil && aboutMe != nil && aboutMe.UserID != "" {
				conflictInfo := ca.inlineResolver.CheckAndAskForConflicts(aboutMe.UserID, ctx.ConversationID)
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

				{
					// Generate response, optionally with Socratic deepening
					var socraticQuestion *models.SocraticQuestion

					// ORDERING: the Socratic question is asked only when the deepening gates allowed it (shouldDeepen).
					if socraticAllowed(workflow == WorkflowAckWithSocratic, shouldDeepen, ca.socraticSelector != nil, hasAboutMe, hasContact, hasIntention) {
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
									riskLevel := "none"
									recordErr := qhRepo.RecordQuestion(ctx.AboutMe.UserID, ctx.ConversationID, question, emotionState, riskLevel)
									if recordErr != nil {
										log.Printf("[ConversationAgent] Warning: Failed to record Socratic question: %v", recordErr)
									}
								}
							}
						}
					}

					var genErr error
					if generatedResponse, genErr = ca.generateConversationalResponse(ctx, analysisCtx, userMessage, socraticQuestion, responseType); genErr != nil {
						return nil, genErr
					}
					privacy.Register(generatedResponse)
					log.Printf("[ConversationAgent] [✓] Generated %s response: %.100s...", responseType, generatedResponse)
				}
			}
		}
		response.Response = ca.groundedReply(userMessage, generatedResponse, dialogueBlock(analysisCtx, ctx))

		// Track suggestions offered in response (basic implementation)
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

				// Log suggestion offering for analytics
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
		// The insights come from the extraction of this message (traits, interests, aim), which already ran: a second extraction of
		// the same message used to be made here (plan step 3, 2026-10-15).
		if reflection := reflectionFromExtraction(ctx.ExtractedContext); reflection != nil {
			response.Reflection = reflection
			log.Printf("[ConversationAgent] [✓] Learned about user: %d characteristics", len(reflection.Characteristics))

			// Save reflection to database for learning system
			if ctx.ConversationID != "" && ctx.AboutMe != nil && ctx.AboutMe.UserID != "" && ca.db != nil {
				// Populate the reflection fields that the extraction does not give
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
		// a short request ("Yes") is read against the question it answers
		molyBefore, _ := models.LastMolyMessage(models.MessagesBefore(ctx.ConversationHistory, userMessage))
		lastQuestion := ""
		if molyBefore != "" {
			lastQuestion = "Moly's message before it: \"" + molyBefore + "\"\n"
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		// The evaluator judges what a person asks or plans. Moly's own draft is judged with the request it answers, and is told that the
		// text is Moly's reply: a message the user asked for to send to someone else is not the user's plan to harm that person.
		verdict, err := ca.constitutionalEvaluator.EvaluateWithContextAndMaturity(ctx, response.Response,
			lastQuestion+"The user asked Moly: \""+userMessage+"\"\nThe text to evaluate is Moly's reply, which the user will read. It may contain wording the user could send to another person.", 1.0)
		cancel()

		if err != nil {
			// Same fail-open strategy as user input validation: treat as allowed on LLM error
			log.Printf("[ConversationAgent] Constitutional analysis failed (%v), treating response as allowed", err)
		} else if verdict != nil && len(verdict.MatchedPrinciples) > 0 {
			log.Printf("[ConversationAgent] Constitutional violation detected: severity=%s, principles=%d",
				verdict.OverallSeverity, len(verdict.MatchedPrinciples))

			// Map verdict severity to intervention level. Only obvious harm replaces the reply, as for the user's message an
			// ambiguous finding on a reply the user asked for took the result away (found live, 2026-10-15: a first message to a woman the
			// user liked, judged "high" at confidence 0.45, was replaced by "What would you like to focus on first?").
			if verdict.IsObviousHarm || verdict.OverallSeverity == "critical" {
				// BLOCK: Replace response with safe response
				log.Printf("[ConversationAgent] 🚫 BLOCK: %s - replacing the draft with one question", verdict.OverallSeverity)
				response.Response = ca.replacementForBlockedReply(userMessage)
				response.Metadata["ethicalIntervention"] = "blocked"
				response.Metadata["blockSeverity"] = verdict.OverallSeverity
				response.Metadata["blockedPrinciples"] = fmt.Sprintf("%d principles violated", len(verdict.MatchedPrinciples))
				response.Metadata["originalResponseBlocked"] = true
				log.Printf("[ConversationAgent] [✓] Blocked response recorded (severity: %s)", verdict.OverallSeverity)

			} else {
				// Keep response but log for internal monitoring (don't show warning to user)
				log.Printf("[ConversationAgent] ℹ️ Principle consideration on the reply, not obvious harm: %s (kept)", verdict.OverallSeverity)
				// Don't add warning metadata - user should see clean response
				log.Printf("[ConversationAgent] [✓] Logged principle consideration (severity: %s, not shown to user)", verdict.OverallSeverity)
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

	// Moral values are now incorporated into response generation prompt
	// No post-generation ethical gate needed - trust the LLM to generate helpful, safe responses
	log.Printf("[ConversationAgent] [✓] Response complete with moral values integrated in generation")

	response.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())
	log.Printf("[ConversationAgent] [✓] Response ready in %d ms", response.ProcessingTimeMs)

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

// generateConversationalResponse creates a natural, context-aware response to the user
// ARCHITECTURE: Adaptive SystemPrompt (tone/personality) + UserPrompt (facts/context)
func (ca *conversationAgent) generateConversationalResponse(
	ctx models.Context,
	analysisCtx *models.AnalysisContext,
	userMessage string,
	socraticQuestion *models.SocraticQuestion,
	responseType ResponseType, // Phase 3: Use routing decision to shape response
) (string, error) {
	if ca.llmClient == nil {
		return "", fmt.Errorf("no AI model is available")
	}

	// STEP 1: DETECT USER CONTEXT (for adaptive tone)
	// Detect communication style preference
	communicationStyle := "warm"
	// the style the user saved in Settings is their explicit choice; the style read from a message is used only without one
	if ctx.AboutMe != nil && ctx.AboutMe.CommunicationStyle != "" {
		communicationStyle = ctx.AboutMe.CommunicationStyle
	} else if ctx.ExtractedContext != nil && ctx.ExtractedContext.Style != nil && ctx.ExtractedContext.Style.Confidence > 0.6 {
		communicationStyle = ctx.ExtractedContext.Style.Style
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
	systemPrompt := ca.buildAdaptiveSystemPrompt(communicationStyle, socraticQuestion != nil, ctx.IsFirstMessageInConversation, responseType, hasSelfReference)
	if ctx.AboutMe != nil {
		if ins := toneInstruction(ctx.AboutMe.PreferredTone); ins != "" {
			systemPrompt += "\n\nTONE: " + ins
		}
	}
	maxTokens := 150
	if ctx.ResultNow {
		systemPrompt += resultNowGuidance
		maxTokens = 400
	}
	if ctx.AboutMoly {
		systemPrompt += "\n\n" + ProductKnowledge + "\nAnswer in a few short sentences and do not ask a question."
		maxTokens = 300
	}
	log.Printf("[ConversationAgent] SystemPrompt adapted: style=%s responseType=%s (isFirstMessage=%v, hasSelfReference=%v)", communicationStyle, responseType, ctx.IsFirstMessageInConversation, hasSelfReference)

	// STEP 3: BUILD USERPROMPT (facts and context for this conversation)
	userPrompt := ca.buildUserPromptContext(ctx, analysisCtx, userMessage, socraticQuestion)
	if ctx.ResultNow {
		userPrompt = resultNowRequestBlock(ctx.ConversationHistory) + userPrompt
	}

	// STEP 4: SEND TO LLM
	req := &tools.LLMRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Temperature:  0.7,
		MaxTokens:    maxTokens,
		Effort:       "high", // the reply the user reads: thinking on, and strict instruction following (no added facts, one question)
		Role:         tools.RoleReply,
	}

	resp, err := ca.llmClient.Call(context.Background(), req)
	if err != nil {
		// No canned reply: it would look like an answer (and be saved as one) while the model, the key or the connection is down. The turn
		// fails, is not saved, and the person is told to try again.
		log.Printf("[ConversationAgent] LLM call failed: %v", err)
		return "", fmt.Errorf("the AI model could not be reached: %w", err)
	}

	if resp == nil || strings.TrimSpace(resp.Content) == "" {
		return "", fmt.Errorf("the AI model answered with nothing")
	}

	return withoutPlaceholderLines(strings.TrimSpace(resp.Content)), nil
}

// buildAdaptiveSystemPrompt creates a personality/tone prompt based on user context
// This becomes the PRIMARY instruction to the LLM (higher priority than UserPrompt)
// Phase 3: Accepts responseType to tailor response approach
func (ca *conversationAgent) buildAdaptiveSystemPrompt(style string, hasSocraticQuestion bool, isFirstMessageOfSession bool, responseType ResponseType, hasSelfReference bool) string {
	// Base personality - Moly is always a good listener
	basePersonality := "You are Moly, a thoughtful listener and communication coach." + readsNoPages + tools.StatesOnlyWhatIsStated + tools.InUsersLanguage

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
- Speak in the first person
- Be personal and warm, not clinical
- Show personality and genuine engagement

Answer what was said to you in your own words, in the user's language: a question to you gets your honest view, thanks get a short acknowledgement. Do not reuse stock phrases.

Key principle: Direct address means the user sees you as a person/coach, not just a service.
Respond with warmth and personality.`

		if isGreeting {
			responseGuidance += ` Additionally, this is a GREETING. Acknowledge it warmly and simply first.

Greeting Response Pattern:
1. First: Greet back warmly, in the user's language and in your own words
2. Optional: Brief acknowledgment of intent
3. Skip: Analysis, questions, or over-explanation

IMPORTANT: Greetings are not prompts for context gathering.
Just greet the person back, then optionally continue the conversation naturally.`
		}
	} else if isGreeting {
		// Greeting without direct self-reference
		responseGuidance = ` This is a GREETING. Respond warmly and simply with acknowledgment.

Greeting Response Pattern:
1. Simple warm greeting back, in the user's language and in your own words
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
			// Reached only when every question exit declined: what had to be asked was asked (found live, 2026-10-15: this reply asked
			// again which detail of her profile the user liked, a question already asked, outside the budget and the repeat check).
			responseGuidance = " They shared something or reacted to something. Respond to what they said and move their goal forward with what you know. Do not ask a question: every question that was needed has already been asked."
		case ResponseValidation:
			responseGuidance = " They're expressing emotion. Validate their feelings and show support."
		case ResponseConfirmation:
			responseGuidance = " They're confirming understanding. Confirm what they said or gently reframe if needed."
		}
	}

	// A reply that is not a question reply asks nothing: every question that was needed has been asked by the reply exits (found in dynamic
	// runs: the help text asked again what the question replies had asked two turns before).
	if !hasSocraticQuestion && responseType != ResponseDeepeningQ && !isGreeting {
		responseGuidance += " Ask the user no question in this reply: every question that was needed has already been asked."
	}

	// STEP 1: Adapt tone to communication style preference
	styleTone := styleInstruction(style)

	// STEP 2: How they feel and what they talk about is read by the model from their words, in the dialogue. (It used to be decided
	// by word lists and two more model calls that saw only the bare lower-cased message.)
	emotionGuidance := " Read how they feel from what they say, in the light of the dialogue, and answer that: if they are in distress, validate and support before advising (do not diagnose, and do not point to help lines or services unless they ask); if they are glad, share it. If they describe a situation in their life, ask about the concrete situation and what they need, not about feelings in general."
	topicGuidance := ""

	// STEP 4: Socratic guidance (if applicable)
	socraticGuidance := ""
	if hasSocraticQuestion {
		socraticGuidance = " A specific question is waiting below—integrate it naturally into your response, not as a separate item. Let it guide your curiosity."
	}

	// Combine into full system prompt (Phase 3: add responseGuidance)
	return fmt.Sprintf(`%s%s%s%s%s%s%s

CRITICAL: Respect user preferences above all. If they ask for formality, be formal. If they're in distress, prioritize support. If they ask direct questions, answer directly.

Keep responses concise (1-3 sentences) unless they're sharing something complex or ask you to write something for them. Don't use emojis. Show genuine understanding, not canned warmth.

WHEN YOU ASK THE USER SOMETHING: ask exactly one question about one thing. Do not join two questions with "and" or "or", and do not offer a list of options to choose from. (A question inside a message you write for them belongs to that message.)

IF THE USER MAY BE IN DANGER OR DISTRESS: ask one caring question about their safety, or say that a specialist or someone they trust can help. Never write a phone number, a hotline, a crisis line or a website; the product decided to give none.

ENOUGH TO WRITE: when what you know is enough to write the message the user needs, write it now instead of asking for more; a detail you lack is left out of the message, not asked for.

WRITING FOR THE USER: when they ask you to write a message or a note, write it ready to send, in their voice, using only what they told you. Do not invent facts, events or feelings they did not give. Do not use placeholders such as [Your Name]; if you do not know a name, leave it out.`, basePersonality, sessionGuidance, responseGuidance, styleTone, emotionGuidance, topicGuidance, socraticGuidance)
}

// buildUserPromptContext creates facts/context about this conversation
func (ca *conversationAgent) buildUserPromptContext(ctx models.Context, analysisCtx *models.AnalysisContext, userMessage string, socraticQuestion *models.SocraticQuestion) string {
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
		if len(ctx.AboutMe.Characteristics) > 0 {
			userProfile += fmt.Sprintf("About themselves: %s\n", strings.Join(ctx.AboutMe.Characteristics, ", "))
		}
		if len(ctx.AboutMe.Goals) > 0 {
			userProfile += fmt.Sprintf("Goals: %s\n", strings.Join(ctx.AboutMe.Goals, ", "))
		}
		if len(ctx.AboutMe.UserInstructions) > 0 {
			userProfile += fmt.Sprintf("How they want to be understood: %s\n", strings.Join(ctx.AboutMe.UserInstructions, ", "))
		}
	}

	// Include Moly's self-awareness (user's feedback about the system)
	systemPreferencesText := ""
	if ctx.SystemContext != nil {
		if len(ctx.SystemContext.SystemInstructions) > 0 {
			systemPreferencesText += fmt.Sprintf("System behavior instructions: %s\n", strings.Join(ctx.SystemContext.SystemInstructions, ", "))
		}
		if len(ctx.SystemContext.UserDirectives) > 0 {
			systemPreferencesText += fmt.Sprintf("User's preferences for Moly: %s\n", strings.Join(ctx.SystemContext.UserDirectives, ", "))
		}
		if ctx.SystemContext.PreferredInteractionStyle != "" {
			systemPreferencesText += fmt.Sprintf("Preferred interaction style: %s\n", ctx.SystemContext.PreferredInteractionStyle)
		}
		if len(ctx.SystemContext.UserFeedback) > 0 {
			systemPreferencesText += fmt.Sprintf("User's feedback: %s\n", strings.Join(ctx.SystemContext.UserFeedback, ", "))
		}
		if systemPreferencesText != "" {
			systemPreferencesText += "\n"
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
	// The dialogue itself: summary, goal, people, what was learned, open questions and the last messages (Moly's last reply
	// included, so a comment on a draft is read against the draft). Before this the reply model was told only that
	// "you have prior conversation history" and never saw any of it.
	conversationContext := ""
	if analysisCtx != nil {
		conversationContext = analysisCtx.Dialogue.Text()
	} else if len(ctx.ConversationHistory) > 1 {
		molyLast, _ := models.LastMolyMessage(models.MessagesBefore(ctx.ConversationHistory, userMessage))
		conversationContext = (&models.DialoguePacket{MolyLast: molyLast}).Text()
	}
	if ctx.PastIntention != "" {
		conversationContext += fmt.Sprintf("Earlier they mentioned: %s\n", ctx.PastIntention)
	}

	// Extracted context from THIS message (highest priority)
	extractedContext := ""
	if ctx.ExtractedContext != nil {
		if ctx.ExtractedContext.Contact != nil && ctx.ExtractedContext.Contact.Confidence > 0.6 {
			// What the person is comes from the relations (whose father, whose boss); the bare kind of relationship ("family") would
			// read as the user's own family.
			if ctx.ContactProfile != nil && ctx.ContactProfile.RelationsText() != "" {
				extractedContext += fmt.Sprintf("Talking about: %s (%s)\n", ctx.ExtractedContext.Contact.Name, ctx.ContactProfile.RelationsText())
			} else {
				extractedContext += fmt.Sprintf("Talking about: %s (%s)\n", ctx.ExtractedContext.Contact.Name, ctx.ExtractedContext.Contact.Relationship)
			}

			// WHAT Context: What does user want/need with this contact?
			if ctx.ContactProfile != nil && len(ctx.ContactProfile.InvolvedInIntentions) > 0 {
				extractedContext += fmt.Sprintf("User wants to: %s\n", strings.Join(ctx.ContactProfile.InvolvedInIntentions, ", "))
			}
			if ctx.ContactProfile != nil && ctx.ContactProfile.RelationsText() != "" {
				extractedContext += fmt.Sprintf("What they are: %s\n", ctx.ContactProfile.RelationsText())
			} else if ctx.ContactProfile != nil && ctx.ContactProfile.ContactRole != "" {
				extractedContext += fmt.Sprintf("What they are to the user: %s\n", ctx.ContactProfile.ContactRole)
			}
			if ctx.ContactProfile != nil && len(ctx.ContactProfile.PastSuccesses) > 0 {
				extractedContext += fmt.Sprintf("Past help: %s\n", strings.Join(ctx.ContactProfile.PastSuccesses, ", "))
			}
		}
		if ctx.ExtractedContext.Intention != "" {
			extractedContext += fmt.Sprintf("Their intention: %s\n", ctx.ExtractedContext.Intention)
		}
		if len(ctx.ExtractedContext.Goals) > 0 {
			extractedContext += fmt.Sprintf("Goals mentioned: %s\n", strings.Join(ctx.ExtractedContext.Goals, ", "))
		}
	}

	// Accumulated context from previous messages (Phase 1 - loop architecture)
	accumulatedContext := ""
	if analysisCtx != nil && len(analysisCtx.AccumulatedExtractedEntities) > 0 {
		accumulatedContext = "Context from previous messages in this conversation:\n"

		accumulatedContext += formatAccumulatedEntities(analysisCtx.AccumulatedExtractedEntities)

		if accumulatedContext != "Context from previous messages in this conversation:\n" {
			accumulatedContext += "\n"
			log.Printf("[ConversationAgent] FIX #1: Including %d accumulated entities in LLM prompt", len(analysisCtx.AccumulatedExtractedEntities))
		} else {
			accumulatedContext = ""
		}
	}

	// Socratic question (if available)
	socraticText := ""
	if socraticQuestion != nil {
		socraticText = fmt.Sprintf("\nKEY QUESTION TO EXPLORE: \"%s\"\n(Approach: %s - weave it naturally into your response)\n", socraticQuestion.Text, socraticQuestion.SocraticApproach)
		log.Printf("[ConversationAgent] Including Socratic question: %s (%s)", socraticQuestion.ID, socraticQuestion.SocraticApproach)
	}

	multiTopicGuidance := "" // one question per reply is enforced by singleQuestion; the dialogue shows what is open

	// The actual message
	messagePrompt := fmt.Sprintf("They just said: \"%s\"\n\nRespond directly to what they said. Address their specific concern, not just be generally friendly.", userMessage)
	if conversationContext != "" {
		messagePrompt += "\nRead what they said as the next turn of the dialogue above: a comment such as \"too formal\" or a short reply such as \"yes\" is about your last message, not a new subject."
	}

	// Combine into user prompt (includes system preferences from SystemContext)
	// Order - accumulated context comes after conversation history but before current extraction
	return fmt.Sprintf(`%s%s%s%s%s%s%s%s%s`, userProfile, systemPreferencesText, reflectionsText, conversationContext, accumulatedContext, extractedContext, multiTopicGuidance, socraticText, messagePrompt)
}

// runAnalyzePhase - Determine the type of interaction
// runSafetyPhase - Check for crisis/illegal content
// reflectionFromExtraction builds the insights about the user from what the extraction of this message found: their traits, their
// interests, and the aim they stated with the words that show it. It returns nil when the extraction found nothing about them.
func reflectionFromExtraction(ec *models.ExtractedContext) *models.Reflection {
	if ec == nil {
		return nil
	}
	reflection := &models.Reflection{
		Characteristics: ec.UserCharacteristics,
		Interests:       ec.UserInterests,
		Status:          "pending_approval",
	}
	if ec.Intention != "" {
		reflection.Intentions = []string{ec.Intention}
		if ec.IntentionEvidence != "" {
			reflection.UserQuotes = []string{ec.IntentionEvidence}
		}
	}
	if len(reflection.Characteristics) == 0 && len(reflection.Interests) == 0 && len(reflection.Intentions) == 0 {
		return nil
	}
	return reflection
}

// Layer 10: Persistent Questioning After Insistence
// When user continues asking about something after we've raised principle concerns
// Try deeper questioning to help them reconsider rather than immediately complying
func (ca *conversationAgent) detectRepeatedConcern(userMessage string, history []models.Message, extractedContext *models.ExtractedContext) (bool, string) {
	// The history is [current, oldest ... newest]. The question Moly asked last is the newest assistant entry.
	var last *models.Message
	for i := len(history) - 1; i >= 1; i-- {
		if history[i].Role == "assistant" {
			last = &history[i]
			break
		}
	}
	// Persistence only exists after Moly raised a principle concern. The reply kind is recorded in the stored
	// metadata; no word in either message decides this.
	if last == nil || ca.llmClient == nil {
		return false, ""
	}
	kind, _ := last.Metadata["replyExit"].(string)
	if ReplyKind(kind) != KindPrinciple && ReplyKind(kind) != KindPersistent {
		return false, ""
	}
	req := &tools.LLMRequest{
		SystemPrompt: "You judge whether a user keeps to the same course of action after a concern was raised.",
		UserPrompt: "Moly asked: \"" + last.Content + "\"\nThe user replied: \"" + userMessage + "\"\n\n" +
			"Is the user pressing on with the same course of action without engaging with the concern? " +
			"An answer to the question, or a change of subject, is not persisting.\n" +
			"Respond with ONLY JSON: {\"persisting\": true|false}",
		MaxTokens:   60,
		Temperature: 0.1,
	}
	resp, err := ca.llmClient.Call(context.Background(), req)
	if err != nil {
		log.Printf("[ConversationAgent] Layer 10: persistence check failed: %v", err)
		return false, ""
	}
	var out struct {
		Persisting bool `json:"persisting"`
	}
	if err := tools.SafeJSONParse("PersistenceCheck", []byte(resp.Content), &out); err != nil || !out.Persisting {
		return false, ""
	}
	log.Printf("[ConversationAgent] Layer 10: User persisting after a raised concern")
	return true, last.Content
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
func (ca *conversationAgent) detectPrincipleConcerns(userMessage, block string, extractedContext *models.ExtractedContext) (bool, string, string) {
	if ca.constitution == nil || ca.llmClient == nil {
		return false, "", ""
	}

	// Build prompt asking LLM to identify engaged principles
	userPrompt := block + fmt.Sprintf(`Analyze this message to identify which principles from our ethical framework are ENGAGED (involved in the situation).

Message: "%s"

Our principles (write the id, exactly as given before the colon):
- harm_prevention: involves potential harm to self or others
- user_autonomy: involves user's own decisions and agency
- consent_and_respect: involves respecting others' choices and perspectives
- stakeholder_consideration: involves impact on other people
- transparency: involves honest communication or disclosure
- growth_and_learning: involves personal growth, reflection, or development

Return JSON with:
{
  "engagedPrinciples": ["id", ...],
  "primaryPrinciple": "id of the principle that matters most here",
  "evidence": "brief quote showing engagement",
  "needsClarification": true/false
}

needsClarification is true ONLY when what the user plans could wrong another person, hide something important from them, or decide for them without them (acting behind someone's back, keeping something important from them, pressuring them). It is false for an ordinary request, for a conflict the user wants help with, for a feeling the user expresses, for a short follow-up or a comment on a draft, and whenever nobody is being kept in the dark or pushed.

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
	if err := tools.SafeJSONParse("ConversationAgent.evalPrinciples", []byte(response.Content), &analysis); err != nil {
		log.Printf("[ConversationAgent] Layer 6-7: Failed to parse principle analysis: %v", err)
		return false, "", ""
	}

	analysis.PrimaryPrinciple = principleID(analysis.PrimaryPrinciple)
	// If no principles engaged or no clarification needed, no concern
	if len(analysis.EngagedPrinciples) == 0 || !analysis.NeedsClarification {
		log.Printf("[ConversationAgent] Layer 6-7: no principle concern (engaged=%v, primary=%q, needsClarification=%v)", analysis.EngagedPrinciples, analysis.PrimaryPrinciple, analysis.NeedsClarification)
		return false, "", ""
	}

	// The question is written by the model from the conversation (found live, 2026-10-15: a fixed question, "Have you talked to the
	// people involved about what they want?", was asked twice and said nothing about the situation); the fixed text is only the
	// fallback when the model cannot be reached.
	clarificationQ := ca.principleQuestion(analysis.PrimaryPrinciple, analysis.Evidence, userMessage, block)

	if clarificationQ == "" {
		log.Printf("[ConversationAgent] Layer 6-7: principle %q has no question (engaged=%v)", analysis.PrimaryPrinciple, analysis.EngagedPrinciples)
		return false, "", ""
	}

	log.Printf("[ConversationAgent] Layer 6-7: Principle concern detected - %s (evidence: %s)", analysis.PrimaryPrinciple, analysis.Evidence)
	return true, analysis.PrimaryPrinciple, clarificationQ
}

// principleMeaning says in words what each principle is about, for the model that writes the question.
var principleMeaning = map[string]string{
	"stakeholder_consideration": "how it affects the other people involved",
	"consent_and_respect":       "respecting what the other person wants and agrees to",
	"user_autonomy":             "the user's own decision, which only they can make",
	"transparency":              "what should be said openly and honestly",
	"harm_prevention":           "a risk of harm to someone",
	"growth_and_learning":       "what the user can learn or how they can grow from it",
}

// principleQuestion asks the model for one question about an engaged principle, written from the conversation: it uses only what the
// conversation states and assumes nothing about what another person wants. The reply model writes it; if its text is unusable the fast
// model tries once more. "" when neither gives a usable question: there is no fixed sentence (it was English whatever the user spoke).
func (ca *conversationAgent) principleQuestion(principle, evidence, userMessage, dialogue string) string {
	for _, role := range []string{tools.RoleReply, tools.RoleFast} {
		if q := ca.principleQuestionAs(role, principle, evidence, userMessage, dialogue); q != "" {
			return q
		}
	}
	return ""
}

func (ca *conversationAgent) principleQuestionAs(role, principle, evidence, userMessage, dialogue string) string {
	meaning, ok := principleMeaning[principle]
	if !ok || ca.llmClient == nil {
		return ""
	}
	req := &tools.LLMRequest{
		Role:         role,
		SystemPrompt: "You write one question for Moly, a communication thinking partner." + readsNoPages + tools.StatesOnlyWhatIsStated + tools.InUsersLanguage,
		UserPrompt: dialogue + "The user said: \"" + userMessage + "\"\n\n" +
			"What matters in what they plan is: " + meaning + ". The words that show it: \"" + evidence + "\".\n" +
			"Write one short, warm question, in the language the user writes in (the language of their own words, never that of a name, a number or a quoted word), that helps the user think about it in THIS situation. " +
			"Use only what the conversation states; assume nothing about what another person wants or knows, and give nobody a role the conversation does not state. " +
			"Do not ask what the user has already answered in the conversation. The whole reply holds exactly one question mark. Do not give advice and do not refuse. Output only the question.",
		MaxTokens:   120,
		Temperature: 0.4,
	}
	resp, err := ca.llmClient.Call(context.Background(), req)
	if err != nil {
		log.Printf("[ConversationAgent] Principle question generation failed: %v", err)
		return ""
	}
	out := strings.Trim(strings.TrimSpace(resp.Content), "\"“”")
	if !meetsQuestionContract(out) {
		// found live (2026-10-15): the fixed text appeared and the log did not say why. Only counts are logged, not the text.
		log.Printf("[ConversationAgent] Principle question not usable (%d question marks, %d words, limit %d words): no question is asked",
			countQuestionMarks(out), len(strings.Fields(out)), maxQuestionWords)
		return ""
	}
	return out
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
	if err := tools.SafeJSONParse("parseJSONArray", []byte(jsonStr), &result); err != nil {
		return nil, err
	}
	return result, nil
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
		Role:        tools.RoleReply,
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

// MEDIUM FIX: Helper functions for safe operations with logging

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

// REMAINING ISSUES 3: Deduplicate Common Code Patterns

// REMAINING ISSUES 7: Dataflow Optimization & Validation

// formatAccumulatedEntities renders accumulated contacts and traits for the reply prompt.
// Only traits whose subject is the user are presented as the user's own.
func formatAccumulatedEntities(entities []models.ExtractedEntity) string {
	out := ""
	// Group entities by type and by whose trait they are
	contactsMap := make(map[string]bool)
	var contactsList []string
	userTraits := []string{}
	userTraitsSeen := make(map[string]bool)
	otherTraits := make(map[string][]string) // subject -> traits (a contact or a pronoun)
	otherSubjects := []string{}
	traitsSeen := make(map[string]bool)

	for _, entity := range entities {
		if entity.Type == "contact" && !contactsMap[entity.Value] {
			contactsMap[entity.Value] = true
			contactsList = append(contactsList, entity.Value)
		} else if entity.Type == "characteristic" {
			// A trait is only attributed to the user when its subject says so.
			// Unknown subjects are left out rather than assumed to be the user's.
			switch {
			case entity.Subject == "user" && !userTraitsSeen[entity.Value]:
				userTraitsSeen[entity.Value] = true
				userTraits = append(userTraits, entity.Value)
			case entity.Subject != "" && entity.Subject != "user" && !traitsSeen[entity.Subject+"|"+entity.Value]:
				traitsSeen[entity.Subject+"|"+entity.Value] = true
				if _, known := otherTraits[entity.Subject]; !known {
					otherSubjects = append(otherSubjects, entity.Subject)
				}
				otherTraits[entity.Subject] = append(otherTraits[entity.Subject], entity.Value)
			}
		}
	}

	// Format contacts
	if len(contactsList) > 0 {
		out += fmt.Sprintf("Contacts mentioned: %s\n", strings.Join(contactsList, ", "))
	}

	// Format the user's own traits
	if len(userTraits) > 0 {
		out += fmt.Sprintf("They've described themselves as: %s\n", strings.Join(userTraits, ", "))
	}

	// Traits of other people are labelled with who they belong to
	for _, subject := range otherSubjects {
		out += fmt.Sprintf("Traits of %s (NOT the user): %s\n", subject, strings.Join(otherTraits[subject], ", "))
	}
	return out
}

// socraticAllowed reports whether a Socratic question may be asked for this response.
// It requires the deepening gates to have passed (shouldDeepen), not just the workflow name.
// Before this rule, the ack_socratic workflow asked the question even when the gates said "need clarification first".
func socraticAllowed(isAckWithSocratic, shouldDeepen, hasSelector, hasAboutMe, hasContact, hasIntention bool) bool {
	return isAckWithSocratic && shouldDeepen && hasSelector && hasAboutMe && hasContact && hasIntention
}

// NameQuestion is the question asked when a person has no name yet. It is fixed text, not generated,
// because it is protocol: Moly needs the name before it can talk about this person again later.
func NameQuestion(label string) string {
	// A label is a short noun ("manager"). A clause the model copied from the message ("someone I met in a chat room") reads as broken
	// English in the question (found in a dynamic run), so a long or empty label is asked about as "person".
	label = strings.TrimSpace(label)
	if label == "" || len(strings.Fields(label)) > 3 {
		label = "person"
	}
	return fmt.Sprintf("Can you give me a name for the %s you mentioned?", label)
}

// principleID turns what the model wrote for a principle into the id the code uses: "Stakeholder Consideration", "stakeholder
// consideration" and "stakeholder_consideration" are the same principle. (The prompt used to list display names while the code
// matched ids, so a principle question was never asked: 0 of 12 on Claude haiku.)
func principleID(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	n = strings.NewReplacer("&", "and", "-", " ", "_", " ").Replace(n)
	n = strings.Join(strings.Fields(n), "_")
	return n
}

// CareCheckIn is the one question that opens the turn after Moly refused a crisis statement. It is fixed on purpose: nothing a model
// might write may soften or omit it.
const CareCheckIn = "Before we go on, I want to check in with you. Are you safe right now?"

// LastAssistantMessage returns Moly's last message in a history that is [current, oldest ... newest], or nil.
func LastAssistantMessage(history []models.Message) *models.Message {
	for i := len(history) - 1; i >= 1; i-- {
		if history[i].Role == "assistant" {
			return &history[i]
		}
	}
	return nil
}
