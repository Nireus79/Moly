package safety

import (

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











