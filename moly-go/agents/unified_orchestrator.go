package agents

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"moly/database"
	"moly/models"
	"moly/storage"
	"moly/tools"
)

// UnifiedOrchestrator coordinates all 11 layers of message processing
// It takes the fragmented components and wires them into a unified pipeline
type UnifiedOrchestrator struct {
	// Layers (initialized in order)
	layers []tools.Layer

	// Infrastructure
	cache     *tools.ExtractionCache
	metrics   *tools.OrchestratorMetrics
	debugMode bool

	// Dependencies
	contextExtractor      *ContextExtractor
	constitutionalEval    *tools.ConstitutionalEvaluator
	maturityService       *storage.MaturityService
	conflictDetector      *ConflictDetector
	layer5ConflictHandler *Layer5ConflictHandler
	llmClient             tools.LLMProvider
	// FIX #52: ContextChangeTracker created per-conversation (not orchestrator-wide)
	// Prevents data contamination across users/conversations

	// Database and repositories
	db                       *database.Database
	clarificationRepo        *database.ClarificationQuestionRepository
	clarificationHistoryRepo *database.ClarificationHistoryRepository // FIX #5 (Phase 5)
	sentenceAnalysisRepo     *database.SentenceAnalysisRepository     // FIX #14
}

// NewUnifiedOrchestrator creates a new orchestrator with all dependencies
func NewUnifiedOrchestrator(
	contextExtractor *ContextExtractor,
	constitutionalEval *tools.ConstitutionalEvaluator,
	maturityService *storage.MaturityService,
	conflictDetector *ConflictDetector,
	layer5ConflictHandler *Layer5ConflictHandler,
	db *database.Database,
	llmClient tools.LLMProvider,
) *UnifiedOrchestrator {
	orch := &UnifiedOrchestrator{
		layers:                   make([]tools.Layer, 0),
		cache:                    tools.NewExtractionCache(),
		metrics:                  tools.NewOrchestratorMetrics(),
		debugMode:                false,
		contextExtractor:         contextExtractor,
		constitutionalEval:       constitutionalEval,
		maturityService:          maturityService,
		conflictDetector:         conflictDetector,
		layer5ConflictHandler:    layer5ConflictHandler,
		llmClient:                llmClient,
		db:                       db,
		clarificationRepo:        database.NewClarificationQuestionRepository(db),
		clarificationHistoryRepo: database.NewClarificationHistoryRepository(db),             // FIX #5 (Phase 5)
		sentenceAnalysisRepo:     database.NewSentenceAnalysisRepository(db.GetConnection()), // FIX #14
	}

	// Initialize layers in order
	orch.initializeLayers()

	return orch
}

// initializeLayers registers all layers in the correct order
func (uo *UnifiedOrchestrator) initializeLayers() {
	log.Printf("[UnifiedOrchestrator] Initializing 11-layer pipeline")

	// Phase 0-1: Extraction (FIXED: properly initialized with dependencies)
	layer1Adapter := NewLayer1ContextExtractionAdapter(
		uo.contextExtractor,
		uo.cache,
	)
	uo.addLayer(layer1Adapter)

	// Phase 1-2: Principle Checking
	uo.addLayer(NewLayer2PrincipleCheckAdapter(uo.constitutionalEval))

	// Phase 1-3: Maturity Assessment
	uo.addLayer(NewLayer3MaturityAssessmentAdapter(uo.maturityService))

	// Layer 4: Gap Detection (FIX #75: Pass LLM for dynamic gap generation)
	layer4 := NewLayer4GapDetector(uo.llmClient)
	layer4.SetClarificationHistory(uo.clarificationHistoryRepo) // FIX #5 (Phase 5): Enable history tracking
	uo.addLayer(layer4)

	// Layer 5: Conflict Detection
	uo.addLayer(NewLayer5UnifiedConflictDetection(
		uo.conflictDetector,
		uo.layer5ConflictHandler,
	))

	// Layer 6: Ambiguous Request Handling
	uo.addLayer(NewLayer6AmbiguousRequestHandler(uo.llmClient))

	// Layer 7: Principle Violation Clarification
	uo.addLayer(NewLayer7PrincipleViolationClarification(uo.llmClient))

	// Layer 8: Socratic Deepening
	uo.addLayer(NewLayer8SocraticDeepening())

	// Layer 9: Topic Shift Detection
	uo.addLayer(NewLayer9TopicShiftDetection())

	// Layer 10: Persistent Questioning
	uo.addLayer(NewLayer10PersistentQuestioning(uo.llmClient, uo.db))

	// Layer 11: Denial Protocol
	uo.addLayer(NewLayer11DenialProtocol())

	log.Printf("[UnifiedOrchestrator] ✓ Pipeline initialized with %d layers", len(uo.layers))
}

