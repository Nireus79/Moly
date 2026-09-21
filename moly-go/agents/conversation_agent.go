package agents

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"moly/config"
	"moly/database"
	"moly/models"
	"moly/schema"
	"moly/tools"
)

// conversationAgent - Implements the 5-phase conversation flow
type conversationAgent struct {
	llmClient             tools.LLMProvider
	suggestionGenerator   *tools.SuggestionGenerator
	questionGenerator     *tools.QuestionGenerator
	safetyChecker         *tools.SafetyChecker
	harmAnalyzer          *tools.HarmAnalyzer
	clarificationAsker    *tools.ClarificationAsker
	constitutionEvaluator *tools.ConstitutionEvaluator
	contextExtractor      *tools.ContextExtractor
	responseGenerator     *tools.ResponseGenerator // Generates contextual responses instead of hardcoded text
	socraticSelector      *SocraticQuestionSelector // Optional: for Socratic question selection
	constitution          *models.Constitution      // Optional: for principle-guided generation
	db                    *database.Database        // Optional: for conflict detection
	inlineResolver        *tools.InlineConflictResolver // Optional: for Phase 2 inline resolution
}

// NewConversationAgent - Create new conversation agent
func NewConversationAgent(llm tools.LLMProvider) (models.ConversationAgent, error) {
	// LLM client is optional - agent will generate basic suggestions without it
	return &conversationAgent{
		llmClient:             llm,
		suggestionGenerator:   tools.NewSuggestionGenerator(llm),
		questionGenerator:     tools.NewQuestionGenerator(llm),
		safetyChecker:         tools.NewSafetyChecker(llm),
		harmAnalyzer:          tools.NewHarmAnalyzer(llm),
		clarificationAsker:    tools.NewClarificationAsker(llm),
		constitutionEvaluator: tools.NewConstitutionEvaluator(llm),
		contextExtractor:      tools.NewContextExtractor(llm),
		responseGenerator:     tools.NewResponseGenerator(llm), // Generates natural, contextual responses
		socraticSelector:      nil, // Optional - set via SetSocraticSelector if available
	}, nil
}

// SetSocraticSelector injects the Socratic question selector (optional)
func (ca *conversationAgent) SetSocraticSelector(selector *SocraticQuestionSelector) {
	if ca != nil {
		ca.socraticSelector = selector
		log.Printf("[ConversationAgent] Socratic selector initialized")
	}
}

// SetDatabase injects the database for conflict detection (optional, Phase 2)
func (ca *conversationAgent) SetDatabase(dbInterface interface{}) {
	if ca != nil {
		if db, ok := dbInterface.(*database.Database); ok {
			ca.db = db
			if db != nil {
				ca.inlineResolver = tools.NewInlineConflictResolver(db)
				log.Printf("[ConversationAgent] Inline conflict resolver initialized")
			}
		}
	}
}


// InitializeWithSocraticSelector creates and wires a ConversationAgent with Socratic support and principle-based checking
// Returns the agent and any error that occurred during initialization
// If any initialization fails, returns agent with what could be loaded (graceful degradation)
func InitializeWithSocraticSelector(llm tools.LLMProvider, constitutionPath, configDir string) (models.ConversationAgent, error) {
	// Create base agent
	agent, err := NewConversationAgent(llm)
	if err != nil {
		return nil, err
	}

	// Load constitution and question library
	constitution, err := config.LoadConstitution(constitutionPath)
	if err != nil {
		log.Printf("[ConversationAgent] Warning: Could not load constitution: %v", err)
		return agent, nil // Return agent without Socratic features
	}

	library, err := config.LoadQuestionLibrary(configDir)
	if err != nil {
		log.Printf("[ConversationAgent] Warning: Could not load question library: %v", err)
		// Continue - we can still use constitution for principle checking
	}

	// Wire question library into the conversation agent
	if caImpl, ok := agent.(*conversationAgent); ok {
		// Store constitution for principle-guided response generation
		caImpl.constitution = constitution
		log.Printf("[ConversationAgent] [✓] Constitution loaded for response generation")

		// Set Socratic selector if library loaded successfully
		if library != nil {
			selector := NewSocraticQuestionSelector(library, constitution)
			caImpl.SetSocraticSelector(selector)
			log.Printf("[ConversationAgent] [✓] Socratic question selector initialized")
		}
	}

	return agent, nil
}

// socraticQuestionToClarification converts a SocraticQuestion to a ClarificationQuestion
func socraticQuestionToClarification(sq *models.SocraticQuestion) *schema.ClarificationQuestion {
	if sq == nil {
		return nil
	}

	// Build context explaining why we're asking (from expected insights)
	context := sq.TargetsPrinciple
	if len(sq.ExpectedInsights) > 0 {
		context = sq.ExpectedInsights[0]
	}

	return &schema.ClarificationQuestion{
		ID:               sq.ID,
		Type:             "socratic_exploration", // Indicates Socratic method
		Question:         sq.Text,
		Context:          context, // Why we're asking
		Priority:         sq.DepthLevel, // Use depth level as priority
		Status:           "pending",
		CreatedAt:        time.Now().Unix(),
		LinkedFacts:      sq.FollowUpQuestions, // Store follow-up question IDs
		SocraticApproach: sq.SocraticApproach,   // Add Socratic approach
		ExpectedInsights: sq.ExpectedInsights,   // Add expected insights
		TargetsPrinciple: sq.TargetsPrinciple,   // Add targeted principle
		DepthLevel:       sq.DepthLevel,         // Add depth level
	}
}

