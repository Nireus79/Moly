package agents

import (
	"context"
	"log"
	"time"

	"moly/models"
	"moly/storage"
	"moly/tools"
)

// Layer1ContextExtractionAdapter wraps ContextExtractor as a Layer interface
type Layer1ContextExtractionAdapter struct {
	extractor *ContextExtractor
	cache     *tools.ExtractionCache
}

// NewLayer1ContextExtractionAdapter creates a new Layer 1 adapter
func NewLayer1ContextExtractionAdapter(
	extractor *ContextExtractor,
	cache *tools.ExtractionCache,
) *Layer1ContextExtractionAdapter {
	return &Layer1ContextExtractionAdapter{
		extractor: extractor,
		cache:     cache,
	}
}

// Name returns the layer identifier
func (l1 *Layer1ContextExtractionAdapter) Name() string {
	return "Layer1-ContextExtraction"
}

// CanSkip returns true if message is empty
func (l1 *Layer1ContextExtractionAdapter) CanSkip(lc *tools.LayerContext) bool {
	return lc.GetMessage() == ""
}

// Priority returns layer priority (Layer 1 is critical)
func (l1 *Layer1ContextExtractionAdapter) Priority() int {
	return 90
}

// Process executes Layer 1 extraction
func (l1 *Layer1ContextExtractionAdapter) Process(ctx context.Context, lc *tools.LayerContext) (*tools.LayerContext, error) {
	startTime := time.Now()
	message := lc.GetMessage()

	if message == "" {
		log.Printf("[Layer1] Skipping empty message")
		return lc, nil
	}

	// Check cache first
	if cachedEntities, found := l1.cache.Get(lc.UserID, lc.MessageID); found {
		log.Printf("[Layer1] ✓ Using cached extraction (%d entities)", len(cachedEntities))

		lc.Analysis.CachedEntities = cachedEntities
		lc.Layer1 = &tools.Layer1Result{
			Confidence: 0.95,
			Duration:   time.Since(startTime).Seconds(),
		}
		return lc, nil
	}

	// Run extraction
	extractedCtx, err := l1.extractor.Extract(ctx, message)
	if err != nil {
		log.Printf("[Layer1] ⚠️ Extraction failed: %v", err)
		// Graceful degradation - continue without new extraction
		lc.Layer1 = &tools.Layer1Result{
			ExtractedContext: nil,
			Confidence:       0,
			Duration:         time.Since(startTime).Seconds(),
		}
		return lc, nil
	}

	// Cache for future use
	if extractedCtx != nil && extractedCtx.Contact != nil {
		l1.cache.Set(lc.UserID, lc.MessageID, []models.ExtractedEntity{
			{
				Value:      extractedCtx.Contact.Name,
				Type:       "contact",
				Confidence: extractedCtx.Contact.Confidence,
			},
		})
	}

	// Store results
	lc.Layer1 = &tools.Layer1Result{
		ExtractedContext: extractedCtx,
		Confidence:       extractedCtx.Contact.Confidence,
		Duration:         time.Since(startTime).Seconds(),
	}

	log.Printf("[Layer1] ✓ Extracted context (duration=%.2fs)", lc.Layer1.Duration)
	return lc, nil
}

// Layer2PrincipleCheckAdapter wraps ConstitutionalEvaluator as Layer 2
type Layer2PrincipleCheckAdapter struct {
	evaluator *tools.ConstitutionalEvaluator
}

// NewLayer2PrincipleCheckAdapter creates a new Layer 2 adapter
func NewLayer2PrincipleCheckAdapter(evaluator *tools.ConstitutionalEvaluator) *Layer2PrincipleCheckAdapter {
	return &Layer2PrincipleCheckAdapter{
		evaluator: evaluator,
	}
}

// Name returns layer identifier
func (l2 *Layer2PrincipleCheckAdapter) Name() string {
	return "Layer2-PrincipleCheck"
}

// CanSkip returns false - Layer 2 always runs
func (l2 *Layer2PrincipleCheckAdapter) CanSkip(lc *tools.LayerContext) bool {
	return lc.GetMessage() == ""
}

// Priority returns layer priority
func (l2 *Layer2PrincipleCheckAdapter) Priority() int {
	return 95 // Critical
}

