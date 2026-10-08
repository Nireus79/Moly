# MOLY Production Deployment Guide

**Date:** October 8, 2026  
**Status:** ✅ PRODUCTION READY
**Version:** 1.0 (All FIX #1-11 Deployed)

---

## Pre-Deployment Checklist

### Code Readiness
- [x] All FIX #1-11 deployed and tested
- [x] Security hardening complete (FIX #5 & #6)
- [x] Dead code removed (Phase 3-5)
- [x] Schema optimized (7 tables removed)
- [x] Build clean and verified: 22MB
- [x] No new compilation errors

### Database Readiness
- [x] Migrations 041-046 created
- [x] Schema consolidated (29 → 22 tables)
- [x] Foreign keys updated
- [x] Indexes optimized

### Testing
- [x] Build verification: ✅ PASS
- [x] Core functionality: ✅ PASS
- [x] Security layer tests: ✅ PASS
- [x] Pre-existing test failures documented (not blocking)

---

## Deployment Steps

### Step 1: Database Migration (CRITICAL)

```bash
# Run migrations in order
sqlite3 moly.db < database/migrations/041_create_conversation_summaries_table.sql
sqlite3 moly.db < database/migrations/042_add_accumulated_insights_to_summary.sql
sqlite3 moly.db < database/migrations/043_create_conversation_execution_state_table.sql
sqlite3 moly.db < database/migrations/044_consolidate_clarification_questions_schema.sql
sqlite3 moly.db < database/migrations/045_drop_orphaned_tables.sql
sqlite3 moly.db < database/migrations/046_deep_schema_consolidation.sql

# Verify schema
sqlite3 moly.db ".tables"  # Should show 22 core tables
```

### Step 2: Binary Deployment

```bash
# Build production binary
cd moly-go
go build -o ../bin/moly .

# Deploy
cp ../bin/moly /path/to/production/moly

# Verify
./moly -version  # Should show git commit hash
```

### Step 3: Service Start

```bash
# Start Moly service
systemctl start moly

# Verify running
curl http://localhost:8080/health

# Check logs
journalctl -u moly -f
```

---

## Validation After Deployment

### Health Checks (First 5 minutes)

```bash
# API is responding
curl http://localhost:8080/health

# Database connected
curl http://localhost:8080/users/test  # Should require auth

# Migrations applied
sqlite3 moly.db "SELECT COUNT(*) FROM conversations;"
```

### Functional Tests (First hour)

1. **User Registration Flow**
   - Create new user
   - Verify data isolation
   - Check security gates

2. **Message Processing**
   - Send message
   - Verify 11-layer orchestrator runs
   - Check all layers complete

3. **Data Isolation**
   - Try to access other user's data (should fail)
   - Verify error messages correct
   - Check logs for attempts

4. **Error Handling**
   - Trigger error condition
   - Verify logged properly
   - Check no silent failures

---

## Monitoring Dashboard Setup

### Key Metrics to Watch (24 hours)

```
1. Error Rate
   - Should be < 0.1% for known issues
   - 0% for new errors = good sign

2. Latency
   - Message processing: < 2 seconds (including LLM)
   - API responses: < 100ms
   - Database queries: < 50ms

3. Data Integrity
   - No orphaned records
   - User isolation: 100%
   - RowsAffected validation: passing

4. Security
   - No unauthorized access attempts logged
   - All userID checks passing
   - Error messages not leaking data
```

### Alert Thresholds

```
CRITICAL:
- Error rate > 5%
- Latency > 10 seconds
- Data isolation failure

WARNING:
- Error rate > 1%
- Latency > 5 seconds
- Any unauthorized access attempt
```

---

## Rollback Procedure

If critical issues arise:

```bash
# Stop service
systemctl stop moly

# Rollback binary (if code issue)
cp /backup/moly-previous /path/to/production/moly
systemctl start moly

# Rollback database (if schema issue)
# Note: Migrations 045-046 can be rolled back if needed
# Restore backup: cp moly.db.backup moly.db
```

---

## Post-Deployment Tasks

### Day 1
- [ ] Monitor error logs
- [ ] Verify no data corruption
- [ ] Test all main flows
- [ ] Confirm users can register/login

### Week 1
- [ ] Monitor latency trends
- [ ] Check data isolation enforcement
- [ ] Verify security gates working
- [ ] Collect baseline metrics

### Week 2
- [ ] Review technical debt log (PHASE_6_TECHNICAL_DEBT.md)
- [ ] Schedule Phase 7 cleanup if needed
- [ ] Performance optimization pass
- [ ] Documentation updates

---

## Contacts & Support

**On-Call Engineer:**
- Check oncall schedule
- Primary: [Name] - [Contact]
- Secondary: [Name] - [Contact]

**Escalation Path:**
1. On-call engineer
2. Engineering lead
3. CTO

**Critical Issues:**
- Data loss: Immediate rollback
- Security breach: Immediate lockdown
- Widespread outages: Page on-call

---

## Final Sign-Off

- [x] Code Review: APPROVED
- [x] Security Review: APPROVED
- [x] Database Review: APPROVED
- [x] QA Testing: APPROVED
- [x] Production Ready: ✅ YES

**System is cleared for production deployment.**

---

**Build:** 22MB Clean  
**Migrations:** 041-046 Ready  
**Security:** Hardened  
**Performance:** Optimized  

**STATUS: ✅ READY TO DEPLOY**
