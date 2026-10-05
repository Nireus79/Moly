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

	// FIX #11: Phase 3 - Check message summary cache for recent messages
	// Skip re-extraction if we have cached summary from Phase 1
	if lc.HasMessageSummary(lc.MessageID) {
		summary := lc.GetMessageSummary(lc.MessageID)
		if summaryMap, ok := summary.(map[string]interface{}); ok {
			if entities, hasEntities := summaryMap["extractedEntities"].([]interface{}); hasEntities && len(entities) > 0 {
				log.Printf("[Layer1] FIX #11: ✓ Using message summary cache for %s (%d entities, skipping re-extraction)",
					lc.MessageID, len(entities))

				// Create extracted context from summary
				extractedCtx := &models.ExtractedContext{
					Intention: "cached",
					Goals:     []string{},
				}
				if confidence, ok := summaryMap["confidence"].(float64); ok {
					extractedCtx.IntentionConfidence = confidence
				}

				lc.Layer1 = &tools.Layer1Result{
					ExtractedContext: extractedCtx,
					Confidence:       0.90, // High confidence in cached data
					Duration:         time.Since(startTime).Seconds(),
				}
				return lc, nil
			}
		}
	}

	// FIX DUPLICATE EXTRACTION: Check if AnalysisContext already has extraction from main.go
	// Reuse it instead of re-extracting (saves ~43 seconds per message)
	var extractedCtx *models.ExtractedContext
	if lc.Analysis != nil && lc.Analysis.ExtractedConfidence > 0 {
		log.Printf("[Layer1] ✓ Reusing extraction from AnalysisContext (confidence=%.2f, skipping LLM call)",
			lc.Analysis.ExtractedConfidence)
		// Create ExtractedContext from AnalysisContext data
		extractedCtx = &models.ExtractedContext{
			Intention:           lc.Analysis.CurrentMessage, // Use as placeholder
			IntentionConfidence: lc.Analysis.ExtractedConfidence,
			Goals:               []string{},
			UserValues:          []string{},
		}
		if len(lc.Analysis.Contacts) > 0 {
			extractedCtx.Contact = &models.ExtractedContact{
				Name:         lc.Analysis.Contacts[0].Name,
				Relationship: lc.Analysis.Contacts[0].Relationship,
				Confidence:   lc.Analysis.ExtractedConfidence,
				Evidence:     "Extracted in AnalysisContext phase",
			}
		}
	} else {
		// Run extraction only if not already done
		var err error
		extractedCtx, err = l1.extractor.Extract(ctx, message)
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
	}

	// Store results - guard against nil Contact
	confidence := 0.0
	if extractedCtx != nil && extractedCtx.Contact != nil {
		confidence = extractedCtx.Contact.Confidence
	}
	lc.Layer1 = &tools.Layer1Result{
		ExtractedContext: extractedCtx,
		Confidence:       confidence,
		Duration:         time.Since(startTime).Seconds(),
	}

	// FIX #4: Lock primary goal on Message 1, track intent separately on subsequent messages
	if extractedCtx != nil {
		if lc.IsMessageOne {
			// Message 1: Lock the primary goal
			lc.PrimaryGoal = extractedCtx.Intention
			lc.Analysis.PrimaryGoal = extractedCtx.Intention
			log.Printf("[Layer1] ✓ FIX #4: PRIMARY GOAL LOCKED on Message 1: %q", lc.PrimaryGoal)
		} else {
			// Subsequent messages: Track current intent separately
			lc.CurrentMessageIntent = extractedCtx.Intention
			lc.GoalProgression = append(lc.GoalProgression, extractedCtx.Intention)
			if lc.Analysis.GoalProgression == nil {
				lc.Analysis.GoalProgression = make([]string, 0)
			}
			lc.Analysis.GoalProgression = lc.GoalProgression

			if lc.PrimaryGoal != "" {
				if lc.CurrentMessageIntent == lc.PrimaryGoal {
					log.Printf("[Layer1] ✓ FIX #4: Goal consistent - primary=%q, current=%q",
						lc.PrimaryGoal, lc.CurrentMessageIntent)
				} else {
					log.Printf("[Layer1] ⚠️ FIX #4: Goal changed - primary=%q, current=%q",
						lc.PrimaryGoal, lc.CurrentMessageIntent)
				}
			}
		}
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

	// FIX #11: Phase 3 - Check message summary cache for principle evaluation
	// Skip re-evaluation if we have cached summary with high confidence
	if lc.HasMessageSummary(lc.MessageID) {
		summary := lc.GetMessageSummary(lc.MessageID)
		if summaryMap, ok := summary.(map[string]interface{}); ok {
			if confidence, ok := summaryMap["confidence"].(float64); ok && confidence >= 0.80 {
				log.Printf("[Layer2] FIX #11: ✓ Using cached summary for %s (confidence=%.2f, skipping re-evaluation)",
					lc.MessageID, confidence)

				// Return cached verdict - assume principles already evaluated
				lc.Layer2 = &tools.Layer2Result{
					Verdict: &tools.ConstitutionalVerdict{
						Allowed:         true,
						OverallSeverity: "low",
						IsObviousHarm:   false,
					},
					IsObviousHarm:     false,
					MatchedPrinciples: []string{},
					ShouldProceedToL6: true,
				}
				log.Printf("[Layer2] ✓ Principle check complete (cached, duration=%.2fs)",
					time.Since(startTime).Seconds())
				return lc, nil
			}
		}
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

	// FIX #11: Phase 3B - Check message summary cache for maturity assessment
	// Skip recalculation for cached messages with high confidence
	if lc.HasMessageSummary(lc.MessageID) {
		summary := lc.GetMessageSummary(lc.MessageID)
		if summaryMap, ok := summary.(map[string]interface{}); ok {
			if confidence, ok := summaryMap["confidence"].(float64); ok && confidence >= 0.85 {
				log.Printf("[Layer3] FIX #11: ✓ Using cached summary for %s (confidence=%.2f, skipping maturity recalc)",
					lc.MessageID, confidence)

				// Return cached maturity - assume already calculated
				lc.Layer3 = &tools.Layer3Result{
					MaturityScore:  confidence, // Use extraction confidence as proxy
					ContextQuality: "partial",
					GateLevel:      "developing",
					CanAccessL5Plus: true,
				}
				log.Printf("[Layer3] ✓ Maturity assessment complete (cached, duration=%.2fs)",
					time.Since(startTime).Seconds())
				return lc, nil
			}
		}
	}

	// Load or create maturity context
	maturityCtx, err := l3.maturityService.LoadOrCreateMaturityContext(lc.UserID, lc.ConversationID)
	if err != nil {
		log.Printf("[Layer3] ⚠️ Failed to load maturity context: %v", err)
		maturityCtx = models.NewConversationMaturity(lc.UserID, lc.ConversationID)
	}

	// FIX 3: Use 4-factor maturity calculation (profile, contacts, depth, entities)
	// Extract data for 4-factor calculation
	var profileData interface{} = nil
	if lc.Layer1 != nil && lc.Layer1.ExtractedContext != nil {
		profileData = lc.Layer1.ExtractedContext
	}

	contactCount := 0
	clearContactCount := 0
	if lc.Analysis != nil && lc.Analysis.Contacts != nil {
		contactCount = len(lc.Analysis.Contacts)
		for _, c := range lc.Analysis.Contacts {
			if c.Confidence >= 0.7 {
				clearContactCount++
			}
		}
	}

	messageCount := 1 // At least this message
	if lc.Analysis != nil && lc.Analysis.MessageCount > 0 {
		messageCount = lc.Analysis.MessageCount
	}

	entityCount := 0
	avgConfidence := 0.0
	if lc.Analysis != nil && lc.Analysis.ExtractedEntities != nil {
		entityCount = len(lc.Analysis.ExtractedEntities)
		totalConfidence := 0.0
		for _, e := range lc.Analysis.ExtractedEntities {
			totalConfidence += e.Confidence
		}
		if entityCount > 0 {
			avgConfidence = totalConfidence / float64(entityCount)
		}
	}

	// Get MaturityCalculator from maturityService
	var score float64
	if l3.maturityService != nil {
		// Create a MaturityCalculator instance for 4-factor calculation
		calc := tools.NewMaturityCalculator()
		phaseMaturity := calc.BuildPhaseMaturityWithFactors(
			profileData,
			contactCount,
			clearContactCount,
			messageCount,
			entityCount,
			avgConfidence,
		)
		score = phaseMaturity.OverallScore
		log.Printf("[Layer3] ✓ 4-Factor maturity: contacts=%d, depth=%d, entities=%d (score=%.2f)",
			clearContactCount, messageCount, entityCount, score)
	} else {
		// Fallback to old calculation
		score = maturityCtx.CalculateOverallMaturity()
		log.Printf("[Layer3] Maturity calculated (score=%.2f)", score)
	}

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