// addLayer registers a layer (internal)
func (uo *UnifiedOrchestrator) addLayer(layer tools.Layer) {
	uo.layers = append(uo.layers, layer)
}

// ProcessMessage runs all layers in sequence for a message
func (uo *UnifiedOrchestrator) ProcessMessage(
	ctx context.Context,
	message string,
	userID string,
	conversationID string,
	messageID string,
	analysisCtx *models.AnalysisContext,
	maturityContext *models.ConversationMaturity,
) (*tools.LayerContext, error) {
	startTime := time.Now()

	if message == "" {
		return nil, fmt.Errorf("message cannot be empty")
	}

	if analysisCtx == nil {
		return nil, fmt.Errorf("analysis context cannot be nil")
	}

	// FIX #2: LOOP PATTERN: Detect if this is a clarification response to a previous question
	// If so, skip Layers 1-3 and jump to Layer 4 with accumulated context
	// FIXED: Check database for actual pending clarifications, not just entity presence
	var pendingClarifications []*database.ClarificationQuestion
	var clarificationErr error
	if uo.clarificationRepo != nil {
		pendingClarifications, clarificationErr = uo.getPendingClarifications(userID, conversationID)
		if clarificationErr != nil {
			log.Printf("[UnifiedOrchestrator] ⚠️ FIX #2: Warning - failed to check pending clarifications: %v", clarificationErr)
			pendingClarifications = make([]*database.ClarificationQuestion, 0)
		}
	}

	isAnsweringClarification := len(pendingClarifications) > 0 && uo.addressesClarification(message, pendingClarifications)

	// FIX #52: Create NEW ContextChangeTracker per conversation (not shared across conversations!)
	// Previous bug: shared tracker caused data contamination between users/conversations
	conversationTracker := NewContextChangeTrackerFromState(nil)
	if analysisCtx != nil {
		conversationTracker = NewContextChangeTrackerFromState(analysisCtx.ContextTracker)
	}

	// FIX #43-45: Track context changes (intention, goals, meta-instructions)
	if analysisCtx != nil {
		// FIX #43: Detect intention changes
		intentionChanged, prevIntent, currIntent := conversationTracker.DetectIntentionChange(analysisCtx)
		if intentionChanged {
			log.Printf("[UnifiedOrchestrator] 🚨 FIX #43: Intent shifted from '%s' to '%s' - may need clarification",
				prevIntent, currIntent)
		}

		// FIX #44: Detect goal changes
		goalChanged, added, removed := conversationTracker.DetectGoalChange(analysisCtx)
		if goalChanged {
			log.Printf("[UnifiedOrchestrator] 🚨 FIX #44: Goals changed - added %d, removed %d",
				len(added), len(removed))
			if len(removed) > 0 {
				log.Printf("[UnifiedOrchestrator]   Removed goals: %v", removed)
			}
		}

		// FIX #45: Track meta-instructions from current message
		conversationTracker.TrackMetaInstruction(message)

		// FIX #70: DISABLED - Meta-instruction conflict detection
		// Reason: "be direct" + "be casual" are communication style descriptors, not contradictory instructions
		// Creating false gaps and unnecessary clarifications
		// Will re-enable with improved heuristics later
		// if conversationTracker.HasContradictoryInstructions() {
		// 	log.Printf("[UnifiedOrchestrator] ⚠️ FIX #45: User has given contradictory meta-instructions - may need clarification")
		// }
	}

	if analysisCtx != nil {
		analysisCtx.ContextTracker = conversationTracker.ExportState()
	}

	// Create layer context
	lc := tools.NewLayerContext(analysisCtx, userID, messageID, conversationID, maturityContext)

	// FIX #52: Wire per-conversation tracker (NOT shared)
	lc.ContextChangeTracker = conversationTracker

	// FIX #22 & #23: Wire insights and reflections to layers
	// Provides context about previous conversations and contact understanding
	if analysisCtx != nil {
		// RecentInsights: Previous insights from this conversation
		lc.RecentInsights = analysisCtx.RecentInsights
		if len(lc.RecentInsights) > 0 {
			log.Printf("[UnifiedOrchestrator] ✓ FIX #22: Wired %d recent insights to layers", len(lc.RecentInsights))
		}

		// RelevantReflections: Previous reflections about contacts/topics
		lc.RelevantReflections = analysisCtx.RelevantReflections
		if len(lc.RelevantReflections) > 0 {
			log.Printf("[UnifiedOrchestrator] ✓ FIX #23: Wired %d relevant reflections to layers", len(lc.RelevantReflections))
		}
	}

	// FIX #12: Wire pending clarifications to Layer 4
	// This allows Layer 4 to filter out already-asked gaps
	lc.PendingClarifications = pendingClarifications
	if len(pendingClarifications) > 0 {
		log.Printf("[UnifiedOrchestrator] ✓ FIX #12: Wired %d pending clarifications to Layer 4", len(pendingClarifications))
	}

	// FIX #14: Wire sentence analyses to Layer 5
	// Load sentence analyses from this message, use for conflict detection
	var sentenceAnalyses []*database.SentenceAnalysisData
	if uo.sentenceAnalysisRepo != nil {
		var sentenceErr error
		sentenceAnalyses, sentenceErr = uo.sentenceAnalysisRepo.GetSentenceAnalysesByMessage(userID, messageID)
		if sentenceErr != nil {
			log.Printf("[UnifiedOrchestrator] ⚠️ FIX #14: Warning - failed to load sentence analyses: %v", sentenceErr)
			sentenceAnalyses = make([]*database.SentenceAnalysisData, 0)
		}
	}
	lc.SentenceAnalyses = sentenceAnalyses
	if len(sentenceAnalyses) > 0 {
		log.Printf("[UnifiedOrchestrator] ✓ FIX #14: Wired %d sentence analyses to Layer 5", len(sentenceAnalyses))
	}

	// FIX #1: Wire SetAccumulatedContext() - load previous extraction state
	// This populates PreviousGoal, PreviousValues, AccumulatedExtractedEntities from prior messages
	if analysisCtx != nil {
		lc.SetAccumulatedContext(
			analysisCtx.AccumulatedExtractedEntities,
			analysisCtx.PreviousGoal,
			analysisCtx.PreviousValues,
		)
		if lc.PreviousGoal != "" {
			log.Printf("[UnifiedOrchestrator] ✓ FIX #1: Loaded previous goal: %q", lc.PreviousGoal)
		}
		if len(lc.AccumulatedExtractedEntities) > 0 {
			log.Printf("[UnifiedOrchestrator] ✓ FIX #1: Loaded %d accumulated entities from previous messages", len(lc.AccumulatedExtractedEntities))
		}

		// PHASE 2: no message-summary cache is given to the layers. Each layer evaluates the current message.

		// Populate primary goal if we have it
		if analysisCtx.PrimaryGoal != "" {
			lc.PrimaryGoal = analysisCtx.PrimaryGoal
			log.Printf("[UnifiedOrchestrator] ✓ FIX #4: Primary goal loaded: %q", lc.PrimaryGoal)
		}
	}

	if uo.debugMode {
		layerInfo := "all layers L1-11"
		log.Printf("[UnifiedOrchestrator] Starting message processing (user=%s, msgID=%s, layers=%s, isClarification=%v)",
			userID, messageID, layerInfo, isAnsweringClarification)
	}

	// PHASE 2: PRE-LAYER-1 CONTACT WORKFLOW (DISABLED - Handled in main.go Phases 3B-5)
	// NOTE: Contact detection, ambiguity checking, and clarification are now handled BEFORE orchestrator invocation:
	// - Phase 3B (main.go): Clarification response detection & processing
	// - Phase 4 (main.go): Progressive naming detection & application
	// - Phase 5 (main.go): Response formatting with contact awareness
	// The orchestrator receives contacts that are already resolved and should NOT re-detect or re-clarify.

	log.Printf("[UnifiedOrchestrator] ℹ Contact workflow handled upstream in message processor (Phases 3B-5)")

	// Load pre-resolved contacts from AnalysisContext (already processed by main.go)
	var activeContacts []*models.Contact
	if analysisCtx != nil && len(analysisCtx.RelevantContacts) > 0 {
		for i := range analysisCtx.RelevantContacts {
			activeContacts = append(activeContacts, &analysisCtx.RelevantContacts[i])
		}
		log.Printf("[UnifiedOrchestrator] ✓ Using pre-resolved contacts from AnalysisContext: %d contacts", len(activeContacts))
	}

	// Wire active contacts to LayerContext (already resolved, no ambiguity checking needed)
	lc.ActiveContacts = activeContacts
	if len(activeContacts) > 0 {
		lc.ContactContext = buildContactContextString(activeContacts)
		log.Printf("[UnifiedOrchestrator] ✓ Contact context wired (pre-resolved): %s", lc.ContactContext)
	}

	// Run each layer in sequence
	var lastError error
	layersExecuted := 0
	layersSkipped := 0

	for i, layer := range uo.layers {
		// Check if we should stop early
		if lc.ShouldStop {
			if uo.debugMode {
				log.Printf("[UnifiedOrchestrator] ⏹️ Stopping at layer %d (%s): %s", i+1, layer.Name(), lc.StopReason)
			}
			break
		}

		// Check if layer can be skipped
		if layer.CanSkip(lc) {
			if uo.debugMode {
				log.Printf("[UnifiedOrchestrator] ⊘ Skipping %s (no work)", layer.Name())
			}
			uo.metrics.RecordLayerSkip(layer.Name())
			layersSkipped++
			continue
		}

		// Execute layer
		layerStart := time.Now()

		result, err := layer.Process(ctx, lc)
		if err != nil {
			log.Printf("[UnifiedOrchestrator] ⚠️ Layer %s error: %v", layer.Name(), err)
			lastError = err
			uo.metrics.RecordError()

			// For critical layers, stop immediately
			if layer.Priority() > 80 {
				log.Printf("[UnifiedOrchestrator] Stopping due to critical layer error")
				lc.StopReason = fmt.Sprintf("layer_error: %s", layer.Name())
				lc.ShouldStop = true
				break
			}

			// For non-critical layers, continue (graceful degradation)
			continue
		}

		// CRITICAL FIX: Validate layer result before applying
		if result == nil {
			log.Printf("[UnifiedOrchestrator] ERROR: Layer %s returned nil result (critical bug)", layer.Name())
			lc.ShouldStop = true
			break
		}

		// Update context
		lc = result

		// FIX #5-6: After Layer 1 (extraction), populate UserGoal and ConversationTopic
		// These are used by gap detection (Layer 4) and response generation
		if i == 0 && lc.Layer1 != nil { // Layer 1 (index 0)
			if lc.Layer1.ExtractedGoal != "" {
				lc.UserGoal = lc.Layer1.ExtractedGoal
				log.Printf("[UnifiedOrchestrator] ✓ FIX #5: Set UserGoal from Layer 1: %q", lc.UserGoal)
			}
			if lc.Layer1.ConversationTopic != "" {
				lc.ConversationTopic = lc.Layer1.ConversationTopic
				log.Printf("[UnifiedOrchestrator] ✓ FIX #6: Set ConversationTopic from Layer 1: %q", lc.ConversationTopic)
			}
		}

		// ARCHITECTURAL FIX #2: Use phase-aware proportional gating (not hardcoded threshold)
		// After Layer 3 (Maturity): Get phase and apply proportional severity gate
		if i == 2 && lc.Layer3 != nil { // Layer 3 (index 2)
			// The maturity the layers decide with is the gap maturity: answered questions over answered plus open ones.
			// The accomplishment score of Layer 3 (contacts, message count, entities) rises with the length of the chat,
			// not with what was answered, so it does not decide anything.
			maturity := ConversationGapMaturity(uo.clarificationRepo, lc.ConversationID)
			lc.Layer3 = layer3FromScore(maturity)
			log.Printf("[UnifiedOrchestrator] ✓ Maturity (gap maturity): score=%.2f, gate=%s", maturity, lc.Layer3.GateLevel)

			// FIX #72: Analyze goal coherence (how current goal relates to primary goal)
			// This enables goal-aligned gap detection
			goalCoherence := AnalyzeGoalCoherence(lc)
			lc.GoalCoherence = goalCoherence
			log.Printf("[UnifiedOrchestrator] ✓ FIX #72: Goal coherence analyzed: primary=%q, current=%q, progression=%s (confidence=%.2f)",
				goalCoherence.PrimaryGoal, goalCoherence.CurrentGoal, goalCoherence.GoalProgression, goalCoherence.Confidence)
		}

		// FIX #2 (Phase 2): Wire Layer 5 (Conflict Detection) clarifications to pending clarifications
		// This allows conflicts detected across messages to trigger clarification questions
		if i == 4 && lc.Layer5 != nil && len(lc.Layer5.ClarificationQuestions) > 0 { // Layer 5 (index 4)
			for _, conflictQ := range lc.Layer5.ClarificationQuestions {
				// Convert from database.ClarificationQuestion to same type for consistency
				// These conflict clarifications have priority over other gaps
				pendingClarifications = append(pendingClarifications, conflictQ)
			}

			log.Printf("[UnifiedOrchestrator] ✓ FIX #2 (Phase 2): Wired %d conflict clarifications from Layer 5",
				len(lc.Layer5.ClarificationQuestions))

			// Update LayerContext with new clarifications for Layer 6+ to see
			lc.PendingClarifications = pendingClarifications
		}

		// Record metrics
		layerDuration := time.Since(layerStart).Milliseconds()
		uo.metrics.RecordLayerTime(layer.Name(), layerDuration)
		layersExecuted++

		if uo.debugMode {
			log.Printf("[UnifiedOrchestrator] ✓ %s complete (duration=%dms)", layer.Name(), layerDuration)
		}
	}

	totalTime := time.Since(startTime).Milliseconds()
	uo.metrics.MessagesProcessed++

	log.Printf("[UnifiedOrchestrator] ✓ Message processing complete (layers=%d/%d, skipped=%d, total_time=%dms, stopped=%v)",
		layersExecuted, len(uo.layers), layersSkipped, totalTime, lc.ShouldStop)

	// Return context and any error encountered
	return lc, lastError
}

