package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"moly/database"
	"moly/services"
)

// setupHandlersTest creates test environment
func setupHandlersTest(t *testing.T) *ProfileHandlers {
	db, err := database.Init(":memory:", "test-key")
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	profileService := services.NewProfileService(db)
	return NewProfileHandlers(profileService)
}

// TestProfileHandlersGetProfileMissingAuth tests missing auth header
func TestProfileHandlersGetProfileMissingAuth(t *testing.T) {
	handlers := setupHandlersTest(t)

	req := httptest.NewRequest("GET", "/api/v2.1/profile", nil)
	w := httptest.NewRecorder()

	// Create mux and register routes
	mux := http.NewServeMux()
	handlers.RegisterRoutes(mux)

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", w.Code)
	}

	t.Log("✓ Missing auth header handled correctly")
}

// TestProfileHandlersGetProfile tests retrieving full profile
func TestProfileHandlersGetProfile(t *testing.T) {
	handlers := setupHandlersTest(t)

	req := httptest.NewRequest("GET", "/api/v2.1/profile", nil)
	req.Header.Set("X-User-ID", "test_user")
	w := httptest.NewRecorder()

	mux := http.NewServeMux()
	handlers.RegisterRoutes(mux)

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["data"] == nil {
		t.Error("Expected data in response")
	}

	t.Log("✓ GetProfile endpoint works")
}

// TestProfileHandlersGetAboutMe tests retrieving about-me profile
func TestProfileHandlersGetAboutMe(t *testing.T) {
	handlers := setupHandlersTest(t)

	req := httptest.NewRequest("GET", "/api/v2.1/profile/about-me", nil)
	req.Header.Set("X-User-ID", "test_user")
	w := httptest.NewRecorder()

	mux := http.NewServeMux()
	handlers.RegisterRoutes(mux)

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	t.Log("✓ GetAboutMe endpoint works")
}

// TestProfileHandlersGetContacts tests retrieving contacts
func TestProfileHandlersGetContacts(t *testing.T) {
	handlers := setupHandlersTest(t)

	req := httptest.NewRequest("GET", "/api/v2.1/profile/contacts", nil)
	req.Header.Set("X-User-ID", "test_user")
	w := httptest.NewRecorder()

	mux := http.NewServeMux()
	handlers.RegisterRoutes(mux)

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["message"] == nil {
		t.Error("Expected message in response")
	}

	t.Log("✓ GetContacts endpoint works")
}

// TestProfileHandlersGetPatterns tests retrieving patterns
func TestProfileHandlersGetPatterns(t *testing.T) {
	handlers := setupHandlersTest(t)

	req := httptest.NewRequest("GET", "/api/v2.1/profile/patterns", nil)
	req.Header.Set("X-User-ID", "test_user")
	w := httptest.NewRecorder()

	mux := http.NewServeMux()
	handlers.RegisterRoutes(mux)

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	t.Log("✓ GetPatterns endpoint works")
}

// TestProfileHandlersGetGoals tests retrieving goals
func TestProfileHandlersGetGoals(t *testing.T) {
	handlers := setupHandlersTest(t)

	req := httptest.NewRequest("GET", "/api/v2.1/profile/goals", nil)
	req.Header.Set("X-User-ID", "test_user")
	w := httptest.NewRecorder()

	mux := http.NewServeMux()
	handlers.RegisterRoutes(mux)

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	t.Log("✓ GetGoals endpoint works")
}

// TestProfileHandlersGetLearnings tests retrieving learnings
func TestProfileHandlersGetLearnings(t *testing.T) {
	handlers := setupHandlersTest(t)

	req := httptest.NewRequest("GET", "/api/v2.1/profile/learnings", nil)
	req.Header.Set("X-User-ID", "test_user")
	w := httptest.NewRecorder()

	mux := http.NewServeMux()
	handlers.RegisterRoutes(mux)

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	t.Log("✓ GetLearnings endpoint works")
}

// TestProfileHandlersGetReflections tests retrieving reflections
func TestProfileHandlersGetReflections(t *testing.T) {
	handlers := setupHandlersTest(t)

	req := httptest.NewRequest("GET", "/api/v2.1/profile/reflections", nil)
	req.Header.Set("X-User-ID", "test_user")
	w := httptest.NewRecorder()

	mux := http.NewServeMux()
	handlers.RegisterRoutes(mux)

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	t.Log("✓ GetReflections endpoint works")
}

// TestProfileHandlersCreateReflection tests creating reflection
func TestProfileHandlersCreateReflection(t *testing.T) {
	handlers := setupHandlersTest(t)

	body := CreateReflectionRequest{
		Content:   "Interesting conversation about assertiveness",
		EntryType: "reflection",
		Tags:      []string{"work", "assertiveness"},
	}

	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v2.1/reflections", bytes.NewReader(bodyBytes))
	req.Header.Set("X-User-ID", "test_user")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux := http.NewServeMux()
	handlers.RegisterRoutes(mux)

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected 201, got %d", w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["data"] == nil {
		t.Error("Expected data in response")
	}

	t.Log("✓ CreateReflection endpoint works")
}

