package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleFrontendErrors(t *testing.T) {
	testCases := []struct {
		name           string
		method         string
		requestBody    interface{}
		expectedStatus int
		expectedField  string
	}{
		{
			name:           "POST with valid errors",
			method:         http.MethodPost,
			requestBody:    map[string]interface{}{"errors": []map[string]interface{}{{"message": "test"}}, "session": "s1", "extension_version": "1.0"},
			expectedStatus: http.StatusOK,
			expectedField:  "success",
		},
		{
			name:           "POST with empty errors",
			method:         http.MethodPost,
			requestBody:    map[string]interface{}{"errors": []map[string]interface{}{}, "session": "s1"},
			expectedStatus: http.StatusOK,
			expectedField:  "success",
		},
		{
			name:           "GET not allowed",
			method:         http.MethodGet,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "DELETE not allowed",
			method:         http.MethodDelete,
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var body []byte
			if tc.requestBody != nil {
				body, _ = json.Marshal(tc.requestBody)
			}

			req := httptest.NewRequest(tc.method, "/api/frontend-errors", bytes.NewReader(body))
			if tc.requestBody != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()

			handleFrontendErrors(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, w.Code)
			}

			if tc.expectedField != "" {
				var resp map[string]interface{}
				json.NewDecoder(w.Body).Decode(&resp)
				if _, ok := resp[tc.expectedField]; !ok {
					t.Errorf("Response missing field: %s", tc.expectedField)
				}
			}
		})
	}
}

func TestHandleFrontendErrorsInvalidJSON(t *testing.T) {
	body := []byte("invalid json")
	req := httptest.NewRequest(http.MethodPost, "/api/frontend-errors", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleFrontendErrors(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleProvidersGet(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/providers", nil)
	w := httptest.NewRecorder()

	handleProviders(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code: expected 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if _, ok := response["providers"]; !ok {
		t.Error("Response should contain 'providers'")
	}
}

func TestHandleProvidersOnlyGet(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/providers", nil)
	w := httptest.NewRecorder()

	handleProviders(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status code: expected 405, got %d", w.Code)
	}
}

func TestHandleProvidersInvalidMethod(t *testing.T) {
	req := httptest.NewRequest("DELETE", "/api/providers", nil)
	w := httptest.NewRecorder()

	handleProviders(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status code: expected 405, got %d", w.Code)
	}
}

func TestHandleCheckSafetyMethodValidation(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/check-safety", nil)
	w := httptest.NewRecorder()

	handleCheckSafety(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestHandleEvaluateConstitution(t *testing.T) {
	reqBody := map[string]interface{}{
		"message": "test message",
		"context": "test context",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/evaluate-constitution", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleEvaluateConstitution(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestHandleConversationsInvalidMethod(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/conversations", nil)
	w := httptest.NewRecorder()

	handleConversations(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status code: expected 405, got %d", w.Code)
	}
}

func TestHandleConversationContextMethods(t *testing.T) {
	req := httptest.NewRequest("DELETE", "/api/conversations/context", nil)
	w := httptest.NewRecorder()

	handleConversationContext(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status code: expected 405, got %d", w.Code)
	}
}

func TestHandleDeleteContactInvalidMethod(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/contacts/delete?id=1", nil)
	w := httptest.NewRecorder()

	handleDeleteContact(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405, got %d", w.Code)
	}
}

func TestHandleAnalyzeModeShiftInvalidMethod(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/analyze-mode-shift", nil)
	w := httptest.NewRecorder()

	handleAnalyzeModeShift(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestHandleGenerateQuestionsInvalidMethod(t *testing.T) {
	req := httptest.NewRequest("DELETE", "/api/generate-questions", nil)
	w := httptest.NewRecorder()

	handleGenerateQuestions(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestHandleGetPrinciples(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/constitution-principles", nil)
	w := httptest.NewRecorder()

	handleGetPrinciples(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code: expected 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if _, ok := response["principles"]; !ok {
		t.Error("Response should contain 'principles'")
	}
}

func TestResponseHeaders(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/status", nil)
	w := httptest.NewRecorder()

	handleStatus(w, req)

	if w.Header().Get("Content-Type") != "application/json" {
		t.Error("Response should have Content-Type: application/json")
	}

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("Response should have Access-Control-Allow-Origin: *")
	}
}

func TestRespondJSON(t *testing.T) {
	w := httptest.NewRecorder()
	testData := map[string]string{"key": "value"}

	respondJSON(w, http.StatusOK, testData)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]string
	json.NewDecoder(w.Body).Decode(&response)

	if response["key"] != "value" {
		t.Error("Response data not correctly encoded")
	}
}

func TestRespondError(t *testing.T) {
	w := httptest.NewRecorder()

	respondError(w, http.StatusBadRequest, "Test error message")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["error"] != "Test error message" {
		t.Error("Error message not correctly set")
	}
}
