package agents

import (
	"testing"
)

// Test saving a context attribute
func TestSaveAttribute(t *testing.T) {
	repo := setupTestAttributeRepo()
	manager := NewContextAttributeManager(repo)

	attr, err := manager.SaveAttribute(
		"user1",
		"conv1",
		"style",
		"casual",
		"user",
		"general",
		"I'm very casual in how I communicate",
		0.9,
	)

	if err != nil {
		t.Fatalf("SaveAttribute failed: %v", err)
	}

	if attr.ID == 0 {
		t.Errorf("Expected ID to be set, got 0")
	}

	if attr.FactValue != "casual" {
		t.Errorf("Expected FactValue 'casual', got %s", attr.FactValue)
	}

	if attr.AttributedTo != "user" {
		t.Errorf("Expected AttributedTo 'user', got %s", attr.AttributedTo)
	}
}

// Test retrieving context for subject
func TestGetContextForSubject(t *testing.T) {
	repo := setupTestAttributeRepo()
	manager := NewContextAttributeManager(repo)

	// Save multiple attributes
	manager.SaveAttribute("user1", "conv1", "style", "formal", "contact_boss_sarah", "work", "At work I'm very formal", 0.9)
	manager.SaveAttribute("user1", "conv1", "style", "casual", "contact_boss_sarah", "social", "With friends I'm casual", 0.8)
	manager.SaveAttribute("user1", "conv1", "trait", "detail-oriented", "contact_boss_sarah", "work", "Very detail-oriented", 0.85)

	context, err := manager.GetContextForSubject("user1", "contact_boss_sarah")
	if err != nil {
		t.Fatalf("GetContextForSubject failed: %v", err)
	}

	if len(context) == 0 {
		t.Errorf("Expected context, got empty map")
	}

	// Check that we have both style and trait
	if styles, ok := context["style"]; !ok || len(styles) == 0 {
		t.Errorf("Expected styles in context")
	}

	if traits, ok := context["trait"]; !ok || len(traits) == 0 {
		t.Errorf("Expected traits in context")
	}
}

// Test style retrieval for subject
func TestGetStyleForSubject(t *testing.T) {
	repo := setupTestAttributeRepo()
	manager := NewContextAttributeManager(repo)

	manager.SaveAttribute("user1", "conv1", "style", "formal", "user", "work", "", 0.9)
	manager.SaveAttribute("user1", "conv1", "style", "detailed", "user", "work", "", 0.8)

	styles, err := manager.GetStyleForSubject("user1", "user")
	if err != nil {
		t.Fatalf("GetStyleForSubject failed: %v", err)
	}

	if len(styles) != 2 {
		t.Errorf("Expected 2 styles, got %d", len(styles))
	}
}

// Test checking if attribute exists
func TestHasAttribute(t *testing.T) {
	repo := setupTestAttributeRepo()
	manager := NewContextAttributeManager(repo)

	manager.SaveAttribute("user1", "conv1", "style", "casual", "user", "general", "", 0.9)

	has, err := manager.HasAttribute("user1", "user", "style", "casual")
	if err != nil {
		t.Fatalf("HasAttribute failed: %v", err)
	}

	if !has {
		t.Errorf("Expected to find attribute")
	}

	has, err = manager.HasAttribute("user1", "user", "style", "formal")
	if err != nil {
		t.Fatalf("HasAttribute failed: %v", err)
	}

	if has {
		t.Errorf("Expected not to find attribute")
	}
}

// Test getting all user context
func TestGetAllUserContext(t *testing.T) {
	repo := setupTestAttributeRepo()
	manager := NewContextAttributeManager(repo)

	// Save attributes for different subjects
	manager.SaveAttribute("user1", "conv1", "style", "formal", "user", "general", "", 0.9)
	manager.SaveAttribute("user1", "conv1", "style", "casual", "contact_manager_bob", "social", "", 0.8)
	manager.SaveAttribute("user1", "conv1", "trait", "organized", "contact_manager_bob", "work", "", 0.85)

	context, err := manager.GetAllUserContext("user1")
	if err != nil {
		t.Fatalf("GetAllUserContext failed: %v", err)
	}

	if len(context) == 0 {
		t.Errorf("Expected context for multiple subjects")
	}

	if _, ok := context["user"]; !ok {
		t.Errorf("Expected user context")
	}

	if _, ok := context["contact_manager_bob"]; !ok {
		t.Errorf("Expected contact_manager_bob context")
	}
}

// Test conversation context retrieval
func TestGetConversationContext(t *testing.T) {
	repo := setupTestAttributeRepo()
	manager := NewContextAttributeManager(repo)

	manager.SaveAttribute("user1", "conv1", "style", "casual", "user", "general", "", 0.9)
	manager.SaveAttribute("user1", "conv1", "trait", "organized", "contact_alice", "work", "", 0.85)

	attrs, err := manager.GetConversationContext("conv1")
	if err != nil {
		t.Fatalf("GetConversationContext failed: %v", err)
	}

	if len(attrs) != 2 {
		t.Errorf("Expected 2 attributes in conversation, got %d", len(attrs))
	}
}

