package main

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"moly/database"
	"moly/tools"
)

func TestFullV2Server(t *testing.T) {
	t.Log("[FULL SERVER] ========== FULL V2SERVER CREATION TEST ==========")

	// Initialize database
	t.Log("[FULL SERVER] Initializing database...")
	v2dbPath := filepath.Join(os.ExpandEnv("$HOME/.moly"), "moly-v2-server-test.db")
	os.RemoveAll(v2dbPath)

	v2db, err := database.Init(v2dbPath)
	if err != nil {
		t.Fatalf("[FULL SERVER] Failed to init database: %v", err)
	}
	t.Log("[FULL SERVER] ✅ Database initialized")

	// Test ping
	if err := v2db.GetConnection().Ping(); err != nil {
		t.Fatalf("[FULL SERVER] Ping failed after init: %v", err)
	}
	t.Log("[FULL SERVER] ✅ Database Ping OK before V2APIServer")

	// Create LLM client
	t.Log("[FULL SERVER] Creating LLM client...")
	llmClient, err := tools.NewLLMClient()
	if err != nil {
		t.Fatalf("[FULL SERVER] Failed to create LLM client: %v", err)
	}
	t.Log("[FULL SERVER] ✅ LLM client created")

	// Test ping before NewV2APIServer
	if err := v2db.GetConnection().Ping(); err != nil {
		t.Fatalf("[FULL SERVER] Ping failed before NewV2APIServer: %v", err)
	}
	t.Log("[FULL SERVER] ✅ Database Ping OK before NewV2APIServer call")

	// Now the critical call: NewV2APIServer
	t.Log("[FULL SERVER] Calling NewV2APIServer()...")
	v2Server, err := NewV2APIServer(llmClient, v2db)
	if err != nil {
		t.Fatalf("[FULL SERVER] NewV2APIServer failed: %v", err)
	}
	t.Log("[FULL SERVER] ✅ NewV2APIServer returned successfully")

	// Test ping AFTER NewV2APIServer
	if err := v2db.GetConnection().Ping(); err != nil {
		t.Fatalf("[FULL SERVER] ❌ DATABASE CLOSED AFTER NewV2APIServer! Error: %v", err)
	}
	t.Log("[FULL SERVER] ✅ Database Ping OK AFTER NewV2APIServer")

	// Try accessing v2Server fields
	t.Log("[FULL SERVER] TEST: Accessing v2Server handler functions...")
	if v2Server == nil {
		t.Fatal("[FULL SERVER] v2Server is nil!")
	}

	// Try to access the handler functions
	_ = v2Server.MessageProcessorHandler
	_ = v2Server.ClarificationResponseHandler

	if err := v2db.GetConnection().Ping(); err != nil {
		t.Fatalf("[FULL SERVER] ❌ DATABASE CLOSED after accessing handlers: %v", err)
	}
	t.Log("[FULL SERVER] ✅ Database OK after accessing handlers")

	// Try http.HandleFunc registration
	t.Log("[FULL SERVER] TEST: Registering HTTP handlers...")
	http.HandleFunc("/api/test1", v2Server.MessageProcessorHandler)

	if err := v2db.GetConnection().Ping(); err != nil {
		t.Fatalf("[FULL SERVER] ❌ DATABASE CLOSED after handler registration: %v", err)
	}
	t.Log("[FULL SERVER] ✅ Database OK after handler registration")

	t.Log("[FULL SERVER] ========== ROUTE REGISTRATION TEST PASSED - DATABASE ALIVE ==========")

	_ = v2Server
}
