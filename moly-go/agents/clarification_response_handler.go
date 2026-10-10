package agents

import (
	"moly/database"
	"moly/models"
	"moly/schema"
)

// ClarificationResponseHandler processes user's answers to clarification questions
type ClarificationResponseHandler struct {
	temporaryFactStore *TemporaryFactStore
	contactManager     *ContactManager
	contextAttrManager *ContextAttributeManager
	userID             string
	database           *database.Database
}

// ClarificationResponseRequest represents a user's answer to a question
type ClarificationResponseRequest struct {
	QuestionID     string `json:"questionId"`
	FactID         string `json:"factId"`
	SelectedOption string `json:"selectedOption"`
	UserResponse   string `json:"userResponse"`
}

// ClarificationResponseResult shows what happened when processing the answer
type ClarificationResponseResult struct {
	QuestionAnswered bool                            `json:"questionAnswered"`
	FactID           string                          `json:"factId"`
	Status           string                          `json:"status"`             // "active", "answered", "skipped", "cancelled"
	RemainingQs      []*schema.ClarificationQuestion `json:"remainingQuestions"` // Full question objects
	CreatedContact   *models.Contact                 `json:"createdContact,omitempty"`
	SavedAttribute   *database.ContextAttribute      `json:"savedAttribute,omitempty"`
	Error            string                          `json:"error,omitempty"`
}
