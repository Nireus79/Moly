package agents

import (
	"context"
	"fmt"
	"log"
	"moly/privacy"
	"os"
	"strings"

	"moly/models"
	"moly/tools"
)

// NOTE: ExtractedContext, ExtractedContact, ExtractedStyle types are now defined in models/agent_types.go
// This file uses the models.* versions for consistency

// ContextExtractor uses LLM-based semantic extraction via principle-based evaluation
type ContextExtractor struct {
	llmClient tools.LLMProvider
}

// NewContextExtractor creates a new context extractor with LLM-based semantic extraction
func NewContextExtractor(llmClient tools.LLMProvider) *ContextExtractor {
	return &ContextExtractor{
		llmClient: llmClient,
	}
}

// Extract analyzes a message and returns structured context with confidence scores
// Uses semantic extraction first, LLM fallback for low-confidence fields
func (ce *ContextExtractor) Extract(ctx context.Context, userMessage string) (*models.ExtractedContext, error) {
	return ce.ExtractWithExchange(ctx, userMessage, "")
}

// ExtractWithExchange extracts with the dialogue in view (a prompt block: the packet text, or an exchange written with
// models.WithExchange), so a short reply ("yes", a name) is read as the answer to Moly's last message and the people already
// known are not invented again. Only what the CURRENT message states or confirms is extracted.
func (ce *ContextExtractor) ExtractWithExchange(ctx context.Context, userMessage, block string) (*models.ExtractedContext, error) {
	if userMessage == "" {
		return &models.ExtractedContext{}, nil
	}

	log.Printf("[ContextExtractor] LLM-based semantic extraction: message_len=%d", len(userMessage))

	// STEP 1: LLM-based semantic extraction (principle-based evaluation, no pattern matching)
	// Uses buildExtractionPrompt() which asks LLM to extract in ONE call:
	// - Goal/intention (user's primary objective)
	// - Contacts (who they're discussing)
	// - Style/tone (how they communicate)
	// - Values (what matters to them)
	// - Characteristics (traits about user and contacts)
	// - Entities (all important concepts)

	prompt := ce.buildExtractionPrompt(userMessage, block)
	req := &tools.LLMRequest{
		UserPrompt:  prompt,
		Temperature: 0.3,
		MaxTokens:   2000,
		Retries:     2,
		JSONMode:    true,
	}
	llmResponse, err := ce.llmClient.Call(ctx, req)
	if err != nil {
		log.Printf("[ContextExtractor] ⚠️ LLM extraction failed: %v, using fallback", err)
		return ce.basicExtraction(userMessage), nil
	}

	// Parse LLM JSON response
	extracted, parseErr := ce.parseLLMExtraction(llmResponse.Content)
	if parseErr != nil {
		log.Printf("[ContextExtractor] ⚠️ Failed to parse LLM response: %v, using fallback", parseErr)
		return ce.basicExtraction(userMessage), nil
	}

	applyFactConfidence(extracted, userMessage)

	log.Printf("[ContextExtractor] ✅ LLM semantic extraction complete - Goal conf: %.2f, Contact: %v, Characteristics: %d",
		extracted.IntentionConfidence, extracted.Contact != nil, len(extracted.UserCharacteristics))

	// FIX #26 & #27: Validate all ExtractedContext fields against specification
	if extracted != nil {
		// Sanitize intention (FIX #25: prevent full message echo)
		extracted.Intention = tools.SanitizeIntention(extracted.Intention)

		// Validate IntentionConfidence (0-1 range)
		if extracted.IntentionConfidence < 0 || extracted.IntentionConfidence > 1 {
			log.Printf("[ContextExtractor] FIX #27: IntentionConfidence out of range (%.2f), clamping to [0,1]", extracted.IntentionConfidence)
			if extracted.IntentionConfidence < 0 {
				extracted.IntentionConfidence = 0
			} else if extracted.IntentionConfidence > 1 {
				extracted.IntentionConfidence = 1
			}
		}

		// Validate Goals array (should have 1-5 items if present)
		if len(extracted.Goals) > 5 {
			log.Printf("[ContextExtractor] FIX #27: Goals array too large (%d items), keeping first 5", len(extracted.Goals))
			extracted.Goals = extracted.Goals[:5]
		}

		// Validate each goal is reasonable length (1-50 chars)
		for i, goal := range extracted.Goals {
			if len(goal) > 50 {
				log.Printf("[ContextExtractor] FIX #27: Goal %d too long (%d chars), truncating", i, len(goal))
				extracted.Goals[i] = goal[:50]
			}
		}

		// Ensure arrays aren't nil (FIX #27: prevent nil dereference)
		if extracted.Goals == nil {
			extracted.Goals = []string{}
		}
		if extracted.UserValues == nil {
			extracted.UserValues = []string{}
		}
		if extracted.UserCharacteristics == nil {
			extracted.UserCharacteristics = []string{}
		}
		if extracted.IntentionPrinciples == nil {
			extracted.IntentionPrinciples = []string{}
		}
		if extracted.ContactCharacteristics == nil {
			extracted.ContactCharacteristics = make(map[string][]string)
		}
	}

	log.Printf("[ContextExtractor] Successfully extracted context (intention=%q, goals=%d, confidence=%.2f)",
		extracted.Intention, len(extracted.Goals), extracted.IntentionConfidence)
	return extracted, nil
}

