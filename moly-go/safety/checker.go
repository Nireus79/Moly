package safety

import (
	"context"
	"log"
	"strings"

	"moly/tools"
)

type AlertSeverity string

const (
	ALERT_SEVERITY_IMMEDIATE AlertSeverity = "immediate"
	ALERT_SEVERITY_HIGH      AlertSeverity = "high"
	ALERT_SEVERITY_WARNING   AlertSeverity = "warning"
)

type AlertType string

const (
	ALERT_CRISIS  AlertType = "crisis"
	ALERT_ILLEGAL AlertType = "illegal"
	ALERT_NONE    AlertType = "none"
)

type SafetyAlert struct {
	AlertType       AlertType        `json:"alert_type"`
	Severity        AlertSeverity    `json:"severity"`
	Title           string           `json:"title"`
	Message         string           `json:"message"`
	Indicators      []string         `json:"indicators"`
	Resources       []CrisisResource `json:"resources"`
	Recommendations []string         `json:"recommendations"`
}

type CrisisResource struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Number      string `json:"number"`
	URL         string `json:"url"`
	Region      string `json:"region"`
}

type Checker struct {
	llmClient tools.LLMProvider // Use LLM for intelligent detection
}

func NewChecker() *Checker {
	return &Checker{
		llmClient: nil,
	}
}

func NewCheckerWithLLM(llm tools.LLMProvider) *Checker {
	return &Checker{
		llmClient: llm,
	}
}

// CheckMessage performs context-aware safety check on user messages
// Strategy: Only block OBVIOUS crises (specific keywords).
// Ambiguous cases are handled by ConversationAgent + clarification questions + HarmAnalyzer.
// This prevents false positives while maintaining safety.
func (sc *Checker) CheckMessage(text string) *SafetyAlert {
	if text == "" {
		return nil
	}

	text = strings.TrimSpace(text)

	// ONLY check for obvious crisis keywords (high confidence only)
	// Ambiguous cases should NOT trigger early blocking
	// Instead: let agent run → ask clarification questions → check response
	return sc.detectObviousCrisisOnly(text)
}

// detectCrisisLLMImproved uses a more calibrated LLM prompt
// Only flags ACTUAL crises, not general concerns
func (sc *Checker) detectCrisisLLMImproved(message string) *SafetyAlert {
	if sc.llmClient == nil {
		return sc.detectCrisisHeuristic(message)
	}

	req := &tools.LLMRequest{
		SystemPrompt: `You are a mental health safety expert. Classify messages into these EXACT categories only:

ONLY respond with ONE of these words:
- "crisis" IF the person explicitly says they want to hurt/kill themselves or others (imminent danger)
- "illegal" IF they're asking how to do something illegal (drugs, weapons, fraud)
- "safe" for EVERYTHING ELSE (including sadness, breakup, depression talk, seeking help)

Examples:
- "I want to kill myself" → crisis
- "I'm having suicidal thoughts" → crisis
- "I want to hurt my ex" → crisis
- "How do I make a bomb?" → illegal
- "I'm sad about my girlfriend" → safe
- "My friend is depressed, what should I say?" → safe
- "I need help" → safe
- "I want to talk about a girl" → safe`,
		UserPrompt:  "Classify: " + message,
		MaxTokens:   10,
		Temperature: 0.1,
		Retries:     1,
	}

	resp, err := sc.llmClient.Call(context.Background(), req)
	if err != nil {
		log.Printf("[SafetyChecker] LLM error: %v, falling back to heuristic", err)
		return sc.detectCrisisHeuristic(message)
	}

	lower := strings.ToLower(strings.TrimSpace(resp.Content))
	log.Printf("[SafetyChecker] LLM response to '%s': '%s'", message, lower)

	// Only accept exact matches or prefix
	if lower == "crisis" || strings.HasPrefix(lower, "crisis ") {
		return &SafetyAlert{
			AlertType:       ALERT_CRISIS,
			Severity:        ALERT_SEVERITY_IMMEDIATE,
			Title:           "Crisis Support Available",
			Message:         "I detected language suggesting you or someone else might be in crisis. Your safety matters. You're not alone.",
			Resources:       getCrisisResources(),
			Recommendations: getCrisisRecommendations(),
		}
	}

	if lower == "illegal" || strings.HasPrefix(lower, "illegal ") {
		return &SafetyAlert{
			AlertType:       ALERT_ILLEGAL,
			Severity:        ALERT_SEVERITY_HIGH,
			Title:           "Cannot Assist",
			Message:         "I cannot help with illegal activities. Please consult with appropriate professionals.",
			Resources:       []CrisisResource{},
			Recommendations: getIllegalRecommendations(),
		}
	}

	return nil
}