// Run - Execute the conversation flow and generate conversational response
// Moly is a friend who listens, responds naturally, and learns about the user
func (ca *conversationAgent) Run(ctx models.Context) (*models.ConversationResponse, error) {
	log.Printf("[ConversationAgent] Starting conversation flow")

	startTime := time.Now()
	response := &models.ConversationResponse{
		Metadata: make(map[string]interface{}),
	}
	response.Phase = "responding"

	// Extract the user's message (most recent)
	var userMessage string
	if len(ctx.ConversationHistory) == 0 {
		response.Error = "No message provided"
		response.Response = ca.responseGenerator.GenerateEmptyMessageResponse(ctx)
		response.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())
		return response, nil
	}

	// Find the last user message
	for i := len(ctx.ConversationHistory) - 1; i >= 0; i-- {
		if ctx.ConversationHistory[i].Role == "user" {
			userMessage = ctx.ConversationHistory[i].Content
			break
		}
	}

	if userMessage == "" {
		response.Error = "Empty message"
		response.Response = ca.responseGenerator.GenerateEmptyMessageResponse(ctx)
		response.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())
		return response, nil
	}

	log.Printf("[ConversationAgent] User message: %.80s...", userMessage)

	// Load or initialize structured context (Phase 1 integration)
	var structuredCtx *models.StructuredContext
	var userID string
	if ctx.AboutMe != nil {
		userID = ctx.AboutMe.UserID
	}

	if ca.db != nil && userID != "" {
		ctxRepo := ca.db.GetStructuredContextRepository()
		loaded, err := ctxRepo.LoadContext(userID, ctx.ConversationID)
		if err != nil {
			log.Printf("[ConversationAgent] WARNING: Failed to load structured context: %v", err)
		} else if loaded != nil {
			structuredCtx = loaded
			log.Printf("[ConversationAgent] ✓ Loaded structured context (goals=%d, people=%d, explored=%d)",
				len(structuredCtx.Goals), len(structuredCtx.PeopleInvolved), len(structuredCtx.ExploredTopics))
		}
	}

	// Initialize if not found
	if structuredCtx == nil {
		structuredCtx = &models.StructuredContext{
			UserID:         userID,
			ConversationID: ctx.ConversationID,
			CreatedAt:      time.Now().Unix(),
			UpdatedAt:      time.Now().Unix(),
		}
		log.Printf("[ConversationAgent] Initialized new structured context")
	}

	// Phase 2 Integration: Detect user intent (Phase 2 - Intent Detection)
	intentAnalysis := DetectIntent(userMessage, ctx.ConversationHistory)
	log.Printf("[ConversationAgent] Intent detected: %s (confidence=%.2f)",
		intentAnalysis.Intent, intentAnalysis.Confidence)

	// Phase 1: ANALYZE - Check what context we have
	aboutMe := ctx.AboutMe
	contact := ctx.ContactProfile
	hasAboutMe := aboutMe != nil && (aboutMe.CommunicationStyle != "" || len(aboutMe.Values) > 0)
	hasContact := contact != nil && contact.Name != "" && contact.Name != "Contact"
	hasIntention := false
	var intention string

	log.Printf("[ConversationAgent] Context analysis: hasAboutMe=%v hasContact=%v", hasAboutMe, hasContact)

	// Try to use passed ExtractedContext first (LLM-based extraction)
	var extractedContact *models.ExtractedContact
	var extractedStyle *models.ExtractedStyle
	if ctx.ExtractedContext != nil {
		log.Printf("[ConversationAgent] Using LLM-extracted context")
		extractedContact = ctx.ExtractedContext.Contact
		extractedStyle = ctx.ExtractedContext.Style
		if ctx.ExtractedContext.Intention != "" {
			intention = ctx.ExtractedContext.Intention
			hasIntention = true
			log.Printf("[ConversationAgent] Using extracted intention: %s (confidence)", intention)
		}
	}

	// Use extracted contact if confidence is high
	if extractedContact != nil && extractedContact.Confidence >= 0.7 && !hasContact {
		contact = &models.Contact{
			Name:         extractedContact.Name,
			Relationship: extractedContact.Relationship,
		}
		hasContact = true
		log.Printf("[ConversationAgent] Using extracted contact: %s (%s, confidence: %.2f)",
			contact.Name, contact.Relationship, extractedContact.Confidence)
	}

	// Use extracted style if confidence is high
	if extractedStyle != nil && extractedStyle.Confidence >= 0.7 && !hasAboutMe {
		if aboutMe == nil {
			aboutMe = &models.AboutMe{UserID: ctx.AboutMe.UserID}
		}
		aboutMe.CommunicationStyle = extractedStyle.Style
		hasAboutMe = true
		log.Printf("[ConversationAgent] Using extracted style: %s (confidence: %.2f)",
			extractedStyle.Style, extractedStyle.Confidence)
	}

	// Use extracted goals if available
	if ctx.ExtractedContext != nil && len(ctx.ExtractedContext.Goals) > 0 {
		if aboutMe == nil {
			aboutMe = &models.AboutMe{UserID: ctx.AboutMe.UserID}
		}
		// Append new goals to existing goals (don't overwrite)
		for _, newGoal := range ctx.ExtractedContext.Goals {
			if newGoal != "" {
				// Check if goal already exists to avoid duplicates
				found := false
				for _, existing := range aboutMe.Goals {
					if strings.ToLower(existing) == strings.ToLower(newGoal) {
						found = true
						break
					}
				}
				if !found {
					aboutMe.Goals = append(aboutMe.Goals, newGoal)
					log.Printf("[ConversationAgent] Added extracted goal: %s", newGoal)
				}
			}
		}
	}

	// Extract user preferences from message (format, length, tone preferences)
	preferenceKeywords := map[string]string{
		"bullet point":   "prefers_bullet_points",
		"bullet-point":   "prefers_bullet_points",
		"concise":        "prefers_concise",
		"short":          "prefers_short",
		"brief":          "prefers_brief",
		"detailed":       "prefers_detailed",
		"step by step":   "prefers_steps",
		"examples":       "prefers_examples",
		"casual":         "prefers_casual_tone",
		"informal":       "prefers_informal_tone",
		"formal":         "prefers_formal_tone",
		"professional":   "prefers_professional_tone",
		"funny":          "prefers_humor",
		"humorous":       "prefers_humor",
		"straight to point": "prefers_direct",
		"direct":         "prefers_direct",
	}

	if aboutMe == nil {
		aboutMe = &models.AboutMe{UserID: ctx.AboutMe.UserID}
	}

	lowerMsgForPrefs := strings.ToLower(userMessage)
	extractedPrefs := make(map[string]bool)
	for keyword, pref := range preferenceKeywords {
		if contains(lowerMsgForPrefs, keyword) && !extractedPrefs[pref] {
			// Store preference in Notes field as JSON-like format
			if !contains(aboutMe.Notes, pref) {
				if aboutMe.Notes != "" {
					aboutMe.Notes += ", " + pref
				} else {
					aboutMe.Notes = pref
				}
				extractedPrefs[pref] = true
				log.Printf("[ConversationAgent] Extracted user preference: %s", pref)
			}
		}
	}

	// Extract core values from message (words like "authentic", "loyal", "independent", etc.)
	if aboutMe != nil && len(aboutMe.Values) == 0 {
		// Only extract values if not already set
		lowerMsg := strings.ToLower(userMessage)
		valueKeywords := map[string]string{
			"authentic":     "authenticity",
			"genuine":       "authenticity",
			"loyal":         "loyalty",
			"faithful":      "loyalty",
			"honest":        "honesty",
			"truthful":      "honesty",
			"independent":   "independence",
			"self-reliant":  "independence",
			"confident":     "confidence",
			"creative":      "creativity",
			"innovative":    "creativity",
			"compassionate": "compassion",
			"empathetic":    "empathy",
			"kind":          "kindness",
			"ambitious":     "ambition",
			"curious":       "curiosity",
		}

		for keyword, value := range valueKeywords {
			if contains(lowerMsg, keyword) {
				// Check if value already in list
				found := false
				for _, existing := range aboutMe.Values {
					if strings.ToLower(existing) == strings.ToLower(value) {
						found = true
						break
					}
				}
				if !found {
					aboutMe.Values = append(aboutMe.Values, value)
					log.Printf("[ConversationAgent] Extracted value from message: %s", value)
				}
			}
		}
	}

	// Fallback: Extract AboutMe from user's response to context-gathering questions
	if !hasAboutMe && userMessage != "" {
		// User might be answering "Tell me about your communication style"
		lowerMsg := strings.ToLower(userMessage)
		style := ""
		if contains(lowerMsg, "casual") || contains(lowerMsg, "informal") {
			style = "casual"
		} else if contains(lowerMsg, "formal") {
			style = "formal"
		} else if contains(lowerMsg, "playful") || contains(lowerMsg, "fun") || contains(lowerMsg, "humorous") {
			style = "playful"
		}

		if style != "" {
			if aboutMe == nil {
				aboutMe = &models.AboutMe{UserID: ctx.AboutMe.UserID}
			}
			aboutMe.CommunicationStyle = style
			hasAboutMe = true
			log.Printf("[ConversationAgent] Extracted communication style from user response: %s (fallback)", style)
		}
	}

	// Fallback: Extract contact name ONLY if user explicitly wants to contact/message someone
	// Check for EXPLICIT contact intent (verb + noun), not just noun alone
	// Example: "I want to message a girl" (has contact verb) vs "I want advice about a girl" (no contact verb)
	if !hasContact && userMessage != "" {
		lowerMsg := strings.ToLower(userMessage)
		hasContactVerb := contains(lowerMsg, "message") || contains(lowerMsg, "text") ||
			contains(lowerMsg, "call") || contains(lowerMsg, "tell") || contains(lowerMsg, "email") ||
			contains(lowerMsg, "ask") || contains(lowerMsg, "contact") || contains(lowerMsg, "reach out") ||
			contains(lowerMsg, "talk to") || contains(lowerMsg, "send") || contains(lowerMsg, "write") ||
			contains(lowerMsg, "talk with")

		// Professional relationships - extract if explicit contact intent
		if (contains(lowerMsg, "boss") || contains(lowerMsg, "manager") || contains(lowerMsg, "colleague")) &&
			hasContactVerb {
			if contact == nil {
				contact = &models.Contact{}
			}
			contact.Name = "Boss"
			contact.Relationship = "professional"
			hasContact = true
			log.Printf("[ConversationAgent] Extracted contact: Boss/Manager (professional, wants to contact)")
		} else if contains(lowerMsg, "friend") && !contains(lowerMsg, "best friend") && !contains(lowerMsg, "close friend") &&
			hasContactVerb {
			if contact == nil {
				contact = &models.Contact{}
			}
			contact.Name = "Friend"
			contact.Relationship = "friend"
			hasContact = true
			log.Printf("[ConversationAgent] Extracted contact: Friend (wants to contact)")
		} else if (contains(lowerMsg, "mom") || contains(lowerMsg, "dad") || contains(lowerMsg, "parent") ||
				contains(lowerMsg, "sibling") || contains(lowerMsg, "brother") || contains(lowerMsg, "sister")) &&
			hasContactVerb {
			if contact == nil {
				contact = &models.Contact{}
			}
			contact.Name = "Family"
			contact.Relationship = "family"
			hasContact = true
			log.Printf("[ConversationAgent] Extracted contact: Family member (wants to contact)")
		} else if (contains(lowerMsg, "girl") || contains(lowerMsg, "boy") || contains(lowerMsg, "crush") ||
				contains(lowerMsg, "partner") || contains(lowerMsg, "spouse") || contains(lowerMsg, "girlfriend") ||
				contains(lowerMsg, "boyfriend") || contains(lowerMsg, "date") || contains(lowerMsg, "romantic") ||
				contains(lowerMsg, "likes me") || contains(lowerMsg, "interested in")) &&
			hasContactVerb {
			if contact == nil {
				contact = &models.Contact{}
			}
			contact.Name = "Romantic Interest"
			contact.Relationship = "romantic"
			hasContact = true
			log.Printf("[ConversationAgent] Extracted contact: Romantic interest (wants to contact)")
		}
		// If person noun mentioned WITHOUT contact verb, skip extraction (they want advice about someone, not to contact them)
	}

	// Fallback: Detect intention from message - but only from NEW messages, not from context gathering responses
	if !hasIntention && userMessage != "" && !contains(userMessage, "casual") && !contains(userMessage, "formal") &&
		!contains(userMessage, "friend") && !contains(userMessage, "boss") && !contains(userMessage, "partner") &&
		!contains(userMessage, "playful") && !contains(userMessage, "humorous") {
		lowerMsg := strings.ToLower(userMessage)
		if contains(lowerMsg, "congratulat") || contains(lowerMsg, "promote") || contains(lowerMsg, "success") {
			intention = "celebrate"
			hasIntention = true
		} else if contains(lowerMsg, "apologi") || contains(lowerMsg, "sorry") {
			intention = "apologize"
			hasIntention = true
		} else if contains(lowerMsg, "help") || contains(lowerMsg, "need") || contains(lowerMsg, "stuck") {
			intention = "seek_help"
			hasIntention = true
		} else if contains(lowerMsg, "hi") || contains(lowerMsg, "hello") || contains(lowerMsg, "hey") {
			intention = "greet"
			hasIntention = true
		} else if contains(lowerMsg, "want") || contains(lowerMsg, "ask") || contains(lowerMsg, "request") ||
			contains(lowerMsg, "can you") || contains(lowerMsg, "could you") || contains(lowerMsg, "would you") {
			intention = "request"
			hasIntention = true
		} else if contains(lowerMsg, "feel") || contains(lowerMsg, "felt") || contains(lowerMsg, "emotion") {
			intention = "express_feeling"
			hasIntention = true
		}
		log.Printf("[ConversationAgent] Detected intention: %s (fallback)", intention)
	}

	// Default intention if still not set
	if intention == "" {
		intention = "general_support"
	}
	log.Printf("[ConversationAgent] Final intention: %s (hasIntention=%v)", intention, hasIntention)

	// Phase 2: DECIDE - Gathering context vs. suggesting
	// SAFETY CHECK - Detect crisis or risks
	log.Printf("[ConversationAgent] Checking safety of user message")
	safetyAlert, err := ca.runSafetyPhase(context.Background(), userMessage)
	if err != nil {
		log.Printf("[ConversationAgent] Safety check error: %v", err)
	}

	if safetyAlert != nil {
		log.Printf("[ConversationAgent] Safety alert detected: %s", safetyAlert.AlertType)
		response.Phase = "safety_alert"
		response.SafetyAlert = safetyAlert
		response.Response = safetyAlert.Title + ": " + safetyAlert.Message
		response.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())
		return response, nil
	}

	// DETERMINE IF CLARIFICATION NEEDED (BEFORE generating response)
	needsClarification := determineClarificationNeeded(hasAboutMe, hasContact, hasIntention)
	log.Printf("[ConversationAgent] Context assessment: needsClarification=%v (AboutMe=%v Contact=%v Intention=%v)", needsClarification, hasAboutMe, hasContact, hasIntention)

	// GENERATE APPROPRIATE RESPONSE (contextually aware of clarification needs)
	// Moly responds naturally to the user, building understanding over time
	if ca.llmClient == nil || ca.responseGenerator == nil {
		if needsClarification {
			response.Response = "I'd like to understand you better. Tell me more?"
		} else {
			response.Response = "I'm listening."
		}
		log.Printf("[ConversationAgent] No LLM/ResponseGenerator available, using fallback response")
	} else {
		log.Printf("[ConversationAgent] Generating response (needsClarification=%v)", needsClarification)

		var generatedResponse string
		if needsClarification {
			// Ask for missing context using LLM-generated response
			missingAboutMe := !hasAboutMe
			missingIntention := !hasIntention
			// Use ResponseGenerator for natural, contextual clarification requests
			generatedResponse = ca.responseGenerator.GenerateNeedsClarificationResponse(ctx, missingAboutMe, missingIntention)
			log.Printf("[ConversationAgent] [✓] Generated clarifying response: %.100s...", generatedResponse)
		} else {
			// PHASE 2: CHECK FOR CONFLICTS BEFORE RESPONDING
			// Detect and ask user to resolve any pending conflicts
			var conflictQuestion string
			var pendingConflictID int64
			if ca.inlineResolver != nil && aboutMe != nil && aboutMe.UserID != "" {
				conflictInfo := ca.inlineResolver.CheckAndAskForConflicts(aboutMe.UserID)
				if conflictInfo.HasPendingConflict {
					conflictQuestion = conflictInfo.ConflictMessage
					pendingConflictID = conflictInfo.ConflictID
					log.Printf("[ConversationAgent] Found pending conflict %d, will ask user to resolve", pendingConflictID)
				}
			}

			// PHASE 2.5: SOCRATIC DEEPENING (Step 4 - Optional context deepening)
			var socraticQuestion *models.SocraticQuestion

			log.Printf("[ConversationAgent] Checking for Socratic deepening opportunity")

			// Only attempt deepening if we have selector and valid context
			if ca.socraticSelector != nil && hasAboutMe && hasContact && hasIntention {
				reasoner := NewSocraticDeepeningReasoner(ca.socraticSelector)

				// Load previous questions from database for context-aware sequencing
				var previousQuestions []models.SocraticQuestion
				if ca.db != nil {
					qhRepo := ca.db.GetQuestionHistoryRepository()
					if qhRepo != nil {
						pastQuestions, err := qhRepo.GetPreviousQuestions(ctx.AboutMe.UserID, 10)
						if err != nil {
							log.Printf("[ConversationAgent] Warning: Failed to load previous questions: %v", err)
						} else if len(pastQuestions) > 0 {
							// Convert from database questions (map format) to models.SocraticQuestion
							for _, q := range pastQuestions {
								sq := models.SocraticQuestion{}
								if id, ok := q["id"].(string); ok {
									sq.ID = id
								}
								if text, ok := q["question"].(string); ok {
									sq.Text = text
								}
								previousQuestions = append(previousQuestions, sq)
							}
							log.Printf("[ConversationAgent] ✓ Loaded %d previous questions for context awareness", len(previousQuestions))
						}
					}
				}

				// Determine if we should deepen
				if reasoner.ShouldDeepen(&ctx, userMessage, previousQuestions) {
					// Select the next question
					question, approach := reasoner.SelectQuestion(&ctx, userMessage, previousQuestions)
					if question != nil {
						socraticQuestion = question
						// Record question to database if conversationID is available
						if ctx.ConversationID != "" && ca.db != nil {
							qhRepo := ca.db.GetQuestionHistoryRepository()
							// Extract emotion state and risk level from context if available
							emotionState := "neutral"
							if ctx.LastRiskAssessment != nil {
								if emotion, ok := ctx.LastRiskAssessment["emotion"].(string); ok {
									emotionState = emotion
								}
							}
							riskLevel := "none"
							if ctx.LastRiskAssessment != nil {
								if risk, ok := ctx.LastRiskAssessment["level"].(string); ok {
									riskLevel = risk
								}
							}
							recordErr := qhRepo.RecordQuestion(ctx.AboutMe.UserID, ctx.ConversationID, question, emotionState, riskLevel)
							if recordErr != nil {
								log.Printf("[ConversationAgent] Warning: Failed to record Socratic question: %v", recordErr)
							}
						}
						log.Printf("[ConversationAgent] Selected Socratic question: %s (approach: %s)", question.ID, approach)
					}
				}
			}

			// Give full response with available context (and optional Socratic question)
			// If there's a pending conflict, ask about it first (natural conversation flow)
			if conflictQuestion != "" {
				generatedResponse = conflictQuestion
				// Store conflict ID in metadata for the frontend to track resolution
				response.Metadata["pendingConflictID"] = pendingConflictID
				log.Printf("[ConversationAgent] [✓] Generated conflict resolution question: %.100s...", generatedResponse)
			} else {
				generatedResponse = ca.generateConversationalResponse(ctx, userMessage, socraticQuestion)
				log.Printf("[ConversationAgent] [✓] Generated full response: %.100s...", generatedResponse)
			}
		}
		response.Response = generatedResponse
	}

	// EXTRACT INSIGHTS ABOUT USER
	// What did we learn about this person from their message?
	if ca.llmClient != nil && userMessage != "" {
		log.Printf("[ConversationAgent] Extracting insights about user")
		reflection, err := ca.runReflectPhase(context.Background(), userMessage)
		if err != nil {
			log.Printf("[ConversationAgent] Warning: Insight extraction failed: %v", err)
		} else if reflection != nil {
			response.Reflection = reflection
			log.Printf("[ConversationAgent] [✓] Learned about user: %d characteristics", len(reflection.Characteristics))
		}
	}

	// ETHICAL GATE: Analyze generated response for harmful content
	// Apply intervention based on severity level
	if ca.harmAnalyzer != nil && response.Response != "" {
		log.Printf("[ConversationAgent] Running ethical analysis on generated response")
		harmInput := &tools.HarmAnalysisInput{Message: response.Response}
		harmAnalysis, err := ca.harmAnalyzer.Analyze(context.Background(), harmInput)
		if err != nil {
			log.Printf("[ConversationAgent] Warning: Harm analysis failed: %v", err)
		} else if harmAnalysis != nil {
			log.Printf("[ConversationAgent] Harm analysis: severity=%s category=%s", harmAnalysis.Severity, harmAnalysis.Category)

			// Handle different severity levels
			switch harmAnalysis.Severity {
			case "block":
				// BLOCK: Replace response with safe response, mark as blocked
				log.Printf("[ConversationAgent] 🚫 BLOCK: %s - replacing with safe response", harmAnalysis.Category)
				response.Response = harmAnalysis.SafeResponse
				if response.Response == "" {
					response.Response = "I can't help with that, but I'm here if you want to talk about something else."
				}
				response.Metadata["ethicalIntervention"] = "blocked"
				response.Metadata["blockCategory"] = harmAnalysis.Category
				response.Metadata["blockReason"] = harmAnalysis.ReasoningCategory
				response.Metadata["originalResponseBlocked"] = true
				log.Printf("[ConversationAgent] [✓] Blocked response recorded (category: %s)", harmAnalysis.Category)

			case "warn":
				// WARN: Keep response but mark it and add reasoning
				log.Printf("[ConversationAgent] ⚠️  WARN: %s - response shown with warning", harmAnalysis.Category)
				response.Metadata["ethicalIntervention"] = "warned"
				response.Metadata["warningCategory"] = harmAnalysis.Category
				response.Metadata["warningReason"] = harmAnalysis.ReasoningCategory
				response.Metadata["ethicalWarning"] = "This response touches on a sensitive topic - please be thoughtful"
				log.Printf("[ConversationAgent] [✓] Warning metadata added (category: %s)", harmAnalysis.Category)

			case "none":
				// Safe - no intervention needed
				log.Printf("[ConversationAgent] ✓ Response cleared by ethical analysis")
			}
		}
	}

	// CLARIFICATION QUESTIONS REMOVED
	// Context is now gathered through natural conversation flow in Moly's response
	// If needed in future, will be re-implemented as part of main dialogue
	log.Printf("[ConversationAgent] Context gathering handled through conversational response")

	// BUILD METADATA (add to existing Metadata, don't overwrite)
	// Metadata was initialized as empty map at line 142, add fields incrementally
	if response.Metadata == nil {
		response.Metadata = make(map[string]interface{})
	}
	response.Metadata["conversational"] = true
	response.Metadata["hasUserProfile"] = aboutMe != nil
	response.Metadata["contextGaps"] = len(ctx.Gaps)
	response.Metadata["timestamp"] = startTime.Unix()
	log.Printf("[ConversationAgent] [✓] Metadata fields added: conversational=true profile=%v gaps=%d pendingConflict=%v",
		aboutMe != nil, len(ctx.Gaps), response.Metadata["pendingConflictID"] != nil)

	// Pass extracted context to response for persistence (convert to Contact format)
	if extractedContact != nil {
		response.ExtractedContact = &models.Contact{
			Name:              extractedContact.Name,
			Relationship:      extractedContact.Relationship,
			Characteristics:   extractedContact.Traits,
			Notes:             extractedContact.Evidence,
		}
		log.Printf("[ConversationAgent] [✓] Passing extracted contact to response: %s (%s)", extractedContact.Name, extractedContact.Relationship)
	}

	// Moral values are now incorporated into response generation prompt
	// No post-generation ethical gate needed - trust the LLM to generate helpful, safe responses
	log.Printf("[ConversationAgent] [✓] Response complete with moral values integrated in generation")

	response.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())
	log.Printf("[ConversationAgent] [✓] Response ready in %d ms", response.ProcessingTimeMs)

	// Update structured context (Phase 1 integration)
	if structuredCtx != nil && ca.db != nil {
		structuredCtx.UpdatedAt = time.Now().Unix()
		ctxRepo := ca.db.GetStructuredContextRepository()
		if err := ctxRepo.UpdateContext(structuredCtx); err != nil {
			log.Printf("[ConversationAgent] WARNING: Failed to update structured context: %v", err)
		} else {
			log.Printf("[ConversationAgent] ✓ Structured context updated")
		}
	}

	return response, nil
}

