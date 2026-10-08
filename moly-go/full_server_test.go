package main

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"moly/database"
	"moly/tools"
)

func TestFullServer(t *testing.T) {
	t.Log("[FULL SERVER] ========== FULL SERVER CREATION TEST ==========")

	// Initialize database
	t.Log("[FULL SERVER] Initializing database...")
	dbPath := filepath.Join(os.ExpandEnv("$HOME/.moly"), "moly-server-test.db")
	os.RemoveAll(dbPath)

	appDB, err := database.Init(dbPath)
	if err != nil {
		t.Fatalf("[FULL SERVER] Failed to init database: %v", err)
	}
	t.Log("[FULL SERVER] ✅ Database initialized")

	// Test ping
	if err := appDB.GetConnection().Ping(); err != nil {
		t.Fatalf("[FULL SERVER] Ping failed after init: %v", err)
	}
	t.Log("[FULL SERVER] ✅ Database Ping OK before APIServer")

	// Create LLM client
	t.Log("[FULL SERVER] Creating LLM client...")
	llmClient, err := tools.NewLLMClient()
	if err != nil {
		t.Fatalf("[FULL SERVER] Failed to create LLM client: %v", err)
	}
	t.Log("[FULL SERVER] ✅ LLM client created")

	// Test ping before NewAPIServer
	if err := appDB.GetConnection().Ping(); err != nil {
		t.Fatalf("[FULL SERVER] Ping failed before NewAPIServer: %v", err)
	}
	t.Log("[FULL SERVER] ✅ Database Ping OK before NewAPIServer call")

	// Now the critical call: NewAPIServer
	t.Log("[FULL SERVER] Calling NewAPIServer()...")
	apiServer, err := NewAPIServer(llmClient, appDB)
	if err != nil {
		t.Fatalf("[FULL SERVER] NewAPIServer failed: %v", err)
	}
	t.Log("[FULL SERVER] ✅ NewAPIServer returned successfully")

	// Test ping AFTER NewAPIServer
	if err := appDB.GetConnection().Ping(); err != nil {
		t.Fatalf("[FULL SERVER] ❌ DATABASE CLOSED AFTER NewAPIServer! Error: %v", err)
	}
	t.Log("[FULL SERVER] ✅ Database Ping OK AFTER NewAPIServer")

	// Try accessing apiServer fields
	t.Log("[FULL SERVER] TEST: Accessing apiServer handler functions...")
	if apiServer == nil {
		t.Fatal("[FULL SERVER] apiServer is nil!")
	}

	// Try to access the handler functions
	_ = apiServer.MessageProcessorHandler
	_ = apiServer.ClarificationResponseHandler

	if err := appDB.GetConnection().Ping(); err != nil {
		t.Fatalf("[FULL SERVER] ❌ DATABASE CLOSED after accessing handlers: %v", err)
	}
	t.Log("[FULL SERVER] ✅ Database OK after accessing handlers")

	// Try http.HandleFunc registration
	t.Log("[FULL SERVER] TEST: Registering HTTP handlers...")
	http.HandleFunc("/api/test1", apiServer.MessageProcessorHandler)

	if err := appDB.GetConnection().Ping(); err != nil {
		t.Fatalf("[FULL SERVER] ❌ DATABASE CLOSED after handler registration: %v", err)
	}
	t.Log("[FULL SERVER] ✅ Database OK after handler registration")

	t.Log("[FULL SERVER] ========== ROUTE REGISTRATION TEST PASSED - DATABASE ALIVE ==========")

	_ = apiServer
}
