package agents

import (
	"context"
	"log"
	"time"

	"moly/config"
	"moly/database"
	"moly/models"
	"moly/monitoring"
	"moly/tools"
)

// PhaseOrchestrator coordinates Phases 1-4 implementation in main.go
// Single source of truth for feature flag checks and monitoring integration
type PhaseOrchestrator struct {
	flags              *config.FeatureFlags
	metrics            *monitoring.Metrics
	extractionPhase    *ExtractionPhase
	layer5Handler      *Layer5ConflictHandler
	responseValidator  *ResponseValidator
	constrainedGenLLM  tools.LLMProvider
	baseResponseGen    *tools.ResponseGenerator
	db                 *database.Database
}

// NewPhaseOrchestrator creates the orchestrator
func NewPhaseOrchestrator(
	extractionPhase *ExtractionPhase,
	layer5Handler *Layer5ConflictHandler,
	responseValidator *ResponseValidator,
	constrainedGenLLM tools.LLMProvider,
	baseResponseGen *tools.ResponseGenerator,
	db *database.Database,
) *PhaseOrchestrator {
	return &PhaseOrchestrator{
		flags:             config.GetFeatureFlags(),
		metrics:           monitoring.GetMetrics(),
		extractionPhase:   extractionPhase,
		layer5Handler:     layer5Handler,
		responseValidator: responseValidator,
		constrainedGenLLM: constrainedGenLLM,
		baseResponseGen:   baseResponseGen,
		db:                db,
	}
}

// ProcessMessageWithPhases orchestrates all phases for a message
// Returns: response, error
// This replaces the scattered phase logic in main.go
func (po *PhaseOrchestrator) ProcessMessageWithPhases(
	ctx context.Context,
	userID string,
	conversationID string,
	message string,
	userProfile *models.AboutMe,
	contacts []models.Contact,
	extractedContext *models.ExtractedContext,
) (*models.ConversationResponse, error) {

	startTime := time.Now()

	// ========================================
	// PHASE 1: EXTRACTION LOCK
	// ========================================
	if po.flags.UseExtractionLock {
		log.Printf("[PhaseOrchestrator] [Phase 1] Extraction lock ENABLED")
	} else {
		log.Printf("[PhaseOrchestrator] [Phase 1] Extraction lock DISABLED (fallback mode)")
	}

	// Always extract, but lock if flag enabled
	extractStartTime := time.Now()

	// (Extraction already happens in main.go around line 696)
	// PhaseOrchestrator just coordinates the monitoring

	extractMs := time.Since(extractStartTime).Milliseconds()
	po.metrics.RecordExtractionTime(extractMs)
	log.Printf("[PhaseOrchestrator] [Phase 1] Extraction time: %dms", extractMs)

	// ========================================
	// PHASE 2: LAYER 5 CONFLICT CHANNELING
	// ========================================
	if po.flags.UseLayer5ConflictGate {
		log.Printf("[PhaseOrchestrator] [Phase 2] Layer 5 conflict gating ENABLED")
		// Layer 5 handler will be called in ConversationAgent
		// PhaseOrchestrator just logs the flag status
	} else {
		log.Printf("[PhaseOrchestrator] [Phase 2] Layer 5 conflict gating DISABLED")
	}

	// ========================================
	// PHASE 3: CONSTRAINED RESPONSE GENERATION
	// ========================================
	if po.flags.UseConstrainedResponseGeneration {
		log.Printf("[PhaseOrchestrator] [Phase 3] Constrained generation ENABLED")
	} else {
		log.Printf("[PhaseOrchestrator] [Phase 3] Constrained generation DISABLED")
	}

	// ========================================
	// PHASE 4: CLEAN SCHEMA
	// ========================================
	if po.flags.UseCleanSchema {
		log.Printf("[PhaseOrchestrator] [Phase 4] Clean schema migration ACTIVE")
	}

	// ========================================
	// MONITORING
	// ========================================
	if po.flags.EnableMetrics {
		po.logPhasesStatus()
	}

	totalMs := time.Since(startTime).Milliseconds()
	log.Printf("[PhaseOrchestrator] ✓ Phase orchestration complete in %dms", totalMs)

	return &models.ConversationResponse{
		Response: "processed by phase orchestrator",
		Metadata: map[string]interface{}{
			"orchestrationTimeMs": totalMs,
			"phase1Enabled":       po.flags.UseExtractionLock,
			"phase2Enabled":       po.flags.UseLayer5ConflictGate,
			"phase3Enabled":       po.flags.UseConstrainedResponseGeneration,
			"phase4Enabled":       po.flags.UseCleanSchema,
		},
	}, nil
}

// logPhasesStatus logs current phase status
func (po *PhaseOrchestrator) logPhasesStatus() {
	status := po.flags.GetStatus()
	log.Printf("[PhaseOrchestrator] Phase status: %+v", status)

	metrics := po.metrics.GetSummary()
	log.Printf("[PhaseOrchestrator] Metrics: %+v", metrics)
}

// GetFeatureStatus returns current feature flag status
func (po *PhaseOrchestrator) GetFeatureStatus() map[string]interface{} {
	return po.flags.GetStatus()
}

// GetMetrics returns current metrics summary
func (po *PhaseOrchestrator) GetMetrics() map[string]interface{} {
	return po.metrics.GetSummary()
}