// detectCrisisLLM uses LLM with ethical frameworks to detect crisis
func (sc *Checker) detectCrisisLLM(message string) *SafetyAlert {
	if sc.llmClient == nil {
		// Fallback to heuristic if no LLM available
		return sc.detectCrisisHeuristic(message)
	}

	req := &tools.LLMRequest{
		SystemPrompt: `You are a mental health safety expert. Analyze messages for safety using logical frameworks:
1. Does this indicate self-harm, suicide, or crisis?
2. Does this describe illegal activity?
3. Is this a threat to others?

Respond with ONLY one word:
- "crisis" if self-harm/suicide/mental health emergency
- "illegal" if illegal activity
- "safe" if normal conversation`,
		UserPrompt:  "Analyze: " + message,
		MaxTokens:   20,
		Temperature: 0.1,
		Retries:     1,
	}

	resp, err := sc.llmClient.Call(context.Background(), req)
	if err != nil {
		log.Printf("[SafetyChecker] LLM error, falling back to heuristic: %v", err)
		return sc.detectCrisisHeuristic(message)
	}

	lower := strings.ToLower(strings.TrimSpace(resp.Content))
	log.Printf("[SafetyChecker] LLM response to '%s': '%s'", message, lower)

	// Check if response starts with "crisis" or "illegal" (not just contains)
	// This prevents false positives from LLM explaining why something is NOT a crisis
	if lower == "crisis" || strings.HasPrefix(lower, "crisis ") {
		return &SafetyAlert{
			AlertType:       ALERT_CRISIS,
			Severity:        ALERT_SEVERITY_IMMEDIATE,
			Title:           "Crisis Support Available",
			Message:         "I detected language suggesting you or someone else might be in crisis. Your safety matters. You're not alone.",
			Resources:       getCrisisResources(),
			Recommendations: getCrisisRecommendations(),
		}
	}

	if lower == "illegal" || strings.HasPrefix(lower, "illegal ") {
		return &SafetyAlert{
			AlertType:       ALERT_ILLEGAL,
			Severity:        ALERT_SEVERITY_HIGH,
			Title:           "Cannot Assist",
			Message:         "I cannot help with illegal activities. Please consult with appropriate professionals.",
			Resources:       []CrisisResource{},
			Recommendations: getIllegalRecommendations(),
		}
	}

	return nil
}

// detectObviousCrisisOnly checks ONLY for obvious, high-confidence crisis keywords
// Does NOT use LLM or ambiguous heuristics
// Ambiguous cases (sad, depressed, problems) are handled by agent + clarification
func (sc *Checker) detectObviousCrisisOnly(message string) *SafetyAlert {
	lower := strings.ToLower(message)

	// OBVIOUS crisis indicators - specific, high-confidence keywords only
	// Avoid false positives by being very specific
	obviousCrisisIndicators := []string{
		"kill myself", "suicide", "suicidal",
		"hurt myself", "harm myself", "self harm",
		"want to die", "end my life", "end it all",
		"i'm going to kill", "i will kill", "i'm killing",
		"i want to kill",
	}

	for _, indicator := range obviousCrisisIndicators {
		if strings.Contains(lower, indicator) {
			log.Printf("[SafetyChecker] OBVIOUS CRISIS KEYWORD DETECTED: %s", indicator)
			return &SafetyAlert{
				AlertType:       ALERT_CRISIS,
				Severity:        ALERT_SEVERITY_IMMEDIATE,
				Title:           "Crisis Support Available",
				Message:         "I detected language suggesting you or someone else might be in crisis. Your safety matters. You're not alone.",
				Indicators:      []string{indicator},
				Resources:       getCrisisResources(),
				Recommendations: getCrisisRecommendations(),
			}
		}
	}

	// OBVIOUS illegal/threat indicators
	obviousIllegalIndicators := []string{
		"make a bomb", "build a bomb", "how to bomb",
		"make an explosive", "create a weapon",
		"planning to assault", "going to hurt someone",
		"going to kill someone", "murder someone",
	}

	for _, indicator := range obviousIllegalIndicators {
		if strings.Contains(lower, indicator) {
			log.Printf("[SafetyChecker] OBVIOUS THREAT DETECTED: %s", indicator)
			return &SafetyAlert{
				AlertType:       ALERT_ILLEGAL,
				Severity:        ALERT_SEVERITY_HIGH,
				Title:           "Cannot Assist",
				Message:         "I cannot help with illegal activities or threats. Please consult with appropriate professionals.",
				Indicators:      []string{indicator},
				Resources:       []CrisisResource{},
				Recommendations: getIllegalRecommendations(),
			}
		}
	}

	// No obvious crisis/threat detected
	// Ambiguous cases (sad, depressed, girl, problems, etc) will be handled by
	// ConversationAgent → clarification questions → context refinement → HarmAnalyzer
	return nil
}

