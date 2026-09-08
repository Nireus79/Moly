//go:build integration
// +build integration

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestBackendStartup verifies the server can start with default configuration
func TestBackendStartup(t *testing.T) {
	// Save original env vars
	originalPort := os.Getenv("MOLY_PORT")
	originalHost := os.Getenv("MOLY_HOST")
	defer func() {
		os.Setenv("MOLY_PORT", originalPort)
		os.Setenv("MOLY_HOST", originalHost)
	}()

	// Set test configuration
	os.Setenv("MOLY_PORT", ":9999")
	os.Setenv("MOLY_HOST", "127.0.0.1")

	config := LoadConfig()

	if config.Port != ":9999" {
		t.Errorf("Expected port :9999, got %s", config.Port)
	}
	if config.Host != "127.0.0.1" {
		t.Errorf("Expected host 127.0.0.1, got %s", config.Host)
	}
}

// TestDatabasePath verifies database path is configured correctly
func TestDatabasePath(t *testing.T) {
	config := LoadConfig()

	if config.DatabasePath == "" {
		t.Error("Database path should not be empty")
	}

	if !bytes.Contains([]byte(config.DatabasePath), []byte("moly")) {
		t.Errorf("Database path should contain 'moly': %s", config.DatabasePath)
	}
}

// TestConfigurationLoading verifies config loads from file and environment
func TestConfigurationLoading(t *testing.T) {
	testCases := []struct {
		name          string
		envVars       map[string]string
		expectedPort  string
		expectedHost  string
		expectedLevel string
	}{
		{
			name:          "Defaults",
			envVars:       map[string]string{},
			expectedPort:  ":11436",
			expectedHost:  "127.0.0.1",
			expectedLevel: "info",
		},
		{
			name: "Environment override",
			envVars: map[string]string{
				"MOLY_PORT": ":8888",
				"MOLY_HOST": "localhost",
			},
			expectedPort:  ":8888",
			expectedHost:  "localhost",
			expectedLevel: "info",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set env vars
			for key, val := range tc.envVars {
				os.Setenv(key, val)
			}
			defer func() {
				for key := range tc.envVars {
					os.Unsetenv(key)
				}
			}()

			config := LoadConfig()

			if config.Port != tc.expectedPort {
				t.Errorf("Port: expected %s, got %s", tc.expectedPort, config.Port)
			}
			if config.Host != tc.expectedHost {
				t.Errorf("Host: expected %s, got %s", tc.expectedHost, config.Host)
			}
			if config.LogLevel != tc.expectedLevel {
				t.Errorf("Log level: expected %s, got %s", tc.expectedLevel, config.LogLevel)
			}
		})
	}
}

// TestErrorReportingEndpoint verifies frontend error endpoint works
func TestErrorReportingEndpoint(t *testing.T) {
	errors := []map[string]interface{}{
		{
			"id":        "error_123",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"level":     "error",
			"component": "api",
			"message":   "Test error",
			"stack":     "stack trace here",
			"context": map[string]interface{}{
				"endpoint": "/api/status",
			},
		},
	}

	payload := map[string]interface{}{
		"errors":            errors,
		"session":           "session_test_123",
		"extension_version": "1.0.0",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/frontend-errors", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handleFrontendErrors(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)

	if response["success"] != true {
		t.Error("Response should have success=true")
	}
}

// TestStatusEndpoint verifies the status endpoint works
func TestStatusEndpoint(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/status", nil)
	w := httptest.NewRecorder()

	handleStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]string
	json.NewDecoder(w.Body).Decode(&response)

	if response["status"] != "running" {
		t.Errorf("Expected status 'running', got '%s'", response["status"])
	}
}

// TestSettingsEndpoint verifies settings can be retrieved and updated
func TestSettingsEndpoint(t *testing.T) {
	// Test GET
	req := httptest.NewRequest("GET", "/api/settings", nil)
	w := httptest.NewRecorder()

	handleSettings(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET: Expected status 200, got %d", w.Code)
	}

	var getResponse map[string]interface{}
	json.NewDecoder(w.Body).Decode(&getResponse)

	if getResponse["version"] == nil {
		t.Error("Settings should contain version")
	}

	// Test POST with valid JSON
	updates := map[string]interface{}{
		"theme": "dark",
		"mode":  "socratic",
	}
	postBody, _ := json.Marshal(updates)

	postReq := httptest.NewRequest("POST", "/api/settings", bytes.NewReader(postBody))
	postReq.Header.Set("Content-Type", "application/json")
	postW := httptest.NewRecorder()

	handleSettings(postW, postReq)

	if postW.Code != http.StatusOK {
		t.Errorf("POST: Expected status 200, got %d", postW.Code)
	}
}

// TestResponseFormats verifies all responses are properly formatted
func TestResponseFormats(t *testing.T) {
	testCases := []struct {
		name         string
		handler      func(http.ResponseWriter, *http.Request)
		method       string
		expectedType string
	}{
		{
			name:         "Status endpoint",
			handler:      handleStatus,
			method:       "GET",
			expectedType: "application/json",
		},
		{
			name:         "Providers endpoint",
			handler:      handleProviders,
			method:       "GET",
			expectedType: "application/json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/api/test", nil)
			w := httptest.NewRecorder()

			tc.handler(w, req)

			contentType := w.Header().Get("Content-Type")
			if contentType != tc.expectedType {
				t.Errorf("Expected Content-Type %s, got %s", tc.expectedType, contentType)
			}

			cors := w.Header().Get("Access-Control-Allow-Origin")
			if cors != "*" {
				t.Errorf("Expected CORS header *, got %s", cors)
			}
		})
	}
}

// TestErrorHandling verifies error responses are properly formatted
func TestErrorHandling(t *testing.T) {
	testCases := []struct {
		name           string
		handler        func(http.ResponseWriter, *http.Request)
		method         string
		expectedStatus int
	}{
		{
			name:           "Status accepts any method",
			handler:        handleStatus,
			method:         "DELETE",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Providers with DELETE",
			handler:        handleProviders,
			method:         "DELETE",
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/api/test", nil)
			w := httptest.NewRecorder()

			tc.handler(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, w.Code)
			}

			// Only check for error field if status is not OK
			if tc.expectedStatus != http.StatusOK {
				var response map[string]interface{}
				json.NewDecoder(w.Body).Decode(&response)

				if _, ok := response["error"]; !ok {
					t.Error("Error response should contain 'error' field")
				}
			}
		})
	}
}

// TestLoggerInitialization verifies logger can be initialized
func TestLoggerInitialization(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	err := InitializeLogger("info")
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	// Verify log file was created
	logPath := filepath.Join(tmpDir, ".config", "moly", "moly.log")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Errorf("Log file should be created at %s", logPath)
	}
}
