package tools

import (
	"time"

	"github.com/sirupsen/logrus"
)

// SetupProductionLogging initializes production logging with sensible defaults
func SetupProductionLogging() error {
	config := LoggerConfig{
		Level:      "INFO",
		Format:     "json",
		FilePath:   "logs/moly.log",
		MaxSize:    100,
		MaxBackups: 10,
		MaxAge:     30,
	}
	return InitLogger(config)
}

// SetupDebugLogging initializes debug logging
func SetupDebugLogging() error {
	config := LoggerConfig{
		Level:      "DEBUG",
		Format:     "text",
		FilePath:   "logs/moly.debug.log",
		MaxSize:    50,
		MaxBackups: 5,
		MaxAge:     7,
	}
	return InitLogger(config)
}

// LogRequestStart logs the start of a request
func LogRequestStart(requestID, userID, conversationID string) {
	Logger.WithFields(logrus.Fields{
		"request_id":      requestID,
		"user_id":         userID,
		"conversation_id": conversationID,
		"timestamp":       time.Now(),
	}).Info("Request started")
}

// LogRequestEnd logs the end of a request
func LogRequestEnd(requestID string, durationMs float64, success bool) {
	level := logrus.InfoLevel
	if !success {
		level = logrus.ErrorLevel
	}

	Logger.WithFields(logrus.Fields{
		"request_id":  requestID,
		"duration_ms": durationMs,
		"success":     success,
	}).Log(level, "Request completed")
}

// LogLayerStart logs layer execution start
func LogLayerStart(layerID int, requestID string) {
	Logger.WithFields(logrus.Fields{
		"layer":      layerID,
		"request_id": requestID,
	}).Debug("Layer execution started")
}

// LogLayerResult logs layer result
func LogLayerResult(layerID int, requestID string, resultCount int, durationMs float64) {
	Logger.WithFields(logrus.Fields{
		"layer":        layerID,
		"request_id":   requestID,
		"result_count": resultCount,
		"duration_ms":  durationMs,
	}).Debug("Layer execution completed")
}

// LogLLMCall logs LLM API call
func LogLLMCall(requestID string, prompt string, model string, durationMs float64, tokens int) {
	Logger.WithFields(logrus.Fields{
		"request_id":  requestID,
		"model":       model,
		"prompt_len":  len(prompt),
		"duration_ms": durationMs,
		"tokens":      tokens,
	}).Debug("LLM API call completed")
}

// LogDatabaseOp logs database operation
func LogDatabaseOp(operation string, table string, durationMs float64, rowsAffected int, err error) {
	fields := logrus.Fields{
		"operation":     operation,
		"table":         table,
		"duration_ms":   durationMs,
		"rows_affected": rowsAffected,
	}

	level := logrus.DebugLevel
	if err != nil {
		fields["error"] = err.Error()
		level = logrus.ErrorLevel
	}

	Logger.WithFields(fields).Log(level, "Database operation")
}

// LogValidationError logs validation errors
func LogValidationError(field string, reason string, value interface{}) {
	Logger.WithFields(logrus.Fields{
		"field":  field,
		"reason": reason,
		"value":  value,
	}).Warn("Validation error")
}

// LogSecurityEvent logs security-relevant events
func LogSecurityEvent(eventType string, details string, userID string) {
	Logger.WithFields(logrus.Fields{
		"event_type": eventType,
		"user_id":    userID,
		"details":    details,
		"timestamp":  time.Now(),
	}).Warn("Security event detected")
}