// GetMetrics returns performance metrics
func (uo *UnifiedOrchestrator) GetMetrics() *tools.OrchestratorMetrics {
	return uo.metrics
}

// FIX #2: Check if there are pending clarification questions for this conversation
func (uo *UnifiedOrchestrator) getPendingClarifications(
	userID string,
	conversationID string,
) ([]*database.ClarificationQuestion, error) {
	if uo.clarificationRepo == nil {
		return nil, fmt.Errorf("clarification repository not available")
	}

	// Get all pending clarifications for the user
	allPending, err := uo.clarificationRepo.GetPendingQuestions(userID)
	if err != nil {
		return nil, err
	}

	// Filter to only those for this conversation
	conversationPending := make([]*database.ClarificationQuestion, 0)
	for _, q := range allPending {
		if q.ConversationID == conversationID {
			conversationPending = append(conversationPending, q)
		}
	}

	return conversationPending, nil
}

// FIX #2: Check if message addresses a pending clarification question
func (uo *UnifiedOrchestrator) addressesClarification(
	message string,
	questions []*database.ClarificationQuestion,
) bool {
	if len(questions) == 0 {
		return false
	}

	// Normalize message for comparison
	msgLower := strings.ToLower(message)

	// Check if message contains keywords from any pending question
	for _, q := range questions {
		questionKeywords := extractKeywords(q.QuestionText)
		for _, keyword := range questionKeywords {
			if strings.Contains(msgLower, strings.ToLower(keyword)) {
				log.Printf("[UnifiedOrchestrator] ✓ FIX #2: Message addresses clarification question %q (keyword=%q)",
					q.QuestionText, keyword)
				return true
			}
		}
	}

	return false
}

