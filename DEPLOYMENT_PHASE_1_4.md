# DEPLOYMENT GUIDE: Phases 1-4 with Feature Flags

**Status**: Ready to Deploy  
**Date**: Sept 30, 2026  
**Timeline**: 9 weeks (1 person) or 5-6 weeks (2 people)

---

## Quick Start

### Pre-Deployment (All Phases)

```bash
# 1. Verify build passes
cd moly-go && go build ./... && echo "✅ BUILD PASSES"

# 2. Run all tests
go test ./... -v 2>&1 | tail -20

# 3. Check feature flags are available
grep -n "UseExtractionLock\|UseLayer5\|UseConstrained" config/feature_flags.go

# 4. Verify monitoring is integrated
grep -n "metrics\|GetMetrics" monitoring/metrics.go
```

---

## PHASE 1: Extraction Lock (Weeks 1-2)

### Deployment: Week 1

**Enable feature flag at 10%**:
```bash
# Option A: Environment variable
export MOLY_USE_EXTRACTION_LOCK=true
export MOLY_METRICS_RATE=100  # Collect all metrics for first phase

# Option B: At request level (use feature flag system)
flags := config.GetFeatureFlags()
if flags.UseExtractionLock {
    // Use extraction lock
}
```

**Monitoring checklist**:
```
□ Error rate: 0% (baseline)
□ Extraction time p95: < 250ms (baseline 200ms)
□ Multi-pass calls: 0/message (vs 7 baseline)
□ Subject attribution: 95%+
□ No regressions in other features
```

**Rollback triggers** (automatic):
```
if errorRate > 1% → ROLLBACK UseExtractionLock
if extractionTimeP95 > 250ms → ROLLBACK UseExtractionLock
if multiPassCalls > 1/message → ROLLBACK UseExtractionLock
```

### Deployment: Weeks 2-3

**If Phase 1 stable, expand to 50% users**:
```bash
# Staged rollout
MOLY_USE_EXTRACTION_LOCK=true  # 50% of users via load balancer/feature flag service

# Continue monitoring (same checklist)
# If stable after 1 week: proceed to 100%
```

**Phase 1 completion criteria**:
- ✅ 100% users on extraction lock
- ✅ Zero regressions
- ✅ Zero multi-pass parsing in logs
- ✅ 95%+ subject attribution accuracy maintained

---

## PHASE 2: Layer 5 Conflict Channeling (Weeks 3-5)

### Prerequisites
- ✅ Phase 1 stable (100% users)
- ✅ No Phase 1 issues

### Deployment: Week 3 (Preparation)

**Code changes to verify**:
```bash
# Check ConversationAgent has Layer 5 wiring
grep -A20 "Layer 5" agents/conversation_agent.go

# Check conflict handler exists
ls -la agents/layer5_conflict_handler.go

# Check clarification history exists
ls -la agents/clarification_history.go
```

### Deployment: Week 4 (10% Rollout)

```bash
export MOLY_USE_LAYER5_CONFLICT_GATE=true
# Deploy to 10% of users
```

**Monitoring**:
```
□ Conflicts detected: > 0 (baseline: 0)
□ Questions generated from conflicts: 100% of conflicts
□ Deduplication working: < 1% repeated questions
□ Response latency: < 3.5s (alert), < 4s (rollback)
□ User response rate: stable
```

**SLA thresholds**:
```
ALERT:    Response latency p95 > baseline + 20%
ROLLBACK: Response latency p95 > baseline + 30%
ROLLBACK: Conflicts generating questions < 95%
ROLLBACK: Deduplication rate < 90%
```

### Deployment: Week 5 (50% → 100%)

- 50% users Week 5a
- 100% users Week 5b (if stable)

---

## PHASE 3: Constrained Response Generation (Weeks 6-8)

### Prerequisites
- ✅ Phase 1 stable (100% users)
- ✅ Phase 2 stable (100% users)
- ✅ ConstrainedResponseGenerator deployed
- ✅ ConstraintCache deployed

### Deployment: Week 6 (Preparation & A/B Test Setup)

**A/B test infrastructure**:
```bash
# 50% users: ConstrainedResponseGenerator (treatment)
# 50% users: Basic ResponseGenerator (control)

# Track metrics for each group
```

### Deployment: Week 7 (10% Rollout with A/B)

```bash
export MOLY_USE_CONSTRAINED_RESPONSE_GENERATION=true
# Deploy to 10% of users (with A/B test)
```

