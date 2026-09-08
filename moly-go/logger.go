package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
)

// Logger is the global structured logger instance
var Logger = logrus.New()

// InitializeLogger sets up structured logging with JSON output and file rotation
func InitializeLogger(logLevel string) error {
	// Set log level based on configuration
	level := logrus.InfoLevel
	switch strings.ToLower(logLevel) {
	case "debug":
		level = logrus.DebugLevel
	case "info":
		level = logrus.InfoLevel
	case "warn", "warning":
		level = logrus.WarnLevel
	case "error":
		level = logrus.ErrorLevel
	default:
		level = logrus.InfoLevel
	}

	Logger.SetLevel(level)

	// Use JSON formatter for structured logging
	Logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		PrettyPrint:     false, // Compact JSON for production
	})

	// Get log file path (platform-specific)
	logFilePath := getLogFilePath()

	// Create log directory if it doesn't exist
	logDir := filepath.Dir(logFilePath)
	if err := os.MkdirAll(logDir, 0700); err != nil {
		return err
	}

	// Set up log file with rotation (max 100MB per file, keep 3 files, max 300MB total)
	logFile := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    100, // megabytes
		MaxBackups: 3,   // number of backups to keep
		MaxAge:     7,   // days
		Compress:   true,
	}

	// Write logs to both stdout and file
	multiWriter := &multiWriteCloser{
		writers: []interface{}{
			os.Stdout,
			logFile,
		},
	}

	Logger.SetOutput(multiWriter)

	Logger.WithFields(logrus.Fields{
		"component": "logger",
		"version":   "1.0.0",
		"log_level": logLevel,
		"log_file":  logFilePath,
	}).Info("[Moly] Logger initialized")

	return nil
}

// getLogFilePath returns the platform-specific log file path
func getLogFilePath() string {
	configDir := getConfigDir("moly.log")
	return configDir
}

// multiWriteCloser allows writing to multiple outputs
type multiWriteCloser struct {
	writers []interface{}
}

func (m *multiWriteCloser) Write(p []byte) (n int, err error) {
	for _, w := range m.writers {
		if file, ok := w.(*os.File); ok {
			file.Write(p)
		} else if rotatingFile, ok := w.(*lumberjack.Logger); ok {
			rotatingFile.Write(p)
		}
	}
	return len(p), nil
}

func (m *multiWriteCloser) Close() error {
	for _, w := range m.writers {
		if closer, ok := w.(interface{ Close() error }); ok {
			closer.Close()
		}
	}
	return nil
}

// LogConfig logs the current configuration for debugging
func LogConfig(config ServerConfig) {
	Logger.WithFields(logrus.Fields{
		"port":            config.Port,
		"host":            config.Host,
		"log_level":       config.LogLevel,
		"cors_proxy_port": config.CORSProxyPort,
		"database_path":   config.DatabasePath,
		"ollama_endpoint": config.OllamaEndpoint,
	}).Info("[Moly] Server configuration loaded")
}

// LogStartup logs the startup process
func LogStartup(port, host string) {
	Logger.WithFields(logrus.Fields{
		"component": "server",
		"port":      port,
		"host":      host,
	}).Info("[Moly] Starting Moly backend")
}

// LogError logs errors with context
func LogError(component string, err error, context map[string]interface{}) {
	fields := logrus.Fields{"component": component}
	for k, v := range context {
		fields[k] = v
	}
	Logger.WithFields(fields).WithError(err).Error("[Moly] Error occurred")
}

// LogAPICall logs incoming API calls
func LogAPICall(method, path string, statusCode int, duration float64) {
	Logger.WithFields(logrus.Fields{
		"component":   "api",
		"method":      method,
		"path":        path,
		"status_code": statusCode,
		"duration_ms": duration,
	}).Debug("[Moly] API call")
}

// LogShutdown logs the shutdown process
func LogShutdown(reason string) {
	Logger.WithFields(logrus.Fields{
		"component": "server",
		"reason":    reason,
	}).Info("[Moly] Shutting down")
}
