package agents

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"moly/database"
	"moly/models"
	"moly/tools"
)

// exitInputs are facts the agent already derived from the extraction before the decision.
type exitInputs struct {
	intent              IntentAnalysis
	contact             *models.ExtractedContact
	contactMessage      bool // the message concerns a named or known person
	directCommunication bool // the intention involves communicating directly (transparency or autonomy principles)
}

// exitEvidence is what the probes found out. A render step reads only its own part.
type exitEvidence struct {
	layerCtx         *tools.LayerContext
	deepen           bool
	deepenPrinciples []string // the principles the deepening judgement found
	contactCheck     string
	clarif           ClarificationNeed
	principle        string
	principleQ       string
	persistent       string
	lastClar         string
	shift            SubjectShift
}

// phaseLabel is the old accomplishment-based phase of the conversation. It is a label for the metadata only: no reply
// decision reads it. Whether Moly asks is decided by the open gaps and the risk checks, never by this phase.
func phaseLabel(ctx models.Context) string {
	if ctx.Maturity == nil {
		return "initial"
	}
	return ctx.Maturity.EstimateCurrentPhase()
}

// replyByExits decides whether one of the question exits answers this message, and renders it if so.
// ORCHESTRATOR_DESIGN.md step 1: the decision is DecideReply; the probes only find facts, in precedence order.
// It returns true when the reply is complete. When it returns false, the normal reply generation continues.
func (ca *conversationAgent) replyByExits(ctx models.Context, analysisCtx *models.AnalysisContext, userMessage string, in exitInputs, response *models.ConversationResponse, startTime time.Time) (bool, bool) {
	intent := in.intent
	var ev exitEvidence
	if analysisCtx != nil && analysisCtx.LayerResults != nil {
		ev.layerCtx, _ = analysisCtx.LayerResults.(*tools.LayerContext)
	}

	probes := []factProbe{
		{"deny", func(f *ReplyFacts) {
			f.Deny = ev.layerCtx != nil && ev.layerCtx.Layer11 != nil && ev.layerCtx.Layer11.ShouldDeny
		}},
		{"safety_alert", func(f *ReplyFacts) {
			f.SafetyAlert = ctx.PrecomputedSafetyVerdict != nil
		}},
		{"clarity", func(f *ReplyFacts) {
			if ca.clarityAnalyzer == nil {
				log.Printf("[ConversationAgent] WARNING: Clarity analyzer not initialized, skipping diagnostic gate")
				return
			}
			var clarity *MessageAnalysis
			if ctx.BoundedAnalysisContext != nil {
				clarity = ca.clarityAnalyzer.AnalyzeWithAnalysisContext(ctx.BoundedAnalysisContext)
			} else {
				clarity = ca.clarityAnalyzer.Analyze(userMessage, ctx.ConversationHistory)
			}
			log.Printf("[ConversationAgent] Clarity assessment: priority=%s clarity=%.2f can_proceed=%v clarifications=%d",
				clarity.Priority, clarity.ClarityScore, clarity.CanProceed, len(clarity.RequiredClarifications))
			response.Metadata["clarityScore"] = clarity.ClarityScore
			response.Metadata["messageQuality"] = clarity.MessageQuality
			response.Metadata["priority"] = clarity.Priority
			if len(clarity.KeyConcerns) > 0 {
				response.Metadata["keyConcerns"] = clarity.KeyConcerns
			}
			if len(clarity.RequiredClarifications) > 0 {
				response.Metadata["clarificationsNeeded"] = len(clarity.RequiredClarifications)
			}
			if !clarity.CanProceed && len(clarity.RequiredClarifications) > 0 {
				ev.clarif = clarity.RequiredClarifications[0]
				f.Clarity = true
			}
		}},
		{"gap", func(f *ReplyFacts) {
			if len(ctx.Gaps) < 1 || ca.responseGenerator == nil {
				return
			}
			f.OpenGap = true
		}},
		{"principle", func(f *ReplyFacts) {
			if ctx.ExtractedContext == nil {
				return
			}
			has, id, q := ca.detectPrincipleConcerns(userMessage, ctx.ExtractedContext)
			if !has || id == "" || q == "" {
				return
			}
			ev.principle, ev.principleQ = id, q
			f.PrincipleOpen = true
		}},
		{"persistent", func(f *ReplyFacts) {
			if len(ctx.ConversationHistory) < 2 {
				return
			}
			repeated, last := ca.detectRepeatedConcern(userMessage, ctx.ConversationHistory, ctx.ExtractedContext)
			if !repeated {
				return
			}
			ev.lastClar = last
			f.Persistent = true
		}},
		{"topic_shift", func(f *ReplyFacts) {
			if len(ctx.ConversationHistory) <= 1 || ca.subjectShiftDetector == nil || len(ctx.Gaps) != 0 {
				return
			}
			// The history is [current, oldest ... newest]: the previous user message is the last earlier user entry.
			previous := ""
			for i := len(ctx.ConversationHistory) - 1; i >= 1; i-- {
				if ctx.ConversationHistory[i].Role == "user" {
					previous = ctx.ConversationHistory[i].Content
					break
				}
			}
			if previous == "" {
				return
			}
			shiftCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			shifts, err := ca.subjectShiftDetector.DetectShiftsWithContext(shiftCtx, userMessage, previous)
			cancel()
			if err == context.DeadlineExceeded {
				log.Printf("[ConversationAgent] [Layer 9] Subject shift detection timed out, continuing without shift analysis")
				response.Metadata["subject_shift_fallback"] = true
				response.Metadata["subject_shift_reason"] = "timeout"
				return
			}
			if len(shifts) == 0 {
				return
			}
			ev.shift = shifts[0]
			f.TopicShift = true
		}},
		{"contact", func(f *ReplyFacts) {
			if !(in.contactMessage || in.directCommunication) {
				return
			}
			if in.contact != nil && in.contact.Name != "" &&
				ca.shouldRequireClarificationForContact(in.contact, ctx.ConversationHistory, ctx.ExtractedContext, ctx.ConfirmedUserPreferences) {
				ev.contactCheck = "failed"
				f.ContactUnclear = true
				return
			}
			if intent.Confidence < 0.7 {
				f.ContactUnclear = true
			}
		}},
		{"intent_check", func(f *ReplyFacts) {
			f.IntentUnclear = intent.Confidence < 0.6
		}},
		{"socratic", func(f *ReplyFacts) {
			ev.deepen, ev.deepenPrinciples = ca.deepeningAllowed(&ctx, userMessage, intent)
			f.DeepenAllowed = ev.deepen
		}},
	}

	plan, by := decideByProbes(ReplyFacts{Greeting: ctx.IsGreeting, ConfirmFact: ctx.DoubtfulFact != "", Bypass: ctx.ResultNow}, probes)
	response.Metadata["replyExit"] = string(plan.Kind)
	response.Metadata["replyExitReason"] = plan.Reason
	log.Printf("[ConversationAgent] Reply exit: %s (decided by %q): %s", plan.Kind, by, plan.Reason)

	finish := func() (bool, bool) {
		response.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())
		return true, ev.deepen
	}

	switch plan.Kind {
	case KindDeny:
		lc := ev.layerCtx.Layer11
		log.Printf("[ConversationAgent] 🚫 Layer 11 denial protocol: generating denial response")
		msg := lc.DenialMessage
		if msg == "" {
			msg = "I'm unable to help with that request. " + lc.Reason
		}
		if lc.AltSuggestion != "" {
			msg += "\n\nInstead, I'd suggest: " + lc.AltSuggestion
		}
		if len(lc.Resources) > 0 {
			msg += "\n\nHere are some resources that might help:"
			for _, r := range lc.Resources {
				msg += "\n- " + r
			}
		}
		response.Response = msg
		response.Phase = "denial"
		response.Metadata["denialApplied"] = true
		return finish()

	case KindSafetyAlert:
		alert := ctx.PrecomputedSafetyVerdict
		log.Printf("[ConversationAgent] Safety alert detected: %s", alert.AlertType)
		response.Phase = "safety_alert"
		response.SafetyAlert = alert
		response.Response = alert.Message
		return finish()

	case KindGreeting:
		response.Response = ca.greetingReply(userMessage)
		response.Metadata["greeting"] = true
		return finish()

	case KindConfirm:
		response.Phase = "clarification"
		response.Response = ca.confirmQuestion(userMessage, ctx.DoubtfulFact)
		response.Metadata["confirmFact"] = true
		return finish()

	case KindContact:
		response.Phase = "clarification"
		response.Response = ca.singleQuestion(userMessage, ca.generateContextualClarification(userMessage, ctx.ExtractedContext))
		response.Metadata["clarificationNeeded"] = "true"
		if ev.contactCheck != "" {
			response.Metadata["layer4Check"] = ev.contactCheck
		}
		return finish()

	case KindIntentCheck:
		if ca.responseGenerator == nil {
			return false, ev.deepen
		}
		text := ca.responseGenerator.GenerateIntentClarificationResponse(ctx, userMessage)
		if text == "" {
			return false, ev.deepen
		}
		text = ca.singleQuestion(userMessage, text)
		response.Response = text
		response.Metadata["intentCheck"] = true
		ca.saveQuestion(ctx, "intent_q_", "goal", text, "", 1, "intent question", false)
		return finish()

	case KindClarity:
		c := ev.clarif
		c.Question = ca.singleQuestion(userMessage, c.Question)
		response.Response = c.Question
		response.Metadata["clarityGate"] = c.Type
		response.Metadata["clarificationNeeded"] = c.Description
		response.Metadata["priority"] = c.Priority
		ca.saveQuestion(ctx, "t1_clarif_q_", c.Type, c.Question, c.Description, c.Priority, "Tier 1 clarification", true)
		log.Printf("[ConversationAgent] [✓] LLM-driven clarification: %s (priority=%d)", c.Type, c.Priority)
		return finish()

	case KindGapQuestion:
		top := []string{ctx.Gaps[0]}
		text := ca.responseGenerator.GenerateGapClarificationResponse(ctx, top)
		if text == "" {
			return false, ev.deepen
		}
		response.Response = ca.singleQuestion(userMessage, text)
		response.Metadata["gapGate"] = true
		response.Metadata["gapCount"] = len(ctx.Gaps)
		response.Metadata["gapsPrioritized"] = 1
		response.Metadata["gaps"] = ctx.Gaps
		response.Metadata["gate"] = "gap_prioritization"
		response.Metadata["maturity"] = ctx.ContextMaturity
		priority := 2
		if ev.layerCtx != nil && ev.layerCtx.Layer4 != nil {
			for _, g := range ev.layerCtx.Layer4.DetectedGaps {
				if g.Description == ctx.Gaps[0] {
					priority = priorityOfSeverity(g.Severity) // kept, so a skipped question comes back at the same weight
					break
				}
			}
		}
		ca.saveQuestion(ctx, "gap_clarif_q_", "gap", ctx.Gaps[0],
			fmt.Sprintf("Gap-based clarification: %d context gaps identified (maturity=%.2f)", len(ctx.Gaps), ctx.ContextMaturity), priority, "gap-based clarification", false)
		return finish()

	case KindPrinciple:
		phase := phaseLabel(ctx)
		ev.principleQ = ca.singleQuestion(userMessage, ev.principleQ)
		response.Response = ev.principleQ
		response.Metadata["principleGate"] = ev.principle
		response.Metadata["layer"] = "6-7"
		response.Metadata["concernType"] = "principle_clarification"
		response.Metadata["phase"] = phase
		ca.saveQuestion(ctx, "layer67_clarif_q_", "goal", ev.principleQ,
			fmt.Sprintf("Principle: %s - Message may involve this principle (maturity=%.2f)", ev.principle, ctx.ContextMaturity), 1, "Layer 6-7 clarification", false)
		return finish()

	case KindPersistent:
		phase := phaseLabel(ctx)
		principleID := "unknown"
		if v, ok := safeGetMetadataString(response.Metadata, "principleGate", "Layer 10 detection"); ok {
			principleID = v
		}
		response.Response = ca.singleQuestion(userMessage, ca.generatePersistentQuestion(userMessage, principleID, ev.lastClar))
		response.Metadata["persistentGate"] = principleID
		response.Metadata["layer"] = "10"
		response.Metadata["attemptNumber"] = 2
		response.Metadata["phase"] = phase
		return finish()

	case KindTopicShift:
		phase := phaseLabel(ctx)
		sh := ev.shift
		response.Response = fmt.Sprintf("I notice we shifted from %s to %s. Are these connected, or is this a new focus?", sh.From, sh.To)
		response.Metadata["topicShift"] = sh
		response.Metadata["layer"] = "9"
		response.Metadata["shiftFrom"] = sh.From
		response.Metadata["shiftTo"] = sh.To
		response.Metadata["shiftConfidence"] = sh.Confidence
		response.Metadata["phase"] = phase
		log.Printf("[ConversationAgent] [✓] Layer 9: Detected topic shift: %s → %s (confidence=%.2f, maturity=%.2f)", sh.From, sh.To, sh.Confidence, ctx.ContextMaturity)
		return finish()

	case KindSocratic:
		if ev.layerCtx != nil && ev.layerCtx.Layer8 != nil && len(ev.layerCtx.Layer8.SocraticQuestions) > 0 && ca.responseGenerator != nil {
			l8 := ev.layerCtx.Layer8
			log.Printf("[ConversationAgent] 📚 Layer 8: Using Socratic questions (strategy=%s, depth=%s)", l8.QuestionStrategy, l8.Depth)
			text := l8.SocraticQuestions[0]
			if len(ctx.ConversationHistory) <= 2 {
				text = "Hi! I'd love to help you think through this.\n\n" + text
			}
			response.Response = ca.singleQuestion(userMessage, text)
			response.Metadata["socraticQuestionUsed"] = true
			response.Metadata["socraticStrategy"] = l8.QuestionStrategy
			response.Metadata["socraticDepth"] = l8.Depth
			return finish()
		}
		if ca.renderPrincipleSocratic(ctx, userMessage, ev.deepenPrinciples, response) {
			return finish()
		}
		log.Printf("[ConversationAgent] [Layer 8] Socratic deepening gates did not trigger or question generation failed")
	}
	return false, ev.deepen
}

