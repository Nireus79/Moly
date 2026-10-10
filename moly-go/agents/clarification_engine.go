package agents

import (
	"fmt"
	"log"
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

// ProcessResponse handles user's answer to a clarification question
// Returns the resolved value or additional questions if needed
func (e *ClarificationEngine) ProcessResponse(response ClarificationResponse) (string, error) {
	log.Printf("[Moly] ClarificationEngine: processing response to question %s", response.QuestionID)

	if response.SelectedOption != "" {
		log.Printf("[Moly]   User selected option: %s", response.SelectedOption)
		return response.SelectedOption, nil
	}

	if response.UserResponse != "" {
		log.Printf("[Moly]   User provided text response: %s", response.UserResponse)
		return response.UserResponse, nil
	}

	return "", fmt.Errorf("no response provided")
}
