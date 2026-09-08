# Phase 1.2 Schema Deployment Guide

**Status**: Ready for Deployment  
**Date**: September 8, 2026  
**Environments**: Development, Staging, Production

---

## Quick Start (5 minutes)

### For Development (SQLite - Local)

```bash
cd Moly/moly-go

# Method 1: Using Go (Recommended - includes verification)
go run ./database/schema_deployer.go

# Method 2: Using SQLite CLI directly
sqlite3 moly.db < database/deploy_schema_v2_1_phase_1_2.sql

# Verify deployment
sqlite3 moly.db "SELECT COUNT(*) as tables FROM sqlite_master WHERE type='table' AND name LIKE '%me_%' OR name LIKE '%contact%' OR name LIKE '%pattern%';"
# Should return: 9 (tables created)
```

**Expected Output:**
```
✅ Schema deployment script executed
✅ 9 permanent + ephemeral tables created
✅ 20+ indexes created
✅ Database ready for Phase 1.2
```

---

## Pre-Deployment Checklist

- [ ] **Backup existing database**: `cp moly.db moly.db.backup.$(date +%s)`
- [ ] **Verify database connectivity**: `sqlite3 moly.db "PRAGMA integrity_check;"`
- [ ] **Check disk space**: Need at least 100MB free
- [ ] **Stop backend services**: `pkill -f "go run main.go"`
- [ ] **Read deployment script**: Review `deploy_schema_v2_1_phase_1_2.sql`
- [ ] **Test on copy first**: Deploy to backup first, verify, then deploy to live

---

## Deployment Methods

### Method 1: Go Deployer (Recommended - Automated)

**Advantages**: 
- Automatic verification
- Better error messages
- Rollback capability
- Logging

**Steps**:
```bash
cd moly-go
go run ./database/schema_deployer.go --env production --verify
```

### Method 2: SQLite CLI

**Advantages**:
- Direct control
- No Go dependencies
- Transparent (can see exact SQL)

**Steps**:
```bash
cd moly-go
sqlite3 moly.db < database/deploy_schema_v2_1_phase_1_2.sql
```

### Method 3: Manual (Learning/Debug)

```bash
sqlite3 moly.db

-- Then paste the SQL from deploy_schema_v2_1_phase_1_2.sql
-- Run line by line or all at once
```

---

## Detailed Deployment Steps

### Step 1: Backup Current Database

```bash
# Create timestamped backup
BACKUP_FILE="moly.db.backup.$(date +%Y%m%d_%H%M%S)"
cp moly.db $BACKUP_FILE
echo "Backup created: $BACKUP_FILE"

# Keep last 3 backups
ls -t moly.db.backup.* | tail -n +4 | xargs rm -f
```

### Step 2: Verify Database Health

```bash
sqlite3 moly.db << 'EOF'
-- Check integrity
PRAGMA integrity_check;

-- Count existing tables
SELECT COUNT(*) as existing_tables FROM sqlite_master WHERE type='table';

-- List all tables
SELECT name FROM sqlite_master WHERE type='table' ORDER BY name;
EOF
```

Expected output:
```
ok
(existing_tables)
(various phase 1.1 tables)
```

### Step 3: Deploy Schema

```bash
sqlite3 moly.db < database/deploy_schema_v2_1_phase_1_2.sql
```

Or with error logging:

```bash
sqlite3 moly.db < database/deploy_schema_v2_1_phase_1_2.sql 2>&1 | tee deployment.log
```

### Step 4: Verify Deployment

```bash
sqlite3 moly.db << 'EOF'
-- Verify all Phase 1.2 tables exist
SELECT name FROM sqlite_master 
WHERE type='table' 
AND name IN ('about_me_profile', 'contacts', 'contact_communication_patterns',
             'communication_patterns', 'communication_goals', 'reflection_journal',
             'implicit_learning', 'conversation_ephemeral', 'extraction_queue')
ORDER BY name;

-- Verify indexes created
SELECT COUNT(*) as index_count FROM sqlite_master 
WHERE type='index' AND name LIKE 'idx_%';

-- Verify foreign keys work
PRAGMA foreign_keys = ON;
PRAGMA foreign_key_list(contacts);
EOF
```

**Expected Output:**
```
about_me_profile
contact_communication_patterns
contacts
communication_goals
communication_patterns
conversation_ephemeral
extraction_queue
implicit_learning
reflection_journal

index_count: 21

(foreign key info showing user_id → users.id)
```

