package main

import (
	"regexp"
	"strings"
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
	AlertType   AlertType      `json:"alert_type"`
	Severity    AlertSeverity  `json:"severity"`
	Title       string         `json:"title"`
	Message     string         `json:"message"`
	Indicators  []string       `json:"indicators"`
	Resources   []CrisisResource `json:"resources"`
	Recommendations []string   `json:"recommendations"`
}

type CrisisResource struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Number      string `json:"number"`
	URL         string `json:"url"`
	Region      string `json:"region"`
}

type SafetyChecker struct {
	crisisPatterns  []*regexp.Regexp
	illegalPatterns []*regexp.Regexp
}

func NewSafetyChecker() *SafetyChecker {
	sc := &SafetyChecker{
		crisisPatterns:  compileCrisisPatterns(),
		illegalPatterns: compileIllegalPatterns(),
	}
	return sc
}

func compileCrisisPatterns() []*regexp.Regexp {
	patterns := []string{
		`\b(kill|hurt|harm)\s+(myself|myself|me|my self)\b`,
		`\b(suicide|suicidal|end it all)\b`,
		`\b(want to die|don't want to live)\b`,
		`\b(cut|slash|overdose|od)\s+(myself|me|wrist|arm)\b`,
		`\b(jump|hang)\s+(myself|me|from)\b`,
		`\b(no reason to live|can't go on|not worth it)\b`,
		`\b(kill|hurt|harm)\s+(them|him|her|you)\s+(if they|when they|because they)\b`,
		`\b(going to attack|going to hurt|will harm)\s+(them|him|her|my partner)\b`,
		`\b(murder|kill|stab|shoot)\s+(them|him|her|my|the)\b`,
		`\b(violent thoughts|urge to harm|urge to hurt)\b`,
	}

	var compiled []*regexp.Regexp
	for _, p := range patterns {
		if re, err := regexp.Compile("(?i)" + p); err == nil {
			compiled = append(compiled, re)
		}
	}
	return compiled
}

func compileIllegalPatterns() []*regexp.Regexp {
	patterns := []string{
		`\b(sell|dealing|distribute|supply)\s+(drugs|cocaine|heroin|meth|fentanyl|opioids)\b`,
		`\b(make|manufacture|cook|produce)\s+(drugs|methamphetamine|cocaine)\b`,
		`\b(rob|steal|burglar|burglary|theft)\s+(them|bank|store|house)\b`,
		`\b(traffic|trafficking|export)\s+(people|human|sex|organ)\b`,
		`\b(exploit|abuse|child)\s+(child|minor|kid)\b`,
		`\b(blackmail|extort|ransom)\b`,
		`\b(create|distribute|possess)\s+(child sexual abuse material|csam|cp)\b`,
		`\b(rape|sexual assault|molest)\b`,
		`\b(fraud|scam|embezzle)\s+(them|bank|company)\b`,
	}

	var compiled []*regexp.Regexp
	for _, p := range patterns {
		if re, err := regexp.Compile("(?i)" + p); err == nil {
			compiled = append(compiled, re)
		}
	}
	return compiled
}

func (sc *SafetyChecker) CheckMessage(text string) *SafetyAlert {
	if text == "" {
		return nil
	}

	text = strings.TrimSpace(text)

	// Check for crisis indicators
	for _, pattern := range sc.crisisPatterns {
		if pattern.MatchString(text) {
			return sc.createCrisisAlert(text, pattern)
		}
	}

	// Check for illegal activity
	for _, pattern := range sc.illegalPatterns {
		if pattern.MatchString(text) {
			return sc.createIllegalAlert(text)
		}
	}

	return nil
}

func (sc *SafetyChecker) createCrisisAlert(text string, pattern *regexp.Regexp) *SafetyAlert {
	indicators := []string{}
	if strings.Contains(strings.ToLower(text), "kill") || strings.Contains(strings.ToLower(text), "harm") {
		indicators = append(indicators, "Expression of intent to harm")
	}
	if strings.Contains(strings.ToLower(text), "suicide") || strings.Contains(strings.ToLower(text), "die") {
		indicators = append(indicators, "Suicidal ideation")
	}
	if strings.Contains(strings.ToLower(text), "violent") {
		indicators = append(indicators, "Violent thoughts")
	}

	alert := &SafetyAlert{
		AlertType:  ALERT_CRISIS,
		Severity:   ALERT_SEVERITY_IMMEDIATE,
		Title:      "Crisis Support Available",
		Message:    "I detected language suggesting you or someone else might be in crisis. Your safety matters. You're not alone.",
		Indicators: indicators,
		Resources:  getCrisisResources(),
		Recommendations: []string{
			"Call 911 or your local emergency number if in immediate danger",
			"Contact a crisis counselor using resources below",
			"Reach out to a trusted friend or family member",
			"Go to the nearest emergency room",
			"Text a crisis service if calling feels difficult",
		},
	}

	return alert
}

func (sc *SafetyChecker) createIllegalAlert(text string) *SafetyAlert {
	return &SafetyAlert{
		AlertType:  ALERT_ILLEGAL,
		Severity:   ALERT_SEVERITY_HIGH,
		Title:      "Cannot Assist",
		Message:    "I cannot help with illegal activities. Moly is designed for healthy relationship communication. Please consult with legal counsel if you have questions about your rights or obligations.",
		Indicators: []string{"Illegal activity detected"},
		Resources: []CrisisResource{},
		Recommendations: []string{
			"Seek advice from a qualified attorney",
			"Reconsider this course of action",
			"Explore legal alternatives",
		},
	}
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

func (sc *SafetyChecker) ContainsContactThreat(text string) bool {
	threatPatterns := []string{
		`\b(going to hurt|will harm|going to kill|will attack)\s+(you|me)\b`,
		`\b(meet.*hurt|meet.*kill|meet.*harm)\b`,
		`\b(threat|threaten|threatening)\b`,
		`\b(watch.*hurt|stalk.*hurt)\b`,
	}

	for _, p := range threatPatterns {
		if re, err := regexp.Compile("(?i)" + p); err == nil {
			if re.MatchString(text) {
				return true
			}
		}
	}
	return false
}
