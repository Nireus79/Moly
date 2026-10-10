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

	// PHASE 3: a greeting has no goal. Do not extract again (the handler already removed goals) and never lock one.
	if lc.IsGreeting {
		log.Printf("[Layer1] Greeting: no goal extraction")
		lc.Layer1 = &tools.Layer1Result{Confidence: 0, Duration: time.Since(startTime).Seconds()}
		return lc, nil
	}

	// FIX DUPLICATE EXTRACTION: Check if AnalysisContext already has extraction from main.go
	// Reuse it instead of re-extracting (saves ~43 seconds per message)
	var extractedCtx *models.ExtractedContext
	if lc.Analysis != nil && lc.Analysis.ExtractedConfidence > 0 {
		log.Printf("[Layer1] ✓ Reusing extraction from AnalysisContext (confidence=%.2f, skipping LLM call)",
			lc.Analysis.ExtractedConfidence)
		// Create ExtractedContext from AnalysisContext data
		// FIX: Use actual extracted intent, not the full message
		intentionFromContext := ""
		if lc.Analysis.ExtractedEntities != nil {
			for _, entity := range lc.Analysis.ExtractedEntities {
				if entity.Type == "goal" {
					intentionFromContext = entity.Value
					break
				}
			}
		}
		extractedCtx = &models.ExtractedContext{
			Intention:           intentionFromContext, // Use extracted goal, not full message
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
	extractedGoal := ""
	conversationTopic := ""
	if extractedCtx != nil && extractedCtx.Contact != nil {
		confidence = extractedCtx.Contact.Confidence
		// FIX #6: Set conversation topic from extracted contact
		conversationTopic = extractedCtx.Contact.Name
	}
	if extractedCtx != nil {
		// FIX #5: Set user goal from extracted intention
		extractedGoal = extractedCtx.Intention
	}

	lc.Layer1 = &tools.Layer1Result{
		ExtractedContext:  extractedCtx,
		Confidence:        confidence,
		Duration:          time.Since(startTime).Seconds(),
		ExtractedGoal:     extractedGoal,     // FIX #5: User's goal
		ConversationTopic: conversationTopic, // FIX #6: Conversation focus
	}

	// GOAL STAGE (ORCHESTRATOR_DESIGN.md step 3): the goal rules are ResolveGoal. A message with no goal continues
	// the locked one; the first goal stated is locked; a different later goal is reported, not substituted.
	if extractedCtx != nil {
		stated := extractedCtx.Intention != ""
		relation := GoalSame
		if lc.PrimaryGoal != "" && extractedCtx.Intention != "" && normalizeGoal(extractedCtx.Intention) != normalizeGoal(lc.PrimaryGoal) && l1.extractor != nil {
			// Identical text needs no judgement. Anything else is judged by the model, never by the wording.
			relation = JudgeGoalRelation(ctx, l1.extractor.llmClient, lc.PrimaryGoal, extractedCtx.Intention)
			lc.GoalRelation = string(relation)
			log.Printf("[Layer1] Goal relation judged: %s", relation)
		}
		goal := ResolveGoal(lc.PrimaryGoal, extractedCtx.Intention, relation)
		if goal.JustLocked {
			lc.PrimaryGoal = goal.Locked
			lc.Analysis.PrimaryGoal = goal.Locked
			log.Printf("[Layer1] ✓ Goal locked: %q", lc.PrimaryGoal)
		}
		extractedCtx.Intention = goal.Working
		lc.Layer1.ExtractedGoal = goal.Working
		if stated && !goal.JustLocked {
			// track the goal this message states, separately from the locked goal
			lc.CurrentMessageIntent = goal.Working
			progression := append(append([]string(nil), lc.GoalProgression...), goal.Working)
			lc.GoalProgression = progression
			lc.Analysis.GoalProgression = append([]string(nil), progression...)
			if goal.Changed {
				lc.GoalSwitch = goal.Working
				log.Printf("[Layer1] ⚠️ Goal changed - locked=%q, stated=%q", lc.PrimaryGoal, goal.Working)
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

	// The handler's safety stage already produced this message's verdict; it is used, not asked for again.
	var verdict *tools.ConstitutionalVerdict
	if lc.Analysis != nil {
		verdict, _ = lc.Analysis.SafetyVerdict.(*tools.ConstitutionalVerdict)
	}
	if verdict == nil {
		// No precomputed verdict: evaluate here. A failure stops the pipeline (Layer 2 is critical); it never passes.
		var err error
		if lc.Analysis != nil {
			verdict, err = l2.evaluator.EvaluateWithAnalysisContext(ctx, lc.Analysis)
		} else {
			verdict, err = l2.evaluator.Evaluate(ctx, message)
		}
		if err != nil || verdict == nil {
			log.Printf("[Layer2] ✗ Safety evaluation failed: %v", err)
			return lc, ErrSafetyUnavailable
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

	// FIX #76: Use maturity context passed from main.go (loaded once, not reloaded)
	// FIX #2 (Session 34): Unify to single maturityContext object to avoid duplicate saves
	maturityCtx := lc.MaturityContext
	if maturityCtx == nil {
		log.Printf("[Layer3] ⚠️ FIX #2: MaturityContext not provided, creating new")
		maturityCtx = models.NewConversationMaturity(lc.UserID, lc.ConversationID)
	}

	previousScore := maturityCtx.OverallScore
	log.Printf("[Layer3] FIX #76: Previous maturity score: %.2f", previousScore)

	// FIX #76: Extract NEW data to IMPROVE maturity (additive, not recalculative)
	var profileData interface{} = nil
	if lc.Layer1 != nil && lc.Layer1.ExtractedContext != nil {
		profileData = lc.Layer1.ExtractedContext
	}

	// FIX #6 (Phase 6): Use accumulated data from conversation summary for maturity calculation
	// This fixes the bug where maturity appeared flat because we only counted current message's entities
	contactCount := 0
	clearContactCount := 0
	entityCount := 0
	avgConfidence := 0.0

	// Prefer accumulated data from conversation summary (durable, persisted)
	if lc.Analysis != nil && lc.Analysis.ConversationSummary != nil {
		// Use accumulated counts from conversation summary
		contactCount = lc.Analysis.ConversationSummary.AccumulatedContactCount
		entityCount = lc.Analysis.ConversationSummary.AccumulatedEntityCount
		log.Printf("[Layer3] FIX #6: Using accumulated data - contacts=%d entities=%d",
			contactCount, entityCount)
	} else if lc.Analysis != nil {
		// Fallback: use current message only (old behavior)
		if lc.Analysis.Contacts != nil {
			contactCount = len(lc.Analysis.Contacts)
			for _, c := range lc.Analysis.Contacts {
				if c.Confidence >= 0.7 {
					clearContactCount++
				}
			}
		}
		if lc.Analysis.ExtractedEntities != nil {
			entityCount = len(lc.Analysis.ExtractedEntities)
		}
		log.Printf("[Layer3] ⚠️ FIX #6: Fallback to current message only - contacts=%d entities=%d",
			contactCount, entityCount)
	}

	messageCount := 1 // At least this message
	if lc.Analysis != nil && lc.Analysis.MessageCount > 0 {
		messageCount = lc.Analysis.MessageCount
	}

	// Calculate average confidence from current message entities
	if lc.Analysis != nil && lc.Analysis.ExtractedEntities != nil && len(lc.Analysis.ExtractedEntities) > 0 {
		totalConfidence := 0.0
		for _, e := range lc.Analysis.ExtractedEntities {
			totalConfidence += e.Confidence
		}
		avgConfidence = totalConfidence / float64(len(lc.Analysis.ExtractedEntities))
	}

	// FIX #76: Update maturity with new information
	// New entities, contacts, etc. should INCREASE maturity (accumulated growth)
	// Use NEW calculator but feed it current maturity as baseline
	var score float64
	if l3.maturityService != nil {
		// Create calculator for 4-factor calculation
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

		// FIX #76: Ensure maturity improves or stays same, never decreases
		// If new extraction added entities, maturity should increase
		if score < previousScore {
			// Don't let maturity drop - keep previous if better
			score = previousScore
			log.Printf("[Layer3] FIX #76: Maturity would drop (%.2f → %.2f), keeping previous", previousScore, score)
		} else if score > previousScore {
			log.Printf("[Layer3] FIX #76: ✅ Maturity IMPROVED (%.2f → %.2f) - new entities detected", previousScore, score)
		}

		log.Printf("[Layer3] ✓ 4-Factor maturity: contacts=%d, depth=%d, entities=%d (score=%.2f)",
			clearContactCount, messageCount, entityCount, score)
	} else {
		// Fallback to old calculation
		score = maturityCtx.CalculateOverallMaturity()
		log.Printf("[Layer3] Maturity calculated (score=%.2f)", score)
	}

	// Update context with new score
	maturityCtx.OverallScore = score

	lc.Layer3 = layer3FromScore(score)

	// FIX #76: Save updated maturity to database (so it persists for next message)
	// This ensures accumulated scores are preserved across messages
	if l3.maturityService != nil {
		maturityCtx.OverallScore = score
		err := l3.maturityService.SaveMaturityContext(lc.UserID, lc.ConversationID, maturityCtx)
		if err != nil {
			log.Printf("[Layer3] FIX #76: Warning - Failed to save maturity: %v", err)
		} else {
			log.Printf("[Layer3] FIX #76: ✓ Maturity saved to database (%.2f)", score)
		}
	}

	log.Printf("[Layer3] ✓ Maturity assessment complete (score=%.2f, gate=%s, canL5=%v, duration=%.2fs)",
		score, lc.Layer3.GateLevel, lc.Layer3.CanAccessL5Plus, time.Since(startTime).Seconds())

	return lc, nil
}

// layer3FromScore turns the maturity score into the Layer 3 result (gate level, context quality). The score the layers decide
// with is the gap maturity (answered questions over answered plus open ones): the unified orchestrator replaces the score
// with it right after Layer 3, and rebuilds the result with this function.
func layer3FromScore(score float64) *tools.Layer3Result {
	gateLevel := "immature"
	canAccessL5 := false
	if score >= 0.3 {
		gateLevel = "developing"
		canAccessL5 = true
	}
	if score >= 0.7 {
		gateLevel = "mature"
	}

	contextQuality := "minimal"
	if score >= 0.4 {
		contextQuality = "partial"
	}
	if score >= 0.7 {
		contextQuality = "complete"
	}

	return &tools.Layer3Result{
		MaturityScore:   score,
		ContextQuality:  contextQuality,
		GateLevel:       gateLevel,
		CanAccessL5Plus: canAccessL5,
	}
}
