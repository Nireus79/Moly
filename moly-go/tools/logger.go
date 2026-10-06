package tools

import (
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// LoggerConfig holds logging configuration
type LoggerConfig struct {
	Level      string // DEBUG, INFO, WARN, ERROR
	Format     string // json or text
	FilePath   string // where to write logs
	MaxSize    int    // megabytes
	MaxBackups int    // number of backups
	MaxAge     int    // days
}

// Logger is the global structured logger instance
var Logger *logrus.Logger

// InitLogger initializes the structured logging system
func InitLogger(config LoggerConfig) error {
	Logger = logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(config.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	Logger.SetLevel(level)

	// Set format
	if config.Format == "json" {
		Logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.999Z07:00",
		})
	} else {
		Logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02T15:04:05",
		})
	}

	// Setup file output if path provided
	if config.FilePath != "" {
		// Create directory if needed
		dir := filepath.Dir(config.FilePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}

		// Setup log rotation
		fileLogger := &lumberjack.Logger{
			Filename:   config.FilePath,
			MaxSize:    config.MaxSize,    // megabytes
			MaxBackups: config.MaxBackups, // number of backups
			MaxAge:     config.MaxAge,     // days
			Compress:   true,              // compress old logs
		}

		Logger.SetOutput(fileLogger)
	} else {
		// Log to stdout
		Logger.SetOutput(os.Stdout)
	}

	return nil
}

// LogWithContext logs with correlation and request context
func LogWithContext(level logrus.Level, message string, fields map[string]interface{}) {
	entry := Logger.WithFields(fields)
	switch level {
	case logrus.ErrorLevel:
		entry.Error(message)
	case logrus.WarnLevel:
		entry.Warn(message)
	case logrus.InfoLevel:
		entry.Info(message)
	case logrus.DebugLevel:
		entry.Debug(message)
	default:
		entry.Info(message)
	}
}

// LogError logs an error with context
func LogError(component string, err error, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	fields["component"] = component
	fields["error"] = err.Error()
	Logger.WithFields(fields).Error("Error occurred")
}

// LogLayer logs layer execution
func LogLayer(layerID int, message string, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	fields["layer"] = layerID
	Logger.WithFields(fields).Info(message)
}

// LogPerformance logs performance metrics
func LogPerformance(component string, operation string, durationMs float64, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	fields["component"] = component
	fields["operation"] = operation
	fields["duration_ms"] = durationMs
	Logger.WithFields(fields).Debug("Performance metric")
}

// LogLLMResponse logs LLM response details
func LogLLMResponse(component string, durationMs float64, tokenCount int, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	fields["component"] = component
	fields["duration_ms"] = durationMs
	fields["tokens"] = tokenCount
	Logger.WithFields(fields).Info("LLM response received")
}

// DefaultLoggerConfig returns default configuration
func DefaultLoggerConfig() LoggerConfig {
	return LoggerConfig{
		Level:      "INFO",
		Format:     "json",
		FilePath:   "logs/moly.log",
		MaxSize:    100,  // 100MB
		MaxBackups: 10,   // keep 10 backups
		MaxAge:     30,   // 30 days retention
	}
}

// DebugLoggerConfig returns debug configuration
func DebugLoggerConfig() LoggerConfig {
	return LoggerConfig{
		Level:      "DEBUG",
		Format:     "text",
		FilePath:   "logs/moly.debug.log",
		MaxSize:    50,
		MaxBackups: 5,
		MaxAge:     7,
	}
}