### Step 5: Verify Data Consistency

```bash
sqlite3 moly.db << 'EOF'
-- Check no data loss in Phase 1.1 tables (if they exist)
SELECT COUNT(*) as user_count FROM users;
SELECT COUNT(*) as session_count FROM sessions;

-- Check Phase 1.2 tables are empty (new deployment)
SELECT COUNT(*) as aboutme_count FROM about_me_profile;
SELECT COUNT(*) as contacts_count FROM contacts;
SELECT COUNT(*) as patterns_count FROM communication_patterns;

-- All should be 0 for new deployment
EOF
```

### Step 6: Test with Backend

```bash
# Start backend
cd moly-go
go run main.go

# In another terminal, test API
curl -X GET http://localhost:8080/api/v2.1/profile \
  -H "X-User-ID: test_user" \
  -H "Content-Type: application/json"

# Should return 200 OK with empty profile
# or error if backend not fully configured
```

---

## Deployment Verification Checklist

After deployment, verify:

- [ ] **Tables exist**: `SELECT COUNT(*) FROM sqlite_master WHERE type='table';` = 11+ (including Phase 1.1 tables)
- [ ] **Indexes exist**: `SELECT COUNT(*) FROM sqlite_master WHERE type='index';` = 21+
- [ ] **Constraints work**: Foreign keys not violated
- [ ] **Data intact**: No Phase 1.1 data lost
- [ ] **Tables empty**: Phase 1.2 tables have 0 rows (expected for new deployment)
- [ ] **Backend starts**: `go run main.go` starts without database errors
- [ ] **API responds**: Profile endpoint returns valid JSON
- [ ] **No schema errors**: No SQL errors in deployment log

---

## Rollback Procedure

If something goes wrong:

### Option 1: Restore from Backup (Complete)

```bash
# Stop backend
pkill -f "go run main.go"

# Restore from backup
BACKUP_FILE="moly.db.backup.20260908_120000"  # Use actual backup filename
rm moly.db
cp $BACKUP_FILE moly.db

# Restart backend
go run main.go
```

### Option 2: Drop Phase 1.2 Tables Only

```bash
sqlite3 moly.db << 'EOF'
-- Drop Phase 1.2 tables (keeps Phase 1.1 intact)
DROP TABLE IF EXISTS implicit_learning;
DROP TABLE IF EXISTS extraction_queue;
DROP TABLE IF EXISTS conversation_ephemeral;
DROP TABLE IF EXISTS reflection_journal;
DROP TABLE IF EXISTS communication_goals;
DROP TABLE IF EXISTS communication_patterns;
DROP TABLE IF EXISTS contact_communication_patterns;
DROP TABLE IF EXISTS contacts;
DROP TABLE IF EXISTS about_me_profile;

-- Indexes automatically dropped with tables
EOF
```

**Note**: Phase 1.2 cannot coexist with Phase 1.1 (different data model). Must use complete backup restore if rollback needed.

---

## Production Deployment

### Pre-Production Testing

1. **Test on Development Database First**
   ```bash
   # Make copy of production DB
   cp /prod/moly.db /dev/moly_prod_test.db
   
   # Test deployment on copy
   sqlite3 /dev/moly_prod_test.db < database/deploy_schema_v2_1_phase_1_2.sql
   
   # Verify
   sqlite3 /dev/moly_prod_test.db "SELECT COUNT(*) FROM sqlite_master WHERE type='table';"
   ```

2. **Dry Run with Backup**
   ```bash
   # Backup production
   cp /prod/moly.db /backup/moly.db.pre_phase_1_2.$(date +%s)
   
   # Deploy to backup
   sqlite3 /backup/moly.db.pre_phase_1_2.* < database/deploy_schema_v2_1_phase_1_2.sql
   
   # Verify backup deployment
   sqlite3 /backup/moly.db.pre_phase_1_2.* "SELECT COUNT(*) FROM about_me_profile;"
   ```

3. **Deploy to Production**
   ```bash
   # Maintenance window (tell users backend is updating)
   
   # Stop services
   sudo systemctl stop moly-backend
   
   # Backup
   sudo cp /prod/moly.db /backup/moly.db.$(date +%s)
   
   # Deploy
   sudo sqlite3 /prod/moly.db < database/deploy_schema_v2_1_phase_1_2.sql
   
   # Verify
   sudo sqlite3 /prod/moly.db "PRAGMA integrity_check;"
   
   # Start services
   sudo systemctl start moly-backend
   
   # Monitor logs
   sudo journalctl -u moly-backend -f
   ```