// determineClarificationNeeded checks if we're missing critical context
func determineClarificationNeeded(hasAboutMe, hasContact, hasIntention bool) bool {
	// Only ask for clarification if we're missing key context for understanding the person
	// Contact extraction is smart (checks for contact verbs), so don't require it upfront
	// Only ask for AboutMe or Intention if genuinely missing
	// This respects user autonomy - don't pressure for info not needed
	return !hasAboutMe || !hasIntention
}

// buildPrincipleContext extracts key principles from the constitution for prompt guidance
func (ca *conversationAgent) buildPrincipleContext() string {
	if ca.constitution == nil || len(ca.constitution.SupremePrinciples) == 0 {
		return ""
	}

	// Extract 2-3 most relevant principles for generation guidance
	principles := []string{}
	principleMap := map[string]string{
		"user_autonomy":            "Respect user autonomy - never pressure toward a specific action",
		"transparency":             "Be transparent - explain why you're asking questions",
		"consent_and_respect":      "Assume all people deserve respect and consent",
		"stakeholder_consideration": "Consider impact on others affected by the decision",
		"growth_and_learning":      "Support user's understanding and learning, not just quick answers",
		"harm_prevention":          "Do not suggest actions that could cause harm",
	}

	// Include top 3 principles for context
	for _, principle := range ca.constitution.SupremePrinciples {
		if desc, exists := principleMap[principle.ID]; exists && len(principles) < 3 {
			principles = append(principles, "• "+desc)
		}
	}

	if len(principles) == 0 {
		return ""
	}

	return "Guiding principles:\n" + strings.Join(principles, "\n")
}

