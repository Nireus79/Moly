package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
	"moly/models"
)

// LoadConstitution loads the constitution from YAML file
func LoadConstitution(filepath string) (*models.Constitution, error) {
	if filepath == "" {
		filepath = "config/constitution.yaml"
	}

	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read constitution file: %w", err)
	}

	var constitution models.Constitution
	err = yaml.Unmarshal(data, &constitution)
	if err != nil {
		return nil, fmt.Errorf("failed to parse constitution YAML: %w", err)
	}

	// Validate constitution
	if err := validateConstitution(&constitution); err != nil {
		return nil, err
	}

	log.Printf("[Config] Constitution loaded: %d principles, %d frameworks",
		len(constitution.SupremePrinciples),
		len(constitution.EthicalFrameworks))

	return &constitution, nil
}

// validateConstitution performs basic validation on the constitution
func validateConstitution(c *models.Constitution) error {
	if len(c.SupremePrinciples) == 0 {
		return fmt.Errorf("constitution must have at least one principle")
	}

	// Verify all principles have required fields
	for i, p := range c.SupremePrinciples {
		if p.ID == "" {
			return fmt.Errorf("principle %d is missing ID", i)
		}
		if p.Name == "" {
			return fmt.Errorf("principle %s is missing Name", p.ID)
		}
		if p.Severity == "" {
			return fmt.Errorf("principle %s is missing Severity", p.ID)
		}
	}

	if len(c.EthicalFrameworks) == 0 {
		return fmt.Errorf("constitution must have at least one ethical framework")
	}

	// Verify all frameworks have required fields
	for i, f := range c.EthicalFrameworks {
		if f.ID == "" {
			return fmt.Errorf("framework %d is missing ID", i)
		}
		if f.Name == "" {
			return fmt.Errorf("framework %s is missing Name", f.ID)
		}
	}

	return nil
}

// LoadQuestionLibrary loads all question files and builds a library
func LoadQuestionLibrary(configDir string) (*models.QuestionLibrary, error) {
	if configDir == "" {
		configDir = "config"
	}

	library := models.NewQuestionLibrary()

	// List of question category files to load
	questionFiles := []string{
		"questions_stakeholder.yaml",
		"questions_consequence.yaml",
		"questions_principle.yaml",
		"questions_assumption.yaml",
		"questions_alternative.yaml",
	}

	totalLoaded := 0

	for _, filename := range questionFiles {
		filepath := filepath.Join(configDir, filename)
		questionsInFile, err := loadQuestionsFromFile(filepath)
		if err != nil {
			return nil, fmt.Errorf("failed to load %s: %w", filename, err)
		}

		// Add each question to the library
		for _, q := range questionsInFile {
			if err := library.AddQuestion(q); err != nil {
				return nil, fmt.Errorf("invalid question %s: %w", q.ID, err)
			}
		}

		totalLoaded += len(questionsInFile)
		log.Printf("[Config] Loaded %d questions from %s", len(questionsInFile), filename)
	}

	if library.GetSize() == 0 {
		return nil, fmt.Errorf("no questions loaded from any file")
	}

	log.Printf("[Config] Question library complete: %d total questions, %d approaches, %d categories",
		library.GetSize(),
		len(library.GetApproaches()),
		len(library.GetCategories()))

	return library, nil
}

// loadQuestionsFromFile loads questions from a single YAML file
func loadQuestionsFromFile(filepath string) ([]*models.SocraticQuestion, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse the YAML structure with "questions" key
	var fileContent struct {
		Questions []*models.SocraticQuestion `yaml:"questions"`
	}

	err = yaml.Unmarshal(data, &fileContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return fileContent.Questions, nil
}

// ValidateQuestionLibrary performs comprehensive validation
func ValidateQuestionLibrary(library *models.QuestionLibrary) error {
	if library.GetSize() != 40 {
		log.Printf("[Warning] Expected 40 questions, found %d", library.GetSize())
	}

	// Check for duplicate IDs (shouldn't happen but verify)
	seen := make(map[string]bool)
	for id := range library.AllQuestions {
		if seen[id] {
			return fmt.Errorf("duplicate question ID: %s", id)
		}
		seen[id] = true
	}

	// Check that all approaches are represented
	requiredApproaches := map[string]bool{
		"identifying_stakeholders": false,
		"exploring_consequences":   false,
		"testing_universality":     false,
		"revealing_assumptions":    false,
		"exploring_alternatives":   false,
	}

	for _, question := range library.AllQuestions {
		requiredApproaches[question.SocraticApproach] = true
	}

	for approach, found := range requiredApproaches {
		if !found {
			return fmt.Errorf("missing questions for approach: %s", approach)
		}
	}

	log.Printf("[Config] Question library validation passed")
	return nil
}

