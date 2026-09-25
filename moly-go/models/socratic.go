package models

import "fmt"

// Constitution contains all ethical principles and frameworks that guide Moly
type Constitution struct {
	Metadata         ConstitutionMetadata `yaml:"metadata"`
	SupremePrinciples []Principle         `yaml:"supreme_principles"`
	EthicalFrameworks []Framework         `yaml:"ethical_frameworks"`
	Mappings         map[string][]string  `yaml:"mappings"` // principle -> frameworks
	MolyIntegration  IntegrationConfig    `yaml:"moly_integration"`
}

type ConstitutionMetadata struct {
	Version     string `yaml:"version"`
	LastUpdated string `yaml:"last_updated"`
	Author      string `yaml:"author"`
	Description string `yaml:"description"`
}

// Principle represents a supreme ethical principle that Moly follows
type Principle struct {
	ID                  string   `yaml:"id"`                    // e.g., "user_autonomy"
	Name                string   `yaml:"name"`                  // e.g., "User Autonomy"
	Severity            string   `yaml:"severity"`              // critical, high, medium
	Description         string   `yaml:"description"`
	Domains             []string `yaml:"domains"`               // where it applies
	Violations          []string `yaml:"violations"`            // what breaks it
	SupportingFrameworks []string `yaml:"supporting_frameworks"` // which frameworks support it
}

// Framework represents an ethical framework for decision-making
type Framework struct {
	ID           string   `yaml:"id"`            // e.g., "kantian"
	Name         string   `yaml:"name"`          // e.g., "Kantian Ethics"
	Principle    string   `yaml:"principle"`     // core principle
	KeyQuestions []string `yaml:"key_questions"` // how to test
	Violations   []string `yaml:"violations"`    // what breaks it
	TestFor      []string `yaml:"test_for"`      // what to look for
}

// SocraticQuestion represents a single question in the library
type SocraticQuestion struct {
	ID                string   `yaml:"id"`                 // e.g., "q_stakeholder_001"
	Text              string   `yaml:"text"`               // the actual question
	SocraticApproach  string   `yaml:"socratic_approach"`  // e.g., "identifying_stakeholders"
	Category          string   `yaml:"category"`           // e.g., "stakeholder"
	TargetsPrinciple  string   `yaml:"targets_principle"`  // e.g., "user_autonomy"
	TargetsFramework  string   `yaml:"targets_framework"`  // e.g., "rights_based"
	ExpectedInsights  []string `yaml:"expected_insights"`  // what this reveals
	DepthLevel        int      `yaml:"depth_level"`        // 1-5 progression
	FollowUpQuestions []string `yaml:"follow_up_questions"` // next questions
	Domains           []string `yaml:"domains"`            // where it applies
}

// QuestionLibrary holds all questions and provides lookup methods
type QuestionLibrary struct {
	AllQuestions         map[string]*SocraticQuestion // keyed by ID
	QuestionsByApproach  map[string][]*SocraticQuestion
	QuestionsByCategory  map[string][]*SocraticQuestion
	QuestionsByPrinciple map[string][]*SocraticQuestion
	QuestionsByFramework map[string][]*SocraticQuestion
}

// IntegrationConfig describes how Moly's systems use the constitution
type IntegrationConfig struct {
	SafetyChecker    SystemIntegration `yaml:"safety_checker"`
	ClarificationQs  SystemIntegration `yaml:"clarification_questions"`
	SocraticSelector SystemIntegration `yaml:"socratic_selector"`
}

type SystemIntegration struct {
	Description         string   `yaml:"description"`
	PrinciplesChecked   []string `yaml:"principles_checked"`
	PrinciplesTested    []string `yaml:"principles_tested"`
	PrinciplesCovered   []string `yaml:"principles_covered"`
	PrinciplesTargeted  []string `yaml:"principles_targeted"`
	SeverityThreshold   string   `yaml:"severity_threshold"`
}

// NewQuestionLibrary creates an empty question library with initialized maps
func NewQuestionLibrary() *QuestionLibrary {
	return &QuestionLibrary{
		AllQuestions:         make(map[string]*SocraticQuestion),
		QuestionsByApproach:  make(map[string][]*SocraticQuestion),
		QuestionsByCategory:  make(map[string][]*SocraticQuestion),
		QuestionsByPrinciple: make(map[string][]*SocraticQuestion),
		QuestionsByFramework: make(map[string][]*SocraticQuestion),
	}
}

// AddQuestion adds a question to the library and builds all indexes
func (ql *QuestionLibrary) AddQuestion(q *SocraticQuestion) error {
	if q.ID == "" {
		return fmt.Errorf("question ID cannot be empty")
	}
	if q.Text == "" {
		return fmt.Errorf("question text cannot be empty")
	}
	if q.SocraticApproach == "" {
		return fmt.Errorf("socratic_approach cannot be empty")
	}
	if q.Category == "" {
		return fmt.Errorf("category cannot be empty")
	}
	if q.TargetsPrinciple == "" {
		return fmt.Errorf("targets_principle cannot be empty")
	}

	// Add to all questions
	ql.AllQuestions[q.ID] = q

	// Index by approach
	ql.QuestionsByApproach[q.SocraticApproach] = append(
		ql.QuestionsByApproach[q.SocraticApproach], q)

	// Index by category
	ql.QuestionsByCategory[q.Category] = append(
		ql.QuestionsByCategory[q.Category], q)

	// Index by principle
	ql.QuestionsByPrinciple[q.TargetsPrinciple] = append(
		ql.QuestionsByPrinciple[q.TargetsPrinciple], q)

	// Index by framework
	ql.QuestionsByFramework[q.TargetsFramework] = append(
		ql.QuestionsByFramework[q.TargetsFramework], q)

	return nil
}

// FindByID retrieves a question by its ID
func (ql *QuestionLibrary) FindByID(id string) *SocraticQuestion {
	return ql.AllQuestions[id]
}

// FindByApproach retrieves all questions using a specific approach
func (ql *QuestionLibrary) FindByApproach(approach string) []*SocraticQuestion {
	return ql.QuestionsByApproach[approach]
}

// FindByCategory retrieves all questions in a specific category
func (ql *QuestionLibrary) FindByCategory(category string) []*SocraticQuestion {
	return ql.QuestionsByCategory[category]
}

// FindByPrinciple retrieves all questions targeting a specific principle
func (ql *QuestionLibrary) FindByPrinciple(principle string) []*SocraticQuestion {
	return ql.QuestionsByPrinciple[principle]
}

// FindByApproachAndCategory retrieves questions by both approach and category
func (ql *QuestionLibrary) FindByApproachAndCategory(approach, category string) *SocraticQuestion {
	questionsInCategory := ql.QuestionsByCategory[category]
	for _, q := range questionsInCategory {
		if q.SocraticApproach == approach {
			return q
		}
	}
	return nil
}

// GetSize returns the total number of questions in the library
func (ql *QuestionLibrary) GetSize() int {
	return len(ql.AllQuestions)
}

// GetApproaches returns all unique approaches in the library
func (ql *QuestionLibrary) GetApproaches() []string {
	approaches := make([]string, 0, len(ql.QuestionsByApproach))
	for approach := range ql.QuestionsByApproach {
		approaches = append(approaches, approach)
	}
	return approaches
}

// GetCategories returns all unique categories in the library
func (ql *QuestionLibrary) GetCategories() []string {
	categories := make([]string, 0, len(ql.QuestionsByCategory))
	for category := range ql.QuestionsByCategory {
		categories = append(categories, category)
	}
	return categories
}

