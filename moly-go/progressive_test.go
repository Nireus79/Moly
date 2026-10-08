package main

import (
	"os"
	"path/filepath"
	"testing"

	"moly/agents"
	"moly/config"
	"moly/database"
	"moly/storage"
	"moly/tools"
)

func TestProgressiveInitialization(t *testing.T) {
	t.Log("[PROGRESSIVE] ========== PROGRESSIVE COMPONENT TEST ==========")

	// SETUP: Initialize basic database and repos
	t.Log("[PROGRESSIVE] SETUP: Initializing database...")
	dbPath := filepath.Join(os.ExpandEnv("$HOME/.moly"), "moly-prog-test.db")
	os.RemoveAll(dbPath)

	appDB, err := database.Init(dbPath)
	if err != nil {
		t.Fatalf("[PROGRESSIVE] Failed to init database: %v", err)
	}
	t.Log("[PROGRESSIVE] ✅ Database initialized")

	conn := appDB.GetConnection()
	chatRepo := database.NewChatMessageRepository(conn)
	ctxAttrRepo := database.NewContextAttributeRepository(appDB)
	convSummaryRepo := database.NewConversationSummaryRepository(conn)
	analysisCtxBuilder := database.NewAnalysisContextBuilder(appDB, convSummaryRepo, chatRepo, ctxAttrRepo)

	t.Log("[PROGRESSIVE] ✅ Basic repos created")

	// TEST 1: LLMCache
	t.Log("[PROGRESSIVE] TEST 1: Creating LLMCache...")
	llmCache := tools.NewDefaultLLMCache()
	if err := appDB.GetConnection().Ping(); err != nil {
		t.Fatalf("[PROGRESSIVE] ❌ Database closed after LLMCache: %v", err)
	}
	t.Log("[PROGRESSIVE] ✅ LLMCache OK")
	_ = llmCache

	// TEST 2: ExtractionStore
	t.Log("[PROGRESSIVE] TEST 2: Creating ExtractionStore...")
	extractionStore := tools.NewExtractionStore()
	if err := appDB.GetConnection().Ping(); err != nil {
		t.Fatalf("[PROGRESSIVE] ❌ Database closed after ExtractionStore: %v", err)
	}
	t.Log("[PROGRESSIVE] ✅ ExtractionStore OK")
	_ = extractionStore

	// TEST 3: ConflictDetector
	t.Log("[PROGRESSIVE] TEST 3: Creating ConflictDetector...")
	conflictDetector := agents.NewConflictDetector(appDB)
	if err := appDB.GetConnection().Ping(); err != nil {
		t.Fatalf("[PROGRESSIVE] ❌ Database closed after ConflictDetector: %v", err)
	}
	t.Log("[PROGRESSIVE] ✅ ConflictDetector OK")

	// TEST 4: MaturityService
	t.Log("[PROGRESSIVE] TEST 4: Creating MaturityService...")
	maturityService := storage.NewMaturityService(appDB)
	if err := appDB.GetConnection().Ping(); err != nil {
		t.Fatalf("[PROGRESSIVE] ❌ Database closed after MaturityService: %v", err)
	}
	t.Log("[PROGRESSIVE] ✅ MaturityService OK")
	_ = maturityService

	// TEST 5: ResponseValidator
	t.Log("[PROGRESSIVE] TEST 5: Creating ResponseValidator...")
	responseValidator := agents.NewResponseValidator(appDB)
	if err := appDB.GetConnection().Ping(); err != nil {
		t.Fatalf("[PROGRESSIVE] ❌ Database closed after ResponseValidator: %v", err)
	}
	t.Log("[PROGRESSIVE] ✅ ResponseValidator OK")
	_ = responseValidator

	// TEST 6: Layer5ConflictHandler
	t.Log("[PROGRESSIVE] TEST 6: Creating Layer5ConflictHandler...")
	layer5Handler := agents.NewLayer5ConflictHandler(appDB)
	if err := appDB.GetConnection().Ping(); err != nil {
		t.Fatalf("[PROGRESSIVE] ❌ Database closed after Layer5ConflictHandler: %v", err)
	}
	t.Log("[PROGRESSIVE] ✅ Layer5ConflictHandler OK")
	_ = layer5Handler

	// TEST 7: Constitution
	t.Log("[PROGRESSIVE] TEST 7: Loading Constitution...")
	constitution, err := config.LoadConstitution("config/constitution.yaml")
	if err != nil {
		t.Logf("[PROGRESSIVE] Note: Constitution load failed (expected in test): %v\n", err)
	} else {
		if err := appDB.GetConnection().Ping(); err != nil {
			t.Fatalf("[PROGRESSIVE] ❌ Database closed after Constitution: %v", err)
		}
		t.Log("[PROGRESSIVE] ✅ Constitution OK")
	}

	t.Log("[PROGRESSIVE] ========== ALL PROGRESSIVE TESTS PASSED ==========")
	_ = constitution
	_ = conflictDetector
	_ = analysisCtxBuilder
	_ = chatRepo
	_ = ctxAttrRepo
	_ = convSummaryRepo
}
