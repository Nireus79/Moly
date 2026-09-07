package agents_test

import (
	"testing"

	"moly/agents"
	"moly/models"
)

func setupContextManager(t *testing.T) models.ContextManagerAgent {
	manager, err := agents.NewContextManager("user123")
	if err != nil {
		t.Fatalf("Failed to create context manager: %v", err)
	}

	return manager
}

func TestContextManagerCreation(t *testing.T) {
	manager := setupContextManager(t)

	if manager == nil {
		t.Error("Expected context manager")
	}
}

func TestContextManagerEmptyUserID(t *testing.T) {
	_, err := agents.NewContextManager("")
	if err == nil {
		t.Error("Expected error for empty userID")
	}
}

func TestAboutMeStorage(t *testing.T) {
	manager := setupContextManager(t)

	aboutMe := &models.AboutMe{
		UserID:            "user123",
		CommunicationStyle: "friendly",
		Notes:             "I like to be direct",
	}

	err := manager.SetAboutMe("user123", aboutMe)
	if err != nil {
		t.Fatalf("SetAboutMe() error = %v", err)
	}

	retrieved, err := manager.GetAboutMe("user123")
	if err != nil {
		t.Fatalf("GetAboutMe() error = %v", err)
	}

	if retrieved == nil {
		t.Error("Expected AboutMe to be retrieved")
	}

	if retrieved.UserID != "user123" {
		t.Errorf("UserID = %s, want user123", retrieved.UserID)
	}
}

func TestAboutMeNil(t *testing.T) {
	manager := setupContextManager(t)

	err := manager.SetAboutMe("user123", nil)
	if err == nil {
		t.Error("Expected error for nil AboutMe")
	}
}

func TestContactManagement(t *testing.T) {
	manager := setupContextManager(t)

	contact := &models.Contact{
		Name:         "Alice",
		Relationship: "close_friend",
	}

	created, err := manager.CreateContact("user123", contact)
	if err != nil {
		t.Fatalf("CreateContact() error = %v", err)
	}

	if created == nil {
		t.Error("Expected created contact")
	}

	if created.UserID != "user123" {
		t.Errorf("UserID = %s, want user123", created.UserID)
	}
}

func TestGetContact(t *testing.T) {
	manager := setupContextManager(t)

	contact, err := manager.GetContact("user123", "contact1")
	if err != nil {
		t.Fatalf("GetContact() error = %v", err)
	}

	if contact == nil {
		t.Error("Expected contact")
	}

	if contact.UserID != "user123" {
		t.Errorf("UserID = %s, want user123", contact.UserID)
	}
}

func TestGetContacts(t *testing.T) {
	manager := setupContextManager(t)

	contacts, err := manager.GetContacts("user123")
	if err != nil {
		t.Fatalf("GetContacts() error = %v", err)
	}

	if contacts == nil {
		t.Error("Expected contacts slice")
	}
}

func TestUpdateContact(t *testing.T) {
	manager := setupContextManager(t)

	updates := models.Contact{
		Name: "Alice Updated",
	}

	err := manager.UpdateContact("user123", "contact1", updates)
	if err != nil {
		t.Fatalf("UpdateContact() error = %v", err)
	}
}

func TestConversationHistory(t *testing.T) {
	manager := setupContextManager(t)

	message := &models.Message{
		ConversationID: "conv1",
		Role:          "user",
		Content:       "Hello!",
		Type:          "message",
	}

	err := manager.AppendMessage("conv1", message)
	if err != nil {
		t.Fatalf("AppendMessage() error = %v", err)
	}

	if message.Timestamp == 0 {
		t.Error("Expected Timestamp to be set")
	}

	if message.ConversationID != "conv1" {
		t.Errorf("ConversationID = %s, want conv1", message.ConversationID)
	}
}

func TestReflectionWorkflow(t *testing.T) {
	manager := setupContextManager(t)

	reflection := &models.Reflection{
		ConversationID: "conv1",
		Characteristics: []string{"patient", "kind"},
		Interests:      []string{"reading", "hiking"},
	}

	err := manager.SaveReflection("conv1", reflection)
	if err != nil {
		t.Fatalf("SaveReflection() error = %v", err)
	}

	if reflection.Status != "pending_approval" {
		t.Errorf("Status = %s, want pending_approval", reflection.Status)
	}
}

func TestApproveReflection(t *testing.T) {
	manager := setupContextManager(t)

	reflection := &models.Reflection{
		ConversationID: "conv1",
	}

	err := manager.ApproveReflection("conv1", reflection)
	if err != nil {
		t.Fatalf("ApproveReflection() error = %v", err)
	}

	if reflection.Status != "approved" {
		t.Errorf("Status = %s, want approved", reflection.Status)
	}

	if reflection.ApprovedAt == 0 {
		t.Error("Expected ApprovedAt to be set")
	}
}

func TestGetRelevantContext(t *testing.T) {
	manager := setupContextManager(t)

	context, err := manager.GetRelevantContext("conv1", "user123")
	if err != nil {
		t.Fatalf("GetRelevantContext() error = %v", err)
	}

	if context == nil {
		t.Error("Expected context")
	}

	if context.ContextQuality == "" {
		t.Error("Expected ContextQuality to be set")
	}
}
