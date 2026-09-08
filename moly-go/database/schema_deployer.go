package database

import (
	"database/sql"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DeploymentConfig holds deployment configuration
type DeploymentConfig struct {
	DatabasePath string
	BackupBefore bool
	Verify       bool
	Environment  string // "development", "staging", "production"
	DryRun       bool
	Verbose      bool
}

// DeploymentResult holds deployment results
type DeploymentResult struct {
	Success         bool
	TablesCreated   int
	IndexesCreated  int
	ExecutionTime   time.Duration
	BackupPath      string
	Errors          []string
	Warnings        []string
	VerificationPassed bool
}

// SchemaDeployer handles Phase 1.2 schema deployment
type SchemaDeployer struct {
	config DeploymentConfig
	db     *sql.DB
	result DeploymentResult
	logger *log.Logger
}

// NewSchemaDeployer creates a new schema deployer
func NewSchemaDeployer(config DeploymentConfig) *SchemaDeployer {
	return &SchemaDeployer{
		config:  config,
		logger:  log.New(os.Stdout, "[SchemaDeployer] ", log.LstdFlags),
		result:  DeploymentResult{},
	}
}

// Deploy executes the schema deployment
func (sd *SchemaDeployer) Deploy() error {
	startTime := time.Now()
	defer func() {
		sd.result.ExecutionTime = time.Since(startTime)
	}()

	sd.logger.Println("Starting Phase 1.2 schema deployment...")
	sd.logger.Printf("Environment: %s\n", sd.config.Environment)
	sd.logger.Printf("Database: %s\n", sd.config.DatabasePath)

	// Step 1: Backup
	if sd.config.BackupBefore {
		sd.logger.Println("Step 1: Backing up database...")
		if err := sd.backupDatabase(); err != nil {
			sd.addError("backup failed: " + err.Error())
			return err
		}
	}

	// Step 2: Verify health
	sd.logger.Println("Step 2: Verifying database health...")
	if err := sd.verifyDatabaseHealth(); err != nil {
		sd.addError("health check failed: " + err.Error())
		return err
	}

	// Step 3: Connect to database
	sd.logger.Println("Step 3: Connecting to database...")
	db, err := sql.Open("sqlite3", sd.config.DatabasePath)
	if err != nil {
		sd.addError("failed to open database: " + err.Error())
		return err
	}
	defer db.Close()
	sd.db = db

	// Enable foreign keys
	if _, err := sd.db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		sd.addWarning("failed to enable foreign keys: " + err.Error())
	}

	// Step 4: Load and execute schema
	sd.logger.Println("Step 4: Loading schema file...")
	schemaSQL, err := sd.loadSchema()
	if err != nil {
		sd.addError("failed to load schema: " + err.Error())
		return err
	}

	sd.logger.Println("Step 5: Executing schema deployment...")
	if err := sd.executeSchema(schemaSQL); err != nil {
		sd.addError("schema execution failed: " + err.Error())
		return err
	}

	// Step 6: Verify deployment (if enabled)
	if sd.config.Verify {
		sd.logger.Println("Step 6: Verifying deployment...")
		if err := sd.verifyDeployment(); err != nil {
			sd.addError("verification failed: " + err.Error())
			return err
		}
	}

	sd.result.Success = true
	sd.logger.Printf("✅ Deployment successful in %v\n", sd.result.ExecutionTime)

	return nil
}

// backupDatabase creates a backup of the database
func (sd *SchemaDeployer) backupDatabase() error {
	if _, err := os.Stat(sd.config.DatabasePath); err != nil {
		return fmt.Errorf("database not found: %w", err)
	}

	backupPath := fmt.Sprintf("%s.backup.%s", sd.config.DatabasePath, time.Now().Format("20060102_150405"))

	if sd.config.DryRun {
		sd.logger.Printf("DRY RUN: Would backup to %s\n", backupPath)
		sd.result.BackupPath = backupPath
		return nil
	}

	// Use SQLite's VACUUM INTO for atomic backup
	db, err := sql.Open("sqlite3", sd.config.DatabasePath)
	if err != nil {
		return err
	}
	defer db.Close()

	backupSQL := fmt.Sprintf("VACUUM INTO '%s';", backupPath)
	if _, err := db.Exec(backupSQL); err != nil {
		// Fallback: use manual copy
		data, err := ioutil.ReadFile(sd.config.DatabasePath)
		if err != nil {
			return fmt.Errorf("backup failed: %w", err)
		}
		if err := ioutil.WriteFile(backupPath, data, 0644); err != nil {
			return fmt.Errorf("backup write failed: %w", err)
		}
	}

	sd.logger.Printf("✅ Backup created: %s\n", backupPath)
	sd.result.BackupPath = backupPath
	return nil
}

// verifyDatabaseHealth checks database integrity
func (sd *SchemaDeployer) verifyDatabaseHealth() error {
	db, err := sql.Open("sqlite3", sd.config.DatabasePath)
	if err != nil {
		return fmt.Errorf("failed to open database for health check: %w", err)
	}
	defer db.Close()

	var result string
	err = db.QueryRow("PRAGMA integrity_check;").Scan(&result)
	if err != nil {
		return fmt.Errorf("integrity check failed: %w", err)
	}

	if result != "ok" {
		return fmt.Errorf("database corruption detected: %s", result)
	}

	sd.logger.Println("✅ Database integrity verified")
	return nil
}