---

## Troubleshooting

### "database is locked" Error

**Cause**: Backend or another process has database open

**Solution**:
```bash
# Stop backend
pkill -f "go run main.go"

# Wait 5 seconds
sleep 5

# Try again
sqlite3 moly.db < database/deploy_schema_v2_1_phase_1_2.sql
```

### "table already exists" Error (on some tables)

**Cause**: Partial previous deployment

**Solution**:
```bash
# This is safe - "IF NOT EXISTS" prevents errors
# The script will skip existing tables and create missing ones
sqlite3 moly.db < database/deploy_schema_v2_1_phase_1_2.sql

# Verify what's missing
sqlite3 moly.db ".tables"
```

### "FOREIGN KEY constraint failed" Error

**Cause**: Foreign key references missing

**Solution**:
```bash
# Check if users table exists (required by all Phase 1.2 tables)
sqlite3 moly.db "SELECT COUNT(*) FROM users;"

# If users table doesn't exist, need to create it (from Phase 1)
# This shouldn't happen if upgrading from Phase 1.1
```

### "PRAGMA integrity_check" Fails

**Cause**: Database corruption

**Solution**:
```bash
# Restore from backup
rm moly.db
cp moly.db.backup.XXXXXXX moly.db

# Don't attempt to fix corrupted database
# Restore and redeploy instead
```

---

## Performance Optimization After Deployment

### 1. Vacuum Database (Optimize)

```bash
sqlite3 moly.db << 'EOF'
-- Optimize database (removes fragmentation)
VACUUM;

-- Analyze table statistics (helps query planner)
ANALYZE;
EOF
```

### 2. Enable Query Logging (Development Only)

```bash
sqlite3 moly.db << 'EOF'
-- Log slow queries (>100ms)
.timer on
.eqp full
EOF
```

### 3. Verify Index Usage

```bash
sqlite3 moly.db << 'EOF'
-- Check which indexes are used
PRAGMA index_list(contacts);
PRAGMA index_list(communication_patterns);
PRAGMA index_list(reflection_journal);
EOF
```

---

## Database Maintenance

### Weekly

```bash
sqlite3 moly.db << 'EOF'
-- Update statistics
ANALYZE;

-- Vacuum
VACUUM;
EOF
```

### Monthly

```bash
# Create backup
cp moly.db moly.db.monthly.$(date +%Y%m%d)

# Run integrity check
sqlite3 moly.db "PRAGMA integrity_check;" > integrity_check.log

# Archive old backups
tar -czf backups/moly_backups_$(date +%Y%m).tar.gz moly.db.daily.*
```

---

## Monitoring

### Check Database Size

```bash
# Current size
du -h moly.db

# Size limit warning (if > 500MB)
SIZE=$(du -b moly.db | cut -f1)
if [ $SIZE -gt 524288000 ]; then
  echo "WARNING: Database exceeds 500MB"
  echo "Consider archiving old conversations from conversation_ephemeral"
fi
```

### Cleanup Old Ephemeral Data

```bash
sqlite3 moly.db << 'EOF'
-- Delete conversations expired > 24h ago
DELETE FROM conversation_ephemeral 
WHERE expires_at < datetime('now', '-2 days');

-- Delete failed extraction attempts > 7 days old
DELETE FROM extraction_queue 
WHERE status = 'failed' AND attempted_at < datetime('now', '-7 days');
EOF
```

---

## Summary

| Step | Time | Verification |
|------|------|--------------|
| Backup | 1 min | `ls -la moly.db.backup.*` |
| Health Check | 1 min | `PRAGMA integrity_check;` |
| Deploy | 1 min | Execution completes |
| Verify Tables | 1 min | 9 Phase 1.2 tables exist |
| Verify Indexes | 1 min | 21 indexes exist |
| Verify Integrity | 1 min | No corruption |
| Test Backend | 2 min | API responds |
| **Total** | **~8 minutes** | **Full system ready** |

---

## Support

- **Questions**: See PHASE_1_2_COMPLETE.md for architecture details
- **Issues**: Check troubleshooting section above
- **Rollback**: Use backup restore procedure
- **Performance**: Run ANALYZE and VACUUM monthly

---

**Created**: September 8, 2026  
**Status**: Ready for Production  
**Next Step**: Execute deployment script