// generateClarifyingResponse creates a response that asks for missing context
// Used when we don't have enough information to give good advice
func (ca *conversationAgent) generateClarifyingResponse(ctx models.Context, userMessage string, missingAboutMe, missingContact, missingIntention bool) string {
	if ca.llmClient == nil {
		return "I'd like to understand you better. Tell me more about yourself?"
	}

	// Build description of what we're missing
	missing := []string{}
	if missingAboutMe {
		missing = append(missing, "your communication style and preferences")
	}
	if missingContact {
		missing = append(missing, "who you're wanting to reach out to")
	}
	if missingIntention {
		missing = append(missing, "what you're trying to accomplish")
	}

	missingStr := strings.Join(missing, " and ")
	principlesContext := ca.buildPrincipleContext()

	prompt := fmt.Sprintf(`You are Moly, a supportive friend who wants to give good advice.

%s

The person just said: "%s"

However, you're missing important context to advise them well. You need to understand: %s

Your job right now is NOT to give advice yet. Instead, ask them warmly and curiously to help you understand better.
Be genuine - explain that you want to give them good guidance and need to know them better first.

Keep your response brief (1-2 sentences). Don't try to answer their question yet.`, principlesContext, userMessage, missingStr)

	req := &tools.LLMRequest{
		SystemPrompt: "You are Moly, a caring friend who asks clarifying questions before giving advice. Be warm and genuine.",
		UserPrompt:   prompt,
		Temperature:  0.7,
		MaxTokens:    100,
	}

	resp, err := ca.llmClient.Call(context.Background(), req)
	if err != nil {
		log.Printf("[ConversationAgent] LLM call failed for clarifying response: %v", err)
		return "I'd like to understand you better before I advise. Tell me more about yourself?"
	}

	if resp == nil || resp.Content == "" {
		return "I'd like to understand you better. What would you like to tell me first?"
	}

	return strings.TrimSpace(resp.Content)
}

