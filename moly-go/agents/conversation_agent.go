package agents

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"moly/config"
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
	constitutionEvaluator *tools.ConstitutionEvaluator
	contextExtractor      *tools.ContextExtractor
	harmAnalyzer          *tools.HarmAnalyzer
	socraticSelector      *SocraticQuestionSelector // Optional: for Socratic question selection
}

// NewConversationAgent - Create new conversation agent
func NewConversationAgent(llm tools.LLMProvider) (models.ConversationAgent, error) {
	// LLM client is optional - agent will generate basic suggestions without it
	return &conversationAgent{
		llmClient:             llm,
		suggestionGenerator:   tools.NewSuggestionGenerator(llm),
		questionGenerator:     tools.NewQuestionGenerator(llm),
		safetyChecker:         tools.NewSafetyChecker(llm),
		constitutionEvaluator: tools.NewConstitutionEvaluator(llm),
		contextExtractor:      tools.NewContextExtractor(llm),
		harmAnalyzer:          tools.NewHarmAnalyzer(llm),
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

// SetConstitution injects the constitution into the harm analyzer for principle-based checking (optional)
func (ca *conversationAgent) SetConstitution(constitution *models.Constitution) {
	if ca != nil && ca.harmAnalyzer != nil {
		ca.harmAnalyzer.SetConstitution(constitution)
		log.Printf("[ConversationAgent] Constitution injected into HarmAnalyzer")
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

	// Wire constitution into the conversation agent
	if caImpl, ok := agent.(*conversationAgent); ok {
		// Set constitution for principle-based checking in HarmAnalyzer
		caImpl.SetConstitution(constitution)
		log.Printf("[ConversationAgent] ✓ Constitution loaded for principle-based checking")

		// Set Socratic selector if library loaded successfully
		if library != nil {
			selector := NewSocraticQuestionSelector(library, constitution)
			caImpl.SetSocraticSelector(selector)
			log.Printf("[ConversationAgent] ✓ Socratic question selector initialized")
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
		response.Response = "I didn't receive your message. Please try again."
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
		response.Response = "Your message was empty. What's on your mind?"
		response.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())
		return response, nil
	}

	log.Printf("[ConversationAgent] User message: %.80s...", userMessage)

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

	// Fallback: Extract contact name from user response if they mention who they want to message
	if !hasContact && userMessage != "" {
		lowerMsg := strings.ToLower(userMessage)
		// Professional relationships
		if contains(lowerMsg, "boss") || contains(lowerMsg, "manager") || contains(lowerMsg, "colleague") {
			if contact == nil {
				contact = &models.Contact{}
			}
			contact.Name = "Boss"
			contact.Relationship = "professional"
			hasContact = true
			log.Printf("[ConversationAgent] Extracted contact: Boss/Manager (professional, fallback)")
		} else if contains(lowerMsg, "friend") && !contains(lowerMsg, "best friend") && !contains(lowerMsg, "close friend") {
			if contact == nil {
				contact = &models.Contact{}
			}
			contact.Name = "Friend"
			contact.Relationship = "friend"
			hasContact = true
			log.Printf("[ConversationAgent] Extracted contact: Friend (fallback)")
		} else if contains(lowerMsg, "mom") || contains(lowerMsg, "dad") || contains(lowerMsg, "parent") ||
				contains(lowerMsg, "sibling") || contains(lowerMsg, "brother") || contains(lowerMsg, "sister") {
			if contact == nil {
				contact = &models.Contact{}
			}
			contact.Name = "Family"
			contact.Relationship = "family"
			hasContact = true
			log.Printf("[ConversationAgent] Extracted contact: Family member (fallback)")
		} else if contains(lowerMsg, "girl") || contains(lowerMsg, "boy") || contains(lowerMsg, "crush") ||
				contains(lowerMsg, "partner") || contains(lowerMsg, "spouse") || contains(lowerMsg, "girlfriend") ||
				contains(lowerMsg, "boyfriend") || contains(lowerMsg, "date") || contains(lowerMsg, "romantic") ||
				contains(lowerMsg, "likes me") || contains(lowerMsg, "interested in") {
			if contact == nil {
				contact = &models.Contact{}
			}
			contact.Name = "Romantic Interest"
			contact.Relationship = "romantic"
			hasContact = true
			log.Printf("[ConversationAgent] Extracted contact: Romantic interest (from message context, fallback)")
		}
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
	if ca.llmClient == nil {
		if needsClarification {
			response.Response = "I'd like to understand you better. Tell me more?"
		} else {
			response.Response = "I'm listening."
		}
		log.Printf("[ConversationAgent] No LLM available, using fallback response")
	} else {
		log.Printf("[ConversationAgent] Generating response (needsClarification=%v)", needsClarification)

		var generatedResponse string
		if needsClarification {
			// Ask for missing context first
			missingAboutMe := !hasAboutMe
			missingContact := !hasContact
			missingIntention := !hasIntention
			generatedResponse = ca.generateClarifyingResponse(ctx, userMessage, missingAboutMe, missingContact, missingIntention)
			log.Printf("[ConversationAgent] ✓ Generated clarifying response: %.100s...", generatedResponse)
		} else {
			// Give full response with available context
			generatedResponse = ca.generateConversationalResponse(ctx, userMessage)
			log.Printf("[ConversationAgent] ✓ Generated full response: %.100s...", generatedResponse)
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
			log.Printf("[ConversationAgent] ✓ Learned about user: %d characteristics", len(reflection.Characteristics))
		}
	}

	// GENERATE CLARIFICATION QUESTIONS
	// If Moly is missing critical context, ask clarifying questions to understand better
	log.Printf("[ConversationAgent] Checking if clarification questions needed (hasAboutMe=%v hasContact=%v hasIntention=%v)", hasAboutMe, hasContact, hasIntention)

	var questions []*schema.ClarificationQuestion

	// Try using Socratic selector if available
	if ca.socraticSelector != nil && (hasAboutMe || hasContact || hasIntention) {
		log.Printf("[ConversationAgent] Using Socratic selector for question generation")
		socraticQ := ca.socraticSelector.SelectNextQuestion(&ctx, userMessage, []models.SocraticQuestion{})
		if socraticQ != nil {
			clarQ := socraticQuestionToClarification(socraticQ)
			questions = append(questions, clarQ)
			log.Printf("[ConversationAgent] ✓ Generated Socratic question: %s (approach: %s)", socraticQ.ID, socraticQ.SocraticApproach)
		}
	}

	// Fallback to deterministic template-based questions if no Socratic selector
	if len(questions) == 0 {
		questions = generateContextGatheringQuestions(hasAboutMe, hasContact, hasIntention, userMessage)
		log.Printf("[ConversationAgent] Using template-based questions (Socratic selector unavailable)")
	}

	if len(questions) > 0 {
		response.Questions = questions
		log.Printf("[ConversationAgent] ✓ Generated %d clarification question(s)", len(questions))
	}

	// BUILD METADATA (initialize BEFORE ethical gate, so interventions can write into it)
	response.Metadata = map[string]interface{}{
		"conversational":        true,
		"hasUserProfile":        aboutMe != nil,
		"contextGaps":           len(ctx.Gaps),
		"clarificationRequired": len(questions) > 0,
		"questionsCount":        len(questions),
		"timestamp":             startTime.Unix(),
	}
	log.Printf("[ConversationAgent] ✓ Metadata initialized with base fields: conversational=true profile=%v gaps=%d",
		aboutMe != nil, len(ctx.Gaps))

	// ETHICAL GATE: Check if response could cause harm
	// This runs BEFORE we return, applying logic-based harm reasoning
	log.Printf("[ConversationAgent] Starting ethical gate analysis...")

	userVuln := ca.buildUserVulnerability(ctx)
	contactTraits := ca.buildContactTraits(contact)
	log.Printf("[ConversationAgent] Built vulnerability context: trauma=%v mental_issues=%d patterns=%d",
		userVuln.TraumaHistory, len(userVuln.MentalHealthIssues), len(userVuln.Patterns))

	harmAnalysis, err := ca.harmAnalyzer.AnalyzeResponse(
		context.Background(),
		response.Response,
		userVuln,
		contactTraits,
	)

	if err != nil {
		log.Printf("[ConversationAgent] ⚠️ Harm analysis failed: %v (proceeding without analysis)", err)
	} else if harmAnalysis != nil {
		// Log the analysis
		log.Printf("[ConversationAgent] ✓ Harm analysis complete: severity=%s intervention=%s principles_violated=%d affected=%v",
			harmAnalysis.Severity, harmAnalysis.Intervention, len(harmAnalysis.ViolatedPrinciples), harmAnalysis.AffectedParties)

		// Add violated principles to metadata (always included if present)
		if len(harmAnalysis.ViolatedPrinciples) > 0 {
			response.Metadata["violatedPrinciples"] = harmAnalysis.ViolatedPrinciples
			log.Printf("[ConversationAgent] 📋 Added violated principles to metadata: %v", harmAnalysis.ViolatedPrinciples)
		}

		// Apply intervention if needed
		if harmAnalysis.ShouldBlock() {
			log.Printf("[ConversationAgent] 🚫 BLOCKING response: %s", harmAnalysis.Reasoning)
			response.Response = "I need to be careful with my advice here. " + harmAnalysis.Explanation
			response.Metadata["ethicalIntervention"] = "blocked"
			response.Metadata["blockReason"] = harmAnalysis.Reasoning
			log.Printf("[ConversationAgent] ✓ Metadata updated: ethicalIntervention=blocked blockReason=%v", harmAnalysis.Reasoning != "")
		} else if harmAnalysis.ShouldModify() {
			log.Printf("[ConversationAgent] 🔄 MODIFYING response: %s", harmAnalysis.Reasoning)
			if harmAnalysis.ModifiedResponse != "" {
				response.Response = harmAnalysis.ModifiedResponse
			}
			response.Metadata["ethicalIntervention"] = "modified"
			response.Metadata["modificationReason"] = harmAnalysis.Reasoning
			if harmAnalysis.Explanation != "" {
				response.Metadata["ethicalNote"] = harmAnalysis.Explanation
			}
			log.Printf("[ConversationAgent] ✓ Metadata updated: ethicalIntervention=modified modificationReason=%v note=%v",
				harmAnalysis.Reasoning != "", harmAnalysis.Explanation != "")
		} else if harmAnalysis.ShouldWarn() {
			log.Printf("[ConversationAgent] ⚠️ WARNING about response: %s", harmAnalysis.Reasoning)
			response.Metadata["ethicalIntervention"] = "warned"
			response.Metadata["warningReason"] = harmAnalysis.Reasoning
			if harmAnalysis.Explanation != "" {
				response.Metadata["ethicalWarning"] = harmAnalysis.Explanation
			}
			log.Printf("[ConversationAgent] ✓ Metadata updated: ethicalIntervention=warned warningReason=%v warning=%v",
				harmAnalysis.Reasoning != "", harmAnalysis.Explanation != "")
		} else {
			log.Printf("[ConversationAgent] ✓ No intervention needed - proceeding with original response")
		}
	} else {
		log.Printf("[ConversationAgent] ℹ️ No harm analysis result")
	}

	response.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())
	log.Printf("[ConversationAgent] ✓ Response ready in %d ms", response.ProcessingTimeMs)

	return response, nil
}

// determineClarificationNeeded checks if we're missing critical context
func determineClarificationNeeded(hasAboutMe, hasContact, hasIntention bool) bool {
	// If we're missing ANY critical context piece, we need clarification
	return !hasAboutMe || !hasContact || !hasIntention
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

	prompt := fmt.Sprintf(`You are Moly, a supportive friend who wants to give good advice.

The person just said: "%s"

However, you're missing important context to advise them well. You need to understand: %s

Your job right now is NOT to give advice yet. Instead, ask them warmly and curiously to help you understand better.
Be genuine - explain that you want to give them good guidance and need to know them better first.

Keep your response brief (1-2 sentences). Don't try to answer their question yet.`, userMessage, missingStr)

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

// generateConversationalResponse creates a natural, empathetic response from Moly
func (ca *conversationAgent) generateConversationalResponse(ctx models.Context, userMessage string) string {
	if ca.llmClient == nil {
		return "I'm listening."
	}

	// Build context about the user for the prompt
	userProfile := ""
	if ctx.AboutMe != nil {
		if ctx.AboutMe.CommunicationStyle != "" {
			userProfile += fmt.Sprintf("Communication style: %s\n", ctx.AboutMe.CommunicationStyle)
		}
		if len(ctx.AboutMe.Values) > 0 {
			userProfile += fmt.Sprintf("Values: %s\n", strings.Join(ctx.AboutMe.Values, ", "))
		}
	}

	// Reference conversation history for context
	pastContext := ""
	if len(ctx.ConversationHistory) > 1 {
		pastContext = "Recent conversation context has been shared with you.\n"
	}

	prompt := fmt.Sprintf(`You are Moly, a supportive friend who listens deeply and learns about the person you're talking with.

About this person:
%s

%s

The person just said: "%s"

Respond naturally and conversationally. Be warm, understanding, and genuinely curious about them. Don't be robotic or clinical. Ask follow-up questions if appropriate. Show that you're listening and that you care about what they're sharing.

Keep your response concise (1-3 sentences) unless they're sharing something complex.`, userProfile, pastContext, userMessage)

	req := &tools.LLMRequest{
		SystemPrompt: "You are Moly, a good friend who understands and cares about people. Be natural, warm, and authentic in your responses.",
		UserPrompt:   prompt,
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

	// Gather context in progressive order: AboutMe → Contact → Intention
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

	if !hasContact {
		// Use LLM to generate context-specific question based on message content
		question := generateContactQuestionLLM(userMessage)
		return []*schema.ClarificationQuestion{
			{
				ID:           fmt.Sprintf("q_contact_%d", now),
				Type:         "context_gathering",
				Question:     question,
				Priority:     2,
				Status:       "pending",
				CreatedAt:    now,
				LinkedFacts:  []string{fmt.Sprintf("fact_contact_%d", now)},
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

// generateLLMSuggestions - Generate suggestions using LLM
func (ca *conversationAgent) generateLLMSuggestions(ctx models.Context, userMessage string, intention string) []models.Suggestion {
	if ca.llmClient == nil || ca.suggestionGenerator == nil {
		return []models.Suggestion{}
	}

	aboutMe := ctx.AboutMe
	contact := ctx.ContactProfile

	// Build input for suggestion generator
	input := &tools.SuggestionGeneratorInput{
		UserMessage:            userMessage,
		UserCommunicationStyle: "friendly",
		UserValues:             []string{},
		ContactCharacteristics: []string{},
		ContactInterests:       []string{},
		ContactRelationship:    "friend",
		UserIntention:          intention,
		Mode:                   "direct",
		Tone:                   "friendly",
	}

	if aboutMe != nil {
		input.UserCommunicationStyle = aboutMe.CommunicationStyle
		input.UserValues = aboutMe.Values
		input.Tone = aboutMe.PreferredTone
	}

	if contact != nil {
		input.ContactRelationship = contact.Relationship
		input.ContactCharacteristics = contact.Characteristics
		input.ContactInterests = contact.Interests
	}

	// Call suggestion generator
	genCtx := context.Background()
	output, err := ca.suggestionGenerator.Generate(genCtx, input)
	if err != nil {
		return []models.Suggestion{}
	}

	if output == nil || len(output.Suggestions) == 0 {
		return []models.Suggestion{}
	}

	// Convert to models.Suggestion
	suggestions := make([]models.Suggestion, len(output.Suggestions))
	for i, s := range output.Suggestions {
		suggestions[i] = models.Suggestion{
			Index:      s.Index,
			Text:       s.Text,
			Tone:       s.Tone,
			Reasoning:  s.Reasoning,
			Confidence: s.Confidence,
		}
	}

	return suggestions
}

// generateContextualSuggestions creates personalized suggestions based on context
func generateContextualSuggestions(aboutMe *models.AboutMe, contact *models.Contact, userMessage string, intention string) []models.Suggestion {
	suggestions := []models.Suggestion{}

	// Get user's communication style
	userStyle := "friendly"
	if aboutMe != nil && aboutMe.CommunicationStyle != "" {
		userStyle = aboutMe.CommunicationStyle
	}

	// Get contact's known preferences
	contactName := "them"
	if contact != nil && contact.Name != "" {
		contactName = contact.Name
	}

	// Generate suggestions based on intention and context
	switch intention {
	case "celebrate":
		suggestions = []models.Suggestion{
			{
				Index:      0,
				Text:       "That's amazing! I'm so happy for you! 🎉",
				Tone:       userStyle,
				Reasoning:  fmt.Sprintf("Genuine celebration in your authentic %s style, perfect for %s", userStyle, contactName),
				Confidence: 0.92,
			},
			{
				Index:      1,
				Text:       "Congratulations! You deserve this. Tell me everything!",
				Tone:       userStyle,
				Reasoning:  fmt.Sprintf("Shows genuine interest and excitement, matches how you naturally communicate"),
				Confidence: 0.88,
			},
			{
				Index:      2,
				Text:       "This is huge! I'd love to hear all about it.",
				Tone:       userStyle,
				Reasoning:  fmt.Sprintf("Enthusiastic but not over-the-top, allows space for them to share"),
				Confidence: 0.85,
			},
		}
	case "apologize":
		suggestions = []models.Suggestion{
			{
				Index:      0,
				Text:       "I'm sorry for how I handled that. I should have communicated better.",
				Tone:       userStyle,
				Reasoning:  fmt.Sprintf("Takes responsibility without over-explaining, authentic to your style"),
				Confidence: 0.90,
			},
			{
				Index:      1,
				Text:       "I want to make this right. What can I do?",
				Tone:       userStyle,
				Reasoning:  fmt.Sprintf("Action-oriented, shows commitment to resolution"),
				Confidence: 0.86,
			},
			{
				Index:      2,
				Text:       "I regret that. Can we talk about it?",
				Tone:       userStyle,
				Reasoning:  fmt.Sprintf("Direct and respectful, opens dialogue without being defensive"),
				Confidence: 0.84,
			},
		}
	case "seek_help":
		suggestions = []models.Suggestion{
			{
				Index:      0,
				Text:       "I'm dealing with something and could really use your perspective.",
				Tone:       userStyle,
				Reasoning:  fmt.Sprintf("Vulnerable but specific, respects their time and expertise"),
				Confidence: 0.89,
			},
			{
				Index:      1,
				Text:       "I'm stuck on something. Do you have time to talk?",
				Tone:       userStyle,
				Reasoning:  fmt.Sprintf("Clear and direct, gives them the choice to engage"),
				Confidence: 0.87,
			},
			{
				Index:      2,
				Text:       "Can I get your advice on something?",
				Tone:       userStyle,
				Reasoning:  fmt.Sprintf("Values their input, shows respect for their opinion"),
				Confidence: 0.85,
			},
		}
	case "greet":
		suggestions = []models.Suggestion{
			{
				Index:      0,
				Text:       "Hey! How's it going?",
				Tone:       userStyle,
				Reasoning:  fmt.Sprintf("Warm and casual, matches your natural communication style with %s", contactName),
				Confidence: 0.88,
			},
			{
				Index:      1,
				Text:       "Hi! What's new with you?",
				Tone:       userStyle,
				Reasoning:  fmt.Sprintf("Friendly opener that invites them to share"),
				Confidence: 0.85,
			},
			{
				Index:      2,
				Text:       "Great to hear from you! What's up?",
				Tone:       userStyle,
				Reasoning:  fmt.Sprintf("Shows genuine warmth and interest in their updates"),
				Confidence: 0.84,
			},
		}
	default:
		// General fallback suggestions
		suggestions = []models.Suggestion{
			{
				Index:      0,
				Text:       "That sounds important. Tell me more.",
				Tone:       userStyle,
				Reasoning:  fmt.Sprintf("Shows genuine interest, matches your authentic communication style"),
				Confidence: 0.85,
			},
			{
				Index:      1,
				Text:       "I'm listening. What's on your mind?",
				Tone:       userStyle,
				Reasoning:  fmt.Sprintf("Open and welcoming, invites deeper conversation"),
				Confidence: 0.82,
			},
			{
				Index:      2,
				Text:       "How are you feeling about all this?",
				Tone:       userStyle,
				Reasoning:  fmt.Sprintf("Empathetic and present, helps them reflect"),
				Confidence: 0.80,
			},
		}
	}

	return suggestions
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

// buildUserVulnerability extracts user vulnerability factors from available context
func (ca *conversationAgent) buildUserVulnerability(ctx models.Context) *tools.UserVulnerability {
	if ctx.RelevantReflections == nil || len(ctx.RelevantReflections) == 0 {
		return nil
	}

	vuln := &tools.UserVulnerability{}
	mentionalHealthSet := make(map[string]bool)
	patternsSet := make(map[string]bool)

	// Look for trauma indicators in user characteristics
	for _, reflection := range ctx.RelevantReflections {
		if reflection.Characteristics != nil {
			for _, char := range reflection.Characteristics {
				charLower := strings.ToLower(char)
				if contains(char, "trauma") || contains(char, "abuse") || contains(char, "ptsd") {
					vuln.TraumaHistory = true
				}
				if contains(char, "depression") || contains(char, "anxiety") || contains(char, "panic") {
					mentionalHealthSet[charLower] = true
				}
				if contains(char, "conflict-avoidant") || contains(char, "people-pleaser") || contains(char, "perfectionist") {
					patternsSet[charLower] = true
				}
			}
		}
	}

	// Convert sets to slices
	for key := range mentionalHealthSet {
		vuln.MentalHealthIssues = append(vuln.MentalHealthIssues, key)
	}
	for key := range patternsSet {
		vuln.Patterns = append(vuln.Patterns, key)
	}

	// Extract communication style from AboutMe if available
	if ctx.AboutMe != nil {
		if ctx.AboutMe.CommunicationStyle != "" {
			vuln.CommunicationStyle = ctx.AboutMe.CommunicationStyle
		}
	}

	// Infer confidence level from patterns (rough heuristic)
	if len(vuln.Patterns) > 0 {
		vuln.Confidence = "low"
	} else if vuln.TraumaHistory || len(vuln.MentalHealthIssues) > 0 {
		vuln.Confidence = "medium"
	} else {
		vuln.Confidence = "high"
	}

	if vuln.TraumaHistory || len(vuln.MentalHealthIssues) > 0 || len(vuln.Patterns) > 0 {
		log.Printf("[ConversationAgent] ℹ️ User vulnerability: trauma=%v mental_health=%v patterns=%v confidence=%s",
			vuln.TraumaHistory, len(vuln.MentalHealthIssues) > 0, len(vuln.Patterns) > 0, vuln.Confidence)
		return vuln
	}

	return nil
}

// buildContactTraits extracts contact trait data from contact profile
func (ca *conversationAgent) buildContactTraits(contact *models.Contact) *tools.ContactTraits {
	if contact == nil {
		return nil
	}

	traits := &tools.ContactTraits{}

	// Extract relationship type from Notes if available
	if contact.Notes != "" {
		notesLower := strings.ToLower(contact.Notes)
		if strings.Contains(notesLower, "romantic") || strings.Contains(notesLower, "partner") || strings.Contains(notesLower, "spouse") {
			traits.Relationship = "romantic"
		} else if strings.Contains(notesLower, "friend") {
			traits.Relationship = "friend"
		} else if strings.Contains(notesLower, "family") || strings.Contains(notesLower, "parent") || strings.Contains(notesLower, "sibling") {
			traits.Relationship = "family"
		} else if strings.Contains(notesLower, "colleague") || strings.Contains(notesLower, "work") || strings.Contains(notesLower, "boss") {
			traits.Relationship = "professional"
		}
	}

	// Extract characteristics/traits from contact Characteristics field
	if contact.Characteristics != nil {
		traits.Traits = contact.Characteristics
		// Check sensitivity level from traits
		for _, char := range contact.Characteristics {
			charLower := strings.ToLower(char)
			if strings.Contains(charLower, "sensitive") || strings.Contains(charLower, "anxious") || strings.Contains(charLower, "quiet") {
				traits.Sensitivity = "high"
			} else if strings.Contains(charLower, "volatile") || strings.Contains(charLower, "reactive") {
				traits.Stability = "unstable"
			}
		}
	}

	// Default values if not detected
	if traits.Sensitivity == "" {
		traits.Sensitivity = "medium"
	}
	if traits.Stability == "" {
		traits.Stability = "stable"
	}

	if traits.Relationship != "" || len(traits.Traits) > 0 {
		log.Printf("[ConversationAgent] ℹ️ Contact traits: relationship=%s sensitivity=%s traits=%v",
			traits.Relationship, traits.Sensitivity, traits.Traits)
		return traits
	}

	return nil
}