func (ce *ContextExtractor) buildExtractionPrompt(userMessage, block string) string {
	// The short prompt is the default (27/30 against 20/30 on Haiku, and cheaper); MOLY_EXTRACT_PROMPT=long selects the old one.
	if os.Getenv("MOLY_EXTRACT_PROMPT") != "long" {
		return shortExtractionPrompt(userMessage, block)
	}
	if block != "" {
		// The live model, given the exchange, filled the goal and the person of a bare "Yes" from the exchange.
		block += "The dialogue is ONLY for understanding the current message. Fill intention, contact, traits and every other field ONLY from the words of the CURRENT message. " +
			"A reply such as \"yes\", \"no\" or \"ok\" states no goal and points to no person: then intention is empty and contact is null. A reply that gives a name or a detail about a person is about the person Moly was talking about: extract that name and that detail.\n\n"
	}
	return block + fmt.Sprintf(`You are Μώλυ (also called Moly in English), an AI thinking partner. The user is talking TO you.

Your task: Extract structured information about OTHER PEOPLE the user wants to discuss or get advice about.

SELF-AWARENESS:
YOU are Μώλυ/Moly - both names (Greek and English) refer to YOU (the AI system).
References to "Moly", "Μώλυ", "you" (addressing you), "I"/"me" (the user), are about the conversation happening between you and the user - NOT about a contact to discuss.

A CONTACT is a THIRD PERSON the user wants advice/help with.
Extract ONLY contacts that are separate people (not Moly/Μώλυ, not self-references).

Message: "%s"

Extract and return JSON with:
- contact: {name, label, relationship (romantic|professional|family|friend|other), traits[], confidence (0-1), evidence, basis} [OMIT if user is addressing you or discussing themselves]
- otherContacts: a list of further people in the same shape, when the message mentions more than one person. contact is the person the message is mostly about. [] when there is only one.
  name: the person's real name ONLY if the user gave one in this message. Otherwise name is an empty string.
  evidence: the exact words of the message that show this person.
  basis: "stated" if the user said it in so many words, "implied" if it follows from what they said, "guessed" if you are filling in something they did not say.
  label: the user's own words for the person, copied from this message. Never copy label into name. If this message does not mention a third person, omit contact entirely: never take a person from these instructions.
- intention: main goal/purpose (1-2 sentences capturing the FULL semantic goal. Include what they're trying to accomplish, who it involves, and what constraints matter. Leave it EMPTY (empty string) when the message is only a greeting, small talk, or a question about Moly itself with no goal of the user's. Also leave it EMPTY when the message only comments on, corrects or asks to adjust something Moly just wrote (a request to change its length or tone, say): that continues the current goal and is not a new one. Also leave it EMPTY when the message only agrees or disagrees (a bare yes or no) or only answers a question that was just asked, for example giving a name or a detail: an answer is not a new goal, and you must not invent one such as identifying a person.)
- intentionBasis: "stated", "implied" or "guessed", for the intention, as for the contact basis
- intentionEvidence: the exact words of the message that state the goal, copied from it (a few words are enough). Empty when the intention is empty. If you cannot point to words of the message that state a goal, the intention must be empty.
- intentionPrinciples: constitutional principles engaged by this intention (select from: transparency, autonomy, empathy, fairness, growth, stakeholder)
  * transparency: communicating honestly/openly with others
  * autonomy: making own choices, standing up for self, independence
  * empathy: understanding/considering others' perspectives and needs
  * fairness: equity, just treatment, reciprocity in relationships
  * growth: learning, self-improvement, developing capabilities
  * stakeholder: considering impact on others, multiple perspectives
- userCharacteristics: [traits about the USER/person writing] - tag subject explicitly (FIX #5) - keep each trait 1-3 words
- contactCharacteristics: {[contact_name]: [traits about this contact]} - tag with actual contact name (FIX #5) - keep each trait 1-3 words
- systemFeedback: {isFeedback, feedbackType (positive|negative|directive|perception), feedback[], directives[], perceptions[], style, confidence (0-1), evidence}
  When user addresses Moly (says "you", "Moly", gives feedback about the system):
  * feedback: "too verbose", "helpful", "confusing", "clear"
  * directives: "be more concise", "ask questions", "stop asking about family"
  * perceptions: "good at analysis", "lacks empathy", "can't do legal"
  * style: ONLY if user gives directive about interaction style: "direct", "socratic", "collaborative"
  [OMIT if message is not about the system]
- entities: [{name, type (topic|goal_component|concern|context), confidence (0-1), evidence}] - COMBINED EXTRACTION (FIX #6)
  Extract ALL important entities/concepts from the message, not just contact info.
  This includes: topics discussed, goals mentioned, concerns raised, key concepts.

CRITICAL - SUBJECT TAGGING FOR ALL CHARACTERISTICS
When extracting any characteristic, preference, or value, ALWAYS tag the subject:
- If about USER (the person writing): tag as "USER|trait|confidence"
- If about CONTACT: tag as "CONTACT_[actual_name]|trait|confidence"

NEVER create undefined contacts:
- Only extract contact characteristics if clearly linked to a contact name
- Always reference the actual contact name from extracted contact object
- If no contact name known, don't extract contact characteristics

Only include fields that are clearly evident. If field is not mentioned, omit it.
Confidence should reflect how certain you are based on explicit mentions.
Evidence should be a quote or reference from the message.

Return ONLY valid JSON, no other text.`, userMessage)
}