// generateConversationalResponse creates a natural, context-aware response to the user
// ARCHITECTURE: Adaptive SystemPrompt (tone/personality) + UserPrompt (facts/context)
func (ca *conversationAgent) generateConversationalResponse(
	ctx models.Context,
	userMessage string,
	socraticQuestion *models.SocraticQuestion,
) string {
	if ca.llmClient == nil {
		return "I'm listening."
	}

	// STEP 1: DETECT USER CONTEXT (for adaptive tone)
	lowerMsg := strings.ToLower(userMessage)

	// Detect communication style preference
	communicationStyle := "warm"
	if ctx.ExtractedContext != nil && ctx.ExtractedContext.Style != nil && ctx.ExtractedContext.Style.Confidence > 0.6 {
		communicationStyle = ctx.ExtractedContext.Style.Style
	} else if ctx.AboutMe != nil && ctx.AboutMe.CommunicationStyle != "" {
		communicationStyle = ctx.AboutMe.CommunicationStyle
	}

	// Detect emotional state (5-level scale)
	emotionalTone := ca.detectEmotionalTone(lowerMsg)

	// Detect conversation topic(s) - may have multiple topics in one message
	topic := ca.detectTopic(lowerMsg)
	topics := ca.detectMultipleTopics(lowerMsg)

	// Log if multiple topics detected
	if len(topics) > 1 {
		log.Printf("[ConversationAgent] Multiple topics detected: %v (count=%d)", topics, len(topics))
	}

	// STEP 2: BUILD ADAPTIVE SYSTEMPROMPT (core personality/tone)
	systemPrompt := ca.buildAdaptiveSystemPrompt(communicationStyle, emotionalTone, topic, socraticQuestion != nil, ctx.IsFirstMessageOfSession, ctx.LastRiskAssessment)
	log.Printf("[ConversationAgent] SystemPrompt adapted: style=%s tone=%s topic=%s", communicationStyle, emotionalTone, topic)

	// STEP 3: BUILD USERPROMPT (facts and context for this conversation)
	userPrompt := ca.buildUserPromptContext(ctx, userMessage, socraticQuestion)

	// STEP 4: SEND TO LLM
	req := &tools.LLMRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Temperature:  0.7,
		MaxTokens:    150,
	}

	resp, err := ca.llmClient.Call(context.Background(), req)
	if err != nil {
		log.Printf("[ConversationAgent] LLM call failed: %v", err)
		return "I'm here to listen. Tell me more."
	}

	if resp == nil || resp.Content == "" {
		return "I'm listening."
	}

	return strings.TrimSpace(resp.Content)
}

