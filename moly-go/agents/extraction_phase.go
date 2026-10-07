package agents

import (
	"context"
	"fmt"
	"log"
	"time"

	"moly/database"
	"moly/models"
	"moly/tools"
)

// ExtractionPhaseInput contains everything needed for extraction phase
type ExtractionPhaseInput struct {
	UserID              string
	ConversationID      string
	MessageID           string
	Message             string
	MessageCount        int
	RecentMessages      []models.Message
	UserProfile         *models.AboutMe
	Cache               *tools.LLMCache
	PreviousExtraction  interface{} // FIX #9: Accumulated context from previous messages (type: *PreviousExtraction from main.go)
}

// ExtractionPhaseOutput contains all results from extraction phase
type ExtractionPhaseOutput struct {
	Artifact               *models.ExtractionArtifact     // The extraction results
	ExtractedContext      *models.ExtractedContext        // FIX #6: High-level context from combined extraction
	AnalysisContext       *models.AnalysisContext         // Context built from extraction
	Conflicts             []ConflictDetectorResult        // Conflicts detected
	AmbiguousEntities     []models.ExtractedEntity        // Entities needing clarification
	HighConfidenceContacts []models.ExtractedEntity       // High-confidence contacts
	ClarificationQuestions []*database.ClarificationQuestion // FIX #3 Phase 3: Confidence-driven clarifications
}

// ExtractionPhase: Layer 0 of orchestrator - centralized extraction
// FIX #6: Now includes both high-level context AND entity extraction in one phase
type ExtractionPhase struct {
	intentDetector    *LLMIntentDetector
	contextExtractor  *ContextExtractor  // FIX #6: Combined extraction
	extractionStore   *tools.ExtractionStore
	conflictDetector  *ConflictDetector
	database          *database.Database
}

// NewExtractionPhase creates a new extraction phase
// FIX #6: Now accepts contextExtractor for combined extraction
func NewExtractionPhase(
	intentDetector *LLMIntentDetector,
	contextExtractor *ContextExtractor,
	extractionStore *tools.ExtractionStore,
	conflictDetector *ConflictDetector,
	db *database.Database,
) *ExtractionPhase {
	return &ExtractionPhase{
		intentDetector:   intentDetector,
		contextExtractor: contextExtractor,
		extractionStore:  extractionStore,
		conflictDetector: conflictDetector,
		database:         db,
	}
}

