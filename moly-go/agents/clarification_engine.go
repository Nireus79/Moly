package agents

import (
	"fmt"
	"log"
	"strings"
	"time"

	"moly/schema"
)

// ClarificationResponse represents user's answer to a clarification question
type ClarificationResponse struct {
	QuestionID     string `json:"questionId"`
	UserResponse   string `json:"userResponse"`
	SelectedOption string `json:"selectedOption"`
	Timestamp      int64  `json:"timestamp"`
}

// ClarificationEngine generates and processes clarification questions
type ClarificationEngine struct {
	questionCounter int
}

// NewClarificationEngine creates a new clarification engine
func NewClarificationEngine() *ClarificationEngine {
	return &ClarificationEngine{
		questionCounter: 0,
	}
}

// GenerateSubjectClarification creates a question for ambiguous pronouns/names
// Called when SubjectAnalyzer returns clarity="ambiguous"
func (e *ClarificationEngine) GenerateSubjectClarification(fact ExtractedFact, subject string) *schema.ClarificationQuestion {
	e.questionCounter++

	log.Printf("[V2] ClarificationEngine: generating subject clarification for subject=%s", subject)

	question := &schema.ClarificationQuestion{
		ID:        fmt.Sprintf("q_subject_%d_%d", time.Now().Unix(), e.questionCounter),
		Type:      "subject_clarification",
		Priority:  1, // Critical - must know who we're talking about
		Status:    "pending",
		CreatedAt: time.Now().Unix(),
		LinkedFacts: []string{
			fact.ID,
		},
	}

	// Generate question based on pronoun/name
	if subject == "unknown_female" || subject == "she" {
		question.Question = "Who is she? (e.g., boss, manager, colleague, friend, family member)"
		question.Options = []string{
			"Boss / Manager",
			"Colleague / Coworker",
			"Friend",
			"Family member",
			"Other (I'll specify)",
		}
		question.Context = fmt.Sprintf("You said: \"%s\"\n\nWho is this person so I can understand the context better?", fact.Evidence)
	} else if subject == "unknown_male" || subject == "he" {
		question.Question = "Who is he? (e.g., boss, colleague, partner, friend, family member)"
		question.Options = []string{
			"Boss / Manager",
			"Colleague / Coworker",
			"Partner / Spouse",
			"Friend",
			"Family member",
			"Other (I'll specify)",
		}
		question.Context = fmt.Sprintf("You said: \"%s\"\n\nWho is this person?", fact.Evidence)
	} else if subject == "unknown_group" || subject == "they" {
		question.Question = "Who are they? (e.g., team, department, friends, family)"
		question.Options = []string{
			"Work team / Department",
			"Group of friends",
			"Family members",
			"Other group (I'll specify)",
		}
		question.Context = fmt.Sprintf("You said: \"%s\"\n\nWho is this group?", fact.Evidence)
	} else if subject == "unknown_thing" || subject == "it" {
		question.Question = "What are you referring to?"
		question.Context = fmt.Sprintf("You said: \"%s\"\n\nWhat is this?", fact.Evidence)
	} else if strings.HasPrefix(subject, "contact_pending_") {
		// Name without relationship context
		name := extractNameFromPending(subject)
		question.Question = fmt.Sprintf("What is %s's relationship to you?", name)
		question.Options = []string{
			"Boss / Manager",
			"Colleague / Coworker",
			"Friend",
			"Partner",
			"Family member",
			"Other",
		}
		question.Context = fmt.Sprintf("You mentioned: \"%s\"\n\nHow do you know %s?", fact.Evidence, name)
	} else {
		question.Question = fmt.Sprintf("Who are you referring to? (subject: %s)", subject)
		question.Context = fmt.Sprintf("You said: \"%s\"", fact.Evidence)
	}

	log.Printf("[V2]   Generated question: %s", question.Question)
	return question
}

// GenerateContactConfirmation creates a question asking if user wants to save a new contact
// Called when system identifies a new person to track
func (e *ClarificationEngine) GenerateContactConfirmation(name string, relationship string, discoveredTraits []string) *schema.ClarificationQuestion {
	e.questionCounter++

	log.Printf("[V2] ClarificationEngine: generating contact confirmation for %s (%s)", name, relationship)

	traitsList := ""
	if len(discoveredTraits) > 0 {
		traitsList = "From our conversation, I've noticed:\n"
		for _, trait := range discoveredTraits {
			traitsList += fmt.Sprintf("  • %s\n", trait)
		}
	}

	question := &schema.ClarificationQuestion{
		ID:       fmt.Sprintf("q_contact_%d_%d", time.Now().Unix(), e.questionCounter),
		Type:     "contact_confirmation",
		Priority: 2, // Important but not critical
		Status:   "pending",
		Context: fmt.Sprintf("%s\nShould I add %s to your contacts?", traitsList, name),
		Question: fmt.Sprintf("Should I save %s as your %s?", name, relationship),
		Options: []string{
			fmt.Sprintf("Yes, save %s", name),
			"Not yet, I'll add manually",
			"Save with different details",
		},
		CreatedAt: time.Now().Unix(),
	}

	log.Printf("[V2]   Generated confirmation: should save %s?", name)
	return question
}

