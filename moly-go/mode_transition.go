package main

import (
	"fmt"
	"strings"
)

type RelationshipMode string

const (
	PROFESSIONAL      RelationshipMode = "professional"
	FRIENDLY          RelationshipMode = "friendly"
	CASUAL_FLIRTING   RelationshipMode = "casual_flirting"
	ROMANTIC          RelationshipMode = "romantic"
	INTIMATE          RelationshipMode = "intimate_sexual"
	POWER_EXCHANGE    RelationshipMode = "power_exchange"
)

type RiskFactor struct {
	Category    string   `json:"category"`
	Level       string   `json:"level"`
	Description string   `json:"description"`
	Mitigation  []string `json:"mitigation"`
}

type TransitionPhase struct {
	Phase       int      `json:"phase"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Duration    string   `json:"duration"`
	Tactics     []string `json:"tactics"`
	SignalsToWatch []string `json:"signals_to_watch"`
	RedFlags    []string `json:"red_flags"`
}

type ModeTransitionAnalysis struct {
	ModeShiftDetected  bool                `json:"mode_shift_detected"`
	CurrentMode        RelationshipMode    `json:"current_mode"`
	DesiredMode        RelationshipMode    `json:"desired_mode"`
	RiskLevel          string              `json:"risk_level"`
	OverallRiskScore   int                 `json:"overall_risk_score"`
	Implications       []string            `json:"implications"`
	RiskFactors        []RiskFactor        `json:"risk_factors"`
	Phases             []TransitionPhase   `json:"phases"`
	CriticalQuestions  []string            `json:"critical_questions"`
	Recommendations    []string            `json:"recommendations"`
	ProCons            map[string][]string `json:"pro_cons"`
}

type ModeTransitionEngine struct {
	db *Database
}

func NewModeTransitionEngine(db *Database) *ModeTransitionEngine {
	return &ModeTransitionEngine{db: db}
}

func (mte *ModeTransitionEngine) AnalyzeModeShift(contactID int, currentMode, desiredMode, context string) (*ModeTransitionAnalysis, error) {
	contact, err := mte.db.getContact(contactID)
	if err != nil {
		return nil, fmt.Errorf("contact not found: %v", err)
	}

	// Check if there's actually a mode shift
	if currentMode == desiredMode {
		return &ModeTransitionAnalysis{
			ModeShiftDetected: false,
			CurrentMode:       RelationshipMode(currentMode),
			DesiredMode:       RelationshipMode(desiredMode),
		}, nil
	}

	analysis := &ModeTransitionAnalysis{
		ModeShiftDetected: true,
		CurrentMode:       RelationshipMode(currentMode),
		DesiredMode:       RelationshipMode(desiredMode),
		ProCons:           make(map[string][]string),
	}

	// Analyze the specific transition
	switch {
	case currentMode == "professional" && desiredMode == "romantic":
		mte.analyzeProfessionalToRomantic(analysis, contact, context)
	case currentMode == "professional" && desiredMode == "power_exchange":
		mte.analyzeProfessionalToPowerExchange(analysis, contact, context)
	case currentMode == "friendly" && desiredMode == "romantic":
		mte.analyzeFriendlyToRomantic(analysis, contact, context)
	case currentMode == "friendly" && desiredMode == "power_exchange":
		mte.analyzeFriendlyToPowerExchange(analysis, contact, context)
	case currentMode == "casual_flirting" && desiredMode == "romantic":
		mte.analyzeCasualToRomantic(analysis, contact, context)
	case currentMode == "casual_flirting" && desiredMode == "power_exchange":
		mte.analyzeCasualToPowerExchange(analysis, contact, context)
	default:
		mte.analyzeGenericTransition(analysis, contact, context)
	}

	// Add contact-specific context
	mte.addContactContext(analysis, contact)

	return analysis, nil
}

func (mte *ModeTransitionEngine) analyzeProfessionalToRomantic(a *ModeTransitionAnalysis, contact *Contact, context string) {
	a.RiskLevel = "high"
	a.OverallRiskScore = 75

	a.Implications = []string{
		"This could fundamentally change the professional dynamic",
		"Risk of making work awkward if not reciprocated",
		"Power imbalances (especially if mentoring/hierarchical) create complications",
		"Rejection could affect work environment",
	}

	a.RiskFactors = []RiskFactor{
		{
			Category:    "workplace_dynamics",
			Level:       "high",
			Description: "Romantic interest could compromise professional relationship",
			Mitigation: []string{
				"Wait until one person leaves the organization",
				"Ensure no power imbalance (not your boss/mentor)",
				"Keep initial signals extremely subtle",
				"Be prepared to preserve professionalism if rejected",
			},
		},
		{
			Category:    "rejection_impact",
			Level:       "high",
			Description: "If they're not interested, daily interaction becomes awkward",
			Mitigation: []string{
				"Gauge their personal interest first through small talk",
				"Start with friendship signals before romantic ones",
				"Have an exit strategy (transfer teams, new job, etc.)",
			},
		},
		{
			Category:    "timing",
			Level:       "medium",
			Description: "Professional relationships need time before romantic signals",
			Mitigation: []string{
				"Phase 1: Break out of purely professional mode",
				"Share personal interests and vulnerabilities",
				"Increase non-work hangouts gradually",
			},
		},
	}

	a.Phases = []TransitionPhase{
		{
			Phase:       1,
			Name:        "Break the Professional Bubble",
			Description: "Shift from purely professional to friendly colleague",
			Duration:    "2-4 weeks of regular interaction",
			Tactics: []string{
				"Share personal interests beyond work",
				"Suggest casual lunches or coffee",
				"Show vulnerability about non-work topics",
				"Ask about their life, family, interests",
				"Find common ground outside work",
			},
			SignalsToWatch: []string{
				"Do they initiate personal conversations?",
				"Do they seem interested in your life?",
				"Do they suggest hanging out outside work?",
			},
			RedFlags: []string{
				"They always keep conversations work-focused",
				"They mention their significant other frequently",
				"They seem uncomfortable with personal questions",
			},
		},
		{
			Phase:       2,
			Name:        "Introduce Romantic Signals",
			Description: "Subtle flirting and increased intimacy",
			Duration:    "2-3 weeks of building chemistry",
			Tactics: []string{
				"Compliment personality, not just appearance",
				"Increase eye contact and genuine interest",
				"Find reasons for one-on-one time",
				"Share more personal/vulnerable stories",
				"Use light humor and playfulness",
				"Suggest more personal hangouts (dinner, drinks)",
			},
			SignalsToWatch: []string{
				"Do they mirror your body language?",
				"Do they seem happy to see you?",
				"Do they initiate contact (touch arm, lean in)?",
				"Do they reciprocate personal sharing?",
			},
			RedFlags: []string{
				"They maintain professional distance",
				"They don't reciprocate personal sharing",
				"They suggest group hangouts instead of one-on-one",
			},
		},
		{
			Phase:       3,
			Name:        "Express Interest Explicitly",
			Description: "Clear communication of romantic feelings",
			Duration:    "When signals are positive",
			Tactics: []string{
				"Find private moment to be direct",
				"Acknowledge the work dynamic",
				"Express genuine feelings",
				"Make it safe to say no",
				"Ask if they've felt similar",
				"Discuss how to handle at work if they reciprocate",
			},
			SignalsToWatch: []string{
				"Their response tone and hesitation",
				"Whether they need time to think",
				"If they express interest or concern about work",
			},
			RedFlags: []string{
				"Immediate hard no",
				"Concern about work complications without interest",
				"Suddenly increased professional distance",
			},
		},
	}

	a.CriticalQuestions = []string{
		"Do you have any power dynamic issues (is one of you supervising the other)?",
		"Would rejection significantly impact your work life?",
		"Are you prepared to transfer teams or find a new job if things get awkward?",
		"Have you observed any personal interest signals from them?",
		"What happens to your professional relationship if they say no?",
		"Are they currently in a relationship?",
		"How long have you known them in this professional capacity?",
	}

	a.Recommendations = []string{
		"BEST CASE: Work in separate departments or one person leaves soon",
		"Move SLOWLY through phases - rushing signals will backfire",
		"Keep all initial interactions deniable (could be friendly colleague)",
		"Never pressure or make them feel uncomfortable at work",
		"Have a genuine friendship foundation before romantic signals",
		"Be prepared to accept rejection gracefully",
	}

	a.ProCons["Pros"] = []string{
		"You already know them well professionally",
		"You share work interests and understanding",
		"Workplace propinquity (lots of time together)",
	}

	a.ProCons["Cons"] = []string{
		"Power dynamics and hierarchy",
		"Awkwardness if rejected",
		"Complicates professional environment",
		"Limited privacy for dating",
	}
}

func (mte *ModeTransitionEngine) analyzeFriendlyToRomantic(a *ModeTransitionAnalysis, contact *Contact, context string) {
	a.RiskLevel = "medium"
	a.OverallRiskScore = 55

	a.Implications = []string{
		"This transforms a friendship into potential romance",
		"Risk of changing the friendship if not reciprocated",
		"They may have friend-zoned you without realizing",
		"Clear communication needed to preserve friendship if they decline",
	}

	a.RiskFactors = []RiskFactor{
		{
			Category:    "friendship_preservation",
			Level:       "medium",
			Description: "Romantic interest could end the friendship",
			Mitigation: []string{
				"Assess if they've shown any romantic interest",
				"Be prepared to accept friendship-only status",
				"Plan how to move forward if they decline",
			},
		},
		{
			Category:    "expectations",
			Level:       "medium",
			Description: "They may see you only as a friend",
			Mitigation: []string{
				"Look for signs they might be interested romantically",
				"Start with subtle signals before explicit",
				"Be ready for honest conversation about feelings",
			},
		},
	}

	a.Phases = []TransitionPhase{
		{
			Phase:       1,
			Name:        "Increase Intimate Friendship",
			Description: "Deepen emotional connection",
			Duration:    "2-4 weeks",
			Tactics: []string{
				"Increase one-on-one time",
				"Share deeper personal stories",
				"Be more vulnerable and authentic",
				"Show you care beyond typical friendship",
				"Create memorable experiences together",
			},
			SignalsToWatch: []string{
				"Do they seek out your company?",
				"Do they share personal vulnerabilities with you?",
				"Do they seem emotionally intimate?",
			},
			RedFlags: []string{
				"They treat you like one of their friends",
				"They don't reciprocate emotional intimacy",
				"They talk about dating others around you",
			},
		},
		{
			Phase:       2,
			Name:        "Introduce Romantic Tension",
			Description: "Add flirtation and physical closeness",
			Duration:    "1-3 weeks",
			Tactics: []string{
				"Increase physical contact (appropriate to context)",
				"Use playful teasing and flirtation",
				"Create intimate settings (candlelit dinner, etc.)",
				"Let longer silences happen (intimacy building)",
				"Increase compliments on attractiveness",
			},
			SignalsToWatch: []string{
				"Do they reciprocate physical contact?",
				"Do they respond to flirtation?",
				"Do they create reasons to be close?",
			},
			RedFlags: []string{
				"They pull back from physical contact",
				"They don't reciprocate flirtation",
				"They seem uncomfortable with the shift",
			},
		},
		{
			Phase:       3,
			Name:        "Express Romantic Feelings",
			Description: "Have honest conversation about relationship",
			Duration:    "When ready",
			Tactics: []string{
				"Choose a comfortable, private setting",
				"Be honest about your feelings",
				"Emphasize the friendship is valuable regardless",
				"Ask how they feel",
				"Listen to their response without pressure",
			},
			SignalsToWatch: []string{
				"Their emotional response",
				"Whether they mention enjoying your company",
				"If they express confusion or clarity",
			},
			RedFlags: []string{
				"Immediate rejection",
				"They need too much time to think",
				"They mention someone else they're interested in",
			},
		},
	}

	a.CriticalQuestions = []string{
		"Have you ever felt any romantic chemistry from them?",
		"Do they treat you differently than their other friends?",
		"Would you be OK staying friends if they're not interested romantically?",
		"How long have you been close friends?",
		"Have they mentioned dating or relationship interests?",
		"Do you know their type/what they're attracted to?",
	}

	a.Recommendations = []string{
		"BEST CASE: You already have a strong emotional foundation",
		"Friendships can transition to romance - it happens often",
		"Be authentic and honest about your feelings",
		"Respect their response, whatever it is",
		"Plan how you'll handle friendship after rejection",
	}

	a.ProCons["Pros"] = []string{
		"Strong existing foundation and trust",
		"You know each other well",
		"Many successful relationships start as friendships",
		"Emotional intimacy already established",
	}

	a.ProCons["Cons"] = []string{
		"Risk losing the friendship",
		"They may have friend-zoned you",
		"Changing dynamic can be awkward",
		"Recovery from rejection is harder",
	}
}

func (mte *ModeTransitionEngine) analyzeCasualToRomantic(a *ModeTransitionAnalysis, contact *Contact, context string) {
	a.RiskLevel = "low"
	a.OverallRiskScore = 30

	a.Implications = []string{
		"Natural progression from casual flirting to relationship",
		"Both parties have indicated romantic interest already",
		"Risk is mainly about timing and clarity",
	}

	a.RiskFactors = []RiskFactor{
		{
			Category:    "clarity",
			Level:       "low",
			Description: "Need to be clear about relationship expectations",
			Mitigation: []string{
				"Have explicit conversation about commitment",
				"Discuss what relationship means to both",
				"Align on timeline and exclusivity",
			},
		},
	}

	a.Phases = []TransitionPhase{
		{
			Phase:       1,
			Name:        "Deepen Connection",
			Description: "Move from surface-level flirting to genuine intimacy",
			Duration:    "1-2 weeks",
			Tactics: []string{
				"Share more personal stories",
				"Spend more exclusive time together",
				"Increase physical intimacy gradually",
				"Show genuine interest in their life",
			},
			SignalsToWatch: []string{
				"Are they wanting more time together?",
				"Do they seem to want deeper connection?",
				"Do they initiate more serious conversations?",
			},
			RedFlags: []string{
				"They seem to want to keep it casual",
				"They don't reciprocate intimacy",
				"They mention other people they're seeing",
			},
		},
		{
			Phase:       2,
			Name:        "Define the Relationship",
			Description: "Have clear conversation about commitment",
			Duration:    "When it feels right",
			Tactics: []string{
				"Express that you're catching feelings",
				"Ask about their feelings and intentions",
				"Discuss what you both want from this",
				"Make it official if mutually interested",
			},
			SignalsToWatch: []string{
				"Their response to your feelings",
				"Whether they express similar feelings",
				"Their openness to commitment",
			},
			RedFlags: []string{
				"They want to stay casual",
				"They seem hesitant about commitment",
				"They mention other romantic interests",
			},
		},
	}

	a.CriticalQuestions = []string{
		"Are you ready for a committed relationship with this person?",
		"What does commitment mean to you both?",
		"Are you both emotionally available?",
		"What are your timeline and relationship goals?",
	}

	a.Recommendations = []string{
		"BEST CASE: Natural progression from existing connection",
		"Be honest about wanting more commitment",
		"Discuss expectations early",
		"Enjoy the chemistry while building deeper connection",
	}

	a.ProCons["Pros"] = []string{
		"Existing romantic attraction",
		"Natural chemistry already established",
		"Lower social risk than professional transitions",
		"Both know what they're looking for",
	}

	a.ProCons["Cons"] = []string{
		"They might want to stay casual",
		"Timing mismatches possible",
		"Clarity needed to avoid hurt",
	}
}

func (mte *ModeTransitionEngine) analyzeFriendlyToPowerExchange(a *ModeTransitionAnalysis, contact *Contact, context string) {
	a.RiskLevel = "high"
	a.OverallRiskScore = 80

	a.Implications = []string{
		"Introduces BDSM/kink element to existing friendship",
		"Changes power dynamics significantly",
		"Requires deep trust and extensive communication",
		"Friendship may not survive if mismatched interests",
	}

	a.RiskFactors = []RiskFactor{
		{
			Category:    "kink_compatibility",
			Level:       "high",
			Description: "You don't know if they're interested in BDSM",
			Mitigation: []string{
				"Carefully gauge interest before proposing",
				"Start conversations about sexuality generally",
				"Look for signs they're into kink",
				"Never pressure or shame if not interested",
			},
		},
		{
			Category:    "consent_and_communication",
			Level:       "high",
			Description: "Power exchange requires constant communication",
			Mitigation: []string{
				"Extensive pre-negotiation conversations",
				"Clear boundaries and safewords",
				"Regular check-ins about comfort",
				"Written or verbal contracts recommended",
			},
		},
		{
			Category:    "friendship_risk",
			Level:       "high",
			Description: "Introducing kink could end friendship if mishandled",
			Mitigation: []string{
				"Gauge interest cautiously",
				"Make it clear you value their friendship",
				"Give them space to decline without shame",
				"Plan how to preserve friendship if they're not interested",
			},
		},
	}

	a.Phases = []TransitionPhase{
		{
			Phase:       1,
			Name:        "Gauge Kink Interest",
			Description: "Carefully explore if they're interested in BDSM",
			Duration:    "4-6 weeks of conversations",
			Tactics: []string{
				"Ask general questions about sexuality",
				"Share your own interests (vulnerability builds trust)",
				"Recommend kink-friendly shows/books casually",
				"Listen for hints about their interests",
				"Mention BDSM positively in conversation",
			},
			SignalsToWatch: []string{
				"Do they ask follow-up questions about BDSM?",
				"Do they share kink interests or curiosities?",
				"Do they seem non-judgmental about alternative sexuality?",
				"Do they mention exploring or curiosity?",
			},
			RedFlags: []string{
				"They seem uncomfortable with the topic",
				"They make negative comments about BDSM",
				"They change the subject",
				"They seem judgmental about sexuality",
			},
		},
		{
			Phase:       2,
			Name:        "Express Interest Cautiously",
			Description: "Share your desires with clear boundaries",
			Duration:    "1-2 conversations",
			Tactics: []string{
				"Be vulnerable about what you're into",
				"Emphasize consent and their boundaries",
				"Make it clear there's no pressure",
				"Ask what they're curious about",
				"Discuss whether this could work for you both",
			},
			SignalsToWatch: []string{
				"Are they interested but cautious?",
				"Do they ask thoughtful questions?",
				"Do they express their own interests?",
				"Do they take time to think?",
			},
			RedFlags: []string{
				"Immediate rejection or discomfort",
				"They seem to only want to please you",
				"They don't express their own boundaries/limits",
				"They seem coerced into considering it",
			},
		},
		{
			Phase:       3,
			Name:        "Extensive Negotiation",
			Description: "Deep conversations about consent, roles, boundaries",
			Duration:    "Multiple conversations over weeks",
			Tactics: []string{
				"Discuss roles (Dom/sub/switch dynamics)",
				"Establish hard limits (what's absolutely off-limits)",
				"Discuss soft limits (things to be cautious about)",
				"Create a safeword system",
				"Talk about expectations and frequency",
				"Discuss how to handle the friendship aspect",
			},
			SignalsToWatch: []string{
				"Are they actively participating in negotiation?",
				"Are they expressing their own needs clearly?",
				"Are they asking important safety questions?",
				"Do they seem enthusiastic and consensual?",
			},
			RedFlags: []string{
				"They seem hesitant or unsure",
				"They're not expressing their own needs",
				"They seem pressured to agree",
				"Communication feels one-sided",
			},
		},
	}

	a.CriticalQuestions = []string{
		"Do you actually know if they're interested in BDSM at all?",
		"Are you prepared if they say no?",
		"Have you thoroughly discussed consent and boundaries?",
		"What role(s) do you each want (Dom, sub, switch)?",
		"What are your hard limits and their hard limits?",
		"How will this affect your friendship?",
		"Are you both emotionally mature enough to handle power exchange?",
		"Have you discussed safewords and check-ins?",
	}

	a.Recommendations = []string{
		"CRITICAL: Introduce this VERY gradually",
		"Only proceed if they actively express interest",
		"Never pressure or coerce - consent is everything",
		"Extensive communication is NOT optional",
		"Read books together about BDSM and consent",
		"Consider talking to experienced people in the community",
		"Start with very light power exchange before anything intense",
		"Make regular check-ins part of your dynamic",
	}

	a.ProCons["Pros"] = []string{
		"Existing trust foundation is crucial for BDSM",
		"You know each other well",
		"Can build slowly and safely",
	}

	a.ProCons["Cons"] = []string{
		"Very high risk of misunderstanding",
		"If they're not interested, friendship suffers",
		"Requires constant communication",
		"Introduces complex power dynamics",
		"Potential for harm if not handled carefully",
	}
}

func (mte *ModeTransitionEngine) analyzeCasualToPowerExchange(a *ModeTransitionAnalysis, contact *Contact, context string) {
	a.RiskLevel = "medium"
	a.OverallRiskScore = 50

	a.Implications = []string{
		"You're already in flirty/sexual territory",
		"Power exchange adds structure to existing attraction",
		"Both parties are already interested in each other",
		"Main risk is clarity and consent",
	}

	a.RiskFactors = []RiskFactor{
		{
			Category:    "interest_alignment",
			Level:       "medium",
			Description: "Casual doesn't always mean wanting power exchange",
			Mitigation: []string{
				"Ask about their interests in BDSM",
				"Gauge enthusiasm level",
				"Don't assume interest just because they're flirty",
			},
		},
		{
			Category:    "consent_clarity",
			Level:       "medium",
			Description: "Need very clear consent for power exchange",
			Mitigation: []string{
				"Have explicit conversations about roles",
				"Establish safewords and boundaries",
				"Regular check-ins about comfort",
			},
		},
	}

	a.Phases = []TransitionPhase{
		{
			Phase:       1,
			Name:        "Explore Kink Interests",
			Description: "Discuss BDSM and power exchange desires",
			Duration:    "1-2 conversations",
			Tactics: []string{
				"Ask what they're into sexually",
				"Share your interest in power exchange",
				"Discuss dominant/submissive dynamics",
				"Gauge enthusiasm and comfort level",
			},
			SignalsToWatch: []string{
				"Do they express interest in power play?",
				"Do they ask questions about BDSM?",
				"Do they seem excited about exploring together?",
			},
			RedFlags: []string{
				"They're not interested in BDSM",
				"They seem uncomfortable",
				"They only want vanilla sex",
			},
		},
		{
			Phase:       2,
			Name:        "Establish Structure",
			Description: "Define roles, boundaries, and agreements",
			Duration:    "Before first scene",
			Tactics: []string{
				"Decide on Dom/sub/switch roles",
				"Establish safewords",
				"Discuss hard and soft limits",
				"Talk about frequency and intensity",
				"Plan first \"scene\" or power exchange activity",
			},
			SignalsToWatch: []string{
				"Are they actively engaged in planning?",
				"Do they clearly express their needs?",
				"Do they ask safety questions?",
			},
			RedFlags: []string{
				"They seem hesitant",
				"Communication feels one-sided",
				"They're not expressing boundaries",
			},
		},
	}

	a.CriticalQuestions = []string{
		"Are you both interested in BDSM/power exchange?",
		"What role(s) do you each want?",
		"What are your hard limits?",
		"What's your safeword system?",
		"How will you handle this alongside casual dating?",
		"Are you ready for the communication this requires?",
	}

	a.Recommendations = []string{
		"Easier transition than casual to friendship BDSM",
		"You already have chemistry and interest",
		"Still need clear negotiation and consent",
		"Start with light power exchange",
		"Build intensity gradually based on comfort",
	}

	a.ProCons["Pros"] = []string{
		"You already like each other",
		"Sexual interest aligned",
		"Can build the dynamic step-by-step",
	}

	a.ProCons["Cons"] = []string{
		"Casual might not mean wanting BDSM",
		"Need more communication than casual implies",
		"Intensity could change the dynamic unexpectedly",
	}
}

func (mte *ModeTransitionEngine) analyzeProfessionalToPowerExchange(a *ModeTransitionAnalysis, contact *Contact, context string) {
	a.RiskLevel = "extreme"
	a.OverallRiskScore = 95

	a.Implications = []string{
		"This combines the highest risks from both transitions",
		"Professional + power exchange = severe complications",
		"Workplace dynamics make power exchange very problematic",
		"Recommend NOT pursuing this combination",
	}

	a.RiskFactors = []RiskFactor{
		{
			Category:    "professional_consequences",
			Level:       "extreme",
			Description: "Power exchange in workplace is extremely risky",
			Mitigation: []string{
				"STRONGLY RECOMMEND: Wait until one person leaves",
				"Do NOT pursue BDSM with coworkers/mentors",
				"Workplace harassment concerns",
				"Career and reputation at stake",
			},
		},
	}

	a.Phases = []TransitionPhase{
		{
			Phase:       1,
			Name:        "WAIT - Do Not Proceed",
			Description: "Strongly recommend against pursuing this",
			Duration:    "Until career situation changes",
			Tactics: []string{
				"Do not express BDSM interest to this person",
				"Wait until no workplace relationship exists",
				"Consider finding BDSM partners outside work",
				"Prioritize your career and reputation",
			},
			SignalsToWatch: []string{},
			RedFlags: []string{
				"Everything about this is a red flag",
			},
		},
	}

	a.CriticalQuestions = []string{
		"Are you absolutely certain there's no power imbalance?",
		"Are you prepared for severe workplace consequences?",
		"Is there any way this could be perceived as harassment?",
		"What's your exit strategy if things go wrong?",
	}

	a.Recommendations = []string{
		"DO NOT PURSUE THIS",
		"The risk far outweighs the potential benefit",
		"Wait until the professional relationship ends",
		"Find other BDSM partners outside your workplace",
		"Protect your career and reputation",
		"If they're interested, agree to revisit when circumstances change",
	}

	a.ProCons["Pros"] = []string{
		"(There are virtually no pros to this combination)",
	}

	a.ProCons["Cons"] = []string{
		"Career-ending if it goes wrong",
		"Severe power imbalance concerns",
		"Workplace harassment potential",
		"Could affect both of your jobs",
		"Extremely difficult to keep private",
		"Power exchange + professional power = disaster",
	}
}

func (mte *ModeTransitionEngine) analyzeGenericTransition(a *ModeTransitionAnalysis, contact *Contact, context string) {
	a.RiskLevel = "medium"
	a.OverallRiskScore = 50
	a.Implications = []string{
		"Unique transition requiring careful consideration",
		"Assess compatibility and shared interests",
	}
	a.Recommendations = []string{
		"Understand what each mode requires",
		"Communicate clearly about intentions",
		"Go slowly and gauge interest at each step",
		"Respect boundaries and don't pressure",
	}
	a.ProCons["Pros"] = []string{"New possibilities if both interested"}
	a.ProCons["Cons"] = []string{"Risk of changing existing relationship"}
}

func (mte *ModeTransitionEngine) addContactContext(a *ModeTransitionAnalysis, contact *Contact) {
	// Add personalization based on contact info
	if strings.Contains(strings.ToLower(contact.Notes), "work") || strings.Contains(strings.ToLower(contact.Relationship), "colleague") {
		a.OverallRiskScore += 15
		a.Implications = append(a.Implications, fmt.Sprintf("NOTE: %s is work-related. This adds complexity.", contact.Name))
	}

	if contact.InteractionCount == 0 {
		a.OverallRiskScore += 10
		a.Implications = append(a.Implications, fmt.Sprintf("You have no interaction history with %s yet. Start very cautiously.", contact.Name))
	}
}