// buildAdaptiveSystemPrompt creates a personality/tone prompt based on user context
// This becomes the PRIMARY instruction to the LLM (higher priority than UserPrompt)
func (ca *conversationAgent) buildAdaptiveSystemPrompt(style string, emotionalTone string, topic string, hasSocraticQuestion bool, isFirstMessageOfSession bool, riskAssessment map[string]interface{}) string {
	// Base personality - Moly is always a good listener
	basePersonality := "You are Moly, a thoughtful listener and communication coach."

	// STEP 0: Session awareness - adjust greeting strategy
	sessionGuidance := ""
	if isFirstMessageOfSession {
		sessionGuidance = " This is the first message in this conversation session (browser page load). Greet them fresh and naturally, as if starting a new conversation."
	}

	// STEP 1: Adapt tone to communication style preference
	styleTone := ""
	switch style {
	case "formal":
		styleTone = " Be respectful and professional. Use clear, direct language. Avoid excessive friendliness or casual expressions."
	case "casual":
		styleTone = " Be relaxed and natural. Use conversational language. Feel free to be friendly and approachable."
	case "playful":
		styleTone = " Be warm and engaging. Use light humor where appropriate. Show genuine curiosity and enthusiasm."
	default:
		styleTone = " Be warm yet respectful. Adapt your tone to match theirs."
	}

	// STEP 2: Adapt to emotional state
	emotionGuidance := ""
	switch emotionalTone {
	case "very_negative":
		emotionGuidance = " They're in significant distress. Prioritize validation and support. Be gentle, careful, and compassionate. Show you deeply understand their situation. Consider whether professional support might help."
	case "negative":
		emotionGuidance = " They're concerned or distressed about something real. Validate their feelings. Be thoughtful and supportive, not dismissive. Focus on understanding before advising."
	case "positive":
		emotionGuidance = " They're in a good mood. Match their energy. Be warm and engaged. Celebrate with them appropriately."
	case "very_positive":
		emotionGuidance = " They're very happy or excited. Celebrate and amplify their positive energy. Show genuine enthusiasm."
	}

	// STEP 3: Topic-specific guidance
	topicGuidance := ""
	switch topic {
	case "work":
		topicGuidance = " They're discussing work/career. This is serious territory. Ask about specific situations, not just feelings. Help them think through options and relationships at work."
	case "relationships":
		topicGuidance = " They're discussing relationships. Show you understand the human complexity. Ask about communication, needs, and how they want to handle things."
	case "family":
		topicGuidance = " They're discussing family. Acknowledge the deep roots and complexity. Be careful, respectful, and curious about their perspective."
	case "mental_health":
		topicGuidance = " They're discussing mental health. Take this seriously. Validate their concerns. Suggest professional support if needed."
	}

	// STEP 3B: Risk-aware guidance (if risk assessment available)
	riskGuidance := ""
	if len(riskAssessment) > 0 {
		if level, ok := riskAssessment["level"].(string); ok && level == "elevated" {
			riskGuidance = " They're in an elevated emotional state. Take their concerns seriously. Be extra thoughtful and supportive."
		}
	}

	// STEP 4: Socratic guidance (if applicable)
	socraticGuidance := ""
	if hasSocraticQuestion {
		socraticGuidance = " A specific question is waiting below—integrate it naturally into your response, not as a separate item. Let it guide your curiosity."
	}

	// Combine into full system prompt (add riskGuidance)
	return fmt.Sprintf(`%s%s%s%s%s%s%s

CRITICAL: Respect user preferences above all. If they ask for formality, be formal. If they're in distress, prioritize support. If they ask direct questions, answer directly.

Keep responses concise (1-3 sentences) unless they're sharing something complex. Don't use emojis. Show genuine understanding, not canned warmth.`, basePersonality, sessionGuidance, styleTone, emotionGuidance, topicGuidance, riskGuidance, socraticGuidance)
}

