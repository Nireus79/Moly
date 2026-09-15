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

// CheckMessage performs LLM-based safety check with ethical frameworks
func (sc *Checker) CheckMessage(text string) *SafetyAlert {
	if text == "" {
		return nil
	}

	text = strings.TrimSpace(text)

	// Use LLM if available, otherwise use heuristic
	return sc.detectCrisisLLM(text)
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

	if strings.Contains(lower, "crisis") {
		return &SafetyAlert{
			AlertType:       ALERT_CRISIS,
			Severity:        ALERT_SEVERITY_IMMEDIATE,
			Title:           "Crisis Support Available",
			Message:         "I detected language suggesting you or someone else might be in crisis. Your safety matters. You're not alone.",
			Resources:       getCrisisResources(),
			Recommendations: getCrisisRecommendations(),
		}
	}

	if strings.Contains(lower, "illegal") {
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
