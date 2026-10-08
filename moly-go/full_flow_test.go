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
	dbPath := filepath.Join(os.ExpandEnv("$HOME/.moly"), "moly-flow-test.db")
	os.RemoveAll(dbPath)

	appDB, err := database.Init(dbPath)
	if err != nil {
		t.Fatalf("[FULL FLOW] ❌ Database init failed: %v", err)
	}
	t.Log("[FULL FLOW] ✅ Database initialized")

	if err := appDB.GetConnection().Ping(); err != nil {
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

	if err := appDB.GetConnection().Ping(); err != nil {
		t.Fatalf("[FULL FLOW] ❌ Database closed after LLM init: %v", err)
	}
	t.Log("[FULL FLOW] ✅ Database OK after LLM")

	// STEP 3: Container initialization
	t.Log("[FULL FLOW] STEP 3: Container initialization...")
	container := config.GetContainer()
	if err := container.Initialize(appDB, llmClient); err != nil {
		t.Fatalf("[FULL FLOW] ❌ Container init failed: %v", err)
	}
	t.Log("[FULL FLOW] ✅ Container initialized")

	if err := appDB.GetConnection().Ping(); err != nil {
		t.Fatalf("[FULL FLOW] ❌ DATABASE CLOSED AFTER CONTAINER INIT: %v", err)
	}
	t.Log("[FULL FLOW] ✅ Database OK after container init")

	// STEP 4: Create APIServer
	t.Log("[FULL FLOW] STEP 4: Creating APIServer...")
	apiServer, err := NewAPIServer(llmClient, appDB)
	if err != nil {
		t.Fatalf("[FULL FLOW] ❌ APIServer creation failed: %v", err)
	}
	t.Log("[FULL FLOW] ✅ APIServer created")

	if err := appDB.GetConnection().Ping(); err != nil {
		t.Fatalf("[FULL FLOW] ❌ DATABASE CLOSED AFTER APIServer: %v", err)
	}
	t.Log("[FULL FLOW] ✅ Database OK after APIServer")

	// STEP 5: Create Auth server
	t.Log("[FULL FLOW] STEP 5: Creating Auth server...")
	authDB := config.GetContainer().GetDatabase().GetConnection()
	t.Logf("[FULL FLOW] Auth DB pointer: %p\n", authDB)

	if err := appDB.GetConnection().Ping(); err != nil {
		t.Fatalf("[FULL FLOW] ❌ DATABASE CLOSED after GetContainer: %v", err)
	}
	t.Log("[FULL FLOW] ✅ Database OK after GetContainer")

	userAuthServer := auth.NewUserAuthServer(authDB)
	t.Log("[FULL FLOW] ✅ UserAuthServer created")

	if err := appDB.GetConnection().Ping(); err != nil {
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

	if err := appDB.GetConnection().Ping(); err != nil {
		t.Fatalf("[FULL FLOW] ❌ DATABASE CLOSED AFTER AUTH ROUTES: %v", err)
	}
	t.Log("[FULL FLOW] ✅ Database OK after auth routes")

	// STEP 7: Register all major routes
	t.Log("[FULL FLOW] STEP 7: Registering all API routes...")
	http.HandleFunc("/api/message-processor", apiServer.MessageProcessorHandler)
	http.HandleFunc("/api/clarification/respond", apiServer.ClarificationResponseHandler)
	http.HandleFunc("/api/incoming-message/analyze", apiServer.AnalyzeIncomingMessageHandler)
	http.HandleFunc("/api/suggestion/choice", apiServer.SuggestionChoiceHandler)
	http.HandleFunc("/api/about-me", apiServer.AboutMeHandler)
	http.HandleFunc("/api/context", apiServer.ContextHandler)
	http.HandleFunc("GET /api/conversations", apiServer.ConversationsHandler)
	http.HandleFunc("POST /api/conversations", apiServer.ConversationsHandler)
	http.HandleFunc("DELETE /api/conversations/{conversationID}", apiServer.ConversationsHandler)
	http.HandleFunc("GET /api/contacts", apiServer.ContactsHandler)
	http.HandleFunc("POST /api/contacts", apiServer.ContactsHandler)
	http.HandleFunc("GET /api/contacts/{contactID}", apiServer.ContactDetailHandler)
	http.HandleFunc("GET /api/questions/effectiveness", apiServer.QuestionEffectivenessHandler)
	http.HandleFunc("POST /api/conversations/analyze", apiServer.AnalyzeConversationHandler)
	http.HandleFunc("/api/metrics", apiServer.MetricsHandler)
	http.HandleFunc("DELETE /api/user/delete", apiServer.DeleteProfileHandler)
	http.HandleFunc("/api/status", handleStatus(apiServer.database))
	http.HandleFunc("/api/analyze-mode-shift", handleAnalyzeModeShift)
	http.HandleFunc("/api/generate-questions", handleGenerateQuestions)
	http.HandleFunc("/api/constitution-principles", handleGetPrinciples)
	http.HandleFunc("/api/frontend-errors", handleFrontendErrors)
	t.Log("[FULL FLOW] ✅ All routes registered")

	if err := appDB.GetConnection().Ping(); err != nil {
		t.Fatalf("[FULL FLOW] ❌ DATABASE CLOSED AFTER ALL ROUTES: %v", err)
	}
	t.Log("[FULL FLOW] ✅ Database OK after all routes")

	// STEP 8: CORS middleware
	t.Log("[FULL FLOW] STEP 8: Setting up CORS middleware...")
	handler := corsMiddleware(http.DefaultServeMux)
	t.Log("[FULL FLOW] ✅ CORS middleware configured")

	if err := appDB.GetConnection().Ping(); err != nil {
		t.Fatalf("[FULL FLOW] ❌ DATABASE CLOSED AFTER CORS: %v", err)
	}
	t.Log("[FULL FLOW] ✅ Database OK after CORS")

	// Final check
	t.Log("[FULL FLOW] Final database check...")
	if err := appDB.GetConnection().Ping(); err != nil {
		t.Fatalf("[FULL FLOW] ❌ FINAL PING FAILED: %v", err)
	}
	t.Log("[FULL FLOW] ✅ FINAL: Database is ALIVE!")

	t.Log("[FULL FLOW] ========== FULL FLOW TEST PASSED - DATABASE ALIVE ==========")

	_ = apiServer
	_ = userAuthServer
	_ = handler
}