// saveQuestion stores a question the user is shown, so it can be matched and marked answered later.
func (ca *conversationAgent) saveQuestion(ctx models.Context, idPrefix, qType, text, notes string, priority int, label string, validateType bool) {
	if ctx.ConversationID == "" || ctx.AboutMe == nil || ctx.AboutMe.UserID == "" || ca.db == nil {
		return
	}
	repo := ca.db.GetClarificationQuestionRepository()
	if repo == nil {
		return
	}
	// A question the user skipped earlier and Moly asks again becomes active again: it is the same question, not a new row.
	if again, err := database.NewClarificationHistoryRepository(ca.db).ReactivateSkippedByText(ctx.AboutMe.UserID, ctx.ConversationID, text); err != nil {
		log.Printf("[ConversationAgent] Warning: could not reactivate a skipped question: %v", err)
	} else if again {
		return
	}
	if validateType {
		valid := map[string]bool{"gap": true, "goal": true, "contact": true, "context": true, "safety": true}
		if !valid[qType] {
			log.Printf("[ConversationAgent] Warning: Invalid clarificationType %q, defaulting to 'context'", qType)
			qType = "context"
		}
	}
	q := &database.ClarificationQuestion{
		ID:                fmt.Sprintf("%s%d", idPrefix, time.Now().UnixNano()),
		UserID:            ctx.AboutMe.UserID,
		ConversationID:    ctx.ConversationID,
		ClarificationType: qType,
		QuestionText:      text,
		ContextNotes:      notes,
		Priority:          priority,
		Status:            "active",
		CreatedAt:         time.Now().Unix(),
	}
	if err := repo.SaveQuestion(q); err != nil {
		log.Printf("[ConversationAgent] Warning: Failed to save %s: %v", label, err)
		return
	}
	log.Printf("[ConversationAgent] [✓] %s saved: %s", label, q.ID)
}

