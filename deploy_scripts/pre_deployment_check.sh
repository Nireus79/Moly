#!/bin/bash
set -e

echo "🔍 PRE-DEPLOYMENT VERIFICATION CHECKLIST"
echo "=========================================="
echo ""

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

ERRORS=0
WARNINGS=0

# Check 1: Build passes
echo "1️⃣  Checking build..."
if cd moly-go && go build ./... 2>&1 | grep -q "error"; then
    echo -e "${RED}✗ BUILD FAILED${NC}"
    ERRORS=$((ERRORS+1))
else
    echo -e "${GREEN}✓ Build passes${NC}"
fi

# Check 2: Tests pass
echo "2️⃣  Checking tests..."
if go test ./... -q 2>&1 | grep -q "FAIL"; then
    echo -e "${RED}✗ TESTS FAILED${NC}"
    ERRORS=$((ERRORS+1))
else
    echo -e "${GREEN}✓ All tests pass${NC}"
fi

# Check 3: Feature flags exist
echo "3️⃣  Checking feature flags..."
if grep -q "UseExtractionLock" config/feature_flags.go; then
    echo -e "${GREEN}✓ Feature flags configured${NC}"
else
    echo -e "${RED}✗ Feature flags missing${NC}"
    ERRORS=$((ERRORS+1))
fi

# Check 4: Monitoring integrated
echo "4️⃣  Checking monitoring..."
if grep -q "RecordExtractionTime" ../monitoring/metrics.go; then
    echo -e "${GREEN}✓ Monitoring system ready${NC}"
else
    echo -e "${RED}✗ Monitoring not integrated${NC}"
    WARNINGS=$((WARNINGS+1))
fi

# Check 5: Phase Orchestrator exists
echo "5️⃣  Checking Phase Orchestrator..."
if [ -f "agents/phase_orchestrator.go" ]; then
    echo -e "${GREEN}✓ Phase orchestrator present${NC}"
else
    echo -e "${RED}✗ Phase orchestrator missing${NC}"
    ERRORS=$((ERRORS+1))
fi

# Check 6: Database migrations
echo "6️⃣  Checking database migrations..."
if [ -f "database/migrations/030_create_clean_schema.sql" ] && [ -f "database/migrations/031_add_response_validation_tracking.sql" ]; then
    echo -e "${GREEN}✓ Migrations present${NC}"
else
    echo -e "${RED}✗ Migrations missing${NC}"
    WARNINGS=$((WARNINGS+1))
fi

# Check 7: Documentation
echo "7️⃣  Checking documentation..."
if [ -f "../IMPLEMENTATION_COMPLETE_STATUS.md" ] && [ -f "../DEPLOYMENT_PHASE_1_4.md" ]; then
    echo -e "${GREEN}✓ Documentation complete${NC}"
else
    echo -e "${RED}✗ Documentation incomplete${NC}"
    WARNINGS=$((WARNINGS+1))
fi

# Check 8: Environment variables
echo "8️⃣  Checking environment..."
echo "  MOLY_USE_EXTRACTION_LOCK: ${MOLY_USE_EXTRACTION_LOCK:-not set}"
echo "  MOLY_USE_LAYER5_CONFLICT_GATE: ${MOLY_USE_LAYER5_CONFLICT_GATE:-not set}"
echo "  MOLY_USE_CONSTRAINED_RESPONSE_GENERATION: ${MOLY_USE_CONSTRAINED_RESPONSE_GENERATION:-not set}"
echo "  MOLY_METRICS_RATE: ${MOLY_METRICS_RATE:-100}"

# Check 9: Dependencies
echo "9️⃣  Checking dependencies..."
if go mod verify 2>&1 | grep -q "error"; then
    echo -e "${RED}✗ Module dependencies broken${NC}"
    ERRORS=$((ERRORS+1))
else
    echo -e "${GREEN}✓ Dependencies OK${NC}"
fi

# Check 10: Git status
echo "🔟 Checking git status..."
if git status --short | grep -q "^??"; then
    echo -e "${YELLOW}⚠ Untracked files (should be committed)${NC}"
    WARNINGS=$((WARNINGS+1))
fi

if git status --short | grep -q "^ M"; then
    echo -e "${RED}✗ Uncommitted changes (must commit before deploy)${NC}"
    ERRORS=$((ERRORS+1))
fi

if [ $ERRORS -eq 0 ] && [ $WARNINGS -eq 0 ]; then
    echo -e "${GREEN}✓ All checks passed${NC}"
else
    echo ""
fi

echo ""
echo "SUMMARY:"
echo "--------"
echo "Errors:   $ERRORS"
echo "Warnings: $WARNINGS"
echo ""

if [ $ERRORS -gt 0 ]; then
    echo -e "${RED}❌ DEPLOYMENT BLOCKED - Fix errors above${NC}"
    exit 1
elif [ $WARNINGS -gt 0 ]; then
    echo -e "${YELLOW}⚠️  DEPLOYMENT READY - Address warnings${NC}"
    exit 0
else
    echo -e "${GREEN}✅ READY FOR DEPLOYMENT${NC}"
    exit 0
fi
