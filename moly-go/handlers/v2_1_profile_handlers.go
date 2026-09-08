package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"moly/services"
)

// ProfileHandlers manages all profile-related endpoints
type ProfileHandlers struct {
	profileService *services.ProfileService
}

// NewProfileHandlers creates handlers for profile endpoints
func NewProfileHandlers(profileService *services.ProfileService) *ProfileHandlers {
	if profileService == nil {
		log.Fatal("[ProfileHandlers] ProfileService cannot be nil")
	}
	return &ProfileHandlers{
		profileService: profileService,
	}
}

// ErrorResponse wraps error responses
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// SuccessResponse wraps success responses
type SuccessResponse struct {
	Data      interface{} `json:"data"`
	Message   string      `json:"message"`
	Timestamp int64       `json:"timestamp"`
}

// RegisterRoutes registers all profile routes
func (ph *ProfileHandlers) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v2.1/profile", ph.handleGetProfile)
	mux.HandleFunc("GET /api/v2.1/profile/about-me", ph.handleGetAboutMe)
	mux.HandleFunc("GET /api/v2.1/profile/contacts", ph.handleGetContacts)
	mux.HandleFunc("GET /api/v2.1/profile/goals", ph.handleGetGoals)
	mux.HandleFunc("GET /api/v2.1/profile/patterns", ph.handleGetPatterns)
	mux.HandleFunc("GET /api/v2.1/profile/learnings", ph.handleGetLearnings)
	mux.HandleFunc("GET /api/v2.1/profile/reflections", ph.handleGetReflections)

	mux.HandleFunc("POST /api/v2.1/goals", ph.handleCreateGoal)
	mux.HandleFunc("PATCH /api/v2.1/goals/{id}", ph.handleUpdateGoal)

	mux.HandleFunc("POST /api/v2.1/reflections", ph.handleCreateReflection)

	mux.HandleFunc("POST /api/v2.1/learnings/{id}/confirm", ph.handleConfirmLearning)
	mux.HandleFunc("POST /api/v2.1/learnings/{id}/reject", ph.handleRejectLearning)
}

// getUserID extracts userID from request (header or context)
func (ph *ProfileHandlers) getUserID(r *http.Request) (string, error) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		return "", fmt.Errorf("X-User-ID header required")
	}
	return userID, nil
}

// respondJSON writes JSON response
func (ph *ProfileHandlers) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError writes error response
func (ph *ProfileHandlers) respondError(w http.ResponseWriter, status int, message string) {
	ph.respondJSON(w, status, ErrorResponse{
		Error:   http.StatusText(status),
		Message: message,
		Code:    status,
	})
}

// handleGetProfile GET /api/v2.1/profile
func (ph *ProfileHandlers) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	userID, err := ph.getUserID(r)
	if err != nil {
		ph.respondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	profile, err := ph.profileService.GetUserProfile(userID)
	if err != nil {
		log.Printf("[ProfileHandlers] Error getting profile: %v", err)
		ph.respondError(w, http.StatusInternalServerError, "Failed to retrieve profile")
		return
	}

	ph.respondJSON(w, http.StatusOK, SuccessResponse{
		Data:      profile,
		Message:   "Profile retrieved",
		Timestamp: int64(0), // Should be current time in ms
	})
}

// handleGetAboutMe GET /api/v2.1/profile/about-me
func (ph *ProfileHandlers) handleGetAboutMe(w http.ResponseWriter, r *http.Request) {
	userID, err := ph.getUserID(r)
	if err != nil {
		ph.respondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	aboutMe, err := ph.profileService.GetAboutMe(userID)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			ph.respondJSON(w, http.StatusOK, SuccessResponse{
				Data:      nil,
				Message:   "No about-me profile yet",
				Timestamp: 0,
			})
			return
		}
		log.Printf("[ProfileHandlers] Error getting AboutMe: %v", err)
		ph.respondError(w, http.StatusInternalServerError, "Failed to retrieve about-me profile")
		return
	}

	ph.respondJSON(w, http.StatusOK, SuccessResponse{
		Data:      aboutMe,
		Message:   "About-me profile retrieved",
		Timestamp: 0,
	})
}