// basicExtraction provides fallback extraction without LLM
// Returns safe defaults without any keyword matching
// Reasoning: No data is safer than wrong data from hardcoded keywords
// User will get asked clarifying questions naturally in conversation flow
func (ce *ContextExtractor) basicExtraction(userMessage string) *models.ExtractedContext {
	log.Printf("[ContextExtractor] Using safe fallback extraction (no keyword matching)")

	// Return safe defaults - all fields empty/nil (not extracted)
	// This forces clarification questions in the normal workflow
	extracted := &models.ExtractedContext{}
	extracted.IntentionPrinciples = []string{} // No principles detected

	return extracted
}

// parseLLMExtraction parses the JSON response from LLM extraction into ExtractedContext
func (ce *ContextExtractor) parseLLMExtraction(llmJSON string) (*models.ExtractedContext, error) {
	// Decode JSON response from LLM
	var response map[string]interface{}
	// SafeJSONParse also reads JSON that the model wrapped in a code fence or a sentence. Found in a dynamic run on Claude: a fenced
	// answer failed here, the fallback returned an empty extraction, and the message was understood as saying nothing.
	if err := tools.SafeJSONParse("ContextExtractor", []byte(llmJSON), &response); err != nil {
		return nil, fmt.Errorf("failed to parse LLM JSON response: %w", err)
	}

	extracted := &models.ExtractedContext{}

	// Extract intention
	if intention, ok := response["intention"].(string); ok && intention != "" {
		privacy.Register(intention) // before anything logs it
		extracted.Intention = intention
		if basis, ok := response["intentionBasis"].(string); ok {
			extracted.IntentionBasis = strings.ToLower(strings.TrimSpace(basis))
		}
		if evidence, ok := response["intentionEvidence"].(string); ok {
			extracted.IntentionEvidence = strings.TrimSpace(evidence)
		}
	}

	// Extract intentionPrinciples
	if principles, ok := response["intentionPrinciples"].([]interface{}); ok {
		for _, p := range principles {
			if prin, ok := p.(string); ok {
				extracted.IntentionPrinciples = append(extracted.IntentionPrinciples, prin)
			}
		}
	}

	// Extract goals
	if goals, ok := response["goals"].([]interface{}); ok {
		for _, g := range goals {
			if goal, ok := g.(string); ok {
				extracted.Goals = append(extracted.Goals, goal)
			}
		}
	}

	// Extract the secrets the user typed (they are cut out of the message before anything is stored)
	if secrets, ok := response["sensitive"].([]interface{}); ok {
		for _, v := range secrets {
			if secret, ok := v.(string); ok && strings.TrimSpace(secret) != "" {
				extracted.Sensitive = append(extracted.Sensitive, secret)
			}
		}
	}

	// Extract userCharacteristics
	if chars, ok := response["userCharacteristics"].([]interface{}); ok {
		for _, c := range chars {
			if char, ok := c.(string); ok {
				extracted.UserCharacteristics = append(extracted.UserCharacteristics, char)
			}
		}
	}

	// Extract contact, and any further people the message mentions
	if contactObj, ok := response["contact"].(map[string]interface{}); ok {
		if contact := parseContactObject(contactObj); contact != nil {
			extracted.Contact = contact
		}
	}
	if others, ok := response["otherContacts"].([]interface{}); ok {
		for _, o := range others {
			obj, ok := o.(map[string]interface{})
			if !ok {
				continue
			}
			contact := parseContactObject(obj)
			if contact == nil || (extracted.Contact != nil && sameExtractedPerson(extracted.Contact, contact)) {
				continue
			}
			if extracted.Contact == nil {
				extracted.Contact = contact
				continue
			}
			extracted.OtherContacts = append(extracted.OtherContacts, contact)
		}
	}

	// DISCONNECTED: Style and UserValues extraction
	// These are communication preferences (config), not extracted data
	// Intentionally omitted from extraction to avoid false inference

	// Extract contactCharacteristics
	if contactChars, ok := response["contactCharacteristics"].(map[string]interface{}); ok {
		extracted.ContactCharacteristics = make(map[string][]string)
		for contactName, chars := range contactChars {
			if charSlice, ok := chars.([]interface{}); ok {
				for _, c := range charSlice {
					if char, ok := c.(string); ok {
						extracted.ContactCharacteristics[contactName] = append(extracted.ContactCharacteristics[contactName], char)
					}
				}
			}
		}
	}

	// Extract systemFeedback (when user gives feedback about Moly)
	if sysFeedback, ok := response["systemFeedback"].(map[string]interface{}); ok {
		feedback := &models.SystemFeedbackInfo{}
		if isFb, ok := sysFeedback["isFeedback"].(bool); ok {
			feedback.IsFeedback = isFb
		}
		if fbType, ok := sysFeedback["feedbackType"].(string); ok {
			feedback.FeedbackType = fbType
		}
		if feedbackArr, ok := sysFeedback["feedback"].([]interface{}); ok {
			for _, f := range feedbackArr {
				if fb, ok := f.(string); ok {
					feedback.Feedback = append(feedback.Feedback, fb)
				}
			}
		}
		if directivesArr, ok := sysFeedback["directives"].([]interface{}); ok {
			for _, d := range directivesArr {
				if dir, ok := d.(string); ok {
					feedback.Directives = append(feedback.Directives, dir)
				}
			}
		}
		if perceptionsArr, ok := sysFeedback["perceptions"].([]interface{}); ok {
			for _, p := range perceptionsArr {
				if perc, ok := p.(string); ok {
					feedback.Perceptions = append(feedback.Perceptions, perc)
				}
			}
		}
		if style, ok := sysFeedback["style"].(string); ok {
			feedback.Style = style
		}
		if conf, ok := sysFeedback["confidence"].(float64); ok {
			feedback.Confidence = conf
		}
		if evid, ok := sysFeedback["evidence"].(string); ok {
			feedback.Evidence = evid
		}

		if feedback.IsFeedback || len(feedback.Feedback) > 0 || len(feedback.Directives) > 0 || len(feedback.Perceptions) > 0 {
			extracted.SystemFeedback = feedback
		}
	}

	// Ensure all arrays are initialized (not nil)
	if extracted.Goals == nil {
		extracted.Goals = []string{}
	}
	if extracted.UserValues == nil {
		extracted.UserValues = []string{}
	}
	if extracted.UserCharacteristics == nil {
		extracted.UserCharacteristics = []string{}
	}
	if extracted.IntentionPrinciples == nil {
		extracted.IntentionPrinciples = []string{}
	}
	if extracted.ContactCharacteristics == nil {
		extracted.ContactCharacteristics = make(map[string][]string)
	}

	return extracted, nil
}