// detectCrisisHeuristic uses simple logic when LLM unavailable
func (sc *Checker) detectCrisisHeuristic(message string) *SafetyAlert {
	lower := strings.ToLower(message)

	// Crisis indicators: look for key concepts and relationships
	crisisIndicators := []string{
		"hurt myself", "harm myself", "kill myself",
		"want to die", "end it all", "don't want to live",
		"feel like hurting", "suicidal", "suicide",
		"hurt them", "harm them", "kill them",
	}

	for _, indicator := range crisisIndicators {
		if strings.Contains(lower, indicator) {
			log.Printf("[SafetyChecker] Detected crisis indicator: %s", indicator)
			return &SafetyAlert{
				AlertType:       ALERT_CRISIS,
				Severity:        ALERT_SEVERITY_IMMEDIATE,
				Title:           "Crisis Support Available",
				Message:         "I detected language suggesting you or someone else might be in crisis. Your safety matters. You're not alone.",
				Indicators:      []string{indicator},
				Resources:       getCrisisResources(),
				Recommendations: getCrisisRecommendations(),
			}
		}
	}

	// Check for obvious illegal content
	illegalIndicators := []string{
		"sell drugs", "drug dealing", "distribute drugs",
		"make meth", "manufacture cocaine",
		"human trafficking", "child abuse",
	}

	for _, indicator := range illegalIndicators {
		if strings.Contains(lower, indicator) {
			log.Printf("[SafetyChecker] Detected illegal activity: %s", indicator)
			return &SafetyAlert{
				AlertType:       ALERT_ILLEGAL,
				Severity:        ALERT_SEVERITY_HIGH,
				Title:           "Cannot Assist",
				Message:         "I cannot help with illegal activities.",
				Indicators:      []string{indicator},
				Resources:       []CrisisResource{},
				Recommendations: getIllegalRecommendations(),
			}
		}
	}

	return nil
}

func getCrisisResources() []CrisisResource {
	return []CrisisResource{
		{
			Name:        "National Suicide Prevention Lifeline (US)",
			Description: "Free, confidential support 24/7",
			Number:      "988",
			URL:         "https://suicidepreventionlifeline.org",
			Region:      "USA",
		},
		{
			Name:        "Crisis Text Line (US)",
			Description: "Text-based crisis support",
			Number:      "Text HOME to 741741",
			URL:         "https://www.crisistextline.org",
			Region:      "USA",
		},
		{
			Name:        "Samaritans (UK)",
			Description: "Emotional support for anyone in distress",
			Number:      "116 123",
			URL:         "https://www.samaritans.org",
			Region:      "UK",
		},
		{
			Name:        "Befrienders (Australia)",
			Description: "24-hour suicide prevention service",
			Number:      "1300 22 4636",
			URL:         "https://www.lifeline.org.au",
			Region:      "Australia",
		},
		{
			Name:        "International Association for Suicide Prevention",
			Description: "Crisis centers worldwide",
			Number:      "",
			URL:         "https://www.iasp.info/resources/Crisis_Centres/",
			Region:      "International",
		},
		{
			Name:        "Telelife (EU)",
			Description: "European crisis helplines directory",
			Number:      "",
			URL:         "https://www.telelife.be/en",
			Region:      "Europe",
		},
	}
}

func getCrisisRecommendations() []string {
	return []string{
		"Call 911 or your local emergency number if in immediate danger",
		"Contact a crisis counselor using resources below",
		"Reach out to a trusted friend or family member",
		"Go to the nearest emergency room",
		"Text a crisis service if calling feels difficult",
	}
}

func getIllegalRecommendations() []string {
	return []string{
		"Seek advice from qualified legal professionals",
		"Reconsider this course of action",
		"Explore legal alternatives",
	}
}
