package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"moly/database"
)

func TestDBLifecycle(t *testing.T) {
	t.Log("[TEST] ========== MINIMAL DATABASE TEST ==========")

	// STEP 1: Initialize database
	t.Log("[TEST] STEP 1: Initializing database...")
	dbPath := filepath.Join(os.ExpandEnv("$HOME/.moly"), "moly-test-lifecycle.db")
	os.RemoveAll(dbPath)

	appDB, err := database.Init(dbPath)
	if err != nil {
		t.Fatalf("[TEST] Failed to init database: %v", err)
	}
	t.Log("[TEST] ✅ Database initialized")

	// Test database is working
	if err := appDB.GetConnection().Ping(); err != nil {
		t.Fatalf("[TEST] Database Ping failed after init: %v", err)
	}
	t.Log("[TEST] ✅ Database Ping OK")

	// STEP 2: Get connection and create repos
	t.Log("[TEST] STEP 2: Creating repositories...")
	conn := appDB.GetConnection()
	t.Logf("[TEST] Got connection: %p\n", conn)

	chatRepo := database.NewChatMessageRepository(conn)
	ctxAttrRepo := database.NewContextAttributeRepository(appDB)
	convSummaryRepo := database.NewConversationSummaryRepository(conn)
	analysisCtxBuilder := database.NewAnalysisContextBuilder(appDB, convSummaryRepo, chatRepo, ctxAttrRepo)

	t.Log("[TEST] ✅ All repositories created")

	// STEP 3: Check if database is still working
	t.Log("[TEST] STEP 3: Testing database after repo creation...")
	if err := appDB.GetConnection().Ping(); err != nil {
		t.Fatalf("[TEST] ❌ DATABASE IS CLOSED after repo creation! Error: %v", err)
	}
	t.Log("[TEST] ✅ Database is STILL VALID")

	// STEP 4: Keep references alive
	t.Log("[TEST] STEP 4: Creating server struct with references...")
	type TestServer struct {
		db                 *database.Database
		chatRepo           *database.ChatMessageRepository
		ctxAttrRepo        *database.ContextAttributeRepository
		convSummaryRepo    *database.ConversationSummaryRepository
		analysisCtxBuilder *database.AnalysisContextBuilder
	}

	testServer := &TestServer{
		db:                 appDB,
		chatRepo:           chatRepo,
		ctxAttrRepo:        ctxAttrRepo,
		convSummaryRepo:    convSummaryRepo,
		analysisCtxBuilder: analysisCtxBuilder,
	}

	// Final check
	if err := appDB.GetConnection().Ping(); err != nil {
		t.Fatalf("[TEST] ❌ DATABASE IS CLOSED after struct creation! Error: %v", err)
	}
	t.Log("[TEST] ✅ Database is STILL VALID after struct creation")
	t.Log("[TEST] ✅ All references kept alive by testServer")

	fmt.Printf("[TEST] TestServer: %+v\n", testServer)
	t.Log("[TEST] ========== TEST COMPLETE - DATABASE ALIVE ==========")
}
