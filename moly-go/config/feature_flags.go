package config

import (
	"fmt"
	"os"
	"strconv"
	"sync"
)

// FeatureFlags controls experimental and gradual rollout features
// Each flag starts with default false and can be enabled via environment variable
type FeatureFlags struct {
	// Phase 1: Extraction Lock
	UseExtractionLock bool // Default: false. Start at 10% users, expand to 100%

	// Phase 2: Layer 5 Conflict Channeling
	UseLayer5ConflictGate bool // Default: false. Start at 10% users, expand to 100%

	// Phase 3: Constrained Response Generation
	UseConstrainedResponseGeneration bool // Default: false. Start at 10% users, expand to 100%

	// Phase 4: Clean Schema Migration
	UseCleanSchema bool // Default: false. One-time migration flag

	// Monitoring and metrics
	EnableMetrics         bool // Default: true. Collect performance metrics
	EnableDetailedLogging bool // Default: false. Verbose logging for debugging
	MetricsCollectionRate int  // Percentage of requests to collect metrics (0-100)
	DetailedLoggingRate   int  // Percentage of requests to log detailed output (0-100)

	mu sync.RWMutex
}

// Global feature flags instance
var globalFlags *FeatureFlags
var flagsMutex sync.Once

// GetFeatureFlags returns the global feature flags instance
func GetFeatureFlags() *FeatureFlags {
	flagsMutex.Do(func() {
		globalFlags = &FeatureFlags{
			UseExtractionLock:                getBoolEnv("MOLY_USE_EXTRACTION_LOCK", false),
			UseLayer5ConflictGate:            getBoolEnv("MOLY_USE_LAYER5_CONFLICT_GATE", false),
			UseConstrainedResponseGeneration: getBoolEnv("MOLY_USE_CONSTRAINED_GENERATION", false),
			UseCleanSchema:                   getBoolEnv("MOLY_USE_CLEAN_SCHEMA", false),
			EnableMetrics:                    getBoolEnv("MOLY_ENABLE_METRICS", true),
			EnableDetailedLogging:            getBoolEnv("MOLY_ENABLE_DETAILED_LOGGING", false),
			MetricsCollectionRate:            getIntEnv("MOLY_METRICS_RATE", 100),
			DetailedLoggingRate:              getIntEnv("MOLY_LOGGING_RATE", 0),
		}
	})
	return globalFlags
}

// SetFlag updates a feature flag at runtime (for testing)
func (ff *FeatureFlags) SetFlag(flagName string, value bool) error {
	ff.mu.Lock()
	defer ff.mu.Unlock()

	switch flagName {
	case "UseExtractionLock":
		ff.UseExtractionLock = value
	case "UseLayer5ConflictGate":
		ff.UseLayer5ConflictGate = value
	case "UseConstrainedResponseGeneration":
		ff.UseConstrainedResponseGeneration = value
	case "UseCleanSchema":
		ff.UseCleanSchema = value
	case "EnableMetrics":
		ff.EnableMetrics = value
	case "EnableDetailedLogging":
		ff.EnableDetailedLogging = value
	default:
		return fmt.Errorf("unknown feature flag: %s", flagName)
	}

	return nil
}

// IsEnabled checks if a feature flag is enabled
func (ff *FeatureFlags) IsEnabled(flagName string) bool {
	ff.mu.RLock()
	defer ff.mu.RUnlock()

	switch flagName {
	case "UseExtractionLock":
		return ff.UseExtractionLock
	case "UseLayer5ConflictGate":
		return ff.UseLayer5ConflictGate
	case "UseConstrainedResponseGeneration":
		return ff.UseConstrainedResponseGeneration
	case "UseCleanSchema":
		return ff.UseCleanSchema
	case "EnableMetrics":
		return ff.EnableMetrics
	case "EnableDetailedLogging":
		return ff.EnableDetailedLogging
	default:
		return false
	}
}

// GetStatus returns current status of all flags
func (ff *FeatureFlags) GetStatus() map[string]interface{} {
	ff.mu.RLock()
	defer ff.mu.RUnlock()

	return map[string]interface{}{
		"phase1_extraction_lock":        ff.UseExtractionLock,
		"phase2_layer5_conflict_gate":   ff.UseLayer5ConflictGate,
		"phase3_constrained_generation": ff.UseConstrainedResponseGeneration,
		"phase4_clean_schema":           ff.UseCleanSchema,
		"metrics_enabled":               ff.EnableMetrics,
		"detailed_logging_enabled":      ff.EnableDetailedLogging,
		"metrics_collection_rate":       ff.MetricsCollectionRate,
		"detailed_logging_rate":         ff.DetailedLoggingRate,
	}
}

// Helper functions
func getBoolEnv(key string, defaultVal bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return defaultVal
	}
	return b
}

func getIntEnv(key string, defaultVal int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	if i < 0 || i > 100 {
		return defaultVal
	}
	return i
}
