package main

import (
	"strings"
)

type PrincipleSeverity string

const (
	SEVERITY_CRITICAL PrincipleSeverity = "critical"
	SEVERITY_HIGH     PrincipleSeverity = "high"
	SEVERITY_MEDIUM   PrincipleSeverity = "medium"
)

type CommunicationPrinciple struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Severity    PrincipleSeverity  `json:"severity"`
	Description string             `json:"description"`
	Questions   []string           `json:"questions"`
}

type CommunicationConstitution struct {
	SupremePrinciple string                           `json:"supreme_principle"`
	Principles       map[string]CommunicationPrinciple `json:"principles"`
}

type PrincipleViolation struct {
	PrincipleID string             `json:"principle_id"`
	Principle   string             `json:"principle"`
	Severity    PrincipleSeverity  `json:"severity"`
	Description string             `json:"description"`
	Reasoning   string             `json:"reasoning"`
}

type ConstitutionalAnalysis struct {
	AnalyzedAction    string                `json:"analyzed_action"`
	Violations        []PrincipleViolation  `json:"violations"`
	AlignedPrinciples []string              `json:"aligned_principles"`
	OverallRiskLevel  string                `json:"overall_risk_level"`
	CriticalConcerns  []string              `json:"critical_concerns"`
	Recommendations   []string              `json:"recommendations"`
	IsConstitutional  bool                  `json:"is_constitutional"`
}

type ConstitutionEvaluator struct {
	constitution CommunicationConstitution
}

func NewConstitutionEvaluator() *ConstitutionEvaluator {
	return &ConstitutionEvaluator{
		constitution: getCommunicationConstitution(),
	}
}

func getCommunicationConstitution() CommunicationConstitution {
	return CommunicationConstitution{
		SupremePrinciple: "Communicate with honesty, respect, and integrity. Seek genuine understanding and mutual benefit in all interactions - whether personal, professional, or formal.",
		Principles: map[string]CommunicationPrinciple{
			"honesty": {
				ID:       "honesty",
				Name:     "Honesty & Truthfulness",
				Severity: SEVERITY_CRITICAL,
				Description: "Be truthful about facts, intentions, and capabilities. Avoid deception, manipulation, or " +
					"misrepresentation. Others should be able to trust what you say.",
				Questions: []string{
					"Are you being truthful?",
					"Are you hiding important information?",
					"Are you trying to manipulate their perception?",
					"Would they make the same choice if they knew the full truth?",
					"Could this be seen as deceptive?",
				},
			},
			"consent_agreement": {
				ID:       "consent_agreement",
				Name:     "Clear Agreement & Consent",
				Severity: SEVERITY_CRITICAL,
				Description: "Ensure the other party explicitly agrees to what's being proposed. Agreement must be informed, " +
					"freely given, and based on complete information.",
				Questions: []string{
					"Do they know what they're agreeing to?",
					"Have you discussed this clearly?",
					"Can they freely decline without pressure?",
					"Do they have all the information they need?",
					"Are they saying yes or just not saying no?",
				},
			},
			"respect_boundaries": {
				ID:       "respect_boundaries",
				Name:     "Respect Boundaries & Limits",
				Severity: SEVERITY_CRITICAL,
				Description: "Respect stated and implied boundaries. No pressure to violate comfort zones or do things " +
					"someone has said no to.",
				Questions: []string{
					"Have they stated their boundaries?",
					"Are you pressuring them to do something they said no to?",
					"Are you respecting their limits?",
					"Would they feel safe declining your request?",
					"Are you crossing any lines?",
				},
			},
			"clarity": {
				ID:       "clarity",
				Name:     "Clarity & Clear Communication",
				Severity: SEVERITY_HIGH,
				Description: "Communicate clearly about intentions, expectations, and what you're asking for. Ambiguity " +
					"can lead to misunderstanding and harm.",
				Questions: []string{
					"Is your message clear?",
					"Could this be misunderstood?",
					"Have you stated your intentions plainly?",
					"Do they know what you want or need?",
					"Is there room for confusion?",
				},
			},
			"reciprocity": {
				ID:       "reciprocity",
				Name:     "Fairness & Reciprocity",
				Severity: SEVERITY_HIGH,
				Description: "Seek mutual benefit. Don't take advantage or exploit. Both parties' interests should matter.",
				Questions: []string{
					"Is this fair to both parties?",
					"Are you taking advantage?",
					"Would you accept this treatment from someone else?",
					"Are both parties getting value?",
					"Is this transactional or mutual?",
				},
			},
			"accountability": {
				ID:       "accountability",
				Name:     "Accountability & Follow-Through",
				Severity: SEVERITY_HIGH,
				Description: "Do what you say you'll do. Be reliable and follow through on commitments. Don't make promises " +
					"you won't keep.",
				Questions: []string{
					"Can you actually do what you're promising?",
					"Will you follow through?",
					"Are you making commitments you can't keep?",
					"Would they call you unreliable?",
					"Are you dodging responsibility?",
				},
			},
			"respect_autonomy": {
				ID:       "respect_autonomy",
				Name:     "Respect Autonomy & Agency",
				Severity: SEVERITY_CRITICAL,
				Description: "Respect the other person's right to make their own decisions. Don't coerce, manipulate, " +
					"or remove their ability to choose.",
				Questions: []string{
					"Are you giving them real choice?",
					"Are you coercing or manipulating?",
					"Could they freely decline?",
					"Are you limiting their options artificially?",
					"Are you respecting their autonomy?",
				},
			},
			"transparency": {
				ID:       "transparency",
				Name:     "Transparency About Intentions",
				Severity: SEVERITY_HIGH,
				Description: "Be open about your motivations and what you want. Hidden agendas undermine trust and fairness.",
				Questions: []string{
					"Are your motivations transparent?",
					"Are you hiding what you really want?",
					"Would they be concerned if they knew your full intent?",
					"Do you have hidden agendas?",
					"Could you be fully honest about what you want?",
				},
			},
			"safety": {
				ID:       "safety",
				Name:     "Safety & Non-Harm",
				Severity: SEVERITY_CRITICAL,
				Description: "Do no harm. Don't threaten, coerce, abuse, or endanger. Ensure psychological and physical safety.",
				Questions: []string{
					"Could this cause harm?",
					"Are you threatening or coercing?",
					"Is there risk of abuse?",
					"Would they feel safe?",
					"Could this escalate into dangerous territory?",
				},
			},
			"context_awareness": {
				ID:       "context_awareness",
				Name:     "Context & Power Awareness",
				Severity: SEVERITY_MEDIUM,
				Description: "Be aware of power dynamics and context. Don't exploit positions of power, influence, or advantage. " +
					"Account for vulnerability and asymmetry.",
				Questions: []string{
					"Is there a power imbalance?",
					"Are you exploiting an advantage?",
					"Are they in a vulnerable position?",
					"Is the playing field level?",
					"Would they feel comfortable disagreeing?",
				},
			},
		},
	}
}