// applyFactConfidence works out the confidence of each saved fact (see fact_confidence.go). It replaces the old
// constants (an intention was always 0.90, a contact without a number was 0.85).
func applyFactConfidence(ec *models.ExtractedContext, message string) {
	if ec == nil {
		return
	}
	if ec.Intention != "" {
		// A goal is only as sure as its quote: the exact words that state it must be in the message. Without them it is
		// capped as a guess, held and confirmed, never locked (a goal was invented from messages that state none).
		grounded := QuoteInMessage(message, ec.IntentionEvidence)
		ec.IntentionConfidence = FactConfidence(1, ec.IntentionBasis, grounded)
		log.Printf("[FactConfidence] goal: basis=%q grounded=%v -> %.2f (in doubt: %v)", ec.IntentionBasis, grounded, ec.IntentionConfidence, FactInDoubt(ec.IntentionConfidence))
	} else {
		ec.IntentionConfidence = 0
	}
	for _, c := range ec.People() {
		grounded := QuoteInMessage(message, c.Evidence) || QuoteInMessage(message, c.Label) || (c.NameKnown && QuoteInMessage(message, c.Name))
		model := c.Confidence
		c.Confidence = FactConfidence(model, c.Basis, grounded)
		log.Printf("[FactConfidence] person: model=%.2f basis=%q grounded=%v -> %.2f (in doubt: %v)", model, c.Basis, grounded, c.Confidence, FactInDoubt(c.Confidence))
	}
}