// Run executes the extraction phase
func (ep *ExtractionPhase) Run(ctx context.Context, input *ExtractionPhaseInput) (*ExtractionPhaseOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("extraction phase input is nil")
	}

	if input.Message == "" {
		log.Printf("[ExtractionPhase] Skipping extraction for empty message")
		return &ExtractionPhaseOutput{
			Artifact:         nil,
			AnalysisContext:  nil,
			Conflicts:        []ConflictDetectorResult{},
			AmbiguousEntities: []models.ExtractedEntity{},
		}, nil
	}

	log.Printf("[ExtractionPhase] Starting extraction phase for user=%s, message_len=%d", input.UserID, len(input.Message))

	// PHASE 2: Step 0 - Unified Linguistic Analysis (LinguisticParser.Parse)
	// Extracts: greeting, focus directives, constraints, priorities, and other linguistic patterns
	// All results flow through same extraction pipeline (no special cases)
	parser := tools.NewLinguisticParser()
	linguisticResults := parser.Parse(input.Message)

	// Process linguistic results: log all detected patterns
	// (focus/constraint/priority are currently handled by existing MetaInstructionDetector in main.go)
	// TODO FUTURE: Full wiring of these results to context updates
	if len(linguisticResults) > 0 {
		for _, result := range linguisticResults {
			switch result.Type {
			case "self_reference":
				if result.Property == "greeting" {
					log.Printf("[ExtractionPhase] ✓ Detected greeting (confidence=%.2f)", result.Confidence)
				}
			case "focus":
				log.Printf("[ExtractionPhase] ✓ Detected focus: %s (confidence=%.2f)", result.Property, result.Confidence)
			case "constraint":
				log.Printf("[ExtractionPhase] ✓ Detected constraint: %s (confidence=%.2f)", result.Property, result.Confidence)
			case "priority":
				log.Printf("[ExtractionPhase] ✓ Detected priority: %s (confidence=%.2f)", result.Property, result.Confidence)
			}
		}
	}

	// FIX #6: Step 0 - Extract high-level context (contact, style, intention)
	// This is now done ONCE here, not separately in main.go
	// FIX #59: Handle extraction errors instead of silently ignoring
	var extractedCtx *models.ExtractedContext
	if ep.contextExtractor != nil {
		var extractErr error
		extractedCtx, extractErr = ep.contextExtractor.Extract(ctx, input.Message)
		if extractErr != nil {
			log.Printf("[ExtractionPhase] ⚠️ FIX #59: Error extracting context: %v", extractErr)
			// Continue with nil context - downstream will handle
		} else if extractedCtx != nil {
			log.Printf("[ExtractionPhase] ✓ Extracted context (contact=%v, style=%v)",
				extractedCtx.Contact != nil, extractedCtx.Style != nil)
		}
	}

	// FIX #9: Load accumulated entities from previous extraction
	var accumulatedEntities []models.ExtractedEntity
	if input.PreviousExtraction != nil {
		// Cast to check if it's a PreviousExtraction struct from main.go
		// Using interface{} to avoid circular imports
		if prevExt, ok := input.PreviousExtraction.(*models.ExtractedEntity); ok {
			// Single entity
			accumulatedEntities = append(accumulatedEntities, *prevExt)
		} else if prevExtracts, ok := input.PreviousExtraction.([]models.ExtractedEntity); ok {
			// Slice of entities
			accumulatedEntities = prevExtracts
		}
		// Note: if PreviousExtraction is from main.go type, we'd need to access .Entities field
		// but due to package isolation, we handle this in main.go instead
		log.Printf("[ExtractionPhase] FIX #9: Will use accumulated context from previous messages")
	}

	// FIX #3 Phase 3: CONSOLIDATED EXTRACTION - Semantic extraction only
	// REMOVED: SmartExtractEntities call (old LLM-based duplicate system)
	// NOW: Semantic extraction is primary + only method

	if extractedCtx == nil {
		log.Printf("[ExtractionPhase] ⚠️ Semantic extraction returned nil context")
		return nil, fmt.Errorf("extraction failed: semantic extractor returned nil")
	}

	log.Printf("[ExtractionPhase] ✓ FIX #3 Phase 3: Unified semantic extraction (removed duplicate LLM system)")

	// Convert ExtractedContext to ExtractionArtifact for downstream compatibility
	artifact := &models.ExtractionArtifact{
		UserID:         input.UserID,
		ConversationID: input.ConversationID,
		MessageID:      input.MessageID,
		Source:         "semantic-linguistic", // Mark as semantic, not LLM
		CreatedAt:      time.Now().Unix(),
		Duration:       0,
		AverageConfidence: extractedCtx.IntentionConfidence,
		LLMSuccess:     true,
	}

	// Map semantic extraction results to entities for compatibility
	if extractedCtx.Intention != "" {
		artifact.Entities = append(artifact.Entities, models.ExtractedEntity{
			Type:       "goal",
			Value:      extractedCtx.Intention,
			Confidence: extractedCtx.IntentionConfidence,
		})
	}

	if extractedCtx.Contact != nil && extractedCtx.Contact.Name != "" {
		artifact.Entities = append(artifact.Entities, models.ExtractedEntity{
			Type:       "contact",
			Value:      extractedCtx.Contact.Name,
			Confidence: extractedCtx.Contact.Confidence,
		})
	}

	for _, v := range extractedCtx.UserValues {
		artifact.Entities = append(artifact.Entities, models.ExtractedEntity{
			Type:       "value",
			Value:      v,
			Confidence: 0.85,
		})
	}

	if extractedCtx.Style != nil && extractedCtx.Style.Style != "" {
		artifact.Entities = append(artifact.Entities, models.ExtractedEntity{
			Type:       "style",
			Value:      extractedCtx.Style.Style,
			Confidence: extractedCtx.Style.Confidence,
		})
	}

	for _, c := range extractedCtx.UserCharacteristics {
		artifact.Entities = append(artifact.Entities, models.ExtractedEntity{
			Type:       "characteristic",
			Value:      c,
			Confidence: 0.85,
		})
	}

	for _, p := range extractedCtx.IntentionPrinciples {
		artifact.Entities = append(artifact.Entities, models.ExtractedEntity{
			Type:       "concern",
			Value:      p,
			Confidence: 0.80,
		})
	}

	log.Printf("[ExtractionPhase] ✓ Semantic extraction yielded %d entities", len(artifact.Entities))

	// FIX #9: MERGE accumulated + current entities BEFORE further processing
	// This preserves full context history for downstream layers
	if len(accumulatedEntities) > 0 && len(artifact.Entities) > 0 {
		artifact.Entities = mergeExtractedEntities(accumulatedEntities, artifact.Entities)
		log.Printf("[ExtractionPhase] FIX #9: ✓ Merged extraction: %d accumulated + current = %d total",
			len(accumulatedEntities), len(artifact.Entities))
	} else if len(accumulatedEntities) > 0 {
		// No new entities, keep accumulated
		artifact.Entities = accumulatedEntities
		log.Printf("[ExtractionPhase] FIX #9: No new entities, using %d accumulated", len(accumulatedEntities))
	}

	// PHASE 1: Step 1.5 - Lock extraction immediately (prevent re-parsing)
	lockReason := fmt.Sprintf("extraction_complete: source=%s, entities=%d, confidence=%.2f",
		artifact.Source, len(artifact.Entities), artifact.AverageConfidence)
	if err := artifact.Lock(lockReason); err != nil {
		log.Printf("[ExtractionPhase] ✗ CRITICAL: Failed to lock extraction: %v", err)
		return nil, fmt.Errorf("failed to lock extraction artifact: %w", err)
	}

	// Set TTL (30 minutes) for automatic cleanup
	artifact.ExpiresAt = time.Now().Add(30 * time.Minute).Unix()

	log.Printf("[ExtractionPhase] ✓ Locked extraction (PHASE 1): %s", lockReason)

	// Step 2: Save to extraction store (5-min TTL)
	ep.extractionStore.Save(artifact)
	log.Printf("[ExtractionPhase] ✓ Saved locked extraction to store")

	// Step 3: Detect conflicts with database
	// FIX #9: Now pass accumulated entities if available (for multi-message conflict detection)
	conflicts := []ConflictDetectorResult{}
	if ep.conflictDetector != nil {
		detectedConflicts, err := ep.conflictDetector.DetectConflicts(
			ctx,
			input.UserID,
			input.ConversationID,
			artifact.Entities,
			accumulatedEntities, // FIX #9: Pass accumulated entities for multi-message conflict detection
		)
		if err != nil {
			log.Printf("[ExtractionPhase] Warning: Conflict detection failed: %v", err)
		} else {
			conflicts = detectedConflicts
			if len(conflicts) > 0 {
				log.Printf("[ExtractionPhase] ✓ Detected %d conflicts", len(conflicts))
			}
		}
	}

	// Step 4: Get ambiguous entities
	ambiguousEntities := artifact.AmbiguousEntities()
	if len(ambiguousEntities) > 0 {
		log.Printf("[ExtractionPhase] ⚠ Identified %d ambiguous entities needing clarification", len(ambiguousEntities))
	}

	// Step 5: Get high-confidence contacts
	highConfidenceContacts := artifact.HighConfidenceContacts()
	if len(highConfidenceContacts) > 0 {
		log.Printf("[ExtractionPhase] ✓ Identified %d high-confidence contacts", len(highConfidenceContacts))
	}

	// Step 6: Build AnalysisContext FROM extraction (not from DB!)
	analysisCtx := ep.buildContextFromExtraction(artifact, input)

	log.Printf("[ExtractionPhase] ✓ Built AnalysisContext from extraction (extraction_source=%s)", artifact.Source)

	// FIX #3 Phase 3: Generate confidence-driven clarifications
	clarifications := []*database.ClarificationQuestion{}
	if extractedCtx != nil {
		cbc := NewConfidenceBasedClarifications()
		clarifications = cbc.GenerateClarificationsForExtraction(input.Message, extractedCtx)
		if len(clarifications) > 0 {
			log.Printf("[ExtractionPhase] ✓ FIX #3 Phase 3: Generated %d confidence-driven clarifications", len(clarifications))
		}
	}

	return &ExtractionPhaseOutput{
		Artifact:               artifact,
		ExtractedContext:      extractedCtx,  // FIX #6: Return high-level context
		AnalysisContext:       analysisCtx,
		Conflicts:             conflicts,
		AmbiguousEntities:     ambiguousEntities,
		HighConfidenceContacts: highConfidenceContacts,
		ClarificationQuestions: clarifications, // FIX #3 Phase 3: Confidence-driven clarifications
	}, nil
}

