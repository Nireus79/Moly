#!/bin/bash

# ============================================================================
# Phase 1.2 Schema Deployment Script
# Quick deployment for development, staging, and production environments
# ============================================================================

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
DB_PATH="${1:-moly.db}"
ENVIRONMENT="${2:-development}"
BACKUP_DIR="./backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Phase 1.2 Schema Deployment${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "Database: ${YELLOW}${DB_PATH}${NC}"
echo -e "Environment: ${YELLOW}${ENVIRONMENT}${NC}"
echo -e "Timestamp: ${YELLOW}${TIMESTAMP}${NC}"
echo ""

# ============================================================================
# Pre-flight checks
# ============================================================================

echo -e "${BLUE}[1/6]${NC} Running pre-flight checks..."

if [ ! -f "$DB_PATH" ]; then
    echo -e "${RED}❌ Database not found: ${DB_PATH}${NC}"
    exit 1
fi

if ! command -v sqlite3 &> /dev/null; then
    echo -e "${RED}❌ sqlite3 not installed${NC}"
    exit 1
fi

if [ ! -f "moly-go/database/deploy_schema_v2_1_phase_1_2.sql" ]; then
    echo -e "${RED}❌ Schema file not found: moly-go/database/deploy_schema_v2_1_phase_1_2.sql${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Pre-flight checks passed${NC}"
echo ""

# ============================================================================
# Backup database
# ============================================================================

echo -e "${BLUE}[2/6]${NC} Creating database backup..."

mkdir -p "$BACKUP_DIR"
BACKUP_FILE="${BACKUP_DIR}/${DB_PATH%.*}.backup.${TIMESTAMP}.db"

cp "$DB_PATH" "$BACKUP_FILE"
echo -e "${GREEN}✅ Backup created: ${BACKUP_FILE}${NC}"
echo ""

# ============================================================================
# Verify database health
# ============================================================================

echo -e "${BLUE}[3/6]${NC} Verifying database health..."

INTEGRITY=$(sqlite3 "$DB_PATH" "PRAGMA integrity_check;")
if [ "$INTEGRITY" != "ok" ]; then
    echo -e "${RED}❌ Database corruption detected: ${INTEGRITY}${NC}"
    echo -e "${YELLOW}Restoring from backup...${NC}"
    cp "$BACKUP_FILE" "$DB_PATH"
    exit 1
fi

echo -e "${GREEN}✅ Database integrity verified${NC}"
echo ""

# ============================================================================
# Deploy schema
# ============================================================================

echo -e "${BLUE}[4/6]${NC} Deploying Phase 1.2 schema..."

if sqlite3 "$DB_PATH" < moly-go/database/deploy_schema_v2_1_phase_1_2.sql 2> /tmp/deploy_error.log; then
    echo -e "${GREEN}✅ Schema deployment successful${NC}"
else
    echo -e "${RED}❌ Schema deployment failed${NC}"
    cat /tmp/deploy_error.log
    echo -e "${YELLOW}Rolling back...${NC}"
    cp "$BACKUP_FILE" "$DB_PATH"
    exit 1
fi
echo ""

# ============================================================================
# Verify deployment
# ============================================================================

echo -e "${BLUE}[5/6]${NC} Verifying deployment..."

TABLES_COUNT=$(sqlite3 "$DB_PATH" \
  "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN (
    'about_me_profile', 'contacts', 'contact_communication_patterns',
    'communication_patterns', 'communication_goals', 'reflection_journal',
    'implicit_learning', 'conversation_ephemeral', 'extraction_queue'
  );")

INDEXES_COUNT=$(sqlite3 "$DB_PATH" \
  "SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name LIKE 'idx_%';")

if [ "$TABLES_COUNT" -eq 9 ]; then
    echo -e "${GREEN}✅ All 9 Phase 1.2 tables created${NC}"
else
    echo -e "${RED}❌ Expected 9 tables, found ${TABLES_COUNT}${NC}"
    cp "$BACKUP_FILE" "$DB_PATH"
    exit 1
fi

echo -e "${GREEN}✅ Created ${INDEXES_COUNT} indexes${NC}"

# Final integrity check
FINAL_INTEGRITY=$(sqlite3 "$DB_PATH" "PRAGMA integrity_check;")
if [ "$FINAL_INTEGRITY" != "ok" ]; then
    echo -e "${RED}❌ Post-deployment integrity check failed${NC}"
    cp "$BACKUP_FILE" "$DB_PATH"
    exit 1
fi

echo -e "${GREEN}✅ Post-deployment integrity verified${NC}"
echo ""

# ============================================================================
# Summary
# ============================================================================

echo -e "${BLUE}[6/6]${NC} Deployment summary..."
echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}✅ DEPLOYMENT SUCCESSFUL${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "Database: ${YELLOW}${DB_PATH}${NC}"
echo -e "Backup: ${YELLOW}${BACKUP_FILE}${NC}"
echo -e "Tables: ${YELLOW}${TABLES_COUNT}/9${NC}"
echo -e "Indexes: ${YELLOW}${INDEXES_COUNT}${NC}"
echo -e "Environment: ${YELLOW}${ENVIRONMENT}${NC}"
echo ""

# ============================================================================
# Post-deployment instructions
# ============================================================================

echo -e "${BLUE}Next steps:${NC}"
echo "1. Verify database tables:"
echo "   sqlite3 $DB_PATH '.tables'"
echo ""
echo "2. Start backend:"
echo "   cd moly-go && go run main.go"
echo ""
echo "3. Test profile API:"
echo "   curl -X GET http://localhost:8080/api/v2.1/profile \\"
echo "     -H \"X-User-ID: test_user\""
echo ""
echo "4. To rollback (if needed):"
echo "   cp $BACKUP_FILE $DB_PATH"
echo ""
echo -e "${GREEN}Phase 1.2 schema deployment complete!${NC}"