// shortExtractionPrompt is the extraction prompt in its short form: a JSON skeleton with every key present, two rules and two
// examples. On Claude haiku the long prompt filled the intention and the person right in 20 of 30 replays (0 of 3 for "I need to
// write to my manager"), this one in 27 of 30.
func shortExtractionPrompt(msg, block string) string {
	return block + `You read one message that a user wrote to Moly, a communication coach, and you fill in a JSON form about it.
Use only the words of the message. Return ONLY the JSON object, with every key present.

Message: "` + msg + `"

{
  "intention": "what the user wants to do or get help with, in one sentence; \"\" when the message states none",
  "intentionEvidence": "the exact words of the message that state it; \"\" when intention is \"\"",
  "intentionBasis": "stated | implied | guessed",
  "intentionPrinciples": ["from: transparency, autonomy, empathy, fairness, growth, stakeholder"],
  "contact": null or {"name": "the person's real name if the message gives one, else \"\"", "label": "the user's words for the person, without my/your/the", "role": "what the person is to the user, in the user's words, else \"\"", "relationship": "romantic | professional | family | friend | other", "traits": [], "confidence": 0.0 to 1.0, "evidence": "the exact words of the message that show this person", "basis": "stated | implied | guessed"},
  "otherContacts": [] or the same shape for each further person the message mentions,
  "userCharacteristics": ["traits of the user that the message states, 1-3 words each"],
  "contactCharacteristics": {"person name or label": ["traits the message states"]},
  "sensitive": ["secrets the user typed in this message, copied exactly as written: passwords, PINs, security codes, card or bank account numbers, ID or passport or tax numbers, API keys, recovery phrases"],
  "systemFeedback": null
}

Rules:
- sensitive lists only real secrets that appear in the message, exactly as written; [] when there are none. A name, an address or a date is not listed.
- The intention is a goal the user states in this message. A greeting, a bare yes or no, a name or a detail given in answer to Moly's question, and a comment on what Moly just wrote (a request to change its length or tone) state no goal: intention is "".
- contact is a third person the message mentions or points to (not the user, not Moly). An organization or institution of any kind is not a person: it is not a contact. If the message mentions none, contact is null. When it mentions more than one person, contact is the one it is mostly about and every other person goes in otherContacts. A reply that gives a name or a detail about a person Moly asked about is about that person.

Examples:
Message: "Help me tell my sister I can't come to her party." -> {"intention": "tell my sister I can't come to her party", "intentionEvidence": "tell my sister I can't come to her party", "intentionBasis": "stated", "intentionPrinciples": ["transparency"], "contact": {"name": "", "label": "sister", "relationship": "family", "traits": [], "confidence": 0.95, "evidence": "my sister", "basis": "stated"}, "userCharacteristics": [], "contactCharacteristics": {}, "systemFeedback": null}
Message: "Yes" -> {"intention": "", "intentionEvidence": "", "intentionBasis": "stated", "intentionPrinciples": [], "contact": null, "userCharacteristics": [], "contactCharacteristics": {}, "systemFeedback": null}
`
}

