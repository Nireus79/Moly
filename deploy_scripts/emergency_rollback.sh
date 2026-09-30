#!/bin/bash
set -e

# EMERGENCY ROLLBACK SCRIPT
# Use when SLA thresholds are exceeded
# Disables the feature flag immediately and restarts service

PHASE=${1:-""}

if [ -z "$PHASE" ]; then
    echo "🚨 EMERGENCY ROLLBACK"
    echo "===================="
    echo ""
    echo "Usage: ./emergency_rollback.sh <1|2|3|4>"
    echo ""
    echo "This immediately:"
    echo "  1. Disables the specified phase"
    echo "  2. Restarts Moly service"
    echo "  3. Notifies on-call team"
    echo ""
    exit 1
fi

echo "🚨 EMERGENCY ROLLBACK INITIATED"
echo "==============================="
echo "Phase: $PHASE"
echo "Timestamp: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo ""

# Verify phase is valid
if [ "$PHASE" != "1" ] && [ "$PHASE" != "2" ] && [ "$PHASE" != "3" ] && [ "$PHASE" != "4" ]; then
    echo "❌ Phase must be 1, 2, 3, or 4"
    exit 1
fi

# Step 1: Disable the feature
echo "Step 1️⃣  Disabling Phase $PHASE..."
case $PHASE in
    1)
        export MOLY_USE_EXTRACTION_LOCK=false
        PHASE_NAME="Extraction Lock"
        ;;
    2)
        export MOLY_USE_LAYER5_CONFLICT_GATE=false
        PHASE_NAME="Layer 5 Conflict Gating"
        ;;
    3)
        export MOLY_USE_CONSTRAINED_RESPONSE_GENERATION=false
        PHASE_NAME="Constrained Response Generation"
        ;;
    4)
        export MOLY_USE_CLEAN_SCHEMA=false
        PHASE_NAME="Clean Schema"
        ;;
esac

echo "✓ Phase $PHASE ($PHASE_NAME) DISABLED"
echo ""

# Step 2: Restart service
echo "Step 2️⃣  Restarting Moly service..."
if systemctl is-active --quiet moly; then
    systemctl restart moly
    echo "✓ Moly service restarted"

    # Wait for service to come up
    sleep 3

    if systemctl is-active --quiet moly; then
        echo "✓ Service is running"
    else
        echo "❌ Service failed to start!"
        exit 1
    fi
else
    echo "⚠️  Moly service not running (systemctl)"
    echo "  Try manual restart: sudo service moly restart"
fi

echo ""

# Step 3: Verify service is healthy
echo "Step 3️⃣  Verifying service health..."
if curl -s http://localhost:8080/health | grep -q "ok"; then
    echo "✓ Service health check passed"
else
    echo "⚠️  Health check failed - investigate manually"
fi

echo ""

# Step 4: Log the incident
echo "Step 4️⃣  Logging incident..."
INCIDENT_LOG="/var/log/moly/rollback_incidents.log"
echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] EMERGENCY ROLLBACK: Phase $PHASE ($PHASE_NAME) disabled" >> $INCIDENT_LOG 2>/dev/null || true
echo "✓ Incident logged"

echo ""
echo "🚨 ROLLBACK COMPLETE"
echo "===================="
echo ""
echo "What to do next:"
echo "1. ✓ Phase $PHASE has been disabled"
echo "2. ✓ Service has been restarted"
echo "3. ⏭  Investigate what triggered the rollback"
echo "4. ⏭  Review metrics and logs for root cause"
echo "5. ⏭  Fix the issue before re-enabling"
echo "6. ⏭  Update JIRA ticket with incident details"
echo ""
echo "To view logs:"
echo "  journalctl -u moly -n 100 -f"
echo ""
echo "To check current phase status:"
echo "  ./check_phases.sh"
echo ""
echo "To investigate the incident:"
echo "  grep -i phase /var/log/moly/*.log | tail -50"
echo ""

# Step 5: Alert on-call (if configured)
if command -v send_alert &> /dev/null; then
    send_alert "🚨 EMERGENCY: Phase $PHASE rollback triggered at $(date -u +%Y-%m-%dT%H:%M:%SZ)"
    echo "✓ On-call team notified"
else
    echo "ℹ️  (Configure send_alert for automatic on-call notification)"
fi

echo ""
echo "✅ Rollback procedure complete"