// buildContextFromExtraction builds AnalysisContext directly from extraction results
// This is the KEY FIX: Don't rebuild from database, use the LLM extraction results
func (ep *ExtractionPhase) buildContextFromExtraction(
	artifact *models.ExtractionArtifact,
	input *ExtractionPhaseInput,
) *models.AnalysisContext {

	ctx := &models.AnalysisContext{
		UserID:         input.UserID,
		ConversationID: input.ConversationID,
		MessageCount:   input.MessageCount,

		// Built from extraction, not database rebuild
		ExtractedEntities:   artifact.Entities,
		ExtractedSource:     artifact.Source,
		ExtractedConfidence: artifact.AverageConfidence,
		ExtractionQuality: &models.ExtractionQuality{
			SubjectAttributed: artifact.SubjectAttributed,
			NegationPreserved: artifact.NegationPreserved,
			LLMExtraction:     artifact.LLMSuccess,
		},

		// Contacts from extraction (high-confidence)
		Contacts:      []models.Contact{},

		// Recent messages
		RecentMessages: input.RecentMessages,

		// User profile
		UserProfile: input.UserProfile,

		// Extraction timing
		ExtractionDuration: artifact.Duration,
	}

	// Add contacts from extraction (only high-confidence ones)
	for _, entity := range artifact.HighConfidenceContacts() {
		contact := models.Contact{
			Name:       entity.Value,
			Confidence: entity.Confidence,
		}
		ctx.Contacts = append(ctx.Contacts, contact)
	}

	// Add preferences from extraction
	preferences := []string{}
	characteristics := []string{}
	for _, entity := range artifact.Entities {
		if entity.Confidence >= 0.70 {
			switch entity.Type {
			case "preference", "negation":
				preferences = append(preferences, entity.Value)
			case "characteristic":
				characteristics = append(characteristics, entity.Value)
			}
		}
	}
	ctx.ExtractedPreferences = preferences
	ctx.ExtractedCharacteristics = characteristics

	// FIX #21: Restore response metadata from previous messages
	// Extract important state from last assistant message's metadata
	if len(input.RecentMessages) > 0 {
		for i := len(input.RecentMessages) - 1; i >= 0; i-- {
			msg := input.RecentMessages[i]
			// Find the most recent assistant message (our previous response)
			if msg.Role == "assistant" && msg.Metadata != nil {
				// Restore maturity state
				if maturityScore, ok := msg.Metadata["maturityScore"].(float64); ok {
					log.Printf("[ExtractionPhase] FIX #21: Restored previous maturityScore=%.2f from metadata", maturityScore)
				}
				if phase, ok := msg.Metadata["phase"].(string); ok {
					log.Printf("[ExtractionPhase] FIX #21: Restored previous phase=%s from metadata", phase)
				}
				// Restore strategy info
				if strategy, ok := msg.Metadata["responseStrategy"].(string); ok {
					log.Printf("[ExtractionPhase] FIX #21: Restored previous strategy=%s from metadata", strategy)
				}
				// Store entire metadata for use by later layers
				ctx.PreviousResponseMetadata = msg.Metadata
				break
			}
		}
	}

	log.Printf("[ExtractionPhase] Context built: %d entities, %d contacts, %d preferences, %d characteristics",
		len(artifact.Entities), len(ctx.Contacts), len(preferences), len(characteristics))

	return ctx
}

// FIX #9: mergeExtractedEntities combines accumulated and current entities, avoiding duplicates
// Keeps highest confidence version of duplicate entities by (type + value) key
func mergeExtractedEntities(accumulated, current []models.ExtractedEntity) []models.ExtractedEntity {
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
				log.Printf("[ExtractionPhase] FIX #9: Updated %s:%s confidence (%.2f → %.2f)",
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
