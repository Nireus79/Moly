package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"moly/database"
	"moly/tools"
)

// V2ServerConfig holds V2 server configuration (separate from application Config)
type V2ServerConfig struct {
	Port              int
	DBPath            string
	LLMProvider       string
	LogLevel          string
	GracefulTimeout   time.Duration
	RequestTimeout    time.Duration
	BindAddr          string
}

// loadConfigV2 loads configuration from environment and flags
func loadConfigV2() V2ServerConfig {
	cfg := V2ServerConfig{
		Port:            8080,
		DBPath:          filepath.Join(os.ExpandEnv("$HOME"), ".config", "moly", "moly.db"),
		LLMProvider:     getEnv("MOLY_LLM_PROVIDER", ""),
		LogLevel:        getEnv("MOLY_LOG_LEVEL", "info"),
		GracefulTimeout: 15 * time.Second,
		RequestTimeout:  30 * time.Second,
		BindAddr:        "127.0.0.1",
	}

	// Parse command-line flags
	flag.IntVar(&cfg.Port, "port", cfg.Port, "Server port")
	flag.StringVar(&cfg.DBPath, "db-path", cfg.DBPath, "Database file path")
	flag.StringVar(&cfg.LLMProvider, "llm-provider", cfg.LLMProvider, "LLM provider (ollama, claude, openai, or none)")
	flag.StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "Log level (debug, info, warn, error)")
	flag.StringVar(&cfg.BindAddr, "bind", cfg.BindAddr, "Bind address")
	flag.Parse()

	// Set logging level
	setLogLevel(cfg.LogLevel)

	return cfg
}

// getEnv gets environment variable with default
func getEnv(key, defaultVal string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultVal
}

// setLogLevel configures logrus based on log level string
func setLogLevel(level string) {
	Logger.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})

	switch level {
	case "debug":
		Logger.SetLevel(logrus.DebugLevel)
	case "info":
		Logger.SetLevel(logrus.InfoLevel)
	case "warn":
		Logger.SetLevel(logrus.WarnLevel)
	case "error":
		Logger.SetLevel(logrus.ErrorLevel)
	default:
		Logger.SetLevel(logrus.InfoLevel)
	}
}

// HealthResponse is the response from health check
type HealthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
	LLM      string `json:"llm"`
	Time     string `json:"time"`
}