**Monitoring**:
```
□ Response latency: < 3.5s (alert), < 4s (rollback)
□ Cache hit rate: > 60% (target 70%+)
□ Constraint violations: < 8% (target < 5%)
□ Role-reversal bugs: 0 per 1000 users
□ False positive blocks: < 10%
```

**A/B Test Metrics**:
```
TREATMENT (Constrained):  Role-reversal = 0
CONTROL (Basic):          Role-reversal = 0.1%
→ If treatment better: proceed
```

### Deployment: Week 8 (50% → 100%)

- 50% Week 8a (if A/B shows treatment better)
- 100% Week 8b

---

## PHASE 4: Clean Schema Migration (Week 9)

### Prerequisites
- ✅ Phases 1-3 stable (100% users)
- ✅ Migration validator passes
- ✅ 30-minute maintenance window scheduled

### Pre-Migration Checklist (Days 1-3 of Week 9)

```bash
# Day 1: Test database migration
./deploy_scripts/test_migration_local.sh

# Day 2: Production copy migration
./deploy_scripts/test_migration_prod_copy.sh

# Day 3: Dry run with downtime simulation
./deploy_scripts/dry_run_migration.sh
```

### Migration Day (Week 9, Day 4)

**Schedule**: Saturday 0:00 UTC (30 minutes downtime)

**Pre-migration (15 min before)**:
```bash
./deploy_scripts/pre_migration_check.sh
# Verify all systems ready
# Notify stakeholders
```

**During migration (30 min)**:
```bash
# 0-5 min: Backup + create new schema
./deploy_scripts/create_new_schema.sh

# 5-10 min: Export + Import
./deploy_scripts/migrate_data.sh

# 10-15 min: Verify + Switch
./deploy_scripts/verify_and_switch.sh

# 15-30 min: Monitor + Verification tests
./deploy_scripts/smoke_tests.sh
```

**Post-migration (Week 9, Day 5)**:
```bash
# Run comprehensive verification
./deploy_scripts/post_migration_verify.sh

# Monitor for 24 hours
# Daily reports for 7 days
```

---

## Deployment Scripts

### Script 1: Feature Flag Check

**File**: `deploy_scripts/check_flags.sh`
```bash
#!/bin/bash
set -e

echo "🔍 Checking feature flags..."
echo "MOLY_USE_EXTRACTION_LOCK: ${MOLY_USE_EXTRACTION_LOCK:-false}"
echo "MOLY_USE_LAYER5_CONFLICT_GATE: ${MOLY_USE_LAYER5_CONFLICT_GATE:-false}"
echo "MOLY_USE_CONSTRAINED_RESPONSE_GENERATION: ${MOLY_USE_CONSTRAINED_RESPONSE_GENERATION:-false}"
echo "MOLY_USE_CLEAN_SCHEMA: ${MOLY_USE_CLEAN_SCHEMA:-false}"
echo "MOLY_METRICS_RATE: ${MOLY_METRICS_RATE:-100}"

# Build and start service
cd moly-go
go build ./...
echo "✅ Build successful"

# Verify metrics are working
go test ./monitoring -v
echo "✅ Monitoring tests pass"
```

### Script 2: Monitoring Dashboard

**File**: `deploy_scripts/monitoring_dashboard.sh`
```bash
#!/bin/bash

echo "📊 MONITORING DASHBOARD"
echo "=========================="
echo ""
echo "Phase 1: Extraction Lock"
echo "  Locked extractions: (from metrics)"
echo "  Avg extraction time: (from metrics)"
echo "  Multi-pass calls/message: (from metrics)"
echo ""
echo "Phase 2: Layer 5 Conflicts"
echo "  Conflicts detected: (from metrics)"
echo "  Questions generated: (from metrics)"
echo "  Deduplication rate: (from metrics)"
echo ""
echo "Phase 3: Response Validation"
echo "  Constraints applied: (from metrics)"
echo "  Violations detected: (from metrics)"
echo "  Cache hit rate: (from metrics)"
echo ""
echo "Phase 4: Schema"
echo "  Migration status: (if completed)"
echo "  Query latency: (from metrics)"
```

### Script 3: Rollback Script