const socraticQuestionPrefix = "layer8_socratic_q_"

// deepeningAllowed is the one place the deepening gates are decided. A Socratic question is allowed only when
// every gate passes; any gate that fails keeps the reply on clarification or help.
func (ca *conversationAgent) deepeningAllowed(ctx *models.Context, userMessage string, intent IntentAnalysis) (bool, []string) {
	if ca.socraticSelector == nil {
		return false, nil
	}
	// Maturity is the gap maturity (ctx.ContextMaturity): reflecting needs at least one answered question, so the user has
	// started to work with Moly. An open gap never reaches here (it is asked first), so no separate gap gate is needed.
	if !WayOutOpen(ctx.ContextMaturity) {
		log.Printf("[ConversationAgent] Gate: gap maturity %.2f, nothing answered yet, preventing Socratic deepening", ctx.ContextMaturity)
		return false, nil
	}
	if ctx.IsFirstMessageInConversation {
		log.Printf("[ConversationAgent] Gate 1: First message in conversation, preventing deepening")
		return false, nil
	}
	dr := NewSocraticDeepeningReasoner(ca.socraticSelector)
	if !dr.ShouldDeepen(ctx, userMessage, ca.socraticAsked(ctx)) {
		return false, nil
	}
	for _, incident := range ctx.RecentSafetyIncidents {
		if incident.Severity == "high" || incident.Severity == "critical" {
			log.Printf("[ConversationAgent] Gate 4: Recent %s severity incident, preventing deepening", incident.Severity)
			return false, nil
		}
	}
	if intent.Confidence < 0.5 {
		log.Printf("[ConversationAgent] Gate 6: Low intent confidence (%.2f < 0.5), preventing deepening", intent.Confidence)
		return false, nil
	}
	if ctx.UserBehaviorProfile != nil && ctx.UserBehaviorProfile.Confidence > 0.7 {
		if prefers, ok := ctx.UserBehaviorProfile.SuggestionChoices["prefers_advice"].(bool); ok && prefers {
			if asks, ok := ctx.UserBehaviorProfile.SuggestionChoices["prefers_questions"].(bool); !ok || !asks {
				log.Printf("[ConversationAgent] Gate 7: User prefers advice, preventing Socratic deepening")
				return false, nil
			}
		}
	}
	if ctx.ContextMaturity >= 0.8 {
		log.Printf("[ConversationAgent] Gate 8: maturity=%.2f >= 0.8, skip Socratic and provide help", ctx.ContextMaturity)
		return false, nil
	}
	if ca.llmClient == nil || ca.constitution == nil {
		return false, nil
	}
	// Last, because it is the one model call: does the situation call for a reflective question, and which principles does it touch?
	goal := ctx.PastIntention
	contact := ""
	if ctx.ExtractedContext != nil {
		if ctx.ExtractedContext.Intention != "" {
			goal = ctx.ExtractedContext.Intention
		}
		if ctx.ExtractedContext.Contact != nil {
			contact = ctx.ExtractedContext.Contact.Name
		}
	}
	j := JudgeDeepening(context.Background(), ca.llmClient, userMessage, goal, contact, ctx.ConversationHistory, ca.constitution.SupremePrinciples)
	return j.Worth, j.Principles
}