// bareLabel is the user's words for a person without the possessive or article in front ("my manager" -> "manager"), so the name
// question reads "a name for the manager you mentioned" and the role reads "the user's manager" (found in a dynamic run:
// "for the my boss" and "the user's my boss").
func bareLabel(label string) string {
	l := strings.TrimSpace(label)
	lower := strings.ToLower(l)
	for _, p := range []string{"my ", "your ", "our ", "his ", "her ", "their ", "the ", "a ", "an "} {
		if strings.HasPrefix(lower, p) {
			return strings.TrimSpace(l[len(p):])
		}
	}
	return l
}

// parseContactObject reads one person from the model's JSON. It returns nil when the object names no one.
func parseContactObject(contactObj map[string]interface{}) *models.ExtractedContact {
	contact := &models.ExtractedContact{}
	if name, ok := contactObj["name"].(string); ok {
		contact.Name = strings.TrimSpace(name)
		privacy.Register(contact.Name)
	}
	// PHASE 5: a contact without a given name keeps the user's own words as a label, and is not treated as named
	if label, ok := contactObj["label"].(string); ok && strings.TrimSpace(label) != "" {
		contact.Label = bareLabel(label)
	}
	if role, ok := contactObj["role"].(string); ok {
		contact.Role = bareLabel(role)
	}
	contact.NameKnown = contact.Name != ""
	if contact.Name == "" && contact.Label != "" {
		contact.Name = contact.Label // stored under the label until the user gives a name
	}
	if rel, ok := contactObj["relationship"].(string); ok {
		contact.Relationship = rel
	}
	// The model's own number, or 0 when it gave none. The confidence the rest of Moly uses is worked out per fact
	// in applyFactConfidence; a missing number is never read as high.
	if conf, ok := contactObj["confidence"].(float64); ok {
		contact.Confidence = conf
	}
	if ev, ok := contactObj["evidence"].(string); ok {
		contact.Evidence = strings.TrimSpace(ev)
	}
	if basis, ok := contactObj["basis"].(string); ok {
		contact.Basis = strings.ToLower(strings.TrimSpace(basis))
	}
	if traits, ok := contactObj["traits"].([]interface{}); ok {
		for _, t := range traits {
			if trait, ok := t.(string); ok {
				contact.Traits = append(contact.Traits, trait)
			}
		}
	}
	if contact.Name == "" {
		return nil
	}
	return contact
}

// sameExtractedPerson is true when two entries are the same person written twice (same name, ignoring case).
func sameExtractedPerson(a, b *models.ExtractedContact) bool {
	return strings.EqualFold(strings.TrimSpace(a.Name), strings.TrimSpace(b.Name))
}