// Helper to extract keywords from question text
func extractKeywords(text string) []string {
	// Remove common stopwords and extract meaningful tokens
	stopwords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"is": true, "are": true, "was": true, "were": true, "be": true,
		"what": true, "how": true, "why": true, "when": true, "where": true,
		"you": true, "i": true, "we": true, "they": true, "their": true, "your": true,
	}

	words := strings.Fields(text)
	keywords := make([]string, 0)

	for _, word := range words {
		// Remove punctuation and lowercase
		cleaned := strings.ToLower(strings.Trim(word, "?.,!;:"))
		if len(cleaned) > 2 && !stopwords[cleaned] {
			keywords = append(keywords, cleaned)
		}
	}

	return keywords
}

// buildContactContextString builds context string for extraction
func buildContactContextString(contacts []*models.Contact) string {
	if len(contacts) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Previous contacts in this conversation:\n")

	for i, c := range contacts {
		pronounStr := strings.Join(c.Pronouns, "/")
		if pronounStr == "" {
			pronounStr = "unknown"
		}

		sb.WriteString(fmt.Sprintf("  - Contact %d: %s, type=%s, pronouns=%s\n",
			i+1, c.Name, c.Relationship, pronounStr))
	}

	sb.WriteString("\nWhen extracting characteristics, tag them with the correct contact name.")
	return sb.String()
}