func (ce *ConstitutionEvaluator) EvaluateAction(action string) *ConstitutionalAnalysis {
	analysis := &ConstitutionalAnalysis{
		AnalyzedAction:    action,
		Violations:        []PrincipleViolation{},
		AlignedPrinciples: []string{},
	}

	// Check each principle
	for _, principle := range ce.constitution.Principles {
		violation := ce.checkPrincipleViolation(action, principle)
		if violation != nil {
			analysis.Violations = append(analysis.Violations, *violation)
		} else {
			analysis.AlignedPrinciples = append(analysis.AlignedPrinciples, principle.Name)
		}
	}

	// Determine overall risk level
	analysis.calculateRiskLevel()

	// Generate recommendations
	analysis.generateRecommendations()

	// Determine if constitutional
	analysis.IsConstitutional = len(analysis.Violations) == 0

	return analysis
}

func (ce *ConstitutionEvaluator) checkPrincipleViolation(action string, principle CommunicationPrinciple) *PrincipleViolation {
	actionLower := strings.ToLower(action)

	switch principle.ID {
	case "consent_agreement":
		if ce.violatesConsent(actionLower) {
			return &PrincipleViolation{
				PrincipleID: principle.ID,
				Principle:   principle.Name,
				Severity:    principle.Severity,
				Description: principle.Description,
				Reasoning:   ce.getConsentViolationReasoning(actionLower),
			}
		}

	case "honesty":
		if ce.violatesHonesty(actionLower) {
			return &PrincipleViolation{
				PrincipleID: principle.ID,
				Principle:   principle.Name,
				Severity:    principle.Severity,
				Description: principle.Description,
				Reasoning:   "This involves deception, dishonesty, or misrepresentation.",
			}
		}

	case "respect_boundaries":
		if ce.violatesBoundaries(actionLower) {
			return &PrincipleViolation{
				PrincipleID: principle.ID,
				Principle:   principle.Name,
				Severity:    principle.Severity,
				Description: principle.Description,
				Reasoning:   ce.getBoundariesViolationReasoning(actionLower),
			}
		}

	case "clarity":
		if ce.violatesClarity(actionLower) {
			return &PrincipleViolation{
				PrincipleID: principle.ID,
				Principle:   principle.Name,
				Severity:    principle.Severity,
				Description: principle.Description,
				Reasoning:   "This communication is unclear or ambiguous, potentially leading to misunderstanding.",
			}
		}

	case "reciprocity":
		if ce.violatesReciprocity(actionLower) {
			return &PrincipleViolation{
				PrincipleID: principle.ID,
				Principle:   principle.Name,
				Severity:    principle.Severity,
				Description: principle.Description,
				Reasoning:   ce.getReciprocityViolationReasoning(actionLower),
			}
		}

	case "accountability":
		if ce.violatesAccountability(actionLower) {
			return &PrincipleViolation{
				PrincipleID: principle.ID,
				Principle:   principle.Name,
				Severity:    principle.Severity,
				Description: principle.Description,
				Reasoning:   "This involves breaking commitments or failing to follow through on promises.",
			}
		}

	case "respect_autonomy":
		if ce.violatesAutonomy(actionLower) {
			return &PrincipleViolation{
				PrincipleID: principle.ID,
				Principle:   principle.Name,
				Severity:    principle.Severity,
				Description: principle.Description,
				Reasoning:   "This pressures, coerces, or manipulates the other person instead of respecting their autonomy.",
			}
		}

	case "transparency":
		if ce.violatesTransparency(actionLower) {
			return &PrincipleViolation{
				PrincipleID: principle.ID,
				Principle:   principle.Name,
				Severity:    principle.Severity,
				Description: principle.Description,
				Reasoning:   "This involves hidden agendas or lack of transparency about intentions.",
			}
		}

	case "safety":
		if ce.violatesSafety(actionLower) {
			return &PrincipleViolation{
				PrincipleID: principle.ID,
				Principle:   principle.Name,
				Severity:    principle.Severity,
				Description: principle.Description,
				Reasoning:   ce.getSafetyViolationReasoning(actionLower),
			}
		}

	case "context_awareness":
		if ce.violatesContextAwareness(actionLower) {
			return &PrincipleViolation{
				PrincipleID: principle.ID,
				Principle:   principle.Name,
				Severity:    principle.Severity,
				Description: principle.Description,
				Reasoning:   "This fails to account for power dynamics or exploits an asymmetry.",
			}
		}
	}

	return nil
}