// GenerateConflictClarification creates a question about contradictory statements
// Called when ConflictDetector finds a real conflict (same subject, different values)
func (e *ClarificationEngine) GenerateConflictClarification(
	subject string,
	previousValue string,
	currentValue string,
) *schema.ClarificationQuestion {
	e.questionCounter++

	log.Printf("[V2] ClarificationEngine: generating conflict clarification for %s: %s → %s", subject, previousValue, currentValue)

	question := &schema.ClarificationQuestion{
		ID:       fmt.Sprintf("q_conflict_%d_%d", time.Now().Unix(), e.questionCounter),
		Type:     "conflict_clarification",
		Priority: 1, // Critical - must resolve contradictions
		Status:   "pending",
		Question: fmt.Sprintf("You mentioned '%s' is %s, but now you say %s. Are these different contexts, or has something changed?",
			subject, previousValue, currentValue),
		Options: []string{
			"Both are true in different contexts (work vs. personal)",
			fmt.Sprintf("It changed: %s is now correct", currentValue),
			fmt.Sprintf("Keep previous: %s", previousValue),
		},
		Context: fmt.Sprintf("I want to make sure I understand correctly. " +
			"Are we talking about different situations, or has something changed?"),
		CreatedAt: time.Now().Unix(),
	}

	log.Printf("[V2]   Generated conflict resolution question")
	return question
}

// GenerateContextClarification asks if user is still talking about the same subject
// Called when SubjectShiftDetector detects a potential shift
func (e *ClarificationEngine) GenerateContextClarification(
	previousSubject string,
	shiftTrigger string,
) *schema.ClarificationQuestion {
	e.questionCounter++

	log.Printf("[V2] ClarificationEngine: generating context clarification, trigger=%s", shiftTrigger)

	question := &schema.ClarificationQuestion{
		ID:       fmt.Sprintf("q_context_%d_%d", time.Now().Unix(), e.questionCounter),
		Type:     "context_clarification",
		Priority: 2, // Important
		Status:   "pending",
		Question: fmt.Sprintf("Are you still talking about %s, or is this about someone/something else?", previousSubject),
		Options: []string{
			fmt.Sprintf("Still about %s", previousSubject),
			"About someone else",
			"About a different situation",
		},
		Context: fmt.Sprintf("I noticed the word '%s' which often signals a change in topic. " +
			"Just want to make sure I'm following along correctly.", shiftTrigger),
		CreatedAt: time.Now().Unix(),
	}

	log.Printf("[V2]   Generated context clarification question")
	return question
}

// GenerateUserContextClarification creates questions to establish context for facts about the user
// Called when fact is about the user but about_me profile doesn't exist
func (e *ClarificationEngine) GenerateUserContextClarification(fact ExtractedFact) []*schema.ClarificationQuestion {
	e.questionCounter++
	baseID := fmt.Sprintf("%d", time.Now().Unix()+int64(e.questionCounter))

	log.Printf("[V2] ClarificationEngine: generating user context clarification for trait=%s", fact.Value)

	questions := []*schema.ClarificationQuestion{}

	// Question 1: Context (when/where does this apply?)
	contextQ := &schema.ClarificationQuestion{
		ID:        fmt.Sprintf("user_ctx_context_%s", baseID),
		Type:      "user_context",
		Question:  fmt.Sprintf("You mentioned you're %s. When or where does this show up most?", fact.Value),
		Options:   []string{"Primarily at work", "Primarily in personal life", "In all situations", "Depends on the situation"},
		Priority:  1,
		Status:    "pending",
		Context:   fmt.Sprintf("Understanding context helps me know you better. You said: \"%s\"", fact.Evidence),
		CreatedAt: time.Now().Unix(),
		LinkedFacts: []string{fact.ID},
	}
	questions = append(questions, contextQ)

	// Question 2: Consistency (is this consistent?)
	consistencyQ := &schema.ClarificationQuestion{
		ID:        fmt.Sprintf("user_ctx_consistency_%s", baseID),
		Type:      "user_context",
		Question:  fmt.Sprintf("Is being %s consistent for you, or does it vary by situation?", fact.Value),
		Options:   []string{"Very consistent", "Mostly consistent", "Situation-dependent", "Varies a lot"},
		Priority:  2,
		Status:    "pending",
		Context:   "This helps me understand how reliable this trait is across different contexts.",
		CreatedAt: time.Now().Unix(),
		LinkedFacts: []string{fact.ID},
	}
	questions = append(questions, consistencyQ)

	// Question 3: Example/Evidence (tell me more)
	evidenceQ := &schema.ClarificationQuestion{
		ID:       fmt.Sprintf("user_ctx_evidence_%s", baseID),
		Type:     "user_context",
		Question: fmt.Sprintf("Can you tell me about a recent time you noticed you're %s?", fact.Value),
		Priority: 2,
		Status:   "pending",
		Context:  "Specific examples help me understand this better.",
		CreatedAt: time.Now().Unix(),
		LinkedFacts: []string{fact.ID},
	}
	questions = append(questions, evidenceQ)

	log.Printf("[V2]   Generated %d user context questions for trait=%s", len(questions), fact.Value)
	return questions
}