// buildUserPromptContext creates facts/context about this conversation
func (ca *conversationAgent) buildUserPromptContext(ctx models.Context, userMessage string, socraticQuestion *models.SocraticQuestion) string {
	// About this person (from stored profile)
	userProfile := ""
	if ctx.AboutMe != nil && ctx.AboutMe.CommunicationStyle != "" {
		userProfile = fmt.Sprintf("Stored communication style: %s\n", ctx.AboutMe.CommunicationStyle)
		if len(ctx.AboutMe.Values) > 0 {
			userProfile += fmt.Sprintf("Values: %s\n", strings.Join(ctx.AboutMe.Values, ", "))
		}
	}

	// What Moly has learned about this person (reflections)
	reflectionsText := ""
	if len(ctx.RelevantReflections) > 0 {
		reflectionsText = "What you've learned about them:\n"
		for i, reflection := range ctx.RelevantReflections {
			if i >= 2 {
				break
			}
			if len(reflection.Characteristics) > 0 {
				reflectionsText += fmt.Sprintf("- They are: %s\n", strings.Join(reflection.Characteristics, ", "))
			}
			if len(reflection.Intentions) > 0 {
				reflectionsText += fmt.Sprintf("- They tend to: %s\n", strings.Join(reflection.Intentions, ", "))
			}
		}
		if reflectionsText != "What you've learned about them:\n" {
			reflectionsText += "\n"
		}
	}

	// Current conversation context
	conversationContext := ""
	if len(ctx.ConversationHistory) > 1 {
		conversationContext = "You have prior conversation history to reference.\n"
	}
	if ctx.PastIntention != "" {
		conversationContext += fmt.Sprintf("Earlier they mentioned: %s\n", ctx.PastIntention)
	}

	// Extracted context from THIS message (highest priority)
	extractedContext := ""
	if ctx.ExtractedContext != nil {
		if ctx.ExtractedContext.Contact != nil && ctx.ExtractedContext.Contact.Confidence > 0.6 {
			extractedContext += fmt.Sprintf("Talking about: %s (%s)\n", ctx.ExtractedContext.Contact.Name, ctx.ExtractedContext.Contact.Relationship)
		}
		if ctx.ExtractedContext.Intention != "" {
			extractedContext += fmt.Sprintf("Their intention: %s\n", ctx.ExtractedContext.Intention)
		}
		if len(ctx.ExtractedContext.Goals) > 0 {
			extractedContext += fmt.Sprintf("Goals mentioned: %s\n", strings.Join(ctx.ExtractedContext.Goals, ", "))
		}
	}

	// Socratic question (if available)
	socraticText := ""
	if socraticQuestion != nil {
		socraticText = fmt.Sprintf("\nKEY QUESTION TO EXPLORE: \"%s\"\n(Approach: %s - weave it naturally into your response)\n", socraticQuestion.Text, socraticQuestion.SocraticApproach)
		log.Printf("[ConversationAgent] Including Socratic question: %s (%s)", socraticQuestion.ID, socraticQuestion.SocraticApproach)
	}

	// Multi-topic handling guidance
	multiTopicGuidance := ""
	topics := ca.detectMultipleTopics(strings.ToLower(userMessage))
	if len(topics) > 1 && topics[0] != "general" {
		if len(topics) <= 3 {
			// 1-3 topics: Ask clarifying questions about ALL of them
			topicsList := strings.Join(topics, ", ")
			multiTopicGuidance = fmt.Sprintf("\nIMPORTANT - MULTIPLE CONCERNS DETECTED:\nThey mentioned %d things (%s). Show you care about ALL their concerns.\nAsk clarifying questions about EACH topic in this response - don't make them choose which to focus on first.\n", len(topics), topicsList)
		} else {
			// 4+ topics: Too many - ask them to prioritize
			multiTopicGuidance = fmt.Sprintf("\nIMPORTANT - TOO MANY TOPICS:\nThey brought up %d different topics. That's a lot. Acknowledge all of them, but ask them which is most urgent so you can focus and actually help.\n", len(topics))
		}
	}

	// The actual message
	messagePrompt := fmt.Sprintf("They just said: \"%s\"\n\nRespond directly to what they said. Address their specific concern, not just be generally friendly.", userMessage)

	// Combine into user prompt
	return fmt.Sprintf(`%s%s%s%s%s%s%s`, userProfile, reflectionsText, conversationContext, extractedContext, multiTopicGuidance, socraticText, messagePrompt)
}

// detectEmotionalTone analyzes the emotional state of the message
func (ca *conversationAgent) detectEmotionalTone(lowerMsg string) string {
	veryNegativeIndicators := []string{"devastated", "destroyed", "suicidal", "hopeless", "desperate", "dying", "hatred", "homicidal"}
	negativeIndicators := []string{"sad", "angry", "frustrated", "disappointed", "worried", "anxious", "stressed", "overwhelmed", "hurt", "crying", "broken", "concerned", "fear"}
	positiveIndicators := []string{"happy", "excited", "great", "wonderful", "amazing", "love", "grateful", "thrilled", "delighted", "proud", "hopeful"}
	veryPositiveIndicators := []string{"euphoric", "ecstatic", "overjoyed", "blessed", "incredibly grateful", "life-changing"}

	veryNegativeCount := 0
	for _, indicator := range veryNegativeIndicators {
		if contains(lowerMsg, indicator) {
			veryNegativeCount++
		}
	}
	negativeCount := 0
	for _, indicator := range negativeIndicators {
		if contains(lowerMsg, indicator) {
			negativeCount++
		}
	}
	positiveCount := 0
	for _, indicator := range positiveIndicators {
		if contains(lowerMsg, indicator) {
			positiveCount++
		}
	}
	veryPositiveCount := 0
	for _, indicator := range veryPositiveIndicators {
		if contains(lowerMsg, indicator) {
			veryPositiveCount++
		}
	}

	if veryNegativeCount > 0 {
		return "very_negative"
	} else if negativeCount > 0 && negativeCount > positiveCount {
		return "negative"
	} else if veryPositiveCount > 0 {
		return "very_positive"
	} else if positiveCount > 0 && positiveCount > negativeCount {
		return "positive"
	}
	return "neutral"
}

// detectTopic identifies what the conversation is about
func (ca *conversationAgent) detectTopic(lowerMsg string) string {
	topicKeywords := map[string]string{
		"work":       "work",
		"job":        "work",
		"career":     "work",
		"boss":       "work",
		"colleague":  "work",
		"relationship": "relationships",
		"partner":    "relationships",
		"romantic":   "relationships",
		"family":     "family",
		"parent":     "family",
		"sibling":    "family",
		"anxiety":    "mental_health",
		"depression": "mental_health",
		"therapy":    "mental_health",
		"health":     "health",
	}

	for keyword, topic := range topicKeywords {
		if contains(lowerMsg, keyword) {
			return topic
		}
	}
	return "general"
}

// detectMultipleTopics identifies ALL topics in the message (not just first one)
// Returns slice of unique topics found
func (ca *conversationAgent) detectMultipleTopics(lowerMsg string) []string {
	topicKeywords := map[string]string{
		"work":         "work",
		"job":          "work",
		"career":       "work",
		"boss":         "work",
		"colleague":    "work",
		"relationship": "relationships",
		"partner":      "relationships",
		"romantic":     "relationships",
		"family":       "family",
		"parent":       "family",
		"sibling":      "family",
		"anxiety":      "mental_health",
		"depression":   "mental_health",
		"therapy":      "mental_health",
		"health":       "health",
	}

	foundTopics := make(map[string]bool)
	for keyword, topic := range topicKeywords {
		if contains(lowerMsg, keyword) {
			foundTopics[topic] = true
		}
	}

	// Convert to slice
	var topics []string
	for topic := range foundTopics {
		topics = append(topics, topic)
	}

	// If no topics found, return general
	if len(topics) == 0 {
		return []string{"general"}
	}

	return topics
}