// TestProfileHandlersCreateReflectionValidation tests validation
func TestProfileHandlersCreateReflectionValidation(t *testing.T) {
	handlers := setupHandlersTest(t)

	// Missing content
	body := CreateReflectionRequest{
		Content:   "",
		EntryType: "reflection",
	}

	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v2.1/reflections", bytes.NewReader(bodyBytes))
	req.Header.Set("X-User-ID", "test_user")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux := http.NewServeMux()
	handlers.RegisterRoutes(mux)

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for missing content, got %d", w.Code)
	}

	t.Log("✓ CreateReflection validation works")
}

// TestProfileHandlersUpdateGoal tests updating goal
func TestProfileHandlersUpdateGoal(t *testing.T) {
	handlers := setupHandlersTest(t)

	body := UpdateGoalRequest{
		ProgressNotes: "Made progress today",
		Status:        "active",
	}

	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("PATCH", "/api/v2.1/goals/999", bytes.NewReader(bodyBytes))
	req.Header.Set("X-User-ID", "test_user")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux := http.NewServeMux()
	handlers.RegisterRoutes(mux)

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404 for non-existent goal, got %d", w.Code)
	}

	t.Log("✓ UpdateGoal endpoint works")
}

// TestProfileHandlersUpdateGoalInvalidID tests invalid goal ID
func TestProfileHandlersUpdateGoalInvalidID(t *testing.T) {
	handlers := setupHandlersTest(t)

	body := UpdateGoalRequest{
		ProgressNotes: "Progress",
		Status:        "active",
	}

	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("PATCH", "/api/v2.1/goals/invalid", bytes.NewReader(bodyBytes))
	req.Header.Set("X-User-ID", "test_user")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mux := http.NewServeMux()
	handlers.RegisterRoutes(mux)

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid ID, got %d", w.Code)
	}

	t.Log("✓ UpdateGoal ID validation works")
}

// TestProfileHandlersConfirmLearning tests confirming learning
func TestProfileHandlersConfirmLearning(t *testing.T) {
	handlers := setupHandlersTest(t)

	req := httptest.NewRequest("POST", "/api/v2.1/learnings/999/confirm", nil)
	req.Header.Set("X-User-ID", "test_user")
	w := httptest.NewRecorder()

	mux := http.NewServeMux()
	handlers.RegisterRoutes(mux)

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404 for non-existent learning, got %d", w.Code)
	}

	t.Log("✓ ConfirmLearning endpoint works")
}

// TestProfileHandlersRejectLearning tests rejecting learning
func TestProfileHandlersRejectLearning(t *testing.T) {
	handlers := setupHandlersTest(t)

	req := httptest.NewRequest("POST", "/api/v2.1/learnings/999/reject", nil)
	req.Header.Set("X-User-ID", "test_user")
	w := httptest.NewRecorder()

	mux := http.NewServeMux()
	handlers.RegisterRoutes(mux)

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404 for non-existent learning, got %d", w.Code)
	}

	t.Log("✓ RejectLearning endpoint works")
}

// TestProfileHandlersResponseFormat tests response format consistency
func TestProfileHandlersResponseFormat(t *testing.T) {
	handlers := setupHandlersTest(t)

	req := httptest.NewRequest("GET", "/api/v2.1/profile", nil)
	req.Header.Set("X-User-ID", "test_user")
	w := httptest.NewRecorder()

	mux := http.NewServeMux()
	handlers.RegisterRoutes(mux)

	mux.ServeHTTP(w, req)

	// Check content type
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Expected application/json, got %s", ct)
	}

	// Parse response
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if response["data"] == nil {
		t.Error("Expected 'data' field in response")
	}

	if response["message"] == nil {
		t.Error("Expected 'message' field in response")
	}

	t.Log("✓ Response format is correct")
}

// TestProfileHandlersGate - Complete handlers verification
func TestProfileHandlersGate(t *testing.T) {
	tests := []struct {
		name string
		test func(*testing.T)
	}{
		{"GetProfileMissingAuth", TestProfileHandlersGetProfileMissingAuth},
		{"GetProfile", TestProfileHandlersGetProfile},
		{"GetAboutMe", TestProfileHandlersGetAboutMe},
		{"GetContacts", TestProfileHandlersGetContacts},
		{"GetPatterns", TestProfileHandlersGetPatterns},
		{"GetGoals", TestProfileHandlersGetGoals},
		{"GetLearnings", TestProfileHandlersGetLearnings},
		{"GetReflections", TestProfileHandlersGetReflections},
		{"CreateReflection", TestProfileHandlersCreateReflection},
		{"CreateReflectionValidation", TestProfileHandlersCreateReflectionValidation},
		{"UpdateGoal", TestProfileHandlersUpdateGoal},
		{"UpdateGoalInvalidID", TestProfileHandlersUpdateGoalInvalidID},
		{"ConfirmLearning", TestProfileHandlersConfirmLearning},
		{"RejectLearning", TestProfileHandlersRejectLearning},
		{"ResponseFormat", TestProfileHandlersResponseFormat},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.test(t)
		})
	}

	t.Log("\n✓ Profile handlers gate PASSED - REST API ready")
}
