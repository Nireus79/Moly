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
	db                   *database.Database
	clarificationRepo    *database.ClarificationQuestionRepository
	sentenceAnalysisRepo *database.SentenceAnalysisRepository // FIX #14
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
		layers:                make([]tools.Layer, 0),
		cache:                 tools.NewExtractionCache(),
		metrics:               tools.NewOrchestratorMetrics(),
		debugMode:             false,
		contextExtractor:      contextExtractor,
		constitutionalEval:    constitutionalEval,
		maturityService:       maturityService,
		conflictDetector:      conflictDetector,
		layer5ConflictHandler: layer5ConflictHandler,
		llmClient:             llmClient,
		db:                    db,
		clarificationRepo:     database.NewClarificationQuestionRepository(db),
		sentenceAnalysisRepo:  database.NewSentenceAnalysisRepository(db.GetConnection()), // FIX #14
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
	uo.addLayer(NewLayer4GapDetector(uo.llmClient))

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

	// FIX #3: ALWAYS run Layer 1 (extraction) and Layer 3 (maturity)
	// Only skip Layer 2 (principle checking) when clarifying
	// Reason: L1 must extract fresh + merge with accumulated, L3 must recalculate maturity
	// Layer indices: 0=L1, 1=L2, 2=L3, 3=L4, ...
	skipLayer2OnClarification := isAnsweringClarification
	if skipLayer2OnClarification {
		log.Printf("[UnifiedOrchestrator] 🔄 FIX #3: LOOP PATTERN - Clarification detected, pending=%d - skipping only L2 (index 1)", len(pendingClarifications))
		// Note: Layer 2 (principle checking) is deterministic - doesn't need rerun
	}

	// FIX #52: Create NEW ContextChangeTracker per conversation (not shared across conversations!)
	// Previous bug: shared tracker caused data contamination between users/conversations
	conversationTracker := NewContextChangeTracker()

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

	// Create layer context
	lc := tools.NewLayerContext(analysisCtx, userID, messageID, conversationID, maturityContext)

	// FIX #3 Phase 3: Wire confidence-driven clarifications from extraction to Layer 4
	if analysisCtx.ClarificationQuestions != nil {
		// Cast clarifications to proper type for Layer 4
		if clarQuestions, ok := analysisCtx.ClarificationQuestions.([]*database.ClarificationQuestion); ok {
			if lc.Layer4 == nil {
				lc.Layer4 = &tools.Layer4Result{}
			}
			lc.Layer4.ClarificationQuestions = clarQuestions
			log.Printf("[UnifiedOrchestrator] ✓ FIX #3 Phase 3: Wired %d confidence-driven clarifications to Layer 4",
				len(clarQuestions))
		}
	}

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

		// FIX #11: Build message summary cache for Phase 3 optimization
		// Layers can use cached summaries instead of re-processing recent messages
		lc.MessageSummaryCache = uo.buildMessageSummaryCache(analysisCtx)
		if len(lc.MessageSummaryCache) > 0 {
			log.Printf("[UnifiedOrchestrator] FIX #11: ✓ Populated message summary cache (%d summaries available to layers)",
				len(lc.MessageSummaryCache))
		}

		// FIX #4: Detect if this is Message 1 (first message in conversation)
		// Message count tells us: 1 = first message, 2+ = continuation
		lc.IsMessageOne = (analysisCtx.TotalMessages <= 1)
		if lc.IsMessageOne {
			log.Printf("[UnifiedOrchestrator] ✓ FIX #4: Message 1 detected - will lock primary goal")
		} else {
			log.Printf("[UnifiedOrchestrator] ℹ️ FIX #4: Message %d - tracking current intent separately", analysisCtx.TotalMessages)
		}

		// Populate primary goal if we have it
		if analysisCtx.PrimaryGoal != "" {
			lc.PrimaryGoal = analysisCtx.PrimaryGoal
			log.Printf("[UnifiedOrchestrator] ✓ FIX #4: Primary goal loaded: %q", lc.PrimaryGoal)
		}
	}

	if uo.debugMode {
		layerInfo := "all layers L1-11"
		if skipLayer2OnClarification {
			layerInfo = "L1, L3-11 (skipping L2)"
		}
		log.Printf("[UnifiedOrchestrator] Starting message processing (user=%s, msgID=%s, layers=%s, isClarification=%v)",
			userID, messageID, layerInfo, isAnsweringClarification)
	}

	// PHASE 2: PRE-LAYER-1 CONTACT WORKFLOW
	// Detect contacts, resolve pronouns, check for ambiguities
	log.Printf("[UnifiedOrchestrator] ▶ PRE-Layer-1: Starting Contact Workflow")

	contactDetector := NewContactDetector(uo.db)
	detectedContacts := contactDetector.DetectInMessage(message, analysisCtx)
	log.Printf("[UnifiedOrchestrator] Contact Workflow: Detected %d contacts", len(detectedContacts))

	// Build active contacts list
	var activeContacts []*models.Contact
	activeContacts = append(activeContacts, detectedContacts...)

	// Check for contact ambiguity
	confidenceCalc := NewConfidenceCalculator()
	// Convert []models.Contact to []*models.Contact for ambiguity calculation
	var relevantContactPtrs []*models.Contact
	if analysisCtx != nil {
		for i := range analysisCtx.RelevantContacts {
			relevantContactPtrs = append(relevantContactPtrs, &analysisCtx.RelevantContacts[i])
		}
	}
	contactAmbiguity := confidenceCalc.CalculateContactAmbiguity(message, detectedContacts, relevantContactPtrs)

	// If ambiguous, ask for clarification before running layers
	if contactAmbiguity > 0.60 && len(activeContacts) > 1 {
		lc.ClarificationNeeded = true
		lc.ClarificationID = generateClarificationID()
		lc.ClarificationFlag = "contact_ambiguity"
		lc.ClarificationConfidence = contactAmbiguity
		lc.ClarificationQuestion = buildContactClarificationQuestion(activeContacts)
		lc.ClarificationOptions = buildClarificationOptions(activeContacts)

		log.Printf("[UnifiedOrchestrator] 🔴 Contact ambiguity detected (confidence=%.2f) - asking for clarification", contactAmbiguity)
		return lc, nil
	}

	// Wire active contacts to LayerContext
	lc.ActiveContacts = activeContacts
	if len(activeContacts) > 0 {
		lc.ContactContext = buildContactContextString(activeContacts)
		log.Printf("[UnifiedOrchestrator] ✓ Contact context wired: %s", lc.ContactContext)
	}

	// Run each layer in sequence
	var lastError error
	layersExecuted := 0
	layersSkipped := 0

	for i, layer := range uo.layers {
		// LOOP PATTERN: Skip only Layer 2 (index 1) if answering clarification
		// Layer 1 (extraction) must ALWAYS run to get fresh data
		// Layer 2 (principle checking) can be skipped as it's deterministic
		if skipLayer2OnClarification && i == 1 {
			if uo.debugMode {
				log.Printf("[UnifiedOrchestrator] ⊘ Loop pattern: Skipping Layer 2 (principle checking is deterministic, using accumulated context)")
			}
			uo.metrics.RecordLayerSkip(layer.Name())
			layersSkipped++
			continue
		}
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
			maturity := lc.Layer3.MaturityScore

			// Use maturity calculator to get phase-aware gate
			mc := tools.NewMaturityCalculator()
			currentPhase := mc.EstimateCurrentPhase(maturity)
			severityGate := mc.GetEvaluationSeverityGate(maturity)

			log.Printf("[UnifiedOrchestrator] ✓ Maturity Analysis: score=%.2f, phase=%s, severity_gate=%.2f",
				maturity, currentPhase, severityGate)

			// Store for Layer 4 to use in gap filtering
			lc.MaturityPhase = currentPhase
			lc.MaturitySeverityGate = severityGate

			// FIX #72: Analyze goal coherence (how current goal relates to primary goal)
			// This enables goal-aligned gap detection
			goalCoherence := AnalyzeGoalCoherence(lc)
			lc.GoalCoherence = goalCoherence
			log.Printf("[UnifiedOrchestrator] ✓ FIX #72: Goal coherence analyzed: primary=%q, current=%q, progression=%s (confidence=%.2f)",
				goalCoherence.PrimaryGoal, goalCoherence.CurrentGoal, goalCoherence.GoalProgression, goalCoherence.Confidence)
		}

		// After Layer 4 (Gap Detection): Filter gaps by severity gate (proportional, not block)
		if i == 3 && lc.Layer4 != nil && lc.Layer4.ShouldClarify { // Layer 4 (index 3)
			severityGate := lc.MaturitySeverityGate

			// Filter gaps: only keep those with sufficient confidence for this phase
			// This uses the proportional gate, not a binary block
			filtered := []tools.Gap{}
			for _, gap := range lc.Layer4.DetectedGaps {
				if gap.Confidence >= severityGate {
					filtered = append(filtered, gap)
				}
			}

			log.Printf("[UnifiedOrchestrator] ✓ Gap Filtering: %d gaps → %d after severity gate (%.2f)",
				len(lc.Layer4.DetectedGaps), len(filtered), severityGate)

			// Replace gaps with filtered ones (removes low-confidence gaps for this phase)
			lc.Layer4.DetectedGaps = filtered

			// Continue processing - do NOT block the pipeline
			// Layer 6+ will use these filtered gaps for clarification
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

// SetDebugMode enables/disables debug logging
func (uo *UnifiedOrchestrator) SetDebugMode(enabled bool) {
	uo.debugMode = enabled
	log.Printf("[UnifiedOrchestrator] Debug mode: %v", enabled)
}

// GetCache returns the extraction cache
func (uo *UnifiedOrchestrator) GetCache() *tools.ExtractionCache {
	return uo.cache
}

// SetLayer1Adapter updates the Layer 1 adapter (for testing/customization)
func (uo *UnifiedOrchestrator) SetLayer1Adapter(adapter tools.Layer) error {
	if len(uo.layers) == 0 {
		return fmt.Errorf("no layers registered")
	}

	// Replace first layer (should be Layer 1)
	uo.layers[0] = adapter
	log.Printf("[UnifiedOrchestrator] ✓ Updated Layer 1 adapter")
	return nil
}

// AddLayer adds a new layer to the pipeline (for building out Layers 4-11)
func (uo *UnifiedOrchestrator) AddLayer(layer tools.Layer) error {
	if layer == nil {
		return fmt.Errorf("cannot add nil layer")
	}

	uo.layers = append(uo.layers, layer)
	log.Printf("[UnifiedOrchestrator] ✓ Added layer %s (total: %d)", layer.Name(), len(uo.layers))
	return nil
}

// GetLayerCount returns number of registered layers
func (uo *UnifiedOrchestrator) GetLayerCount() int {
	return len(uo.layers)
}

// ListLayers returns names of all registered layers
func (uo *UnifiedOrchestrator) ListLayers() []string {
	names := make([]string, 0, len(uo.layers))
	for _, layer := range uo.layers {
		names = append(names, layer.Name())
	}
	return names
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

// FIX #11: Build message summary cache from AnalysisContext
// Phase 3 optimization: Create lookup map of message summaries for layers to use
func (uo *UnifiedOrchestrator) buildMessageSummaryCache(analysisCtx *models.AnalysisContext) map[string]interface{} {
	cache := make(map[string]interface{})

	if analysisCtx == nil || len(analysisCtx.RecentMessageSummaries) == 0 {
		return cache
	}

	// Build cache from recent message summaries
	// BUG FIX: Handle MessageSummary structs correctly (not map[string]interface{})
	for _, summaryIface := range analysisCtx.RecentMessageSummaries {
		// Handle MessageSummary struct directly (not map)
		if summary, ok := summaryIface.(*models.MessageSummary); ok {
			if summary != nil && summary.MessageID != "" {
				cache[summary.MessageID] = summary
				log.Printf("[UnifiedOrchestrator] FIX #11 BUG FIX: ✓ Cached summary for message %s (entities=%d, confidence=%.2f)",
					summary.MessageID, len(summary.ExtractedEntities), summary.Confidence)
			}
		} else if summary, ok := summaryIface.(models.MessageSummary); ok {
			// Handle value type as well (in case not pointer)
			if summary.MessageID != "" {
				cache[summary.MessageID] = summary
				log.Printf("[UnifiedOrchestrator] FIX #11 BUG FIX: ✓ Cached summary for message %s (entities=%d, confidence=%.2f)",
					summary.MessageID, len(summary.ExtractedEntities), summary.Confidence)
			}
		}
	}

	if len(cache) > 0 {
		log.Printf("[UnifiedOrchestrator] FIX #11: ✓ Built message summary cache (%d summaries)", len(cache))
	}

	return cache
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

// generateClarificationID creates a unique ID for this clarification
func generateClarificationID() string {
	return "clr_" + fmt.Sprintf("%d", time.Now().UnixNano())
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

// buildContactClarificationQuestion builds a question for contact disambiguation
func buildContactClarificationQuestion(contacts []*models.Contact) string {
	if len(contacts) < 2 {
		return "Which contact are you referring to?"
	}

	var sb strings.Builder
	sb.WriteString("I want to make sure I understand correctly. You mentioned:\n")

	for i, c := range contacts {
		pronounStr := ""
		if len(c.Pronouns) > 0 {
			pronounStr = " (" + strings.Join(c.Pronouns, "/") + ")"
		}
		sb.WriteString(fmt.Sprintf("  %c) %s%s\n", 'A'+rune(i), c.Name, pronounStr))
	}

	sb.WriteString("\nWhich one are you referring to?")
	return sb.String()
}

// buildClarificationOptions builds the A/B/C options for selection
func buildClarificationOptions(contacts []*models.Contact) []string {
	var options []string
	for i, c := range contacts {
		option := fmt.Sprintf("%c) %s", 'A'+rune(i), c.Name)
		if len(c.Pronouns) > 0 {
			option += " (" + strings.Join(c.Pronouns, "/") + ")"
		}
		options = append(options, option)
	}
	return options
}
