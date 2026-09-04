package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleStatus(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/status", nil)
	w := httptest.NewRecorder()

	handleStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code: expected 200, got %d", w.Code)
	}

	var response map[string]string
	json.NewDecoder(w.Body).Decode(&response)

	if response["status"] != "running" {
		t.Errorf("Status: expected 'running', got '%s'", response["status"])
	}
}

func TestHandleFirstRunCheck(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/first-run-check", nil)
	w := httptest.NewRecorder()

	handleFirstRunCheck(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code: expected 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if _, ok := response["first_run_complete"]; !ok {
		t.Error("Response should contain 'first_run_complete'")
	}
	if _, ok := response["ollama_installed"]; !ok {
		t.Error("Response should contain 'ollama_installed'")
	}
	if _, ok := response["ollama_running"]; !ok {
		t.Error("Response should contain 'ollama_running'")
	}
}

func TestHandleSettingsGet(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/settings", nil)
	w := httptest.NewRecorder()

	handleSettings(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code: expected 200, got %d", w.Code)
	}

	var config Config
	json.NewDecoder(w.Body).Decode(&config)

	if config.Version == "" {
		t.Error("Config should have version")
	}
	if config.Provider == "" {
		t.Error("Config should have provider")
	}
}

func TestHandleSettingsPost(t *testing.T) {
	updates := map[string]interface{}{
		"model": "llama2",
		"mode":  "socratic",
	}
	body, _ := json.Marshal(updates)

	req := httptest.NewRequest("POST", "/api/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleSettings(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code: expected 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["success"] != true {
		t.Error("Response should have success=true")
	}
}

func TestHandleSettingsPostInvalidJSON(t *testing.T) {
	body := []byte("invalid json")

	req := httptest.NewRequest("POST", "/api/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleSettings(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status code: expected 400, got %d", w.Code)
	}
}

func TestHandleSettingsInvalidMethod(t *testing.T) {
	req := httptest.NewRequest("DELETE", "/api/settings", nil)
	w := httptest.NewRecorder()

	handleSettings(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status code: expected 405, got %d", w.Code)
	}
}

func TestHandleListModels(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/models/list", nil)
	w := httptest.NewRecorder()

	handleListModels(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code: expected 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if _, ok := response["models"]; !ok {
		t.Error("Response should contain 'models'")
	}
}

func TestHandleRoot(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handleRoot(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code: expected 200, got %d", w.Code)
	}

	var response map[string]string
	json.NewDecoder(w.Body).Decode(&response)

	if response["message"] != "Moly Desktop App" {
		t.Errorf("Message: expected 'Moly Desktop App', got '%s'", response["message"])
	}
}

func TestHandleRemoveModelInvalidMethod(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/models/remove?name=test", nil)
	w := httptest.NewRecorder()

	handleRemoveModel(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status code: expected 405, got %d", w.Code)
	}
}

func TestHandleRemoveModelMissingName(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/models/remove", nil)
	w := httptest.NewRecorder()

	handleRemoveModel(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status code: expected 400, got %d", w.Code)
	}
}

func TestHandlePullModelInvalidMethod(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/models/pull?name=mistral", nil)
	w := httptest.NewRecorder()

	handlePullModel(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status code: expected 405, got %d", w.Code)
	}
}

func TestHandlePullModelMissingName(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/models/pull", nil)
	w := httptest.NewRecorder()

	handlePullModel(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status code: expected 400, got %d", w.Code)
	}
}

func TestHandleStartOllamaInvalidMethod(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/ollama/start", nil)
	w := httptest.NewRecorder()

	handleStartOllama(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status code: expected 405, got %d", w.Code)
	}
}

func TestHandleStopOllamaInvalidMethod(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/ollama/stop", nil)
	w := httptest.NewRecorder()

	handleStopOllama(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status code: expected 405, got %d", w.Code)
	}
}

func TestHandleChatInvalidMethod(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/chat", nil)
	w := httptest.NewRecorder()

	handleChat(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status code: expected 405, got %d", w.Code)
	}
}

func TestHandleChatMissingMessage(t *testing.T) {
	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest("POST", "/api/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleChat(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status code: expected 400, got %d", w.Code)
	}
}

func TestHandleChatInvalidJSON(t *testing.T) {
	body := []byte("invalid json")
	req := httptest.NewRequest("POST", "/api/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleChat(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status code: expected 400, got %d", w.Code)
	}
}

func TestHandleSidebarHTML(t *testing.T) {
	req := httptest.NewRequest("GET", "/sidebar.html", nil)
	w := httptest.NewRecorder()

	handleSidebarHTML(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code: expected 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("Content-Type: expected 'text/html; charset=utf-8', got '%s'", contentType)
	}

	if !contains(w.Body.String(), "<html") {
		t.Error("Response should contain HTML")
	}
}

func TestResponseJSONHeaders(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/status", nil)
	w := httptest.NewRecorder()

	handleStatus(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type: expected 'application/json', got '%s'", contentType)
	}

	corsHeader := w.Header().Get("Access-Control-Allow-Origin")
	if corsHeader != "*" {
		t.Errorf("Access-Control-Allow-Origin: expected '*', got '%s'", corsHeader)
	}
}
