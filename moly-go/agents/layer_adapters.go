package agents

import (
	"context"
	"log"
	"time"

	"moly/models"
	"moly/database"
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

// Layer3MaturityAssessmentAdapter is Layer 3: it reads the conversation's gap maturity (answered questions over answered
// plus open ones) and turns it into the gate level and context quality the later layers read.
type Layer3MaturityAssessmentAdapter struct {
	repo *database.ClarificationQuestionRepository
}

// NewLayer3MaturityAssessmentAdapter creates a new Layer 3 adapter
func NewLayer3MaturityAssessmentAdapter(repo *database.ClarificationQuestionRepository) *Layer3MaturityAssessmentAdapter {
	return &Layer3MaturityAssessmentAdapter{repo: repo}
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
	score := ConversationGapMaturity(l3.repo, lc.ConversationID)
	lc.Layer3 = layer3FromScore(score)
	log.Printf("[Layer3] ✓ Maturity assessment complete (score=%.2f, gate=%s, canL5=%v, duration=%.2fs)",
		score, lc.Layer3.GateLevel, lc.Layer3.CanAccessL5Plus, time.Since(startTime).Seconds())
	return lc, nil
}

// layer3FromScore turns the gap maturity into the Layer 3 result (gate level, context quality).
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
