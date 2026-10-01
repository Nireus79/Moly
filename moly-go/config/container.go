package config

import (
	"fmt"
	"log"
	"sync"

	"moly/database"
	"moly/tools"
)

// ServiceContainer manages all service dependencies
// PHASE 2.2: Dependency Injection Framework - Replaces global variables
type ServiceContainer struct {
	db              *database.Database
	llmClient       tools.LLMProvider
	featureFlags    *FeatureFlags
	metrics         interface{} // monitoring.Metrics
	initialized     bool
	mu              sync.RWMutex
	cleanupHandlers []func() error
}

// NewServiceContainer creates a new DI container
func NewServiceContainer() *ServiceContainer {
	return &ServiceContainer{
		cleanupHandlers: make([]func() error, 0),
	}
}

// Initialize sets up all container dependencies
func (sc *ServiceContainer) Initialize(db *database.Database, llm tools.LLMProvider) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if sc.initialized {
		return fmt.Errorf("container already initialized")
	}

	if db == nil {
		return fmt.Errorf("database cannot be nil")
	}

	if llm == nil {
		return fmt.Errorf("LLM client cannot be nil")
	}

	sc.db = db
	sc.llmClient = llm
	sc.featureFlags = GetFeatureFlags()
	sc.initialized = true

	log.Printf("[ServiceContainer] ✅ DI container initialized (db=%p, llm=%p)", db, llm)
	return nil
}

// GetDatabase returns the database connection
func (sc *ServiceContainer) GetDatabase() *database.Database {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	if !sc.initialized {
		log.Printf("[ServiceContainer] ERROR: Container not initialized, returning nil database")
		return nil
	}

	return sc.db
}

// GetLLMClient returns the LLM client
func (sc *ServiceContainer) GetLLMClient() tools.LLMProvider {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	if !sc.initialized {
		log.Printf("[ServiceContainer] ERROR: Container not initialized, returning nil LLM")
		return nil
	}

	return sc.llmClient
}

// GetFeatureFlags returns the feature flags
func (sc *ServiceContainer) GetFeatureFlags() *FeatureFlags {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	if sc.featureFlags == nil {
		sc.featureFlags = GetFeatureFlags()
	}

	return sc.featureFlags
}

// RegisterCleanup registers a cleanup function to be called on shutdown
// PHASE 2.2: Resource lifecycle management
func (sc *ServiceContainer) RegisterCleanup(fn func() error) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.cleanupHandlers = append(sc.cleanupHandlers, fn)
}

// Shutdown gracefully shuts down the container
func (sc *ServiceContainer) Shutdown() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if !sc.initialized {
		return fmt.Errorf("container not initialized")
	}

	var errs []error

	// Call cleanup handlers in reverse order
	for i := len(sc.cleanupHandlers) - 1; i >= 0; i-- {
		if err := sc.cleanupHandlers[i](); err != nil {
			log.Printf("[ServiceContainer] Cleanup error: %v", err)
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("cleanup errors: %v", errs)
	}

	log.Printf("[ServiceContainer] ✅ Container shutdown complete")
	sc.initialized = false
	return nil
}

// Global container instance
var globalContainer *ServiceContainer
var containerMutex sync.Once

// GetContainer returns the global service container
func GetContainer() *ServiceContainer {
	containerMutex.Do(func() {
		globalContainer = NewServiceContainer()
		log.Printf("[ServiceContainer] ✅ Global container created")
	})
	return globalContainer
}