// handleGetContacts GET /api/v2.1/profile/contacts
func (ph *ProfileHandlers) handleGetContacts(w http.ResponseWriter, r *http.Request) {
	userID, err := ph.getUserID(r)
	if err != nil {
		ph.respondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	contacts, err := ph.profileService.GetContacts(userID)
	if err != nil {
		log.Printf("[ProfileHandlers] Error getting contacts: %v", err)
		ph.respondError(w, http.StatusInternalServerError, "Failed to retrieve contacts")
		return
	}

	if contacts == nil {
		contacts = []services.ContactProfile{}
	}

	ph.respondJSON(w, http.StatusOK, SuccessResponse{
		Data:      contacts,
		Message:   fmt.Sprintf("Retrieved %d contacts", len(contacts)),
		Timestamp: 0,
	})
}

// handleGetPatterns GET /api/v2.1/profile/patterns
func (ph *ProfileHandlers) handleGetPatterns(w http.ResponseWriter, r *http.Request) {
	userID, err := ph.getUserID(r)
	if err != nil {
		ph.respondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	patterns, err := ph.profileService.GetPatterns(userID)
	if err != nil {
		log.Printf("[ProfileHandlers] Error getting patterns: %v", err)
		ph.respondError(w, http.StatusInternalServerError, "Failed to retrieve patterns")
		return
	}

	if patterns == nil {
		patterns = []services.PatternProfile{}
	}

	ph.respondJSON(w, http.StatusOK, SuccessResponse{
		Data:      patterns,
		Message:   fmt.Sprintf("Retrieved %d patterns", len(patterns)),
		Timestamp: 0,
	})
}

// handleGetGoals GET /api/v2.1/profile/goals
func (ph *ProfileHandlers) handleGetGoals(w http.ResponseWriter, r *http.Request) {
	userID, err := ph.getUserID(r)
	if err != nil {
		ph.respondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	goals, err := ph.profileService.GetGoals(userID)
	if err != nil {
		log.Printf("[ProfileHandlers] Error getting goals: %v", err)
		ph.respondError(w, http.StatusInternalServerError, "Failed to retrieve goals")
		return
	}

	if goals == nil {
		goals = []services.GoalProfile{}
	}

	ph.respondJSON(w, http.StatusOK, SuccessResponse{
		Data:      goals,
		Message:   fmt.Sprintf("Retrieved %d goals", len(goals)),
		Timestamp: 0,
	})
}

// handleGetLearnings GET /api/v2.1/profile/learnings
func (ph *ProfileHandlers) handleGetLearnings(w http.ResponseWriter, r *http.Request) {
	userID, err := ph.getUserID(r)
	if err != nil {
		ph.respondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	learnings, err := ph.profileService.GetLearnings(userID)
	if err != nil {
		log.Printf("[ProfileHandlers] Error getting learnings: %v", err)
		ph.respondError(w, http.StatusInternalServerError, "Failed to retrieve learnings")
		return
	}

	if learnings == nil {
		learnings = []services.LearningProfile{}
	}

	ph.respondJSON(w, http.StatusOK, SuccessResponse{
		Data:      learnings,
		Message:   fmt.Sprintf("Retrieved %d learnings", len(learnings)),
		Timestamp: 0,
	})
}

// handleGetReflections GET /api/v2.1/profile/reflections
func (ph *ProfileHandlers) handleGetReflections(w http.ResponseWriter, r *http.Request) {
	userID, err := ph.getUserID(r)
	if err != nil {
		ph.respondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	reflections, err := ph.profileService.GetReflections(userID)
	if err != nil {
		log.Printf("[ProfileHandlers] Error getting reflections: %v", err)
		ph.respondError(w, http.StatusInternalServerError, "Failed to retrieve reflections")
		return
	}

	if reflections == nil {
		reflections = []services.ReflectionEntry{}
	}

	ph.respondJSON(w, http.StatusOK, SuccessResponse{
		Data:      reflections,
		Message:   fmt.Sprintf("Retrieved %d reflections", len(reflections)),
		Timestamp: 0,
	})
}

// CreateGoalRequest represents goal creation payload
type CreateGoalRequest struct {
	Goal       string `json:"goal"`
	Category   string `json:"category"`
	TargetDate *int64 `json:"targetDate"`
}

// handleCreateGoal POST /api/v2.1/goals
func (ph *ProfileHandlers) handleCreateGoal(w http.ResponseWriter, r *http.Request) {
	_, err := ph.getUserID(r)
	if err != nil {
		ph.respondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	var req CreateGoalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ph.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Goal == "" || req.Category == "" {
		ph.respondError(w, http.StatusBadRequest, "Goal and category required")
		return
	}

	// For now, this endpoint would create a goal via ProfileService or GoalService
	// Implementation would depend on whether we add a CreateGoal method

	ph.respondJSON(w, http.StatusCreated, SuccessResponse{
		Data:      map[string]interface{}{"message": "Goal creation not yet implemented"},
		Message:   "Goal creation endpoint ready for implementation",
		Timestamp: 0,
	})
}

// UpdateGoalRequest represents goal update payload
type UpdateGoalRequest struct {
	ProgressNotes string `json:"progressNotes"`
	Status        string `json:"status"`
}

// handleUpdateGoal PATCH /api/v2.1/goals/{id}
func (ph *ProfileHandlers) handleUpdateGoal(w http.ResponseWriter, r *http.Request) {
	userID, err := ph.getUserID(r)
	if err != nil {
		ph.respondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// Extract goal ID from URL
	goalIDStr := r.PathValue("id")
	goalID, err := strconv.Atoi(goalIDStr)
	if err != nil {
		ph.respondError(w, http.StatusBadRequest, "Invalid goal ID")
		return
	}

	var req UpdateGoalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ph.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Status == "" {
		ph.respondError(w, http.StatusBadRequest, "Status required")
		return
	}

	err = ph.profileService.UpdateGoalProgress(userID, goalID, req.ProgressNotes, req.Status)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			ph.respondError(w, http.StatusNotFound, "Goal not found")
			return
		}
		log.Printf("[ProfileHandlers] Error updating goal: %v", err)
		ph.respondError(w, http.StatusInternalServerError, "Failed to update goal")
		return
	}

	ph.respondJSON(w, http.StatusOK, SuccessResponse{
		Data:      map[string]interface{}{"goalId": goalID},
		Message:   "Goal updated",
		Timestamp: 0,
	})
}

// CreateReflectionRequest represents reflection creation payload
type CreateReflectionRequest struct {
	Content        string   `json:"content"`
	EntryType      string   `json:"entryType"`
	Tags           []string `json:"tags"`
	AboutContactID *int     `json:"aboutContactId"`
}

// handleCreateReflection POST /api/v2.1/reflections
func (ph *ProfileHandlers) handleCreateReflection(w http.ResponseWriter, r *http.Request) {
	userID, err := ph.getUserID(r)
	if err != nil {
		ph.respondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	var req CreateReflectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ph.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Content == "" || req.EntryType == "" {
		ph.respondError(w, http.StatusBadRequest, "Content and entryType required")
		return
	}

	id, err := ph.profileService.AddReflection(userID, req.Content, req.EntryType, req.Tags, req.AboutContactID)
	if err != nil {
		log.Printf("[ProfileHandlers] Error adding reflection: %v", err)
		ph.respondError(w, http.StatusInternalServerError, "Failed to add reflection")
		return
	}

	ph.respondJSON(w, http.StatusCreated, SuccessResponse{
		Data:      map[string]interface{}{"reflectionId": id},
		Message:   "Reflection added",
		Timestamp: 0,
	})
}

// LearningActionRequest represents learning action payload
type LearningActionRequest struct {
	Message string `json:"message"`
}

// handleConfirmLearning POST /api/v2.1/learnings/{id}/confirm
func (ph *ProfileHandlers) handleConfirmLearning(w http.ResponseWriter, r *http.Request) {
	userID, err := ph.getUserID(r)
	if err != nil {
		ph.respondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	learningIDStr := r.PathValue("id")
	learningID, err := strconv.Atoi(learningIDStr)
	if err != nil {
		ph.respondError(w, http.StatusBadRequest, "Invalid learning ID")
		return
	}

	err = ph.profileService.ConfirmLearning(userID, learningID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			ph.respondError(w, http.StatusNotFound, "Learning not found")
			return
		}
		log.Printf("[ProfileHandlers] Error confirming learning: %v", err)
		ph.respondError(w, http.StatusInternalServerError, "Failed to confirm learning")
		return
	}

	ph.respondJSON(w, http.StatusOK, SuccessResponse{
		Data:      map[string]interface{}{"learningId": learningID},
		Message:   "Learning confirmed",
		Timestamp: 0,
	})
}

// handleRejectLearning POST /api/v2.1/learnings/{id}/reject
func (ph *ProfileHandlers) handleRejectLearning(w http.ResponseWriter, r *http.Request) {
	userID, err := ph.getUserID(r)
	if err != nil {
		ph.respondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	learningIDStr := r.PathValue("id")
	learningID, err := strconv.Atoi(learningIDStr)
	if err != nil {
		ph.respondError(w, http.StatusBadRequest, "Invalid learning ID")
		return
	}

	err = ph.profileService.RejectLearning(userID, learningID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			ph.respondError(w, http.StatusNotFound, "Learning not found")
			return
		}
		log.Printf("[ProfileHandlers] Error rejecting learning: %v", err)
		ph.respondError(w, http.StatusInternalServerError, "Failed to reject learning")
		return
	}

	ph.respondJSON(w, http.StatusOK, SuccessResponse{
		Data:      map[string]interface{}{"learningId": learningID},
		Message:   "Learning rejected",
		Timestamp: 0,
	})
}
