package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"moly/database"
	"moly/tools"
)

func testFullV2Server() {
	log.Println("[FULL SERVER] ========== FULL V2SERVER CREATION TEST ==========")

	// Initialize database
	log.Println("[FULL SERVER] Initializing database...")
	v2dbPath := filepath.Join(os.ExpandEnv("$HOME/.moly"), "moly-v2-full.db")
	os.RemoveAll(v2dbPath)

	v2db, err := database.Init(v2dbPath)
	if err != nil {
		log.Fatalf("[FULL SERVER] Failed to init database: %v", err)
	}
	log.Println("[FULL SERVER] ✅ Database initialized")

	// Test ping
	if err := v2db.GetConnection().Ping(); err != nil {
		log.Fatalf("[FULL SERVER] Ping failed after init: %v", err)
	}
	log.Println("[FULL SERVER] ✅ Database Ping OK before V2APIServer")

	// Create LLM client
	log.Println("[FULL SERVER] Creating LLM client...")
	llmClient, err := tools.NewLLMClient()
	if err != nil {
		log.Fatalf("[FULL SERVER] Failed to create LLM client: %v", err)
	}
	log.Println("[FULL SERVER] ✅ LLM client created")

	// Test ping before NewV2APIServer
	if err := v2db.GetConnection().Ping(); err != nil {
		log.Fatalf("[FULL SERVER] Ping failed before NewV2APIServer: %v", err)
	}
	log.Println("[FULL SERVER] ✅ Database Ping OK before NewV2APIServer call")

	// Now the critical call: NewV2APIServer
	log.Println("[FULL SERVER] Calling NewV2APIServer()...")
	v2Server, err := NewV2APIServer(llmClient, v2db)
	if err != nil {
		log.Fatalf("[FULL SERVER] NewV2APIServer failed: %v", err)
	}
	log.Println("[FULL SERVER] ✅ NewV2APIServer returned successfully")

	// Test ping AFTER NewV2APIServer
	if err := v2db.GetConnection().Ping(); err != nil {
		log.Fatalf("[FULL SERVER] ❌ DATABASE CLOSED AFTER NewV2APIServer! Error: %v", err)
	}
	log.Println("[FULL SERVER] ✅ Database Ping OK AFTER NewV2APIServer")

	log.Println("[FULL SERVER] NewV2APIServer created successfully")

	// Now test route registration (this is where database closes in real main())
	log.Println("[FULL SERVER] Testing route registration...")

	// Try accessing v2Server fields (what http handlers need)
	log.Println("[FULL SERVER] TEST: Accessing v2Server handler functions...")
	if v2Server == nil {
		log.Fatalf("[FULL SERVER] v2Server is nil!")
	}

	// Try to access the handler functions
	_ = v2Server.MessageProcessorHandler
	_ = v2Server.ClarificationResponseHandler

	if err := v2db.GetConnection().Ping(); err != nil {
		log.Fatalf("[FULL SERVER] ❌ DATABASE CLOSED after accessing handlers: %v", err)
	}
	log.Println("[FULL SERVER] ✅ Database OK after accessing handlers")

	// Try http.HandleFunc registration
	log.Println("[FULL SERVER] TEST: Registering HTTP handlers...")
	http.HandleFunc("/api/test1", v2Server.MessageProcessorHandler)

	if err := v2db.GetConnection().Ping(); err != nil {
		log.Fatalf("[FULL SERVER] ❌ DATABASE CLOSED after handler registration: %v", err)
	}
	log.Println("[FULL SERVER] ✅ Database OK after handler registration")

	log.Println("[FULL SERVER] ========== ROUTE REGISTRATION TEST PASSED - DATABASE ALIVE ==========")

	_ = v2Server
}
