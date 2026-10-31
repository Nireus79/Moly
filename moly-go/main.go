package main

import (
	"context"
	"log"
	"moly/keystore"
	"moly/privacy"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"moly/auth"
	"moly/database"
	"moly/secrets"
)

// SECURITY: Message length limits No message length limit)
const (
	MaxMessageLength       = 50000 // Maximum characters per message (DoS prevention)
	MaxClarificationLength = 10000 // Maximum characters per clarification response
)

func main() {
	// Native host, host registration and version run without the server (host.go, registerhost.go).
	if handled, code := runSubcommand(os.Args[1:]); handled {
		os.Exit(code)
	}
	if err := ensureConfigDir(); err != nil {
		log.Fatalf("[Moly] cannot prepare the configuration files: %v", err)
	}

	// Load configuration first
	config := LoadConfig()

	// What the user said is masked in every log line (package privacy); MOLY_LOG_CONTENT=1 shows it for debugging.
	// Production mode never shows it: a user's log holds no names, whatever the environment says.
	if productionMode() && os.Getenv("MOLY_LOG_CONTENT") != "" {
		os.Unsetenv("MOLY_LOG_CONTENT")
		privacy.Default.Mask()
		log.Printf("[Moly] MOLY_LOG_CONTENT is ignored in production mode")
	}
	log.SetOutput(privacy.Default)

	// Initialize structured logging Complete logging system)
	if err := InitializeLogger(config.LogLevel); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// Log startup configuration
	LogStartup(config.Host, config.Port)

	// Production mode (MOLY_PRODUCTION=1): take the loopback port first, so a second Moly stops here and never opens
	// the database; answer "starting" until the real handler is ready (see security.go and lifecycle.go).
	var (
		prodListener net.Listener
		prodServer   *http.Server
		startup      = &startupSwitch{}
		policy       *securityPolicy
	)
	if productionMode() {
		if err := secrets.HardenProcess(); err != nil {
			log.Printf("[Moly] could not make the process non-dumpable: %v", err)
		}
		var lerr error
		if prodListener, lerr = listenLoopback(config.Port); lerr != nil {
			log.Fatalf("[Moly] %v", lerr)
		}
		if policy, lerr = newSecurityPolicy(config.Port, allowedExtensionIDs()); lerr != nil {
			log.Fatalf("[Moly] cannot create the session token: %v", lerr)
		}
		if lerr = policy.writeToken(molyDataDir()); lerr != nil {
			log.Fatalf("[Moly] cannot write the session token: %v", lerr)
		}
		if len(allowedExtensionIDs()) == 0 {
			log.Printf("[Moly] WARNING: production mode with no extension IDs: no browser origin is allowed (set MOLY_EXTENSION_IDS)")
		}
		prodServer = &http.Server{Handler: startup}
		go prodServer.Serve(prodListener)
		log.Printf("[Moly] production mode: listening on 127.0.0.1%s, token in %s", config.Port, filepath.Join(molyDataDir(), tokenFile))
	}
	Logger.WithField("config_file", getConfigFilePath()).Info("Configuration loaded")

	// The database key: the system keychain, or what the person chose when there is none (package keystore). Production mode
	// keeps answering while it waits for a passphrase; development mode stops with the reason.
	dbPath := filepath.Join(molyDataDir(), "moly.db")
	keys := keystore.Open(molyDataDir(), dbPath, keystore.SystemKeyring{})
	hosts := newHostRegistry()
	quit := make(chan struct{}, 1)
	if st := keys.Status(); st.State != keystore.StateReady {
		if !productionMode() {
			log.Fatalf("[Moly] the database key is not available (%s): %s", st.State, st.Reason)
		}
		log.Printf("[Moly] the database key is not available yet (%s): waiting for the person", st.State)
		gate := http.NewServeMux()
		hosts.register(gate)
		registerShutdown(gate, quit)
		registerKeyRoutes(gate, keys, "2.1")
		startup.Set(policy.middleware(gate))
		if !waitForKey(keys, hosts, quit, os.Getenv("MOLY_IDLE_EXIT") == "1") {
			shutdownServer(prodServer)
			return
		}
		startup.Set(nil) // back to "starting" until the full handler is ready
		log.Printf("[Moly] the database key is available")
	}
	dbKey, _ := keys.Key()
	var err error
	appDB, err = database.InitWithKey(dbPath, dbKey)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	// CRITICAL BUG FIX: Do NOT close database here via defer
	// RegisterCleanup (line ~5560) already registers database cleanup
	// Having TWO closures causes: "sql: database is closed" when cleanup runs while requests still in-flight
	// The cleanup handler ensures graceful shutdown on signal, this defer would close too early

	Logger.WithField("database_path", dbPath).Info("Database initialized")

	// Initialize auth tables
	conn := appDB.GetConnection()
	Logger.WithField("component", "database").Debug("Got connection - database stays open for server lifetime")
	// CRITICAL FIX: REMOVED defer conn.Close()
	// Database must stay open for the entire server lifetime!
	// Previously, when http.ListenAndServe() returned with error (port in use),
	// main() would exit and defer would close the database prematurely.
	if err := conn.Ping(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// The AI provider of each user: chosen, unlocked and kept per user by the manager (llm_settings.go). The server starts
	// without any: a user who has none yet is shown the first-run screen, one whose key is locked is asked for the passphrase.
	// The one exception is the operator's override (LLM_PROVIDER in the environment), which serves every user of this run and
	// must work, or the program stops with the reason.
	llmMgr, err := newLLMManager(appDB, func(r *http.Request) (string, error) { return extractAndValidateToken(r, appDB) }, os.Getenv("LLM_PROVIDER"))
	if err != nil {
		log.Fatalf("\n%v\n", err)
	}
	llmMgr.register(http.DefaultServeMux)
	if llmMgr.envFixed {
		log.Println("[Moly] the AI provider is fixed by LLM_PROVIDER for every user of this run")
	}

	// Initialize Dependency Injection Container
	// TODO: Implement in Phase 2.3 - DI container not yet complete
	// container := config.GetContainer()
	// if err := container.Initialize(appDB, llmClient); err != nil {
	// 	log.Fatalf("Failed to initialize DI container: %v", err)
	// }
	// log.Println("[Moly] DI Container initialized")
	// log.Printf("[VERIFICATION] After container init - appDB=%p, appDB.conn=%p", appDB, appDB.GetConnection())
	// log.Printf("[GOROUTINE TRACKING] After container init: %d goroutines active", runtime.NumGoroutine())

	// CRITICAL FIX: Commenting out database close handler
	// The cleanup handler was closing the database prematurely!
	// Keeping database open for the lifetime of the application
	// container.RegisterCleanup(func() error {
	// 	return appDB.Close()
	// })

	// Initialize API Server with agents and orchestration
	apiServer, err = NewAPIServer(llmMgr, appDB)
	if err != nil {
		log.Fatalf("Failed to initialize API server: %v", err)
	}
	log.Println("[Moly] API Server initialized")

	// Auth API routes with container access
	// NOTE: GetConnection() returns the database handle (*sql.DB), not a single connection
	// Do NOT close it - it's the main database handle that the entire application uses
	// TODO: Phase 2.3 - Use container when implemented
	// authDB := getContainerDB().GetConnection()
	authDB := appDB.GetConnection()
	log.Printf("[Moly] Auth DB pointer: %p, appDB pointer: %p", authDB, appDB.GetConnection())
	userAuthServer := auth.NewUserAuthServer(authDB)
	http.HandleFunc("/api/auth/register", userAuthServer.RegisterHandler)
	http.HandleFunc("/api/auth/login", userAuthServer.LoginHandler)
	http.HandleFunc("/api/auth/verify", userAuthServer.VerifyTokenHandler)
	http.HandleFunc("/api/auth/reset", userAuthServer.ResetHandler)
	http.HandleFunc("POST /api/auth/password", userAuthServer.ChangePasswordHandler)
	http.HandleFunc("GET /api/auth/profiles", userAuthServer.ProfilesHandler)
	http.HandleFunc("/api/auth/logout", userAuthServer.LogoutHandler)
	if productionMode() {
		// Installed Moly: one local profile without email or password (the API token already keeps other programs out).
		http.HandleFunc("POST /api/auth/local", userAuthServer.LocalProfileHandler)
	}
	log.Println("[Moly] Auth API routes registered (Email/Password authentication)")

	// Phase 5: Full orchestration endpoints
	http.HandleFunc("/api/message-processor", apiServer.MessageProcessorHandler)
	http.HandleFunc("GET /api/turn-status", apiServer.TurnStatusHandler)
	http.HandleFunc("POST /api/turn-stop", apiServer.TurnStopHandler)
	http.HandleFunc("/api/clarification/respond", apiServer.ClarificationResponseHandler)
	log.Println("[Moly] Phase 5 API routes registered (full orchestration + clarification + suggestion tracking)")

	// Context binding endpoints
	http.HandleFunc("/api/about-me", apiServer.AboutMeHandler)
	http.HandleFunc("/api/context", apiServer.ContextHandler)
	// Use Go 1.22+ pattern syntax to handle GET, POST at /api/conversations
	// and DELETE with ID parameter at /api/conversations/{conversationID}
	http.HandleFunc("GET /api/conversations", apiServer.ConversationsHandler)
	http.HandleFunc("POST /api/conversations", apiServer.ConversationsHandler)
	http.HandleFunc("DELETE /api/conversations/{conversationID}", apiServer.ConversationsHandler)
	http.HandleFunc("GET /api/conversations/{conversationID}/memory", apiServer.ConversationMemoryHandler)
	http.HandleFunc("GET /api/contacts", apiServer.ContactsHandler)
	http.HandleFunc("POST /api/contacts", apiServer.ContactsHandler)
	http.HandleFunc("GET /api/contacts/{contactID}", apiServer.ContactDetailHandler)
	http.HandleFunc("PUT /api/contacts/{contactID}", apiServer.ContactDetailHandler)
	http.HandleFunc("DELETE /api/contacts/{contactID}", apiServer.ContactDetailHandler)
	http.HandleFunc("/api/messages", apiServer.MessagesHandler)
	http.HandleFunc("/api/reflections", apiServer.ReflectionsHandler)
	http.HandleFunc("/api/conflicts", apiServer.ConflictsHandler)
	http.HandleFunc("/api/conflicts/resolve", apiServer.ConflictResolveHandler)
	http.HandleFunc("/api/metrics", apiServer.MetricsHandler)
	log.Println("[Moly] Context binding API routes registered (about-me + conversations + contacts + metrics + analysis)")

	// User account management
	http.HandleFunc("DELETE /api/user/delete", apiServer.DeleteProfileHandler)
	http.HandleFunc("GET /api/user/export", apiServer.ExportProfileHandler)
	http.HandleFunc("POST /api/user/import", apiServer.ImportProfileHandler)
	log.Println("[Moly] User account management routes registered (delete profile)")

	// Health check
	http.HandleFunc("/api/status", handleStatus(apiServer.database))

	// Error Reporting Endpoint
	http.HandleFunc("/api/frontend-errors", handleFrontendErrors)

	// Apply CORS middleware
	// Rate limiting sits inside CORS so that a 429 answer carries the CORS headers and the extension can read it.
	handler := corsMiddleware(rateLimitMiddleware(NewRateLimiter(), http.DefaultServeMux))

	log.Printf("[Moly] Server starting on http://localhost%s", config.Port)

	// DETAILED INVESTIGATION: Test connection validity step-by-step
	connForTest := appDB.GetConnection()

	pingErr := connForTest.Ping()

	// DETAILED DEBUG: If database is closed, collect diagnostics
	if pingErr != nil {
		if err2 := appDB.GetConnection().Ping(); err2 != nil {
		}

		// Dump goroutines
		buf := make([]byte, 16384)
		n := runtime.Stack(buf, true)
		log.Printf("[GOROUTINE DUMP]\n%s", buf[:n])
	} else {
	}

	// INVESTIGATION: Keep database alive to prevent GC from closing it
	runtime.KeepAlive(appDB)

	if productionMode() {
		runProduction(startup, prodServer, policy, rateLimitMiddleware(NewRateLimiter(), http.DefaultServeMux), appDB, hosts, quit)
		return
	}

	if err := http.ListenAndServe(config.Port, handler); err != nil {
		log.Fatalf("[VERIFICATION] http.ListenAndServe() returned with error: %v", err)
	}
}

// runProduction switches the early listener to the real handler and waits for a stop signal or, with MOLY_IDLE_EXIT=1,
// for the last browser host to go away. It then closes the server and the database cleanly.
func runProduction(startup *startupSwitch, srv *http.Server, policy *securityPolicy, app http.Handler, db *database.Database, hosts *hostRegistry, quit chan struct{}) {
	hosts.register(http.DefaultServeMux)
	registerShutdown(http.DefaultServeMux, quit)
	activity := newActivityTracker()
	activity.register(http.DefaultServeMux)
	startup.Set(policy.middleware(activity.wrap(app)))
	log.Printf("[Moly] production mode: ready")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	idleExit := os.Getenv("MOLY_IDLE_EXIT") == "1"
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()
	for running := true; running; {
		select {
		case sig := <-stop:
			log.Printf("[Moly] %v received, shutting down", sig)
			running = false
		case <-quit:
			log.Printf("[Moly] stop requested, shutting down")
			running = false
		case <-tick.C:
			if idleExit && hosts.Idle() {
				log.Printf("[Moly] no browser attached for %v, shutting down", idleGrace)
				running = false
			}
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("[Moly] shutdown: %v", err)
	}
	if err := db.Shutdown(); err != nil {
		log.Printf("[Moly] database shutdown: %v", err)
	}
}

func shutdownServer(srv *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("[Moly] shutdown: %v", err)
	}
}

// Helper functions for container access
// These provide safe access to container dependencies

// AUDIT FIXES: Helper functions

// ============================================================================
// MEDIUM PRIORITY AUDIT FIXES (Issues #1, #3, #4, #9, #12, #19, #20, #22, #27)
// ============================================================================

// FIX: Conversation history null logging
// When conversation_id provided but no messages found, log warning (see line 1228)
// When len(conversationHistory) == 0 after query, issue logged

// FIX: State cleanup ordering
// Message state cleanup should happen in defer block or BEFORE logging
// Location: Implement in next pass when refactoring message state management

// FIX: Extraction artifact concurrency
// Replace boolean locked flag with sync.RWMutex
// Location: models/extraction_artifact.go - use sync.RWMutex instead of bool

// FIX: Maturity recalculation timing
// Recalculate maturity after new context arrives (clarifications)
// Currently: calculated once at Layer 3. Should recalculate post-clarification.
// Impl: Add maturity recalculation trigger in clarification processing

// FIX: Response type selection logging
// When response type chosen, log the decision reason
// Add metadata["responseTypeReason"] tracking
// Log: "Response type selected: [type] (reason: [gap/conflict/maturity/etc])"

// FIX: Contact loading efficiency
// Batch load contacts by ID instead of individual queries
// Current: If extracting 3 contacts, might do 3 separate queries
// Impl: Use IN clause: SELECT * FROM contacts WHERE id IN (?, ?, ?)

// FIX: LLM timeout verification
// Verify 15-second timeout in intent_detector is applied everywhere
// Current status: Increased to 1800s in layer3, verify no overrides exist
// Check: grep -r "timeout\|15000\|15s" agents/ for hardcoded values

// FIX: Goroutine cleanup guarantee
// Ensure risk assessment goroutine cleanup in defer
// Current: riskChan is buffered size 1, but add explicit cleanup
// Add: defer close(riskChan) or use context.Context for cancellation

// FIX: Conflict detection duplication
// Remove duplicate conflict detection (main.go ~2700 AND Layer 5)
// Use only Layer 5.DetectedConflicts, remove main.go detection
// Consolidate to single source of truth in Layer 5
