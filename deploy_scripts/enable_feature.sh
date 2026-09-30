#!/bin/bash
set -e

# Enable/disable a feature flag
# Usage: ./enable_feature.sh <phase> <enable|disable> [rollout_percentage]

PHASE=${1:-""}
ACTION=${2:-""}
ROLLOUT=${3:-"100"}

if [ -z "$PHASE" ] || [ -z "$ACTION" ]; then
    echo "Usage: ./enable_feature.sh <1|2|3|4> <enable|disable> [rollout_percentage]"
    echo ""
    echo "Examples:"
    echo "  ./enable_feature.sh 1 enable 10    # Enable Phase 1 for 10% of users"
    echo "  ./enable_feature.sh 2 enable 100   # Enable Phase 2 for 100% of users"
    echo "  ./enable_feature.sh 3 disable      # Disable Phase 3 (rollback)"
    exit 1
fi

echo "🔧 FEATURE CONTROL"
echo "=================="
echo "Phase: $PHASE"
echo "Action: $ACTION"
echo "Rollout: $ROLLOUT%"
echo ""

# Validate inputs
if [ "$ACTION" != "enable" ] && [ "$ACTION" != "disable" ]; then
    echo "❌ Action must be 'enable' or 'disable'"
    exit 1
fi

if [ "$ROLLOUT" -lt 0 ] || [ "$ROLLOUT" -gt 100 ]; then
    echo "❌ Rollout must be 0-100"
    exit 1
fi

# Set environment variables
case $PHASE in
    1)
        if [ "$ACTION" = "enable" ]; then
            export MOLY_USE_EXTRACTION_LOCK=true
            echo "✓ Phase 1 (Extraction Lock) ENABLED at $ROLLOUT%"
        else
            export MOLY_USE_EXTRACTION_LOCK=false
            echo "✓ Phase 1 (Extraction Lock) DISABLED"
        fi
        ;;
    2)
        if [ "$ACTION" = "enable" ]; then
            export MOLY_USE_LAYER5_CONFLICT_GATE=true
            echo "✓ Phase 2 (Layer 5 Conflicts) ENABLED at $ROLLOUT%"
        else
            export MOLY_USE_LAYER5_CONFLICT_GATE=false
            echo "✓ Phase 2 (Layer 5 Conflicts) DISABLED"
        fi
        ;;
    3)
        if [ "$ACTION" = "enable" ]; then
            export MOLY_USE_CONSTRAINED_RESPONSE_GENERATION=true
            echo "✓ Phase 3 (Constrained Generation) ENABLED at $ROLLOUT%"
        else
            export MOLY_USE_CONSTRAINED_RESPONSE_GENERATION=false
            echo "✓ Phase 3 (Constrained Generation) DISABLED"
        fi
        ;;
    4)
        if [ "$ACTION" = "enable" ]; then
            export MOLY_USE_CLEAN_SCHEMA=true
            echo "✓ Phase 4 (Clean Schema) ENABLED"
        else
            export MOLY_USE_CLEAN_SCHEMA=false
            echo "✓ Phase 4 (Clean Schema) DISABLED"
        fi
        ;;
    *)
        echo "❌ Phase must be 1, 2, 3, or 4"
        exit 1
        ;;
esac

# Set metrics rate for early phases (high monitoring)
if [ "$PHASE" -lt "3" ]; then
    export MOLY_METRICS_RATE=100
    echo "✓ Metrics collection set to 100%"
fi

# Write to .env for persistent configuration (optional)
if [ -f ".env" ]; then
    echo "🔒 Update .env file if needed:"
    echo "  MOLY_USE_EXTRACTION_LOCK=$MOLY_USE_EXTRACTION_LOCK"
    echo "  MOLY_USE_LAYER5_CONFLICT_GATE=$MOLY_USE_LAYER5_CONFLICT_GATE"
    echo "  MOLY_USE_CONSTRAINED_RESPONSE_GENERATION=$MOLY_USE_CONSTRAINED_RESPONSE_GENERATION"
    echo "  MOLY_USE_CLEAN_SCHEMA=$MOLY_USE_CLEAN_SCHEMA"
    echo "  MOLY_METRICS_RATE=$MOLY_METRICS_RATE"
fi

echo ""
echo "✅ Feature configuration updated"
echo ""
echo "To restart services with new configuration:"
echo "  systemctl restart moly"