// Violation checkers
func (ce *ConstitutionEvaluator) violatesConsent(action string) bool {
	consentViolators := []string{
		"without asking", "without permission", "without consent", "without telling them",
		"manipulate", "trick", "deceive", "pressure", "coerce", "force",
		"without discussing", "surprise", "spring on them", "assume consent",
	}
	return ce.containsAny(action, consentViolators)
}

func (ce *ConstitutionEvaluator) violatesHonesty(action string) bool {
	honestyViolators := []string{
		"lie", "deceive", "pretend", "fake", "mislead", "manipulate",
		"be dishonest", "hide", "misrepresent", "mask",
		"false", "untruthful",
	}
	return ce.containsAny(action, honestyViolators)
}

func (ce *ConstitutionEvaluator) violatesBoundaries(action string) bool {
	boundariesViolators := []string{
		"pressure", "coerce", "force", "push past", "ignore their no", "guilt trip",
		"manipulate into", "without consent", "boundary violation", "push them",
		"ignore boundaries", "violate their limits", "make them uncomfortable",
	}
	return ce.containsAny(action, boundariesViolators)
}

func (ce *ConstitutionEvaluator) violatesReciprocity(action string) bool {
	reciprocityViolators := []string{
		"one-sided", "take advantage", "exploit", "use them", "transactional",
		"only for yourself", "all the effort", "one person doing", "narcissistic",
		"selfish", "don't care about", "not mutual", "just getting what you want",
	}
	return ce.containsAny(action, reciprocityViolators)
}

func (ce *ConstitutionEvaluator) violatesClarity(action string) bool {
	clarityViolators := []string{
		"vague", "ambiguous", "unclear", "unclear intentions",
		"not explained", "leave them guessing", "confused",
	}
	return ce.containsAny(action, clarityViolators)
}

func (ce *ConstitutionEvaluator) violatesAccountability(action string) bool {
	accountabilityViolators := []string{
		"break promise", "don't follow through", "won't deliver",
		"untrustworthy", "dodging", "avoid responsibility", "unreliable",
	}
	return ce.containsAny(action, accountabilityViolators)
}

func (ce *ConstitutionEvaluator) violatesAutonomy(action string) bool {
	autonomyViolators := []string{
		"pressure", "coerce", "manipulate", "force", "remove choice",
		"make them do", "no choice", "if you don't", "or else",
	}
	return ce.containsAny(action, autonomyViolators)
}

func (ce *ConstitutionEvaluator) violatesTransparency(action string) bool {
	transparencyViolators := []string{
		"hide agenda", "hidden motives", "don't tell them why",
		"secret reason", "real reason", "ulterior", "unspoken",
	}
	return ce.containsAny(action, transparencyViolators)
}

