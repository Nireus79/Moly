package agents

import (
	"context"
	"log"
	"moly/privacy"
	"strings"
	"time"

	"moly/models"
	"moly/tools"
)

// exitInputs are facts the agent already derived from the extraction before the decision.
type exitInputs struct {
	intent IntentAnalysis
}

// exitEvidence is what the probes found out. A render step reads only its own part.
type exitEvidence struct {
	layerCtx         *tools.LayerContext
	deepen           bool
	deepenPrinciples []string // the principles the deepening judgement found
	principle        string
	principleQ       string
	persistent       string
	lastClar         string
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
		{"memory_question", func(f *ReplyFacts) {
			// What is missing comes from the memory update after the question discipline (docs/MEMORY_REDESIGN.md, section 12).
			n := ctx.MemoryNext
			if n == nil || n.Action != "ask" || n.Kind == "name" || ca.budgetSpent(&ctx) {
				return
			}
			missing := strings.TrimSpace(n.Missing)
			if missing == "" {
				missing = strings.TrimSpace(n.Topic)
			}
			if missing == "" {
				return
			}
			ctx.Gaps = []string{missing}
			f.OpenGap = true
		}},
		{"principle", func(f *ReplyFacts) {
			if ctx.ExtractedContext == nil || ca.budgetSpent(&ctx) {
				return
			}
			has, id, q := ca.detectPrincipleConcerns(userMessage, dialogueBlock(analysisCtx, ctx), ctx.ExtractedContext)
			if !has || id == "" || q == "" {
				return
			}
			if ca.principleAskedBefore(q, ctx.ConversationHistory, analysisCtx) {
				log.Printf("[ConversationAgent] The principle question was asked before: not asked again")
				return
			}
			ev.principle, ev.principleQ = id, q
			f.PrincipleOpen = true
		}},
		{"persistent", func(f *ReplyFacts) {
			if len(ctx.ConversationHistory) < 2 || ca.budgetSpent(&ctx) {
				return
			}
			repeated, last := ca.detectRepeatedConcern(userMessage, ctx.ConversationHistory, ctx.ExtractedContext)
			if !repeated {
				return
			}
			ev.lastClar = last
			f.Persistent = true
		}},
		{"intent_check", func(f *ReplyFacts) {
			f.IntentUnclear = intent.Confidence < 0.6 && !ca.budgetSpent(&ctx)
		}},
		{"socratic", func(f *ReplyFacts) {
			if ca.budgetSpent(&ctx) {
				return
			}
			ev.deepen, ev.deepenPrinciples = ca.deepeningAllowed(&ctx, userMessage, dialogueBlock(analysisCtx, ctx), intent)
			f.DeepenAllowed = ev.deepen
		}},
	}

	ctx.Gaps = nil // the gaps of the old pipeline are gone: only the memory's question can fill them

	plan, by := decideByProbes(ReplyFacts{Greeting: ctx.IsGreeting, AboutMoly: ctx.AboutMoly, ConfirmFact: ctx.DoubtfulFact != "", Revision: intent.AdjustsDraft, Bypass: ctx.ResultNow}, probes)
	response.Metadata["replyExit"] = string(plan.Kind)
	response.Metadata["replyExitReason"] = plan.Reason
	log.Printf("[ConversationAgent] Reply exit: %s (decided by %q): %s", plan.Kind, by, plan.Reason)

	finish := func() (bool, bool) {
		switch plan.Kind {
		case KindConfirm, KindClarity, KindIntentCheck, KindGapQuestion, KindPrinciple, KindPersistent, KindSocratic:
			// written by the model: it may take for granted what the conversation does not state
			response.Response = ca.groundedReply(userMessage, response.Response, dialogueBlock(analysisCtx, ctx))
		}
		privacy.Register(response.Response)
		response.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())
		return true, ev.deepen
	}

	// Every question the reply is about to ask passes finalQuestion: it becomes one short question, and a question that repeats an earlier one
	// in other words is not asked (the reply moves on to the help text). A skipped question that comes back on purpose may be asked once more.
	finalQuestion := func(text string, comesBack bool) (string, bool) {
		text = ca.singleQuestion(userMessage, text)
		limit := 1
		if comesBack {
			limit = 2 // asked once and ignored: it may be asked once more, not a third time
		}
		if ca.questionAskedTimes(text, ctx.ConversationHistory, analysisCtx, limit) {
			log.Printf("[ConversationAgent] The %s question was asked before: not asked again", plan.Kind)
			response.Response = ""
			response.Metadata["replyExit"] = string(KindHelp)
			return "", false
		}
		return text, true
	}

	switch plan.Kind {
	case KindDeny:
		lc := ev.layerCtx.Layer11
		log.Printf("[ConversationAgent] 🚫 Layer 11 denial protocol: generating denial response")
		msg := lc.DenialMessage
		if msg == "" {
			msg = "I'm unable to help with that request. " + lc.Reason
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
		if q := ca.questionBeforeRefusal(ctx, userMessage, alert); q != "" {
			response.Phase = "clarification"
			response.Response = q
			response.Metadata["refusalQuestion"] = true
			response.Metadata["replyExit"] = "refusal_question"
			return finish()
		}
		response.Phase = "safety_alert"
		localized := *alert // the alert the user sees carries the refusal in the language of their words
		localized.Message = ca.refusalInUsersLanguage(userMessage, alert.Message)
		response.SafetyAlert = &localized
		response.Response = localized.Message
		return finish()

	case KindGreeting:
		response.Response = ca.greetingReply(userMessage)
		response.Metadata["greeting"] = true
		return finish()

	case KindConfirm:
		response.Phase = "clarification"
		response.Response = ca.confirmQuestion(userMessage, ctx.DoubtfulFact, dialogueBlock(analysisCtx, ctx))
		response.Metadata["confirmFact"] = true
		return finish()

	case KindIntentCheck:
		if ca.responseGenerator == nil {
			return false, ev.deepen
		}
		text := ca.responseGenerator.GenerateIntentClarificationResponse(ctx, userMessage, dialogueBlock(analysisCtx, ctx))
		if text == "" {
			return false, ev.deepen
		}
		text, ok := finalQuestion(text, false)
		if !ok {
			return false, ev.deepen
		}
		response.Response = text
		response.Metadata["intentCheck"] = true
		return finish()

	case KindGapQuestion:
		top := []string{ctx.Gaps[0]}
		text := ca.responseGenerator.GenerateGapClarificationResponse(ctx, top, dialogueBlock(analysisCtx, ctx), userMessage)
		if text == "" {
			return false, ev.deepen
		}
		// a question the user skipped comes back on purpose (the question discipline marked it)
		q, ok := finalQuestion(text, ctx.MemoryNext != nil && ctx.MemoryNext.Returning)
		if !ok {
			return false, ev.deepen
		}
		response.Response = q
		response.Metadata["gapGate"] = true
		response.Metadata["gapCount"] = len(ctx.Gaps)
		response.Metadata["gapsPrioritized"] = 1
		response.Metadata["gaps"] = ctx.Gaps
		response.Metadata["gate"] = "gap_prioritization"
		response.Metadata["maturity"] = ctx.ContextMaturity
		return finish()

	case KindPrinciple:
		q, ok := finalQuestion(ev.principleQ, false)
		if !ok {
			return false, ev.deepen
		}
		ev.principleQ = q
		response.Response = ev.principleQ
		response.Metadata["principleGate"] = ev.principle
		response.Metadata["layer"] = "6-7"
		response.Metadata["concernType"] = "principle_clarification"
		return finish()

	case KindPersistent:
		principleID := "unknown"
		if v, ok := safeGetMetadataString(response.Metadata, "principleGate", "Layer 10 detection"); ok {
			principleID = v
		}
		q, ok := finalQuestion(ca.generatePersistentQuestion(userMessage, principleID, ev.lastClar), false)
		if !ok {
			return false, ev.deepen
		}
		response.Response = q
		response.Metadata["persistentGate"] = principleID
		response.Metadata["layer"] = "10"
		response.Metadata["attemptNumber"] = 2
		return finish()

	case KindSocratic:
		if ev.layerCtx != nil && ev.layerCtx.Layer8 != nil && len(ev.layerCtx.Layer8.SocraticAims) > 0 {
			l8 := ev.layerCtx.Layer8
			log.Printf("[ConversationAgent] 📚 Layer 8: writing a Socratic question from its aim (strategy=%s, depth=%s)", l8.QuestionStrategy, l8.Depth)
			// The model writes the question from the aim, in the language of the user's words. If it cannot, the reply moves on
			// (principle question, help): there is no fixed sentence to fall back to.
			if text := ca.writeQuestionFromAim(userMessage, dialogueBlock(analysisCtx, ctx), l8.SocraticAims[0]); text != "" {
				q, ok := finalQuestion(text, false)
				if !ok {
					return false, ev.deepen
				}
				response.Response = q
				response.Metadata["socraticQuestionUsed"] = true
				response.Metadata["socraticStrategy"] = l8.QuestionStrategy
				response.Metadata["socraticDepth"] = l8.Depth
				return finish()
			}
		}
		if ca.renderPrincipleSocratic(ctx, userMessage, ev.deepenPrinciples, response) {
			return finish()
		}
		log.Printf("[ConversationAgent] [Layer 8] Socratic deepening gates did not trigger or question generation failed")
	}
	return false, ev.deepen
}