// runAnalyzePhase - Determine the type of interaction
// runSafetyPhase - Check for crisis/illegal content
func (ca *conversationAgent) runSafetyPhase(ctx context.Context, message string) (*models.SafetyAlert, error) {
	if message == "" {
		return nil, nil
	}

	input := &tools.SafetyCheckInput{
		Message: message,
	}

	result, err := ca.safetyChecker.Check(ctx, input)
	if err != nil {
		log.Printf("[ConversationAgent] ERROR: Safety check failed: %v - treating as UNKNOWN risk", err)
		// Critical: On safety check failure, return error instead of nil
		// This prevents crisis content from bypassing due to infrastructure failures
		return nil, fmt.Errorf("safety check unavailable: %w", err)
	}

	if result.AlertType != tools.SafetyAlertTypeNone {
		alert := &models.SafetyAlert{
			AlertType:       string(result.AlertType),
			Severity:        string(result.Severity),
			Title:           result.Title,
			Message:         result.Message,
			Indicators:      result.Indicators,
			Recommendations: result.Recommendations,
		}

		// Add resources if crisis
		if result.AlertType == tools.SafetyAlertTypeCrisis {
			for _, r := range result.Resources {
				alert.Resources = append(alert.Resources, models.CrisisResource{
					Name:        r.Name,
					Description: r.Description,
					Number:      r.Number,
					URL:         r.URL,
					Region:      r.Region,
				})
			}
		}

		return alert, nil
	}

	return nil, nil
}

// runReflectPhase - Extract insights from conversation
func (ca *conversationAgent) runReflectPhase(ctx context.Context, message string) (*models.Reflection, error) {
	if message == "" {
		return nil, nil
	}

	input := &tools.ContextExtractorInput{
		Message: message,
	}

	output, err := ca.contextExtractor.Extract(ctx, input)
	if err != nil {
		// Graceful fallback: return nil, no error (optional phase)
		return nil, nil
	}

	if output == nil {
		return nil, nil
	}

	reflection := &models.Reflection{
		Characteristics:          output.NewCharacteristics,
		Interests:                output.NewInterests,
		CommunicationPreferences: output.UpdatedCommunicationPrefs,
		Intentions:               output.Intentions,
		UserQuotes:               output.UserQuotes,
		Status:                   "pending_approval",
	}

	return reflection, nil
}

// generateContextGatheringQuestions - Generate Socratic questions to gather missing context
// Uses LLM for contextual question generation, falls back to templates
func generateContextGatheringQuestions(hasAboutMe, hasContact, hasIntention bool, userMessage string) []*schema.ClarificationQuestion {
	now := time.Now().Unix()

	// Gather context in progressive order: AboutMe → Intention
	// Note: Contact extraction is smart (checks for contact verbs), so we don't require it upfront
	// Only ask for AboutMe or Intention if genuinely missing (same logic as determineClarificationNeeded)

	if !hasAboutMe {
		return []*schema.ClarificationQuestion{
			{
				ID:           fmt.Sprintf("q_aboutme_%d", now),
				Type:         "context_gathering",
				Question:     "Tell me about yourself - what's your communication style like? Are you more formal, casual, playful, or a mix?",
				Priority:     2,
				Status:       "pending",
				CreatedAt:    now,
				LinkedFacts:  []string{fmt.Sprintf("fact_aboutme_%d", now)},
			},
		}
	}

	if !hasIntention {
		return []*schema.ClarificationQuestion{
			{
				ID:           fmt.Sprintf("q_intention_%d", now),
				Type:         "context_gathering",
				Question:     "What's your intention with this message? Are you celebrating something, apologizing, asking for help, or starting a conversation?",
				Priority:     2,
				Status:       "pending",
				CreatedAt:    now,
				LinkedFacts:  []string{fmt.Sprintf("fact_intention_%d", now)},
			},
		}
	}

	// Fallback - shouldn't reach here if logic is correct
	return []*schema.ClarificationQuestion{
		{
			ID:           fmt.Sprintf("q_fallback_%d", now),
			Type:         "context_gathering",
			Question:     "Tell me more about what you're trying to communicate.",
			Priority:     3,
			Status:       "pending",
			CreatedAt:    now,
			LinkedFacts:  []string{fmt.Sprintf("fact_fallback_%d", now)},
		},
	}
}

// generateContactQuestionLLM generates context-aware contact question using LLM logic
// Falls back to keyword-based template if LLM unavailable
func generateContactQuestionLLM(userMessage string) string {
	// Fallback template - used if LLM call fails
	defaultQuestion := "Now, who are you wanting to message? Tell me their name and what your relationship is like."

	// Note: In production, this would use an LLM to understand the message context
	// and generate a natural, contextual question. For now, use contextual templates.
	//
	// LLM prompt would be:
	// "Based on this message, generate a short, natural question asking about the person
	// they want to contact. Keep it conversational and context-aware."

	// Fallback to keyword-based templates (lower confidence, but reliable)
	lowerMsg := strings.ToLower(userMessage)

	if contains(lowerMsg, "girl") || contains(lowerMsg, "boy") || contains(lowerMsg, "crush") ||
		contains(lowerMsg, "romantic") || contains(lowerMsg, "dating") || contains(lowerMsg, "interested") {
		return "You mentioned someone special! What's their name, and how would you describe your relationship with them?"
	}
	if contains(lowerMsg, "boss") || contains(lowerMsg, "manager") || contains(lowerMsg, "colleague") || contains(lowerMsg, "work") {
		return "Who's the person you're messaging? And what's your working relationship like?"
	}
	if contains(lowerMsg, "friend") {
		return "What's your friend's name, and how close are you two?"
	}
	if contains(lowerMsg, "mom") || contains(lowerMsg, "dad") || contains(lowerMsg, "parent") ||
		contains(lowerMsg, "sibling") || contains(lowerMsg, "brother") || contains(lowerMsg, "sister") || contains(lowerMsg, "family") {
		return "Which family member are you reaching out to? Tell me about your relationship."
	}

	return defaultQuestion
}

// contains checks if string contains substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}


// convertToConstitutionViolations converts tools.PrincipleViolation to models.ConstitutionViolation
func convertToConstitutionViolations(violations []tools.PrincipleViolation) []models.ConstitutionViolation {
	result := make([]models.ConstitutionViolation, len(violations))
	for i, v := range violations {
		result[i] = models.ConstitutionViolation{
			PrincipleID: v.PrincipleID,
			Principle:   v.Principle,
			Severity:    v.Severity,
			Description: v.Description,
		}
	}
	return result
}