func (ce *ConstitutionEvaluator) violatesContextAwareness(action string) bool {
	contextViolators := []string{
		"power imbalance", "exploit", "take advantage",
		"vulnerable", "use their weakness", "asymmetry",
	}
	return ce.containsAny(action, contextViolators)
}

func (ce *ConstitutionEvaluator) violatesSafety(action string) bool {
	safetyViolators := []string{
		"harm", "abuse", "threaten", "coerce", "force", "assault",
		"control", "isolate", "threaten to hurt", "use violence", "physical abuse",
		"emotional abuse", "financial control", "no way out", "trap",
	}
	return ce.containsAny(action, safetyViolators)
}

// Reasoning generators
func (ce *ConstitutionEvaluator) getConsentViolationReasoning(action string) string {
	if ce.containsAny(action, []string{"without asking", "without permission", "assume consent"}) {
		return "This action proceeds without explicit agreement from the other person. Consent must be informed and freely given."
	}
	if ce.containsAny(action, []string{"manipulate", "trick", "deceive"}) {
		return "This involves deception or manipulation to get agreement, which violates informed consent."
	}
	if ce.containsAny(action, []string{"pressure", "coerce", "force"}) {
		return "This applies pressure or coercion, which negates genuine consent."
	}
	return "This action does not ensure informed, freely-given consent from the other person."
}


func (ce *ConstitutionEvaluator) getSafetyViolationReasoning(action string) string {
	return "This creates risk of physical, emotional, or psychological harm."
}

func (ce *ConstitutionEvaluator) getBoundariesViolationReasoning(action string) string {
	return "This involves pressuring, ignoring, or violating the other person's stated or implied boundaries."
}

func (ce *ConstitutionEvaluator) getReciprocityViolationReasoning(action string) string {
	return "This prioritizes one person's needs over fairness and mutual benefit."
}

func (ce *ConstitutionEvaluator) containsAny(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func (ca *ConstitutionalAnalysis) calculateRiskLevel() {
	criticalCount := 0
	highCount := 0

	for _, violation := range ca.Violations {
		if violation.Severity == SEVERITY_CRITICAL {
			criticalCount++
		} else if violation.Severity == SEVERITY_HIGH {
			highCount++
		}
	}

	if criticalCount > 0 {
		ca.OverallRiskLevel = "EXTREME"
		ca.CriticalConcerns = append(ca.CriticalConcerns,
			"This violates foundational principles of ethical communication",
			"Proceeding could cause significant harm or damage trust",
			"Strongly recommend reconsidering this approach",
		)
	} else if highCount >= 2 {
		ca.OverallRiskLevel = "HIGH"
		ca.CriticalConcerns = append(ca.CriticalConcerns,
			"Multiple principles are compromised",
			"Significant risk to communication integrity and trust",
			"Proceed with extreme caution",
		)
	} else if len(ca.Violations) > 0 {
		ca.OverallRiskLevel = "MEDIUM"
		ca.CriticalConcerns = append(ca.CriticalConcerns,
			"Some principles are violated",
			"Consider adjusting approach",
		)
	} else {
		ca.OverallRiskLevel = "SAFE"
	}
}

func (ca *ConstitutionalAnalysis) generateRecommendations() {
	switch ca.OverallRiskLevel {
	case "EXTREME":
		ca.Recommendations = []string{
			"DO NOT proceed with this action",
			"Reconsider your approach fundamentally",
			"Prioritize honesty, consent, and respect",
			"Ensure the other person's autonomy is protected",
			"Seek objective feedback from trusted advisors",
		}
	case "HIGH":
		ca.Recommendations = []string{
			"Pause and reconsider this approach",
			"How can you modify this to be more ethical?",
			"Have you discussed this openly and honestly?",
			"What would they say if they knew your full plan?",
			"Look for ways to align with ethical principles",
		}
	case "MEDIUM":
		ca.Recommendations = []string{
			"Be mindful of the principles being compromised",
			"Look for ways to improve alignment with these principles",
			"Ensure you're being transparent and honest",
			"Consider the other person's perspective and autonomy",
		}
	case "SAFE":
		ca.Recommendations = []string{
			"This approach aligns with ethical communication principles",
			"Focus on genuine communication and mutual benefit",
			"Build trust through consistent, respectful behavior",
			"Respect boundaries and honor commitments",
		}
	}
}

func (ce *ConstitutionEvaluator) GetPrinciples() []CommunicationPrinciple {
	principles := []CommunicationPrinciple{}
	for _, p := range ce.constitution.Principles {
		principles = append(principles, p)
	}
	return principles
}

func (ce *ConstitutionEvaluator) GetSupremePrinciple() string {
	return ce.constitution.SupremePrinciple
}