// socraticAsked returns one placeholder per Socratic question already asked in this conversation, so the reasoner's
// limit on questions works. A store that cannot be read counts none.
func (ca *conversationAgent) socraticAsked(ctx *models.Context) []models.SocraticQuestion {
	asked := []models.SocraticQuestion{}
	if ca.db == nil || ctx.ConversationID == "" {
		return asked
	}
	repo := ca.db.GetClarificationQuestionRepository()
	if repo == nil {
		return asked
	}
	questions, err := repo.GetConversationQuestions(ctx.ConversationID)
	if err != nil {
		return asked
	}
	for _, q := range questions {
		if strings.HasPrefix(q.ID, socraticQuestionPrefix) {
			asked = append(asked, models.SocraticQuestion{})
		}
	}
	return asked
}

// renderPrincipleSocratic writes a principle-based Socratic question. It returns false when none could be written.
func (ca *conversationAgent) renderPrincipleSocratic(ctx models.Context, userMessage string, principles []string, response *models.ConversationResponse) bool {
	if len(principles) == 0 {
		log.Printf("[ConversationAgent] [Layer 8] No principles identified, continuing without Socratic deepening")
		return false
	}
	question := ca.generateSocraticQuestionWithPrinciples(userMessage, &ctx, principles)
	if question == "" {
		return false
	}
	question = ca.singleQuestion(userMessage, question)
	response.Response = question
	response.Metadata["orchestrator_gate"] = "layer_8_socratic_deepening"
	response.Metadata["principles"] = principles
	response.Metadata["shouldDeepen"] = true
	response.Metadata["maturity"] = ctx.ContextMaturity
	ca.saveQuestion(ctx, socraticQuestionPrefix, "goal", question, fmt.Sprintf("Layer 8: Principles=%v", principles), 2, "Layer 8 Socratic question", false)
	return true
}