// loadSchema reads the schema SQL file
func (sd *SchemaDeployer) loadSchema() (string, error) {
	schemaPath := filepath.Join(filepath.Dir(os.Args[0]), "database", "deploy_schema_v2_1_phase_1_2.sql")

	// Try multiple paths
	possiblePaths := []string{
		schemaPath,
		"moly-go/database/deploy_schema_v2_1_phase_1_2.sql",
		"database/deploy_schema_v2_1_phase_1_2.sql",
		"./deploy_schema_v2_1_phase_1_2.sql",
	}

	var data []byte
	var lastErr error

	for _, path := range possiblePaths {
		data, err := ioutil.ReadFile(path)
		if err == nil {
			sd.logger.Printf("✅ Schema loaded from: %s\n", path)
			return string(data), nil
		}
		lastErr = err
	}

	return "", fmt.Errorf("schema file not found: %w", lastErr)
}

// executeSchema executes the schema SQL
func (sd *SchemaDeployer) executeSchema(schemaSQL string) error {
	if sd.config.DryRun {
		sd.logger.Println("DRY RUN: Schema execution skipped")
		return nil
	}

	// Split into individual statements (simple approach)
	statements := strings.Split(schemaSQL, ";")

	tx, err := sd.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}

		if sd.config.Verbose {
			sd.logger.Printf("Executing: %s...\n", stmt[:min(50, len(stmt))])
		}

		if _, err := tx.Exec(stmt); err != nil {
			tx.Rollback()
			return fmt.Errorf("schema execution failed: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit schema changes: %w", err)
	}

	sd.logger.Println("✅ Schema executed successfully")
	return nil
}

// verifyDeployment verifies the schema was deployed correctly
func (sd *SchemaDeployer) verifyDeployment() error {
	// Check if Phase 1.2 tables exist
	tables := []string{
		"about_me_profile",
		"contacts",
		"contact_communication_patterns",
		"communication_patterns",
		"communication_goals",
		"reflection_journal",
		"implicit_learning",
		"conversation_ephemeral",
		"extraction_queue",
	}

	var tableCount int
	for _, table := range tables {
		var count int
		err := sd.db.QueryRow(
			"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?",
			table,
		).Scan(&count)
		if err != nil || count == 0 {
			sd.addError(fmt.Sprintf("table not found: %s", table))
			return fmt.Errorf("verification failed: table %s not found", table)
		}
		tableCount++
	}

	sd.result.TablesCreated = tableCount

	// Check indexes
	var indexCount int
	err := sd.db.QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name LIKE 'idx_%'",
	).Scan(&indexCount)
	if err != nil {
		sd.addWarning("failed to count indexes: " + err.Error())
	}
	sd.result.IndexesCreated = indexCount

	// Verify integrity again
	var integrity string
	err = sd.db.QueryRow("PRAGMA integrity_check;").Scan(&integrity)
	if err != nil || integrity != "ok" {
		sd.addError("database integrity check failed after deployment")
		return fmt.Errorf("integrity check failed: %s", integrity)
	}

	sd.result.VerificationPassed = true
	sd.logger.Printf("✅ Verification passed: %d tables, %d indexes\n", tableCount, indexCount)

	return nil
}

// addError adds an error to the result
func (sd *SchemaDeployer) addError(msg string) {
	sd.result.Errors = append(sd.result.Errors, msg)
	sd.logger.Printf("❌ ERROR: %s\n", msg)
}

// addWarning adds a warning to the result
func (sd *SchemaDeployer) addWarning(msg string) {
	sd.result.Warnings = append(sd.result.Warnings, msg)
	sd.logger.Printf("⚠️  WARNING: %s\n", msg)
}

// PrintResult prints the deployment result
func (sd *SchemaDeployer) PrintResult() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("DEPLOYMENT RESULT")
	fmt.Println(strings.Repeat("=", 60))

	if sd.result.Success {
		fmt.Println("✅ DEPLOYMENT SUCCESSFUL")
	} else {
		fmt.Println("❌ DEPLOYMENT FAILED")
	}

	fmt.Printf("Execution Time: %v\n", sd.result.ExecutionTime)
	fmt.Printf("Tables Created: %d\n", sd.result.TablesCreated)
	fmt.Printf("Indexes Created: %d\n", sd.result.IndexesCreated)

	if sd.result.BackupPath != "" {
		fmt.Printf("Backup Path: %s\n", sd.result.BackupPath)
	}

	if len(sd.result.Warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, w := range sd.result.Warnings {
			fmt.Printf("  - %s\n", w)
		}
	}

	if len(sd.result.Errors) > 0 {
		fmt.Println("\nErrors:")
		for _, e := range sd.result.Errors {
			fmt.Printf("  - %s\n", e)
		}
	}

	if sd.result.VerificationPassed {
		fmt.Println("\n✅ Verification: PASSED")
	} else if sd.result.Verify {
		fmt.Println("\n⚠️  Verification: SKIPPED or FAILED")
	}

	fmt.Println(strings.Repeat("=", 60) + "\n")
}

// Main entry point for standalone execution
func main() {
	config := DeploymentConfig{}

	flag.StringVar(&config.DatabasePath, "db", "moly.db", "Path to database file")
	flag.BoolVar(&config.BackupBefore, "backup", true, "Backup database before deployment")
	flag.BoolVar(&config.Verify, "verify", true, "Verify deployment after execution")
	flag.StringVar(&config.Environment, "env", "development", "Deployment environment")
	flag.BoolVar(&config.DryRun, "dry-run", false, "Dry run (don't modify database)")
	flag.BoolVar(&config.Verbose, "v", false, "Verbose output")

	flag.Parse()

	deployer := NewSchemaDeployer(config)

	err := deployer.Deploy()
	deployer.PrintResult()

	if err != nil {
		os.Exit(1)
	}
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
