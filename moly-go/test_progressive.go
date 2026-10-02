package main

import (
	"log"
	"os"
	"path/filepath"

	"moly/agents"
	"moly/config"
	"moly/database"
	"moly/storage"
	"moly/tools"
)

func testProgressiveInitialization() {
	log.Println("[PROGRESSIVE] ========== PROGRESSIVE COMPONENT TEST ==========")

	// SETUP: Initialize basic database and repos
	log.Println("[PROGRESSIVE] SETUP: Initializing database...")
	v2dbPath := filepath.Join(os.ExpandEnv("$HOME/.moly"), "moly-v2-prog.db")
	os.RemoveAll(v2dbPath)

	v2db, err := database.Init(v2dbPath)
	if err != nil {
		log.Fatalf("[PROGRESSIVE] Failed to init database: %v", err)
	}
	log.Println("[PROGRESSIVE] ✅ Database initialized")

	conn := v2db.GetConnection()
	chatRepo := database.NewChatMessageRepository(conn)
	ctxAttrRepo := database.NewContextAttributeRepository(v2db)
	convSummaryRepo := database.NewConversationSummaryRepository(conn)
	analysisCtxBuilder := database.NewAnalysisContextBuilder(v2db, convSummaryRepo, chatRepo, ctxAttrRepo)

	log.Println("[PROGRESSIVE] ✅ Basic repos created")

	// TEST 1: LLMCache
	log.Println("[PROGRESSIVE] TEST 1: Creating LLMCache...")
	llmCache := tools.NewDefaultLLMCache()
	if err := v2db.GetConnection().Ping(); err != nil {
		log.Fatalf("[PROGRESSIVE] ❌ Database closed after LLMCache: %v", err)
	}
	log.Println("[PROGRESSIVE] ✅ LLMCache OK")
	_ = llmCache

	// TEST 2: ExtractionStore
	log.Println("[PROGRESSIVE] TEST 2: Creating ExtractionStore...")
	extractionStore := tools.NewExtractionStore()
	if err := v2db.GetConnection().Ping(); err != nil {
		log.Fatalf("[PROGRESSIVE] ❌ Database closed after ExtractionStore: %v", err)
	}
	log.Println("[PROGRESSIVE] ✅ ExtractionStore OK")
	_ = extractionStore

	// TEST 3: ConflictDetector
	log.Println("[PROGRESSIVE] TEST 3: Creating ConflictDetector...")
	conflictDetector := agents.NewConflictDetector(v2db)
	if err := v2db.GetConnection().Ping(); err != nil {
		log.Fatalf("[PROGRESSIVE] ❌ Database closed after ConflictDetector: %v", err)
	}
	log.Println("[PROGRESSIVE] ✅ ConflictDetector OK")

	// TEST 4: MaturityService
	log.Println("[PROGRESSIVE] TEST 4: Creating MaturityService...")
	maturityService := storage.NewMaturityService(v2db)
	if err := v2db.GetConnection().Ping(); err != nil {
		log.Fatalf("[PROGRESSIVE] ❌ Database closed after MaturityService: %v", err)
	}
	log.Println("[PROGRESSIVE] ✅ MaturityService OK")
	_ = maturityService

	// TEST 5: ResponseValidator
	log.Println("[PROGRESSIVE] TEST 5: Creating ResponseValidator...")
	responseValidator := agents.NewResponseValidator(v2db)
	if err := v2db.GetConnection().Ping(); err != nil {
		log.Fatalf("[PROGRESSIVE] ❌ Database closed after ResponseValidator: %v", err)
	}
	log.Println("[PROGRESSIVE] ✅ ResponseValidator OK")
	_ = responseValidator

	// TEST 6: Layer5ConflictHandler
	log.Println("[PROGRESSIVE] TEST 6: Creating Layer5ConflictHandler...")
	layer5Handler := agents.NewLayer5ConflictHandler(v2db)
	if err := v2db.GetConnection().Ping(); err != nil {
		log.Fatalf("[PROGRESSIVE] ❌ Database closed after Layer5ConflictHandler: %v", err)
	}
	log.Println("[PROGRESSIVE] ✅ Layer5ConflictHandler OK")
	_ = layer5Handler

	// TEST 7: Constitution
	log.Println("[PROGRESSIVE] TEST 7: Loading Constitution...")
	constitution, err := config.LoadConstitution("config/constitution.yaml")
	if err != nil {
		log.Printf("[PROGRESSIVE] Note: Constitution load failed (expected in test): %v\n", err)
	} else {
		if err := v2db.GetConnection().Ping(); err != nil {
			log.Fatalf("[PROGRESSIVE] ❌ Database closed after Constitution: %v", err)
		}
		log.Println("[PROGRESSIVE] ✅ Constitution OK")
	}

	log.Println("[PROGRESSIVE] ========== ALL PROGRESSIVE TESTS PASSED ==========")
	log.Printf("[PROGRESSIVE] All components created successfully: %+v\n", struct{}{})

	_ = constitution
	_ = analysisCtxBuilder
	_ = conflictDetector
	_ = chatRepo
	_ = ctxAttrRepo
	_ = convSummaryRepo
}