// GenerateContactContextClarification creates questions to establish contact relationship and context
// Called when fact is about a new contact that doesn't exist yet
func (e *ClarificationEngine) GenerateContactContextClarification(contactName string, fact ExtractedFact) []*schema.ClarificationQuestion {
	e.questionCounter++
	baseID := fmt.Sprintf("%d", time.Now().Unix()+int64(e.questionCounter))

	log.Printf("[V2] ClarificationEngine: generating contact context clarification for %s", contactName)

	questions := []*schema.ClarificationQuestion{}

	// Question 1: Relationship (who are they?)
	relationshipQ := &schema.ClarificationQuestion{
		ID:       fmt.Sprintf("contact_ctx_relationship_%s", baseID),
		Type:     "contact_context",
		Question: fmt.Sprintf("What's your relationship with %s?", contactName),
		Options:  []string{"Boss / Manager", "Colleague / Coworker", "Friend", "Family member", "Partner / Spouse", "Other"},
		Priority: 1,
		Status:   "pending",
		Context:  fmt.Sprintf("You mentioned %s. Understanding your relationship helps me keep better track.", contactName),
		CreatedAt: time.Now().Unix(),
		LinkedFacts: []string{fact.ID},
	}
	questions = append(questions, relationshipQ)

	// Question 2: Context (work/personal/both)
	contextQ := &schema.ClarificationQuestion{
		ID:        fmt.Sprintf("contact_ctx_interaction_%s", baseID),
		Type:      "contact_context",
		Question:  fmt.Sprintf("In what context do you interact with %s?", contactName),
		Options:   []string{"Primarily at work", "Primarily personal", "Both work and personal", "Varies"},
		Priority:  1,
		Status:    "pending",
		Context:   "Knowing where you interact with them helps me understand the relationship better.",
		CreatedAt: time.Now().Unix(),
		LinkedFacts: []string{fact.ID},
	}
	questions = append(questions, contextQ)

	// Question 3: Duration (how long known)
	durationQ := &schema.ClarificationQuestion{
		ID:       fmt.Sprintf("contact_ctx_duration_%s", baseID),
		Type:     "contact_context",
		Question: fmt.Sprintf("How long have you known %s?", contactName),
		Options:  []string{"Recent (less than a year)", "A few years", "Many years", "Just met"},
		Priority: 2,
		Status:   "pending",
		Context:  "This helps me understand the depth of your relationship.",
		CreatedAt: time.Now().Unix(),
		LinkedFacts: []string{fact.ID},
	}
	questions = append(questions, durationQ)

	log.Printf("[V2]   Generated %d contact context questions for %s", len(questions), contactName)
	return questions
}

// ProcessResponse handles user's answer to a clarification question
// Returns the resolved value or additional questions if needed
func (e *ClarificationEngine) ProcessResponse(response ClarificationResponse) (string, error) {
	log.Printf("[V2] ClarificationEngine: processing response to question %s", response.QuestionID)

	if response.SelectedOption != "" {
		log.Printf("[V2]   User selected option: %s", response.SelectedOption)
		return response.SelectedOption, nil
	}

	if response.UserResponse != "" {
		log.Printf("[V2]   User provided text response: %s", response.UserResponse)
		return response.UserResponse, nil
	}

	return "", fmt.Errorf("no response provided")
}

// ExtractClarifiedSubject converts a user response into a resolved subject
// E.g., "Boss / Manager" → "contact_boss"
func (e *ClarificationEngine) ExtractClarifiedSubject(response string) string {
	lower := strings.ToLower(response)

	// Map common responses to subjects
	if containsAny(lower, "boss", "manager", "director", "supervisor") {
		return "contact_boss"
	}
	if containsAny(lower, "colleague", "coworker", "teammate") {
		return "contact_colleague"
	}
	if containsAny(lower, "friend") {
		return "contact_friend"
	}
	if containsAny(lower, "partner", "spouse", "wife", "husband") {
		return "contact_partner"
	}
	if containsAny(lower, "family", "parent", "mother", "father", "sibling") {
		return "contact_family"
	}

	// If it's a free-text response with a name, return it
	if len(response) > 0 && response[0] >= 'A' && response[0] <= 'Z' {
		return fmt.Sprintf("contact_pending_%s", sanitizeName(response))
	}

	return "unknown"
}

// extractNameFromPending extracts the name from pending subject like "contact_pending_sarah"
func extractNameFromPending(subject string) string {
	if strings.HasPrefix(subject, "contact_pending_") {
		name := strings.TrimPrefix(subject, "contact_pending_")
		return strings.ReplaceAll(name, "_", " ")
	}
	return subject
}