// Process executes Layer 2 principle checking
func (l2 *Layer2PrincipleCheckAdapter) Process(ctx context.Context, lc *tools.LayerContext) (*tools.LayerContext, error) {
	startTime := time.Now()
	message := lc.GetMessage()

	if message == "" {
		return lc, nil
	}

	// Use AnalysisContext if available, otherwise use message
	var verdict *tools.ConstitutionalVerdict
	var err error

	if lc.Analysis != nil {
		verdict, err = l2.evaluator.EvaluateWithAnalysisContext(ctx, lc.Analysis)
	} else {
		verdict, err = l2.evaluator.Evaluate(ctx, message)
	}

	if err != nil {
		log.Printf("[Layer2] ⚠️ Evaluation failed: %v", err)
		// Create default verdict on error
		verdict = &tools.ConstitutionalVerdict{
			Allowed:         true,
			OverallSeverity: "unknown",
		}
	}

	// Extract principle names
	principleNames := make([]string, 0, len(verdict.MatchedPrinciples))
	for _, m := range verdict.MatchedPrinciples {
		principleNames = append(principleNames, m.Name)
	}

	// Store results
	lc.Layer2 = &tools.Layer2Result{
		Verdict:           verdict,
		IsObviousHarm:     verdict.IsObviousHarm,
		MatchedPrinciples: principleNames,
		ShouldProceedToL6: !verdict.IsObviousHarm,
	}

	// Check for obvious harm - stop if found
	if verdict.IsObviousHarm {
		log.Printf("[Layer2] ⚠️ OBVIOUS HARM detected - will proceed to Layer 11 (denial)")
		lc.ShouldStop = true
		lc.StopReason = "obvious_harm"
	}

	log.Printf("[Layer2] ✓ Principle check complete (allowed=%v, severity=%s, duration=%.2fs)",
		verdict.Allowed, verdict.OverallSeverity, time.Since(startTime).Seconds())

	return lc, nil
}

// Layer3MaturityAssessmentAdapter wraps MaturityService as Layer 3
type Layer3MaturityAssessmentAdapter struct {
	maturityService *storage.MaturityService
}

// NewLayer3MaturityAssessmentAdapter creates a new Layer 3 adapter
func NewLayer3MaturityAssessmentAdapter(maturityService *storage.MaturityService) *Layer3MaturityAssessmentAdapter {
	return &Layer3MaturityAssessmentAdapter{
		maturityService: maturityService,
	}
}

// Name returns layer identifier
func (l3 *Layer3MaturityAssessmentAdapter) Name() string {
	return "Layer3-MaturityAssessment"
}

// CanSkip returns false - Layer 3 always runs
func (l3 *Layer3MaturityAssessmentAdapter) CanSkip(lc *tools.LayerContext) bool {
	return false
}

// Priority returns layer priority
func (l3 *Layer3MaturityAssessmentAdapter) Priority() int {
	return 85
}

// Process executes Layer 3 maturity assessment
func (l3 *Layer3MaturityAssessmentAdapter) Process(ctx context.Context, lc *tools.LayerContext) (*tools.LayerContext, error) {
	startTime := time.Now()

	// Load or create maturity context
	maturityCalc, err := l3.maturityService.LoadOrCreateMaturityContext(lc.UserID, lc.ConversationID)
	if err != nil {
		log.Printf("[Layer3] ⚠️ Failed to load maturity context: %v", err)
		maturityCalc = tools.NewMaturityCalculator()
	}

	// Calculate maturity NOW (with full context from AnalysisContext)
	score := maturityCalc.CalculateOverallMaturity()

	// Determine gate level based on maturity
	gateLevel := "immature"
	canAccessL5 := false
	if score >= 0.3 {
		gateLevel = "developing"
		canAccessL5 = true
	}
	if score >= 0.7 {
		gateLevel = "mature"
	}

	// Determine context quality
	contextQuality := "minimal"
	if score >= 0.4 {
		contextQuality = "partial"
	}
	if score >= 0.7 {
		contextQuality = "complete"
	}

	// Store results
	lc.Layer3 = &tools.Layer3Result{
		MaturityScore:   score,
		ContextQuality:  contextQuality,
		GateLevel:       gateLevel,
		CanAccessL5Plus: canAccessL5,
	}

	log.Printf("[Layer3] ✓ Maturity assessment complete (score=%.2f, gate=%s, canL5=%v, duration=%.2fs)",
		score, gateLevel, canAccessL5, time.Since(startTime).Seconds())

	return lc, nil
}