// deepeningAllowed is the one place the deepening gates are decided. A Socratic question is allowed only when
// every gate passes; any gate that fails keeps the reply on clarification or help.
func (ca *conversationAgent) deepeningAllowed(ctx *models.Context, userMessage, block string, intent IntentAnalysis) (bool, []string) {
	if ca.socraticSelector == nil {
		return false, nil
	}
	// An open gap never reaches here (it is asked first). Reflecting needs no answered question: a plain request with something
	// worth thinking through gets a reflective question as well (user decision, 2026-10-19).
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
	if ca.lastReplyWasQuestion(ctx) {
		log.Printf("[ConversationAgent] Moly's last reply was a question: no reflective question in a row")
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
	j := JudgeDeepening(context.Background(), ca.llmClient, userMessage, goal, contact, block, ca.constitution.SupremePrinciples)
	return j.Worth, j.Principles
}

// lastReplyWasQuestion reports whether Moly's latest reply in the conversation was one of the optional questions (the history is
// [current, oldest ... newest]), so two questions never come in a row.
func (ca *conversationAgent) lastReplyWasQuestion(ctx *models.Context) bool {
	for i := len(ctx.ConversationHistory) - 1; i >= 0; i-- {
		m := ctx.ConversationHistory[i]
		if m.Role != "assistant" {
			continue
		}
		kind, _ := m.Metadata["replyExit"].(string)
		return strings.HasSuffix(kind, "_question")
	}
	return false
}

// socraticAsked returns one placeholder per Socratic question Moly already asked in this conversation (its replies carry the exit kind), so the
// reasoner's limit on questions works.
func (ca *conversationAgent) socraticAsked(ctx *models.Context) []models.SocraticQuestion {
	asked := []models.SocraticQuestion{}
	for _, m := range ctx.ConversationHistory {
		if m.Role != "assistant" || m.Metadata == nil {
			continue
		}
		if kind, _ := m.Metadata["replyExit"].(string); ReplyKind(kind) == KindSocratic {
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
	return true
}

// dialogueBlock is what the model is told about the conversation: the packet when the analysis context has one, otherwise the
// last messages of the agent's history.
func dialogueBlock(analysisCtx *models.AnalysisContext, ctx models.Context) string {
	if analysisCtx != nil {
		if block := analysisCtx.Dialogue.Text(); block != "" {
			return block
		}
	}
	return models.WithExchange(models.ExchangeBeforeCurrentFirst(ctx.ConversationHistory, 1))
}