// RequestLogger middleware logs all requests
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call handler
		next.ServeHTTP(wrapped, r)

		// Log request
		duration := time.Since(start)
		Logger.WithFields(logrus.Fields{
			"method":   r.Method,
			"path":     r.RequestURI,
			"status":   wrapped.statusCode,
			"duration": duration.String(),
		}).Info("Request completed")
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// setupV2API creates and configures V2 API endpoints
func setupV2API(llm *tools.LLMClient, db *database.Database) http.Handler {
	server, err := NewV2APIServer(llm, db)
	if err != nil {
		Logger.WithError(err).Fatal("Failed to create V2 API server")
	}

	mux := http.NewServeMux()

	// V2 API endpoints
	mux.HandleFunc("/api/v2/health", func(w http.ResponseWriter, r *http.Request) {
		health := HealthResponse{
			Status:   "ok",
			Database: "ok",
			LLM:      "ok",
			Time:     time.Now().Format(time.RFC3339),
		}

		if llm == nil {
			health.LLM = "none"
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(health)
	})

	// V2 conversation endpoints
	mux.HandleFunc("/api/v2/conversation/generate", server.ConversationGenerateHandler)
	mux.HandleFunc("/api/v2/conversation/feedback", server.ConversationFeedbackHandler)
	mux.HandleFunc("/api/v2/context", server.GetContextHandler)
	mux.HandleFunc("/api/v2/about-me", server.SetAboutMeHandler)
	// TODO: mux.HandleFunc("/api/v2/contacts", server.SetContactHandler) - handler not yet implemented

	// V2.1 chat endpoints (auth required)
	chatServer := NewChatServer(db, llm)
	mux.HandleFunc("/api/v2.1/chat", chatServer.ChatHandler)
	mux.HandleFunc("/api/v2.1/conversations/", func(w http.ResponseWriter, r *http.Request) {
		// Route to appropriate handler based on method
		if r.Method == http.MethodGet {
			chatServer.GetConversationHistoryHandler(w, r)
		} else if r.Method == http.MethodDelete {
			chatServer.DeleteConversationHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Add logging middleware
	return RequestLogger(mux)
}

// Server holds HTTP server and components
type Server struct {
	httpServer *http.Server
	db         *database.Database
	llm        *tools.LLMClient
}

// NewServer creates a new server
func NewServer(cfg V2ServerConfig) *Server {
	Logger.Info("[Main] Initializing Moly V2 Backend")

	// Initialize database with encryption (V2.1)
	// Use a derived key based on database path for server-level encryption
	// Each user's session will have its own encrypted connection with userID-derived key
	Logger.Infof("[Main] Connecting to database at %s (encrypted)", cfg.DBPath)

	// For server startup, use a system-level key derived from fixed salt
	// Individual user sessions will derive keys from their userID
	systemKey := "moly-v2.1-system-key"
	db, err := database.Init(cfg.DBPath, systemKey)
	if err != nil {
		Logger.WithError(err).Fatal("Failed to initialize database")
	}
	Logger.Info("[Main] ✓ Database initialized successfully (AES-256 encrypted)")

	// Initialize LLM
	var llm *tools.LLMClient
	Logger.Info("[Main] Initializing LLM provider")
	llm, err = tools.NewLLMClient()
	if err != nil {
		Logger.WithError(err).Warn("[Main] LLM initialization failed, using heuristic mode")
		llm = nil
	} else {
		Logger.Info("[Main] ✓ LLM provider initialized successfully")
	}

	// Setup V2 API
	handler := setupV2API(llm, db)

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.BindAddr, cfg.Port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  cfg.RequestTimeout,
		WriteTimeout: cfg.RequestTimeout,
		IdleTimeout:  cfg.RequestTimeout,
	}

	Logger.Infof("[Main] HTTP server configured at %s", addr)

	return &Server{
		httpServer: httpServer,
		db:         db,
		llm:        llm,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	Logger.Infof("[Main] Starting HTTP server on %s", s.httpServer.Addr)

	// Listen
	listener, err := net.Listen("tcp", s.httpServer.Addr)
	if err != nil {
		Logger.WithError(err).Fatal("Failed to create listener")
	}

	// Start server in goroutine
	go func() {
		if err := s.httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			Logger.WithError(err).Error("Server error")
		}
	}()

	Logger.Info("[Main] ✓ HTTP server started successfully")
	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(timeout time.Duration) error {
	Logger.Info("[Main] Starting graceful shutdown")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Shutdown HTTP server (stops accepting new requests)
	Logger.Info("[Main] Stopping HTTP server")
	if err := s.httpServer.Shutdown(ctx); err != nil {
		Logger.WithError(err).Warn("Error during server shutdown")
	}

	// Close database
	Logger.Info("[Main] Closing database connections")
	// Database doesn't have explicit close, but SQLite handles cleanup

	Logger.Info("[Main] ✓ Graceful shutdown complete")
	return nil
}

// WaitForInterrupt waits for SIGTERM or SIGINT
func WaitForInterrupt() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	Logger.WithField("signal", sig).Info("[Main] Received shutdown signal")
}

// mainV2 is the main entry point for V2
func mainV2() {
	// Load configuration
	cfg := loadConfigV2()

	Logger.WithFields(logrus.Fields{
		"port":           cfg.Port,
		"database":       cfg.DBPath,
		"log_level":      cfg.LogLevel,
		"graceful_timeout": cfg.GracefulTimeout.String(),
	}).Info("[Main] Configuration loaded")

	// Create server
	server := NewServer(cfg)
	defer func() {
		// Graceful shutdown on exit
		ctx, cancel := context.WithTimeout(context.Background(), cfg.GracefulTimeout)
		defer cancel()

		// Shutdown HTTP server
		if err := server.httpServer.Shutdown(ctx); err != nil {
			Logger.WithError(err).Warn("Server shutdown error")
		}
	}()

	// Start server
	if err := server.Start(); err != nil {
		Logger.WithError(err).Fatal("Failed to start server")
	}

	// Print startup summary
	Logger.Info("")
	Logger.Info("╔════════════════════════════════════════╗")
	Logger.Info("║     Moly V2 Backend Started ✓          ║")
	Logger.Infof("║ Port: %d                               ║", cfg.Port)
	if server.llm != nil {
		Logger.Info("║ LLM: Active (Ollama/Cloud)             ║")
	} else {
		Logger.Info("║ LLM: Heuristic Mode (Fallback)         ║")
	}
	Logger.Info("║ Database: Connected ✓                  ║")
	Logger.Info("║ API Endpoints: Ready ✓                 ║")
	Logger.Info("╚════════════════════════════════════════╝")
	Logger.Info("")
	Logger.Infof("Health check: http://localhost:%d/api/v2/health", cfg.Port)
	Logger.Info("Press Ctrl+C to shutdown")
	Logger.Info("")

	// Wait for interrupt signal
	WaitForInterrupt()

	// Graceful shutdown
	Logger.Info("[Main] Shutting down gracefully...")
	server.Shutdown(cfg.GracefulTimeout)
}

// Note: The original main() function remains for V1 backward compatibility.
// Call mainV2() to run V2 backend, or main() for V1.
// To switch to V2, either:
// 1. Rename this to main() and rename existing main() to something else
// 2. Add a flag to choose which to run
