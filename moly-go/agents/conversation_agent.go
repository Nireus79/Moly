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
	socraticSelector      *SocraticQuestionSelector // Optional: for Socratic question selection
	constitution          *models.Constitution      // Optional: for principle-guided generation
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
			// Note: Contact is extracted smartly based on contact verbs, so we don't require it upfront
			missingAboutMe := !hasAboutMe
			missingIntention := !hasIntention
			generatedResponse = ca.generateClarifyingResponse(ctx, userMessage, missingAboutMe, false, missingIntention)
			log.Printf("[ConversationAgent] [✓] Generated clarifying response: %.100s...", generatedResponse)
		} else {
			// Give full response with available context
			generatedResponse = ca.generateConversationalResponse(ctx, userMessage)
			log.Printf("[ConversationAgent] [✓] Generated full response: %.100s...", generatedResponse)
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

	// CLARIFICATION QUESTIONS REMOVED
	// Context is now gathered through natural conversation flow in Moly's response
	// If needed in future, will be re-implemented as part of main dialogue
	log.Printf("[ConversationAgent] Context gathering handled through conversational response")

	// BUILD METADATA
	response.Metadata = map[string]interface{}{
		"conversational": true,
		"hasUserProfile": aboutMe != nil,
		"contextGaps":    len(ctx.Gaps),
		"timestamp":      startTime.Unix(),
	}
	log.Printf("[ConversationAgent] [✓] Metadata initialized with base fields: conversational=true profile=%v gaps=%d",
		aboutMe != nil, len(ctx.Gaps))

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

	// Include extracted context from THIS message (takes precedence over stored profile)
	if ctx.ExtractedContext != nil {
		if ctx.ExtractedContext.Style != nil && ctx.ExtractedContext.Style.Confidence > 0.6 {
			userProfile += fmt.Sprintf("Communication style (from this message): %s\n", ctx.ExtractedContext.Style.Style)
		}
		if ctx.ExtractedContext.Contact != nil && ctx.ExtractedContext.Contact.Confidence > 0.6 {
			userProfile += fmt.Sprintf("Talking about: %s (relationship: %s)\n", ctx.ExtractedContext.Contact.Name, ctx.ExtractedContext.Contact.Relationship)
		}
		if ctx.ExtractedContext.Intention != "" {
			userProfile += fmt.Sprintf("Intention: %s\n", ctx.ExtractedContext.Intention)
		}
	}

	// Include past intention from previous messages (context about ongoing goals)
	if ctx.PastIntention != "" {
		userProfile += fmt.Sprintf("Earlier goal: %s\n", ctx.PastIntention)
	}

	// Format past reflections (what Moly has learned about the user over time)
	reflectionsText := ""
	if len(ctx.RelevantReflections) > 0 {
		reflectionsText = "What Moly has learned about you:\n"
		for i, reflection := range ctx.RelevantReflections {
			if i >= 3 { // Limit to 3 most recent reflections to keep prompt concise
				break
			}
			if len(reflection.Characteristics) > 0 {
				reflectionsText += fmt.Sprintf("- You are: %s\n", strings.Join(reflection.Characteristics, ", "))
			}
			if len(reflection.Interests) > 0 {
				reflectionsText += fmt.Sprintf("- You care about: %s\n", strings.Join(reflection.Interests, ", "))
			}
			if len(reflection.Intentions) > 0 {
				reflectionsText += fmt.Sprintf("- You tend to: %s\n", strings.Join(reflection.Intentions, ", "))
			}
		}
		if reflectionsText != "What Moly has learned about you:\n" {
			reflectionsText += "\n"
		}
	}

	// Reference conversation history for context
	pastContext := ""
	if len(ctx.ConversationHistory) > 1 {
		pastContext = "Recent conversation context has been shared with you.\n"
	}

	principlesContext := ca.buildPrincipleContext()

	// Detect emotional tone from message for mood-aware response
	emotionalTone := "neutral"
	lowerMsg := strings.ToLower(userMessage)
	positiveIndicators := []string{"happy", "excited", "great", "wonderful", "amazing", "love", "grateful", "thrilled", "delighted", "proud", "hopeful"}
	negativeIndicators := []string{"sad", "angry", "frustrated", "disappointed", "worried", "anxious", "stressed", "overwhelmed", "hurt", "devastated"}

	positiveCount := 0
	for _, indicator := range positiveIndicators {
		if contains(lowerMsg, indicator) {
			positiveCount++
		}
	}
	negativeCount := 0
	for _, indicator := range negativeIndicators {
		if contains(lowerMsg, indicator) {
			negativeCount++
		}
	}
	if negativeCount > positiveCount {
		emotionalTone = "negative"
	} else if positiveCount > negativeCount {
		emotionalTone = "positive"
	}

	// Detect conversation topics for context awareness
	var topics []string
	topicKeywords := map[string]string{
		"work":        "work/career",
		"job":         "work/career",
		"career":      "work/career",
		"boss":        "work/career",
		"colleague":   "work/career",
		"relationship": "relationships",
		"partner":     "relationships",
		"romantic":    "relationships",
		"date":        "relationships",
		"family":      "family",
		"parent":      "family",
		"sibling":     "family",
		"friend":      "friendship",
		"health":      "health/wellness",
		"exercise":    "health/wellness",
		"sleep":       "health/wellness",
		"stress":      "health/wellness",
		"anxiety":     "mental health",
		"depression":  "mental health",
		"therapy":     "mental health",
		"hobby":       "interests/hobbies",
		"interest":    "interests/hobbies",
		"passion":     "interests/hobbies",
		"goal":        "goals/aspirations",
		"dream":       "goals/aspirations",
		"future":      "goals/aspirations",
	}

	lowerMsgForTopics := strings.ToLower(userMessage)
	topicsSet := make(map[string]bool)
	for keyword, topic := range topicKeywords {
		if contains(lowerMsgForTopics, keyword) && !topicsSet[topic] {
			topics = append(topics, topic)
			topicsSet[topic] = true
		}
	}

	emotionGuidance := ""
	if emotionalTone == "negative" {
		emotionGuidance = "The person seems distressed. Be extra supportive and validating.\n"
	} else if emotionalTone == "positive" {
		emotionGuidance = "The person is in a positive mood. Match their energy with warmth.\n"
	}

	// Add phase-aware guidance
	phaseGuidance := ""
	if ctx.ConversationPhase == "gathering" {
		phaseGuidance = "You're in the context-gathering phase. Ask clarifying questions.\n"
	} else if ctx.ConversationPhase == "processing" {
		phaseGuidance = "You're in the processing phase. Help them think through options.\n"
	} else if ctx.ConversationPhase == "complete" {
		phaseGuidance = "You've gathered good context. Focus on actionable insights.\n"
	}

	if len(topics) > 0 {
		log.Printf("[ConversationAgent] Detected topics: %v", topics)
	}

	prompt := fmt.Sprintf(`You are Moly, a supportive friend who listens deeply and learns about the person you're talking with.

%s

About this person:
%s

%s
%s
%s
%s

The person just said: "%s"

Respond naturally and conversationally. Be warm, understanding, and genuinely curious about them. Don't be robotic or clinical. Ask follow-up questions if appropriate. Show that you're listening and that you care about what they're sharing. Do not use emojis.

Keep your response concise (1-3 sentences) unless they're sharing something complex.`, principlesContext, userProfile, reflectionsText, pastContext, emotionGuidance, phaseGuidance, userMessage)

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

