package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"moly/auth"
	"moly/config"
	"moly/database"
	"moly/tools"
)

func testFullFlow() {
	log.Println("[FULL FLOW] ========== EXACT MAIN() SEQUENCE TEST ==========")

	// STEP 1: Initialize database
	log.Println("[FULL FLOW] STEP 1: Database initialization...")
	v2dbPath := filepath.Join(os.ExpandEnv("$HOME/.moly"), "moly-v2-flow.db")
	os.RemoveAll(v2dbPath)

	v2db, err := database.Init(v2dbPath)
	if err != nil {
		log.Fatalf("[FULL FLOW] ❌ Database init failed: %v", err)
	}
	log.Println("[FULL FLOW] ✅ Database initialized")

	if err := v2db.GetConnection().Ping(); err != nil {
		log.Fatalf("[FULL FLOW] ❌ Ping failed after init: %v", err)
	}
	log.Println("[FULL FLOW] ✅ Database Ping OK")

	// STEP 2: Initialize LLM client
	log.Println("[FULL FLOW] STEP 2: Creating LLM client...")
	llmClient, err := tools.NewLLMClient()
	if err != nil {
		log.Fatalf("[FULL FLOW] ❌ LLM client failed: %v", err)
	}
	log.Println("[FULL FLOW] ✅ LLM client created")

	if err := v2db.GetConnection().Ping(); err != nil {
		log.Fatalf("[FULL FLOW] ❌ Database closed after LLM init: %v", err)
	}
	log.Println("[FULL FLOW] ✅ Database OK after LLM")

	// STEP 3: Container initialization
	log.Println("[FULL FLOW] STEP 3: Container initialization...")
	container := config.GetContainer()
	if err := container.Initialize(v2db, llmClient); err != nil {
		log.Fatalf("[FULL FLOW] ❌ Container init failed: %v", err)
	}
	log.Println("[FULL FLOW] ✅ Container initialized")

	if err := v2db.GetConnection().Ping(); err != nil {
		log.Fatalf("[FULL FLOW] ❌ DATABASE CLOSED AFTER CONTAINER INIT: %v", err)
	}
	log.Println("[FULL FLOW] ✅ Database OK after container init")

	// STEP 4: Create V2APIServer
	log.Println("[FULL FLOW] STEP 4: Creating V2APIServer...")
	v2Server, err := NewV2APIServer(llmClient, v2db)
	if err != nil {
		log.Fatalf("[FULL FLOW] ❌ V2APIServer creation failed: %v", err)
	}
	log.Println("[FULL FLOW] ✅ V2APIServer created")

	if err := v2db.GetConnection().Ping(); err != nil {
		log.Fatalf("[FULL FLOW] ❌ DATABASE CLOSED AFTER V2APIServer: %v", err)
	}
	log.Println("[FULL FLOW] ✅ Database OK after V2APIServer")

	// STEP 5: Create Auth server
	log.Println("[FULL FLOW] STEP 5: Creating Auth server...")
	authDB := config.GetContainer().GetDatabase().GetConnection()
	log.Printf("[FULL FLOW] Auth DB pointer: %p\n", authDB)

	if err := v2db.GetConnection().Ping(); err != nil {
		log.Fatalf("[FULL FLOW] ❌ DATABASE CLOSED after GetContainer: %v", err)
	}
	log.Println("[FULL FLOW] ✅ Database OK after GetContainer")

	userAuthServer := auth.NewUserAuthServer(authDB)
	log.Println("[FULL FLOW] ✅ UserAuthServer created")

	if err := v2db.GetConnection().Ping(); err != nil {
		log.Fatalf("[FULL FLOW] ❌ DATABASE CLOSED AFTER UserAuthServer: %v", err)
	}
	log.Println("[FULL FLOW] ✅ Database OK after UserAuthServer")

	// STEP 6: Register auth routes
	log.Println("[FULL FLOW] STEP 6: Registering auth routes...")
	http.HandleFunc("/api/auth/register", userAuthServer.RegisterHandler)
	http.HandleFunc("/api/auth/login", userAuthServer.LoginHandler)
	http.HandleFunc("/api/auth/verify", userAuthServer.VerifyTokenHandler)
	http.HandleFunc("/api/auth/logout", userAuthServer.LogoutHandler)
	log.Println("[FULL FLOW] ✅ Auth routes registered")

	if err := v2db.GetConnection().Ping(); err != nil {
		log.Fatalf("[FULL FLOW] ❌ DATABASE CLOSED AFTER AUTH ROUTES: %v", err)
	}
	log.Println("[FULL FLOW] ✅ Database OK after auth routes")

	// STEP 7: Register phase 5 routes
	log.Println("[FULL FLOW] STEP 7: Registering Phase 5 routes...")
	http.HandleFunc("/api/v2/message-processor", v2Server.MessageProcessorHandler)
	http.HandleFunc("/api/v2/clarification/respond", v2Server.ClarificationResponseHandler)
	http.HandleFunc("/api/v2/incoming-message/analyze", v2Server.AnalyzeIncomingMessageHandler)
	http.HandleFunc("/api/v2/suggestion/choice", v2Server.SuggestionChoiceHandler)
	log.Println("[FULL FLOW] ✅ Phase 5 routes registered")

	if err := v2db.GetConnection().Ping(); err != nil {
		log.Fatalf("[FULL FLOW] ❌ DATABASE CLOSED AFTER PHASE 5 ROUTES: %v", err)
	}
	log.Println("[FULL FLOW] ✅ Database OK after phase 5 routes")

	// STEP 8: Register context routes
	log.Println("[FULL FLOW] STEP 8: Registering context routes...")
	http.HandleFunc("/api/v2/about-me", v2Server.AboutMeHandler)
	http.HandleFunc("/api/v2/context", v2Server.ContextHandler)
	http.HandleFunc("GET /api/v2/conversations", v2Server.ConversationsHandler)
	http.HandleFunc("POST /api/v2/conversations", v2Server.ConversationsHandler)
	http.HandleFunc("DELETE /api/v2/conversations/{conversationID}", v2Server.ConversationsHandler)
	log.Println("[FULL FLOW] ✅ Context routes registered")

	if err := v2db.GetConnection().Ping(); err != nil {
		log.Fatalf("[FULL FLOW] ❌ DATABASE CLOSED AFTER CONTEXT ROUTES: %v", err)
	}
	log.Println("[FULL FLOW] ✅ Database OK after context routes")

	// STEP 9: Register contact routes
	log.Println("[FULL FLOW] STEP 9: Registering contact routes...")
	http.HandleFunc("GET /api/v2/contacts", v2Server.ContactsHandler)
	http.HandleFunc("POST /api/v2/contacts", v2Server.ContactsHandler)
	http.HandleFunc("GET /api/v2/contacts/{contactID}", v2Server.ContactDetailHandler)
	log.Println("[FULL FLOW] ✅ Contact routes registered")

	if err := v2db.GetConnection().Ping(); err != nil {
		log.Fatalf("[FULL FLOW] ❌ DATABASE CLOSED AFTER CONTACT ROUTES: %v", err)
	}
	log.Println("[FULL FLOW] ✅ Database OK after contact routes")

	// STEP 10: Register additional routes (questions, metrics, etc.)
	log.Println("[FULL FLOW] STEP 10: Registering additional routes...")
	http.HandleFunc("GET /api/v2/questions/effectiveness", v2Server.QuestionEffectivenessHandler)
	http.HandleFunc("POST /api/v2/conversations/analyze", v2Server.AnalyzeConversationHandler)
	http.HandleFunc("/api/v2/metrics", v2Server.MetricsHandler)
	log.Println("[FULL FLOW] ✅ Additional routes registered")

	if err := v2db.GetConnection().Ping(); err != nil {
		log.Fatalf("[FULL FLOW] ❌ DATABASE CLOSED AFTER ADDITIONAL ROUTES: %v", err)
	}
	log.Println("[FULL FLOW] ✅ Database OK after additional routes")

	// STEP 11: User account management routes
	log.Println("[FULL FLOW] STEP 11: Registering user account routes...")
	http.HandleFunc("DELETE /api/v2/user/delete", v2Server.DeleteProfileHandler)
	log.Println("[FULL FLOW] ✅ User account routes registered")

	if err := v2db.GetConnection().Ping(); err != nil {
		log.Fatalf("[FULL FLOW] ❌ DATABASE CLOSED AFTER USER ROUTES: %v", err)
	}
	log.Println("[FULL FLOW] ✅ Database OK after user routes")

	// STEP 12: CRITICAL - Health check handler (this calls handleStatus with v2Server.database!)
	log.Println("[FULL FLOW] STEP 12: Registering health check handler...")
	log.Println("[FULL FLOW] ⚠️  About to call handleStatus(v2Server.database) - THIS IS WHERE DATABASE CLOSES!")
	http.HandleFunc("/api/status", handleStatus(v2Server.database))
	log.Println("[FULL FLOW] ✅ Health check handler registered")

	if err := v2db.GetConnection().Ping(); err != nil {
		log.Fatalf("[FULL FLOW] ❌ DATABASE CLOSED AFTER HEALTH CHECK: %v", err)
	}
	log.Println("[FULL FLOW] ✅ Database OK after health check")

	// STEP 13: Safety & Ethics endpoints
	log.Println("[FULL FLOW] STEP 13: Registering safety endpoints...")
	http.HandleFunc("/api/analyze-mode-shift", handleAnalyzeModeShift)
	http.HandleFunc("/api/generate-questions", handleGenerateQuestions)
	http.HandleFunc("/api/constitution-principles", handleGetPrinciples)
	http.HandleFunc("/api/frontend-errors", handleFrontendErrors)
	log.Println("[FULL FLOW] ✅ Safety endpoints registered")

	if err := v2db.GetConnection().Ping(); err != nil {
		log.Fatalf("[FULL FLOW] ❌ DATABASE CLOSED AFTER SAFETY ENDPOINTS: %v", err)
	}
	log.Println("[FULL FLOW] ✅ Database OK after safety endpoints")

	// STEP 14: CORS middleware
	log.Println("[FULL FLOW] STEP 14: Setting up CORS middleware...")
	handler := corsMiddleware(http.DefaultServeMux)
	log.Println("[FULL FLOW] ✅ CORS middleware configured")

	if err := v2db.GetConnection().Ping(); err != nil {
		log.Fatalf("[FULL FLOW] ❌ DATABASE CLOSED AFTER CORS: %v", err)
	}
	log.Println("[FULL FLOW] ✅ Database OK after CORS")

	// STEP 15: Final check
	log.Println("[FULL FLOW] STEP 15: Final database check...")
	if err := v2db.GetConnection().Ping(); err != nil {
		log.Fatalf("[FULL FLOW] ❌ FINAL PING FAILED: %v", err)
	}
	log.Println("[FULL FLOW] ✅ FINAL: Database is ALIVE!")

	log.Println("[FULL FLOW] ========== FULL FLOW TEST PASSED - DATABASE ALIVE ==========")
	log.Printf("[FULL FLOW] Handler ready: %v\n", handler != nil)

	_ = v2Server
	_ = userAuthServer
	_ = handler
}
