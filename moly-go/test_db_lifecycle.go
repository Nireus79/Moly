package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"moly/database"
)

func testDBLifecycle() {
	log.Println("[TEST] ========== MINIMAL DATABASE TEST ==========")

	// STEP 1: Initialize database
	log.Println("[TEST] STEP 1: Initializing database...")
	v2dbPath := filepath.Join(os.ExpandEnv("$HOME/.moly"), "moly-v2-test.db")
	os.RemoveAll(v2dbPath)

	v2db, err := database.Init(v2dbPath)
	if err != nil {
		log.Fatalf("[TEST] Failed to init database: %v", err)
	}
	log.Println("[TEST] ✅ Database initialized")

	// Test database is working
	if err := v2db.GetConnection().Ping(); err != nil {
		log.Fatalf("[TEST] Database Ping failed after init: %v", err)
	}
	log.Println("[TEST] ✅ Database Ping OK")

	// STEP 2: Get connection and create repos
	log.Println("[TEST] STEP 2: Creating repositories...")
	conn := v2db.GetConnection()
	log.Printf("[TEST] Got connection: %p\n", conn)

	chatRepo := database.NewChatMessageRepository(conn)
	ctxAttrRepo := database.NewContextAttributeRepository(v2db)
	convSummaryRepo := database.NewConversationSummaryRepository(conn)
	analysisCtxBuilder := database.NewAnalysisContextBuilder(v2db, convSummaryRepo, chatRepo, ctxAttrRepo)

	log.Println("[TEST] ✅ All repositories created")

	// STEP 3: Check if database is still working
	log.Println("[TEST] STEP 3: Testing database after repo creation...")
	if err := v2db.GetConnection().Ping(); err != nil {
		log.Fatalf("[TEST] ❌ DATABASE IS CLOSED after repo creation! Error: %v", err)
	}
	log.Println("[TEST] ✅ Database is STILL VALID")

	// STEP 4: Keep references alive
	log.Println("[TEST] STEP 4: Creating server struct with references...")
	type TestServer struct {
		db                 *database.Database
		chatRepo           *database.ChatMessageRepository
		ctxAttrRepo        *database.ContextAttributeRepository
		convSummaryRepo    *database.ConversationSummaryRepository
		analysisCtxBuilder *database.AnalysisContextBuilder
	}

	testServer := &TestServer{
		db:                 v2db,
		chatRepo:           chatRepo,
		ctxAttrRepo:        ctxAttrRepo,
		convSummaryRepo:    convSummaryRepo,
		analysisCtxBuilder: analysisCtxBuilder,
	}

	// Final check
	if err := v2db.GetConnection().Ping(); err != nil {
		log.Fatalf("[TEST] ❌ DATABASE IS CLOSED after struct creation! Error: %v", err)
	}
	log.Println("[TEST] ✅ Database is STILL VALID after struct creation")
	log.Println("[TEST] ✅ All references kept alive by testServer")

	fmt.Printf("[TEST] TestServer: %+v\n", testServer)
	log.Println("[TEST] ========== TEST COMPLETE - DATABASE ALIVE ==========")
}
