package main

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"moly/auth"
	"moly/config"
	"moly/database"
	"moly/tools"
)

func TestFullFlow(t *testing.T) {
	t.Log("[FULL FLOW] ========== EXACT MAIN() SEQUENCE TEST ==========")

	// STEP 1: Initialize database (using REAL path like main() does)
	t.Log("[FULL FLOW] STEP 1: Database initialization...")
	v2dbPath := filepath.Join(os.ExpandEnv("$HOME/.moly"), "moly-v2-flow-test.db")
	os.RemoveAll(v2dbPath)

	v2db, err := database.Init(v2dbPath)
	if err != nil {
		t.Fatalf("[FULL FLOW] ❌ Database init failed: %v", err)
	}
	t.Log("[FULL FLOW] ✅ Database initialized")

	if err := v2db.GetConnection().Ping(); err != nil {
		t.Fatalf("[FULL FLOW] ❌ Ping failed after init: %v", err)
	}
	t.Log("[FULL FLOW] ✅ Database Ping OK")

	// STEP 2: Initialize LLM client
	t.Log("[FULL FLOW] STEP 2: Creating LLM client...")
	llmClient, err := tools.NewLLMClient()
	if err != nil {
		t.Fatalf("[FULL FLOW] ❌ LLM client failed: %v", err)
	}
	t.Log("[FULL FLOW] ✅ LLM client created")

	if err := v2db.GetConnection().Ping(); err != nil {
		t.Fatalf("[FULL FLOW] ❌ Database closed after LLM init: %v", err)
	}
	t.Log("[FULL FLOW] ✅ Database OK after LLM")

	// STEP 3: Container initialization
	t.Log("[FULL FLOW] STEP 3: Container initialization...")
	container := config.GetContainer()
	if err := container.Initialize(v2db, llmClient); err != nil {
		t.Fatalf("[FULL FLOW] ❌ Container init failed: %v", err)
	}
	t.Log("[FULL FLOW] ✅ Container initialized")

	if err := v2db.GetConnection().Ping(); err != nil {
		t.Fatalf("[FULL FLOW] ❌ DATABASE CLOSED AFTER CONTAINER INIT: %v", err)
	}
	t.Log("[FULL FLOW] ✅ Database OK after container init")

	// STEP 4: Create V2APIServer
	t.Log("[FULL FLOW] STEP 4: Creating V2APIServer...")
	v2Server, err := NewV2APIServer(llmClient, v2db)
	if err != nil {
		t.Fatalf("[FULL FLOW] ❌ V2APIServer creation failed: %v", err)
	}
	t.Log("[FULL FLOW] ✅ V2APIServer created")

	if err := v2db.GetConnection().Ping(); err != nil {
		t.Fatalf("[FULL FLOW] ❌ DATABASE CLOSED AFTER V2APIServer: %v", err)
	}
	t.Log("[FULL FLOW] ✅ Database OK after V2APIServer")

	// STEP 5: Create Auth server
	t.Log("[FULL FLOW] STEP 5: Creating Auth server...")
	authDB := config.GetContainer().GetDatabase().GetConnection()
	t.Logf("[FULL FLOW] Auth DB pointer: %p\n", authDB)

	if err := v2db.GetConnection().Ping(); err != nil {
		t.Fatalf("[FULL FLOW] ❌ DATABASE CLOSED after GetContainer: %v", err)
	}
	t.Log("[FULL FLOW] ✅ Database OK after GetContainer")

	userAuthServer := auth.NewUserAuthServer(authDB)
	t.Log("[FULL FLOW] ✅ UserAuthServer created")

	if err := v2db.GetConnection().Ping(); err != nil {
		t.Fatalf("[FULL FLOW] ❌ DATABASE CLOSED AFTER UserAuthServer: %v", err)
	}
	t.Log("[FULL FLOW] ✅ Database OK after UserAuthServer")

	// STEP 6: Register auth routes
	t.Log("[FULL FLOW] STEP 6: Registering auth routes...")
	http.HandleFunc("/api/auth/register", userAuthServer.RegisterHandler)
	http.HandleFunc("/api/auth/login", userAuthServer.LoginHandler)
	http.HandleFunc("/api/auth/verify", userAuthServer.VerifyTokenHandler)
	http.HandleFunc("/api/auth/logout", userAuthServer.LogoutHandler)
	t.Log("[FULL FLOW] ✅ Auth routes registered")

	if err := v2db.GetConnection().Ping(); err != nil {
		t.Fatalf("[FULL FLOW] ❌ DATABASE CLOSED AFTER AUTH ROUTES: %v", err)
	}
	t.Log("[FULL FLOW] ✅ Database OK after auth routes")

	// STEP 7: Register all major routes
	t.Log("[FULL FLOW] STEP 7: Registering all API routes...")
	http.HandleFunc("/api/v2/message-processor", v2Server.MessageProcessorHandler)
	http.HandleFunc("/api/v2/clarification/respond", v2Server.ClarificationResponseHandler)
	http.HandleFunc("/api/v2/incoming-message/analyze", v2Server.AnalyzeIncomingMessageHandler)
	http.HandleFunc("/api/v2/suggestion/choice", v2Server.SuggestionChoiceHandler)
	http.HandleFunc("/api/v2/about-me", v2Server.AboutMeHandler)
	http.HandleFunc("/api/v2/context", v2Server.ContextHandler)
	http.HandleFunc("GET /api/v2/conversations", v2Server.ConversationsHandler)
	http.HandleFunc("POST /api/v2/conversations", v2Server.ConversationsHandler)
	http.HandleFunc("DELETE /api/v2/conversations/{conversationID}", v2Server.ConversationsHandler)
	http.HandleFunc("GET /api/v2/contacts", v2Server.ContactsHandler)
	http.HandleFunc("POST /api/v2/contacts", v2Server.ContactsHandler)
	http.HandleFunc("GET /api/v2/contacts/{contactID}", v2Server.ContactDetailHandler)
	http.HandleFunc("GET /api/v2/questions/effectiveness", v2Server.QuestionEffectivenessHandler)
	http.HandleFunc("POST /api/v2/conversations/analyze", v2Server.AnalyzeConversationHandler)
	http.HandleFunc("/api/v2/metrics", v2Server.MetricsHandler)
	http.HandleFunc("DELETE /api/v2/user/delete", v2Server.DeleteProfileHandler)
	http.HandleFunc("/api/status", handleStatus(v2Server.database))
	http.HandleFunc("/api/analyze-mode-shift", handleAnalyzeModeShift)
	http.HandleFunc("/api/generate-questions", handleGenerateQuestions)
	http.HandleFunc("/api/constitution-principles", handleGetPrinciples)
	http.HandleFunc("/api/frontend-errors", handleFrontendErrors)
	t.Log("[FULL FLOW] ✅ All routes registered")

	if err := v2db.GetConnection().Ping(); err != nil {
		t.Fatalf("[FULL FLOW] ❌ DATABASE CLOSED AFTER ALL ROUTES: %v", err)
	}
	t.Log("[FULL FLOW] ✅ Database OK after all routes")

	// STEP 8: CORS middleware
	t.Log("[FULL FLOW] STEP 8: Setting up CORS middleware...")
	handler := corsMiddleware(http.DefaultServeMux)
	t.Log("[FULL FLOW] ✅ CORS middleware configured")

	if err := v2db.GetConnection().Ping(); err != nil {
		t.Fatalf("[FULL FLOW] ❌ DATABASE CLOSED AFTER CORS: %v", err)
	}
	t.Log("[FULL FLOW] ✅ Database OK after CORS")

	// Final check
	t.Log("[FULL FLOW] Final database check...")
	if err := v2db.GetConnection().Ping(); err != nil {
		t.Fatalf("[FULL FLOW] ❌ FINAL PING FAILED: %v", err)
	}
	t.Log("[FULL FLOW] ✅ FINAL: Database is ALIVE!")

	t.Log("[FULL FLOW] ========== FULL FLOW TEST PASSED - DATABASE ALIVE ==========")

	_ = v2Server
	_ = userAuthServer
	_ = handler
}
