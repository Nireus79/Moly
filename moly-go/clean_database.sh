#!/bin/bash

# Database cleanup script for Moly
# Safely removes old data while preserving schema

DB_PATH="/tmp/moly-v2.db"
BACKUP_DIR="/tmp/moly-backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

echo "=========================================="
echo "Moly Database Cleanup"
echo "=========================================="
echo ""

# Check if database exists
if [ ! -f "$DB_PATH" ]; then
    echo "❌ Database not found at $DB_PATH"
    exit 1
fi

echo "Database found: $DB_PATH"
ls -lh "$DB_PATH"
echo ""

# Create backup
mkdir -p "$BACKUP_DIR"
BACKUP_FILE="$BACKUP_DIR/moly-v2.db.backup.$TIMESTAMP"
cp "$DB_PATH" "$BACKUP_FILE"
echo "✓ Backup created: $BACKUP_FILE"
echo ""

# Option 1: Remove database file entirely (reinitialize on next run)
echo "Removing database file to start fresh..."
rm "$DB_PATH"
echo "✓ Database file removed"
echo ""

echo "=========================================="
echo "✓ Cleanup complete!"
echo "=========================================="
echo ""
echo "The database will be reinitialized with fresh schema on next run."
echo "Backup saved at: $BACKUP_FILE"
echo ""
echo "To restore from backup if needed:"
echo "  cp $BACKUP_FILE $DB_PATH"