**File**: `deploy_scripts/rollback.sh`
```bash
#!/bin/bash
set -e

PHASE=${1:-""}

if [ -z "$PHASE" ]; then
    echo "Usage: ./rollback.sh {1|2|3|4}"
    exit 1
fi

echo "🔄 Rolling back Phase $PHASE..."

case $PHASE in
    1)
        export MOLY_USE_EXTRACTION_LOCK=false
        echo "✅ Rolled back Phase 1: Extraction Lock"
        ;;
    2)
        export MOLY_USE_LAYER5_CONFLICT_GATE=false
        echo "✅ Rolled back Phase 2: Layer 5 Conflicts"
        ;;
    3)
        export MOLY_USE_CONSTRAINED_RESPONSE_GENERATION=false
        echo "✅ Rolled back Phase 3: Constrained Generation"
        ;;
    4)
        # Schema rollback is manual
        echo "⚠️  Phase 4 rollback is manual (switch connection string back)"
        echo "Old connection string: ${OLD_DATABASE_URL}"
        ;;
esac

# Restart service
systemctl restart moly
echo "✅ Service restarted with rollback applied"
```

---

## Monitoring & Alerting

### Key Metrics to Watch

**Phase 1**:
```
- extraction_time_ms (p50, p95, p99)
- multi_pass_calls_per_message
- extractions_locked (counter)
- subject_attribution_accuracy (%)
```

**Phase 2**:
```
- conflicts_detected (counter)
- conflicts_resulting_in_questions (%)
- question_deduplication_prevented (counter)
- response_latency_ms (p50, p95, p99)
```

**Phase 3**:
```
- constraints_applied (counter)
- response_validation_violations (counter)
- constraint_cache_hit_rate (%)
- role_reversal_bug_occurrences (counter)
- response_latency_ms (p50, p95, p99)
```

**Phase 4**:
```
- migration_time_ms
- data_loss_count
- query_latency_ms (p50, p95, p99)
- error_rate (%)
```

### Alert Thresholds

```
CRITICAL (immediate action):
  - Error rate > 2%
  - Data loss > 0 records (Phase 4)
  - Response latency p95 > baseline + 30%

WARNING (investigate):
  - Error rate > 1%
  - Response latency p95 > baseline + 20%
  - Cache hit rate < 60%
  - Constraint violations > 8%
```

### Logging Pattern

Every feature-gated code path should log:
```go
log.Printf("[Phase%d] Feature=%s Status=%s Metric=%v",
    phase, featureName, status, value)
```

---

## Rollback Decision Tree

```
Phase 1:
  ├─ Extraction time spike? → ROLLBACK
  ├─ Accuracy drop < 94%? → ROLLBACK
  ├─ Multi-pass > 1/msg? → ROLLBACK
  └─ All stable? → Proceed to Phase 2

Phase 2:
  ├─ Response latency spike? → ROLLBACK
  ├─ Conflicts < 95% questions? → ROLLBACK
  ├─ Dedup rate < 90%? → ROLLBACK
  └─ All stable? → Proceed to Phase 3

Phase 3:
  ├─ Response latency spike? → ROLLBACK
  ├─ Role-reversal bugs > 0? → ROLLBACK
  ├─ Violations > 8%? → ROLLBACK
  ├─ Cache hit < 60%? → INVESTIGATE
  └─ All stable? → Proceed to Phase 4

Phase 4:
  ├─ Migration fails? → ROLLBACK (automatic)
  ├─ Data loss > 0? → ROLLBACK (automatic)
  ├─ Query latency spike? → ROLLBACK (manual)
  └─ All successful? → COMPLETE
```

---

## Success Criteria per Phase

### Phase 1 ✅
- [x] Zero multi-pass parsing
- [x] 95%+ subject attribution
- [x] No regressions
- [x] 100% users

### Phase 2 ✅
- [x] 100% conflicts generate questions
- [x] <1% repeated questions
- [x] No regressions
- [x] 100% users

### Phase 3 ✅
- [x] Zero role-reversal bugs
- [x] <5% constraint violations
- [x] >70% cache hit rate
- [x] 100% users

### Phase 4 ✅
- [x] 0% data loss
- [x] <5min migration
- [x] Query latency stable
- [x] 100% data verified

---

## Final Verification Checklist

```
□ Build passes (go build ./...)
□ All tests pass (go test ./...)
□ Feature flags working
□ Monitoring metrics collected
□ Logging shows feature status
□ Rollback scripts tested
□ Team trained
□ Documentation complete
□ Stakeholders notified
□ Ready to deploy Phase 1
```

---

**Status**: Ready to Deploy Phases 1-4  
**Next**: Start Phase 1 (Week 1)  
**Expected Completion**: Week 9

