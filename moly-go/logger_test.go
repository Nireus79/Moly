package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestInitializeLoggerDebugLevel(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	oldEnv := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldEnv)

	// Reinitialize logger
	Logger = logrus.New()

	err := InitializeLogger("debug")
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	if Logger.Level != logrus.DebugLevel {
		t.Errorf("Expected debug level, got %v", Logger.Level)
	}
}

func TestInitializeLoggerInfoLevel(t *testing.T) {
	Logger = logrus.New()

	err := InitializeLogger("info")
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	if Logger.Level != logrus.InfoLevel {
		t.Errorf("Expected info level, got %v", Logger.Level)
	}
}

func TestInitializeLoggerWarnLevel(t *testing.T) {
	Logger = logrus.New()

	err := InitializeLogger("warning")
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	if Logger.Level != logrus.WarnLevel {
		t.Errorf("Expected warn level, got %v", Logger.Level)
	}
}

func TestInitializeLoggerErrorLevel(t *testing.T) {
	Logger = logrus.New()

	err := InitializeLogger("error")
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	if Logger.Level != logrus.ErrorLevel {
		t.Errorf("Expected error level, got %v", Logger.Level)
	}
}

func TestLoggerJSONFormat(t *testing.T) {
	// Create a buffer to capture output
	output := &bytes.Buffer{}

	// Create test logger
	testLogger := logrus.New()
	testLogger.SetOutput(output)
	testLogger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		PrettyPrint:     false,
	})
	testLogger.SetLevel(logrus.InfoLevel)

	// Log a test message
	testLogger.WithFields(logrus.Fields{
		"component": "test",
		"count":     42,
	}).Info("Test message")

	// Parse the output as JSON
	var logEntry map[string]interface{}
	err := json.Unmarshal(output.Bytes(), &logEntry)
	if err != nil {
		t.Fatalf("Failed to parse JSON log: %v", err)
	}

	// Verify required fields
	if msg, ok := logEntry["msg"]; !ok || msg != "Test message" {
		t.Errorf("Missing or incorrect msg field")
	}
	if level, ok := logEntry["level"]; !ok || level != "info" {
		t.Errorf("Missing or incorrect level field")
	}
	if component, ok := logEntry["component"]; !ok || component != "test" {
		t.Errorf("Missing or incorrect component field")
	}
	if _, ok := logEntry["time"]; !ok {
		t.Errorf("Missing time field")
	}
}

func TestLogConfigLogging(t *testing.T) {
	// Create a buffer to capture output
	output := &bytes.Buffer{}

	// Create test logger
	testLogger := logrus.New()
	testLogger.SetOutput(output)
	testLogger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		PrettyPrint:     false,
	})
	testLogger.SetLevel(logrus.InfoLevel)

	// Temporarily replace global logger
	oldLogger := Logger
	Logger = testLogger

	config := ServerConfig{
		Port:           ":11436",
		Host:           "127.0.0.1",
		LogLevel:       "info",
		CORSProxyPort:  ":11435",
		DatabasePath:   "/tmp/moly.db",
		OllamaEndpoint: "http://127.0.0.1:11434",
	}

	LogConfig(config)

	// Restore global logger
	Logger = oldLogger

	// Parse the output as JSON
	var logEntry map[string]interface{}
	err := json.Unmarshal(output.Bytes(), &logEntry)
	if err != nil {
		t.Fatalf("Failed to parse JSON log: %v", err)
	}

	// Verify configuration fields are logged
	if port, ok := logEntry["port"]; !ok || port != ":11436" {
		t.Errorf("Missing or incorrect port field")
	}
	if host, ok := logEntry["host"]; !ok || host != "127.0.0.1" {
		t.Errorf("Missing or incorrect host field")
	}
}

func TestLogFileCreation(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	oldEnv := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldEnv)

	// Reinitialize logger
	Logger = logrus.New()

	err := InitializeLogger("info")
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	// Check if log file was created
	logPath := filepath.Join(tempDir, ".config", "moly", "moly.log")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Errorf("Log file was not created at %s", logPath)
	}
}

func TestLogLevelFiltering(t *testing.T) {
	// Create a buffer to capture output
	output := &bytes.Buffer{}

	// Create test logger with WARN level
	testLogger := logrus.New()
	testLogger.SetOutput(output)
	testLogger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		PrettyPrint:     false,
	})
	testLogger.SetLevel(logrus.WarnLevel)

	// Log messages at different levels
	testLogger.Debug("Debug message")
	testLogger.Info("Info message")
	testLogger.Warn("Warn message")

	// Should only contain warn and error messages
	outputStr := output.String()

	if bytes.Contains([]byte(outputStr), []byte("Debug message")) {
		t.Error("Debug message should not be in output when level is WARN")
	}
	if bytes.Contains([]byte(outputStr), []byte("Info message")) {
		t.Error("Info message should not be in output when level is WARN")
	}
	if !bytes.Contains([]byte(outputStr), []byte("Warn message")) {
		t.Error("Warn message should be in output when level is WARN")
	}
}

func TestGetLogFilePath(t *testing.T) {
	// Save old HOME
	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)

	// Set test HOME
	testHome := "/tmp/test-home"
	os.Setenv("HOME", testHome)

	path := getLogFilePath()

	// Verify path is correct
	if !bytes.Contains([]byte(path), []byte("moly.log")) {
		t.Errorf("Log file path should contain 'moly.log': %s", path)
	}
}
